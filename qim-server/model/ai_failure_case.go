package model

import "time"

// AIFailureCase 一条自包含的失败案例快照。
// 在差评（带原因）或质量门拒绝时写入：记录当时的问题、坏回复、类别与场景，
// 供失败聚类与补丁的黄金回放直接使用，无需事后回溯多表（消息可能已被撤回/删除）。
type AIFailureCase struct {
	ID uint `json:"id" gorm:"primarykey"`
	// Category 失败类别（off_topic/hallucinated/wrong_trigger/incomplete/factual_error/bad_tone/quality/other）
	Category string `json:"category" gorm:"size:32;not null;index:idx_ai_fcase_cat,priority:1"`
	// Scene bot_dm/group_mention/group_auto/avatar
	Scene string `json:"scene" gorm:"size:32"`
	// Question 触发失败回复的用户问题
	Question string `json:"question" gorm:"type:text"`
	// BadReply 被判定为坏的回复内容
	BadReply string `json:"bad_reply" gorm:"type:text"`
	// ConversationID 来源会话
	ConversationID uint `json:"conversation_id"`
	// Consumed 是否已被用于生成补丁（避免一个案例反复催生补丁）
	Consumed  bool      `json:"consumed"`
	CreatedAt time.Time `json:"created_at" gorm:"not null;index:idx_ai_fcase_cat,priority:2"`
}

// TableName 显式表名。
func (AIFailureCase) TableName() string { return "ai_failure_cases" }
