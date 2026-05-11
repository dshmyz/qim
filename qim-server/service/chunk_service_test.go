package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"os"
	"testing"

	"qim-server/model"
	"qim-server/repository"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
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

func TestChunkService_InitUpload_NewUpload(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)

	// 创建临时目录用于存储分片
	tempDir := t.TempDir()

	service := NewChunkService(db, tempDir)

	task, uploadedIndexes, isInstant, err := service.InitUpload(user.ID, "test.txt", 15*1024*1024, "file-hash-123", nil)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.NotEmpty(t, task.UploadID)
	assert.Equal(t, user.ID, task.UserID)
	assert.Equal(t, "test.txt", task.Filename)
	assert.Equal(t, int64(15*1024*1024), task.FileSize)
	assert.Equal(t, "file-hash-123", task.FileHash)
	assert.Equal(t, 3, task.TotalChunks) // 15MB / 5MB = 3 chunks
	assert.Equal(t, "pending", task.Status)
	assert.False(t, isInstant)
	assert.Empty(t, uploadedIndexes)
}

func TestChunkService_InitUpload_InstantUpload(t *testing.T) {
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
	service := NewChunkService(db, tempDir)

	// 尝试上传相同哈希的文件
	task, uploadedIndexes, isInstant, err := service.InitUpload(user.ID, "new.txt", 1024, "same-hash-123", nil)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.True(t, isInstant)
	assert.Empty(t, uploadedIndexes)
	assert.Equal(t, "completed", task.Status)
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
	service := NewChunkService(db, tempDir)

	// 尝试继续上传
	task, uploadedIndexes, isInstant, err := service.InitUpload(user.ID, "test.txt", 15*1024*1024, "file-hash-456", nil)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.False(t, isInstant)
	assert.Equal(t, "existing-upload-id", task.UploadID)
	assert.Equal(t, []int{0}, uploadedIndexes)
}

func TestChunkService_UploadChunk(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)

	tempDir := t.TempDir()
	service := NewChunkService(db, tempDir)

	// 创建上传任务
	task, _, _, err := service.InitUpload(user.ID, "test.txt", 15*1024*1024, "file-hash-789", nil)
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
	err = service.UploadChunk(task.UploadID, 0, chunkData, chunkHash)
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

func TestChunkService_UploadChunk_InvalidHash(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)

	tempDir := t.TempDir()
	service := NewChunkService(db, tempDir)

	// 创建上传任务
	task, _, _, err := service.InitUpload(user.ID, "test.txt", 15*1024*1024, "file-hash-invalid", nil)
	assert.NoError(t, err)

	// 准备分片数据
	chunkData := make([]byte, 5*1024*1024)

	// 使用错误的哈希
	err = service.UploadChunk(task.UploadID, 0, chunkData, "wrong-hash")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "分片哈希不匹配")
}

func TestChunkService_CompleteUpload(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)

	tempDir := t.TempDir()
	service := NewChunkService(db, tempDir)

	// 创建上传任务
	task, _, _, err := service.InitUpload(user.ID, "test.txt", 15*1024*1024, "file-hash-complete", nil)
	assert.NoError(t, err)

	// 上传所有分片
	for i := 0; i < task.TotalChunks; i++ {
		chunkData := make([]byte, 5*1024*1024)
		for j := range chunkData {
			chunkData[j] = byte((i*256 + j) % 256)
		}
		hash := md5.Sum(chunkData)
		chunkHash := hex.EncodeToString(hash[:])

		err = service.UploadChunk(task.UploadID, i, chunkData, chunkHash)
		assert.NoError(t, err)
	}

	// 完成上传
	file, err := service.CompleteUpload(task.UploadID)
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

func TestChunkService_CancelUpload(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)

	tempDir := t.TempDir()
	service := NewChunkService(db, tempDir)

	// 创建上传任务
	task, _, _, err := service.InitUpload(user.ID, "test.txt", 15*1024*1024, "file-hash-cancel", nil)
	assert.NoError(t, err)

	// 上传一个分片
	chunkData := make([]byte, 5*1024*1024)
	hash := md5.Sum(chunkData)
	chunkHash := hex.EncodeToString(hash[:])
	err = service.UploadChunk(task.UploadID, 0, chunkData, chunkHash)
	assert.NoError(t, err)

	// 取消上传
	err = service.CancelUpload(task.UploadID)
	assert.NoError(t, err)

	// 验证任务已删除
	_, err = repository.NewChunkRepository(db).GetUploadTask(context.Background(), task.UploadID)
	assert.Error(t, err)

	// 验证分片已删除
	chunks, _ := repository.NewChunkRepository(db).GetChunksByUploadID(context.Background(), task.UploadID)
	assert.Empty(t, chunks)
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

func TestChunkService_GetMimeType(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"test.txt", "text/plain"},
		{"document.pdf", "application/pdf"},
		{"image.jpg", "image/jpeg"},
		{"image.jpeg", "image/jpeg"},
		{"image.png", "image/png"},
		{"video.mp4", "video/mp4"},
		{"audio.mp3", "audio/mpeg"},
		{"unknown.xyz", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := getMimeType(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
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
	service := NewChunkService(db, tempDir)

	task, _, _, err := service.InitUpload(user.ID, "test.txt", 10*1024*1024, "file-hash-folder", &folder.ID)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, folder.ID, *task.FolderID)
}

func TestChunkService_CompleteUpload_VerifyFileIntegrity(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)

	tempDir := t.TempDir()
	service := NewChunkService(db, tempDir)

	// 创建一个小文件用于测试
	fileSize := int64(3 * 1024 * 1024) // 3MB
	task, _, _, err := service.InitUpload(user.ID, "integrity.txt", fileSize, "file-hash-integrity", nil)
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

		err = service.UploadChunk(task.UploadID, i, chunkData, chunkHash)
		assert.NoError(t, err)
	}

	// 完成上传
	file, err := service.CompleteUpload(task.UploadID)
	assert.NoError(t, err)
	assert.NotNil(t, file)
	assert.Equal(t, expectedChecksum, file.Checksum)

	// 验证合并后的文件内容
	mergedData, err := os.ReadFile(file.StoragePath)
	assert.NoError(t, err)
	assert.Equal(t, fullData, mergedData)
}

func TestChunkService_UploadChunk_OutOfOrder(t *testing.T) {
	db := setupChunkServiceTestDB(t)
	user := createTestUser(t, db)

	tempDir := t.TempDir()
	service := NewChunkService(db, tempDir)

	// 创建上传任务
	task, _, _, err := service.InitUpload(user.ID, "test.txt", 15*1024*1024, "file-hash-order", nil)
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

		err = service.UploadChunk(task.UploadID, i, chunkData, chunkHash)
		assert.NoError(t, err)
	}

	// 验证所有分片都已上传
	chunks, err := repository.NewChunkRepository(db).GetChunksByUploadID(context.Background(), task.UploadID)
	assert.NoError(t, err)
	assert.Len(t, chunks, 3)

	// 完成上传
	file, err := service.CompleteUpload(task.UploadID)
	assert.NoError(t, err)
	assert.NotNil(t, file)
}
