package repository

import (
	"context"
	"testing"

	"qim-server/model"

	"github.com/stretchr/testify/assert"
	"qim-server/pkg/sqlite"
	"gorm.io/gorm"
)

func setupChunkTestDB(t *testing.T) *gorm.DB {
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

func TestChunkRepository_CreateUploadTask(t *testing.T) {
	db := setupChunkTestDB(t)
	repo := NewChunkRepository(db)
	ctx := context.Background()

	// 创建测试用户
	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hash",
	}
	db.Create(user)

	task := &model.UploadTask{
		UploadID:    "test-upload-id-123",
		UserID:      user.ID,
		Filename:    "test.txt",
		FileSize:    1024,
		FileHash:    "abc123",
		TotalChunks: 5,
	}

	err := repo.CreateUploadTask(ctx, task)
	assert.NoError(t, err)
	assert.NotZero(t, task.ID)
}

func TestChunkRepository_GetUploadTask(t *testing.T) {
	db := setupChunkTestDB(t)
	repo := NewChunkRepository(db)
	ctx := context.Background()

	// 创建测试用户
	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hash",
	}
	db.Create(user)

	// 创建测试任务
	task := &model.UploadTask{
		UploadID:    "test-upload-id-456",
		UserID:      user.ID,
		Filename:    "test.txt",
		FileSize:    1024,
		FileHash:    "abc123",
		TotalChunks: 5,
	}
	repo.CreateUploadTask(ctx, task)

	// 测试获取
	found, err := repo.GetUploadTask(ctx, "test-upload-id-456")
	assert.NoError(t, err)
	assert.Equal(t, task.UploadID, found.UploadID)
	assert.Equal(t, task.Filename, found.Filename)

	// 测试不存在的任务
	_, err = repo.GetUploadTask(ctx, "not-exist")
	assert.Error(t, err)
}

func TestChunkRepository_UpdateUploadTask(t *testing.T) {
	db := setupChunkTestDB(t)
	repo := NewChunkRepository(db)
	ctx := context.Background()

	// 创建测试用户
	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hash",
	}
	db.Create(user)

	// 创建测试任务
	task := &model.UploadTask{
		UploadID:       "test-upload-id-789",
		UserID:         user.ID,
		Filename:       "test.txt",
		FileSize:       1024,
		FileHash:       "abc123",
		TotalChunks:    5,
		UploadedChunks: 0,
		Status:         "pending",
	}
	repo.CreateUploadTask(ctx, task)

	// 更新任务
	task.UploadedChunks = 3
	task.Status = "uploading"
	err := repo.UpdateUploadTask(ctx, task)
	assert.NoError(t, err)

	// 验证更新
	found, _ := repo.GetUploadTask(ctx, "test-upload-id-789")
	assert.Equal(t, 3, found.UploadedChunks)
	assert.Equal(t, "uploading", found.Status)
}

func TestChunkRepository_DeleteUploadTask(t *testing.T) {
	db := setupChunkTestDB(t)
	repo := NewChunkRepository(db)
	ctx := context.Background()

	// 创建测试用户
	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hash",
	}
	db.Create(user)

	// 创建测试任务
	task := &model.UploadTask{
		UploadID:    "test-upload-id-delete",
		UserID:      user.ID,
		Filename:    "test.txt",
		FileSize:    1024,
		FileHash:    "abc123",
		TotalChunks: 5,
	}
	repo.CreateUploadTask(ctx, task)

	// 删除任务
	err := repo.DeleteUploadTask(ctx, "test-upload-id-delete")
	assert.NoError(t, err)

	// 验证删除
	_, err = repo.GetUploadTask(ctx, "test-upload-id-delete")
	assert.Error(t, err)
}

func TestChunkRepository_CreateChunk(t *testing.T) {
	db := setupChunkTestDB(t)
	repo := NewChunkRepository(db)
	ctx := context.Background()

	chunk := &model.FileChunk{
		UploadID:    "test-upload-id",
		FileHash:    "abc123",
		ChunkIndex:  0,
		ChunkHash:   "chunk-hash-0",
		ChunkSize:   1024,
		StoragePath: "/tmp/chunks/chunk-0",
		Status:      "pending",
	}

	err := repo.CreateChunk(ctx, chunk)
	assert.NoError(t, err)
	assert.NotZero(t, chunk.ID)
}

func TestChunkRepository_GetChunk(t *testing.T) {
	db := setupChunkTestDB(t)
	repo := NewChunkRepository(db)
	ctx := context.Background()

	// 创建测试分片
	chunk := &model.FileChunk{
		UploadID:    "test-upload-id-get",
		FileHash:    "abc123",
		ChunkIndex:  1,
		ChunkHash:   "chunk-hash-1",
		ChunkSize:   1024,
		StoragePath: "/tmp/chunks/chunk-1",
		Status:      "pending",
	}
	repo.CreateChunk(ctx, chunk)

	// 测试获取
	found, err := repo.GetChunk(ctx, "test-upload-id-get", 1)
	assert.NoError(t, err)
	assert.Equal(t, chunk.UploadID, found.UploadID)
	assert.Equal(t, chunk.ChunkIndex, found.ChunkIndex)

	// 测试不存在的分片
	_, err = repo.GetChunk(ctx, "test-upload-id-get", 999)
	assert.Error(t, err)
}

func TestChunkRepository_GetChunksByUploadID(t *testing.T) {
	db := setupChunkTestDB(t)
	repo := NewChunkRepository(db)
	ctx := context.Background()

	// 创建多个测试分片
	for i := 0; i < 3; i++ {
		chunk := &model.FileChunk{
			UploadID:    "test-upload-id-list",
			FileHash:    "abc123",
			ChunkIndex:  i,
			ChunkHash:   "chunk-hash-" + string(rune('0'+i)),
			ChunkSize:   1024,
			StoragePath: "/tmp/chunks/chunk-" + string(rune('0'+i)),
			Status:      "pending",
		}
		repo.CreateChunk(ctx, chunk)
	}

	// 测试获取所有分片
	chunks, err := repo.GetChunksByUploadID(ctx, "test-upload-id-list")
	assert.NoError(t, err)
	assert.Len(t, chunks, 3)
}

func TestChunkRepository_GetUploadedChunkIndexes(t *testing.T) {
	db := setupChunkTestDB(t)
	repo := NewChunkRepository(db)
	ctx := context.Background()

	// 创建多个测试分片，部分已上传
	statuses := []string{"uploaded", "pending", "uploaded", "pending"}
	for i, status := range statuses {
		chunk := &model.FileChunk{
			UploadID:    "test-upload-id-indexes",
			FileHash:    "abc123",
			ChunkIndex:  i,
			ChunkHash:   "chunk-hash-" + string(rune('0'+i)),
			ChunkSize:   1024,
			StoragePath: "/tmp/chunks/chunk-" + string(rune('0'+i)),
			Status:      status,
		}
		repo.CreateChunk(ctx, chunk)
	}

	// 测试获取已上传分片索引
	indexes, err := repo.GetUploadedChunkIndexes(ctx, "test-upload-id-indexes")
	assert.NoError(t, err)
	assert.Len(t, indexes, 2)
	assert.Contains(t, indexes, 0)
	assert.Contains(t, indexes, 2)
}

func TestChunkRepository_UpdateChunkStatus(t *testing.T) {
	db := setupChunkTestDB(t)
	repo := NewChunkRepository(db)
	ctx := context.Background()

	// 创建测试分片
	chunk := &model.FileChunk{
		UploadID:    "test-upload-id-status",
		FileHash:    "abc123",
		ChunkIndex:  0,
		ChunkHash:   "chunk-hash-0",
		ChunkSize:   1024,
		StoragePath: "/tmp/chunks/chunk-0",
		Status:      "pending",
	}
	repo.CreateChunk(ctx, chunk)

	// 更新状态
	err := repo.UpdateChunkStatus(ctx, "test-upload-id-status", 0, "uploaded")
	assert.NoError(t, err)

	// 验证更新
	found, _ := repo.GetChunk(ctx, "test-upload-id-status", 0)
	assert.Equal(t, "uploaded", found.Status)
}

func TestChunkRepository_DeleteChunksByUploadID(t *testing.T) {
	db := setupChunkTestDB(t)
	repo := NewChunkRepository(db)
	ctx := context.Background()

	// 创建多个测试分片
	for i := 0; i < 3; i++ {
		chunk := &model.FileChunk{
			UploadID:    "test-upload-id-delete-chunks",
			FileHash:    "abc123",
			ChunkIndex:  i,
			ChunkHash:   "chunk-hash-" + string(rune('0'+i)),
			ChunkSize:   1024,
			StoragePath: "/tmp/chunks/chunk-" + string(rune('0'+i)),
			Status:      "pending",
		}
		repo.CreateChunk(ctx, chunk)
	}

	// 删除所有分片
	err := repo.DeleteChunksByUploadID(ctx, "test-upload-id-delete-chunks")
	assert.NoError(t, err)

	// 验证删除
	chunks, _ := repo.GetChunksByUploadID(ctx, "test-upload-id-delete-chunks")
	assert.Len(t, chunks, 0)
}

func TestChunkRepository_GetFileByHash(t *testing.T) {
	db := setupChunkTestDB(t)
	repo := NewChunkRepository(db)
	ctx := context.Background()

	// 创建测试用户
	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hash",
	}
	db.Create(user)

	// 创建测试文件
	file := &model.File{
		UserID:       user.ID,
		Name:         "test.txt",
		OriginalName: "test.txt",
		Size:         1024,
		MimeType:     "text/plain",
		StoragePath:  "/tmp/files/test.txt",
		Checksum:     "file-hash-123",
	}
	db.Create(file)

	// 测试获取
	found, err := repo.GetFileByHash(ctx, "file-hash-123")
	assert.NoError(t, err)
	assert.Equal(t, file.Checksum, found.Checksum)

	// 测试不存在的文件
	_, err = repo.GetFileByHash(ctx, "not-exist")
	assert.Error(t, err)
}
