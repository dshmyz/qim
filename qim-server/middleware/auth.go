package middleware

import (
	"net/http"
	"strings"

	"qim-server/database"
	"qim-server/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 首先从Authorization头获取token
		authHeader := c.GetHeader("Authorization")
		var tokenString string
		
		if authHeader != "" {
			// 从Authorization头获取
			parts := strings.SplitN(authHeader, " ", 2)
			if !(len(parts) == 2 && parts[0] == "Bearer") {
				c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "认证令牌格式错误"})
				c.Abort()
				return
			}
			tokenString = parts[1]
		} else {
			// 从URL参数获取（用于WebSocket连接）
			tokenString = c.Query("token")
			if tokenString == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未提供认证令牌"})
				c.Abort()
				return
			}
		}
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "认证令牌无效"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未认证"})
			c.Abort()
			return
		}
		
		var userRoles []model.UserRole
		if err := database.GetDB().Where("user_id = ?", userID).Find(&userRoles).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限操作"})
			c.Abort()
			return
		}
		
		hasRole := false
		for _, ur := range userRoles {
			for _, role := range roles {
				if ur.Role == role {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}
		
		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限操作"})
			c.Abort()
			return
		}
		
		c.Next()
	}
}
