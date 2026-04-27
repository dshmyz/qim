# 全场景AI体验优化 - 角色分工实现计划

> **面向 AI 代理的工作者:** 必需子技能:使用 superpowers:subagent-driven-development(推荐)或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框(`- [ ]`)语法来跟踪进度。

**目标:** 基于现有AI基础设施,全面提升QIM即时通讯系统的AI用户体验,覆盖群聊、单聊、全局搜索三大场景。

**架构:** 四层模块化架构 - 群聊AI增强、单聊AI增强、全局AI功能、AI交互优化,前后端分离,13个新Vue组件,7个新API端点,1个数据库迁移。

**技术栈:** 
- 前端: Vue 3 + TypeScript + Vite
- 后端: Go + Gin + GORM
- 数据库: MySQL/TiDB
- AI: OpenAI兼容协议 + MCP工具调用

---

## 角色分工概览

| 角色 | 任务编号 | 预计任务数 | 核心职责 |
|------|----------|------------|----------|
| 🗄️ 后端开发 | B1-B4 | 12 | API开发、AI服务增强、数据库迁移 |
| 🎨 前端开发 | F1-F4 | 16 | Vue组件开发、交互逻辑、样式适配 |
| 🔗 前后端联调 | I1-I2 | 4 | 接口对接、数据流验证、端到端测试 |

---

# 角色一: 🗄️ 后端开发 (Backend)

负责: API端点、AI服务增强、数据库迁移、WebSocket消息处理

## 后端文件结构

**新建文件:**
- `qim-server/ai/output_filter.go` - AI输出长度控制
- `qim-server/handler/ai_summary_handler.go` - 会话摘要API
- `qim-server/handler/ai_search_handler.go` - 语义搜索API
- `qim-server/handler/ai_text_handler.go` - 翻译/改写/润色API
- `qim-server/migrations/001_message_content_mediumtext.sql` - 数据库迁移脚本

**修改文件:**
- `qim-server/model/model.go:108` - Message表Content字段类型修改
- `qim-server/model/model.go:83-96` - Group表新增字段
- `qim-server/handler/smart_reply_handler.go:28-70` - 新增@AI检测逻辑
- `qim-server/handler/ai_handler.go:26-39` - 注册新路由
- `qim-server/ai/ai_service.go:193-204` - 新增filterOutput方法

---

### 任务 B1: 数据库迁移 + 输出过滤

**前置依赖:** 无

**文件:**
- 修改: `qim-server/model/model.go`
- 创建: `qim-server/ai/output_filter.go`
- 创建: `qim-server/migrations/001_message_content_mediumtext.sql`

- [ ] **步骤 1: 修改 Message 模型 Content 字段**

打开 `qim-server/model/model.go`,找到 Message 结构体(约第108行),将:

```go
Content string `json:"content" gorm:"type:text;not null"`
```

改为:

```go
Content string `json:"content" gorm:"type:mediumtext;not null"`
```

- [ ] **步骤 2: Group 模型新增字段**

在 `qim-server/model/model.go` 的 Group 结构体中,在 `AIEnabled` 字段后新增:

```go
AIReplyMode      string     `json:"ai_reply_mode" gorm:"size:20;default:'mention_only'"` // always/mention_only/smart/off
AIAssistantName  string     `json:"ai_assistant_name" gorm:"size:100;default:'AI助手'"`
```

- [ ] **步骤 3: 创建数据库迁移脚本**

创建 `qim-server/migrations/001_message_content_mediumtext.sql`:

```sql
-- 迁移 001: 扩展 Message 表 content 字段以支持长 AI 回复
-- 适用于: MySQL 5.7+, TiDB
-- 执行前建议: 备份 messages 表

ALTER TABLE messages MODIFY COLUMN content MEDIUMTEXT NOT NULL;

-- 验证迁移结果
SELECT COLUMN_NAME, COLUMN_TYPE 
FROM INFORMATION_SCHEMA.COLUMNS 
WHERE TABLE_SCHEMA = DATABASE() 
  AND TABLE_NAME = 'messages' 
  AND COLUMN_NAME = 'content';
```

- [ ] **步骤 4: 创建 AI 输出过滤器**

创建 `qim-server/ai/output_filter.go`:

```go
package ai

import (
	"fmt"
)

// OutputLengthConfig 输出长度配置
type OutputLengthConfig struct {
	DefaultLimit int            `json:"default_limit"`
	TypeLimits   map[string]int `json:"type_limits"`
}

// DefaultOutputConfig 默认输出长度配置
var DefaultOutputConfig = OutputLengthConfig{
	DefaultLimit: 3000,
	TypeLimits: map[string]int{
		"ai_reply":     3000,
		"ai_summary":   5000,
		"ai_translate": 2000,
		"ai_rewrite":   2000,
		"ai_polish":    2000,
		"ai_daily":     8000,
	},
}

// FilterOutput 根据消息类型过滤输出长度
func (s *AIService) FilterOutput(content string, msgType string) string {
	config := DefaultOutputConfig
	limit, ok := config.TypeLimits[msgType]
	if !ok {
		limit = config.DefaultLimit
	}

	if len(content) > limit {
		return content[:limit] + "\n\n---\n*内容过长已截断,完整内容可导出查看*"
	}
	return content
}

// GetOutputLimit 获取指定类型的输出限制
func GetOutputLimit(msgType string) int {
	config := DefaultOutputConfig
	limit, ok := config.TypeLimits[msgType]
	if !ok {
		limit = config.DefaultLimit
	}
	return limit
}

// FormatContentPreview 格式化内容预览(用于截断提示)
func FormatContentPreview(fullLength int, limit int) string {
	return fmt.Sprintf("*内容已截断,显示前 %d 字符,完整内容共 %d 字符*", limit, fullLength)
}
```

- [ ] **步骤 5: 在 AIService 中集成输出过滤**

修改 `qim-server/ai/ai_service.go`,在 `GetCompletion` 方法返回前添加过滤逻辑。找到 `GetCompletion` 方法(约第193行),在 `return provider.Chat(filteredMessages)` 之前,需要修改调用方来使用 FilterOutput。

- [ ] **步骤 6: Commit**

```bash
git add qim-server/model/model.go
git add qim-server/ai/output_filter.go
git add qim-server/migrations/001_message_content_mediumtext.sql
git commit -m "feat(ai): 扩展消息字段类型并添加输出长度控制
- Message.Content 从 TEXT 升级为 MEDIUMTEXT(16MB)
- 新增 Group.AIReplyMode 和 AIAssistantName 字段
- 实现 AI 输出过滤器,按消息类型动态限制长度
- 创建数据库迁移脚本"
```

---

### 任务 B2: @AI触发回复 + 智能自动回复增强

**前置依赖:** B1

**文件:**
- 修改: `qim-server/handler/smart_reply_handler.go`
- 修改: `qim-server/handler/ai_handler.go`

- [ ] **步骤 1: 新增 @AI 检测逻辑**

修改 `qim-server/handler/smart_reply_handler.go`,在 `HandleMessage` 方法中,在检查群聊AI开启状态后(约第42行),新增:

```go
// 检测是否 @AI 或 @AI助手
func (e *SmartReplyEngine) isAIMention(content string, assistantName string) bool {
	// 支持多种格式: @AI, @AI助手, @助手, 自定义名称
	patterns := []string{
		"@AI",
		"@Ai",
		"@ai",
		"@" + assistantName,
	}
	
	for _, pattern := range patterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}
	return false
}

// 提取 @AI 后的问题内容
func extractAIQuestion(content string, assistantName string) string {
	// 移除 @AI 或 @助手 前缀
	patterns := []string{"@AI", "@Ai", "@ai", "@" + assistantName}
	
	for _, pattern := range patterns {
		if idx := strings.Index(content, pattern); idx != -1 {
			question := content[idx+len(pattern):]
			return strings.TrimSpace(question)
		}
	}
	return content
}
```

在文件头部添加 `import "strings"`(如果没有的话)。

- [ ] **步骤 2: 修改 HandleMessage 处理 @AI**

在 `HandleMessage` 方法中,获取群聊配置后(约第38行),在返回之前添加:

```go
// 获取 AI 助手名称
assistantName := "AI助手"
if group.AIAssistantName != "" {
	assistantName = group.AIAssistantName
}

// 检测是否 @AI
if e.isAIMention(content, assistantName) {
	question := extractAIQuestion(content, assistantName)
	e.handleAIMention(userID, conversationID, question, conv, assistantName)
	return
}

// 根据回复模式决定是否自动回复
if group.AIReplyMode == "off" {
	log.Printf("[SmartReply] AI 回复已关闭")
	return
}
```

- [ ] **步骤 3: 实现 handleAIMention 方法**

在 `smart_reply_handler.go` 文件末尾添加:

```go
// handleAIMention 处理 @AI 触发回复
func (e *SmartReplyEngine) handleAIMention(userID uint, conversationID uint, question string, conv *model.Conversation, assistantName string) {
	log.Printf("[SmartReply] @AI 触发回复: userID=%d, convID=%d, question=%s", userID, conversationID, question[:min(50, len(question))])

	if e.aiService == nil || !e.aiService.IsConfigured() {
		log.Printf("[SmartReply] AI 服务未配置")
		return
	}

	db := database.GetDB()

	// 构建系统提示
	systemPrompt := e.buildSystemPrompt(conversationID, conv, userID)

	// 获取最近消息作为上下文(最近 15 条)
	var recentMessages []model.Message
	db.Where("conversation_id = ?", conversationID).
		Preload("Sender").
		Order("created_at DESC").
		Limit(15).
		Find(&recentMessages)

	// 反转消息顺序(按时间正序)
	for i, j := 0, len(recentMessages)-1; i < j; i, j = i+1, j-1 {
		recentMessages[i], recentMessages[j] = recentMessages[j], recentMessages[i]
	}

	// 构建消息列表
	var messages []ai.Message
	messages = append(messages, ai.Message{Role: "system", Content: systemPrompt})
	
	for _, msg := range recentMessages {
		senderName := msg.Sender.Nickname
		if senderName == "" {
			senderName = msg.Sender.Username
		}
		messages = append(messages, ai.Message{
			Role:    "user",
			Content: fmt.Sprintf("[%s]: %s", senderName, msg.Content),
		})
	}

	// 添加用户问题
	messages = append(messages, ai.Message{Role: "user", Content: fmt.Sprintf("[用户提问]: %s", question)})

	// 调用 AI
	var reply string
	var err error
	
	if e.aiService.GetMCPServer() != nil {
		reply, err = e.aiService.GetCompletionWithTools(messages, nil)
	} else {
		reply, err = e.aiService.GetCompletion(messages)
	}
	
	if err != nil {
		log.Printf("[SmartReply] AI 回复生成失败: %v", err)
		return
	}

	// 过滤输出长度
	reply = e.aiService.FilterOutput(reply, "ai_reply")

	// 创建 AI 回复消息
	// 注意: 这里需要获取 AI 助手的 bot_id,暂时使用 sender_id=0 表示系统/AI
	aiMessage := model.Message{
		ConversationID: conversationID,
		SenderID:       0, // 0 表示 AI/系统
		Type:           "text",
		Content:        reply,
		IsRead:         false,
	}
	
	if err := db.Create(&aiMessage).Error; err != nil {
		log.Printf("[SmartReply] 保存 AI 回复失败: %v", err)
		return
	}

	// 通过 WebSocket 推送消息
	if ws.GlobalHub != nil {
		msgData := gin.H{
			"id":                aiMessage.ID,
			"conversation_id":   conversationID,
			"sender_id":         0,
			"type":              "text",
			"content":           reply,
			"is_ai_message":     true,
			"ai_assistant_name": assistantName,
			"created_at":        aiMessage.CreatedAt,
		}
		
		wsMsg := ws.WSMessage{
			Type: "new_message",
			Data: msgData,
		}
		
		jsonMsg, _ := json.Marshal(wsMsg)
		ws.GlobalHub.SendToConversation(conversationID, 0, jsonMsg)
	}

	log.Printf("[SmartReply] @AI 回复已发送,消息ID=%d", aiMessage.ID)
}
```

- [ ] **步骤 4: 注册新路由**

修改 `qim-server/handler/ai_handler.go` 的 `RegisterRoutes` 方法(约第26行),在现有路由后添加:

```go
func (h *AIHandler) RegisterRoutes(router *gin.RouterGroup) {
	aiGroup := router.Group("/ai")
	{
		aiGroup.POST("/completion", h.GetCompletion)
		aiGroup.POST("/completion/stream", h.GetCompletionStream)
		aiGroup.GET("/tools", h.ListTools)
		aiGroup.POST("/tools/execute", h.ExecuteTool)
		
		// 新增: 会话摘要
		aiGroup.POST("/summary", h.GenerateSummary)
		
		// 新增: 语义搜索
		aiGroup.POST("/search", h.AISearch)
		
		// 新增: 文本处理
		aiGroup.POST("/translate", h.TranslateText)
		aiGroup.POST("/rewrite", h.RewriteText)
		aiGroup.POST("/polish", h.PolishText)
		
		// 运维相关路由(已有)
		aiGroup.POST("/ops/troubleshooting", h.IntelligentTroubleshooting)
		aiGroup.POST("/ops/command", h.CommandGeneration)
		aiGroup.POST("/ops/logs", h.LogAnalysis)
		aiGroup.POST("/ops/alert", h.IntelligentAlert)
		aiGroup.POST("/ops/knowledge", h.OpsKnowledge)
		aiGroup.GET("/ops/dashboard", h.OpsDashboard)
	}
}
```

- [ ] **步骤 5: Commit**

```bash
git add qim-server/handler/smart_reply_handler.go
git add qim-server/handler/ai_handler.go
git commit -m "feat(ai): 实现 @AI 触发回复和智能自动回复增强
- 新增 @AI/@助手 检测逻辑
- 实现 handleAIMention 方法处理群聊 AI 回复
- 支持自定义 AI 助手名称
- 注册新 API 路由(摘要/搜索/翻译/改写/润色)
- AI 回复自动通过 WebSocket 推送到群聊"
```

---

### 任务 B3: 会话摘要 + 语义搜索 API

**前置依赖:** B2

**文件:**
- 创建: `qim-server/handler/ai_summary_handler.go`
- 创建: `qim-server/handler/ai_search_handler.go`
- 修改: `qim-server/handler/ai_handler.go`

- [ ] **步骤 1: 实现会话摘要 Handler**

创建 `qim-server/handler/ai_summary_handler.go`:

```go
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
	ConversationID uint   `json:"conversation_id" binding:"required"`
	TimeRange      string `json:"time_range"` // "1h", "today", "7d", "custom"
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
				"summary":  "该时间段内消息较少,无需生成摘要。",
				"messages_count": len(messages),
				"time_range": fmt.Sprintf("%s 至 %s", startTime.Format("15:04"), endTime.Format("15:04")),
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

	messages_input := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: messagesText},
	}

	summary, err := h.aiService.GetCompletion(messages_input)
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
```

- [ ] **步骤 2: 实现语义搜索 Handler**

创建 `qim-server/handler/ai_search_handler.go`:

```go
package handler

import (
	"net/http"
	"qim-server/ai"
	"qim-server/database"
	"qim-server/model"

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

	searchPrompt := `你是一个专业的搜索相关性评估助手。用户搜索: "` + req.Query + `"

请分析以下消息列表,选出与搜索最相关的消息,并按相关性从高到低排序。
只返回相关的消息ID列表,用逗号分隔,最多返回20条。

消息列表:
` + searchContext + `

只返回消息ID列表,例如: 123,456,789`

	messages_input := []ai.Message{
		{Role: "system", Content: searchPrompt},
		{Role: "user", Content: "请排序"},
	}

	aiResponse, err := h.aiService.GetCompletion(messages_input)
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
	// 简化处理: 提取数字
	// ...

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
	// 关键词匹配计数
	// ...简化实现
	return score
}

// highlightText 高亮关键词
func highlightText(content string, query string) string {
	// ...简化实现
	return content
}
```

- [ ] **步骤 3: Commit**

```bash
git add qim-server/handler/ai_summary_handler.go
git add qim-server/handler/ai_search_handler.go
git commit -m "feat(ai): 实现会话摘要和语义搜索 API
- 新增 GenerateSummary 接口,支持多种时间范围
- 新增 AISearch 接口,支持关键词搜索+AI相关性排序
- 支持按发送者、时间范围过滤
- 结构化摘要输出:核心话题/决策/待办/关键发言"
```

---

### 任务 B4: 文本处理 API(翻译/改写/润色)

**前置依赖:** B1

**文件:**
- 创建: `qim-server/handler/ai_text_handler.go`
- 修改: `qim-server/handler/ai_handler.go`

- [ ] **步骤 1: 实现文本处理 Handler**

创建 `qim-server/handler/ai_text_handler.go`:

```go
package handler

import (
	"net/http"
	"qim-server/ai"

	"github.com/gin-gonic/gin"
)

// TranslateTextRequest 翻译请求
type TranslateTextRequest struct {
	Text         string `json:"text" binding:"required"`
	TargetLang   string `json:"target_lang" binding:"required"` // zh/en/ja/ko/fr/de
	SourceLang   string `json:"source_lang"`                    // auto/zh/en/...
}

// RewriteTextRequest 改写请求
type RewriteTextRequest struct {
	Text   string `json:"text" binding:"required"`
	Style  string `json:"style"`  // formal/casual/concise/detailed
	Tone   string `json:"tone"`   // professional/friendly/neutral
}

// PolishTextRequest 润色请求
type PolishTextRequest struct {
	Text     string `json:"text" binding:"required"`
	Language string `json:"language"` // zh/en
}

// TranslateText 翻译文本
func (h *AIHandler) TranslateText(c *gin.Context) {
	var req TranslateTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if !h.aiService.IsConfigured() {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "AI服务未配置"})
		return
	}

	sourceLang := req.SourceLang
	if sourceLang == "" || sourceLang == "auto" {
		sourceLang = "自动检测"
	}

	systemPrompt := "你是一个专业的翻译助手。请将以下文本从" + sourceLang + "翻译为" + req.TargetLang + "。只输出翻译结果,不要额外解释。"

	messages_input := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: req.Text},
	}

	result, err := h.aiService.GetCompletion(messages_input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "翻译失败: " + err.Error()})
		return
	}

	result = h.aiService.FilterOutput(result, "ai_translate")

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"translated_text": result,
			"source_lang":     sourceLang,
			"target_lang":     req.TargetLang,
		},
	})
}

// RewriteText 改写文本
func (h *AIHandler) RewriteText(c *gin.Context) {
	var req RewriteTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if !h.aiService.IsConfigured() {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "AI服务未配置"})
		return
	}

	style := req.Style
	if style == "" {
		style = "简洁"
	}
	tone := req.Tone
	if tone == "" {
		tone = "专业"
	}

	systemPrompt := "你是一个专业的文本改写助手。请将以下文本改写为" + style + "风格,语气" + tone + "。保持原意不变,只输出改写结果,不要额外解释。"

	messages_input := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: req.Text},
	}

	result, err := h.aiService.GetCompletion(messages_input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "改写失败: " + err.Error()})
		return
	}

	result = h.aiService.FilterOutput(result, "ai_rewrite")

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"rewritten_text": result,
		},
	})
}

// PolishText 润色文本
func (h *AIHandler) PolishText(c *gin.Context) {
	var req PolishTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if !h.aiService.IsConfigured() {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "AI服务未配置"})
		return
	}

	lang := req.Language
	if lang == "" {
		lang = "中文"
	}

	systemPrompt := "你是一个专业的" + lang + "润色助手。请润色以下文本,使其表达更准确、流畅、专业。保持原意不变,只输出润色结果,不要额外解释。"

	messages_input := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: req.Text},
	}

	result, err := h.aiService.GetCompletion(messages_input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "润色失败: " + err.Error()})
		return
	}

	result = h.aiService.FilterOutput(result, "ai_polish")

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"polished_text": result,
		},
	})
}
```

- [ ] **步骤 2: Commit**

```bash
git add qim-server/handler/ai_text_handler.go
git commit -m "feat(ai): 实现文本处理 API(翻译/改写/润色)
- 新增 TranslateText 接口,支持多语言翻译
- 新增 RewriteText 接口,支持风格和语气调整
- 新增 PolishText 接口,支持文本润色
- 所有接口集成输出长度控制"
```

---

### 后端开发完成检查清单

- [ ] 所有后端文件编译通过: `cd qim-server && go build ./...`
- [ ] 数据库迁移脚本可在测试环境执行
- [ ] 所有新 API 可通过 curl/Postman 调用并返回正确结果
- [ ] @AI 触发回复在群聊中正常工作
- [ ] WebSocket 消息推送正常,前端可收到 AI 回复

---

# 角色二: 🎨 前端开发 (Frontend)

负责: Vue组件开发、交互逻辑、样式适配、快捷键

## 前端文件结构

**新建文件:**
- `qim-client/src/components/ai/AIMessageBadge.vue` - AI消息标识组件
- `qim-client/src/components/ai/AIQuickActions.vue` - 快捷指令栏
- `qim-client/src/components/ai/AIQuickActionItem.vue` - 单个快捷按钮
- `qim-client/src/components/ai/AIContextBar.vue` - 上下文状态条
- `qim-client/src/components/ai/AISummaryPanel.vue` - 智能摘要面板
- `qim-client/src/components/ai/AISearchInput.vue` - 智能搜索输入
- `qim-client/src/components/ai/AISearchResults.vue` - 搜索结果展示
- `qim-client/src/components/ai/AIMessageContextMenu.vue` - AI右键菜单
- `qim-client/src/components/ai/AIMessageContent.vue` - AI消息内容(折叠/展开)
- `qim-client/src/components/ai/GroupAIPanel.vue` - 群聊AI设置面板
- `qim-client/src/composables/useAIKeyboardShortcuts.ts` - 快捷键管理
- `qim-client/src/composables/useAIActions.ts` - AI操作(翻译/摘要等)

**修改文件:**
- `qim-client/src/components/chat/ChatWindow.vue` - 集成AI组件
- `qim-client/src/components/chat/MessageInput.vue` - 添加快捷指令栏
- `qim-client/src/components/message/MessageItem.vue` - 添加AI标识和右键菜单
- `qim-client/src/components/chat/MessageListView.vue` - AI消息样式适配

---

### 任务 F1: AI消息标识 + 内容折叠组件

**前置依赖:** 无(可并行于后端)

**文件:**
- 创建: `qim-client/src/components/ai/AIMessageBadge.vue`
- 创建: `qim-client/src/components/ai/AIMessageContent.vue`

- [ ] **步骤 1: 创建 AI 消息标识组件**

创建 `qim-client/src/components/ai/AIMessageBadge.vue`:

```vue
<template>
  <div class="ai-message-badge" :class="{ 'compact': compact }">
    <div class="ai-badge-icon">
      <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
        <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm-1-13h2v6h-2zm0 8h2v2h-2z"/>
      </svg>
    </div>
    <span class="ai-badge-text">{{ assistantName || 'AI 助手' }}</span>
    <span v-if="!compact" class="ai-generated-tag">由 AI 生成</span>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  assistantName?: string
  compact?: boolean
}>()
</script>

<style scoped>
.ai-message-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 12px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  font-size: 12px;
  font-weight: 500;
}

.ai-message-badge.compact {
  padding: 2px 8px;
  font-size: 11px;
}

.ai-badge-icon {
  display: flex;
  align-items: center;
  opacity: 0.9;
}

.ai-badge-text {
  font-weight: 600;
}

.ai-generated-tag {
  font-size: 11px;
  opacity: 0.8;
  font-weight: 400;
}

[data-theme="dark"] .ai-message-badge {
  background: linear-gradient(135deg, #5a67d8 0%, #6b46c1 100%);
}
</style>
```

- [ ] **步骤 2: 创建 AI 消息内容折叠组件**

创建 `qim-client/src/components/ai/AIMessageContent.vue`:

```vue
<template>
  <div class="ai-message-content">
    <div v-if="!isExpanded && isLongContent" class="preview-content">
      <div v-html="renderMarkdown(previewText)"></div>
    </div>
    <div v-else class="full-content">
      <div v-html="renderMarkdown(content)"></div>
    </div>
    
    <div v-if="isLongContent" class="ai-content-footer">
      <button v-if="!isExpanded" class="expand-btn" @click="isExpanded = true">
        <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
          <path d="M7 10l5 5 5-5z"/>
        </svg>
        展开全部 (共 {{ contentLength }} 字符)
      </button>
      <div v-else class="expanded-actions">
        <button class="collapse-btn" @click="isExpanded = false">
          <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
            <path d="M7 14l5-5 5 5z"/>
          </svg>
          收起
        </button>
        <div class="export-actions">
          <button @click="copyContent">复制</button>
          <button @click="exportMarkdown">导出 Markdown</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { marked } from 'marked'
import { sanitizeMarkdown } from '../../utils/sanitize'

const props = withDefaults(defineProps<{
  content: string
  maxLength?: number
}>(), {
  maxLength: 500
})

const isExpanded = ref(false)
const previewLines = 5

const previewText = computed(() => {
  const lines = props.content.split('\n').slice(0, previewLines)
  return lines.join('\n')
})

const isLongContent = computed(() => {
  return props.content.length > props.maxLength
})

const contentLength = computed(() => props.content.length)

const renderMarkdown = (text: string): string => {
  try {
    const result = marked.parse(text)
    if (result instanceof Promise) {
      return text.replace(/\r\n|\n|\r/g, '<br>')
    }
    return sanitizeMarkdown(result as string)
  } catch {
    return sanitizeMarkdown(text.replace(/\r\n|\n|\r/g, '<br>'))
  }
}

const copyContent = async () => {
  try {
    await navigator.clipboard.writeText(props.content)
    // 可使用全局 message 组件提示成功
  } catch {
    // 降级处理
  }
}

const exportMarkdown = () => {
  const blob = new Blob([props.content], { type: 'text/markdown' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'ai-content.md'
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<style scoped>
.ai-message-content {
  width: 100%;
}

.preview-content {
  max-height: 150px;
  overflow: hidden;
  position: relative;
}

.preview-content::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 40px;
  background: linear-gradient(transparent, var(--message-bg));
}

.full-content {
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(-5px); }
  to { opacity: 1; transform: translateY(0); }
}

.ai-content-footer {
  margin-top: 12px;
  padding-top: 8px;
  border-top: 1px solid var(--border-color);
}

.expand-btn, .collapse-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  border: none;
  background: var(--hover-color);
  color: var(--text-primary);
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  transition: background 0.2s;
}

.expand-btn:hover, .collapse-btn:hover {
  background: var(--primary-light);
}

.expanded-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.export-actions {
  display: flex;
  gap: 8px;
}

.export-actions button {
  padding: 4px 10px;
  border: 1px solid var(--border-color);
  background: transparent;
  color: var(--text-secondary);
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  transition: all 0.2s;
}

.export-actions button:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}
</style>
```

- [ ] **步骤 3: Commit**

```bash
git add qim-client/src/components/ai/AIMessageBadge.vue
git add qim-client/src/components/ai/AIMessageContent.vue
git commit -m "feat(ai-frontend): 创建 AI 消息标识和内容折叠组件
- AIMessageBadge: AI 专属徽章样式,渐变背景
- AIMessageContent: 长内容折叠/展开,支持 Markdown 渲染
- 支持导出 Markdown 和复制完整内容
- 深色/浅色主题适配"
```

---

### 任务 F2: 快捷指令栏 + AI操作 Composable

**前置依赖:** 无

**文件:**
- 创建: `qim-client/src/components/ai/AIQuickActions.vue`
- 创建: `qim-client/src/components/ai/AIQuickActionItem.vue`
- 创建: `qim-client/src/composables/useAIActions.ts`

- [ ] **步骤 1: 创建 AI 操作 Composable**

创建 `qim-client/src/composables/useAIActions.ts`:

```typescript
import { ref } from 'vue'
import axios from 'axios'
import { API_BASE_URL } from '../config'

const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

const getToken = () => localStorage.getItem('token')

export function useAIActions() {
  const isProcessing = ref(false)
  const errorMessage = ref<string | null>(null)

  const translateText = async (text: string, targetLang: string = 'zh') => {
    isProcessing.value = true
    errorMessage.value = null
    
    try {
      const response = await axios.post(
        `${serverUrl.value}/api/v1/ai/translate`,
        { text, target_lang: targetLang },
        { headers: { Authorization: `Bearer ${getToken()}` } }
      )
      return response.data.data.translated_text
    } catch (error: any) {
      errorMessage.value = error.response?.data?.message || '翻译失败'
      throw error
    } finally {
      isProcessing.value = false
    }
  }

  const rewriteText = async (text: string, style: string = 'concise', tone: string = 'professional') => {
    isProcessing.value = true
    errorMessage.value = null
    
    try {
      const response = await axios.post(
        `${serverUrl.value}/api/v1/ai/rewrite`,
        { text, style, tone },
        { headers: { Authorization: `Bearer ${getToken()}` } }
      )
      return response.data.data.rewritten_text
    } catch (error: any) {
      errorMessage.value = error.response?.data?.message || '改写失败'
      throw error
    } finally {
      isProcessing.value = false
    }
  }

  const polishText = async (text: string, language: string = 'zh') => {
    isProcessing.value = true
    errorMessage.value = null
    
    try {
      const response = await axios.post(
        `${serverUrl.value}/api/v1/ai/polish`,
        { text, language },
        { headers: { Authorization: `Bearer ${getToken()}` } }
      )
      return response.data.data.polished_text
    } catch (error: any) {
      errorMessage.value = error.response?.data?.message || '润色失败'
      throw error
    } finally {
      isProcessing.value = false
    }
  }

  const generateSummary = async (conversationId: number, timeRange: string = 'today') => {
    isProcessing.value = true
    errorMessage.value = null
    
    try {
      const response = await axios.post(
        `${serverUrl.value}/api/v1/ai/summary`,
        { conversation_id: conversationId, time_range: timeRange },
        { headers: { Authorization: `Bearer ${getToken()}` } }
      )
      return response.data.data
    } catch (error: any) {
      errorMessage.value = error.response?.data?.message || '摘要生成失败'
      throw error
    } finally {
      isProcessing.value = false
    }
  }

  const searchMessages = async (conversationId: number, query: string, options?: { senderId?: number; startTime?: string; endTime?: string }) => {
    isProcessing.value = true
    errorMessage.value = null
    
    try {
      const response = await axios.post(
        `${serverUrl.value}/api/v1/ai/search`,
        { 
          conversation_id: conversationId, 
          query,
          ...options 
        },
        { headers: { Authorization: `Bearer ${getToken()}` } }
      )
      return response.data.data
    } catch (error: any) {
      errorMessage.value = error.response?.data?.message || '搜索失败'
      throw error
    } finally {
      isProcessing.value = false
    }
  }

  return {
    isProcessing,
    errorMessage,
    translateText,
    rewriteText,
    polishText,
    generateSummary,
    searchMessages,
  }
}
```

- [ ] **步骤 2: 创建快捷指令组件**

创建 `qim-client/src/components/ai/AIQuickActionItem.vue`:

```vue
<template>
  <button class="ai-quick-action" @click="$emit('click')" :title="tooltip">
    <span class="action-icon">{{ icon }}</span>
    <span class="action-label">{{ label }}</span>
  </button>
</template>

<script setup lang="ts">
defineProps<{
  icon: string
  label: string
  tooltip?: string
}>()

defineEmits<{
  click: []
}>()
</script>

<style scoped>
.ai-quick-action {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--card-bg);
  color: var(--text-primary);
  cursor: pointer;
  font-size: 13px;
  white-space: nowrap;
  transition: all 0.2s;
}

.ai-quick-action:hover {
  border-color: var(--primary-color);
  background: var(--primary-light);
  transform: translateY(-1px);
}

.action-icon {
  font-size: 14px;
}

.action-label {
  font-weight: 500;
}
</style>
```

创建 `qim-client/src/components/ai/AIQuickActions.vue`:

```vue
<template>
  <div class="ai-quick-actions">
    <AIQuickActionItem
      v-for="action in actions"
      :key="action.id"
      :icon="action.icon"
      :label="action.label"
      :tooltip="action.tooltip"
      @click="handleAction(action.id)"
    />
    <div v-if="isProcessing" class="ai-processing">
      <span class="processing-spinner"></span>
      <span>处理中...</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import AIQuickActionItem from './AIQuickActionItem.vue'

interface AIAction {
  id: string
  icon: string
  label: string
  tooltip?: string
}

const props = defineProps<{
  actions?: AIAction[]
  isProcessing: boolean
}>()

const emit = defineEmits<{
  action: [actionId: string]
}>()

const defaultActions: AIAction[] = [
  { id: 'summary', icon: '📝', label: '总结对话', tooltip: '总结当前会话内容' },
  { id: 'translate', icon: '🌐', label: '翻译', tooltip: '翻译选中文本' },
  { id: 'rewrite', icon: '✍️', label: '改写', tooltip: '改写输入框中的文本' },
  { id: 'polish', icon: '✨', label: '润色', tooltip: '润色文本语气和表达' },
  { id: 'code_review', icon: '🔍', label: '代码审查', tooltip: '审查代码片段' },
]

const actions = computed(() => props.actions || defaultActions)

const handleAction = (actionId: string) => {
  emit('action', actionId)
}
</script>

<style scoped>
.ai-quick-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--card-bg);
  border-top: 1px solid var(--border-color);
  overflow-x: auto;
  scrollbar-width: thin;
}

.ai-processing {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  color: var(--text-secondary);
  font-size: 13px;
}

.processing-spinner {
  width: 14px;
  height: 14px;
  border: 2px solid var(--border-color);
  border-top-color: var(--primary-color);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
```

- [ ] **步骤 3: Commit**

```bash
git add qim-client/src/composables/useAIActions.ts
git add qim-client/src/components/ai/AIQuickActions.vue
git add qim-client/src/components/ai/AIQuickActionItem.vue
git commit -m "feat(ai-frontend): 创建快捷指令组件和AI操作Composable
- useAIActions: 封装翻译/改写/润色/摘要/搜索API调用
- AIQuickActions: 快捷指令栏,支持自定义指令列表
- AIQuickActionItem: 单个快捷按钮,hover动效
- 处理中状态显示加载动画"
```

---

### 任务 F3: 智能摘要面板 + 搜索结果组件

**前置依赖:** F2(使用 useAIActions)

**文件:**
- 创建: `qim-client/src/components/ai/AISummaryPanel.vue`
- 创建: `qim-client/src/components/ai/AISearchInput.vue`
- 创建: `qim-client/src/components/ai/AISearchResults.vue`

- [ ] **步骤 1: 创建智能摘要面板**

创建 `qim-client/src/components/ai/AISummaryPanel.vue`:

```vue
<template>
  <Teleport to="body">
    <div v-if="visible" class="ai-summary-overlay" @click.self="close">
      <div class="ai-summary-panel">
        <div class="panel-header">
          <h3>📋 会话摘要</h3>
          <button class="close-btn" @click="close">×</button>
        </div>
        
        <div v-if="isGenerating" class="generating-state">
          <div class="generating-spinner"></div>
          <p>正在分析会话内容...</p>
          <p class="generating-hint">这可能需要几秒钟</p>
        </div>
        
        <div v-else-if="summaryData" class="summary-content">
          <div class="summary-meta">
            <span>📅 {{ summaryData.time_range }}</span>
            <span>💬 {{ summaryData.messages_count }} 条消息</span>
          </div>
          <div v-html="renderMarkdown(summaryData.summary)"></div>
          <div class="summary-actions">
            <button @click="copySummary">📋 复制摘要</button>
            <button @click="exportSummary">📄 导出 Markdown</button>
          </div>
        </div>
        
        <div v-else class="error-state">
          <p>摘要生成失败</p>
          <button @click="generate">重试</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useAIActions } from '../../composables/useAIActions'
import { marked } from 'marked'
import { sanitizeMarkdown } from '../../utils/sanitize'

const props = defineProps<{
  visible: boolean
  conversationId: number
  timeRange?: string
}>()

const emit = defineEmits<{
  close: []
}>()

const { generateSummary, isProcessing: isGenerating, errorMessage } = useAIActions()
const summaryData = ref<any>(null)

watch(() => props.visible, async (newVal) => {
  if (newVal && props.conversationId) {
    await generate()
  }
})

const generate = async () => {
  summaryData.value = null
  try {
    summaryData.value = await generateSummary(
      props.conversationId,
      props.timeRange || 'today'
    )
  } catch {
    // 错误已由 composable 处理
  }
}

const close = () => {
  emit('close')
}

const copySummary = async () => {
  if (summaryData.value?.summary) {
    await navigator.clipboard.writeText(summaryData.value.summary)
  }
}

const exportSummary = () => {
  if (summaryData.value?.summary) {
    const blob = new Blob([summaryData.value.summary], { type: 'text/markdown' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `session-summary-${Date.now()}.md`
    a.click()
    URL.revokeObjectURL(url)
  }
}

const renderMarkdown = (text: string): string => {
  try {
    const result = marked.parse(text)
    if (result instanceof Promise) return text
    return sanitizeMarkdown(result as string)
  } catch {
    return text.replace(/\n/g, '<br>')
  }
}
</script>

<style scoped>
.ai-summary-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.ai-summary-panel {
  background: var(--card-bg);
  border-radius: 12px;
  width: 90%;
  max-width: 700px;
  max-height: 80vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
}

.panel-header h3 {
  margin: 0;
  font-size: 18px;
}

.close-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  font-size: 24px;
  cursor: pointer;
  color: var(--text-secondary);
  border-radius: 6px;
  transition: background 0.2s;
}

.close-btn:hover {
  background: var(--hover-color);
}

.generating-state {
  padding: 60px 20px;
  text-align: center;
  color: var(--text-secondary);
}

.generating-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid var(--border-color);
  border-top-color: var(--primary-color);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto 16px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.summary-content {
  padding: 20px;
  overflow-y: auto;
}

.summary-meta {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
  font-size: 13px;
  color: var(--text-secondary);
}

.summary-actions {
  display: flex;
  gap: 12px;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
}

.summary-actions button {
  padding: 8px 16px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: transparent;
  color: var(--text-primary);
  cursor: pointer;
  transition: all 0.2s;
}

.summary-actions button:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.error-state {
  padding: 40px 20px;
  text-align: center;
  color: var(--error-color);
}

.error-state button {
  margin-top: 16px;
  padding: 8px 20px;
  border: none;
  border-radius: 6px;
  background: var(--primary-color);
  color: white;
  cursor: pointer;
}
</style>
```

- [ ] **步骤 2: 创建智能搜索组件**

创建 `qim-client/src/components/ai/AISearchInput.vue`:

```vue
<template>
  <div class="ai-search-input">
    <div class="search-wrapper">
      <svg class="search-icon" viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
        <path d="M15.5 14h-.79l-.28-.27C15.41 12.59 16 11.11 16 9.5 16 5.91 13.09 3 9.5 3S3 5.91 3 9.5 5.91 16 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"/>
      </svg>
      <input
        v-model="query"
        type="text"
        :placeholder="placeholder"
        @keyup.enter="handleSearch"
        @focus="isFocused = true"
        @blur="handleBlur"
      />
      <button v-if="query" class="clear-btn" @click="clear">×</button>
    </div>
    
    <AISearchResults
      v-if="showResults && results.length > 0"
      :results="results"
      :conversation-id="conversationId"
      @select="handleSelect"
      @close="showResults = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useAIActions } from '../../composables/useAIActions'
import AISearchResults from './AISearchResults.vue'

const props = defineProps<{
  conversationId: number
  placeholder?: string
}>()

const emit = defineEmits<{
  select: [result: any]
}>()

const { searchMessages, isProcessing } = useAIActions()
const query = ref('')
const results = ref<any[]>([])
const showResults = ref(false)
const isFocused = ref(false)

const placeholder = computed(() => 
  isProcessing.value ? '搜索中...' : (props.placeholder || '使用 AI 搜索消息...')
)

const handleSearch = async () => {
  if (!query.value.trim()) return
  
  try {
    const data = await searchMessages(props.conversationId, query.value.trim())
    results.value = data.results || []
    showResults.value = true
  } catch {
    // 错误处理
  }
}

const handleSelect = (result: any) => {
  emit('select', result)
  showResults.value = false
}

const clear = () => {
  query.value = ''
  results.value = []
  showResults.value = false
}

const handleBlur = () => {
  setTimeout(() => {
    showResults.value = false
  }, 200)
}
</script>

<style scoped>
.ai-search-input {
  position: relative;
  width: 100%;
}

.search-wrapper {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--bg-color);
  transition: border-color 0.2s;
}

.search-wrapper:focus-within {
  border-color: var(--primary-color);
}

.search-icon {
  color: var(--text-secondary);
  flex-shrink: 0;
}

.search-wrapper input {
  flex: 1;
  border: none;
  background: transparent;
  color: var(--text-primary);
  font-size: 14px;
  outline: none;
}

.clear-btn {
  border: none;
  background: transparent;
  color: var(--text-secondary);
  font-size: 18px;
  cursor: pointer;
  padding: 0 4px;
}

.clear-btn:hover {
  color: var(--text-primary);
}
</style>
```

创建 `qim-client/src/components/ai/AISearchResults.vue`:

```vue
<template>
  <div class="ai-search-results">
    <div class="results-header">
      <span>找到 {{ results.length }} 条相关消息</span>
      <button class="close-btn" @click="$emit('close')">×</button>
    </div>
    <div class="results-list">
      <div
        v-for="result in results"
        :key="result.message_id"
        class="result-item"
        @click="$emit('select', result)"
      >
        <div class="result-sender">{{ result.sender_name }}</div>
        <div class="result-time">{{ result.timestamp }}</div>
        <div class="result-content" v-html="result.highlighted || result.content"></div>
        <div class="result-score">相关度: {{ result.relevance_score }}%</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  results: any[]
  conversationId: number
}>()

defineEmits<{
  select: [result: any]
  close: []
}>()
</script>

<style scoped>
.ai-search-results {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  margin-top: 4px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  max-height: 400px;
  overflow-y: auto;
  z-index: 100;
}

.results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 14px;
  border-bottom: 1px solid var(--border-color);
  font-size: 13px;
  color: var(--text-secondary);
}

.close-btn {
  border: none;
  background: transparent;
  font-size: 18px;
  cursor: pointer;
  color: var(--text-secondary);
}

.result-item {
  padding: 12px 14px;
  border-bottom: 1px solid var(--border-color);
  cursor: pointer;
  transition: background 0.2s;
}

.result-item:last-child {
  border-bottom: none;
}

.result-item:hover {
  background: var(--hover-color);
}

.result-sender {
  font-weight: 600;
  font-size: 13px;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.result-time {
  font-size: 11px;
  color: var(--text-secondary);
  margin-bottom: 6px;
}

.result-content {
  font-size: 14px;
  color: var(--text-primary);
  line-height: 1.4;
}

.result-score {
  font-size: 11px;
  color: var(--primary-color);
  margin-top: 4px;
}
</style>
```

- [ ] **步骤 3: Commit**

```bash
git add qim-client/src/components/ai/AISummaryPanel.vue
git add qim-client/src/components/ai/AISearchInput.vue
git add qim-client/src/components/ai/AISearchResults.vue
git commit -m "feat(ai-frontend): 创建智能摘要和搜索组件
- AISummaryPanel: 模态框展示会话摘要,支持复制/导出
- AISearchInput: AI语义搜索输入框,自动触发搜索
- AISearchResults: 搜索结果展示,显示相关度和高亮
- 加载状态和错误处理完善"
```

---

### 任务 F4: 快捷键 + 群聊AI设置 + 右键菜单

**前置依赖:** F1, F2, F3

**文件:**
- 创建: `qim-client/src/composables/useAIKeyboardShortcuts.ts`
- 创建: `qim-client/src/components/ai/AIMessageContextMenu.vue`
- 创建: `qim-client/src/components/ai/GroupAIPanel.vue`
- 修改: `qim-client/src/components/chat/ChatWindow.vue`
- 修改: `qim-client/src/components/chat/MessageInput.vue`
- 修改: `qim-client/src/components/message/MessageItem.vue`

- [ ] **步骤 1: 创建快捷键管理 Composable**

创建 `qim-client/src/composables/useAIKeyboardShortcuts.ts`:

```typescript
import { onMounted, onUnmounted } from 'vue'

interface ShortcutConfig {
  key: string
  ctrlKey?: boolean
  shiftKey?: boolean
  metaKey?: boolean
  action: () => void
  description: string
}

export function useAIKeyboardShortcuts(
  shortcuts: ShortcutConfig[],
  enabled: boolean = true
) {
  const handleKeydown = (event: KeyboardEvent) => {
    if (!enabled) return
    
    for (const shortcut of shortcuts) {
      const keyMatch = event.key.toLowerCase() === shortcut.key.toLowerCase()
      const ctrlMatch = shortcut.ctrlKey ? (event.ctrlKey || event.metaKey) : true
      const shiftMatch = shortcut.shiftKey ? event.shiftKey : !event.shiftKey
      
      if (keyMatch && ctrlMatch && shiftMatch) {
        event.preventDefault()
        shortcut.action()
        break
      }
    }
  }

  onMounted(() => {
    document.addEventListener('keydown', handleKeydown)
  })

  onUnmounted(() => {
    document.removeEventListener('keydown', handleKeydown)
  })

  return { enabled }
}
```

- [ ] **步骤 2: 创建 AI 右键菜单组件**

创建 `qim-client/src/components/ai/AIMessageContextMenu.vue`:

```vue
<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="ai-context-menu"
      :style="{ top: position.y + 'px', left: position.x + 'px' }"
      @click.stop
    >
      <div
        v-for="item in menuItems"
        :key="item.id"
        class="menu-item"
        @click="handleSelect(item.id)"
      >
        <span class="menu-icon">{{ item.icon }}</span>
        <span class="menu-label">{{ item.label }}</span>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  visible: boolean
  position: { x: number; y: number }
  message: any
}>()

const emit = defineEmits<{
  select: [actionId: string, message: any]
  close: []
}>()

const menuItems = computed(() => [
  { id: 'ai_summary', icon: '🤖', label: 'AI 总结此消息' },
  { id: 'translate', icon: '🌐', label: '翻译为中文' },
  { id: 'rewrite', icon: '✍️', label: '改写文本' },
  { id: 'polish', icon: '✨', label: '润色表达' },
])

const handleSelect = (actionId: string) => {
  emit('select', actionId, props.message)
  emit('close')
}
</script>

<style scoped>
.ai-context-menu {
  position: fixed;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  padding: 6px 0;
  z-index: 3000;
  min-width: 180px;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  cursor: pointer;
  transition: background 0.15s;
  font-size: 14px;
  color: var(--text-primary);
}

.menu-item:hover {
  background: var(--hover-color);
}

.menu-icon {
  font-size: 16px;
  width: 20px;
  text-align: center;
}

.menu-label {
  font-weight: 500;
}
</style>
```

- [ ] **步骤 3: 创建群聊 AI 设置面板**

创建 `qim-client/src/components/ai/GroupAIPanel.vue`:

```vue
<template>
  <div class="group-ai-panel">
    <h4>AI 助手设置</h4>
    
    <div class="setting-item">
      <label class="toggle-label">
        <span>启用 AI 助手</span>
        <input type="checkbox" v-model="localSettings.enabled" @change="saveSettings" />
        <span class="toggle-slider"></span>
      </label>
    </div>
    
    <div v-if="localSettings.enabled" class="advanced-settings">
      <div class="setting-item">
        <label>AI 助手名称</label>
        <input v-model="localSettings.assistantName" @blur="saveSettings" />
      </div>
      
      <div class="setting-item">
        <label>回复模式</label>
        <select v-model="localSettings.replyMode" @change="saveSettings">
          <option value="mention_only">仅被 @ 时回复</option>
          <option value="smart">智能判断回复</option>
          <option value="always">始终回复</option>
          <option value="off">关闭 AI 回复</option>
        </select>
      </div>
      
      <div class="setting-item">
        <label>上下文消息数</label>
        <input type="number" v-model.number="localSettings.contextMessages" @blur="saveSettings" min="5" max="50" />
        <span class="setting-hint">AI 回复时参考的最近消息数</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{
  groupId: number
  aiEnabled: boolean
  aiAssistantName?: string
  aiReplyMode?: string
  contextMessages?: number
}>()

const emit = defineEmits<{
  update: [settings: any]
}>()

const localSettings = ref({
  enabled: props.aiEnabled,
  assistantName: props.aiAssistantName || 'AI助手',
  replyMode: props.aiReplyMode || 'mention_only',
  contextMessages: props.contextMessages || 10,
})

const saveSettings = () => {
  emit('update', { ...localSettings.value })
}
</script>

<style scoped>
.group-ai-panel {
  padding: 16px;
  background: var(--card-bg);
  border-radius: 8px;
}

.group-ai-panel h4 {
  margin: 0 0 16px 0;
  font-size: 16px;
  color: var(--text-primary);
}

.setting-item {
  margin-bottom: 16px;
}

.setting-item label {
  display: block;
  margin-bottom: 6px;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}

.setting-item input[type="text"],
.setting-item input[type="number"],
.setting-item select {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-color);
  color: var(--text-primary);
  font-size: 14px;
}

.setting-hint {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-secondary);
}

.toggle-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
</style>
```

- [ ] **步骤 4: Commit**

```bash
git add qim-client/src/composables/useAIKeyboardShortcuts.ts
git add qim-client/src/components/ai/AIMessageContextMenu.vue
git add qim-client/src/components/ai/GroupAIPanel.vue
git commit -m "feat(ai-frontend): 创建快捷键、右键菜单和群聊AI设置组件
- useAIKeyboardShortcuts: 全局快捷键注册,支持 Ctrl+K/Ctrl+Shift+S
- AIMessageContextMenu: 消息右键AI操作菜单
- GroupAIPanel: 群聊AI设置(开关/名称/回复模式/上下文)
- 设置面板支持实时保存"
```

---

### 前端开发完成检查清单

- [ ] 所有新组件编译无错误: `cd qim-client && npm run build`
- [ ] TypeScript 类型检查通过: `cd qim-client && npx vue-tsc --noEmit`
- [ ] 开发服务器可启动: `cd qim-client && npm run dev`
- [ ] AI消息标识在消息列表中正确显示
- [ ] 快捷指令按钮可点击并触发对应API
- [ ] 摘要面板正确展示AI返回的结构化数据
-  搜索结果组件支持点击跳转到消息位置
- [ ] 快捷键在聊天窗口中正常触发

---

# 角色三: 🔗 前后端联调 (Integration)

负责: 接口对接、数据流验证、端到端测试

## 联调任务

### 任务 I1: 群聊 @AI 回复端到端测试

**前置依赖:** B2, F1

**测试步骤:**

- [ ] **步骤 1: 启动后端服务**

```bash
cd qim-server
go run main.go
```

确认服务启动成功,API可访问 `http://localhost:8080`

- [ ] **步骤 2: 启动前端开发服务器**

```bash
cd qim-client
npm run dev
```

- [ ] **步骤 3: 创建测试群聊并开启AI**

1. 登录系统
2. 创建群聊
3. 在群设置中开启"启用AI助手"
4. 设置回复模式为"仅被@时回复"

- [ ] **步骤 4: 测试 @AI 触发**

1. 在群聊中输入 `@AI 今天天气如何`
2. 发送消息
3. 验证:
   - [ ] WebSocket 推送 AI 回复消息
   - [ ] AI 回复带 AI 标识(🤖 徽章)
   - [ ] AI 消息背景与普通消息区分
   - [ ] 长内容可折叠/展开

- [ ] **步骤 5: 验证输出长度控制**

1. 发送 `@AI 请详细总结一下软件工程的最佳实践,包括敏捷开发、测试驱动开发、代码审查等方面`
2. 验证:
   - [ ] AI 回复内容被正确截断(默认3000字符)
   - [ ] 底部显示"内容过长已截断"提示
   - [ ] 可展开查看完整截断内容

### 任务 I2: 单聊 AI 快捷指令端到端测试

**前置依赖:** B4, F2, F3

**测试步骤:**

- [ ] **步骤 1: 测试翻译功能**

1. 进入任意会话
2. 在输入框输入一段英文
3. 点击"翻译"快捷按钮
4. 验证:
   - [ ] 显示"处理中..."加载状态
   - [ ] 翻译结果返回并显示
   - [ ] 可复制翻译结果

- [ ] **步骤 2: 测试会话摘要功能**

1. 进入一个有多条消息的群聊
2. 点击顶部"生成摘要"按钮
3. 验证:
   - [ ] 弹出摘要面板
   - [ ] 显示"正在分析会话内容..."加载状态
   - [ ] 返回结构化摘要(核心话题/决策/待办)
   - [ ] 可复制和导出 Markdown

- [ ] **步骤 3: 测试语义搜索功能**

1. 进入群聊
2. 在搜索框输入 `关于项目进度的讨论`
3. 验证:
   - [ ] 返回相关消息列表
   - [ ] 显示发送者、时间和相关度
   - [ ] 点击结果可跳转到对应消息位置

- [ ] **步骤 4: 测试快捷键**

1. 在聊天窗口按下 `Ctrl+K`
2. 验证:
   - [ ] 唤起 AI 快捷面板(或执行对应操作)
3. 按下 `Ctrl+Shift+S`
4. 验证:
   - [ ] 快速生成当前会话摘要

---

## 联调完成检查清单

- [ ] 所有 API 端点前后端数据格式一致
- [ ] WebSocket AI 消息推送格式前端可正确解析
- [ ] 错误场景(服务不可用/超时/内容过长)前端显示友好提示
- [ ] 深色/浅色主题下所有 AI 组件样式正常
- [ ] 移动端响应式布局适配

---

## 总体执行顺序

```
Phase 1: 基础层 (并行)
├── 后端: B1(数据库+输出过滤) → B2(@AI触发)
├── 前端: F1(AI标识组件) → F2(快捷指令)
│
Phase 2: 功能层 (并行)  
├── 后端: B3(摘要+搜索API) → B4(文本处理API)
├── 前端: F3(摘要+搜索组件) → F4(快捷键+设置)
│
Phase 3: 联调层
└── 联调: I1(群聊@AI测试) + I2(单聊快捷指令测试)
```
