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

	"qim-server/ai"
	"qim-server/auth"
	"qim-server/auth/provider"
	"qim-server/config"
	"qim-server/database"
	"qim-server/di"
	"qim-server/middleware"
	"qim-server/model"
	"qim-server/pkg/logger"
	"qim-server/pkg/response"

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
	aiService = ai.NewAIService(&c.AI)
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
		var mapping model.ExternalUserMapping
		err := db.Where("provider_name = ? AND external_user_id = ?", providerName, result.UserID).First(&mapping).Error
		if err == nil {
			if err := db.First(&user, mapping.UserID).Error; err != nil {
				logger.WithModule("Auth").Info("Login failed", "user", req.Username, "ip", ip, "os", op, "version", clientVersion, "error", "mapped user not found")
				response.Unauthorized(c, "用户不存在")
				return
			}
		} else {
			username := req.Username
			nickname := getStringFromUserInfo(result.UserInfo, "nickname", "username")
			email := getStringFromUserInfo(result.UserInfo, "email")
			phone := getStringFromUserInfo(result.UserInfo, "phone")
			avatar := getStringFromUserInfo(result.UserInfo, "avatar")

			tx := db.Begin()

			user = model.User{
				Username:     username,
				PasswordHash: "",
				Nickname:     nickname,
				Email:        email,
				Phone:        phone,
				Avatar:       avatar,
				Status:       "offline",
			}
			if err := tx.Where("username = ?", username).FirstOrCreate(&user).Error; err != nil {
				tx.Rollback()
				logger.WithModule("Auth").Error("Auto-create user failed", "username", username, "error", err)
				response.InternalServerError(c, "创建用户失败")
				return
			}

			mapping = model.ExternalUserMapping{
				UserID:         user.ID,
				ProviderName:   providerName,
				ExternalUserID: result.UserID,
			}
			if err := tx.Create(&mapping).Error; err != nil {
				tx.Rollback()
				logger.WithModule("Auth").Error("Create external mapping failed", "error", err)
				response.InternalServerError(c, "创建用户映射失败")
				return
			}

			if err := tx.Commit().Error; err != nil {
				tx.Rollback()
				response.InternalServerError(c, "保存用户信息失败")
				return
			}

			logger.WithModule("Auth").Info("External user auto-created", "username", username, "provider", providerName, "userID", user.ID)
		}
	}

	if (user.Type == "bot_assistant" || user.Type == "bot_avatar") || user.Type == "system" || user.Type == "api" {
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
		"token": token,
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
	user.Status = "online"
	db.Save(&user)

	logger.WithModule("Auth").Info("2FA verification success", "user", req.Username)

	response.Success(c, gin.H{
		"token": token,
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

	response.Success(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"nickname": user.Nickname,
			"avatar":   user.Avatar,
		},
	})
}

func RefreshToken(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	token := generateToken(userID.(uint), username.(string))

	response.Success(c, gin.H{
		"token": token,
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

	response.Success(c, gin.H{
		"message": "登出成功",
	})
}

func generateToken(userID uint, username string) string {
	claims := middleware.Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.JWT.Expire) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(cfg.JWT.Secret))
	return tokenString
}
