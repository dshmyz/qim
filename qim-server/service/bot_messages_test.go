package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupReadBot 复用 streaming 测试拓扑：人类用户 + bot 虚拟用户 + 一对一会话，
// 返回 (bot, 人类用户, 会话)。用 stream_ 前缀避免与 setupStreamingBot 冲突。
func setupReadBot(t *testing.T, db *gorm.DB) (*model.Bot, *model.User, *model.Conversation) {
	t.Helper()
	user := &model.User{Username: "read-user", PasswordHash: "hash", Nickname: "Read User"}
	virtualUser := &model.User{Username: "read-virtual-bot", PasswordHash: "hash", Nickname: "Read Bot", Type: "bot"}
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Create(virtualUser).Error)

	conv := &model.Conversation{Type: "bot"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: user.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: virtualUser.ID}).Error)

	bot := &model.Bot{Name: "Reader", Type: model.BotTypeAssistant, IsActive: true, VirtualUserID: &virtualUser.ID}
	require.NoError(t, db.Create(bot).Error)
	require.NoError(t, db.Create(&model.BotConversation{BotID: bot.ID, UserID: user.ID, ConversationID: conv.ID}).Error)
	return bot, user, conv
}

// TestListBotMessages_OwnershipAndIncrement 覆盖：归属拒绝、after_id 增量、只返该会话、limit。
func TestListBotMessages_OwnershipAndIncrement(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewBotMessagingService(db, nil)

	bot, user, conv := setupReadBot(t, db)

	// 用户发 3 条，bot 回 1 条
	db.Create(&model.Message{ConversationID: conv.ID, SenderID: user.ID, Type: "text", Content: "m1", Origin: "user"})
	db.Create(&model.Message{ConversationID: conv.ID, SenderID: user.ID, Type: "text", Content: "m2", Origin: "user"})
	m3 := model.Message{ConversationID: conv.ID, SenderID: user.ID, Type: "text", Content: "m3", Origin: "user"}
	db.Create(&m3)
	db.Create(&model.Message{ConversationID: conv.ID, SenderID: *bot.VirtualUserID, Type: "markdown", Content: "bot-reply", Origin: "bot"})

	// 1. 全量（afterID=0）按 id 升序
	msgs, err := svc.ListBotMessages(bot, conv.ID, 0, 50)
	require.NoError(t, err)
	assert.Len(t, msgs, 4)
	assert.Equal(t, "m1", msgs[0].Content)
	assert.Equal(t, "bot-reply", msgs[3].Content)
	// sender 字段预加载
	assert.Equal(t, "Read User", msgs[0].Sender.Nickname)
	assert.Equal(t, "bot", msgs[3].Sender.Type)

	// 2. after_id 增量：只返 m3 之后的 bot-reply
	msgs, err = svc.ListBotMessages(bot, conv.ID, m3.ID, 50)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "bot-reply", msgs[0].Content)

	// 3. limit 截断
	msgs, err = svc.ListBotMessages(bot, conv.ID, 0, 2)
	require.NoError(t, err)
	assert.Len(t, msgs, 2)

	// 4. 归属拒绝：另一个 bot（无该会话 BotConversation）读取应失败
	virtualB := &model.User{Username: "read-virtual-b", PasswordHash: "hash", Nickname: "B", Type: "bot"}
	require.NoError(t, db.Create(virtualB).Error)
	botB := &model.Bot{Name: "Other", Type: model.BotTypeAssistant, IsActive: true, VirtualUserID: &virtualB.ID}
	require.NoError(t, db.Create(botB).Error)
	_, err = svc.ListBotMessages(botB, conv.ID, 0, 50)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "会话不属于该 bot")
}
