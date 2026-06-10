package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/dshmyz/qim/qim-server/config"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupUserHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(
		&model.User{},
		&model.UserRole{},
		&model.Department{},
		&model.DepartmentEmployee{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func setupUserHandlerDI(t *testing.T, db *gorm.DB) {
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

	// 先设置 database.DB，这样 InitContainer 内部的 database.GetDB() 会使用我们的内存数据库
	database.DB = db
	di.InitContainer(cfg, nil)
	// 确保 GlobalContainer.DB 也指向我们的内存数据库
	di.GlobalContainer.DB = db
	SetConfig(cfg)
}

// --- S10: GetUserByID 返回 IP 字段（内网部署需要） ---

func Test_GetUserByID_ReturnsIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupUserHandlerTestDB(t)
	setupUserHandlerDI(t, db)

	// 创建一个带有 IP 的用户
	user := model.User{
		Username: "testuser",
		Nickname: "TestUser",
		IP:       "192.168.1.100",
		Status:   "online",
	}
	db.Create(&user)

	r := gin.New()
	r.GET("/users/:id", GetUserByID)

	req := httptest.NewRequest("GET", "/users/"+uintToStr(user.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 解析响应 JSON，断言包含 ip 字段
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err, "response should be valid JSON")

	data, ok := resp["data"].(map[string]interface{})
	assert.True(t, ok, "response should contain data field")

	ipValue, hasIP := data["ip"]
	assert.True(t, hasIP, "GetUserByID response should contain 'ip' field for intranet deployment")
	assert.Equal(t, "192.168.1.100", ipValue, "IP should match the user's IP")
}

// uintToStr 将 uint 转为字符串
func uintToStr(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}
