package handler

import (
	"fmt"
	"net/http"
	"qim-server/ai"
	"qim-server/database"
	"qim-server/model"
	"time"

	"github.com/gin-gonic/gin"
)

// GenerateSummaryRequest 生成摘要请求
type GenerateSummaryRequest struct {
	ConversationID uint       `json:"conversation_id" binding:"required"`
	TimeRange      string     `json:"time_range"` // "1h", "today", "7d", "custom"
	StartTime      *time.Time `json:"start_time"`
	EndTime        *time.Time `json:"end_time"`
}

// GenerateSummary 生成会话摘要
func (h *AIHandler) GenerateSummary(c *gin.Context) {
	var req GenerateSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if !h.aiService.IsConfigured() {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "AI服务未配置"})
		return
	}

	db := database.GetDB()

	// 获取会话信息
	var conv model.Conversation
	if err := db.First(&conv, req.ConversationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	// 计算时间范围
	var startTime time.Time
	endTime := time.Now()

	switch req.TimeRange {
	case "1h":
		startTime = endTime.Add(-1 * time.Hour)
	case "today":
		startTime = time.Now().Truncate(24 * time.Hour)
	case "7d":
		startTime = endTime.Add(-7 * 24 * time.Hour)
	case "custom":
		if req.StartTime != nil && req.EndTime != nil {
			startTime = *req.StartTime
			endTime = *req.EndTime
		} else {
			startTime = endTime.Add(-24 * time.Hour)
		}
	default:
		startTime = endTime.Add(-24 * time.Hour)
	}

	// 获取消息
	var messages []model.Message
	db.Where("conversation_id = ? AND created_at >= ? AND created_at <= ?",
		req.ConversationID, startTime, endTime).
		Preload("Sender").
		Order("created_at ASC").
		Limit(200).
		Find(&messages)

	if len(messages) < 3 {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "success",
			"data": gin.H{
				"summary":        "该时间段内消息较少,无需生成摘要。",
				"messages_count": len(messages),
				"time_range":     fmt.Sprintf("%s 至 %s", startTime.Format("15:04"), endTime.Format("15:04")),
			},
		})
		return
	}

	// 构建消息文本
	messagesText := ""
	for _, msg := range messages {
		senderName := msg.Sender.Nickname
		if senderName == "" {
			senderName = msg.Sender.Username
		}
		messagesText += fmt.Sprintf("[%s] %s: %s\n", msg.CreatedAt.Format("15:04"), senderName, msg.Content)
	}

	// 构建系统提示
	systemPrompt := `你是一个专业的会议摘要助手。请分析以下聊天记录,生成结构化的会话摘要。

请严格按照以下格式输出:

📋 会话摘要
⏰ 时间范围: [起止时间]
📊 消息数量: X 条

🔥 核心话题
1. [话题一] - [简要说明] (讨论热度: 高/中/低, 参与人数: X)
2. [话题二] - [简要说明] (讨论热度: 高/中/低, 参与人数: X)

✅ 重要决策
- [决策一] (决策人: [姓名])
- [决策二] (决策人: [姓名])

📌 待办事项
- [ ] [待办一] (负责人: [姓名])
- [ ] [待办二] (负责人: [姓名])

💬 关键发言
- [姓名]: [重要观点摘要]
- [姓名]: [重要观点摘要]

如果某些部分没有内容,请省略该部分。保持简洁专业。`

	messagesInput := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: messagesText},
	}

	summary, err := h.aiService.GetCompletion(messagesInput)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "摘要生成失败: " + err.Error()})
		return
	}

	// 过滤输出
	summary = h.aiService.FilterOutput(summary, "ai_summary")

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"summary":        summary,
			"messages_count": len(messages),
			"time_range":     fmt.Sprintf("%s 至 %s", startTime.Format("2006-01-02 15:04"), endTime.Format("2006-01-02 15:04")),
		},
	})
}
