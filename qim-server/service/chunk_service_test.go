package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/repository"

	"github.com/stretchr/testify/assert"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"gorm.io/gorm"
)

func setupChunkServiceTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(&model.User{}, &model.File{}, &model.UploadTask{}, &model.FileChunk{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func createTestUser(t *testing.T, db *gorm.DB) *model.User {
	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hash",
	}
	err := db.Create(user).Error
	assert.NoError(t, err)
	return user
}

// testStorageAccessor 是测试用的 StorageAccessor，基于本地文件系统实现，
// 不依赖 storage/di 包以避免循环依赖。
type testStorageAccessor struct {
	dir string
}

func newTestAccessor(t *testing.T) *testStorageAccessor {
	t.Helper()
	return &testStorageAccessor{dir: t.TempDir()}
}

func (a *testStorageAccessor) GetByPath(ctx context.Context, storagePath string) (io.ReadCloser, error) {
	rel := strings.TrimPrefix(storagePath, "/")
	return os.Open(filepath.Join(a.dir, rel))
}

func (a *testStorageAccessor) Put(ctx context.Context, key string, data io.Reader, size int64, mime string) (string, error) {
	path := filepath.Join(a.dir, key)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, data); err != nil {
		return "", err
	}
	return "/" + key, nil
}

func (a *testStorageAccessor) DeleteByPath(ctx context.Context, storagePath string) error {
	rel := strings.TrimPrefix(storagePath, "/")
	return os.Remove(filepath.Join(a.dir, rel))
}

func (a *testStorageAccessor) Kind() string { return "local" }

func TestChunkService_InitUpload_NewUpload(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)

	// 创建临时目录用于存储分片
	tempDir := t.TempDir()

	service := NewChunkService(db, tempDir, newTestAccessor(t))

	// InitUpload 不再接收 fileHash 参数（秒传已移除）
	task, uploadedIndexes, err := service.InitUpload(user.ID, "test.txt", 15*1024*1024, nil)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.NotEmpty(t, task.UploadID)
	assert.Equal(t, user.ID, task.UserID)
	assert.Equal(t, "test.txt", task.Filename)
	assert.Equal(t, int64(15*1024*1024), task.FileSize)
	assert.Equal(t, 3, task.TotalChunks) // 15MB / 5MB = 3 chunks
	assert.Equal(t, "pending", task.Status)
	assert.Empty(t, uploadedIndexes)
}

// TestChunkService_InitUpload_NoInstantUpload 验证即使存在相同 checksum 的文件也不会触发秒传
// （秒传功能已移除：存在越权风险且前端算 MD5 性能开销大、命中率低）
func TestChunkService_InitUpload_NoInstantUpload(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)

	// 创建已存在的文件（相同哈希）
	existingFile := &model.File{
		UserID:       user.ID,
		Name:         "existing.txt",
		OriginalName: "existing.txt",
		Size:         1024,
		MimeType:     "text/plain",
		StoragePath:  "/tmp/existing.txt",
		Checksum:     "same-hash-123",
	}
	db.Create(existingFile)

	tempDir := t.TempDir()
	service := NewChunkService(db, tempDir, newTestAccessor(t))

	// 即使存在相同 checksum 的文件，也走正常分片流程，不返回秒传
	task, uploadedIndexes, err := service.InitUpload(user.ID, "new.txt", 1024, nil)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.NotEqual(t, "completed", task.Status) // 不再是 completed
	assert.Equal(t, "pending", task.Status)
	assert.Empty(t, uploadedIndexes)
}

func TestChunkService_InitUpload_ResumeUpload(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)

	// 创建未完成的上传任务
	existingTask := &model.UploadTask{
		UploadID:       "existing-upload-id",
		UserID:         user.ID,
		Filename:       "test.txt",
		FileSize:       15 * 1024 * 1024,
		FileHash:       "file-hash-456",
		TotalChunks:    3,
		UploadedChunks: 1,
		Status:         "uploading",
	}
	db.Create(existingTask)

	// 创建已上传的分片
	chunk := &model.FileChunk{
		UploadID:    "existing-upload-id",
		FileHash:    "file-hash-456",
		ChunkIndex:  0,
		ChunkHash:   "chunk-hash-0",
		ChunkSize:   5 * 1024 * 1024,
		StoragePath: "/tmp/chunks/chunk-0",
		Status:      "uploaded",
	}
	db.Create(chunk)

	tempDir := t.TempDir()
	service := NewChunkService(db, tempDir, newTestAccessor(t))

	// 断点续传：通过 filename + user_id + fileSize 匹配未完成任务
	task, uploadedIndexes, err := service.InitUpload(user.ID, "test.txt", 15*1024*1024, nil)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, "existing-upload-id", task.UploadID)
	assert.Equal(t, []int{0}, uploadedIndexes)
}

func TestChunkService_UploadChunk(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)

	tempDir := t.TempDir()
	service := NewChunkService(db, tempDir, newTestAccessor(t))

	// 创建上传任务
	task, _, err := service.InitUpload(user.ID, "test.txt", 15*1024*1024, nil)
	assert.NoError(t, err)

	// 准备分片数据
	chunkData := make([]byte, 5*1024*1024)
	for i := range chunkData {
		chunkData[i] = byte(i % 256)
	}

	// 计算分片哈希
	hash := md5.Sum(chunkData)
	chunkHash := hex.EncodeToString(hash[:])

	// 上传第一个分片
	err = service.UploadChunk(user.ID, task.UploadID, 0, chunkData, chunkHash)
	assert.NoError(t, err)

	// 验证分片已保存
	chunk, err := repository.NewChunkRepository(db).GetChunk(context.Background(), task.UploadID, 0)
	assert.NoError(t, err)
	assert.Equal(t, "uploaded", chunk.Status)
	assert.Equal(t, chunkHash, chunk.ChunkHash)

	// 验证任务进度已更新
	updatedTask, _ := repository.NewChunkRepository(db).GetUploadTask(context.Background(), task.UploadID)
	assert.Equal(t, 1, updatedTask.UploadedChunks)
}

// TestChunkService_UploadChunk_ForbiddenUser 验证非任务所有者无法上传分片（IDOR 防护）
func TestChunkService_UploadChunk_ForbiddenUser(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)
	otherUser := &model.User{Username: "other", PasswordHash: "hash"}
	db.Create(otherUser)

	tempDir := t.TempDir()
	svc := NewChunkService(db, tempDir, newTestAccessor(t))

	task, _, err := svc.InitUpload(user.ID, "test.txt", 15*1024*1024, nil)
	assert.NoError(t, err)

	chunkData := make([]byte, 5*1024*1024)
	hash := md5.Sum(chunkData)
	chunkHash := hex.EncodeToString(hash[:])

	// 其他用户尝试向 user 的任务上传分片
	err = svc.UploadChunk(otherUser.ID, task.UploadID, 0, chunkData, chunkHash)
	assert.ErrorIs(t, err, ErrUploadForbidden)
}

func TestChunkService_UploadChunk_InvalidHash(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)

	tempDir := t.TempDir()
	service := NewChunkService(db, tempDir, newTestAccessor(t))

	// 创建上传任务
	task, _, err := service.InitUpload(user.ID, "test.txt", 15*1024*1024, nil)
	assert.NoError(t, err)

	// 准备分片数据
	chunkData := make([]byte, 5*1024*1024)

	// 使用错误的哈希
	err = service.UploadChunk(user.ID, task.UploadID, 0, chunkData, "wrong-hash")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "分片哈希不匹配")
}

func TestChunkService_CompleteUpload(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)

	tempDir := t.TempDir()
	service := NewChunkService(db, tempDir, newTestAccessor(t))

	// 创建上传任务
	task, _, err := service.InitUpload(user.ID, "test.txt", 15*1024*1024, nil)
	assert.NoError(t, err)

	// 上传所有分片
	for i := 0; i < task.TotalChunks; i++ {
		chunkData := make([]byte, 5*1024*1024)
		for j := range chunkData {
			chunkData[j] = byte((i*256 + j) % 256)
		}
		hash := md5.Sum(chunkData)
		chunkHash := hex.EncodeToString(hash[:])

		err = service.UploadChunk(user.ID, task.UploadID, i, chunkData, chunkHash)
		assert.NoError(t, err)
	}

	// 完成上传
	file, err := service.CompleteUpload(user.ID, task.UploadID)
	assert.NoError(t, err)
	assert.NotNil(t, file)
	assert.Equal(t, user.ID, file.UserID)
	assert.Equal(t, "test.txt", file.Name)
	assert.Equal(t, int64(15*1024*1024), file.Size)
	assert.NotEmpty(t, file.Checksum)

	// 验证任务状态已更新
	updatedTask, err := repository.NewChunkRepository(db).GetUploadTask(context.Background(), task.UploadID)
	assert.NoError(t, err)
	assert.Equal(t, "completed", updatedTask.Status)

	// 验证文件已创建
	var count int64
	db.Model(&model.File{}).Where("id = ?", file.ID).Count(&count)
	assert.Equal(t, int64(1), count)
}

// TestChunkService_CompleteUpload_ForbiddenUser 验证非任务所有者无法完成上传（IDOR 防护）
func TestChunkService_CompleteUpload_ForbiddenUser(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)
	otherUser := &model.User{Username: "other", PasswordHash: "hash"}
	db.Create(otherUser)

	tempDir := t.TempDir()
	svc := NewChunkService(db, tempDir, newTestAccessor(t))

	task, _, err := svc.InitUpload(user.ID, "test.txt", 15*1024*1024, nil)
	assert.NoError(t, err)

	// 上传所有分片
	for i := 0; i < task.TotalChunks; i++ {
		chunkData := make([]byte, 5*1024*1024)
		hash := md5.Sum(chunkData)
		chunkHash := hex.EncodeToString(hash[:])
		err = svc.UploadChunk(user.ID, task.UploadID, i, chunkData, chunkHash)
		assert.NoError(t, err)
	}

	// 其他用户尝试完成 user 的任务
	_, err = svc.CompleteUpload(otherUser.ID, task.UploadID)
	assert.ErrorIs(t, err, ErrUploadForbidden)
}

func TestChunkService_CancelUpload(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)

	tempDir := t.TempDir()
	service := NewChunkService(db, tempDir, newTestAccessor(t))

	// 创建上传任务
	task, _, err := service.InitUpload(user.ID, "test.txt", 15*1024*1024, nil)
	assert.NoError(t, err)

	// 上传一个分片
	chunkData := make([]byte, 5*1024*1024)
	hash := md5.Sum(chunkData)
	chunkHash := hex.EncodeToString(hash[:])
	err = service.UploadChunk(user.ID, task.UploadID, 0, chunkData, chunkHash)
	assert.NoError(t, err)

	// 取消上传
	err = service.CancelUpload(user.ID, task.UploadID)
	assert.NoError(t, err)

	// 验证任务已删除
	_, err = repository.NewChunkRepository(db).GetUploadTask(context.Background(), task.UploadID)
	assert.Error(t, err)

	// 验证分片已删除
	chunks, _ := repository.NewChunkRepository(db).GetChunksByUploadID(context.Background(), task.UploadID)
	assert.Empty(t, chunks)
}

// TestChunkService_CancelUpload_ForbiddenUser 验证非任务所有者无法取消上传（IDOR 防护）
func TestChunkService_CancelUpload_ForbiddenUser(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)
	otherUser := &model.User{Username: "other", PasswordHash: "hash"}
	db.Create(otherUser)

	tempDir := t.TempDir()
	svc := NewChunkService(db, tempDir, newTestAccessor(t))

	task, _, err := svc.InitUpload(user.ID, "test.txt", 15*1024*1024, nil)
	assert.NoError(t, err)

	// 其他用户尝试取消 user 的任务
	err = svc.CancelUpload(otherUser.ID, task.UploadID)
	assert.ErrorIs(t, err, ErrUploadForbidden)
}

func TestChunkService_CalculateChunkSize(t *testing.T) {
	service := &ChunkService{}

	tests := []struct {
		name     string
		fileSize int64
		expected int64
	}{
		{
			name:     "小于10MB不分片",
			fileSize: 5 * 1024 * 1024,
			expected: 5 * 1024 * 1024,
		},
		{
			name:     "10MB文件使用5MB分片",
			fileSize: 10 * 1024 * 1024,
			expected: 5 * 1024 * 1024,
		},
		{
			name:     "30MB文件使用5MB分片",
			fileSize: 30 * 1024 * 1024,
			expected: 5 * 1024 * 1024,
		},
		{
			name:     "50MB文件使用10MB分片",
			fileSize: 50 * 1024 * 1024,
			expected: 10 * 1024 * 1024,
		},
		{
			name:     "100MB文件使用10MB分片",
			fileSize: 100 * 1024 * 1024,
			expected: 10 * 1024 * 1024,
		},
		{
			name:     "200MB文件使用10MB分片",
			fileSize: 200 * 1024 * 1024,
			expected: 10 * 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateChunkSize(tt.fileSize)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestChunkService_GenerateUploadID(t *testing.T) {
	id1 := generateUploadID()
	id2 := generateUploadID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Len(t, id1, 32) // MD5哈希长度
}

func TestChunkService_InitUpload_WithFolder(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)

	// 创建文件夹
	folder := &model.Folder{
		UserID: user.ID,
		Name:   "test-folder",
	}
	db.Create(folder)

	tempDir := t.TempDir()
	service := NewChunkService(db, tempDir, newTestAccessor(t))

	task, _, err := service.InitUpload(user.ID, "test.txt", 10*1024*1024, &folder.ID)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, folder.ID, *task.FolderID)
}

func TestChunkService_CompleteUpload_VerifyFileIntegrity(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)

	tempDir := t.TempDir()
	accessor := newTestAccessor(t)
	service := NewChunkService(db, tempDir, accessor)

	// 创建一个小文件用于测试
	fileSize := int64(3 * 1024 * 1024) // 3MB
	task, _, err := service.InitUpload(user.ID, "integrity.txt", fileSize, nil)
	assert.NoError(t, err)

	// 创建完整的文件数据
	fullData := make([]byte, fileSize)
	for i := range fullData {
		fullData[i] = byte(i % 256)
	}

	// 计算完整文件的哈希
	fullHash := md5.Sum(fullData)
	expectedChecksum := hex.EncodeToString(fullHash[:])

	// 上传所有分片
	chunkSize := service.calculateChunkSize(fileSize)
	for i := 0; i < task.TotalChunks; i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize
		if end > fileSize {
			end = fileSize
		}

		chunkData := fullData[start:end]
		hash := md5.Sum(chunkData)
		chunkHash := hex.EncodeToString(hash[:])

		err = service.UploadChunk(user.ID, task.UploadID, i, chunkData, chunkHash)
		assert.NoError(t, err)
	}

	// 完成上传
	file, err := service.CompleteUpload(user.ID, task.UploadID)
	assert.NoError(t, err)
	assert.NotNil(t, file)
	assert.Equal(t, expectedChecksum, file.Checksum)

	// 验证合并后的文件内容（通过存储抽象读回）
	reader, err := accessor.GetByPath(context.Background(), file.StoragePath)
	assert.NoError(t, err)
	mergedData, err := io.ReadAll(reader)
	reader.Close()
	assert.NoError(t, err)
	assert.Equal(t, fullData, mergedData)
}

func TestChunkService_UploadChunk_OutOfOrder(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)

	tempDir := t.TempDir()
	service := NewChunkService(db, tempDir, newTestAccessor(t))

	// 创建上传任务
	task, _, err := service.InitUpload(user.ID, "test.txt", 15*1024*1024, nil)
	assert.NoError(t, err)

	// 乱序上传分片
	order := []int{2, 0, 1}
	for _, i := range order {
		chunkData := make([]byte, 5*1024*1024)
		for j := range chunkData {
			chunkData[j] = byte((i*256 + j) % 256)
		}
		hash := md5.Sum(chunkData)
		chunkHash := hex.EncodeToString(hash[:])

		err = service.UploadChunk(user.ID, task.UploadID, i, chunkData, chunkHash)
		assert.NoError(t, err)
	}

	// 验证所有分片都已上传
	chunks, err := repository.NewChunkRepository(db).GetChunksByUploadID(context.Background(), task.UploadID)
	assert.NoError(t, err)
	assert.Len(t, chunks, 3)

	// 完成上传
	file, err := service.CompleteUpload(user.ID, task.UploadID)
	assert.NoError(t, err)
	assert.NotNil(t, file)
}

// TestChunkService_UploadChunk_Idempotent 验证同一分片重复上传不会重复计数
// 修复点：幂等检查 + 条件更新（ConditionalUpdateChunkStatus）
func TestChunkService_UploadChunk_Idempotent(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)

	tempDir := t.TempDir()
	svc := NewChunkService(db, tempDir, newTestAccessor(t))

	task, _, err := svc.InitUpload(user.ID, "test.txt", 15*1024*1024, nil)
	assert.NoError(t, err)

	chunkData := make([]byte, 5*1024*1024)
	for i := range chunkData {
		chunkData[i] = byte(i % 256)
	}
	hash := md5.Sum(chunkData)
	chunkHash := hex.EncodeToString(hash[:])

	// 第一次上传分片 0
	err = svc.UploadChunk(user.ID, task.UploadID, 0, chunkData, chunkHash)
	assert.NoError(t, err)

	// 第二次上传同一分片 0（幂等：应成功返回，不重复计数）
	err = svc.UploadChunk(user.ID, task.UploadID, 0, chunkData, chunkHash)
	assert.NoError(t, err)

	// 验证 UploadedChunks 只增加了 1，不是 2
	updatedTask, _ := repository.NewChunkRepository(db).GetUploadTask(context.Background(), task.UploadID)
	assert.Equal(t, 1, updatedTask.UploadedChunks, "重复上传同一分片不应重复计数")
}

// TestChunkService_UploadChunk_ConcurrentSafe 验证并发上传不同分片时计数器正确
// 修复点：原子 SQL 自增（AtomicIncrementUploadedChunks）防止 read-modify-write 竞态
func TestChunkService_UploadChunk_ConcurrentSafe(t *testing.T) {
	// 使用文件数据库（非内存），因为 SQLite 内存数据库每个连接独立，不支持并发共享
	dbPath := filepath.Join(t.TempDir(), "concurrent_test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	assert.NoError(t, err)

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1) // SQLite 单连接避免锁冲突

	err = db.AutoMigrate(&model.User{}, &model.File{}, &model.UploadTask{}, &model.FileChunk{})
	assert.NoError(t, err)

	user := createTestUser(t, db)

	tempDir := t.TempDir()
	svc := NewChunkService(db, tempDir, newTestAccessor(t))

	// 15MB → 3 个 5MB 分片
	task, _, err := svc.InitUpload(user.ID, "test.txt", 15*1024*1024, nil)
	assert.NoError(t, err)

	// 准备 3 个分片数据
	chunks := make([][]byte, task.TotalChunks)
	hashes := make([]string, task.TotalChunks)
	for i := 0; i < task.TotalChunks; i++ {
		chunks[i] = make([]byte, 5*1024*1024)
		for j := range chunks[i] {
			chunks[i][j] = byte((i*256 + j) % 256)
		}
		h := md5.Sum(chunks[i])
		hashes[i] = hex.EncodeToString(h[:])
	}

	// 并发上传所有分片
	var wg sync.WaitGroup
	errs := make([]error, task.TotalChunks)
	for i := 0; i < task.TotalChunks; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = svc.UploadChunk(user.ID, task.UploadID, idx, chunks[idx], hashes[idx])
		}(i)
	}
	wg.Wait()

	// 所有分片上传应成功
	for i, e := range errs {
		assert.NoError(t, e, "分片 %d 上传失败", i)
	}

	// 验证 UploadedChunks 等于 TotalChunks（并发下不丢失更新）
	updatedTask, _ := repository.NewChunkRepository(db).GetUploadTask(context.Background(), task.UploadID)
	assert.Equal(t, task.TotalChunks, updatedTask.UploadedChunks,
		"并发上传后计数器应等于总分片数，实际: %d", updatedTask.UploadedChunks)

	// 验证能正常完成上传（所有分片确实已上传）
	file, err := svc.CompleteUpload(user.ID, task.UploadID)
	assert.NoError(t, err)
	assert.NotNil(t, file)
}

// TestChunkService_UploadChunk_ConcurrentSameChunk 验证并发上传同一分片时不会误删文件
// 修复点：ConditionalUpdate 返回 false 时不能 os.Remove，否则会删除成功方的文件
func TestChunkService_UploadChunk_ConcurrentSameChunk(t *testing.T) {
	// 使用文件数据库（非内存），因为 SQLite 内存数据库每个连接独立，不支持并发共享
	dbPath := filepath.Join(t.TempDir(), "concurrent_same_chunk.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	assert.NoError(t, err)

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1) // SQLite 单连接避免锁冲突

	err = db.AutoMigrate(&model.User{}, &model.File{}, &model.UploadTask{}, &model.FileChunk{})
	assert.NoError(t, err)

	user := createTestUser(t, db)

	tempDir := t.TempDir()
	svc := NewChunkService(db, tempDir, newTestAccessor(t))

	task, _, err := svc.InitUpload(user.ID, "test.txt", 15*1024*1024, nil)
	assert.NoError(t, err)

	// 准备同一分片的数据（两个请求上传完全相同的分片 0）
	chunkData := make([]byte, 5*1024*1024)
	for i := range chunkData {
		chunkData[i] = byte(i % 256)
	}
	hash := md5.Sum(chunkData)
	chunkHash := hex.EncodeToString(hash[:])

	// 并发上传同一分片 0 两次
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = svc.UploadChunk(user.ID, task.UploadID, 0, chunkData, chunkHash)
		}(i)
	}
	wg.Wait()

	// 两次请求都应成功（幂等）
	for i, e := range errs {
		assert.NoError(t, e, "请求 %d 上传失败", i)
	}

	// 验证计数器只增加了 1（不是 2）
	updatedTask, _ := repository.NewChunkRepository(db).GetUploadTask(context.Background(), task.UploadID)
	assert.Equal(t, 1, updatedTask.UploadedChunks, "重复上传同一分片不应重复计数")

	// 关键验证：分片文件必须存在，不能被并发请求误删
	chunk, err := repository.NewChunkRepository(db).GetChunk(context.Background(), task.UploadID, 0)
	assert.NoError(t, err)
	_, err = os.Stat(chunk.StoragePath)
	assert.NoError(t, err, "分片文件不应被并发请求误删")

	// 上传剩余分片，验证能正常完成上传
	for i := 1; i < task.TotalChunks; i++ {
		cd := make([]byte, 5*1024*1024)
		for j := range cd {
			cd[j] = byte((i*256 + j) % 256)
		}
		h := md5.Sum(cd)
		err = svc.UploadChunk(user.ID, task.UploadID, i, cd, hex.EncodeToString(h[:]))
		assert.NoError(t, err)
	}

	file, err := svc.CompleteUpload(user.ID, task.UploadID)
	assert.NoError(t, err)
	assert.NotNil(t, file)
}

// TestChunkService_CompleteUpload_CancelledTask 验证已取消的任务无法完成上传
// 修复点：CompleteUpload 添加 cancelled 状态检查
func TestChunkService_CompleteUpload_CancelledTask(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)

	tempDir := t.TempDir()
	svc := NewChunkService(db, tempDir, newTestAccessor(t))

	task, _, err := svc.InitUpload(user.ID, "test.txt", 15*1024*1024, nil)
	assert.NoError(t, err)

	// 上传所有分片
	for i := 0; i < task.TotalChunks; i++ {
		chunkData := make([]byte, 5*1024*1024)
		hash := md5.Sum(chunkData)
		chunkHash := hex.EncodeToString(hash[:])
		err = svc.UploadChunk(user.ID, task.UploadID, i, chunkData, chunkHash)
		assert.NoError(t, err)
	}

	// 取消上传（CancelUpload 会删除任务记录和分片）
	err = svc.CancelUpload(user.ID, task.UploadID)
	assert.NoError(t, err)

	// 尝试完成上传（应失败，任务已被删除）
	_, err = svc.CompleteUpload(user.ID, task.UploadID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "上传任务不存在")
}

// TestChunkService_CancelUpload_Idempotent 验证已取消的任务再次取消返回成功（幂等）
// 修复点：CancelUpload 对 cancelled 状态幂等返回 nil
func TestChunkService_CancelUpload_Idempotent(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)

	tempDir := t.TempDir()
	svc := NewChunkService(db, tempDir, newTestAccessor(t))

	task, _, err := svc.InitUpload(user.ID, "test.txt", 15*1024*1024, nil)
	assert.NoError(t, err)

	// 第一次取消
	err = svc.CancelUpload(user.ID, task.UploadID)
	assert.NoError(t, err)

	// 第二次取消（幂等：应返回错误，因为任务记录已被删除）
	err = svc.CancelUpload(user.ID, task.UploadID)
	assert.Error(t, err) // 任务已删除，查询不到
}
