package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/aiprompt"
	"github.com/dshmyz/qim/qim-server/pkg/mention"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/dshmyz/qim/qim-server/utils"
	"github.com/dshmyz/qim/qim-server/ws"
	"gorm.io/gorm"
)

// SmartReplyEngine 智能回复引擎
type SmartReplyEngine struct {
	aiService        *ai.AIService
	intentDetector   *ai.IntentDetector
	knowledgeSvc     *KnowledgeService
	unifiedKnowledge *service.UnifiedKnowledgeService
	memorySvc        *service.AvatarMemoryService
	groupMemorySvc   *service.GroupMemoryService
	promptBuilder    *SmartPromptBuilder
	messageSender    *WebSocketMessageSender
	avatarWorkerPool *service.AvatarWorkerPool
	avatarTriggerSvc AvatarTriggerDecider
	smartReplyGraph  *service.SmartReplyGraph
	convReplyWorker  *service.ReplyOrchestrator // 群助手回复的并发控制（限并发+会话串行）
}

// AvatarTriggerDecider decides whether a configured avatar should reply.
// The interface keeps the real-time trigger path independent from a concrete AI provider.
// DecideReply 是触发决策的唯一入口：排除列表/时间窗/模式分发全部收敛于此，
// 实时路径 shouldTriggerAvatar 与预览接口 CheckTrigger 共用，避免双套判断。
type AvatarTriggerDecider interface {
	DecideReply(config model.AvatarConfig, conversationID uint, message string, senderName string, isGroupChat bool, mentionUserIDs []uint) (bool, string, error)
}

// NewSmartReplyEngine 创建智能回复引擎
func NewSmartReplyEngine(aiService *ai.AIService, detector *ai.IntentDetector) *SmartReplyEngine {
	knowledgeSvc := NewKnowledgeService(aiService)
	return &SmartReplyEngine{
		aiService:      aiService,
		intentDetector: detector,
		knowledgeSvc:   knowledgeSvc,
		promptBuilder:  NewSmartPromptBuilder(knowledgeSvc),
		messageSender:  NewWebSocketMessageSender(ws.GlobalHub, di.GlobalContainer.UserService),
		// 群助手 / 自动回复的并发控制（限并发+按会话串行），防洪峰+防同会话乱序/重复回复
		convReplyWorker: service.NewReplyOrchestrator(service.ReplyOrchestratorOpts{
			Workers:   8,
			Serialize: true, // 群助手：按会话串行
		}),
	}
}

// SetUnifiedKnowledge 设置统一知识检索服务（向量库+MySQL兜底）
func (e *SmartReplyEngine) SetUnifiedKnowledge(uk *service.UnifiedKnowledgeService) {
	e.unifiedKnowledge = uk
}

// SetAvatarWorkerPool 设置分身工作池
func (e *SmartReplyEngine) SetAvatarWorkerPool(pool *service.AvatarWorkerPool) {
	e.avatarWorkerPool = pool
}

// SetAvatarTriggerService injects the smart-avatar decision service.
func (e *SmartReplyEngine) SetAvatarTriggerService(triggerSvc AvatarTriggerDecider) {
	e.avatarTriggerSvc = triggerSvc
}

// SetMemoryService sets the avatar memory service for the smart reply engine
func (e *SmartReplyEngine) SetMemoryService(ms *service.AvatarMemoryService) {
	e.memorySvc = ms
}

// Close 关闭群助手回复编排引擎，回收 worker goroutine。供服务优雅退出时调用。
func (e *SmartReplyEngine) Close() {
	if e.convReplyWorker != nil {
		e.convReplyWorker.Close()
	}
}

// SetGroupMemoryService 注入群聊助手的群级记忆服务（与分身记忆隔离）。
func (e *SmartReplyEngine) SetGroupMemoryService(gms *service.GroupMemoryService) {
	e.groupMemorySvc = gms
}

// SetMCPGateway 注入外部 MCP 客户端网关，使群 @AI 白名单能按位点放行外部工具。
// 为 nil 时保持默认白名单，无行为变化。
func (e *SmartReplyEngine) SetMCPGateway(gw *service.MCPClientGateway) {
	if e.smartReplyGraph != nil && gw != nil {
		e.smartReplyGraph.SetMCPGateway(gw)
	}
}

func (e *SmartReplyEngine) InitSmartReplyGraph() error {
	log.Printf("[SmartReplyGraph] 创建 SmartReplyGraph 实例...")
	e.smartReplyGraph = service.NewSmartReplyGraph(
		e.aiService,
		database.GetDB(),
		e.unifiedKnowledge,
		e.knowledgeSvc,
		e.groupMemorySvc,
		di.GlobalContainer.UserService,
	)

	// 注入 AI 阈值服务：群 @AI 的知识来源分数门槛从 system_configs 读取，
	// 后台修改即生效；容器中未初始化（nil）时 graph 内部回退默认 0.6。
	e.smartReplyGraph.SetThresholdService(di.GlobalContainer.AiThresholdService)

	// 注入被引用文件正文读取器，使 @AI 引用文件消息时可把文件内容喂给 AI（nil 时安全降级）。
	if gds := di.GlobalContainer.GroupDocumentService; gds != nil {
		e.smartReplyGraph.SetQuotedFileReader(gds)
	}

	log.Printf("[SmartReplyGraph] 开始编译 Graph...")
	err := e.smartReplyGraph.BuildGraph()
	if err != nil {
		log.Printf("[SmartReplyGraph] BuildGraph 失败: %v", err)
	} else {
		log.Printf("[SmartReplyGraph] BuildGraph 成功")
	}
	return err
}

func (e *SmartReplyEngine) HandleMessage(msg *model.Message, mentionUserIDs []uint) {
	userID := msg.SenderID
	conversationID := msg.ConversationID
	content := msg.Content
	if e.aiService == nil || !e.aiService.IsConfigured() {
		log.Printf("[SmartReply] AI 服务未配置，跳过处理")
		return
	}

	configSvc := service.NewSystemConfigService(database.GetDB())
	publicConfigs, err := configSvc.GetPublicConfigs()
	if err == nil {
		if enableAI, ok := publicConfigs["enableAI"]; ok {
			if !enableAI.(bool) {
				log.Printf("[SmartReply] AI 功能已关闭 (enableAI=false)，跳过处理")
				return
			}
		}
	}

	// 异步把发送者本人的消息择要写入其记忆库（分身越用越懂主人）。不阻塞主流程。
	// 仅文本类消息参与记忆形成：图片/文件等媒体的 content 是 JSON，反射只会空转一轮 LLM。
	if msg.Type == "text" || msg.Type == "markdown" {
		e.maybeRememberSenderMessage(userID, conversationID, content)
	}

	log.Printf("[SmartReply] 开始处理消息: userID=%d convID=%d content=%s", userID, conversationID, content[:min(50, len(content))])

	db := database.GetDB()

	var conv model.Conversation
	if err := db.First(&conv, conversationID).Error; err != nil {
		log.Printf("[SmartReply] 查找会话失败: convID=%d err=%v", conversationID, err)
		return
	}

	log.Printf("[SmartReply] 会话类型: %s", conv.Type)

	if e.avatarWorkerPool != nil {
		e.checkAvatarTriggers(userID, &conv, content, msg.Type, mentionUserIDs)
	}

	if conv.Type == "bot" {
		log.Printf("[SmartReply] bot 会话，跳过 AI 助手回复")
		return
	}

	// 私聊不走全局 AI 助手自动回复：分身回复已在上面的 checkAvatarTriggers 处理，
	// 两个真人之间的私聊不应让 AI 助手插话
	if conv.Type == "single" {
		log.Printf("[SmartReply] 私聊会话，跳过 AI 助手自动回复 (convID=%d)", conv.ID)
		return
	}

	if conv.Type == "group" || conv.Type == "discussion" {
		var group model.Group
		if err := db.Where("conversation_id = ?", conversationID).First(&group).Error; err != nil {
			log.Printf("[SmartReply] 查找群聊失败: convID=%d err=%v", conversationID, err)
			return
		}
		aiConfig := group.GetAIConfig()
		assistantName := "AI助手"
		if aiConfig.AssistantName != "" {
			assistantName = aiConfig.AssistantName
		}

		// 反刷屏依赖 DB 最近 AI 消息查询，结果作为纯决策入参传入 DecideGroupAIReply
		antiSpamBlocked := false
		if aiConfig.AntiSpamInterval > 0 {
			var lastAIMsg model.Message
			err := db.Where("conversation_id = ? AND origin = ? AND created_at > ?",
				conversationID, "assistant", time.Now().Add(-time.Duration(aiConfig.AntiSpamInterval)*time.Minute)).
				Order("created_at DESC").First(&lastAIMsg).Error
			if err == nil {
				antiSpamBlocked = true
				log.Printf("[SmartReply] 反垃圾策略：AI 最近已回复，跳过 (interval=%dmin)", aiConfig.AntiSpamInterval)
			}
		}

		log.Printf("[SmartReply] 群聊 AI 配置: enabled=%v replyMode=%s triggerKeywords=%s",
			aiConfig.Enabled, aiConfig.ReplyMode, aiConfig.TriggerKeywords)

		// 群级记忆写入：群 AI 启用且未显式关闭学习时，异步把值得记的群消息择要写入本群记忆库。
		// LearnEnabled 为 opt-out：默认学习（GetAIConfig 默认 true），仅显式 false 才停写。
		// 仅文本类消息参与记忆形成（媒体消息 content 是 JSON，反射空转一轮 LLM）。
		if aiConfig.Enabled && aiConfig.LearnEnabled && (msg.Type == "text" || msg.Type == "markdown") {
			e.maybeRememberGroupMessage(group.ID, conversationID, content)
		}

		// 直接发图/发文件（不引用）：合成自引用复用引用机制的多模态读取（读图/读正文）。
		// 媒体消息一律不再落入文本意图检测，避免把 {"url":...} JSON 当普通消息回。
		// 触发范围：auto/smart/keyword 等启用模式；off / mention_only 维持现状不回复。
		// video/audio 仅跳过（无法解析进上下文），同样不落入意图检测。
		if e.smartReplyGraph != nil && isDirectMediaType(msg.Type) {
			if decideDirectMediaReply(aiConfig, content, antiSpamBlocked) && (msg.Type == "image" || msg.Type == "file") {
				question := "请识别用户发送的这张图片，并结合上下文回复。"
				if msg.Type == "file" {
					question = "请阅读用户发送的这份文件的内容，并结合上下文回复。"
				}
				e.submitConvReply(conversationID, func() {
					e.handleAIMention(userID, conversationID, question, content, &conv, assistantName, msg)
				})
			}
			return
		}

		switch DecideGroupAIReply(*aiConfig, content, assistantName, antiSpamBlocked) {
		case GroupAIMentionReply:
			question := extractAIQuestion(content, assistantName)
			e.submitConvReply(conversationID, func() {
				e.handleAIMention(userID, conversationID, question, content, &conv, assistantName, msg)
			})
			return
		case GroupAISkipReply:
			return
		case GroupAIAutoReply:
			// 落到下方意图检测自动回复
		}
	}

	intent, err := e.intentDetector.Detect(content, userID, conversationID)
	if err != nil {
		log.Printf("[SmartReply] 意图检测失败: %v", err)
		return
	}

	log.Printf("[SmartReply] 意图检测结果: type=%s confidence=%.2f", intent.Type, intent.Confidence)

	shouldReply := e.intentDetector.ShouldTriggerAIReply(intent, conv.Type)

	if !shouldReply {
		log.Printf("[SmartReply] 意图类型 %s (confidence=%.2f) 不触发 AI 回复", intent.Type, intent.Confidence)
	}

	if shouldReply {
		e.submitConvReply(conversationID, func() { e.generateAndSendReply(userID, conversationID, content, intent) })
	}
}

// isDirectMediaType 直接发送的媒体消息类型（引用消息不属于此场景）。
// 图片/文件可走多模态读取；视频/语音当前无法解析，仅用于跳过文本意图检测。
func isDirectMediaType(t string) bool {
	return t == "image" || t == "file" || t == "video" || t == "audio"
}

// decideDirectMediaReply 直接发媒体消息时群 AI 是否触发回复。
// off / mention_only 维持现状不触发（mention_only 语义是"仅@触发"）；
// 其余启用模式（auto/smart 等）触发，与 DecideGroupAIReply 的 Auto 判定保持一致。
// 关键词门控与 DecideGroupAIReply 对齐：开启 TriggerKeywords 的群，直发媒体正文是
// {"url":...} JSON，几乎不可能命中关键词，等效于不自动回复直发媒体--
// 关键词模式的语义就是仅关键词命中的内容才唤起助手，媒体消息不得绕过。
func decideDirectMediaReply(cfg *model.GroupAIConfig, content string, antiSpamBlocked bool) bool {
	if !cfg.Enabled || antiSpamBlocked {
		return false
	}
	if cfg.ReplyMode == "off" || cfg.ReplyMode == "mention_only" {
		return false
	}
	if cfg.TriggerKeywords != "" && !groupAIKeywordMatches(content, cfg.TriggerKeywords) {
		return false
	}
	return true
}

// selfQuoteMessageID 直接发图/发文件（不引用）时合成自引用：返回触发消息自身 ID，
// 供图上下文复用引用机制读取图片/正文。显式引用优先——原消息已引用其他消息时不覆盖。
// 非图片/文件（文本/视频/语音）或 nil 消息原样返回原引用（可能为 nil）。
func selfQuoteMessageID(origMsg *model.Message) *uint {
	if origMsg == nil {
		return nil
	}
	if origMsg.QuotedMessageID == nil && (origMsg.Type == "image" || origMsg.Type == "file") {
		return &origMsg.ID
	}
	return origMsg.QuotedMessageID
}

// submitConvReply 把一次群助手/自动回复提交到按会话串行的异步队列。
// 队列就绪即异步执行（与原 SafeGo / 裸 go 直发语义一致），worker 内按会话串行，
// 保证同会话回复不乱序、不重复；worker 缺失或入队超时降级为直接执行，避免回复丢失。
func (e *SmartReplyEngine) submitConvReply(convID uint, handle func()) {
	if e.convReplyWorker != nil {
		if err := e.convReplyWorker.Submit(convID, handle); err == nil {
			return
		} else {
			log.Printf("[SmartReply] 回复入队繁忙，降级直接执行: convID=%d err=%v", convID, err)
		}
	}
	utils.SafeGo(handle)
}

// generateAndSendReply 生成并发送智能回复
func (e *SmartReplyEngine) generateAndSendReply(userID uint, conversationID uint, userContent string, intent *ai.MessageIntent) {
	if e.smartReplyGraph != nil {
		e.generateAndSendReplyWithGraph(userID, conversationID, userContent, intent)
		return
	}
	e.generateAndSendReplyLegacy(userID, conversationID, userContent, intent)
}

func (e *SmartReplyEngine) generateAndSendReplyWithGraph(userID uint, conversationID uint, userContent string, intent *ai.MessageIntent) {
	ctx := context.Background()
	input := &service.SmartReplyContext{
		Message:         userContent,
		OriginalContent: userContent,
		UserID:          userID,
		ConversationID:  conversationID,
		Intent:          intent,
		IsAIMention:     false,
	}

	result, err := e.smartReplyGraph.Execute(ctx, input)
	if err != nil {
		log.Printf("[SmartReplyGraph] AI 回复生成失败: %v", err)
		return
	}

	log.Printf("[SmartReplyGraph] 生成回复长度: %d 字符", len(result.Reply))

	// 空/纯空白回复不发送，避免 AI 不可用时落空白气泡
	if strings.TrimSpace(result.Reply) == "" {
		log.Printf("[SmartReplyGraph] AI 回复内容为空，跳过发送: convID=%d", conversationID)
		return
	}

	// 命中的知识来源（标题/相关度）随回复写入 Extra，刷新/回放后「知识来源」徽章仍可见。
	// 已由注入前 reranker 过滤误召回，徽章展示通过的真实来源。
	err = e.messageSender.SendAIMessage(conversationID, result.Reply, "AI助手", knowledgeSourcesExtra(input.KnowledgeSources))
	if err != nil {
		log.Printf("[SmartReply] 发送 AI 消息失败: %v", err)
		return
	}

	log.Printf("[SmartReplyGraph] 已发送智能回复到会话 %d", conversationID)
}

func (e *SmartReplyEngine) generateAndSendReplyLegacy(userID uint, conversationID uint, userContent string, intent *ai.MessageIntent) {
	ctx := e.promptBuilder.BuildPromptContext(conversationID, userID)
	if ctx == nil {
		log.Printf("[SmartReply] 构建提示词上下文失败")
		return
	}

	systemPrompt := e.promptBuilder.BuildSystemPrompt(ctx)

	var knowledgeSources []service.KnowledgeSource
	if e.unifiedKnowledge != nil && ctx.Group != nil {
		// 一次检索同时产出上下文串与命中的知识来源（供自动回复把来源随回复下发）。
		var kctx string
		kctx, knowledgeSources = e.unifiedKnowledge.BuildContextWithSources(userContent, ctx.Group.ID, 3)
		if kctx != "" {
			systemPrompt += "\n\n" + kctx
		}
	} else if e.knowledgeSvc != nil {
		knowledgeCtx := e.knowledgeSvc.BuildKnowledgeContext(userContent)
		if knowledgeCtx != "" {
			systemPrompt += "\n\n" + knowledgeCtx
		}
	}

	if e.groupMemorySvc != nil && ctx.Group != nil {
		memoryResults, err := e.groupMemorySvc.Recall(ctx.Group.ID, userContent, 2)
		if err == nil && len(memoryResults) > 0 {
			var parts []string
			for _, r := range memoryResults {
				parts = append(parts, r.Content)
			}
			memoryCtx := "💡 群聊记忆：\n" + strings.Join(parts, "\n")
			systemPrompt += "\n\n" + memoryCtx
		}
	}

	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userContent},
	}

	callerCtx := &ai.CallerContext{
		UserID: userID,
	}
	reply, err := e.aiService.GetCompletionWithTools(ai.TaskTypeChat, messages, callerCtx)
	if err != nil {
		log.Printf("[SmartReply] AI 回复生成失败: %v", err)
		return
	}

	// 命中的知识来源（标题/相关度）随回复写入 Extra，刷新/回放后「知识来源」徽章仍可见。
	err = e.messageSender.SendAIMessage(conversationID, reply, "AI助手", knowledgeSourcesExtra(knowledgeSources))
	if err != nil {
		log.Printf("[SmartReply] 发送 AI 消息失败: %v", err)
		return
	}

	log.Printf("[SmartReply] 已发送智能回复到会话 %d", conversationID)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RateLimiter 简易令牌桶限流器
type RateLimiter struct {
	ticker *time.Ticker
	ch     chan struct{}
	once   sync.Once
}

// NewRateLimiter 创建限流器，interval 为两次放行间隔，burst 为突发上限
func NewRateLimiter(interval time.Duration, burst int) *RateLimiter {
	rl := &RateLimiter{
		ticker: time.NewTicker(interval),
		ch:     make(chan struct{}, burst),
	}
	// 预填令牌
	for i := 0; i < burst; i++ {
		rl.ch <- struct{}{}
	}
	go func() {
		for range rl.ticker.C {
			select {
			case rl.ch <- struct{}{}:
			default:
			}
		}
	}()
	return rl
}

// Wait 阻塞直到获取一个令牌
func (rl *RateLimiter) Wait() {
	<-rl.ch
}

// Stop 停止限流器
func (rl *RateLimiter) Stop() {
	rl.once.Do(func() {
		rl.ticker.Stop()
		close(rl.ch)
	})
}

// isAIMention 判断消息是否 @ 了群 AI 助手。委托纯函数 groupAIMentionsAI，
// 以便 DecideGroupAIReply 与本方法共用同一套判定逻辑（smart_reply_handler_test.go 直接用本方法）。
func (e *SmartReplyEngine) isAIMention(content string, assistantName string) bool {
	return groupAIMentionsAI(content, assistantName)
}

// GroupAIReplyAction 群聊 AI 对单条消息的纯决策结果。反刷屏（AntiSpamInterval）
// 依赖数据库最近 AI 消息查询，不在纯函数内判定，由调用方查库后以 antiSpamBlocked 入参传入。
type GroupAIReplyAction int

const (
	GroupAISkipReply    GroupAIReplyAction = iota // 不回复
	GroupAIMentionReply                           // @AI 提及，直接回复
	GroupAIAutoReply                              // 走意图检测自动回复
)

// DecideGroupAIReply 是群聊 AI 触发的纯决策入口。
//
// 决策顺序：启用 → 反刷屏 → @AI 提及 → 关键词门控 → 模式判定。
// @AI 提及优先于关键词门控：用户显式点名应总能触达助手，关键词门控仅对
// 自动回复路径生效。反刷屏优先级最高——即使 @AI 提及，命中反刷屏窗口也跳过。
// 以上行为均由表测试钉死（见 smart_reply_group_ai_test.go），调整请同步更新。
func DecideGroupAIReply(cfg model.GroupAIConfig, content, assistantName string, antiSpamBlocked bool) GroupAIReplyAction {
	if !cfg.Enabled {
		return GroupAISkipReply
	}
	if antiSpamBlocked {
		return GroupAISkipReply
	}
	if groupAIMentionsAI(content, assistantName) {
		return GroupAIMentionReply
	}
	// 关键词门控仅在非 mention_only 模式下、且对自动回复路径生效
	if cfg.ReplyMode != "mention_only" && cfg.TriggerKeywords != "" {
		if !groupAIKeywordMatches(content, cfg.TriggerKeywords) {
			return GroupAISkipReply
		}
	}
	if cfg.ReplyMode == "off" || cfg.ReplyMode == "mention_only" {
		return GroupAISkipReply
	}
	return GroupAIAutoReply
}

// groupAIKeywordMatches 判断消息内容是否命中任一触发关键词（逗号分隔，大小写不敏感）。
// keywords 为空时视为无限制（命中），调用方已保证仅在 TriggerKeywords 非空时调用。
func groupAIKeywordMatches(content, keywords string) bool {
	if keywords == "" {
		return true
	}
	lower := strings.ToLower(content)
	for _, kw := range strings.Split(keywords, ",") {
		kw = strings.TrimSpace(kw)
		if kw != "" && strings.Contains(lower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// groupAIMentionsAI 判断消息是否 @ 了群 AI 助手（mention token 或明文 @AI/@助手名）。
// isAIMention 的纯逻辑副本，供 DecideGroupAIReply 与运行时路径共用，避免双套判断。
func groupAIMentionsAI(content, assistantName string) bool {
	for _, m := range mention.Parse(content) {
		if isAIAssistantMentionName(m.Name, assistantName) {
			return true
		}
	}
	patterns := []string{
		"@AI",
		"@Ai",
		"@ai",
		"@" + assistantName,
	}
	for _, pattern := range patterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}
	return false
}

func isAIAssistantMentionName(name string, assistantName string) bool {
	if name == "" {
		return false
	}
	if assistantName != "" && name == assistantName {
		return true
	}
	return strings.EqualFold(name, "ai")
}

// extractAIQuestion 提取 @AI 后的问题内容
func extractAIQuestion(content string, assistantName string) string {
	for _, m := range mention.Parse(content) {
		if isAIAssistantMentionName(m.Name, assistantName) {
			return strings.TrimSpace(content[m.End:])
		}
	}

	patterns := []string{"@AI", "@Ai", "@ai", "@" + assistantName}

	for _, pattern := range patterns {
		if idx := strings.Index(content, pattern); idx != -1 {
			question := content[idx+len(pattern):]
			return strings.TrimSpace(question)
		}
	}
	return content
}

// handleAIMention 处理 @AI 触发回复
// origMsg 为触发这条 @AI 的原始消息（透传完整对象，供读取引用消息等属性）。
func (e *SmartReplyEngine) handleAIMention(userID uint, conversationID uint, question string, originalContent string, conv *model.Conversation, assistantName string, origMsg *model.Message) {
	log.Printf("[SmartReply] @AI 触发回复: userID=%d, convID=%d, question=%s", userID, conversationID, question[:min(50, len(question))])

	if e.aiService == nil || !e.aiService.IsConfigured() {
		log.Printf("[SmartReply] AI 服务未配置")
		return
	}

	if e.smartReplyGraph != nil {
		e.handleAIMentionWithGraph(userID, conversationID, question, originalContent, assistantName, origMsg)
		return
	}

	e.handleAIMentionLegacy(userID, conversationID, question, originalContent, conv, assistantName)
}

func (e *SmartReplyEngine) handleAIMentionWithGraph(userID uint, conversationID uint, question string, originalContent string, assistantName string, origMsg *model.Message) {
	ctx := context.Background()
	// AI 开始处理：先推开始事件供前端「思考中」占位。放在分支前统一覆盖三分支——
	// 纯流式分支（无工具）是大多数群的默认路径，只有这里能兜住；带工具分支内部
	// handleAIMentionWithTools 会再推一次，幂等无害（仅重置占位计时）。
	e.messageSender.NotifyReplyStarted(conversationID, assistantName)
	// 直接发图/发文件（不引用）时由 selfQuoteMessageID 合成自引用；显式引用优先。
	quotedMessageID := selfQuoteMessageID(origMsg)
	isSelfQuote := origMsg != nil && origMsg.QuotedMessageID == nil && (origMsg.Type == "image" || origMsg.Type == "file")
	input := &service.SmartReplyContext{
		Message:         question,
		OriginalContent: originalContent,
		UserID:          userID,
		ConversationID:  conversationID,
		IsAIMention:     true,
		AssistantName:   assistantName,
		QuotedMessageID: quotedMessageID,
		IsSelfQuote:     isSelfQuote,
	}

	// 管理操作指令（踢人/加人/禁言等）走带工具路径：注入 AI 群管理工具，
	// LLM 返回 tool call 时真实执行。仅系统管理员发起的指令会被工具执行。
	if intent, derr := e.intentDetector.Detect(question, userID, conversationID); derr == nil && ShouldUseToolsForMention(intent) {
		input.SkipKnowledge = true
		e.handleAIMentionWithTools(ctx, input, conversationID, assistantName, userID, "带",
			func(c context.Context, i *service.SmartReplyContext, fb ai.ReActStepCallback) (string, error) {
				return e.smartReplyGraph.ExecuteWithTools(c, i, service.ToolsetBuiltin, fb)
			},
			func(c context.Context, i *service.SmartReplyContext, fb ai.ReActStepCallback, onChunk func(ai.StreamChunk) error) (bool, error) {
				return e.smartReplyGraph.ExecuteWithToolsStream(c, i, service.ToolsetBuiltin, fb, onChunk)
			})
		return
	}

	// 后台开启外部 MCP 群启用时，普通提问也走带外部工具的流式 ReAct：
	// LLM 可在自然语言下自主调用外部工具，挂起期间向用户发"正在调用…"过程反馈。
	// 关闭时（默认）保持下面原有纯流式路径，零行为变化。
	// 注意：不再设 SkipKnowledge=true，否则群即使开了外部工具，@AI 普通提问也会失去
	// 知识库/记忆上下文，导致无依据应答。知识注入由 prepareInput 按需独立判断（如引用图片时跳过）。
	if e.smartReplyGraph.HasExternalTools() {
		e.handleAIMentionWithExternalTools(ctx, input, conversationID, assistantName, userID)
		return
	}

	stream, err := e.smartReplyGraph.ExecuteStream(ctx, input)
	if err != nil {
		log.Printf("[SmartReplyGraph] @AI 流式回复失败: %v", err)
		return
	}

	sendChunk, getMsg, finish, err := e.messageSender.SendStreamingAIMessage(conversationID, assistantName)
	if err != nil {
		log.Printf("[SmartReply] 创建流式消息失败: %v", err)
		return
	}

	// @提问者模式：读取配置
	mentionPrefix := e.buildMentionPrefix(conversationID, userID)

	chunkCount := 0
	totalLen := 0
	var streamErr error
	for {
		msg, err := stream.Recv()
		if err != nil {
			// io.EOF = 正常结束；其余为真实错误（如模型调用失败）。
			// 之前把两者都当 break 处理，导致模型出错时静默无回复，
			// 且预创建的流式空消息残留卡住。这里保留真实错误用于兜底。
			if !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
				streamErr = err
			}
			break
		}
		// 剥离 AI 自带的 mention token，避免重复
		cleanContent := mention.StripTokens(msg.Content)
		if cleanContent == "" {
			continue
		}
		chunkCount++
		totalLen += len(cleanContent)
		// 第一个 chunk 拼 mention 前缀
		if mentionPrefix != "" {
			cleanContent = mentionPrefix + cleanContent
			mentionPrefix = "" // 只拼一次
		}
		sendChunk(cleanContent)
	}

	log.Printf("[SmartReplyGraph] @AI 流式回复完成: %d 个 chunk, 总长度 %d 字符, streamErr=%v", chunkCount, totalLen, streamErr)

	// 模型调用出错：不再静默无回复。SendStreamingAIMessage 采用懒创建——
	// 首个非空正文块才创建消息，因此若 chunkCount == 0 则无消息残留；
	// 若已产出部分内容则发送兜底文案收尾，保证用户得到反馈。
	if streamErr != nil {
		log.Printf("[SmartReplyGraph] @AI 流式回复出错: %v", streamErr)
		if chunkCount == 0 {
			fallback := "⚠️ 这条消息暂时没能回复（模型调用出错），请稍后再试。"
			if mentionPrefix != "" {
				fallback = mentionPrefix + fallback
				mentionPrefix = ""
			}
			if err := sendChunk(fallback); err != nil {
				log.Printf("[SmartReplyGraph] 发送兜底文案失败: %v", err)
			}
		}
		// 与 handleAIMentionWithTools 错误路径一致：若已收集到知识来源，仍需把 Extra 落库，
		// 否则刷新/REST 回放后「知识来源」徽章丢失。
		e.persistAIMessageExtra(getMsg, nil, input.KnowledgeSources)
		finish()
		return
	}

	if chunkCount == 0 {
		log.Printf("[SmartReplyGraph] AI 回复内容为空，跳过保存")
		return
	}

	// 命中的知识来源（标题/相关度）持久化到消息 Extra，刷新/REST 回放后「知识来源」徽章仍可见。
	// 已由注入前 reranker 过滤误召回，徽章展示通过的真实来源。
	e.persistAIMessageExtra(getMsg, nil, input.KnowledgeSources)

	if finish() == nil {
		log.Printf("[SmartReplyGraph] 完成流式消息失败")
		return
	}

	log.Printf("[SmartReplyGraph] @AI 流式回复已完成")
}

// handleAIMentionWithTools 带工具的 @AI 回复，用于管理操作指令（踢人/加人/禁言等）与
// 普通提问（外部 MCP 工具）两种路径共用。优先走真·流式（execStream）：final 答案逐 token
// 送、工具事件实时推卡片；Provider 不支持流式 tool-call 时降级到非流式（exec +
// splitReplyChunks 切块打字感）。两路径复用 SendStreamingAIMessage 的流式气泡与同一壳
// 骨架（创建流式气泡 → @提问者前缀 → 工具卡片收集 → 兜底空答 → Extra 持久化 → 收尾），
// 避免把整段逻辑复制成两份。
//
// 内置/外部工具调用均以独立卡片呈现（onStep 反馈实时推 ai_tool_call 事件，前端在气泡
// 下方渲染工具卡片，不再拼进正文），工具调用列表写进消息 Extra（tool_calls）供回放/刷新
// 后卡片仍可见。kind 取 "带"（内置群管理工具）或 "外部"（外部 MCP 工具），用于区分日志。
// execStream 可为 nil（无流式能力时）——此时直接走下面的非流式路径。
func (e *SmartReplyEngine) handleAIMentionWithTools(ctx context.Context, input *service.SmartReplyContext, conversationID uint, assistantName string, userID uint, kind string, exec func(context.Context, *service.SmartReplyContext, ai.ReActStepCallback) (string, error), execStream func(context.Context, *service.SmartReplyContext, ai.ReActStepCallback, func(ai.StreamChunk) error) (bool, error)) {
	// AI 开始处理：先推开始事件供前端「思考中」占位（首个流式帧到达前）。
	// 本函数仅在 messageSender 非 nil 时被调用（引擎装配保证），此处不再重复判空。
	e.messageSender.NotifyReplyStarted(conversationID, assistantName)

	sendChunk, getMsg, finish, err := e.messageSender.SendStreamingAIMessage(conversationID, assistantName)
	if err != nil {
		log.Printf("[SmartReply] 创建流式消息失败: %v", err)
		return
	}

	// @提问者模式：读取配置（与流式路径一致）
	mentionPrefix := e.buildMentionPrefix(conversationID, userID)

	// 仅 @提问者模式需在正文前发送 mention 前缀（非空 → 触发懒创建）。
	// 普通模式不在此提前建/投递任何消息占位：SendStreamingAIMessage 采用懒创建，
	// 消息延迟到首个非空正文块或首个工具调用才真正落库，避免空占位排在用户后续
	// 消息之上、等答案回填后造成顺序错乱。
	if mentionPrefix != "" {
		if err := sendChunk(mentionPrefix); err != nil {
			log.Printf("[SmartReplyGraph] 发送 @ 前缀失败: %v", err)
			finish()
			return
		}
	}

	// 收集本次 ReAct 用到的工具调用：start/end 阶段实时推 ai_tool_call 事件供前端
	// 渲染工具卡片（进行态→终态按 ID 对齐），终态记录写入消息 Extra 持久化。
	var toolCalls []ToolCallRecord
	feedback := newToolCallFeedback(e.messageSender, conversationID, getMsg, &toolCalls, e.smartReplyGraph.MCPToolTitles(), e.smartReplyGraph.MCPToolDescriptions())

	// 先尝试真·流式路径：execStream 内 final 答案逐 token 经 onChunk → sendChunk 送出，
	// 工具事件经 feedback 实时推卡片。streamed=true 表示已走流式逐 token。
	if execStream != nil {
		streamed, serr := execStream(ctx, input, feedback, func(chunk ai.StreamChunk) error {
			return sendChunk(chunk.Content)
		})
		if serr != nil && !errors.Is(serr, ai.ErrStreamingToolsNotSupported) {
			log.Printf("[SmartReplyGraph] @AI %s工具流式回复失败: %v", kind, serr)
			// 流式中途失败：若已产出正文/工具、且拿到了知识来源，仍需把 Extra 落库，
			// 否则刷新/REST 回放后工具卡片与「知识来源」徽章丢失（正文已由 finish 收尾）。
			e.persistAIMessageExtra(getMsg, toolCalls, input.KnowledgeSources)
			finish()
			return
		}
		if streamed {
			// 工具调用记录 + 命中的知识来源合并持久化到消息 Extra，回放/刷新后卡片与徽章仍可见。
			e.persistAIMessageExtra(getMsg, toolCalls, input.KnowledgeSources)
			if finish() == nil {
				log.Printf("[SmartReplyGraph] 完成流式消息失败")
			} else {
				log.Printf("[SmartReplyGraph] @AI %s工具流式回复已完成, %d 次工具调用", kind, len(toolCalls))
			}
			return
		}
		// streamed=false（Provider 不支持流式 tool-call）→ 降级到下面非流式路径。
		log.Printf("[SmartReplyGraph] @AI %s工具流式不可用，降级到非流式回复", kind)
	}

	reply, err := exec(ctx, input, feedback)
	if err != nil {
		log.Printf("[SmartReplyGraph] @AI %s工具回复失败: %v", kind, err)
		// 完全没产出时发兜底文案，避免残留空白卡住的空气泡
		if reply == "" {
			_ = sendChunk("⚠️ 这条消息暂时没能回复（调用出错），请稍后再试。")
		}
		// 与流式路径一致：若已收集到工具调用/知识来源，仍需把 Extra 落库，
		// 否则刷新/REST 回放后工具卡片与「知识来源」徽章丢失。
		e.persistAIMessageExtra(getMsg, toolCalls, input.KnowledgeSources)
		finish()
		return
	}

	reply = mention.StripTokens(reply)
	if strings.TrimSpace(reply) == "" {
		log.Printf("[SmartReplyGraph] @AI %s工具回复内容为空，跳过保存", kind)
		finish()
		return
	}

	// 按句子切子块流式送出最终答案，保留打字感；空块跳过。
	for _, chunk := range splitReplyChunks(reply) {
		if chunk == "" {
			continue
		}
		if err := sendChunk(chunk); err != nil {
			log.Printf("[SmartReplyGraph] 发送回复分块失败: %v", err)
			break
		}
	}

	// 工具调用记录 + 命中的知识来源合并持久化到消息 Extra，回放/刷新后卡片与徽章仍可见。
	e.persistAIMessageExtra(getMsg, toolCalls, input.KnowledgeSources)

	if finish() == nil {
		log.Printf("[SmartReplyGraph] 完成流式消息失败")
		return
	}
	log.Printf("[SmartReplyGraph] @AI %s工具回复已完成, %d 次工具调用", kind, len(toolCalls))
}

// handleAIMentionWithExternalTools 普通提问（非管理指令）走带外部 MCP 工具的流式 ReAct
// 回复。与 handleAIMentionWithTools 共用骨架，仅工具执行走外部 MCP 工具集，详见其上注释。
// 传入流式执行器：Provider 支持流式 tool-call 时 final 答案逐 token 送出；否则降级到
// 非流式 ExecuteWithTools（ToolsetExternal）。
func (e *SmartReplyEngine) handleAIMentionWithExternalTools(ctx context.Context, input *service.SmartReplyContext, conversationID uint, assistantName string, userID uint) {
	e.handleAIMentionWithTools(ctx, input, conversationID, assistantName, userID, "外部",
		func(c context.Context, i *service.SmartReplyContext, fb ai.ReActStepCallback) (string, error) {
			return e.smartReplyGraph.ExecuteWithTools(c, i, service.ToolsetExternal, fb)
		},
		func(c context.Context, i *service.SmartReplyContext, fb ai.ReActStepCallback, onChunk func(ai.StreamChunk) error) (bool, error) {
			return e.smartReplyGraph.ExecuteWithToolsStream(c, i, service.ToolsetExternal, fb, onChunk)
		})
}

// persistAIMessageExtra 把群助手回复的工具调用与知识来源合并写进消息 Extra（JSON），
// 供刷新/REST 回放后工具卡片与「知识来源」徽章仍可见。两者任一无内容则只写有内容的键，
// 全空时不写（消息 Extra 维持默认值）。sources 为命中的知识来源（标题/相关度）。
//
// 入参为 getMsg func()（惰性）：仅在确有工具调用或知识来源需落库时才调用 getMsg() 触发
// 懒创建，避免把 getMsg() 作为参数无条件求值——否则在流式一开始就失败、既无正文也无工具
// 或无知识来源的路径上，会强制落库一条空消息并在收尾时广播空白气泡（配合 finish 在
// created==nil 时跳过以彻底杜绝）。全部调用点原先传 getMsg() 返回值，现改为传 getMsg 函数。
func (e *SmartReplyEngine) persistAIMessageExtra(getMsg func() *model.Message, toolCalls []ToolCallRecord, sources []service.KnowledgeSource) {
	service.PersistAIMessageExtra(getMsg, toolCalls, sources)
}

// knowledgeSourcesExtra 把命中的知识来源转成供 SendAIMessage 写入 Extra 的 map；
// 无命中时返回 nil（不写 Extra）。
func knowledgeSourcesExtra(sources []service.KnowledgeSource) map[string]interface{} {
	if len(sources) == 0 {
		return nil
	}
	return map[string]interface{}{"knowledge_sources": sources}
}

// buildMentionPrefix 查询群 AI 配置：如果 mention_reply_mode == "mention"，
// 返回 "@用户昵称\n\n" 前缀（供 AI 回复时 @ 提问者）；否则返回空串。
// handleAIMentionWithGraph 和 handleAIMentionWithTools 共用此函数。
func (e *SmartReplyEngine) buildMentionPrefix(conversationID, userID uint) string {
	db := database.GetDB()
	var group model.Group
	if err := db.Where("conversation_id = ?", conversationID).First(&group).Error; err != nil {
		return ""
	}
	if group.GetAIConfig().MentionReplyMode != "mention" {
		return ""
	}
	var mentionUser model.User
	if err := db.First(&mentionUser, userID).Error; err != nil {
		return ""
	}
	name := mentionUser.Nickname
	if name == "" {
		name = mentionUser.Username
	}
	return mention.Encode(userID, name) + "\n\n"
}

// newToolCallFeedback 构造群 @AI 带工具回复的 ReAct 进度回调（内置工具与外部工具路径共用）。
// 逻辑已下沉到 service.NewToolCallFeedback（与专属机器人回复路径共用，避免重复），
// 此处仅做类型适配：*WebSocketMessageSender 满足 service.StreamingAISender 接口。
func newToolCallFeedback(sender *WebSocketMessageSender, conversationID uint, getMsg func() *model.Message, toolCalls *[]ToolCallRecord, toolTitles, toolDescriptions map[string]string) ai.ReActStepCallback {
	return service.NewToolCallFeedback(sender, conversationID, getMsg, toolCalls, toolTitles, toolDescriptions)
}

// friendlyToolLabel 把内部工具名映射为面向用户的中文动作名词。逻辑已下沉到
// service.FriendlyToolLabel，此处保留薄包装供 handler 内既有调用点零改动。
func friendlyToolLabel(tool string) string {
	return service.FriendlyToolLabel(tool)
}

// splitReplyChunks 把完整回复切成适合逐块送达的小段（尽量在句子/停顿处断开），
// 用于在 sendChunk 上保留流式打字感。实现极简：按常见中文/英文标点断句，
// 单段不超过 maxChunk 个字符，过长则硬截断。
func splitReplyChunks(reply string) []string {
	const maxChunk = 160
	var chunks []string
	runes := []rune(reply)
	for i := 0; i < len(runes); {
		// 在当前段内找尽可能靠后且在 i+maxChunk 以内的断句点
		cut := i + maxChunk
		if cut > len(runes) {
			cut = len(runes)
		}
		best := -1
		for j := cut; j > i; j-- {
			switch runes[j-1] {
			case '。', '！', '？', '；', '：', '\n', '.', '!', '?', ';', '，', ',':
				best = j
				j = i // 只取离 cut 最近（从后往前第一个）断点
			}
		}
		if best == -1 || best <= i {
			best = cut
		}
		chunks = append(chunks, string(runes[i:best]))
		i = best
	}
	return chunks
}

// ShouldUseToolsForMention 判断 @AI 提及是否应走带工具路径（管理操作指令）。
// command 意图（移除/踢出/添加/邀请/禁言/解封/设置管理员/取消管理员等）走带工具，
// 其他意图（chat/query/alert/todo）保持流式纯文本回复。
func ShouldUseToolsForMention(intent *ai.MessageIntent) bool {
	return intent != nil && intent.Type == "command"
}

func (e *SmartReplyEngine) handleAIMentionLegacy(userID uint, conversationID uint, question string, originalContent string, conv *model.Conversation, assistantName string) {
	ctx := e.promptBuilder.BuildPromptContext(conversationID, userID)
	if ctx == nil {
		log.Printf("[SmartReply] 构建提示词上下文失败")
		return
	}

	// AI 开始处理：先推开始事件供前端「思考中」占位（graph 未装配时的兜底流式路径）。
	e.messageSender.NotifyReplyStarted(conversationID, assistantName)

	systemPrompt := e.promptBuilder.BuildSystemPrompt(ctx)

	if e.unifiedKnowledge != nil && conv.Type == "group" {
		var g model.Group
		if err := database.GetDB().Where("conversation_id = ?", conversationID).First(&g).Error; err == nil {
			knowledgeCtx := e.unifiedKnowledge.BuildContext(question, g.ID)
			if knowledgeCtx != "" {
				systemPrompt += "\n\n" + knowledgeCtx
			}
		}
	} else if e.knowledgeSvc != nil {
		knowledgeCtx := e.knowledgeSvc.BuildKnowledgeContext(question)
		if knowledgeCtx != "" {
			systemPrompt += "\n\n" + knowledgeCtx
		}
	}

	db := database.GetDB()
	var recentMessages []model.Message
	db.Where("conversation_id = ?", conversationID).
		Preload("Sender").
		Order("created_at DESC").
		Limit(20).
		Find(&recentMessages)

	for i, j := 0, len(recentMessages)-1; i < j; i, j = i+1, j-1 {
		recentMessages[i], recentMessages[j] = recentMessages[j], recentMessages[i]
	}

	var messages []ai.Message
	for _, msg := range recentMessages {
		if originalContent != "" && msg.SenderID == userID && msg.Content == originalContent {
			continue
		}

		if msg.Origin == "assistant" {
			messages = append(messages, ai.Message{
				Role:    "assistant",
				Content: msg.Content,
			})
		} else {
			senderName := msg.Sender.Nickname
			if senderName == "" {
				senderName = msg.Sender.Username
			}
			messages = append(messages, ai.Message{
				Role:    "user",
				Content: fmt.Sprintf("[%s]: %s", senderName, msg.Content),
			})
		}
	}

	messages = append(messages, ai.Message{Role: "user", Content: fmt.Sprintf("💬 请回答：%s", question)})

	// 将 system prompt 插入消息列表头部
	messages = append([]ai.Message{{Role: "system", Content: systemPrompt}}, messages...)

	sendChunk, _, finish, err := e.messageSender.SendStreamingAIMessage(conversationID, assistantName)
	if err != nil {
		log.Printf("[SmartReply] 创建流式消息失败: %v", err)
		return
	}

	chunkCount := 0
	err = e.aiService.GetCompletionStream(ai.TaskTypeChat, messages, func(chunk ai.StreamChunk) error {
		if chunk.Content != "" {
			chunkCount++
		}
		return sendChunk(chunk.Content)
	})

	if err != nil {
		log.Printf("[SmartReply] AI 流式回复失败: %v", err)
		// 与 handleAIMentionWithGraph 一致：已产出内容时需 finish() 收尾广播，
		// 避免残留未完成的流式消息；懒创建模式下无内容则 finish() 安全跳过。
		if chunkCount > 0 {
			finish()
		}
		return
	}

	if finish() == nil {
		log.Printf("[SmartReply] 完成流式消息失败")
		return
	}

	log.Printf("[SmartReply] @AI 流式回复已完成")
}

// GroupSummaryJob 群聊总结定时任务
type GroupSummaryJob struct {
	aiService *ai.AIService
}

// NewGroupSummaryJob 创建群聊总结任务
func NewGroupSummaryJob(aiService *ai.AIService) *GroupSummaryJob {
	return &GroupSummaryJob{
		aiService: aiService,
	}
}

// GenerateDailySummaries 生成所有群的每日总结
func (j *GroupSummaryJob) GenerateDailySummaries() {
	if j.aiService == nil || !j.aiService.IsConfigured() {
		log.Printf("[GroupSummary] AI 服务未配置，跳过总结")
		return
	}

	db := database.GetDB()

	var groups []model.Conversation
	db.Where("type = ?", "group").Find(&groups)

	log.Printf("[GroupSummary] 开始为 %d 个群生成每日总结", len(groups))

	const workerCount = 5
	sem := make(chan struct{}, workerCount)
	// 共享令牌桶：每秒 2 次 AI 调用，突发上限 1
	rl := NewRateLimiter(500*time.Millisecond, 1)
	// 函数结束后停止限流器，释放 ticker 和 goroutine，避免每日执行累积泄露
	defer rl.Stop()
	var wg sync.WaitGroup
	successCount := 0
	failCount := 0
	var mu sync.Mutex

	for _, group := range groups {
		wg.Add(1)
		sem <- struct{}{}
		go func(g model.Conversation) {
			defer wg.Done()
			defer func() { <-sem }()

			rl.Wait() // 等待令牌，只在真正调用 AI 前阻塞
			if j.generateGroupSummary(&g) {
				mu.Lock()
				successCount++
				mu.Unlock()
			} else {
				mu.Lock()
				failCount++
				mu.Unlock()
			}
		}(group)
	}

	wg.Wait()
	log.Printf("[GroupSummary] 每日总结生成完成，成功: %d, 失败: %d", successCount, failCount)
}

// generateGroupSummary 生成单个群的总结
func (j *GroupSummaryJob) generateGroupSummary(group *model.Conversation) bool {
	db := database.GetDB()

	var groupInfo model.Group
	if err := db.Where("conversation_id = ?", group.ID).First(&groupInfo).Error; err != nil {
		log.Printf("[GroupSummary] 获取群聊信息失败: %v", err)
		return false
	}

	today := time.Now().Truncate(24 * time.Hour)
	yesterday := today.Add(-24 * time.Hour)

	var messages []model.Message
	db.Where("conversation_id = ? AND created_at >= ? AND created_at < ?",
		group.ID, yesterday, today).
		Preload("Sender").
		Order("created_at ASC").
		Limit(200).
		Find(&messages)

	if len(messages) < 5 {
		return false
	}

	messagesText := ""
	for _, msg := range messages {
		senderName := msg.Sender.Nickname
		if senderName == "" {
			senderName = msg.Sender.Username
		}
		messagesText += senderName + ": " + msg.Content + "\n"
	}

	systemPrompt := aiprompt.CurrentTimeLine() + "\n\n" + `你是一个群聊总结助手。请分析以下群聊记录，生成简洁的每日总结。

总结格式：
📋 【群聊日报】- {日期}

📊 概览
- 今日消息数：X 条
- 活跃成员：X 人

🔥 热门话题
1. 话题一（参与人数）
2. 话题二（参与人数）

✅ 待办事项
- [ ] 待办一（负责人）
- [ ] 待办二（负责人）

💡 重要决策
- 决策一
- 决策二

请只输出总结内容，不要其他说明。`

	messages_input := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: messagesText},
	}

	summary, err := j.aiService.GetCompletion(ai.TaskTypeAnalysis, messages_input)
	if err != nil {
		log.Printf("[GroupSummary] 群 %d 总结生成失败: %v", group.ID, err)
		return false
	}

	summaryMsg := model.SystemMessage{
		Title:      "📋 群聊日报 - " + groupInfo.Name,
		Content:    summary,
		SenderID:   1,
		Status:     "active",
		TargetType: "group",
		TargetID:   &group.ID,
		CreatedAt:  time.Now(),
	}
	db.Create(&summaryMsg)

	log.Printf("[GroupSummary] 群 %d (%s) 总结已生成", group.ID, groupInfo.Name)
	return true
}

// maybeRememberSenderMessage 异步把"发送者本人发的消息"择要写入发送者记忆库。
// 写入对象是主人自己说的话（而非触发消息/分身回复）——这是最能反映主人偏好的信号。
// 三层门控压成本：1. 便宜规则预筛（无 LLM）2. 去重（向量近邻，无 LLM）3. LLM 质量门。
// 仅当发送者分身已启用时才记；不阻塞消息主流程。
func (e *SmartReplyEngine) maybeRememberSenderMessage(senderID uint, conversationID uint, content string) {
	if e.memorySvc == nil {
		return
	}
	if !looksMemorable(content) {
		return
	}
	go func() {
		db := database.GetDB()
		var config model.AvatarConfig
		if err := db.Select("enabled", "knowledge_scope_json").Where("user_id = ?", senderID).First(&config).Error; err != nil || !config.Enabled {
			return
		}
		// 记忆总开关：分身「记忆」关闭时后台也不再学习写入（避免关了还在偷偷攒记忆）。
		// 判定与回复召回共用 config.MemoryEnabled，保证"关掉记忆 = 既不读也不学"。
		if !config.MemoryEnabled() {
			return
		}
		// 一次 Recall(top-3)：top-1 用于去重，全部传给 Consolidate 避免二次查询
		existing, _ := e.memorySvc.Recall(senderID, content, 3)
		if len(existing) > 0 && existing[0].Score > 0.85 {
			return // 已有高度相似记忆，跳过
		}
		// 分数门槛：低于 0.6 的召回结果视为噪音，不传给 Consolidate 避免误记
		filtered := existing[:0]
		for _, r := range existing {
			if r.Score >= 0.6 {
				filtered = append(filtered, r)
			}
		}
		existing = filtered
		// 对话上下文：取最近 5 条消息，帮助 LLM 理解"这句话在讨论什么"
		ctx := fetchRecentMessages(db, conversationID, content, 5)
		// 记忆反射闭环：传入预取记忆，内部不再重复 Recall
		if ok, err := e.memorySvc.ConsolidateMessage(senderID, conversationID, content, ctx, existing...); err != nil {
			log.Printf("[AvatarMemory] 反射失败: user=%d err=%v", senderID, err)
		} else if ok {
			log.Printf("[AvatarMemory] 反射记忆已写入 user=%d", senderID)
		}
	}()
}

// maybeRememberGroupMessage 异步把值得记的群消息择要写入本群群级记忆库。
// 三层门控压成本：1. 便宜规则预筛（looksMemorable，无 LLM）2. 去重（向量近邻，无 LLM）3. LLM 质量门。
// 仅群 AI 启用时由调用方触发；不阻塞消息主流程。与分身记忆（按 userID 键）隔离。
func (e *SmartReplyEngine) maybeRememberGroupMessage(groupID uint, conversationID uint, content string) {
	if e.groupMemorySvc == nil {
		return
	}
	if !looksMemorable(content) {
		return
	}
	go func() {
		// 一次 Recall(top-3)：top-1 用于去重，全部传给 Consolidate 避免二次查询
		existing, _ := e.groupMemorySvc.Recall(groupID, content, 3)
		if len(existing) > 0 && existing[0].Score > 0.85 {
			return // 已有高度相似记忆，跳过
		}
		// 分数门槛：低于 0.6 的召回结果视为噪音，不传给 Consolidate 避免误记
		filtered := existing[:0]
		for _, r := range existing {
			if r.Score >= 0.6 {
				filtered = append(filtered, r)
			}
		}
		existing = filtered
		db := database.GetDB()
		// 对话上下文：取最近 5 条消息
		ctx := fetchRecentMessages(db, conversationID, content, 5)
		// 群知识片段：语义召回本群知识库，作为反射的折叠素材（群记忆可以借用群知识）
		var knowledge []string
		if e.unifiedKnowledge != nil {
			for _, snip := range e.unifiedKnowledge.Search(content, groupID, 3) {
				if snip.Content != "" {
					knowledge = append(knowledge, snip.Content)
				}
			}
		}
		// 群记忆反射闭环：内部完成"是否值得记 + 重要度 + 折叠既有群记忆/群知识"，仅值得记时落库
		if ok, err := e.groupMemorySvc.ConsolidateGroupMessage(groupID, conversationID, content, knowledge, ctx, existing...); err != nil {
			log.Printf("[GroupMemory] 反射失败: group=%d err=%v", groupID, err)
		} else if ok {
			log.Printf("[GroupMemory] 反射记忆已写入 group=%d", groupID)
		}
	}()
}

// fetchRecentMessages 从 DB 查最近 N 条消息（不含当前消息），格式化为
// "[发言人]: 内容" 供记忆反射的 LLM 理解对话语境。查不到时返回 nil（向后兼容）。
// 当前消息是最新的一条（触发本函数的那条），DESC 排序后跳过第一条即可。
func fetchRecentMessages(db *gorm.DB, conversationID uint, currentContent string, limit int) []string {
	var messages []model.Message
	db.Where("conversation_id = ? AND type IN ?", conversationID, []string{"text", "markdown"}).
		Preload("Sender").
		Order("created_at DESC").
		Limit(limit + 1). // +1 是当前消息，跳过
		Find(&messages)

	var result []string
	skippedCurrent := false
	for _, msg := range messages {
		// 跳过当前消息：DESC 排序下第一条就是最新的（当前消息），
		// 用 content 匹配兜底（极端情况下两条消息内容相同时仍可能误跳，
		// 但概率极低且影响仅为少一条上下文，可接受）。
		if !skippedCurrent && msg.Content == currentContent {
			skippedCurrent = true
			continue
		}
		name := msg.Sender.Nickname
		if name == "" {
			name = msg.Sender.Username
		}
		result = append(result, fmt.Sprintf("[%s]: %s", name, msg.Content))
		if len(result) >= limit {
			break
		}
	}
	// 倒序：让结果按时间正序（旧→新）
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}

// looksMemorable 便宜预筛，过滤明显不值得记的消息（短消息、纯 @AI 指令）。
func looksMemorable(content string) bool {
	if len([]rune(content)) <= 15 {
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(content))
	if strings.HasPrefix(lower, "@ai") {
		return false
	}
	return true
}

// checkAvatarTriggers 检查是否有用户的分身需要触发
func (e *SmartReplyEngine) checkAvatarTriggers(senderID uint, conv *model.Conversation, content string, triggerMsgType string, mentionUserIDs []uint) {
	db := database.GetDB()

	// 会话级覆盖：有 session 行=显式（AvatarEnabled=true→opt-in / false→opt-out），无行=跟随全局默认
	var sessions []model.AvatarSession
	db.Where("conversation_id = ?", conv.ID).Find(&sessions)
	override := make(map[uint]*model.AvatarSession, len(sessions))
	optInCount := 0
	for i := range sessions {
		override[sessions[i].UserID] = &sessions[i]
		if sessions[i].AvatarEnabled {
			optInCount++
		}
	}

	var convMembers []model.ConversationMember
	db.Where("conversation_id = ? AND user_id != ?", conv.ID, senderID).Find(&convMembers)

	// 候选集 = 显式 opt-in session + 无覆盖行且全局(Enabled && ActivateByDefault)的成员
	candidates := make([]*model.AvatarSession, 0, len(sessions))
	for i := range sessions {
		if sessions[i].AvatarEnabled {
			candidates = append(candidates, &sessions[i])
		}
	}

	var noOverrideUserIDs []uint
	for _, m := range convMembers {
		if _, has := override[m.UserID]; !has {
			noOverrideUserIDs = append(noOverrideUserIDs, m.UserID)
		}
	}
	defaultOnCount := 0
	if len(noOverrideUserIDs) > 0 {
		var configs []model.AvatarConfig
		db.Where("user_id IN ? AND enabled = ? AND activate_by_default = ?", noOverrideUserIDs, true, true).Find(&configs)
		defaultOnCount = len(configs)
		for i := range configs {
			// 默认开成员无 session 行：构造合成分身会话（无 takeover 状态），供 shouldTriggerAvatar 复用
			candidates = append(candidates, &model.AvatarSession{
				UserID:         configs[i].UserID,
				ConversationID: conv.ID,
				AvatarEnabled:  true,
			})
		}
	}

	log.Printf("[AvatarTrigger] 候选分身: convID=%d senderID=%d opt-in=%d 默认开=%d", conv.ID, senderID, optInCount, defaultOnCount)

	// 发送者信息在循环外只查一次，供触发判断（smart 模式需要 senderName）与任务构造复用
	var sender model.User
	if err := db.First(&sender, senderID).Error; err != nil {
		log.Printf("[AvatarTrigger] 获取发送者失败，跳过分身触发: senderID=%d err=%v", senderID, err)
		return
	}
	senderName := sender.Nickname
	if senderName == "" {
		senderName = sender.Username
	}

	isGroupChat := conv.Type == "group" || conv.Type == "discussion"
	for _, session := range candidates {
		if session.UserID == senderID {
			log.Printf("[AvatarTrigger] 跳过自己的分身: userID=%d == senderID=%d", session.UserID, senderID)
			continue
		}
		triggered := e.shouldTriggerAvatar(session, content, isGroupChat, mentionUserIDs, senderName)
		log.Printf("[AvatarTrigger] 触发判断结果: userID=%d triggered=%v (isGroupChat=%v mentionUserIDs=%v)", session.UserID, triggered, isGroupChat, mentionUserIDs)

		if triggered {
			groupName := ""
			if isGroupChat {
				var group model.Group
				if err := db.Where("conversation_id = ?", conv.ID).First(&group).Error; err == nil {
					groupName = group.Name
				}
			}

			task := service.AvatarTask{
				UserID:         session.UserID,
				ConversationID: conv.ID,
				TriggerMessage: content,
				TriggerMsgType: triggerMsgType,
				TriggerUserID:  senderID,
				IsGroupChat:    isGroupChat,
				GroupName:      groupName,
				TriggerName:    sender.Nickname,
			}

			if err := e.avatarWorkerPool.Submit(task); err != nil {
				log.Printf("[AvatarTrigger] 提交分身任务失败: %v", err)
			} else {
				log.Printf("[AvatarTrigger] 已触发用户 %d 的分身 (convID=%d triggerUserID=%d)", session.UserID, conv.ID, senderID)
			}
		}
	}
}

// shouldTriggerAvatar 判断是否应该触发分身。
// 排除列表/时间窗/模式分发全部委托给 AvatarTriggerService.DecideReply 统一决策，
// 这里只保留会话级 takeover 早退与配置加载，避免与触发服务重复判断。
func (e *SmartReplyEngine) shouldTriggerAvatar(session *model.AvatarSession, content string, isGroupChat bool, mentionUserIDs []uint, senderName string) bool {
	db := database.GetDB()

	if session.TakeoverUntil != nil && session.TakeoverUntil.After(time.Now()) {
		log.Printf("[AvatarTrigger] 分身接管期内，不触发: userID=%d takeoverUntil=%v", session.UserID, session.TakeoverUntil)
		return false
	}

	var config model.AvatarConfig
	if err := db.Where("user_id = ?", session.UserID).First(&config).Error; err != nil {
		log.Printf("[AvatarTrigger] 分身配置未找到: userID=%d err=%v", session.UserID, err)
		return false
	}

	if e.avatarTriggerSvc == nil {
		log.Printf("[AvatarTrigger] 触发决策服务未初始化: userID=%d", session.UserID)
		return false
	}

	shouldReply, reason, err := e.avatarTriggerSvc.DecideReply(config, session.ConversationID, content, senderName, isGroupChat, mentionUserIDs)
	if err != nil {
		log.Printf("[AvatarTrigger] 触发决策失败: userID=%d err=%v", session.UserID, err)
		return false
	}
	log.Printf("[AvatarTrigger] 触发判断结果: userID=%d shouldReply=%v reason=%s (isGroupChat=%v mentionUserIDs=%v)", session.UserID, shouldReply, reason, isGroupChat, mentionUserIDs)
	return shouldReply
}
