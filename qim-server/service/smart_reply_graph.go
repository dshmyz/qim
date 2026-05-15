package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"qim-server/ai"
	"qim-server/database"
	"qim-server/model"

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

type GroupAIConfig struct {
	Personality  string
	Language     string
	MaxLength    string
	CustomPrompt string
	MembersList  string
	Stats        string
}

type SmartReplyContext struct {
	Message         string
	OriginalContent string
	UserID          uint
	ConversationID  uint
	IsAIMention     bool
	AssistantName   string
	Intent          *ai.MessageIntent
	KnowledgeCtx    string
	MemoryCtx       string
	ChatHistory     string
	PendingTasks    string
	Group           *model.Group
	GroupConfig     *GroupAIConfig
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
	EnsureMergeRegistered(func(vs []*SmartReplyContext) (*SmartReplyContext, error) {
		return vs[0], nil
	})

	graph := compose.NewGraph[*SmartReplyContext, *SmartReplyResult]()

	graph.AddLambdaNode("prepare", compose.InvokableLambda(g.prepare))
	graph.AddLambdaNode("knowledge", compose.InvokableLambda(g.retrieveKnowledge))
	graph.AddLambdaNode("memory", compose.InvokableLambda(g.retrieveMemory))
	graph.AddLambdaNode("history", compose.InvokableLambda(g.retrieveHistory))
	graph.AddLambdaNode("merge", compose.InvokableLambda(g.merge))
	graph.AddLambdaNode("build_messages", compose.InvokableLambda(g.buildMessages))
	graph.AddChatModelNode("model", NewEinoChatModel(g.aiService))
	graph.AddLambdaNode("format", compose.InvokableLambda(g.formatReply))

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

	runnable, err := compileGraph(graph, "SmartReply")
	if err != nil {
		return err
	}
	g.replyGraph = runnable
	return nil
}

func (g *SmartReplyGraph) ExecuteStream(ctx context.Context, input *SmartReplyContext) (*schema.StreamReader[*schema.Message], error) {
	// Use the same preparation pipeline as the Graph
	prepared, err := g.prepare(ctx, input)
	if err != nil {
		return nil, err
	}

	// Use the same knowledge/memory retrieval
	prepared.KnowledgeCtx = g.knowledgeContent(prepared)
	prepared.MemoryCtx = g.memoryContent(prepared)

	// Build messages using the proper DB-based approach
	systemUserID := uint(0)
	if g.userSvc != nil {
		systemUserID = g.userSvc.GetSystemUserID()
	}
	messages := g.buildHistoryMessages(prepared, systemUserID)

	// Set CallerContext with proper GroupID
	callerCtx := &ai.CallerContext{UserID: prepared.UserID}
	if prepared.Group != nil {
		callerCtx.GroupID = prepared.Group.ID
	}
	ctx = WithCallerContext(ctx, callerCtx)

	// Stream through model
	chatModel := NewEinoChatModel(g.aiService)
	return chatModel.Stream(ctx, messages)
}

func (g *SmartReplyGraph) Execute(ctx context.Context, input *SmartReplyContext) (*SmartReplyResult, error) {
	if g.replyGraph == nil {
		return nil, fmt.Errorf("回复 Graph 未编译")
	}
	callerCtx := &ai.CallerContext{UserID: input.UserID}
	if input.Group != nil {
		callerCtx.GroupID = input.Group.ID
	}
	ctx = WithCallerContext(ctx, callerCtx)
	startTime := time.Now()
	result, err := g.replyGraph.Invoke(ctx, input)
	if err != nil {
		return nil, err
	}
	log.Printf("[SmartReplyGraph] 生成回复耗时: %v", time.Since(startTime))
	return result, nil
}

func (g *SmartReplyGraph) prepare(ctx context.Context, input *SmartReplyContext) (*SmartReplyContext, error) {
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
	return input, nil
}

func (g *SmartReplyGraph) retrieveKnowledge(ctx context.Context, input *SmartReplyContext) (*SmartReplyContext, error) {
	input.KnowledgeCtx = g.knowledgeContent(input)
	return input, nil
}

func (g *SmartReplyGraph) knowledgeContent(input *SmartReplyContext) string {
	if g.unifiedKnowledge != nil && input.Group != nil {
		return g.unifiedKnowledge.BuildContext(input.Message, input.Group.ID)
	}
	if g.legacyKnowledge != nil {
		return g.legacyKnowledge.BuildKnowledgeContext(input.Message)
	}
	return ""
}

func (g *SmartReplyGraph) retrieveMemory(ctx context.Context, input *SmartReplyContext) (*SmartReplyContext, error) {
	input.MemoryCtx = g.memoryContent(input)
	return input, nil
}

func (g *SmartReplyGraph) memoryContent(input *SmartReplyContext) string {
	if g.memorySvc == nil {
		return ""
	}
	memoryResults, err := g.memorySvc.Recall(input.UserID, input.Message, 2)
	if err != nil || len(memoryResults) == 0 {
		return ""
	}
	var parts []string
	for _, r := range memoryResults {
		parts = append(parts, r.Content)
	}
	return "💡 用户历史记忆：\n" + strings.Join(parts, "\n")
}

func (g *SmartReplyGraph) retrieveHistory(ctx context.Context, input *SmartReplyContext) (*SmartReplyContext, error) {
	db := database.GetDB()
	var messages []model.Message
	db.Where("conversation_id = ?", input.ConversationID).Preload("Sender").
		Order("created_at DESC").Limit(20).Find(&messages)

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
}

func (g *SmartReplyGraph) merge(ctx context.Context, input *SmartReplyContext) (*SmartReplyContext, error) {
	return input, nil
}

func (g *SmartReplyGraph) buildMessages(ctx context.Context, input *SmartReplyContext) ([]*schema.Message, error) {
	var result []*schema.Message
	result = append(result, &schema.Message{Role: schema.System, Content: g.buildSystemPrompt(input)})

	if input.KnowledgeCtx != "" {
		result = append(result, &schema.Message{Role: schema.User, Content: fmt.Sprintf("【知识库参考】\n%s", input.KnowledgeCtx)})
		result = append(result, &schema.Message{Role: schema.Assistant, Content: "收到知识库信息，我将优先参考这些内容来回答。"})
	}
	if input.MemoryCtx != "" {
		result = append(result, &schema.Message{Role: schema.User, Content: input.MemoryCtx})
		result = append(result, &schema.Message{Role: schema.Assistant, Content: "我记住了这些历史信息。"})
	}
	if input.ChatHistory != "" {
		for _, line := range strings.Split(input.ChatHistory, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "[assistant]:") {
				result = append(result, &schema.Message{Role: schema.Assistant, Content: strings.TrimSpace(strings.TrimPrefix(line, "[assistant]:"))})
			} else if strings.HasPrefix(line, "[user:") {
				idx := strings.Index(line, "]: ")
				if idx != -1 {
					result = append(result, &schema.Message{Role: schema.User, Content: strings.TrimSpace(line[idx+3:])})
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
}

func (g *SmartReplyGraph) formatReply(ctx context.Context, msg *schema.Message) (*SmartReplyResult, error) {
	return &SmartReplyResult{Reply: msg.Content, IsStream: false}, nil
}

func (g *SmartReplyGraph) buildHistoryMessages(input *SmartReplyContext, systemUserID uint) []*schema.Message {
	var result []*schema.Message
	result = append(result, &schema.Message{Role: schema.System, Content: g.buildSystemPrompt(input)})

	if input.KnowledgeCtx != "" {
		result = append(result, &schema.Message{Role: schema.User, Content: fmt.Sprintf("【知识库参考】\n%s", input.KnowledgeCtx)})
		result = append(result, &schema.Message{Role: schema.Assistant, Content: "收到知识库信息，我将优先参考这些内容来回答。"})
	}
	if input.MemoryCtx != "" {
		result = append(result, &schema.Message{Role: schema.User, Content: input.MemoryCtx})
		result = append(result, &schema.Message{Role: schema.Assistant, Content: "我记住了这些历史信息。"})
	}

	db := database.GetDB()
	var messages []model.Message
	db.Where("conversation_id = ?", input.ConversationID).Preload("Sender").
		Order("created_at DESC").Limit(20).Find(&messages)

	if len(messages) == 0 {
		q := input.Message
		if input.IsAIMention {
			q = fmt.Sprintf("💬 请回答：%s", input.Message)
		}
		return append(result, &schema.Message{Role: schema.User, Content: q})
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	for _, msg := range messages {
		if input.OriginalContent != "" && msg.SenderID == input.UserID && msg.Content == input.OriginalContent {
			continue
		}
		senderName := msg.Sender.Nickname
		if senderName == "" {
			senderName = msg.Sender.Username
		}
		if msg.SenderID == systemUserID {
			result = append(result, &schema.Message{Role: schema.Assistant, Content: msg.Content})
		} else {
			result = append(result, &schema.Message{Role: schema.User, Content: fmt.Sprintf("[%s]: %s", senderName, msg.Content)})
		}
	}

	q := input.Message
	if input.IsAIMention {
		q = fmt.Sprintf("💬 请回答：%s", input.Message)
	}
	return append(result, &schema.Message{Role: schema.User, Content: q})
}

func (g *SmartReplyGraph) buildSystemPrompt(input *SmartReplyContext) string {
	pb := NewPromptBuilder(g.defaultRole(input.GroupConfig))

	if input.GroupConfig != nil && input.GroupConfig.CustomPrompt != "" {
		pb = NewPromptBuilder(input.GroupConfig.CustomPrompt)
	}

	pb.SetParam("当前时间", time.Now().Format("2006-01-02 15:04"))

	if input.IsAIMention {
		pb.SetParam("触发方式", fmt.Sprintf("用户通过 @%s 直接向你提问，请直接回答问题", input.AssistantName))
	}
	if input.Group != nil {
		pb.SetParam("群聊信息", fmt.Sprintf("群名：%s", input.Group.Name))
	}
	if input.Intent != nil {
		pb.SetParam("意图识别", fmt.Sprintf("类型: %s, 置信度: %.2f", input.Intent.Type, input.Intent.Confidence))
	}

	pb.AddRules(
		"- 优先使用知识库中的内容回答",
		"- 如果知识库中没有相关内容，使用你的通用知识回答，但明确说明\"以下回答基于通用知识，建议核实\"",
	)

	if input.GroupConfig != nil && input.GroupConfig.Language == "en" {
		pb.AddRule("- Please answer in English")
	} else {
		pb.AddRule("- 请使用中文回答")
	}

	if input.GroupConfig != nil {
		switch input.GroupConfig.MaxLength {
		case "short":
			pb.AddRule("- 回答要简短，控制在50字以内")
		case "medium":
			pb.AddRule("- 回答适中，控制在150字以内")
		case "long":
			pb.AddRule("- 回答详细，可以展开说明")
		}
	} else {
		pb.AddRule("- 回答要简洁、专业、准确")
	}

	return pb.BuildSystem()
}

func (g *SmartReplyGraph) defaultRole(cfg *GroupAIConfig) string {
	if cfg == nil {
		return "你是 QIM 企业即时通讯系统中的智能助手，风格专业严谨。回答要专业、客观、有条理。"
	}
	switch cfg.Personality {
	case "casual":
		return "你是 QIM 企业即时通讯系统中的 AI 助手，性格轻松幽默。在回答中可以适当使用表情和emoji，语气活泼。"
	case "concise":
		return "你是 QIM 企业即时通讯系统中的 AI 助手，风格简洁高效。回答直奔主题，不废话，只说重点。"
	case "friendly":
		return "你是 QIM 企业即时通讯系统中的 AI 助手，性格温暖亲切。回答要有耐心，语气友善，像一个贴心的伙伴。"
	case "technical":
		return "你是 QIM 企业即时通讯系统中的技术专家 AI 助手。回答要有技术深度，关注细节，必要时提供代码示例和技术方案。"
	default:
		return "你是 QIM 企业即时通讯系统中的智能助手，风格专业严谨。回答要专业、客观、有条理。"
	}
}
