package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
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

// memberWithType 用于一次 JOIN 同时拿到成员校验信息和会话类型，
// 替代原来分两次查询的 member + convType。
type memberWithType struct {
	model.ConversationMember
	ConvType string `gorm:"column:conv_type"`
}

// NoteSearcher 笔记检索接口（用于 bot 回复时按创建者 scope 检索笔记作为知识库）。
// *NoteVectorService 天然实现此接口；测试时可注入 mock。
type NoteSearcher interface {
	SearchNotes(userID uint, query string, topK int) ([]SearchResult, error)
}

type MessageService struct {
	db  *gorm.DB
	hub *ws.Hub

	aiService            *ai.AIService
	noteSearcher         NoteSearcher          // bot 回复时按创建者 scope 检索笔记；nil=降级（不检索）
	groupMemorySvc       *GroupMemoryService   // 群记忆（外部 bot 被 @ 时注入群上下文）；nil=降级（不拼接）
	groupKnowledgeSvc    *GroupDocumentService // 群知识库；nil=降级（不拼接）
	sensitiveWordCache   []model.SensitiveWord
	sensitiveWordCacheMu sync.RWMutex
	sensitiveWordLoaded  bool

	contextAsm     *ContextAssembler  // 上下文注入预制（笔记/记忆/群文档知识源，声明式）；nil=跳过
	botReplyWorker *ReplyOrchestrator // 专属机器人回复的并发控制（限并发+会话串行）；nil=走旧直发

	// streamingSender 流式 AI 消息发送 + 工具调用事件推送能力（handler.WebSocketMessageSender 实现）。
	// 注入后专属机器人 1:1 回复走流式逐 token + 工具调用（与群 @AI 同款基建）；nil=降级到老的
	// 「collect 全文 -> 单条消息广播」非流式路径。
	streamingSender StreamingAISender

	// 文件处理能力：bot 会话收到 file/image 消息时，下载+解析文件内容注入 AI 上下文。
	// storageAccessor 用于从存储后端读取文件，docParser 用于解析文档提取文本。
	// 两者均 nil 时文件消息跳过 bot 回复（降级：不解析、不触发 AI）。
	storageAccessor StorageAccessor
	docParser       *DocumentParser
}

func NewMessageService(db *gorm.DB, hub *ws.Hub, aiService *ai.AIService) *MessageService {
	return &MessageService{
		db:         db,
		hub:        hub,
		aiService:  aiService,
		contextAsm: NewContextAssembler(db),
		botReplyWorker: NewReplyOrchestrator(ReplyOrchestratorOpts{
			Workers:   8,
			Serialize: true, // 专属机器人：按会话串行，防同会话乱序/重复回复
		}),
	}
}

// Close 关闭专属机器人回复编排引擎，回收 worker goroutine。供服务优雅退出时调用。
func (s *MessageService) Close() {
	if s.botReplyWorker != nil {
		s.botReplyWorker.Close()
	}
}

// submitBotReply 把一次专属机器人回复提交到并发控制的异步队列。
// 队列就绪即异步执行（与原 SafeGo 直发语义一致），worker 内按会话串行，
// 保证同会话回复不乱序、不重复；入队超时降级为直接执行，避免消息完全丢失。
func (s *MessageService) submitBotReply(senderID, convID uint, msgType, content string) {
	logger.WithModule("handleBotMessage").Debug("bot 回复提交异步队列",
		"senderID", senderID, "convID", convID, "msgType", msgType)
	handle := func() { s.handleBotMessage(senderID, convID, msgType, content) }
	if s.botReplyWorker != nil {
		if err := s.botReplyWorker.Submit(convID, handle); err == nil {
			return
		} else {
			// 入队超时：降级直发，保底不丢回复
			logger.WithModule("MessageService").Warn("bot 回复入队繁忙，降级直接执行", "convID", convID, "error", err)
		}
	}
	utils.SafeGo(handle)
}

func (s *MessageService) SetAIService(aiService *ai.AIService) {
	s.aiService = aiService
}

// SetNoteSearcher 注入笔记检索服务（用于 bot internal_ai 模式下读取创建者笔记）。
// 传 nil 即可关闭该能力（向量库未配置时安全降级）。
// 同源透传到 contextAsm，保证 bot 笔记注入与统一装配走同一条检索路径。
func (s *MessageService) SetNoteSearcher(searcher NoteSearcher) {
	s.noteSearcher = searcher
	if s.contextAsm != nil {
		s.contextAsm.SetNoteSearcher(searcher)
	}
}

// SetGroupContextServices 注入群记忆 + 群知识库服务，供外部 agent bot 被 @ 时把
// 群积累上下文注入 webhook payload。两者均可传 nil：任一缺失即跳过对应片段，
// 不影响主流程（群未开启向量库/记忆时安全降级）。
func (s *MessageService) SetGroupContextServices(memorySvc *GroupMemoryService, knowledgeSvc *GroupDocumentService) {
	s.groupMemorySvc = memorySvc
	s.groupKnowledgeSvc = knowledgeSvc
	if s.contextAsm != nil {
		// 统一装配的长期记忆/群文档源使用与外部 bot 同源的群知识服务；长期记忆面向 bot 用户。
		s.contextAsm.SetGroupContextServices(nil, knowledgeSvc)
	}
}

// SetStreamingAISender 注入流式 AI 消息发送器，使专属机器人 1:1 回复走流式逐 token +
// 工具调用（复用群 @AI 的 SendStreamingAIMessage / GetCompletionWithToolsStreamMultiStep
// / SendToolCallEvent 基建）。nil=降级到非流式老路径（保底不丢回复能力）。
func (s *MessageService) SetStreamingAISender(sender StreamingAISender) {
	s.streamingSender = sender
}

// SetFileCapabilities 注入文件处理能力（存储访问 + 文档解析），使 bot 会话收到
// file/image 消息时能下载解析文件内容并注入 AI 上下文。两者均可传 nil：
// 缺失时文件消息跳过 bot 回复（降级：不解析、不触发 AI）。
func (s *MessageService) SetFileCapabilities(store StorageAccessor, parser *DocumentParser) {
	s.storageAccessor = store
	s.docParser = parser
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

	// 成员校验 + 会话类型合并为一次 JOIN（原来分两次查询）
	var mwt memberWithType
	if err := db.Table("conversation_members cm").
		Select("cm.*, c.type as conv_type").
		Joins("JOIN conversations c ON c.id = cm.conversation_id").
		Where("cm.conversation_id = ? AND cm.user_id = ?", convID, senderID).
		First(&mwt).Error; err != nil {
		return nil, ErrMessageForbidden
	}
	member := mwt.ConversationMember
	convType := mwt.ConvType

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

	// 写入操作合并为一次事务：Create + UPDATE conversations + unread/mention 计数。
	// @all 成员展开在事务内读取，保证未读 @ 计数与广播名单基于同一数据库快照。
	now := time.Now()
	var mentionUserIDs []uint
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&msg).Error; err != nil {
			return err
		}

		if err := tx.Exec("UPDATE conversations SET last_message_id = ?, last_message_at = ? WHERE id = ?",
			msg.ID, now, convID).Error; err != nil {
			return err
		}

		if convType == "bot" {
			return nil // bot 会话无需更新成员计数
		}

		mentionUserIDs = s.resolveMentionUserIDs(tx, convID, mentions, senderID)

		if err := tx.Model(&model.ConversationMember{}).
			Where("conversation_id = ? AND user_id != ?", convID, senderID).
			UpdateColumn("unread_count", gorm.Expr("unread_count + 1")).Error; err != nil {
			return err
		}

		if len(mentionUserIDs) > 0 {
			if err := tx.Model(&model.ConversationMember{}).
				Where("conversation_id = ? AND user_id IN ?", convID, mentionUserIDs).
				UpdateColumn("unread_at_mention_count", gorm.Expr("unread_at_mention_count + 1")).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	// 按需加载 Sender 和 QuotedMessage（替代原来的 4 次 Preload/First）。
	// 放在事务后：Create 失败时不白跑查询，且 Create 时 msg 不带关联字段，
	// 避免 GORM 对 belongs-to 关联做多余保存。
	db.First(&msg.Sender, msg.SenderID)
	if msg.QuotedMessageID != nil {
		db.First(&msg.QuotedMessage, *msg.QuotedMessageID)
		if msg.QuotedMessage != nil {
			db.First(&msg.QuotedMessage.Sender, msg.QuotedMessage.SenderID)
		}
	}

	// 恢复会话显示：非关键操作，表不存在时不阻塞消息发送（与原行为一致）
	db.Model(&model.ConversationSession{}).
		Where("conversation_id = ? AND is_hidden = ?", convID, true).
		Update("is_hidden", false)

	if convType == "bot" {
		s.submitBotReply(senderID, convID, msgType, content)
	} else {
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
// excludeUserID 用于排除指定用户（如发送者）。db 参数用于事务内/外复用，
// 保证 @all 成员展开与未读 @ 计数处于同一数据库快照。
func (s *MessageService) resolveMentionUserIDs(db *gorm.DB, convID uint, mentions []mention.Mention, excludeUserID uint) []uint {
	var allMembers []model.ConversationMember
	if mention.IsAllMentioned(mentions) {
		if err := db.Where("conversation_id = ?", convID).Find(&allMembers).Error; err != nil {
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
	return s.resolveMentionUserIDs(s.db, convID, mentions, 0)
}

// HandleGroupBotMention 群聊外部 agent 触发：群消息 @ 到某 bot 虚拟用户时，把该消息
// 转发给对应外部 webhook agent（thread_id = 群会话 id）。仅在群会话、且 bot 通过
// BotConversation 关联到本会话时触发——天然克制（不被 @ 就不转发）。
// 内部 AI bot（assistant/system）暂不接群聊，保持既有单聊行为不变。
func (s *MessageService) HandleGroupBotMention(convID, senderID uint, mentionedUserIDs []uint, content string) {
	if len(mentionedUserIDs) == 0 {
		logger.WithModule("handleBotMessage").Debug("群消息未 @ 到任何用户，跳过")
		return
	}
	var convType string
	if err := s.db.Model(&model.Conversation{}).Where("id = ?", convID).Select("type").
		First(&convType).Error; err != nil || convType != "group" {
		logger.WithModule("handleBotMessage").Debug("非群会话或不存在的会话，跳过",
			"convID", convID, "convType", convType, "err", err)
		return // 仅群会话
	}

	// 反查被 @ 的用户里，哪些是本会话已关联 bot 的虚拟用户。
	// BotConversation{conversation_id=群} 在 Phase 3「拉 bot 进群」时建立。
	var bots []model.Bot
	if err := s.db.
		Joins("JOIN bot_conversations bc ON bc.bot_id = bots.id AND bc.conversation_id = ?", convID).
		Where("bots.virtual_user_id IN ?", mentionedUserIDs).
		Find(&bots).Error; err != nil {
		logger.WithModule("handleBotMessage").Error("查询群内被 @ 的 bot 失败", "convID", convID, "error", err)
		return
	}
	if len(bots) == 0 {
		logger.WithModule("handleBotMessage").Debug("被 @ 的用户里没有外部 webhook bot",
			"convID", convID, "mentionedUserIDs", mentionedUserIDs)
		return
	}
	for _, bot := range bots {
		if bot.VirtualUserID == nil || !bot.IsActive {
			logger.WithModule("handleBotMessage").Debug("bot 缺少虚拟用户或未激活，跳过",
				"botID", bot.ID, "virtualUserID", bot.VirtualUserID, "isActive", bot.IsActive)
			continue
		}
		cfg := ParseBotConfig(bot.Config)
		if !cfg.IsExternalWebhook() {
			logger.WithModule("handleBotMessage").Debug("bot 非 external_webhook 模式，跳过",
				"botID", bot.ID, "config", bot.Config)
			continue // 仅外部 webhook agent
		}
		if cfg.WebhookURL == "" {
			// pull 模式外部 bot：无回调地址，不投递也不会自动回复。
			// 向群内回一条系统提示，避免成员 @ 后无反应却不知何故（可感知）。
			// 仅在被 @ 该 bot 时触发，天然克制；系统消息不经 SendMessage，不会递归再触发 OnMessageSent。
			logger.WithModule("handleBotMessage").Info("bot 为 pull 模式（webhook_url 空），发系统提示",
				"botID", bot.ID)
			s.notifyPullModeBot(convID, senderID, bot.Name)
			continue
		}
		logger.WithModule("handleBotMessage").Debug("命中 external webhook bot，准备转发",
			"botID", bot.ID, "convID", convID, "webhookURL", cfg.WebhookURL)
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

func (s *MessageService) handleBotMessage(userID, convID uint, msgType, content string) {
	logger.WithModule("handleBotMessage").Debug("bot 回复开始处理",
		"userID", userID, "convID", convID, "msgType", msgType, "contentLen", len(content))
	db := s.db

	var botConv model.BotConversation
	if err := db.Where("conversation_id = ?", convID).First(&botConv).Error; err != nil {
		logger.WithModule("handleBotMessage").Debug("未找到 bot_conversations 记录，跳过",
			"convID", convID, "userID", userID)
		return
	}

	var bot model.Bot
	if err := db.First(&bot, botConv.BotID).Error; err != nil {
		logger.WithModule("handleBotMessage").Debug("bot 记录不存在或已删除，跳过",
			"botID", botConv.BotID, "convID", convID)
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
		logger.WithModule("handleBotMessage").Debug("命中 external_webhook，准备转发",
			"botID", bot.ID, "convID", convID, "webhookURL", cfg.WebhookURL)
		s.forwardBotMessageToWebhook(bot, cfg.WebhookURL, cfg.WebhookSecret, userID, convID, content, "")
		return
	}

	logger.WithModule("handleBotMessage").Debug("非 external_webhook，走内部 AI",
		"botID", bot.ID, "config", bot.Config)

	// 消息类型分流：文本走流式+工具，文件/图片走解析+注入，其他类型跳过
	switch msgType {
	case "file":
		s.handleBotFileMessage(userID, convID, bot, content)
		return
	case "image":
		s.handleBotImageMessage(userID, convID, bot, content)
		return
	case "text", "markdown":
		// 继续走下面的文本处理路径
	default:
		return // sticker/system 等不触发 bot 回复
	}

	var messages []model.Message
	db.Where("conversation_id = ?", convID).Order("created_at ASC").Limit(20).Find(&messages)

	aiMessages := make([]ai.Message, 0, len(messages)+1)

	botCfg := ParseBotConfig(bot.Config)

	// 创建者笔记作为知识库：仅当开关开启 + 创建者本人在和自己 bot 对话时注入。
	// 隐私：他人和 bot 对话时（userID != bot.CreatorID）不注入创建者笔记，避免泄漏创建者
	// 私有数据；此时 bot 改经 search_knowledge 工具按 talker scope 检索其自己的笔记/知识。
	// 降级：noteSearcher==nil（向量库未配）或检索出错时静默跳过，不影响主流程。
	// 命中笔记写入 knowledgeSources，供消息 Extra 持久化（前端「知识来源」徽章）。
	var knowledgeSources []KnowledgeSource
	if botCfg.UseCreatorNotes && s.noteSearcher != nil && bot.CreatorID > 0 && s.contextAsm != nil && userID == bot.CreatorID {
		bundle := s.contextAsm.Assemble(context.Background(), content, []ContextSource{
			{Type: SourceNotes, Key: bot.CreatorID, TopK: 3},
		})
		if len(bundle.Messages) > 0 {
			aiMessages = append(aiMessages, bundle.Messages...)
		}
		knowledgeSources = bundle.KnowledgeSources
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

	// 流式 + 工具路径（streamingSender 已注入）：复用群 @AI 同款基建，逐 token 流出 +
	// 工具调用卡片 + Extra 持久化。未注入（测试/降级）走 handleBotMessageLegacy 老路径。
	// TODO(v2.1): streamingSender mandatory 注入 + 生产验证稳定后，移除 legacy 分支。
	s.dispatchBotReply(userID, convID, bot, botCfg, aiMessages, knowledgeSources)
}

// dispatchBotReply 统一分发 bot 回复到流式或遗留路径。
// 消除 handleBotMessage / handleBotFileMessage / handleBotImageMessage 中重复的
// if streamingSender != nil { streaming } else { legacy } 分支。
func (s *MessageService) dispatchBotReply(userID, convID uint, bot model.Bot, botCfg BotConfig, aiMessages []ai.Message, knowledgeSources []KnowledgeSource) {
	if s.streamingSender != nil && s.aiService != nil {
		s.handleBotMessageStreaming(userID, convID, bot, botCfg, aiMessages, knowledgeSources)
	} else {
		s.handleBotMessageLegacy(userID, convID, bot, botCfg, aiMessages, knowledgeSources)
	}
}

// buildBotHistory 从 bot 1:1 会话中提取最近 N 条消息，转为 ai.Message 格式。
// bot 虚拟用户的消息标记为 "assistant"，人类消息标记为 "user"。
// 供 handleBotMessage / handleBotFileMessage / handleBotImageMessage 共用。
func (s *MessageService) buildBotHistory(bot model.Bot, convID uint, limit int) []ai.Message {
	var messages []model.Message
	s.db.Where("conversation_id = ?", convID).Order("created_at ASC").Limit(limit).Find(&messages)
	aiMessages := make([]ai.Message, 0, len(messages))
	for _, msg := range messages {
		role := "user"
		if bot.VirtualUserID != nil && msg.SenderID == *bot.VirtualUserID {
			role = "assistant"
		}
		aiMessages = append(aiMessages, ai.Message{Role: role, Content: msg.Content})
	}
	return aiMessages
}

// botFileContentJSON 从文件消息 content JSON 中解析出 {url, id, name, size}。
// url 即 StoragePath，id 即 model.File.ID。
type botFileContentJSON struct {
	URL  string `json:"url"`
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// handleBotFileMessage 处理 bot 1:1 会话中的文件消息（type="file"）。
// 下载文件到临时文件 → DocumentParser 解析纯文本 → 注入 aiMessages 作为上下文 →
// 走流式+工具回复路径。不支持的格式回复友好提示；文件过大/读取失败等降级为错误回复。
func (s *MessageService) handleBotFileMessage(userID, convID uint, bot model.Bot, content string) {
	log := logger.WithModule("handleBotMessage")

	// 依赖缺失：存储或解析器未注入 → 跳过
	if s.storageAccessor == nil || s.docParser == nil {
		log.Warn("文件处理能力未注入（storageAccessor/docParser），跳过", "convID", convID)
		s.sendBotTextReply(userID, convID, bot, "📎 文件处理能力暂不可用，请稍后再试。")
		return
	}

	var fc botFileContentJSON
	if err := json.Unmarshal([]byte(content), &fc); err != nil {
		log.Warn("文件消息 content 解析失败", "convID", convID, "error", err)
		s.sendBotTextReply(userID, convID, bot, "📎 文件消息格式异常，请重新发送。")
		return
	}
	if fc.ID == 0 {
		log.Warn("文件消息缺少有效 ID", "convID", convID)
		s.sendBotTextReply(userID, convID, bot, "📎 文件消息格式异常，请重新发送。")
		return
	}

	// 查文件记录
	var file model.File
	if err := s.db.First(&file, fc.ID).Error; err != nil {
		log.Warn("文件记录不存在", "fileID", fc.ID, "error", err)
		s.sendBotTextReply(userID, convID, bot, "📎 文件不存在或已被删除。")
		return
	}

	// 大小门控：与群 AI 引用文件一致（20MB）
	const maxFileSize int64 = 20 * 1024 * 1024
	if file.Size > maxFileSize {
		s.sendBotTextReply(userID, convID, bot,
			fmt.Sprintf("📎 文件「%s」过大（%.1fMB），当前支持的最大文件为 20MB。请压缩后重新发送。",
				file.Name, float64(file.Size)/(1024*1024)))
		return
	}

	// 下载到临时文件
	reader, err := s.storageAccessor.GetByPath(context.Background(), file.StoragePath)
	if err != nil {
		log.Error("读取文件失败", "fileID", file.ID, "storagePath", file.StoragePath, "error", err)
		s.sendBotTextReply(userID, convID, bot, "📎 读取文件失败，请稍后重试。")
		return
	}
	tmpFile, err := os.CreateTemp("", "qim-bot-*"+filepath.Ext(file.Name))
	if err != nil {
		reader.Close()
		log.Error("创建临时文件失败", "error", err)
		s.sendBotTextReply(userID, convID, bot, "📎 读取文件失败，请稍后重试。")
		return
	}
	tmpPath := tmpFile.Name()
	if _, err := io.Copy(tmpFile, reader); err != nil {
		reader.Close()
		tmpFile.Close()
		os.Remove(tmpPath)
		log.Error("写入临时文件失败", "error", err)
		s.sendBotTextReply(userID, convID, bot, "📎 读取文件失败，请稍后重试。")
		return
	}
	reader.Close()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// 解析文件内容
	parsedText, err := s.docParser.Parse(tmpPath)
	if err != nil {
		log.Warn("文件解析失败", "fileID", file.ID, "name", file.Name, "error", err)
		// 用 describeParseError 透传底层语义（可能是扫描件/无文字层、损坏、格式不支持等），
		// 而不笼统报"暂不支持该格式"——避免扫描版 PDF 被误解为格式不受支持。
		s.sendBotTextReply(userID, convID, bot, "📎 文件解析失败："+describeParseError(err))
		return
	}

	if strings.TrimSpace(parsedText) == "" {
		s.sendBotTextReply(userID, convID, bot, "📎 文件内容为空，无法处理。")
		return
	}

	// 截断过长内容，避免撑爆 token 上限
	const maxFileContextLen = 30000 // ~30KB 文本
	if len(parsedText) > maxFileContextLen {
		parsedText = parsedText[:maxFileContextLen] + "\n\n...（内容过长，已截断）"
	}

	// 构造 aiMessages：历史 + 文件内容作为用户消息
	aiMessages := s.buildBotHistory(bot, convID, 20)
	aiMessages = append(aiMessages, ai.Message{
		Role:    "user",
		Content: fmt.Sprintf("用户发送了文件「%s」，内容如下：\n\n%s", file.Name, parsedText),
	})

	// 走流式+工具回复（复用 handleBotMessage 的分流逻辑）
	botCfg := ParseBotConfig(bot.Config)
	s.dispatchBotReply(userID, convID, bot, botCfg, aiMessages, nil)
}

// handleBotImageMessage 处理 bot 1:1 会话中的图片消息（type="image"）。
// 读取图片字节转 base64 data URL → 设 ai.Message.ImageURL（多模态）→ 走流式+工具回复。
// Provider 不支持多模态时 AI 只看到文字提示"请识别其内容"然后诚实回复看不了——合理降级。
func (s *MessageService) handleBotImageMessage(userID, convID uint, bot model.Bot, content string) {
	log := logger.WithModule("handleBotMessage")

	if s.storageAccessor == nil {
		log.Warn("存储未注入，跳过图片处理", "convID", convID)
		s.sendBotTextReply(userID, convID, bot, "📎 图片处理能力暂不可用，请稍后再试。")
		return
	}

	var fc botFileContentJSON
	if err := json.Unmarshal([]byte(content), &fc); err != nil {
		log.Warn("图片消息 content 解析失败", "convID", convID, "error", err)
		s.sendBotTextReply(userID, convID, bot, "📎 图片消息格式异常，请重新发送。")
		return
	}
	if fc.ID == 0 {
		log.Warn("图片消息缺少有效 ID", "convID", convID)
		s.sendBotTextReply(userID, convID, bot, "📎 图片消息格式异常，请重新发送。")
		return
	}

	var file model.File
	if err := s.db.First(&file, fc.ID).Error; err != nil {
		log.Warn("图片文件记录不存在", "fileID", fc.ID, "error", err)
		s.sendBotTextReply(userID, convID, bot, "📎 图片文件不存在或已被删除。")
		return
	}

	// 大小门控：与群 AI 引用图片一致（5MB）
	const maxImageSize int64 = 5 * 1024 * 1024
	if file.Size > maxImageSize {
		s.sendBotTextReply(userID, convID, bot,
			fmt.Sprintf("🖼️ 图片「%s」过大（%.1fMB），当前支持的最大图片为 5MB。请压缩后重新发送。",
				file.Name, float64(file.Size)/(1024*1024)))
		return
	}

	// 读取图片字节
	reader, err := s.storageAccessor.GetByPath(context.Background(), file.StoragePath)
	if err != nil {
		log.Error("读取图片失败", "fileID", file.ID, "error", err)
		s.sendBotTextReply(userID, convID, bot, "📎 读取图片失败，请稍后重试。")
		return
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		log.Error("读取图片字节失败", "error", err)
		s.sendBotTextReply(userID, convID, bot, "📎 读取图片失败，请稍后重试。")
		return
	}
	if len(data) > int(maxImageSize) {
		log.Warn("图片实际大小超过限制", "fileID", file.ID, "actualSize", len(data))
		s.sendBotTextReply(userID, convID, bot,
			fmt.Sprintf("🖼️ 图片实际大小超出限制（%.1fMB），请压缩后重新发送。", float64(len(data))/(1024*1024)))
		return
	}

	// 构造 base64 data URL
	contentType := mime.TypeByExtension(filepath.Ext(file.Name))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	dataURL := "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)

	// 构造 aiMessages：历史 + 图片（多模态）
	aiMessages := s.buildBotHistory(bot, convID, 20)
	aiMessages = append(aiMessages, ai.Message{
		Role:     "user",
		Content:  fmt.Sprintf("用户发送了一张图片「%s」，请识别其内容。", file.Name),
		ImageURL: dataURL,
	})

	botCfg := ParseBotConfig(bot.Config)
	s.dispatchBotReply(userID, convID, bot, botCfg, aiMessages, nil)
}

// sendBotTextReply 直接发送一条 bot 纯文本消息（不经过 AI 管道）。
// 用于文件/图片处理降级时的用户反馈——错误提示必须原样送达，不得被 AI 重写。
// 复用 handleBotMessageLegacy 的落库 + 广播逻辑，跳过 AI completion。
func (s *MessageService) sendBotTextReply(userID, convID uint, bot model.Bot, reply string) {
	log := logger.WithModule("sendBotTextReply")
	db := s.db

	botReply := model.Message{
		ConversationID: convID,
		SenderID:       *bot.VirtualUserID,
		Type:           "markdown",
		Content:        reply,
		Origin:         "assistant",
		IsRead:         true,
	}
	if err := db.Create(&botReply).Error; err != nil {
		log.Error("创建 bot 兜底消息失败", "error", err, "convID", convID)
		return
	}

	// 预加载 Sender 供前端渲染头像
	if err := db.Preload("Sender").First(&botReply, botReply.ID).Error; err != nil {
		log.Error("预加载 bot 兜底消息发送者失败", "error", err, "msgID", botReply.ID)
	}

	now := time.Now()
	if err := db.Model(&model.Conversation{}).
		Where("id = ?", convID).
		Updates(map[string]interface{}{
			"last_message_id": botReply.ID,
			"last_message_at": now,
		}).Error; err != nil {
		log.Error("更新会话最后消息失败", "error", err, "convID", convID)
		return
	}

	if s.hub != nil {
		wsMsg := ws.WSMessage{
			Type: "new_message",
			Data: BuildMessageResponse(botReply, MessageResponseOptions{BroadcastWS: true}),
		}
		jsonMsg, _ := json.Marshal(wsMsg)
		s.hub.SendToUser(userID, jsonMsg)
	}

	log.Info("bot 兜底消息已发送", "convID", convID, "content", reply)
}

// botAllowedTools 专属机器人 1:1 会话可调用的工具白名单。
// 按 talker scope（CallerContext.UserID = 和 bot 对话的用户），不暴露创建者私有数据：
// list_tasks / create_user_task / search_knowledge 均按 ctx.UserID 检索，别人和 bot 对话
// 只能读到他自己的任务/知识。不含 send_message（防 bot 代用户向其他会话发消息，滥用风险）；
// 不含 summarize_conversation（群场景导向，1:1 价值有限）。
var botAllowedTools = []string{
	"list_tasks",
	"create_user_task",
	"search_knowledge",
}

// handleBotMessageStreaming 专属机器人 1:1 流式 + 工具调用回复。
//
// 复用群 @AI 的 SendStreamingAIMessage（懒创建流式消息）+ GetCompletionWithToolsStreamMultiStep
// （带工具的流式 ReAct）+ SendToolCallEvent（工具卡片实时反馈）基建，使 bot 回复具备逐 token
// 打字感与工具调用能力。工具按 talker scope（CallerContext.UserID = 和 bot 对话的用户），
// 不暴露创建者私有数据。
//
// 模型路径：
//   - 系统默认：带工具流式 ReAct；Provider 不支持流式 tool-call 时降级纯流式（无工具）。
//   - 自定义模型（创建者自选 provider）：纯流式、不带工具（不能假设其 provider 支持 tool-call）；
//     生成失败且未流出内容时回退系统默认（带工具），兑现「不阻断回复」契约。
func (s *MessageService) handleBotMessageStreaming(userID, convID uint, bot model.Bot, botCfg BotConfig, aiMessages []ai.Message, knowledgeSources []KnowledgeSource) {
	sendChunk, getMsg, finish, err := s.streamingSender.SendStreamingAIMessage(convID, bot.Name)
	if err != nil {
		logger.WithModule("handleBotMessage").Error("创建流式消息失败，降级到非流式", "error", err, "convID", convID)
		s.handleBotMessageLegacy(userID, convID, bot, botCfg, aiMessages, knowledgeSources)
		return
	}

	// 工具调用进度回调：start 推 running 卡片，end 推终态 + 收集记录供 Extra 持久化
	var toolCalls []ToolCallRecord
	feedback := NewToolCallFeedback(s.streamingSender, convID, getMsg, &toolCalls, nil, nil)

	// 隐私关键：callerCtx 用 talker(userID)，工具按 talker scope 检索任务/知识，不读创建者私有数据
	callerCtx := &ai.CallerContext{UserID: userID}

	// contentProduced 跟踪是否已流出正文：用于自定义模型失败时判断是否回退（已流出则保留部分内容，
	// 流式中途无法干净衔接重试）。与老路径 builder.Len()==0 判断等价。
	var contentProduced bool
	onChunk := func(chunk ai.StreamChunk) error {
		if chunk.Content != "" {
			contentProduced = true
		}
		return sendChunk(chunk.Content)
	}

	// 自定义模型来源：bot 配置了「使用我的自定义配置」（!UseSystemConfig 且 UserConfigID 非空）时，
	// 解析出创建者自选的 provider。解析失败（配置被删/禁用/密钥解密失败）-> 回退系统默认。
	var custom *customProvider
	if !botCfg.UseSystemConfig && botCfg.UserConfigID != nil {
		custom = resolveUserAIConfigProvider(s.db, bot.CreatorID, *botCfg.UserConfigID)
		if custom == nil || custom.ProviderName == "" {
			logger.WithModule("handleBotMessage").Warn("bot 自定义模型解析失败，回退系统默认",
				"botID", bot.ID, "userConfigID", *botCfg.UserConfigID)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var streamErr error
	if custom != nil && custom.ProviderName != "" {
		// 自定义模型：纯流式、不带工具（不能假设创建者自选 provider 支持 tool-call）
		streamErr = s.aiService.ChatStreamWithProviderConfig(ctx, ai.TaskTypeChat, aiMessages, custom.ProviderName, custom.Config, onChunk)
		// 生成期失败（密钥失效/配额耗尽/网络错/模型名错误）且尚未流出内容时，回退系统默认（带工具），
		// 兑现「回退系统默认…不阻断回复」契约——单条用户自定义配置出问题不应拖垮整个 bot 回复。
		// 已流出部分内容则保留（流式中途无法干净衔接重试）。
		if streamErr != nil && !contentProduced {
			logger.WithModule("handleBotMessage").Warn("bot 自定义模型生成失败，回退系统默认",
				"botID", bot.ID, "provider", custom.ProviderName, "error", streamErr)
			streamErr = s.aiService.GetCompletionWithToolsStreamMultiStep(ctx, ai.TaskTypeChat, aiMessages, callerCtx, botAllowedTools, ai.MaxReActSteps, feedback, onChunk)
			if errors.Is(streamErr, ai.ErrStreamingToolsNotSupported) {
				streamErr = s.aiService.GetCompletionStreamWithContext(ctx, ai.TaskTypeChat, aiMessages, onChunk)
			}
		}
	} else {
		// 系统默认：带工具的流式 ReAct
		streamErr = s.aiService.GetCompletionWithToolsStreamMultiStep(ctx, ai.TaskTypeChat, aiMessages, callerCtx, botAllowedTools, ai.MaxReActSteps, feedback, onChunk)
		if errors.Is(streamErr, ai.ErrStreamingToolsNotSupported) {
			// Provider 不支持流式 tool-call -> 降级纯流式（无工具）
			logger.WithModule("handleBotMessage").Info("bot 流式工具不可用，降级纯流式", "convID", convID)
			streamErr = s.aiService.GetCompletionStreamWithContext(ctx, ai.TaskTypeChat, aiMessages, onChunk)
		}
	}

	if streamErr != nil {
		logger.WithModule("handleBotMessage").Error("bot 流式回复出错", "error", streamErr, "convID", convID)
		// 完全无产出（无正文且无工具调用）-> 发兜底文案，保证用户得到反馈，避免残留空气泡。
		// 已有正文/工具调用则保留（finish 收尾），不覆盖。
		if !contentProduced && len(toolCalls) == 0 {
			_ = sendChunk("抱歉，AI 服务暂时不可用，请稍后再试。")
		}
	}

	// 工具调用记录 + 命中的知识来源合并持久化到消息 Extra，回放/刷新后卡片与徽章仍可见
	PersistAIMessageExtra(getMsg, toolCalls, knowledgeSources)

	if finish() == nil {
		logger.WithModule("handleBotMessage").Warn("bot 流式回复无内容产出", "convID", convID)
	} else {
		logger.WithModule("handleBotMessage").Info("bot 流式回复完成", "convID", convID, "toolCalls", len(toolCalls))
	}
}

// handleBotMessageLegacy 专属机器人 1:1 非流式回复（降级路径）。
//
// 当 streamingSender 未注入（测试场景 / 装配缺失）或流式消息创建失败时使用：collect 全文到
// builder -> 一次性建消息 -> hub.SendToUser 广播。无流式打字感、无工具调用，但保底不丢回复。
// 保留原 60s 超时 + 自定义模型回退系统默认的契约。
//
// TODO(v2.1): 当 streamingSender mandatory 注入且生产验证稳定后移除此路径。
func (s *MessageService) handleBotMessageLegacy(userID, convID uint, bot model.Bot, botCfg BotConfig, aiMessages []ai.Message, knowledgeSources []KnowledgeSource) {
	db := s.db
	var builder strings.Builder
	var streamErr error
	if s.aiService == nil {
		streamErr = fmt.Errorf("AI 服务未配置")
	} else {
		// 自定义模型来源：解析失败 -> 回退系统默认（与流式路径一致）。
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
				// 生成期失败且尚未流出内容时，回退系统默认，兑现「不阻断回复」契约。
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
	// 「已流出部分内容则保留」契约。仅在完全没流出任何内容（含回退系统默认也失败）时用兜底文案。
	if response == "" && streamErr != nil {
		logger.WithModule("handleBotMessage").Error("AI API error", "error", streamErr)
		response = "抱歉，AI 服务暂时不可用，请稍后再试。"
	}

	botReply := model.Message{
		ConversationID: convID,
		SenderID:       *bot.VirtualUserID,
		Type:           "markdown",
		Content:        response,
		Origin:         "assistant",
		IsRead:         true, // Bot 回复默认已读
	}
	// 命中笔记时把 knowledge_sources 写入 Extra，供前端折叠展示
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
			Data: BuildMessageResponse(botReply, MessageResponseOptions{BroadcastWS: true}),
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

// MessageResponseOptions 控制消息响应载荷构建的 per-channel / per-user 差异。
// 唯一构建入口：HTTP 响应、WS 广播、AI 消息广播全部经由 BuildMessageResponse，
// 避免多处手搓载荷导致字段漂移（sender_type/is_streaming/extra 等曾只进 HTTP 版）。
type MessageResponseOptions struct {
	// MentionUserIDs 已展开的 @ 名单（含 @all），WS 广播场景由发送路径传入；
	// nil 时若提供 AllMemberIDs 则按 content 重新展开。
	MentionUserIDs []uint
	// AllMemberIDs 会话全体成员（@all 展开 + is_at_mention 计算用），HTTP 场景传入。
	AllMemberIDs []uint
	// UserReadSet per-user 已读回执集（HTTP 历史拉取）；nil 时回退 msg.IsRead。
	UserReadSet map[uint]bool
	// CurrentUserID 当前请求用户（HTTP）；>0 时计算 is_at_mention，并按回执判定已读。
	CurrentUserID uint
	// BroadcastWS 标记为 WS 广播：无 per-user 上下文，is_read 固定 false
	// （避免 A 已读后 B 收到的广播也显示已读），不输出 is_at_mention（前端按 mention_user_ids 自己算）。
	BroadcastWS bool
}

// BuildMessageResponse 构建消息响应体，HTTP 响应、WS 广播、AI 消息广播三路共用。
// 字段集含：id/conversation_id/sender_id/sender_type/type/content/quoted_message_id/
// is_recalled/is_read/is_avatar_reply/is_ai_message/is_streaming/ai_assistant_name/
// origin/recalled_at/created_at/sender/quoted_message/mention_user_ids/extra，
// 以及按需的 is_at_mention（HTTP）、avatar_name（分身）和 Extra 解析出的
// tool_calls/knowledge_sources/sources。
func BuildMessageResponse(msg model.Message, opts MessageResponseOptions) map[string]interface{} {
	// @ 名单：广播场景直接使用传入的已展开名单；HTTP 场景从内容展开
	mentionUserIDs := opts.MentionUserIDs
	if mentionUserIDs == nil {
		mentionUserIDs = []uint{}
		if opts.AllMemberIDs != nil {
			mentions := mention.Parse(msg.Content)
			mentionUserIDs = mention.ExtractUserIDs(mentions, opts.AllMemberIDs, msg.SenderID)
		}
	}

	// per-user / per-channel 已读
	isRead := msg.IsRead
	if opts.BroadcastWS {
		isRead = false
	} else if opts.UserReadSet != nil {
		isRead = opts.CurrentUserID > 0 && (msg.SenderID == opts.CurrentUserID || opts.UserReadSet[msg.ID])
	}

	senderType := msg.Sender.Type
	if msg.Origin == "assistant" || msg.Origin == "avatar" {
		senderType = "bot"
	}
	isAvatarReply := msg.Origin == "avatar"
	isAIMessage := msg.Origin == "assistant" || msg.Origin == "avatar" ||
		msg.Sender.Type == "bot" || msg.Sender.Type == "system"

	resp := map[string]interface{}{
		"id":                msg.ID,
		"conversation_id":   msg.ConversationID,
		"sender_id":         msg.SenderID,
		"sender_type":       senderType,
		"type":              msg.Type,
		"content":           msg.Content,
		"quoted_message_id": msg.QuotedMessageID,
		"is_recalled":       msg.IsRecalled,
		"is_read":           isRead,
		"is_avatar_reply":   isAvatarReply,
		"is_ai_message":     isAIMessage,
		// 流式消息标记：type=streaming 即进行中，客户端据此显示 typing 动画；
		// 已 finish 的消息 type 已是 markdown，is_streaming=false。刷新后不卡空气泡。
		"is_streaming":      msg.Type == "streaming",
		"ai_assistant_name": resolveAIName(msg),
		"origin":            msg.Origin,
		"recalled_at":       msg.RecalledAt,
		"created_at":        msg.CreatedAt,
		"sender":            msg.Sender,
		"quoted_message":    msg.QuotedMessage,
		"mention_user_ids":  mentionUserIDs,
		// 透出 Extra（JSON 字符串）。撤回消息时 RecallMessage 将 original_content
		// 写入 Extra，「撤回后重新编辑」依赖它回填输入框；通道缺失会导致切窗口后无法回填。
		"extra": msg.Extra,
	}

	// is_at_mention 仅 HTTP 场景计算（per-recipient）；WS 广播前端按 mention_user_ids 自己算
	if !opts.BroadcastWS && opts.CurrentUserID > 0 && opts.AllMemberIDs != nil {
		resp["is_at_mention"] = msg.SenderID != opts.CurrentUserID && containsUint(mentionUserIDs, opts.CurrentUserID)
	}

	// 分身消息：透出分身名称
	if msg.Origin == "avatar" {
		resp["avatar_name"] = GetAINameCache().GetAvatarName(msg.SenderID)
	}

	// 解析 Extra：透出 tool_calls / knowledge_sources / sources，供前端渲染工具卡片
	// 与「知识来源/依据」徽章。
	for k, v := range ParseMessageExtraFields(msg.Extra) {
		resp[k] = v
	}

	return resp
}

// resolveAIName 解析 AI 消息的展示名：分身名 / 群 AI 配置名 / bot 昵称，兜底 "AI助手"。
func resolveAIName(msg model.Message) string {
	nameCache := GetAINameCache()
	db := database.GetDB()

	if msg.Origin == "assistant" {
		var group model.Group
		if err := db.Select("ai_config").
			Where("conversation_id = ?", msg.ConversationID).
			First(&group).Error; err == nil && group.AIConfigJSON != "" {
			aiConfig := group.GetAIConfig()
			if aiConfig.AssistantName != "" {
				return aiConfig.AssistantName
			}
		}
	}
	if msg.Origin == "avatar" {
		if name := nameCache.GetAvatarName(msg.SenderID); name != "" {
			return name
		}
	}
	if msg.Sender.Type == "bot" || msg.Sender.Type == "system" {
		return msg.Sender.Nickname
	}
	return "AI助手"
}

// containsUint 判断 uint 切片是否包含 v。
func containsUint(s []uint, v uint) bool {
	for _, item := range s {
		if item == v {
			return true
		}
	}
	return false
}

// ParseMessageExtraFields 解析消息 Extra（JSON 字符串），提取 tool_calls、knowledge_sources、
// sources 三个顶层字段，返回供消息响应透出的 map。解析失败或为空时返回 nil。
func ParseMessageExtraFields(extra string) map[string]interface{} {
	if extra == "" {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(extra), &m); err != nil {
		return nil
	}
	result := make(map[string]interface{})
	for _, key := range []string{"tool_calls", "knowledge_sources", "sources"} {
		if v, ok := m[key]; ok {
			result[key] = v
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// broadcastMessage 广播消息到会话所有成员（排除发送者）。
// mention_user_ids 数组随消息一起广播，前端据此计算 is_at_mention。
// 无需 per-recipient 发送，效率与原方案一致。
func (s *MessageService) broadcastMessage(msg *model.Message, mentionUserIDs []uint, senderID uint) {
	if s.hub == nil {
		return
	}
	payload := BuildMessageResponse(*msg, MessageResponseOptions{
		MentionUserIDs: mentionUserIDs,
		BroadcastWS:    true,
	})
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
