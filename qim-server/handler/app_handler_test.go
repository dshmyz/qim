package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAppHandlerTestDB(t *testing.T) *gorm.DB {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.App{}))
	di.GlobalContainer = &di.Container{AppService: service.NewAppService(db)}
	return db
}

// newAppCtx 构造带 user_id/roles 上下文和可选 JSON 请求体的 gin 测试上下文。
// 路径末尾为数字时（如 /api/v1/apps/1）自动挂到 :id 路由参数。
func newAppCtx(w *httptest.ResponseRecorder, method, path string, body interface{}, userID uint, roles []string) *gin.Context {
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	if body != nil {
		b, _ := json.Marshal(body)
		c.Request = httptest.NewRequest(method, path, bytes.NewReader(b))
		c.Request.Header.Set("Content-Type", "application/json")
	}
	c.Set("user_id", userID)
	c.Set("roles", roles)
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if last := segments[len(segments)-1]; last != "" {
		if _, err := strconv.ParseUint(last, 10, 32); err == nil {
			c.Params = gin.Params{{Key: "id", Value: last}}
		}
	}
	return c
}

// TestCreateApp_GlobalRequiresAdmin 只有 system_admin 才能创建全局应用
func TestCreateApp_GlobalRequiresAdmin(t *testing.T) {
	db := setupAppHandlerTestDB(t)

	// 非管理员创建全局应用 → 403
	w := httptest.NewRecorder()
	c := newAppCtx(w, http.MethodPost, "/api/v1/apps", map[string]interface{}{
		"name": "全局应用", "category": "工具", "url": "https://g.com",
		"openType": "in-app", "isGlobal": true,
	}, uint(1), []string{"user"})
	CreateApp(c)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// 管理员创建全局应用 → 成功且 is_global=true
	w2 := httptest.NewRecorder()
	c2 := newAppCtx(w2, http.MethodPost, "/api/v1/apps", map[string]interface{}{
		"name": "全局应用", "category": "工具", "url": "https://g.com",
		"openType": "in-app", "isGlobal": true, "scopeType": "all",
	}, uint(1), []string{"system_admin"})
	CreateApp(c2)
	assert.Equal(t, http.StatusOK, w2.Code)
	var app model.App
	require.NoError(t, db.First(&app).Error)
	assert.True(t, app.IsGlobal)
}

// TestUpdateApp_AdminEditsOthersPersonalApp 管理员可以编辑他人的个人应用
func TestUpdateApp_AdminEditsOthersPersonalApp(t *testing.T) {
	db := setupAppHandlerTestDB(t)
	require.NoError(t, db.Create(&model.App{UserID: 2, Name: "别人的应用", URL: "https://b.com"}).Error)

	w := httptest.NewRecorder()
	c := newAppCtx(w, http.MethodPut, "/api/v1/apps/1", map[string]interface{}{
		"name": "改名后的应用",
	}, uint(1), []string{"system_admin"})
	UpdateApp(c)
	assert.Equal(t, http.StatusOK, w.Code)

	var app model.App
	require.NoError(t, db.First(&app, 1).Error)
	assert.Equal(t, "改名后的应用", app.Name)
}

// TestUpdateApp_NonAdminCannotEditOthersPersonalApp 非管理员编辑他人个人应用 → 404
func TestUpdateApp_NonAdminCannotEditOthersPersonalApp(t *testing.T) {
	db := setupAppHandlerTestDB(t)
	require.NoError(t, db.Create(&model.App{UserID: 2, Name: "别人的应用", URL: "https://b.com"}).Error)

	w := httptest.NewRecorder()
	c := newAppCtx(w, http.MethodPut, "/api/v1/apps/1", map[string]interface{}{
		"name": "改名",
	}, uint(1), []string{"user"})
	UpdateApp(c)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestDeleteApp_AdminDeletesOthersApp 管理员可删除他人的个人应用
func TestDeleteApp_AdminDeletesOthersApp(t *testing.T) {
	db := setupAppHandlerTestDB(t)
	require.NoError(t, db.Create(&model.App{UserID: 2, Name: "别人的应用", URL: "https://b.com"}).Error)

	w := httptest.NewRecorder()
	c := newAppCtx(w, http.MethodDelete, "/api/v1/apps/1", nil, uint(1), []string{"system_admin"})
	DeleteApp(c)
	assert.Equal(t, http.StatusOK, w.Code)

	var count int64
	db.Model(&model.App{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

// TestDeleteApp_NonAdminDeletesOthersApp 非管理员删除他人个人应用 → 404 且数据保留
func TestDeleteApp_NonAdminDeletesOthersApp(t *testing.T) {
	db := setupAppHandlerTestDB(t)
	require.NoError(t, db.Create(&model.App{UserID: 2, Name: "别人的应用", URL: "https://b.com"}).Error)

	w := httptest.NewRecorder()
	c := newAppCtx(w, http.MethodDelete, "/api/v1/apps/1", nil, uint(1), []string{"user"})
	DeleteApp(c)
	assert.Equal(t, http.StatusNotFound, w.Code)

	var count int64
	db.Model(&model.App{}).Count(&count)
	assert.Equal(t, int64(1), count)
}
