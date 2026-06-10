package middleware

import (
	"strings"

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

		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			response.Unauthorized(c, "认证令牌无效")
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
