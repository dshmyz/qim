package handler

import (
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/dshmyz/qim/qim-server/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// setupStreamingSenderTestDB 构造流式发送器测试上下文：
// 临时文件 sqlite（内存库多连接互相看不到表）+ bot 会话种子 + 全局 DB/DI 容器注入
// （finish 内的 broadcastNewMessage 依赖 database.GetDB 与 GlobalContainer）。
func setupStreamingSenderTestDB(t *testing.T) (*gorm.DB, model.Conversation) {
	t.Helper()
	dbPath := t.TempDir() + "/streaming.db"
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		// 静音：广播构建会按 conversation_id 查群配置，bot 会话必然 not found（预期噪音）
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.ConversationSession{},
		&model.Message{},
		&model.Bot{},
		&model.BotConversation{},
		&model.Group{}, // broadcastNewMessage → BuildMessageResponse 会查群 ai_config
	))

	prevDB, prevContainer, prevHub := database.DB, di.GlobalContainer, ws.GlobalHub
	database.DB = db
	ws.GlobalHub = nil // 测试不接 WS，避免 SafeGo 广播
	di.GlobalContainer = &di.Container{DB: db, ConversationService: service.NewConversationService(db)}
	t.Cleanup(func() {
		database.DB, di.GlobalContainer, ws.GlobalHub = prevDB, prevContainer, prevHub
	})

	vUser := &model.User{Username: "bot_stream", Nickname: "StreamBot", Type: "bot"}
	require.NoError(t, db.Create(vUser).Error)
	bot := &model.Bot{Name: "StreamBot", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID}
	require.NoError(t, db.Create(bot).Error)
	conv := model.Conversation{Type: "bot"}
	require.NoError(t, db.Create(&conv).Error)
	require.NoError(t, db.Create(&model.BotConversation{ConversationID: conv.ID, BotID: bot.ID}).Error)

	return db, conv
}

// 节流窗口内的中间块不落库，首块立即落库，finish 落最终全量内容。
func TestSendStreamingAIMessage_ThrottlesChunkSaves(t *testing.T) {
	db, conv := setupStreamingSenderTestDB(t)

	prevInterval := streamChunkSaveInterval
	streamChunkSaveInterval = time.Hour // 首块之外不再落库，便于观察节流
	t.Cleanup(func() { streamChunkSaveInterval = prevInterval })

	sender := &WebSocketMessageSender{db: db}
	sendChunk, _, finish, err := sender.SendStreamingAIMessage(conv.ID, "StreamBot")
	require.NoError(t, err)

	require.NoError(t, sendChunk("甲"))
	var msg model.Message
	require.NoError(t, db.Where("conversation_id = ?", conv.ID).First(&msg).Error)
	assert.Equal(t, "甲", msg.Content, "首块应因 lastSavedAt 零值立即落库")

	require.NoError(t, sendChunk("乙"))
	require.NoError(t, db.Where("conversation_id = ?", conv.ID).First(&msg).Error)
	assert.Equal(t, "甲", msg.Content, "窗口内的块不应重复落库")

	final := finish()
	require.NotNil(t, final)
	require.NoError(t, db.First(&msg, final.ID).Error)
	assert.Equal(t, "甲乙", msg.Content)
	assert.Equal(t, "markdown", msg.Type)
}

// 流式落库只写 messages 行，不应顺带 upsert Sender 关联（users 表）。
func TestSendStreamingAIMessage_WritesOnlyMessageRow(t *testing.T) {
	db, conv := setupStreamingSenderTestDB(t)

	prevInterval := streamChunkSaveInterval
	streamChunkSaveInterval = 0 // 每块都落库，便于计数
	t.Cleanup(func() { streamChunkSaveInterval = prevInterval })

	var msgWrites, userWrites int
	count := func(d *gorm.DB) {
		switch d.Statement.Table {
		case "messages":
			msgWrites++
		case "users":
			userWrites++
		}
	}
	require.NoError(t, db.Callback().Create().After("gorm:create").Register("test:count_create", count))
	require.NoError(t, db.Callback().Update().After("gorm:update").Register("test:count_update", count))

	sender := &WebSocketMessageSender{db: db}
	sendChunk, _, finish, err := sender.SendStreamingAIMessage(conv.ID, "StreamBot")
	require.NoError(t, err)

	require.NoError(t, sendChunk("张"))
	// 建消息（INSERT）可能顺带 upsert 一次 Sender，取基线后再看后续块写不写 users。
	userWritesAfterFirst := userWrites

	require.NoError(t, sendChunk("三"))
	require.NoError(t, sendChunk("丰"))
	require.NotNil(t, finish())

	assert.Equal(t, userWritesAfterFirst, userWrites, "流式落库不应重复写 users 表（关联 upsert 需 Omit 排除）")
	// 1 次 INSERT + 3 次分块 UPDATE + 1 次 finish UPDATE
	assert.Equal(t, 5, msgWrites)
}
