package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/mention"
	"github.com/dshmyz/qim/qim-server/ws"
	"gorm.io/gorm"
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

// massAssignThreshold 大范围指派的二次确认阈值：指派人数超过它时，AI 的单次判定
// 不再直接落库，而是给触发者发确认通知。assignee=all 常见于几百人的大群，
// AI 一次误判（把闲聊提成待办）就是全群几百条任务+通知轰炸，必须有人工兜底。
const massAssignThreshold = 10

// ExtractedTodo 提取的待办结构
type ExtractedTodo struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	Assignee      string `json:"assignee,omitempty"`
	DueDate       string `json:"due_date,omitempty"`
	Priority      string `json:"priority,omitempty"`
	RemindMinutes int    `json:"remind_minutes,omitempty"` // 提前提醒分钟数
}

// maxExtractContentRunes 送入提取 prompt 的消息正文上限（rune 数）：超长粘贴/长引用
// 一条消息就能把提取调用的 token 打爆。截断后附加标记，让模型知道内容不完整。
const maxExtractContentRunes = 2000

// runePrefix 按 rune 数取前缀——日志里的 content/result 切片如果按字节切，
// 会把多字节 UTF-8 切出乱码。
func runePrefix(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		r = r[:n]
	}
	return string(r)
}

// ExtractAndCreateTodos 从消息中提取待办并创建
func (e *TodoExtractor) ExtractAndCreateTodos(content string, senderID uint, conversationID uint) {
	if e.aiService == nil || !e.aiService.IsConfigured() {
		logger.WithModule("TodoExtractor").Info("AI 服务未配置，跳过待办提取")
		return
	}

	db := database.GetDB()

	// 消息落库存的是 mention token 原文（@{mention:id|urlencoded姓名}，中文经 URL 编码），
	// 解码成 "@姓名" 纯文本再进 prompt——原样送 AI 会让「识别 @人」规则对着百分号乱码
	// 工作，负责人识别不出来后静默回退给发送者。
	content = mention.Decode(content)

	// 超长正文截断（rune 安全）
	if runes := []rune(content); len(runes) > maxExtractContentRunes {
		content = string(runes[:maxExtractContentRunes]) + "\n…（消息过长已截断）"
	}

	// 获取发送者信息
	var sender model.User
	db.First(&sender, senderID)
	senderName := sender.Nickname
	if senderName == "" {
		senderName = sender.Username
	}

	logger.WithModule("TodoExtractor").Info("开始提取待办", "sender", senderName, "convID", conversationID, "content", runePrefix(content, 50))

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

	// 动态生成当前日期时间的参考信息
	now := time.Now()
	today := now
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)
	dayAfter := today.AddDate(0, 0, 2)
	dayAfter3 := today.AddDate(0, 0, 3)
	nextWeek := today.AddDate(0, 0, 7)
	saturday := today.AddDate(0, 0, int(time.Saturday-today.Weekday()))

	systemPrompt := `你是一个待办提取助手。分析以下群聊消息，提取其中的待办事项。

群成员：` + strings.TrimRight(memberNames, "、") + `

当前日期时间：` + now.Format("2006-01-02 15:04") + ` (` + now.Weekday().String() + `)

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
- 截止时间必须转换为 YYYY-MM-DD HH:mm 格式，使用上面的对照表
- 如果只提到时间没提到日期（如"下午两点"），默认为今天；如果今天该时间已过，则为明天
- 如果只提到日期没提到时间，使用 23:59 作为截止时间
- 如果提到"所有人"、"大家"、"@all"等，assignee 填 "all"
- 如果消息提到"提醒"、"记得"、"别忘了"等，设置 remind_minutes（默认 30 分钟）；如果有明确的提前时间（如"提前1小时"），按实际设置
- 如果没有明确的待办，返回空数组

只返回 JSON 数组格式：
[{"title": "任务标题", "description": "任务描述", "assignee": "负责人名称", "due_date": "YYYY-MM-DD HH:mm", "priority": "low|medium|high", "remind_minutes": 30}]

如果没有待办，返回 []。`

	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: senderName + ": " + content},
	}

	result, err := e.aiService.GetCompletion(ai.TaskTypeAnalysis, messages)
	if err != nil {
		logger.WithModule("TodoExtractor").Error("AI 提取失败", "error", err)
		return
	}

	logger.WithModule("TodoExtractor").Debug("AI 返回结果", "result", runePrefix(result, 200))

	todos, err := e.parseAIResult(result)
	if err != nil {
		logger.WithModule("TodoExtractor").Error("解析 AI 结果失败", "error", err, "raw", runePrefix(result, 100))
		return
	}

	if len(todos) == 0 {
		logger.WithModule("TodoExtractor").Info("未提取到待办")
		return
	}

	logger.WithModule("TodoExtractor").Info("提取到待办", "count", len(todos))

	// 创建待办
	for _, todo := range todos {
		logger.WithModule("TodoExtractor").Debug("处理待办", "title", todo.Title, "assignee", todo.Assignee, "dueDate", todo.DueDate, "priority", todo.Priority)
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

	// 人类成员（排除 bot 虚拟用户）：拉 bot 进群会写 ConversationMember，已删 bot 的
	// 虚拟用户也残留成员表——指派给它会生成没有任何人能看到的幽灵任务。
	humanMemberIDs := make([]uint, 0, len(conv.Members))
	for _, m := range conv.Members {
		if m.User.Type == "bot" {
			continue
		}
		humanMemberIDs = append(humanMemberIDs, m.UserID)
	}

	var assigneeIDs []uint

	if todo.Assignee == "all" {
		// 指派给所有人类成员（排除发送者）
		for _, memberID := range humanMemberIDs {
			if memberID != senderID {
				assigneeIDs = append(assigneeIDs, memberID)
			}
		}
	} else if todo.Assignee != "" && len(humanMemberIDs) > 0 {
		// 只在群成员内匹配负责人：全库 nickname/username 精确查会命中群外的同名用户
		//（重名时 First() 按主键序还可能取到更小 ID 的那个），任务跨群派给无关的人。
		var user model.User
		if err := db.Where("id IN ? AND (nickname = ? OR username = ?)", humanMemberIDs, todo.Assignee, todo.Assignee).First(&user).Error; err == nil {
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
		dueDate = parseDueDate(todo.DueDate)
	}

	priority := "medium"
	if todo.Priority != "" {
		priority = todo.Priority
	}

	// 大范围指派二次确认：AI 一次误判的爆炸半径是全群（几百条任务+通知轰炸），
	// 超阈值时先落待确认记录、由触发者从通知中心确认后才批量创建。
	if len(assigneeIDs) > massAssignThreshold {
		e.pendMassAssign(todo, priority, dueDate, senderID, conversationID, assigneeIDs)
		return
	}

	e.createTasksForAssignees(todo, priority, dueDate, senderID, conversationID, assigneeIDs)
}

// pendMassAssign 落大范围指派的待确认记录，并给触发者发一条确认通知（只发给触发者，
// 不打扰其他成员）。确认/忽略走通知中心通用动作分发（PATCH /notifications/:id）。
func (e *TodoExtractor) pendMassAssign(todo ExtractedTodo, priority string, dueDate *time.Time, senderID, conversationID uint, assigneeIDs []uint) {
	db := database.GetDB()

	idsJSON, err := json.Marshal(assigneeIDs)
	if err != nil {
		logger.WithModule("TodoExtractor").Error("序列化指派名单失败", "error", err)
		return
	}

	pend := model.TodoMassAssign{
		CreatorID:      senderID,
		ConversationID: conversationID,
		Title:          todo.Title,
		Description:    todo.Description,
		DueDate:        dueDate,
		Priority:       priority,
		Reminder:       todo.RemindMinutes,
		AssigneeIDs:    string(idsJSON),
		Status:         model.TodoMassAssignStatusPending,
		ExpiresAt:      time.Now().Add(24 * time.Hour), // 与分身草稿同款 TTL：24h 内未确认视为放弃
	}
	if err := db.Create(&pend).Error; err != nil {
		logger.WithModule("TodoExtractor").Error("落待确认记录失败", "error", err)
		return
	}

	notification := model.Notification{
		UserID:        senderID,
		Type:          "todo_mass_pending",
		Title:         "待确认的全员待办",
		Content:       fmt.Sprintf("AI 从群消息提取到「%s」，将指派给 %d 位成员，确认后才会创建", todo.Title, len(assigneeIDs)),
		Read:          false,
		Priority:      "important",
		ActionType:    "todo_mass_confirm",
		ActionPayload: fmt.Sprintf(`{"pending_id":%d}`, pend.ID),
	}
	db.Create(&notification)
	if ws.GlobalHub != nil {
		wsMsg := ws.WSMessage{Type: "new_notification", Data: notification}
		if jsonMsg, err := json.Marshal(wsMsg); err == nil {
			ws.GlobalHub.SendToUser(senderID, jsonMsg)
		}
	}

	logger.WithModule("TodoExtractor").Info("大范围指派待确认", "title", todo.Title,
		"assignees", len(assigneeIDs), "pendingID", pend.ID)
}

// createTasksForAssignees 按名单逐个创建待办（幂等：同一负责人已存在未完成的同名+同截止
// 任务时跳过，防止 AI 提取对同一条待办重复触发——单次提取重复返回、或同一待办在多条消息里
// 被反复提到——导致任务表堆重复行，list_tasks 再把它们原样吐给模型引发对账混乱）。
func (e *TodoExtractor) createTasksForAssignees(todo ExtractedTodo, priority string, dueDate *time.Time, senderID, conversationID uint, assigneeIDs []uint) {
	db := database.GetDB()
	for _, assigneeID := range assigneeIDs {
		dupQuery := db.Model(&model.Task{}).
			Where("user_id = ? AND title = ? AND status != 'done'", assigneeID, todo.Title)
		if dueDate != nil {
			dupQuery = dupQuery.Where("due_date = ?", *dueDate)
		} else {
			dupQuery = dupQuery.Where("due_date IS NULL")
		}
		var dupCount int64
		if err := dupQuery.Count(&dupCount).Error; err == nil && dupCount > 0 {
			logger.WithModule("TodoExtractor").Info("待办已存在，跳过重复创建",
				"title", todo.Title, "assigneeID", assigneeID, "existing", dupCount)
			continue
		}

		task := model.Task{
			UserID:         assigneeID,
			ConversationID: conversationID,
			Title:          todo.Title,
			Description:    todo.Description,
			DueDate:        dueDate,
			Priority:       priority,
			Status:         "todo",
			Reminder:       todo.RemindMinutes,
		}
		db.Create(&task)

		logger.WithModule("TodoExtractor").Info("创建待办", "title", todo.Title, "assigneeID", assigneeID)

		// 通知负责人
		if assigneeID != senderID {
			e.notifyTodoAssigned(task, assigneeID, senderID, conversationID)
		}
	}

	if len(assigneeIDs) > 1 {
		logger.WithModule("TodoExtractor").Info("已为多人创建待办", "count", len(assigneeIDs), "title", todo.Title)
	}
}

// confirmTodoMassAssign 触发者确认大范围指派：按锁定名单批量创建任务并逐人通知。
// 状态机约束：仅创建者本人、仅 pending、未过期（过期时记录置 expired 拒绝执行）。
func confirmTodoMassAssign(db *gorm.DB, userID, pendingID uint) error {
	var pend model.TodoMassAssign
	if err := db.First(&pend, pendingID).Error; err != nil {
		return errors.New("待确认记录不存在")
	}
	if pend.CreatorID != userID {
		return errors.New("仅触发者本人可确认")
	}
	if pend.Status != model.TodoMassAssignStatusPending {
		return fmt.Errorf("该记录已处理（%s），无法重复确认", pend.Status)
	}
	if time.Now().After(pend.ExpiresAt) {
		db.Model(&pend).Update("status", model.TodoMassAssignStatusExpired)
		return errors.New("确认已过期（24 小时），请在群里重新发送该待办触发提取")
	}

	var assigneeIDs []uint
	if err := json.Unmarshal([]byte(pend.AssigneeIDs), &assigneeIDs); err != nil {
		return errors.New("指派名单已损坏，无法创建")
	}

	e := &TodoExtractor{}
	e.createTasksForAssignees(
		ExtractedTodo{Title: pend.Title, Description: pend.Description, RemindMinutes: pend.Reminder},
		pend.Priority, pend.DueDate, pend.CreatorID, pend.ConversationID, assigneeIDs,
	)
	db.Model(&pend).Update("status", model.TodoMassAssignStatusApproved)

	logger.WithModule("TodoExtractor").Info("大范围指派已确认创建", "pendingID", pend.ID, "assignees", len(assigneeIDs))
	return nil
}

// rejectTodoMassAssign 触发者忽略大范围指派：记录置 rejected，不创建任何任务。
func rejectTodoMassAssign(db *gorm.DB, userID, pendingID uint) error {
	var pend model.TodoMassAssign
	if err := db.First(&pend, pendingID).Error; err != nil {
		return errors.New("待确认记录不存在")
	}
	if pend.CreatorID != userID {
		return errors.New("仅触发者本人可忽略")
	}
	if pend.Status != model.TodoMassAssignStatusPending {
		return fmt.Errorf("该记录已处理（%s）", pend.Status)
	}
	db.Model(&pend).Update("status", model.TodoMassAssignStatusRejected)

	logger.WithModule("TodoExtractor").Info("大范围指派已忽略", "pendingID", pend.ID, "title", pend.Title)
	return nil
}

// parseDueDate 解析待办截止时间字符串。
// - 带时间的格式：保留解析到的时间
// - 纯日期格式（如 "2006-01-02" 或 "01-02"）：统一使用 23:59 作为截止时间
func parseDueDate(s string) *time.Time {
	formats := []struct {
		layout   string
		dateOnly bool
	}{
		{"2006-01-02 15:04", false},
		{"2006-01-02 15:04:05", false},
		{"2006/01/02 15:04", false},
		{"2006-01-02T15:04:05", false},
		{"2006-01-02", true},
		{"2006/01/02", true},
		{"01-02", true},
		{"01/02", true},
	}
	for _, f := range formats {
		t, err := time.ParseInLocation(f.layout, s, time.Local)
		if err == nil {
			if t.Year() == 0 {
				t = time.Date(time.Now().Year(), t.Month(), t.Day(), 23, 59, 0, 0, time.Local)
			} else if f.dateOnly {
				t = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 0, 0, time.Local)
			}
			return &t
		}
	}
	return nil
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

	logger.WithModule("TodoExtractor").Info("已通知用户", "userID", assigneeID, "title", task.Title)
}
