package model

import "time"

const (
	ApprovalTypeAvatar  = "avatar"
	ApprovalTypeBot     = "bot"
	ApprovalTypeChannel = "channel"
)

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
