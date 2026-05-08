package handler

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"qim-server/ai"
	"qim-server/config"
	"qim-server/database"
	"qim-server/di"
	"qim-server/middleware"
	"qim-server/model"
	"qim-server/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var cfg *config.Config
var aiService *ai.AIService

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

	db := database.GetDB()
	var user model.User
	if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		log.Printf("Login failed: user=%s, ip=%s, os=%s, version=%s, error=user not found", req.Username, ip, op, clientVersion)
		response.Unauthorized(c, "用户名或密码错误")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		log.Printf("Login failed: user=%s, ip=%s, os=%s, version=%s, error=invalid password", req.Username, ip, op, clientVersion)
		response.Unauthorized(c, "用户名或密码错误")
		return
	}

	if user.TwoFactorEnabled {
		response.Unauthorized(c, "需要双因素认证")
		return
	}

	token := generateToken(user.ID, user.Username)
	user.Status = "online"
	user.IP = ip
	db.Save(&user)

	log.Printf("Login success: user=%s, ip=%s, os=%s, version=%s", req.Username, ip, op, clientVersion)

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
	random := rand.Intn(10000)
	return fmt.Sprintf("%d%d", timestamp, random)
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

	db := database.GetDB()
	var user model.User
	if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	token := generateToken(user.ID, user.Username)
	user.Status = "online"
	db.Save(&user)

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

func CheckVersion(c *gin.Context) {
	var req struct {
		Version string `json:"version" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	latestVersion := "1.0.0"
	forceUpdate := true
	needUpdate := compareVersions(req.Version, latestVersion)

	response.Success(c, gin.H{
		"latest_version":  latestVersion,
		"current_version": req.Version,
		"need_update":     needUpdate,
		"force_update":    forceUpdate,
		"update_url":      "https://example.com/download/qim-latest",
		"release_notes":   "\n1. 修复了消息发送失败的问题\n2. 优化了系统性能\n3. 增加了版本更新提示功能\n",
	})
}

func compareVersions(current, latest string) bool {
	return current != latest
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
