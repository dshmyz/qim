package handler

import (
	"strconv"

	"qim-server/di"
	"qim-server/pkg/response"
	"qim-server/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetRoles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	adminSvc := di.GlobalContainer.AdminService
	roles, total, err := adminSvc.GetRoles(service.RoleQuery{
		Page:     page,
		PageSize: pageSize,
		Keyword:  keyword,
	})
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	response.Success(c, gin.H{
		"list":  roles,
		"total": total,
	})
}

func CreateRole(c *gin.Context) {
	var req struct {
		UserID uint   `json:"user_id" binding:"required"`
		Role   string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	adminSvc := di.GlobalContainer.AdminService
	userSvc := di.GlobalContainer.UserService

	_, err := userSvc.GetUser(req.UserID)
	if err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	userRole, err := adminSvc.CreateRole(req.UserID, req.Role)
	if err != nil {
		if err == gorm.ErrDuplicatedKey {
			response.Conflict(c, "用户已有此角色")
			return
		}
		response.InternalServerError(c, "创建失败")
		return
	}

	response.Success(c, gin.H{
		"id":      userRole.ID,
		"user_id": userRole.UserID,
		"role":    userRole.Role,
	})
}

func UpdateRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的角色ID")
		return
	}

	var req struct {
		Role string `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	adminSvc := di.GlobalContainer.AdminService
	userRole, err := adminSvc.GetRole(uint(id))
	if err != nil {
		response.NotFound(c, "角色不存在")
		return
	}

	if req.Role != "" {
		userRole.Role = req.Role
		adminSvc.UpdateRole(userRole)
	}

	response.Success(c, userRole)
}

func DeleteRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的角色ID")
		return
	}

	adminSvc := di.GlobalContainer.AdminService
	if _, err := adminSvc.GetRole(uint(id)); err != nil {
		response.NotFound(c, "角色不存在")
		return
	}

	adminSvc.DeleteRole(uint(id))

	response.SuccessWithMessage(c, "删除成功", nil)
}

func GetRoleUsers(c *gin.Context) {
	role := c.Param("role")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	adminSvc := di.GlobalContainer.AdminService
	users, total, err := adminSvc.GetRoleUsers(role, page, pageSize)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	response.Success(c, gin.H{
		"list":  users,
		"total": total,
	})
}

func AdminDeleteGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群组ID")
		return
	}

	adminSvc := di.GlobalContainer.AdminService
	if err := adminSvc.DeleteGroup(uint(id)); err != nil {
		response.NotFound(c, "群组不存在")
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

func AdminGetUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	adminSvc := di.GlobalContainer.AdminService
	users, total, err := adminSvc.GetUsers(page, pageSize, keyword)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	response.Success(c, gin.H{
		"list":  users,
		"total": total,
	})
}

func AdminGetChannels(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	adminSvc := di.GlobalContainer.AdminService
	channels, total, err := adminSvc.GetChannels(page, pageSize, keyword)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	response.Success(c, gin.H{
		"list":  channels,
		"total": total,
	})
}

func AdminUpdateChannel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的频道ID")
		return
	}

	var req struct {
		Name              *string `json:"name"`
		Description       *string `json:"description"`
		Avatar            *string `json:"avatar"`
		Status            *string `json:"status"`
		PublishPermission *string `json:"publish_permission"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	adminSvc := di.GlobalContainer.AdminService
	if _, err := adminSvc.GetChannel(uint(id)); err != nil {
		response.NotFound(c, "频道不存在")
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}
	if req.Status != nil {
		if *req.Status != "active" && *req.Status != "inactive" {
			response.BadRequest(c, "无效的状态值")
			return
		}
		updates["status"] = *req.Status
	}
	if req.PublishPermission != nil {
		if *req.PublishPermission != "creator_only" && *req.PublishPermission != "all_subscribers" {
			response.BadRequest(c, "无效的发布权限")
			return
		}
		updates["publish_permission"] = *req.PublishPermission
	}

	if len(updates) > 0 {
		if err := adminSvc.UpdateChannel(uint(id), updates); err != nil {
			response.InternalServerError(c, "更新失败")
			return
		}
	}

	channel, _ := adminSvc.GetChannel(uint(id))
	response.Success(c, channel)
}

func AdminDeleteChannel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的频道ID")
		return
	}

	adminSvc := di.GlobalContainer.AdminService
	if err := adminSvc.DeleteChannel(uint(id)); err != nil {
		response.NotFound(c, "频道不存在")
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

func AdminGetGroups(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	adminSvc := di.GlobalContainer.AdminService
	groups, total, err := adminSvc.GetGroups(page, pageSize, keyword)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	response.Success(c, gin.H{
		"list":  groups,
		"total": total,
	})
}

func AdminGetStatistics(c *gin.Context) {
	adminSvc := di.GlobalContainer.AdminService
	stats, err := adminSvc.GetStatistics()
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	response.Success(c, gin.H{
		"totalUsers":    stats.TotalUsers,
		"onlineUsers":   stats.OnlineUsers,
		"totalGroups":   stats.TotalGroups,
		"totalChannels": stats.TotalChannels,
		"totalMessages": stats.TotalMessages,
		"activeUsers":   stats.ActiveUsers,
		"messagesToday": stats.MessagesToday,
		"growthRate": gin.H{
			"users":    12.5,
			"groups":   8.3,
			"messages": 15.7,
		},
	})
}

func AdminGetRecentRegistrations(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	adminSvc := di.GlobalContainer.AdminService
	registrations, total, err := adminSvc.GetRecentRegistrations(page, pageSize)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	response.Success(c, gin.H{
		"list":  registrations,
		"total": total,
	})
}

// AdminGetUserAIConfigs 管理员获取指定用户的AI配置列表
func AdminGetUserAIConfigs(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	svc := di.GlobalContainer.AIConfigService
	configs, total, err := svc.ListUserConfigs(uint(userID), page, pageSize)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	responses := make([]ConfigResponse, len(configs))
	for i, cfg := range configs {
		responses[i] = toConfigResponse(cfg)
	}

	response.Success(c, gin.H{
		"list":     responses,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// AdminUpdateUserAIConfig 管理员更新指定用户的AI配置
func AdminUpdateUserAIConfig(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	configIDStr := c.Param("configId")
	configID, err := strconv.ParseUint(configIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的配置ID")
		return
	}

	var req struct {
		ConfigName  string  `json:"config_name"`
		Provider    string  `json:"provider"`
		APIKey      string  `json:"api_key"`
		ModelName   string  `json:"model_name"`
		BaseURL     string  `json:"base_url"`
		AIEnabled   *bool   `json:"ai_enabled"`
		DailyLimit  *int    `json:"daily_limit"`
		MaxTokens   *int    `json:"max_tokens"`
		Temperature *float64 `json:"temperature"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	svc := di.GlobalContainer.AIConfigService
	config, err := svc.UpdateConfig(uint(userID), uint(configID), req.ConfigName, req.Provider, req.APIKey, req.ModelName, req.BaseURL)
	if err != nil {
		response.InternalServerError(c, "更新配置失败")
		return
	}

	// 更新额外的配置字段
	db := di.GlobalContainer.DB
	updates := make(map[string]interface{})
	if req.AIEnabled != nil {
		updates["ai_enabled"] = *req.AIEnabled
	}
	if req.DailyLimit != nil {
		updates["daily_limit"] = *req.DailyLimit
	}
	if req.MaxTokens != nil {
		updates["max_tokens"] = *req.MaxTokens
	}
	if req.Temperature != nil {
		updates["temperature"] = *req.Temperature
	}
	if len(updates) > 0 {
		db.Model(config).Updates(updates)
	}

	response.Success(c, toConfigResponse(*config))
}
