package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/middleware"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupTokensDB 建内存库，迁移含 BotToken。
func setupTokensDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Bot{}, &model.BotToken{}))
	database.DB = db
	return db
}

// newBotAPIWithUser 构造一个挂 ListBotTokens 的路由，以 userID 身份鉴权。
func newBotAPIWithUser(t *testing.T, userID uint, roles []string) (*gin.Engine, *BotAPIHandler) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("roles", roles)
		c.Next()
	})
	h := NewBotAPIHandler(nil) // ListBotTokens 不依赖 botMessaging
	r.GET("/bots/:id/tokens", h.ListBotTokens)
	return r, h
}

func TestListBotTokens_OwnerSeesActiveOnly(t *testing.T) {
	db := setupTokensDB(t)
	virtualUser := &model.User{Username: "vbot", PasswordHash: "h", Nickname: "V", Type: "bot"}
	require.NoError(t, db.Create(virtualUser).Error)
	bot := &model.Bot{Name: "B", Type: model.BotTypeAssistant, IsActive: true, CreatorID: 1, VirtualUserID: &virtualUser.ID}
	require.NoError(t, db.Create(bot).Error)

	// 两个有效 token + 一个软删除（撤销）
	require.NoError(t, db.Create(&model.BotToken{BotID: bot.ID, Name: "cli", TokenHash: middleware.HashBotToken("qbot_a")}).Error)
	require.NoError(t, db.Create(&model.BotToken{BotID: bot.ID, Name: "bridge", TokenHash: middleware.HashBotToken("qbot_b")}).Error)
	require.NoError(t, db.Create(&model.BotToken{BotID: bot.ID, Name: "old", TokenHash: middleware.HashBotToken("qbot_c"), DeletedAt: gorm.DeletedAt{Valid: true}}).Error)

	r, _ := newBotAPIWithUser(t, 1, []string{})
	req := httptest.NewRequest(http.MethodGet, "/bots/1/tokens", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data struct {
			Tokens []map[string]any `json:"tokens"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Data.Tokens, 2, "软删除的 token 不应返回")
	// 不含 hash 字段
	for _, tk := range resp.Data.Tokens {
		_, hasHash := tk["token_hash"]
		assert.False(t, hasHash, "不应返回 token_hash")
		assert.NotEmpty(t, tk["name"])
	}
}

func TestListBotTokens_NonOwnerForbidden(t *testing.T) {
	db := setupTokensDB(t)
	virtualUser := &model.User{Username: "vbot2", PasswordHash: "h", Nickname: "V", Type: "bot"}
	require.NoError(t, db.Create(virtualUser).Error)
	bot := &model.Bot{Name: "B2", Type: model.BotTypeAssistant, IsActive: true, CreatorID: 1, VirtualUserID: &virtualUser.ID}
	require.NoError(t, db.Create(bot).Error)

	// 非创建者、非 system_admin -> 403
	r, _ := newBotAPIWithUser(t, 999, []string{})
	req := httptest.NewRequest(http.MethodGet, "/bots/1/tokens", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestListBotTokens_AdminCanReadOthers(t *testing.T) {
	db := setupTokensDB(t)
	virtualUser := &model.User{Username: "vbot3", PasswordHash: "h", Nickname: "V", Type: "bot"}
	require.NoError(t, db.Create(virtualUser).Error)
	bot := &model.Bot{Name: "B3", Type: model.BotTypeAssistant, IsActive: true, CreatorID: 1, VirtualUserID: &virtualUser.ID}
	require.NoError(t, db.Create(bot).Error)
	require.NoError(t, db.Create(&model.BotToken{BotID: bot.ID, Name: "admin-view", TokenHash: middleware.HashBotToken("qbot_x")}).Error)

	// system_admin 可读他人 bot
	r, _ := newBotAPIWithUser(t, 999, []string{"system_admin"})
	req := httptest.NewRequest(http.MethodGet, "/bots/1/tokens", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
