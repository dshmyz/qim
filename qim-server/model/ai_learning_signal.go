package model

import "time"

// AILearningSignal AI 自我进化的学习信号：用户行为中提炼的高价值样本。
// 与 AIMessageFeedback（显式打分）互补，这里承载带上下文的结构化样本，
// 例如「草稿编辑 diff」=主人亲手写出的理想回复（信号类型 draft_edit）。
// 后续阶段更多信号类型（隐式负反馈、重问等）统一收敛到本表。
type AILearningSignal struct {
	ID             uint      `json:"id" gorm:"primarykey"`
	OwnerID        uint      `json:"owner_id" gorm:"not null;index:idx_ai_lsignal_owner,priority:1"`
	ConversationID uint      `json:"conversation_id" gorm:"not null;index"`
	SignalType     string    `json:"signal_type" gorm:"size:32;not null"`
	PayloadJSON    string    `json:"payload" gorm:"type:text"`
	CreatedAt      time.Time `json:"created_at" gorm:"not null;index:idx_ai_lsignal_owner,priority:2"`
}

// TableName 显式表名。
func (AILearningSignal) TableName() string { return "ai_learning_signals" }
