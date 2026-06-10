package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	err = db.AutoMigrate(&model.User{}, &model.UserRole{}, &model.OperationLog{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	// 设置全局 DB，供 OperationLogMiddleware 使用
	database.DB = db
	return db
}

func TestRequireRole_AllowsAuthorizedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupAuthTestDB(t)

	user := &model.User{Username: "admin", Nickname: "Admin", Status: "online"}
	db.Create(user)
	db.Create(&model.UserRole{UserID: user.ID, Role: "system_admin"})

	userSvc := service.NewUserService(db)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", user.ID)
		c.Set("username", "admin")
		c.Next()
	})
	r.GET("/admin/config", RequireRole(userSvc, "system_admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0})
	})

	req := httptest.NewRequest("GET", "/admin/config", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireRole_BlocksUnauthorizedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupAuthTestDB(t)

	user := &model.User{Username: "normal", Nickname: "Normal", Status: "online"}
	db.Create(user)

	userSvc := service.NewUserService(db)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", user.ID)
		c.Set("username", "normal")
		c.Next()
	})
	r.GET("/admin/config", RequireRole(userSvc, "system_admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0})
	})

	req := httptest.NewRequest("GET", "/admin/config", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAdminRoutes_RequireAdminRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupAuthTestDB(t)

	adminUser := &model.User{Username: "admin", Nickname: "Admin", Status: "online"}
	db.Create(adminUser)
	db.Create(&model.UserRole{UserID: adminUser.ID, Role: "system_admin"})

	normalUser := &model.User{Username: "normal", Nickname: "Normal", Status: "online"}
	db.Create(normalUser)

	userSvc := service.NewUserService(db)

	// 模拟 adminRoutes 的中间件链：认证 + 操作日志 + 角色校验
	setupRouter := func(userID uint) *gin.Engine {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Set("username", "test")
			c.Next()
		})
		r.Use(OperationLogMiddleware())
		r.Use(RequireRole(userSvc, "system_admin"))
		r.GET("/system/config", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"code": 0})
		})
		return r
	}

	// 普通用户应被拒绝
	r := setupRouter(normalUser.ID)
	req := httptest.NewRequest("GET", "/system/config", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// 管理员应被允许
	r2 := setupRouter(adminUser.ID)
	req2 := httptest.NewRequest("GET", "/system/config", nil)
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

// TestAdminRoutes_WithoutRequireRole_AllowsNormalUser 验证：
// adminRoutes 必须添加 RequireRole，否则普通用户可以访问管理接口。
// 此测试模拟修复后的行为：adminRoutes 添加了 RequireRole，普通用户应被拒绝。
func TestAdminRoutes_WithoutRequireRole_AllowsNormalUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupAuthTestDB(t)

	normalUser := &model.User{Username: "normal", Nickname: "Normal", Status: "online"}
	db.Create(normalUser)

	userSvc := service.NewUserService(db)

	// 模拟修复后的 adminRoutes 中间件链：操作日志 + RequireRole
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", normalUser.ID)
		c.Set("username", "normal")
		c.Next()
	})
	r.Use(OperationLogMiddleware())
	r.Use(RequireRole(userSvc, "system_admin"))
	r.GET("/system/config", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "sensitive"})
	})

	req := httptest.NewRequest("GET", "/system/config", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 修复后：普通用户应被拒绝（返回 403）
	assert.Equal(t, http.StatusForbidden, w.Code,
		"adminRoutes should block normal users without system_admin role")
}

// --- S4: 移除 URL Query 传 Token ---

// generateTestToken 生成一个有效的 JWT token 用于测试
func generateTestToken(secret string, userID uint, username string) string {
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func Test_AuthMiddleware_RejectsQueryToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupAuthTestDB(t)

	user := &model.User{Username: "testuser", Nickname: "Test", Status: "online"}
	db.Create(user)

	secret := "test-secret-key-for-auth-middleware"
	userSvc := service.NewUserService(db)

	r := gin.New()
	r.Use(AuthMiddleware(secret, userSvc))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0})
	})

	// 通过 URL query 传递 token，应该被拒绝（返回 401）
	tokenString := generateTestToken(secret, user.ID, "testuser")
	req := httptest.NewRequest("GET", "/protected?token="+tokenString, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"Auth middleware should reject token passed via URL query parameter")
}

func Test_AuthMiddleware_AcceptsBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupAuthTestDB(t)

	user := &model.User{Username: "testuser", Nickname: "Test", Status: "online"}
	db.Create(user)

	secret := "test-secret-key-for-auth-middleware"
	userSvc := service.NewUserService(db)

	r := gin.New()
	r.Use(AuthMiddleware(secret, userSvc))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0})
	})

	// 通过 Authorization header 传递 Bearer token，应该被接受
	tokenString := generateTestToken(secret, user.ID, "testuser")
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code,
		"Auth middleware should accept token passed via Authorization Bearer header")
}

func Test_AuthMiddleware_RejectsQueryTokenForWebSocket(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupAuthTestDB(t)

	user := &model.User{Username: "wsuser", Nickname: "WS", Status: "online"}
	db.Create(user)

	secret := "test-secret-key-for-auth-middleware"
	userSvc := service.NewUserService(db)

	r := gin.New()
	r.Use(AuthMiddleware(secret, userSvc))
	r.GET("/ws", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0})
	})

	// WebSocket 连接通过 URL query 传递 token，也应该被拒绝
	tokenString := generateTestToken(secret, user.ID, "wsuser")
	req := httptest.NewRequest("GET", "/ws?token="+tokenString, nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"Auth middleware should reject token via URL query even for WebSocket connections")
}

// --- S8: 节点通信接口添加内部认证 ---

func Test_NodeAuthMiddleware_RejectsWithoutSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(NodeAuthMiddleware("test-node-secret"))
	r.POST("/node/broadcast", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0})
	})

	req := httptest.NewRequest("POST", "/node/broadcast", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"NodeAuthMiddleware should reject request without Node-Secret header")
}

func Test_NodeAuthMiddleware_RejectsWrongSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(NodeAuthMiddleware("test-node-secret"))
	r.POST("/node/broadcast", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0})
	})

	req := httptest.NewRequest("POST", "/node/broadcast", nil)
	req.Header.Set("Node-Secret", "wrong-secret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"NodeAuthMiddleware should reject request with wrong Node-Secret header")
}

func Test_NodeAuthMiddleware_AcceptsValidSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(NodeAuthMiddleware("test-node-secret"))
	r.POST("/node/broadcast", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0})
	})

	req := httptest.NewRequest("POST", "/node/broadcast", nil)
	req.Header.Set("Node-Secret", "test-node-secret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code,
		"NodeAuthMiddleware should accept request with correct Node-Secret header")
}
