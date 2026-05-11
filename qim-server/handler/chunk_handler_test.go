package handler

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"qim-server/config"
	"qim-server/database"
	"qim-server/di"
	"qim-server/model"
	"qim-server/pkg/response"
	"qim-server/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupChunkHandlerTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(
		&model.User{},
		&model.File{},
		&model.Folder{},
		&model.UploadTask{},
		&model.FileChunk{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func setupChunkHandlerTestRouter(t *testing.T) (*gin.Engine, *gorm.DB, string) {
	gin.SetMode(gin.TestMode)
	db := setupChunkHandlerTestDB(t)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key",
			Expire: 3600,
		},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"*"},
		},
		Storage: config.StorageConfig{
			Type: "local",
			Local: config.LocalStorageConfig{
				Path: t.TempDir(),
			},
		},
	}

	// 初始化 DI 容器
	di.InitContainer(cfg, nil)
	database.DB = db
	SetConfig(cfg)

	// 注册 ChunkService 到 DI 容器
	tempDir := t.TempDir()
	chunkService := service.NewChunkService(db, tempDir)
	di.GlobalContainer.ChunkService = chunkService

	r := gin.New()

	// 认证中间件
	authMiddleware := func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("username", "testuser")
		c.Set("roles", []string{})
		c.Next()
	}

	authed := r.Group("/api/v1")
	authed.Use(authMiddleware)
	{
		authed.POST("/files/upload/init", InitUpload)
		authed.POST("/files/upload/chunk", UploadChunk)
		authed.POST("/files/upload/complete", CompleteUpload)
		authed.POST("/files/upload/cancel", CancelUpload)
	}

	return r, db, tempDir
}

func createChunkTestUser(t *testing.T, db *gorm.DB) *model.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &model.User{
		Username:     "testuser",
		PasswordHash: string(hash),
		Nickname:     "Test User",
		Status:       "offline",
	}
	db.Create(user)
	return user
}

// Test 1: InitUpload - 初始化上传
func TestInitUpload_Success(t *testing.T) {
	r, db, _ := setupChunkHandlerTestRouter(t)
	createChunkTestUser(t, db)

	body := map[string]interface{}{
		"filename":  "test.txt",
		"file_size": 15 * 1024 * 1024,
		"file_hash": "test-hash-123",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/files/upload/init", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 0, resp.Code)

	// 验证返回的数据包含 upload_id
	data, ok := resp.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.NotEmpty(t, data["upload_id"])
	assert.Equal(t, float64(3), data["total_chunks"]) // 15MB / 5MB = 3 chunks
}

func TestInitUpload_MissingFields(t *testing.T) {
	r, db, _ := setupChunkHandlerTestRouter(t)
	createChunkTestUser(t, db)

	body := map[string]interface{}{
		"filename": "test.txt",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/files/upload/init", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInitUpload_WithFolder(t *testing.T) {
	r, db, _ := setupChunkHandlerTestRouter(t)
	user := createChunkTestUser(t, db)

	// 创建文件夹
	folder := &model.Folder{
		UserID: user.ID,
		Name:   "test-folder",
	}
	db.Create(folder)

	body := map[string]interface{}{
		"filename":  "test.txt",
		"file_size": 10 * 1024 * 1024,
		"file_hash": "test-hash-folder",
		"folder_id": folder.ID,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/files/upload/init", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 0, resp.Code)
}

// Test 2: UploadChunk - 上传分片
func TestUploadChunk_Success(t *testing.T) {
	r, db, tempDir := setupChunkHandlerTestRouter(t)
	createChunkTestUser(t, db)

	// 先初始化上传
	chunkService := service.NewChunkService(db, tempDir)
	task, _, _, err := chunkService.InitUpload(1, "test.txt", 15*1024*1024, "test-hash-upload", nil)
	assert.NoError(t, err)

	// 准备分片数据
	chunkData := make([]byte, 5*1024*1024)
	for i := range chunkData {
		chunkData[i] = byte(i % 256)
	}
	hash := md5.Sum(chunkData)
	chunkHash := hex.EncodeToString(hash[:])

	// 创建 multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加 upload_id 字段
	_ = writer.WriteField("upload_id", task.UploadID)

	// 添加 chunk_index 字段
	_ = writer.WriteField("chunk_index", "0")

	// 添加 chunk_hash 字段
	_ = writer.WriteField("chunk_hash", chunkHash)

	// 添加文件字段
	part, _ := writer.CreateFormFile("chunk", "chunk-0")
	io.Copy(part, bytes.NewReader(chunkData))

	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/files/upload/chunk", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 0, resp.Code)
}

func TestUploadChunk_InvalidUploadID(t *testing.T) {
	r, db, _ := setupChunkHandlerTestRouter(t)
	createChunkTestUser(t, db)

	// 准备分片数据
	chunkData := make([]byte, 5*1024*1024)
	hash := md5.Sum(chunkData)
	chunkHash := hex.EncodeToString(hash[:])

	// 创建 multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	_ = writer.WriteField("upload_id", "invalid-upload-id")
	_ = writer.WriteField("chunk_index", "0")
	_ = writer.WriteField("chunk_hash", chunkHash)

	part, _ := writer.CreateFormFile("chunk", "chunk-0")
	io.Copy(part, bytes.NewReader(chunkData))

	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/files/upload/chunk", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// Test 3: CompleteUpload - 完成上传
func TestCompleteUpload_Success(t *testing.T) {
	r, db, tempDir := setupChunkHandlerTestRouter(t)
	createChunkTestUser(t, db)

	// 初始化并上传所有分片
	chunkService := service.NewChunkService(db, tempDir)
	task, _, _, err := chunkService.InitUpload(1, "test.txt", 15*1024*1024, "test-hash-complete", nil)
	assert.NoError(t, err)

	// 上传所有分片
	for i := 0; i < task.TotalChunks; i++ {
		chunkData := make([]byte, 5*1024*1024)
		for j := range chunkData {
			chunkData[j] = byte((i*256 + j) % 256)
		}
		hash := md5.Sum(chunkData)
		chunkHash := hex.EncodeToString(hash[:])
		err = chunkService.UploadChunk(task.UploadID, i, chunkData, chunkHash)
		assert.NoError(t, err)
	}

	// 完成上传
	body := map[string]interface{}{
		"upload_id": task.UploadID,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/files/upload/complete", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 0, resp.Code)

	// 验证返回的文件信息
	data, ok := resp.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.NotEmpty(t, data["id"])
	assert.Equal(t, "test.txt", data["name"])
}

func TestCompleteUpload_IncompleteChunks(t *testing.T) {
	r, db, tempDir := setupChunkHandlerTestRouter(t)
	createChunkTestUser(t, db)

	// 初始化但不上传所有分片
	chunkService := service.NewChunkService(db, tempDir)
	task, _, _, err := chunkService.InitUpload(1, "test.txt", 15*1024*1024, "test-hash-incomplete", nil)
	assert.NoError(t, err)

	// 只上传一个分片
	chunkData := make([]byte, 5*1024*1024)
	hash := md5.Sum(chunkData)
	chunkHash := hex.EncodeToString(hash[:])
	err = chunkService.UploadChunk(task.UploadID, 0, chunkData, chunkHash)
	assert.NoError(t, err)

	// 尝试完成上传
	body := map[string]interface{}{
		"upload_id": task.UploadID,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/files/upload/complete", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// Test 4: CancelUpload - 取消上传
func TestCancelUpload_Success(t *testing.T) {
	r, db, tempDir := setupChunkHandlerTestRouter(t)
	createChunkTestUser(t, db)

	// 初始化上传
	chunkService := service.NewChunkService(db, tempDir)
	task, _, _, err := chunkService.InitUpload(1, "test.txt", 15*1024*1024, "test-hash-cancel", nil)
	assert.NoError(t, err)

	// 取消上传
	body := map[string]interface{}{
		"upload_id": task.UploadID,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/files/upload/cancel", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 0, resp.Code)
}

func TestCancelUpload_InvalidUploadID(t *testing.T) {
	r, db, _ := setupChunkHandlerTestRouter(t)
	createChunkTestUser(t, db)

	body := map[string]interface{}{
		"upload_id": "invalid-upload-id",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/files/upload/cancel", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
