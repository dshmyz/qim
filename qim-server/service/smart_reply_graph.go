package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"

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

// GroupAIConfig 群组专属 AI 配置（非群聊场景下为 nil）
type GroupAIConfig struct {
	Personality  string // casual, concise, friendly, technical
	Language     string // zh, en
	MaxLength    string // short, medium, long
	CustomPrompt string // 自定义提示词
	MembersList  string // 群成员名单
	Stats        string // 群统计数据
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
	memorySvc        *AvatarMemoryService
	userSvc          *UserService
}

func NewSmartReplyGraph(
	aiService *ai.AIService,
	db *gorm.DB,
	unifiedKnowledge KnowledgeRetriever,
	legacyKnowledge LegacyKnowledgeService,
	memorySvc *AvatarMemoryService,
	userSvc *UserService,
) *SmartReplyGraph {
	return &SmartReplyGraph{
		aiService:        aiService,
		db:               db,
		unifiedKnowledge: unifiedKnowledge,
		legacyKnowledge:  legacyKnowledge,
		memorySvc:        memorySvc,
		userSvc:          userSvc,
	}
}

func (g *SmartReplyGraph) BuildGraph() error {
	if err := g.buildReplyGraph(); err != nil {
		return fmt.Errorf("构建回复 Graph 失败: %w", err)
	}
	return nil
}

func (g *SmartReplyGraph) buildReplyGraph() error {
	compose.RegisterValuesMergeFunc(func(vs []*SmartReplyContext) (*SmartReplyContext, error) {
		return vs[0], nil
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
	if g.memorySvc != nil {
		memoryResults, err := g.memorySvc.Recall(input.UserID, input.Message, 2)
		if err == nil && len(memoryResults) > 0 {
			var parts []string
			for _, r := range memoryResults {
				parts = append(parts, r.Content)
			}
			memoryCtx = "💡 用户历史记忆：\n" + strings.Join(parts, "\n")
		}
	}
	input.MemoryCtx = memoryCtx

	historyMessages := g.buildHistoryMessages(input)

	chatModel := NewEinoChatModel(g.aiService, ai.TaskTypeChat, input.UserID)
	return chatModel.Stream(ctx, historyMessages)
}

func (g *SmartReplyGraph) buildHistoryMessages(input *SmartReplyContext) []*schema.Message {
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

	db := database.GetDB()
	var messages []model.Message
	db.Where("conversation_id = ?", input.ConversationID).
		Preload("Sender").
		Order("created_at DESC").
		Limit(20).
		Find(&messages)

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

	for _, msg := range filteredMessages {
		senderName := msg.Sender.Nickname
		if senderName == "" {
			senderName = msg.Sender.Username
		}

		if msg.Origin == "assistant" {
			result = append(result, &schema.Message{
				Role:    schema.Assistant,
				Content: msg.Content,
			})
		} else {
			result = append(result, &schema.Message{
				Role:    schema.User,
				Content: fmt.Sprintf("[%s]: %s", senderName, msg.Content),
			})
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
		if g.memorySvc == nil {
			input.MemoryCtx = ""
			return input, nil
		}

		memoryResults, err := g.memorySvc.Recall(input.UserID, input.Message, 2)
		if err != nil || len(memoryResults) == 0 {
			input.MemoryCtx = ""
			return input, nil
		}

		var parts []string
		for _, r := range memoryResults {
			parts = append(parts, r.Content)
		}
		input.MemoryCtx = "💡 用户历史记忆：\n" + strings.Join(parts, "\n")
		return input, nil
	})
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

		if len(messages) == 0 {
			input.ChatHistory = ""
			return input, nil
		}

		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}

		var parts []string
		for _, msg := range messages {
			if input.OriginalContent != "" && msg.SenderID == input.UserID && msg.Content == input.OriginalContent {
				continue
			}

			senderName := msg.Sender.Nickname
			if senderName == "" {
				senderName = msg.Sender.Username
			}

			if msg.Origin == "assistant" {
				parts = append(parts, fmt.Sprintf("[assistant]: %s", msg.Content))
			} else {
				parts = append(parts, fmt.Sprintf("[user:%s]: %s", senderName, msg.Content))
			}
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
		switch input.GroupConfig.Personality {
		case "casual":
			sb.WriteString("你是 QIM 企业即时通讯系统中的 AI 助手，性格轻松幽默。在回答中可以适当使用表情和emoji，语气活泼。\n\n")
		case "concise":
			sb.WriteString("你是 QIM 企业即时通讯系统中的 AI 助手，风格简洁高效。回答直奔主题，不废话，只说重点。\n\n")
		case "friendly":
			sb.WriteString("你是 QIM 企业即时通讯系统中的 AI 助手，性格温暖亲切。回答要有耐心，语气友善，像一个贴心的伙伴。\n\n")
		case "technical":
			sb.WriteString("你是 QIM 企业即时通讯系统中的技术专家 AI 助手。回答要有技术深度，关注细节，必要时提供代码示例和技术方案。\n\n")
		default:
			sb.WriteString("你是 QIM 企业即时通讯系统中的智能助手，风格专业严谨。回答要专业、客观、有条理。\n\n")
		}
	} else {
		// 默认人设（私聊场景）
		sb.WriteString("你是 QIM 企业即时通讯系统中的智能助手，风格专业严谨。回答要专业、客观、有条理。\n\n")
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

	if input.GroupConfig != nil && input.GroupConfig.Language == "en" {
		sb.WriteString("- Please answer in English\n")
	} else {
		sb.WriteString("- 请使用中文回答\n")
	}

	if input.GroupConfig != nil {
		switch input.GroupConfig.MaxLength {
		case "short":
			sb.WriteString("- 回答要简短，控制在50字以内\n")
		case "medium":
			sb.WriteString("- 回答适中，控制在150字以内\n")
		case "long":
			sb.WriteString("- 回答详细，可以展开说明\n")
		}
	} else {
		sb.WriteString("- 回答要简洁、专业、准确\n")
	}

	sb.WriteString("- 优先使用知识库中的内容回答\n")
	sb.WriteString("- 如果知识库中没有相关内容，使用你的通用知识回答，但明确说明\"以下回答基于通用知识，建议核实\"\n")

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
	return `【QIM 产品知识】
QIM 是企业即时通讯系统，核心功能如下：
- 单聊/群聊/讨论组：左侧会话列表点击进入，群聊支持 @成员、消息置顶、群公告
- AI 助手：在群聊中 @AI 或 @AI助手 提问；群聊还可配置关键词自动触发 AI 回复
- AI 分身：个人设置中开启，可配置触发模式（@触发/离线自动/关键词/全部消息/智能判断），AI 分身会以你的身份自动回复
- 知识库：群聊设置中上传文档，AI 回答时会优先参考知识库内容
- 笔记：左侧导航「笔记」，支持创建、编辑、搜索个人笔记
- 任务：左侧导航「任务」，可创建待办任务、设置截止日期和优先级
- 日历：左侧导航「日历」，管理日程安排
- 文件管理：左侧导航「文件」，查看和管理上传的文件
- 统一搜索：顶部搜索栏支持搜索消息、笔记、文件、知识库
当用户询问产品功能或使用方法时，根据以上信息引导用户操作。`
}
