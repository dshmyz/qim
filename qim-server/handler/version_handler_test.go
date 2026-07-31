package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUpdateCLIVersionPersistsBinaryFields(t *testing.T) {
	db := setupHandlerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.ClientVersion{}))
	database.DB = db

	svc := service.NewVersionService(db, nil)
	created, err := svc.Create(service.CreateVersionInput{
		Version:     "1.2.0",
		AppType:     "cli",
		Os:          "darwin",
		Arch:        "arm64",
		DownloadURL: "/api/v1/public/files/1/download",
		Sha256:      "old-sha",
		FileSize:    100,
	})
	require.NoError(t, err)

	body := bytes.NewBufferString(`{"downloadUrl":"/api/v1/public/files/2/download","sha256":"new-sha","fileSize":200}`)
	req := httptest.NewRequest(http.MethodPut, "/v1/admin/cli/versions/1", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PUT("/v1/admin/cli/versions/:id", UpdateCLIVersion)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated model.ClientVersion
	require.NoError(t, db.First(&updated, created.ID).Error)
	if updated.DownloadURL != "/api/v1/public/files/2/download" {
		t.Fatalf("download URL was not updated: %q", updated.DownloadURL)
	}
	if updated.Sha256 != "new-sha" {
		t.Fatalf("sha256 was not updated: %q", updated.Sha256)
	}
	if updated.FileSize != 200 {
		t.Fatalf("file size was not updated: %d", updated.FileSize)
	}

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
}
