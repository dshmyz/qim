package model

import (
	"time"

	"gorm.io/gorm"
)

// AvatarConfig 分身配置
type AvatarConfig struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	UserID    uint           `json:"user_id" gorm:"uniqueIndex;not null"`
	Name      string         `json:"name" gorm:"size:100;default:'我的分身'"`
	Enabled   bool           `json:"enabled" gorm:"default:false"`

	// 人设相关
	AutoLearnedPersona string     `json:"auto_learned_persona" gorm:"type:text"`
	CustomPersonaAddon string     `json:"custom_persona_addon" gorm:"type:text"`
	PersonaVersion     int        `json:"persona_version" gorm:"default:0"`
	LastLearnedAt      *time.Time `json:"last_learned_at"`

	// 配置 JSON 字段
	KnowledgeScopeJSON string `json:"-" gorm:"type:text"`
	TriggerRulesJSON   string `json:"-" gorm:"type:text"`
	ReplyStrategyJSON  string `json:"-" gorm:"type:text"`

	// 模型配置
	ModelConfigID   *uint `json:"model_config_id"`
	UseSystemConfig bool  `json:"use_system_config" gorm:"default:true"`

	// 接管冷却时间（分钟）
	TakeoverCooldown int `json:"takeover_cooldown" gorm:"default:10"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	User        User          `json:"user,omitempty" gorm:"foreignkey:UserID"`
	ModelConfig *UserAIConfig `json:"model_config,omitempty" gorm:"foreignkey:ModelConfigID"`
}

// AvatarSession 会话级分身状态
type AvatarSession struct {
	ID             uint       `json:"id" gorm:"primarykey"`
	ConversationID uint       `json:"conversation_id" gorm:"not null;uniqueIndex:idx_avatar_user_conv"`
	UserID         uint       `json:"user_id" gorm:"not null;uniqueIndex:idx_avatar_user_conv"`
	AvatarEnabled  bool       `json:"avatar_enabled" gorm:"default:false"`
	TakeoverUntil  *time.Time `json:"takeover_until"`
	LastReplyAt    *time.Time `json:"last_reply_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	Conversation Conversation `json:"conversation,omitempty" gorm:"foreignkey:ConversationID"`
	User         User         `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

// AvatarLearnTask 人设学习任务
type AvatarLearnTask struct {
	ID           uint       `json:"id" gorm:"primarykey"`
	UserID       uint       `json:"user_id" gorm:"not null;index"`
	Status       string     `json:"status" gorm:"size:20;default:'pending'"` // pending, processing, completed, failed
	Progress     int        `json:"progress" gorm:"default:0"`
	MessageCount int        `json:"message_count"`
	Error        string     `json:"error" gorm:"type:text"`
	StartedAt    *time.Time `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	User User `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

// AvatarKnowledgeScope 分身知识范围配置
type AvatarKnowledgeScope struct {
	ConversationHistory bool `json:"conversation_history"`
	KnowledgeDocs       bool `json:"knowledge_docs"`
	Notes               bool `json:"notes"`
	Tasks               bool `json:"tasks"`
}

// AvatarTriggerRules 分身触发规则
type AvatarTriggerRules struct {
	Mode                string             `json:"mode"` // auto, manual, keyword, schedule
	Keywords            []string           `json:"keywords"`
	TimeRanges          []AvatarTimeRange  `json:"time_ranges"`
	ExcludedConversations []uint           `json:"excluded_conversations"`
}

// AvatarTimeRange 时间范围配置
type AvatarTimeRange struct {
	DayOfWeek []int `json:"day_of_week"` // 0-6, 0=Sunday
	StartHour int   `json:"start_hour"`
	EndHour   int   `json:"end_hour"`
}

// AvatarReplyStrategy 分身回复策略
type AvatarReplyStrategy struct {
	MaxReplyLength       string  `json:"max_reply_length"` // short, medium, long
	ReplyDelay           int     `json:"reply_delay"`      // 秒
	ConfidenceThreshold  float64 `json:"confidence_threshold"`
	DisclaimerStyle      string  `json:"disclaimer_style"` // none, subtle, explicit
}
