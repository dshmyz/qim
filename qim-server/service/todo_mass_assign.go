package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/ws"
	"gorm.io/gorm"
)

// MassTaskSpec 批量创建待办的参数（AI 提取与二次确认共用）。
type MassTaskSpec struct {
	Title       string
	Description string
	DueDate     *time.Time
	Priority    string
	Reminder    int
}

// CreateTasksForAssignees 按名单逐个创建待办（幂等：同一负责人已存在未完成的同名+同截止
// 任务时跳过，防止 AI 提取对同一条待办重复触发——单次提取重复返回、或同一待办在多条消息里
// 被反复提到——导致任务表堆重复行，list_tasks 再把它们原样吐给模型引发对账混乱）。
func CreateTasksForAssignees(db *gorm.DB, spec MassTaskSpec, senderID, conversationID uint, assigneeIDs []uint) {
	for _, assigneeID := range assigneeIDs {
		dupQuery := db.Model(&model.Task{}).
			Where("user_id = ? AND title = ? AND status != 'done'", assigneeID, spec.Title)
		if spec.DueDate != nil {
			dupQuery = dupQuery.Where("due_date = ?", *spec.DueDate)
		} else {
			dupQuery = dupQuery.Where("due_date IS NULL")
		}
		var dupCount int64
		if err := dupQuery.Count(&dupCount).Error; err == nil && dupCount > 0 {
			logger.WithModule("TodoExtractor").Info("待办已存在，跳过重复创建",
				"title", spec.Title, "assigneeID", assigneeID, "existing", dupCount)
			continue
		}

		task := model.Task{
			UserID:         assigneeID,
			ConversationID: conversationID,
			Title:          spec.Title,
			Description:    spec.Description,
			DueDate:        spec.DueDate,
			Priority:       spec.Priority,
			Status:         "todo",
			Reminder:       spec.Reminder,
		}
		db.Create(&task)

		logger.WithModule("TodoExtractor").Info("创建待办", "title", spec.Title, "assigneeID", assigneeID)

		// 通知负责人（带任务来源：群会话 + 指派人）
		if assigneeID != senderID {
			notifyTodoAssigned(db, task, assigneeID, senderID)
		}
	}

	if len(assigneeIDs) > 1 {
		logger.WithModule("TodoExtractor").Info("已为多人创建待办", "count", len(assigneeIDs), "title", spec.Title)
	}
}

// notifyTodoAssigned 通知负责人有新的待办，文案带任务来源（所属群 + 指派人昵称）——
// 被指派的人不该收到一条无从判断归属的匿名任务。
func notifyTodoAssigned(db *gorm.DB, task model.Task, assigneeID, creatorID uint) {
	var creator model.User
	if err := db.Unscoped().Select("nickname", "username").First(&creator, creatorID).Error; err != nil {
		creator = model.User{} // 查不到（已注销等）：两者皆空，走下方占位名回退
	}
	creatorName := creator.Nickname
	if creatorName == "" {
		creatorName = creator.Username
	}
	if creatorName == "" {
		// 创建者查不到（已注销等）：回退占位名，避免渲染出「由  指派」空名字
		creatorName = "未知成员"
	}
	sourceLine := fmt.Sprintf("由 %s 指派", creatorName)
	if task.ConversationID != 0 {
		var conv model.Conversation
		if err := db.First(&conv, task.ConversationID).Error; err == nil {
			var group model.Group
			if err := db.Where("conversation_id = ?", conv.ID).First(&group).Error; err == nil && group.Name != "" {
				sourceLine = fmt.Sprintf("来自群「%s」，由 %s 指派", group.Name, creatorName)
			}
		}
	}

	notification := model.Notification{
		UserID:        assigneeID,
		Type:          "todo_assigned",
		Title:         "新的待办事项",
		Content:       fmt.Sprintf("%s（%s）", task.Title, sourceLine),
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
	if jsonMsg, err := json.Marshal(wsMsg); err == nil {
		ws.GlobalHub.SendToUser(assigneeID, jsonMsg)
	}

	logger.WithModule("TodoExtractor").Info("已通知用户", "userID", assigneeID, "title", task.Title)
}

// ConfirmTodoMassAssign 触发者确认大范围指派：按锁定名单批量创建任务并逐人通知。
// 状态机约束：仅创建者本人、未过期；pending→approved 的翻转是一条原子条件更新，
// 抢占失败（并发确认/两通道同时点击）直接报错，保证批量创建只执行一次。
func ConfirmTodoMassAssign(db *gorm.DB, userID, pendingID uint) error {
	var pend model.TodoMassAssign
	if err := db.First(&pend, pendingID).Error; err != nil {
		return errors.New("待确认记录不存在")
	}
	if pend.CreatorID != userID {
		return errors.New("仅触发者本人可确认")
	}
	var assigneeIDs []uint
	if err := json.Unmarshal([]byte(pend.AssigneeIDs), &assigneeIDs); err != nil {
		return errors.New("指派名单已损坏，无法创建")
	}
	if time.Now().After(pend.ExpiresAt) {
		db.Model(&model.TodoMassAssign{}).
			Where("id = ? AND status = ?", pendingID, model.TodoMassAssignStatusPending).
			Update("status", model.TodoMassAssignStatusExpired)
		return errors.New("确认已过期（24 小时），请在群里重新发送该待办触发提取")
	}

	// 原子抢占：仅 pending→approved 一条路径能走到批量创建
	res := db.Model(&model.TodoMassAssign{}).
		Where("id = ? AND status = ?", pendingID, model.TodoMassAssignStatusPending).
		Update("status", model.TodoMassAssignStatusApproved)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("该记录已处理，无法重复确认")
	}

	CreateTasksForAssignees(db, MassTaskSpec{
		Title: pend.Title, Description: pend.Description,
		DueDate: pend.DueDate, Priority: pend.Priority, Reminder: pend.Reminder,
	}, pend.CreatorID, pend.ConversationID, assigneeIDs)

	logger.WithModule("TodoExtractor").Info("大范围指派已确认创建", "pendingID", pend.ID, "assignees", len(assigneeIDs))
	return nil
}

// RejectTodoMassAssign 触发者忽略大范围指派：记录置 rejected，不创建任何任务。
// 与确认共用同一原子抢占语义，防止并发下「确认已 approved」被忽略操作覆盖。
func RejectTodoMassAssign(db *gorm.DB, userID, pendingID uint) error {
	var pend model.TodoMassAssign
	if err := db.First(&pend, pendingID).Error; err != nil {
		return errors.New("待确认记录不存在")
	}
	if pend.CreatorID != userID {
		return errors.New("仅触发者本人可忽略")
	}
	res := db.Model(&model.TodoMassAssign{}).
		Where("id = ? AND status = ?", pendingID, model.TodoMassAssignStatusPending).
		Update("status", model.TodoMassAssignStatusRejected)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("该记录已处理，无法重复忽略")
	}

	logger.WithModule("TodoExtractor").Info("大范围指派已忽略", "pendingID", pend.ID, "title", pend.Title)
	return nil
}

// ParseTodoConfirmCard 识别群内全员待办确认卡。非确认卡返回 ok=false。
func ParseTodoConfirmCard(content string) (uint, bool) {
	var p struct {
		Kind      string `json:"kind"`
		PendingID uint   `json:"pending_id"`
	}
	if err := json.Unmarshal([]byte(content), &p); err != nil || p.Kind != "todo_confirm" || p.PendingID == 0 {
		return 0, false
	}
	return p.PendingID, true
}

// SendTodoConfirmCard 以默认 AI 助手身份在群里发一张全员待办确认卡：
// 确认动作发生在消息上下文里，全员可见任务来源与确认状态。
// 发送侧不要求 BotConversation（那是点击链路的校验），普通群没有 bot 入群记录也能发。
func (s *BotMessagingService) SendTodoConfirmCard(convID uint, bot model.Bot, pendingID uint, title string, assigneeCount int, creatorName string) error {
	if bot.VirtualUserID == nil {
		return errors.New("bot 未配置虚拟用户")
	}
	value := fmt.Sprintf("pending:%d", pendingID)
	payload := map[string]interface{}{
		"kind":       "todo_confirm",
		"pending_id": pendingID,
		"title":      "待确认的全员待办",
		"text": fmt.Sprintf("AI 从群消息提取到「%s」，将指派给 %d 位成员。\n发起人：%s · 确认后才会创建任务",
			title, assigneeCount, creatorName),
		"buttons": []cardButton{
			{ID: "confirm", Text: "确认创建", Style: "primary", Value: value},
			{ID: "cancel", Text: "忽略", Style: "secondary", Value: value},
		},
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if err := validateCardContent(string(content)); err != nil {
		return err
	}

	// 复用 sendToConversation 管道（IsRead/last_message/未读计数/广播一整套）
	_, err = s.sendToConversation(&bot, convID, string(content), "card", nil)
	return err
}

// HandleTodoConfirmCard 群内全员待办确认卡的按钮点击：校验成员身份后执行平台侧
// 确认/忽略状态机，落点击记录与 card_action 气泡，并把卡片回写成终态。
// 独立于 bot 卡链路（ForwardCardAction 要求 BotConversation，普通群没有 bot 入群记录）；
// pending_id 以卡片内容（服务端落库）为准，不信任客户端 value。
func (s *BotMessagingService) HandleTodoConfirmCard(msg *model.Message, userID uint, actionID string) error {
	pendingID, ok := ParseTodoConfirmCard(msg.Content)
	if !ok {
		return errors.New("非待确认卡片")
	}
	// 防伪造：todo_confirm 卡只应由默认 AI 助手发出。外部 agent bot 可发送任意
	// card 内容，若无此校验，agent 可伪造确认卡诱导 pending 创建者提前确认/忽略。
	var senderBot model.Bot
	if err := s.db.Where("virtual_user_id = ?", msg.SenderID).First(&senderBot).Error; err != nil ||
		senderBot.Type != model.BotTypeAssistant || senderBot.CreatorID != 0 ||
		senderBot.GroupID != nil || senderBot.IsTemplate {
		return errors.New("卡片来源无效")
	}
	// 仅该会话的人类成员可点击
	var member model.ConversationMember
	if err := s.db.Where("conversation_id = ? AND user_id = ?", msg.ConversationID, userID).First(&member).Error; err != nil {
		return errors.New("无权操作此卡片")
	}

	var execErr error
	switch actionID {
	case "confirm":
		execErr = ConfirmTodoMassAssign(s.db, userID, pendingID)
	case "cancel":
		execErr = RejectTodoMassAssign(s.db, userID, pendingID)
	default:
		return errors.New("未知按钮动作")
	}
	if execErr != nil {
		// 失败（过期/非本人/已处理）不落点击记录、不回写终态：按钮保持可点
		return execErr
	}

	// 同步触发者通知中心那条确认通知：置 Handled，按钮不再悬空可点
	s.db.Model(&model.Notification{}).
		Where("user_id = ? AND type = ? AND action_payload = ?", userID, "todo_mass_pending",
			fmt.Sprintf(`{"pending_id":%d}`, pendingID)).
		Updates(map[string]interface{}{"handled": true, "read": true, "read_at": time.Now()})

	// 点击记录（幂等索引；重复点击冲突忽略）
	botID := uint(0)
	var cardBot model.Bot
	if err := s.db.Where("virtual_user_id = ?", msg.SenderID).First(&cardBot).Error; err == nil {
		botID = cardBot.ID
	}
	s.db.Create(&model.CardActionRecord{MessageID: msg.ID, UserID: userID, ActionID: actionID, BotID: botID})

	// card_action 气泡：与 bot 卡同款「✓ 已选择:xxx」展示
	if actionText, err := cardActionText(msg.Content, actionID); err == nil {
		actionContent, _ := json.Marshal(map[string]interface{}{
			"action_id":       actionID,
			"action_text":     actionText,
			"value":           fmt.Sprintf("pending:%d", pendingID),
			"card_message_id": msg.ID,
		})
		actionMsg := model.Message{
			ConversationID: msg.ConversationID,
			SenderID:       userID,
			Type:           "card_action",
			Content:        string(actionContent),
			Origin:         "user",
		}
		if createErr := s.db.Create(&actionMsg).Error; createErr == nil {
			s.db.Preload("Sender").First(&actionMsg, actionMsg.ID)
			if s.hub != nil {
				resp := BuildMessageResponse(actionMsg, MessageResponseOptions{BroadcastWS: true})
				wsMsg := ws.WSMessage{Type: "new_message", Data: resp}
				if jsonMsg, err := json.Marshal(wsMsg); err == nil {
					s.hub.SendToConversation(msg.ConversationID, userID, jsonMsg)
				}
			}
		}
	}

	// 终态回写：卡片发送者是默认 AI 助手（虚拟用户），推送排除其本人
	line := "✓ 已确认创建"
	if actionID == "cancel" {
		line = "✓ 已忽略"
	}
	s.rewriteTodoConfirmCard(msg, line)
	return nil
}

// rewriteTodoConfirmCard 把确认卡回写成终态（追加结果行），推送排除卡片发送者。
// 与 rewriteConfirmCard 的差异：推送排除者直接取卡片消息的发送者 ID，
// 不需要 bot 指针（发送侧是默认 AI 助手，无需会话归属）。
func (s *BotMessagingService) rewriteTodoConfirmCard(msg *model.Message, resultLine string) {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(msg.Content), &payload); err != nil {
		return
	}
	text, _ := payload["text"].(string)
	payload["text"] = text + "\n\n" + resultLine
	content, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if err := validateCardContent(string(content)); err != nil {
		logger.WithModule("BotMessaging").Error("确认卡终态回写校验失败", "messageID", msg.ID, "error", err)
		return
	}
	if err := s.db.Model(&model.Message{}).Where("id = ?", msg.ID).
		Update("content", string(content)).Error; err != nil {
		logger.WithModule("BotMessaging").Error("确认卡终态回写失败", "messageID", msg.ID, "error", err)
		return
	}
	var latest model.Message
	if err := s.db.First(&latest, msg.ID).Error; err != nil {
		return
	}
	s.pushMessageUpdated(latest, msg.SenderID, false)
}

// SyncTodoConfirmCardTerminal 通知中心通道确认/忽略成功后，同步群内确认卡终态：
// 按会话+pending_id 找到确认卡并回写结果行，群成员看到的按钮不再悬空。
func (s *BotMessagingService) SyncTodoConfirmCardTerminal(convID, pendingID uint, resultLine string) {
	var msgs []model.Message
	if err := s.db.Where("conversation_id = ? AND type = ?", convID, "card").
		Order("id DESC").Limit(20).Find(&msgs).Error; err != nil {
		return
	}
	for i := range msgs {
		if pid, ok := ParseTodoConfirmCard(msgs[i].Content); ok && pid == pendingID {
			s.rewriteTodoConfirmCard(&msgs[i], resultLine)
			return
		}
	}
	logger.WithModule("TodoExtractor").Info("未找到待同步终态的确认卡", "convID", convID, "pendingID", pendingID)
}
