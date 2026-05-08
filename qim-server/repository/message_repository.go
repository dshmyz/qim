package repository

import (
	"context"
	"time"

	"qim-server/model"

	"gorm.io/gorm"
)

type messageRepository struct {
	*baseRepository[model.Message]
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{
		baseRepository: &baseRepository[model.Message]{db: db},
		db:             db,
	}
}

func (r *messageRepository) FindByConversationID(ctx context.Context, conversationID uint, limit, offset int) ([]*model.Message, error) {
	var messages []*model.Message
	query := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Where("is_recalled = ?", false).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Preload("Sender").Find(&messages).Error
	return messages, err
}

func (r *messageRepository) FindLatestByConversationID(ctx context.Context, conversationID uint) (*model.Message, error) {
	var message model.Message
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Where("is_recalled = ?", false).
		Order("created_at DESC").
		First(&message).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *messageRepository) RecallMessage(ctx context.Context, id uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_recalled": true,
			"recalled_at": now,
		}).Error
}

func (r *messageRepository) MarkAsRead(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("id = ?", id).
		Update("is_read", true).Error
}

func (r *messageRepository) WithTx(tx *gorm.DB) BaseRepository[model.Message] {
	return &messageRepository{
		baseRepository: &baseRepository[model.Message]{db: tx},
		db:             tx,
	}
}
