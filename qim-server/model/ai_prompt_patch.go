package model

import "time"

// AIPromptPatch 一条从失败聚类中学习到的系统提示补丁及其完整生命周期。
//
// 状态流转：
//
//	proposed（LLM 已生成规则句，待回放）→ validated（黄金回放通过，待人工批准）
//	→ active（已批准并注入 system prompt）→ reverted（已停用）
//	                          rejected（黄金回放未通过）
type AIPromptPatch struct {
	ID uint `json:"id" gorm:"primarykey"`
	// Category 针对的失败类别
	Category string `json:"category" gorm:"size:32;not null;index"`
	// Scene 作用场景（空=通用，所有 AI 入口）
	Scene string `json:"scene" gorm:"size:32"`
	// Rule 补丁规则句（注入到 system prompt 的修正规则）
	Rule string `json:"rule" gorm:"type:text;not null"`
	// Rationale 生成依据（为什么这条规则能修复该类失败）
	Rationale string `json:"rationale" gorm:"type:text"`
	// CaseCount 生成时依据的案例数
	CaseCount int `json:"case_count"`
	// Replay 回放结果
	ReplayTotal  int `json:"replay_total"`
	ReplayPassed int `json:"replay_passed"`
	// State proposed/validated/active/rejected/reverted
	State     string    `json:"state" gorm:"size:16;not null;index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 显式表名。
func (AIPromptPatch) TableName() string { return "ai_prompt_patches" }
