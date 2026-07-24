package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/utils"
	"github.com/gin-gonic/gin"
)

// HashBotToken 返回明文 token 的 sha256 十六进制摘要。
// 库内只存哈希，明文仅在签发时返回一次。
func HashBotToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// BotAuthMiddleware 校验外部 agent 调用 Bot API 的访问令牌。
// 令牌来自 Authorization: Bearer <token>；按 sha256 查 BotToken（软删除即撤销），
// 预加载 Bot 并校验其处于启用状态。通过后向 context 注入 bot_id / bot / virtual_user_id。
// 仿 NodeAuthMiddleware，但改为 per-bot token 而非全局共享密钥。
func BotAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}
		if tokenString == "" {
			response.Unauthorized(c, "未提供 Bot 令牌")
			c.Abort()
			return
		}

		db := database.GetDB()
		var botToken model.BotToken
		if err := db.Where("token_hash = ?", HashBotToken(tokenString)).
			Preload("Bot").First(&botToken).Error; err != nil {
			response.Unauthorized(c, "Bot 令牌无效")
			c.Abort()
			return
		}

		bot := botToken.Bot
		if !bot.IsActive {
			response.Forbidden(c, "机器人未启用")
			c.Abort()
			return
		}
		if bot.VirtualUserID == nil {
			response.Forbidden(c, "机器人未配置虚拟用户")
			c.Abort()
			return
		}

		c.Set("bot_id", bot.ID)
		c.Set("bot", &bot)
		c.Set("virtual_user_id", *bot.VirtualUserID)
		c.Set("bot_token_id", botToken.ID)
		// 便于 OperationLogService.LogUserOperation 将出站审计归属到 bot 虚拟用户
		c.Set("user_id", *bot.VirtualUserID)
		c.Set("username", bot.Name)

		// 异步刷新最后使用时间，避免探测时序泄漏
		tokenID := botToken.ID
		utils.SafeGoWithLabel("bot-token-used", func() {
			now := time.Now()
			if err := database.GetDB().Model(&model.BotToken{}).
				Where("id = ?", tokenID).
				Update("last_used_at", now).Error; err != nil {
				// 仅记录，不影响主流程
				_ = err
			}
		})

		c.Next()
	}
}
