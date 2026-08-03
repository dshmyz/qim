package repository

import (
	"context"
	"errors"
	"strconv"

	"github.com/dshmyz/qim/qim-server/model"

	"gorm.io/gorm"
)

// ErrInvalidConversationID 表示传入的会话 ID 无效（如 0）
var ErrInvalidConversationID = errors.New("invalid conversation id")

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
	// 返回自己创建的 + 被指派给自己的任务（assignee_id 存用户 ID 字符串）
	err := r.db.WithContext(ctx).Where("user_id = ? OR assignee_id = ?", userID, strconv.FormatUint(uint64(userID), 10)).Order("created_at DESC").Find(&tasks).Error
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

// FindByConversationAndID 按会话 + 任务 ID 查询单条任务（用于消息里任务引用卡片的渲染）
// repo 仅做数据查询：会话任务（conversation_id=指定值）+ 私人任务（conversation_id=0）都能查到
// 不再按 user_id 过滤——会话任务对会话成员共见；私人任务的越权防护由 service 层特判完成
func (r *taskRepository) FindByConversationAndID(ctx context.Context, conversationID, id uint) (*model.Task, error) {
	var task model.Task
	err := r.db.WithContext(ctx).
		Where("id = ? AND (conversation_id = ? OR conversation_id = 0)", id, conversationID).
		First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// FindByConversationID 列出该会话可引用的全部任务（用于输入框 /task 自动补全）
// 含：该会话关联的任务（不限创建者，会话任务对会话成员共见）+ 自己的私人任务（conversation_id=0）
// 别人的私人任务由 user_id 过滤防越权；conversationID=0 直接返回错误，防枚举所有私人任务
func (r *taskRepository) FindByConversationID(ctx context.Context, userID, conversationID uint) ([]*model.Task, error) {
	if conversationID == 0 {
		return nil, ErrInvalidConversationID
	}
	var tasks []*model.Task
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? OR (conversation_id = 0 AND user_id = ?)", conversationID, userID).
		Order("created_at DESC").
		Find(&tasks).Error
	return tasks, err
}

func (r *taskRepository) WithTx(tx *gorm.DB) BaseRepository[model.Task] {
	return &taskRepository{
		baseRepository: &baseRepository[model.Task]{db: tx},
		db:             tx,
	}
}
