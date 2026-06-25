package model

import "time"

// RealtimeSession 实时会话
type RealtimeSession struct {
	ID             string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Type           string     `json:"type" gorm:"type:varchar(20);not null;index"` // screen_share, voice_call, video_call
	InitiatorID    uint       `json:"initiator_id" gorm:"not null;index"`
	ConversationID uint       `json:"conversation_id" gorm:"not null;index"`
	Status         string     `json:"status" gorm:"type:varchar(20);default:'pending'"` // pending, active, paused, ended
	StartedAt      *time.Time `json:"started_at"`
	EndedAt        *time.Time `json:"ended_at"`
	Metadata       string     `json:"metadata" gorm:"type:text"` // JSON 扩展字段
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	Initiator    User                  `json:"initiator,omitempty" gorm:"foreignKey:InitiatorID"`
	Participants []RealtimeParticipant `json:"participants,omitempty" gorm:"foreignKey:SessionID"`
}

// RealtimeParticipant 实时会话参与者
type RealtimeParticipant struct {
	ID          string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	SessionID   string     `json:"session_id" gorm:"type:varchar(36);not null;index"`
	UserID      uint       `json:"user_id" gorm:"not null;index"`
	Role        string     `json:"role" gorm:"type:varchar(20);default:'viewer'"`    // initiator, viewer
	Status      string     `json:"status" gorm:"type:varchar(20);default:'pending'"` // pending, approved, rejected, joined, left
	RequestedAt time.Time  `json:"requested_at"`
	ApprovedAt  *time.Time `json:"approved_at"`
	JoinedAt    *time.Time `json:"joined_at"`
	LeftAt      *time.Time `json:"left_at"`

	User    User            `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Session RealtimeSession `json:"session,omitempty" gorm:"foreignKey:SessionID"`
}

// TableName 指定表名
func (RealtimeSession) TableName() string {
	return "realtime_sessions"
}

func (RealtimeParticipant) TableName() string {
	return "realtime_participants"
}
