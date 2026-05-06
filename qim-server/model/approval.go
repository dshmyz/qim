package model

import "time"

const (
	ApprovalTypeAvatar  = "avatar"
	ApprovalTypeBot     = "bot"
	ApprovalTypeChannel = "channel"
	ApprovalTypeGroupAI = "group_ai" // 群聊AI助手审批
)

// 审批类型名称映射
var ApprovalTypeNames = map[string]string{
	ApprovalTypeAvatar:  "分身功能",
	ApprovalTypeBot:     "机器人创建",
	ApprovalTypeChannel: "频道创建",
	ApprovalTypeGroupAI: "群聊AI助手",
}

type ApprovalEntity interface {
	GetID() uint
	GetCreatorID() uint
	GetApprovalStatus() string
	GetApprovalType() string
	SetApprovalStatus(status string)
	SetApprovedAt(t *time.Time)
	SetApprovedBy(adminID uint)
	SetRejectReason(reason string)
	GetRejectReason() string
}

// ApprovalConfig 审批配置
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
