package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"qim-server/database"
	"qim-server/model"
	"qim-server/ws"

	"github.com/gin-gonic/gin"
)

func GetNotifications(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()
	var notifications []model.Notification
	db.Where("user_id = ?", userID).Order("created_at DESC").Find(&notifications)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": notifications,
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

	db := database.GetDB()
	var notification model.Notification
	if err := db.Where("id = ? AND user_id = ?", uint(notificationID), userID).First(&notification).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "通知不存在"})
		return
	}

	notification.Read = true
	now := time.Now()
	notification.ReadAt = &now
	db.Save(&notification)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "标记已读成功",
		"data":    notification,
	})
}

func MarkAllNotificationsAsRead(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()
	now := time.Now()

	db.Model(&model.Notification{}).Where("user_id = ? AND read = ?", userID, false).Updates(map[string]interface{}{
		"read":    true,
		"read_at": now,
	})

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "标记所有通知已读成功",
	})
}

func GetEvents(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()
	var events []model.Event
	db.Where("user_id = ?", userID).Order("start DESC").Find(&events)

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

	db := database.GetDB()
	event := model.Event{
		UserID:      userID.(uint),
		Title:       req.Title,
		Description: req.Description,
		Start:       req.Start,
		End:         req.End,
		AllDay:      req.AllDay,
		Reminder:    req.Reminder,
	}
	db.Create(&event)

	if req.Reminder > 0 {
		go func() {
			reminderTime := req.Start.Add(-time.Duration(req.Reminder) * time.Minute)
			now := time.Now()

			if reminderTime.Before(now) {
				return
			}

			waitDuration := reminderTime.Sub(now)
			timer := time.NewTimer(waitDuration)
			<-timer.C

			var currentEvent model.Event
			if err := db.First(&currentEvent, event.ID).Error; err != nil {
				return
			}

			if time.Now().After(currentEvent.End) {
				return
			}

			notification := model.Notification{
				UserID:  userID.(uint),
				Type:    "event_reminder",
				Title:   "事件提醒",
				Content: fmt.Sprintf("您设置的事件「%s」即将开始", currentEvent.Title),
			}
			db.Create(&notification)

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

	db := database.GetDB()
	var event model.Event
	if err := db.Where("id = ? AND user_id = ?", uint(eventID), userID).First(&event).Error; err != nil {
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

	db := database.GetDB()
	var event model.Event
	if err := db.Where("id = ? AND user_id = ?", uint(eventID), userID).First(&event).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "事件不存在"})
		return
	}

	event.Title = req.Title
	event.Description = req.Description
	event.Start = req.Start
	event.End = req.End
	event.AllDay = req.AllDay
	event.Reminder = req.Reminder
	db.Save(&event)

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

	db := database.GetDB()
	if err := db.Where("id = ? AND user_id = ?", uint(eventID), userID).Delete(&model.Event{}).Error; err != nil {
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

	db := database.GetDB()
	var tasks []model.Task
	db.Where("user_id = ?", userID).Order("created_at DESC").Find(&tasks)

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
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

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

	task := model.Task{
		UserID:      userID.(uint),
		Title:       req.Title,
		Description: req.Description,
		DueDate:     dueDate,
		Priority:    priority,
		Status:      status,
	}

	if err := db.Create(&task).Error; err != nil {
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
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	var task model.Task
	if err := db.Where("id = ? AND user_id = ?", uint(taskID), userID).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "任务不存在"})
		return
	}

	if req.Title != "" {
		task.Title = req.Title
	}
	if req.Description != "" {
		task.Description = req.Description
	}
	if req.DueDate != "" {
		parsedDate, err := time.Parse("2006-01-02", req.DueDate)
		if err == nil {
			task.DueDate = &parsedDate
		}
	}
	if req.Priority != "" {
		task.Priority = req.Priority
	}
	if req.Status != "" {
		task.Status = req.Status
	}

	if err := db.Save(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新任务失败"})
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

	db := database.GetDB()
	if err := db.Where("id = ? AND user_id = ?", uint(taskID), userID).Delete(&model.Task{}).Error; err != nil {
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

	db := database.GetDB()
	var task model.Task
	if err := db.Where("id = ? AND user_id = ?", uint(taskID), userID).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "任务不存在"})
		return
	}

	task.Status = req.Status

	if err := db.Save(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新任务状态失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": task,
	})
}
