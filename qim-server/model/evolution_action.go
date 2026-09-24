package model

import "time"

// EvolutionAction 执行器每一次动作的审计记录。
// 自动调整必须留痕：改了什么（before/after）、依据哪批信号（basis）、
// 当前状态（proposed/active/reverted），支持一键回滚与全局追溯。
type EvolutionAction struct {
	ID uint `json:"id" gorm:"primarykey"`
	// Kind 动作类型，P3 为 model_degrade / model_restore
	Kind string `json:"kind" gorm:"size:32;not null"`
	// Target 作用对象（provider 名）
	Target     string `json:"target" gorm:"size:64;not null"`
	BeforeJSON string `json:"before" gorm:"type:text"`
	AfterJSON  string `json:"after" gorm:"type:text"`
	// Basis 依据描述（窗口、样本量、触发指标）
	Basis string `json:"basis" gorm:"size:500"`
	// State proposed（dry-run 建议）/ active（已生效）/ reverted（已回滚）
	State     string    `json:"state" gorm:"size:16;not null;index"`
	CreatedAt time.Time `json:"created_at" gorm:"not null"`
}

// TableName 显式表名。
func (EvolutionAction) TableName() string { return "evolution_actions" }
