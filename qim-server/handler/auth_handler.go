package handler

import (
	"context"
	cr "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	mrand "math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/auth"
	"github.com/dshmyz/qim/qim-server/auth/provider"
	"github.com/dshmyz/qim/qim-server/config"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/middleware"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var cfg *config.Config
var aiService *ai.AIService

// twoFASession 存储 2FA 验证会话
type twoFASession struct {
	Code      string
	Username  string
	ExpiresAt time.Time
}

var (
	twoFASessions   = make(map[string]twoFASession)
	twoFASessionsMu sync.Mutex
)

func init() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			twoFASessionsMu.Lock()
			now := time.Now()
			for id, s := range twoFASessions {
				if now.After(s.ExpiresAt) {
					delete(twoFASessions, id)
				}
			}
			twoFASessionsMu.Unlock()
		}
	}()
}

func generateTwoFASession(username string, code string) string {
	b := make([]byte, 16)
	cr.Read(b)
	sessionID := hex.EncodeToString(b)

	twoFASessionsMu.Lock()
	twoFASessions[sessionID] = twoFASession{
		Code:      code,
		Username:  username,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	twoFASessionsMu.Unlock()
	return sessionID
}

func generateTwoFACode() string {
	n, _ := cr.Int(cr.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", n.Int64())
}

func SetConfig(c *config.Config) {
	cfg = c
	// 复用 DI 容器中已从数据库加载 Provider 的 AIService，避免重新创建导致丢失 DB Provider
	if di.GlobalContainer != nil && di.GlobalContainer.AIService != nil {
		aiService = di.GlobalContainer.AIService
	} else {
		aiService = ai.NewAIService(&c.AI)
		// GlobalContainer 尚未初始化时（理论上不应发生），先创建一个空容器
		if di.GlobalContainer == nil {
			di.GlobalContainer = &di.Container{}
		}
	}
	di.GlobalContainer.Config = c
	di.GlobalContainer.AIService = aiService

	if di.GlobalContainer.MessageService != nil {
		di.GlobalContainer.MessageService.SetAIService(aiService)
	}
	if di.GlobalContainer.AvatarService != nil {
		di.GlobalContainer.AvatarService.SetAIService(aiService)
	}
}

// InitWSHandlers 注册 WebSocket 消息处理回调，统一使用 MessageService
func InitWSHandlers() {
	if di.GlobalContainer.WebSocketHub != nil && di.GlobalContainer.MessageService != nil {
		msgSvc := di.GlobalContainer.MessageService
		di.GlobalContainer.WebSocketHub.HandleMessage = msgSvc.SendMessage
		di.GlobalContainer.WebSocketHub.HandleReadMessage = msgSvc.MarkAsRead
		logger.WithModule("Init").Info("WS HandleMessage / HandleReadMessage 已注册到 MessageService")
	}
}

func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Version  string `json:"version"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	op := userAgent
	clientVersion := req.Version

	authChain := auth.GetAuthChain()
	if authChain == nil {
		response.InternalServerError(c, "认证服务未初始化")
		return
	}

	creds := &provider.Credentials{
		Username: req.Username,
		Password: req.Password,
	}

	result, providerName, err := authChain.AuthenticateDirect(context.Background(), creds)
	if err != nil {
		logger.WithModule("Auth").Error("Auth chain error", "error", err)
		response.InternalServerError(c, "认证失败")
		return
	}

	if !result.Success {
		logger.WithModule("Auth").Info("Login failed", "user", req.Username, "ip", ip, "os", op, "version", clientVersion, "error", result.Message)
		middleware.RecordLoginFailure(ip)
		response.Unauthorized(c, result.Message)
		return
	}

	db := database.GetDB()
	var user model.User

	if providerName == "local" {
		if err := db.First(&user, result.UserID).Error; err != nil {
			logger.WithModule("Auth").Info("Login failed", "user", req.Username, "ip", ip, "os", op, "version", clientVersion, "error", "user not found after auth")
			response.Unauthorized(c, "用户不存在")
			return
		}
	} else {
		username := req.Username
		if err := db.Where("username = ?", username).First(&user).Error; err != nil {
			nickname := getStringFromUserInfo(result.UserInfo, "nickname", "username")
			email := getStringFromUserInfo(result.UserInfo, "email")
			phone := getStringFromUserInfo(result.UserInfo, "phone")
			avatar := getStringFromUserInfo(result.UserInfo, "avatar")

			user = model.User{
				Username:     username,
				PasswordHash: "",
				Nickname:     nickname,
				Email:        email,
				Phone:        phone,
				Avatar:       avatar,
				Status:       "offline",
			}
			if err := db.Create(&user).Error; err != nil {
				logger.WithModule("Auth").Error("Auto-create user failed", "username", username, "error", err)
				response.InternalServerError(c, "创建用户失败")
				return
			}
			logger.WithModule("Auth").Info("External user auto-created", "username", username, "provider", providerName, "userID", user.ID)
		}
	}

	if user.Type == "bot" || user.Type == "system" || user.Type == "api" {
		logger.WithModule("Auth").Info("Login blocked", "user", req.Username, "type", user.Type, "ip", ip, "reason", "非用户账户禁止登录")
		response.Forbidden(c, "该账户类型不支持登录")
		return
	}

	if user.TwoFactorEnabled {
		code := generateTwoFACode()
		sessionID := generateTwoFASession(req.Username, code)
		logger.WithModule("Auth").Info("2FA required", "user", req.Username, "session", sessionID)
		response.ErrorWithDetail(c, http.StatusUnauthorized, 1002, "需要双因素认证", gin.H{
			"two_factor_required": true,
			"session":             sessionID,
		})
		return
	}

	token := generateToken(user.ID, user.Username)
	refreshToken := generateRefreshToken(user.ID, user.Username)
	user.Status = "online"
	user.IP = ip
	db.Save(&user)

	logger.WithModule("Auth").Info("Login success", "user", req.Username, "ip", ip, "os", op, "version", clientVersion, "provider", providerName)

	var userRoles []model.UserRole
	db.Where("user_id = ?", user.ID).Find(&userRoles)
	roleNames := make([]string, 0, len(userRoles))
	for _, ur := range userRoles {
		roleNames = append(roleNames, ur.Role)
	}

	response.Success(c, gin.H{
		"token":         token,
		"refresh_token": refreshToken,
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

func parseOS(userAgent string) string {
	if strings.Contains(userAgent, "Windows") {
		return "Windows"
	} else if strings.Contains(userAgent, "Macintosh") {
		return "macOS"
	} else if strings.Contains(userAgent, "Linux") {
		return "Linux"
	} else if strings.Contains(userAgent, "Android") {
		return "Android"
	} else if strings.Contains(userAgent, "iPhone") || strings.Contains(userAgent, "iPad") {
		return "iOS"
	} else {
		return "Unknown"
	}
}

func generateSessionID() string {
	timestamp := time.Now().UnixNano()
	random := mrand.Intn(10000)
	return fmt.Sprintf("%d%d", timestamp, random)
}

func getStringFromUserInfo(userInfo map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v, ok := userInfo[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

func VerifyTwoFA(c *gin.Context) {
	var req struct {
		Session  string `json:"session" binding:"required"`
		Code     string `json:"code" binding:"required"`
		Username string `json:"username" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	// 验证 2FA 会话
	twoFASessionsMu.Lock()
	session, exists := twoFASessions[req.Session]
	if exists {
		delete(twoFASessions, req.Session)
	}
	twoFASessionsMu.Unlock()

	if !exists {
		response.Unauthorized(c, "验证会话不存在或已过期")
		return
	}

	if time.Now().After(session.ExpiresAt) {
		response.Unauthorized(c, "验证会话已过期，请重新登录")
		return
	}

	if session.Code != req.Code {
		logger.WithModule("Auth").Warn("2FA code mismatch", "user", req.Username)
		response.Unauthorized(c, "验证码错误")
		return
	}

	if session.Username != req.Username {
		response.Unauthorized(c, "会话用户名不匹配")
		return
	}

	db := database.GetDB()
	var user model.User
	if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	token := generateToken(user.ID, user.Username)
	refreshToken := generateRefreshToken(user.ID, user.Username)
	user.Status = "online"
	db.Save(&user)

	logger.WithModule("Auth").Info("2FA verification success", "user", req.Username)

	response.Success(c, gin.H{
		"token":         token,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":                 user.ID,
			"username":           user.Username,
			"nickname":           user.Nickname,
			"avatar":             user.Avatar,
			"signature":          user.Signature,
			"phone":              user.Phone,
			"email":              user.Email,
			"two_factor_enabled": user.TwoFactorEnabled,
		},
	})
}

func CheckTwoFAStatus(c *gin.Context) {
	// 系统级 2FA 开关
	if cfg, err := di.GlobalContainer.SystemConfigService.GetConfig("enable2FA"); err == nil && cfg.Value == "false" {
		response.Success(c, gin.H{
			"twoFactorEnabled": false,
		})
		return
	}

	var req struct {
		Username string `json:"username" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()
	var user model.User
	if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		response.Success(c, gin.H{
			"twoFactorEnabled": false,
		})
		return
	}

	response.Success(c, gin.H{
		"twoFactorEnabled": user.TwoFactorEnabled,
	})
}

func Register(c *gin.Context) {
	// 检查是否允许注册
	if cfg, err := di.GlobalContainer.SystemConfigService.GetConfig("enableRegistration"); err == nil && cfg.Value == "false" {
		response.Forbidden(c, "注册功能已关闭")
		return
	}

	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Nickname string `json:"nickname"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	// 密码强度校验
	if err := validatePassword(req.Password); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	db := database.GetDB()

	var count int64
	db.Model(&model.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		response.BadRequest(c, "用户名已存在")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.InternalServerError(c, "密码加密失败")
		return
	}

	nickname := req.Nickname
	if nickname == "" {
		nickname = req.Username
	}

	user := model.User{
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		Nickname:     nickname,
		Avatar:       "",
		Status:       "online",
	}

	tx := db.Begin()
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "注册失败")
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "注册失败")
		return
	}

	token := generateToken(user.ID, user.Username)
	refreshToken := generateRefreshToken(user.ID, user.Username)

	response.Success(c, gin.H{
		"token":         token,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"nickname": user.Nickname,
			"avatar":   user.Avatar,
		},
	})
}

func RefreshToken(c *gin.Context) {
	// 从 context 获取认证信息（由 AuthMiddleware 设置）
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	// 验证当前 token 是否为 refresh token
	tokenType, _ := c.Get("token_type")
	if tokenType != "refresh" {
		response.Unauthorized(c, "无效的刷新令牌，请使用 refresh_token")
		c.Abort()
		return
	}

	// 生成新的 access token
	token := generateAccessToken(userID.(uint), username.(string))
	// 生成新的 refresh token
	refreshToken := generateRefreshToken(userID.(uint), username.(string))

	response.Success(c, gin.H{
		"token":         token,
		"refresh_token": refreshToken,
	})
}

func Logout(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()
	var user model.User
	if err := db.First(&user, userID).Error; err == nil {
		user.Status = "offline"
		db.Save(&user)
	}

	result := gin.H{
		"message": "登出成功",
	}

	var authProviders []model.AuthProvider
	db.Where("enabled = ? AND type = ?", true, "redirect").Find(&authProviders)

	logoutURLs := gin.H{}
	for _, ap := range authProviders {
		switch ap.Protocol {
		case model.AuthProviderProtocolCAS:
			casProvider, err := provider.NewCASProvider(ap.Name, ap.Enabled, ap.Priority, ap.Config)
			if err == nil {
				logoutURLs[ap.Name] = casProvider.GetLogoutURL()
			}
		case model.AuthProviderProtocolOAuth:
			oauthProvider, err := provider.NewOAuthProvider(ap.Name, ap.Enabled, ap.Priority, ap.Config)
			if err == nil && oauthProvider.GetConfig().RevokeURL != "" {
				logoutURLs[ap.Name] = oauthProvider.GetConfig().RevokeURL
			}
		}
	}

	if len(logoutURLs) > 0 {
		result["logout_urls"] = logoutURLs
	}

	response.Success(c, result)
}

func RefreshOAuthToken(c *gin.Context) {
	var req struct {
		ProviderName string `json:"provider_name" binding:"required"`
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()
	var authProvider model.AuthProvider
	if err := db.Where("name = ? AND enabled = ? AND protocol = ?", req.ProviderName, true, model.AuthProviderProtocolOAuth).First(&authProvider).Error; err != nil {
		response.NotFound(c, "OAuth认证提供者不存在或未启用")
		return
	}

	oauthProvider, err := provider.NewOAuthProvider(authProvider.Name, authProvider.Enabled, authProvider.Priority, authProvider.Config)
	if err != nil {
		response.InternalServerError(c, "创建OAuth提供者失败")
		return
	}

	token, err := oauthProvider.RefreshToken(context.Background(), req.RefreshToken)
	if err != nil {
		response.BadRequest(c, "刷新令牌失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"access_token":  token.AccessToken,
		"token_type":    token.TokenType,
		"expires_in":    token.Expiry.Unix() - time.Now().Unix(),
		"refresh_token": token.RefreshToken,
	})
}

func generateToken(userID uint, username string) string {
	return generateAccessToken(userID, username)
}

func generateAccessToken(userID uint, username string) string {
	claims := middleware.Claims{
		UserID:    userID,
		Username:  username,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.JWT.Expire) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.JWT.Secret))
	if err != nil {
		logger.WithModule("Auth").Error("Failed to sign JWT access token", "error", err)
		return ""
	}
	return tokenString
}

func generateRefreshToken(userID uint, username string) string {
	refreshDays := cfg.JWT.RefreshExpireDays
	if refreshDays <= 0 {
		refreshDays = 7 // 默认 7 天
	}
	claims := middleware.Claims{
		UserID:    userID,
		Username:  username,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().AddDate(0, 0, refreshDays)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.JWT.Secret))
	if err != nil {
		logger.WithModule("Auth").Error("Failed to sign JWT refresh token", "error", err)
		return ""
	}
	return tokenString
}

// validatePassword 校验密码强度：至少 8 位，包含字母和数字
func validatePassword(password string) error {
	if len(password) == 0 {
		return fmt.Errorf("密码不能为空")
	}
	if len(password) < 8 {
		return fmt.Errorf("密码长度不能少于8位")
	}
	hasLetter := false
	hasDigit := false
	for _, r := range password {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			hasLetter = true
		}
		if r >= '0' && r <= '9' {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return fmt.Errorf("密码必须包含字母和数字")
	}
	return nil
}
