package service

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/ws"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupStreamingBot 准备一个启用外部 webhook 的 bot + 其虚拟用户 + 一对一会话，
// 返回 (bot, 人类用户, 会话)。复用 TestMessageService_SendMessageToBotPublishesReplyAndUpdatesConversation 的拓扑。
func setupStreamingBot(t *testing.T, db *gorm.DB) (*model.Bot, *model.User, *model.Conversation) {
	t.Helper()
	user := &model.User{Username: "stream-user", PasswordHash: "hash", Nickname: "Stream User"}
	virtualUser := &model.User{Username: "stream-virtual-bot", PasswordHash: "hash", Nickname: "Stream Bot", Type: "bot"}
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Create(virtualUser).Error)

	conv := &model.Conversation{Type: "bot"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: user.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: virtualUser.ID}).Error)

	bot := &model.Bot{Name: "Streamer", Type: model.BotTypeAssistant, IsActive: true, VirtualUserID: &virtualUser.ID}
	require.NoError(t, db.Create(bot).Error)
	require.NoError(t, db.Create(&model.BotConversation{BotID: bot.ID, UserID: user.ID, ConversationID: conv.ID}).Error)
	return bot, user, conv
}

// dialUserWS 以人类用户身份连入真实 hub，返回 WS 连接（测试结束自动关闭）。
func dialUserWS(t *testing.T, hub *ws.Hub, userID uint) *websocket.Conn {
	t.Helper()
	router := gin.New()
	router.GET("/ws", func(c *gin.Context) {
		c.Set("user_id", userID)
		ws.ServeWs(hub, c)
	})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

// readWS 读一条 WS 消息并解析到 data 结构，超时 2s。
func readWS(t *testing.T, conn *websocket.Conn) (string, map[string]interface{}) {
	t.Helper()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))
	var incoming ws.WSMessage
	require.NoError(t, conn.ReadJSON(&incoming))
	dataBytes, err := json.Marshal(incoming.Data)
	require.NoError(t, err)
	var data map[string]interface{}
	require.NoError(t, json.Unmarshal(dataBytes, &data))
	return incoming.Type, data
}

// TestBotStreaming_HappyPath 覆盖完整流式契约：
//  1. SendOutbound(streaming) 建空流式消息，new_message 含 is_streaming:true / type:streaming
//  2. StreamChunk(delta) 累加内容，message_updated 全量 content + is_streaming:true
//  3. StreamChunk(finish) 收尾 -> type:markdown / is_streaming:false，会话 last_message 更新
func TestBotStreaming_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupServiceTestDB(t)
	hub := ws.NewHub(db, "test-secret", "http")
	go hub.Run()
	svc := NewBotMessagingService(db, hub)

	bot, user, _ := setupStreamingBot(t, db)
	conn := dialUserWS(t, hub, user.ID)

	// 1. 建流式消息
	msg, err := svc.SendOutbound(bot, user.ID, "", "streaming", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, msg)

	msgType, data := readWS(t, conn)
	require.Equal(t, "new_message", msgType)
	assert.Equal(t, "streaming", data["type"])
	assert.Equal(t, true, data["is_streaming"])
	assert.Equal(t, "", data["content"])
	assert.Equal(t, float64(msg.ID), data["id"])

	// 2. 追加第一段
	require.NoError(t, svc.StreamChunk(bot, msg.ID, "Hello", false))
	msgType, data = readWS(t, conn)
	require.Equal(t, "message_updated", msgType)
	assert.Equal(t, "Hello", data["content"])
	assert.Equal(t, "streaming", data["type"])
	assert.Equal(t, true, data["is_streaming"])

	// 3. 追加第二段（验证累加为全量替换语义）
	require.NoError(t, svc.StreamChunk(bot, msg.ID, " world", false))
	msgType, data = readWS(t, conn)
	require.Equal(t, "message_updated", msgType)
	assert.Equal(t, "Hello world", data["content"])
	assert.Equal(t, true, data["is_streaming"])

	// 4. 收尾 -> markdown + is_streaming:false + 会话最后消息更新
	require.NoError(t, svc.StreamChunk(bot, msg.ID, "", true))
	msgType, data = readWS(t, conn)
	require.Equal(t, "message_updated", msgType)
	assert.Equal(t, "Hello world", data["content"])
	assert.Equal(t, "markdown", data["type"])
	assert.Equal(t, false, data["is_streaming"])

	// DB 终态：消息 type 已落 markdown
	var finalMsg model.Message
	require.NoError(t, db.First(&finalMsg, msg.ID).Error)
	assert.Equal(t, "markdown", finalMsg.Type)
	assert.Equal(t, "Hello world", finalMsg.Content)

	// 会话最后消息指向该流式消息
	var conv model.Conversation
	require.NoError(t, db.First(&conv, finalMsg.ConversationID).Error)
	require.NotNil(t, conv.LastMessageID)
	assert.Equal(t, msg.ID, *conv.LastMessageID)
	require.NotNil(t, conv.LastMessageAt)
}

// TestStreamChunk_RefreshesUpdatedAt 回归：StreamChunk 的 map Updates 必须刷新 updated_at。
// 僵尸流式清理（CleanupStaleStreamingMessages）按 updated_at < now-10min 判定僵尸，
// 若活跃流每段不刷 updated_at，活跃流超 10min 就被误杀收尾。此测试锁住该安全前提。
func TestStreamChunk_RefreshesUpdatedAt(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewBotMessagingService(db, nil)

	bot, user, _ := setupStreamingBot(t, db)
	msg, err := svc.SendOutbound(bot, user.ID, "", "streaming", nil, nil)
	require.NoError(t, err)

	var before model.Message
	require.NoError(t, db.First(&before, msg.ID).Error)

	// 等 >1s 超过任何亚秒精度，确保时间戳确有推进空间
	time.Sleep(1100 * time.Millisecond)

	require.NoError(t, svc.StreamChunk(bot, msg.ID, "delta", false))

	var after model.Message
	require.NoError(t, db.First(&after, msg.ID).Error)
	assert.True(t, after.UpdatedAt.After(before.UpdatedAt),
		"StreamChunk 应刷新 updated_at，否则活跃流会被僵尸清理误杀。before=%s after=%s",
		before.UpdatedAt.Format(time.RFC3339Nano), after.UpdatedAt.Format(time.RFC3339Nano))
	assert.Equal(t, "delta", after.Content, "delta 应已累加")
	assert.Equal(t, "streaming", after.Type, "未 finish 时保持 streaming")
}

// TestStreamChunk_RejectsNonStreaming 非流式消息拒绝分段追加。
func TestStreamChunk_RejectsNonStreaming(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewBotMessagingService(db, nil) // hub=nil，拒绝路径不需要推送

	bot, user, _ := setupStreamingBot(t, db)
	// 建一条普通 markdown 消息
	msg, err := svc.SendOutbound(bot, user.ID, "hi", "markdown", nil, nil)
	require.NoError(t, err)

	err = svc.StreamChunk(bot, msg.ID, " more", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "非流式消息")
}

// TestStreamChunk_RejectsNonOwningBot 非归属 bot 拒绝追加。
func TestStreamChunk_RejectsNonOwningBot(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewBotMessagingService(db, nil)

	botA, user, _ := setupStreamingBot(t, db)
	// bot A 建流式消息
	msg, err := svc.SendOutbound(botA, user.ID, "", "streaming", nil, nil)
	require.NoError(t, err)

	// bot B：独立的虚拟用户，但在该会话上无 BotConversation
	virtualB := &model.User{Username: "stream-virtual-b", PasswordHash: "hash", Nickname: "B", Type: "bot"}
	require.NoError(t, db.Create(virtualB).Error)
	botB := &model.Bot{Name: "Other", Type: model.BotTypeAssistant, IsActive: true, VirtualUserID: &virtualB.ID}
	require.NoError(t, db.Create(botB).Error)

	err = svc.StreamChunk(botB, msg.ID, "x", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "会话不属于该 bot")
}
