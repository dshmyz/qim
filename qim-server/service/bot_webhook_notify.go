package service

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/ws"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// webhook 投递失败的用户感知：首次失败 / 死信时在会话里回一条系统提示，
// 让用户知道机器人当前不可达（避免"发了消息没反应"的静默失败）。
// 系统提示落库 + WS new_message 广播，复用群系统消息的既有渲染路径
// （IM 窗口居中灰条 SystemMessage；BotChat 侧显示为左侧文本气泡）。

// webhookNoticeKind 提示类型，决定文案与去重窗口。
type webhookNoticeKind int

const (
	// webhookNoticeFirstFailure 首次投递失败：已自动重试，请稍候。
	webhookNoticeFirstFailure webhookNoticeKind = iota
	// webhookNoticeDead 死信：重试耗尽，不再自动重试。
	webhookNoticeDead
)

// webhookNoticeCooldown 会话级去重冷却期：
// 用户停摆期间连发多条消息会各自产生一条 delivery（首次失败 / 相继死信），
// 若每条都提示会刷屏；同一会话在冷却期内只提示一次。
// 首次失败 5 分钟、死信 30 分钟——死信窗口（约 13 分钟）内的多条死信合并为一条。
var webhookNoticeCooldown = map[webhookNoticeKind]time.Duration{
	webhookNoticeFirstFailure: 5 * time.Minute,
	webhookNoticeDead:         30 * time.Minute,
}

// noticeLastAt 各提示类型最近一次发送时间（key = conversation_id）。
// 单进程内存即可：投递/调度/重投都在同一进程内，无跨实例去重要求。
var (
	noticeMu     sync.Mutex
	noticeLastAt = map[webhookNoticeKind]map[uint]time.Time{
		webhookNoticeFirstFailure: {},
		webhookNoticeDead:         {},
	}
)

// notifyWebhookDeliveryIssue 在 payload.ThreadID（conversation_id）对应会话中
// 写入一条 type=system 的投递问题提示并广播。内部做会话级去重，失败静默不阻断投递流程。
func notifyWebhookDeliveryIssue(db *gorm.DB, payload BotWebhookPayload, kind webhookNoticeKind) {
	if payload.ThreadID == 0 {
		return
	}

	// 会话级去重：冷却期内已提示过则跳过（去重记录先占位，防 DB 失败重入风暴）
	noticeMu.Lock()
	now := time.Now()
	cooldown := webhookNoticeCooldown[kind]
	if last, ok := noticeLastAt[kind][payload.ThreadID]; ok && now.Sub(last) < cooldown {
		noticeMu.Unlock()
		return
	}
	noticeLastAt[kind][payload.ThreadID] = now
	noticeMu.Unlock()

	// 机器人名：查不到用兜底
	botName := "机器人"
	var bot model.Bot
	if err := db.First(&bot, payload.BotID).Error; err == nil && bot.Name != "" {
		botName = bot.Name
	}

	var content string
	if kind == webhookNoticeFirstFailure {
		content = "机器人「" + botName + "」暂时无法回复，系统已自动重试，请稍候。"
	} else {
		content = "机器人「" + botName + "」多次投递失败，您的消息可能未送达。请稍后重试或检查机器人配置。"
	}

	// 创建系统消息（SenderID=0：系统提示无真实操作者，客户端对 type=system 只渲染内容）
	systemMsg := &model.Message{
		ConversationID: payload.ThreadID,
		SenderID:       0,
		Type:           "system",
		Content:        content,
		IsRead:         true,
	}
	if err := db.Create(systemMsg).Error; err != nil {
		logger.WithModule("BotWebhookNotify").Error("创建投递失败系统提示失败", "error", err)
		return
	}

	// 更新会话最后消息
	db.Model(&model.Conversation{}).Where("id = ?", payload.ThreadID).Updates(map[string]interface{}{
		"last_message_id": systemMsg.ID,
		"last_message_at": time.Now(),
	})

	// WebSocket 广播（与群系统消息同构）
	if ws.GlobalHub == nil {
		return
	}
	newMsg := ws.WSMessage{
		Type: "new_message",
		Data: gin.H{
			"id":              systemMsg.ID,
			"conversation_id": systemMsg.ConversationID,
			"sender_id":       systemMsg.SenderID,
			"type":            systemMsg.Type,
			"content":         systemMsg.Content,
			"is_read":         systemMsg.IsRead,
			"created_at":      systemMsg.CreatedAt,
		},
	}
	jsonMsg, _ := json.Marshal(newMsg)
	ws.GlobalHub.SendToConversation(payload.ThreadID, 0, jsonMsg)
}
