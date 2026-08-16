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
	"github.com/dshmyz/qim/qim-server/pkg/aiprompt"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"gorm.io/gorm"
)

// AvatarReplyContext 分身回复图上下文。命中的知识来源统一用 KnowledgeSource 结构
// （与群助手/Bot 的「知识来源」同构），随 WS 下发前端展示「依据」。
// 仅记录实际参与注入的笔记/群知识/记忆，任务仅作附加注入不打标记。

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
	// Sources 本条回复命中的知识来源（笔记/群知识/记忆标题、分数与摘要），供下发展示依据
	Sources []KnowledgeSource
	// SkipReply 命中"知识范围外且配置为不回复"时置位，Execute 据此跳过 LLM 调用
	SkipReply bool
	// HistoryBefore 对话历史锚点：非 nil 时只取该时间之前的消息作为上下文。
	// 「帮我回复」草稿模式由 handler 传入目标消息的 CreatedAt——目标可能不是会话最新一条，
	// 若仍按"整个会话最近 N 条"取历史，会把目标之后的后续对话混进来导致答非所问。
	// nil（分身自动回复等触发消息即最新一条）保持原"最近 N 条"语义。
	HistoryBefore *time.Time
	// CustomProvider 非 nil 表示分身配置了「使用自定义模型」（!UseSystemConfig && ModelConfigID）。
	// 命中时回复生成走该 provider（图外临时创建），绕开编译图内固定使用系统配置的 model 节点。
	CustomProvider *customProvider
}

// 用户自选模型在回复阶段的临时描述统一由 ai_config_service.go 的 customProvider 承担
// （与 resolveUserAIConfigProvider 共用），此处不再单独定义 customModelProvider 以免重复漂移。

type AvatarReplyGraph struct {
	runnable    compose.Runnable[*AvatarReplyContext, string]
	template    prompt.ChatTemplate // 供 ExecuteStream 在图外渲染消息后直接流式
	aiService   *ai.AIService
	db          *gorm.DB
	noteSvc     *NoteVectorService
	memorySvc   *AvatarMemoryService
	groupDocSvc *GroupDocumentService
	// thresholdSvc 阈值读取服务；nil 时记忆召回门槛用默认 0.5（向后兼容）。
	thresholdSvc *AiThresholdService
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

// SetThresholdService 注入阈值读取服务；nil 时记忆召回门槛用默认 0.5。
func (g *AvatarReplyGraph) SetThresholdService(t *AiThresholdService) {
	g.thresholdSvc = t
}

// memoryRecallThreshold 返回记忆召回相关度门槛：未注入阈值服务时回退默认 0.5。
func (g *AvatarReplyGraph) memoryRecallThreshold() float64 {
	if g.thresholdSvc != nil {
		return g.thresholdSvc.GetFloat("ai.memory_recall_threshold", 0.5)
	}
	return 0.5
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
- {LengthHint}
- 回答必须优先基于上方【相关笔记知识】【群知识库】【相关记忆】等资料；若资料不足以回答，请明确说明资料不足并给出你的理解，切勿假装引用或编造。`},
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

	timeInfo := aiprompt.CurrentTimeLine()

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
	reply, _, err := g.executeWithSources(ctx, userID, conversationID, message, preloaded)
	return reply, err
}

// ExecuteWithSources 与 Execute 等价，额外返回本条回复命中的知识来源（供下发展示「依据」）。
// 若命中"不回复"策略（SkipReply）或生成失败，sources 可能为空。
func (g *AvatarReplyGraph) ExecuteWithSources(ctx context.Context, userID uint, conversationID uint, message string, preloaded *model.AvatarConfig) (string, []KnowledgeSource, error) {
	return g.executeWithSources(ctx, userID, conversationID, message, preloaded)
}

func (g *AvatarReplyGraph) executeWithSources(ctx context.Context, userID uint, conversationID uint, message string, preloaded *model.AvatarConfig) (string, []KnowledgeSource, error) {
	if g.runnable == nil {
		return "", nil, fmt.Errorf("Graph 未编译，请先调用 BuildGraph")
	}

	input := &AvatarReplyContext{
		Message:        message,
		ConversationID: conversationID,
		UserID:         userID,
	}

	// 先在图外完成上下文准备，以便在命中"不回复"时直接跳过 LLM 调用
	if err := g.prepare(ctx, input, preloaded); err != nil {
		return "", nil, err
	}
	if input.SkipReply {
		log.Printf("[AvatarReplyGraph] 命中不回复策略，跳过 LLM: userID=%d convID=%d", userID, conversationID)
		return "", nil, nil
	}

	startTime := time.Now()
	var reply string
	var err error
	if input.CustomProvider != nil {
		// 自选模型：绕开编译图（其 model 节点固定走系统配置），图外渲染 prompt 后
		// 用用户自选 provider 生成，与编译图行为一致（同一套模板变量 / 截断逻辑）。
		reply, err = g.generateWithCustomProvider(ctx, input)
		if err != nil {
			// 生成期失败（密钥失效/配额/网络错/provider 未配置）→ 回退系统默认，
			// 兑现「回退系统默认…不阻断回复」契约——单条自定义配置出问题不应让分身整条不复回。
			// 系统回退同样失败时才真正返回错误。
			log.Printf("[AvatarReplyGraph] 自选模型生成失败，回退系统默认: userID=%d err=%v", input.UserID, err)
			reply, err = g.runnable.Invoke(ctx, input)
		}
	} else {
		reply, err = g.runnable.Invoke(ctx, input)
	}
	if err != nil {
		return "", nil, err
	}

	log.Printf("[AvatarReplyGraph] 生成回复耗时: %v", time.Since(startTime))

	reply = truncateReply(input, reply)

	return reply, input.Sources, nil
}

// ExecuteWithImageSources 供分身识别图片触发消息：读图成功后调用此方法生成回复。
// 差异点：1) 不走编译图（其模板 User 消息为纯文本，无法携带图片），复用 buildTemplateVars +
// template.Format 图外渲染；2) 把最后一条 user 消息替换为携带图片的 MultiContent 多模态消息
// （text + base64 data URL，经 einoMessagesToAIMessages 提取为 ai.Message.ImageURL），
// 由 GetCompletion 以 OpenAI image_url 数组格式交给视觉模型识别；3) 忽略 SkipReply（图片消息
// 走视觉识别，不因"知识范围外"而静默，与"尽力而为失败则跳过"的降级语义配合——能看图就回，
// 看不了/模型不支持则由 worker 跳过）。返回与 ExecuteWithSources 同构：回复 + 命中的知识来源。
func (g *AvatarReplyGraph) ExecuteWithImageSources(ctx context.Context, userID uint, conversationID uint, message string, imageURL string, imageName string, preloaded *model.AvatarConfig) (string, []KnowledgeSource, error) {
	input := &AvatarReplyContext{
		Message:        message,
		ConversationID: conversationID,
		UserID:         userID,
	}

	// 复用上下文组装（笔记/群知识/记忆/任务/历史/范围外判定），与 Execute 一致
	if err := g.prepare(ctx, input, preloaded); err != nil {
		return "", nil, err
	}

	messageList, err := g.renderPrompt(ctx, input)
	if err != nil {
		return "", nil, err
	}

	// 把 user 消息替换为携带图片的 MultiContent 多模态消息：
	// 原文本保留（包含对话上下文与"对方说"），追加图片识别指令，并携带 base64 data URL。
	messageList = injectSingleImage(messageList, imageURL, fmt.Sprintf("📷 用户发送了一张图片「%s」，请识别图片内容并结合以上对话与图片回复。", imageName))

	return g.completeReply(input, einoMessagesToAIMessages(messageList))
}

// ExecuteBatchWithImagesSources 供分身对「合并窗口内连发的一批消息」生成一条合并回复。
// 批内既可有纯文本消息，也可有图片消息（多模态）：所有文本按序拼进 user 消息提示分身
// 这是对方连发的一批、请整体理解回复；批内每张图片逐张已由调用方读成 base64 data URL，
// 全部注入最后一条 user 消息的 MultiContent（多个 image_url part）。与 ExecuteWithImageSources
// 同走图外渲染 + GetCompletion，忽略 SkipReply（能看图就回，看不了由 worker 按尽力而为跳过）。
//
// 入参 videoURLs 为批内每张图片的 data URL（顺序与 orderTexts 对应，两个 slice 长度一致），
// 由 AvatarService 在调用本方法前用 groupDocSvc.ImageURLForContext 逐图读取；读图在调用方
// 完成并返回错误时 worker 整批跳过。返回回复 + 命中的知识来源。
func (g *AvatarReplyGraph) ExecuteBatchWithImagesSources(ctx context.Context, userID uint, conversationID uint, orderTexts []string, imageURLs []string, imageNames []string, preloaded *model.AvatarConfig) (string, []KnowledgeSource, error) {
	// 把批内文本拼成一条合并 prompt：带序号并提示这是对方连发的一组消息
	lines := make([]string, 0, len(orderTexts))
	seq := 1
	for _, txt := range orderTexts {
		lines = append(lines, fmt.Sprintf("%d.%s", seq, txt))
		seq++
	}
	merged := strings.Join(lines, "\n")

	input := &AvatarReplyContext{
		Message:        merged,
		ConversationID: conversationID,
		UserID:         userID,
	}
	// 复用上下文组装（笔记/群知识/记忆/任务/历史/范围外判定），与单条图文路径一致
	if err := g.prepare(ctx, input, preloaded); err != nil {
		return "", nil, err
	}

	messageList, err := g.renderPrompt(ctx, input)
	if err != nil {
		return "", nil, err
	}

	// 把 user 消息替换为携带批内全部图片的 MultiContent 多模态消息
	messageList = injectMultiImage(messageList, imageURLs, "📷 对方连发了一组消息（含图片），请整体理解这些内容并结合对话回复。")

	return g.completeReply(input, einoMessagesToAIMessages(messageList))
}

// generateWithCustomProvider 用分身「自选模型」生成回复（非流式）。
// 复用 renderPrompt 渲染 prompt（含模板变量拼装），再用用户自选 provider 完成一次对话。
// 转换统一走 einoMessagesToAIMessages（含 MultiContent 图片提取），不再手拼纯文本 ai.Message。
func (g *AvatarReplyGraph) generateWithCustomProvider(ctx context.Context, input *AvatarReplyContext) (string, error) {
	messageList, err := g.renderPrompt(ctx, input)
	if err != nil {
		return "", err
	}
	return g.aiService.GetCompletionWithProviderConfig(ai.TaskTypeChat, einoMessagesToAIMessages(messageList), input.CustomProvider.ProviderName, input.CustomProvider.Config)
}

// renderPrompt 渲染分身 prompt（buildTemplateVars + template.Format），图内/图外各路径共用；
// 未编译（template 未设置）时返回明确错误。
func (g *AvatarReplyGraph) renderPrompt(ctx context.Context, input *AvatarReplyContext) ([]*schema.Message, error) {
	if g.template == nil {
		return nil, fmt.Errorf("Graph 未编译，请先调用 BuildGraph")
	}
	vars := g.buildTemplateVars(input)
	messageList, err := g.template.Format(ctx, vars)
	if err != nil {
		return nil, fmt.Errorf("渲染分身 prompt 失败: %w", err)
	}
	return messageList, nil
}

// completeReply 用整组 aiMessages 完成一次非流式生成并统一截断：自选模型走临时 provider、
// 否则走系统配置。返回回复 + 命中的知识来源。注意：自选模型失败不在此回退系统默认——
// executeWithSources 的「自选失败回退 runnable.Invoke」契约保留在调用方。
func (g *AvatarReplyGraph) completeReply(input *AvatarReplyContext, aiMessages []ai.Message) (string, []KnowledgeSource, error) {
	var reply string
	var err error
	if input.CustomProvider != nil {
		reply, err = g.aiService.GetCompletionWithProviderConfig(ai.TaskTypeChat, aiMessages, input.CustomProvider.ProviderName, input.CustomProvider.Config)
	} else {
		reply, err = g.aiService.GetCompletion(ai.TaskTypeChat, aiMessages)
	}
	if err != nil {
		return "", input.Sources, err
	}
	return truncateReply(input, reply), input.Sources, nil
}

// truncateReply 按分身 MaxReplyLength 对回复做 rune 截断（避免中文等变长 UTF-8 在字节中切断），
// 0 表示不截断。
func truncateReply(input *AvatarReplyContext, reply string) string {
	if maxRunes := avatarMaxReplyChars(input.ReplyStrategy.MaxReplyLength); maxRunes > 0 {
		runes := []rune(reply)
		if len(runes) > maxRunes {
			reply = strings.TrimSpace(string(runes[:maxRunes])) + "…"
		}
	}
	return reply
}

// injectSingleImage 把 prompt 中的 user 消息替换为携带单张图片的 MultiContent 多模态消息：
// 原文本保留，追加 instruct 指令文本，并携带 imageURL 的 base64 data URL（供视觉模型识别）。
// 与 injectMultiImage 共用同一「user 消息替换」骨架。
func injectSingleImage(messageList []*schema.Message, imageURL string, instruct string) []*schema.Message {
	out := make([]*schema.Message, 0, len(messageList))
	for _, m := range messageList {
		if string(m.Role) != "user" {
			out = append(out, m)
			continue
		}
		imgText := "\n\n" + instruct
		out = append(out, &schema.Message{
			Role:    schema.User,
			Content: m.Content + imgText,
			MultiContent: []schema.ChatMessagePart{
				{Type: schema.ChatMessagePartTypeText, Text: m.Content + imgText},
				{Type: schema.ChatMessagePartTypeImageURL, ImageURL: &schema.ChatMessageImageURL{URL: imageURL}},
			},
		})
	}
	return out
}

// injectMultiImage 与 injectSingleImage 同骨架，但把 user 消息替换为携带批内多张图片的
// MultiContent（多个 image_url part），供「合并窗口连发一批消息」的多模态批量路径使用。
func injectMultiImage(messageList []*schema.Message, imageURLs []string, instruct string) []*schema.Message {
	out := make([]*schema.Message, 0, len(messageList))
	for _, m := range messageList {
		if string(m.Role) != "user" {
			out = append(out, m)
			continue
		}
		imgText := "\n\n" + instruct
		parts := []schema.ChatMessagePart{
			{Type: schema.ChatMessagePartTypeText, Text: m.Content + imgText},
		}
		for _, url := range imageURLs {
			parts = append(parts, schema.ChatMessagePart{
				Type:     schema.ChatMessagePartTypeImageURL,
				ImageURL: &schema.ChatMessageImageURL{URL: url},
			})
		}
		out = append(out, &schema.Message{
			Role:         schema.User,
			Content:      m.Content + imgText,
			MultiContent: parts,
		})
	}
	return out
}

// ExecuteStream 以流式生成分身回复（供"帮我回复"草稿模式使用）。
// 与 Execute 的差异：1) 不走编译图，渲染模板后直接调 chatModel.Stream；2) 忽略 SkipReply
// （用户主动要草稿，即使命中"知识范围外静默"也照常生成）；3) 不做 MaxReplyLength 截断
// （草稿进输入框由用户自行编辑，截断无意义）。
func (g *AvatarReplyGraph) ExecuteStream(ctx context.Context, userID uint, conversationID uint, message string, historyBefore *time.Time, preloaded *model.AvatarConfig) (*schema.StreamReader[*schema.Message], error) {
	input := &AvatarReplyContext{
		Message:        message,
		ConversationID: conversationID,
		UserID:         userID,
		HistoryBefore:  historyBefore,
	}

	if err := g.prepare(ctx, input, preloaded); err != nil {
		return nil, err
	}
	// 草稿模式：忽略 SkipReply（用户主动要草稿，不该因"超知识范围"静默）

	messageList, err := g.renderPrompt(ctx, input)
	if err != nil {
		return nil, err
	}

	return g.executeStream(ctx, input, messageList)
}

// ExecuteStreamWithImageSources 供"帮我回复"草稿目标消息是图片时流式生成草稿：
// 按 fileID 读出的 base64 data URL 注入最后一条 user 消息的 MultiContent（text + image_url），
// 使草稿基于图片内容生成。与 ExecuteStream 同走图外渲染 + 流式，仅多一步图片注入；
// 读图失败由调用方（AvatarService）降级为纯文本草稿，不落到这里。
func (g *AvatarReplyGraph) ExecuteStreamWithImageSources(ctx context.Context, userID uint, conversationID uint, message string, imageURL string, imageName string, historyBefore *time.Time, preloaded *model.AvatarConfig) (*schema.StreamReader[*schema.Message], error) {
	input := &AvatarReplyContext{
		Message:        message,
		ConversationID: conversationID,
		UserID:         userID,
		HistoryBefore:  historyBefore,
	}

	if err := g.prepare(ctx, input, preloaded); err != nil {
		return nil, err
	}

	messageList, err := g.renderPrompt(ctx, input)
	if err != nil {
		return nil, err
	}

	// 把 user 消息替换为携带图片的 MultiContent 多模态消息：原文本保留，追加草稿语境
	// 的图片识别指令，并携带 base64 data URL。由 executeStream（EinoChatModel.Stream →
	// einoMessagesToAIMessages）提取图片透传给模型。
	messageList = injectSingleImage(messageList, imageURL, fmt.Sprintf("📷 对方发来了一张图片「%s」，请识别图片内容并结合以上对话起草一条回复。", imageName))

	return g.executeStream(ctx, input, messageList)
}

// executeStream 草稿模式流式生成的公共核心：按 messageList（含图片时已由调用方注入
// MultiContent）走流式。自选模型分支用 einoMessagesToAIMessages 提取图片透传（手拼
// ai.Message 会丢 ImageURL），与系统配置的 EinoChatModel 路径行为一致。
func (g *AvatarReplyGraph) executeStream(ctx context.Context, input *AvatarReplyContext, messageList []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
	// 自选模型走临时 provider 流式；否则走系统配置的 EinoChatModel。
	if input.CustomProvider != nil {
		aiMessages := einoMessagesToAIMessages(messageList)
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
				input.Sources = append(input.Sources, KnowledgeSource{Source: "notes", Title: r.Metadata["title"], Score: r.Score, ID: r.DocID, Snippet: r.Content})
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
						input.Sources = append(input.Sources, KnowledgeSource{Source: "knowledge", Title: r.Metadata["title"], Score: r.Score, ID: r.DocID, Snippet: r.Content})
					}
					groupKnowledge = "【群知识库】\n" + strings.Join(parts, "\n\n")

					// 知识图谱关系扩展（GraphRAG MVP）：在向量命中的文档上做 GraphBFS，
					// 追加"该文档关联的实体/文档"，补足"XX 关联了谁/哪些文档"这类关系问答。
					// 无图谱数据时 ExpandGraphKnowledge 返回空串，不影响 normal 答复。
					if graphCtx := g.groupDocSvc.ExpandGraphKnowledge(group.ID, input.Message, 3); graphCtx != "" {
						groupKnowledge += "\n\n" + graphCtx
					}
				}
			}
		}
	}
	input.GroupKnowledge = groupKnowledge

	memoryCtx := ""
	bestMemoryScore := 0.0
	if g.memorySvc != nil {
		memoryResults, err := g.memorySvc.Recall(input.UserID, input.Message, 2)
		if err == nil && len(memoryResults) > 0 {
			var parts []string
			for _, r := range memoryResults {
				parts = append(parts, r.Content)
				input.Sources = append(input.Sources, KnowledgeSource{Source: "memory", Score: r.Score, ID: r.DocID, Snippet: r.Content})
				if r.Score > bestMemoryScore {
					bestMemoryScore = r.Score
				}
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
		history = g.getConversationHistory(input.ConversationID, 10, input.Message, input.HistoryBefore)
	}
	input.History = history

	// 自选模型：配置了「使用自定义模型」时解析出 provider，供 Execute/ExecuteStream 走临时 provider 生成。
	// 解析失败（配置不存在/密钥解密失败等）时置为 nil，静默回退系统默认配置，不阻断回复。
	if !input.Config.UseSystemConfig && input.Config.ModelConfigID != nil {
		input.CustomProvider = g.resolveCustomProvider(input.UserID, *input.Config.ModelConfigID)
	}

	// 范围外静默（硬门控）：ReplyOutOfScope=false 时，分身只应回复「与自身知识/上下文相关」的消息。
	// 有笔记/群知识命中 → 属于范围内，正常回复；无任何知识命中 → 属于范围外，直接静默。
	// 记忆参与判定但需过相关度阈值（≥0.5）：低分噪音记忆不算"知识命中"，避免无关问题
	// 因 Recall 出历史记忆而误回复；高分记忆（用户确实问过相关内容）则正常放行。
	// 任务不参与范围内判定（任务只是附加注入的知识，不应让"有任务就什么都回"旁路门控）。
	memoryHit := memoryCtx != "" && bestMemoryScore >= g.memoryRecallThreshold()
	hasKnowledge := noteCtx != "" || groupKnowledge != "" || memoryHit
	if !input.ReplyStrategy.ReplyOutOfScope && !hasKnowledge {
		input.SkipReply = true
	}

	return nil
}

// resolveCustomProvider 根据分身配置的 modelConfigID 解析出自选模型 provider。
// 返回 nil 表示未命中或解析失败（回退系统默认配置）。userID 用于校验该 AIConfig 归属。
// 解析逻辑与 bot 共用 ai_config_service.go 的 resolveUserAIConfigProvider，避免两处分叉漂移。
// 仅需 db，不依赖任何注入的服务（历史上曾误挂 aiConfigSvc，已消除该死字段）。
func (g *AvatarReplyGraph) resolveCustomProvider(userID, configID uint) *customProvider {
	return resolveUserAIConfigProvider(g.db, userID, configID)
}

// buildCustomProviderExtraParams 构建分身自选模型的 ExtraParams。
// 仅透传有效值：
//   - max_tokens>0 才传（0 对部分 provider 会被拒绝或解释为无限制，且 max_tokens=0 无确定性语义）
//   - temperature 一律透传，包括 0：AIConfig.Temperature 在 DB 层默认 0.7，读到的 0 必然是用户
//     显式设置的确定性输出（"未设置"会落为 0.7，绝不会是 0），若跳过则用户想要的 temp=0 被
//     provider 默认 0.7 静默覆盖
func buildCustomProviderExtraParams(maxTokens int, temperature float64) map[string]interface{} {
	params := map[string]interface{}{}
	if maxTokens > 0 {
		params["max_tokens"] = maxTokens
	}
	params["temperature"] = temperature
	return params
}

// needReplyForOutOfScope 已废弃：范围外静默已改为硬门控（无知识命中即 SkipReply），
// 不再需要 LLM 二次判断。保留此函数已无调用点，故移除。

func (g *AvatarReplyGraph) getConversationHistory(conversationID uint, limit int, triggerMessage string, before *time.Time) string {
	// 不再一刀切排除 avatar 自回复：近期（selfTurnWindow 内）的 avatar 自回复保留作
	// 多轮指代锚点（用户可能追问”你刚说的”），只滤掉远期自回复（自我复制污染源）。
	// 为此多取一段（limit+1），在内存里丢弃远期自回复 + 触发消息后仍尽量满足 limit。
	// before 非 nil（草稿模式锚定到目标消息）时，只取该时间之前的历史——目标可能不是
	// 会话最新一条，按”整个会话最近 N 条”会把目标之后的后续对话混进来导致答非所问。
	query := g.db.Where("conversation_id = ?", conversationID).
		Where("type = ?", "text")
	if before != nil {
		query = query.Where("created_at < ?", *before)
	}
	var messages []model.Message
	query.Order("created_at DESC").
		Limit(limit + 8).
		Find(&messages)

	// 在 Go 侧筛掉远期自身回复；近期自身回复保留。
	filtered := messages[:0]
	for _, m := range messages {
		if m.Origin == "avatar" && !isNearSelf(m) {
			continue
		}
		filtered = append(filtered, m)
	}
	messages = filtered

	// 诊断：统计本会话里被过滤排除的『分身自己回复』条数，确认“潜在多轮失忆”
	// 是否真实发生（分身刚说的话是否被历史排除、导致追问“你刚说的”断链）。
	var selfFiltered int64
	g.db.Model(&model.Message{}).
		Where("conversation_id = ? AND origin = ?", conversationID, "avatar").
		Count(&selfFiltered)
	// 已滤远期 = 全量 avatar 回复 - 保留下来的近期 avatar 回复
	keptAvatar := 0
	for _, m := range messages {
		if m.Origin == "avatar" {
			keptAvatar++
		}
	}
	logHistoryDiagnostics("分身/历史", conversationID, messages, nil)
	log.Printf("[ContextDiag] 分身/历史 conv=%d avatar总=%d 保留=%d 已滤=%d",
		conversationID, selfFiltered, keptAvatar, int(selfFiltered)-keptAvatar)
	// 纳入窗口聚合：分身过滤有效性（保留率/已滤率）供定时快照判断是否仍失忆
	aggregateAvatarFilter(int(selfFiltered), keptAvatar, int(selfFiltered)-keptAvatar)

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
	case "very_long":
		return "回复可以较详细，控制在 400 字以内"
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
	case "very_long":
		return 400
	case "long":
		return 2000
	default:
		return 0
	}
}
