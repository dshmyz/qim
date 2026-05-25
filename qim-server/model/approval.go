package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	ApprovalTypeAvatar  = "avatar"
	ApprovalTypeBot     = "bot"
	ApprovalTypeChannel = "channel"
	ApprovalTypeGroupAI = "group_ai" // 群聊AI助手审批
)

var ApprovalTypeNames = map[string]string{
	ApprovalTypeAvatar:  "分身功能",
	ApprovalTypeBot:     "机器人创建",
	ApprovalTypeChannel: "频道创建",
	ApprovalTypeGroupAI: "群聊AI助手",
}

const (
	ApprovalStatusPending  = "pending"
	ApprovalStatusApproved = "approved"
	ApprovalStatusRejected = "rejected"
	ApprovalStatusNone     = "none"
)

type Approval struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	TargetType  string         `json:"target_type" gorm:"size:30;not null"`     // 审批类型
	TargetID    uint           `json:"target_id" gorm:"not null"`               // 目标记录ID
	Status      string         `json:"status" gorm:"size:20;default:'pending'"` // 状态
	AppliedAt   time.Time      `json:"applied_at" gorm:"not null"`              // 申请时间
	AppliedBy   uint           `json:"applied_by" gorm:"not null"`              // 申请人ID
	ApprovedAt  *time.Time     `json:"approved_at"`                             // 审批时间
	ApprovedBy  *uint          `json:"approved_by"`                             // 审批人ID
	RejectReason string        `json:"reject_reason" gorm:"type:text"`          // 拒绝原因
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Approval) TableName() string {
	return "approvals"
}

type ApprovalConfig struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	Type        string    `json:"type" gorm:"uniqueIndex;size:20;not null"` // 审批类型
	Enabled     bool      `json:"enabled" gorm:"default:false"`             // 是否启用审批
	Description string    `json:"description" gorm:"size:200"`              // 审批说明
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ApprovalListItem struct {
	ID             uint       `json:"id"`
	Type           string     `json:"type"`
	CreatorID      uint       `json:"creator_id"`
	CreatorName    string     `json:"creator_name"`
	CreatorAvatar  string     `json:"creator_avatar"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	ApprovalStatus string     `json:"approval_status"`
	AppliedAt      *time.Time `json:"applied_at"`
	ApprovedAt     *time.Time `json:"approved_at"`
	RejectReason   string     `json:"reject_reason"`
	CreatedAt      time.Time  `json:"created_at"`
	Extra          any        `json:"extra,omitempty"`
}