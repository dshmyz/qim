package model

import "time"

// AIMessageFeedback 用户对单条 AI 回复的显式反馈（👍/👎）。
// 与 ai_reply_metrics（自动质量度量）互补：主观反馈是质量闭环的用户侧信号源。
// 同一用户对同一消息一条记录，改评即覆盖，rating=0 表示撤销。
type AIMessageFeedback struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	MessageID uint      `json:"message_id" gorm:"not null;uniqueIndex:idx_ai_msg_feedback_unique"`
	UserID    uint      `json:"user_id" gorm:"not null;uniqueIndex:idx_ai_msg_feedback_unique"`
	Rating    int       `json:"rating" gorm:"not null"` // 1=赞，-1=踩（0 不落库，撤销即删行）
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 显式表名。
func (AIMessageFeedback) TableName() string { return "ai_message_feedbacks" }
