package model

import "time"

// AI 待确认动作状态机
const (
	AIPendingActionStatusPending   = "pending"   // 待用户确认
	AIPendingActionStatusSending   = "sending"   // 已被某次确认原子占有，发送执行中
	AIPendingActionStatusConfirmed = "confirmed" // 已确认并执行
	AIPendingActionStatusCancelled = "cancelled" // 已取消
	AIPendingActionStatusExpired   = "expired"   // 超时过期
)

// AIPendingAction AI 敏感工具调用的待确认记录。
// 侧边栏 AI 的 send_message 为确认制：工具执行时不真正发送，而是落一条 pending 记录，
// 客户端渲染确认条；用户确认后经 /ai/pending-actions/:id/confirm 执行，取消/超时则作废。
type AIPendingAction struct {
	ID                   uint      `json:"id" gorm:"primarykey"`
	UserID               uint      `json:"user_id" gorm:"not null;index"`            // 发起者（确认时校验归属）
	ConversationID       uint      `json:"conversation_id" gorm:"index"`             // 发起时的上下文会话（侧边栏关联会话，可为 0）
	TargetConversationID uint      `json:"target_conversation_id" gorm:"not null"`   // 待发送目标会话
	TargetName           string    `json:"target_name" gorm:"size:200"`              // 目标会话展示名（创建时解析冗余，仅供确认条展示）
	Content              string    `json:"content" gorm:"type:text;not null"`        // 待发送内容
	Status               string    `json:"status" gorm:"size:20;not null;default:pending;index"`
	MessageID            uint      `json:"message_id"` // 确认执行后实际发出的消息 ID
	ExpiresAt            time.Time `json:"expires_at" gorm:"index"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// TableName 显式表名，避免 GORM 默认复数策略歧义。
func (AIPendingAction) TableName() string { return "ai_pending_actions" }
