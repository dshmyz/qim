package model

import (
	"time"

	"gorm.io/gorm"
)

// 用户
type User struct {
	ID               uint           `json:"id" gorm:"primarykey"`
	Username         string         `json:"username" gorm:"uniqueIndex;size:50;not null"`
	PasswordHash     string         `json:"-" gorm:"size:255;not null"`
	Nickname         string         `json:"nickname" gorm:"size:100"`
	Avatar           string         `json:"avatar" gorm:"size:500"`
	Signature        string         `json:"signature" gorm:"type:text"`
	Phone            string         `json:"phone" gorm:"size:20"`
	Email            string         `json:"email" gorm:"size:100"`
	Status           string         `json:"status" gorm:"size:20;default:'offline'"`
	IP               string         `json:"ip" gorm:"size:50"`
	TwoFactorEnabled bool           `json:"two_factor_enabled" gorm:"default:false"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

// 部门
type Department struct {
	ID             uint           `json:"id" gorm:"primarykey"`
	Name           string         `json:"name" gorm:"size:100;not null"`
	ParentID       *uint          `json:"parent_id" gorm:"index"`
	Level          int            `json:"level" gorm:"not null"`
	Path           string         `json:"path" gorm:"size:500"`
	SortOrder      int            `json:"sort_order" gorm:"default:0"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
	SubDepartments []Department   `json:"subDepartments,omitempty" gorm:"foreignkey:ParentID"`
	Employees      []User         `json:"employees,omitempty" gorm:"many2many:department_employees"`
}

// 部门员工关联
type DepartmentEmployee struct {
	ID           uint       `json:"id" gorm:"primarykey"`
	UserID       uint       `json:"user_id" gorm:"not null;index"`
	DepartmentID uint       `json:"department_id" gorm:"not null;index"`
	Position     string     `json:"position" gorm:"size:100"`
	IsPrimary    bool       `json:"is_primary" gorm:"default:true"`
	CreatedAt    time.Time  `json:"created_at"`
	User         User       `json:"user,omitempty" gorm:"foreignkey:UserID"`
	Department   Department `json:"department,omitempty" gorm:"foreignkey:DepartmentID"`
}

// 会话
type Conversation struct {
	ID            uint                 `json:"id" gorm:"primarykey"`
	Type          string               `json:"type" gorm:"size:20;not null"`
	IsDeleted     bool                 `json:"is_deleted" gorm:"default:false"`
	LastMessageID *uint                `json:"last_message_id"`
	LastMessageAt *time.Time           `json:"last_message_at"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	Members       []ConversationMember `json:"members,omitempty" gorm:"foreignkey:ConversationID"`
	LastMessage   *Message             `json:"last_message,omitempty" gorm:"foreignkey:LastMessageID"`
}

// 群聊
type Group struct {
	ID                 uint            `json:"id" gorm:"primarykey"`
	ConversationID     uint            `json:"conversation_id" gorm:"uniqueIndex;not null"`
	GroupType          string          `json:"group_type" gorm:"size:20;not null"` // "group" 或 "discussion"
	Name               string          `json:"name" gorm:"size:200;not null"`
	Avatar             string          `json:"avatar" gorm:"size:500"`
	CreatorID          uint            `json:"creator_id" gorm:"not null"`
	Announcement       string          `json:"announcement" gorm:"type:text"`
	InvitePermission   string          `json:"invite_permission" gorm:"size:20;default:'owner_admin'"`
	AIEnabled          bool            `json:"ai_enabled" gorm:"default:false"`
	AIReplyMode        string          `json:"ai_reply_mode" gorm:"size:20;default:'mention_only'"` // always/mention_only/smart/off
	AIAssistantName    string          `json:"ai_assistant_name" gorm:"size:100;default:'AI助手'"`
	AIPersonality      string          `json:"ai_personality" gorm:"size:20;default:'professional'"`
	AICustomPrompt     string          `json:"ai_custom_prompt" gorm:"type:text"`
	AILanguage         string          `json:"ai_language" gorm:"size:10;default:'auto'"`
	AIMaxLength        string          `json:"ai_max_length" gorm:"size:10;default:'medium'"`
	AIMentionReplyMode string          `json:"ai_mention_reply_mode" gorm:"size:10;default:'mention'"`
	AIAntiSpamInterval int             `json:"ai_anti_spam_interval" gorm:"default:5"`
	AITriggerKeywords  string          `json:"ai_trigger_keywords" gorm:"type:text"`
	AILearnEnabled     bool            `json:"ai_learn_enabled" gorm:"default:false"`
	Documents          []GroupDocument `json:"documents,omitempty" gorm:"foreignkey:GroupID"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	Conversation       Conversation    `json:"conversation,omitempty" gorm:"foreignkey:ConversationID"`
}

// 群聊文档关联
type GroupDocument struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	GroupID   uint      `json:"group_id" gorm:"not null;index"`
	FileID    uint      `json:"file_id" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	Group     Group     `json:"group,omitempty" gorm:"foreignkey:GroupID"`
}

// 会话成员
type ConversationMember struct {
	ID             uint         `json:"id" gorm:"primarykey"`
	ConversationID uint         `json:"conversation_id" gorm:"not null;index"`
	UserID         uint         `json:"user_id" gorm:"not null;index"`
	Role           string       `json:"role" gorm:"size:20;default:'member'"`
	UnreadCount    int          `json:"unread_count" gorm:"default:0"`
	Muted          bool         `json:"muted" gorm:"default:false"`
	LastReadAt     *time.Time   `json:"last_read_at"`
	JoinedAt       time.Time    `json:"joined_at"`
	User           User         `json:"user,omitempty" gorm:"foreignkey:UserID"`
	Conversation   Conversation `json:"conversation,omitempty" gorm:"foreignkey:ConversationID"`
}

// 消息
type Message struct {
	ID              uint           `json:"id" gorm:"primarykey"`
	ConversationID  uint           `json:"conversation_id" gorm:"not null;index"`
	SenderID        uint           `json:"sender_id" gorm:"not null;index"`
	Type            string         `json:"type" gorm:"size:20;not null"`
	Content         string         `json:"content" gorm:"type:mediumtext;not null"`
	QuotedMessageID *uint          `json:"quoted_message_id"`
	IsRecalled      bool           `json:"is_recalled" gorm:"default:false"`
	IsRead          bool           `json:"is_read" gorm:"default:false"`
	RecalledAt      *time.Time     `json:"recalled_at"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
	Sender          User           `json:"sender,omitempty" gorm:"foreignkey:SenderID"`
	QuotedMessage   *Message       `json:"quoted_message,omitempty" gorm:"foreignkey:QuotedMessageID"`
}

// 文件
type File struct {
	ID           uint           `json:"id" gorm:"primarykey"`
	UserID       uint           `json:"user_id" gorm:"not null;index"`
	Name         string         `json:"name" gorm:"size:255;not null"`
	OriginalName string         `json:"original_name" gorm:"size:255"`
	Size         int64          `json:"size" gorm:"not null"`
	MimeType     string         `json:"mime_type" gorm:"size:100"`
	StoragePath  string         `json:"storage_path" gorm:"size:500;not null"`
	Checksum     string         `json:"checksum" gorm:"size:64"`
	FolderID     *uint          `json:"folder_id" gorm:"index"`
	Source       string         `json:"source" gorm:"size:20;default:'upload'"`
	SourceID     string         `json:"source_id" gorm:"size:100"`
	IsStarred    bool           `json:"is_starred" gorm:"default:false"`
	StarredAt    *time.Time     `json:"starred_at"`
	Tags         string         `json:"tags" gorm:"size:500"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

// 文件夹
type Folder struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	UserID    uint           `json:"user_id" gorm:"not null;index"`
	Name      string         `json:"name" gorm:"size:255;not null"`
	ParentID  *uint          `json:"parent_id" gorm:"index"`
	SortOrder int            `json:"sort_order" gorm:"default:0"`
	Icon      string         `json:"icon" gorm:"size:50"`
	Color     string         `json:"color" gorm:"size:20"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// 笔记
type Note struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	UserID    uint           `json:"user_id" gorm:"not null;index"`
	Title     string         `json:"title" gorm:"size:500;not null"`
	Content   string         `json:"content" gorm:"type:text;not null"`
	Type      string         `json:"type" gorm:"size:20;default:'note'"`
	Style     string         `json:"style" gorm:"type:text;default:'{}'"`
	Tags      string         `json:"tags" gorm:"type:text"`    // JSON 数组字符串
	Summary   string         `json:"summary" gorm:"type:text"` // AI 生成的摘要
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// 会话记录（用于置顶、隐藏等）
type ConversationSession struct {
	ID             uint       `json:"id" gorm:"primarykey"`
	UserID         uint       `json:"user_id" gorm:"not null;index;uniqueIndex:idx_user_conv"`
	ConversationID uint       `json:"conversation_id" gorm:"not null;index;uniqueIndex:idx_user_conv"`
	IsPinned       bool       `json:"is_pinned" gorm:"default:false"`
	IsHidden       bool       `json:"is_hidden" gorm:"default:false"`
	PinnedAt       *time.Time `json:"pinned_at"`
	HiddenAt       *time.Time `json:"hidden_at"`
	LastVisitedAt  time.Time  `json:"last_visited_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

// 消息已读回执
type MessageReadReceipt struct {
	ID             uint      `json:"id" gorm:"primarykey"`
	MessageID      uint      `json:"message_id" gorm:"not null;uniqueIndex:idx_message_user_receipt"`
	ConversationID uint      `json:"conversation_id" gorm:"not null;index"`
	UserID         uint      `json:"user_id" gorm:"not null;uniqueIndex:idx_message_user_receipt"`
	CreatedAt      time.Time `json:"created_at"`
	User           *User     `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

// 机器人
type Bot struct {
	ID              uint      `json:"id" gorm:"primarykey"`
	Name            string    `json:"name" gorm:"size:100;not null"`
	Avatar          string    `json:"avatar" gorm:"size:500"`
	Description     string    `json:"description" gorm:"type:text"`
	Type            string    `json:"type" gorm:"size:50;not null"` // system, custom, ai
	Config          string    `json:"config" gorm:"type:text"`      // JSON配置
	IsActive        bool      `json:"is_active" gorm:"default:true"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	ApprovalStatus  string    `json:"approval_status" gorm:"size:20;default:'approved'"` // pending, approved, rejected
	CreatorID       uint      `json:"creator_id" gorm:"default:0"`                       // 0=系统创建
	CreatorName     string    `json:"creator_name" gorm:"size:100;default:''"`
	RejectReason    string    `json:"reject_reason" gorm:"type:text"`
	IsTemplate      bool      `json:"is_template" gorm:"default:false"`
	UserConfigID    *uint     `json:"user_config_id" gorm:"index"`
	UseSystemConfig bool      `json:"use_system_config" gorm:"default:true"`
}

// AI使用日志
type AIUsageLog struct {
	ID             uint      `json:"id" gorm:"primarykey"`
	UserID         uint      `json:"user_id" gorm:"not null;index"`
	BotID          uint      `json:"bot_id" gorm:"not null;index"`
	MessagePreview string    `json:"message_preview" gorm:"size:100"`
	CallType       string    `json:"call_type" gorm:"size:20"` // chat, ops
	CreatedAt      time.Time `json:"created_at"`
}

// 机器人会话
type BotConversation struct {
	ID             uint         `json:"id" gorm:"primarykey"`
	BotID          uint         `json:"bot_id" gorm:"not null;index"`
	UserID         uint         `json:"user_id" gorm:"not null;index"`
	ConversationID uint         `json:"conversation_id" gorm:"not null;index"`
	CreatedAt      time.Time    `json:"created_at"`
	Bot            Bot          `json:"bot,omitempty" gorm:"foreignkey:BotID"`
	User           User         `json:"user,omitempty" gorm:"foreignkey:UserID"`
	Conversation   Conversation `json:"conversation,omitempty" gorm:"foreignkey:ConversationID"`
}

// 日历事件
type Event struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	UserID      uint           `json:"user_id" gorm:"not null;index"`
	Title       string         `json:"title" gorm:"size:500;not null"`
	Description string         `json:"description" gorm:"type:text"`
	Start       time.Time      `json:"start" gorm:"not null"`
	End         time.Time      `json:"end" gorm:"not null"`
	AllDay      bool           `json:"all_day" gorm:"default:false"`
	Reminder    int            `json:"reminder" gorm:"default:0"` // 提醒时间（分钟）
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// 用户角色
type UserRole struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	UserID    uint      `json:"user_id" gorm:"not null;index;uniqueIndex:idx_user_role"`
	Role      string    `json:"role" gorm:"size:50;not null;uniqueIndex:idx_user_role"` // system_admin, system_publisher, etc.
	CreatedAt time.Time `json:"created_at"`
	User      User      `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

// 系统消息
type SystemMessage struct {
	ID         uint           `json:"id" gorm:"primarykey"`
	Title      string         `json:"title" gorm:"size:500;not null"`
	Content    string         `json:"content" gorm:"type:text;not null"`
	SenderID   uint           `json:"sender_id" gorm:"not null"`
	Status     string         `json:"status" gorm:"size:20;default:'active'"`
	TargetType string         `json:"target_type" gorm:"size:20"`
	TargetID   *uint          `json:"target_id"`
	CreatedAt  time.Time      `json:"created_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
	Sender     User           `json:"sender,omitempty" gorm:"foreignkey:SenderID"`
}

// 任务
type Task struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	UserID      uint           `json:"user_id" gorm:"not null;index"`
	Title       string         `json:"title" gorm:"size:500;not null"`
	Description string         `json:"description" gorm:"type:text"`
	DueDate     *time.Time     `json:"due_date"`
	Priority    string         `json:"priority" gorm:"size:20;default:'medium'"` // low, medium, high
	Status      string         `json:"status" gorm:"size:20;default:'todo'"`     // todo, in_progress, completed
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	User        User           `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

// 小程序
type MiniApp struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	AppID       string         `json:"app_id" gorm:"size:100;uniqueIndex;not null"`
	Name        string         `json:"name" gorm:"size:200;not null"`
	Description string         `json:"description" gorm:"type:text"`
	Icon        string         `json:"icon" gorm:"size:500"`
	Path        string         `json:"path" gorm:"size:500"`
	Status      string         `json:"status" gorm:"size:20;default:'inactive'"`
	Permissions string         `json:"permissions" gorm:"type:text"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// 应用
type App struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	UserID    uint           `json:"user_id" gorm:"not null;index"`
	Name      string         `json:"name" gorm:"size:200;not null"`
	Icon      string         `json:"icon" gorm:"size:500"`
	Category  string         `json:"category" gorm:"size:100"`
	URL       string         `json:"url" gorm:"size:500"`
	Status    string         `json:"status" gorm:"size:20;default:'active'"`
	OpenType  string         `json:"open_type" gorm:"size:20;default:'in-app'"` // in-app: 在应用内打开, external: 使用默认浏览器打开
	IsGlobal  bool           `json:"is_global" gorm:"default:false"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	User      User           `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

// 通知
type Notification struct {
	ID            uint           `json:"id" gorm:"primarykey"`
	UserID        uint           `json:"user_id" gorm:"not null;index"`
	Type          string         `json:"type" gorm:"size:30;not null"`
	Title         string         `json:"title" gorm:"size:500;not null"`
	Content       string         `json:"content" gorm:"type:text;not null"`
	Read          bool           `json:"read" gorm:"default:false"`
	ReadAt        *time.Time     `json:"read_at"`
	Priority      string         `json:"priority" gorm:"size:10;default:normal"`
	ActionType    string         `json:"action_type" gorm:"size:30;default:''"`
	ActionPayload string         `json:"action_payload" gorm:"type:text;default:''"`
	Pinned        bool           `json:"pinned" gorm:"default:false"`
	Important     bool           `json:"important" gorm:"default:false"`
	Handled       bool           `json:"handled" gorm:"default:false"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
	User          User           `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

// 频道
type Channel struct {
	ID                uint           `json:"id" gorm:"primarykey"`
	Name              string         `json:"name" gorm:"size:200;not null"`
	Description       string         `json:"description" gorm:"type:text"`
	Avatar            string         `json:"avatar" gorm:"size:500"`
	CreatorID         uint           `json:"creator_id" gorm:"not null"`
	Status            string         `json:"status" gorm:"size:20;default:'active'"`
	PublishPermission string         `json:"publish_permission" gorm:"size:20;default:'creator_only'"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
	Creator           User           `json:"creator,omitempty" gorm:"foreignkey:CreatorID"`
}

// 频道订阅者
type ChannelSubscriber struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	ChannelID uint      `json:"channel_id" gorm:"not null;index;uniqueIndex:idx_channel_user"`
	UserID    uint      `json:"user_id" gorm:"not null;index;uniqueIndex:idx_channel_user"`
	JoinedAt  time.Time `json:"joined_at"`
	Channel   Channel   `json:"channel,omitempty" gorm:"foreignkey:ChannelID"`
	User      User      `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

// 频道消息
type ChannelMessage struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	ChannelID uint           `json:"channel_id" gorm:"not null;index"`
	SenderID  uint           `json:"sender_id" gorm:"not null"`
	Content   string         `json:"content" gorm:"type:text;not null"`
	Type      string         `json:"type" gorm:"size:20;not null;default:'text'"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	Channel   Channel        `json:"channel,omitempty" gorm:"foreignkey:ChannelID"`
	Sender    User           `json:"sender,omitempty" gorm:"foreignkey:SenderID"`
}

// 短链接
type ShortLink struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	UserID      uint           `json:"user_id" gorm:"not null;index"`
	OriginalURL string         `json:"original_url" gorm:"type:text;not null"`
	Code        string         `json:"code" gorm:"size:20;uniqueIndex;not null"`
	VisitCount  int            `json:"visit_count" gorm:"default:0"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	User        User           `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

// AI配置
type AIConfig struct {
	ID               uint   `json:"id" gorm:"primarykey"`
	UserID           uint   `json:"user_id" gorm:"not null;uniqueIndex"`
	Provider         string `json:"provider" gorm:"size:50;default:'openai'"`
	OpenAIAPIKey     string `json:"openai_api_key" gorm:"size:500"`
	OpenAIModel      string `json:"openai_model" gorm:"size:100"`
	OpenAIBaseURL    string `json:"openai_base_url" gorm:"size:500"`
	BaiduAPIKey      string `json:"baidu_api_key" gorm:"size:500"`
	BaiduSecretKey   string `json:"baidu_secret_key" gorm:"size:500"`
	BaiduModel       string `json:"baidu_model" gorm:"size:100"`
	BaiduBaseURL     string `json:"baidu_base_url" gorm:"size:500"`
	AlibabaAPIKey    string `json:"alibaba_api_key" gorm:"size:500"`
	AlibabaModel     string `json:"alibaba_model" gorm:"size:100"`
	AlibabaBaseURL   string `json:"alibaba_base_url" gorm:"size:500"`
	TencentSecretID  string `json:"tencent_secret_id" gorm:"size:500"`
	TencentSecretKey string `json:"tencent_secret_key" gorm:"size:500"`
	TencentModel     string `json:"tencent_model" gorm:"size:100"`
	TencentBaseURL   string `json:"tencent_base_url" gorm:"size:500"`
	BytedanceAPIKey  string `json:"bytedance_api_key" gorm:"size:500"`
	BytedanceModel   string `json:"bytedance_model" gorm:"size:100"`
	BytedanceBaseURL string `json:"bytedance_base_url" gorm:"size:500"`
	AnthropicAPIKey  string `json:"anthropic_api_key" gorm:"size:500"`
	AnthropicModel   string `json:"anthropic_model" gorm:"size:100"`
	AnthropicBaseURL string `json:"anthropic_base_url" gorm:"size:500"`
	// 全局管控字段
	AIEnabled   bool      `json:"ai_enabled" gorm:"default:true"` // 是否可以使用 AI
	DailyLimit  int       `json:"daily_limit" gorm:"default:0"`   // 每日调用限制，0=不限制
	MaxTokens   int       `json:"max_tokens" gorm:"default:1000"`
	Temperature float64   `json:"temperature" gorm:"default:0.7"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	User        User      `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

// 用户AI配置
type UserAIConfig struct {
	ID              uint       `json:"id" gorm:"primarykey"`
	UserID          uint       `json:"user_id" gorm:"not null;index"`
	ConfigName      string     `json:"config_name" gorm:"size:50;not null"`
	Provider        string     `json:"provider" gorm:"size:20;not null"`
	APIKeyEncrypted string     `json:"-" gorm:"type:text;not null"`
	ModelName       string     `json:"model_name" gorm:"size:50;not null"`
	BaseURL         string     `json:"base_url" gorm:"size:255"`
	Temperature     float64    `json:"temperature" gorm:"default:0.7"`
	MaxTokens       int        `json:"max_tokens" gorm:"default:1000"`
	IsVerified      bool       `json:"is_verified" gorm:"default:false"`
	LastTestedAt    *time.Time `json:"last_tested_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	User            User       `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

// 敏感词
type SensitiveWord struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	Word      string         `json:"word" gorm:"size:100;uniqueIndex;not null"`
	Level     string         `json:"level" gorm:"size:20;default:'medium'"` // low, medium, high
	Enabled   bool           `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// 系统配置
type SystemConfig struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	Key       string    `json:"key" gorm:"size:100;uniqueIndex;not null"`
	Value     string    `json:"value" gorm:"type:text;not null"`
	Type      string    `json:"type" gorm:"size:20;default:'string'"` // string, number, boolean, json
	Desc      string    `json:"desc" gorm:"size:500"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// 操作日志
type OperationLog struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	UserID      uint      `json:"user_id" gorm:"not null;index"`
	Username    string    `json:"username" gorm:"size:100"`
	Action      string    `json:"action" gorm:"size:100;not null"`
	Module      string    `json:"module" gorm:"size:50"`
	IP          string    `json:"ip" gorm:"size:50"`
	UserAgent   string    `json:"user_agent" gorm:"type:text"`
	RequestURL  string    `json:"request_url" gorm:"size:500"`
	RequestBody string    `json:"request_body" gorm:"type:text"`
	Response    string    `json:"response" gorm:"type:text"`
	Duration    int       `json:"duration"` // ms
	CreatedAt   time.Time `json:"created_at"`
}

// 客户端版本
type ClientVersion struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	Version     string         `json:"version" gorm:"size:50;uniqueIndex;not null"`
	Platform    string         `json:"platform" gorm:"size:20;not null"`   // windows, mac, linux
	Type        string         `json:"type" gorm:"size:20;default:'full'"` // full, patch
	DownloadURL string         `json:"download_url" gorm:"size:500"`
	Changelog   string         `json:"changelog" gorm:"type:text"`
	ForceUpdate bool           `json:"force_update" gorm:"default:false"`
	Enabled     bool           `json:"enabled" gorm:"default:true"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// 黑名单
type Blacklist struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	UserID    uint      `json:"user_id" gorm:"not null;index;uniqueIndex:idx_blacklist_user"`
	Reason    string    `json:"reason" gorm:"type:text"`
	Operator  string    `json:"operator" gorm:"size:100"`
	CreatedAt time.Time `json:"created_at"`
	User      User      `json:"user,omitempty" gorm:"foreignkey:UserID"`
}
