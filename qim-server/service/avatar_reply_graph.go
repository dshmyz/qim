package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/aiprompt"
	"github.com/dshmyz/qim/qim-server/pkg/logger"

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
	// BypassScope 置位表示本条回复是用户主动触发的宽松路径（草稿"帮我回复"/图片识别），
	// 不受「知识范围外」策略约束：prompt 范围策略按宽松渲染，即便命中范围外也正常生成。
	BypassScope bool
}

// 用户自选模型在回复阶段的临时描述统一由 ai_config_service.go 的 customProvider 承担
// （与 resolveUserAIConfigProvider 共用），此处不再单独定义 customModelProvider 以免重复漂移。

type AvatarReplyGraph struct {
	aiService   *ai.AIService
	db          *gorm.DB
	noteSvc     *NoteVectorService
	memorySvc   *AvatarMemoryService
	groupDocSvc *GroupDocumentService
	// thresholdSvc 阈值读取服务；nil 时记忆召回门槛用默认 0.5（向后兼容）。
	thresholdSvc *AiThresholdService
	// reranker 知识相关性二次判定器（与群助手/Bot 共用 LLMReranker）；nil 时不做判定（保留全部）。
	reranker KnowledgeReranker
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
		reranker:    NewLLMReranker(aiService),
	}
}

// SetReranker 替换相关性判定器（测试注入 mock 用）。传 nil 可关闭判定，走纯阈值模式。
func (g *AvatarReplyGraph) SetReranker(r KnowledgeReranker) {
	g.reranker = r
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

// knowledgeScoreFloor 返回知识来源硬下限：低于此分数的笔记/群知识召回视为噪音，
// 不注入 prompt、不进"依据"徽章、也不计入范围判定。未注入阈值服务时回退默认 0.3。
func (g *AvatarReplyGraph) knowledgeScoreFloor() float64 {
	if g.thresholdSvc != nil {
		return g.thresholdSvc.GetFloat("ai.knowledge_score_threshold", 0.3)
	}
	return 0.3
}

// selectTopByScore 过滤 + 排序 + 取前 K：先滤掉低于下限的噪音，再按分数降序，取前 k 条。
// 供分身笔记/群知识注入共用——宽召回（TopK 大于最终注入数）后必须收敛到 Top-K，
// 否则 rerank 通过的低分命中会一股脑塞进 prompt。纯函数，便于单测。
func selectTopByScore(snippets []KnowledgeSnippet, floor float64, k int) []KnowledgeSnippet {
	out := make([]KnowledgeSnippet, 0, len(snippets))
	for _, s := range snippets {
		if s.Score >= floor {
			out = append(out, s)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > k {
		out = out[:k]
	}
	return out
}

// historyLimit 返回注入 prompt 的会话历史条数上限：接入 ai.context_history_limit 配置
// （后台可调），未注入阈值服务时回退默认 10（与分身历史默认一致）。
func (g *AvatarReplyGraph) historyLimit() int {
	if g.thresholdSvc != nil {
		return g.thresholdSvc.GetInt("ai.context_history_limit", 10)
	}
	return 10
}

// recentAIMessagesLimit 返回上下文中保留的近期 AI 回复条数上限（防自我复制）：
// 接入 ai.recent_ai_messages_limit 配置（后台可调），未注入阈值服务时回退默认 5。
func (g *AvatarReplyGraph) recentAIMessagesLimit() int {
	if g.thresholdSvc != nil {
		return g.thresholdSvc.GetInt("ai.recent_ai_messages_limit", 5)
	}
	return 5
}

// BuildGraph 兼容保留（空操作）：分身回复图已不再使用 Eino 编译图——消息块在 renderPrompt
// 代码级拼装（与群助手 buildHistoryMessages 同构），无需预编译，也消除了"编译图模板"与
// "图外渲染"两套 prompt 拼装的分叉。保留签名以兼容 avatar_service 初始化/重建流程与测试。
func (g *AvatarReplyGraph) BuildGraph() error {
	return nil
}

// buildSystemPrompt 构造分身 system 消息（人设 + 补充说明 + 回复要求 + 范围策略行）。
// 原模板变量拼装逻辑（buildTemplateVars）收敛于此，供 renderPrompt 拼装消息块使用。
func (g *AvatarReplyGraph) buildSystemPrompt(input *AvatarReplyContext) string {
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

	// 范围策略行：ReplyOutOfScope=false（且非用户主动触发的宽松路径）时严格基于资料作答，
	// 资料不足就明说不回，不再"给出理解"自由发挥——与"不回答知识范围外"的设置联动。
	scopePolicy := scopePolicyLine(input.ReplyStrategy.ReplyOutOfScope || input.BypassScope)

	userName := input.User.Nickname
	if userName == "" {
		userName = input.User.Username
	}

	return fmt.Sprintf(`你是%s的AI分身，需要以TA的身份回复消息。

%s
%s
%s
【回复要求】
- 以第一人称回复，就像你就是这个人
- 保持自然的对话风格
- %s
- 回答必须优先基于上方【相关笔记知识】【群知识库】【相关记忆】等资料。
%s`, userName, timeInfo, personaSection, supplementSection, avatarLengthHint(input.ReplyStrategy.MaxReplyLength), scopePolicy)
}

// scopePolicyLine 按「是否回复知识范围外」渲染范围策略行：
// 宽松（范围外也回 / 用户主动触发的草稿·图片路径）→ 允许资料不足时给出理解；
// 严格（范围外静默）→ 只准基于资料作答，资料不足直接说明，不自由发挥。
// 与范围外静默门控（prepare 的 SkipReply 判定）联动，堵住"弱命中放行后模型乱发挥"。
func scopePolicyLine(relaxed bool) string {
	if relaxed {
		return "- 若资料不足以回答，请明确说明资料不足并给出你的理解，切勿假装引用或编造。"
	}
	return "- 回答必须严格基于上方提供的资料；若资料不足以回答当前问题，请直接说明「我这边没有相关资料」，不要使用资料之外的信息作答。"
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
		logger.WithModule("diag").Info(fmt.Sprintf("[Diag] 分身命中不回复策略，跳过 LLM: userID=%d convID=%d", userID, conversationID))
		return "", nil, nil
	}

	// 统一走 renderPrompt 消息块拼装 + 直连 GetCompletion（不再经 Eino 编译图，
	// 与图片/草稿路径同一条 prompt 拼装逻辑，消除双套分叉）。
	startTime := time.Now()
	messageList, err := g.renderPrompt(ctx, input)
	if err != nil {
		return "", nil, err
	}
	aiMsgs := einoMessagesToAIMessages(messageList)
	var reply string
	if input.CustomProvider != nil {
		// 自选模型：用用户自选 provider 生成，与其它路径共用同一套消息块/截断逻辑。
		reply, err = g.aiService.GetCompletionWithProviderConfig(ai.TaskTypeChat, aiMsgs, input.CustomProvider.ProviderName, input.CustomProvider.Config)
		if err != nil {
			// 生成期失败（密钥失效/配额/网络错/provider 未配置）→ 回退系统默认，
			// 兑现「回退系统默认…不阻断回复」契约——单条自定义配置出问题不应让分身整条不复回。
			// 系统回退同样失败时才真正返回错误。
			logger.WithModule("diag").Warn(fmt.Sprintf("[Diag] 分身自选模型失败回退系统默认: userID=%d err=%v", input.UserID, err))
			reply, err = g.aiService.GetCompletion(ai.TaskTypeChat, aiMsgs)
		}
	} else {
		reply, err = g.aiService.GetCompletion(ai.TaskTypeChat, aiMsgs)
	}
	if err != nil {
		return "", nil, err
	}

	logger.WithModule("diag").Info(fmt.Sprintf("[Diag] 分身生成回复耗时: %v", time.Since(startTime)))

	reply = truncateReply(input, reply)

	return reply, input.Sources, nil
}

// ExecuteWithImageSources 供分身识别图片触发消息：读图成功后调用此方法生成回复。
// 差异点：1) 走 renderPrompt 拼装的消息块，把最后一条 user 消息替换为携带图片的
// MultiContent 多模态消息（text + base64 data URL，经 einoMessagesToAIMessages 提取为
// ai.Message.ImageURL），由 GetCompletion 以 OpenAI image_url 数组格式交给视觉模型识别；
// 2) 忽略 SkipReply（图片消息走视觉识别，不因"知识范围外"而静默，与"尽力而为失败则跳过"
// 的降级语义配合——能看图就回，看不了/模型不支持则由 worker 跳过）。
// 返回与 ExecuteWithSources 同构：回复 + 命中的知识来源。
func (g *AvatarReplyGraph) ExecuteWithImageSources(ctx context.Context, userID uint, conversationID uint, message string, imageURL string, imageName string, preloaded *model.AvatarConfig) (string, []KnowledgeSource, error) {
	input := &AvatarReplyContext{
		Message:        message,
		ConversationID: conversationID,
		UserID:         userID,
	}
	// 图片识别：用户主动触发，忽略范围外静默（prompt 范围策略按宽松渲染）
	input.BypassScope = true

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
	// 批内图片识别：用户主动触发，忽略范围外静默（prompt 范围策略按宽松渲染）
	input.BypassScope = true

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

// renderPrompt 拼装分身 prompt 消息块（所有路径共用：自动回复/图片/草稿/自选模型）：
// system 消息 + 每个非空知识来源独立成「user 上下文块 + assistant 确认对」（对齐群助手
// buildContextBlocks），最后一条 user 消息为"对方说：{Message}"。成对出现避免连续 user
// 消息（部分 provider 拒绝同角色连续消息），也让模型能区分"上下文"与"应答"，
// 缓解长上下文 lost-in-the-middle。无需预编译，拼装失败时返回明确错误。
func (g *AvatarReplyGraph) renderPrompt(ctx context.Context, input *AvatarReplyContext) ([]*schema.Message, error) {
	userName := input.User.Nickname
	if userName == "" {
		userName = input.User.Username
	}

	msgs := []*schema.Message{
		{Role: schema.System, Content: g.buildSystemPrompt(input)},
	}

	// 各知识来源块（含自带标题头）：非空才注入
	blocks := []struct {
		content string
		ack     string
	}{
		{input.NoteContext, "收到笔记信息，我会优先参考。"},
		{input.GroupKnowledge, "收到群知识库信息，我会优先参考。"},
		{input.MemoryContext, "收到记忆信息，我会优先参考。"},
		{input.TaskContext, "收到任务信息，我会优先参考。"},
	}
	for _, b := range blocks {
		if b.content == "" {
			continue
		}
		msgs = append(msgs,
			&schema.Message{Role: schema.User, Content: b.content},
			&schema.Message{Role: schema.Assistant, Content: b.ack},
		)
	}
	// 历史单独成块（表头在此拼接，须先判空——表头本身非空会导致空历史也产出空块）
	if input.History != "" {
		msgs = append(msgs,
			&schema.Message{Role: schema.User, Content: "【对话历史】\n" + input.History},
			&schema.Message{Role: schema.Assistant, Content: "已了解对话历史。"},
		)
	}

	msgs = append(msgs, &schema.Message{
		Role:    schema.User,
		Content: fmt.Sprintf("对方说：%s\n\n请以%s的身份回复：", input.Message, userName),
	})
	return msgs, nil
}

// completeReply 用整组 aiMessages 完成一次非流式生成并统一截断：自选模型走临时 provider、
// 否则走系统配置。返回回复 + 命中的知识来源。注意：自选模型失败不在此回退系统默认——
// executeWithSources 的「自选失败回退系统」契约保留在调用方。
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
	out := make([]*schema.Message, len(messageList))
	copy(out, messageList)
	// 消息块结构下存在多个 user 消息（知识块 + 提问），图片必须注入最后一条 user（"对方说"），
	// 而非首条知识块——倒序找最后一条 user 替换为携带图片的 MultiContent。
	for i := len(out) - 1; i >= 0; i-- {
		m := out[i]
		if string(m.Role) != "user" {
			continue
		}
		imgText := "\n\n" + instruct
		out[i] = &schema.Message{
			Role:    schema.User,
			Content: m.Content + imgText,
			MultiContent: []schema.ChatMessagePart{
				{Type: schema.ChatMessagePartTypeText, Text: m.Content + imgText},
				{Type: schema.ChatMessagePartTypeImageURL, ImageURL: &schema.ChatMessageImageURL{URL: imageURL}},
			},
		}
		break
	}
	return out
}

// injectMultiImage 与 injectSingleImage 同骨架：把最后一条 user 消息替换为携带批内多张图片的
// MultiContent（多个 image_url part），供「合并窗口连发一批消息」的多模态批量路径使用。
func injectMultiImage(messageList []*schema.Message, imageURLs []string, instruct string) []*schema.Message {
	out := make([]*schema.Message, len(messageList))
	copy(out, messageList)
	for i := len(out) - 1; i >= 0; i-- {
		m := out[i]
		if string(m.Role) != "user" {
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
		out[i] = &schema.Message{
			Role:         schema.User,
			Content:      m.Content + imgText,
			MultiContent: parts,
		}
		break
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
	// 草稿模式：用户主动要草稿，忽略 SkipReply（即使命中"知识范围外"也照常生成），
	// 且范围策略按宽松渲染（资料不足允许给出理解，供用户编辑后再发）
	input.BypassScope = true

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
	// 图片草稿：用户主动要草稿，忽略范围外静默（prompt 范围策略按宽松渲染）
	input.BypassScope = true

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

	// 会话历史先于知识检索组装：既注入 prompt，也作为「上下文感知检索」的 query 前缀——
	// 多轮追问（"那后来呢？"）借此携带话题上下文，否则向量召回必然失败。
	history := ""
	// ConversationHistory nil（存量配置未显式设置）按默认 true 处理，避免升级后静默丢失历史
	historyEnabled := true
	if input.KnowledgeScope.ConversationHistory != nil {
		historyEnabled = *input.KnowledgeScope.ConversationHistory
	}
	if historyEnabled && input.ConversationID > 0 {
		history = g.getConversationHistory(input.ConversationID, g.historyLimit(), input.Message, input.HistoryBefore)
	}
	input.History = history
	// 检索 query：最近 4 行对话 + 当前提问，让笔记/群知识/记忆的 embedding 携带话题上下文
	retrievalQuery := contextualQuery(input.History, input.Message, 4)

	// 检索诊断计数（供末尾汇总日志：命中几条 vs 实际注入几条，定位"为什么没标/乱回复"）
	noteHits, groupHits, memoryHits := 0, 0, 0

	noteCtx := ""
	// 笔记检索受 Notes 开关门控，与其他知识来源对齐（否则 Notes 关不掉，且会破坏 docs-only 的范围控制）
	if g.noteSvc != nil && input.KnowledgeScope.Notes {
		// 宽召回（TopK 6）→ LLM 相关性二次判定 → 硬下限过滤 → 取前 3：
		// 与群助手/Bot 的"宽召回+精排"范式对齐——此前召回 3 条再 rerank 形同虚设，
		// 精排没有候选集可以筛。
		noteResults, err := g.noteSvc.SearchNotes(input.UserID, retrievalQuery, 6)
		if err == nil && len(noteResults) > 0 {
			noteHits = len(noteResults)
			// 先转成统一 snippet 供 LLM 相关性二次判定（与群助手/Bot 同款）
			snippets := make([]KnowledgeSnippet, 0, len(noteResults))
			for _, r := range noteResults {
				snippets = append(snippets, KnowledgeSnippet{
					Title:    r.Metadata["title"],
					Content:  r.Content,
					Score:    r.Score,
					Source:   "notes",
					DocID:    r.DocID,
					Metadata: r.Metadata,
				})
			}
			snippets = filterSnippetsByReranker(g.reranker, g.thresholdSvc, 0, retrievalQuery, snippets)
			picked := selectTopByScore(snippets, g.knowledgeScoreFloor(), 3)
			var parts []string
			for _, snip := range picked {
				parts = append(parts, fmt.Sprintf("[笔记: %s]\n%s", snip.Title, snip.Content))
				input.Sources = append(input.Sources, KnowledgeSource{Source: "notes", Title: snip.Title, Score: snip.Score, ID: snip.DocID, Snippet: snip.Content})
			}
			if len(parts) > 0 {
				noteCtx = "【相关笔记知识】\n" + strings.Join(parts, "\n\n")
			}
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
				// 与笔记同款宽召回（TopK 4 → rerank → floor → 取前 2）
				results, err := g.groupDocSvc.SearchKnowledge(group.ID, retrievalQuery, 4)
				if err == nil && len(results) > 0 {
					groupHits = len(results)
					snippets := make([]KnowledgeSnippet, 0, len(results))
					for _, r := range results {
						snippets = append(snippets, KnowledgeSnippet{
							Title:    r.Metadata["title"],
							Content:  r.Content,
							Score:    r.Score,
							Source:   "knowledge",
							DocID:    r.DocID,
							Metadata: r.Metadata,
						})
					}
					snippets = filterSnippetsByReranker(g.reranker, g.thresholdSvc, 0, retrievalQuery, snippets)
					picked := selectTopByScore(snippets, g.knowledgeScoreFloor(), 2)
					var parts []string
					for _, snip := range picked {
						parts = append(parts, fmt.Sprintf("[群知识库: %s]\n%s", snip.Title, snip.Content))
						input.Sources = append(input.Sources, KnowledgeSource{Source: "knowledge", Title: snip.Title, Score: snip.Score, ID: snip.DocID, Snippet: snip.Content})
					}
					if len(parts) > 0 {
						groupKnowledge = "【群知识库】\n" + strings.Join(parts, "\n\n")

						// 知识图谱关系扩展（GraphRAG MVP）：在向量命中的文档上做 GraphBFS，
						// 追加"该文档关联的实体/文档"，补足"XX 关联了谁/哪些文档"这类关系问答。
						// 无图谱数据时 ExpandGraphKnowledge 返回空串；仅作已命中知识之上的增强，
						// 不在无直接命中时单独充当"范围内"依据。
						if graphCtx := g.groupDocSvc.ExpandGraphKnowledge(group.ID, input.Message, 3); graphCtx != "" {
							groupKnowledge += "\n\n" + graphCtx
						}
					}
				}
			}
		}
	}
	input.GroupKnowledge = groupKnowledge

	memoryCtx := ""
	// 记忆受 Memory 开关门控（nil 默认启用）：关闭时跳过召回，不注入不进徽章也不参与范围判定。
	// 判定复用 model.AvatarKnowledgeScope.MemoryEnabled，与后台学习写入（maybeRememberSenderMessage）
	// 共用同一语义——"关掉记忆 = 既不读也不学"。
	if g.memorySvc != nil && input.KnowledgeScope.MemoryEnabled() {
		// 记忆召回同样用上下文感知 query（TopK 3）：追问场景下历史话题能让记忆命中
		memoryResults, err := g.memorySvc.Recall(input.UserID, retrievalQuery, 3)
		if err == nil && len(memoryResults) > 0 {
			memoryHits = len(memoryResults)
			// 记忆只按召回门槛（默认 0.5）过滤注入：低分噪音记忆不进 prompt，
			// 避免"被记忆干扰"。过滤在注入前做，不进 Sources（依据徽章）也不参与范围判定。
			var parts []string
			threshold := g.memoryRecallThreshold()
			for _, r := range memoryResults {
				if r.Score < threshold {
					continue
				}
				parts = append(parts, r.Content)
				input.Sources = append(input.Sources, KnowledgeSource{Source: "memory", Score: r.Score, ID: r.DocID, Snippet: r.Content})
			}
			if len(parts) > 0 {
				memoryCtx = "【相关记忆】\n" + strings.Join(parts, "\n\n")
			}
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

	// 自选模型：配置了「使用自定义模型」时解析出 provider，供 Execute/ExecuteStream 走临时 provider 生成。
	// 解析失败（配置不存在/密钥解密失败等）时置为 nil，静默回退系统默认配置，不阻断回复。
	if !input.Config.UseSystemConfig && input.Config.ModelConfigID != nil {
		input.CustomProvider = g.resolveCustomProvider(input.UserID, *input.Config.ModelConfigID)
	}

	// 范围外静默（硬门控）：ReplyOutOfScope=false 时，分身只应回复「与自身知识相关」的消息。
	// 有笔记/群知识命中（且过 0.3 硬下限与 LLM 相关性校验）→ 属于范围内，正常回复；
	// 无任何知识命中 → 属于范围外，直接静默。记忆不再单独放行：选了"不回答知识范围外"，
	// 就不该因"以前聊过类似话题"（记忆召回命中）被放行——记忆只作辅助上下文注入。
	// 任务不参与范围内判定（任务只是附加注入的知识，不应让"有任务就什么都回"旁路门控）。
	hasKnowledge := noteCtx != "" || groupKnowledge != ""
	if !input.ReplyStrategy.ReplyOutOfScope && !hasKnowledge {
		input.SkipReply = true
	}

	// 检索诊断汇总（独立 diag.log）：每次回复命中几条→实际注入几条、是否静默、标记了几条依据。
	// 排查"为什么没标/乱回复"：命中数 > 注入数即存在被 rerank/硬下限/Top-K 滤掉的命中，
	// 逐条被滤原因见 diag.log 中 filterSnippetsByReranker 的"LLM 判定不相关，已过滤"。
	logger.WithModule("diag").Info("[Diag] 分身检索",
		"conv", input.ConversationID, "user", input.UserID,
		"msg", truncateRunes(input.Message, 60),
		"notes", fmt.Sprintf("%d→%d", noteHits, countSourcesByType(input.Sources, "notes")),
		"group", fmt.Sprintf("%d→%d", groupHits, countSourcesByType(input.Sources, "knowledge")),
		"memory", fmt.Sprintf("%d→%d", memoryHits, countSourcesByType(input.Sources, "memory")),
		"skip", input.SkipReply, "sources", len(input.Sources))

	return nil
}

// countSourcesByType 统计 Sources 中指定来源类型（notes/knowledge/memory）的条数，诊断汇总用。
func countSourcesByType(sources []KnowledgeSource, typ string) int {
	n := 0
	for _, s := range sources {
		if s.Source == typ {
			n++
		}
	}
	return n
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

	// 在 Go 侧筛掉远期自身回复；近期自身回复保留（最多 recentAIMessagesLimit 条，
	// 超出按远期折叠——接入 ai.recent_ai_messages_limit，后台可调防自我复制上限）。
	filtered := messages[:0]
	keptSelf := 0
	maxSelf := g.recentAIMessagesLimit()
	for _, m := range messages {
		if m.Origin == "avatar" {
			if !isNearSelf(m) || keptSelf >= maxSelf {
				continue
			}
			keptSelf++
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
	logger.WithModule("diag").Info(fmt.Sprintf("[ContextDiag] 分身/历史 conv=%d avatar总=%d 保留=%d 已滤=%d",
		conversationID, selfFiltered, keptAvatar, int(selfFiltered)-keptAvatar))
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
		// 逐条截断历史消息：单条超长（粘贴的长文本/合并转发）会被 rune 截到 800 字，
		// 避免一条消息撑爆 prompt 并稀释注意力。尾巴通常含关键信息，故保留开头。
		parts = append(parts, fmt.Sprintf("%s: %s", sender.Nickname, truncateRunes(msg.Content, 800)))
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
