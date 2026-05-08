package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"qim-server/di"
	"qim-server/model"
	"qim-server/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetNotifications(c *gin.Context) {
	userID, _ := c.Get("user_id")

	pageStr := c.Query("page")
	pageSizeStr := c.Query("pageSize")

	page := 1
	pageSize := 10

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	svc := di.GlobalContainer.NotificationService
	notifications, total, err := svc.GetNotifications(userID.(uint), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取通知失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  notifications,
			"total": total,
			"page":  page,
		},
	})
}

func MarkNotificationAsRead(c *gin.Context) {
	userID, _ := c.Get("user_id")
	notificationIDStr := c.Param("id")

	notificationID, err := strconv.ParseUint(notificationIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的通知ID"})
		return
	}

	svc := di.GlobalContainer.NotificationService
	notification, err := svc.MarkAsRead(userID.(uint), uint(notificationID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "通知不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "标记已读成功",
		"data":    notification,
	})
}

func MarkAllNotificationsAsRead(c *gin.Context) {
	userID, _ := c.Get("user_id")

	svc := di.GlobalContainer.NotificationService
	if err := svc.MarkAllAsRead(userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "标记所有通知已读失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "标记所有通知已读成功",
	})
}

func ClearAllNotifications(c *gin.Context) {
	userID, _ := c.Get("user_id")

	svc := di.GlobalContainer.NotificationService
	if err := svc.ClearAll(userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "清空通知失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "清空通知成功",
	})
}

func HandleNotificationAction(c *gin.Context) {
	userID, _ := c.Get("user_id")
	notificationIDStr := c.Param("id")

	notificationID, err := strconv.ParseUint(notificationIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的通知ID"})
		return
	}

	var req struct {
		Action string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	svc := di.GlobalContainer.NotificationService
	notification, err := svc.GetByID(userID.(uint), uint(notificationID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "通知不存在"})
		return
	}

	db := di.GlobalContainer.DB
	switch req.Action {
	case "accept":
		handleAcceptAction(db, notification)
	case "ignore":
		handleIgnoreAction(db, notification)
	case "confirm":
		handleConfirmAction(db, notification)
	case "reschedule":
		handleRescheduleAction(db, notification)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不支持的操作"})
		return
	}

	notification.Handled = true
	now := time.Now()
	notification.ReadAt = &now
	svc.Save(notification)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "操作成功",
		"data":    notification,
	})
}

func handleAcceptAction(db *gorm.DB, notification *model.Notification) {
	if notification.Type == "group_invitation" {
		var convID uint
		if _, err := fmt.Sscanf(notification.ActionPayload, `{"conversation_id":%d}`, &convID); err != nil || convID == 0 {
			return
		}
		var existing model.ConversationMember
		if err := db.Where("conversation_id = ? AND user_id = ?", convID, notification.UserID).First(&existing).Error; err == nil {
			return
		}
		member := model.ConversationMember{
			ConversationID: convID,
			UserID:         notification.UserID,
			Role:           "member",
			JoinedAt:       time.Now(),
		}
		db.Create(&member)
	}
}

func handleIgnoreAction(db *gorm.DB, notification *model.Notification) {
}

func handleConfirmAction(db *gorm.DB, notification *model.Notification) {
	if notification.Type == "todo_assigned" {
		var taskID uint
		if _, err := fmt.Sscanf(notification.ActionPayload, `{"task_id":%d}`, &taskID); err == nil && taskID > 0 {
			db.Model(&model.Task{}).Where("id = ?", taskID).Update("status", "in_progress")
		}
	}
}

func handleRescheduleAction(db *gorm.DB, notification *model.Notification) {
}

func TogglePinNotification(c *gin.Context) {
	userID, _ := c.Get("user_id")
	notificationIDStr := c.Param("id")

	notificationID, err := strconv.ParseUint(notificationIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的通知ID"})
		return
	}

	svc := di.GlobalContainer.NotificationService
	pinned, err := svc.TogglePin(userID.(uint), uint(notificationID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "通知不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "操作成功",
		"pinned":  pinned,
	})
}

func ToggleImportantNotification(c *gin.Context) {
	userID, _ := c.Get("user_id")
	notificationIDStr := c.Param("id")

	notificationID, err := strconv.ParseUint(notificationIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的通知ID"})
		return
	}

	svc := di.GlobalContainer.NotificationService
	important, err := svc.ToggleImportant(userID.(uint), uint(notificationID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "通知不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":      0,
		"message":   "操作成功",
		"important": important,
	})
}

func GetEvents(c *gin.Context) {
	userID, _ := c.Get("user_id")

	svc := di.GlobalContainer.EventService
	events, err := svc.GetEvents(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取事件失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": events,
	})
}

func CreateEvent(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Title       string    `json:"title" binding:"required"`
		Description string    `json:"description"`
		Start       time.Time `json:"start" binding:"required"`
		End         time.Time `json:"end" binding:"required"`
		AllDay      bool      `json:"all_day"`
		Reminder    int       `json:"reminder"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	svc := di.GlobalContainer.EventService
	event := &model.Event{
		UserID:      userID.(uint),
		Title:       req.Title,
		Description: req.Description,
		Start:       req.Start,
		End:         req.End,
		AllDay:      req.AllDay,
		Reminder:    req.Reminder,
	}
	if err := svc.CreateEvent(event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建事件失败"})
		return
	}

	if req.Reminder > 0 {
		go func() {
			svc.CreateReminderNotification(userID.(uint), event)

			notification := model.Notification{
				UserID:        userID.(uint),
				Type:          "event_reminder",
				Title:         "事件提醒",
				Content:       fmt.Sprintf("您设置的事件「%s」即将开始", event.Title),
				Priority:      "important",
				ActionType:    "confirm_reschedule",
				ActionPayload: fmt.Sprintf(`{"event_id":%d}`, event.ID),
			}
			di.GlobalContainer.NotificationService.Create(&notification)

			if ws.GlobalHub != nil {
				notificationMsg := ws.WSMessage{
					Type: "notification",
					Data: notification,
				}
				jsonMsg, _ := json.Marshal(notificationMsg)
				ws.GlobalHub.SendToUser(userID.(uint), jsonMsg)
			}
		}()
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": event,
	})
}

func GetEvent(c *gin.Context) {
	userID, _ := c.Get("user_id")
	eventIDStr := c.Param("id")

	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的事件ID"})
		return
	}

	svc := di.GlobalContainer.EventService
	event, err := svc.GetEvent(userID.(uint), uint(eventID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "事件不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": event,
	})
}

func UpdateEvent(c *gin.Context) {
	userID, _ := c.Get("user_id")
	eventIDStr := c.Param("id")

	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的事件ID"})
		return
	}

	var req struct {
		Title       string    `json:"title" binding:"required"`
		Description string    `json:"description"`
		Start       time.Time `json:"start" binding:"required"`
		End         time.Time `json:"end" binding:"required"`
		AllDay      bool      `json:"all_day"`
		Reminder    int       `json:"reminder"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	svc := di.GlobalContainer.EventService
	updates := &model.Event{
		Title:       req.Title,
		Description: req.Description,
		Start:       req.Start,
		End:         req.End,
		AllDay:      req.AllDay,
		Reminder:    req.Reminder,
	}
	event, err := svc.UpdateEvent(userID.(uint), uint(eventID), updates)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "事件不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": event,
	})
}

func DeleteEvent(c *gin.Context) {
	userID, _ := c.Get("user_id")
	eventIDStr := c.Param("id")

	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的事件ID"})
		return
	}

	svc := di.GlobalContainer.EventService
	if err := svc.DeleteEvent(userID.(uint), uint(eventID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除事件失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除事件成功",
	})
}

func GetTasks(c *gin.Context) {
	userID, _ := c.Get("user_id")

	svc := di.GlobalContainer.TaskService
	tasks, err := svc.GetTasks(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取任务失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": tasks,
	})
}

func CreateTask(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
		DueDate     string `json:"due_date"`
		Priority    string `json:"priority"`
		Status      string `json:"status"`
		AssigneeID  string `json:"assignee_id"`
		Tags        string `json:"tags"`
		SubTasks    string `json:"sub_tasks"`
		Position    *int   `json:"position"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	var dueDate *time.Time
	if req.DueDate != "" {
		parsedDate, err := time.Parse("2006-01-02", req.DueDate)
		if err == nil {
			dueDate = &parsedDate
		}
	}

	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}

	status := req.Status
	if status == "" {
		status = "todo"
	}

	position := 0
	if req.Position != nil {
		position = *req.Position
	}

	svc := di.GlobalContainer.TaskService
	task := &model.Task{
		UserID:      userID.(uint),
		Title:       req.Title,
		Description: req.Description,
		DueDate:     dueDate,
		Priority:    priority,
		Status:      status,
		AssigneeID:  req.AssigneeID,
		Tags:        req.Tags,
		SubTasks:    req.SubTasks,
		Position:    position,
	}

	if err := svc.CreateTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建任务失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": task,
	})
}

func UpdateTask(c *gin.Context) {
	userID, _ := c.Get("user_id")
	taskIDStr := c.Param("id")

	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的任务ID"})
		return
	}

	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		DueDate     string `json:"due_date"`
		Priority    string `json:"priority"`
		Status      string `json:"status"`
		AssigneeID  string `json:"assignee_id"`
		Tags        string `json:"tags"`
		SubTasks    string `json:"sub_tasks"`
		Position    *int   `json:"position"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	updates := make(map[string]interface{})
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.DueDate != "" {
		parsedDate, err := time.Parse("2006-01-02", req.DueDate)
		if err == nil {
			updates["due_date"] = &parsedDate
		}
	}
	if req.Priority != "" {
		updates["priority"] = req.Priority
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.AssigneeID != "" {
		updates["assignee_id"] = req.AssigneeID
	}
	if req.Tags != "" {
		updates["tags"] = req.Tags
	}
	if req.SubTasks != "" {
		updates["sub_tasks"] = req.SubTasks
	}
	if req.Position != nil {
		updates["position"] = *req.Position
	}

	svc := di.GlobalContainer.TaskService
	task, err := svc.UpdateTask(userID.(uint), uint(taskID), updates)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "任务不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": task,
	})
}

func DeleteTask(c *gin.Context) {
	userID, _ := c.Get("user_id")
	taskIDStr := c.Param("id")

	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的任务ID"})
		return
	}

	svc := di.GlobalContainer.TaskService
	if err := svc.DeleteTask(userID.(uint), uint(taskID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除任务失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除任务成功",
	})
}

func UpdateTaskStatus(c *gin.Context) {
	userID, _ := c.Get("user_id")
	taskIDStr := c.Param("id")

	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的任务ID"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	svc := di.GlobalContainer.TaskService
	task, err := svc.UpdateTaskStatus(userID.(uint), uint(taskID), req.Status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "任务不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": task,
	})
}
