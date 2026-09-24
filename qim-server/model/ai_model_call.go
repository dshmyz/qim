package model

import "time"

// AIModelCall 一次在线模型调用的事实行（调用级）。
// 与 AIUsageLog（token 用量，仅成功路径）互补：本表无论成功、失败、长度截断
// 都记录，供执行器按 provider/model 聚合真实的完成率、错误率与延迟。
type AIModelCall struct {
	ID       uint   `json:"id" gorm:"primarykey"`
	Provider string `json:"provider" gorm:"size:64;not null;index:idx_ai_mcall_agg,priority:1"`
	Model    string `json:"model" gorm:"size:128;not null;index:idx_ai_mcall_agg,priority:2"`
	TaskType string `json:"task_type" gorm:"size:32"`
	// FinishReason stop / length / ""（错误时为空）
	FinishReason string    `json:"finish_reason" gorm:"size:32"`
	Success      bool      `json:"success"`
	DurationMs   int64     `json:"duration_ms"`
	CreatedAt    time.Time `json:"created_at" gorm:"not null;index:idx_ai_mcall_agg,priority:3"`
}

// TableName 显式表名。
func (AIModelCall) TableName() string { return "ai_model_calls" }
