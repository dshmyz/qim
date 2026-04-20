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
	ParentID       *uint          `json:"parent_id"`
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
	Name          string               `json:"name" gorm:"size:200"`
	Avatar        string               `json:"avatar" gorm:"size:500"`
	CreatorID     uint                 `json:"creator_id"`
	Announcement  string               `json:"announcement" gorm:"type:text"`
	LastMessageID *uint                `json:"last_message_id"`
	LastMessageAt *time.Time           `json:"last_message_at"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	Members       []ConversationMember `json:"members,omitempty" gorm:"foreignkey:ConversationID"`
	LastMessage   *Message             `json:"last_message,omitempty" gorm:"foreignkey:LastMessageID"`
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
	Content         string         `json:"content" gorm:"type:text;not null"`
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
	FolderID     *uint          `json:"folder_id"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

// 文件夹
type Folder struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	UserID    uint           `json:"user_id" gorm:"not null;index"`
	Name      string         `json:"name" gorm:"size:255;not null"`
	ParentID  *uint          `json:"parent_id"`
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
	Color     string         `json:"color" gorm:"size:20;default:'yellow'"`
	Type      string         `json:"type" gorm:"size:20;default:'note'"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// 会话记录（用于置顶、排序等）
type ConversationSession struct {
	ID             uint       `json:"id" gorm:"primarykey"`
	UserID         uint       `json:"user_id" gorm:"not null;index;uniqueIndex:idx_user_conv"`
	ConversationID uint       `json:"conversation_id" gorm:"not null;index;uniqueIndex:idx_user_conv"`
	IsPinned       bool       `json:"is_pinned" gorm:"default:false"`
	PinnedAt       *time.Time `json:"pinned_at"`
	LastVisitedAt  time.Time  `json:"last_visited_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

// 消息已读回执
type MessageReadReceipt struct {
	ID             uint      `json:"id" gorm:"primarykey"`
	MessageID      uint      `json:"message_id" gorm:"not null;index"`
	ConversationID uint      `json:"conversation_id" gorm:"not null;index"`
	UserID         uint      `json:"user_id" gorm:"not null;index"`
	CreatedAt      time.Time `json:"created_at"`
	User           *User     `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

// 机器人
type Bot struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	Name        string    `json:"name" gorm:"size:100;not null"`
	Avatar      string    `json:"avatar" gorm:"size:500"`
	Description string    `json:"description" gorm:"type:text"`
	Type        string    `json:"type" gorm:"size:50;not null"` // system, custom, ai
	Config      string    `json:"config" gorm:"type:text"`      // JSON配置
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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

// 小程序
type MiniApp struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	AppID       string         `json:"app_id" gorm:"size:100;uniqueIndex;not null"`
	Name        string         `json:"name" gorm:"size:200;not null"`
	Description string         `json:"description" gorm:"type:text"`
	Icon        string         `json:"icon" gorm:"size:500"`
	Path        string         `json:"path" gorm:"size:500"`
	Status      string         `json:"status" gorm:"size:20;default:'active'"`
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
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	User      User           `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

// 通知
type Notification struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	UserID    uint           `json:"user_id" gorm:"not null;index"`
	Type      string         `json:"type" gorm:"size:20;not null"`
	Title     string         `json:"title" gorm:"size:500;not null"`
	Content   string         `json:"content" gorm:"type:text;not null"`
	Read      bool           `json:"read" gorm:"default:false"`
	ReadAt    *time.Time     `json:"read_at"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	User      User           `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

// 频道
type Channel struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	Name        string         `json:"name" gorm:"size:200;not null"`
	Description string         `json:"description" gorm:"type:text"`
	Avatar      string         `json:"avatar" gorm:"size:500"`
	CreatorID   uint           `json:"creator_id" gorm:"not null"`
	Status      string         `json:"status" gorm:"size:20;default:'active'"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	Creator     User           `json:"creator,omitempty" gorm:"foreignkey:CreatorID"`
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
