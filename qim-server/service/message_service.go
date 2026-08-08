package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/mention"
	"github.com/dshmyz/qim/qim-server/utils"
	"github.com/dshmyz/qim/qim-server/ws"

	"gorm.io/gorm"
)

var ErrMessageNotFound = errors.New("message not found")
var ErrMessageForbidden = errors.New("access forbidden")
var ErrMessageAlreadyRecalled = errors.New("message already recalled")
var ErrMessageRecallTimeout = errors.New("message recall timeout")
var ErrSensitiveWordBlocked = errors.New("message contains sensitive words")
var ErrMuted = errors.New("you are muted in this conversation")

// NoteSearcher 笔记检索接口（用于 bot 回复时按创建者 scope 检索笔记作为知识库）。
// *NoteVectorService 天然实现此接口；测试时可注入 mock。
type NoteSearcher interface {
	SearchNotes(userID uint, query string, topK int) ([]SearchResult, error)
}

type MessageService struct {
	db  *gorm.DB
	hub *ws.Hub

	aiService            *ai.AIService
	noteSearcher         NoteSearcher // bot 回复时按创建者 scope 检索笔记；nil=降级（不检索）
	groupMemorySvc       *GroupMemoryService // 群记忆（外部 bot 被 @ 时注入群上下文）；nil=降级（不拼接）
	groupKnowledgeSvc    *GroupDocumentService // 群知识库；nil=降级（不拼接）
	sensitiveWordCache   []model.SensitiveWord
	sensitiveWordCacheMu sync.RWMutex
	sensitiveWordLoaded  bool
}

func NewMessageService(db *gorm.DB, hub *ws.Hub, aiService *ai.AIService) *MessageService {
	return &MessageService{
		db:        db,
		hub:       hub,
		aiService: aiService,
	}
}

func (s *MessageService) SetAIService(aiService *ai.AIService) {
	s.aiService = aiService
}

// SetNoteSearcher 注入笔记检索服务（用于 bot internal_ai 模式下读取创建者笔记）。
// 传 nil 即可关闭该能力（向量库未配置时安全降级）。
func (s *MessageService) SetNoteSearcher(searcher NoteSearcher) {
	s.noteSearcher = searcher
}

// SetGroupContextServices 注入群记忆 + 群知识库服务，供外部 agent bot 被 @ 时把
// 群积累上下文注入 webhook payload。两者均可传 nil：任一缺失即跳过对应片段，
// 不影响主流程（群未开启向量库/记忆时安全降级）。
func (s *MessageService) SetGroupContextServices(memorySvc *GroupMemoryService, knowledgeSvc *GroupDocumentService) {
	s.groupMemorySvc = memorySvc
	s.groupKnowledgeSvc = knowledgeSvc
}

// loadSensitiveWords 从数据库加载启用的敏感词到内存缓存，返回 DB 错误。
// 历史问题：原先用 if err == nil 静默吞掉错误，导致 CRUD 成功但缓存刷新失败时
// 管理员看到"成功"，实际新词未被过滤。现返回 error 由调用方处理并记录日志。
func (s *MessageService) loadSensitiveWords() error {
	var words []model.SensitiveWord
	if err := s.db.Where("enabled = ?", true).Find(&words).Error; err != nil {
		return err
	}
	s.sensitiveWordCacheMu.Lock()
	s.sensitiveWordCache = words
	s.sensitiveWordLoaded = true
	s.sensitiveWordCacheMu.Unlock()
	return nil
}

// RefreshSensitiveWordCache 刷新敏感词缓存，返回 DB 错误以便调用方感知失败。
func (s *MessageService) RefreshSensitiveWordCache() error {
	return s.loadSensitiveWords()
}

func (s *MessageService) CheckSensitiveContent(content string) (bool, []string) {
	s.sensitiveWordCacheMu.RLock()
	loaded := s.sensitiveWordLoaded
	cache := s.sensitiveWordCache
	s.sensitiveWordCacheMu.RUnlock()

	if !loaded {
		// 加载失败时记录日志（fail-open：不阻断消息发送，但需让运维感知过滤已降级）。
		// 历史问题：原先静默吞掉错误，DB 异常时敏感词过滤悄悄失效却无人知晓。
		if err := s.loadSensitiveWords(); err != nil {
			logger.WithModule("SensitiveWord").Error("加载敏感词缓存失败，敏感词过滤暂时降级", "error", err)
		}
		s.sensitiveWordCacheMu.RLock()
		cache = s.sensitiveWordCache
		s.sensitiveWordCacheMu.RUnlock()
	}

	found := []string{}
	for _, word := range cache {
		if strings.Contains(content, word.Word) {
			found = append(found, word.Word)
		}
	}
	return len(found) > 0, found
}

type MessageQuery struct {
	ConvID      uint
	UserID      uint
	BeforeMsgID uint
	AfterMsgID  uint
	Limit       int
	Offset      int
	MessageType string
	Keyword     string
	StartDate   string
	EndDate     string
}

type MessageResult struct {
	Messages    []model.Message
	Total       int64
	TotalPages  int
	CurrentPage int
	PageSize    int
}

func (s *MessageService) SendMessage(convID, senderID uint, msgType, content string, quotedMessageID *uint) (*model.Message, error) {
	db := s.db

	if (msgType == "text" || msgType == "markdown") && content != "" {
		if contains, words := s.CheckSensitiveContent(content); contains {
			return nil, fmt.Errorf("%w: %v", ErrSensitiveWordBlocked, words)
		}
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, senderID).First(&member).Error; err != nil {
		return nil, ErrMessageForbidden
	}

	// 群级禁言检查：被禁言且未到期则拒绝发言（群主/管理员豁免，保证管理动作不受阻）
	if member.MutedUntil != nil && member.MutedUntil.After(time.Now()) && member.Role != "owner" && member.Role != "admin" {
		return nil, ErrMuted
	}

	// 解析 content 中的 @ mention token（content 是唯一事实源）
	mentions := mention.Parse(content)

	msg := model.Message{
		ConversationID:  convID,
		SenderID:        senderID,
		Type:            msgType,
		Content:         content,
		QuotedMessageID: quotedMessageID,
		IsRead:          false,
	}
	if err := db.Create(&msg).Error; err != nil {
		return nil, err
	}

	// 优化：单次预加载而非 3 次 Preload 调用
	db.Preload("Sender").Preload("QuotedMessage").Preload("QuotedMessage.Sender").First(&msg, msg.ID)

	now := time.Now()
	// 优化：合并查会话 + 更新会话为单次 UPDATE
	result := db.Exec("UPDATE conversations SET last_message_id = ?, last_message_at = ? WHERE id = ?", msg.ID, now, convID)
	if result.Error != nil {
		return nil, result.Error
	}

	// 获取会话类型用于判断 bot/正常
	var convType string
	db.Model(&model.Conversation{}).Where("id = ?", convID).Select("type").First(&convType)

	if convType == "bot" {
		utils.SafeGo(func() { s.handleBotMessage(senderID, convID, content) })
	} else {
		// 恢复会话显示：新消息到来时，如果会话被隐藏则恢复显示
		db.Model(&model.ConversationSession{}).
			Where("conversation_id = ? AND is_hidden = ?", convID, true).
			Update("is_hidden", false)

		db.Model(&model.ConversationMember{}).
			Where("conversation_id = ? AND user_id != ?", convID, senderID).
			UpdateColumn("unread_count", gorm.Expr("unread_count + 1"))

		// 计算被提及的用户 ID（@all 展开为全体成员，排除发送者）
		mentionUserIDs := s.resolveMentionUserIDs(convID, mentions, senderID)

		// 更新被提及成员的未读 @ 计数
		if len(mentionUserIDs) > 0 {
			db.Model(&model.ConversationMember{}).
				Where("conversation_id = ? AND user_id IN ?", convID, mentionUserIDs).
				UpdateColumn("unread_at_mention_count", gorm.Expr("unread_at_mention_count + 1"))
		}

		if s.hub != nil {
			// 广播（mention_user_ids 数组随消息发送，前端据此算 is_at_mention）
			s.broadcastMessage(&msg, mentionUserIDs, senderID)
			// HTTP 与 WebSocket 都经由本 service 发送，因此在此统一触发一次智能回复/分身回调。
			if s.hub.OnMessageSent != nil && !mention.IsAllMentioned(mentions) {
				utils.SafeGo(func() {
					s.hub.OnMessageSent(&msg, mentionUserIDs)
				})
			}
		}
	}

	return &msg, nil
}

// resolveMentionUserIDs 解析 mentions，展开 @all，返回被提及的用户 ID 列表。
// excludeUserID 用于排除指定用户（如发送者）。
func (s *MessageService) resolveMentionUserIDs(convID uint, mentions []mention.Mention, excludeUserID uint) []uint {
	var allMembers []model.ConversationMember
	if mention.IsAllMentioned(mentions) {
		if err := s.db.Where("conversation_id = ?", convID).Find(&allMembers).Error; err != nil {
			allMembers = nil
		}
	}
	allMemberIDs := make([]uint, 0, len(allMembers))
	for _, m := range allMembers {
		allMemberIDs = append(allMemberIDs, m.UserID)
	}
	return mention.ExtractUserIDs(mentions, allMemberIDs, excludeUserID)
}

// MentionUserIDsForAI 解析 content 中的 @ mention，展开 @all，返回被提及的用户 ID 列表。
// 供外部 AI 触发逻辑使用（如 app/routes.go 的 OnMessageSent 回调），不持久化。
func (s *MessageService) MentionUserIDsForAI(convID uint, content string) []uint {
	mentions := mention.Parse(content)
	// excludeUserID=0：不排除任何用户（AI 触发场景由调用方决定）
	return s.resolveMentionUserIDs(convID, mentions, 0)
}

// HandleGroupBotMention 群聊外部 agent 触发：群消息 @ 到某 bot 虚拟用户时，把该消息
// 转发给对应外部 webhook agent（thread_id = 群会话 id）。仅在群会话、且 bot 通过
// BotConversation 关联到本会话时触发——天然克制（不被 @ 就不转发）。
// 内部 AI bot（assistant/system）暂不接群聊，保持既有单聊行为不变。
func (s *MessageService) HandleGroupBotMention(convID, senderID uint, mentionedUserIDs []uint, content string) {
	if len(mentionedUserIDs) == 0 {
		return
	}
	var convType string
	if err := s.db.Model(&model.Conversation{}).Where("id = ?", convID).Select("type").
		First(&convType).Error; err != nil || convType != "group" {
		return // 仅群会话
	}

	// 反查被 @ 的用户里，哪些是本会话已关联 bot 的虚拟用户。
	// BotConversation{conversation_id=群} 在 Phase 3「拉 bot 进群」时建立。
	var bots []model.Bot
	if err := s.db.
		Joins("JOIN bot_conversations bc ON bc.bot_id = bots.id AND bc.conversation_id = ?", convID).
		Where("bots.virtual_user_id IN ?", mentionedUserIDs).
		Find(&bots).Error; err != nil {
		logger.WithModule("HandleGroupBotMention").Error("查询群内被 @ 的 bot 失败", "convID", convID, "error", err)
		return
	}
	for _, bot := range bots {
		if bot.VirtualUserID == nil || !bot.IsActive {
			continue
		}
		cfg := ParseBotConfig(bot.Config)
		if !cfg.IsExternalWebhook() {
			continue // 仅外部 webhook agent
		}
		if cfg.WebhookURL == "" {
			// pull 模式外部 bot：无回调地址，不投递也不会自动回复。
			// 向群内回一条系统提示，避免成员 @ 后无反应却不知何故（可感知）。
			// 仅在被 @ 该 bot 时触发，天然克制；系统消息不经 SendMessage，不会递归再触发 OnMessageSent。
			s.notifyPullModeBot(convID, senderID, bot.Name)
			continue
		}
		groupContext := s.buildGroupBotContext(convID, content)
		s.forwardBotMessageToWebhook(bot, cfg.WebhookURL, cfg.WebhookSecret, senderID, convID, content, groupContext)
	}
}

// buildGroupBotContext 为群聊外部 agent 被 @ 时，拼接本群「群记忆 + 群知识库」上下文，
// 注入 webhook payload（GroupContext），让外部 bot 能参考群积累内容回复。
// 规则与群 AI 助手（smart_reply_graph.go handleAIMention）对齐：
//   - 仅群会话且本群开启 AI（ai_config.enabled）时拼接；否则返回空串（单聊/未启用群不注入）。
//   - 记忆取 top2 条，知识取 top3 条；任一服务未注入（nil）或检索失败即跳过对应片段，不阻断投递。
//
// 返回空串表示无上下文，调用方直接置空，不影响既有 payload 结构（向后兼容）。
func (s *MessageService) buildGroupBotContext(convID uint, content string) string {
	if s.groupMemorySvc == nil && s.groupKnowledgeSvc == nil {
		return ""
	}
	// 反查群：非群会话无 groups 行，快速跳过
	var group model.Group
	if err := s.db.Where("conversation_id = ?", convID).First(&group).Error; err != nil {
		return "" // 非群会话 / 群已解散
	}
	// 群未启用 AI 时不注入（与群助手触发条件对齐）
	if cfg := group.GetAIConfig(); cfg == nil || !cfg.Enabled {
		return ""
	}

	var parts []string
	if s.groupMemorySvc != nil {
		if memories, err := s.groupMemorySvc.Recall(group.ID, content, 2); err == nil && len(memories) > 0 {
			memParts := make([]string, 0, len(memories))
			for _, mem := range memories {
				if mem.Content == "" {
					continue
				}
				memParts = append(memParts, fmt.Sprintf("• %s（相关度: %.1f%%）", mem.Content, mem.Score*100))
			}
			if len(memParts) > 0 {
				parts = append(parts, "🧠 本群近期记忆：\n"+strings.Join(memParts, "\n"))
			}
		} else if err != nil {
			logger.WithModule("handleBotMessage").Debug("群记忆检索失败，跳过记忆片段",
				"groupID", group.ID, "convID", convID, "error", err)
		}
	}
	if s.groupKnowledgeSvc != nil {
		if kn, err := s.groupKnowledgeSvc.SearchKnowledge(group.ID, content, 3); err == nil && len(kn) > 0 {
			knParts := make([]string, 0, len(kn))
			for _, k := range kn {
				if k.Content == "" {
					continue
				}
				prefix := "• "
				if title := k.Metadata["title"]; title != "" {
					prefix = fmt.Sprintf("• 【%s】", title)
				} else if name := k.Metadata["name"]; name != "" {
					prefix = fmt.Sprintf("• 【%s】", name)
				}
				knParts = append(knParts, prefix+k.Content)
			}
			if len(knParts) > 0 {
				parts = append(parts, "📚 本群知识库相关内容：\n"+strings.Join(knParts, "\n"))
			}
		} else if err != nil {
			logger.WithModule("handleBotMessage").Debug("群知识库检索失败，跳过知识片段",
				"groupID", group.ID, "convID", convID, "error", err)
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n\n")
}

// notifyPullModeBot 在群会话内落一条系统提示消息：被 @ 的外部 bot 未配 webhook 回调（pull 模式），不会自动回复。
// 复用 NotifyMembersJoined 的系统消息 + WS 广播模式，便于成员在聊天流里看到原因。
func (s *MessageService) notifyPullModeBot(convID, senderID uint, botName string) {
	content := fmt.Sprintf("机器人「%s」未配置回调地址（pull 模式），不会自动回复；如需 @ 即回复，请到机器人设置填写 webhook_url。", botName)
	systemMsg := &model.Message{
		ConversationID: convID,
		SenderID:       senderID,
		Type:           "system",
		Content:        content,
		IsRead:         true,
	}
	if err := s.db.Create(systemMsg).Error; err != nil {
		logger.WithModule("HandleGroupBotMention").Error("创建 pull 模式提示消息失败", "convID", convID, "error", err)
		return
	}
	s.db.Model(&model.Conversation{}).Where("id = ?", convID).Updates(map[string]interface{}{
		"last_message_id": systemMsg.ID,
		"last_message_at": time.Now(),
	})
	if s.hub != nil {
		var sender model.User
		if err := s.db.First(&sender, senderID).Error; err == nil {
			systemMsg.Sender = sender
		}
		newMsg := ws.WSMessage{
			Type: "new_message",
			Data: map[string]interface{}{
				"id":              systemMsg.ID,
				"conversation_id": systemMsg.ConversationID,
				"sender_id":       systemMsg.SenderID,
				"type":            systemMsg.Type,
				"content":         systemMsg.Content,
				"is_read":         systemMsg.IsRead,
				"created_at":      systemMsg.CreatedAt,
				"sender":          systemMsg.Sender,
			},
		}
		jsonMsg, _ := json.Marshal(newMsg)
		s.hub.SendToConversation(convID, 0, jsonMsg)
	}
}

func (s *MessageService) handleBotMessage(userID, convID uint, content string) {
	db := s.db

	var botConv model.BotConversation
	if err := db.Where("conversation_id = ?", convID).First(&botConv).Error; err != nil {
		return
	}

	var bot model.Bot
	if err := db.First(&bot, botConv.BotID).Error; err != nil {
		return
	}

	// Bot 必须有虚拟用户才能回复
	if bot.VirtualUserID == nil {
		logger.WithModule("handleBotMessage").Warn("Bot 没有虚拟用户", "botID", bot.ID)
		return
	}

	// 外部 agent 模式：转发用户消息到 webhook，不再走内部 AI
	// 单聊（1:1 bot 会话）不注入群上下文——本函数仅群聊 @ 场景需要群记忆/知识。
	if cfg := ParseBotConfig(bot.Config); cfg.IsExternalWebhook() {
		s.forwardBotMessageToWebhook(bot, cfg.WebhookURL, cfg.WebhookSecret, userID, convID, content, "")
		return
	}

	var messages []model.Message
	db.Where("conversation_id = ?", convID).Order("created_at ASC").Limit(20).Find(&messages)

	aiMessages := make([]ai.Message, 0, len(messages)+1)

	// 创建者笔记作为知识库：开关开启 + noteSearcher 可用时，按 bot.CreatorID scope 检索
	// scope 隔离：SearchNotes 内部按 user_notes_<userID> 分集合，只能读到创建者自己的笔记
	// 降级策略：noteSearcher==nil（向量库未配）或检索出错时静默跳过，不影响主流程
	// 命中笔记的标题/分数会写入 message.Extra（JSON），前端据此渲染折叠「知识来源」标签
	botCfg := ParseBotConfig(bot.Config)
	var knowledgeSources []map[string]interface{}
	if botCfg.UseCreatorNotes && s.noteSearcher != nil && bot.CreatorID > 0 {
		noteResults, err := s.noteSearcher.SearchNotes(bot.CreatorID, content, 3)
		if err == nil && len(noteResults) > 0 {
			parts := make([]string, 0, len(noteResults))
			// 记录命中笔记的 ID / 标题 / 分数，便于审计与排查检索质量
			hitLogs := make([]string, 0, len(noteResults))
			knowledgeSources = make([]map[string]interface{}, 0, len(noteResults))
			for _, r := range noteResults {
				title := r.Metadata["title"]
				if title == "" {
					title = "未命名"
				}
				parts = append(parts, fmt.Sprintf("[笔记: %s]\n%s", title, r.Content))
				hitLogs = append(hitLogs, fmt.Sprintf("docID=%s title=%s score=%.4f", r.DocID, title, r.Score))
				// 收集前端展示所需的最小结构（不暴露笔记正文，避免 message 响应体过大/泄漏）
				knowledgeSources = append(knowledgeSources, map[string]interface{}{
					"title": title,
					"score": r.Score,
				})
			}
			logger.WithModule("handleBotMessage").Info("命中创建者笔记",
				"botID", bot.ID, "creatorID", bot.CreatorID, "hits", len(noteResults), "notes", strings.Join(hitLogs, " | "))
			aiMessages = append(aiMessages, ai.Message{
				Role: "system",
				Content: "以下是创建者的相关笔记，可作为回答参考（请基于笔记内容作答，" +
					"笔记未覆盖的问题按你的通用能力回答）：\n\n" +
					strings.Join(parts, "\n\n"),
			})
		} else if err != nil {
			logger.WithModule("handleBotMessage").Warn("笔记检索失败，降级为不注入", "botID", bot.ID, "error", err)
		}
	}

	for _, msg := range messages {
		role := "user"
		if msg.SenderID == *bot.VirtualUserID {
			role = "assistant"
		}
		aiMessages = append(aiMessages, ai.Message{
			Role:    role,
			Content: msg.Content,
		})
	}

	var builder strings.Builder
	var streamErr error
	if s.aiService == nil {
		streamErr = fmt.Errorf("AI 服务未配置")
	} else {
		// 自定义模型来源：bot 配置了「使用我的自定义配置」（!UseSystemConfig 且 UserConfigID 非空）时，
		// 解析出创建者自选的 provider。解析失败（配置被删/禁用/密钥解密失败）→ 回退系统默认。
		// external_webhook 已在前面 return，走到这里必定是 internal_ai。
		var custom *customProvider
		if !botCfg.UseSystemConfig && botCfg.UserConfigID != nil {
			custom = resolveUserAIConfigProvider(db, bot.CreatorID, *botCfg.UserConfigID)
			if custom == nil || custom.ProviderName == "" {
				logger.WithModule("handleBotMessage").Warn("bot 自定义模型解析失败，回退系统默认",
					"botID", bot.ID, "userConfigID", *botCfg.UserConfigID)
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		done := make(chan struct{})
		go func() {
			onChunk := func(chunk ai.StreamChunk) error {
				builder.WriteString(chunk.Content)
				return nil
			}
			if custom != nil && custom.ProviderName != "" {
				streamErr = s.aiService.ChatStreamWithProviderConfig(ctx, ai.TaskTypeChat, aiMessages, custom.ProviderName, custom.Config, onChunk)
				// 生成期失败（密钥失效/配额耗尽/网络错/模型名错误）且尚未流出内容时，回退系统默认，
				// 兑现「回退系统默认…不阻断回复」契约——单条用户自定义配置出问题不应拖垮整个 bot 回复。
				// 已流出部分内容则保留（流式中途无法干净衔接重试）。
				if streamErr != nil && builder.Len() == 0 {
					logger.WithModule("handleBotMessage").Warn("bot 自定义模型生成失败，回退系统默认",
						"botID", bot.ID, "provider", custom.ProviderName, "error", streamErr)
					streamErr = s.aiService.GetCompletionStreamWithContext(ctx, ai.TaskTypeChat, aiMessages, onChunk)
				}
			} else {
				streamErr = s.aiService.GetCompletionStreamWithContext(ctx, ai.TaskTypeChat, aiMessages, onChunk)
			}
			close(done)
		}()

		// 超时保护：AI 调用最长 60 秒
		select {
		case <-done:
			// 正常完成
		case <-ctx.Done():
			streamErr = fmt.Errorf("AI 响应超时")
			cancel() // 取消 context，关闭 HTTP 连接，防止协程泄漏
			<-done   // 等待子 goroutine 退出，避免并发读写 builder
		}
	}

	response := builder.String()
	// 自定义模型已流出部分内容后中途失败（builder.Len()>0）：保留已生成的部分，兑现
	// 「已流出部分内容则保留」注释契约，而不是把整个回复降级成统一错误文案。
	// 仅在完全没流出任何内容（含回退系统默认也失败）时，才用兜底错误文案。
	if response == "" && streamErr != nil {
		logger.WithModule("handleBotMessage").Error("AI API error", "error", streamErr)
		response = "抱歉，AI 服务暂时不可用，请稍后再试。"
	}

	senderID := *bot.VirtualUserID
	botReply := model.Message{
		ConversationID: convID,
		SenderID:       senderID,
		Type:           "markdown",
		Content:        response,
		Origin:         "assistant",
		IsRead:         true, // Bot 回复默认已读
	}
	// 命中笔记时把 knowledge_sources 写入 Extra，供前端折叠展示
	// 无命中时留空（默认值），前端按空字符串判断不展示
	if len(knowledgeSources) > 0 {
		if extraBytes, err := json.Marshal(map[string]interface{}{
			"knowledge_sources": knowledgeSources,
		}); err == nil {
			botReply.Extra = string(extraBytes)
		} else {
			logger.WithModule("handleBotMessage").Warn("序列化 knowledge_sources 失败", "botID", bot.ID, "error", err)
		}
	}
	if err := db.Create(&botReply).Error; err != nil {
		logger.WithModule("handleBotMessage").Error("创建 Bot 回复失败", "error", err)
		return
	}

	if err := db.Preload("Sender").First(&botReply, botReply.ID).Error; err != nil {
		logger.WithModule("handleBotMessage").Error("预加载 Bot 回复发送者失败", "error", err, "messageID", botReply.ID)
		return
	}

	now := time.Now()
	if err := db.Model(&model.Conversation{}).
		Where("id = ?", convID).
		Updates(map[string]interface{}{
			"last_message_id": botReply.ID,
			"last_message_at": now,
		}).Error; err != nil {
		logger.WithModule("handleBotMessage").Error("更新 Bot 会话最后消息失败", "error", err, "conversationID", convID, "messageID", botReply.ID)
		return
	}

	if s.hub != nil {
		wsMsg := ws.WSMessage{
			Type: "new_message",
			Data: s.buildMessageResponse(botReply, nil),
		}
		jsonMsg, _ := json.Marshal(wsMsg)
		s.hub.SendToUser(userID, jsonMsg)
	}
}

// forwardBotMessageToWebhook 将用户在 bot 会话中的回复转发到外部 agent webhook。
// 由 handleBotMessage 在 external_webhook 模式下异步调用（handleBotMessage 本身已 SafeGo）。
// 经 outbox：先落表再立即 best-effort 投递一次，失败由调度器指数退避重试，超阈值死信。
// 成功路径与原直发等价（无额外延迟），失败路径由静默丢变为有兜底。
func (s *MessageService) forwardBotMessageToWebhook(bot model.Bot, webhookURL, webhookSecret string, userID, convID uint, content, groupContext string) {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		logger.WithModule("handleBotMessage").Error("查询用户失败", "userID", userID, "error", err)
		return
	}

	// 取刚发送的用户消息（用于 message_id / msg_type）
	var lastMsg model.Message
	msgType := "text"
	if err := s.db.Where("conversation_id = ? AND sender_id = ?", convID, userID).
		Order("created_at DESC").First(&lastMsg).Error; err == nil {
		msgType = lastMsg.Type
	}

	payload := BotWebhookPayload{
		BotID:        bot.ID,
		ThreadID:     convID,
		MessageID:    lastMsg.ID,
		UserID:       userID,
		UserNickname: user.Nickname,
		UserAvatar:   user.Avatar,
		Content:      content,
		MsgType:      msgType,
		GroupContext: groupContext,
	}
	payloadJSON, _ := json.Marshal(payload)

	// 纯 pull 模式（webhook_url 空）：不投 webhook，用户消息已在会话内，agent 靠 GET /bot/messages 拉取。
	if webhookURL == "" {
		logger.WithModule("handleBotMessage").Info("外部 bot 未配 webhook_url，走纯 pull 模式（不投递）",
			"botID", bot.ID, "convID", convID)
		return
	}

	deliveryID, err := EnqueueWebhookDelivery(s.db, bot.ID, "bot.message", string(payloadJSON), webhookURL, webhookSecret)
	if err != nil {
		logger.WithModule("handleBotMessage").Error("webhook outbox 入队失败",
			"botID", bot.ID, "convID", convID, "error", err)
		return
	}
	// 立即 best-effort 投递一次：成功等价于原直发，失败落 pending 待重试
	if err := DeliverOnce(s.db, deliveryID); err != nil {
		logger.WithModule("handleBotMessage").Warn("webhook 立即投递失败，已入重试队列",
			"deliveryID", deliveryID, "botID", bot.ID, "convID", convID, "error", err)
	}
}

func (s *MessageService) GetMessages(query MessageQuery) (*MessageResult, error) {
	db := s.db

	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Limit > 100 {
		query.Limit = 100
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", query.ConvID, query.UserID).First(&member).Error; err != nil {
		return nil, ErrMessageForbidden
	}

	var total int64
	db.Model(&model.Message{}).Where("conversation_id = ?", query.ConvID).Count(&total)

	totalPages := int(total) / query.Limit
	if int(total)%query.Limit > 0 {
		totalPages++
	}

	var messages []model.Message
	q := db.Where("conversation_id = ?", query.ConvID)

	if query.AfterMsgID > 0 {
		var afterMsg model.Message
		if err := db.First(&afterMsg, query.AfterMsgID).Error; err != nil {
			return &MessageResult{
				Messages:    []model.Message{},
				Total:       0,
				TotalPages:  0,
				CurrentPage: 1,
				PageSize:    query.Limit,
			}, nil
		}
		q = q.Where("created_at > ?", afterMsg.CreatedAt).Order("created_at ASC").Limit(query.Limit)
		q.Preload("Sender").Preload("QuotedMessage").Preload("QuotedMessage.Sender").Find(&messages)
		return &MessageResult{
			Messages:    messages,
			Total:       int64(len(messages)),
			TotalPages:  1,
			CurrentPage: 1,
			PageSize:    query.Limit,
		}, nil
	}

	if query.BeforeMsgID > 0 {
		var beforeMsg model.Message
		if err := db.First(&beforeMsg, query.BeforeMsgID).Error; err == nil {
			q = q.Where("created_at < ?", beforeMsg.CreatedAt)
		}
	}

	// DESC 查最新的 N 条，再翻转为正序（BeforeMsgID 游标分页需要 DESC）
	q.Preload("Sender").Preload("QuotedMessage").Preload("QuotedMessage.Sender").Order("created_at DESC").Limit(query.Limit).Offset(query.Offset).Find(&messages)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return &MessageResult{
		Messages:    messages,
		Total:       total,
		TotalPages:  totalPages,
		CurrentPage: query.Offset/query.Limit + 1,
		PageSize:    query.Limit,
	}, nil
}

func (s *MessageService) GetMessagesByFilter(query MessageQuery) (*MessageResult, error) {
	db := s.db

	if query.Limit <= 0 {
		query.Limit = 10
	}
	if query.Limit > 100 {
		query.Limit = 100
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", query.ConvID, query.UserID).First(&member).Error; err != nil {
		return nil, ErrMessageForbidden
	}

	dbQuery := db.Where("conversation_id = ?", query.ConvID)

	if query.MessageType != "" {
		dbQuery = dbQuery.Where("type = ?", query.MessageType)
	}

	// 优化：使用全文索引搜索
	if query.Keyword != "" {
		if database.D.SupportsFulltext() {
			dbQuery = dbQuery.Where("MATCH(content) AGAINST(? IN BOOLEAN MODE)", query.Keyword)
		} else {
			// SQLite / TiDB 降级：LIKE 搜索
			dbQuery = dbQuery.Where("content LIKE ?", "%"+query.Keyword+"%")
		}
	}

	if query.StartDate != "" {
		dbQuery = dbQuery.Where("created_at >= ?", query.StartDate)
	}
	if query.EndDate != "" {
		dbQuery = dbQuery.Where("created_at <= ?", query.EndDate+" 23:59:59")
	}

	var total int64
	dbQuery.Model(&model.Message{}).Count(&total)

	var messages []model.Message
	dbQuery.Preload("Sender").Preload("QuotedMessage").Preload("QuotedMessage.Sender").Order("created_at DESC").Limit(query.Limit).Offset(query.Offset).Find(&messages)

	totalPages := int(total) / query.Limit
	if int(total)%query.Limit > 0 {
		totalPages++
	}

	return &MessageResult{
		Messages:    messages,
		Total:       total,
		TotalPages:  totalPages,
		CurrentPage: query.Offset/query.Limit + 1,
		PageSize:    query.Limit,
	}, nil
}

func (s *MessageService) SearchMessages(userID uint, keyword string, convID *uint, limit, offset int) ([]model.Message, error) {
	db := s.db

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	query := db.Model(&model.Message{}).Joins("JOIN conversation_members ON messages.conversation_id = conversation_members.conversation_id").Where("conversation_members.user_id = ?", userID)

	if keyword != "" {
		// 优化：使用全文索引搜索
		if database.D.SupportsFulltext() {
			query = query.Where("MATCH(messages.content) AGAINST(? IN BOOLEAN MODE)", keyword)
		} else {
			// SQLite / TiDB 降级：LIKE 搜索
			query = query.Where("messages.content LIKE ?", "%"+keyword+"%")
		}
	}

	if convID != nil {
		query = query.Where("messages.conversation_id = ?", *convID)
	}

	var messages []model.Message
	if err := query.Preload("Sender").Preload("Conversation").Preload("Conversation.Members").Preload("Conversation.Members.User").Order("messages.created_at DESC").Limit(limit).Offset(offset).Find(&messages).Error; err != nil {
		return nil, err
	}

	return messages, nil
}

func (s *MessageService) RecallMessage(msgID, userID uint) (*model.Message, error) {
	db := s.db

	var msg model.Message
	if err := db.First(&msg, msgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	if msg.SenderID != userID {
		return nil, ErrMessageForbidden
	}

	if msg.IsRecalled {
		return nil, ErrMessageAlreadyRecalled
	}

	configSvc := NewSystemConfigService(db)
	publicConfigs, err := configSvc.GetPublicConfigs()
	if err == nil {
		recallTimeLimit := 120
		if v, ok := publicConfigs["messageRecallTime"]; ok {
			if iv, ok := v.(int); ok {
				recallTimeLimit = iv
			}
		}
		if recallTimeLimit == 0 {
			return nil, ErrMessageRecallTimeout
		}
		if time.Since(msg.CreatedAt) > time.Duration(recallTimeLimit)*time.Second {
			return nil, ErrMessageRecallTimeout
		}
	}

	// 保留原始内容到 Extra 字段，供「撤回后重新编辑」使用
	originalContent := msg.Content
	extraData := make(map[string]interface{})
	if msg.Extra != "" {
		// 如果 Extra 已有内容，解析并合并
		if err := json.Unmarshal([]byte(msg.Extra), &extraData); err != nil {
			// 解析失败则使用空 map
			extraData = make(map[string]interface{})
		}
	}
	extraData["original_content"] = originalContent
	if extraBytes, err := json.Marshal(extraData); err == nil {
		msg.Extra = string(extraBytes)
	}

	msg.IsRecalled = true
	msg.Content = "[消息已撤回]"
	now := time.Now()
	msg.RecalledAt = &now
	if err := db.Save(&msg).Error; err != nil {
		return nil, err
	}

	db.Preload("Sender").First(&msg, msg.ID)

	if s.hub != nil {
		recallMsg := ws.WSMessage{
			Type: "message_recalled",
			Data: msg,
		}
		jsonMsg, _ := json.Marshal(recallMsg)
		s.hub.SendToConversation(msg.ConversationID, 0, jsonMsg)
	}

	return &msg, nil
}

func (s *MessageService) DeleteMessage(msgID, userID uint) error {
	db := s.db

	var msg model.Message
	if err := db.First(&msg, msgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMessageNotFound
		}
		return err
	}

	if msg.SenderID != userID {
		return ErrMessageForbidden
	}

	if err := db.Delete(&msg).Error; err != nil {
		return err
	}

	if s.hub != nil {
		deleteMsg := ws.WSMessage{
			Type: "message_deleted",
			Data: map[string]interface{}{
				"message_id":      msg.ID,
				"conversation_id": msg.ConversationID,
			},
		}
		jsonMsg, _ := json.Marshal(deleteMsg)
		s.hub.SendToConversation(msg.ConversationID, 0, jsonMsg)
	}

	return nil
}

func (s *MessageService) MarkAsRead(convID, userID uint) error {
	db := s.db

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, userID).First(&member).Error; err != nil {
		return ErrMessageForbidden
	}

	// 用 per-user 的 message_read_receipts 表判断"该用户尚未读过的消息"，
	// 不能用 messages.is_read 全局字段——那是"是否被任何人读过"的缓存，
	// 会被第一个读者置 true，导致后续读者无法插入回执（群聊已读人数卡在 1）。
	now := time.Now()

	// 查询该用户尚未写入回执的消息 ID（即对该用户的未读消息）
	var unreadMsgIDs []uint
	if err := db.Model(&model.Message{}).
		Where("conversation_id = ? AND sender_id != ?", convID, userID).
		Where("id NOT IN (?)", db.Model(&model.MessageReadReceipt{}).Select("message_id").Where("user_id = ?", userID)).
		Pluck("id", &unreadMsgIDs).Error; err != nil {
		return err
	}

	// 即使没有未读，也要清零 unread_count 和推进 last_read_at（保证幂等调用正确）
	db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		UpdateColumns(map[string]interface{}{
			"unread_count":            0,
			"unread_at_mention_count": 0,
			"last_read_at":            now,
		})

	if len(unreadMsgIDs) == 0 {
		return nil
	}

	// 批量插入回执（按数据库方言选语法，唯一索引保证幂等）
	if database.D.Type() == "mysql" {
		db.Exec(`
			INSERT IGNORE INTO message_read_receipts (message_id, conversation_id, user_id, created_at)
			SELECT id, ?, ?, ?
			FROM messages
			WHERE conversation_id = ? AND sender_id != ?
			  AND id NOT IN (SELECT message_id FROM message_read_receipts WHERE user_id = ?)
		`, convID, userID, now, convID, userID, userID)
	} else {
		db.Exec(`
			INSERT INTO message_read_receipts (message_id, conversation_id, user_id, created_at)
			SELECT id, ?, ?, ?
			FROM messages
			WHERE conversation_id = ? AND sender_id != ?
			  AND id NOT IN (SELECT message_id FROM message_read_receipts WHERE user_id = ?)
			ON CONFLICT (message_id, user_id) DO NOTHING
		`, convID, userID, now, convID, userID, userID)
	}

	// messages.is_read 仅作为"是否被任何人读过"的缓存标志：第一个读者置 true 即可
	db.Model(&model.Message{}).
		Where("id IN ? AND is_read = false", unreadMsgIDs).
		UpdateColumn("is_read", true)

	// 推送带 message_ids 的已读事件，让发送方前端精确刷新已读人数
	s.notifyMessageReadWithMsgIDs(convID, userID, unreadMsgIDs)

	return nil
}

func (s *MessageService) notifyMessageRead(convID, userID uint) {
	s.notifyMessageReadWithMsgIDs(convID, userID, nil)
}

// notifyMessageReadWithMsgIDs 推送已读事件，msgIDs 为本次新写入回执的消息 ID 列表。
// 群聊场景下发送方前端可用 message_ids 精确刷新对应消息的已读人数；
// 单聊场景下 message_ids 可为空，前端按 isRead=true 渲染即可。
func (s *MessageService) notifyMessageReadWithMsgIDs(convID, userID uint, msgIDs []uint) {
	if s.hub == nil {
		return
	}

	db := s.db

	var conv model.Conversation
	if err := db.First(&conv, convID).Error; err != nil {
		return
	}

	data := map[string]interface{}{
		"conversation_id": convID,
		"user_id":         userID,
		"timestamp":       time.Now().Unix(),
	}
	if len(msgIDs) > 0 {
		data["message_ids"] = msgIDs
	}

	readMsg := ws.WSMessage{
		Type: "message_read",
		Data: data,
	}
	jsonMsg, _ := json.Marshal(readMsg)

	if conv.Type == "single" {
		var otherMember model.ConversationMember
		db.Where("conversation_id = ? AND user_id != ?", convID, userID).First(&otherMember)
		s.hub.SendToUser(otherMember.UserID, jsonMsg)
	} else if conv.Type == "group" {
		s.hub.SendToConversation(convID, userID, jsonMsg)
	}
}

func (s *MessageService) GetMessageByID(msgID uint) (*model.Message, error) {
	db := s.db

	var msg model.Message
	if err := db.Preload("Sender").Preload("QuotedMessage").Preload("QuotedMessage.Sender").First(&msg, msgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	return &msg, nil
}

func (s *MessageService) GetMessageQuoteChain(msgID, userID uint) ([]model.Message, error) {
	db := s.db

	var msg model.Message
	if err := db.First(&msg, msgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", msg.ConversationID, userID).First(&member).Error; err != nil {
		return nil, ErrMessageForbidden
	}

	var quoteChain []model.Message
	currentMsg := msg

	for i := 0; i < 3 && currentMsg.QuotedMessageID != nil; i++ {
		var quotedMsg model.Message
		if err := db.Preload("Sender").Preload("QuotedMessage").Preload("QuotedMessage.Sender").First(&quotedMsg, *currentMsg.QuotedMessageID).Error; err == nil {
			quoteChain = append(quoteChain, quotedMsg)
			currentMsg = quotedMsg
		} else {
			break
		}
	}

	return quoteChain, nil
}

func (s *MessageService) GetMessageReadUsers(msgID, userID uint) ([]model.User, int64, error) {
	db := s.db

	var msg model.Message
	if err := db.First(&msg, msgID).Error; err != nil {
		return nil, 0, ErrMessageNotFound
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", msg.ConversationID, userID).First(&member).Error; err != nil {
		return nil, 0, ErrMessageForbidden
	}

	var readReceipts []model.MessageReadReceipt
	if err := db.Where("message_id = ?", msgID).Preload("User").Order("created_at DESC").Find(&readReceipts).Error; err != nil {
		return nil, 0, err
	}

	readUsers := make([]model.User, 0, len(readReceipts))
	for _, receipt := range readReceipts {
		if receipt.User != nil && receipt.User.ID != userID {
			readUsers = append(readUsers, *receipt.User)
		}
	}

	var totalMembers int64
	// 分母排除机器人虚拟用户：已读回执只由真人产生（MarkAsRead 仅真人），
	// 若把 bot 虚拟用户计入总数，会稀释已读比例（bot 永远不可能已读）。
	// 注意：不按 deleted_at 过滤——即使 bot 记录已被软删除，其虚拟用户仍可能残留在
	// conversation_members 里（如已删的"代码助手/AI助手"仍显示在群成员），
	// 它们同样不是真人、不该进分母。凡 user_id 命中某 bot 的 virtual_user_id 一律排除。
	db.Model(&model.ConversationMember{}).
		Where("conversation_id = ?", msg.ConversationID).
		Where("user_id NOT IN (SELECT virtual_user_id FROM bots WHERE virtual_user_id IS NOT NULL)").
		Count(&totalMembers)

	return readUsers, totalMembers, nil
}

func (s *MessageService) BatchGetMessageReadUsers(msgIDs []uint, userID uint) (map[uint]struct {
	ReadUsers    []model.User `json:"read_users"`
	TotalMembers int64        `json:"total_members"`
	ReadCount    int64        `json:"read_count"`
}, error) {
	if len(msgIDs) == 0 {
		return make(map[uint]struct {
			ReadUsers    []model.User `json:"read_users"`
			TotalMembers int64        `json:"total_members"`
			ReadCount    int64        `json:"read_count"`
		}), nil
	}

	db := s.db

	// 优化：一次性查询所有消息
	var messages []model.Message
	db.Where("id IN ?", msgIDs).Find(&messages)

	// 权限校验：请求用户必须是每条消息所在会话的成员，否则跳过该消息（防越权）
	var convIDs []uint
	convIDByMsg := make(map[uint]uint)
	for _, m := range messages {
		convIDByMsg[m.ID] = m.ConversationID
		convIDs = append(convIDs, m.ConversationID)
	}

	// 一次性查询用户在这些会话中的成员身份
	allowedConvs := make(map[uint]bool)
	if len(convIDs) > 0 {
		var memberConvs []uint
		db.Model(&model.ConversationMember{}).
			Where("conversation_id IN ? AND user_id = ?", convIDs, userID).
			Distinct("conversation_id").
			Pluck("conversation_id", &memberConvs)
		for _, cid := range memberConvs {
			allowedConvs[cid] = true
		}
	}

	// 仅保留用户有权访问的消息 ID
	var accessibleMsgIDs []uint
	for _, msgID := range msgIDs {
		if convID, ok := convIDByMsg[msgID]; ok && allowedConvs[convID] {
			accessibleMsgIDs = append(accessibleMsgIDs, msgID)
		}
	}

	// 优化：一次性查询所有（有权访问的）消息的已读回执
	var readReceipts []model.MessageReadReceipt
	if len(accessibleMsgIDs) > 0 {
		db.Where("message_id IN ?", accessibleMsgIDs).Preload("User").Find(&readReceipts)
	}

	// 按消息 ID 分组
	receiptsByMsg := make(map[uint][]model.MessageReadReceipt)
	for _, r := range readReceipts {
		receiptsByMsg[r.MessageID] = append(receiptsByMsg[r.MessageID], r)
	}

	type convCount struct {
		ConversationID uint
		Count          int64
	}
	var convCounts []convCount
	// 分母排除机器人虚拟用户，与分子（已读回执，仅真人）对齐，避免稀释已读比例。
	// 不按 deleted_at 过滤：已软删除 bot 的虚拟用户仍可能残留在成员表，同样非真人、须排除。
	db.Model(&model.ConversationMember{}).
		Select("conversation_id, COUNT(*) as count").
		Where("conversation_id IN ?", convIDs).
		Where("user_id NOT IN (SELECT virtual_user_id FROM bots WHERE virtual_user_id IS NOT NULL)").
		Group("conversation_id").
		Scan(&convCounts)

	memberCountByConv := make(map[uint]int64)
	for _, cc := range convCounts {
		memberCountByConv[cc.ConversationID] = cc.Count
	}

	result := make(map[uint]struct {
		ReadUsers    []model.User `json:"read_users"`
		TotalMembers int64        `json:"total_members"`
		ReadCount    int64        `json:"read_count"`
	}, len(msgIDs))

	for _, msgID := range msgIDs {
		receipts := receiptsByMsg[msgID]
		readUsers := make([]model.User, 0, len(receipts))
		for _, r := range receipts {
			if r.User != nil && r.User.ID != userID {
				readUsers = append(readUsers, *r.User)
			}
		}

		totalMembers := int64(0)
		if convID, ok := convIDByMsg[msgID]; ok {
			totalMembers = memberCountByConv[convID]
		}

		result[msgID] = struct {
			ReadUsers    []model.User `json:"read_users"`
			TotalMembers int64        `json:"total_members"`
			ReadCount    int64        `json:"read_count"`
		}{
			ReadUsers:    readUsers,
			TotalMembers: totalMembers,
			ReadCount:    int64(len(readUsers)),
		}
	}

	return result, nil
}

func (s *MessageService) SearchMessagesByFullText(userID uint, keyword string, convID *uint, limit, offset int) ([]model.Message, error) {
	if keyword == "" {
		return s.SearchMessages(userID, "", convID, limit, offset)
	}

	db := s.db

	if database.D.SupportsFulltext() {
		var messages []model.Message
		query := db.Model(&model.Message{}).
			Joins("JOIN conversation_members ON messages.conversation_id = conversation_members.conversation_id").
			Where("conversation_members.user_id = ?", userID).
			Where("MATCH(messages.content) AGAINST(? IN BOOLEAN MODE)", keyword)

		if convID != nil {
			query = query.Where("messages.conversation_id = ?", *convID)
		}

		if err := query.Preload("Sender").
			Preload("Conversation").
			Preload("Conversation.Members").
			Preload("Conversation.Members.User").
			Order("messages.created_at DESC").
			Limit(limit).
			Offset(offset).
			Find(&messages).Error; err != nil {
			return nil, err
		}
		return messages, nil
	}

	// SQLite：使用 FTS5 虚拟表
	// 其他不支持 FULLTEXT 的数据库：降级到 LIKE 搜索
	if database.D.Type() == "sqlite" {
		if limit <= 0 {
			limit = 20
		}
		if limit > 100 {
			limit = 100
		}

		var messages []model.Message
		query := db.Raw(`
			SELECT m.* FROM messages m
			JOIN messages_fts5 fts ON m.id = fts.rowid
			JOIN conversation_members cm ON m.conversation_id = cm.conversation_id
			WHERE cm.user_id = ? AND fts.content MATCH ?
		`, userID, keyword)

		if convID != nil {
			query = db.Raw(`
				SELECT m.* FROM messages m
				JOIN messages_fts5 fts ON m.id = fts.rowid
				JOIN conversation_members cm ON m.conversation_id = cm.conversation_id
				WHERE cm.user_id = ? AND fts.content MATCH ? AND m.conversation_id = ?
			`, userID, keyword, *convID)
		}

		if err := query.Find(&messages).Error; err != nil {
			return nil, err
		}

		// 预加载关联数据
		if len(messages) > 0 {
			msgIDs := make([]uint, len(messages))
			for i, m := range messages {
				msgIDs[i] = m.ID
			}
			db.Where("id IN ?", msgIDs).
				Preload("Sender").
				Preload("Conversation").
				Preload("Conversation.Members").
				Preload("Conversation.Members.User").
				Find(&messages)
		}

		return messages, nil
	}

	// TiDB / 其他不支持 FULLTEXT 的数据库：降级到 LIKE 搜索
	return s.SearchMessages(userID, keyword, convID, limit, offset)
}

// buildMessageResponse 构建消息响应体。
// HTTP 响应、WS 广播、历史拉取三路共用。
// mentionUserIDs 为已展开（含 @all 展开）的被提及用户 ID 列表。
// is_at_mention 不在此处计算——per-recipient，由调用方按当前用户算。
func (s *MessageService) buildMessageResponse(msg model.Message, mentionUserIDs []uint) map[string]interface{} {
	if mentionUserIDs == nil {
		mentionUserIDs = []uint{}
	}
	isAvatarReply := msg.Origin == "avatar"
	isAIMessage := msg.Origin == "assistant" || msg.Origin == "avatar" ||
		msg.Sender.Type == "bot" || msg.Sender.Type == "system"
	resp := map[string]interface{}{
		"id":                msg.ID,
		"conversation_id":   msg.ConversationID,
		"sender_id":         msg.SenderID,
		"type":              msg.Type,
		"content":           msg.Content,
		"quoted_message_id": msg.QuotedMessageID,
		"is_recalled":       msg.IsRecalled,
		"is_read":           msg.IsRead,
		"is_avatar_reply":   isAvatarReply,
		"is_ai_message":     isAIMessage,
		"origin":            msg.Origin,
		"recalled_at":       msg.RecalledAt,
		"created_at":        msg.CreatedAt,
		"sender":            msg.Sender,
		"quoted_message":    msg.QuotedMessage,
		"mention_user_ids":  mentionUserIDs,
	}
	// 解析 Extra：当前承载 knowledge_sources（Bot 回复命中笔记时的标题/分数）
	// 解析失败或为空时，前端约定 knowledge_sources 不存在或为空数组
	if msg.Extra != "" {
		var extra map[string]interface{}
		if err := json.Unmarshal([]byte(msg.Extra), &extra); err == nil {
			if ks, ok := extra["knowledge_sources"]; ok {
				resp["knowledge_sources"] = ks
			}
			// 外部工具调用记录（tool_calls）：前端据此渲染独立工具卡片（回放）。
			if tc, ok := extra["tool_calls"]; ok {
				resp["tool_calls"] = tc
			}
		}
	}
	return resp
}

// broadcastMessage 广播消息到会话所有成员（排除发送者）。
// mention_user_ids 数组随消息一起广播，前端据此计算 is_at_mention。
// 无需 per-recipient 发送，效率与原方案一致。
func (s *MessageService) broadcastMessage(msg *model.Message, mentionUserIDs []uint, senderID uint) {
	if s.hub == nil {
		return
	}
	payload := s.buildMessageResponse(*msg, mentionUserIDs)
	wsMsg := ws.WSMessage{Type: "new_message", Data: payload}
	jsonMsg, _ := json.Marshal(wsMsg)
	s.hub.SendToConversation(msg.ConversationID, senderID, jsonMsg)
}

func (s *MessageService) CreateMessage(msg *model.Message) error {
	db := s.db
	return db.Create(msg).Error
}

func (s *MessageService) IsAIMessage(senderID uint) bool {
	systemUserID := s.GetSystemUserID()
	return systemUserID > 0 && senderID == systemUserID
}

func (s *MessageService) GetSystemUserID() uint {
	var systemUser model.User
	if err := s.db.Where("type = ?", "system").First(&systemUser).Error; err != nil {
		return 0
	}
	return systemUser.ID
}
