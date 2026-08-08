package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/aiprompt"
	"github.com/dshmyz/qim/qim-server/pkg/productname"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"gorm.io/gorm"
)

type KnowledgeRetriever interface {
	BuildContext(query string, groupID uint) string
}

type LegacyKnowledgeService interface {
	BuildKnowledgeContext(query string) string
}

// QuotedDocumentReader 按 fileID 读取单个被引用对象内容（不入库）。
// *GroupDocumentService 实现此接口；注入 nil 时被引用文件/图片解析自动降级（不回读）。
type QuotedDocumentReader interface {
	ExtractTextForContext(fileID uint) (name string, text string, err error)
	// ImageURLForContext 读取被引用图片原始字节并转成 base64 data URL，供群 AI 多模态识别。
	// 可选：未实现时（旧注入或 nil reader）图片分支按"读不了"降级。
	ImageURLForContext(fileID uint) (name string, dataURL string, err error)
}

// GroupAIConfig 群组专属 AI 配置（非群聊场景下为 nil）
type GroupAIConfig struct {
	Personality  string // casual, concise, friendly, technical
	Language     string // zh, en
	MaxLength    string // short, medium, long
	CustomPrompt string // 自定义提示词
	MembersList  string // 群成员名单
	Stats        string // 群统计数据
}

// QuotedKind 标注被引用对象（文件正文 / 图片 / 读取失败）的判别类型。
// 用单一 Kind 替代此前 QuotedFileCtx / QuotedFileFailed / QuotedImageURL 三个字符串字段：
// 那些字段把同一对"成功/失败"语义散在三处，且图片成功时需 QuotedFileCtx(提示词)与
// QuotedImageURL(数据)两字段并存，互斥关系靠注释维系；改为判别联合后，成功/失败成单一
// 分支，新增媒体类型（如 audio/video）无需再加字段。
type QuotedKind string

const (
	QuotedNone   QuotedKind = ""       // 无被引用对象（未引用或引用非文件/图片）
	QuotedFile   QuotedKind = "file"   // 成功读到被引用文件正文
	QuotedImage  QuotedKind = "image"  // 成功读到被引用图片（可多模态识别）
	QuotedFailed QuotedKind = "failed" // 读取失败/类型不支持/过大/缺信息，Text 为提示语
)

// QuotedContext 被引用对象（文件正文 / 图片）的上下文注入。
// Name 为可读名称（文件名等），Text 为注入 prompt 的成句内容（含提示语），
// ImageURL 仅 QuotedImage 时非空（base64 data URL）。三者按 Kind 取用。
type QuotedContext struct {
	Kind     QuotedKind
	Name     string
	Text     string // file: 「被引用文件「x」的内容：…」；image/failed: 提示语成句
	ImageURL string // image: base64 data URL
}

// SmartReplyContext 通用智能回复上下文
type SmartReplyContext struct {
	// 基础信息
	Message         string
	OriginalContent string
	UserID          uint
	ConversationID  uint
	IsAIMention     bool
	AssistantName   string
	Intent          *ai.MessageIntent
	// QuotedMessageID 用户 @AI 文本消息所引用的消息 ID（可能为 nil）。
	// 引用了一条文件/图片消息时，AI 可借此读取其内容作为上下文。
	QuotedMessageID *uint
	// Quoted 被引用对象（文件正文 / 图片 / 读取失败）的上下文注入，nil 表示无被引用内容。
	// 由 prepareInput 按被引用消息类型与成败设值；判别联合见 QuotedContext。
	Quoted *QuotedContext

	// 动态上下文
	KnowledgeCtx string // 知识库检索结果
	MemoryCtx    string // 长期记忆检索结果
	ChatHistory  string // 历史对话记录
	PendingTasks string // 用户待办任务（仅群聊或特定场景）

	// 扩展配置
	Group       *model.Group
	GroupConfig *GroupAIConfig

	// 补齐 Legacy 上下文
	User        *model.User  // 当前提问用户
	Tasks       []model.Task // 用户未完成任务
	MemberNames string       // 群成员名单（逗号分隔）
	GroupStats  string       // 群统计信息
}

type SmartReplyResult struct {
	Reply    string
	IsStream bool
}

type SmartReplyGraph struct {
	replyGraph       compose.Runnable[*SmartReplyContext, *SmartReplyResult]
	aiService        *ai.AIService
	db               *gorm.DB
	unifiedKnowledge KnowledgeRetriever
	legacyKnowledge  LegacyKnowledgeService
	groupMemorySvc   *GroupMemoryService
	userSvc          *UserService
	quotedFile       QuotedDocumentReader
	mcpGateway       *MCPClientGateway
}

func NewSmartReplyGraph(
	aiService *ai.AIService,
	db *gorm.DB,
	unifiedKnowledge KnowledgeRetriever,
	legacyKnowledge LegacyKnowledgeService,
	groupMemorySvc *GroupMemoryService,
	userSvc *UserService,
) *SmartReplyGraph {
	return &SmartReplyGraph{
		aiService:        aiService,
		db:               db,
		unifiedKnowledge: unifiedKnowledge,
		legacyKnowledge:  legacyKnowledge,
		groupMemorySvc:   groupMemorySvc,
		userSvc:          userSvc,
	}
}

func (g *SmartReplyGraph) BuildGraph() error {
	if err := g.buildReplyGraph(); err != nil {
		return fmt.Errorf("构建回复 Graph 失败: %w", err)
	}
	return nil
}

// SetQuotedFileReader 注入被引用文件正文读取器（*GroupDocumentService 实现）。
// 传 nil 可关闭按引用读取文件正文的能力（安全降级）。
func (g *SmartReplyGraph) SetQuotedFileReader(reader QuotedDocumentReader) {
	g.quotedFile = reader
}

// SetMCPGateway 注入外部 MCP 客户端网关。非 nil 时，若后台开启了
// external_mcp:group_enabled，群 @AI 会把网关注册的外部 MCP 工具（mcp_*）
// 追加进可用工具集，供 ReAct 循环调用；未注入（nil）或未开启则行为不变。
func (g *SmartReplyGraph) SetMCPGateway(gateway *MCPClientGateway) {
	g.mcpGateway = gateway
}

// groupAssistantAllowedTools 计算群 @AI 实际可用的工具白名单：内置群管理工具
// + （若开启）外部 MCP 工具。返回新 slice，避免改动包级白名单。
func (g *SmartReplyGraph) groupAssistantAllowedTools() []string {
	allowed := append([]string(nil), groupAssistantToolWhitelist...)
	if g.mcpGateway != nil && g.mcpGateway.GroupEnabled() {
		allowed = append(allowed, g.mcpGateway.ListExternalToolNames()...)
	}
	return allowed
}

var registerReplyMergeOnce sync.Once

func (g *SmartReplyGraph) buildReplyGraph() error {
	registerReplyMergeOnce.Do(func() {
		compose.RegisterValuesMergeFunc(func(vs []*SmartReplyContext) (*SmartReplyContext, error) {
			return vs[0], nil
		})
	})

	graph := compose.NewGraph[*SmartReplyContext, *SmartReplyResult]()

	graph.AddLambdaNode("prepare", g.createPrepareNode())
	graph.AddLambdaNode("knowledge", g.createKnowledgeNode())
	graph.AddLambdaNode("memory", g.createMemoryNode())
	graph.AddLambdaNode("history", g.createHistoryNode())
	graph.AddLambdaNode("merge", g.createMergeNode())

	// 直接构建 Messages 节点（避免 ChatTemplate 变量替换问题）
	graph.AddLambdaNode("build_messages", g.createBuildMessagesNode())

	graph.AddChatModelNode("model", NewEinoChatModel(g.aiService, ai.TaskTypeChat, 0))

	graph.AddLambdaNode("format", g.createFormatReplyNode())

	graph.AddEdge(compose.START, "prepare")
	graph.AddEdge("prepare", "knowledge")
	graph.AddEdge("prepare", "memory")
	graph.AddEdge("prepare", "history")
	graph.AddEdge("knowledge", "merge")
	graph.AddEdge("memory", "merge")
	graph.AddEdge("history", "merge")
	graph.AddEdge("merge", "build_messages")
	graph.AddEdge("build_messages", "model")
	graph.AddEdge("model", "format")
	graph.AddEdge("format", compose.END)

	ctx := context.Background()
	runnable, err := graph.Compile(ctx, compose.WithGraphName("SmartReply"))
	if err != nil {
		return fmt.Errorf("编译 Graph 失败: %w", err)
	}
	g.replyGraph = runnable

	return nil
}

func (g *SmartReplyGraph) ExecuteStream(ctx context.Context, input *SmartReplyContext) (*schema.StreamReader[*schema.Message], error) {
	if err := g.prepareInput(input); err != nil {
		return nil, err
	}
	historyMessages := g.buildHistoryMessages(input)
	// 被引用图片时走视觉任务类型（多模态），否则常规对话。
	// 若未显式配置视觉路由，TaskTypeVision 会回退到 defaultTask（纯文本 chat 模型），
	// 把图片 base64 发给它必然 400。此时把被引用图片降级为 QuotedFailed 提示语并在
	// 常规对话任务下走完，让 AI 诚实说明"当前模型不支持看图"，而不触发模型调用错误。
	taskType := ai.TaskTypeChat
	if input.Quoted != nil && input.Quoted.Kind == QuotedImage {
		if g.aiService.HasVisionRoute() {
			taskType = ai.TaskTypeVision
		} else {
			log.Printf("[SmartReplyGraph] 未配置视觉路由，引用图片降级为普通对话")
			input.Quoted = &QuotedContext{
				Kind: QuotedFailed,
				Name: input.Quoted.Name,
				Text: fmt.Sprintf("📷 你引用了一条图片消息「%s」，但当前配置的模型不支持查看图片。请如实说明你看不到图片，可请对方把图片里的关键信息用文字发出来。", input.Quoted.Name),
			}
		}
	}
	chatModel := NewEinoChatModel(g.aiService, taskType, input.UserID)
	return chatModel.Stream(ctx, historyMessages)
}

// groupAssistantToolWhitelist 群聊助手可用的工具白名单：只含群聊相关工具，
// 排除运维工具（intelligent_troubleshooting 等）和系统级用户管理工具。
var groupAssistantToolWhitelist = []string{
	"group_management", "create_group_task", "search_messages", "group_summary", "system_notification",
}

// ExecuteWithTools 带工具的非流式回复，用于 @AI 管理操作指令（踢人/加人/禁言等）。
// 走 GetCompletionWithToolsMultiStep 注入白名单 AI 工具（含 GroupManagementTool）并多步循环，
// LLM 返回 tool call 时真实执行。callerCtx 用 input.UserID，isSystemAdmin 校验生效，
// 即仅群主/管理员发起的管理指令会被工具执行，普通成员指令被工具拒绝。
// 采用 MultiStep 而非单轮 core 的原因：工具执行出错时（如"用户不存在"）MultiStep 会把
// 错误以 tool 角色消息回喂给 LLM（见 ai_service.go 中 ReAct 循环），让群助手基于错误
// 生成自然回复，而不是像旧路径那样把错误硬抛到 handler 直接静默失败。
func (g *SmartReplyGraph) ExecuteWithTools(ctx context.Context, input *SmartReplyContext) (string, error) {
	if err := g.prepareInput(input); err != nil {
		return "", err
	}
	historyMessages := g.buildHistoryMessages(input)
	callerCtx := &ai.CallerContext{UserID: input.UserID}
	return g.aiService.GetCompletionWithToolsMultiStep(
		ai.TaskTypeChat, einoMessagesToAIMessages(historyMessages), callerCtx, g.groupAssistantAllowedTools(),
		ai.MaxReActSteps, nil,
	)
}

// HasExternalTools 报告群 AI 普通提问路径是否有可用的外部 MCP 工具。
// 仅当「网关已注入 && 后台开启 external_mcp:group_enabled && 确有已注册外部工具」
// 才返回 true，用于路由普通提问进带外部工具的流式 ReAct；其余情况（默认关闭）
// 保持原有纯流式路径，零行为变化。
func (g *SmartReplyGraph) HasExternalTools() bool {
	if g.mcpGateway == nil || !g.mcpGateway.GroupEnabled() {
		return false
	}
	return len(g.mcpGateway.ListExternalToolNames()) > 0
}

// ExecuteStreamWithExternalTools 普通提问（非管理指令）走带外部 MCP 工具的
// 多步 ReAct 循环，供群 AI 在自然语言下自主调用外部工具；最终返回完整回复文本。
//
// 与 ExecuteWithTools 的区别：白名单只用外部 MCP 工具名（mcp_*），不注入内建
// 群管理工具——普通提问不该被 group_management 等扩权，精确对焦"外部工具在
// 普通提问可用"。feedback（ReActStepCallback）在每步工具执行后被调用，handler
// 可借此向流式气泡追加"🔧 正在调用 XXX…"的过程反馈，回应 tool_call 挂起时段。
//
// 注：这是"ReAct 循环 + 过程反馈"的非逐 token 变体（复用 GetCompletionWithToolsMultiStep），
// 不调工具的回合最终答案由调用方切子块发送以保留打字感；真·流式 tool-call 解析留后续。
func (g *SmartReplyGraph) ExecuteStreamWithExternalTools(ctx context.Context, input *SmartReplyContext, feedback ai.ReActStepCallback) (string, error) {
	if err := g.prepareInput(input); err != nil {
		return "", err
	}
	historyMessages := g.buildHistoryMessages(input)
	callerCtx := &ai.CallerContext{UserID: input.UserID}
	return g.aiService.GetCompletionWithToolsMultiStep(
		ai.TaskTypeChat, einoMessagesToAIMessages(historyMessages), callerCtx,
		g.mcpGateway.ListExternalToolNames(), ai.MaxReActSteps, feedback,
	)
}

// prepareInput 补齐 SmartReplyContext 的群/用户/待办/成员/知识库/记忆等上下文，
// 供 ExecuteStream 与 ExecuteWithTools 复用。
func (g *SmartReplyGraph) prepareInput(input *SmartReplyContext) error {
	var conv model.Conversation
	if err := g.db.First(&conv, input.ConversationID).Error; err != nil {
		return fmt.Errorf("会话不存在")
	}
	if conv.Type == "group" || conv.Type == "discussion" {
		var group model.Group
		if err := g.db.Where("conversation_id = ?", input.ConversationID).First(&group).Error; err == nil {
			input.Group = &group
			aiConfig := group.GetAIConfig()
			input.GroupConfig = &GroupAIConfig{
				Personality:  aiConfig.Personality,
				Language:     aiConfig.Language,
				MaxLength:    aiConfig.MaxLength,
				CustomPrompt: aiConfig.CustomPrompt,
			}
		}
	}

	// 补齐：当前提问用户
	var user model.User
	if err := g.db.First(&user, input.UserID).Error; err == nil {
		input.User = &user
	}

	// 补齐：用户待办任务
	var tasks []model.Task
	g.db.Where("user_id = ? AND status = 'todo'", input.UserID).
		Order("due_date ASC").
		Limit(5).
		Find(&tasks)
	input.Tasks = tasks

	// 补齐：群成员列表 + 群统计
	if input.Group != nil {
		var members []model.ConversationMember
		if err := g.db.Preload("User").Where("conversation_id = ?", input.ConversationID).Find(&members).Error; err == nil {
			names := make([]string, 0, len(members))
			for _, m := range members {
				name := m.User.Nickname
				if name == "" {
					name = m.User.Username
				}
				names = append(names, name)
			}
			input.MemberNames = strings.Join(names, "、")
		}

		var totalMessages int64
		g.db.Model(&model.Message{}).Where("conversation_id = ?", input.ConversationID).Count(&totalMessages)
		var memberCount int64
		g.db.Model(&model.ConversationMember{}).Where("conversation_id = ?", input.ConversationID).Count(&memberCount)
		input.GroupStats = fmt.Sprintf("总消息数：%d\n成员数：%d", totalMessages, memberCount)
	}

	knowledgeCtx := ""
	if g.unifiedKnowledge != nil && input.Group != nil {
		query := input.Message
		if query == "" && input.Group.Name != "" {
			query = input.Group.Name
		}
		knowledgeCtx = g.unifiedKnowledge.BuildContext(query, input.Group.ID)
	} else if g.legacyKnowledge != nil {
		knowledgeCtx = g.legacyKnowledge.BuildKnowledgeContext(input.Message)
	}
	input.KnowledgeCtx = knowledgeCtx

	memoryCtx := ""
	if g.groupMemorySvc != nil && input.Group != nil {
		memoryResults, err := g.groupMemorySvc.Recall(input.Group.ID, input.Message, 2)
		if err == nil && len(memoryResults) > 0 {
			var parts []string
			for _, r := range memoryResults {
				parts = append(parts, r.Content)
			}
			memoryCtx = "💡 群聊记忆：\n" + strings.Join(parts, "\n")
		}
	}
	input.MemoryCtx = memoryCtx

	// 被引用对象：@AI 消息引用了消息时，若被引用的是文件/图片消息则尝试读取其内容注入上下文。
	// 仅对 @AI 提及（IsAIMention）场景生效。所有边界（非文件/图片、解析失败、过大、为空、缺 id）
	// 都显式落成 QuotedFailed 提示语，让 AI 诚实说明"看不了"，而不是假装读取——不依赖大模型自行识别。
	// 成功时按内容类型设 QuotedFile(正文) / QuotedImage(data URL)，失败统一设 QuotedFailed，成功/失败互斥。
	if input.IsAIMention && input.QuotedMessageID != nil && g.quotedFile != nil {
		warn := func(t QuotedKind, name, msg string) {
			input.Quoted = &QuotedContext{Kind: t, Name: name, Text: msg}
		}
		var quoted model.Message
		if err := g.db.First(&quoted, *input.QuotedMessageID).Error; err == nil && quoted.Type == "file" {
			quotedName := nameOfQuoted(quoted.Content)
			fileID := parseQuotedFileID(quoted.Content)
			if fileID == 0 {
				// 被引用的是文件消息但拿不到文件 id（Content 缺失/非 JSON）
				warn(QuotedFailed, quotedName, fmt.Sprintf("📄 你引用了一条文件消息「%s」，但其内容缺少可解析的文件信息，无法读取正文。请说明你无法读取该文件，可请对方重新发送。", quotedName))
			} else if name, text, err := g.quotedFile.ExtractTextForContext(fileID); err == nil {
				text = truncateQuotedFileText(text)
				if text != "" {
					// 截断后仍可能有内容则注入正文
					warn(QuotedFile, name, fmt.Sprintf("📄 被引用文件「%s」的内容：\n%s", name, text))
				} else {
					// 解析成功但正文为空（如空文档），明确告知而非假装读到
					warn(QuotedFailed, quotedName, fmt.Sprintf("📄 你引用了一条文件消息「%s」，但其内容为空或无法提取出文字，请如实说明。", quotedName))
				}
			} else if errors.Is(err, ErrQuotedFileTooLarge) {
				// 文件过大：提示语区别于"类型不支持"
				warn(QuotedFailed, quotedName, fmt.Sprintf("📄 你引用了一条文件消息「%s」，但其体积过大（超过 20MB），无法一次性读入上下文。请说明你只能读取较小的文本/文档文件，可建议对方拆分成多份。", quotedName))
			} else {
				// 类型不支持 / 解析失败 / 存储读取失败：显式告知 AI，让它诚实回复"读不了"
				warn(QuotedFailed, quotedName, fmt.Sprintf("📄 你引用了一条文件消息「%s」，但该文件无法读取正文（类型不在可读取范围：txt/md/csv/json/pdf/docx/xlsx/pptx，或存在其他读取/解析错误）。请说明你无法读取该文件内容，可建议对方转成上述格式或上传到群知识库。", quotedName))
			}
		} else if err == nil && (quoted.Type == "image" || quoted.Type == "video" || quoted.Type == "audio") {
			quotedName := nameOfQuoted(quoted.Content)
			if quoted.Type == "image" {
				// 图片：走多模态路径，成功则注入 base64 data URL，失败降级为"看不了"。
				fileID := parseQuotedFileID(quoted.Content)
				if fileID == 0 {
					warn(QuotedFailed, quotedName, fmt.Sprintf("📷 你引用了一条图片消息「%s」，但其内容缺少可解析的图片信息，无法读取。请如实说明你看不到该图片，可请对方重新发送。", quotedName))
				} else if name, dataURL, derr := g.quotedFile.ImageURLForContext(fileID); derr == nil && dataURL != "" {
					warn(QuotedImage, name, fmt.Sprintf("📷 你引用了一张图片「%s」，请识别其内容并结合用户的问题回答。", name))
					input.Quoted.ImageURL = dataURL
				} else if errors.Is(derr, ErrQuotedImageTooLarge) {
					warn(QuotedFailed, quotedName, fmt.Sprintf("📷 你引用了一条图片消息「%s」，但其体积过大（超过 5MB），无法读入上下文。请如实说明你看不到该图片，可建议对方压缩后重新发送。", quotedName))
				} else {
					// 读取失败 / 存储读取失败 / reader 未实现多模态：显式告知 AI 看不到图，诚实回复。
					warn(QuotedFailed, quotedName, fmt.Sprintf("📷 你引用了一条图片消息「%s」，但该图片当前无法读入上下文（读取失败或图片不可用）。请如实说明你看不到该图片，可建议对方重新发送。", quotedName))
				}
			} else {
				// 引用的是视频/语音消息：当前不支持解析，显式告知而不是当作无引用
				warn(QuotedFailed, quotedName, fmt.Sprintf("📄 你引用了一条%s消息，但该类型目前无法解析其内容进上下文。请如实说明你无法读取该%s。", mediaTypeName(quoted.Type), mediaTypeName(quoted.Type)))
			}
		}
	}

	return nil
}

// parseQuotedFileID 从文件消息的 Content（{"url":"...","id":123}）解析出 file id。
func parseQuotedFileID(content string) uint {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "{") {
		return 0
	}
	var payload struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return 0
	}
	return payload.ID
}

// parseQuotedFileName 从文件消息的 Content（{"url":"...","name":"xx.pdf"}）解析出文件名，供提示语展示；解析失败返回空串。
func parseQuotedFileName(content string) string {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "{") {
		return ""
	}
	var payload struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return ""
	}
	return payload.Name
}

// nameOfQuoted 取被引用消息的可展示名称：优先取 Content 里解析出的文件名，取不到则回退为原始 Content（最多截 80 字符），保证提示语里至少有个可辨识的占位。
func nameOfQuoted(content string) string {
	if name := parseQuotedFileName(content); name != "" {
		return name
	}
	trimmed := strings.TrimSpace(content)
	if len(trimmed) > 80 {
		return trimmed[:80] + "…"
	}
	if trimmed == "" {
		return "该文件"
	}
	return trimmed
}

// mediaTypeName 把消息类型映射成人话，用于"引用了富媒体但读不了"的提示语。
func mediaTypeName(t string) string {
	switch t {
	case "image":
		return "图片"
	case "video":
		return "视频"
	case "audio":
		return "语音"
	default:
		return "消息"
	}
}

// truncateQuotedFileText 截断被引用文件正文，避免大文件撑爆 prompt（约 4000 字符/1-2k token）。
func truncateQuotedFileText(text string) string {
	const maxRun = 4000
	if len(text) <= maxRun {
		return text
	}
	return text[:maxRun] + "\n……（内容过长已截断）"
}

// buildContextBlocks 构造知识库/记忆/被引用对象的上下文注入消息块。
// 纯函数：仅依赖 input 字段，不访问 DB，可独立单测。
// 被引用对象按 Quoted.Kind 区分"成功读到"(File/Image)与"读取失败"(Failed)两种语义，
// 各自搭配对应的 assistant 应答话术，避免失败时误称"已读到文件内容"。
func buildContextBlocks(input *SmartReplyContext) []*schema.Message {
	var result []*schema.Message

	if input.KnowledgeCtx != "" {
		result = append(result, &schema.Message{
			Role:    schema.User,
			Content: fmt.Sprintf("【知识库参考】\n%s", input.KnowledgeCtx),
		})
		result = append(result, &schema.Message{
			Role:    schema.Assistant,
			Content: "收到知识库信息，我将优先参考这些内容来回答。",
		})
	}

	if input.MemoryCtx != "" {
		result = append(result, &schema.Message{
			Role:    schema.User,
			Content: input.MemoryCtx,
		})
		result = append(result, &schema.Message{
			Role:    schema.Assistant,
			Content: "我记住了这些历史信息。",
		})
	}

	if input.Quoted != nil {
		switch input.Quoted.Kind {
		case QuotedFile, QuotedImage:
			// 成功读到被引用对象（文件正文 / 图片）：注入内容并确认已读到
			result = append(result, buildQuotedContextMessage(input.Quoted))
			result = append(result, &schema.Message{
				Role:    schema.Assistant,
				Content: "我已读到被引用的文件内容。",
			})
		case QuotedFailed:
			// 读不到：如实告知，不假装读到
			result = append(result, &schema.Message{
				Role:    schema.User,
				Content: input.Quoted.Text,
			})
			result = append(result, &schema.Message{
				Role:    schema.Assistant,
				Content: "我未能读取该文件，将如实向用户说明原因。",
			})
		}
	}

	return result
}

// buildQuotedContextMessage 组装被引用对象（文件正文 / 图片）的用户消息块。
// 图片场景下同时注入 prompt 文本与 base64 data URL（经 MultiContent 携带，供
// einoMessagesToAIMessages 提取为 ai.Message.ImageURL 交给多模态模型识别）。
func buildQuotedContextMessage(q *QuotedContext) *schema.Message {
	if q.ImageURL == "" {
		return &schema.Message{Role: schema.User, Content: q.Text}
	}
	return &schema.Message{
		Role:    schema.User,
		Content: q.Text,
		MultiContent: []schema.ChatMessagePart{
			{Type: schema.ChatMessagePartTypeText, Text: q.Text},
			{
				Type: schema.ChatMessagePartTypeImageURL,
				ImageURL: &schema.ChatMessageImageURL{
					URL: q.ImageURL,
				},
			},
		},
	}
}

func (g *SmartReplyGraph) buildHistoryMessages(input *SmartReplyContext) []*schema.Message {
	var result []*schema.Message

	systemPrompt := g.buildSystemPrompt(input)
	result = append(result, &schema.Message{Role: schema.System, Content: systemPrompt})

	// 上下文注入段（知识库/记忆/被引用文件）：与后续 DB 查询历史消息解耦，抽成纯函数以支持无 DB 单测。
	result = append(result, buildContextBlocks(input)...)

	db := database.GetDB()
	var messages []model.Message
	db.Where("conversation_id = ? AND type IN ?", input.ConversationID, []string{"text", "markdown"}).
		Preload("Sender").
		Order("created_at DESC").
		Limit(20).
		Find(&messages)

	logHistoryDiagnostics("群助手/buildHistory", input.ConversationID, messages, nil)

	if len(messages) == 0 {
		currentQuestion := input.Message
		if input.IsAIMention {
			currentQuestion = fmt.Sprintf("💬 请回答：%s", input.Message)
		}
		result = append(result, &schema.Message{Role: schema.User, Content: currentQuestion})
		return result
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	filteredMessages := make([]model.Message, 0, len(messages))
	for _, msg := range messages {
		if input.OriginalContent != "" && msg.SenderID == input.UserID && msg.Content == input.OriginalContent {
			continue
		}
		filteredMessages = append(filteredMessages, msg)
	}

	var foldedFar []string
	hasRecentAI := false
	for _, msg := range filteredMessages {
		senderName := msg.Sender.Nickname
		if senderName == "" {
			senderName = msg.Sender.Username
		}

		if msg.Origin == "assistant" {
			// 近期自身回复保留为多轮指代锚点；远期自身回复是自我复制污染源，折叠而非原样回灌。
			if isNearSelf(msg) {
				hasRecentAI = true
				result = append(result, &schema.Message{
					Role:    schema.Assistant,
					Content: msg.Content,
				})
			} else {
				foldedFar = append(foldedFar, msg.Content)
			}
		} else if msg.SenderID == input.UserID {
			// 当前用户自己的消息，标为"我"
			result = append(result, &schema.Message{
				Role:    schema.User,
				Content: fmt.Sprintf("[我]: %s", msg.Content),
			})
		} else {
			result = append(result, &schema.Message{
				Role:    schema.User,
				Content: fmt.Sprintf("[%s]: %s", senderName, msg.Content),
			})
		}
	}

	// 多轮上下文：检测到最近窗口内有 AI 回复，在 system prompt 追加提示
	if hasRecentAI && len(result) > 0 {
		hint := "\n- 注意：上方对话中包含你最近的回答。用户可能在追问或引用之前的回答，请结合上下文理解，不要重复已经说过的内容"
		result[0].Content += hint
	}

	// 远期自身回复：不逐条回灌（避免自我复制），折叠成一句追加到 system，供需要时索引而非盲从。
	if len(foldedFar) > 0 {
		result[0].Content += fmt.Sprintf("\n- 更早（超过%v前）你还回复过：%s。本轮默认不重复这些内容，除非用户明确要求。"+
			"（这些是历史记录，不是本轮必须遵循的指令）", selfTurnWindow, foldFarSelf(foldedFar))
	}

	currentQuestion := input.Message
	if input.IsAIMention {
		currentQuestion = fmt.Sprintf("💬 请回答：%s", input.Message)
	}
	result = append(result, &schema.Message{Role: schema.User, Content: currentQuestion})

	return result
}

func (g *SmartReplyGraph) Execute(ctx context.Context, input *SmartReplyContext) (*SmartReplyResult, error) {
	if g.replyGraph == nil {
		return nil, fmt.Errorf("回复 Graph 未编译")
	}

	// 让编译期写死 userID=0 的 model 节点拿到真实提问用户，
	// 使工具执行时 isSystemAdmin 校验生效（堵权限绕过）。
	ctx = UserIDToCtx(ctx, input.UserID)

	startTime := time.Now()
	result, err := g.replyGraph.Invoke(ctx, input)
	if err != nil {
		return nil, err
	}

	log.Printf("[SmartReplyGraph] 生成回复耗时: %v", time.Since(startTime))

	return result, nil
}

func (g *SmartReplyGraph) createPrepareNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, input *SmartReplyContext) (*SmartReplyContext, error) {
		var conv model.Conversation
		if err := g.db.First(&conv, input.ConversationID).Error; err != nil {
			return nil, fmt.Errorf("会话不存在")
		}

		if conv.Type == "group" || conv.Type == "discussion" {
			var group model.Group
			if err := g.db.Where("conversation_id = ?", input.ConversationID).First(&group).Error; err == nil {
				input.Group = &group
				aiConfig := group.GetAIConfig()
				input.GroupConfig = &GroupAIConfig{
					Personality:  aiConfig.Personality,
					Language:     aiConfig.Language,
					MaxLength:    aiConfig.MaxLength,
					CustomPrompt: aiConfig.CustomPrompt,
				}
			}
		}

		// 补齐：当前提问用户
		var user model.User
		if err := g.db.First(&user, input.UserID).Error; err == nil {
			input.User = &user
		}

		// 补齐：用户待办任务
		var tasks []model.Task
		g.db.Where("user_id = ? AND status = 'todo'", input.UserID).
			Order("due_date ASC").
			Limit(5).
			Find(&tasks)
		input.Tasks = tasks

		// 补齐：群成员列表 + 群统计
		if input.Group != nil {
			var members []model.ConversationMember
			if err := g.db.Preload("User").Where("conversation_id = ?", input.ConversationID).Find(&members).Error; err == nil {
				names := make([]string, 0, len(members))
				for _, m := range members {
					name := m.User.Nickname
					if name == "" {
						name = m.User.Username
					}
					names = append(names, name)
				}
				input.MemberNames = strings.Join(names, "、")
			}

			var totalMessages int64
			g.db.Model(&model.Message{}).Where("conversation_id = ?", input.ConversationID).Count(&totalMessages)
			var memberCount int64
			g.db.Model(&model.ConversationMember{}).Where("conversation_id = ?", input.ConversationID).Count(&memberCount)
			input.GroupStats = fmt.Sprintf("总消息数：%d\n成员数：%d", totalMessages, memberCount)
		}

		return input, nil
	})
}

func (g *SmartReplyGraph) createKnowledgeNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, input *SmartReplyContext) (*SmartReplyContext, error) {
		content := ""

		if g.unifiedKnowledge != nil && input.Group != nil {
			// 群聊场景优先用用户消息检索，无消息时回退到群名
			query := input.Message
			if query == "" && input.Group.Name != "" {
				query = input.Group.Name
			}
			content = g.unifiedKnowledge.BuildContext(query, input.Group.ID)
		} else if g.legacyKnowledge != nil {
			content = g.legacyKnowledge.BuildKnowledgeContext(input.Message)
		}

		input.KnowledgeCtx = content
		return input, nil
	})
}

func (g *SmartReplyGraph) createMemoryNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, input *SmartReplyContext) (*SmartReplyContext, error) {
		input.MemoryCtx = g.recallGroupMemory(input)
		return input, nil
	})
}

// recallGroupMemory 召回本群群级记忆注入上下文。非群场景或无群记忆服务时置空。
// 切断旧路径：不再召回发送者分身记忆（AvatarMemoryService），避免私聊->群泄露。
func (g *SmartReplyGraph) recallGroupMemory(input *SmartReplyContext) string {
	if g.groupMemorySvc == nil || input.Group == nil {
		return ""
	}
	memoryResults, err := g.groupMemorySvc.Recall(input.Group.ID, input.Message, 2)
	if err != nil || len(memoryResults) == 0 {
		return ""
	}
	var parts []string
	for _, r := range memoryResults {
		parts = append(parts, r.Content)
	}
	return "💡 群聊记忆：\n" + strings.Join(parts, "\n")
}

func (g *SmartReplyGraph) createHistoryNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, input *SmartReplyContext) (*SmartReplyContext, error) {
		db := database.GetDB()
		var messages []model.Message
		db.Where("conversation_id = ?", input.ConversationID).
			Preload("Sender").
			Order("created_at DESC").
			Limit(20).
			Find(&messages)

		logHistoryDiagnostics("群助手/graph", input.ConversationID, messages, nil)

		if len(messages) == 0 {
			input.ChatHistory = ""
			return input, nil
		}

		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}

		var parts []string
		var foldedFar []string
		for _, msg := range messages {
			if input.OriginalContent != "" && msg.SenderID == input.UserID && msg.Content == input.OriginalContent {
				continue
			}

			senderName := msg.Sender.Nickname
			if senderName == "" {
				senderName = msg.Sender.Username
			}

			if msg.Origin == "assistant" {
				// 近期自身回复保留为多轮锚点；远期自身回复折叠，避免自我复制。
				if isNearSelf(msg) {
					parts = append(parts, fmt.Sprintf("[assistant]: %s", msg.Content))
				} else {
					foldedFar = append(foldedFar, msg.Content)
				}
			} else {
				parts = append(parts, fmt.Sprintf("[user:%s]: %s", senderName, msg.Content))
			}
		}

		if len(foldedFar) > 0 {
			parts = append(parts, fmt.Sprintf("[system-note]: 更早（超过%v前）你还回复过：%s。本轮默认不重复，除非用户明确要求。",
				selfTurnWindow, foldFarSelf(foldedFar)))
		}

		input.ChatHistory = strings.Join(parts, "\n")
		return input, nil
	})
}

func (g *SmartReplyGraph) createMergeNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, input *SmartReplyContext) (*SmartReplyContext, error) {
		return input, nil
	})
}

func (g *SmartReplyGraph) createBuildMessagesNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, input *SmartReplyContext) ([]*schema.Message, error) {
		var result []*schema.Message

		systemPrompt := g.buildSystemPrompt(input)
		result = append(result, &schema.Message{Role: schema.System, Content: systemPrompt})

		if input.KnowledgeCtx != "" {
			result = append(result, &schema.Message{
				Role:    schema.User,
				Content: fmt.Sprintf("【知识库参考】\n%s", input.KnowledgeCtx),
			})
			result = append(result, &schema.Message{
				Role:    schema.Assistant,
				Content: "收到知识库信息，我将优先参考这些内容来回答。",
			})
		}

		if input.MemoryCtx != "" {
			result = append(result, &schema.Message{
				Role:    schema.User,
				Content: input.MemoryCtx,
			})
			result = append(result, &schema.Message{
				Role:    schema.Assistant,
				Content: "我记住了这些历史信息。",
			})
		}

		if input.ChatHistory != "" {
			lines := strings.Split(input.ChatHistory, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if strings.HasPrefix(line, "[assistant]:") {
					content := strings.TrimPrefix(line, "[assistant]:")
					result = append(result, &schema.Message{Role: schema.Assistant, Content: strings.TrimSpace(content)})
				} else if strings.HasPrefix(line, "[user:") {
					idx := strings.Index(line, "]: ")
					if idx != -1 {
						content := line[idx+3:]
						result = append(result, &schema.Message{Role: schema.User, Content: strings.TrimSpace(content)})
					}
				} else {
					result = append(result, &schema.Message{Role: schema.User, Content: line})
				}
			}
		}

		currentQuestion := input.Message
		if input.IsAIMention {
			currentQuestion = fmt.Sprintf("💬 请回答：%s", input.Message)
		}
		result = append(result, &schema.Message{Role: schema.User, Content: currentQuestion})

		return result, nil
	})
}

func (g *SmartReplyGraph) createFormatReplyNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (*SmartReplyResult, error) {
		return &SmartReplyResult{
			Reply:    msg.Content,
			IsStream: false,
		}, nil
	})
}

func (g *SmartReplyGraph) buildSystemPrompt(input *SmartReplyContext) string {
	var sb strings.Builder

	// 1. 核心人设 (Personality)
	if input.GroupConfig != nil && input.GroupConfig.CustomPrompt != "" {
		sb.WriteString(input.GroupConfig.CustomPrompt + "\n\n")
	} else if input.GroupConfig != nil {
		sb.WriteString(aiprompt.BuildPersona(input.GroupConfig.Personality) + "\n\n")
	} else {
		// 默认人设（私聊场景）
		sb.WriteString(aiprompt.BuildPersona("") + "\n\n")
	}

	sb.WriteString(fmt.Sprintf("当前时间：%s (%s)\n\n", time.Now().Format("2006-01-02 15:04"), time.Now().Weekday().String()))

	// 产品基础知识：仅在用户问题与产品使用相关时注入，从 system_configs 读取
	if isProductQuestion(input.Message) {
		productKB := g.getProductKnowledge()
		if productKB != "" {
			sb.WriteString(productKB)
			sb.WriteString("\n\n")
		}
	}

	if input.IsAIMention {
		sb.WriteString(fmt.Sprintf("【触发方式】用户通过 @%s 直接向你提问，请直接回答问题。\n\n", input.AssistantName))
	}

	if input.Group != nil {
		sb.WriteString(fmt.Sprintf("📋 群组信息：\n- 群名：%s\n\n", input.Group.Name))

		if input.MemberNames != "" {
			sb.WriteString(fmt.Sprintf("- 群成员：%s\n\n", input.MemberNames))
		}

		if input.GroupStats != "" {
			sb.WriteString(fmt.Sprintf("📊 当前群状态：\n%s\n\n", input.GroupStats))
		}
	}

	// 当前提问用户
	if input.User != nil {
		sb.WriteString(fmt.Sprintf("👤 当前提问用户：%s\n\n", input.User.Nickname))
	}

	// 用户待办任务
	if len(input.Tasks) > 0 {
		sb.WriteString("📋 用户待办任务（未完成）：\n")
		for _, task := range input.Tasks {
			dueStr := "无截止日期"
			if task.DueDate != nil {
				dueStr = task.DueDate.Format("2006-01-02")
			}
			prio := task.Priority
			if prio == "" {
				prio = "medium"
			}
			sb.WriteString(fmt.Sprintf("- [%s] %s (截止: %s)\n", strings.ToUpper(prio[:1]), task.Title, dueStr))
		}
		sb.WriteString("\n")
	}

	if input.Intent != nil {
		sb.WriteString(fmt.Sprintf("【意图识别】类型: %s, 置信度: %.2f\n\n", input.Intent.Type, input.Intent.Confidence))
	}

	// 2. 回复规则
	sb.WriteString("【回复规则】\n")

	var rules []string
	if input.GroupConfig != nil {
		if langRule := aiprompt.LanguageRule(input.GroupConfig.Language); langRule != "" {
			rules = append(rules, langRule)
		}
		if lenRule := aiprompt.LengthRule(input.GroupConfig.MaxLength); lenRule != "" {
			rules = append(rules, lenRule)
		}
	} else {
		rules = append(rules, "请使用中文回答")
		rules = append(rules, "回答要简洁、专业、准确")
	}
	rules = append(rules, aiprompt.ReplyRules()...)

	for _, r := range rules {
		sb.WriteString("- " + r + "\n")
	}

	return sb.String()
}

// isProductQuestion 判断用户问题是否与 QIM 产品使用相关
func isProductQuestion(message string) bool {
	// 必须包含至少一个产品相关词
	productTerms := []string{
		"QIM", "qim",
		"群聊", "群组", "讨论组", "频道",
		"AI助手", "AI分身", "分身", "机器人",
		"笔记", "待办", "任务", "日历", "日程",
		"知识库", "文件管理",
		"搜索", "会话",
	}
	hasProductTerm := false
	msgLower := strings.ToLower(message)
	for _, term := range productTerms {
		if strings.Contains(msgLower, strings.ToLower(term)) {
			hasProductTerm = true
			break
		}
	}
	if !hasProductTerm {
		return false
	}
	// 包含产品词 + 操作性问题词
	actionWords := []string{
		"怎么", "如何", "在哪", "设置", "开启", "关闭",
		"创建", "添加", "删除", "配置", "使用",
		"可以", "功能", "有哪些",
	}
	for _, w := range actionWords {
		if strings.Contains(msgLower, strings.ToLower(w)) {
			return true
		}
	}
	return false
}

// 产品知识缓存（DB 配置变更后 5 分钟自动刷新）
var (
	productKBCache    string
	productKBCacheMu  sync.RWMutex
	productKBCacheExp time.Time
)

// getProductKnowledge 从 system_configs 读取产品知识（带 5 分钟缓存）
func (g *SmartReplyGraph) getProductKnowledge() string {
	productKBCacheMu.RLock()
	if time.Now().Before(productKBCacheExp) && productKBCache != "" {
		defer productKBCacheMu.RUnlock()
		return productKBCache
	}
	productKBCacheMu.RUnlock()

	productKBCacheMu.Lock()
	defer productKBCacheMu.Unlock()

	// double check
	if time.Now().Before(productKBCacheExp) && productKBCache != "" {
		return productKBCache
	}

	var cfg model.SystemConfig
	err := g.db.Where("config_key = ?", "ai_product_knowledge").First(&cfg).Error
	if err != nil || cfg.Value == "" {
		// DB 无配置时使用默认值
		productKBCache = defaultProductKnowledge()
	} else {
		productKBCache = cfg.Value
	}
	productKBCacheExp = time.Now().Add(5 * time.Minute)
	return productKBCache
}

func defaultProductKnowledge() string {
	return fmt.Sprintf(`【%s 产品知识】
%s 是企业即时通讯系统，核心功能如下：
- 单聊/群聊/讨论组：左侧会话列表点击进入，群聊支持 @成员、消息置顶、群公告
- AI 助手：在群聊中 @AI 或 @AI助手 提问；群聊还可配置关键词自动触发 AI 回复
- AI 分身：个人设置中开启，可配置触发模式（@触发/离线自动/关键词/全部消息/智能判断），AI 分身会以你的身份自动回复
- 知识库：群聊设置中上传文档，AI 回答时会优先参考知识库内容
- 笔记：左侧导航「笔记」，支持创建、编辑、搜索个人笔记
- 任务：左侧导航「任务」，可创建待办任务、设置截止日期和优先级
- 日历：左侧导航「日历」，管理日程安排
- 文件管理：左侧导航「文件」，查看和管理上传的文件
- 统一搜索：顶部搜索栏支持搜索消息、笔记、文件、知识库
当用户询问产品功能或使用方法时，根据以上信息引导用户操作。`, productname.Name, productname.Name)
}
