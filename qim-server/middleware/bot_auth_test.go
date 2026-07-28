package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupBotAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Bot{}, &model.BotToken{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	// BotAuthMiddleware 通过 database.GetDB() 取全局 DB
	database.DB = db
	return db
}

func newBotAuthRouter() *gin.Engine {
	r := gin.New()
	r.Use(BotAuthMiddleware())
	r.POST("/bot/messages", func(c *gin.Context) {
		botID, _ := c.Get("bot_id")
		c.JSON(http.StatusOK, gin.H{"code": 0, "bot_id": botID})
	})
	return r
}

func TestHashBotToken_Deterministic(t *testing.T) {
	h1 := HashBotToken("qbot_abc")
	h2 := HashBotToken("qbot_abc")
	h3 := HashBotToken("qbot_other")
	assert.Equal(t, h1, h2)
	assert.NotEqual(t, h1, h3)
	assert.Len(t, h1, 64) // sha256 hex
}

func TestBotAuthMiddleware_AcceptsValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupBotAuthTestDB(t)

	vUser := &model.User{Username: "bot_1", Nickname: "MyBot", Type: "bot"}
	db.Create(vUser)
	bot := &model.Bot{Name: "MyBot", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID}
	db.Create(bot)

	plain := "qbot_valid_token_123"
	db.Create(&model.BotToken{BotID: bot.ID, TokenHash: HashBotToken(plain), Name: "test"})

	r := newBotAuthRouter()
	req := httptest.NewRequest("POST", "/bot/messages", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"bot_id":`)
}

func TestBotAuthMiddleware_RejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupBotAuthTestDB(t)
	r := newBotAuthRouter()
	req := httptest.NewRequest("POST", "/bot/messages", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBotAuthMiddleware_RejectsRevokedToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupBotAuthTestDB(t)

	vUser := &model.User{Username: "bot_2", Nickname: "Bot2", Type: "bot"}
	db.Create(vUser)
	bot := &model.Bot{Name: "Bot2", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID}
	db.Create(bot)

	plain := "qbot_revoked_token"
	tok := &model.BotToken{BotID: bot.ID, TokenHash: HashBotToken(plain), Name: "test"}
	db.Create(tok)
	db.Delete(tok) // 软删除 = 撤销

	r := newBotAuthRouter()
	req := httptest.NewRequest("POST", "/bot/messages", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBotAuthMiddleware_RejectsInactiveBot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupBotAuthTestDB(t)

	vUser := &model.User{Username: "bot_3", Nickname: "Bot3", Type: "bot"}
	db.Create(vUser)
	bot := &model.Bot{Name: "Bot3", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID}
	db.Create(bot)
	// 管理员停用：用 Update 显式置 false（Create 时 GORM default:true 会把零值 false 覆盖为 true）
	db.Model(&model.Bot{}).Where("id = ?", bot.ID).Update("is_active", false)

	plain := "qbot_inactive_token"
	db.Create(&model.BotToken{BotID: bot.ID, TokenHash: HashBotToken(plain), Name: "test"})

	r := newBotAuthRouter()
	req := httptest.NewRequest("POST", "/bot/messages", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}
