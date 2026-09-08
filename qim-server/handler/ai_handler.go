package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/aiprompt"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/productname"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AIResponse 标准AI响应结构体
type AIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

var errConversationAccessDenied = errors.New("conversation access denied")

// errNoVisionRouteMsg 未配置视觉路由时对带图请求的诚实提示。流式/非流式端点共用同一文案。
const errNoVisionRouteMsg = "当前未配置「视觉理解」任务路由（管理后台 → AI 模型配置 → 模型路由），无法识别图片"

func ensureConversationAccess(db *gorm.DB, conversationID, userID uint) error {
	if conversationID == 0 || userID == 0 {
		return errConversationAccessDenied
	}
	var count int64
	if err := db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errConversationAccessDenied
	}
	return nil
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

// AIHandlerDeps AIHandler 的全部可选依赖。任一依赖为 nil 时对应能力降级
// （如 SummaryGraph 为 nil 则摘要走旧路径），与原逐个 Set* 注入的语义一致。
type AIHandlerDeps struct {
	AIService          *ai.AIService
	ToolRegistry       *ai.ToolRegistry
	AvatarService      *service.AvatarService     // 帮我回复草稿模式：复用分身生成
	SummaryGraph       *service.SummaryGraph
	TextProcessGraph   *service.TextProcessGraph
	UnifiedSearchGraph *service.UnifiedSearchGraph
	SmartDigestGraph   *service.SmartDigestGraph
	ContextAssembler   *service.ContextAssembler  // 上下文预制（侧边栏 current 模式历史注入）；nil=跳过
	PendingActions     *service.AIPendingActionService // send_message 等敏感工具的待确认执行
	ToolScopes         *service.ToolScopeService       // 工具面配置服务；nil=用代码默认白名单
	Feedback           *service.AIFeedbackService      // AI 回复用户反馈（👍/👎）
	ConfigSvc          *service.SystemConfigService    // system_configs 读写（推荐提示词等通用配置键）
}

// AIHandler AI处理器
type AIHandler struct {
	aiService          *ai.AIService
	avatarService      *service.AvatarService // 帮我回复草稿模式：复用分身生成
	toolRegistry       *ai.ToolRegistry
	summaryGraph       *service.SummaryGraph
	textProcessGraph   *service.TextProcessGraph
	unifiedSearchGraph *service.UnifiedSearchGraph
	smartDigestGraph   *service.SmartDigestGraph
	contextAsm         *service.ContextAssembler // 上下文预制（侧边栏 current 模式历史注入）；nil=跳过
	pendingActions     *service.AIPendingActionService
	toolScopes         *service.ToolScopeService
	feedback           *service.AIFeedbackService
	configSvc          *service.SystemConfigService
}

// NewAIHandler 创建AI处理器。依赖经 Deps 一次性注入，不再逐个 Set*。
func NewAIHandler(deps AIHandlerDeps) *AIHandler {
	return &AIHandler{
		aiService:          deps.AIService,
		toolRegistry:       deps.ToolRegistry,
		avatarService:      deps.AvatarService,
		summaryGraph:       deps.SummaryGraph,
		textProcessGraph:   deps.TextProcessGraph,
		unifiedSearchGraph: deps.UnifiedSearchGraph,
		smartDigestGraph:   deps.SmartDigestGraph,
		contextAsm:         deps.ContextAssembler,
		pendingActions:     deps.PendingActions,
		toolScopes:         deps.ToolScopes,
		feedback:           deps.Feedback,
		configSvc:          deps.ConfigSvc,
	}
}

// RegisterRoutes 注册路由
func (h *AIHandler) RegisterRoutes(router *gin.RouterGroup) {
	aiGroup := router.Group("/ai")
	aiGroup.Use(checkAIEnabledMiddleware())
	{
		aiGroup.POST("/completion", h.GetCompletion)
		aiGroup.POST("/completion/stream", h.GetCompletionStream)
		// 侧边栏 AI 敏感工具（send_message）的待确认执行：确认后发送 / 取消
		aiGroup.POST("/pending-actions/:id/confirm", h.ConfirmPendingAction)
		aiGroup.POST("/pending-actions/:id/cancel", h.CancelPendingAction)
		// AI 回复用户反馈（👍/👎）
		aiGroup.POST("/feedback", h.SetAIMessageFeedback)
		aiGroup.GET("/feedback/:messageId", h.GetAIMessageFeedback)
		// 批量恢复反馈选中态（会话列表层一次拉取，替代每条 AI 消息挂载时各自 GET 的 N+1）
		aiGroup.POST("/feedback/batch", h.GetAIMessageFeedbackBatch)
		// 推荐提示词（侧边栏指令条 / bot 会话示例；admin 经 /admin/ai/suggested-prompts 配置）
		aiGroup.GET("/suggested-prompts", h.GetSuggestedPrompts)
		// 过时：同步「帮我回复」端点，无前端消费者（前端统一走 /draft-reply/stream），
		// 保留仅向后兼容，后续择机移除。
		aiGroup.POST("/draft-reply", h.DraftReply)
		aiGroup.POST("/draft-reply/stream", h.DraftReplyStream)
		aiGroup.GET("/tools", h.ListTools)

		// 新增: 会话摘要
		aiGroup.POST("/summary", h.GenerateSummary)
		aiGroup.POST("/summary/stream", h.GenerateSummaryStream)
		aiGroup.POST("/summary/meta", h.GenerateSummaryMeta)

		// 新增: 语义搜索
		aiGroup.POST("/search", h.AISearch)

		// 新增: 文本处理
		aiGroup.POST("/translate", h.TranslateText)
		aiGroup.POST("/rewrite", h.RewriteText)
		aiGroup.POST("/polish", h.PolishText)
		aiGroup.POST("/text-process", h.TextProcess)

		// 新增: 图片翻译
		aiGroup.POST("/translate/image", h.TranslateImage)

		// 图片识别/描述（复用图片翻译同一套视觉路由）
		aiGroup.POST("/describe-image", h.DescribeImage)

		// 新增: 智能消息速览
		aiGroup.GET("/digest", h.GetDigest)

		// 新增: 侧边栏元对话（随时问 AI，不改变会话本身）
		aiGroup.POST("/sidebar/stream", h.SidebarStream)

		// 运维相关路由已移除（5 个 ops handler 为死代码，前端未接入）
		// OpsDashboard 保留，通过 /admin/ai/dashboard 暴露（见 app/routes.go）
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
// @Router /api/v1/ai/completion [post]
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

	// 获取AI完成（统一回复路由 ResolveReplyTaskType：带图走视觉模型；纯文本按复杂度分级，
	// 复杂→digest 思考模型，简单→chat 快模型）
	// 带图但 ResolveReplyTaskType 仍返回 chat，即未配置视觉路由：诚实提示，
	// 而不是把 base64 塞给纯文本模型吃 400。
	taskType := h.aiService.ResolveReplyTaskType(req.Messages)
	if taskType == ai.TaskTypeChat && ai.HasImages(req.Messages) {
		response.BadRequest(c, errNoVisionRouteMsg)
		return
	}
	result, err := h.aiService.GetCompletion(taskType, req.Messages)
	if err != nil {
		response.InternalServerError(c, "AI请求失败：" + ai.UserMessage(err))
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

// DraftReply 根据对话上下文起草回复（过时）
// @Summary 帮我回复（已过时，将移除）
// @Description 根据目标消息及上下文生成回复草稿。⚠️ 已过时：前端已统一改走流式
// /draft-reply/stream，本同步端点无调用方，保留仅向后兼容，后续择机移除。
// @Tags AI
// @Accept json
// @Produce json
// @Param request body DraftReplyRequest true "起草请求"
// @Success 200 {object} AIResponse "成功响应"
// @Router /api/v1/ai/draft-reply [post]
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
			stream, err := h.draftReplyAvatarStream(c, userID, req, target, &avatarCfg)
			if err != nil {
				response.InternalServerError(c, "生成回复失败: " + err.Error())
				return
			}
			defer stream.Close()
			var reply strings.Builder
			for {
				msg, recvErr := stream.Recv()
				if recvErr != nil {
					if !errors.Is(recvErr, io.EOF) {
						response.InternalServerError(c, "生成回复失败: " + recvErr.Error())
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
	userIDAny, _ := c.Get("user_id")
	currentUserID, _ := userIDAny.(uint)
	messages, err := buildDraftReplyMessages(req, currentUserID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.aiService.GetCompletion(ai.TaskTypeChat, messages)
	if err != nil {
		response.InternalServerError(c, "AI请求失败：" + ai.UserMessage(err))
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
	if err := database.GetDB().Preload("Sender").First(&target, req.MessageID).Error; err != nil {
		return nil, fmt.Errorf("消息不存在")
	}
	if target.ConversationID != req.ConversationID {
		return nil, fmt.Errorf("消息与会话不匹配")
	}
	return &target, nil
}

// draftReplyAvatarStream 分身路径的草稿流：目标消息是图片时按 content 里的 file id 读图注入
// 多模态草稿（读图失败由 AvatarService 降级为纯文本草稿），否则按原文本起草。
// 对话历史锚定到目标消息的 CreatedAt——右键历史消息时目标可能不是会话最新一条，
// 只有取目标之前的消息作上下文，草稿才针对目标而非混进其后的后续对话。
func (h *AIHandler) draftReplyAvatarStream(c *gin.Context, userID uint, req DraftReplyRequest, target *model.Message, avatarCfg *model.AvatarConfig) (*schema.StreamReader[*schema.Message], error) {
	if target.Type != "image" {
		return h.avatarService.GenerateReplyStream(c.Request.Context(), userID, req.ConversationID, target.Content, &target.CreatedAt, avatarCfg)
	}
	fileID, name, _ := parseDraftImageMeta(target.Content)
	return h.avatarService.GenerateReplyStreamWithImageSources(c.Request.Context(), userID, req.ConversationID, name, fileID, &target.CreatedAt, avatarCfg)
}

// parseDraftImageMeta 从图片消息 Content（{"url":"...","id":123,"name":"xx.png"}）解析 file id 与文件名，
// 供帮我回复对图片目标走多模态草稿。解析失败返回 ok=false（fileID=0，AvatarService 侧统一降级）。
func parseDraftImageMeta(content string) (fileID uint, name string, ok bool) {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "{") {
		return 0, "", false
	}
	var payload struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil || payload.ID == 0 {
		return 0, "", false
	}
	return payload.ID, payload.Name, true
}

func buildDraftReplyMessages(req DraftReplyRequest, currentUserID uint) ([]ai.Message, error) {
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

	// 按时间正序拼装上下文文本，区分"我"和"对方"
	var myName string
	contextLines := []string{}
	for i := len(contextMsgs) - 1; i >= 0; i-- {
		m := contextMsgs[i]
		name := m.Sender.Nickname
		if name == "" {
			name = m.Sender.Username
		}
		if m.SenderID == currentUserID {
			contextLines = append(contextLines, fmt.Sprintf("我: %s", m.Content))
			if myName == "" {
				myName = name
			}
		} else {
			contextLines = append(contextLines, fmt.Sprintf("%s: %s", name, m.Content))
		}
	}
	contextText := ""
	if len(contextLines) > 0 {
		contextText = "对话上下文：\n" + fmt.Sprintf("%s\n\n", strings.Join(contextLines, "\n"))
	}

	targetName := target.Sender.Nickname
	if targetName == "" {
		targetName = "对方"
	}

	if myName == "" {
		// 目标消息前没有我的消息，从当前用户查昵称
		var me model.User
		if db.First(&me, currentUserID).Error == nil {
			myName = me.Nickname
			if myName == "" {
				myName = me.Username
			}
		}
	}

	// 图片目标：content 是 {"url":...} JSON，直接塞给模型只会让它对着 JSON 胡编。
	// 此路径无存储访问读不了图，换成可读占位文案让草稿至少指向图片本身。
	targetContent := target.Content
	if target.Type == "image" {
		if _, name, ok := parseDraftImageMeta(target.Content); ok && name != "" {
			targetContent = fmt.Sprintf("（对方发来一张图片「%s」，当前无法识别图片内容，请结合对话上下文起草回复。）", name)
		} else {
			targetContent = "（对方发来一张图片，当前无法识别图片内容，请结合对话上下文起草回复。）"
		}
	}

	userPrompt := fmt.Sprintf("%s需要回复的消息（来自 %s）：\n%s",
		contextText, targetName, targetContent)

	return []ai.Message{
		{Role: "system", Content: fmt.Sprintf("%s\n\n你是%s，需要以第一人称回复对方的消息。根据下面的对话上下文，起草一条回复。语气自然、简短，直接返回回复内容，不要加任何前缀、引号或解释。", aiprompt.CurrentTimeLine(), myName)},
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
// @Router /api/v1/ai/draft-reply/stream [post]
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
			stream, err := h.draftReplyAvatarStream(c, userID, req, target, &avatarCfg)
			if err != nil {
				logger.WithModule("AIHandler").Warn("分身流式生成失败，回退简单起草", "error", err)
				// 回退到简单起草路径
			} else {
				streamCompletionFromReader(c, stream)
				return
			}
		}
		// 无分身配置或分身失败 -> 回退简单起草
	}

	// 2) 回退：中立起草 prompt + 10 条上下文
	userIDAny, _ := c.Get("user_id")
	currentUserID, _ := userIDAny.(uint)
	messages, err := buildDraftReplyMessages(req, currentUserID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	h.streamCompletion(c, messages, ai.TaskTypeChat)
}

// streamCompletion 将一组消息以 SSE 流式推给前端（GetCompletionStream / DraftReplyStream 共用）
// streamSSE 设置 SSE 响应头，pump 通过 write 推送任意 StreamChunk 帧（content/pending/tool_event 等）；
// 结束后发 finish 事件。write 返回 error：客户端断开后写入失败时，pump 可提前终止，避免浪费 AI 调用。
// pump 返回非 nil error 时改为推送错误事件。供 streamCompletion / streamCompletionFromReader 共用，
// 避免 SSE 响应头与结束事件逻辑重复。
func streamSSE(c *gin.Context, pump func(write func(chunk ai.StreamChunk) error) error) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	write := func(chunk ai.StreamChunk) error {
		data, _ := json.Marshal(chunk)
		if _, err := c.Writer.Write([]byte("data: " + string(data) + "\n\n")); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}

	if err := pump(write); err != nil {
		errStr := "AI请求失败：" + ai.UserMessage(err)
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

// streamCompletion 流式推送一组 messages（经 aiService.GetCompletionStream，统一回复路由 ResolveReplyTaskType）
func (h *AIHandler) streamCompletion(c *gin.Context, messages []ai.Message, taskType ai.TaskType) {
	streamSSE(c, func(write func(ai.StreamChunk) error) error {
		return h.aiService.GetCompletionStream(taskType, messages, func(chunk ai.StreamChunk) error {
			if chunk.Content != "" {
				return write(ai.StreamChunk{Content: chunk.Content})
			}
			return nil
		})
	})
}

// toolDisplayName 将工具名映射为用户友好的中文提示
func toolDisplayName(toolName string) string {
	// 用户侧工具标签单一来源在 service.tool_label.go（与 bot 卡片 FriendlyToolLabel 共用）；
	// 未命中时按原语义回退工具原名
	if label, ok := service.UserToolLabel(toolName); ok {
		return label
	}
	return toolName
}

// streamCompletionWithTools 执行 ReAct 工具调用，实时推送执行进度，最后流式输出最终答案。
// 走真·流式 GetCompletionWithToolsStreamMultiStep：final 回合内容逐 token 经 onChunk 送出
// （真·打字机效果），工具事件仍经 onStep 实时推卡片。首回合若 Provider 不支持流式
// tool-call（如 Anthropic，返回 ErrStreamingToolsNotSupported）则降级到非流式
// GetCompletionWithToolsMultiStep、以切字块（保留打字感）发送结果。
// 白名单单一来源：service.SidebarAllowedTools（能力自述与实际放行同源，避免两处漂移）。

func (h *AIHandler) streamCompletionWithTools(c *gin.Context, messages []ai.Message, userID uint, conversationID uint) {
	callerCtx := &ai.CallerContext{
		UserID:         userID,
		ConversationID: conversationID,
		// 确认制工具清单单一来源：service.ConfirmToolsFor(ToolScopeSidebar)
		// （模型只生成待确认请求，用户在确认条点击后才真正发出）
		ConfirmTools: service.ConfirmToolsFor(service.ToolScopeSidebar),
	}
	// 生效工具面：admin 覆盖配置优先，未覆盖回退代码默认（nil 接收者安全）
	allowedTools := h.toolScopes.ScopeTools(service.ToolScopeSidebar)

	// streamSSE 的 write 直接收 StreamChunk 帧：content 文本帧、pending 确认帧、tool_event 进度帧
	// 统一走同一个写帧器（此前 writeEvent 是它的重复实现）。
	streamSSE(c, func(write func(ai.StreamChunk) error) error {
		// 注意：不再把「🤔 正在思考...」写进流内容——该文字会随回复一起落库、永久残留在
		// 侧边栏每条回复顶部。思考态改由前端组件 ThinkingIndicator（三点动画）在首个实
		// 际内容 chunk 到达前渲染，内容到达即自然替换，与主窗口/ BotChatView 保持一致。

		// 每步工具调用推结构化进度事件：start=running / end=ok|error，
		// 前端按 tool_call_id upsert 成工具轨迹（复用 ToolCallTrace，与 bot 会话同构）。
		// 确认制 send_message 成功时改推 pending 帧（确认条），不进轨迹避免双份展示。
		onStep := func(step int, toolCallID, phase, toolName string, args map[string]interface{}, result interface{}, err error) {
			if phase == "end" && err == nil && toolName == "send_message" {
				if info := service.PendingSendFromResult(result); info != nil {
					_ = write(ai.StreamChunk{Pending: info})
					return
				}
			}
			ev := &ai.ToolEvent{
				Step:       step,
				ToolCallID: toolCallID,
				ToolName:   toolName,
				Label:      toolDisplayName(toolName),
				Args:       args,
			}
			switch {
			case phase == "start":
				ev.Status = "running"
			case err != nil:
				ev.Status = "error"
				ev.Error = err.Error()
			default:
				ev.Status = "ok"
			}
			_ = write(ai.StreamChunk{ToolEvent: ev})
		}

		streamErr := h.aiService.GetCompletionWithToolsStreamMultiStep(
			c.Request.Context(),
			ai.TaskTypeChat,
			messages,
			callerCtx,
			allowedTools,
			0,
			onStep,
			func(chunk ai.StreamChunk) error {
				// final 回合内容逐 token 实时流出（真·打字机）
				if chunk.Content != "" {
					return write(ai.StreamChunk{Content: chunk.Content})
				}
				return nil
			},
		)
		if streamErr != nil && !errors.Is(streamErr, ai.ErrStreamingToolsNotSupported) {
			_ = write(ai.StreamChunk{Content: fmt.Sprintf("\n[错误：%s]", streamErr.Error())})
			return nil
		}

		// 首回合不支持流式 tool-call → 降级到非流式，跑完再切字块发送
		if errors.Is(streamErr, ai.ErrStreamingToolsNotSupported) {
			finalContent, err := h.aiService.GetCompletionWithToolsMultiStep(
				ai.TaskTypeChat,
				messages,
				callerCtx,
				allowedTools,
				0,
				onStep,
			)
			if err != nil {
				_ = write(ai.StreamChunk{Content: fmt.Sprintf("\n[错误：%s]", err.Error())})
				return nil
			}
			if err := write(ai.StreamChunk{Content: "\n"}); err != nil {
				return err
			}
			runes := []rune(finalContent)
			chunkSize := 4
			for i := 0; i < len(runes); i += chunkSize {
				end := i + chunkSize
				if end > len(runes) {
					end = len(runes)
				}
				if err := write(ai.StreamChunk{Content: string(runes[i:end])}); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// streamCompletionFromReader 把 Eino StreamReader 逐块以 SSE 推给前端（分身草稿流式路径）
// streamCompletionFromReader 把 Eino StreamReader 逐块以 SSE 推给前端（分身草稿流式路径）。
// Recv 返回 io.EOF 表示正常结束；其他 err 为上游真实错误，交 streamSSE 推送错误事件。
// defer Close 确保客户端断开或出错时释放 reader 端资源。
func streamCompletionFromReader(c *gin.Context, stream *schema.StreamReader[*schema.Message]) {
	defer stream.Close()
	streamSSE(c, func(write func(ai.StreamChunk) error) error {
		for {
			msg, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			}
			if msg != nil && msg.Content != "" {
				if err := write(ai.StreamChunk{Content: msg.Content}); err != nil {
					return err
				}
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

// groupCollectionRe 匹配群集合名（group_{id}）。命中时 GetKnowledgeGraph 走按群构建拓扑，
// 否则回落到按集合平铺向量块的旧逻辑。
var groupCollectionRe = regexp.MustCompile(`^group_(\d+)$`)

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
// @Router /api/v1/admin/knowledge-graph [get]
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

	// 群集合（group_{id}）走真正的拓扑构建（文档/实体节点 + 关系边 + 实体反查），
	// 对齐分身知识图谱形态；非群集合回落到下方按集合平铺向量块的旧逻辑。
	if m := groupCollectionRe.FindStringSubmatch(collection); m != nil {
		groupID, _ := strconv.ParseUint(m[1], 10, 32)
		docSvc := di.GlobalContainer.GroupDocumentService
		if docSvc != nil {
			graph, err := docSvc.BuildGroupKnowledgeGraph(uint(groupID), query, maxNodes)
			if err != nil {
				logger.WithModule("GroupKnowledgeGraph").Error("构建群知识图谱失败", "groupID", groupID, "error", err)
			} else {
				response.Success(c, gin.H{
					"nodes":           graph.Nodes,
					"edges":           graph.Edges,
					"total_nodes":     graph.TotalNodes,
					"total_edges":     graph.TotalEdges,
					"knowledge_count": graph.KnowledgeCount,
				})
				return
			}
		}
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

// ToolRegistryConfigRequest 工具注册表配置请求
type ToolRegistryConfigRequest struct {
	Enabled *bool `json:"enabled" binding:"required"`
}

// ListToolRegistryTools 列出所有 AI 工具（包含启用状态）
// @Summary 列出所有 AI 工具（管理后台）
// @Description 列出所有 AI 工具及其启用状态，用于管理后台配置
// @Tags AI工具管理
// @Produce json
// @Success 200 {object} AIResponse "成功响应"
// @Router /api/v1/admin/tool-registry/tools [get]
func (h *AIHandler) ListToolRegistryTools(c *gin.Context) {
	if h.toolRegistry == nil {
		response.InternalServerError(c, "AI工具注册表未初始化")
		return
	}

	tools := h.toolRegistry.ListTools()
	response.Success(c, gin.H{
		"tools": tools,
		"total": len(tools),
	})
}

// UpdateToolRegistryConfig 更新 AI 工具配置
// @Summary 更新 AI 工具配置
// @Description 启用或禁用指定的 AI 工具
// @Tags AI工具管理
// @Accept json
// @Produce json
// @Param tool_name path string true "工具名称"
// @Param request body ToolRegistryConfigRequest true "工具配置请求"
// @Success 200 {object} AIResponse "成功响应"
// @Failure 400 {object} AIResponse "参数错误"
// @Failure 404 {object} AIResponse "工具不存在"
// @Router /api/v1/admin/tool-registry/tools/{tool_name} [put]
func (h *AIHandler) UpdateToolRegistryConfig(c *gin.Context) {
	if h.toolRegistry == nil {
		response.InternalServerError(c, "AI工具注册表未初始化")
		return
	}

	toolName := c.Param("tool_name")
	var req ToolRegistryConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	var err error
	if req.Enabled != nil && *req.Enabled {
		err = h.toolRegistry.EnableTool(toolName)
	} else {
		err = h.toolRegistry.DisableTool(toolName)
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
// @Router /api/v1/admin/ai/dashboard [get]
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

	if h.toolRegistry != nil {
		tools := h.toolRegistry.ListTools()
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
// @Router /api/v1/ai/completion/stream [post]
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

	// 带图但 ResolveReplyTaskType 仍返回 chat，即未配置视觉路由：诚实提示而非硬塞 400（与非流式端点一致）
	taskType := h.aiService.ResolveReplyTaskType(req.Messages)
	if taskType == ai.TaskTypeChat && ai.HasImages(req.Messages) {
		response.BadRequest(c, errNoVisionRouteMsg)
		return
	}
	h.streamCompletion(c, req.Messages, taskType)
}

// ListTools 列出所有 AI 工具
// @Summary 列出所有 AI 工具
// @Description 列出所有可用的 AI 工具
// @Tags AI
// @Produce json
// @Success 200 {object} AIResponse "成功响应"
// @Router /api/v1/ai/tools [get]
func (h *AIHandler) ListTools(c *gin.Context) {
	if h.toolRegistry == nil {
		response.InternalServerError(c, "AI工具注册表未初始化")
		return
	}

	tools := h.toolRegistry.ListTools()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    tools,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// 侧边栏元对话
// ─────────────────────────────────────────────────────────────────────────────

// SidebarStreamRequest 侧边栏元对话请求
type SidebarStreamRequest struct {
	Message        string `json:"message" binding:"required"` // 用户问题
	ConversationID uint   `json:"conversation_id"`            // 可选：当前会话 ID，传入则注入该会话上下文
	Scope          string `json:"scope"`                      // "current"(默认) 或 "cross"(跨会话检索)
}

// SidebarStream 侧边栏元对话（SSE 流式）
// @Summary 侧边栏 AI 元对话
// @Description 在任意会话旁边随时问 AI，不改变对话本身的社交动态。
// @Description scope=current: 注入当前会话最近消息作为上下文；
// @Description scope=cross: 跨会话检索用户全部消息/笔记/知识库后综合回答。
// @Tags AI
// @Accept json
// @Produce text/event-stream
// @Param request body SidebarStreamRequest true "侧边栏请求"
// @Success 200 {string} string "流式输出"
// @Router /api/v1/ai/sidebar/stream [post]
func (h *AIHandler) SidebarStream(c *gin.Context) {
	var req SidebarStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if !h.aiService.IsConfigured() {
		response.InternalServerError(c, "AI服务未配置")
		return
	}

	userIDAny, _ := c.Get("user_id")
	userID, _ := userIDAny.(uint)
	if userID == 0 {
		response.Unauthorized(c, "未登录")
		return
	}

	scope := req.Scope
	if scope == "" {
		scope = "current"
	}

	switch scope {
	case "cross":
		h.sidebarCrossConversation(c, req, userID)
	default:
		h.sidebarCurrentConversation(c, req, userID)
	}
}

// sidebarCurrentConversation 当前会话上下文模式：
// 注入最近 20 条消息 + 会话元信息，让 AI 基于“刚才讨论的内容”回答。
// 历史注入统一走 ContextAssembler 声明式装配（future 加笔记/记忆/群文档源只需在 sources 加一行）；
// 群名属会话元信息（非检索源），保留在此由调用方注入。
func (h *AIHandler) sidebarCurrentConversation(c *gin.Context, req SidebarStreamRequest, userID uint) {
	db := database.GetDB()

	systemPrompt := h.buildSidebarSystemPrompt()

	aiMessages := []ai.Message{
		{Role: "system", Content: systemPrompt},
	}

	if req.ConversationID > 0 {
		if err := ensureConversationAccess(db, req.ConversationID, userID); err != nil {
			response.Forbidden(c, "无权访问该会话")
			return
		}

		// 恢复本函数的既有消息结构：group 名会话元信息 + 历史注入块合成单个 user 上下文消息。
		var contextText string
		// 群名（会话元信息，非检索源）
		var group model.Group
		if db.Where("conversation_id = ?", req.ConversationID).First(&group).Error == nil {
			contextText += fmt.Sprintf("【当前会话】%s\n", group.Name)
		}
		// 历史注入块（统一走 ContextAssembler 声明式装配）
		if h.contextAsm != nil {
			bundle := h.contextAsm.Assemble(c.Request.Context(), req.Message, []service.ContextSource{
				{Type: service.SourceHistory, Key: req.ConversationID, TopK: 20},
			})
			if len(bundle.Messages) > 0 {
				contextText += bundle.Messages[0].Content
			}
		}
		if contextText != "" {
			aiMessages = append(aiMessages, ai.Message{Role: "user", Content: contextText})
			aiMessages = append(aiMessages, ai.Message{Role: "assistant", Content: "已了解当前会话上下文，请提问。"})
		}
	}

	aiMessages = append(aiMessages, ai.Message{Role: "user", Content: req.Message})
	h.streamCompletionWithTools(c, aiMessages, userID, req.ConversationID)
}

// sidebarCrossConversation 跨会话检索模式：
// 调用 UnifiedSearchGraph 检索用户全部消息/笔记/知识库/记忆，综合回答。
func (h *AIHandler) sidebarCrossConversation(c *gin.Context, req SidebarStreamRequest, userID uint) {
	if req.ConversationID > 0 {
		if err := ensureConversationAccess(database.GetDB(), req.ConversationID, userID); err != nil {
			response.Forbidden(c, "无权访问该会话")
			return
		}
	}

	if h.unifiedSearchGraph == nil {
		// 降级：无搜索图时用纯 LLM 回答
		aiMessages := []ai.Message{
			{Role: "system", Content: h.buildSidebarSystemPrompt()},
			{Role: "user", Content: req.Message},
		}
		h.streamCompletionWithTools(c, aiMessages, userID, req.ConversationID)
		return
	}

	// 同步执行搜索图（内部已并行检索 4 路源），拿到结果后流式输出
	input := &service.UnifiedSearchInput{
		Query:          req.Message,
		UserID:         userID,
		ConversationID: req.ConversationID, // 可为 0，表示不限定会话
	}

	result, err := h.unifiedSearchGraph.Execute(c.Request.Context(), input)
	if err != nil {
		logger.WithModule("SidebarStream").Error("跨会话检索失败", "error", err)
		// 降级：纯 LLM
		aiMessages := []ai.Message{
			{Role: "system", Content: h.buildSidebarSystemPrompt()},
			{Role: "user", Content: req.Message},
		}
		h.streamCompletionWithTools(c, aiMessages, userID, req.ConversationID)
		return
	}

	// 将搜索结果作为上下文，流式生成最终回答
	var contextParts []string
	for _, src := range result.Sources {
		label := src.Type
		if src.Title != "" {
			label += ": " + src.Title
		}
		contextParts = append(contextParts, fmt.Sprintf("[%s] %s", label, src.Content))
	}

	aiMessages := []ai.Message{
		{Role: "system", Content: h.buildSidebarSystemPrompt()},
	}
	if len(contextParts) > 0 {
		aiMessages = append(aiMessages, ai.Message{
			Role:    "user",
			Content: "【跨会话检索结果】\n" + strings.Join(contextParts, "\n\n"),
		})
		aiMessages = append(aiMessages, ai.Message{Role: "assistant", Content: "已检索到相关信息，请提问。"})
	}
	aiMessages = append(aiMessages, ai.Message{Role: "user", Content: req.Message})

	h.streamCompletionWithTools(c, aiMessages, userID, req.ConversationID)
}

// buildSidebarSystemPrompt 侧边栏元对话的 system prompt。
// 末尾动态注入能力自述（静态能力 + 侧边栏实际可调工具），让 AI 被问「具备哪些能力」时能如实回答。
func (h *AIHandler) buildSidebarSystemPrompt() string {
	prompt := fmt.Sprintf(`%s

你是 %s 企业即时通讯系统中的 AI 助手，正在通过侧边栏与用户进行元对话。

【你的角色】
- 你是用户的私人智能助理，帮助用户理解、总结、分析当前或历史对话内容
- 你的回复只有用户自己能看到，不会发送到群聊中
- 你可以自由地给出建议、评价、整理待办、提炼结论
- 当用户要求创建任务、设置提醒、记录待办时，你可以调用 create_user_task 工具直接帮用户创建

【回复规则】
- 使用中文回答
- 简洁、结构化，善用列表和标题
- 如果用户问“刚才讨论的结论是什么”，从对话记录中提炼
- 如果用户问“帮我整理待办”，抽取行动项并以 checkbox 格式输出
- 如果用户说“帮我创建任务”或“设置提醒”，使用 create_user_task 工具创建，并告诉用户已创建成功
- 如果信息不足，诚实告知并建议用户提供更多上下文`, aiprompt.CurrentTimeLine(), productname.Name)

	// 能力自述：静态能力 + 侧边栏实际可调工具，随 allowlist 动态变化。
	if capPrompt := h.capabilityPrompt(h.toolScopes.ScopeTools(service.ToolScopeSidebar)); capPrompt != "" {
		prompt += "\n\n【能力与工具】\n" + capPrompt
	}
	return prompt
}

// capabilityPrompt 生成能力自述文本块（静态能力 + 指定工具集，动态）。
// 对 aiService / 其注册表为 nil 做安全降级（返回空串，不注入）。
func (h *AIHandler) capabilityPrompt(allowed []string) string {
	if h.aiService == nil {
		return ""
	}
	return h.aiService.GetToolRegistry().BuildCapabilityPrompt(allowed)
}
