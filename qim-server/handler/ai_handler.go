package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"qim-server/ai"
	"qim-server/di"
	"qim-server/pkg/response"
	"qim-server/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/cortexdb/v2/pkg/core"
)

// AIResponse 标准AI响应结构体
type AIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// AIHandler AI处理器
type AIHandler struct {
	aiService          *ai.AIService
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
	result, err := h.aiService.GetCompletion(req.Messages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "AI请求失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    result,
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

	db := vectorSvc.GetDB()
	if db == nil {
		response.Success(c, gin.H{
			"nodes":       nodes,
			"edges":       edges,
			"total_nodes": 0,
			"total_edges": 0,
		})
		return
	}

	ctx := context.Background()

	searchResults, err := db.Vector().Search(ctx, nil, core.SearchOptions{
		Collection: collection,
		TopK:       maxNodes,
	})

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
		dashboard["provider"] = cfg.Provider
	}

	if h.mcpServer != nil {
		tools := h.mcpServer.ListTools()
		dashboard["tools"] = tools
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
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

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 定义onChunk函数，将 StreamChunk JSON 编码后通过 SSE 发送
	onChunk := func(chunk ai.StreamChunk) error {
		data, err := json.Marshal(chunk)
		if err != nil {
			return err
		}
		c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
		c.Writer.Flush()
		return nil
	}

	// 执行流式请求
	err := h.aiService.GetCompletionStream(req.Messages, onChunk)
	if err != nil {
		// 发送错误事件
		errData, _ := json.Marshal(ai.StreamChunk{Content: err.Error()})
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
		"code":    200,
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
		"code":    200,
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

	result, err := tool.Execute(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "故障排查失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
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

	result, err := tool.Execute(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "命令生成失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
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

	result, err := tool.Execute(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "日志分析失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
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

	result, err := tool.Execute(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "告警处理失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
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

	result, err := tool.Execute(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "知识问答失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    result,
	})
}
