package service

import (
	"context"
	"time"

	"qim-server/model"
	"qim-server/repository"

	"gorm.io/gorm"
)

type NotificationService struct {
	repo repository.NotificationRepository
	db   *gorm.DB
}

func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{
		repo: repository.NewNotificationRepository(db),
		db:   db,
	}
}

func (s *NotificationService) GetNotifications(userID uint, page, pageSize int) ([]model.Notification, int64, error) {
	ctx := context.Background()
	offset := (page - 1) * pageSize

	var total int64
	s.db.WithContext(ctx).Model(&model.Notification{}).Where("user_id = ?", userID).Count(&total)

	var notifications []model.Notification
	s.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&notifications)

	return notifications, total, nil
}

func (s *NotificationService) MarkAsRead(userID, notificationID uint) (*model.Notification, error) {
	ctx := context.Background()

	var notification model.Notification
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", notificationID, userID).First(&notification).Error; err != nil {
		return nil, err
	}

	notification.Read = true
	now := time.Now()
	notification.ReadAt = &now
	s.db.WithContext(ctx).Save(&notification)

	return &notification, nil
}

func (s *NotificationService) MarkAllAsRead(userID uint) error {
	ctx := context.Background()
	return s.repo.MarkAllAsRead(ctx, userID)
}

func (s *NotificationService) ClearAll(userID uint) error {
	ctx := context.Background()
	return s.repo.Delete(ctx, userID)
}

func (s *NotificationService) GetByID(userID, notificationID uint) (*model.Notification, error) {
	ctx := context.Background()
	var notification model.Notification
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", notificationID, userID).First(&notification).Error; err != nil {
		return nil, err
	}
	return &notification, nil
}

func (s *NotificationService) Save(notification *model.Notification) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Save(notification).Error
}

func (s *NotificationService) TogglePin(userID, notificationID uint) (bool, error) {
	ctx := context.Background()
	var notification model.Notification
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", notificationID, userID).First(&notification).Error; err != nil {
		return false, err
	}
	notification.Pinned = !notification.Pinned
	s.db.WithContext(ctx).Model(&notification).Update("pinned", notification.Pinned)
	return notification.Pinned, nil
}

func (s *NotificationService) ToggleImportant(userID, notificationID uint) (bool, error) {
	ctx := context.Background()
	var notification model.Notification
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", notificationID, userID).First(&notification).Error; err != nil {
		return false, err
	}
	notification.Important = !notification.Important
	s.db.WithContext(ctx).Model(&notification).Update("important", notification.Important)
	return notification.Important, nil
}

func (s *NotificationService) Create(notification *model.Notification) error {
	ctx := context.Background()
	return s.repo.Create(ctx, notification)
}
