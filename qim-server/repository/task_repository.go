package repository

import (
	"context"

	"qim-server/model"

	"gorm.io/gorm"
)

type taskRepository struct {
	*baseRepository[model.Task]
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{
		baseRepository: &baseRepository[model.Task]{db: db},
		db:             db,
	}
}

func (r *taskRepository) FindByUserID(ctx context.Context, userID uint) ([]*model.Task, error) {
	var tasks []*model.Task
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&tasks).Error
	return tasks, err
}

func (r *taskRepository) FindByUserIDAndID(ctx context.Context, userID, id uint) (*model.Task, error) {
	var task model.Task
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *taskRepository) DeleteByUserIDAndID(ctx context.Context, userID, id uint) error {
	return r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&model.Task{}).Error
}

func (r *taskRepository) WithTx(tx *gorm.DB) BaseRepository[model.Task] {
	return &taskRepository{
		baseRepository: &baseRepository[model.Task]{db: tx},
		db:             tx,
	}
}
