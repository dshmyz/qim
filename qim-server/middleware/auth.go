package middleware

import (
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	TokenType string `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

func AuthMiddleware(secret string, userSvc *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// 优先从 Authorization header 获取 Bearer token
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		if tokenString == "" {
			response.Unauthorized(c, "未提供认证令牌")
			c.Abort()
			return
		}

		// 用户长期令牌（qusr_…）：供 qim CLI / qim-mcp 等以本人身份调用用户 API。
		// 按 sha256 查 UserToken（软删除即撤销），注入与 JWT 相同的 user_id/username/roles。
		// 必须在 JWT 解析前判断，否则非 JWT 的 qusr_ 串会被当成无效 JWT 直接拒绝。
		if strings.HasPrefix(tokenString, "qusr_") {
			db := database.GetDB()
			var ut model.UserToken
			if err := db.Where("token_hash = ?", HashBotToken(tokenString)).First(&ut).Error; err != nil {
				response.Unauthorized(c, "认证令牌无效")
				c.Abort()
				return
			}

			var user model.User
			if err := db.First(&user, ut.UserID).Error; err != nil {
				response.Unauthorized(c, "认证令牌无效")
				c.Abort()
				return
			}

			now := time.Now()
			db.Model(&model.UserToken{}).Where("id = ?", ut.ID).Update("last_used_at", now)

			c.Set("user_id", ut.UserID)
			c.Set("username", user.Username)
			c.Set("token_type", "user_token")

			roleNames, err := userSvc.GetUserRoles(ut.UserID)
			if err != nil {
				roleNames = []string{}
			}
			c.Set("roles", roleNames)
			c.Next()
			return
		}

		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			response.Unauthorized(c, "认证令牌无效")
			c.Abort()
			return
		}

		// 仅允许 access token 访问受保护资源，refresh token 仅用于 /refresh 端点
		if claims.TokenType != "access" {
			response.Unauthorized(c, "认证令牌类型无效")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("token_type", claims.TokenType)

		roleNames, err := userSvc.GetUserRoles(claims.UserID)
		if err != nil {
			roleNames = []string{}
		}
		c.Set("roles", roleNames)
		c.Next()
	}
}

// RefreshAuthMiddleware 专门用于 refresh token 端点，仅允许 refresh token
func RefreshAuthMiddleware(secret string, userSvc *service.UserService) gin.HandlerFunc {
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
			response.Unauthorized(c, "未提供认证令牌")
			c.Abort()
			return
		}

		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			response.Unauthorized(c, "认证令牌无效")
			c.Abort()
			return
		}

		// 仅允许 refresh token
		if claims.TokenType != "refresh" {
			response.Unauthorized(c, "请使用 refresh_token 刷新令牌")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("token_type", claims.TokenType)

		roleNames, err := userSvc.GetUserRoles(claims.UserID)
		if err != nil {
			roleNames = []string{}
		}
		c.Set("roles", roleNames)
		c.Next()
	}
}

func RequireRole(userSvc *service.UserService, roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Unauthorized(c, "未认证")
			c.Abort()
			return
		}

		userRoles, err := userSvc.GetUserRoles(userID.(uint))
		if err != nil {
			response.Forbidden(c, "无权限操作")
			c.Abort()
			return
		}

		hasRole := false
		for _, ur := range userRoles {
			for _, role := range roles {
				if ur == role {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			response.Forbidden(c, "无权限操作")
			c.Abort()
			return
		}

		c.Next()
	}
}
