package service

import (
	"context"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/repository"

	"gorm.io/gorm"
)

type TaskService struct {
	repo repository.TaskRepository
	db   *gorm.DB
}

func NewTaskService(db *gorm.DB) *TaskService {
	return &TaskService{
		repo: repository.NewTaskRepository(db),
		db:   db,
	}
}

func (s *TaskService) GetTasks(userID uint) ([]model.Task, error) {
	ctx := context.Background()
	tasks, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]model.Task, len(tasks))
	for i, t := range tasks {
		result[i] = *t
	}
	return result, nil
}

func (s *TaskService) CreateTask(task *model.Task) error {
	ctx := context.Background()
	return s.repo.Create(ctx, task)
}

func (s *TaskService) GetTask(userID, taskID uint) (*model.Task, error) {
	ctx := context.Background()
	return s.repo.FindByUserIDAndID(ctx, userID, taskID)
}

func (s *TaskService) UpdateTask(userID, taskID uint, updates map[string]interface{}) (*model.Task, error) {
	ctx := context.Background()
	task, err := s.repo.FindByUserIDAndID(ctx, userID, taskID)
	if err != nil {
		return nil, err
	}

	if title, ok := updates["title"]; ok {
		task.Title = title.(string)
	}
	if desc, ok := updates["description"]; ok {
		task.Description = desc.(string)
	}
	if dueDate, ok := updates["due_date"]; ok {
		if t, ok := dueDate.(*time.Time); ok {
			task.DueDate = t
		}
	}
	if priority, ok := updates["priority"]; ok {
		task.Priority = priority.(string)
	}
	if status, ok := updates["status"]; ok {
		task.Status = status.(string)
	}
	if assigneeID, ok := updates["assignee_id"]; ok {
		task.AssigneeID = assigneeID.(string)
	}
	if tags, ok := updates["tags"]; ok {
		task.Tags = tags.(string)
	}
	if subTasks, ok := updates["sub_tasks"]; ok {
		task.SubTasks = subTasks.(string)
	}
	if position, ok := updates["position"]; ok {
		task.Position = position.(int)
	}

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *TaskService) UpdateTaskStatus(userID, taskID uint, status string) (*model.Task, error) {
	ctx := context.Background()
	task, err := s.repo.FindByUserIDAndID(ctx, userID, taskID)
	if err != nil {
		return nil, err
	}
	task.Status = status
	if err := s.repo.Update(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *TaskService) DeleteTask(userID, taskID uint) error {
	ctx := context.Background()
	return s.repo.DeleteByUserIDAndID(ctx, userID, taskID)
}
