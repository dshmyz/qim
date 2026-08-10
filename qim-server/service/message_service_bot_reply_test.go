package service

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
)

// mockStorageAccessor 测试用存储后端替代。
type mockStorageAccessor struct{}

func (m *mockStorageAccessor) GetByPath(_ context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (m *mockStorageAccessor) Put(_ context.Context, _ string, _ io.Reader, _ int64, _ string) (string, error) {
	return "", nil
}
func (m *mockStorageAccessor) DeleteByPath(_ context.Context, _ string) error { return nil }
func (m *mockStorageAccessor) Kind() string                                    { return "mock" }

// TestSendBotTextReply_SendsDirectMessage 验证 sendBotTextReply 直接发送 bot 消息，
// 不经过 AI 流式/legacy 管道。错误提示（如"文件过大"）必须原样送达用户。
func TestSendBotTextReply_SendsDirectMessage(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	vUser := &model.User{Username: "bot_direct", Nickname: "BotDirect", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "u_direct", Nickname: "UDirect", Type: "user"}
	db.Create(human)

	bot := &model.Bot{
		Name:          "BotDirect",
		Type:          model.BotTypeCustom,
		IsActive:      true,
		VirtualUserID: &vUser.ID,
		CreatorID:     human.ID,
		Config:        `{"mode":"internal_ai"}`,
	}
	db.Create(bot)

	bmSvc := NewBotMessagingService(db, nil)
	conv, _, err := bmSvc.EnsureBotConversation(bot.ID, human.ID)
	assert.NoError(t, err)

	// aiService=nil：如果走 AI 管道会 panic 或生成不了回复；直接发应不依赖 AI
	msgSvc := NewMessageService(db, nil, nil)

	errorText := "📎 文件「report.pdf」过大（50.0MB），当前支持的最大文件为 20MB。请压缩后重新发送。"
	msgSvc.sendBotTextReply(human.ID, conv.ID, *bot, errorText)

	// 验证：数据库里应有一条 bot 消息，内容精确等于 errorText
	var reply model.Message
	err = db.Where("conversation_id = ? AND sender_id = ?", conv.ID, vUser.ID).First(&reply).Error
	assert.NoError(t, err, "应成功创建 bot 回复消息")
	assert.Equal(t, errorText, reply.Content, "bot 回复内容必须与传入的 errorText 完全一致，不得经过 AI 重写")
	assert.Equal(t, "assistant", reply.Origin, "bot 回复 origin 必须是 assistant")
	assert.Equal(t, "markdown", reply.Type, "bot 回复类型必须是 markdown")
	assert.True(t, reply.IsRead, "bot 回复默认已读")
}

// TestSendBotTextReply_UpdatesConversationLastMessage 验证 sendBotTextReply
// 更新会话 last_message_id 和 last_message_at。
func TestSendBotTextReply_UpdatesConversationLastMessage(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	vUser := &model.User{Username: "bot_conv", Nickname: "BotConv", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "u_conv", Nickname: "UConv", Type: "user"}
	db.Create(human)

	bot := &model.Bot{
		Name:          "BotConv",
		Type:          model.BotTypeCustom,
		IsActive:      true,
		VirtualUserID: &vUser.ID,
		CreatorID:     human.ID,
		Config:        `{"mode":"internal_ai"}`,
	}
	db.Create(bot)

	bmSvc := NewBotMessagingService(db, nil)
	conv, _, err := bmSvc.EnsureBotConversation(bot.ID, human.ID)
	assert.NoError(t, err)

	// 记录旧的 last_message_id
	var beforeConv model.Conversation
	db.First(&beforeConv, conv.ID)

	msgSvc := NewMessageService(db, nil, nil)
	msgSvc.sendBotTextReply(human.ID, conv.ID, *bot, "📎 测试消息")

	var afterConv model.Conversation
	db.First(&afterConv, conv.ID)
	assert.NotEqual(t, beforeConv.LastMessageID, afterConv.LastMessageID,
		"sendBotTextReply 应更新会话 last_message_id")
	assert.NotNil(t, afterConv.LastMessageAt, "sendBotTextReply 应设置 last_message_at")
}

// TestHandleBotImageMessage_StorageError_SendsReply 验证图片存储读取失败时
// 向用户发送反馈消息，而非静默丢弃。
func TestHandleBotImageMessage_StorageError_SendsReply(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	vUser := &model.User{Username: "bot_img", Nickname: "BotImg", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "u_img", Nickname: "UImg", Type: "user"}
	db.Create(human)

	bot := &model.Bot{
		Name:          "BotImg",
		Type:          model.BotTypeCustom,
		IsActive:      true,
		VirtualUserID: &vUser.ID,
		CreatorID:     human.ID,
		Config:        `{"mode":"internal_ai"}`,
	}
	db.Create(bot)

	bmSvc := NewBotMessagingService(db, nil)
	conv, _, err := bmSvc.EnsureBotConversation(bot.ID, human.ID)
	assert.NoError(t, err)

	// 创建一个图片文件记录（storageAccessor 为 nil 会导致存储读取跳过）
	file := &model.File{Name: "test.jpg", Size: 1024, StoragePath: "/fake/path"}
	db.Create(file)

	// storageAccessor=nil 时 handleBotImageMessage 应该 sendBotTextReply 而非静默 return
	msgSvc := NewMessageService(db, nil, nil)
	// 故意不注入 storageAccessor，触发 nil 检查
	msgSvc.handleBotImageMessage(human.ID, conv.ID, *bot,
		`{"id":`+fmt.Sprintf("%d", file.ID)+`}`)

	// 验证：应有 bot 回复消息告知用户存储不可用
	var replyCount int64
	db.Model(&model.Message{}).Where("conversation_id = ? AND sender_id = ?", conv.ID, vUser.ID).Count(&replyCount)
	assert.Equal(t, int64(1), replyCount, "存储不可用时应发送兜底消息而非静默丢弃")
}

// TestDispatchBotReply_FallsBackToLegacy 当 streamingSender 为 nil 时应走 legacy 路径。
func TestDispatchBotReply_FallsBackToLegacy(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	vUser := &model.User{Username: "bot_dispatch", Nickname: "BotDispatch", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "u_dispatch", Nickname: "UDispatch", Type: "user"}
	db.Create(human)

	bot := &model.Bot{
		Name:          "BotDispatch",
		Type:          model.BotTypeCustom,
		IsActive:      true,
		VirtualUserID: &vUser.ID,
		CreatorID:     human.ID,
		Config:        `{"mode":"internal_ai"}`,
	}
	db.Create(bot)

	bmSvc := NewBotMessagingService(db, nil)
	conv, _, err := bmSvc.EnsureBotConversation(bot.ID, human.ID)
	assert.NoError(t, err)

	db.Create(&model.Message{ConversationID: conv.ID, SenderID: human.ID, Type: "text", Content: "hi", Origin: "user"})

	// streamingSender=nil，aiService=nil → 走 legacy（handleBotMessageLegacy）→ 生成兜底回复
	msgSvc := NewMessageService(db, nil, nil)
	msgSvc.dispatchBotReply(human.ID, conv.ID, *bot, ParseBotConfig(bot.Config), nil, nil)

	var reply model.Message
	err = db.Where("conversation_id = ? AND sender_id = ?", conv.ID, vUser.ID).First(&reply).Error
	assert.NoError(t, err, "dispatchBotReply 应通过 legacy 路径创建 bot 回复")
	assert.Equal(t, "assistant", reply.Origin)
}

// TestHandleBotFileMessage_FileIDZero_DoesNotLogNilError 验证 fc.ID==0 时不输出误导性 nil error。
func TestHandleBotFileMessage_FileIDZero_SendsReply(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	vUser := &model.User{Username: "bot_id0", Nickname: "BotID0", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "u_id0", Nickname: "UID0", Type: "user"}
	db.Create(human)

	bot := &model.Bot{
		Name:          "BotID0",
		Type:          model.BotTypeCustom,
		IsActive:      true,
		VirtualUserID: &vUser.ID,
		CreatorID:     human.ID,
		Config:        `{"mode":"internal_ai"}`,
	}
	db.Create(bot)

	bmSvc := NewBotMessagingService(db, nil)
	conv, _, err := bmSvc.EnsureBotConversation(bot.ID, human.ID)
	assert.NoError(t, err)

	// JSON 解析成功但 fc.ID==0 → 应发送反馈消息
	msgSvc := NewMessageService(db, nil, nil)
	msgSvc.SetFileCapabilities(&mockStorageAccessor{}, &DocumentParser{})
	msgSvc.handleBotFileMessage(human.ID, conv.ID, *bot, `{"id":0}`)

	var replyCount int64
	db.Model(&model.Message{}).Where("conversation_id = ? AND sender_id = ?", conv.ID, vUser.ID).Count(&replyCount)
	assert.Equal(t, int64(1), replyCount, "fc.ID==0 时应发送兜底消息")
}
