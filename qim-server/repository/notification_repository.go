package repository

import (
	"context"
	"time"

	"qim-server/model"

	"gorm.io/gorm"
)

type notificationRepository struct {
	*baseRepository[model.Notification]
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{
		baseRepository: &baseRepository[model.Notification]{db: db},
		db:             db,
	}
}

func (r *notificationRepository) FindByUserID(ctx context.Context, userID uint, unreadOnly bool) ([]*model.Notification, error) {
	var notifications []*model.Notification
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)

	if unreadOnly {
		query = query.Where("read = ?", false)
	}

	err := query.Order("created_at DESC").Find(&notifications).Error
	return notifications, err
}

func (r *notificationRepository) MarkAsRead(ctx context.Context, id uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"read":    true,
			"read_at": now,
		}).Error
}

func (r *notificationRepository) MarkAllAsRead(ctx context.Context, userID uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Updates(map[string]interface{}{
			"read":    true,
			"read_at": now,
		}).Error
}

func (r *notificationRepository) CountUnread(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Count(&count).Error
	return count, err
}

func (r *notificationRepository) WithTx(tx *gorm.DB) BaseRepository[model.Notification] {
	return &notificationRepository{
		baseRepository: &baseRepository[model.Notification]{db: tx},
		db:             tx,
	}
}
