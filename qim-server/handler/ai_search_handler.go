package handler

import (
	"fmt"
	"net/http"
	"qim-server/ai"
	"qim-server/database"
	"qim-server/model"
	"strings"

	"github.com/gin-gonic/gin"
)

// AISearchRequest 语义搜索请求
type AISearchRequest struct {
	ConversationID uint   `json:"conversation_id" binding:"required"`
	Query          string `json:"query" binding:"required"`
	SenderID       *uint  `json:"sender_id"`
	StartTime      string `json:"start_time"`
	EndTime        string `json:"end_time"`
	Limit          int    `json:"limit"`
}

// AISearchResult 搜索结果
type AISearchResult struct {
	MessageID      uint   `json:"message_id"`
	Content        string `json:"content"`
	SenderName     string `json:"sender_name"`
	Timestamp      string `json:"timestamp"`
	RelevanceScore int    `json:"relevance_score"`
	Highlighted    string `json:"highlighted"`
}

// AISearch 语义搜索消息
func (h *AIHandler) AISearch(c *gin.Context) {
	var req AISearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 20
	}

	db := database.GetDB()

	// 先进行数据库全文搜索(快速召回)
	query := db.Model(&model.Message{}).
		Where("conversation_id = ? AND type = 'text'", req.ConversationID)

	if req.SenderID != nil {
		query = query.Where("sender_id = ?", *req.SenderID)
	}

	if req.StartTime != "" {
		query = query.Where("created_at >= ?", req.StartTime)
	}

	if req.EndTime != "" {
		query = query.Where("created_at <= ?", req.EndTime)
	}

	var messages []model.Message
	query.Preload("Sender").
		Order("created_at DESC").
		Limit(req.Limit * 3). // 多取一些供 AI 排序
		Find(&messages)

	if len(messages) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "success",
			"data": gin.H{
				"results": []AISearchResult{},
				"total":   0,
			},
		})
		return
	}

	// 使用 AI 对搜索结果进行相关性排序
	searchContext := ""
	for _, msg := range messages {
		senderName := msg.Sender.Nickname
		if senderName == "" {
			senderName = msg.Sender.Username
		}
		searchContext += fmt.Sprintf("ID:%d [%s]: %s\n", msg.ID, senderName, msg.Content)
	}

	searchPrompt := fmt.Sprintf(`你是一个专业的搜索相关性评估助手。用户搜索: "%s"

请分析以下消息列表,选出与搜索最相关的消息,并按相关性从高到低排序。
只返回相关的消息ID列表,用逗号分隔,最多返回20条。

消息列表:
%s

只返回消息ID列表,例如: 123,456,789`, req.Query, searchContext)

	messagesInput := []ai.Message{
		{Role: "system", Content: searchPrompt},
		{Role: "user", Content: "请排序"},
	}

	aiResponse, err := h.aiService.GetCompletion(messagesInput)
	if err != nil {
		// AI 排序失败,直接返回数据库搜索结果
		results := buildSearchResults(messages, req.Query, req.Limit)
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "success",
			"data": gin.H{
				"results": results,
				"total":   len(results),
			},
		})
		return
	}

	// 解析 AI 返回的 ID 列表
	orderedIDs := parseOrderedIDs(aiResponse)
	if len(orderedIDs) > 0 {
		messages = reorderMessagesByID(messages, orderedIDs)
	}

	// 构建搜索结果
	results := buildSearchResults(messages, req.Query, req.Limit)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"results": results,
			"total":   len(results),
		},
	})
}

// parseOrderedIDs 从 AI 响应中解析有序 ID 列表
func parseOrderedIDs(response string) []uint {
	// 提取所有数字
	var ids []uint
	parts := strings.Split(response, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// 尝试从文本中提取数字
		var id uint
		if _, err := fmt.Sscanf(part, "%d", &id); err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

// reorderMessagesByID 根据 ID 列表重排消息
func reorderMessagesByID(messages []model.Message, orderedIDs []uint) []model.Message {
	idMap := make(map[uint]model.Message)
	for _, msg := range messages {
		idMap[msg.ID] = msg
	}

	var reordered []model.Message
	for _, id := range orderedIDs {
		if msg, ok := idMap[id]; ok {
			reordered = append(reordered, msg)
			delete(idMap, id)
		}
	}

	// 添加剩余未排序的消息
	for _, msg := range messages {
		if _, exists := idMap[msg.ID]; exists {
			reordered = append(reordered, msg)
		}
	}

	return reordered
}

// buildSearchResults 构建搜索结果
func buildSearchResults(messages []model.Message, query string, limit int) []AISearchResult {
	var results []AISearchResult

	for _, msg := range messages {
		if len(results) >= limit {
			break
		}

		senderName := msg.Sender.Nickname
		if senderName == "" {
			senderName = msg.Sender.Username
		}

		// 简单相关性计算(关键词匹配度)
		score := calculateRelevance(msg.Content, query)

		results = append(results, AISearchResult{
			MessageID:      msg.ID,
			Content:        msg.Content,
			SenderName:     senderName,
			Timestamp:      msg.CreatedAt.Format("2006-01-02 15:04"),
			RelevanceScore: score,
			Highlighted:    highlightText(msg.Content, query),
		})
	}

	return results
}

// calculateRelevance 简单相关性计算
func calculateRelevance(content string, query string) int {
	score := 0
	contentLower := strings.ToLower(content)
	queryLower := strings.ToLower(query)

	// 分词查询
	words := strings.Fields(queryLower)
	for _, word := range words {
		if len(word) < 2 {
			continue
		}
		// 统计词出现次数
		count := strings.Count(contentLower, word)
		score += count * 10
	}

	// 完整查询匹配加分
	if strings.Contains(contentLower, queryLower) {
		score += 50
	}

	return score
}

// highlightText 高亮关键词
func highlightText(content string, query string) string {
	words := strings.Fields(query)
	result := content
	for _, word := range words {
		if len(word) < 2 {
			continue
		}
		// 简单高亮: 使用 <mark> 标签
		result = strings.ReplaceAll(result, word, "<mark>"+word+"</mark>")
		result = strings.ReplaceAll(result, strings.ToLower(word), "<mark>"+strings.ToLower(word)+"</mark>")
	}
	return result
}
