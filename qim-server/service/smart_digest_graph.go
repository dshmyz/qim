package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"qim-server/ai"
	"qim-server/database"
	"qim-server/model"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type DigestInput struct {
	UserID         uint
	ConversationID uint
	UnreadSince    *time.Time
}

type DigestOutput struct {
	GeneratedAt string           `json:"generated_at"`
	UnreadCount int              `json:"unread_count"`
	Categories  []DigestCategory `json:"categories"`
}

type DigestCategory struct {
	Type     string       `json:"type"`
	Priority string       `json:"priority"`
	Items    []DigestItem `json:"items"`
}

type DigestItem struct {
	Summary         string `json:"summary"`
	Sender          string `json:"sender,omitempty"`
	MessageID       uint   `json:"message_id,omitempty"`
	GroupName       string `json:"group_name,omitempty"`
	SuggestedAction string `json:"suggested_action,omitempty"`
}

type SmartDigestGraph struct {
	runnable  compose.Runnable[*DigestInput, *DigestOutput]
	aiService *ai.AIService
	cache     *AICache
}

var registerDigestMergeOnce sync.Once

func NewSmartDigestGraph(aiService *ai.AIService, cache *AICache) *SmartDigestGraph {
	return &SmartDigestGraph{
		aiService: aiService,
		cache:     cache,
	}
}

func (g *SmartDigestGraph) Build() error {
	registerDigestMergeOnce.Do(func() {
		compose.RegisterValuesMergeFunc(func(vs []*DigestInput) (*DigestInput, error) {
			return vs[0], nil
		})
	})

	graph := compose.NewGraph[*DigestInput, *DigestOutput]()

	graph.AddLambdaNode("prepare", g.createPrepareNode())
	graph.AddLambdaNode("build_messages", g.createBuildMessagesNode())
	graph.AddChatModelNode("model", NewEinoChatModel(g.aiService, ai.TaskTypeDigest, 0))
	graph.AddLambdaNode("validate", g.createValidateNode())
	graph.AddLambdaNode("format", g.createFormatNode())

	graph.AddEdge(compose.START, "prepare")
	graph.AddEdge("prepare", "build_messages")
	graph.AddEdge("build_messages", "model")
	graph.AddEdge("model", "validate")
	graph.AddEdge("validate", "format")
	graph.AddEdge("format", compose.END)

	ctx := context.Background()
	runnable, err := graph.Compile(ctx, compose.WithGraphName("SmartDigest"))
	if err != nil {
		return fmt.Errorf("编译 SmartDigest Graph 失败: %w", err)
	}
	g.runnable = runnable
	return nil
}

func (g *SmartDigestGraph) Execute(ctx context.Context, input *DigestInput) (*DigestOutput, error) {
	unreadStr := ""
	if input.UnreadSince != nil {
		unreadStr = input.UnreadSince.Format("20060102150405")
	}
	cacheKey := g.cache.GenerateKey("digest", fmt.Sprintf("%d", input.UserID), fmt.Sprintf("%d", input.ConversationID), unreadStr)
	if cached, ok := g.cache.Get(cacheKey); ok {
		var output DigestOutput
		if err := json.Unmarshal([]byte(cached), &output); err == nil {
			return &output, nil
		}
	}

	if g.runnable == nil {
		return nil, fmt.Errorf("SmartDigestGraph 未编译")
	}

	result, err := g.runnable.Invoke(ctx, input)
	if err != nil {
		return nil, err
	}

	data, _ := json.Marshal(result)
	g.cache.Set(cacheKey, string(data), time.Minute*30)
	return result, nil
}

type digestContext struct {
	input       *DigestInput
	messages    []model.Message
	unreadCount int
}

func (g *SmartDigestGraph) createPrepareNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, input *DigestInput) (*digestContext, error) {
		dc := &digestContext{
			input: input,
		}

		db := database.GetDB()
		var messages []model.Message
		query := db.Where("conversation_id = ?", input.ConversationID)

		if input.UnreadSince != nil {
			query = query.Where("created_at > ?", input.UnreadSince)
		}

		result := query.Preload("Sender").Order("created_at DESC").Limit(100).Find(&messages)
		if result.Error != nil {
			return nil, result.Error
		}

		dc.messages = messages
		dc.unreadCount = len(messages)

		return dc, nil
	})
}

func (g *SmartDigestGraph) createBuildMessagesNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, dc *digestContext) ([]*schema.Message, error) {
		var result []*schema.Message

		systemPrompt := g.buildDigestSystemPrompt(dc)
		result = append(result, &schema.Message{Role: schema.System, Content: systemPrompt})

		if len(dc.messages) == 0 {
			result = append(result, &schema.Message{
				Role:    schema.User,
				Content: "请生成该会话的未读消息摘要（无未读消息）。",
			})
			return result, nil
		}

		var conversationText strings.Builder
		conversationText.WriteString("以下是未读消息记录：\n\n")

		for _, msg := range dc.messages {
			senderName := msg.Sender.Nickname
			if senderName == "" {
				senderName = msg.Sender.Username
			}
			timestamp := msg.CreatedAt.Format("15:04")
			conversationText.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, senderName, msg.Content))
		}

		conversationText.WriteString("\n请为以上未读消息生成结构化的摘要。")
		result = append(result, &schema.Message{
			Role:    schema.User,
			Content: conversationText.String(),
		})

		return result, nil
	})
}

func (g *SmartDigestGraph) createValidateNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (*schema.Message, error) {
		if msg == nil {
			return nil, fmt.Errorf("模型返回空消息")
		}

		content := strings.TrimSpace(msg.Content)
		if content == "" {
			return &schema.Message{
				Role:    schema.Assistant,
				Content: "暂无未读消息。",
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

func (g *SmartDigestGraph) createFormatNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (*DigestOutput, error) {
		return &DigestOutput{
			GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
			UnreadCount: 0,
			Categories: []DigestCategory{
				{
					Type:     "summary",
					Priority: "medium",
					Items: []DigestItem{
						{
							Summary: msg.Content,
						},
					},
				},
			},
		}, nil
	})
}

func (g *SmartDigestGraph) buildDigestSystemPrompt(dc *digestContext) string {
	var sb strings.Builder

	sb.WriteString("你是 QIM 企业即时通讯系统的智能消息摘要助手。你的任务是为用户的未读消息生成简洁、结构化的摘要。\n\n")

	sb.WriteString("【摘要规则】\n")
	sb.WriteString("1. 按消息类型分类：@我的消息、与我相关的讨论、群聊热点话题、紧急事项\n")
	sb.WriteString("2. 提取每类消息的关键信息和决策\n")
	sb.WriteString("3. 识别需要回复或处理的事项\n")
	sb.WriteString("4. 使用简洁的语言，避免冗余\n")
	sb.WriteString("5. 如果涉及多个话题，使用列表形式组织\n")
	sb.WriteString("6. 保持客观，不要添加主观评价\n\n")

	sb.WriteString(fmt.Sprintf("【未读消息数量】%d 条\n", dc.unreadCount))

	return sb.String()
}
