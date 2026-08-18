package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// AvatarConfig 分身配置
type AvatarConfig struct {
	ID      uint   `json:"id" gorm:"primarykey"`
	UserID  uint   `json:"user_id" gorm:"uniqueIndex;not null"`
	Name    string `json:"name" gorm:"size:100;default:'我的分身'"`
	Enabled bool   `json:"enabled" gorm:"default:false"`

	// ActivateByDefault：无显式会话级 session 时，分身是否默认在该会话激活。
	// true=广覆盖（所有会话自动开，逐个 opt-out）；false=逐会话 opt-in。默认 false，避免升级即失控。
	ActivateByDefault bool `json:"activate_by_default" gorm:"default:false"`

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

	// 接管冷却时间（分钟）：点击「接管分身」后，分身暂停回复的时长。与 SelfMessagePause 相互独立。
	TakeoverCooldown int `json:"takeover_cooldown" gorm:"default:10"`

	// 你发消息后，分身暂停回复的时间（分钟）。0=关闭（你发消息不触发分身暂停）。
	// 与手动接管冷却独立：仅当分身主人在某会话发言后触发，复用 TakeoverUntil 门控。
	SelfMessagePause int `json:"self_message_pause" gorm:"default:0"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	User        User      `json:"user,omitempty" gorm:"foreignkey:UserID"`
	ModelConfig *AIConfig `json:"model_config,omitempty" gorm:"foreignkey:ModelConfigID"`
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

// AvatarToolBinding 分身工具绑定
type AvatarToolBinding struct {
	ID        uint   `gorm:"primaryKey"`
	AvatarID  uint   `gorm:"index"`
	ToolID    string `gorm:"size:64"`
	Enabled   bool   `gorm:"default:true"`
	Priority  int    `gorm:"default:1"`
	CreatedAt time.Time
	UpdatedAt time.Time
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
	ConversationHistory *bool `json:"conversationHistory"` // nil=未设置→按默认 true（保留对话历史）；显式 false=关闭
	Memory              *bool `json:"memory"`              // nil=未设置→按默认 true（启用分身记忆）；显式 false=关闭
	KnowledgeDocs       bool  `json:"knowledgeDocs"`
	Notes               bool  `json:"notes"`
	Tasks               bool  `json:"tasks"`
}

// MemoryEnabled 解析知识范围配置中的记忆开关：Memory nil（存量配置未设置）按默认启用。
// 供「回复召回」与「后台学习写入」共用同一判定，保证"关掉记忆 = 既不读也不学"的一致性
// （避免只关召回、后台还在偷偷攒记忆，重开后冒出一堆用户没预料到的记忆）。
func (s AvatarKnowledgeScope) MemoryEnabled() bool {
	return s.Memory == nil || *s.Memory
}

// MemoryEnabled 解析分身配置 KnowledgeScopeJSON 中的记忆开关（存量为空或解析失败按启用处理，
// 不因配置损坏阻断学习/召回）。内部委托给 AvatarKnowledgeScope.MemoryEnabled，判定逻辑单源。
func (c AvatarConfig) MemoryEnabled() bool {
	if c.KnowledgeScopeJSON == "" {
		return true
	}
	var scope AvatarKnowledgeScope
	if err := json.Unmarshal([]byte(c.KnowledgeScopeJSON), &scope); err != nil {
		return true
	}
	return scope.MemoryEnabled()
}

// AvatarTriggerRules 分身触发规则
//
// 与原单一 mode 枚举不同，此处将「触发时机」与「触发意图」拆成相互独立的正交开关：
//   - 时机（门）：OfflineOnly=仅离线才回；否则始终可回。
//   - 意图（门，可多选组合）：RequireMention=群里被@才回；SmartDecide=LLM 判断该不该回；
//     KeywordOnly=关键词命中才回（复用 Keywords）。均未勾选时等同回复所有消息。
//
// Mode 为遗留字段，仅用于读取存量配置时回填上述新字段（见 Normalize），不再参与决策。
type AvatarTriggerRules struct {
	OfflineOnly           bool              `json:"offlineOnly"`    // 触发时机：仅离线时才回
	RequireMention        bool              `json:"requireMention"` // 意图：群里被 @ 才回（私聊无 @ 语义，自动降级为智能判断）
	SmartDecide           bool              `json:"smartDecide"`    // 意图：LLM 判断该不该回
	KeywordOnly           bool              `json:"keywordOnly"`    // 意图：关键词命中才回（复用 Keywords）
	Keywords              []string          `json:"keywords"`
	TimeRanges            []AvatarTimeRange `json:"timeRanges"`
	ExcludedConversations []uint            `json:"excludedConversations"`
	// —— 遗留迁移 ——
	Mode string `json:"mode,omitempty"` // mention, offline, keyword, all, smart（存量配置读取时回填新字段）
}

// Normalize 将遗留的 mode 枚举回填为新正交字段（新字段全空且 mode 非空时触发）。
// 新式正交模型自洽：「均未勾选任何意图门 = 回复所有消息」。空配置（无 mode 且新字段全空）
// 同样归入「回复所有」，与 UI 上「都不勾选=回复所有」一致，不再隐式默认被 @ 才回。
func (r *AvatarTriggerRules) Normalize() {
	// 只要任一新开关已设置，就视为已是新式配置，不再回填
	if r.OfflineOnly || r.RequireMention || r.SmartDecide || r.KeywordOnly {
		return
	}
	switch r.Mode {
	case "offline":
		r.OfflineOnly = true
	case "keyword":
		r.KeywordOnly = true
	case "smart":
		r.SmartDecide = true
	case "all":
		// 所有消息：全部开关保持 false
	case "mention":
		r.RequireMention = true
	default:
		// 空或未知 mode：均未勾选 → 回复所有消息
	}
}

// AvatarTimeRange 时间范围配置
type AvatarTimeRange struct {
	DayOfWeek []int `json:"dayOfWeek"` // 0-6, 0=Sunday
	StartHour int   `json:"startHour"`
	EndHour   int   `json:"endHour"`
}

// AvatarReplyStrategy 分身回复策略
type AvatarReplyStrategy struct {
	MaxReplyLength      string  `json:"maxReplyLength"` // short, medium, long
	ReplyDelay          int     `json:"replyDelay"`     // 秒
	ConfidenceThreshold float64 `json:"confidenceThreshold"`
	DisclaimerStyle     string  `json:"disclaimerStyle"`  // badge, footer, both
	ReplyOutOfScope     bool    `json:"replyOutOfScope"`  // 是否回复知识范围外的消息，false 时静默跳过不回复
	GroupReplyTarget    string  `json:"groupReplyTarget"` // 群聊回复落点：group（默认，回群内）/ private（回触发者私聊）
}
