package model

import "time"

// TodoMassAssign 群待办提取的大范围指派待确认记录。
// assignee=all（或指派人数超阈值）时 AI 一次判定就会给全群成员建任务+推通知，
// 误判的爆炸半径是全群——先落待确认记录、由触发者二次确认后才真正批量创建。
// 无需迁移：独立新表。
type TodoMassAssign struct {
	ID             uint      `json:"id" gorm:"primarykey"`
	CreatorID      uint      `json:"creator_id" gorm:"not null;index"` // 触发提取的发送者，确认权归属
	ConversationID uint      `json:"conversation_id" gorm:"not null;index"`
	Title          string    `json:"title" gorm:"size:200;not null"`
	Description    string    `json:"description" gorm:"type:text"`
	DueDate        *time.Time `json:"due_date"`
	Priority       string    `json:"priority" gorm:"size:20;default:'medium'"`
	Reminder       int       `json:"reminder"`
	AssigneeIDs    string    `json:"assignee_ids" gorm:"type:text;not null"` // JSON 数组，确认时点的群内人类成员名单（不含发送者）
	Status         string    `json:"status" gorm:"size:20;not null;default:'pending';index"` // pending / approved / rejected / expired
	ExpiresAt      time.Time `json:"expires_at" gorm:"index"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TableName 显式表名，避免 GORM 默认复数策略歧义。
func (TodoMassAssign) TableName() string { return "todo_mass_assigns" }

// TodoMassAssign 状态常量
const (
	TodoMassAssignStatusPending  = "pending"
	TodoMassAssignStatusApproved = "approved"
	TodoMassAssignStatusRejected = "rejected"
	TodoMassAssignStatusExpired  = "expired"
)
