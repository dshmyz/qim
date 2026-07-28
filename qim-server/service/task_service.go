package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
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
	var tasks []model.Task
	if err := s.db.Where(
		"reminder > 0 AND reminder_sent = ? AND due_date IS NOT NULL AND status != ?",
		false, "done",
	).Find(&tasks).Error; err != nil {
		return
	}

	for _, task := range tasks {
		reminderTime := task.DueDate.Add(-time.Duration(task.Reminder) * time.Minute)
		if now.After(reminderTime) || now.Equal(reminderTime) {
			// 先标记再发送，防止调度器重复触发
			s.db.Model(&task).Update("reminder_sent", true)
			s.sendTaskReminder(&task)
		}
	}
}

func (s *TaskService) sendTaskReminder(task *model.Task) {
	if ws.GlobalHub == nil {
		return
	}

	db := database.GetDB()
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
	if err := db.Create(&notification).Error; err != nil {
		logger.WithModule("TaskService").Warn("待办提醒通知落库失败",
			"taskID", task.ID, "userID", task.UserID, "error", err)
	}

	// WS 推送
	msg, _ := json.Marshal(ws.WSMessage{
		Type: "new_notification",
		Data: notification,
	})
	ws.GlobalHub.SendToUser(task.UserID, msg)
}
