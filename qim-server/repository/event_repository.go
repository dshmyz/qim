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

// FindByUserIDLimited 最近 limit 条 + 总数（SQL 端 LIMIT，避免全量物化后内存截断）。
// 供 AI list_calendar_events 工具使用；CalendarApp 的 REST 全量拉取不经过此方法。
func (r *eventRepository) FindByUserIDLimited(ctx context.Context, userID uint, limit int) ([]*model.Event, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.Event{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var events []*model.Event
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("start_time DESC").Limit(limit).Find(&events).Error
	return events, total, err
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
