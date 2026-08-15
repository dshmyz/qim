package model

import (
	"time"
)

// BotWebhookDelivery 出站 webhook 的 outbox 投递记录。
// 调用点先落本表（pending），再异步投递；失败按指数退避重试，超阈值进死信（dead）。
// 兜底 agent webhook 端点短暂不可用导致的丢消息（用户->agent 文本、卡片按钮->agent）。
type BotWebhookDelivery struct {
	ID            uint       `json:"id" gorm:"primarykey"`
	BotID         uint       `json:"bot_id" gorm:"not null;index"`
	Event         string     `json:"event" gorm:"size:32;not null"`     // bot.message | bot.card_action
	Payload       string     `json:"payload" gorm:"type:text;not null"` // JSON(BotWebhookPayload)
	WebhookURL    string     `json:"webhook_url" gorm:"size:500"`
	WebhookSecret string     `json:"-" gorm:"size:128"`                    // 出站需签名，随记录存
	Status        string     `json:"status" gorm:"size:16;not null;index"` // pending | done | dead
	Attempts      int        `json:"attempts" gorm:"default:0"`
	LastError     string     `json:"last_error" gorm:"type:text"`
	NextRetryAt   *time.Time `json:"next_retry_at" gorm:"index"` // nil=未到点/已终态
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeliveredAt   *time.Time `json:"delivered_at"` // 成功时间
}
