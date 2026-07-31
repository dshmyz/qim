package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/repository"
	"github.com/dshmyz/qim/qim-server/ws"

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

// ProcessTaskReminders 扫描所有需要提醒的待办并发送通知。
// 由统一调度器（pkg/scheduler）每 30 秒调用一次。
func (s *TaskService) ProcessTaskReminders() {
	now := time.Now()

	// SQLite 下 due_date 以 TEXT 存储。写入端（TodoExtractor/CreateTaskTool）统一用
	// time.Parse（UTC）落库，读出即为 UTC time.Time，与 EventService 一致：
	// 用 now 的绝对时刻比较，不受时区影响。
	// 只扫描未来 24h 内到期的任务，避免全表扫描。
	var tasks []model.Task
	if err := s.db.Where(
		"reminder > 0 AND reminder_sent = ? AND due_date IS NOT NULL AND status != ? AND due_date > ?",
		false, "done", now.Add(-24*time.Hour),
	).Find(&tasks).Error; err != nil {
		return
	}

	for _, task := range tasks {
		reminderTime := task.DueDate.Add(-time.Duration(task.Reminder) * time.Minute)
		if now.After(reminderTime) || now.Equal(reminderTime) {
			// 先发送再标记，发送失败时下次调度仍会重试
			if err := s.sendTaskReminder(&task); err != nil {
				logger.WithModule("TaskService").Warn("待办提醒发送失败，下次重试",
					"taskID", task.ID, "userID", task.UserID, "error", err)
				continue
			}
			s.db.Model(&task).Update("reminder_sent", true)
		}
	}
}

func (s *TaskService) sendTaskReminder(task *model.Task) error {
	if ws.GlobalHub == nil {
		return fmt.Errorf("WebSocket hub 未初始化")
	}

	dueStr := task.DueDate.Local().Format("01/02 15:04")

	// 应用内通知
	notification := model.Notification{
		UserID:        task.UserID,
		Type:          "task_reminder",
		Title:         "待办提醒",
		Content:       fmt.Sprintf("「%s」将在 %s 到期", task.Title, dueStr),
		Priority:      "important",
		ActionType:    "confirm_reschedule",
		ActionPayload: fmt.Sprintf(`{"task_id":%d}`, task.ID),
	}
	if err := s.db.Create(&notification).Error; err != nil {
		return fmt.Errorf("通知落库失败: %w", err)
	}

	// WS 推送
	msg, _ := json.Marshal(ws.WSMessage{
		Type: "new_notification",
		Data: notification,
	})
	ws.GlobalHub.SendToUser(task.UserID, msg)
	return nil
}
