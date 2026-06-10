package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupVersionUpdateTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(
		&model.File{},
		&model.ClientVersion{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	database.DB = db
	return db
}

func TestCreateVersion_RejectsExternalURLWithoutMetadata(t *testing.T) {
	setupVersionUpdateTestDB(t)
	gin.SetMode(gin.TestMode)

	body := map[string]any{
		"version":     "2.0.0",
		"platform":    "windows",
		"downloadUrl": "https://cdn.example.com/QIM-2.0.0.exe",
		"updateNotes": "test release",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/versions", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	CreateVersion(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp.Message, "SHA512")

	var count int64
	database.GetDB().Model(&model.ClientVersion{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestGetLatestYML_UsesAbsoluteURLAndFilenamePathForPublicFileDownload(t *testing.T) {
	db := setupVersionUpdateTestDB(t)
	gin.SetMode(gin.TestMode)

	file := model.File{
		ID:           1,
		Name:         "QIM-2.0.0.dmg",
		OriginalName: "QIM-2.0.0.dmg",
		StoragePath:  "/uploads/updates/QIM-2.0.0.dmg",
		Size:         123456,
	}
	err := db.Create(&file).Error
	assert.NoError(t, err)

	version := model.ClientVersion{
		Version:     "2.0.0",
		Platform:    "macos",
		DownloadURL: "http://localhost:8080/api/v1/public/files/1/download",
		Sha512:      strings.Repeat("a", 128),
		FileSize:    123456,
		Enabled:     true,
	}
	err = db.Create(&version).Error
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "platform", Value: "mac"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/updates/mac/latest-mac.yml", nil)

	GetLatestYML(c)

	assert.Equal(t, http.StatusOK, w.Code)
	yml := w.Body.String()
	assert.NotContains(t, yml, "../../public/files/1/download")
	assert.NotContains(t, yml, "http://localhost:8080/api/v1/public/files/1/download")
	assert.Contains(t, yml, "url: QIM-2.0.0.dmg")
	assert.Contains(t, yml, "path: QIM-2.0.0.dmg")
}

func TestHandleUpdateRequest_RedirectsInstallerFilenameToDownloadURL(t *testing.T) {
	db := setupVersionUpdateTestDB(t)
	gin.SetMode(gin.TestMode)

	file := model.File{
		ID:           1,
		Name:         "QIM-2.0.0.dmg",
		OriginalName: "QIM-2.0.0.dmg",
		StoragePath:  "/uploads/updates/QIM-2.0.0.dmg",
		Size:         123456,
	}
	assert.NoError(t, db.Create(&file).Error)

	version := model.ClientVersion{
		Version:     "2.0.0",
		Platform:    "macos",
		DownloadURL: "http://localhost:8080/api/v1/public/files/1/download",
		Sha512:      strings.Repeat("a", 128),
		FileSize:    123456,
		Enabled:     true,
	}
	assert.NoError(t, db.Create(&version).Error)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "platform", Value: "mac"},
		{Key: "action", Value: "/QIM-2.0.0.dmg"},
	}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/updates/mac/QIM-2.0.0.dmg", nil)

	HandleUpdateRequest(c)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "http://localhost:8080/api/v1/public/files/1/download", w.Header().Get("Location"))
}

func TestGetLatestYML_RejectsIncompleteEnabledVersion(t *testing.T) {
	db := setupVersionUpdateTestDB(t)
	gin.SetMode(gin.TestMode)

	version := model.ClientVersion{
		Version:     "2.0.0",
		Platform:    "windows",
		DownloadURL: "https://cdn.example.com/QIM-2.0.0.exe",
		Enabled:     true,
	}
	err := db.Create(&version).Error
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "platform", Value: "win"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/updates/win/latest.yml", nil)

	GetLatestYML(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Empty(t, w.Body.String())
}
