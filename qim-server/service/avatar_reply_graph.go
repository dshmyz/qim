package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"gorm.io/gorm"
)

type AvatarReplyContext struct {
	Message        string
	ConversationID uint
	UserID         uint
	Config         model.AvatarConfig
	User           model.User
	KnowledgeScope model.AvatarKnowledgeScope
	ReplyStrategy  model.AvatarReplyStrategy
	NoteContext    string
	GroupKnowledge string
	MemoryContext  string
	History        string
}

type AvatarReplyGraph struct {
	runnable    compose.Runnable[*AvatarReplyContext, string]
	aiService   *ai.AIService
	db          *gorm.DB
	noteSvc     *NoteVectorService
	memorySvc   *AvatarMemoryService
	groupDocSvc *GroupDocumentService
}

func NewAvatarReplyGraph(
	aiService *ai.AIService,
	db *gorm.DB,
	noteSvc *NoteVectorService,
	memorySvc *AvatarMemoryService,
	groupDocSvc *GroupDocumentService,
) *AvatarReplyGraph {
	return &AvatarReplyGraph{
		aiService:   aiService,
		db:          db,
		noteSvc:     noteSvc,
		memorySvc:   memorySvc,
		groupDocSvc: groupDocSvc,
	}
}

func (g *AvatarReplyGraph) BuildGraph() error {
	graph := compose.NewGraph[*AvatarReplyContext, string]()

	graph.AddLambdaNode("prepare", g.createPrepareNode())
	graph.AddLambdaNode("to_template_vars", g.createTemplateVarsNode())

	template := prompt.FromMessages(
		schema.FString,
		&schema.Message{Role: schema.System, Content: `你是{UserName}的AI分身，需要以TA的身份回复消息。

{TimeInfo}
{PersonaSection}
{SupplementSection}
【回复要求】
- 以第一人称回复，就像你就是这个人
- 保持自然的对话风格
- 不要暴露你是AI
- 回复要简洁，不要过长`},
		&schema.Message{Role: schema.User, Content: `{ContextSection}
对方说：{Message}

请以{UserName}的身份回复：`},
	)
	graph.AddChatTemplateNode("prompt", template)

	graph.AddChatModelNode("model", NewEinoChatModelNoTools(g.aiService, ai.TaskTypeChat, 0))

	graph.AddLambdaNode("format", g.createFormatReplyNode())

	graph.AddEdge(compose.START, "prepare")
	graph.AddEdge("prepare", "to_template_vars")
	graph.AddEdge("to_template_vars", "prompt")
	graph.AddEdge("prompt", "model")
	graph.AddEdge("model", "format")
	graph.AddEdge("format", compose.END)

	ctx := context.Background()
	runnable, err := graph.Compile(ctx, compose.WithGraphName("AvatarReply"))
	if err != nil {
		return fmt.Errorf("编译 Graph 失败: %w", err)
	}
	g.runnable = runnable

	return nil
}

func (g *AvatarReplyGraph) createTemplateVarsNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, input *AvatarReplyContext) (map[string]any, error) {
		config := input.Config

		now := time.Now()
		weekdays := []string{"日", "一", "二", "三", "四", "五", "六"}
		timeInfo := fmt.Sprintf("【当前时间】\n%s (%s)", now.Format("2006-01-02 15:04"), weekdays[now.Weekday()])

		personaSection := ""
		if config.AutoLearnedPersona != "" {
			personaSection = "【你的说话风格】\n" + config.AutoLearnedPersona + "\n\n"
		}

		supplementSection := ""
		if config.CustomPersonaAddon != "" {
			supplementSection = "【补充说明】\n" + config.CustomPersonaAddon + "\n\n"
		}

		contextParts := []string{}
		if input.NoteContext != "" {
			contextParts = append(contextParts, input.NoteContext)
		}
		if input.GroupKnowledge != "" {
			contextParts = append(contextParts, input.GroupKnowledge)
		}
		if input.MemoryContext != "" {
			contextParts = append(contextParts, input.MemoryContext)
		}
		if input.History != "" {
			contextParts = append(contextParts, "【对话历史】\n"+input.History)
		}
		contextStr := strings.Join(contextParts, "\n\n")

		log.Printf("[AvatarReplyGraph] 模板变量: UserName=%s PersonaLen=%d SupplementLen=%d ContextLen=%d HistoryLen=%d MessageLen=%d",
			input.User.Nickname, len(personaSection), len(supplementSection), len(contextStr), len(input.History), len(input.Message))

		userName := input.User.Nickname
		if userName == "" {
			userName = input.User.Username
		}

		return map[string]any{
			"UserName":          userName,
			"TimeInfo":          timeInfo,
			"PersonaSection":    personaSection,
			"SupplementSection": supplementSection,
			"ContextSection":    contextStr,
			"Message":           input.Message,
		}, nil
	})
}

func (g *AvatarReplyGraph) createFormatReplyNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (string, error) {
		reply := msg.Content
		return reply, nil
	})
}

func (g *AvatarReplyGraph) Execute(ctx context.Context, userID uint, conversationID uint, message string) (string, error) {
	if g.runnable == nil {
		return "", fmt.Errorf("Graph 未编译，请先调用 BuildGraph")
	}

	input := &AvatarReplyContext{
		Message:        message,
		ConversationID: conversationID,
		UserID:         userID,
	}

	startTime := time.Now()
	reply, err := g.runnable.Invoke(ctx, input)
	if err != nil {
		return "", err
	}

	log.Printf("[AvatarReplyGraph] 生成回复耗时: %v", time.Since(startTime))

	return reply, nil
}

func (g *AvatarReplyGraph) createPrepareNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, input *AvatarReplyContext) (*AvatarReplyContext, error) {
		var config model.AvatarConfig
		if err := g.db.Where("user_id = ?", input.UserID).First(&config).Error; err != nil {
			return nil, fmt.Errorf("分身配置不存在")
		}
		input.Config = config

		var user model.User
		if err := g.db.First(&user, input.UserID).Error; err != nil {
			return nil, fmt.Errorf("用户不存在")
		}
		input.User = user

		if config.KnowledgeScopeJSON != "" {
			_ = json.Unmarshal([]byte(config.KnowledgeScopeJSON), &input.KnowledgeScope)
		}
		if config.ReplyStrategyJSON != "" {
			_ = json.Unmarshal([]byte(config.ReplyStrategyJSON), &input.ReplyStrategy)
		}

		noteCtx := ""
		if g.noteSvc != nil {
			noteResults, err := g.noteSvc.SearchNotes(input.UserID, input.Message, 3)
			if err == nil && len(noteResults) > 0 {
				var parts []string
				for _, r := range noteResults {
					parts = append(parts, fmt.Sprintf("[笔记: %s]\n%s", r.Metadata["title"], r.Content))
				}
				noteCtx = "【相关笔记知识】\n" + strings.Join(parts, "\n\n")
			}
		}
		input.NoteContext = noteCtx

		groupKnowledge := ""
		if g.groupDocSvc != nil && input.KnowledgeScope.KnowledgeDocs {
			var memberships []model.ConversationMember
			g.db.Where("user_id = ?", input.UserID).Find(&memberships)
			for _, m := range memberships {
				var conv model.Conversation
				if g.db.First(&conv, m.ConversationID).Error != nil {
					continue
				}
				if conv.Type != "group" && conv.Type != "discussion" {
					continue
				}
				var group model.Group
				if g.db.Where("conversation_id = ?", m.ConversationID).First(&group).Error != nil {
					continue
				}
				results, err := g.groupDocSvc.SearchKnowledge(group.ID, input.Message, 2)
				if err == nil && len(results) > 0 {
					var parts []string
					for _, r := range results {
						parts = append(parts, fmt.Sprintf("[群知识库: %s]\n%s", r.Metadata["title"], r.Content))
					}
					groupKnowledge = "【群知识库】\n" + strings.Join(parts, "\n\n")
					break
				}
			}
		}
		input.GroupKnowledge = groupKnowledge

		memoryCtx := ""
		if g.memorySvc != nil {
			memoryResults, err := g.memorySvc.Recall(input.UserID, input.Message, 2)
			if err == nil && len(memoryResults) > 0 {
				var parts []string
				for _, r := range memoryResults {
					parts = append(parts, r.Content)
				}
				memoryCtx = "【相关记忆】\n" + strings.Join(parts, "\n\n")
			}
		}
		input.MemoryContext = memoryCtx

		history := ""
		if input.ConversationID > 0 {
			history = g.getConversationHistory(input.ConversationID, 10)
		}
		input.History = history

		return input, nil
	})
}

func (g *AvatarReplyGraph) getConversationHistory(conversationID uint, limit int) string {
	var messages []model.Message
	g.db.Where("conversation_id = ?", conversationID).
		Where("type = ?", "text").
		Order("created_at DESC").
		Limit(limit).
		Find(&messages)

	if len(messages) == 0 {
		return ""
	}

	var parts []string
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		var sender model.User
		g.db.First(&sender, msg.SenderID)
		parts = append(parts, fmt.Sprintf("%s: %s", sender.Nickname, msg.Content))
	}

	return strings.Join(parts, "\n")
}
