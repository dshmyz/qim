package handler

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"qim-server/ai"
	"qim-server/database"
	"qim-server/model"
	"qim-server/ws"

	"github.com/gin-gonic/gin"
)

func GetBots(c *gin.Context) {
	db := database.GetDB()

	var bots []model.Bot
	// 返回：系统 Bot + 模板 Bot + 已审批通过的用户自建 Bot
	db.Where(
		"(creator_id = 0 AND is_active = ?) OR (is_template = ? AND is_active = ? AND approval_status = ?) OR (approval_status = ? AND is_active = ?)",
		true, true, true, "approved", "approved", true,
	).Find(&bots)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": bots,
	})
}

func GetSystemMessages(c *gin.Context) {
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

	offset := (page - 1) * pageSize

	db := database.GetDB()
	var systemMessages []model.SystemMessage
	var total int64

	db.Model(&model.SystemMessage{}).Count(&total)
	db.Preload("Sender").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&systemMessages)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  systemMessages,
			"total": total,
			"page":  page,
		},
	})
}

func CreateSystemMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Title      string `json:"title" binding:"required"`
		Content    string `json:"content" binding:"required"`
		TargetType string `json:"target_type"`
		TargetID   *uint  `json:"target_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	systemMessage := model.SystemMessage{
		Title:      req.Title,
		Content:    req.Content,
		SenderID:   userID.(uint),
		Status:     "active",
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
	}

	if err := db.Create(&systemMessage).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建系统消息失败"})
		return
	}

	db.Preload("Sender").First(&systemMessage, systemMessage.ID)

	var usersToNotify []uint

	switch req.TargetType {
	case "all":
		var allUsers []model.User
		db.Find(&allUsers)
		for _, u := range allUsers {
			usersToNotify = append(usersToNotify, u.ID)
		}
	case "department":
		if req.TargetID != nil {
			var deptEmployees []model.DepartmentEmployee
			db.Where("department_id = ?", *req.TargetID).Find(&deptEmployees)
			for _, de := range deptEmployees {
				usersToNotify = append(usersToNotify, de.UserID)
			}
		}
	case "user":
		if req.TargetID != nil {
			usersToNotify = append(usersToNotify, *req.TargetID)
		}
	default:
		usersToNotify = append(usersToNotify, userID.(uint))
	}

	for _, notifyUserID := range usersToNotify {
		notification := model.Notification{
			UserID:  notifyUserID,
			Type:    "system_message",
			Title:   req.Title,
			Content: req.Content,
		}
		db.Create(&notification)

		if ws.GlobalHub != nil {
			notificationMsg := ws.WSMessage{
				Type: "notification",
				Data: notification,
			}
			jsonMsg, _ := json.Marshal(notificationMsg)
			ws.GlobalHub.SendToUser(notifyUserID, jsonMsg)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": systemMessage,
	})
}

func UpdateSystemMessage(c *gin.Context) {
	messageIDStr := c.Param("id")

	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的消息ID"})
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

	var systemMessage model.SystemMessage
	if err := db.First(&systemMessage, uint(messageID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "消息不存在"})
		return
	}

	systemMessage.Status = req.Status
	if err := db.Save(&systemMessage).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新消息状态失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": systemMessage,
	})
}

func BroadcastMessage(c *gin.Context) {
	var req struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}

	if ws.GlobalHub != nil {
		ws.GlobalHub.Broadcast <- []byte(req.Message)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "消息广播成功"})
}

func SendToUserMessage(c *gin.Context) {
	var req struct {
		UserID  uint   `json:"user_id"`
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}

	if ws.GlobalHub != nil {
		ws.GlobalHub.SendToUser(req.UserID, []byte(req.Message))
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "消息发送成功"})
}

func HandleBotMessage(userID uint, convID uint, content string) {
	db := database.GetDB()

	var botConv model.BotConversation
	if err := db.Where("conversation_id = ?", convID).First(&botConv).Error; err != nil {
		return
	}

	var bot model.Bot
	if err := db.First(&bot, botConv.BotID).Error; err != nil {
		log.Printf("[HandleBotMessage] 查找 Bot 失败: %v", err)
		return
	}

	// 检查是否有虚拟用户
	if bot.VirtualUserID == nil {
		log.Printf("[HandleBotMessage] Bot 没有虚拟用户: %d", botConv.BotID)
		return
	}

	var reply string
	switch bot.Type {
	case "system":
		reply = getSystemBotReply(content)
	case "ai":
		if aiService != nil && aiService.IsConfigured() {
			var messages []ai.Message

			systemPrompt := "你是一个智能助手，帮助用户解决问题。"
			if bot.Config != "" {
				var botConfig map[string]interface{}
				if err := json.Unmarshal([]byte(bot.Config), &botConfig); err == nil {
					if prompt, ok := botConfig["system_prompt"].(string); ok {
						systemPrompt = prompt
					}
				}
			}

			messages = append(messages, ai.Message{
				Role:    "system",
				Content: systemPrompt,
			})

			var historyMessages []model.Message
			db.Where("conversation_id = ?", convID).Order("created_at DESC").Limit(10).Find(&historyMessages)

			for i, j := 0, len(historyMessages)-1; i < j; i, j = i+1, j-1 {
				historyMessages[i], historyMessages[j] = historyMessages[j], historyMessages[i]
			}

			for _, msg := range historyMessages {
				role := "user"
				if msg.SenderID == *bot.VirtualUserID {
					role = "assistant"
				}
				messages = append(messages, ai.Message{
					Role:    role,
					Content: msg.Content,
				})
			}

			messages = append(messages, ai.Message{
				Role:    "user",
				Content: content,
			})

			var err error
			reply, err = aiService.GetCompletion(messages)
			if err != nil {
				log.Printf("AI API error: %v", err)
				reply = "抱歉，AI服务暂时不可用，请稍后再试。"
			}
		} else {
			reply = "AI服务未配置，请联系管理员。"
		}
	default:
		reply = "我是一个机器人，有什么可以帮你的吗？"
	}

	// 保存 Bot 回复
	msg := model.Message{
		ConversationID: convID,
		SenderID:       *bot.VirtualUserID, // 使用虚拟用户 ID
		Type:           "markdown",
		Content:        reply,
	}
	db.Create(&msg)

	// 预加载 Sender 信息（虚拟用户）
	db.Preload("Sender").First(&msg, msg.ID)

	now := time.Now()
	var conv model.Conversation
	db.First(&conv, convID)
	conv.LastMessageID = &msg.ID
	conv.LastMessageAt = &now
	db.Save(&conv)

	wsMsg := ws.WSMessage{
		Type: "new_message",
		Data: msg,
	}
	jsonMsg, _ := json.Marshal(wsMsg)

	if ws.GlobalHub != nil {
		ws.GlobalHub.SendToUser(userID, jsonMsg)
	}
}

func getSystemBotReply(content string) string {
	content = strings.ToLower(content)
	if strings.Contains(content, "你好") || strings.Contains(content, "hi") || strings.Contains(content, "hello") {
		return "你好！我是系统助手，有什么可以帮你的吗？"
	} else if strings.Contains(content, "帮助") || strings.Contains(content, "help") {
		return "我可以帮助你了解系统功能，解答常见问题。你可以问我关于系统使用的问题。"
	} else if strings.Contains(content, "时间") || strings.Contains(content, "time") {
		return "当前时间是：" + time.Now().Format("2006-01-02 15:04:05")
	} else {
		return "我是系统助手，有什么可以帮你的吗？"
	}
}

func getAIBotReply(content string) string {
	replies := []string{
		"这是一个有趣的问题！让我想想...",
		"根据我的理解，你是在问关于...",
		"好的，我来帮你解答这个问题。",
		"这个问题很有意思，我认为...",
		"让我分析一下这个问题...",
	}
	return replies[rand.Intn(len(replies))] + "\n\n你刚才说：" + content
}
