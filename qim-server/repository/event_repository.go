package repository

import (
	"context"

	"github.com/dshmyz/qim/qim-server/model"

	"gorm.io/gorm"
)

type eventRepository struct {
	*baseRepository[model.Event]
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) EventRepository {
	return &eventRepository{
		baseRepository: &baseRepository[model.Event]{db: db},
		db:             db,
	}
}

func (r *eventRepository) FindByUserID(ctx context.Context, userID uint) ([]*model.Event, error) {
	var events []*model.Event
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("start_time DESC").Find(&events).Error
	return events, err
}

func (r *eventRepository) FindByUserIDAndID(ctx context.Context, userID, id uint) (*model.Event, error) {
	var event model.Event
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&event).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *eventRepository) DeleteByUserIDAndID(ctx context.Context, userID, id uint) error {
	return r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&model.Event{}).Error
}

func (r *eventRepository) WithTx(tx *gorm.DB) BaseRepository[model.Event] {
	return &eventRepository{
		baseRepository: &baseRepository[model.Event]{db: tx},
		db:             tx,
	}
}
