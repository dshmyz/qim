package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/cache"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/dshmyz/qim/qim-server/utils"
	"github.com/dshmyz/qim/qim-server/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// getUserIDFromContext 从 gin.Context 安全提取 user_id，避免类型断言 panic。
// 调用方应检查 ok 返回值，失败时通常返回 401。
func getUserIDFromContext(c *gin.Context) (uint, bool) {
	v, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	uid, ok := v.(uint)
	return uid, ok
}

// Global smart reply engine instance
var smartReplyEngine *SmartReplyEngine

// mcpGateway 外部 MCP 客户端网关的进程内引用。供 UpdateSystemConfig 在外部 MCP
// 配置变更后触发运行时热同步（无需重启）。与 smartReplyEngine 的注入分离：这里
// 只负责持引用以响应配置更新，工具可及性的门控走 smartReplyEngine 网关。
var mcpGateway *service.MCPClientGateway

type reminderLimiterState struct {
	pending bool
	lastOK  time.Time
}

type reminderLimiterReason int

const (
	reminderAllowed reminderLimiterReason = iota
	reminderPending
	reminderCoolingDown
)

type reminderLimiter struct {
	mu       sync.Mutex
	entries  map[uint]reminderLimiterState
	cooldown time.Duration // 同消息重复提醒冷却时长；由 SetCooldown 按系统配置更新
}

func buildRemindResult(messageID uint, success bool, reminderError, systemName string) ([]byte, error) {
	data := map[string]interface{}{
		"message_id": messageID,
		"success":    success,
		"timestamp":  time.Now().Format(time.RFC3339),
	}
	if success {
		data["system_name"] = systemName
	} else {
		data["error"] = reminderError
	}

	return json.Marshal(map[string]interface{}{
		"type": "remind_result",
		"data": data,
	})
}

func newReminderLimiter() *reminderLimiter {
	return &reminderLimiter{entries: make(map[uint]reminderLimiterState), cooldown: time.Hour}
}

// SetCooldown 更新重复提醒冷却时长（来自系统配置，0=不限制，可反复提醒）。
func (l *reminderLimiter) SetCooldown(d time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cooldown = d
}

// cooldownLocked 返回当前冷却时长。0 表示不限制（超出冷却窗口则永远放行）。
func (l *reminderLimiter) cooldownLocked() time.Duration {
	return l.cooldown
}

func (l *reminderLimiter) cleanupExpiredLocked(now time.Time) {
	cd := l.cooldownLocked()
	if cd <= 0 {
		return // 不限制冷却时无需清理
	}
	for id, entry := range l.entries {
		if !entry.pending && !entry.lastOK.IsZero() && now.Sub(entry.lastOK) >= cd {
			delete(l.entries, id)
		}
	}
}

func (l *reminderLimiter) checkLocked(messageID uint, now time.Time) reminderLimiterReason {
	entry := l.entries[messageID]
	if entry.pending {
		return reminderPending
	}
	cd := l.cooldownLocked()
	if cd > 0 && !entry.lastOK.IsZero() && now.Sub(entry.lastOK) < cd {
		return reminderCoolingDown
	}
	return reminderAllowed
}

func (l *reminderLimiter) check(messageID uint, now time.Time) reminderLimiterReason {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanupExpiredLocked(now)
	return l.checkLocked(messageID, now)
}

func (l *reminderLimiter) start(messageID uint, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanupExpiredLocked(now)
	if l.checkLocked(messageID, now) != reminderAllowed {
		return false
	}

	entry := l.entries[messageID]
	entry.pending = true
	l.entries[messageID] = entry
	return true
}

func (l *reminderLimiter) finish(messageID uint, success bool, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := l.entries[messageID]
	entry.pending = false
	if success {
		entry.lastOK = now
	} else if entry.lastOK.IsZero() {
		delete(l.entries, messageID)
		return
	}
	l.entries[messageID] = entry
}

func reminderLimiterReasonMessage(reason reminderLimiterReason, cooldown time.Duration) string {
	if reason == reminderPending {
		return "提醒发送中，请稍候"
	}
	if cooldown <= 0 {
		return "该消息已提醒过，请稍后再试"
	}
	minutes := int(cooldown.Minutes())
	if minutes >= 1 {
		return fmt.Sprintf("该消息已提醒过，请 %d 分钟后再试", minutes)
	}
	return fmt.Sprintf("该消息已提醒过，请 %d 秒后再试", int(cooldown.Seconds()))
}

func finishReminderAttempt(limiter *reminderLimiter, messageID uint, success *bool) {
	limiter.finish(messageID, *success, time.Now())
}

// remindRateLimiter 消息提醒频率限制器；重启清零，对提醒场景可接受。
var remindRateLimiter = newReminderLimiter()

// TodoExtractorInterface 待办提取接口，便于测试替换
type TodoExtractorInterface interface {
	ExtractAndCreateTodos(content string, senderID uint, conversationID uint)
}

// Global todo extractor instance
var todoExtractor TodoExtractorInterface

// InitSmartReplyEngine initializes the smart reply engine with the given AI service
func InitSmartReplyEngine(aiService *ai.AIService) {
	detector := ai.NewIntentDetector(aiService)
	smartReplyEngine = NewSmartReplyEngine(aiService, detector)
	todoExtractor = NewTodoExtractor(aiService)
}

// SetAvatarWorkerPool sets the avatar worker pool for the smart reply engine
func SetAvatarWorkerPool(pool *service.AvatarWorkerPool) {
	if smartReplyEngine != nil {
		smartReplyEngine.SetAvatarWorkerPool(pool)
	}
}

// SetUnifiedKnowledge sets the unified knowledge service for the smart reply engine
func SetUnifiedKnowledge(uk *service.UnifiedKnowledgeService) {
	if smartReplyEngine != nil {
		smartReplyEngine.SetUnifiedKnowledge(uk)
	}
}

// SetMemoryService sets the avatar memory service for the smart reply engine
func SetMemoryService(ms *service.AvatarMemoryService) {
	if smartReplyEngine != nil {
		smartReplyEngine.SetMemoryService(ms)
	}
}

// SetGroupMemoryService sets the group-level memory service for the smart reply engine
func SetGroupMemoryService(gms *service.GroupMemoryService) {
	if smartReplyEngine != nil {
		smartReplyEngine.SetGroupMemoryService(gms)
	}
}

// SetMCPGateway 注入外部 MCP 客户端网关到 smart reply 引擎，使群 @AI 白名单
// 能按位点放行外部工具；同时保留进程内引用供配置更新后热同步。smartReplyEngine
// 可能未初始化时安全跳过。
func SetMCPGateway(gw *service.MCPClientGateway) {
	mcpGateway = gw
	if smartReplyEngine != nil {
		smartReplyEngine.SetMCPGateway(gw)
	}
}

// ReSyncExternalMCP 在外部 MCP 配置（external_mcp / external_mcp:group_enabled）
// 被后台更新后触发网关热同步，使新增/修改/删除的连接立即生效而不必重启服务。
// 网关未注入时安全跳过（幂等）。
func ReSyncExternalMCP() {
	if mcpGateway == nil {
		return
	}
	mcpGateway.Sync()
}

// InitSmartReplyGraph initializes the Eino Graph for smart reply
func InitSmartReplyGraph() error {
	if smartReplyEngine == nil {
		return fmt.Errorf("SmartReplyEngine not initialized")
	}

	logger.WithModule("SmartReplyGraph").Info("开始初始化...")
	logger.WithModule("SmartReplyGraph").Info("aiService", "initialized", smartReplyEngine.aiService != nil)
	logger.WithModule("SmartReplyGraph").Info("unifiedKnowledge", "initialized", smartReplyEngine.unifiedKnowledge != nil)
	logger.WithModule("SmartReplyGraph").Info("memorySvc", "initialized", smartReplyEngine.memorySvc != nil)

	err := smartReplyEngine.InitSmartReplyGraph()
	if err != nil {
		logger.WithModule("SmartReplyGraph").Error("初始化失败", "error", err)
		return err
	}

	logger.WithModule("SmartReplyGraph").Info("初始化成功")
	return nil
}

// GetSmartReplyEngine returns the smart reply engine instance
func GetSmartReplyEngine() *SmartReplyEngine {
	return smartReplyEngine
}

// TryExtractTodos 独立的待办提取入口。
// 仅依赖群聊配置中的 ExtractTodos 开关，不受智能回复 Enabled/ReplyMode 等条件影响。
func TryExtractTodos(senderID uint, conversationID uint, content string) {
	if todoExtractor == nil {
		return
	}

	db := database.GetDB()

	var conv model.Conversation
	if err := db.First(&conv, conversationID).Error; err != nil {
		return
	}

	TryExtractTodosWithPreloaded(senderID, &conv, conversationID, content)
}

// TryExtractTodosWithPreloaded 使用预加载的 conv 执行待办提取，避免重复查会话表。
// 内部仍需查询 Group 获取 ExtractTodos 配置（每次仅 1 次查询）。
func TryExtractTodosWithPreloaded(senderID uint, conv *model.Conversation, conversationID uint, content string) {
	if todoExtractor == nil || conv == nil {
		return
	}

	if conv.Type != "group" {
		return
	}

	db := database.GetDB()
	var group model.Group
	if err := db.Where("conversation_id = ?", conversationID).First(&group).Error; err != nil {
		return
	}

	if !group.GetAIConfig().ExtractTodos {
		return
	}

	go todoExtractor.ExtractAndCreateTodos(content, senderID, conversationID)
}

// resolveAIName 已收编到 service.resolveAIName（BuildMessageResponse 内部调用）。

// buildUserReadReceiptSet 批量查询当前用户对指定消息的已读回执，
// 返回 set[messageID]bool。用于让 buildMessageResponse 返回 per-user 的 is_read。
func buildUserReadReceiptSet(msgs []model.Message, userID uint) map[uint]bool {
	if len(msgs) == 0 {
		return nil
	}
	ids := make([]uint, len(msgs))
	for i, m := range msgs {
		ids[i] = m.ID
	}
	var receiptMsgIDs []uint
	database.GetDB().Model(&model.MessageReadReceipt{}).
		Where("user_id = ? AND message_id IN ?", userID, ids).
		Pluck("message_id", &receiptMsgIDs)
	set := make(map[uint]bool, len(receiptMsgIDs))
	for _, id := range receiptMsgIDs {
		set[id] = true
	}
	return set
}

// buildMessageResponse 构建单条消息的 HTTP 响应（历史拉取、发送响应共用）。
// allMemberIDs 用于 @all 展开；currentUserID 用于计算 is_at_mention。
// userReadSet 非 nil 时，is_read 取决于当前用户是否在该集合中（per-user 已读状态）；
// 为 nil 时退化为全局 messages.is_read（发送响应等场景）。
// buildMessageResponse 已统一收编到 service.BuildMessageResponse（HTTP/WS/AI 三路共用）。
// handler 层调用点统一传入 per-user 上下文：CurrentUserID/AllMemberIDs/UserReadSet。

// getAllMemberIDs 查询会话全体成员 ID（用于 @all 展开）。
func getAllMemberIDs(convID uint) []uint {
	db := database.GetDB()
	var members []model.ConversationMember
	db.Where("conversation_id = ?", convID).Find(&members)
	ids := make([]uint, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.UserID)
	}
	return ids
}

// buildCardActionMap 批量查当前用户对给定消息中卡片消息的点击记录，返回 message_id -> action_id。
// 单次查询避免 N+1；CardActionRecord 的 message_id+user_id 唯一索引保证每卡至多一条。
// 用于让客户端跨设备一致地恢复卡片"已点击"态（替代仅本设备的 localStorage）。
func buildCardActionMap(msgs []model.Message, userID uint) map[uint]string {
	cardIDs := make([]uint, 0)
	for _, m := range msgs {
		if m.Type == "card" {
			cardIDs = append(cardIDs, m.ID)
		}
	}
	if len(cardIDs) == 0 {
		return map[uint]string{}
	}
	var records []model.CardActionRecord
	database.GetDB().Where("user_id = ? AND message_id IN ?", userID, cardIDs).Find(&records)
	m := make(map[uint]string, len(records))
	for _, r := range records {
		m[r.MessageID] = r.ActionID
	}
	return m
}

func GetMessages(c *gin.Context) {
	uid, ok := getUserIDFromContext(c)
	if !ok {
		response.Unauthorized(c, "用户未登录")
		return
	}
	convID := c.Param("id")

	if strings.HasPrefix(convID, "conv_") {
		convID = strings.TrimPrefix(convID, "conv_")
	}

	pageStr := c.Query("page")
	pageSizeStr := c.Query("page_size")
	afterIDStr := c.Query("after_id")
	beforeIDStr := c.Query("before_id")

	page := 1
	pageSize := 20
	afterID := uint(0)
	beforeID := uint(0)

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	if afterIDStr != "" {
		if id, err := strconv.ParseUint(afterIDStr, 10, 64); err == nil {
			afterID = uint(id)
		}
	}

	if beforeIDStr != "" {
		if id, err := strconv.ParseUint(beforeIDStr, 10, 64); err == nil {
			beforeID = uint(id)
		}
	}

	offset := (page - 1) * pageSize

	convIDUint, _ := strconv.ParseUint(convID, 10, 32)

	convSvc := di.GlobalContainer.ConversationService
	msgSvc := di.GlobalContainer.MessageService

	isMember, _ := convSvc.IsConversationMember(uint(convIDUint), uid)
	if !isMember {
		query := service.MessageQuery{
			ConvID: uint(convIDUint),
			UserID: uid,
			Limit:  1,
		}
		result, _ := msgSvc.GetMessages(query)
		if result == nil || result.Total == 0 {
			response.Forbidden(c, "无权限访问")
			return
		}
	}

	query := service.MessageQuery{
		ConvID: uint(convIDUint),
		UserID: uid,
		Limit:  pageSize,
		Offset: offset,
	}
	if afterID > 0 {
		query.AfterMsgID = afterID
	}
	if beforeID > 0 {
		query.BeforeMsgID = beforeID
	}
	result, err := msgSvc.GetMessages(query)
	if err != nil {
		response.InternalServerError(c, "获取消息失败")
		return
	}

	var responseMessages []gin.H
	allMemberIDs := getAllMemberIDs(uint(convIDUint))
	// 卡片点击态（跨设备一致）：批量查当前用户对本页卡片消息的 CardActionRecord，
	// 附 card_action_id 到 card 消息响应。CardActionRecord 是幂等表（message_id+user_id 唯一），权威。
	// 之前客户端仅靠 localStorage，换设备/清缓存后卡片视觉重置可点击（后端幂等保底不重复触发，但视觉不一致）。
	cardActionMap := buildCardActionMap(result.Messages, uid)
	// 批量查当前用户对本页消息的已读回执，让 is_read 反映 per-user 状态
	userReadSet := buildUserReadReceiptSet(result.Messages, uid)
	for _, msg := range result.Messages {
		resp := service.BuildMessageResponse(msg, service.MessageResponseOptions{
			CurrentUserID: uid,
			AllMemberIDs:  allMemberIDs,
			UserReadSet:   userReadSet,
		})
		if msg.Type == "card" {
			if actionID, ok := cardActionMap[msg.ID]; ok {
				resp["card_action_id"] = actionID
			}
		}
		responseMessages = append(responseMessages, resp)
	}

	totalPages := int(result.Total) / pageSize
	if int(result.Total)%pageSize > 0 {
		totalPages++
	}

	response.Success(c, gin.H{
		"messages": responseMessages,
		"pagination": gin.H{
			"current_page": page,
			"page_size":    pageSize,
			"total":        result.Total,
			"total_pages":  totalPages,
		},
	})
}

func GetMessagesByFilter(c *gin.Context) {
	uid, ok := getUserIDFromContext(c)
	if !ok {
		response.Unauthorized(c, "用户未登录")
		return
	}
	convID := c.Query("conversation_id")
	messageType := c.Query("type")
	pageStr := c.Query("page")
	pageSizeStr := c.Query("page_size")
	search := c.Query("search")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if convID == "" {
		response.BadRequest(c, "会话ID不能为空")
		return
	}

	convSvc := di.GlobalContainer.ConversationService
	msgSvc := di.GlobalContainer.MessageService

	convIDUint, _ := strconv.ParseUint(convID, 10, 32)
	isMember, _ := convSvc.IsConversationMember(uint(convIDUint), uid)
	if !isMember {
		response.Forbidden(c, "无权限访问")
		return
	}

	page := 1
	pageSize := 10
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}
	offset := (page - 1) * pageSize

	query := service.MessageQuery{
		ConvID:      uint(convIDUint),
		UserID:      uid,
		Limit:       pageSize,
		Offset:      offset,
		MessageType: messageType,
		Keyword:     search,
		StartDate:   startDate,
		EndDate:     endDate,
	}
	result, err := msgSvc.GetMessagesByFilter(query)
	if err != nil {
		response.InternalServerError(c, "获取消息失败")
		return
	}

	// 与 GetMessages 对齐：返回 per-user is_read 而非全局字段
	allMemberIDs := getAllMemberIDs(uint(convIDUint))
	userReadSet := buildUserReadReceiptSet(result.Messages, uid)
	var responseMessages []map[string]interface{}
	for _, msg := range result.Messages {
		responseMessages = append(responseMessages, service.BuildMessageResponse(msg, service.MessageResponseOptions{
			CurrentUserID: uid,
			AllMemberIDs:  allMemberIDs,
			UserReadSet:   userReadSet,
		}))
	}

	response.Success(c, gin.H{
		"messages": responseMessages,
		"total":    result.Total,
	})
}

func SendMessage(c *gin.Context) {
	uid, ok := getUserIDFromContext(c)
	if !ok {
		response.Unauthorized(c, "用户未登录")
		return
	}
	convID := c.Param("id")

	if strings.HasPrefix(convID, "conv_") {
		convID = strings.TrimPrefix(convID, "conv_")
	}

	var req struct {
		Type            string                 `json:"type" binding:"required"`
		Content         string                 `json:"content" binding:"required"`
		QuotedMessageID *uint                  `json:"quoted_message_id"`
		ShareData       map[string]interface{} `json:"share_data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	msgSvc := di.GlobalContainer.MessageService
	fileSvc := di.GlobalContainer.FileService

	convIDUint, _ := strconv.ParseUint(convID, 10, 32)

	msg, err := msgSvc.SendMessage(uint(convIDUint), uid, req.Type, req.Content, req.QuotedMessageID)
	if err != nil {
		if err == service.ErrMessageForbidden {
			response.Forbidden(c, "无权限发送消息")
			return
		}
		response.InternalServerError(c, "消息发送失败")
		return
	}

	// 「你发消息后，分身暂停回复」：本人在会话里发言后，给该会话的分身会话设
	// 一个暂停窗口（TakeoverUntil），期间分身不自动回复。仅发送者本人 = 分身主人时生效；
	// 未配置（SelfMessagePause<=0）或该会话无分身会话时静默跳过。
	applyOwnMessagePause(database.GetDB(), uid, uint(convIDUint))

	if req.Type == "file" || req.Type == "image" {
		var fileData struct {
			URL string `json:"url"`
			ID  uint   `json:"id"`
		}
		if err := json.Unmarshal([]byte(req.Content), &fileData); err == nil && fileData.ID > 0 {
			fileSvc.UpdateFileSource(fileData.ID, uid, "chat")
		}
	}

	allMemberIDs := getAllMemberIDs(uint(convIDUint))
	responseData := service.BuildMessageResponse(*msg, service.MessageResponseOptions{
		CurrentUserID: uid,
		AllMemberIDs:  allMemberIDs,
	})

	response.Success(c, responseData)
}

// applyOwnMessagePause 实现「你发消息后，分身暂停回复」：
// 分身主人在会话里发消息后，把该会话的分身会话的 TakeoverUntil 设为 now + SelfMessagePause，
// 期间分身不自动回复（worker pool / smart_reply 均按 TakeoverUntil 门控）。
// 未配置（SelfMessagePause<=0）、该会话无分身会话、或写入失败时静默跳过，不影响主流程。
func applyOwnMessagePause(db *gorm.DB, userID, convID uint) {
	if db == nil {
		return
	}

	// 内存缓存短路：缓存 SelfMessagePause 值（分钟），0 = 无配置或未启用
	cacheKey := fmt.Sprintf("pause:%d", userID)
	pauseMinutes := 0
	if cached, ok := cache.AvatarPauseCache.Get(cacheKey); ok {
		pauseMinutes = cached.(int)
	} else {
		var config model.AvatarConfig
		if err := db.Where("user_id = ?", userID).First(&config).Error; err != nil {
			cache.AvatarPauseCache.Put(cacheKey, 0) // 无配置，缓存 5 分钟
		} else {
			pauseMinutes = config.SelfMessagePause
			cache.AvatarPauseCache.Put(cacheKey, pauseMinutes)
		}
	}
	if pauseMinutes <= 0 {
		return
	}

	// 会话行不存在则无需暂停（分身未在该会话激活）
	var n int64
	db.Model(&model.AvatarSession{}).
		Where("user_id = ? AND conversation_id = ?", userID, convID).
		Count(&n)
	if n == 0 {
		return
	}

	takeoverUntil := time.Now().Add(time.Duration(pauseMinutes) * time.Minute)
	if err := db.Model(&model.AvatarSession{}).
		Where("user_id = ? AND conversation_id = ?", userID, convID).
		Update("takeover_until", takeoverUntil).Error; err != nil {
		logger.WithModule("SendMessage").Error("设置「发消息后分身暂停」失败", "userID", userID, "convID", convID, "error", err)
	}
}

// broadcastNewMessage 广播新消息到会话并更新相关状态（AI 消息广播路径）。
// 载荷构建统一走 service.BuildMessageResponse（与 HTTP 历史拉取、用户消息 WS 广播同源），
// 避免第三套手搓载荷字段漂移。
func broadcastNewMessage(msg *model.Message, excludeUserID uint, conv *model.Conversation) {
	convSvc := di.GlobalContainer.ConversationService

	now := time.Now()
	convSvc.UpdateConversation(msg.ConversationID, map[string]interface{}{
		"last_message_id": msg.ID,
		"last_message_at": now,
	})

	// 恢复会话显示：新消息到来时，如果会话被隐藏则恢复显示
	// is_hidden 存储在 ConversationSession 表中（用户级别），不在 Conversation 表中
	// 这里只更新被隐藏的会话，不影响已显示的会话
	db := database.GetDB()
	db.Model(&model.ConversationSession{}).
		Where("conversation_id = ? AND is_hidden = ?", msg.ConversationID, true).
		Update("is_hidden", false)

	if excludeUserID > 0 {
		convSvc.IncrementUnreadCount(msg.ConversationID, excludeUserID)
	} else if msg.Origin == "assistant" || msg.Origin == "avatar" {
		// AI 消息：为所有非发送者增加未读计数（单条 SQL，排除发送者）
		convSvc.IncrementUnreadCount(msg.ConversationID, msg.SenderID)
	}

	// AI 消息不含 mention（AI 不会 @ 人），mention_user_ids 恒为空数组
	mentionUserIDs := []uint{}

	responseData := service.BuildMessageResponse(*msg, service.MessageResponseOptions{
		MentionUserIDs: mentionUserIDs,
		BroadcastWS:    true,
	})

	if ws.GlobalHub != nil {
		newMsg := ws.WSMessage{
			Type: "new_message",
			Data: responseData,
		}
		jsonMsg, _ := json.Marshal(newMsg)
		utils.SafeGo(func() { ws.GlobalHub.SendToConversationAsync(msg.ConversationID, excludeUserID, jsonMsg) })
	}
}

func StreamMessage(c *gin.Context) {
	uid, ok := getUserIDFromContext(c)
	if !ok {
		response.Unauthorized(c, "用户未登录")
		return
	}
	convID := c.Param("id")

	if strings.HasPrefix(convID, "conv_") {
		convID = strings.TrimPrefix(convID, "conv_")
	}

	var req struct {
		Type            string                 `json:"type" binding:"required"`
		Content         string                 `json:"content" binding:"required"`
		QuotedMessageID *uint                  `json:"quoted_message_id"`
		ShareData       map[string]interface{} `json:"share_data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	convSvc := di.GlobalContainer.ConversationService
	msgSvc := di.GlobalContainer.MessageService

	convIDUint, _ := strconv.ParseUint(convID, 10, 32)

	isMember, _ := convSvc.IsConversationMember(uint(convIDUint), uid)
	if !isMember {
		response.Forbidden(c, "无权限发送消息")
		return
	}

	conv, err := convSvc.GetConversation(uint(convIDUint))
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if conv.Type != "bot" {
		response.BadRequest(c, "仅支持机器人会话的流式消息")
		return
	}

	content := req.Content

	msg := model.Message{
		ConversationID:  uint(convIDUint),
		SenderID:        uid,
		Type:            req.Type,
		Content:         content,
		QuotedMessageID: req.QuotedMessageID,
		IsRead:          false,
	}
	msgSvc.CreateMessage(&msg)

	now := time.Now()
	convSvc.UpdateConversation(conv.ID, map[string]interface{}{
		"last_message_id": msg.ID,
		"last_message_at": now,
	})

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	responseChan := make(chan ai.StreamChunk)

	utils.SafeGoWithLabel("stream-message", func() {
		defer close(responseChan)

		db := database.GetDB()
		var botConv model.BotConversation
		if err := db.Where("conversation_id = ?", convID).First(&botConv).Error; err != nil {
			logger.WithModule("StreamMessage").Error("查找机器人会话关联失败", "error", err)
			return
		}

		var bot model.Bot
		if err := db.First(&bot, botConv.BotID).Error; err != nil {
			logger.WithModule("StreamMessage").Error("查找机器人信息失败", "error", err)
			return
		}

		// 检查机器人是否已启用（审批通过）
		if !bot.IsActive {
			logger.WithModule("StreamMessage").Error("机器人未启用", "botID", botConv.BotID)
			return
		}

		systemPrompt := "你是一个智能助手，帮助用户解决问题。"
		if bot.Config != "" {
			var botConfig map[string]interface{}
			if err := json.Unmarshal([]byte(bot.Config), &botConfig); err == nil {
				if prompt, ok := botConfig["system_prompt"].(string); ok && prompt != "" {
					systemPrompt = prompt
				}
			}
		}

		promptCtx := &service.PromptContext{
			CustomPrompt: systemPrompt,
		}
		systemPrompt = di.GlobalContainer.PromptManager.BuildSystemPrompt(service.SceneBotChat, promptCtx)

		var messages []model.Message
		db.Where("conversation_id = ?", convID).Order("created_at ASC").Limit(20).Find(&messages)

		logger.WithModule("StreamMessage").Info("加载历史消息",
			"conversationID", convID,
			"messageCount", len(messages),
			"botID", botConv.BotID,
			"virtualUserID", bot.VirtualUserID,
		)

		systemUserID := service.NewUserService(db).GetSystemUserID()

		var aiMessages []ai.Message
		aiMessages = append(aiMessages, ai.Message{
			Role:    "system",
			Content: systemPrompt,
		})

		for _, msg := range messages {
			role := "user"
			if msg.SenderID == systemUserID || (bot.VirtualUserID != nil && msg.SenderID == *bot.VirtualUserID) {
				role = "assistant"
			}
			aiMessages = append(aiMessages, ai.Message{
				Role:    role,
				Content: msg.Content,
			})
		}

		// 确保最新的用户问题包含在消息中（防止数据库查询延迟导致新消息未被加载）
		// 检查最后一条消息是否是用户的新问题
		if len(aiMessages) == 1 || aiMessages[len(aiMessages)-1].Role != "user" || aiMessages[len(aiMessages)-1].Content != content {
			aiMessages = append(aiMessages, ai.Message{
				Role:    "user",
				Content: content,
			})
		}

		logger.WithModule("StreamMessage").Info("准备调用 AI", "messageCount", len(aiMessages))
		for i, aiMsg := range aiMessages {
			if i == 0 {
				logger.WithModule("StreamMessage").Info("AI消息 - system", "index", i, "role", aiMsg.Role)
			} else {
				logger.WithModule("StreamMessage").Info("AI消息", "index", i, "role", aiMsg.Role, "content", aiMsg.Content[:min(50, len(aiMsg.Content))])
			}
		}

		var fullResponse string
		err := di.GlobalContainer.AIService.GetCompletionStream(ai.TaskTypeChat, aiMessages, func(chunk ai.StreamChunk) error {
			responseChan <- chunk
			fullResponse += chunk.Content
			return nil
		})

		if err != nil {
			logger.WithModule("StreamMessage").Error("AI API 调用失败", "error", err)
			errorMsg := "抱歉，AI 服务暂时不可用，请稍后再试。"
			responseChan <- ai.StreamChunk{Content: errorMsg}
			fullResponse = errorMsg
		}

		senderID := service.NewUserService(db).GetSystemUserID()
		if bot.VirtualUserID != nil {
			senderID = *bot.VirtualUserID
		}

		botReply := model.Message{
			ConversationID: uint(convIDUint),
			SenderID:       senderID,
			Type:           "markdown",
			Content:        fullResponse,
			Origin:         "assistant",
		}
		if err := db.Create(&botReply).Error; err != nil {
			logger.WithModule("StreamMessage").Error("保存机器人回复失败", "error", err)
			return
		}
		if err := db.Preload("Sender").First(&botReply, botReply.ID).Error; err != nil {
			logger.WithModule("StreamMessage").Error("预加载机器人回复发送者失败", "error", err, "messageID", botReply.ID)
			return
		}

		now := time.Now()
		if err := convSvc.UpdateConversation(conv.ID, map[string]interface{}{
			"last_message_id": botReply.ID,
			"last_message_at": now,
		}); err != nil {
			logger.WithModule("StreamMessage").Error("更新 Bot 会话最后消息失败", "error", err, "conversationID", conv.ID, "messageID", botReply.ID)
			return
		}

		logLength := 100
		if len(fullResponse) < logLength {
			logLength = len(fullResponse)
		}
		logger.WithModule("StreamMessage").Info("机器人回复保存成功", "content", fullResponse[:logLength])
	})

	c.Writer.Write([]byte("data: \n\n"))
	c.Writer.Flush()

	for {
		select {
		case chunk, ok := <-responseChan:
			if !ok {
				finish := "stop"
				doneData, _ := json.Marshal(ai.StreamChunk{Finish: &finish})
				c.Writer.Write([]byte("data: " + string(doneData) + "\n\n"))
				c.Writer.Flush()
				return
			}
			data, _ := json.Marshal(chunk)
			c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}

func RecallMessage(c *gin.Context) {
	uid, ok := getUserIDFromContext(c)
	if !ok {
		response.Unauthorized(c, "用户未登录")
		return
	}
	msgIDStr := c.Param("id")

	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息ID")
		return
	}

	msgSvc := di.GlobalContainer.MessageService

	msg, err := msgSvc.RecallMessage(uint(msgID), uid)
	if err != nil {
		if err == service.ErrMessageNotFound {
			response.NotFound(c, "消息不存在")
			return
		}
		if err == service.ErrMessageForbidden {
			response.Forbidden(c, "只能撤回自己发送的消息")
			return
		}
		if err == service.ErrMessageAlreadyRecalled {
			response.BadRequest(c, "消息已经被撤回")
			return
		}
		if err == service.ErrMessageRecallTimeout {
			response.BadRequest(c, "消息已超过撤回时限")
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "消息撤回成功", "data": msg})
}

func RemindMessage(c *gin.Context) {
	uid, ok := getUserIDFromContext(c)
	if !ok {
		response.Unauthorized(c, "用户未登录")
		return
	}
	msgIDStr := c.Param("id")

	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息ID")
		return
	}

	msgSvc := di.GlobalContainer.MessageService
	db := di.GlobalContainer.DB

	msg, err := msgSvc.GetMessageByID(uint(msgID))
	if err != nil {
		response.NotFound(c, "消息不存在")
		return
	}

	// 校验：是发送者本人
	if msg.SenderID != uid {
		response.Forbidden(c, "无权限发送提醒")
		return
	}

	// 校验：单聊（与前端 canSendReminder 逻辑一致，后端兜底）
	var conv model.Conversation
	if err := db.First(&conv, msg.ConversationID).Error; err != nil {
		response.InternalServerError(c, "会话不存在")
		return
	}
	if conv.Type != "single" {
		response.BadRequest(c, "仅支持单聊消息提醒")
		return
	}

	// 校验：消息未读
	if msg.IsRead {
		response.BadRequest(c, "消息已被读，无需提醒")
		return
	}

	// 校验：超触发门槛（秒，0=禁止提醒），由系统配置 messageRemindTime 控制，默认 3600
	remindTimeLimit := 3600
	remindCooldown := time.Hour
	if configSvc := service.NewSystemConfigService(db); configSvc != nil {
		if publicConfigs, err := configSvc.GetPublicConfigs(); err == nil {
			if v, ok := publicConfigs["messageRemindTime"]; ok {
				if iv, ok := v.(int); ok {
					remindTimeLimit = iv
				}
			}
			if v, ok := publicConfigs["messageRemindRepeatCooldown"]; ok {
				if iv, ok := v.(int); ok {
					remindCooldown = time.Duration(iv) * time.Second
				}
			}
		}
	}
	remindRateLimiter.SetCooldown(remindCooldown)
	if remindTimeLimit == 0 {
		response.BadRequest(c, "消息提醒已关闭")
		return
	}
	if time.Since(msg.CreatedAt) < time.Duration(remindTimeLimit)*time.Second {
		if remindTimeLimit < 60 {
			response.BadRequest(c, fmt.Sprintf("消息发送未满 %d 秒", remindTimeLimit))
		} else {
			response.BadRequest(c, fmt.Sprintf("消息发送未满 %d 分钟", remindTimeLimit/60))
		}
		return
	}

	// 校验：非 bot 消息
	if msg.Sender.Type == "bot" {
		response.BadRequest(c, "不支持机器人消息提醒")
		return
	}

	if reason := remindRateLimiter.check(msg.ID, time.Now()); reason != reminderAllowed {
		response.BadRequest(c, reminderLimiterReasonMessage(reason, remindCooldown))
		return
	}

	// 读取 webhook 配置
	webhookCfg, err := service.LoadWebhookConfig(db)
	if err != nil || !webhookCfg.Enabled {
		response.BadRequest(c, "提醒功能未配置或未启用")
		return
	}

	// 查询接收者（单聊中除发送者外的成员）
	var recipient model.User
	if err := db.Where("id IN (?) AND id != ?",
		db.Model(&model.ConversationMember{}).Select("user_id").Where("conversation_id = ?", msg.ConversationID),
		uid,
	).First(&recipient).Error; err != nil {
		response.InternalServerError(c, "接收者不存在")
		return
	}

	// 构造 MessageURL（可选，依赖 QIM_BASE_URL 环境变量）
	baseURL := os.Getenv("QIM_BASE_URL")
	messageURL := ""
	if baseURL != "" {
		messageURL = fmt.Sprintf("%s/chat?conv=%d&msg=%d", baseURL, msg.ConversationID, msg.ID)
	}

	data := service.RemindData{
		MessageID:             msg.ID,
		ConversationID:        msg.ConversationID,
		ConversationType:      conv.Type,
		SenderID:              msg.Sender.ID,
		SenderUsername:        msg.Sender.Username,
		SenderNickname:        msg.Sender.Nickname,
		SenderEmail:           msg.Sender.Email,
		RecipientID:           recipient.ID,
		RecipientUsername:     recipient.Username,
		RecipientNickname:     recipient.Nickname,
		RecipientEmail:        recipient.Email,
		MessageContentPreview: msg.Content,
		MessageType:           msg.Type,
		MessageSentAt:         msg.CreatedAt.Format(time.RFC3339),
		MessageURL:            messageURL,
	}

	// 频率限制：同一消息成功提醒后在冷却期内最多 1 次（冷却时长由系统配置控制）；调用中阻止重复点击。
	if !remindRateLimiter.start(msg.ID, time.Now()) {
		reason := remindRateLimiter.check(msg.ID, time.Now())
		response.BadRequest(c, reminderLimiterReasonMessage(reason, remindCooldown))
		return
	}

	// 立即返回，异步调用 webhook
	response.Success(c, gin.H{"message": "提醒发送中"})

	// 异步调用 webhook，结果通过 WebSocket 回执给发送方
	senderID := uid
	utils.SafeGoWithLabel("webhook-remind", func() {
		success := false
		defer finishReminderAttempt(remindRateLimiter, msg.ID, &success)

		hub := di.GlobalContainer.WebSocketHub
		var result []byte

		if err := service.SendRemind(webhookCfg, data); err != nil {
			logger.WithModule("Remind").Error("webhook 调用失败",
				"message_id", msg.ID, "error", err)
			result, _ = buildRemindResult(msg.ID, false, err.Error(), "")
		} else {
			success = true
			result, _ = buildRemindResult(msg.ID, true, "", webhookCfg.SystemName)
		}
		hub.SendToUser(senderID, result)
	})
}

func DeleteMessage(c *gin.Context) {
	uid, ok := getUserIDFromContext(c)
	if !ok {
		response.Unauthorized(c, "用户未登录")
		return
	}
	msgIDStr := c.Param("id")

	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息ID")
		return
	}

	msgSvc := di.GlobalContainer.MessageService

	err = msgSvc.DeleteMessage(uint(msgID), uid)
	if err != nil {
		if err == service.ErrMessageNotFound {
			response.NotFound(c, "消息不存在")
			return
		}
		if err == service.ErrMessageForbidden {
			response.Forbidden(c, "只能删除自己发送的消息")
			return
		}
		response.InternalServerError(c, "删除消息失败")
		return
	}

	response.Success(c, gin.H{
		"message": "消息删除成功",
	})
}

func GetMessageReadUsers(c *gin.Context) {
	configSvc := di.GlobalContainer.SystemConfigService
	publicConfigs, _ := configSvc.GetPublicConfigs()
	if enableReadReceipt, ok := publicConfigs["enableReadReceipt"]; ok {
		if !enableReadReceipt.(bool) {
			response.Success(c, gin.H{
				"read_users":    []interface{}{},
				"read_count":    int64(0),
				"total_members": int64(0),
			})
			return
		}
	}

	uid, ok := getUserIDFromContext(c)
	if !ok {
		response.Unauthorized(c, "用户未登录")
		return
	}
	msgIDStr := c.Param("id")

	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息ID")
		return
	}

	msgSvc := di.GlobalContainer.MessageService

	readUsers, totalMembers, err := msgSvc.GetMessageReadUsers(uint(msgID), uid)
	if err != nil {
		if err == service.ErrMessageNotFound {
			response.NotFound(c, "消息不存在")
			return
		}
		if err == service.ErrMessageForbidden {
			response.Forbidden(c, "无权限访问")
			return
		}
		response.InternalServerError(c, "获取已读用户失败")
		return
	}

	response.Success(c, gin.H{
		"read_users":    readUsers,
		"read_count":    int64(len(readUsers)),
		"total_members": totalMembers,
	})
}

func BatchGetMessageReadUsers(c *gin.Context) {
	configSvc := di.GlobalContainer.SystemConfigService
	publicConfigs, _ := configSvc.GetPublicConfigs()
	if enableReadReceipt, ok := publicConfigs["enableReadReceipt"]; ok {
		if !enableReadReceipt.(bool) {
			response.Success(c, map[string]interface{}{})
			return
		}
	}

	uid, ok := getUserIDFromContext(c)
	if !ok {
		response.Unauthorized(c, "用户未登录")
		return
	}

	var req struct {
		MessageIDs []uint `json:"message_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	if len(req.MessageIDs) == 0 {
		response.BadRequest(c, "消息ID列表不能为空")
		return
	}

	if len(req.MessageIDs) > 50 {
		response.BadRequest(c, "一次最多查询50条消息")
		return
	}

	msgSvc := di.GlobalContainer.MessageService

	result, err := msgSvc.BatchGetMessageReadUsers(req.MessageIDs, uid)
	if err != nil {
		response.InternalServerError(c, "批量获取已读用户失败")
		return
	}

	response.Success(c, result)
}

func MarkConversationAsRead(c *gin.Context) {
	uid, ok := getUserIDFromContext(c)
	if !ok {
		response.Unauthorized(c, "用户未登录")
		return
	}
	convIDStr := c.Param("id")

	if strings.HasPrefix(convIDStr, "conv_") {
		convIDStr = strings.TrimPrefix(convIDStr, "conv_")
	}

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的会话ID")
		return
	}

	msgSvc := di.GlobalContainer.MessageService

	err = msgSvc.MarkAsRead(uint(convID), uid)
	if err != nil {
		if err == service.ErrMessageForbidden {
			response.Forbidden(c, "无权限访问")
			return
		}
		response.InternalServerError(c, "标记已读失败")
		return
	}

	response.Success(c, gin.H{
		"message": "标记已读成功",
	})
}

func SearchMessages(c *gin.Context) {
	uid, ok := getUserIDFromContext(c)
	if !ok {
		response.Unauthorized(c, "用户未登录")
		return
	}

	keyword := c.Query("keyword")
	convID := c.Query("conv_id")
	pageStr := c.Query("page")
	pageSizeStr := c.Query("pageSize")

	page := 1
	pageSize := 20

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	offset := (page - 1) * pageSize

	msgSvc := di.GlobalContainer.MessageService

	var convIDPtr *uint
	if convID != "" {
		id, _ := strconv.ParseUint(convID, 10, 32)
		cid := uint(id)
		convIDPtr = &cid
	}

	messages, err := msgSvc.SearchMessages(uid, keyword, convIDPtr, pageSize, offset)
	if err != nil {
		response.InternalServerError(c, "搜索消息失败")
		return
	}

	// 搜索结果可能跨会话，按会话分组获取成员列表 + 统一查已读回执
	convMemberMap := make(map[uint][]uint)
	for _, m := range messages {
		if _, ok := convMemberMap[m.ConversationID]; !ok {
			convMemberMap[m.ConversationID] = getAllMemberIDs(m.ConversationID)
		}
	}
	userReadSet := buildUserReadReceiptSet(messages, uid)
	var responseMessages []map[string]interface{}
	for _, msg := range messages {
		responseMessages = append(responseMessages, service.BuildMessageResponse(msg, service.MessageResponseOptions{
			CurrentUserID: uid,
			AllMemberIDs:  convMemberMap[msg.ConversationID],
			UserReadSet:   userReadSet,
		}))
	}

	response.Success(c, gin.H{
		"list":  responseMessages,
		"total": len(responseMessages),
		"page":  page,
	})
}

func GetMessageQuoteChain(c *gin.Context) {
	uid, ok := getUserIDFromContext(c)
	if !ok {
		response.Unauthorized(c, "用户未登录")
		return
	}
	msgIDStr := c.Param("id")

	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息ID")
		return
	}

	msgSvc := di.GlobalContainer.MessageService

	quoteChain, err := msgSvc.GetMessageQuoteChain(uint(msgID), uid)
	if err != nil {
		if err == service.ErrMessageNotFound {
			response.NotFound(c, "消息不存在")
			return
		}
		if err == service.ErrMessageForbidden {
			response.Forbidden(c, "无权限访问")
			return
		}
		response.InternalServerError(c, "获取引用链失败")
		return
	}

	response.Success(c, gin.H{
		"messages": quoteChain,
	})
}
