package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dshmyz/qim/qim-server/auth/provider"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"

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

	mapping := getAttributeMapping(authProvider)

	username := getStringFromUserInfo(userInfo, "login", "name")
	if username == "" {
		email := getStringFromUserInfo(userInfo, mapping["email"], "email")
		username = fmt.Sprintf("oauth_user_%s", email)
	}

	email := getStringFromUserInfo(userInfo, mapping["email"], "email")
	nickname := getStringFromUserInfo(userInfo, mapping["nickname"], "nickname", "name")
	phone := getStringFromUserInfo(userInfo, mapping["phone"], "phone")
	avatar := getStringFromUserInfo(userInfo, mapping["avatar"], "avatar", "picture")

	var user model.User
	err = db.Where("username = ?", username).First(&user).Error
	if err != nil {
		if email != "" {
			err = db.Where("email = ?", email).First(&user).Error
		}
	}

	if err != nil {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("oauth_default_pass"), bcrypt.DefaultCost)
		user = model.User{
			Username:     username,
			Email:        email,
			Nickname:     nickname,
			Phone:        phone,
			Avatar:       avatar,
			PasswordHash: string(hashedPassword),
			Status:       "online",
		}
		if user.Nickname == "" {
			user.Nickname = user.Username
		}
		if err := db.Create(&user).Error; err != nil {
			response.InternalServerError(c, "创建用户失败")
			return nil, false
		}
	} else {
		updates := buildUserUpdates(userInfo, authProvider)
		db.Model(&user).Updates(updates)
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

	// 获取字段映射
	mapping := getAttributeMapping(authProvider)

	// 通过映射提取其他字段
	nickname := getStringFromUserInfo(userInfo, mapping["nickname"], "nickname", "displayName")
	email := getStringFromUserInfo(userInfo, mapping["email"], "email", "mail")
	phone := getStringFromUserInfo(userInfo, mapping["phone"], "phone")
	avatar := getStringFromUserInfo(userInfo, mapping["avatar"], "avatar")

	var user model.User
	err = db.Where("username = ?", username).First(&user).Error
	if err != nil {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("cas_default_pass"), bcrypt.DefaultCost)
		user = model.User{
			Username:     username,
			Nickname:     nickname,
			Email:        email,
			Phone:        phone,
			Avatar:       avatar,
			PasswordHash: string(hashedPassword),
			Status:       "online",
		}
		if user.Nickname == "" {
			user.Nickname = user.Username
		}
		if err := db.Create(&user).Error; err != nil {
			response.InternalServerError(c, "创建用户失败")
			return nil, false
		}
	} else {
		// 根据属性映射更新用户字段
		updates := buildUserUpdates(userInfo, authProvider)
		db.Model(&user).Updates(updates)
	}

	return &user, true
}

func findCASProvider(c *gin.Context) (*model.AuthProvider, bool) {
	db := database.GetDB()

	var authProvider model.AuthProvider
	if err := db.Where("protocol = ? AND enabled = ?", model.AuthProviderProtocolCAS, true).First(&authProvider).Error; err != nil {
		response.NotFound(c, "未找到CAS认证提供者")
		return nil, false
	}

	return &authProvider, true
}

// getAttributeMapping 从提供者配置中解析属性映射
func getAttributeMapping(authProvider *model.AuthProvider) map[string]string {
	defaultMapping := map[string]string{
		"nickname": "name",
		"email":    "email",
		"phone":    "phone",
		"avatar":   "picture",
	}

	switch authProvider.Protocol {
	case model.AuthProviderProtocolOAuth:
		var cfg provider.OAuthConfig
		if err := json.Unmarshal([]byte(authProvider.Config), &cfg); err == nil && cfg.AttributeMapping != nil {
			for k, v := range cfg.AttributeMapping {
				defaultMapping[k] = v
			}
		}
	case model.AuthProviderProtocolCAS:
		var cfg provider.CASConfig
		if err := json.Unmarshal([]byte(authProvider.Config), &cfg); err == nil && cfg.AttributeMapping != nil {
			for k, v := range cfg.AttributeMapping {
				defaultMapping[k] = v
			}
		}
	}

	return defaultMapping
}

// buildUserUpdates 根据属性映射构建用户更新字段
func buildUserUpdates(userInfo map[string]interface{}, authProvider *model.AuthProvider) map[string]interface{} {
	mapping := getAttributeMapping(authProvider)
	updates := make(map[string]interface{})

	for localField, remoteField := range mapping {
		if localField == "username" {
			continue // username 不允许更新
		}
		if val, ok := userInfo[remoteField]; ok && val != nil {
			if strVal, ok := val.(string); ok && strVal != "" {
				updates[localField] = strVal
			}
		}
	}

	updates["status"] = "online"
	return updates
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
