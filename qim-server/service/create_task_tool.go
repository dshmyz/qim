package service

import (
	"fmt"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

// CreateTaskTool 允许 AI 根据用户意图创建任务提醒。
type CreateTaskTool struct {
	taskService *TaskService
}

func NewCreateTaskTool(taskService *TaskService) *CreateTaskTool {
	return &CreateTaskTool{taskService: taskService}
}

func (t *CreateTaskTool) Name() string {
	return "create_user_task"
}

func (t *CreateTaskTool) Description() string {
	return "为用户创建任务/待办提醒。当用户要求设置提醒、记录待办、创建任务时使用。会返回创建结果。"
}

func (t *CreateTaskTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"title": map[string]interface{}{
			"type":        "string",
			"description": "任务标题，简洁明确",
			"required":    true,
		},
		"description": map[string]interface{}{
			"type":        "string",
			"description": "任务详细描述（可选）",
			"required":    false,
		},
		"due_date": map[string]interface{}{
			"type":        "string",
			"description": "截止日期，格式 yyyy-MM-dd（可选）",
			"required":    false,
		},
		"priority": map[string]interface{}{
			"type":        "string",
			"description": "优先级：low、medium、high（可选，默认 medium）",
			"required":    false,
		},
	}
}

func (t *CreateTaskTool) Execute(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	if t.taskService == nil {
		return nil, fmt.Errorf("task service not available")
	}

	title, ok := params["title"].(string)
	if !ok || title == "" {
		return nil, fmt.Errorf("title is required")
	}

	description := ""
	if d, ok := params["description"].(string); ok {
		description = d
	}

	priority := "medium"
	if p, ok := params["priority"].(string); ok && p != "" {
		priority = p
	}

	var dueDate *time.Time
	if d, ok := params["due_date"].(string); ok && d != "" {
		if parsed, err := time.Parse("2006-01-02", d); err == nil {
			dueDate = &parsed
		}
	}

	var userID uint
	if ctx != nil {
		userID = ctx.UserID
	}
	if userID == 0 {
		return nil, fmt.Errorf("需要登录后才能创建任务")
	}

	task := &model.Task{
		Title:       title,
		Description: description,
		Priority:    priority,
		Status:      "todo",
		UserID:      userID,
	}

	if dueDate != nil {
		task.DueDate = dueDate
	}

	if err := t.taskService.CreateTask(task); err != nil {
		logger.WithModule("CreateTaskTool").Error("create task failed", "error", err)
		return nil, fmt.Errorf("创建任务失败: %w", err)
	}

	return map[string]interface{}{
		"id":       task.ID,
		"title":    task.Title,
		"due_date": task.DueDate,
		"priority": task.Priority,
		"status":   task.Status,
	}, nil
}
