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
	"github.com/dshmyz/qim/qim-server/utils"

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
	// CustomProvider 非 nil 表示分身配置了「使用自定义模型」（!UseSystemConfig && ModelConfigID）。
	// 命中时回复生成走该 provider（图外临时创建），绕开编译图内固定使用系统配置的 model 节点。
	CustomProvider *customModelProvider
}

// customModelProvider 用户自选模型在回复生成阶段的临时描述。
// providerName 是 ai.ProviderFactory.CreateProviderByName 支持的厂商名（openai/anthropic/...）。
type customModelProvider struct {
	ProviderName string
	Config       ai.ProviderConfig
}

type AvatarReplyGraph struct {
	runnable    compose.Runnable[*AvatarReplyContext, string]
	template    prompt.ChatTemplate // 供 ExecuteStream 在图外渲染消息后直接流式
	aiService   *ai.AIService
	db          *gorm.DB
	noteSvc     *NoteVectorService
	memorySvc   *AvatarMemoryService
	groupDocSvc *GroupDocumentService
	aiConfigSvc *AIConfigService // 解析分身「自选模型」配置（modelConfigId → provider）
}

func NewAvatarReplyGraph(
	aiService *ai.AIService,
	db *gorm.DB,
	noteSvc *NoteVectorService,
	memorySvc *AvatarMemoryService,
	groupDocSvc *GroupDocumentService,
	aiConfigSvc *AIConfigService,
) *AvatarReplyGraph {
	return &AvatarReplyGraph{
		aiService:   aiService,
		db:          db,
		noteSvc:     noteSvc,
		memorySvc:   memorySvc,
		groupDocSvc: groupDocSvc,
		aiConfigSvc: aiConfigSvc,
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
	g.template = template

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
		return g.buildTemplateVars(input), nil
	})
}

// buildTemplateVars 构造模板变量（图内节点与 ExecuteStream 图外渲染共用，避免双套拼装逻辑）
func (g *AvatarReplyGraph) buildTemplateVars(input *AvatarReplyContext) map[string]any {
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
	}
}

func (g *AvatarReplyGraph) createFormatReplyNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (string, error) {
		reply := msg.Content
		return reply, nil
	})
}

func (g *AvatarReplyGraph) Execute(ctx context.Context, userID uint, conversationID uint, message string, preloaded *model.AvatarConfig) (string, error) {
	if g.runnable == nil {
		return "", fmt.Errorf("Graph 未编译，请先调用 BuildGraph")
	}

	input := &AvatarReplyContext{
		Message:        message,
		ConversationID: conversationID,
		UserID:         userID,
	}

	// 先在图外完成上下文准备，以便在命中"不回复"时直接跳过 LLM 调用
	if err := g.prepare(ctx, input, preloaded); err != nil {
		return "", err
	}
	if input.SkipReply {
		log.Printf("[AvatarReplyGraph] 命中不回复策略，跳过 LLM: userID=%d convID=%d", userID, conversationID)
		return "", nil
	}

	startTime := time.Now()
	var reply string
	var err error
	if input.CustomProvider != nil {
		// 自选模型：绕开编译图（其 model 节点固定走系统配置），图外渲染 prompt 后
		// 用用户自选 provider 生成，与编译图行为一致（同一套模板变量 / 截断逻辑）。
		reply, err = g.generateWithCustomProvider(ctx, input)
	} else {
		reply, err = g.runnable.Invoke(ctx, input)
	}
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

// generateWithCustomProvider 用分身「自选模型」生成回复（非流式）。
// 复用 buildTemplateVars + template.Format 渲染 prompt，再用用户自选 provider 完成一次对话。
func (g *AvatarReplyGraph) generateWithCustomProvider(ctx context.Context, input *AvatarReplyContext) (string, error) {
	if g.template == nil {
		return "", fmt.Errorf("Graph 未编译，请先调用 BuildGraph")
	}
	vars := g.buildTemplateVars(input)
	messages, err := g.template.Format(ctx, vars)
	if err != nil {
		return "", fmt.Errorf("渲染分身 prompt 失败: %w", err)
	}
	aiMessages := make([]ai.Message, 0, len(messages))
	for _, m := range messages {
		aiMessages = append(aiMessages, ai.Message{Role: string(m.Role), Content: m.Content})
	}
	return g.aiService.GetCompletionWithProviderConfig(ai.TaskTypeChat, aiMessages, input.CustomProvider.ProviderName, input.CustomProvider.Config)
}

// ExecuteStream 以流式生成分身回复（供"帮我回复"草稿模式使用）。
// 与 Execute 的差异：1) 不走编译图，渲染模板后直接调 chatModel.Stream；2) 忽略 SkipReply
// （用户主动要草稿，即使命中"知识范围外静默"也照常生成）；3) 不做 MaxReplyLength 截断
// （草稿进输入框由用户自行编辑，截断无意义）。
func (g *AvatarReplyGraph) ExecuteStream(ctx context.Context, userID uint, conversationID uint, message string, preloaded *model.AvatarConfig) (*schema.StreamReader[*schema.Message], error) {
	if g.template == nil {
		return nil, fmt.Errorf("Graph 未编译，请先调用 BuildGraph")
	}

	input := &AvatarReplyContext{
		Message:        message,
		ConversationID: conversationID,
		UserID:         userID,
	}

	if err := g.prepare(ctx, input, preloaded); err != nil {
		return nil, err
	}
	// 草稿模式：忽略 SkipReply（用户主动要草稿，不该因"超知识范围"静默）

	vars := g.buildTemplateVars(input)
	messageList, err := g.template.Format(ctx, vars)
	if err != nil {
		return nil, fmt.Errorf("渲染分身 prompt 失败: %w", err)
	}

	// 自选模型走临时 provider 流式；否则走系统配置的 EinoChatModel。
	if input.CustomProvider != nil {
		aiMessages := make([]ai.Message, 0, len(messageList))
		for _, m := range messageList {
			aiMessages = append(aiMessages, ai.Message{Role: string(m.Role), Content: m.Content})
		}
		sr, sw := schema.Pipe[*schema.Message](0)
		go func() {
			defer sw.Close()
			err := g.aiService.ChatStreamWithProviderConfig(ctx, ai.TaskTypeChat, aiMessages,
				input.CustomProvider.ProviderName, input.CustomProvider.Config,
				func(chunk ai.StreamChunk) error {
					sw.Send(&schema.Message{Role: schema.Assistant, Content: chunk.Content}, nil)
					return nil
				})
			if err != nil {
				log.Printf("[AvatarReplyGraph] 自选模型流式错误: %v", err)
				sw.Send(nil, err)
			}
		}()
		return sr, nil
	}

	chatModel := NewEinoChatModelNoTools(g.aiService, ai.TaskTypeChat, input.UserID)
	return chatModel.Stream(ctx, messageList)
}

// prepare 加载分身配置、用户、知识范围与历史，并判定是否命中"不回复"策略。
// 抽出为普通方法便于 Execute 在调用 LLM 前短路，也便于后续单测。
// preloaded 非 nil 时复用调用方已加载的配置，避免一次回复流程内重复查 avatar_configs。
func (g *AvatarReplyGraph) prepare(ctx context.Context, input *AvatarReplyContext, preloaded *model.AvatarConfig) error {
	if preloaded != nil {
		input.Config = *preloaded
	} else {
		var config model.AvatarConfig
		if err := g.db.Where("user_id = ?", input.UserID).First(&config).Error; err != nil {
			return fmt.Errorf("分身配置不存在")
		}
		input.Config = config
	}

	var user model.User
	if err := g.db.First(&user, input.UserID).Error; err != nil {
		return fmt.Errorf("用户不存在")
	}
	input.User = user

	if input.Config.KnowledgeScopeJSON != "" {
		_ = json.Unmarshal([]byte(input.Config.KnowledgeScopeJSON), &input.KnowledgeScope)
	}
	if input.Config.ReplyStrategyJSON != "" {
		_ = json.Unmarshal([]byte(input.Config.ReplyStrategyJSON), &input.ReplyStrategy)
	}

	noteCtx := ""
	// 笔记检索受 Notes 开关门控，与其他知识来源对齐（否则 Notes 关不掉，且会破坏 docs-only 的范围控制）
	if g.noteSvc != nil && input.KnowledgeScope.Notes {
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

	// 范围外静默：ReplyOutOfScope=false 时，分身只应回复「与自身知识/上下文相关」的消息。
	// 有笔记/群知识/记忆命中 → 属于范围内，正常回复；无任何上下文命中 → 属于范围外。
	// 任务不参与范围内判定（任务只是附加注入的知识，不应让"有任务就什么都回"旁路门控）。
	// 范围外时用 LLM 判断消息是否有针对性（是否真的需要代表主人回复）：
	// 需要 → 放行；纯闲聊/无关 → 静默。AI 不可用则 fail-closed 静默，避免刷屏。
	hasKnowledge := noteCtx != "" || groupKnowledge != "" || memoryCtx != ""
	if !input.ReplyStrategy.ReplyOutOfScope && !hasKnowledge {
		need, err := g.needReplyForOutOfScope(input)
		if err != nil || !need {
			input.SkipReply = true
		}
	}

	// 自选模型：配置了「使用自定义模型」时解析出 provider，供 Execute/ExecuteStream 走临时 provider 生成。
	// 解析失败（配置不存在/密钥解密失败等）时置为 nil，静默回退系统默认配置，不阻断回复。
	if !input.Config.UseSystemConfig && input.Config.ModelConfigID != nil {
		input.CustomProvider = g.resolveCustomProvider(input.UserID, *input.Config.ModelConfigID)
	}

	return nil
}

// resolveCustomProvider 根据分身配置的 modelConfigID 解析出自选模型 provider。
// 返回 nil 表示未命中或解析失败（回退系统默认配置）。userID 用于校验该 AIConfig 归属。
func (g *AvatarReplyGraph) resolveCustomProvider(userID, configID uint) *customModelProvider {
	if g.aiConfigSvc == nil {
		log.Printf("[AvatarReplyGraph] 分身自选模型已配置但 AIConfigService 未注入，回退系统默认: userID=%d", userID)
		return nil
	}
	var cfg model.AIConfig
	if err := g.db.Where("id = ? AND user_id = ?", configID, userID).First(&cfg).Error; err != nil {
		log.Printf("[AvatarReplyGraph] 分身自选模型配置不存在，回退系统默认: userID=%d configID=%d err=%v", userID, configID, err)
		return nil
	}
	if !cfg.AIEnabled {
		log.Printf("[AvatarReplyGraph] 分身自选模型配置已禁用，回退系统默认: userID=%d configID=%d", userID, configID)
		return nil
	}
	apiKey, err := utils.DecryptAPIKey(cfg.APIKeyEncrypted)
	if err != nil {
		log.Printf("[AvatarReplyGraph] 分身自选模型密钥解密失败，回退系统默认: userID=%d configID=%d err=%v", userID, configID, err)
		return nil
	}
	return &customModelProvider{
		ProviderName: cfg.Provider,
		Config: ai.ProviderConfig{
			APIKey:  apiKey,
			Model:   cfg.ModelName,
			BaseURL: cfg.BaseURL,
			// 透传用户保存的生成参数；零值跳过让 provider 用默认值
			ExtraParams: buildCustomProviderExtraParams(cfg.MaxTokens, cfg.Temperature),
		},
	}
}

// buildCustomProviderExtraParams 构建分身自选模型的 ExtraParams。
// 仅透传有效值（> 0），零值跳过让 provider 用默认值：
// - max_tokens=0 在某些 provider 会被 API 拒绝或解释为无限制
// - temperature=0 会覆盖 provider 默认值（用户未显式配置时不该强制覆盖）
func buildCustomProviderExtraParams(maxTokens int, temperature float64) map[string]interface{} {
	params := map[string]interface{}{}
	if maxTokens > 0 {
		params["max_tokens"] = maxTokens
	}
	if temperature > 0 {
		params["temperature"] = temperature
	}
	return params
}

// needReplyForOutOfScope 判断「知识范围外」的消息是否仍有必要代表主人回复。
// 主要用于默认配置（无知识库/笔记/记忆）下的分身，避免对无关闲聊硬回导致乱回复。
// 返回 false 表示应静默。AI 不可用 / 判断失败时返回 false（fail-closed）。
func (g *AvatarReplyGraph) needReplyForOutOfScope(input *AvatarReplyContext) (bool, error) {
	if g.aiService == nil || !g.aiService.IsConfigured() {
		return false, nil
	}

	prompt := fmt.Sprintf(`你是%s的AI分身。下面的消息并不是在你已知的知识范围内，但仍需判断：这条消息是否明确需要你代表%s回应？

如果是对方真的在向%s提问、托付、或与%s高度相关需要代为处理，回复 true；
如果只是普通寒暄、闲聊、与自己无关、或标点/表情/无意义消息，回复 false。

只返回 JSON：{"should_reply": true/false}
消息：%s`,
		input.User.Nickname, input.User.Nickname, input.User.Nickname, input.User.Nickname, input.Message)

	aiMessages := []ai.Message{{Role: "user", Content: prompt}}
	result, err := g.aiService.GetCompletion(ai.TaskTypeChat, aiMessages)
	if err != nil {
		log.Printf("[AvatarReplyGraph] 范围外针对性判断失败: userID=%d err=%v", input.UserID, err)
		return false, err
	}

	var response struct {
		ShouldReply bool `json:"should_reply"`
	}
	raw := strings.TrimSpace(result)
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		if sub := extractJSONObject(raw); sub != "" {
			if err2 := json.Unmarshal([]byte(sub), &response); err2 == nil {
				return response.ShouldReply, nil
			}
		}
		log.Printf("[AvatarReplyGraph] 解析范围外针对性判断失败，静默跳过: err=%v raw=%s", err, result)
		return false, nil
	}
	return response.ShouldReply, nil
}

func (g *AvatarReplyGraph) getConversationHistory(conversationID uint, limit int, triggerMessage string) string {
	var messages []model.Message
	g.db.Where("conversation_id = ?", conversationID).
		Where("type = ?", "text").
		Where("origin IS NULL OR origin != ?", "avatar").
		Order("created_at DESC").
		Limit(limit + 1). // 多取 1 条，预留触发消息被剔除后仍满 limit
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
