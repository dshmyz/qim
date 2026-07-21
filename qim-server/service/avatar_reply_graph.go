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
	TaskContext    string
	History        string
	// SkipReply 命中"知识范围外且配置为不回复"时置位，Execute 据此跳过 LLM 调用
	SkipReply bool
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
- {LengthHint}`},
		&schema.Message{Role: schema.User, Content: `{ContextSection}
对方说：{Message}

请以{UserName}的身份回复：`},
	)
	graph.AddChatTemplateNode("prompt", template)

	graph.AddChatModelNode("model", NewEinoChatModelNoTools(g.aiService, ai.TaskTypeChat, 0))

	graph.AddLambdaNode("format", g.createFormatReplyNode())

	graph.AddEdge(compose.START, "to_template_vars")
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
		if input.TaskContext != "" {
			contextParts = append(contextParts, input.TaskContext)
		}
		if input.History != "" {
			contextParts = append(contextParts, "【对话历史】\n"+input.History)
		}
		contextStr := strings.Join(contextParts, "\n\n")

		lengthHint := avatarLengthHint(input.ReplyStrategy.MaxReplyLength)

		log.Printf("[AvatarReplyGraph] 模板变量: UserName=%s PersonaLen=%d SupplementLen=%d ContextLen=%d HistoryLen=%d MessageLen=%d LengthHint=%s",
			input.User.Nickname, len(personaSection), len(supplementSection), len(contextStr), len(input.History), len(input.Message), lengthHint)

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
			"LengthHint":        lengthHint,
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

	// 先在图外完成上下文准备，以便在命中"不回复"时直接跳过 LLM 调用
	if err := g.prepare(ctx, input); err != nil {
		return "", err
	}
	if input.SkipReply {
		log.Printf("[AvatarReplyGraph] 命中不回复策略，跳过 LLM: userID=%d convID=%d", userID, conversationID)
		return "", nil
	}

	startTime := time.Now()
	reply, err := g.runnable.Invoke(ctx, input)
	if err != nil {
		return "", err
	}

	log.Printf("[AvatarReplyGraph] 生成回复耗时: %v", time.Since(startTime))

	if maxRunes := avatarMaxReplyChars(input.ReplyStrategy.MaxReplyLength); maxRunes > 0 {
		// 按 rune 截断，避免在多字节 UTF-8（中文）rune 中间切断产生无效 UTF-8
		runes := []rune(reply)
		if len(runes) > maxRunes {
			reply = strings.TrimSpace(string(runes[:maxRunes])) + "…"
		}
	}

	return reply, nil
}

// prepare 加载分身配置、用户、知识范围与历史，并判定是否命中"不回复"策略。
// 抽出为普通方法便于 Execute 在调用 LLM 前短路，也便于后续单测。
func (g *AvatarReplyGraph) prepare(ctx context.Context, input *AvatarReplyContext) error {
	var config model.AvatarConfig
	if err := g.db.Where("user_id = ?", input.UserID).First(&config).Error; err != nil {
		return fmt.Errorf("分身配置不存在")
	}
	input.Config = config

	var user model.User
	if err := g.db.First(&user, input.UserID).Error; err != nil {
		return fmt.Errorf("用户不存在")
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
	// 只检索当前会话所在群的知识库，不再遍历用户全部 memberships（消除 N+1，且避免跨会话串味）
	if g.groupDocSvc != nil && input.KnowledgeScope.KnowledgeDocs && input.ConversationID > 0 {
		var conv model.Conversation
		if g.db.First(&conv, input.ConversationID).Error == nil && (conv.Type == "group" || conv.Type == "discussion") {
			var group model.Group
			if g.db.Where("conversation_id = ?", input.ConversationID).First(&group).Error == nil {
				results, err := g.groupDocSvc.SearchKnowledge(group.ID, input.Message, 2)
				if err == nil && len(results) > 0 {
					var parts []string
					for _, r := range results {
						parts = append(parts, fmt.Sprintf("[群知识库: %s]\n%s", r.Metadata["title"], r.Content))
					}
					groupKnowledge = "【群知识库】\n" + strings.Join(parts, "\n\n")
				}
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

	// 任务作为附加知识注入 prompt（不参与范围外静默门控，避免零任务分身被误静默）
	taskCtx := ""
	if input.KnowledgeScope.Tasks {
		var tasks []model.Task
		g.db.Where("user_id = ? AND (status IS NULL OR status = '' OR status NOT IN (?, ?))",
			input.UserID, "done", "completed").
			Order("created_at DESC").Limit(5).Find(&tasks)
		if len(tasks) > 0 {
			parts := make([]string, 0, len(tasks))
			for _, t := range tasks {
				line := t.Title
				if t.DueDate != nil {
					line += "（截止 " + t.DueDate.Format("01-02 15:04") + "）"
				}
				if t.Priority != "" && t.Priority != "medium" {
					line += " 优先级:" + t.Priority
				}
				parts = append(parts, "- "+line)
			}
			taskCtx = "【我的任务】\n" + strings.Join(parts, "\n")
		}
	}
	input.TaskContext = taskCtx

	history := ""
	// ConversationHistory nil（存量配置未显式设置）按默认 true 处理，避免升级后静默丢失历史
	historyEnabled := true
	if input.KnowledgeScope.ConversationHistory != nil {
		historyEnabled = *input.KnowledgeScope.ConversationHistory
	}
	if historyEnabled && input.ConversationID > 0 {
		history = g.getConversationHistory(input.ConversationID, 10, input.Message)
	}
	input.History = history

	// 命中"不回复"策略：配置了知识来源但三处全空，且策略要求范围外静默。
	// 只计入 prepare 实际会检索的来源（笔记/群文档）；Tasks 暂无检索路径，计入会让仅开 Tasks 的分身永不回复。
	knowledgeConfigured := input.KnowledgeScope.KnowledgeDocs || input.KnowledgeScope.Notes
	noKnowledgeFound := noteCtx == "" && groupKnowledge == "" && memoryCtx == ""
	if knowledgeConfigured && noKnowledgeFound && !input.ReplyStrategy.ReplyOutOfScope {
		input.SkipReply = true
	}

	return nil
}

func (g *AvatarReplyGraph) getConversationHistory(conversationID uint, limit int, triggerMessage string) string {
	var messages []model.Message
	g.db.Where("conversation_id = ?", conversationID).
		Where("type = ?", "text").
		Where("origin IS NULL OR origin != ?", "avatar").
		Order("created_at DESC").
		Limit(limit).
		Find(&messages)

	if len(messages) == 0 {
		return ""
	}

	// 触发消息本身已在 prompt 中以"对方说：{Message}"呈现，这里按 DESC 取到的最新一条若与之相同则剔除，避免模型重复见到
	if triggerMessage != "" && messages[0].Content == triggerMessage {
		messages = messages[1:]
		if len(messages) == 0 {
			return ""
		}
	}

	// 批量查询发送者，避免 N+1
	senderIDs := make(map[uint]struct{}, len(messages))
	for _, msg := range messages {
		senderIDs[msg.SenderID] = struct{}{}
	}
	ids := make([]uint, 0, len(senderIDs))
	for id := range senderIDs {
		ids = append(ids, id)
	}
	var senders []model.User
	g.db.Where("id IN ?", ids).Find(&senders)
	senderMap := make(map[uint]model.User, len(senders))
	for _, s := range senders {
		senderMap[s.ID] = s
	}

	var parts []string
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		sender := senderMap[msg.SenderID]
		parts = append(parts, fmt.Sprintf("%s: %s", sender.Nickname, msg.Content))
	}

	return strings.Join(parts, "\n")
}

// avatarLengthHint 将回复长度偏好枚举映射为提示词文本
func avatarLengthHint(maxReplyLength string) string {
	switch strings.ToLower(strings.TrimSpace(maxReplyLength)) {
	case "short":
		return "回复尽量简短，以一句话为主"
	case "medium":
		return "回复长度适中"
	case "long":
		return "回复可以详细，但仍需自然"
	default:
		return "回复要简洁，不要过长"
	}
}

// avatarMaxReplyChars 将回复长度偏好枚举映射为字符硬上限，0 表示不截断
func avatarMaxReplyChars(maxReplyLength string) int {
	switch strings.ToLower(strings.TrimSpace(maxReplyLength)) {
	case "short":
		return 100
	case "medium":
		return 300
	case "long":
		return 2000
	default:
		return 0
	}
}
