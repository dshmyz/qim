package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strconv"

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
		ToUserID uint   `json:"to_user_id" binding:"required"`
		Content  string `json:"content" binding:"required"`
		MsgType  string `json:"msg_type"`
		ThreadID *uint  `json:"thread_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	msg, err := h.botMessaging.SendOutbound(bot, req.ToUserID, req.Content, req.MsgType, req.ThreadID)
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
		Mode          string `json:"mode"`
		WebhookURL    string `json:"webhook_url"`
		WebhookSecret string `json:"webhook_secret"`
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
