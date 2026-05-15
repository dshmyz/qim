package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"qim-server/ai"
	"qim-server/database"
	"qim-server/model"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type SummaryInput struct {
	ConversationID uint
	TimeRange      string
	StartTime      *time.Time
	EndTime        *time.Time
	UserID         uint
}

type SummaryOutput struct {
	Summary       string
	MessagesCount int
	TimeRange     string
}

type SummaryGraph struct {
	runnable  compose.Runnable[*SummaryInput, *SummaryOutput]
	aiService *ai.AIService
	cache     *AICache
}

type summaryContext struct {
	input          *SummaryInput
	messages       []model.Message
	messagesCount  int
	timeRangeStart time.Time
	timeRangeEnd   time.Time
}

func NewSummaryGraph(aiService *ai.AIService, cache *AICache) *SummaryGraph {
	return &SummaryGraph{
		aiService: aiService,
		cache:     cache,
	}
}

func (g *SummaryGraph) Build() error {
	EnsureMergeRegistered(func(vs []*SummaryInput) (*SummaryInput, error) {
		return vs[0], nil
	})

	graph := compose.NewGraph[*SummaryInput, *SummaryOutput]()

	graph.AddLambdaNode("prepare", compose.InvokableLambda(g.prepare))
	graph.AddLambdaNode("build_messages", compose.InvokableLambda(g.buildMessages))
	AddModelNode(graph, g.aiService)
	graph.AddLambdaNode("validate", compose.InvokableLambda(g.validate))
	graph.AddLambdaNode("format", compose.InvokableLambda(g.format))

	graph.AddEdge(compose.START, "prepare")
	graph.AddEdge("prepare", "build_messages")
	graph.AddEdge("build_messages", "model")
	graph.AddEdge("model", "validate")
	graph.AddEdge("validate", "format")
	graph.AddEdge("format", compose.END)

	runnable, err := compileGraph(graph, "Summary")
	if err != nil {
		return err
	}
	g.runnable = runnable
	return nil
}

func (g *SummaryGraph) Execute(ctx context.Context, input *SummaryInput) (*SummaryOutput, error) {
	cacheKey := g.cache.GenerateKey("summary", fmt.Sprintf("%d", input.ConversationID), input.TimeRange)
	return executeWithCache(ctx, g.cache, cacheKey, time.Hour, g.runnable, input)
}

func (g *SummaryGraph) prepare(ctx context.Context, input *SummaryInput) (*summaryContext, error) {
	sc := &summaryContext{input: input}
	now := time.Now()
	switch input.TimeRange {
	case "1h":
		sc.timeRangeStart = now.Add(-time.Hour)
		sc.timeRangeEnd = now
	case "today":
		sc.timeRangeStart = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		sc.timeRangeEnd = now
	case "7d":
		sc.timeRangeStart = now.AddDate(0, 0, -7)
		sc.timeRangeEnd = now
	default:
		if input.StartTime != nil {
			sc.timeRangeStart = *input.StartTime
		} else {
			sc.timeRangeStart = now.AddDate(0, 0, -7)
		}
		if input.EndTime != nil {
			sc.timeRangeEnd = *input.EndTime
		} else {
			sc.timeRangeEnd = now
		}
	}

	db := database.GetDB()
	var messages []model.Message
	result := db.Where("conversation_id = ? AND created_at >= ? AND created_at <= ?",
		input.ConversationID, sc.timeRangeStart, sc.timeRangeEnd).
		Preload("Sender").Order("created_at ASC").Find(&messages)
	if result.Error != nil {
		return nil, result.Error
	}
	sc.messages = messages
	sc.messagesCount = len(messages)
	return sc, nil
}

func (g *SummaryGraph) buildMessages(ctx context.Context, sc *summaryContext) ([]*schema.Message, error) {
	systemPrompt := g.buildSummarySystemPrompt(sc)
	userContent := "请生成该时间段的会话摘要（无消息记录）。"
	if len(sc.messages) > 0 {
		var sb strings.Builder
		sb.WriteString("以下是需要摘要的对话记录：\n\n")
		for _, msg := range sc.messages {
			senderName := msg.Sender.Nickname
			if senderName == "" {
				senderName = msg.Sender.Username
			}
			sb.WriteString(fmt.Sprintf("[%s] %s: %s\n", msg.CreatedAt.Format("15:04"), senderName, msg.Content))
		}
		sb.WriteString("\n请为以上对话生成一份简洁的摘要。")
		userContent = sb.String()
	}
	return NewPromptBuilder(systemPrompt).ToMessages(userContent), nil
}

func (g *SummaryGraph) validate(ctx context.Context, msg *schema.Message) (*schema.Message, error) {
	if msg == nil {
		return nil, fmt.Errorf("模型返回空消息")
	}
	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return &schema.Message{Role: schema.Assistant, Content: "该时间段内暂无有效对话内容。"}, nil
	}
	if len(content) > 2000 {
		content = content[:2000] + "..."
	}
	return &schema.Message{Role: schema.Assistant, Content: content}, nil
}

func (g *SummaryGraph) format(ctx context.Context, msg *schema.Message) (*SummaryOutput, error) {
	return &SummaryOutput{Summary: msg.Content, TimeRange: ""}, nil
}

func (g *SummaryGraph) buildSummarySystemPrompt(sc *summaryContext) string {
	return NewPromptBuilder("你是 QIM 企业即时通讯系统的对话摘要助手。你的任务是为对话记录生成简洁、准确的摘要。").
		AddRules(
			"1. 提取对话中的关键信息和决策",
			"2. 识别讨论的主要话题",
			"3. 记录重要的结论或待办事项",
			"4. 使用简洁的语言，避免冗余",
			"5. 如果对话涉及多个话题，使用列表形式组织",
			"6. 保持客观，不要添加主观评价",
		).
		SetParam("时间范围", sc.timeRangeStart.Format("2006-01-02 15:04")+" 至 "+sc.timeRangeEnd.Format("2006-01-02 15:04")).
		SetParam("消息数量", fmt.Sprintf("%d 条", sc.messagesCount)).
		BuildSystem()
}
