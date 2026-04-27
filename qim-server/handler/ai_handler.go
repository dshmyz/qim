package handler

import (
	"encoding/json"
	"net/http"
	"qim-server/ai"

	"github.com/gin-gonic/gin"
)

// AIHandler AI处理器
type AIHandler struct {
	aiService *ai.AIService
	mcpServer *ai.MCPServer
}

// NewAIHandler 创建AI处理器
func NewAIHandler(aiService *ai.AIService, mcpServer *ai.MCPServer) *AIHandler {
	return &AIHandler{
		aiService: aiService,
		mcpServer: mcpServer,
	}
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
// @Success 200 {object} gin.H{"code": int, "message": string, "data": string}
// @Failure 400 {object} gin.H{"code": int, "message": string}
// @Failure 500 {object} gin.H{"code": int, "message": string}
// @Router /api/ai/completion [post]
func (h *AIHandler) GetCompletion(c *gin.Context) {
	var req GetCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 检查AI服务是否配置
	if !h.aiService.IsConfigured() {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "AI服务未配置"})
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

// GenerateSummary 生成会话摘要 (B3 待实现)
func (h *AIHandler) GenerateSummary(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"code": 501, "message": "会话摘要功能待实现"})
}

// AISearch 语义搜索消息 (B3 待实现)
func (h *AIHandler) AISearch(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"code": 501, "message": "语义搜索功能待实现"})
}

// OpsDashboard 运维面板数据
// @Summary 运维面板数据
// @Description 获取AI运维面板的统计数据
// @Tags AI
// @Produce json
// @Success 200 {object} gin.H{"code": int, "message": string, "data": interface{}}
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
// @Failure 400 {object} gin.H{"code": int, "message": string}
// @Failure 500 {object} gin.H{"code": int, "message": string}
// @Router /api/ai/completion/stream [post]
func (h *AIHandler) GetCompletionStream(c *gin.Context) {
	var req GetCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 检查AI服务是否配置
	if !h.aiService.IsConfigured() {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "AI服务未配置"})
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
// @Success 200 {object} gin.H{"code": int, "message": string, "data": []map[string]interface{}}
// @Router /api/ai/tools [get]
func (h *AIHandler) ListTools(c *gin.Context) {
	if h.mcpServer == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "MCP服务器未初始化"})
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
// @Success 200 {object} gin.H{"code": int, "message": string, "data": interface{}}
// @Failure 400 {object} gin.H{"code": int, "message": string}
// @Failure 500 {object} gin.H{"code": int, "message": string}
// @Router /api/ai/tools/execute [post]
func (h *AIHandler) ExecuteTool(c *gin.Context) {
	var req ExecuteToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if h.mcpServer == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "MCP服务器未初始化"})
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
// @Success 200 {object} gin.H{"code": int, "message": string, "data": interface{}}
// @Failure 400 {object} gin.H{"code": int, "message": string}
// @Failure 500 {object} gin.H{"code": int, "message": string}
// @Router /api/ai/ops/troubleshooting [post]
func (h *AIHandler) IntelligentTroubleshooting(c *gin.Context) {
	var req IntelligentTroubleshootingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	tool := &ai.IntelligentTroubleshootingTool{}
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
// @Success 200 {object} gin.H{"code": int, "message": string, "data": interface{}}
// @Failure 400 {object} gin.H{"code": int, "message": string}
// @Failure 500 {object} gin.H{"code": int, "message": string}
// @Router /api/ai/ops/command [post]
func (h *AIHandler) CommandGeneration(c *gin.Context) {
	var req CommandGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	tool := &ai.CommandGenerationTool{}
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
// @Success 200 {object} gin.H{"code": int, "message": string, "data": interface{}}
// @Failure 400 {object} gin.H{"code": int, "message": string}
// @Failure 500 {object} gin.H{"code": int, "message": string}
// @Router /api/ai/ops/logs [post]
func (h *AIHandler) LogAnalysis(c *gin.Context) {
	var req LogAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	tool := &ai.LogAnalysisTool{}
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
// @Success 200 {object} gin.H{"code": int, "message": string, "data": interface{}}
// @Failure 400 {object} gin.H{"code": int, "message": string}
// @Failure 500 {object} gin.H{"code": int, "message": string}
// @Router /api/ai/ops/alert [post]
func (h *AIHandler) IntelligentAlert(c *gin.Context) {
	var req IntelligentAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	tool := &ai.IntelligentAlertTool{}
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
// @Success 200 {object} gin.H{"code": int, "message": string, "data": interface{}}
// @Failure 400 {object} gin.H{"code": int, "message": string}
// @Failure 500 {object} gin.H{"code": int, "message": string}
// @Router /api/ai/ops/knowledge [post]
func (h *AIHandler) OpsKnowledge(c *gin.Context) {
	var req OpsKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	tool := &ai.OpsKnowledgeTool{}
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
