package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/middleware"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/gin-gonic/gin"
)

// BotAPIHandler 处理外部 agent 经 Bot API 的出站消息，以及 bot 令牌/配置管理。
type BotAPIHandler struct {
	botMessaging *service.BotMessagingService
}

func NewBotAPIHandler(botMessaging *service.BotMessagingService) *BotAPIHandler {
	return &BotAPIHandler{botMessaging: botMessaging}
}

// SendMessage 外部 agent -> QIM 用户出站消息
// POST /api/v1/bot/messages  (BotAuthMiddleware)
func (h *BotAPIHandler) SendMessage(c *gin.Context) {
	botVal, _ := c.Get("bot")
	bot, ok := botVal.(*model.Bot)
	if !ok || bot == nil {
		response.Unauthorized(c, "Bot 身份无效")
		return
	}

	var req struct {
		ToUserID       uint   `json:"to_user_id"`
		ToUserName     string `json:"to_user_name"` // 可选：按用户名/昵称解析
		Content        string `json:"content"`
		MsgType        string `json:"msg_type"`
		ReplyToID      *uint  `json:"reply_to_id"`
		ThreadID       *uint  `json:"thread_id"`
		ThreadName     string `json:"thread_name"`     // 可选：按名称解析已有会话
		ConversationID uint   `json:"conversation_id"` // 可选：按会话(单聊/群聊)发送，压倒 to_user_id/thread 路径
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	// conversation_id 走「按会话发送」（群聊出站/已建单聊），优先于 to_user 解析。
	if req.ConversationID != 0 {
		if req.ReplyToID != nil && *req.ReplyToID == 0 {
			response.BadRequest(c, "reply_to_id 无效")
			return
		}
		msg, err := h.botMessaging.SendOutboundByConversation(bot, req.ConversationID, req.Content, req.MsgType, req.ReplyToID)
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		if svc := di.GlobalContainer.OperationLogService; svc != nil {
			svc.LogUserOperation(c, "bot", "send_message")
		}
		response.Success(c, gin.H{
			"message_id":      msg.ID,
			"conversation_id": msg.ConversationID,
			"created_at":      msg.CreatedAt,
		})
		return
	}

	// to_user_id 或 to_user_name 二选一
	if req.ToUserID == 0 && req.ToUserName != "" {
		uid, err := h.botMessaging.ResolveUserID(req.ToUserName)
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		req.ToUserID = uid
	}
	if req.ToUserID == 0 {
		response.BadRequest(c, "缺少 to_user_id 或 to_user_name")
		return
	}

	// thread_name 优先：按名称解析已有会话
	if req.ThreadID == nil && req.ThreadName != "" {
		tid, err := h.botMessaging.ResolveBotThread(bot, req.ThreadName)
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		req.ThreadID = &tid
	}

	if req.ReplyToID != nil && *req.ReplyToID == 0 {
		response.BadRequest(c, "reply_to_id 无效")
		return
	}

	msg, err := h.botMessaging.SendOutbound(bot, req.ToUserID, req.Content, req.MsgType, req.ThreadID, req.ReplyToID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if svc := di.GlobalContainer.OperationLogService; svc != nil {
		svc.LogUserOperation(c, "bot", "send_message")
	}

	response.Success(c, gin.H{
		"message_id":      msg.ID,
		"conversation_id": msg.ConversationID,
		"created_at":      msg.CreatedAt,
	})
}

// ListBotGroups 列出该 bot 已入群的群会话（conversation_id + 群名），供 agent 主动群发前发现群。
// GET /api/v1/bot/groups  (BotAuthMiddleware)
func (h *BotAPIHandler) ListBotGroups(c *gin.Context) {
	botVal, _ := c.Get("bot")
	bot, ok := botVal.(*model.Bot)
	if !ok || bot == nil {
		response.Unauthorized(c, "Bot 身份无效")
		return
	}

	groups, err := h.botMessaging.ListBotGroupConversations(bot.ID)
	if err != nil {
		response.InternalServerError(c, "查询群会话失败")
		return
	}
	if groups == nil {
		groups = []service.BotGroupConversation{}
	}
	response.Success(c, groups)
}

// GetBotMessages 外部 agent pull 读取自己会话的消息（增量轮询）。
// GET /api/v1/bot/messages?thread_id=X&after_id=Y&limit=N  (BotAuthMiddleware)
// 也可用 thread_name 代替 thread_id，按用户名/昵称自动解析会话。
func (h *BotAPIHandler) GetBotMessages(c *gin.Context) {
	botVal, _ := c.Get("bot")
	bot, ok := botVal.(*model.Bot)
	if !ok || bot == nil {
		response.Unauthorized(c, "Bot 身份无效")
		return
	}

	// 支持 thread_id 或 thread_name（二选一）
	var threadID uint64
	if name := c.Query("thread_name"); name != "" {
		tid, err := h.botMessaging.ResolveBotThread(bot, name)
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		threadID = uint64(tid)
	} else {
		var err error
		threadID, err = strconv.ParseUint(c.Query("thread_id"), 10, 32)
		if err != nil || threadID == 0 {
			response.BadRequest(c, "缺少 thread_id 或 thread_name")
			return
		}
	}
	var afterID uint64
	if s := c.Query("after_id"); s != "" {
		afterID, _ = strconv.ParseUint(s, 10, 32)
	}
	limit := 50
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			limit = n
		}
	}

	msgs, err := h.botMessaging.ListBotMessages(bot, uint(threadID), uint(afterID), limit)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 精简为 agent 关心字段：sender_type 用于跳过自己发的回复
	type botMsg struct {
		ID             uint      `json:"id"`
		ConversationID uint      `json:"conversation_id"`
		SenderID       uint      `json:"sender_id"`
		SenderType     string    `json:"sender_type"`
		SenderNickname string    `json:"sender_nickname"`
		Content        string    `json:"content"`
		Type           string    `json:"type"`
		Origin         string    `json:"origin"`
		CreatedAt      time.Time `json:"created_at"`
	}
	out := make([]botMsg, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, botMsg{
			ID:             m.ID,
			ConversationID: m.ConversationID,
			SenderID:       m.SenderID,
			SenderType:     m.Sender.Type,
			SenderNickname: m.Sender.Nickname,
			Content:        m.Content,
			Type:           m.Type,
			Origin:         m.Origin,
			CreatedAt:      m.CreatedAt,
		})
	}
	response.Success(c, gin.H{"messages": out})
}

// StreamChunk 外部 agent 追加流式消息分段（delta / finish）。
// POST /api/v1/bot/messages/:id/stream  (BotAuthMiddleware)
func (h *BotAPIHandler) StreamChunk(c *gin.Context) {
	botVal, _ := c.Get("bot")
	bot, ok := botVal.(*model.Bot)
	if !ok || bot == nil {
		response.Unauthorized(c, "Bot 身份无效")
		return
	}

	messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息 ID")
		return
	}

	var req struct {
		ContentDelta string `json:"content_delta"`
		Finish       bool   `json:"finish"`
	}
	_ = c.ShouldBindJSON(&req) // 允许空 body（仅 finish）

	if err := h.botMessaging.StreamChunk(bot, uint(messageID), req.ContentDelta, req.Finish); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if svc := di.GlobalContainer.OperationLogService; svc != nil {
		svc.LogUserOperation(c, "bot", "stream_chunk")
	}

	response.SuccessWithMessage(c, "流式分段已处理", nil)
}

// UpdateMessage 外部 agent 全量更新一条已存在的 bot 消息（用于回写卡片状态）。
// PUT /api/v1/bot/messages/:id  (BotAuthMiddleware)
func (h *BotAPIHandler) UpdateMessage(c *gin.Context) {
	botVal, _ := c.Get("bot")
	bot, ok := botVal.(*model.Bot)
	if !ok || bot == nil {
		response.Unauthorized(c, "Bot 身份无效")
		return
	}

	messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息 ID")
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
		MsgType string `json:"msg_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	if err := h.botMessaging.UpdateMessageContent(bot, uint(messageID), req.Content, req.MsgType); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if svc := di.GlobalContainer.OperationLogService; svc != nil {
		svc.LogUserOperation(c, "bot", "update_message")
	}

	response.SuccessWithMessage(c, "消息已更新", nil)
}

// SubmitCardAction 用户点击 bot 卡片按钮 -> 转发 action 到外部 agent webhook。
// POST /api/v1/messages/:id/card-action  (JWT 用户鉴权，即点击的人类用户)
func (h *BotAPIHandler) SubmitCardAction(c *gin.Context) {
	messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息 ID")
		return
	}

	userIDVal, _ := c.Get("user_id")
	userID, ok := userIDVal.(uint)
	if !ok {
		response.Unauthorized(c, "用户身份无效")
		return
	}

	var req struct {
		ActionID string `json:"action_id" binding:"required"`
		Value    string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	if err := h.botMessaging.ForwardCardAction(uint(messageID), userID, req.ActionID, req.Value); err != nil {
		// 幂等命中：该用户已处理过此卡片，不重复触发 webhook，返回 200"已处理"
		if errors.Is(err, service.ErrCardActionAlreadyHandled) {
			if svc := di.GlobalContainer.OperationLogService; svc != nil {
				svc.LogUserOperation(c, "bot", "card_action_already_handled")
			}
			response.SuccessWithMessage(c, "该卡片已处理", gin.H{"already_handled": true})
			return
		}
		// 已入重试队列：不阻塞用户，返回"已受理将重试"（HTTP 200）
		if errors.Is(err, service.ErrCardActionPendingRetry) {
			if svc := di.GlobalContainer.OperationLogService; svc != nil {
				svc.LogUserOperation(c, "bot", "card_action_pending_retry")
			}
			response.SuccessWithMessage(c, "卡片动作已受理，agent 暂不可用，正在重试投递", nil)
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	if svc := di.GlobalContainer.OperationLogService; svc != nil {
		svc.LogUserOperation(c, "bot", "card_action")
	}

	response.SuccessWithMessage(c, "卡片动作已提交", nil)
}

// generateBotToken 生成随机 bot 访问令牌明文（仅此一次返回）
func generateBotToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "qbot_" + hex.EncodeToString(b), nil
}

// ListBotTokens 列出 bot 的访问令牌（创建者或管理员）。不返回明文/hash。
// GET /api/v1/bots/:id/tokens  (authed)
func (h *BotAPIHandler) ListBotTokens(c *gin.Context) {
	botID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的 Bot ID")
		return
	}

	db := database.GetDB()
	var bot model.Bot
	if err := db.First(&bot, botID).Error; err != nil {
		response.NotFound(c, "机器人不存在")
		return
	}
	if !h.canManageBot(c, bot.CreatorID) {
		response.Forbidden(c, "无权操作此 Bot")
		return
	}

	var tokens []model.BotToken
	if err := db.Where("bot_id = ?", botID).Order("created_at DESC").Find(&tokens).Error; err != nil {
		response.InternalServerError(c, "查询令牌失败")
		return
	}

	type tokenInfo struct {
		ID         uint       `json:"id"`
		Name       string     `json:"name"`
		CreatedAt  time.Time  `json:"created_at"`
		LastUsedAt *time.Time `json:"last_used_at"`
	}
	out := make([]tokenInfo, 0, len(tokens))
	for _, t := range tokens {
		out = append(out, tokenInfo{ID: t.ID, Name: t.Name, CreatedAt: t.CreatedAt, LastUsedAt: t.LastUsedAt})
	}
	response.Success(c, gin.H{"tokens": out})
}

// IssueToken 为 bot 签发访问令牌（创建者或管理员）
// POST /api/v1/bots/:id/token  (authed)
func (h *BotAPIHandler) IssueToken(c *gin.Context) {
	botID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的 Bot ID")
		return
	}

	db := database.GetDB()
	var bot model.Bot
	if err := db.First(&bot, botID).Error; err != nil {
		response.NotFound(c, "机器人不存在")
		return
	}
	if !h.canManageBot(c, bot.CreatorID) {
		response.Forbidden(c, "无权操作此 Bot")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	_ = c.ShouldBindJSON(&req)

	plain, err := generateBotToken()
	if err != nil {
		response.InternalServerError(c, "生成令牌失败")
		return
	}

	token := model.BotToken{
		BotID:     uint(botID),
		TokenHash: middleware.HashBotToken(plain),
		Name:      req.Name,
	}
	if err := db.Create(&token).Error; err != nil {
		response.InternalServerError(c, "保存令牌失败")
		return
	}

	if svc := di.GlobalContainer.OperationLogService; svc != nil {
		svc.LogUserOperation(c, "bot", "issue_token")
	}

	response.Success(c, gin.H{
		"token":      plain,
		"token_id":   token.ID,
		"bot_id":     bot.ID,
		"name":       token.Name,
		"created_at": token.CreatedAt,
	})
}

// RevokeToken 撤销 bot 访问令牌（创建者或管理员）。软删除即时生效。
// DELETE /api/v1/bots/:id/token/:tid  (authed)
func (h *BotAPIHandler) RevokeToken(c *gin.Context) {
	botID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的 Bot ID")
		return
	}
	tokenID, err := strconv.ParseUint(c.Param("tid"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的令牌 ID")
		return
	}

	db := database.GetDB()
	var bot model.Bot
	if err := db.First(&bot, botID).Error; err != nil {
		response.NotFound(c, "机器人不存在")
		return
	}
	if !h.canManageBot(c, bot.CreatorID) {
		response.Forbidden(c, "无权操作此 Bot")
		return
	}

	// 软删除即撤销（仅限该 bot 的令牌）
	result := db.Where("id = ? AND bot_id = ?", tokenID, botID).Delete(&model.BotToken{})
	if result.Error != nil || result.RowsAffected == 0 {
		response.NotFound(c, "令牌不存在或已撤销")
		return
	}

	if svc := di.GlobalContainer.OperationLogService; svc != nil {
		svc.LogUserOperation(c, "bot", "revoke_token")
	}

	response.SuccessWithMessage(c, "令牌已撤销", nil)
}

// UpdateBotConfig 更新 bot 的 Config（webhook 模式/地址/密钥），允许已审批 bot（创建者或管理员）
// PUT /api/v1/bots/:id/config  (authed)
func (h *BotAPIHandler) UpdateBotConfig(c *gin.Context) {
	botID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的 Bot ID")
		return
	}

	db := database.GetDB()
	var bot model.Bot
	if err := db.First(&bot, botID).Error; err != nil {
		response.NotFound(c, "机器人不存在")
		return
	}
	if !h.canManageBot(c, bot.CreatorID) {
		response.Forbidden(c, "无权操作此 Bot")
		return
	}

	var req struct {
		Mode            string `json:"mode"`
		WebhookURL      string `json:"webhook_url"`
		WebhookSecret   string `json:"webhook_secret"`
		UseSystemConfig *bool  `json:"use_system_config"` // 模型来源：nil=不修改；true=系统默认
		UserConfigID    *uint  `json:"user_config_id"`    // 自定义配置 ID；use_system_config=false 时生效
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}
	if req.Mode != "" && req.Mode != "internal_ai" && req.Mode != "external_webhook" {
		response.BadRequest(c, "mode 取值无效")
		return
	}

	// 合并到既有 Config：保留未提供字段
	cfg := service.ParseBotConfig(bot.Config)
	if req.Mode != "" {
		cfg.Mode = req.Mode
	}
	if req.WebhookURL != "" {
		cfg.WebhookURL = req.WebhookURL
	}
	if req.WebhookSecret != "" {
		cfg.WebhookSecret = req.WebhookSecret
	}
	// 模型来源：仅显式传入时更新
	if req.UseSystemConfig != nil {
		cfg.UseSystemConfig = *req.UseSystemConfig
		if *req.UseSystemConfig {
			cfg.UserConfigID = nil // 切回系统默认时清掉自定义配置引用
		} else if req.UserConfigID != nil {
			cfg.UserConfigID = req.UserConfigID
		} else {
			// use_system_config=false 但未带 user_config_id（前端清空了选择）→ 显式清掉旧引用，
			// 避免 UI 显示"未选"、bot 却继续用上次留下的配置。
			cfg.UserConfigID = nil
		}
	} else if req.UserConfigID != nil {
		cfg.UserConfigID = req.UserConfigID
		cfg.UseSystemConfig = false // 显式指定配置即视为自定义
	}
	configJSON, _ := json.Marshal(cfg)

	if err := db.Model(&bot).Update("config", string(configJSON)).Error; err != nil {
		response.InternalServerError(c, "更新配置失败")
		return
	}

	if svc := di.GlobalContainer.OperationLogService; svc != nil {
		svc.LogUserOperation(c, "bot", "update_config")
	}

	response.SuccessWithMessage(c, "配置已更新", gin.H{"config": cfg})
}

// canManageBot 调用方是否为 bot 创建者或系统管理员
func (h *BotAPIHandler) canManageBot(c *gin.Context, creatorID uint) bool {
	userIDVal, _ := c.Get("user_id")
	userID, ok := userIDVal.(uint)
	if !ok {
		return false
	}
	if creatorID == userID {
		return true
	}
	rolesVal, _ := c.Get("roles")
	roles, _ := rolesVal.([]string)
	for _, r := range roles {
		if r == "system_admin" {
			return true
		}
	}
	return false
}
