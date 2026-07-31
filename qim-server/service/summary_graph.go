package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"

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
	GroupName     string
	ActiveMembers []string
}

type SummaryGraph struct {
	runnable  compose.Runnable[*SummaryInput, *SummaryOutput]
	aiService *ai.AIService
	cache     *AICache
}

var registerSummaryMergeOnce sync.Once

func NewSummaryGraph(aiService *ai.AIService, cache *AICache) *SummaryGraph {
	return &SummaryGraph{
		aiService: aiService,
		cache:     cache,
	}
}

func (g *SummaryGraph) Build() error {
	registerSummaryMergeOnce.Do(func() {
		compose.RegisterValuesMergeFunc(func(vs []*SummaryInput) (*SummaryInput, error) {
			return vs[0], nil
		})
	})

	graph := compose.NewGraph[*SummaryInput, *SummaryOutput]()

	graph.AddLambdaNode("prepare", g.createPrepareNode())
	graph.AddLambdaNode("build_messages", g.createBuildMessagesNode())
	graph.AddChatModelNode("model", NewEinoChatModel(g.aiService, ai.TaskTypeAnalysis, 0))
	graph.AddLambdaNode("validate", g.createValidateNode())
	graph.AddLambdaNode("format", g.createFormatNode())

	graph.AddEdge(compose.START, "prepare")
	graph.AddEdge("prepare", "build_messages")
	graph.AddEdge("build_messages", "model")
	graph.AddEdge("model", "validate")
	graph.AddEdge("validate", "format")
	graph.AddEdge("format", compose.END)

	ctx := context.Background()
	runnable, err := graph.Compile(ctx, compose.WithGraphName("Summary"))
	if err != nil {
		return fmt.Errorf("编译 Summary Graph 失败: %w", err)
	}
	g.runnable = runnable
	return nil
}

func (g *SummaryGraph) Execute(ctx context.Context, input *SummaryInput) (*SummaryOutput, error) {
	// 先查询一次元信息，用于构建与消息状态相关的缓存 key
	sc, err := g.prepareData(input)
	if err != nil {
		return nil, err
	}

	cacheKey := g.buildCacheKey(input, sc)
	if cached, ok := g.cache.Get(cacheKey); ok {
		return &SummaryOutput{
			Summary:       cached,
			TimeRange:     input.TimeRange,
			GroupName:     sc.groupName,
			ActiveMembers: sc.activeMembers,
		}, nil
	}

	if g.runnable == nil {
		return nil, fmt.Errorf("SummaryGraph 未编译")
	}

	result, err := g.runnable.Invoke(ctx, input)
	if err != nil {
		return nil, err
	}

	g.cache.Set(cacheKey, result.Summary, time.Hour)
	return result, nil
}

func (g *SummaryGraph) buildCacheKey(input *SummaryInput, sc *summaryContext) string {
	parts := []string{
		"summary",
		fmt.Sprintf("%d", input.ConversationID),
		input.TimeRange,
	}
	if input.StartTime != nil {
		parts = append(parts, fmt.Sprintf("%d", input.StartTime.Unix()))
	}
	if input.EndTime != nil {
		parts = append(parts, fmt.Sprintf("%d", input.EndTime.Unix()))
	}
	parts = append(parts, fmt.Sprintf("%d", sc.messagesCount))
	if sc.latestMessageAt != nil {
		parts = append(parts, fmt.Sprintf("%d", sc.latestMessageAt.Unix()))
	}
	return g.cache.GenerateKey(parts...)
}

func (g *SummaryGraph) prepareData(input *SummaryInput) (*summaryContext, error) {
	sc := &summaryContext{
		input: input,
	}

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

	// 加载会话元信息
	var group model.Group
	if err := db.Where("conversation_id = ?", input.ConversationID).First(&group).Error; err == nil {
		sc.groupName = group.Name
	}

	var messages []model.Message
	result := db.Where("conversation_id = ? AND created_at >= ? AND created_at <= ?",
		input.ConversationID, sc.timeRangeStart, sc.timeRangeEnd).
		Preload("Sender").
		Order("created_at ASC").
		Find(&messages)
	if result.Error != nil {
		return nil, result.Error
	}

	// 过滤：只保留文本和 Markdown 消息，排除撤回、空内容
	filtered := make([]model.Message, 0, len(messages))
	activeMap := make(map[string]bool)
	for _, msg := range messages {
		if msg.Type != "text" && msg.Type != "markdown" {
			continue
		}
		if msg.IsRecalled || strings.TrimSpace(msg.Content) == "" {
			continue
		}
		filtered = append(filtered, msg)

		name := msg.Sender.Nickname
		if name == "" {
			name = msg.Sender.Username
		}
		if name != "" {
			activeMap[name] = true
		}
	}

	// 限制消息数量，保留最新的 100 条
	const maxMessages = 100
	if len(filtered) > maxMessages {
		filtered = filtered[len(filtered)-maxMessages:]
	}

	for name := range activeMap {
		sc.activeMembers = append(sc.activeMembers, name)
	}
	sc.messages = filtered
	sc.messagesCount = len(filtered)
	sc.latestMessageAt = nil
	if len(filtered) > 0 {
		sc.latestMessageAt = &filtered[len(filtered)-1].CreatedAt
	}

	sort.Strings(sc.activeMembers)

	return sc, nil
}

type summaryContext struct {
	input           *SummaryInput
	messages        []model.Message
	messagesCount   int
	timeRangeStart  time.Time
	timeRangeEnd    time.Time
	latestMessageAt *time.Time
	groupName       string
	activeMembers   []string
}

func (g *SummaryGraph) createPrepareNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, input *SummaryInput) (*summaryContext, error) {
		return g.prepareData(input)
	})
}

func (g *SummaryGraph) createBuildMessagesNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, sc *summaryContext) ([]*schema.Message, error) {
		var result []*schema.Message

		systemPrompt := g.buildSummarySystemPrompt(sc)
		result = append(result, &schema.Message{Role: schema.System, Content: systemPrompt})

		if len(sc.messages) == 0 {
			result = append(result, &schema.Message{
				Role:    schema.User,
				Content: "请生成该时间段的会话摘要（无消息记录）。",
			})
			return result, nil
		}

		var conversationText strings.Builder
		conversationText.WriteString("以下是需要摘要的对话记录：\n\n")

		for _, msg := range sc.messages {
			senderName := msg.Sender.Nickname
			if senderName == "" {
				senderName = msg.Sender.Username
			}
			timestamp := msg.CreatedAt.Format("15:04")
			conversationText.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, senderName, msg.Content))
		}

		conversationText.WriteString("\n请严格按照上方结构化输出格式生成摘要。")
		result = append(result, &schema.Message{
			Role:    schema.User,
			Content: conversationText.String(),
		})

		return result, nil
	})
}

func (g *SummaryGraph) createValidateNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (*schema.Message, error) {
		if msg == nil {
			return nil, fmt.Errorf("模型返回空消息")
		}

		content := strings.TrimSpace(msg.Content)
		if content == "" {
			return &schema.Message{
				Role:    schema.Assistant,
				Content: "该时间段内暂无有效对话内容。",
			}, nil
		}

		if len(content) > 3000 {
			content = content[:3000] + "..."
		}

		return &schema.Message{
			Role:    schema.Assistant,
			Content: content,
		}, nil
	})
}

func (g *SummaryGraph) createFormatNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (*SummaryOutput, error) {
		return &SummaryOutput{
			Summary:   msg.Content,
			TimeRange: "",
		}, nil
	})
}

func (g *SummaryGraph) buildSummarySystemPrompt(sc *summaryContext) string {
	var sb strings.Builder

	sb.WriteString("你是 QIM 企业即时通讯系统的对话摘要助手。请根据对话记录生成一份结构化摘要。\n\n")

	sb.WriteString("【输出格式】\n")
	sb.WriteString("请严格按以下四个部分输出（每个部分如果无内容可写“无”）：\n\n")

	sb.WriteString("### 一、讨论要点\n")
	sb.WriteString("- 列出对话中讨论过的核心议题（不超过 5 条）\n")
	sb.WriteString("- 每条用 1-2 句话概括\n\n")

	sb.WriteString("### 二、决策结论\n")
	sb.WriteString("- 列出已达成共识或已作出的决策\n")
	sb.WriteString("- 标注作出该结论的主要依据\n\n")

	sb.WriteString("### 三、待办事项\n")
	sb.WriteString("- 列出明确的行动项，每条包含：事项描述、负责人、建议截止时间（如有）\n")
	sb.WriteString("- 如果负责人不明确，写“待确认”\n\n")

	sb.WriteString("### 四、参与者\n")
	sb.WriteString("- 列出活跃参与讨论的人员及各自主要观点\n\n")

	sb.WriteString("【摘要规则】\n")
	sb.WriteString("1. 提取关键信息、决策和待办事项\n")
	sb.WriteString("2. 识别主要讨论话题\n")
	sb.WriteString("3. 使用简洁、客观的语言，避免冗余\n")
	sb.WriteString("4. 保持事实准确，不要添加主观评价\n")
	sb.WriteString("5. 输出使用 Markdown 格式\n\n")

	if sc.groupName != "" {
		sb.WriteString(fmt.Sprintf("【会话名称】%s\n", sc.groupName))
	}
	sb.WriteString(fmt.Sprintf("【时间范围】%s 至 %s\n",
		sc.timeRangeStart.Format("2006-01-02 15:04"),
		sc.timeRangeEnd.Format("2006-01-02 15:04")))
	sb.WriteString(fmt.Sprintf("【消息数量】%d 条\n", sc.messagesCount))
	if len(sc.activeMembers) > 0 {
		sb.WriteString(fmt.Sprintf("【活跃参与者】%s\n", strings.Join(sc.activeMembers, ", ")))
	}

	return sb.String()
}
