package model

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// 用户
type User struct {
	ID               uint           `json:"id" gorm:"primarykey"`
	Username         string         `json:"username" gorm:"uniqueIndex;size:50;not null"`
	PasswordHash     string         `json:"-" gorm:"size:255;not null"`
	Nickname         string         `json:"nickname" gorm:"size:100"`
	RealName         string         `json:"real_name" gorm:"size:100"`
	Avatar           string         `json:"avatar" gorm:"size:500"`
	Type             string         `json:"type" gorm:"size:30;default:'user';index"` // 'user' | 'bot_assistant' | 'bot_avatar' | 'system' | 'api' | 'admin'
	Gender           string         `json:"gender" gorm:"size:10;default:'secret'"`   // 'male' | 'female' | 'secret'
	Organization     string         `json:"organization" gorm:"size:500"`             // 组织架构信息（冗余存储）
	Signature        string         `json:"signature" gorm:"type:text"`
	Phone            string         `json:"phone" gorm:"size:20;index"`
	Email            string         `json:"email" gorm:"size:100;index"`
	Status           string         `json:"status" gorm:"size:20;default:'offline'"`
	LastOnline       *time.Time     `json:"last_online"`
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
	ExternalID     string         `json:"external_id" gorm:"size:200;index"`
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
	Type          string               `json:"type" gorm:"size:20;not null;index:idx_conv_type_deleted,priority:1"`
	IsDeleted     bool                 `json:"is_deleted" gorm:"default:false;index:idx_conv_type_deleted,priority:2"`
	LastMessageID *uint                `json:"last_message_id"`
	LastMessageAt *time.Time           `json:"last_message_at"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	Members       []ConversationMember `json:"members,omitempty" gorm:"foreignkey:ConversationID"`
	LastMessage   *Message             `json:"last_message,omitempty" gorm:"foreignkey:LastMessageID"`
}

// 群聊
type Group struct {
	ID               uint            `json:"id" gorm:"primarykey"`
	ConversationID   uint            `json:"conversation_id" gorm:"uniqueIndex;not null"`
	GroupType        string          `json:"group_type" gorm:"size:20;not null"` // "group" 或 "discussion"
	Name             string          `json:"name" gorm:"size:200;not null"`
	Avatar           string          `json:"avatar" gorm:"size:500"`
	CreatorID        uint            `json:"creator_id" gorm:"not null"`
	Announcement     string          `json:"announcement" gorm:"type:text"`
	InvitePermission string          `json:"invite_permission" gorm:"size:20;default:'owner_admin'"`
	AIConfigJSON     string          `json:"-" gorm:"type:text;column:ai_config"` // AI配置JSON存储
	Documents        []GroupDocument `json:"documents,omitempty" gorm:"foreignkey:GroupID"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	Conversation     Conversation    `json:"conversation,omitempty" gorm:"foreignkey:ConversationID"`
}

// // TableName 指定表名，groups 是 SQL 保留关键字，加反引号避免冲突
// func (Group) TableName() string {
// 	return "`groups`"
// }

// GroupAIConfig 群聊AI配置
type GroupAIConfig struct {
	Enabled          bool   `json:"enabled"`
	AssistantName    string `json:"assistant_name"`
	ReplyMode        string `json:"reply_mode"` // always/mention_only/smart/off
	Personality      string `json:"personality"`
	CustomPrompt     string `json:"custom_prompt"`
	Language         string `json:"language"`
	MaxLength        string `json:"max_length"`
	MentionReplyMode string `json:"mention_reply_mode"`
	AntiSpamInterval int    `json:"anti_spam_interval"`
	TriggerKeywords  string `json:"trigger_keywords"`
	LearnEnabled     bool   `json:"learn_enabled"`
	ExtractTodos     bool   `json:"extract_todos"`
}

// GetAIConfig 获取AI配置
func (g *Group) GetAIConfig() *GroupAIConfig {
	if g.AIConfigJSON == "" {
		return &GroupAIConfig{
			Enabled:          false,
			AssistantName:    "AI助手",
			ReplyMode:        "mention_only",
			Personality:      "professional",
			Language:         "auto",
			MaxLength:        "medium",
			MentionReplyMode: "mention",
			AntiSpamInterval: 5,
		}
	}
	var config GroupAIConfig
	if err := json.Unmarshal([]byte(g.AIConfigJSON), &config); err != nil {
		return &GroupAIConfig{
			Enabled:          false,
			AssistantName:    "AI助手",
			ReplyMode:        "mention_only",
			Personality:      "professional",
			Language:         "auto",
			MaxLength:        "medium",
			MentionReplyMode: "mention",
			AntiSpamInterval: 5,
		}
	}
	return &config
}

// SetAIConfig 设置AI配置
func (g *Group) SetAIConfig(config *GroupAIConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	g.AIConfigJSON = string(data)
	return nil
}

// 群聊文档关联
type GroupDocument struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	GroupID   uint      `json:"group_id" gorm:"not null;index"`
	FileID    uint      `json:"file_id" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	Group     Group     `json:"group,omitempty" gorm:"foreignkey:GroupID"`
	File      File      `json:"file,omitempty" gorm:"foreignkey:FileID"`
}

// 会话成员
type ConversationMember struct {
	ID             uint         `json:"id" gorm:"primarykey"`
	ConversationID uint         `json:"conversation_id" gorm:"not null;index:idx_conv_member_user,priority:2;uniqueIndex:idx_conv_member_conv_user,priority:1"`
	UserID         uint         `json:"user_id" gorm:"not null;index:idx_conv_member_user,priority:1;uniqueIndex:idx_conv_member_conv_user,priority:2"`
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
	ConversationID  uint           `json:"conversation_id" gorm:"not null;index:idx_msg_conv_created,priority:1;index:idx_msg_conv_read_sender,priority:1"`
	SenderID        uint           `json:"sender_id" gorm:"not null;index;index:idx_msg_conv_read_sender,priority:3"`
	Type            string         `json:"type" gorm:"size:20;not null"`
	Content         string         `json:"content" gorm:"type:mediumtext;not null"`
	QuotedMessageID *uint          `json:"quoted_message_id"`
	IsRecalled      bool           `json:"is_recalled" gorm:"default:false"`
	IsRead          bool           `json:"is_read" gorm:"default:false;index:idx_msg_conv_read_sender,priority:2"`
	AIType          string         `json:"ai_type" gorm:"size:30;default:''"` // '' | 'assistant' | 'avatar'
	RecalledAt      *time.Time     `json:"recalled_at"`
	CreatedAt       time.Time      `json:"created_at" gorm:"index:idx_msg_conv_created,priority:2"`
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
	Style     string         `json:"style" gorm:"type:text"`
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
	ID              uint           `json:"id" gorm:"primarykey"`
	Name            string         `json:"name" gorm:"size:100;not null"`
	Avatar          string         `json:"avatar" gorm:"size:500"`
	Description     string         `json:"description" gorm:"type:text"`
	Type            string         `json:"type" gorm:"size:50;not null"` // system, custom, ai
	Config          string         `json:"config" gorm:"type:text"`      // JSON配置
	IsActive        bool           `json:"is_active" gorm:"default:true"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
	CreatorID       uint           `json:"creator_id" gorm:"default:0"` // 0=系统创建
	CreatorName     string         `json:"creator_name" gorm:"size:100;default:''"`
	VirtualUserID   *uint          `json:"virtual_user_id" gorm:"index"` // 关联虚拟用户 ID
	GroupID         *uint          `json:"group_id" gorm:"index"`        // 群聊AI助手关联的群ID
	IsTemplate      bool           `json:"is_template" gorm:"default:false"`
	UserConfigID    *uint          `json:"user_config_id" gorm:"index"`
	UseSystemConfig bool           `json:"use_system_config" gorm:"default:true"`
}

// AI使用日志
type AIUsageLog struct {
	ID             uint      `json:"id" gorm:"primarykey"`
	UserID         uint      `json:"user_id" gorm:"not null;index"`
	BotID          uint      `json:"bot_id" gorm:"not null;index"`
	Provider       string    `json:"provider" gorm:"size:64"`
	Model          string    `json:"model" gorm:"size:128"`
	TaskType       string    `json:"task_type" gorm:"size:64"`
	MessagePreview string    `json:"message_preview" gorm:"size:100"`
	CallType       string    `json:"call_type" gorm:"size:20"` // chat, ops
	TokensIn       int       `json:"tokens_in"`
	TokensOut      int       `json:"tokens_out"`
	Duration       int64     `json:"duration"`
	Status         string    `json:"status" gorm:"size:32"`
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
	ID           uint           `json:"id" gorm:"primarykey"`
	UserID       uint           `json:"user_id" gorm:"not null;index"`
	Title        string         `json:"title" gorm:"size:500;not null"`
	Description  string         `json:"description" gorm:"type:text"`
	Start        time.Time      `json:"start" gorm:"column:start_time;not null"`
	End          time.Time      `json:"end" gorm:"column:end_time;not null"`
	AllDay       bool           `json:"all_day" gorm:"default:false"`
	Reminder     int            `json:"reminder" gorm:"default:0"`
	ReminderSent bool           `json:"reminder_sent" gorm:"default:false"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
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
	Priority    string         `json:"priority" gorm:"size:20;default:'medium'"`
	Status      string         `json:"status" gorm:"size:20;default:'todo'"`
	AssigneeID  string         `json:"assignee_id" gorm:"size:100"`
	Tags        string         `json:"tags" gorm:"type:text"`
	SubTasks    string         `json:"sub_tasks" gorm:"type:text"`
	Position    int            `json:"position" gorm:"default:0"`
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
	ID       uint   `json:"id" gorm:"primarykey"`
	UserID   uint   `json:"user_id" gorm:"not null;index"`
	Name     string `json:"name" gorm:"size:200;not null"`
	Code     string `json:"code" gorm:"size:100;index"` // 内置应用唯一标识，如 file_manager, calendar
	Icon     string `json:"icon" gorm:"size:500"`
	Category string `json:"category" gorm:"size:100"`
	URL      string `json:"url" gorm:"size:500"`
	Status   string `json:"status" gorm:"size:20;default:'active'"`
	OpenType string `json:"open_type" gorm:"size:20;default:'in-app'"` // in-app: 在应用内打开, external: 使用默认浏览器打开
	IsGlobal bool   `json:"is_global" gorm:"default:false"`
	// 权限范围控制
	ScopeType       string         `json:"scope_type" gorm:"size:20;default:'all'"` // all: 所有人可见, users: 指定用户, organizations: 指定组织, roles: 指定角色
	ScopeValue      string         `json:"scope_value" gorm:"size:1000"`            // 具体的范围值（逗号分隔的ID列表）
	AvailableOrgIDs string         `json:"available_org_ids" gorm:"size:1000"`      // 可用的组织ID列表（逗号分隔）
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
	User            User           `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

// 通知
type Notification struct {
	ID            uint           `json:"id" gorm:"primarykey"`
	UserID        uint           `json:"user_id" gorm:"not null;index"`
	Type          string         `json:"type" gorm:"size:30;not null"`
	Title         string         `json:"title" gorm:"size:500;not null"`
	Content       string         `json:"content" gorm:"type:text;not null"`
	Read          bool           `json:"read" gorm:"column:read;default:false"`
	ReadAt        *time.Time     `json:"read_at"`
	Priority      string         `json:"priority" gorm:"size:10;default:normal"`
	ActionType    string         `json:"action_type" gorm:"size:30;default:''"`
	ActionPayload string         `json:"action_payload" gorm:"type:text"`
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
	CommentPermission string         `json:"comment_permission" gorm:"size:20;default:'all_subscribers'"`
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

type ChannelMessageLike struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	MessageID uint           `json:"message_id" gorm:"not null;uniqueIndex:idx_msg_user"`
	UserID    uint           `json:"user_id" gorm:"not null;uniqueIndex:idx_msg_user"`
	CreatedAt time.Time      `json:"created_at"`
	Message   ChannelMessage `json:"-" gorm:"foreignkey:MessageID"`
	User      User           `json:"-" gorm:"foreignkey:UserID"`
}

type ChannelMessageComment struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	MessageID uint           `json:"message_id" gorm:"not null;index"`
	UserID    uint           `json:"user_id" gorm:"not null"`
	Content   string         `json:"content" gorm:"type:text;not null"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	Message   ChannelMessage `json:"-" gorm:"foreignkey:MessageID"`
	User      User           `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

// 短链接
type ShortLink struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	UserID      uint           `json:"user_id" gorm:"not null;index"`
	OriginalURL string         `json:"original_url" gorm:"type:text;not null"`
	Code        string         `json:"code" gorm:"size:20;uniqueIndex;not null"`
	CustomCode  string         `json:"custom_code" gorm:"size:50;index"` // 自定义短链接后缀
	ExpiresAt   *time.Time     `json:"expires_at"`                       // 过期时间
	Password    string         `json:"-" gorm:"size:255"`                // 访问密码(哈希值)
	VisitCount  int            `json:"visit_count" gorm:"default:0"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	User        User           `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

// AI提供商
type AIProvider struct {
	ID         uint        `json:"id" gorm:"primarykey"`
	Name       string      `json:"name" gorm:"size:100;not null"`
	Provider   string      `json:"provider" gorm:"size:50;not null"` // openai, anthropic, baidu, etc.
	APIType    string      `json:"api_type" gorm:"size:20;not null"` // openai, azure, claude, etc.
	Endpoint   string      `json:"endpoint" gorm:"size:500"`
	APIKey     string      `json:"api_key" gorm:"size:500"`
	Models     StringArray `json:"models" gorm:"type:text"` // JSON array of model names
	Enabled    bool        `json:"enabled" gorm:"default:true"`
	Status     string      `json:"status" gorm:"size:20;default:'connected'"` // connected, error, testing
	Priority   int         `json:"priority" gorm:"default:0"`
	Config     string      `json:"config" gorm:"type:text"` // JSON configuration
	LastTestAt *time.Time  `json:"last_test_at"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// StringArray 自定义类型用于存储字符串数组
type StringArray []string

func (s StringArray) Value() (string, error) {
	if s == nil {
		return "[]", nil
	}
	return "[" + joinStrings(s) + "]", nil
}

func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = StringArray{}
		return nil
	}
	var str string
	switch v := value.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return fmt.Errorf("StringArray.Scan: unsupported type %T", value)
	}
	return json.Unmarshal([]byte(str), s)
}

func joinStrings(strs []string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += ","
		}
		result += "\"" + s + "\""
	}
	return result
}

// AI配置
type AIConfig struct {
	ID              uint       `json:"id" gorm:"primarykey"`
	UserID          uint       `json:"user_id" gorm:"not null;index"`
	ConfigName      string     `json:"config_name" gorm:"size:50"`
	IsDefault       bool       `json:"is_default" gorm:"default:false"`
	Provider        string     `json:"provider" gorm:"size:50;default:'openai'"`
	ConfigJSON      string     `json:"-" gorm:"type:text"` // 存储 provider 特定的配置 JSON
	APIKeyEncrypted string     `json:"-" gorm:"type:text"` // 加密存储的 API Key
	ModelName       string     `json:"model_name" gorm:"size:50"`
	BaseURL         string     `json:"base_url" gorm:"size:255"`
	AIEnabled       bool       `json:"ai_enabled" gorm:"default:true"`
	DailyLimit      int        `json:"daily_limit" gorm:"default:0"`
	MaxTokens       int        `json:"max_tokens" gorm:"default:1000"`
	Temperature     float64    `json:"temperature" gorm:"default:0.7"`
	IsVerified      bool       `json:"is_verified" gorm:"default:false"`
	LastTestedAt    *time.Time `json:"last_tested_at"`
	Overrides       string     `json:"overrides" gorm:"type:json"` // JSON 序列化的 []ai.Override
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	User            User       `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

// AIProviderConfig 供应商配置接口
type AIProviderConfig interface {
	GetProvider() string
}

// OpenAIProviderConfig OpenAI 配置
type OpenAIProviderConfig struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
	BaseURL  string `json:"base_url"`
}

func (c OpenAIProviderConfig) GetProvider() string { return c.Provider }

// BaiduProviderConfig 百度配置
type BaiduProviderConfig struct {
	Provider  string `json:"provider"`
	APIKey    string `json:"api_key"`
	SecretKey string `json:"secret_key"`
	Model     string `json:"model"`
	BaseURL   string `json:"base_url"`
}

func (c BaiduProviderConfig) GetProvider() string { return c.Provider }

// AlibabaProviderConfig 阿里配置
type AlibabaProviderConfig struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
	BaseURL  string `json:"base_url"`
}

func (c AlibabaProviderConfig) GetProvider() string { return c.Provider }

// TencentProviderConfig 腾讯配置
type TencentProviderConfig struct {
	Provider  string `json:"provider"`
	SecretID  string `json:"secret_id"`
	SecretKey string `json:"secret_key"`
	Model     string `json:"model"`
	BaseURL   string `json:"base_url"`
}

func (c TencentProviderConfig) GetProvider() string { return c.Provider }

// BytedanceProviderConfig 字节跳动配置
type BytedanceProviderConfig struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
	BaseURL  string `json:"base_url"`
}

func (c BytedanceProviderConfig) GetProvider() string { return c.Provider }

// AnthropicProviderConfig Anthropic 配置
type AnthropicProviderConfig struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
	BaseURL  string `json:"base_url"`
}

func (c AnthropicProviderConfig) GetProvider() string { return c.Provider }

// GetProviderConfig 获取供应商配置
func (a *AIConfig) GetProviderConfig() (AIProviderConfig, error) {
	if a.ConfigJSON == "" {
		return nil, fmt.Errorf("配置为空")
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(a.ConfigJSON), &config); err != nil {
		return nil, err
	}

	provider, _ := config["provider"].(string)
	switch provider {
	case "openai":
		var cfg OpenAIProviderConfig
		if err := json.Unmarshal([]byte(a.ConfigJSON), &cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	case "baidu":
		var cfg BaiduProviderConfig
		if err := json.Unmarshal([]byte(a.ConfigJSON), &cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	case "alibaba":
		var cfg AlibabaProviderConfig
		if err := json.Unmarshal([]byte(a.ConfigJSON), &cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	case "tencent":
		var cfg TencentProviderConfig
		if err := json.Unmarshal([]byte(a.ConfigJSON), &cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	case "bytedance":
		var cfg BytedanceProviderConfig
		if err := json.Unmarshal([]byte(a.ConfigJSON), &cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	case "anthropic":
		var cfg AnthropicProviderConfig
		if err := json.Unmarshal([]byte(a.ConfigJSON), &cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	default:
		return nil, fmt.Errorf("不支持的供应商: %s", provider)
	}
}

// SetProviderConfig 设置供应商配置
func (a *AIConfig) SetProviderConfig(cfg AIProviderConfig) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	a.ConfigJSON = string(data)
	a.Provider = cfg.GetProvider()
	return nil
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
	ConfigKey string    `json:"key" gorm:"column:config_key;size:100;uniqueIndex;not null"`
	Value     string    `json:"value" gorm:"type:text;not null"`
	Type      string    `json:"type" gorm:"size:20;default:'string'"` // string, number, boolean, json
	Desc      string    `json:"desc" gorm:"column:description;size:500"`
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
	Version     string         `json:"version" gorm:"size:50;uniqueIndex:idx_version_deleted;not null"`
	Platform    string         `json:"platform" gorm:"size:20;not null"`   // windows, mac, linux
	Type        string         `json:"type" gorm:"size:20;default:'full'"` // full, patch
	DownloadURL string         `json:"download_url" gorm:"size:500"`
	Sha512      string         `json:"sha512" gorm:"size:200"`     // 文件 SHA512 哈希（缓存）
	FileSize    int64          `json:"file_size" gorm:"default:0"` // 文件大小（缓存）
	Changelog   string         `json:"changelog" gorm:"type:text"`
	ForceUpdate bool           `json:"force_update" gorm:"default:false"`
	Enabled     bool           `json:"enabled" gorm:"default:true"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"uniqueIndex:idx_version_deleted"`
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
