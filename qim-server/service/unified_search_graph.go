package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/pkg/aiprompt"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/productname"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type UnifiedSearchInput struct {
	Query          string
	ConversationID uint
	UserID         uint
	GroupID        uint
}

type UnifiedSearchOutput struct {
	Query   string
	Answer  string
	Sources []SearchSource
}

type SearchSource struct {
	Type      string  `json:"type"`
	Title     string  `json:"title,omitempty"`
	Content   string  `json:"content"`
	Relevance float64 `json:"relevance"`
	Sender    string  `json:"sender,omitempty"`
	Date      string  `json:"date,omitempty"`
	// DocID 检索层文档 ID（如 msg_N）：跨渠道去重的稳定键；无 ID 的来源为空
	DocID string `json:"doc_id,omitempty"`
}

type retrieveResult struct {
	query   string
	sources []SearchSource
}

type UnifiedSearchGraph struct {
	runnable    compose.Runnable[*UnifiedSearchInput, *UnifiedSearchOutput]
	aiService   *ai.AIService
	noteSvc     *NoteVectorService
	groupDocSvc *GroupDocumentService
	memorySvc   *AvatarMemoryService
}

func NewUnifiedSearchGraph(
	aiService *ai.AIService,
	noteSvc *NoteVectorService,
	groupDocSvc *GroupDocumentService,
	memorySvc *AvatarMemoryService,
) *UnifiedSearchGraph {
	return &UnifiedSearchGraph{
		aiService:   aiService,
		noteSvc:     noteSvc,
		groupDocSvc: groupDocSvc,
		memorySvc:   memorySvc,
	}
}

var registerSearchMergeOnce sync.Once

func (g *UnifiedSearchGraph) Build() error {
	registerSearchMergeOnce.Do(func() {
		compose.RegisterValuesMergeFunc(func(vs []*retrieveResult) (*retrieveResult, error) {
			return vs[0], nil
		})
	})

	graph := compose.NewGraph[*UnifiedSearchInput, *UnifiedSearchOutput]()

	graph.AddLambdaNode("retrieve", g.createRetrieveNode())
	graph.AddLambdaNode("build_prompt", g.createBuildPromptNode())
	// 模型节点用 NoTools：检索已由 retrieve 节点多路完成后，本节点只负责综合成稿。
	// 带工具的内层循环是纯负债——allowed 为空会放行全量注册工具（含 search_knowledge
	// 自身的递归风险、summarize_conversation 等依赖会话上下文的工具），且内层任一工具
	// 失败会以 NodeRunError 炸掉整个 search_knowledge（2026-10-10 "技术交流群讨论内容"
	// 检索失败即此因：内层模型调 summarize_conversation 遇 convID=0）。
	graph.AddChatModelNode("model", NewEinoChatModelNoTools(g.aiService, ai.TaskTypeSearch, 0))
	graph.AddLambdaNode("format", g.createFormatNode())

	graph.AddEdge(compose.START, "retrieve")
	graph.AddEdge("retrieve", "build_prompt")
	graph.AddEdge("build_prompt", "model")
	graph.AddEdge("model", "format")
	graph.AddEdge("format", compose.END)

	ctx := context.Background()
	runnable, err := graph.Compile(ctx, compose.WithGraphName("UnifiedSearch"))
	if err != nil {
		return fmt.Errorf("编译 UnifiedSearch Graph 失败: %w", err)
	}
	g.runnable = runnable
	return nil
}

func (g *UnifiedSearchGraph) Execute(ctx context.Context, input *UnifiedSearchInput) (*UnifiedSearchOutput, error) {
	if g.runnable == nil {
		return nil, fmt.Errorf("UnifiedSearchGraph not built")
	}
	// 图内 model 节点编译期写死 userID=0，靠 UserIDToCtx 从 ctx 读取真实用户
	// （见 EinoChatModel.Generate）。这里统一从显式 input.UserID 注入一次，调用方
	// 无需知道该内部契约（模型节点现走 NoTools 纯生成，此注入保留为通用契约）。
	return g.runnable.Invoke(UserIDToCtx(ctx, input.UserID), input)
}

func (g *UnifiedSearchGraph) createRetrieveNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, input *UnifiedSearchInput) (*retrieveResult, error) {
		var wg sync.WaitGroup
		var mu sync.Mutex
		sources := []SearchSource{}

		wg.Add(4)

		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[UnifiedSearch] Panic in message retrieval: %v", r)
				}
			}()
			msgRetriever := NewMessageRetriever(input.ConversationID, input.UserID, 5)
			docs, err := msgRetriever.Retrieve(ctx, input.Query)
			if err == nil {
				mu.Lock()
				for _, doc := range docs {
					senderName := ""
					if v, ok := doc.MetaData["sender_name"]; ok {
						senderName = fmt.Sprintf("%v", v)
					}
					createdAt := ""
					if v, ok := doc.MetaData["created_at"]; ok {
						createdAt = fmt.Sprintf("%v", v)
					}
					sources = append(sources, SearchSource{
						Type:      "message",
						Content:   doc.Content,
						Relevance: 0.8,
						Sender:    senderName,
						Date:      createdAt,
						DocID:     doc.ID,
					})
				}
				mu.Unlock()
			}
		}()

		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[UnifiedSearch] Panic in note retrieval: %v", r)
				}
			}()
			if g.noteSvc == nil {
				return
			}
			noteRetriever := NewNoteRetriever(g.noteSvc, input.UserID, 5)
			docs, err := noteRetriever.Retrieve(ctx, input.Query)
			if err == nil {
				mu.Lock()
				for _, doc := range docs {
					score := 0.7
					if v, ok := doc.MetaData["score"]; ok {
						if f, ok := v.(float64); ok {
							score = f
						}
					}
					title := ""
					if v, ok := doc.MetaData["title"]; ok {
						title = fmt.Sprintf("%v", v)
					}
					sources = append(sources, SearchSource{
						Type:      "note",
						Title:     title,
						Content:   doc.Content,
						Relevance: score,
						DocID:     doc.ID,
					})
				}
				mu.Unlock()
			}
		}()

		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[UnifiedSearch] Panic in group document retrieval: %v", r)
				}
			}()
			if g.groupDocSvc == nil || input.GroupID == 0 {
				return
			}
			groupDocRetriever := NewGroupDocRetriever(g.groupDocSvc, input.GroupID, 5)
			docs, err := groupDocRetriever.Retrieve(ctx, input.Query)
			if err == nil {
				mu.Lock()
				for _, doc := range docs {
					score := 0.7
					if v, ok := doc.MetaData["score"]; ok {
						if f, ok := v.(float64); ok {
							score = f
						}
					}
					title := ""
					if v, ok := doc.MetaData["title"]; ok {
						title = fmt.Sprintf("%v", v)
					}
					sources = append(sources, SearchSource{
						Type:      "group_document",
						Title:     title,
						Content:   doc.Content,
						Relevance: score,
						DocID:     doc.ID,
					})
				}
				mu.Unlock()
			}
		}()

		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[UnifiedSearch] Panic in memory retrieval: %v", r)
				}
			}()
			if g.memorySvc == nil {
				return
			}
			memoryRetriever := NewMemoryRetriever(g.memorySvc, input.UserID, 3)
			docs, err := memoryRetriever.Retrieve(ctx, input.Query)
			if err == nil {
				mu.Lock()
				for _, doc := range docs {
					score := 0.6
					if v, ok := doc.MetaData["score"]; ok {
						if f, ok := v.(float64); ok {
							score = f
						}
					}
					sources = append(sources, SearchSource{
						Type:      "memory",
						Content:   doc.Content,
						Relevance: score,
						DocID:     doc.ID,
					})
				}
				mu.Unlock()
			}
		}()

		wg.Wait()
		// 跨渠道去重：同一内容可能被多路检索同时命中（同一消息既进 message 渠道又进
		// memory 渠道、或用户真的重复发过同一条消息）。重复来源会让下游模型误以为存在
		// 多条相同事实，反复对账重排——按 DocID 与归一化内容双重去重，保序留首个。
		deduped := dedupSources(sources)
		if dropped := len(sources) - len(deduped); dropped > 0 {
			logger.WithModule("UnifiedSearch").Info("检索来源去重",
				"before", len(sources), "after", len(deduped), "dropped", dropped)
		}
		sources = deduped
		return &retrieveResult{query: input.Query, sources: sources}, nil
	})
}

// dedupSources 对合并后的检索来源去重：DocID（稳定键）与归一化内容（跨渠道撞车、
// 用户重复发送的相同消息）任一命中即视为重复，保序保留首个。
func dedupSources(sources []SearchSource) []SearchSource {
	seenDoc := make(map[string]struct{}, len(sources))
	seenContent := make(map[string]struct{}, len(sources))
	result := make([]SearchSource, 0, len(sources))
	for _, src := range sources {
		if src.DocID != "" {
			if _, ok := seenDoc[src.DocID]; ok {
				continue
			}
			seenDoc[src.DocID] = struct{}{}
		}
		key := strings.TrimSpace(src.Content)
		if _, ok := seenContent[key]; ok {
			continue
		}
		if key != "" {
			seenContent[key] = struct{}{}
		}
		result = append(result, src)
	}
	return result
}

func (g *UnifiedSearchGraph) createBuildPromptNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, result *retrieveResult) ([]*schema.Message, error) {
		var messages []*schema.Message

		systemPrompt := fmt.Sprintf(`%s

你是 %s 企业即时通讯系统中的智能搜索助手。你的任务是根据检索到的多源信息，综合回答用户的问题。

【回答规则】
1. 优先使用检索到的信息回答问题
2. 如果多个来源有相关信息，综合整理后回答
3. 标注信息来源（消息、笔记、群文档、记忆）
4. 如果没有找到相关信息，诚实告知用户
5. 回答要简洁、专业、准确

【信息来源说明】
- message: 历史聊天消息
- note: 用户个人笔记
- group_document: 群文档知识库
- memory: 用户长期记忆`, aiprompt.CurrentTimeLine(), productname.Name)

		messages = append(messages, &schema.Message{Role: schema.System, Content: systemPrompt})

		if len(result.sources) > 0 {
			var contextBuilder strings.Builder
			contextBuilder.WriteString("【检索到的相关信息】\n\n")

			for i, src := range result.sources {
				contextBuilder.WriteString(fmt.Sprintf("--- 来源 %d: %s", i+1, getSourceTypeLabel(src.Type)))
				if src.Title != "" {
					contextBuilder.WriteString(fmt.Sprintf(" (%s)", src.Title))
				}
				if src.Sender != "" {
					contextBuilder.WriteString(fmt.Sprintf(" [发送者: %s]", src.Sender))
				}
				if src.Date != "" {
					contextBuilder.WriteString(fmt.Sprintf(" [时间: %s]", src.Date))
				}
				contextBuilder.WriteString(fmt.Sprintf(" [相关度: %.2f] ---\n", src.Relevance))
				contextBuilder.WriteString(src.Content)
				contextBuilder.WriteString("\n\n")
			}

			messages = append(messages, &schema.Message{
				Role:    schema.User,
				Content: contextBuilder.String(),
			})
			messages = append(messages, &schema.Message{
				Role:    schema.Assistant,
				Content: "我已了解检索到的相关信息，请继续提问。",
			})
		} else {
			messages = append(messages, &schema.Message{
				Role:    schema.User,
				Content: "没有检索到相关信息。",
			})
			messages = append(messages, &schema.Message{
				Role:    schema.Assistant,
				Content: "了解，我将基于通用知识回答。",
			})
		}

		messages = append(messages, &schema.Message{
			Role:    schema.User,
			Content: fmt.Sprintf("请回答：%s", result.query),
		})

		return messages, nil
	})
}

func (g *UnifiedSearchGraph) createFormatNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, input *schema.Message) (*UnifiedSearchOutput, error) {
		return &UnifiedSearchOutput{
			Answer: input.Content,
		}, nil
	})
}

func getSourceTypeLabel(typ string) string {
	switch typ {
	case "message":
		return "聊天消息"
	case "note":
		return "个人笔记"
	case "group_document":
		return "群文档"
	case "memory":
		return "历史记忆"
	default:
		return typ
	}
}
