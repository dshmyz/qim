package middleware

import (
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/gin-gonic/gin"
)

// NodeAuthMiddleware 验证节点间通信的内部认证密钥
// 用于保护 /node/broadcast 和 /node/send-to-user 等内部接口
func NodeAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if secret == "" {
			logger.WithModule("Auth").Warn("节点认证密钥未配置，拒绝所有节点通信请求")
			response.Unauthorized(c, "节点认证密钥未配置")
			c.Abort()
			return
		}

		nodeSecret := c.GetHeader("Node-Secret")
		if nodeSecret == "" {
			response.Unauthorized(c, "缺少节点认证密钥")
			c.Abort()
			return
		}

		if nodeSecret != secret {
			response.Unauthorized(c, "节点认证密钥无效")
			c.Abort()
			return
		}

		c.Next()
	}
}
