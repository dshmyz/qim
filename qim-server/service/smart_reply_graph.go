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
	// BuildContextWithSources 一次检索同时产出注入提示词的上下文串与命中的知识来源
	// （仅标题/相关度），供群助手把来源随回复下发。
	BuildContextWithSources(query string, groupID uint, limit int) (string, []KnowledgeSource)
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
	// KnowledgeSources 本条回复实际命中的知识来源（标题/相关度），由 prepareInput 在
	// 检索群知识库时填充；handler 执行完回复后据此把 knowledge_sources 写入消息 Extra，
	// 供前端渲染「知识来源」折叠标签。无命中时为空，不展示徽章。
	KnowledgeSources []KnowledgeSource

	// 扩展配置
	Group       *model.Group
	GroupConfig *GroupAIConfig

	// 补齐 Legacy 上下文
	User        *model.User  // 当前提问用户
	Tasks       []model.Task // 用户未完成任务
	MemberNames string       // 群成员名单（逗号分隔）
	GroupStats  string       // 群统计信息

	// HasTools 本次回复携带了可调用的工具（内置群管理工具或外部 MCP 工具）。
	// 由带工具路径在准备历史消息前置位，供 buildSystemPrompt 注入「能力边界」：
	// 有工具时声明可调用工具如实执行；无工具时要求模型诚实说明能力边界、不编造结果。
	// 纯流式/无工具路径保持 false（零值），完全不影响原有提示词上下文。
	HasTools bool

	// AllowedTools 本次回复可调用的工具名白名单（由 preparedHistory 按工具集填值）。
	// 供 buildSystemPrompt 用 ai.BuildCapabilityPrompt 动态注入真实能力自述；
	// 为空表示无工具（私人对话/无工具路径），此时只注入静态能力，不声称任何工具。
	AllowedTools []string
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

// externalToolOutputGuideMessage 外部工具 ReAct 路径追加的输出组织指引。
// 只作用于该路径，不写进全局 buildSystemPrompt（避免污染普通流式/管理指令路径）；
// 目标：把工具返回结果组织成自然简洁的中文段落，直接给答案数值，去掉工具原文前缀与客套尾巴。
func externalToolOutputGuideMessage() *schema.Message {
	return &schema.Message{
		Role: schema.System,
		Content: "当你调用外部工具得到结果后，用自然、简洁的中文把答案组织成一个连贯的回复。" +
			"直接把关键数值/结论给用户（如『3.5 × 7 = 24.5』），不要出现『计算结果』『工具返回』等字段式前缀，" +
			"不要罗列工具名或过程，" +
			"不要加『如需进一步查询，请随时告知』之类的客套结尾，" +
			"不要在回答开头再次称呼/点名提问用户（用户姓名已由界面单独展示）。",
	}
}

// SmartReplyToolset 群助手带工具回复时可用的工具集：内置群管理工具 或 外部 MCP 工具。
// 工具轴是调用方的真实选择（普通提问只用外部工具、管理指令只用内置工具），故作为参数传入；
// 流式/非流式因返回类型与降级语义不同，保留为两个独立方法。
type SmartReplyToolset int

const (
	// ToolsetBuiltin 内置群管理工具（groupAssistantAllowedTools）。
	ToolsetBuiltin SmartReplyToolset = iota
	// ToolsetExternal 外部 MCP 工具（mcpGateway.ListExternalToolNames），并追加输出组织指引。
	ToolsetExternal
)

// toolsetToolNames 返回指定工具集的白名单工具名。
func (g *SmartReplyGraph) toolsetToolNames(t SmartReplyToolset) []string {
	if t == ToolsetExternal {
		return g.mcpGateway.ListExternalToolNames()
	}
	return g.groupAssistantAllowedTools()
}

// ExecuteWithTools 带工具的非流式回复（管理指令/普通提问共用，工具集由 t 指定）。
// 走 GetCompletionWithToolsMultiStep 注入白名单 AI 工具并多步循环，LLM 返回 tool call 时
// 真实执行。callerCtx 用 input.UserID，isSystemAdmin 校验生效，即仅群主/管理员发起的
// 管理指令会被工具执行，普通成员指令被工具拒绝。
// 采用 MultiStep 而非单轮 core 的原因：工具执行出错时（如"用户不存在"）MultiStep 会把
// 错误以 tool 角色消息回喂给 LLM（见 ai_service.go 中 ReAct 循环），让群助手基于错误
// 生成自然回复，而不是像旧路径那样把错误硬抛到 handler 直接静默失败。
// feedback 为可选的每步工具执行回调（variadic，nil 即不回调），供调用方收集工具调用
// 记录做卡片展示；与外部工具路径共用同一 MultiStep 钩子（ai.ReActStepCallback）。
func (g *SmartReplyGraph) ExecuteWithTools(ctx context.Context, input *SmartReplyContext, t SmartReplyToolset, feedback ...ai.ReActStepCallback) (string, error) {
	historyMessages, err := g.preparedHistory(input, t)
	if err != nil {
		return "", err
	}
	callerCtx := &ai.CallerContext{UserID: input.UserID}
	var onStep ai.ReActStepCallback
	if len(feedback) > 0 {
		onStep = feedback[0]
	}
	return g.aiService.GetCompletionWithToolsMultiStep(
		ai.TaskTypeChat, einoMessagesToAIMessages(historyMessages), callerCtx, g.toolsetToolNames(t),
		ai.MaxReActSteps, onStep,
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

// ExecuteWithToolsStream 真·流式的带工具 ReAct（管理指令/普通提问共用，工具集由 t 指定），
// 与 ExecuteWithTools 同序（外部工具集含输出组织指引），final 答案逐 token 经 onChunk 流出。
// 返回 (streamed=true) 表示已走流式逐 token；若 Provider 不支持流式 tool-call 则返回
// (streamed=false, err=ai.ErrStreamingToolsNotSupported)，调用方降级到非流式 ExecuteWithTools。
// onStep 与 onChunk 均有意义：工具事件走 onStep（卡片），答案逐 token 走 onChunk。
func (g *SmartReplyGraph) ExecuteWithToolsStream(ctx context.Context, input *SmartReplyContext, t SmartReplyToolset, onStep ai.ReActStepCallback, onChunk func(chunk ai.StreamChunk) error) (streamed bool, err error) {
	historyMessages, err := g.preparedHistory(input, t)
	if err != nil {
		return false, err
	}
	callerCtx := &ai.CallerContext{UserID: input.UserID}
	err = g.aiService.GetCompletionWithToolsStreamMultiStep(
		ctx, ai.TaskTypeChat, einoMessagesToAIMessages(historyMessages), callerCtx, g.toolsetToolNames(t),
		ai.MaxReActSteps, onStep, onChunk,
	)
	if errors.Is(err, ai.ErrStreamingToolsNotSupported) {
		return false, ai.ErrStreamingToolsNotSupported
	}
	return err == nil, err
}

// preparedHistory 补齐输入上下文并构造历史消息；外部工具集追加输出组织指引
// （见 externalToolOutputGuideMessage）。两种流式/非流式路径共用同一准备骨架。
func (g *SmartReplyGraph) preparedHistory(input *SmartReplyContext, t SmartReplyToolset) ([]*schema.Message, error) {
	if err := g.prepareInput(input); err != nil {
		return nil, err
	}
	// 带工具路径：声明本次携带了工具，供 buildSystemPrompt 注入能力边界（有工具=如实执行；无工具=诚实说明）。
	input.HasTools = true
	input.AllowedTools = g.toolsetToolNames(t)
	history := g.buildHistoryMessages(input)
	if t == ToolsetExternal {
		history = append(history, externalToolOutputGuideMessage())
	}
	return history, nil
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
		// 一次检索同时产出上下文串与命中的知识来源（标题/相关度）；无命中时两者皆空。
		knowledgeCtx, input.KnowledgeSources = g.unifiedKnowledge.BuildContextWithSources(query, input.Group.ID, 3)
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

	// 历史范围标注：告知模型本次注入的对话历史条数与时间跨度，避免把陈旧信息当成现状。
	if len(filteredMessages) > 0 {
		var earliest, latest time.Time
		for _, m := range filteredMessages {
			if m.CreatedAt.IsZero() {
				continue
			}
			if earliest.IsZero() || m.CreatedAt.Before(earliest) {
				earliest = m.CreatedAt
			}
			if latest.IsZero() || m.CreatedAt.After(latest) {
				latest = m.CreatedAt
			}
		}
		if !earliest.IsZero() && !latest.IsZero() {
			span := earliest.Format("01-02 15:04") + " ~ " + latest.Format("01-02 15:04")
			if earliest.Equal(latest) {
				span = earliest.Format("2006-01-02 15:04")
			}
			result[0].Content += fmt.Sprintf(
				"\n- 本次注入的对话历史：共 %d 条，时间跨度 %s（截至 %s）。这是注入的上下文，不代表实时状态",
				len(filteredMessages), span, latest.Format("2006-01-02 15:04"))
		}
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
			// 一次检索同时产出上下文串与命中的知识来源（供自动回复路径把来源随回复下发）。
			content, input.KnowledgeSources = g.unifiedKnowledge.BuildContextWithSources(query, input.Group.ID, 3)
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

// capabilityPrompt 生成能力自述文本块（静态能力 + 本会话实际可调工具，动态）。
// 对 g.aiService / 其注册表为 nil 做安全降级（返回空串，不注入），避免单测或未初始化时 panic。
func (g *SmartReplyGraph) capabilityPrompt(allowed []string) string {
	if g.aiService == nil {
		return ""
	}
	return g.aiService.GetToolRegistry().BuildCapabilityPrompt(allowed)
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

	sb.WriteString(aiprompt.CurrentTimeLine() + "\n\n")

	// 产品基础知识：只要可用即注入，交给模型自行判断是否与当前问题相关（避免关键词启发式误判/漏判）。
	// 用「参考性」措辞框定，使无关闲聊时模型自然忽略，不喧宾夺主。
	productKB := g.getProductKnowledge()
	if productKB != "" {
		sb.WriteString("【产品使用参考】以下为系统已知的产品使用说明，仅当你需要回答产品功能/操作问题时才参考" +
			"，与当前问题无关时请忽略：\n\n")
		sb.WriteString(productKB)
		sb.WriteString("\n\n")
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

	// 主动性分层声明（与侧边栏「高主动/执行层」分工）：群聊回复对全群可见，
	// 定位为「建议层」——敢主动给建议/分析/提炼结论，但不擅自执行有副作用的操作；
	// 用户明确要求时才执行（如"帮我建任务"直接调用群待办工具），此时无需再反复确认，
	// 以免让本来合法的指令也变得畏手畏脚（与下方【边界与约束】的否定边界自洽）。
	sb.WriteString("- 主动性：你的回复会对全群成员可见。你可以主动给出建议、分析、提炼结论，" +
		"但未经用户明确要求，不要擅自执行有副作用的操作（创建任务、发送消息、删除或修改内容等）；" +
		"当用户明确提出要求（如『帮我创建群待办』）时，可直接执行对应工具完成任务，无需再回头征求确认。\n")

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
	// 输出格式：无论是否带工具，统一要求"直接给结论、聚焦要点、有出处就标注"。
	rules = append(rules,
		"直接给出结论，回答聚焦要点，避免客套的开场与结尾",
		"若引用了群文档、笔记或知识库内容，用自然语气简要标注来源",
	)

	for _, r := range rules {
		sb.WriteString("- " + r + "\n")
	}

	// 能力自述 + 能力边界。标题恒定存在（保证各分支结构一致）；能力自述依实际可调工具动态注入，
	// 让 AI 被问「具备哪些能力」时能如实回答，而非靠模型通用知识泛泛而谈；
	// 能力边界按 HasTools 分支声明（有工具=可如实调用；无工具=诚实说明，避免虚构执行）。
	sb.WriteString("\n【能力与工具】\n")
	if capPrompt := g.capabilityPrompt(input.AllowedTools); capPrompt != "" {
		sb.WriteString(capPrompt)
	}

	if input.HasTools {
		sb.WriteString("- 你可以调用系统提供的工具来获取实时数据或执行操作；调用前先确认工具是否适合当前问题\n")
		sb.WriteString("- 工具返回结果要如实使用；工具未返回或返回失败时，如实告知，不要编造一个成功的结果\n")
		sb.WriteString("- 没有一个工具能覆盖的问题，坦率说明做不到，不要假装已调用或虚拟出答案\n")
	} else {
		sb.WriteString("- 当前未提供实时查询或执行工具，你只能基于已注入的上下文和你自己的知识回答\n")
		sb.WriteString("- 遇到你无法得知的信息（如实时状态、他人私有数据、需要系统操作的请求），诚实说明你做不到，不要编造或猜测\n")
		sb.WriteString("- 不要假装你已经执行了某些操作（发消息、建日程、删记录等）；你没有执行能力\n")
	}

	// 边界与约束：IM 场景下的通用否定边界，避免虚构与过度动作。
	sb.WriteString("\n【边界与约束】\n")
	sb.WriteString("- 不编造事实：不要虚构用户没说过的话、群成员的态度、或历史中不存在的内容\n")
	sb.WriteString("- 不确定就明说：涉及你不确定的信息，直接说明存疑，不要含糊带过或强行给出确定结论\n")
	sb.WriteString("- 不做不可逆动作：未经明确要求，不代用户发布消息、删除/修改他人内容；需要操作时先说明打算怎么做\n")
	sb.WriteString("- 隐私克制：不要无谓地复述或外泄与当前问题无关的私密群消息细节\n")
	sb.WriteString("- 被引用的消息/文件属于注入上下文，仅作回答依据，不要原样抄录无关内容\n")

	return sb.String()
}

// 产品知识缓存（DB 配置变更后 5 分钟自动刷新）
var (
	productKBCache    string
	productKBCacheMu  sync.RWMutex
	productKBCacheExp time.Time
)

// getProductKnowledge 从 system_configs 读取产品知识（带 5 分钟缓存）
func (g *SmartReplyGraph) getProductKnowledge() string {
	// DB 未就绪（如纯提示词单测）时直接给默认知识，避免 nil 解引用。
	if g.db == nil {
		return defaultProductKnowledge()
	}

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
