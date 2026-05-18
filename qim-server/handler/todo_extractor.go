package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"qim-server/ai"
	"qim-server/database"
	"qim-server/model"
	"qim-server/ws"
	"strings"
	"time"
)

// TodoExtractor 待办提取引擎
type TodoExtractor struct {
	aiService *ai.AIService
}

// NewTodoExtractor 创建待办提取引擎
func NewTodoExtractor(aiService *ai.AIService) *TodoExtractor {
	return &TodoExtractor{
		aiService: aiService,
	}
}

// ExtractedTodo 提取的待办结构
type ExtractedTodo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Assignee    string `json:"assignee,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
	Priority    string `json:"priority,omitempty"`
}

// ExtractAndCreateTodos 从消息中提取待办并创建
func (e *TodoExtractor) ExtractAndCreateTodos(content string, senderID uint, conversationID uint) {
	if e.aiService == nil || !e.aiService.IsConfigured() {
		log.Printf("[TodoExtractor] AI 服务未配置，跳过待办提取")
		return
	}

	db := database.GetDB()

	// 获取发送者信息
	var sender model.User
	db.First(&sender, senderID)
	senderName := sender.Nickname
	if senderName == "" {
		senderName = sender.Username
	}

	log.Printf("[TodoExtractor] 开始提取待办: sender=%s, convID=%d, content=%s", senderName, conversationID, content[:min(50, len(content))])

	// 获取会话信息，获取所有成员
	var conv model.Conversation
	db.Preload("Members.User").First(&conv, conversationID)

	if conv.Type != "group" {
		return // 只在群聊中提取待办
	}

	memberNames := ""
	for _, m := range conv.Members {
		name := m.User.Nickname
		if name == "" {
			name = m.User.Username
		}
		memberNames += name + "、"
	}

	// 动态生成当前日期的参考信息
	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)
	dayAfter := today.AddDate(0, 0, 2)
	dayAfter3 := today.AddDate(0, 0, 3)
	nextWeek := today.AddDate(0, 0, 7)
	saturday := today.AddDate(0, 0, int(time.Saturday-today.Weekday()))

	systemPrompt := `你是一个待办提取助手。分析以下群聊消息，提取其中的待办事项。

群成员：` + strings.TrimRight(memberNames, "、") + `

当前日期：` + today.Format("2006-01-02") + ` (` + today.Weekday().String() + `)

相对日期对照表：
- 昨天 = ` + yesterday.Format("2006-01-02") + `
- 今天 = ` + today.Format("2006-01-02") + `
- 明天 = ` + tomorrow.Format("2006-01-02") + `
- 后天 = ` + dayAfter.Format("2006-01-02") + `
- 大后天 = ` + dayAfter3.Format("2006-01-02") + `
- 下周 = ` + nextWeek.Format("2006-01-02") + `
- 周末 = ` + saturday.Format("2006-01-02") + `

提取规则：
- 如果消息中包含明确的任务/待办/安排，提取出来
- 识别任务标题、描述、负责人（@的人或提到的人名）、截止时间、优先级
- 截止时间必须转换为 YYYY-MM-DD 格式，使用上面的对照表
- 如果提到"所有人"、"大家"、"@all"等，assignee 填 "all"
- 如果没有明确的待办，返回空数组

只返回 JSON 数组格式：
[{"title": "任务标题", "description": "任务描述", "assignee": "负责人名称", "due_date": "YYYY-MM-DD", "priority": "low|medium|high"}]

如果没有待办，返回 []。`

	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: senderName + ": " + content},
	}

	result, err := e.aiService.GetCompletion(ai.TaskTypeAnalysis, messages)
	if err != nil {
		log.Printf("[TodoExtractor] AI 提取失败: %v", err)
		return
	}

	log.Printf("[TodoExtractor] AI 返回结果: %s", result[:min(200, len(result))])

	todos, err := e.parseAIResult(result)
	if err != nil {
		log.Printf("[TodoExtractor] 解析 AI 结果失败: %v, raw=%s", err, result[:min(100, len(result))])
		return
	}

	if len(todos) == 0 {
		log.Printf("[TodoExtractor] 未提取到待办")
		return
	}

	log.Printf("[TodoExtractor] 提取到 %d 条待办", len(todos))

	// 创建待办
	for _, todo := range todos {
		log.Printf("[TodoExtractor] 处理待办: title=%s, assignee=%s, dueDate=%s, priority=%s", todo.Title, todo.Assignee, todo.DueDate, todo.Priority)
		e.createTodo(todo, senderID, conversationID)
	}
}

// parseAIResult 解析 AI 返回的待办列表
func (e *TodoExtractor) parseAIResult(result string) ([]ExtractedTodo, error) {
	jsonStr := result
	if idx := strings.Index(result, "["); idx >= 0 {
		jsonStr = result[idx:]
		if endIdx := strings.LastIndex(jsonStr, "]"); endIdx >= 0 {
			jsonStr = jsonStr[:endIdx+1]
		}
	}

	var todos []ExtractedTodo
	if err := json.Unmarshal([]byte(jsonStr), &todos); err != nil {
		return nil, err
	}

	return todos, nil
}

// createTodo 创建单条待办
func (e *TodoExtractor) createTodo(todo ExtractedTodo, senderID uint, conversationID uint) {
	db := database.GetDB()

	if todo.Title == "" {
		return
	}

	// 获取会话成员（用于"所有人"的情况）
	var conv model.Conversation
	db.Preload("Members.User").First(&conv, conversationID)

	var assigneeIDs []uint

	if todo.Assignee == "all" {
		// 指派给所有群成员（排除发送者）
		for _, m := range conv.Members {
			if m.UserID != senderID {
				assigneeIDs = append(assigneeIDs, m.UserID)
			}
		}
	} else if todo.Assignee != "" {
		// 查找指定负责人
		var user model.User
		if err := db.Where("nickname = ? OR username = ?", todo.Assignee, todo.Assignee).First(&user).Error; err == nil {
			assigneeIDs = append(assigneeIDs, user.ID)
		} else {
			// 找不到就指派给发送者
			assigneeIDs = append(assigneeIDs, senderID)
		}
	} else {
		// 没有指定负责人，默认指派给发送者
		assigneeIDs = append(assigneeIDs, senderID)
	}

	var dueDate *time.Time
	if todo.DueDate != "" {
		formats := []string{
			"2006-01-02",
			"2006/01/02",
			"01-02",
			"01/02",
			"2006-01-02T15:04:05",
		}
		for _, format := range formats {
			t, err := time.ParseInLocation(format, todo.DueDate, time.Local)
			if err == nil {
				if t.Year() == 0 {
					t = time.Date(time.Now().Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.Local)
				} else {
					t = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.Local)
				}
				dueDate = &t
				break
			}
		}
	}

	priority := "medium"
	if todo.Priority != "" {
		priority = todo.Priority
	}

	// 为每个负责人创建待办
	for _, assigneeID := range assigneeIDs {
		task := model.Task{
			UserID:      assigneeID,
			Title:       todo.Title,
			Description: todo.Description,
			DueDate:     dueDate,
			Priority:    priority,
			Status:      "todo",
		}
		db.Create(&task)

		log.Printf("[TodoExtractor] 创建待办: %s -> %d", todo.Title, assigneeID)

		// 通知负责人
		if assigneeID != senderID {
			e.notifyTodoAssigned(task, assigneeID, senderID, conversationID)
		}
	}

	if len(assigneeIDs) > 1 {
		log.Printf("[TodoExtractor] 已为 %d 人创建待办: %s", len(assigneeIDs), todo.Title)
	}
}

// notifyTodoAssigned 通知负责人有新的待办
func (e *TodoExtractor) notifyTodoAssigned(task model.Task, assigneeID uint, creatorID uint, conversationID uint) {
	db := database.GetDB()

	notification := model.Notification{
		UserID:        assigneeID,
		Type:          "todo_assigned",
		Title:         "新的待办事项",
		Content:       task.Title,
		Read:          false,
		Priority:      "important",
		ActionType:    "confirm_reschedule",
		ActionPayload: fmt.Sprintf(`{"task_id":%d}`, task.ID),
	}
	db.Create(&notification)

	// 通过 WebSocket 推送
	wsMsg := ws.WSMessage{
		Type: "new_notification",
		Data: notification,
	}
	jsonMsg, _ := json.Marshal(wsMsg)
	ws.GlobalHub.SendToUser(assigneeID, jsonMsg)

	log.Printf("[TodoExtractor] 已通知用户 %d 有新的待办: %s", assigneeID, task.Title)
}
