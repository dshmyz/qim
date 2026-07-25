package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"

	"github.com/gin-gonic/gin"
)

// AIResponse 标准AI响应结构体
type AIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func checkAIEnabledMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		configSvc := di.GlobalContainer.SystemConfigService
		publicConfigs, err := configSvc.GetPublicConfigs()
		if err == nil {
			if enableAI, ok := publicConfigs["enableAI"]; ok {
				if !enableAI.(bool) {
					response.Forbidden(c, "AI 功能已关闭")
					c.Abort()
					return
				}
			}
		}
		c.Next()
	}
}

// AIHandler AI处理器
type AIHandler struct {
	aiService          *ai.AIService
	avatarService      *service.AvatarService // 帮我回复草稿模式：复用分身生成
	mcpServer          *ai.MCPServer
	summaryGraph       *service.SummaryGraph
	textProcessGraph   *service.TextProcessGraph
	unifiedSearchGraph *service.UnifiedSearchGraph
	smartDigestGraph   *service.SmartDigestGraph
}

// NewAIHandler 创建AI处理器
func NewAIHandler(aiService *ai.AIService, mcpServer *ai.MCPServer) *AIHandler {
	return &AIHandler{
		aiService: aiService,
		mcpServer: mcpServer,
	}
}

// SetAvatarService 注入分身服务（帮我回复草稿模式复用分身生成）
func (h *AIHandler) SetAvatarService(avatarService *service.AvatarService) {
	h.avatarService = avatarService
}

func (h *AIHandler) SetSummaryGraph(graph *service.SummaryGraph) {
	h.summaryGraph = graph
}

func (h *AIHandler) SetTextProcessGraph(graph *service.TextProcessGraph) {
	h.textProcessGraph = graph
}

func (h *AIHandler) SetUnifiedSearchGraph(graph *service.UnifiedSearchGraph) {
	h.unifiedSearchGraph = graph
}

func (h *AIHandler) SetSmartDigestGraph(graph *service.SmartDigestGraph) {
	h.smartDigestGraph = graph
}

// RegisterRoutes 注册路由
func (h *AIHandler) RegisterRoutes(router *gin.RouterGroup) {
	aiGroup := router.Group("/ai")
	aiGroup.Use(checkAIEnabledMiddleware())
	{
		aiGroup.POST("/completion", h.GetCompletion)
		aiGroup.POST("/completion/stream", h.GetCompletionStream)
		aiGroup.POST("/draft-reply", h.DraftReply)
		aiGroup.POST("/draft-reply/stream", h.DraftReplyStream)
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

		// 新增: 图片翻译
		aiGroup.POST("/translate/image", h.TranslateImage)

		// 新增: 智能消息速览
		aiGroup.GET("/digest", h.GetDigest)

		// 运维相关路由(已有)
		aiGroup.POST("/ops/troubleshooting", h.IntelligentTroubleshooting)
		aiGroup.POST("/ops/command", h.CommandGeneration)
		aiGroup.POST("/ops/logs", h.LogAnalysis)
		aiGroup.POST("/ops/alert", h.IntelligentAlert)
		aiGroup.POST("/ops/knowledge", h.OpsKnowledge)
		aiGroup.GET("/ops/dashboard", h.OpsDashboard)
	}
}

// GetCompletionRequest 获取AI完成请求
type GetCompletionRequest struct {
	Messages []ai.Message `json:"messages" binding:"required"`
}

// GetCompletion 获取AI完成
// @Summary 获取AI完成
// @Description 根据消息获取AI完成
// @Tags AI
// @Accept json
// @Produce json
// @Param request body GetCompletionRequest true "AI完成请求"
// @Success 200 {object} AIResponse "成功响应"
// @Failure 400 {object} AIResponse "参数错误"
// @Failure 500 {object} AIResponse "服务器错误"
// @Router /api/ai/completion [post]
func (h *AIHandler) GetCompletion(c *gin.Context) {
	var req GetCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	// 检查AI服务是否配置
	if !h.aiService.IsConfigured() {
		response.InternalServerError(c, "AI服务未配置")
		return
	}

	// 获取AI完成
	result, err := h.aiService.GetCompletion(ai.TaskTypeChat, req.Messages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "AI请求失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// DraftReplyRequest 帮我回复请求
type DraftReplyRequest struct {
	ConversationID uint `json:"conversation_id" binding:"required"`
	MessageID      uint `json:"message_id" binding:"required"`
}

// DraftReply 根据对话上下文起草回复
// @Summary 帮我回复
// @Description 根据目标消息及上下文生成回复草稿
// @Tags AI
// @Accept json
// @Produce json
// @Param request body DraftReplyRequest true "起草请求"
// @Success 200 {object} AIResponse "成功响应"
// @Router /api/ai/draft-reply [post]
func (h *AIHandler) DraftReply(c *gin.Context) {
	var req DraftReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if !h.aiService.IsConfigured() {
		response.InternalServerError(c, "AI服务未配置")
		return
	}

	// 1) 优先复用分身生成（与流式路径一致：以用户身份 + persona/笔记/群知识/记忆/待办/历史，忽略 SkipReply）
	if h.avatarService != nil {
		userIDAny, _ := c.Get("user_id")
		userID, _ := userIDAny.(uint)
		var avatarCfg model.AvatarConfig
		if userID > 0 && database.GetDB().Where("user_id = ?", userID).First(&avatarCfg).Error == nil {
			target, err := loadDraftTarget(req)
			if err != nil {
				response.BadRequest(c, err.Error())
				return
			}
			stream, err := h.avatarService.GenerateReplyStream(c.Request.Context(), userID, req.ConversationID, target.Content, &avatarCfg)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "生成回复失败: " + err.Error()})
				return
			}
			defer stream.Close()
			var reply strings.Builder
			for {
				msg, recvErr := stream.Recv()
				if recvErr != nil {
					if !errors.Is(recvErr, io.EOF) {
						c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "生成回复失败: " + recvErr.Error()})
						return
					}
					break
				}
				if msg != nil {
					reply.WriteString(msg.Content)
				}
			}
			c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"reply": reply.String()}})
			return
		}
		// 无分身配置 -> 回退简单起草
	}

	// 2) 回退：中立起草 prompt + 10 条上下文
	messages, err := buildDraftReplyMessages(req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.aiService.GetCompletion(ai.TaskTypeChat, messages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "AI请求失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    gin.H{"reply": result},
	})
}

// buildDraftReplyMessages 构造帮我回复的上下文消息（供同步/流式复用）
// loadDraftTarget 查询目标消息并校验其属于指定会话（同步/流式起草共用，
// 避免分身路径漏校验导致跨会话越权读取）。
func loadDraftTarget(req DraftReplyRequest) (*model.Message, error) {
	var target model.Message
	if err := database.GetDB().First(&target, req.MessageID).Error; err != nil {
		return nil, fmt.Errorf("消息不存在")
	}
	if target.ConversationID != req.ConversationID {
		return nil, fmt.Errorf("消息与会话不匹配")
	}
	return &target, nil
}

func buildDraftReplyMessages(req DraftReplyRequest) ([]ai.Message, error) {
	db := database.GetDB()

	target, err := loadDraftTarget(req)
	if err != nil {
		return nil, err
	}

	// 取目标消息之前的 10 条作为上下文（含发送者昵称）
	var contextMsgs []model.Message
	db.Preload("Sender").
		Where("conversation_id = ? AND created_at < ? AND type != ? AND is_recalled = ?",
			req.ConversationID, target.CreatedAt, "system", false).
		Order("created_at DESC").
		Limit(10).
		Find(&contextMsgs)

	// 按时间正序拼装上下文文本
	contextLines := []string{}
	for i := len(contextMsgs) - 1; i >= 0; i-- {
		m := contextMsgs[i]
		name := m.Sender.Nickname
		if name == "" {
			name = m.Sender.Username
		}
		contextLines = append(contextLines, fmt.Sprintf("%s: %s", name, m.Content))
	}
	contextText := ""
	if len(contextLines) > 0 {
		contextText = "对话上下文：\n" + fmt.Sprintf("%s\n\n", strings.Join(contextLines, "\n"))
	}

	targetName := target.Sender.Nickname
	if targetName == "" {
		targetName = "对方"
	}

	userPrompt := fmt.Sprintf("%s需要回复的消息（来自 %s）：\n%s",
		contextText, targetName, target.Content)

	return []ai.Message{
		{Role: "system", Content: "你是用户的起草助手。根据下面的对话上下文，帮用户起草一条回复最后一条消息的内容。语气自然、简短，直接返回回复内容，不要加任何前缀、引号或解释。"},
		{Role: "user", Content: userPrompt},
	}, nil
}

// DraftReplyStream 流式起草回复
// @Summary 流式帮我回复
// @Description 根据目标消息及上下文流式生成回复草稿，使用 SSE 返回
// @Tags AI
// @Accept json
// @Produce text/event-stream
// @Param request body DraftReplyRequest true "起草请求"
// @Success 200 {string} string "流式输出"
// @Router /api/ai/draft-reply/stream [post]
func (h *AIHandler) DraftReplyStream(c *gin.Context) {
	var req DraftReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if !h.aiService.IsConfigured() {
		response.InternalServerError(c, "AI服务未配置")
		return
	}

	// 1) 草稿模式优先复用分身生成：以用户身份 + persona/笔记/群知识/记忆/待办/历史
	if h.avatarService != nil {
		userIDAny, _ := c.Get("user_id")
		userID, _ := userIDAny.(uint)
		var avatarCfg model.AvatarConfig
		if userID > 0 && database.GetDB().Where("user_id = ?", userID).First(&avatarCfg).Error == nil {
			target, err := loadDraftTarget(req)
			if err != nil {
				response.BadRequest(c, err.Error())
				return
			}
			stream, err := h.avatarService.GenerateReplyStream(c.Request.Context(), userID, req.ConversationID, target.Content, &avatarCfg)
			if err != nil {
				response.InternalServerError(c, "生成回复失败: "+err.Error())
				return
			}
			streamCompletionFromReader(c, stream)
			return
		}
		// 无分身配置 -> 回退简单起草
	}

	// 2) 回退：中立起草 prompt + 10 条上下文
	messages, err := buildDraftReplyMessages(req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	h.streamCompletion(c, messages)
}

// streamCompletion 将一组消息以 SSE 流式推给前端（GetCompletionStream / DraftReplyStream 共用）
// streamSSE 设置 SSE 响应头，pump 通过 writeChunk 推送内容块；结束后发 finish 事件。
// pump 返回非 nil error 时改为推送错误事件。供 streamCompletion / streamCompletionFromReader 共用，
// 避免 SSE 响应头与结束事件逻辑重复。
func streamSSE(c *gin.Context, pump func(writeChunk func(content string)) error) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	writeChunk := func(content string) {
		data, _ := json.Marshal(ai.StreamChunk{Content: content})
		c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
		c.Writer.Flush()
	}

	if err := pump(writeChunk); err != nil {
		errStr := "AI请求失败: " + err.Error()
		errData, _ := json.Marshal(ai.StreamChunk{Error: &errStr})
		c.Writer.Write([]byte("data: " + string(errData) + "\n\n"))
		c.Writer.Flush()
		return
	}

	// 发送结束事件
	finish := "stop"
	doneData, _ := json.Marshal(ai.StreamChunk{Finish: &finish})
	c.Writer.Write([]byte("data: " + string(doneData) + "\n\n"))
	c.Writer.Flush()
}

// streamCompletion 流式推送一组 messages（经 aiService.GetCompletionStream）
func (h *AIHandler) streamCompletion(c *gin.Context, messages []ai.Message) {
	streamSSE(c, func(writeChunk func(string)) error {
		return h.aiService.GetCompletionStream(ai.TaskTypeChat, messages, func(chunk ai.StreamChunk) error {
			if chunk.Content != "" {
				writeChunk(chunk.Content)
			}
			return nil
		})
	})
}

// streamCompletionFromReader 把 Eino StreamReader 逐块以 SSE 推给前端（分身草稿流式路径）
// streamCompletionFromReader 把 Eino StreamReader 逐块以 SSE 推给前端（分身草稿流式路径）。
// Recv 返回 io.EOF 表示正常结束；其他 err 为上游真实错误，交 streamSSE 推送错误事件。
// defer Close 确保客户端断开或出错时释放 reader 端资源。
func streamCompletionFromReader(c *gin.Context, stream *schema.StreamReader[*schema.Message]) {
	defer stream.Close()
	streamSSE(c, func(writeChunk func(string)) error {
		for {
			msg, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			}
			if msg != nil && msg.Content != "" {
				writeChunk(msg.Content)
			}
		}
	})
}

// GetDigest 获取智能消息速览
func (h *AIHandler) GetDigest(c *gin.Context) {
	conversationIDStr := c.Query("conversation_id")
	conversationID, _ := strconv.ParseUint(conversationIDStr, 10, 32)

	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	if h.smartDigestGraph == nil {
		response.BadRequest(c, "Digest 功能未启用")
		return
	}

	input := &service.DigestInput{
		UserID:         userID.(uint),
		ConversationID: uint(conversationID),
	}

	result, err := h.smartDigestGraph.Execute(c.Request.Context(), input)
	if err != nil {
		response.InternalServerError(c, "生成摘要失败")
		return
	}

	response.Success(c, result)
}

// KnowledgeGraphQueryRequest 知识图谱查询请求
type KnowledgeGraphQueryRequest struct {
	Collection string `json:"collection" binding:"required"`
	Query      string `json:"query"`
	MaxNodes   int    `json:"max_nodes"`
}

// GetKnowledgeGraph 获取知识图谱数据
// @Summary 获取知识图谱数据
// @Description 获取指定集合的知识图谱节点和关系数据
// @Tags 知识图谱
// @Accept json
// @Produce json
// @Param collection query string true "集合名称"
// @Param query query string false "搜索查询"
// @Param max_nodes query int false "最大节点数"
// @Success 200 {object} AIResponse "成功响应"
// @Failure 400 {object} AIResponse "参数错误"
// @Failure 500 {object} AIResponse "服务器错误"
// @Router /api/admin/knowledge-graph [get]
func (h *AIHandler) GetKnowledgeGraph(c *gin.Context) {
	collection := c.Query("collection")
	if collection == "" {
		response.BadRequest(c, "集合名称不能为空")
		return
	}

	query := c.Query("query")
	maxNodes := 50
	if maxNodesStr := c.Query("max_nodes"); maxNodesStr != "" {
		fmt.Sscanf(maxNodesStr, "%d", &maxNodes)
	}

	nodes := make([]map[string]interface{}, 0)
	edges := make([]map[string]interface{}, 0)

	vectorSvc := di.GlobalContainer.VectorService
	if vectorSvc == nil {
		response.Success(c, gin.H{
			"nodes":       nodes,
			"edges":       edges,
			"total_nodes": 0,
			"total_edges": 0,
		})
		return
	}

	ctx := context.Background()

	searchResults, err := vectorSvc.GetByCollection(ctx, collection, maxNodes)

	if err == nil && len(searchResults) > 0 {
		for i, result := range searchResults {
			nodeID := fmt.Sprintf("node_%d", i)
			nodes = append(nodes, map[string]interface{}{
				"id":    nodeID,
				"label": result.DocID,
				"type":  "knowledge",
				"x":     float64(i%10) * 100,
				"y":     float64(i/10) * 100,
				"data": map[string]interface{}{
					"content":    result.Content,
					"score":      result.Score,
					"metadata":   result.Metadata,
					"collection": result.Collection,
				},
			})
		}
	}

	if query != "" && len(nodes) > 0 {
		queryNodeID := "query_node"
		nodes = append(nodes, map[string]interface{}{
			"id":    queryNodeID,
			"label": fmt.Sprintf("搜索: %s", query),
			"type":  "query",
			"x":     500,
			"y":     300,
			"data": map[string]interface{}{
				"query": query,
			},
		})

		for _, node := range nodes[:len(nodes)-1] {
			edges = append(edges, map[string]interface{}{
				"source": queryNodeID,
				"target": node["id"],
				"label":  "related",
				"type":   "search_relation",
			})
		}
	}

	response.Success(c, gin.H{
		"nodes":       nodes,
		"edges":       edges,
		"total_nodes": len(nodes),
		"total_edges": len(edges),
	})
}

// MCPToolConfigRequest MCP工具配置请求
type MCPToolConfigRequest struct {
	Enabled *bool `json:"enabled" binding:"required"`
}

// ListMCPTools 列出所有MCP工具（包含启用状态）
// @Summary 列出所有MCP工具（管理后台）
// @Description 列出所有MCP工具及其启用状态，用于管理后台配置
// @Tags MCP工具管理
// @Produce json
// @Success 200 {object} AIResponse "成功响应"
// @Router /api/admin/mcp/tools [get]
func (h *AIHandler) ListMCPTools(c *gin.Context) {
	if h.mcpServer == nil {
		response.InternalServerError(c, "MCP服务器未初始化")
		return
	}

	tools := h.mcpServer.ListTools()
	response.Success(c, gin.H{
		"tools": tools,
		"total": len(tools),
	})
}

// UpdateMCPToolConfig 更新MCP工具配置
// @Summary 更新MCP工具配置
// @Description 启用或禁用指定的MCP工具
// @Tags MCP工具管理
// @Accept json
// @Produce json
// @Param tool_name path string true "工具名称"
// @Param request body MCPToolConfigRequest true "工具配置请求"
// @Success 200 {object} AIResponse "成功响应"
// @Failure 400 {object} AIResponse "参数错误"
// @Failure 404 {object} AIResponse "工具不存在"
// @Router /api/admin/mcp/tools/{tool_name} [put]
func (h *AIHandler) UpdateMCPToolConfig(c *gin.Context) {
	if h.mcpServer == nil {
		response.InternalServerError(c, "MCP服务器未初始化")
		return
	}

	toolName := c.Param("tool_name")
	var req MCPToolConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	var err error
	if req.Enabled != nil && *req.Enabled {
		err = h.mcpServer.EnableTool(toolName)
	} else {
		err = h.mcpServer.DisableTool(toolName)
	}

	if err != nil {
		response.NotFound(c, "工具不存在或更新失败")
		return
	}

	response.Success(c, gin.H{
		"tool_name": toolName,
		"enabled":   req.Enabled != nil && *req.Enabled,
	})
}

// OpsDashboard 运维面板数据
// @Summary 运维面板数据
// @Description 获取AI运维面板的统计数据
// @Tags AI
// @Produce json
// @Success 200 {object} AIResponse "成功响应"
// @Router /api/ai/ops/dashboard [get]
func (h *AIHandler) OpsDashboard(c *gin.Context) {
	aiConfigured := h.aiService.IsConfigured()

	// 返回运维面板数据
	dashboard := gin.H{
		"ai_configured": aiConfigured,
		"provider":      "",
		"tools":         []gin.H{},
		"stats": gin.H{
			"total_bots":     0,
			"active_bots":    0,
			"total_messages": 0,
			"ai_messages":    0,
		},
	}

	if aiConfigured {
		cfg := h.aiService.GetConfig()
		dashboard["providers_configured"] = len(cfg.AllProviders())
	}

	if h.mcpServer != nil {
		tools := h.mcpServer.ListTools()
		dashboard["tools"] = tools
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    dashboard,
	})
}

// GetCompletionStream 流式获取AI完成
// @Summary 流式获取AI完成
// @Description 流式获取AI完成，使用服务器发送事件(SSE)返回统一JSON格式
// @Tags AI
// @Accept json
// @Produce text/event-stream
// @Param request body GetCompletionRequest true "AI完成请求"
// @Success 200 {string} string "流式输出"
// @Failure 400 {object} AIResponse "参数错误"
// @Failure 500 {object} AIResponse "服务器错误"
// @Router /api/ai/completion/stream [post]
func (h *AIHandler) GetCompletionStream(c *gin.Context) {
	var req GetCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	// 检查AI服务是否配置
	if !h.aiService.IsConfigured() {
		response.InternalServerError(c, "AI服务未配置")
		return
	}

	h.streamCompletion(c, req.Messages)
}

// ListTools 列出所有MCP工具
// @Summary 列出所有MCP工具
// @Description 列出所有可用的MCP工具
// @Tags AI
// @Produce json
// @Success 200 {object} AIResponse "成功响应"
// @Router /api/ai/tools [get]
func (h *AIHandler) ListTools(c *gin.Context) {
	if h.mcpServer == nil {
		response.InternalServerError(c, "MCP服务器未初始化")
		return
	}

	tools := h.mcpServer.ListTools()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    tools,
	})
}

// ExecuteToolRequest 执行工具请求
type ExecuteToolRequest struct {
	ToolName   string                 `json:"tool_name" binding:"required"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ExecuteTool 执行MCP工具
// @Summary 执行MCP工具
// @Description 执行指定的MCP工具
// @Tags AI
// @Accept json
// @Produce json
// @Param request body ExecuteToolRequest true "执行工具请求"
// @Success 200 {object} AIResponse "成功响应"
// @Failure 400 {object} AIResponse "参数错误"
// @Failure 500 {object} AIResponse "服务器错误"
// @Router /api/ai/tools/execute [post]
func (h *AIHandler) ExecuteTool(c *gin.Context) {
	var req ExecuteToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if h.mcpServer == nil {
		response.InternalServerError(c, "MCP服务器未初始化")
		return
	}

	// 执行工具
	result, err := h.mcpServer.ExecuteTool(req.ToolName, req.Parameters, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "工具执行失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// IntelligentTroubleshootingRequest 智能故障排查请求
type IntelligentTroubleshootingRequest struct {
	Symptom string `json:"symptom" binding:"required"`
	Server  string `json:"server"`
	Logs    string `json:"logs"`
}

// IntelligentTroubleshooting 智能故障排查
// @Summary 智能故障排查
// @Description 分析服务器故障并提供解决方案
// @Tags AI
// @Accept json
// @Produce json
// @Param request body IntelligentTroubleshootingRequest true "智能故障排查请求"
// @Success 200 {object} AIResponse "成功响应"
// @Failure 400 {object} AIResponse "参数错误"
// @Failure 500 {object} AIResponse "服务器错误"
// @Router /api/ai/ops/troubleshooting [post]
func (h *AIHandler) IntelligentTroubleshooting(c *gin.Context) {
	var req IntelligentTroubleshootingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	tool := ai.NewIntelligentTroubleshootingTool(h.aiService)
	params := map[string]interface{}{
		"symptom": req.Symptom,
		"server":  req.Server,
		"logs":    req.Logs,
	}

	result, err := tool.Execute(params, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "故障排查失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// CommandGenerationRequest 命令生成请求
type CommandGenerationRequest struct {
	Description string `json:"description" binding:"required"`
	Platform    string `json:"platform"`
	Format      string `json:"format"`
}

// CommandGeneration 命令生成
// @Summary 命令生成
// @Description 根据描述生成运维命令
// @Tags AI
// @Accept json
// @Produce json
// @Param request body CommandGenerationRequest true "命令生成请求"
// @Success 200 {object} AIResponse "成功响应"
// @Failure 400 {object} AIResponse "参数错误"
// @Failure 500 {object} AIResponse "服务器错误"
// @Router /api/ai/ops/command [post]
func (h *AIHandler) CommandGeneration(c *gin.Context) {
	var req CommandGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	tool := ai.NewCommandGenerationTool(h.aiService)
	params := map[string]interface{}{
		"description": req.Description,
		"platform":    req.Platform,
		"format":      req.Format,
	}

	result, err := tool.Execute(params, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "命令生成失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// LogAnalysisRequest 日志分析请求
type LogAnalysisRequest struct {
	LogContent string `json:"log_content" binding:"required"`
	Service    string `json:"service"`
	Severity   string `json:"severity"`
}

// LogAnalysis 日志分析
// @Summary 日志分析
// @Description 分析服务器日志并提供建议
// @Tags AI
// @Accept json
// @Produce json
// @Param request body LogAnalysisRequest true "日志分析请求"
// @Success 200 {object} AIResponse "成功响应"
// @Failure 400 {object} AIResponse "参数错误"
// @Failure 500 {object} AIResponse "服务器错误"
// @Router /api/ai/ops/logs [post]
func (h *AIHandler) LogAnalysis(c *gin.Context) {
	var req LogAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	tool := ai.NewLogAnalysisTool(h.aiService)
	params := map[string]interface{}{
		"log_content": req.LogContent,
		"service":     req.Service,
		"severity":    req.Severity,
	}

	result, err := tool.Execute(params, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "日志分析失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// IntelligentAlertRequest 智能告警处理请求
type IntelligentAlertRequest struct {
	AlertContent string `json:"alert_content" binding:"required"`
	Severity     string `json:"severity"`
	Service      string `json:"service"`
}

// IntelligentAlert 智能告警处理
// @Summary 智能告警处理
// @Description 分析和处理告警
// @Tags AI
// @Accept json
// @Produce json
// @Param request body IntelligentAlertRequest true "智能告警处理请求"
// @Success 200 {object} AIResponse "成功响应"
// @Failure 400 {object} AIResponse "参数错误"
// @Failure 500 {object} AIResponse "服务器错误"
// @Router /api/ai/ops/alert [post]
func (h *AIHandler) IntelligentAlert(c *gin.Context) {
	var req IntelligentAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	tool := ai.NewIntelligentAlertTool(h.aiService)
	params := map[string]interface{}{
		"alert_content": req.AlertContent,
		"severity":      req.Severity,
		"service":       req.Service,
	}

	result, err := tool.Execute(params, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "告警处理失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// OpsKnowledgeRequest 运维知识问答请求
type OpsKnowledgeRequest struct {
	Question string `json:"question" binding:"required"`
	Category string `json:"category"`
}

// OpsKnowledge 运维知识问答
// @Summary 运维知识问答
// @Description 回答运维相关问题
// @Tags AI
// @Accept json
// @Produce json
// @Param request body OpsKnowledgeRequest true "运维知识问答请求"
// @Success 200 {object} AIResponse "成功响应"
// @Failure 400 {object} AIResponse "参数错误"
// @Failure 500 {object} AIResponse "服务器错误"
// @Router /api/ai/ops/knowledge [post]
func (h *AIHandler) OpsKnowledge(c *gin.Context) {
	var req OpsKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	tool := ai.NewOpsKnowledgeTool(h.aiService)
	params := map[string]interface{}{
		"question": req.Question,
		"category": req.Category,
	}

	result, err := tool.Execute(params, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "知识问答失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}
