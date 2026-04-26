package handler

import (
	"strconv"

	"qim-server/database"
	"qim-server/model"
	"qim-server/pkg/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func GetCurrentUser(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()
	var user model.User
	if err := db.First(&user, userID).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	response.Success(c, gin.H{
		"id":                 user.ID,
		"username":           user.Username,
		"nickname":           user.Nickname,
		"avatar":             user.Avatar,
		"signature":          user.Signature,
		"phone":              user.Phone,
		"email":              user.Email,
		"status":             user.Status,
		"two_factor_enabled": user.TwoFactorEnabled,
	})
}

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
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()
	var user model.User
	if err := db.First(&user, userID).Error; err != nil {
		response.NotFound(c, "用户不存在")
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

	response.Success(c, user)
}

func SearchUsers(c *gin.Context) {
	userID, _ := c.Get("user_id")
	query := c.Query("q")

	if query == "" {
		response.BadRequest(c, "搜索关键词不能为空")
		return
	}

	db := database.GetDB()

	var users []model.User
	db.Where("nickname LIKE ? OR username LIKE ?", "%"+query+"%", "%"+query+"%").Find(&users)

	var responseUsers []gin.H
	for _, user := range users {
		if user.ID != userID.(uint) {
			responseUsers = append(responseUsers, gin.H{
				"id":       user.ID,
				"type":     "user",
				"name":     user.Nickname,
				"username": user.Username,
				"avatar":   user.Avatar,
				"status":   user.Status,
			})
		}
	}

	var conversations []model.Conversation
	db.Where("name LIKE ? AND (type = ? OR type = ?) AND deleted_at IS NULL", "%"+query+"%", "group", "discussion").Find(&conversations)

	for _, conv := range conversations {
		var member model.ConversationMember
		isMember := false
		if err := db.Where("conversation_id = ? AND user_id = ?", conv.ID, userID).First(&member).Error; err == nil {
			isMember = true
		}

		responseUsers = append(responseUsers, gin.H{
			"id":       conv.ID,
			"type":     conv.Type,
			"name":     conv.Name,
			"avatar":   conv.Avatar,
			"isMember": isMember,
		})
	}

	response.Success(c, responseUsers)
}

func CreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
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
		response.InternalServerError(c, "创建用户失败")
		return
	}

	response.Success(c, user)
}

func GetUser(c *gin.Context) {
	username := c.Query("username")

	if username == "" {
		response.BadRequest(c, "用户名不能为空")
		return
	}

	db := database.GetDB()
	var user model.User
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	response.Success(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"nickname": user.Nickname,
		"avatar":   user.Avatar,
		"status":   user.Status,
	})
}

func GetUserByID(c *gin.Context) {
	userIDStr := c.Param("id")

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	db := database.GetDB()
	var user model.User
	if err := db.First(&user, uint(userID)).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	response.Success(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"nickname": user.Nickname,
		"avatar":   user.Avatar,
		"status":   user.Status,
	})
}

func AddUserRole(c *gin.Context) {
	userIDStr := c.Param("id")
	targetUserID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	var targetUser model.User
	if err := db.First(&targetUser, uint(targetUserID)).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	var existingRole model.UserRole
	result := db.Where("user_id = ? AND role = ?", uint(targetUserID), req.Role).First(&existingRole)
	if result.Error == nil {
		response.BadRequest(c, "角色已存在")
		return
	}

	userRole := model.UserRole{
		UserID: uint(targetUserID),
		Role:   req.Role,
	}

	if err := db.Create(&userRole).Error; err != nil {
		response.InternalServerError(c, "添加角色失败")
		return
	}

	response.Success(c, gin.H{
		"message": "角色添加成功",
		"data":    userRole,
	})
}

func RemoveUserRole(c *gin.Context) {
	userIDStr := c.Param("id")
	targetUserID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	role := c.Param("role")
	if role == "" {
		response.BadRequest(c, "角色不能为空")
		return
	}

	db := database.GetDB()

	var targetUser model.User
	if err := db.First(&targetUser, uint(targetUserID)).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	var existingRole model.UserRole
	result := db.Where("user_id = ? AND role = ?", uint(targetUserID), role).First(&existingRole)
	if result.Error != nil {
		response.BadRequest(c, "角色不存在")
		return
	}

	if err := db.Delete(&existingRole).Error; err != nil {
		response.InternalServerError(c, "移除角色失败")
		return
	}

	response.Success(c, gin.H{
		"message": "角色移除成功",
		"data":    existingRole,
	})
}

// BatchAssignUserRoles 批量分配用户角色
func BatchAssignUserRoles(c *gin.Context) {
	userIDStr := c.Param("id")
	targetUserID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	var req struct {
		Roles []string `json:"roles" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	var targetUser model.User
	if err := db.First(&targetUser, uint(targetUserID)).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	// 先删除用户所有现有角色
	db.Where("user_id = ?", uint(targetUserID)).Delete(&model.UserRole{})

	// 添加新角色
	for _, roleName := range req.Roles {
		userRole := model.UserRole{
			UserID: uint(targetUserID),
			Role:   roleName,
		}
		db.Create(&userRole)
	}

	response.Success(c, gin.H{
		"message": "角色分配成功",
		"roles":   req.Roles,
	})
}

// DeleteUser 删除用户
func DeleteUser(c *gin.Context) {
	userIDStr := c.Param("id")
	targetUserID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	db := database.GetDB()

	var targetUser model.User
	if err := db.First(&targetUser, uint(targetUserID)).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	// 删除用户关联的角色
	db.Where("user_id = ?", uint(targetUserID)).Delete(&model.UserRole{})

	// 删除用户（软删除）
	if err := db.Delete(&targetUser).Error; err != nil {
		response.InternalServerError(c, "删除用户失败")
		return
	}

	response.Success(c, gin.H{
		"message": "删除成功",
	})
}

func GetAIConfig(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()
	var aiConfig model.AIConfig
	if err := db.Where("user_id = ?", userID).First(&aiConfig).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Success(c, nil)
			return
		}
		response.InternalServerError(c, "查询失败")
		return
	}

	response.Success(c, aiConfig)
}

func UpdateAIConfig(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Provider         string  `json:"provider"`
		OpenAIAPIKey     string  `json:"openai_api_key"`
		OpenAIModel      string  `json:"openai_model"`
		OpenAIBaseURL    string  `json:"openai_base_url"`
		BaiduAPIKey      string  `json:"baidu_api_key"`
		BaiduSecretKey   string  `json:"baidu_secret_key"`
		BaiduModel       string  `json:"baidu_model"`
		BaiduBaseURL     string  `json:"baidu_base_url"`
		AlibabaAPIKey    string  `json:"alibaba_api_key"`
		AlibabaModel     string  `json:"alibaba_model"`
		AlibabaBaseURL   string  `json:"alibaba_base_url"`
		TencentSecretID  string  `json:"tencent_secret_id"`
		TencentSecretKey string  `json:"tencent_secret_key"`
		TencentModel     string  `json:"tencent_model"`
		TencentBaseURL   string  `json:"tencent_base_url"`
		BytedanceAPIKey  string  `json:"bytedance_api_key"`
		BytedanceModel   string  `json:"bytedance_model"`
		BytedanceBaseURL string  `json:"bytedance_base_url"`
		AnthropicAPIKey  string  `json:"anthropic_api_key"`
		AnthropicModel   string  `json:"anthropic_model"`
		AnthropicBaseURL string  `json:"anthropic_base_url"`
		MaxTokens        int     `json:"max_tokens"`
		Temperature      float64 `json:"temperature"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()
	var aiConfig model.AIConfig
	if err := db.Where("user_id = ?", userID).First(&aiConfig).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			aiConfig = model.AIConfig{
				UserID: userID.(uint),
			}
		} else {
			response.InternalServerError(c, "查询失败")
			return
		}
	}

	if req.Provider != "" {
		aiConfig.Provider = req.Provider
	}
	if req.OpenAIAPIKey != "" {
		aiConfig.OpenAIAPIKey = req.OpenAIAPIKey
	}
	if req.OpenAIModel != "" {
		aiConfig.OpenAIModel = req.OpenAIModel
	}
	if req.OpenAIBaseURL != "" {
		aiConfig.OpenAIBaseURL = req.OpenAIBaseURL
	}
	if req.BaiduAPIKey != "" {
		aiConfig.BaiduAPIKey = req.BaiduAPIKey
	}
	if req.BaiduSecretKey != "" {
		aiConfig.BaiduSecretKey = req.BaiduSecretKey
	}
	if req.BaiduModel != "" {
		aiConfig.BaiduModel = req.BaiduModel
	}
	if req.BaiduBaseURL != "" {
		aiConfig.BaiduBaseURL = req.BaiduBaseURL
	}
	if req.AlibabaAPIKey != "" {
		aiConfig.AlibabaAPIKey = req.AlibabaAPIKey
	}
	if req.AlibabaModel != "" {
		aiConfig.AlibabaModel = req.AlibabaModel
	}
	if req.AlibabaBaseURL != "" {
		aiConfig.AlibabaBaseURL = req.AlibabaBaseURL
	}
	if req.TencentSecretID != "" {
		aiConfig.TencentSecretID = req.TencentSecretID
	}
	if req.TencentSecretKey != "" {
		aiConfig.TencentSecretKey = req.TencentSecretKey
	}
	if req.TencentModel != "" {
		aiConfig.TencentModel = req.TencentModel
	}
	if req.TencentBaseURL != "" {
		aiConfig.TencentBaseURL = req.TencentBaseURL
	}
	if req.BytedanceAPIKey != "" {
		aiConfig.BytedanceAPIKey = req.BytedanceAPIKey
	}
	if req.BytedanceModel != "" {
		aiConfig.BytedanceModel = req.BytedanceModel
	}
	if req.BytedanceBaseURL != "" {
		aiConfig.BytedanceBaseURL = req.BytedanceBaseURL
	}
	if req.AnthropicAPIKey != "" {
		aiConfig.AnthropicAPIKey = req.AnthropicAPIKey
	}
	if req.AnthropicModel != "" {
		aiConfig.AnthropicModel = req.AnthropicModel
	}
	if req.AnthropicBaseURL != "" {
		aiConfig.AnthropicBaseURL = req.AnthropicBaseURL
	}
	if req.MaxTokens > 0 {
		aiConfig.MaxTokens = req.MaxTokens
	}
	if req.Temperature > 0 {
		aiConfig.Temperature = req.Temperature
	}

	if aiConfig.ID == 0 {
		if err := db.Create(&aiConfig).Error; err != nil {
			response.InternalServerError(c, "创建失败")
			return
		}
	} else {
		if err := db.Save(&aiConfig).Error; err != nil {
			response.InternalServerError(c, "更新失败")
			return
		}
	}

	response.Success(c, aiConfig)
}
