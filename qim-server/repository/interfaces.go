package repository

import (
	"context"

	"qim-server/model"

	"gorm.io/gorm"
)

type BaseRepository[T any] interface {
	Create(ctx context.Context, entity *T) error
	CreateBatch(ctx context.Context, entities []*T) error
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id uint) error
	HardDelete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*T, error)
	FindAll(ctx context.Context) ([]*T, error)
	Count(ctx context.Context) (int64, error)
	Exists(ctx context.Context, id uint) (bool, error)
	DB() *gorm.DB
	WithTx(tx *gorm.DB) BaseRepository[T]
}

type UserRepository interface {
	BaseRepository[model.User]
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	FindByPhone(ctx context.Context, phone string) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	Search(ctx context.Context, query string, limit int) ([]*model.User, error)
	UpdateStatus(ctx context.Context, id uint, status string) error
	UpdateLastOnline(ctx context.Context, id uint) error
}

type ConversationRepository interface {
	BaseRepository[model.Conversation]
	FindByUserID(ctx context.Context, userID uint) ([]*model.Conversation, error)
	FindSingleConversation(ctx context.Context, userID1, userID2 uint) (*model.Conversation, error)
	AddMember(ctx context.Context, conversationID, userID uint, role string) error
	RemoveMember(ctx context.Context, conversationID, userID uint) error
	UpdateMemberRole(ctx context.Context, conversationID, userID uint, role string) error
	IsMember(ctx context.Context, conversationID, userID uint) (bool, error)
	GetMembers(ctx context.Context, conversationID uint) ([]model.ConversationMember, error)
	SetMute(ctx context.Context, conversationID, userID uint, muted bool) (*model.ConversationMember, error)
}

type MessageRepository interface {
	BaseRepository[model.Message]
	FindByConversationID(ctx context.Context, conversationID uint, limit, offset int) ([]*model.Message, error)
	FindLatestByConversationID(ctx context.Context, conversationID uint) (*model.Message, error)
	RecallMessage(ctx context.Context, id uint) error
	MarkAsRead(ctx context.Context, id uint) error
}

type GroupRepository interface {
	BaseRepository[model.Group]
	FindByConversationID(ctx context.Context, conversationID uint) (*model.Group, error)
	FindByCreatorID(ctx context.Context, creatorID uint) ([]*model.Group, error)
	UpdateAnnouncement(ctx context.Context, id uint, announcement string) error
	AddMember(ctx context.Context, groupID, userID uint) error
	RemoveMember(ctx context.Context, groupID, userID uint) error
}

type FileRepository interface {
	BaseRepository[model.File]
	FindByUserID(ctx context.Context, userID uint) ([]*model.File, error)
	FindByFolderID(ctx context.Context, folderID *uint) ([]*model.File, error)
	FindByChecksum(ctx context.Context, checksum string) (*model.File, error)
	UpdateStarred(ctx context.Context, id uint, starred bool) error
}

type NotificationRepository interface {
	BaseRepository[model.Notification]
	FindByUserID(ctx context.Context, userID uint, unreadOnly bool) ([]*model.Notification, error)
	MarkAsRead(ctx context.Context, id uint) error
	MarkAllAsRead(ctx context.Context, userID uint) error
	CountUnread(ctx context.Context, userID uint) (int64, error)
}

type EventRepository interface {
	BaseRepository[model.Event]
	FindByUserID(ctx context.Context, userID uint) ([]*model.Event, error)
	FindByUserIDAndID(ctx context.Context, userID, id uint) (*model.Event, error)
	DeleteByUserIDAndID(ctx context.Context, userID, id uint) error
}

type TaskRepository interface {
	BaseRepository[model.Task]
	FindByUserID(ctx context.Context, userID uint) ([]*model.Task, error)
	FindByUserIDAndID(ctx context.Context, userID, id uint) (*model.Task, error)
	DeleteByUserIDAndID(ctx context.Context, userID, id uint) error
}
