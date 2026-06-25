package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/config"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/dshmyz/qim/qim-server/service/storage"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupFileHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(
		&model.User{},
		&model.UserRole{},
		&model.File{},
		&model.Folder{},
		&model.SystemConfig{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func setupFileHandlerDI(t *testing.T, db *gorm.DB) {
	t.Helper()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key",
			Expire: 3600,
		},
		Storage: config.StorageConfig{
			Type: "local",
			Local: config.LocalStorageConfig{
				Path: t.TempDir(),
			},
		},
	}

	di.InitContainer(cfg, nil)
	database.DB = db
	SetConfig(cfg)

	// 初始化本地存储
	localStorage, err := storage.NewLocalStorage(cfg.Storage.Local)
	if err != nil {
		t.Fatalf("failed to create local storage: %v", err)
	}
	di.GlobalContainer.DefaultStorage = localStorage
	di.GlobalContainer.StorageManager = storage.NewManager(localStorage)
	di.GlobalContainer.FileService = service.NewFileService(db)
	di.GlobalContainer.SystemConfigService = service.NewSystemConfigService(db)
}

// --- S5: PublicDownloadFile 添加访问控制 ---

func Test_PublicDownloadFile_BlocksPrivateFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupFileHandlerTestDB(t)
	setupFileHandlerDI(t, db)

	// 创建一个 source='upload' 的私有文件
	privateFile := model.File{
		Name:         "private.txt",
		OriginalName: "private.txt",
		StoragePath:  "/uploads/private.txt",
		Size:         100,
		MimeType:     "text/plain",
		UserID:       1,
		Source:       "upload",
		CreatedAt:    time.Now(),
	}
	db.Create(&privateFile)

	r := gin.New()
	r.GET("/public/files/:id", PublicDownloadFile)

	req := httptest.NewRequest("GET", "/public/files/"+strconv.FormatUint(uint64(privateFile.ID), 10), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code,
		"PublicDownloadFile should block download of private (source=upload) files")
}

func Test_PublicDownloadFile_AllowsPublicFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupFileHandlerTestDB(t)
	setupFileHandlerDI(t, db)

	// 创建一个 source='client_update' 的公开文件，同时创建对应的物理文件
	publicFile := model.File{
		Name:         "app-v1.0.exe",
		OriginalName: "app-v1.0.exe",
		StoragePath:  storage.BuildPath("local", "uploads/app-v1.0.exe"),
		Size:         5,
		MimeType:     "application/octet-stream",
		UserID:       1,
		Source:       "client_update",
		CreatedAt:    time.Now(),
	}
	db.Create(&publicFile)

	// 创建物理文件
	st := di.GlobalContainer.DefaultStorage
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = st.Put(ctx, "uploads/app-v1.0.exe", bytes.NewReader([]byte("hello")), 5, "application/octet-stream")

	r := gin.New()
	r.GET("/public/files/:id", PublicDownloadFile)

	req := httptest.NewRequest("GET", "/public/files/"+strconv.FormatUint(uint64(publicFile.ID), 10), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code,
		"PublicDownloadFile should allow download of public (source=client_update) files")
}

func Test_PublicDownloadFile_AllowsVersionSourceFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupFileHandlerTestDB(t)
	setupFileHandlerDI(t, db)

	// 创建一个 source='version' 的公开文件，同时创建对应的物理文件
	versionFile := model.File{
		Name:         "app-v2.0.exe",
		OriginalName: "app-v2.0.exe",
		StoragePath:  storage.BuildPath("local", "uploads/app-v2.0.exe"),
		Size:         5,
		MimeType:     "application/octet-stream",
		UserID:       1,
		Source:       "version",
		CreatedAt:    time.Now(),
	}
	db.Create(&versionFile)

	// 创建物理文件
	st := di.GlobalContainer.DefaultStorage
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = st.Put(ctx, "uploads/app-v2.0.exe", bytes.NewReader([]byte("hello")), 5, "application/octet-stream")

	r := gin.New()
	r.GET("/public/files/:id", PublicDownloadFile)

	req := httptest.NewRequest("GET", "/public/files/"+strconv.FormatUint(uint64(versionFile.ID), 10), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code,
		"PublicDownloadFile should allow download of public (source=version) files")
}

func Test_PublicDownloadFile_BlocksNonExistentFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupFileHandlerTestDB(t)
	setupFileHandlerDI(t, db)

	r := gin.New()
	r.GET("/public/files/:id", PublicDownloadFile)

	req := httptest.NewRequest("GET", "/public/files/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code,
		"PublicDownloadFile should return 404 for non-existent file ID")
}

// --- S6: Content-Disposition 文件名编码 ---

func Test_DownloadFile_EscapesSpecialCharsInFilename(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupFileHandlerTestDB(t)
	setupFileHandlerDI(t, db)

	// 创建文件名为包含双引号的文件
	specialFile := model.File{
		Name:         `test"file.txt`,
		OriginalName: `test"file.txt`,
		StoragePath:  storage.BuildPath("local", "uploads/special.txt"),
		Size:         5,
		MimeType:     "text/plain",
		UserID:       1,
		Source:       "upload",
		CreatedAt:    time.Now(),
	}
	db.Create(&specialFile)

	// 创建物理文件
	st := di.GlobalContainer.DefaultStorage
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = st.Put(ctx, "uploads/special.txt", bytes.NewReader([]byte("hello")), 5, "text/plain")

	r := gin.New()
	// 模拟认证中间件
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("username", "testuser")
		c.Next()
	})
	r.GET("/files/:id/download", DownloadFile)

	req := httptest.NewRequest("GET", "/files/"+strconv.FormatUint(uint64(specialFile.ID), 10)+"/download", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	contentDisposition := w.Header().Get("Content-Disposition")
	// 确保没有未转义的双引号（即不能出现 filename="test"file.txt" 这种情况）
	// 正确的做法是使用 RFC 5987 编码或转义双引号
	assert.NotContains(t, contentDisposition, `filename="test"file.txt"`,
		"Content-Disposition should not contain unescaped double quotes in filename")
}

func Test_DownloadFile_EncodesChineseFilename(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupFileHandlerTestDB(t)
	setupFileHandlerDI(t, db)

	// 创建中文文件名的文件
	chineseFile := model.File{
		Name:         "测试文件.pdf",
		OriginalName: "测试文件.pdf",
		StoragePath:  storage.BuildPath("local", "uploads/chinese.pdf"),
		Size:         5,
		MimeType:     "application/pdf",
		UserID:       1,
		Source:       "upload",
		CreatedAt:    time.Now(),
	}
	db.Create(&chineseFile)

	// 创建物理文件
	st := di.GlobalContainer.DefaultStorage
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = st.Put(ctx, "uploads/chinese.pdf", bytes.NewReader([]byte("hello")), 5, "application/pdf")

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("username", "testuser")
		c.Next()
	})
	r.GET("/files/:id/download", DownloadFile)

	req := httptest.NewRequest("GET", "/files/"+strconv.FormatUint(uint64(chineseFile.ID), 10)+"/download", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	contentDisposition := w.Header().Get("Content-Disposition")
	// 应该使用 RFC 5987 编码（filename*=UTF-8''）
	assert.Contains(t, contentDisposition, "filename*=UTF-8''",
		"Content-Disposition should use RFC 5987 encoding for Chinese filename")
}

// --- S6: sanitizeFilename 单元测试 ---

func Test_SanitizeFilename_EscapesSpecialChars(t *testing.T) {
	// 文件名含双引号，输出不应包含未转义双引号
	result := sanitizeFilename(`test"file.txt`)
	assert.NotContains(t, result, `filename="test"file.txt"`,
		"sanitizeFilename should not produce unescaped double quotes in filename")
	// 应包含 filename*=UTF-8'' 编码
	assert.Contains(t, result, "filename*=UTF-8''",
		"sanitizeFilename should include RFC 5987 encoded filename for special chars")
}

func Test_SanitizeFilename_EscapesChineseFilename(t *testing.T) {
	// 中文文件名，应包含 filename*=UTF-8'' 编码
	result := sanitizeFilename("测试文件.pdf")
	assert.Contains(t, result, "filename*=UTF-8''",
		"sanitizeFilename should include RFC 5987 encoded filename for Chinese characters")
	// 也应包含 fallback 的 filename=
	assert.Contains(t, result, `filename="`,
		"sanitizeFilename should include fallback filename for ASCII-only clients")
}

func Test_SanitizeFilename_PlainFilename(t *testing.T) {
	// 普通文件名，应正常输出
	result := sanitizeFilename("report.pdf")
	assert.Contains(t, result, `filename="report.pdf"`,
		"sanitizeFilename should include plain filename for normal names")
	assert.Contains(t, result, "filename*=UTF-8''",
		"sanitizeFilename should still include RFC 5987 encoded filename")
}
