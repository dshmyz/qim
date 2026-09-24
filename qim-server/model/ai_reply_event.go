package model

import "time"

// AIReplyEvent 一条 AI 回复的终态事实（回复级，区别于 AIUsageLog 的调用级）。
// 正常完成与中断（length 截断/上游静默断连/错误/超时）统一留痕，供治理看板
// 按场景统计完成率与中断原因；「为什么 AI 回复戛然而止」不再无据可查。
type AIReplyEvent struct {
	ID uint `json:"id" gorm:"primarykey"`
	// Scene 场景：bot_dm / group_mention / group_auto / avatar
	Scene string `json:"scene" gorm:"size:32;not null;index:idx_ai_reply_event_time,priority:1"`
	// ActorID 关联主体：bot 创建者 / 分身主人（群助手场景可为 0）
	ActorID        uint `json:"actor_id" gorm:"index"`
	ConversationID uint `json:"conversation_id" gorm:"not null;index"`
	MessageID      uint `json:"message_id"`
	// Outcome: completed / interrupted / not_triggered（触发门判断无需回复）
	Outcome string `json:"outcome" gorm:"size:16;not null"`
	// Reason 中断原因：length / no_done / error / timeout（completed 时为空）
	Reason    string    `json:"reason" gorm:"size:32"`
	CreatedAt time.Time `json:"created_at" gorm:"not null;index:idx_ai_reply_event_time,priority:2"`
}

// TableName 显式表名。
func (AIReplyEvent) TableName() string { return "ai_reply_events" }
