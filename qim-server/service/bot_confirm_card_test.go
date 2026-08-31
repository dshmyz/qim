package service

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dshmyz/qim/qim-server/ws"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newConfirmCardFixture 造 bot 1:1 确认卡环境：用户、bot（虚拟用户）、bot 会话、目标群。
func newConfirmCardFixture(t *testing.T) (*BotMessagingService, *AIPendingActionService, *MessageService, *model.User, *model.Conversation, model.Bot) {
	t.Helper()
	db := setupServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.AIPendingAction{}, &model.CardActionRecord{}))

	user := &model.User{Username: "carol", Nickname: "卡罗尔"}
	require.NoError(t, db.Create(user).Error)
	virtual := &model.User{Username: "ai_assistant_virtual", Nickname: "AI助手"}
	require.NoError(t, db.Create(virtual).Error)

	target := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(target).Error)
	require.NoError(t, db.Create(&model.Group{ConversationID: target.ID, Name: "技术交流群", GroupType: "group", CreatorID: user.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: target.ID, UserID: user.ID}).Error)

	botConv := &model.Conversation{Type: "bot"}
	require.NoError(t, db.Create(botConv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: botConv.ID, UserID: user.ID}).Error)

	bot := model.Bot{Name: "智能助手", Type: model.BotTypeAssistant, IsActive: true, VirtualUserID: &virtual.ID}
	require.NoError(t, db.Create(&bot).Error)
	require.NoError(t, db.Create(&model.BotConversation{BotID: bot.ID, ConversationID: botConv.ID}).Error)

	hub := ws.NewHub(db, "test-secret", "http")
	botMsgSvc := NewBotMessagingService(db, hub)
	msgSvc := NewMessageService(db, hub, nil)
	pendingSvc := NewAIPendingActionService(msgSvc)
	botMsgSvc.SetPendingActions(pendingSvc)
	msgSvc.SetBotMessaging(botMsgSvc)
	return botMsgSvc, pendingSvc, msgSvc, user, target, bot
}

func TestSendAIConfirmCardAndConfirmClick(t *testing.T) {
	botMsgSvc, pendingSvc, _, user, target, bot := newConfirmCardFixture(t)
	var bc model.BotConversation
	require.NoError(t, botMsgSvc.db.Where("bot_id = ?", bot.ID).First(&bc).Error)

	// 真实顺序：工具建 pending → bot 链路发确认卡
	record, err := pendingSvc.CreatePendingSend(user.ID, bc.ConversationID, target.ID, "周五聚餐")
	require.NoError(t, err)
	require.NoError(t, botMsgSvc.SendAIConfirmCard(bc.ConversationID, bot, PendingConfirmCardInfo{
		ID: record.ID, TargetName: record.TargetName, Preview: record.Content,
	}))

	// 卡片落库且契约合法（kind=ai_confirm、两个按钮）
	var card model.Message
	require.NoError(t, botMsgSvc.db.Where("conversation_id = ? AND type = ?", bc.ConversationID, "card").
		Order("id DESC").First(&card).Error)
	var cardPayload map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(card.Content), &cardPayload))
	assert.Equal(t, "ai_confirm", cardPayload["kind"])
	assert.EqualValues(t, record.ID, cardPayload["pending_id"])
	buttons := cardPayload["buttons"].([]interface{})
	assert.Len(t, buttons, 2)

	// 2) 用户点击「确认发送」→ pending 执行 + 终态回写 + 点击记录
	require.NoError(t, botMsgSvc.ForwardCardAction(card.ID, user.ID, "confirm", "pending:1"))

	var confirmed model.AIPendingAction
	require.NoError(t, pendingSvc.db.First(&confirmed, record.ID).Error)
	assert.Equal(t, model.AIPendingActionStatusConfirmed, confirmed.Status)

	// 目标群真实收到消息
	var targetMsgCount int64
	botMsgSvc.db.Model(&model.Message{}).Where("conversation_id = ? AND content = ?", target.ID, "周五聚餐").Count(&targetMsgCount)
	assert.EqualValues(t, 1, targetMsgCount, "确认后目标会话应收到消息")

	// 卡片回写终态文案
	var latest model.Message
	require.NoError(t, botMsgSvc.db.First(&latest, card.ID).Error)
	assert.Contains(t, latest.Content, "已发送到 技术交流群")

	// 点击记录存在（按钮保持禁用态）
	var actionCount int64
	botMsgSvc.db.Model(&model.CardActionRecord{}).Where("message_id = ? AND user_id = ?", card.ID, user.ID).Count(&actionCount)
	assert.EqualValues(t, 1, actionCount)

	// 确认卡走 sendToConversation 管道：人类成员未读 +1（与其他 bot 消息一致，有红点）
	var member model.ConversationMember
	require.NoError(t, botMsgSvc.db.Where("conversation_id = ? AND user_id = ?", bc.ConversationID, user.ID).First(&member).Error)
	assert.EqualValues(t, 1, member.UnreadCount, "确认卡应计入人类成员未读")

	// 3) 重复点击：pending 已终态 → 幂等回写、不再发送
	require.NoError(t, botMsgSvc.ForwardCardAction(card.ID, user.ID, "confirm", "pending:1"))
	botMsgSvc.db.Model(&model.Message{}).Where("conversation_id = ?", target.ID).Count(&targetMsgCount)
	assert.EqualValues(t, 1, targetMsgCount, "重复确认不应重复发送")
}

func TestConfirmCardCancelClick(t *testing.T) {
	botMsgSvc, pendingSvc, _, user, target, bot := newConfirmCardFixture(t)
	var bc model.BotConversation
	require.NoError(t, botMsgSvc.db.Where("bot_id = ?", bot.ID).First(&bc).Error)

	record, err := pendingSvc.CreatePendingSend(user.ID, bc.ConversationID, target.ID, "取消我")
	require.NoError(t, err)
	require.NoError(t, botMsgSvc.SendAIConfirmCard(bc.ConversationID, bot, PendingConfirmCardInfo{
		ID: record.ID, TargetName: record.TargetName, Preview: record.Content,
	}))
	var card model.Message
	require.NoError(t, botMsgSvc.db.Where("conversation_id = ? AND type = ?", bc.ConversationID, "card").
		Order("id DESC").First(&card).Error)

	require.NoError(t, botMsgSvc.ForwardCardAction(card.ID, user.ID, "cancel", ""))

	var cancelled model.AIPendingAction
	require.NoError(t, pendingSvc.db.First(&cancelled, record.ID).Error)
	assert.Equal(t, model.AIPendingActionStatusCancelled, cancelled.Status)

	var latest model.Message
	require.NoError(t, botMsgSvc.db.First(&latest, card.ID).Error)
	assert.Contains(t, latest.Content, "已取消，未发送")

	var targetMsgCount int64
	botMsgSvc.db.Model(&model.Message{}).Where("conversation_id = ?", target.ID).Count(&targetMsgCount)
	assert.EqualValues(t, 0, targetMsgCount, "取消后不应发出消息")
}

func TestBotDMWhitelistIncludesSendMessage(t *testing.T) {
	// bot 1:1 白名单放开 send_message（确认制）；工具在 bot 路径的 callerCtx 下
	// 默认目标 = ctx.ConversationID（bot 会话），未指定时不报"未指定会话"
	assert.Contains(t, botAllowedTools, "send_message")

	db := setupServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.AIPendingAction{}))
	msgSvc := NewMessageService(db, nil, nil)
	pendingSvc := NewAIPendingActionService(msgSvc)
	tool := NewSendMessageTool(msgSvc, pendingSvc)

	result, err := tool.Execute(map[string]interface{}{"content": "默认目标测试"}, &ai.CallerContext{
		UserID: 1, ConversationID: 42, ConfirmTools: []string{"send_message"},
	})
	require.NoError(t, err)
	assert.Equal(t, "pending_confirmation", result.(map[string]interface{})["status"])
}

func TestConfirmCardSendFailureKeepsRetryable(t *testing.T) {
	botMsgSvc, pendingSvc, _, user, target, bot := newConfirmCardFixture(t)
	var bc model.BotConversation
	require.NoError(t, botMsgSvc.db.Where("bot_id = ?", bot.ID).First(&bc).Error)

	// 注入敏感词，让 ConfirmPendingSend 内的 SendMessage 失败（pending 保留可重试）。
	// 直接构造缓存（sensitiveWordLoaded=true），避免触发 sensitive_words 表查询。
	msgSvc := pendingSvc.msgSvc
	msgSvc.sensitiveWordCacheMu.Lock()
	msgSvc.sensitiveWordCache = []model.SensitiveWord{{Word: "违禁词", Enabled: true}}
	msgSvc.sensitiveWordLoaded = true
	msgSvc.sensitiveWordCacheMu.Unlock()

	record, err := pendingSvc.CreatePendingSend(user.ID, bc.ConversationID, target.ID, "包含违禁词的内容")
	require.NoError(t, err)
	require.NoError(t, botMsgSvc.SendAIConfirmCard(bc.ConversationID, bot, PendingConfirmCardInfo{
		ID: record.ID, TargetName: record.TargetName, Preview: record.Content,
	}))
	var card model.Message
	require.NoError(t, botMsgSvc.db.Where("conversation_id = ? AND type = ?", bc.ConversationID, "card").
		Order("id DESC").First(&card).Error)

	// 点击确认：发送失败 → 错误带回，但 pending 保留
	err = botMsgSvc.ForwardCardAction(card.ID, user.ID, "confirm", fmt.Sprintf("pending:%d", record.ID))
	require.Error(t, err, "发送失败应返回错误供前端提示")

	var confirmed model.AIPendingAction
	require.NoError(t, pendingSvc.db.First(&confirmed, record.ID).Error)
	assert.Equal(t, model.AIPendingActionStatusPending, confirmed.Status, "发送失败后 pending 必须保留供重试")

	// 关键：不得落点击记录（否则 card_action_id 会永久禁用按钮）
	var actionCount int64
	botMsgSvc.db.Model(&model.CardActionRecord{}).Where("message_id = ? AND user_id = ?", card.ID, user.ID).Count(&actionCount)
	assert.EqualValues(t, 0, actionCount, "发送失败不应落点击记录锁死按钮")

	// 卡片内容不得回写终态文案
	var latest model.Message
	require.NoError(t, botMsgSvc.db.First(&latest, card.ID).Error)
	assert.NotContains(t, latest.Content, "操作失败", "发送失败不应回写终态，保持原卡片可交互")

	// 清空敏感词缓存后重试 → 成功
	msgSvc.sensitiveWordCacheMu.Lock()
	msgSvc.sensitiveWordCache = nil
	msgSvc.sensitiveWordLoaded = true
	msgSvc.sensitiveWordCacheMu.Unlock()
	require.NoError(t, botMsgSvc.ForwardCardAction(card.ID, user.ID, "confirm", fmt.Sprintf("pending:%d", record.ID)))
	var targetMsgCount int64
	botMsgSvc.db.Model(&model.Message{}).Where("conversation_id = ? AND content = ?", target.ID, "包含违禁词的内容").Count(&targetMsgCount)
	assert.EqualValues(t, 1, targetMsgCount, "解除拦截后重试应成功发出")
}
