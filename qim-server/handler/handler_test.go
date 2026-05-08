package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"qim-server/config"
	"qim-server/database"
	"qim-server/di"
	"qim-server/model"
	"qim-server/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupHandlerTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(
		&model.User{},
		&model.UserRole{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.Message{},
		&model.Notification{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func setupTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)
	db := setupHandlerTestDB(t)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key",
			Expire: 3600,
		},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"*"},
		},
	}

	di.InitContainer(cfg, nil)

	database.DB = db
	SetConfig(cfg)

	r := gin.New()
	r.POST("/api/v1/auth/login", Login)
	r.POST("/api/v1/auth/register", Register)

	authMiddleware := func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("username", "testuser")
		c.Set("roles", []string{})
		c.Next()
	}

	authed := r.Group("/api/v1")
	authed.Use(authMiddleware)
	{
		authed.GET("/users/me", GetCurrentUser)
		authed.PUT("/users/me", UpdateUser)
	}

	return r, db
}

func createTestUser(t *testing.T, db *gorm.DB) *model.User {
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

func TestLogin_Success(t *testing.T) {
	r, db := setupTestRouter(t)
	createTestUser(t, db)

	body := map[string]string{
		"username": "testuser",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 0, resp.Code)
}

func TestLogin_InvalidPassword(t *testing.T) {
	r, db := setupTestRouter(t)
	createTestUser(t, db)

	body := map[string]string{
		"username": "testuser",
		"password": "wrongpassword",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogin_UserNotFound(t *testing.T) {
	r, _ := setupTestRouter(t)

	body := map[string]string{
		"username": "nonexistent",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogin_MissingFields(t *testing.T) {
	r, _ := setupTestRouter(t)

	body := map[string]string{
		"username": "testuser",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegister_Success(t *testing.T) {
	r, _ := setupTestRouter(t)

	body := map[string]string{
		"username": "newuser",
		"password": "password123",
		"nickname": "New User",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRegister_DuplicateUsername(t *testing.T) {
	r, db := setupTestRouter(t)
	createTestUser(t, db)

	body := map[string]string{
		"username": "testuser",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetCurrentUser_Success(t *testing.T) {
	r, db := setupTestRouter(t)
	createTestUser(t, db)

	req := httptest.NewRequest("GET", "/api/v1/users/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 0, resp.Code)
}

func TestUpdateUser_Success(t *testing.T) {
	r, db := setupTestRouter(t)
	createTestUser(t, db)

	body := map[string]string{
		"nickname": "Updated Name",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/api/v1/users/me", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
