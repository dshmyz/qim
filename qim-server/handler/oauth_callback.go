package handler

import (
	"context"
	"fmt"
	"qim-server/auth/provider"
	"qim-server/database"
	"qim-server/model"
	"qim-server/pkg/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func OAuthCallback(c *gin.Context) {
	var req struct {
		Provider string `json:"provider" binding:"required"`
		Code     string `json:"code" binding:"required"`
		State    string `json:"state" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	var authProvider model.AuthProvider
	if err := db.Where("name = ? AND type = ?", req.Provider, "redirect").First(&authProvider).Error; err != nil {
		response.NotFound(c, "认证提供者不存在")
		return
	}

	if !authProvider.Enabled {
		response.Forbidden(c, "该认证方式已禁用")
		return
	}

	oauthProvider, err := provider.NewOAuthProvider(authProvider.Name, authProvider.Enabled, authProvider.Priority, authProvider.Config)
	if err != nil {
		response.InternalServerError(c, fmt.Sprintf("创建OAuth提供者失败: %v", err))
		return
	}

	authResult, err := oauthProvider.HandleCallback(context.Background(), req.Code)
	if err != nil || !authResult.Success {
		msg := "认证成功处理失败"
		if authResult != nil {
			msg = authResult.Message
		}
		response.Unauthorized(c, msg)
		return
	}

	userInfo := authResult.UserInfo
	if userInfo == nil {
		response.InternalServerError(c, "获取用户信息失败")
		return
	}

	email, _ := userInfo["email"].(string)
	name, _ := userInfo["name"].(string)
	login, _ := userInfo["login"].(string)

	username := login
	if username == "" {
		username = name
	}
	if username == "" {
		username = fmt.Sprintf("oauth_user_%s", email)
	}

	var user model.User
	err = db.Where("email = ? OR username = ?", email, username).First(&user).Error
	if err != nil {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("oauth_default_pass"), bcrypt.DefaultCost)
		user = model.User{
			Username:     username,
			Email:        email,
			Nickname:     name,
			PasswordHash: string(hashedPassword),
			Status:       "online",
		}
		if err := db.Create(&user).Error; err != nil {
			response.InternalServerError(c, "创建用户失败")
			return
		}
	} else {
		user.Status = "online"
		if name != "" {
			user.Nickname = name
		}
		db.Save(&user)
	}

	tokenStr := generateToken(user.ID, user.Username)

	var userRoles []model.UserRole
	db.Where("user_id = ?", user.ID).Find(&userRoles)
	roleNames := make([]string, 0, len(userRoles))
	for _, ur := range userRoles {
		roleNames = append(roleNames, ur.Role)
	}

	response.Success(c, gin.H{
		"token": tokenStr,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"nickname": user.Nickname,
			"avatar":   user.Avatar,
			"email":    user.Email,
			"roles":    roleNames,
		},
	})
}
