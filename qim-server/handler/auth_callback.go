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

// findAuthProvider 查找并验证认证提供者
func findAuthProvider(c *gin.Context, providerName string) (*model.AuthProvider, bool) {
	db := database.GetDB()

	var authProvider model.AuthProvider
	if err := db.Where("name = ? AND type = ?", providerName, "redirect").First(&authProvider).Error; err != nil {
		response.NotFound(c, "认证提供者不存在")
		return nil, false
	}

	if !authProvider.Enabled {
		response.Forbidden(c, "该认证方式已禁用")
		return nil, false
	}

	return &authProvider, true
}

// buildAuthResponse 生成token并构建认证成功响应
func buildAuthResponse(c *gin.Context, user *model.User) {
	db := database.GetDB()

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
			"id":                 user.ID,
			"username":           user.Username,
			"nickname":           user.Nickname,
			"avatar":             user.Avatar,
			"signature":          user.Signature,
			"phone":              user.Phone,
			"email":              user.Email,
			"two_factor_enabled": user.TwoFactorEnabled,
			"roles":              roleNames,
		},
	})
}

// authenticateOAuth 执行OAuth认证流程，返回用户和是否成功
func authenticateOAuth(c *gin.Context, authProvider *model.AuthProvider, code string) (*model.User, bool) {
	db := database.GetDB()

	oauthProvider, err := provider.NewOAuthProvider(authProvider.Name, authProvider.Enabled, authProvider.Priority, authProvider.Config)
	if err != nil {
		response.InternalServerError(c, fmt.Sprintf("创建OAuth提供者失败: %v", err))
		return nil, false
	}

	authResult, err := oauthProvider.HandleCallback(context.Background(), code)
	if err != nil || !authResult.Success {
		msg := "认证成功处理失败"
		if authResult != nil {
			msg = authResult.Message
		}
		response.Unauthorized(c, msg)
		return nil, false
	}

	userInfo := authResult.UserInfo
	if userInfo == nil {
		response.InternalServerError(c, "获取用户信息失败")
		return nil, false
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
			return nil, false
		}
	} else {
		user.Status = "online"
		if name != "" {
			user.Nickname = name
		}
		db.Save(&user)
	}

	return &user, true
}

// authenticateCAS 执行CAS认证流程，返回用户和是否成功
func authenticateCAS(c *gin.Context, authProvider *model.AuthProvider, ticket string) (*model.User, bool) {
	db := database.GetDB()

	casProvider, err := provider.NewCASProvider(authProvider.Name, authProvider.Enabled, authProvider.Priority, authProvider.Config)
	if err != nil {
		response.InternalServerError(c, fmt.Sprintf("创建CAS提供者失败: %v", err))
		return nil, false
	}

	authResult, err := casProvider.ValidateTicket(context.Background(), ticket)
	if err != nil || !authResult.Success {
		msg := "CAS认证失败"
		if authResult != nil {
			msg = authResult.Message
		}
		response.Unauthorized(c, msg)
		return nil, false
	}

	userInfo := authResult.UserInfo
	if userInfo == nil {
		response.InternalServerError(c, "获取用户信息失败")
		return nil, false
	}

	username, _ := userInfo["username"].(string)
	if username == "" {
		response.Unauthorized(c, "未获取到用户名")
		return nil, false
	}

	var user model.User
	err = db.Where("username = ?", username).First(&user).Error
	if err != nil {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("cas_default_pass"), bcrypt.DefaultCost)
		user = model.User{
			Username:     username,
			Nickname:     username,
			PasswordHash: string(hashedPassword),
			Status:       "online",
		}
		if err := db.Create(&user).Error; err != nil {
			response.InternalServerError(c, "创建用户失败")
			return nil, false
		}
	} else {
		user.Status = "online"
		db.Save(&user)
	}

	return &user, true
}

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

	authProvider, ok := findAuthProvider(c, req.Provider)
	if !ok {
		return
	}

	user, ok := authenticateOAuth(c, authProvider, req.Code)
	if !ok {
		return
	}

	buildAuthResponse(c, user)
}

func CASCallback(c *gin.Context) {
	var req struct {
		Provider string `json:"provider" binding:"required"`
		Ticket   string `json:"ticket" binding:"required"`
		State    string `json:"state" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	authProvider, ok := findAuthProvider(c, req.Provider)
	if !ok {
		return
	}

	user, ok := authenticateCAS(c, authProvider, req.Ticket)
	if !ok {
		return
	}

	buildAuthResponse(c, user)
}

func UnifiedAuthCallback(c *gin.Context) {
	var req struct {
		Provider string `json:"provider" binding:"required"`
		Code     string `json:"code"`
		Ticket   string `json:"ticket"`
		State    string `json:"state"`
		Type     string `json:"type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	authProvider, ok := findAuthProvider(c, req.Provider)
	if !ok {
		return
	}

	var user *model.User

	switch req.Type {
	case "oauth":
		if req.Code == "" {
			response.BadRequest(c, "缺少授权码")
			return
		}
		user, ok = authenticateOAuth(c, authProvider, req.Code)
		if !ok {
			return
		}

	case "cas":
		if req.Ticket == "" {
			response.BadRequest(c, "缺少票据")
			return
		}
		user, ok = authenticateCAS(c, authProvider, req.Ticket)
		if !ok {
			return
		}

	default:
		response.BadRequest(c, "不支持的认证类型")
		return
	}

	buildAuthResponse(c, user)
}
