package handler

import (
	"strconv"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"

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

// AdminCreateGroup 管理员创建群组
func AdminCreateGroup(c *gin.Context) {
	var input service.AdminCreateGroupInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "参数无效: "+err.Error())
		return
	}

	adminSvc := di.GlobalContainer.AdminService
	conv, err := adminSvc.CreateGroup(input)
	if err != nil {
		response.InternalServerError(c, "创建群组失败")
		return
	}

	response.SuccessWithMessage(c, "创建成功", gin.H{
		"id":              conv.ID,
		"conversationId":  conv.ID,
		"type":            conv.Type,
	})
}

// AdminUpdateGroup 管理员更新群组
func AdminUpdateGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群组ID")
		return
	}

	var input service.AdminUpdateGroupInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "参数无效: "+err.Error())
		return
	}

	adminSvc := di.GlobalContainer.AdminService
	if err := adminSvc.UpdateGroup(uint(id), input); err != nil {
		response.NotFound(c, "群组不存在")
		return
	}

	response.SuccessWithMessage(c, "更新成功", nil)
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
		CommentPermission *string `json:"comment_permission"`
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
	if req.CommentPermission != nil {
		if *req.CommentPermission != "all_subscribers" && *req.CommentPermission != "disabled" {
			response.BadRequest(c, "无效的评论权限")
			return
		}
		updates["comment_permission"] = *req.CommentPermission
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
			"users":    stats.GrowthRate.Users,
			"groups":   stats.GrowthRate.Groups,
			"messages": stats.GrowthRate.Messages,
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

// AdminGetDashboardTrend 获取仪表盘趋势数据（用户增长 + 消息活跃度）
func AdminGetDashboardTrend(c *gin.Context) {
	db := database.GetDB()
	now := time.Now()

	// 用户增长趋势：最近 7 天每天新注册用户数
	userTrend := make([]gin.H, 0, 7)
	for i := 6; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
		dayEnd := dayStart.AddDate(0, 0, 1)

		var count int64
		db.Model(&model.User{}).Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).Count(&count)

		weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
		userTrend = append(userTrend, gin.H{
			"label": weekdays[day.Weekday()],
			"value": count,
		})
	}

	// 消息活跃度：今天按时段统计
	type Period struct {
		Label string
		Start int
		End   int
	}
	periods := []Period{
		{"上午", 6, 12},
		{"下午", 12, 18},
		{"晚间", 18, 24},
		{"凌晨", 0, 6},
	}

	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	activityData := make([]gin.H, 0, len(periods))
	var maxActivity int64 = 1
	for _, p := range periods {
		start := todayStart.Add(time.Duration(p.Start) * time.Hour)
		end := todayStart.Add(time.Duration(p.End) * time.Hour)

		var count int64
		db.Model(&model.Message{}).Where("created_at >= ? AND created_at < ?", start, end).Count(&count)
		if count > maxActivity {
			maxActivity = count
		}
		activityData = append(activityData, gin.H{
			"label": p.Label,
			"value": count,
		})
	}
	// 计算百分比
	for _, item := range activityData {
		count := item["value"].(int64)
		item["percent"] = int(count * 100 / maxActivity)
	}

	// 计算用户趋势百分比
	var maxUser int64 = 1
	for _, item := range userTrend {
		if v := item["value"].(int64); v > maxUser {
			maxUser = v
		}
	}
	for _, item := range userTrend {
		count := item["value"].(int64)
		if maxUser > 0 {
			item["percent"] = int(count * 100 / maxUser)
		} else {
			item["percent"] = 0
		}
	}

	response.Success(c, gin.H{
		"userTrend":    userTrend,
		"activityData": activityData,
	})
}

// AdminGetStatisticsTrend 获取统计页趋势数据
// 返回最近 7 天的用户增长趋势、消息发送趋势，以及今日活动数据
func AdminGetStatisticsTrend(c *gin.Context) {
	db := database.GetDB()
	now := time.Now()

	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

	// 用户增长趋势：最近 7 天每天新注册用户数
	userTrend := make([]gin.H, 0, 7)
	// 消息发送趋势：最近 7 天每天的消息数
	messageTrend := make([]gin.H, 0, 7)
	for i := 6; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
		dayEnd := dayStart.AddDate(0, 0, 1)

		var userCount int64
		db.Model(&model.User{}).Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).Count(&userCount)

		var msgCount int64
		db.Model(&model.Message{}).Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).Count(&msgCount)

		userTrend = append(userTrend, gin.H{
			"label": weekdays[day.Weekday()],
			"value": userCount,
		})
		messageTrend = append(messageTrend, gin.H{
			"label": weekdays[day.Weekday()],
			"value": msgCount,
		})
	}

	// 今日活动数据：登录次数、消息发送、群组创建、频道订阅
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrowStart := todayStart.AddDate(0, 0, 1)

	var activeTodayCount, msgTodayCount, groupCreatedCount, channelSubCount int64
	// 今日活跃用户：今天发过消息的不同用户数
	db.Model(&model.Message{}).Where("created_at >= ? AND created_at < ?", todayStart, tomorrowStart).Distinct("sender_id").Count(&activeTodayCount)
	db.Model(&model.Message{}).Where("created_at >= ? AND created_at < ?", todayStart, tomorrowStart).Count(&msgTodayCount)
	db.Model(&model.Conversation{}).Where("type = ? AND created_at >= ? AND created_at < ?", "group", todayStart, tomorrowStart).Count(&groupCreatedCount)
	db.Model(&model.Channel{}).Where("created_at >= ? AND created_at < ?", todayStart, tomorrowStart).Count(&channelSubCount)

	activityData := []gin.H{
		{"label": "活跃用户", "value": activeTodayCount},
		{"label": "消息发送", "value": msgTodayCount},
		{"label": "群组创建", "value": groupCreatedCount},
		{"label": "频道订阅", "value": channelSubCount},
	}

	response.Success(c, gin.H{
		"userTrend":     userTrend,
		"messageTrend":  messageTrend,
		"activityData":  activityData,
	})
}

// AdminGetDashboardStats 获取仪表盘统计数据
func AdminGetDashboardStats(c *gin.Context) {
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
		"totalMessages": stats.TotalMessages,
	})
}
