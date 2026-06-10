package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/params"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	userService *service.UserService
	convService *service.ConversationService
}

func NewUserHandler(userService *service.UserService, convService *service.ConversationService) *UserHandler {
	return &UserHandler{
		userService: userService,
		convService: convService,
	}
}

func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	uid, ok := params.GetUserID(c)
	if !ok {
		return
	}

	user, err := h.userService.GetUser(uid)
	if err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	roleNames, _ := h.userService.GetUserRoles(user.ID)

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
		"roles":              roleNames,
	})
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	uid, ok := params.GetUserID(c)
	if !ok {
		return
	}

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

	updates := make(map[string]interface{})
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.Avatar != "" {
		updates["avatar"] = req.Avatar
	}
	if req.Signature != "" {
		updates["signature"] = req.Signature
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.TwoFactorEnabled != nil {
		updates["two_factor_enabled"] = *req.TwoFactorEnabled
	}

	user, err := h.userService.UpdateUser(uid, updates)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, user)
}

func (h *UserHandler) SearchUsers(c *gin.Context) {
	uid, ok := params.GetUserID(c)
	if !ok {
		return
	}
	query := c.Query("q")

	if query == "" {
		response.BadRequest(c, "搜索关键词不能为空")
		return
	}

	users, err := h.userService.SearchUsers(query, 20)
	if err != nil {
		response.InternalServerError(c, "搜索失败")
		return
	}

	var responseUsers []gin.H
	for _, user := range users {
		if user.ID != uid {
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

	groups, err := h.convService.SearchGroupsByName(query, uid)
	if err == nil {
		for _, g := range groups {
			responseUsers = append(responseUsers, gin.H{
				"id":       g.ConversationID,
				"type":     g.ConvType,
				"name":     g.Name,
				"avatar":   g.Avatar,
				"isMember": g.IsMember,
			})
		}
	}

	response.Success(c, responseUsers)
}

// 向后兼容的包装函数
func GetCurrentUser(c *gin.Context) {
	handler := NewUserHandler(di.GlobalContainer.UserService, di.GlobalContainer.ConversationService)
	handler.GetCurrentUser(c)
}

func UpdateUser(c *gin.Context) {
	handler := NewUserHandler(di.GlobalContainer.UserService, di.GlobalContainer.ConversationService)
	handler.UpdateUser(c)
}

func SearchUsers(c *gin.Context) {
	handler := NewUserHandler(di.GlobalContainer.UserService, di.GlobalContainer.ConversationService)
	handler.SearchUsers(c)
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

	// 密码强度校验
	if err := validatePassword(req.Password); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	db := di.GlobalContainer.DB

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
		avatar = ""
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

	db := di.GlobalContainer.DB
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

	db := di.GlobalContainer.DB
	var user model.User
	if err := db.First(&user, uint(userID)).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	var departmentName string
	var position string
	var deptEmployee model.DepartmentEmployee
	if err := db.Where("user_id = ? AND is_primary = ?", user.ID, true).Preload("Department").First(&deptEmployee).Error; err == nil {
		if deptEmployee.Department.ID > 0 {
			departmentName = deptEmployee.Department.Name
		}
		position = deptEmployee.Position
	}

	response.Success(c, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"nickname":   user.Nickname,
		"avatar":     user.Avatar,
		"status":     user.Status,
		"email":      user.Email,
		"phone":      user.Phone,
		"signature":  user.Signature,
		"ip":         user.IP,
		"department": departmentName,
		"position":   position,
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

	db := di.GlobalContainer.DB

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

	db := di.GlobalContainer.DB

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

	db := di.GlobalContainer.DB

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
	userID, ok := params.GetUserID(c)
	if !ok {
		return
	}

	svc := di.GlobalContainer.AIConfigService
	config, err := svc.GetDefaultConfig(userID)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	response.Success(c, config)
}

func UpdateAIConfig(c *gin.Context) {
	userID, ok := params.GetUserID(c)
	if !ok {
		return
	}

	var req struct {
		Provider    string  `json:"provider" binding:"required"`
		APIKey      string  `json:"api_key"`
		SecretKey   string  `json:"secret_key"`
		SecretID    string  `json:"secret_id"`
		Model       string  `json:"model"`
		BaseURL     string  `json:"base_url"`
		MaxTokens   int     `json:"max_tokens"`
		Temperature float64 `json:"temperature"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	svc := di.GlobalContainer.AIConfigService
	config, err := svc.UpdateDefaultConfig(userID, req.Provider, req.APIKey, req.SecretKey, req.SecretID, req.Model, req.BaseURL, req.MaxTokens, req.Temperature)
	if err != nil {
		if errors.Is(err, service.ErrUnsupportedProvider) {
			response.BadRequest(c, "不支持的供应商")
			return
		}
		response.InternalServerError(c, "更新失败")
		return
	}

	response.Success(c, gin.H{
		"id":          config.ID,
		"user_id":     config.UserID,
		"provider":    config.Provider,
		"ai_enabled":  config.AIEnabled,
		"daily_limit": config.DailyLimit,
		"max_tokens":  config.MaxTokens,
		"temperature": config.Temperature,
		"created_at":  config.CreatedAt,
		"updated_at":  config.UpdatedAt,
	})
}

// GetUserStatus 查询用户在线状态
// 兼容：
//   - 单个：?user_id=1  返回对象 { user_id, status, last_online }
//   - 批量：?user_ids=1,2,3  返回数组 [{...}, {...}]
func GetUserStatus(c *gin.Context) {
	singleIDStr := c.Query("user_id")
	batchIDsStr := c.Query("user_ids")

	if singleIDStr == "" && batchIDsStr == "" {
		response.BadRequest(c, "user_id 或 user_ids 至少传一个")
		return
	}

	var userIDs []uint
	if batchIDsStr != "" {
		for _, idStr := range strings.Split(batchIDsStr, ",") {
			id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 32)
			if err != nil {
				response.BadRequest(c, "无效的用户ID")
				return
			}
			userIDs = append(userIDs, uint(id))
		}
	} else {
		id, err := strconv.ParseUint(singleIDStr, 10, 32)
		if err != nil {
			response.BadRequest(c, "无效的用户ID")
			return
		}
		userIDs = append(userIDs, uint(id))
	}

	db := di.GlobalContainer.DB
	var users []model.User
	if err := db.Select("id", "status", "last_online").
		Where("id IN ?", userIDs).
		Find(&users).Error; err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	buildItem := func(user model.User) gin.H {
		return gin.H{
			"user_id": user.ID,
			"status":  user.Status,
			"last_online": func() int64 {
				if user.LastOnline != nil {
					return user.LastOnline.Unix()
				}
				return 0
			}(),
		}
	}

	// 单个查询保持向后兼容：返回对象（不存在时 404）
	if singleIDStr != "" && batchIDsStr == "" {
		if len(users) == 0 {
			response.NotFound(c, "用户不存在")
			return
		}
		response.Success(c, buildItem(users[0]))
		return
	}

	result := make([]gin.H, 0, len(users))
	for _, user := range users {
		result = append(result, buildItem(user))
	}
	response.Success(c, result)
}
