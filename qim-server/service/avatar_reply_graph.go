package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"qim-server/ai"
	"qim-server/model"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/components/prompt"
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

	graph.AddLambdaNode("prepare", compose.InvokableLambda(g.prepare))
	graph.AddLambdaNode("to_template_vars", compose.InvokableLambda(g.templateVars))

	template := prompt.FromMessages(
		schema.FString,
		&schema.Message{Role: schema.System, Content: `你是{UserName}的AI分身，需要以TA的身份回复别人发给TA的消息。

{PersonaSection}
{SupplementSection}
【回复要求】
- 以下是一段别人发给{UserName}的消息内容，你需要仔细阅读后直接回应其内容
- 以第一人称回复，就像你就是{UserName}本人
- 保持自然的对话风格
- 不要暴露你是AI
- 回复要简洁，就像在日常聊天中说话一样
- 不要复述对方的问题，直接给出自然的回应
- 如果对方在问问题，请直接回答，不要说"好的我来回答"之类的套话
- 不要输出"XXX的分身代为回复"等前缀或说明，直接输出回复内容本身
- 不要使用"XXX说：""XXX："等署名格式`},
		&schema.Message{Role: schema.User, Content: `{ContextSection}
别人发给我的消息：
"""
{Message}
"""

请以我的身份直接回复上面的消息：`},
	)
	graph.AddChatTemplateNode("prompt", template)
	graph.AddChatModelNode("model", NewEinoChatModel(g.aiService))
	graph.AddLambdaNode("format", compose.InvokableLambda(g.formatReply))

	graph.AddEdge(compose.START, "prepare")
	graph.AddEdge("prepare", "to_template_vars")
	graph.AddEdge("to_template_vars", "prompt")
	graph.AddEdge("prompt", "model")
	graph.AddEdge("model", "format")
	graph.AddEdge("format", compose.END)

	runnable, err := compileGraph(graph, "AvatarReply")
	if err != nil {
		return err
	}
	g.runnable = runnable
	return nil
}

func (g *AvatarReplyGraph) Execute(ctx context.Context, userID uint, conversationID uint, message string) (string, error) {
	input := &AvatarReplyContext{
		Message:        message,
		ConversationID: conversationID,
		UserID:         userID,
	}

	// 构建 CallerContext 注入到 context，供 EinoChatModel 中获取
	callerCtx := &ai.CallerContext{
		UserID:       userID,
		AllowedTools: []string{}, // 禁止所有工具调用，避免与 prepare 预取上下文重复
	}
	if conversationID > 0 {
		var conv model.Conversation
		if g.db.First(&conv, conversationID).Error == nil && (conv.Type == "group" || conv.Type == "discussion") {
			var group model.Group
			if g.db.Where("conversation_id = ?", conversationID).First(&group).Error == nil {
				callerCtx.GroupID = group.ID
			}
		}
	}
	ctx = WithCallerContext(ctx, callerCtx)

	startTime := time.Now()
	reply, err := simpleExecuteWithCache(ctx, nil, "", 0, g.runnable, input)
	if err != nil {
		return "", err
	}
	log.Printf("[AvatarReplyGraph] 生成回复耗时: %v, 回复长度: %d, 回复内容: %q", time.Since(startTime), len(reply), reply)
	return reply, nil
}

func (g *AvatarReplyGraph) prepare(ctx context.Context, input *AvatarReplyContext) (*AvatarReplyContext, error) {
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
		if err := json.Unmarshal([]byte(config.KnowledgeScopeJSON), &input.KnowledgeScope); err != nil {
			log.Printf("[AvatarReplyGraph] 解析 KnowledgeScopeJSON 失败: %v", err)
		}
	}
	if config.ReplyStrategyJSON != "" {
		if err := json.Unmarshal([]byte(config.ReplyStrategyJSON), &input.ReplyStrategy); err != nil {
			log.Printf("[AvatarReplyGraph] 解析 ReplyStrategyJSON 失败: %v", err)
		}
	}

	if g.noteSvc != nil {
		noteResults, err := g.noteSvc.SearchNotes(input.UserID, input.Message, 3)
		if err == nil && len(noteResults) > 0 {
			var parts []string
			for _, r := range noteResults {
				parts = append(parts, fmt.Sprintf("[笔记: %s]\n%s", r.Metadata["title"], r.Content))
			}
			input.NoteContext = "【相关笔记知识】\n" + strings.Join(parts, "\n\n")
		}
	}

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
				input.GroupKnowledge = "【群知识库】\n" + strings.Join(parts, "\n\n")
				break
			}
		}
	}

	if g.memorySvc != nil {
		memoryResults, err := g.memorySvc.Recall(input.UserID, input.Message, 2)
		if err == nil && len(memoryResults) > 0 {
			var parts []string
			for _, r := range memoryResults {
				parts = append(parts, r.Content)
			}
			input.MemoryContext = "【相关记忆】\n" + strings.Join(parts, "\n\n")
		}
	}

	if input.ConversationID > 0 {
		input.History = g.conversationHistory(input.ConversationID, 10)
	}

	return input, nil
}

func (g *AvatarReplyGraph) templateVars(ctx context.Context, input *AvatarReplyContext) (map[string]any, error) {
	personaSection := ""
	if input.Config.AutoLearnedPersona != "" {
		personaSection = "【你的说话风格】\n" + input.Config.AutoLearnedPersona + "\n\n"
	}
	supplementSection := ""
	if input.Config.CustomPersonaAddon != "" {
		supplementSection = "【补充说明】\n" + input.Config.CustomPersonaAddon + "\n\n"
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

	return map[string]any{
		"UserName":          input.User.Nickname,
		"PersonaSection":    personaSection,
		"SupplementSection": supplementSection,
		"ContextSection":    strings.Join(contextParts, "\n\n"),
		"Message":           input.Message,
	}, nil
}

func (g *AvatarReplyGraph) formatReply(ctx context.Context, msg *schema.Message) (string, error) {
	return msg.Content, nil
}

func (g *AvatarReplyGraph) conversationHistory(conversationID uint, limit int) string {
	var messages []model.Message
	g.db.Where("conversation_id = ?", conversationID).
		Where("type = ?", "text").
		Where("ai_type = ?", "").
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
