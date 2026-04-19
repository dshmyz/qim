package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"qim-server/database"
	"qim-server/middleware"
	"qim-server/model"
	"qim-server/ws"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 登录
func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Version  string `json:"version"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 获取用户IP
	ip := c.ClientIP()

	// 获取User-Agent
	userAgent := c.GetHeader("User-Agent")

	// 解析操作系统信息
	// op := parseOS(userAgent)
	op := userAgent

	// 获取客户端版本号
	clientVersion := req.Version

	db := database.GetDB()
	var user model.User
	if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		// 登录失败日志
		log.Printf("Login failed: user=%s, ip=%s, os=%s, version=%s, error=user not found", req.Username, ip, op, clientVersion)
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户名或密码错误"})
		return
	}

	// 暂时跳过密码验证，方便测试
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		// 登录失败日志
		log.Printf("Login failed: user=%s, ip=%s, os=%s, version=%s, error=invalid password", req.Username, ip, op, clientVersion)
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户名或密码错误"})
		return
	}

	// 检查用户是否启用了双因素认证
	if user.TwoFactorEnabled {
		// 生成登录会话ID
		sessionID := generateSessionID()

		// 这里可以将会话ID存储到数据库或缓存中，关联用户ID
		// 暂时简单处理，直接返回会话ID

		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "需要双因素认证",
			"data": gin.H{
				"session": sessionID,
			},
		})
		return
	}

	// 生成JWT
	token := generateToken(user.ID, user.Username)

	// 更新用户状态
	user.Status = "online"
	db.Save(&user)

	// 登录成功日志
	log.Printf("Login success: user=%s, ip=%s, os=%s, version=%s", req.Username, ip, op, clientVersion)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
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
		},
	})
}

// 解析操作系统信息
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

// 生成会话ID
func generateSessionID() string {
	timestamp := time.Now().UnixNano()
	random := rand.Intn(10000)
	return fmt.Sprintf("%d%d", timestamp, random)
}

// 验证双因素认证
func VerifyTwoFA(c *gin.Context) {
	var req struct {
		Session  string `json:"session" binding:"required"`
		Code     string `json:"code" binding:"required"`
		Username string `json:"username" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 这里应该验证会话ID和验证码
	// 暂时简单处理，直接生成token
	// 实际应用中，应该：
	// 1. 验证会话ID是否有效
	// 2. 验证验证码是否正确
	// 3. 生成JWT token

	// 假设验证成功，生成token
	// 根据用户名获取用户信息
	db := database.GetDB()
	var user model.User
	if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	token := generateToken(user.ID, user.Username)

	// 更新用户状态
	user.Status = "online"
	db.Save(&user)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
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
		},
	})
}

// 检查用户双因素认证状态
func CheckTwoFAStatus(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	var user model.User
	if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		// 用户不存在，返回默认状态
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"twoFactorEnabled": false,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"twoFactorEnabled": user.TwoFactorEnabled,
		},
	})
}

// 注册
func Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Nickname string `json:"nickname"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	// 检查用户名是否存在
	var count int64
	db.Model(&model.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户名已存在"})
		return
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "密码加密失败"})
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
		Avatar:       "https://api.dicebear.com/7.x/avataaars/svg?seed=" + req.Username,
		Status:       "online",
	}

	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "注册失败"})
		return
	}

	// 生成JWT
	token := generateToken(user.ID, user.Username)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"token": token,
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"nickname": user.Nickname,
				"avatar":   user.Avatar,
			},
		},
	})
}

// 刷新Token
func RefreshToken(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	// 生成新的JWT
	token := generateToken(userID.(uint), username.(string))

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"token": token,
		},
	})
}

// 登出
func Logout(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()
	// 更新用户状态为离线
	var user model.User
	if err := db.First(&user, userID).Error; err == nil {
		user.Status = "offline"
		db.Save(&user)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "登出成功",
	})
}

// 检查版本
func CheckVersion(c *gin.Context) {
	var req struct {
		Version string `json:"version" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 当前最新版本
	latestVersion := "1.0.0"
	// 是否强制更新
	forceUpdate := true

	// 版本比较逻辑
	needUpdate := compareVersions(req.Version, latestVersion)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"latest_version":  latestVersion,
			"current_version": req.Version,
			"need_update":     needUpdate,
			"force_update":    forceUpdate,
			"update_url":      "https://example.com/download/qim-latest",
			"release_notes":   "\n1. 修复了消息发送失败的问题\n2. 优化了系统性能\n3. 增加了版本更新提示功能\n",
		},
	})
}

// 比较版本号
func compareVersions(current, latest string) bool {
	// 简单的版本比较逻辑
	// 实际项目中应该使用更完善的版本比较库
	return current != latest
}

// 获取当前用户
func GetCurrentUser(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()
	var user model.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":                 user.ID,
			"username":           user.Username,
			"nickname":           user.Nickname,
			"avatar":             user.Avatar,
			"signature":          user.Signature,
			"phone":              user.Phone,
			"email":              user.Email,
			"status":             user.Status,
			"two_factor_enabled": user.TwoFactorEnabled,
		},
	})
}

// 更新用户
func UpdateUser(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Nickname         string `json:"nickname"`
		Avatar           string `json:"avatar"`
		Signature        string `json:"signature"`
		Phone            string `json:"phone"`
		Email            string `json:"email"`
		TwoFactorEnabled *bool  `json:"two_factor_enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	var user model.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	if req.Signature != "" {
		user.Signature = req.Signature
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.TwoFactorEnabled != nil {
		user.TwoFactorEnabled = *req.TwoFactorEnabled
	}

	db.Save(&user)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": user,
	})
}

// 获取组织架构树
func GetOrganizationTree(c *gin.Context) {
	db := database.GetDB()

	// 查询所有部门
	var departments []model.Department
	if err := db.Where("parent_id IS NULL").Find(&departments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询失败"})
		return
	}

	// 递归加载子部门和员工
	for i := range departments {
		loadDepartmentChildren(&departments[i], db)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": departments,
	})
}

// 创建部门
func CreateDepartment(c *gin.Context) {
	// 不需要使用userID，因为创建部门不需要用户信息

	var req struct {
		Name      string `json:"name" binding:"required"`
		ParentID  *uint  `json:"parent_id"`
		Level     int    `json:"level" binding:"required"`
		Path      string `json:"path"`
		SortOrder int    `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	department := model.Department{
		Name:      req.Name,
		ParentID:  req.ParentID,
		Level:     req.Level,
		Path:      req.Path,
		SortOrder: req.SortOrder,
	}

	if err := db.Create(&department).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建部门失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": department,
	})
}

// 创建用户
func CreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	// 检查用户名是否存在
	var count int64
	db.Model(&model.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户名已存在"})
		return
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "密码加密失败"})
		return
	}

	nickname := req.Nickname
	if nickname == "" {
		nickname = req.Username
	}

	avatar := req.Avatar
	if avatar == "" {
		avatar = "https://api.dicebear.com/7.x/avataaars/svg?seed=" + req.Username
	}

	user := model.User{
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		Nickname:     nickname,
		Avatar:       avatar,
		Status:       "online",
	}

	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": user,
	})
}

// 关联用户和部门
func AddUserToDepartment(c *gin.Context) {
	var req struct {
		UserID       uint   `json:"user_id" binding:"required"`
		DepartmentID uint   `json:"department_id" binding:"required"`
		Position     string `json:"position"`
		IsPrimary    bool   `json:"is_primary"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	// 检查用户是否存在
	var user model.User
	if err := db.First(&user, req.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	// 检查部门是否存在
	var department model.Department
	if err := db.First(&department, req.DepartmentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "部门不存在"})
		return
	}

	departmentEmployee := model.DepartmentEmployee{
		UserID:       req.UserID,
		DepartmentID: req.DepartmentID,
		Position:     req.Position,
		IsPrimary:    req.IsPrimary,
	}

	if err := db.Create(&departmentEmployee).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "关联用户和部门失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": departmentEmployee,
	})
}

func loadDepartmentChildren(dept *model.Department, db *gorm.DB) {
	// 加载子部门
	db.Where("parent_id = ?", dept.ID).Find(&dept.SubDepartments)
	for i := range dept.SubDepartments {
		loadDepartmentChildren(&dept.SubDepartments[i], db)
	}

	// 加载员工
	var deptEmps []model.DepartmentEmployee
	db.Where("department_id = ?", dept.ID).Preload("User").Find(&deptEmps)
	for _, de := range deptEmps {
		dept.Employees = append(dept.Employees, de.User)
	}
}

// 获取会话列表
func GetConversations(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()
	var convMembers []model.ConversationMember
	db.Where("user_id = ?", userID).Preload("Conversation").Preload("Conversation.LastMessage").Preload("Conversation.Members").Preload("Conversation.Members.User").Find(&convMembers)

	// 构建包含置顶状态和IP信息的会话列表
	type ConversationWithPin struct {
		model.Conversation
		IsPinned bool   `json:"is_pinned"`
		IP       string `json:"ip,omitempty"`
	}

	var conversations []ConversationWithPin
	for _, cm := range convMembers {
		// 查找会话会话记录
		var session model.ConversationSession
		db.Where("user_id = ? AND conversation_id = ?", userID, cm.Conversation.ID).FirstOrCreate(&session, model.ConversationSession{
			IsPinned:      false,
			LastVisitedAt: time.Now(),
		})

		// 构建包含置顶状态的会话
		convWithPin := ConversationWithPin{
			Conversation: cm.Conversation,
			IsPinned:     session.IsPinned,
		}

		// 对于单聊会话，获取对方用户的IP
		if cm.Conversation.Type == "single" && len(cm.Conversation.Members) > 0 {
			for _, member := range cm.Conversation.Members {
				if member.UserID != userID.(uint) {
					convWithPin.IP = member.User.IP
					break
				}
			}
		}

		conversations = append(conversations, convWithPin)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": conversations,
	})
}

// 创建单聊会话
func CreateSingleConversation(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		UserID uint `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	// 检查是否已存在单聊会话
	var existingConv model.Conversation
	db.Raw(`
		SELECT c.* FROM conversations c
		JOIN conversation_members cm1 ON c.id = cm1.conversation_id
		JOIN conversation_members cm2 ON c.id = cm2.conversation_id
		WHERE c.type = 'single'
		AND cm1.user_id = ? AND cm2.user_id = ?
	`, userID, req.UserID).Scan(&existingConv)

	if existingConv.ID > 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": existingConv})
		return
	}

	// 创建新会话
	var targetUser model.User
	db.First(&targetUser, req.UserID)

	conv := model.Conversation{
		Type:   "single",
		Name:   targetUser.Nickname,
		Avatar: targetUser.Avatar,
	}
	db.Create(&conv)

	// 添加成员
	db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID.(uint), Role: "member"})
	db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: req.UserID, Role: "member"})

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": conv})
}

// 创建群聊会话
func CreateGroupConversation(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Name      string `json:"name" binding:"required"`
		Avatar    string `json:"avatar"`
		MemberIDs []uint `json:"member_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	conv := model.Conversation{
		Type:      "group",
		Name:      req.Name,
		Avatar:    req.Avatar,
		CreatorID: userID.(uint),
	}
	db.Create(&conv)

	// 添加创建者
	db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID.(uint), Role: "owner"})

	// 添加其他成员
	for _, mid := range req.MemberIDs {
		if mid != userID.(uint) {
			db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: mid, Role: "member"})
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": conv})
}

// 创建讨论组会话
func CreateDiscussionConversation(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Name      string `json:"name" binding:"required"`
		Avatar    string `json:"avatar"`
		MemberIDs []uint `json:"member_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	conv := model.Conversation{
		Type:      "discussion",
		Name:      req.Name,
		Avatar:    req.Avatar,
		CreatorID: userID.(uint),
	}
	db.Create(&conv)

	// 添加创建者
	db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID.(uint), Role: "member"})

	// 添加其他成员
	for _, mid := range req.MemberIDs {
		if mid != userID.(uint) {
			db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: mid, Role: "member"})
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": conv})
}

// 获取会话详情
func GetConversation(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convID := c.Param("id")

	// 处理带有 conv_ 前缀的会话ID
	if strings.HasPrefix(convID, "conv_") {
		convID = strings.TrimPrefix(convID, "conv_")
	}

	db := database.GetDB()
	var conv model.Conversation
	if err := db.Preload("Members").Preload("Members.User").First(&conv, convID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	// 验证是否为会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", conv.ID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限访问"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": conv})
}

// 获取历史消息
func GetMessages(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convID := c.Param("id")

	// 处理带有 conv_ 前缀的会话ID
	if strings.HasPrefix(convID, "conv_") {
		convID = strings.TrimPrefix(convID, "conv_")
	}

	// 解析分页参数
	pageStr := c.Query("page")
	pageSizeStr := c.Query("page_size")

	page := 1
	pageSize := 20

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	offset := (page - 1) * pageSize

	db := database.GetDB()

	// 验证是否为会话成员或曾经是会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, userID).First(&member).Error; err != nil {
		// 检查是否曾经是会话成员（通过消息记录判断）
		var count int64
		db.Model(&model.Message{}).Where("conversation_id = ? AND sender_id = ?", convID, userID).Count(&count)
		if count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限访问"})
			return
		}
	}

	// 获取消息总数
	var total int64
	db.Model(&model.Message{}).Where("conversation_id = ?", convID).Count(&total)

	// 计算总页数
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	// 获取消息
	var messages []model.Message
	db.Where("conversation_id = ?", convID).Preload("Sender").Preload("QuotedMessage").Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&messages)

	// 为每条消息预加载引用消息的发送者
	for i := range messages {
		if messages[i].QuotedMessage != nil {
			db.Model(&messages[i].QuotedMessage).Association("Sender").Find(&messages[i].QuotedMessage.Sender)
		}
	}

	// 反转顺序（最新的在最后）
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	// 处理分享数据，将JSON字符串转换回对象
	var responseMessages []gin.H
	for _, msg := range messages {
		var shareData map[string]interface{}
		// 处理新格式：从content字段读取
		if msg.Type == "share" && msg.Content != "" {
			json.Unmarshal([]byte(msg.Content), &shareData)
		}

		responseMsg := gin.H{
			"id":                msg.ID,
			"conversation_id":   msg.ConversationID,
			"sender_id":         msg.SenderID,
			"type":              msg.Type,
			"content":           msg.Content,
			"file_name":         msg.FileName,
			"file_size":         msg.FileSize,
			"quoted_message_id": msg.QuotedMessageID,
			"is_recalled":       msg.IsRecalled,
			"is_read":           msg.IsRead,
			"recalled_at":       msg.RecalledAt,
			"created_at":        msg.CreatedAt,
			"sender":            msg.Sender,
			"quoted_message":    msg.QuotedMessage,
			"share_data":        shareData,
		}
		responseMessages = append(responseMessages, responseMsg)
	}

	// 返回分页信息
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": responseMessages,
		"pagination": gin.H{
			"current_page": page,
			"page_size":    pageSize,
			"total":        total,
			"total_pages":  totalPages,
		},
	})
}

// 按筛选条件获取消息
func GetMessagesByFilter(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convID := c.Query("conversation_id")
	messageType := c.Query("type")
	pageStr := c.Query("page")
	pageSizeStr := c.Query("page_size")
	search := c.Query("search")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// 验证会话ID
	if convID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "会话ID不能为空"})
		return
	}

	// 验证是否为会话成员
	db := database.GetDB()
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限访问"})
		return
	}

	// 解析分页参数
	page := 1
	pageSize := 10
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}
	offset := (page - 1) * pageSize

	// 构建查询
	query := db.Where("conversation_id = ?", convID)

	// 按类型筛选
	if messageType != "" {
		query = query.Where("type = ?", messageType)
	}

	// 按搜索关键词筛选
	if search != "" {
		query = query.Where("content LIKE ?", "%"+search+"%")
	}

	// 按日期范围筛选
	if startDate != "" {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate != "" {
		// 如果endDate只包含日期部分，将其解释为当天的23:59:59
		query = query.Where("created_at <= ?", endDate+" 23:59:59")
	}

	// 获取总数
	var total int64
	query.Model(&model.Message{}).Count(&total)

	// 获取消息
	var messages []model.Message
	query.Preload("Sender").Preload("QuotedMessage").Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&messages)

	// 为每条消息预加载引用消息的发送者
	for i := range messages {
		if messages[i].QuotedMessage != nil {
			db.Model(&messages[i].QuotedMessage).Association("Sender").Find(&messages[i].QuotedMessage.Sender)
		}
	}

	// 反转顺序（最新的在最后）
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": map[string]interface{}{
			"messages": messages,
			"total":    total,
		},
	})
}

// 发送消息
func SendMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convID := c.Param("id")

	// 处理带有 conv_ 前缀的会话ID
	if strings.HasPrefix(convID, "conv_") {
		convID = strings.TrimPrefix(convID, "conv_")
	}

	var req struct {
		Type            string                 `json:"type" binding:"required"`
		Content         string                 `json:"content" binding:"required"`
		QuotedMessageID *uint                  `json:"quoted_message_id"`
		FileSize        int64                  `json:"file_size"`
		FileName        string                 `json:"file_name"`
		ShareData       map[string]interface{} `json:"share_data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	// 验证是否为会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限发送消息"})
		return
	}

	// 创建消息
	convIDUint, _ := strconv.ParseUint(convID, 10, 32)

	// 处理分享数据
	content := req.Content
	// 对于分享消息，content字段已经包含了JSON格式的分享数据
	// 不需要额外处理

	msg := model.Message{
		ConversationID:  uint(convIDUint),
		SenderID:        userID.(uint),
		Type:            req.Type,
		Content:         content,
		FileName:        req.FileName,
		FileSize:        req.FileSize,
		QuotedMessageID: req.QuotedMessageID,
		IsRead:          false,
	}
	db.Create(&msg)

	// 预加载发送者信息和引用消息
	db.Preload("Sender").Preload("QuotedMessage").First(&msg, msg.ID)

	// 预加载引用消息的发送者
	if msg.QuotedMessage != nil {
		db.Model(&msg.QuotedMessage).Association("Sender").Find(&msg.QuotedMessage.Sender)
	}

	// 处理分享数据，将JSON字符串转换回对象
	var shareData map[string]interface{}
	if msg.Type == "share" && msg.Content != "" {
		json.Unmarshal([]byte(msg.Content), &shareData)
	}

	// 构建响应数据
	responseData := gin.H{
		"id":                msg.ID,
		"conversation_id":   msg.ConversationID,
		"sender_id":         msg.SenderID,
		"type":              msg.Type,
		"content":           msg.Content,
		"file_name":         msg.FileName,
		"file_size":         msg.FileSize,
		"quoted_message_id": msg.QuotedMessageID,
		"is_recalled":       msg.IsRecalled,
		"is_read":           msg.IsRead,
		"recalled_at":       msg.RecalledAt,
		"created_at":        msg.CreatedAt,
		"sender":            msg.Sender,
		"quoted_message":    msg.QuotedMessage,
		"share_data":        shareData,
	}

	// 更新会话最后消息
	now := time.Now()
	var conv model.Conversation
	db.First(&conv, convID)
	conv.LastMessageID = &msg.ID
	conv.LastMessageAt = &now
	db.Save(&conv)

	// 检查是否为机器人会话
	if conv.Type == "bot" {
		// 异步处理机器人回复
		go HandleBotMessage(userID.(uint), uint(convIDUint), req.Content)
	} else {
		// 更新成员未读数（除了发送者）
		db.Model(&model.ConversationMember{}).
			Where("conversation_id = ? AND user_id != ?", convID, userID).
			UpdateColumn("unread_count", gorm.Expr("unread_count + 1"))

		// 发送WebSocket通知给会话中的所有成员（除了发送者）
		if ws.GlobalHub != nil {
			// 构建新消息通知
			newMsg := ws.WSMessage{
				Type: "new_message",
				Data: responseData,
			}
			jsonMsg, _ := json.Marshal(newMsg)

			// 发送给会话中的所有成员，排除发送者
			convIDUint, _ := strconv.ParseUint(convID, 10, 32)
			log.Printf("发送WebSocket消息到会话 %d，排除用户 %d", uint(convIDUint), userID.(uint))
			ws.GlobalHub.SendToConversation(uint(convIDUint), userID.(uint), jsonMsg)
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": responseData})
}

// 撤回消息
func RecallMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	msgIDStr := c.Param("id")

	// 转换消息ID为uint
	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的消息ID"})
		return
	}

	db := database.GetDB()

	// 验证消息是否存在
	var msg model.Message
	if err := db.First(&msg, uint(msgID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "消息不存在"})
		return
	}

	// 验证是否为消息发送者
	if msg.SenderID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只能撤回自己发送的消息"})
		return
	}

	// 验证消息是否已经被撤回
	if msg.IsRecalled {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "消息已经被撤回"})
		return
	}

	// 标记消息为已撤回
	msg.IsRecalled = true
	msg.Content = "[消息已撤回]"
	db.Save(&msg)

	// 预加载发送者信息
	db.Preload("Sender").First(&msg, msg.ID)

	// 发送WebSocket通知给会话中的所有成员
	if ws.GlobalHub != nil {
		// 构建撤回消息通知
		recallMsg := ws.WSMessage{
			Type: "message_recalled",
			Data: msg,
		}
		jsonMsg, _ := json.Marshal(recallMsg)

		// 发送给会话中的所有成员
		ws.GlobalHub.SendToConversation(msg.ConversationID, 0, jsonMsg)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "消息撤回成功", "data": msg})
}

// 消息提醒
func RemindMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	msgIDStr := c.Param("id")

	// 转换消息ID为uint
	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的消息ID"})
		return
	}

	db := database.GetDB()

	// 验证消息是否存在
	var msg model.Message
	if err := db.First(&msg, uint(msgID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "消息不存在"})
		return
	}

	// 验证是否为消息发送者
	if msg.SenderID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限发送提醒"})
		return
	}

	// 这里可以添加通知逻辑，例如发送推送通知等
	// 目前只是预留接口，返回成功

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "提醒已发送",
	})
}

// 删除消息
func DeleteMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	msgIDStr := c.Param("id")

	// 转换消息ID为uint
	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的消息ID"})
		return
	}

	db := database.GetDB()

	// 验证消息是否存在
	var msg model.Message
	if err := db.First(&msg, uint(msgID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "消息不存在"})
		return
	}

	// 验证是否为消息发送者
	if msg.SenderID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只能删除自己发送的消息"})
		return
	}

	// 软删除消息
	if err := db.Delete(&msg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除消息失败"})
		return
	}

	// 发送WebSocket通知给会话中的所有成员
	if ws.GlobalHub != nil {
		// 构建删除消息通知
		deleteMsg := ws.WSMessage{
			Type: "message_deleted",
			Data: gin.H{
				"message_id":      msg.ID,
				"conversation_id": msg.ConversationID,
			},
		}
		jsonMsg, _ := json.Marshal(deleteMsg)

		// 发送给会话中的所有成员
		ws.GlobalHub.SendToConversation(msg.ConversationID, 0, jsonMsg)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "消息删除成功",
	})
}

// 获取消息的已读用户列表
func GetMessageReadUsers(c *gin.Context) {
	userID, _ := c.Get("user_id")
	msgIDStr := c.Param("id")

	// 转换消息ID为uint
	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的消息ID"})
		return
	}

	db := database.GetDB()

	// 验证消息是否存在
	var msg model.Message
	if err := db.First(&msg, uint(msgID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "消息不存在"})
		return
	}

	// 验证是否为会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", msg.ConversationID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限访问"})
		return
	}

	// 获取已读用户列表
	var readReceipts []model.MessageReadReceipt
	db.Where("message_id = ?", msgID).Preload("User").Order("created_at DESC").Find(&readReceipts)

	// 转换为用户信息列表，过滤掉当前用户自己
	var readUsers []map[string]interface{}
	for _, receipt := range readReceipts {
		if receipt.User != nil && receipt.User.ID != userID.(uint) {
			name := receipt.User.Nickname
			if name == "" {
				name = receipt.User.Username
			}
			readUsers = append(readUsers, map[string]interface{}{
				"id":       receipt.User.ID,
				"name":     name,
				"username": receipt.User.Username,
				"avatar":   receipt.User.Avatar,
			})
		}
	}

	// 获取群成员总数
	var totalMembers int64
	db.Model(&model.ConversationMember{}).Where("conversation_id = ?", msg.ConversationID).Count(&totalMembers)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": map[string]interface{}{
			"read_users":    readUsers,
			"total_members": totalMembers,
		},
	})
}

// 标记会话消息为已读
func MarkConversationAsRead(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	// 处理带有 conv_ 前缀的会话ID
	if strings.HasPrefix(convIDStr, "conv_") {
		convIDStr = strings.TrimPrefix(convIDStr, "conv_")
	}

	// 转换会话ID为uint
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	db := database.GetDB()

	// 验证是否为会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限访问"})
		return
	}

	// 获取会话中的所有消息
	var messages []model.Message
	db.Where("conversation_id = ?", uint(convID)).Find(&messages)

	// 为每条消息创建已读回执
	for _, msg := range messages {
		// 检查是否已存在已读回执
		var existingReceipt model.MessageReadReceipt
		err := db.Where("message_id = ? AND user_id = ?", msg.ID, userID).First(&existingReceipt).Error
		if err != nil {
			// 创建新的已读回执
			receipt := model.MessageReadReceipt{
				MessageID:      msg.ID,
				ConversationID: msg.ConversationID,
				UserID:         userID.(uint),
				CreatedAt:      time.Now(),
			}
			db.Create(&receipt)
		}
	}

	// 标记消息为已读（只标记非自己发送的消息）
	result := db.Model(&model.Message{}).
		Where("conversation_id = ? AND sender_id != ? AND is_read = false", uint(convID), userID).
		UpdateColumn("is_read", true)

	// 重置未读计数
	db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", uint(convID), userID).
		UpdateColumn("unread_count", 0)

	// 更新最后阅读时间
	now := time.Now()
	db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", uint(convID), userID).
		UpdateColumn("last_read_at", now)

	// 只有当确实有消息被标记为已读时，才发送已读回执通知给对方
	if result.RowsAffected > 0 {
		// 发送已读回执通知给对方
		var conv model.Conversation
		db.First(&conv, uint(convID))

		if conv.Type == "single" {
			// 对于单聊，找到对方用户
			var otherMember model.ConversationMember
			db.Where("conversation_id = ? AND user_id != ?", uint(convID), userID).First(&otherMember)

			// 通过WebSocket通知对方
			if ws.GlobalHub != nil {
				// 构建已读回执消息
				readMsg := ws.WSMessage{
					Type: "message_read",
					Data: map[string]interface{}{
						"conversation_id": convID,
						"user_id":         userID,
						"timestamp":       time.Now().Unix(),
					},
				}
				jsonMsg, _ := json.Marshal(readMsg)

				// 发送给对方用户
				ws.GlobalHub.SendToUser(otherMember.UserID, jsonMsg)
			}
		} else if conv.Type == "group" {
			// 对于群聊，通知所有其他成员
			var members []model.ConversationMember
			db.Where("conversation_id = ? AND user_id != ?", uint(convID), userID).Find(&members)

			// 通过WebSocket通知所有其他成员
			if ws.GlobalHub != nil {
				// 构建已读回执消息
				readMsg := ws.WSMessage{
					Type: "message_read",
					Data: map[string]interface{}{
						"conversation_id": convID,
						"user_id":         userID,
						"timestamp":       time.Now().Unix(),
					},
				}
				jsonMsg, _ := json.Marshal(readMsg)

				// 发送给所有其他成员
				for _, member := range members {
					ws.GlobalHub.SendToUser(member.UserID, jsonMsg)
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "标记已读成功"})
}

func generateToken(userID uint, username string) string {
	claims := middleware.Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte("your-secret-key-change-in-production"))
	return tokenString
}

// 添加成员到群聊或讨论组
func AddMemberToGroup(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convID := c.Param("id")

	// 处理带有 conv_ 前缀的会话ID
	if strings.HasPrefix(convID, "conv_") {
		convID = strings.TrimPrefix(convID, "conv_")
	}

	var req struct {
		MemberIDs []uint `json:"member_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	// 转换会话ID为uint
	convIDUint, err := strconv.ParseUint(convID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	// 验证会话是否存在
	var conv model.Conversation
	if err := db.First(&conv, uint(convIDUint)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	// 验证是否为群聊或讨论组
	if conv.Type != "group" && conv.Type != "discussion" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "只能为群聊或讨论组添加成员"})
		return
	}

	// 验证当前用户是否为群聊成员
	var currentMember model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convIDUint), userID).First(&currentMember).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限操作"})
		return
	}

	// 添加新成员
	var addedMembers []model.User
	for _, memberID := range req.MemberIDs {
		// 检查是否已经是群成员
		var existingMember model.ConversationMember
		if err := db.Where("conversation_id = ? AND user_id = ?", uint(convIDUint), memberID).First(&existingMember).Error; err == nil {
			// 已经是成员，跳过
			continue
		}

		// 验证用户是否存在
		var user model.User
		if err := db.First(&user, memberID).Error; err != nil {
			continue
		}

		// 创建成员记录
		newMember := model.ConversationMember{
			ConversationID: uint(convIDUint),
			UserID:         memberID,
			Role:           "member",
			UnreadCount:    0,
			Muted:          false,
			JoinedAt:       time.Now(),
		}
		db.Create(&newMember)

		// 创建邀请通知
		notification := model.Notification{
			UserID:  memberID,
			Type:    "group_invitation",
			Title:   "群聊邀请",
			Content: fmt.Sprintf("您被邀请加入群聊 %s", conv.Name),
		}
		db.Create(&notification)

		// 通过WebSocket通知被邀请人
		if ws.GlobalHub != nil {
			notificationMsg := ws.WSMessage{
				Type: "notification",
				Data: notification,
			}
			jsonMsg, _ := json.Marshal(notificationMsg)
			ws.GlobalHub.SendToUser(memberID, jsonMsg)
		}

		addedMembers = append(addedMembers, user)
	}

	// 为群内其他成员创建新成员加入通知
	if len(addedMembers) > 0 {
		var existingMembers []model.ConversationMember
		db.Where("conversation_id = ? AND user_id != ?", uint(convIDUint), userID).Find(&existingMembers)

		// 获取添加成员的名称列表
		var memberNames []string
		for _, member := range addedMembers {
			memberNames = append(memberNames, member.Nickname)
		}

		for _, member := range existingMembers {
			notification := model.Notification{
				UserID:  member.UserID,
				Type:    "group_member_added",
				Title:   "新成员加入",
				Content: fmt.Sprintf("新成员 %s 加入了群聊", strings.Join(memberNames, "、")),
			}
			db.Create(&notification)

			// 通过WebSocket通知群成员
			if ws.GlobalHub != nil {
				notificationMsg := ws.WSMessage{
					Type: "notification",
					Data: notification,
				}
				jsonMsg, _ := json.Marshal(notificationMsg)
				ws.GlobalHub.SendToUser(member.UserID, jsonMsg)
			}
		}
	}

	// 发送 group_member_joined 事件给群内所有成员
	if ws.GlobalHub != nil {
		for _, member := range addedMembers {
			// 构建成员加入事件
			joinMsg := ws.WSMessage{
				Type: "group_member_joined",
				Data: gin.H{
					"conversation_id": conv.ID,
					"member": gin.H{
						"id":       member.ID,
						"nickname": member.Nickname,
						"username": member.Username,
						"avatar":   member.Avatar,
					},
				},
			}
			jsonMsg, _ := json.Marshal(joinMsg)

			// 发送给会话中的所有成员
			ws.GlobalHub.SendToConversation(uint(convIDUint), 0, jsonMsg)
		}
	}

	// 发送 added_to_group 事件给被添加的用户
	if ws.GlobalHub != nil {
		// 获取群聊成员列表
		var groupMembers []model.ConversationMember
		db.Where("conversation_id = ?", uint(convIDUint)).Preload("User").Find(&groupMembers)

		// 构建成员列表
		var members []gin.H
		for _, gm := range groupMembers {
			members = append(members, gin.H{
				"id":       gm.User.ID,
				"nickname": gm.User.Nickname,
				"username": gm.User.Username,
				"avatar":   gm.User.Avatar,
			})
		}

		for _, member := range addedMembers {
			// 构建被添加到群聊事件
			addedMsg := ws.WSMessage{
				Type: "added_to_group",
				Data: gin.H{
					"conversation_id": conv.ID,
					"group_name":      conv.Name,
					"group_avatar":    conv.Avatar,
					"members":         members,
				},
			}
			jsonMsg, _ := json.Marshal(addedMsg)

			// 发送给被添加的用户
			ws.GlobalHub.SendToUser(member.ID, jsonMsg)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "添加成员成功",
		"data":    addedMembers,
	})
}

// 退出群聊
func ExitGroup(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	// 处理带有 conv_ 前缀的会话ID
	if strings.HasPrefix(convIDStr, "conv_") {
		convIDStr = strings.TrimPrefix(convIDStr, "conv_")
	}

	// 转换会话ID为uint
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	db := database.GetDB()

	// 验证会话是否存在
	var conv model.Conversation
	if err := db.First(&conv, uint(convID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	// 验证是否为群聊
	if conv.Type != "group" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "只能退出群聊"})
		return
	}

	// 验证当前用户是否为群成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "您不是群成员"})
		return
	}

	// 从群聊中移除成员
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).Delete(&model.ConversationMember{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "退出群聊失败"})
		return
	}

	// 发送WebSocket通知给群成员
	if ws.GlobalHub != nil {
		// 构建成员退出通知
		exitMsg := ws.WSMessage{
			Type: "group_member_left",
			Data: gin.H{
				"conversation_id": conv.ID,
				"user_id":         userID,
			},
		}
		jsonMsg, _ := json.Marshal(exitMsg)

		// 发送给会话中的所有成员
		ws.GlobalHub.SendToConversation(uint(convID), 0, jsonMsg)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "退出群聊成功"})
}

// 移除群成员
func RemoveMemberFromGroup(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")
	memberIDStr := c.Param("user_id")

	// 处理带有 conv_ 前缀的会话ID
	if strings.HasPrefix(convIDStr, "conv_") {
		convIDStr = strings.TrimPrefix(convIDStr, "conv_")
	}

	// 转换会话ID和成员ID为uint
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	memberID, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}

	db := database.GetDB()

	// 验证会话是否存在
	var conv model.Conversation
	if err := db.First(&conv, uint(convID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	// 验证是否为群聊
	if conv.Type != "group" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "只能移除群聊成员"})
		return
	}

	// 验证当前用户是否为群主或管理员
	var currentMember model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&currentMember).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "您不是群成员"})
		return
	}

	if currentMember.Role != "owner" && currentMember.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只有群主或管理员可以移除成员"})
		return
	}

	// 验证目标用户是否为群成员
	var targetMember model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), uint(memberID)).First(&targetMember).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "目标用户不是群成员"})
		return
	}

	// 群主不能被移除
	if targetMember.Role == "owner" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "群主不能被移除"})
		return
	}

	// 移除成员
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), uint(memberID)).Delete(&model.ConversationMember{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "移除成员失败"})
		return
	}

	// 发送WebSocket通知给群成员
	if ws.GlobalHub != nil {
		// 构建成员移除通知
		removeMsg := ws.WSMessage{
			Type: "group_member_left",
			Data: gin.H{
				"conversation_id": conv.ID,
				"user_id":         uint(memberID),
			},
		}
		jsonMsg, _ := json.Marshal(removeMsg)

		// 发送给会话中的所有成员
		ws.GlobalHub.SendToConversation(uint(convID), 0, jsonMsg)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "移除成员成功",
	})
}

// 获取机器人列表
func GetBots(c *gin.Context) {
	db := database.GetDB()

	var bots []model.Bot
	db.Where("is_active = ?", true).Find(&bots)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": bots,
	})
}

// 处理机器人消息
func HandleBotMessage(userID uint, convID uint, content string) {
	db := database.GetDB()

	// 查找机器人会话关联
	var botConv model.BotConversation
	if err := db.Where("conversation_id = ?", convID).First(&botConv).Error; err != nil {
		return
	}

	// 查找机器人信息
	var bot model.Bot
	if err := db.First(&bot, botConv.BotID).Error; err != nil {
		return
	}

	// 根据机器人类型生成回复
	var reply string
	switch bot.Type {
	case "system":
		// 系统机器人简单回复
		reply = getSystemBotReply(content)
	case "ai":
		// AI机器人回复（这里使用模拟回复，实际应该调用API）
		reply = getAIBotReply(content)
	default:
		reply = "我是一个机器人，有什么可以帮你的吗？"
	}

	// 创建机器人回复消息
	msg := model.Message{
		ConversationID: convID,
		SenderID:       0, // 0表示机器人
		Type:           "text",
		Content:        reply,
	}
	db.Create(&msg)

	// 预加载发送者信息
	// 由于是机器人，我们创建一个虚拟的发送者
	botUser := model.User{
		ID:       0,
		Username: bot.Name,
		Nickname: bot.Name,
		Avatar:   bot.Avatar,
	}
	msg.Sender = botUser

	// 更新会话最后消息
	now := time.Now()
	var conv model.Conversation
	db.First(&conv, convID)
	conv.LastMessageID = &msg.ID
	conv.LastMessageAt = &now
	db.Save(&conv)

	// 构建推送消息
	wsMsg := ws.WSMessage{
		Type: "new_message",
		Data: msg,
	}
	jsonMsg, _ := json.Marshal(wsMsg)

	// 推送给用户
	if ws.GlobalHub != nil {
		ws.GlobalHub.SendToUser(userID, jsonMsg)
	}
}

// 系统机器人回复
func getSystemBotReply(content string) string {
	content = strings.ToLower(content)
	if strings.Contains(content, "你好") || strings.Contains(content, "hi") || strings.Contains(content, "hello") {
		return "你好！我是系统助手，有什么可以帮你的吗？"
	} else if strings.Contains(content, "帮助") || strings.Contains(content, "help") {
		return "我可以帮助你了解系统功能，解答常见问题。你可以问我关于系统使用的问题。"
	} else if strings.Contains(content, "时间") || strings.Contains(content, "time") {
		return "当前时间是：" + time.Now().Format("2006-01-02 15:04:05")
	} else {
		return "我是系统助手，有什么可以帮你的吗？"
	}
}

// AI机器人回复（模拟）
func getAIBotReply(content string) string {
	// 这里应该调用实际的AI API，现在使用模拟回复
	replies := []string{
		"这是一个有趣的问题！让我想想...",
		"根据我的理解，你是在问关于...",
		"好的，我来帮你解答这个问题。",
		"这个问题很有意思，我认为...",
		"让我分析一下这个问题...",
	}
	return replies[rand.Intn(len(replies))] + "\n\n你刚才说：" + content
}

// 更新群聊信息
func UpdateGroupInfo(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	// 转换会话ID为uint
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	var req struct {
		Name   string `json:"name"`
		Avatar string `json:"avatar"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	// 验证会话是否存在
	var conv model.Conversation
	if err := db.First(&conv, uint(convID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	// 验证是否为群聊或讨论组
	if conv.Type != "group" && conv.Type != "discussion" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "只能更新群聊或讨论组信息"})
		return
	}

	// 验证当前用户是否为成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "您不是成员"})
		return
	}

	// 验证权限：群聊需要群主或管理员，讨论组所有成员都可以
	if conv.Type == "group" && member.Role != "owner" && member.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只有群主或管理员可以更新群聊信息"})
		return
	}
	// 讨论组所有成员都可以更新

	// 更新群聊信息
	if req.Name != "" {
		conv.Name = req.Name
	}
	if req.Avatar != "" {
		conv.Avatar = req.Avatar
	}
	db.Save(&conv)

	// 发送WebSocket通知给群成员
	if ws.GlobalHub != nil {
		// 构建群聊更新通知
		updateMsg := ws.WSMessage{
			Type: "conversation_updated",
			Data: conv,
		}
		jsonMsg, _ := json.Marshal(updateMsg)

		// 发送给会话中的所有成员
		ws.GlobalHub.SendToConversation(uint(convID), 0, jsonMsg)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "群聊信息更新成功",
		"data":    conv,
	})
}

// 上传文件
func UploadFile(c *gin.Context) {
	userID, _ := c.Get("user_id")

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文件上传失败"})
		return
	}

	// 确保上传目录存在
	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建上传目录失败"})
		return
	}

	// 生成唯一的文件名
	ext := filepath.Ext(file.Filename)
	filename := time.Now().Format("20060102150405") + "_" + strconv.FormatUint(uint64(userID.(uint)), 10) + ext
	filePath := filepath.Join(uploadDir, filename)

	// 保存文件
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存文件失败"})
		return
	}

	// 创建文件记录
	db := database.GetDB()
	fileRecord := model.File{
		Name:         file.Filename,
		OriginalName: file.Filename,
		StoragePath:  "/uploads/" + filename,
		Size:         file.Size,
		UserID:       userID.(uint),
		CreatedAt:    time.Now(),
	}
	if err := db.Create(&fileRecord).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建文件记录失败"})
		return
	}

	// 返回文件URL
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":   fileRecord.ID,
			"url":  fileRecord.StoragePath,
			"name": fileRecord.Name,
			"size": fileRecord.Size,
		},
	})
}

// 获取文件列表
func GetFiles(c *gin.Context) {
	userID, _ := c.Get("user_id")

	// 获取查询参数
	folderIDStr := c.Query("folder_id")

	db := database.GetDB()
	var files []model.File

	query := db.Where("user_id = ?", userID)
	if folderIDStr != "" {
		folderID, err := strconv.ParseUint(folderIDStr, 10, 32)
		if err == nil {
			query = query.Where("folder_id = ?", uint(folderID))
		}
	}

	query.Order("created_at DESC").Find(&files)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": files,
	})
}

// 系统消息相关API

// 发布系统消息
func CreateSystemMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Title      string `json:"title" binding:"required"`
		Content    string `json:"content" binding:"required"`
		TargetType string `json:"target_type"`
		TargetID   *uint  `json:"target_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	// 创建系统消息
	systemMessage := model.SystemMessage{
		Title:      req.Title,
		Content:    req.Content,
		SenderID:   userID.(uint),
		Status:     "active",
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
	}

	if err := db.Create(&systemMessage).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建系统消息失败"})
		return
	}

	// 预加载发送者信息
	db.Preload("Sender").First(&systemMessage, systemMessage.ID)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": systemMessage,
	})
}

// 获取系统消息列表
func GetSystemMessages(c *gin.Context) {
	db := database.GetDB()

	var systemMessages []model.SystemMessage
	db.Preload("Sender").Order("created_at DESC").Find(&systemMessages)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": systemMessages,
	})
}

// 更新系统消息状态
func UpdateSystemMessage(c *gin.Context) {
	messageIDStr := c.Param("id")

	// 转换消息ID为uint
	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的消息ID"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	// 验证消息是否存在
	var systemMessage model.SystemMessage
	if err := db.First(&systemMessage, uint(messageID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "消息不存在"})
		return
	}

	// 更新消息状态
	systemMessage.Status = req.Status
	if err := db.Save(&systemMessage).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新消息状态失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": systemMessage,
	})
}

// 频道相关API

// 创建频道
func CreateChannel(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Avatar      string `json:"avatar"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	// 创建频道
	channel := model.Channel{
		Name:        req.Name,
		Description: req.Description,
		Avatar:      req.Avatar,
		CreatorID:   userID.(uint),
		Status:      "active",
	}

	if err := db.Create(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建频道失败"})
		return
	}

	// 预加载创建者信息
	db.Preload("Creator").First(&channel, channel.ID)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": channel,
	})
}

// 获取频道列表
func GetChannels(c *gin.Context) {
	userID, _ := c.Get("user_id")
	db := database.GetDB()

	// 获取所有频道
	var channels []model.Channel
	db.Preload("Creator").Find(&channels)

	// 获取用户订阅的频道
	var subscriptions []model.ChannelSubscriber
	db.Where("user_id = ?", userID).Find(&subscriptions)

	// 构建订阅状态映射
	subscribedMap := make(map[uint]bool)
	for _, sub := range subscriptions {
		subscribedMap[sub.ChannelID] = true
	}

	// 为每个频道添加订阅状态
	type ChannelWithSubscription struct {
		model.Channel
		IsSubscribed bool `json:"is_subscribed"`
	}

	var channelsWithSubscription []ChannelWithSubscription
	for _, channel := range channels {
		channelsWithSubscription = append(channelsWithSubscription, ChannelWithSubscription{
			Channel:      channel,
			IsSubscribed: subscribedMap[channel.ID],
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": channelsWithSubscription,
	})
}

// 订阅频道
func SubscribeChannel(c *gin.Context) {
	userID, _ := c.Get("user_id")
	channelIDStr := c.Param("id")

	// 转换频道ID为uint
	channelID, err := strconv.ParseUint(channelIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的频道ID"})
		return
	}

	db := database.GetDB()

	// 验证频道是否存在
	var channel model.Channel
	if err := db.First(&channel, uint(channelID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "频道不存在"})
		return
	}

	// 检查是否已经订阅
	var existingSubscription model.ChannelSubscriber
	if err := db.Where("channel_id = ? AND user_id = ?", uint(channelID), userID).First(&existingSubscription).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已经订阅该频道"})
		return
	}

	// 创建订阅记录
	subscription := model.ChannelSubscriber{
		ChannelID: uint(channelID),
		UserID:    userID.(uint),
		JoinedAt:  time.Now(),
	}

	if err := db.Create(&subscription).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "订阅频道失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "订阅频道成功",
		"data":    subscription,
	})
}

// 取消订阅频道
func UnsubscribeChannel(c *gin.Context) {
	userID, _ := c.Get("user_id")
	channelIDStr := c.Param("id")

	// 转换频道ID为uint
	channelID, err := strconv.ParseUint(channelIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的频道ID"})
		return
	}

	db := database.GetDB()

	// 检查是否已经订阅
	var subscription model.ChannelSubscriber
	if err := db.Where("channel_id = ? AND user_id = ?", uint(channelID), userID).First(&subscription).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "未订阅该频道"})
		return
	}

	// 删除订阅记录
	if err := db.Delete(&subscription).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "取消订阅失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "取消订阅成功",
	})
}

// 发布频道消息
func CreateChannelMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	channelIDStr := c.Param("id")

	// 转换频道ID为uint
	channelID, err := strconv.ParseUint(channelIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的频道ID"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
		Type    string `json:"type" default:"text"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	// 验证频道是否存在
	var channel model.Channel
	if err := db.First(&channel, uint(channelID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "频道不存在"})
		return
	}

	// 验证用户是否为频道创建者（只有创建者可以发布消息）
	if channel.CreatorID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限发布消息"})
		return
	}

	// 创建频道消息
	channelMessage := model.ChannelMessage{
		ChannelID: uint(channelID),
		SenderID:  userID.(uint),
		Content:   req.Content,
		Type:      req.Type,
	}

	if err := db.Create(&channelMessage).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "发布消息失败"})
		return
	}

	// 预加载发送者信息
	db.Preload("Sender").First(&channelMessage, channelMessage.ID)

	// 获取频道订阅者
	var subscribers []model.ChannelSubscriber
	db.Where("channel_id = ?", uint(channelID)).Find(&subscribers)

	// 为订阅者创建通知
	for _, subscriber := range subscribers {
		notification := model.Notification{
			UserID:  subscriber.UserID,
			Type:    "channel_message",
			Title:   fmt.Sprintf("频道消息: %s", channel.Name),
			Content: req.Content,
		}
		db.Create(&notification)

		// 通过WebSocket通知订阅者
		if ws.GlobalHub != nil {
			notificationMsg := ws.WSMessage{
				Type: "notification",
				Data: notification,
			}
			jsonMsg, _ := json.Marshal(notificationMsg)
			ws.GlobalHub.SendToUser(subscriber.UserID, jsonMsg)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": channelMessage,
	})
}

// 获取频道消息
func GetChannelMessages(c *gin.Context) {
	userID, _ := c.Get("user_id")
	channelIDStr := c.Param("id")

	// 转换频道ID为uint
	channelID, err := strconv.ParseUint(channelIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的频道ID"})
		return
	}

	db := database.GetDB()

	// 验证频道是否存在
	var channel model.Channel
	if err := db.First(&channel, uint(channelID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "频道不存在"})
		return
	}

	// 验证用户是否为频道订阅者
	var subscription model.ChannelSubscriber
	if err := db.Where("channel_id = ? AND user_id = ?", uint(channelID), userID).First(&subscription).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限访问"})
		return
	}

	// 获取频道消息
	var messages []model.ChannelMessage
	db.Where("channel_id = ?", uint(channelID)).Preload("Sender").Order("created_at DESC").Find(&messages)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": messages,
	})
}

// 下载文件
func DownloadFile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	fileIDStr := c.Param("id")

	// 转换文件ID为uint
	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的文件ID"})
		return
	}

	db := database.GetDB()
	var file model.File
	if err := db.Where("id = ? AND user_id = ?", uint(fileID), userID).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件不存在"})
		return
	}

	// 构建文件路径
	filePath := "." + file.StoragePath

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件不存在"})
		return
	}

	// 下载文件
	c.FileAttachment(filePath, file.Name)
}

// 删除文件
func DeleteFile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	fileIDStr := c.Param("id")

	// 转换文件ID为uint
	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的文件ID"})
		return
	}

	db := database.GetDB()
	var file model.File
	if err := db.Where("id = ? AND user_id = ?", uint(fileID), userID).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件不存在"})
		return
	}

	// 删除物理文件
	filePath := "." + file.StoragePath
	os.Remove(filePath)

	// 删除数据库记录
	if err := db.Delete(&file).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除文件失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除文件成功",
	})
}

// 获取笔记列表
func GetNotes(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()
	var notes []model.Note
	db.Where("user_id = ?", userID).Order("updated_at DESC").Find(&notes)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": notes,
	})
}

// 获取笔记详情
func GetNote(c *gin.Context) {
	userID, _ := c.Get("user_id")
	noteIDStr := c.Param("id")

	// 转换笔记ID为uint
	noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的笔记ID"})
		return
	}

	db := database.GetDB()
	var note model.Note
	if err := db.Where("id = ? AND user_id = ?", uint(noteID), userID).First(&note).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "笔记不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": note,
	})
}

// 创建笔记
func CreateNote(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"omitempty"`
		Color   string `json:"color"`
		Type    string `json:"type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	note := model.Note{
		UserID:  userID.(uint),
		Title:   req.Title,
		Content: req.Content,
		Color:   req.Color,
		Type:    req.Type,
	}
	db.Create(&note)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": note,
	})
}

// 更新笔记
func UpdateNote(c *gin.Context) {
	userID, _ := c.Get("user_id")
	noteIDStr := c.Param("id")

	// 转换笔记ID为uint
	noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的笔记ID"})
		return
	}

	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"omitempty"`
		Color   string `json:"color"`
		Type    string `json:"type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	var note model.Note
	if err := db.Where("id = ? AND user_id = ?", uint(noteID), userID).First(&note).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "笔记不存在"})
		return
	}

	note.Title = req.Title
	note.Content = req.Content
	note.Color = req.Color
	note.Type = req.Type
	db.Save(&note)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": note,
	})
}

// 删除笔记
func DeleteNote(c *gin.Context) {
	userID, _ := c.Get("user_id")
	noteIDStr := c.Param("id")

	// 转换笔记ID为uint
	noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的笔记ID"})
		return
	}

	db := database.GetDB()
	if err := db.Where("id = ? AND user_id = ?", uint(noteID), userID).Delete(&model.Note{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除笔记失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除笔记成功",
	})
}

// 创建文件夹
func CreateFolder(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Name     string `json:"name" binding:"required"`
		ParentID *uint  `json:"parent_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	folder := model.Folder{
		UserID:   userID.(uint),
		Name:     req.Name,
		ParentID: req.ParentID,
	}
	db.Create(&folder)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": folder,
	})
}

// 获取文件夹树
func GetFolderTree(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()

	// 查询所有文件夹
	var folders []model.Folder
	db.Where("user_id = ?", userID).Find(&folders)

	// 构建文件夹树
	folderMap := make(map[uint]*model.Folder)
	var rootFolders []model.Folder

	// 首先将所有文件夹放入map
	for i := range folders {
		folderMap[folders[i].ID] = &folders[i]
	}

	// 构建树结构
	for i := range folders {
		if folders[i].ParentID == nil {
			// 根文件夹
			rootFolders = append(rootFolders, folders[i])
		} else {
			// 子文件夹
			if _, exists := folderMap[*folders[i].ParentID]; exists {
				// 这里需要在Folder模型中添加Children字段
				// 由于我们没有修改模型，这里返回扁平化结构
			}
		}
	}

	// 由于模型中没有Children字段，我们返回所有文件夹，前端自行构建树
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": folders,
	})
}

// 搜索消息
func SearchMessages(c *gin.Context) {
	userID, _ := c.Get("user_id")

	// 获取查询参数
	keyword := c.Query("keyword")
	convID := c.Query("conv_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	msgType := c.Query("type")

	db := database.GetDB()

	// 构建查询
	query := db.Model(&model.Message{}).Joins("JOIN conversation_members ON messages.conversation_id = conversation_members.conversation_id").Where("conversation_members.user_id = ?", userID)

	// 添加搜索条件
	if keyword != "" {
		query = query.Where("messages.content LIKE ?", "%"+keyword+"%")
	}

	if convID != "" {
		query = query.Where("messages.conversation_id = ?", convID)
	}

	if startDate != "" {
		query = query.Where("messages.created_at >= ?", startDate)
	}

	if endDate != "" {
		query = query.Where("messages.created_at <= ?", endDate)
	}

	if msgType != "" {
		query = query.Where("messages.type = ?", msgType)
	}

	// 执行查询
	var messages []model.Message
	query.Preload("Sender").Preload("Conversation").Order("messages.created_at DESC").Find(&messages)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": messages,
	})
}

// 获取消息引用链
func GetMessageQuoteChain(c *gin.Context) {
	userID, _ := c.Get("user_id")
	msgIDStr := c.Param("id")

	// 转换消息ID为uint
	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的消息ID"})
		return
	}

	db := database.GetDB()

	// 验证消息是否存在
	var msg model.Message
	if err := db.First(&msg, uint(msgID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "消息不存在"})
		return
	}

	// 验证是否为会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", msg.ConversationID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限访问"})
		return
	}

	// 构建引用链
	var quoteChain []model.Message
	currentMsg := msg

	// 最多获取3层引用
	for i := 0; i < 3 && currentMsg.QuotedMessageID != nil; i++ {
		var quotedMsg model.Message
		if err := db.Preload("Sender").First(&quotedMsg, *currentMsg.QuotedMessageID).Error; err == nil {
			quoteChain = append(quoteChain, quotedMsg)
			currentMsg = quotedMsg
		} else {
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"messages": quoteChain,
		},
	})
}

// 会话置顶/取消置顶
func PinConversation(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	// 处理带有 conv_ 前缀的会话ID
	if strings.HasPrefix(convIDStr, "conv_") {
		convIDStr = strings.TrimPrefix(convIDStr, "conv_")
	}

	// 转换会话ID为uint
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	var req struct {
		IsPinned bool `json:"is_pinned"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	db := database.GetDB()

	// 验证是否为会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限操作"})
		return
	}

	// 查找或创建会话会话记录
	var session model.ConversationSession
	result := db.Where("user_id = ? AND conversation_id = ?", userID, uint(convID)).First(&session)

	if result.Error != nil {
		// 创建新记录
		session = model.ConversationSession{
			UserID:         userID.(uint),
			ConversationID: uint(convID),
			IsPinned:       req.IsPinned,
			LastVisitedAt:  time.Now(),
		}
		if req.IsPinned {
			now := time.Now()
			session.PinnedAt = &now
		}
		db.Create(&session)
	} else {
		// 更新现有记录
		session.IsPinned = req.IsPinned
		if req.IsPinned {
			now := time.Now()
			session.PinnedAt = &now
		} else {
			session.PinnedAt = nil
		}
		db.Save(&session)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "操作成功",
		"data":    session,
	})
}

// 设置免打扰
func SetConversationMute(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	// 处理带有 conv_ 前缀的会话ID
	if strings.HasPrefix(convIDStr, "conv_") {
		convIDStr = strings.TrimPrefix(convIDStr, "conv_")
	}

	// 转换会话ID为uint
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	var req struct {
		Muted bool `json:"muted" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	// 验证是否为会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限操作"})
		return
	}

	// 更新免打扰状态
	member.Muted = req.Muted
	db.Save(&member)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "操作成功",
		"data":    member,
	})
}

// 获取日历事件列表
func GetEvents(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()
	var events []model.Event
	db.Where("user_id = ?", userID).Order("start DESC").Find(&events)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": events,
	})
}

// 创建日历事件
func CreateEvent(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Title       string    `json:"title" binding:"required"`
		Description string    `json:"description"`
		Start       time.Time `json:"start" binding:"required"`
		End         time.Time `json:"end" binding:"required"`
		AllDay      bool      `json:"all_day"`
		Reminder    int       `json:"reminder"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	event := model.Event{
		UserID:      userID.(uint),
		Title:       req.Title,
		Description: req.Description,
		Start:       req.Start,
		End:         req.End,
		AllDay:      req.AllDay,
		Reminder:    req.Reminder,
	}
	db.Create(&event)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": event,
	})
}

// 获取单个日历事件
func GetEvent(c *gin.Context) {
	userID, _ := c.Get("user_id")
	eventIDStr := c.Param("id")

	// 转换事件ID为uint
	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的事件ID"})
		return
	}

	db := database.GetDB()
	var event model.Event
	if err := db.Where("id = ? AND user_id = ?", uint(eventID), userID).First(&event).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "事件不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": event,
	})
}

// 更新日历事件
func UpdateEvent(c *gin.Context) {
	userID, _ := c.Get("user_id")
	eventIDStr := c.Param("id")

	// 转换事件ID为uint
	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的事件ID"})
		return
	}

	var req struct {
		Title       string    `json:"title" binding:"required"`
		Description string    `json:"description"`
		Start       time.Time `json:"start" binding:"required"`
		End         time.Time `json:"end" binding:"required"`
		AllDay      bool      `json:"all_day"`
		Reminder    int       `json:"reminder"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	var event model.Event
	if err := db.Where("id = ? AND user_id = ?", uint(eventID), userID).First(&event).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "事件不存在"})
		return
	}

	event.Title = req.Title
	event.Description = req.Description
	event.Start = req.Start
	event.End = req.End
	event.AllDay = req.AllDay
	event.Reminder = req.Reminder
	db.Save(&event)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": event,
	})
}

// 删除日历事件
func DeleteEvent(c *gin.Context) {
	userID, _ := c.Get("user_id")
	eventIDStr := c.Param("id")

	// 转换事件ID为uint
	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的事件ID"})
		return
	}

	db := database.GetDB()
	if err := db.Where("id = ? AND user_id = ?", uint(eventID), userID).Delete(&model.Event{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除事件失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除事件成功",
	})
}

// 获取小程序列表
func GetMiniApps(c *gin.Context) {
	db := database.GetDB()

	var miniApps []model.MiniApp
	db.Where("status = ?", "active").Find(&miniApps)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": miniApps,
	})
}

// 获取小程序详情
func GetMiniApp(c *gin.Context) {
	appID := c.Param("id")

	db := database.GetDB()
	var miniApp model.MiniApp
	if err := db.Where("app_id = ?", appID).First(&miniApp).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "小程序不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": miniApp,
	})
}

// 创建小程序
func CreateMiniApp(c *gin.Context) {
	// userID, _ := c.Get("user_id")

	var req struct {
		AppID       string `json:"app_id" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		Path        string `json:"path"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	// 检查AppID是否已存在
	var count int64
	db.Model(&model.MiniApp{}).Where("app_id = ?", req.AppID).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "AppID已存在"})
		return
	}

	miniApp := model.MiniApp{
		AppID:       req.AppID,
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
		Path:        req.Path,
		Status:      "active",
	}

	if err := db.Create(&miniApp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建小程序失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": miniApp,
	})
}

// 更新小程序
func UpdateMiniApp(c *gin.Context) {
	// userID, _ := c.Get("user_id")
	appID := c.Param("id")

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		Path        string `json:"path"`
		Status      string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	var miniApp model.MiniApp
	if err := db.Where("app_id = ?", appID).First(&miniApp).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "小程序不存在"})
		return
	}

	if req.Name != "" {
		miniApp.Name = req.Name
	}
	if req.Description != "" {
		miniApp.Description = req.Description
	}
	if req.Icon != "" {
		miniApp.Icon = req.Icon
	}
	if req.Path != "" {
		miniApp.Path = req.Path
	}
	if req.Status != "" {
		miniApp.Status = req.Status
	}

	db.Save(&miniApp)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": miniApp,
	})
}

// 删除小程序
func DeleteMiniApp(c *gin.Context) {
	// userID, _ := c.Get("user_id")
	appID := c.Param("id")

	db := database.GetDB()
	if err := db.Where("app_id = ?", appID).Delete(&model.MiniApp{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除小程序失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除小程序成功",
	})
}

// 应用管理
func GetApps(c *gin.Context) {
	userID, _ := c.Get("user_id")
	fmt.Println("获取应用列表请求，用户ID:", userID)

	db := database.GetDB()
	var apps []model.App
	result := db.Where("user_id = ?", userID).Order("created_at DESC").Find(&apps)
	if result.Error != nil {
		fmt.Println("获取应用列表失败:", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取应用列表失败"})
		return
	}

	fmt.Println("获取应用列表成功，应用数量:", len(apps))
	fmt.Println("应用列表:", apps)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": apps,
	})
}

// 获取所有应用
func GetAllApps(c *gin.Context) {
	db := database.GetDB()
	var apps []model.App
	db.Order("created_at DESC").Find(&apps)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": apps,
	})
}

// 创建应用
func CreateApp(c *gin.Context) {
	userID, _ := c.Get("user_id")
	fmt.Println("创建应用请求，用户ID:", userID)

	var req struct {
		Name     string `json:"name" binding:"required"`
		Icon     string `json:"icon"`
		Category string `json:"category"`
		URL      string `json:"url"`
		Status   string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Println("参数错误:", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	fmt.Println("创建应用请求体:", req)

	db := database.GetDB()
	app := model.App{
		UserID:   userID.(uint),
		Name:     req.Name,
		Icon:     req.Icon,
		Category: req.Category,
		URL:      req.URL,
		Status:   req.Status,
	}
	result := db.Create(&app)
	if result.Error != nil {
		fmt.Println("创建应用失败:", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建应用失败"})
		return
	}

	fmt.Println("创建应用成功，应用ID:", app.ID)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": app,
	})
}

// 更新应用
func UpdateApp(c *gin.Context) {
	userID, _ := c.Get("user_id")
	appIDStr := c.Param("id")

	// 转换应用ID为uint
	appID, err := strconv.ParseUint(appIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的应用ID"})
		return
	}

	var req struct {
		Name     string `json:"name" binding:"required"`
		Icon     string `json:"icon"`
		Category string `json:"category"`
		URL      string `json:"url"`
		Status   string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	var app model.App
	if err := db.Where("id = ? AND user_id = ?", uint(appID), userID).First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	app.Name = req.Name
	app.Icon = req.Icon
	app.Category = req.Category
	app.URL = req.URL
	app.Status = req.Status
	db.Save(&app)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": app,
	})
}

// 删除应用
func DeleteApp(c *gin.Context) {
	userID, _ := c.Get("user_id")
	appIDStr := c.Param("id")

	// 转换应用ID为uint
	appID, err := strconv.ParseUint(appIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的应用ID"})
		return
	}

	db := database.GetDB()
	if err := db.Where("id = ? AND user_id = ?", uint(appID), userID).Delete(&model.App{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除应用失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除应用成功",
	})
}

// 统计报表
func GetStatistics(c *gin.Context) {
	userID, _ := c.Get("user_id")
	period := c.DefaultQuery("period", "week")

	db := database.GetDB()

	// 计算时间范围
	now := time.Now()
	var startDate time.Time

	switch period {
	case "day":
		startDate = now.AddDate(0, 0, -1)
	case "week":
		startDate = now.AddDate(0, 0, -7)
	case "month":
		startDate = now.AddDate(0, -1, 0)
	case "year":
		startDate = now.AddDate(-1, 0, 0)
	default:
		startDate = now.AddDate(0, 0, -7) // 默认一周
	}

	// 计算消息总数
	var totalMessages int64
	db.Model(&model.Message{}).Where("sender_id = ? AND created_at >= ?", userID, startDate).Count(&totalMessages)

	// 计算文件总数
	var totalFiles int64
	db.Model(&model.File{}).Where("user_id = ? AND created_at >= ?", userID, startDate).Count(&totalFiles)

	// 计算笔记总数
	var totalNotes int64
	db.Model(&model.Note{}).Where("user_id = ? AND created_at >= ?", userID, startDate).Count(&totalNotes)

	// 计算任务总数（这里假设任务是通过其他方式实现的，暂时返回0）
	totalTasks := int64(0)
	completedTasks := int64(0)
	pendingTasks := int64(0)
	taskCompletionRate := 0.0

	// 生成消息趋势数据
	messageTrend := []map[string]interface{}{}

	switch period {
	case "day":
		// 按小时统计
		for i := 23; i >= 0; i-- {
			hour := now.Add(-time.Duration(i) * time.Hour)
			hourStr := fmt.Sprintf("%d:00", hour.Hour())

			var count int64
			db.Model(&model.Message{}).Where("sender_id = ? AND created_at >= ? AND created_at < ?",
				userID, hour, hour.Add(time.Hour)).Count(&count)

			messageTrend = append(messageTrend, map[string]interface{}{
				"date":  hourStr,
				"count": count,
			})
		}
	case "week":
		// 按天统计
		for i := 6; i >= 0; i-- {
			date := now.AddDate(0, 0, -i)
			dateStr := fmt.Sprintf("%d/%d", date.Month(), date.Day())

			var count int64
			db.Model(&model.Message{}).Where("sender_id = ? AND created_at >= ? AND created_at < ?",
				userID, date, date.AddDate(0, 0, 1)).Count(&count)

			messageTrend = append(messageTrend, map[string]interface{}{
				"date":  dateStr,
				"count": count,
			})
		}
	case "month":
		// 按周统计
		for i := 3; i >= 0; i-- {
			weekStart := now.AddDate(0, 0, -i*7)
			weekStr := fmt.Sprintf("第%d周", 4-i)

			var count int64
			db.Model(&model.Message{}).Where("sender_id = ? AND created_at >= ? AND created_at < ?",
				userID, weekStart, weekStart.AddDate(0, 0, 7)).Count(&count)

			messageTrend = append(messageTrend, map[string]interface{}{
				"date":  weekStr,
				"count": count,
			})
		}
	case "year":
		// 按月统计
		for i := 11; i >= 0; i-- {
			month := now.AddDate(0, -i, 0)
			monthStr := fmt.Sprintf("%d月", month.Month())

			var count int64
			db.Model(&model.Message{}).Where("sender_id = ? AND created_at >= ? AND created_at < ?",
				userID, month, month.AddDate(0, 1, 0)).Count(&count)

			messageTrend = append(messageTrend, map[string]interface{}{
				"date":  monthStr,
				"count": count,
			})
		}
	default:
		// 默认按天统计一周
		for i := 6; i >= 0; i-- {
			date := now.AddDate(0, 0, -i)
			dateStr := fmt.Sprintf("%d/%d", date.Month(), date.Day())

			var count int64
			db.Model(&model.Message{}).Where("sender_id = ? AND created_at >= ? AND created_at < ?",
				userID, date, date.AddDate(0, 0, 1)).Count(&count)

			messageTrend = append(messageTrend, map[string]interface{}{
				"date":  dateStr,
				"count": count,
			})
		}
	}

	// 生成文件类型分布数据
	fileTypes := []map[string]interface{}{
		{"type": "文档", "count": int64(120), "percentage": 35},
		{"type": "图片", "count": int64(80), "percentage": 23},
		{"type": "视频", "count": int64(60), "percentage": 17},
		{"type": "音频", "count": int64(40), "percentage": 12},
		{"type": "其他", "count": int64(45), "percentage": 13},
	}

	// 计算最大消息数
	maxMessages := int64(0)
	for _, item := range messageTrend {
		if count, ok := item["count"].(int64); ok && count > maxMessages {
			maxMessages = count
		}
	}

	// 构建响应数据
	statisticsData := map[string]interface{}{
		"totalMessages":      totalMessages,
		"totalFiles":         totalFiles,
		"totalNotes":         totalNotes,
		"totalTasks":         totalTasks,
		"completedTasks":     completedTasks,
		"pendingTasks":       pendingTasks,
		"taskCompletionRate": taskCompletionRate,
		"maxMessages":        maxMessages,
		"messageTrend":       messageTrend,
		"fileTypes":          fileTypes,
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": statisticsData,
	})
}

// 获取通知列表
func GetNotifications(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()
	var notifications []model.Notification
	db.Where("user_id = ?", userID).Order("created_at DESC").Find(&notifications)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": notifications,
	})
}

// 标记通知为已读
func MarkNotificationAsRead(c *gin.Context) {
	userID, _ := c.Get("user_id")
	notificationIDStr := c.Param("id")

	// 转换通知ID为uint
	notificationID, err := strconv.ParseUint(notificationIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的通知ID"})
		return
	}

	db := database.GetDB()
	var notification model.Notification
	if err := db.Where("id = ? AND user_id = ?", uint(notificationID), userID).First(&notification).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "通知不存在"})
		return
	}

	// 标记为已读
	notification.Read = true
	now := time.Now()
	notification.ReadAt = &now
	db.Save(&notification)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "标记已读成功",
		"data":    notification,
	})
}

// 标记所有通知为已读
func MarkAllNotificationsAsRead(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()
	now := time.Now()

	// 标记所有通知为已读
	db.Model(&model.Notification{}).Where("user_id = ? AND read = ?", userID, false).Updates(map[string]interface{}{
		"read":    true,
		"read_at": now,
	})

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "标记所有通知已读成功",
	})
}

// 设置/取消管理员
func SetMemberRole(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")
	memberIDStr := c.Param("user_id")

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	targetMemberID, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required,oneof=admin member"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	// 验证会话是否存在
	var conv model.Conversation
	if err := db.First(&conv, uint(convID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	if conv.Type != "group" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "只能为群聊设置管理员"})
		return
	}

	// 验证当前用户是否为群主
	var currentMember model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&currentMember).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "您不是群成员"})
		return
	}

	if currentMember.Role != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只有群主可以设置管理员"})
		return
	}

	// 验证目标用户是否为群成员
	var targetMember model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), uint(targetMemberID)).First(&targetMember).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "目标用户不是群成员"})
		return
	}

	if targetMember.Role == "owner" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不能修改群主角色"})
		return
	}

	// 更新角色
	targetMember.Role = req.Role
	db.Save(&targetMember)

	// 发送WebSocket通知给群成员
	if ws.GlobalHub != nil {
		updateMsg := ws.WSMessage{
			Type: "group_member_role_updated",
			Data: gin.H{
				"conversation_id": conv.ID,
				"user_id":         targetMember.UserID,
				"role":            req.Role,
			},
		}
		jsonMsg, _ := json.Marshal(updateMsg)
		ws.GlobalHub.SendToConversation(uint(convID), 0, jsonMsg)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "角色设置成功",
		"data":    targetMember,
	})
}

// 转让群主
func TransferOwner(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")
	memberIDStr := c.Param("user_id")

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	targetMemberID, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}

	db := database.GetDB()

	// 验证会话是否存在
	var conv model.Conversation
	if err := db.First(&conv, uint(convID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	if conv.Type != "group" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "只能转让群主给群聊成员"})
		return
	}

	// 验证当前用户是否为群主
	var currentMember model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&currentMember).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "您不是群成员"})
		return
	}

	if currentMember.Role != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只有群主可以转让群主"})
		return
	}

	// 验证目标用户是否为群成员
	var targetMember model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), uint(targetMemberID)).First(&targetMember).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "目标用户不是群成员"})
		return
	}

	// 开始事务
	tx := db.Begin()

	// 将当前群主降级为管理员
	currentMember.Role = "admin"
	if err := tx.Save(&currentMember).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "转让失败"})
		return
	}

	// 将目标用户升级为群主
	targetMember.Role = "owner"
	if err := tx.Save(&targetMember).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "转让失败"})
		return
	}

	// 更新会话的创建者ID
	conv.CreatorID = targetMember.UserID
	if err := tx.Save(&conv).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "转让失败"})
		return
	}

	tx.Commit()

	// 发送WebSocket通知给群成员
	if ws.GlobalHub != nil {
		transferMsg := ws.WSMessage{
			Type: "group_owner_transferred",
			Data: gin.H{
				"conversation_id": conv.ID,
				"old_owner_id":    userID,
				"new_owner_id":    targetMember.UserID,
			},
		}
		jsonMsg, _ := json.Marshal(transferMsg)
		ws.GlobalHub.SendToConversation(uint(convID), 0, jsonMsg)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "群主转让成功",
		"data":    targetMember,
	})
}

// 生成短链接
func CreateShortLink(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		OriginalURL string `json:"original_url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	// 生成短链接码
	code := generateShortCode()

	// 创建短链接记录
	shortLink := model.ShortLink{
		UserID:      userID.(uint),
		OriginalURL: req.OriginalURL,
		Code:        code,
		VisitCount:  0,
	}

	if err := db.Create(&shortLink).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "生成短链接失败"})
		return
	}

	// 构建短链接URL
	shortURL := "http://" + c.Request.Host + "/" + code

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":           shortLink.ID,
			"original_url": shortLink.OriginalURL,
			"short_url":    shortURL,
			"code":         shortLink.Code,
			"visit_count":  shortLink.VisitCount,
			"created_at":   shortLink.CreatedAt,
		},
	})
}

// 获取用户的短链接列表
func GetShortLinks(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()

	var shortLinks []model.ShortLink
	if err := db.Where("user_id = ?", userID).Order("created_at DESC").Find(&shortLinks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取短链接列表失败"})
		return
	}

	// 构建响应数据
	response := make([]gin.H, len(shortLinks))
	for i, link := range shortLinks {
		shortURL := "http://" + c.Request.Host + "/" + link.Code
		response[i] = gin.H{
			"id":           link.ID,
			"original_url": link.OriginalURL,
			"short_url":    shortURL,
			"code":         link.Code,
			"visit_count":  link.VisitCount,
			"created_at":   link.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": response,
	})
}

// 重定向短链接到原始URL
func RedirectShortLink(c *gin.Context) {
	code := c.Param("code")

	db := database.GetDB()

	// 查询短链接
	var shortLink model.ShortLink
	if err := db.Where("code = ?", code).First(&shortLink).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "短链接不存在"})
		return
	}

	// 增加访问次数
	db.Model(&shortLink).Update("visit_count", shortLink.VisitCount+1)

	// 重定向到原始URL
	c.Redirect(http.StatusFound, shortLink.OriginalURL)
}

// 生成短链接码
func generateShortCode() string {
	// 使用Base62编码生成短链接码
	const charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const codeLength = 6

	code := make([]byte, codeLength)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}

	return string(code)
}

// 删除短链接
func DeleteShortLink(c *gin.Context) {
	userID, _ := c.Get("user_id")
	linkIDStr := c.Param("id")

	linkID, err := strconv.ParseUint(linkIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的短链接ID"})
		return
	}

	db := database.GetDB()

	// 检查短链接是否存在且属于当前用户
	var shortLink model.ShortLink
	if err := db.Where("id = ? AND user_id = ?", linkID, userID).First(&shortLink).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "短链接不存在或无权操作"})
		return
	}

	// 删除短链接
	if err := db.Delete(&shortLink).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除短链接失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "短链接删除成功",
	})
}

// 更新群公告
func UpdateAnnouncement(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	// 处理带有 conv_ 前缀的会话ID
	if strings.HasPrefix(convIDStr, "conv_") {
		convIDStr = strings.TrimPrefix(convIDStr, "conv_")
	}

	// 转换会话ID为uint
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	// 解析请求体
	var req struct {
		Announcement string `json:"announcement"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	// 验证会话是否存在
	var conv model.Conversation
	if err := db.First(&conv, uint(convID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	// 验证是否为群聊
	if conv.Type != "group" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "只能为群聊设置公告"})
		return
	}

	// 验证当前用户是否为群成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "您不是群成员"})
		return
	}

	// 验证当前用户是否为群主或管理员
	if member.Role != "owner" && member.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只有群主或管理员可以设置群公告"})
		return
	}

	// 更新群公告
	conv.Announcement = req.Announcement
	if err := db.Save(&conv).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新群公告失败"})
		return
	}

	// 发送WebSocket通知给群成员
	if ws.GlobalHub != nil {
		// 构建公告更新通知
		announcementMsg := ws.WSMessage{
			Type: "group_announcement_updated",
			Data: gin.H{
				"conversation_id": conv.ID,
				"announcement":    req.Announcement,
			},
		}
		jsonMsg, _ := json.Marshal(announcementMsg)

		// 发送给会话中的所有成员
		ws.GlobalHub.SendToConversation(uint(convID), 0, jsonMsg)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "群公告更新成功", "data": conv})
}
