package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupStreamMessageTest(t *testing.T) (*gin.Engine, *gorm.DB, model.User, model.User, model.Conversation) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.Message{},
		&model.Bot{},
		&model.BotConversation{},
	))

	database.DB = db
	di.GlobalContainer = &di.Container{
		DB:                  db,
		AIService:           ai.NewAIService(&ai.AIConfig{}),
		ConversationService: service.NewConversationService(db),
		MessageService:      service.NewMessageService(db, nil, nil),
		PromptManager:       service.NewPromptManager(),
	}

	user := model.User{Username: "stream-user", PasswordHash: "hash", Nickname: "Stream User"}
	virtualUser := model.User{Username: "stream-bot", PasswordHash: "hash", Nickname: "Stream Bot", Type: "bot"}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&virtualUser).Error)

	conv := model.Conversation{Type: "bot"}
	require.NoError(t, db.Create(&conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: user.ID}).Error)

	bot := model.Bot{Name: "Writing Bot", Type: model.BotTypeAssistant, IsActive: true, VirtualUserID: &virtualUser.ID}
	require.NoError(t, db.Create(&bot).Error)
	require.NoError(t, db.Create(&model.BotConversation{BotID: bot.ID, ConversationID: conv.ID}).Error)

	router := gin.New()
	router.POST("/api/v1/conversations/:id/messages/stream", func(c *gin.Context) {
		c.Set("user_id", user.ID)
		StreamMessage(c)
	})

	return router, db, user, virtualUser, conv
}

func TestStreamMessagePersistsBotReplyBeforeFinishing(t *testing.T) {
	router, db, _, virtualUser, conv := setupStreamMessageTest(t)

	body := bytes.NewBufferString(`{"type":"text","content":"你是？"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/1/messages/stream", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "抱歉，AI 服务暂时不可用")

	var botReply model.Message
	require.NoError(t, db.Where("conversation_id = ? AND sender_id = ? AND origin = ?", conv.ID, virtualUser.ID, "assistant").First(&botReply).Error)
	require.Contains(t, botReply.Content, "抱歉，AI 服务暂时不可用")

	var updatedConv model.Conversation
	require.NoError(t, db.First(&updatedConv, conv.ID).Error)
	require.NotNil(t, updatedConv.LastMessageID)
	require.Equal(t, botReply.ID, *updatedConv.LastMessageID)
	require.NotNil(t, updatedConv.LastMessageAt)
}

// 残留会话（bot 已删除但配对行/会话仍在）必须以 Error 帧告知用户，
// 而不是静默 return 让前端流式占位气泡停在空白（历史 bug：用户只看到一条空消息）。
func TestStreamMessageDeletedBotRepliesErrorFrame(t *testing.T) {
	router, db, _, _, conv := setupStreamMessageTest(t)

	// 场景 1：配对行残留、bot 本体已软删 -> 应收到「助手已被删除」Error 帧
	var bot model.Bot
	require.NoError(t, db.First(&bot, "name = ?", "Writing Bot").Error)
	require.NoError(t, db.Delete(&bot).Error) // 软删（model.Bot 带 DeletedAt），配对行保留

	body := bytes.NewBufferString(`{"type":"text","content":"？"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/1/messages/stream", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "该助手已被删除")
	require.Contains(t, w.Body.String(), `"error"`)

	// 场景 2：配对行也不存在（DeleteBot 清理后的残留会话）-> 应收到「会话已失效」Error 帧
	require.NoError(t, db.Where("conversation_id = ?", conv.ID).Delete(&model.BotConversation{}).Error)

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/1/messages/stream", bytes.NewBufferString(`{"type":"text","content":"？"}`))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	require.Equal(t, http.StatusOK, w2.Code)
	require.Contains(t, w2.Body.String(), "该会话已失效")

	// 两种失败场景都不应落库任何 bot 回复（区别于 AI 调用失败会落错误提示回复）
	var assistantReplies int64
	require.NoError(t, db.Model(&model.Message{}).Where("conversation_id = ? AND origin = ?", conv.ID, "assistant").Count(&assistantReplies).Error)
	require.Zero(t, assistantReplies)
}
