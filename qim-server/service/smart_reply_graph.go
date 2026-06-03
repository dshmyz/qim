package service

import (
	"context"
	"fmt"
	"log"
	"strings"
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

	knowledgeCtx := ""
	if g.unifiedKnowledge != nil && input.Group != nil {
		knowledgeCtx = g.unifiedKnowledge.BuildContext(input.Message, input.Group.ID)
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

	systemUserID := uint(0)
	if g.userSvc != nil {
		systemUserID = g.userSvc.GetSystemUserID()
	}

	historyMessages := g.buildHistoryMessages(input, systemUserID)

	chatModel := NewEinoChatModel(g.aiService, ai.TaskTypeChat, input.UserID)
	return chatModel.Stream(ctx, historyMessages)
}

func (g *SmartReplyGraph) buildHistoryMessages(input *SmartReplyContext, systemUserID uint) []*schema.Message {
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

		if msg.SenderID == systemUserID {
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
			}
		}

		return input, nil
	})
}

func (g *SmartReplyGraph) createKnowledgeNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, input *SmartReplyContext) (*SmartReplyContext, error) {
		content := ""

		if g.unifiedKnowledge != nil && input.Group != nil {
			content = g.unifiedKnowledge.BuildContext(input.Message, input.Group.ID)
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

		systemUserID := uint(0)
		if g.userSvc != nil {
			systemUserID = g.userSvc.GetSystemUserID()
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

			if msg.SenderID == systemUserID {
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

	sb.WriteString(fmt.Sprintf("当前时间：%s\n\n", time.Now().Format("2006-01-02 15:04")))

	if input.IsAIMention {
		sb.WriteString(fmt.Sprintf("【触发方式】用户通过 @%s 直接向你提问，请直接回答问题。\n\n", input.AssistantName))
	}

	if input.Group != nil {
		sb.WriteString(fmt.Sprintf("【群聊信息】群名：%s\n\n", input.Group.Name))
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
