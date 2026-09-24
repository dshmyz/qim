package model

import "time"

// AIModelHealth 单个 provider 的滚动健康状态（健康门控的当前判定）。
// 执行器按窗口从 ai_model_calls 聚合后 upsert；SelectProvider 据此在有健康
// 替代时跳过 degraded 的 provider。状态由系统自动维护、可一键回滚。
type AIModelHealth struct {
	ID            uint   `json:"id" gorm:"primarykey"`
	Provider      string `json:"provider" gorm:"size:64;uniqueIndex"`
	WindowCalls   int    `json:"window_calls"`
	Completed     int    `json:"completed"`
	Interrupted   int    `json:"interrupted"`
	AvgDurationMs int64  `json:"avg_duration_ms"`
	// Score 健康分 0-1（完成率，越高越好）
	Score float64 `json:"score"`
	// State healthy / degraded
	State     string    `json:"state" gorm:"size:16;not null"`
	Reason    string    `json:"reason" gorm:"size:255"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 显式表名。
func (AIModelHealth) TableName() string { return "ai_model_health" }
