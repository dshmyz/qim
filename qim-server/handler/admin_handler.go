package handler

import (
	"net/http"
	"strconv"

	"qim-server/database"
	"qim-server/model"

	"github.com/gin-gonic/gin"
)

// Role 角色结构（用于API响应）
type Role struct {
	ID          uint     `json:"id"`
	Name        string   `json:"name"`
	Code        string   `json:"code"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	UserCount   int64    `json:"userCount"`
	CreatedAt   string   `json:"createdAt"`
}

// GetRoles 获取所有角色
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

	db := database.GetDB()

	query := db.Model(&model.UserRole{})
	if keyword != "" {
		query = query.Where("role LIKE ?", "%"+keyword+"%")
	}

	var total int64
	query.Count(&total)

	var userRoles []model.UserRole
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&userRoles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询失败"})
		return
	}

	// 按角色分组，统计每个角色的用户数
	roleMap := make(map[string]int64)
	for _, ur := range userRoles {
		roleMap[ur.Role]++
	}

	// 定义角色列表
	roleDefinitions := []struct {
		Code        string
		Name        string
		Description string
	}{
		{"system_admin", "系统管理员", "拥有系统全部权限"},
		{"system_publisher", "系统发布者", "可以发布系统消息"},
		{"user_manager", "用户管理员", "可以管理用户"},
		{"group_manager", "群组管理员", "可以管理群组"},
		{"channel_manager", "频道管理员", "可以管理频道"},
	}

	roles := make([]Role, 0, len(roleDefinitions))
	for _, rd := range roleDefinitions {
		if keyword != "" && (rd.Code != keyword && rd.Name != keyword) {
			continue
		}
		role := Role{
			ID:          uint(len(roles) + 1),
			Name:        rd.Name,
			Code:        rd.Code,
			Description: rd.Description,
			Permissions: getPermissionsByRole(rd.Code),
			UserCount:   roleMap[rd.Code],
			CreatedAt:   "",
		}
		roles = append(roles, role)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  roles,
			"total": int64(len(roles)),
		},
	})
}

func getPermissionsByRole(roleCode string) []string {
	switch roleCode {
	case "system_admin":
		return []string{"user:read", "user:create", "user:update", "user:delete",
			"group:read", "group:create", "group:update", "group:delete",
			"role:read", "role:create", "role:update", "role:delete",
			"system:config", "system:log"}
	case "system_publisher":
		return []string{"message:write", "system:log"}
	case "user_manager":
		return []string{"user:read", "user:create", "user:update"}
	case "group_manager":
		return []string{"group:read", "group:create", "group:update", "group:delete"}
	case "channel_manager":
		return []string{"channel:read", "channel:create", "channel:update", "channel:delete"}
	default:
		return []string{}
	}
}

// CreateRole 创建角色（实际上是在用户身上添加角色）
func CreateRole(c *gin.Context) {
	var req struct {
		UserID   uint   `json:"user_id" binding:"required"`
		Role     string `json:"role" binding:"required"`
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

	// 检查角色是否已存在
	var existing model.UserRole
	if err := db.Where("user_id = ? AND role = ?", req.UserID, req.Role).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "用户已有此角色"})
		return
	}

	userRole := model.UserRole{
		UserID: req.UserID,
		Role:   req.Role,
	}

	if err := db.Create(&userRole).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":      userRole.ID,
			"user_id": userRole.UserID,
			"role":    userRole.Role,
		},
	})
}

// UpdateRole 更新角色
func UpdateRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的角色ID"})
		return
	}

	var req struct {
		Role string `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	var userRole model.UserRole
	if err := db.First(&userRole, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "角色不存在"})
		return
	}

	if req.Role != "" {
		userRole.Role = req.Role
		db.Save(&userRole)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": userRole,
	})
}

// DeleteRole 删除角色
func DeleteRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的角色ID"})
		return
	}

	db := database.GetDB()

	var userRole model.UserRole
	if err := db.First(&userRole, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "角色不存在"})
		return
	}

	db.Delete(&userRole)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// GetRoleUsers 获取角色的用户列表
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

	db := database.GetDB()

	var userRoles []model.UserRole
	var total int64
	db.Model(&model.UserRole{}).Where("role = ?", role).Count(&total)

	offset := (page - 1) * pageSize
	if err := db.Where("role = ?", role).Offset(offset).Limit(pageSize).Find(&userRoles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询失败"})
		return
	}

	// 获取用户信息
	type UserInfo struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
	}

	users := make([]UserInfo, 0, len(userRoles))
	for _, ur := range userRoles {
		var user model.User
		if err := db.First(&user, ur.UserID).Error; err == nil {
			users = append(users, UserInfo{
				ID:       user.ID,
				Username: user.Username,
				Nickname: user.Nickname,
				Avatar:   user.Avatar,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  users,
			"total": total,
		},
	})
}

// AdminDeleteGroup 管理员删除群组
func AdminDeleteGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的群组ID"})
		return
	}

	db := database.GetDB()

	var conversation model.Conversation
	if err := db.First(&conversation, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "群组不存在"})
		return
	}

	// 标记为删除
	conversation.IsDeleted = true
	db.Save(&conversation)

	// 删除所有成员关联
	db.Where("conversation_id = ?", uint(id)).Delete(&model.ConversationMember{})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// AdminGetUsers 管理员获取所有用户列表（分页）
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

	db := database.GetDB()

	query := db.Model(&model.User{})
	if keyword != "" {
		query = query.Where("username LIKE ? OR nickname LIKE ? OR email LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	query.Count(&total)

	var users []model.User
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询失败"})
		return
	}

	// 获取每个用户的角色
	type UserWithRoles struct {
		model.User
		Roles []string `json:"roles"`
	}

	responseUsers := make([]UserWithRoles, 0, len(users))
	for _, user := range users {
		var roles []model.UserRole
		db.Where("user_id = ?", user.ID).Find(&roles)
		roleNames := make([]string, 0, len(roles))
		for _, role := range roles {
			roleNames = append(roleNames, role.Role)
		}
		responseUsers = append(responseUsers, UserWithRoles{
			User:  user,
			Roles: roleNames,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  responseUsers,
			"total": total,
		},
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

	db := database.GetDB()

	query := db.Model(&model.Channel{})
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}

	var total int64
	query.Count(&total)

	var channels []model.Channel
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Preload("Creator").Order("id DESC").Find(&channels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询失败"})
		return
	}

	type ChannelInfo struct {
		ID                uint   `json:"id"`
		Name              string `json:"name"`
		Avatar            string `json:"avatar"`
		Description       string `json:"description"`
		Status            string `json:"status"`
		PublishPermission string `json:"publish_permission"`
		CreatorName       string `json:"creatorName"`
		MemberCount       int64  `json:"memberCount"`
		CreatedAt         string `json:"createdAt"`
	}

	channelInfos := make([]ChannelInfo, 0, len(channels))
	for _, ch := range channels {
		var memberCount int64
		db.Model(&model.ChannelSubscriber{}).Where("channel_id = ?", ch.ID).Count(&memberCount)

		creatorName := ""
		if ch.Creator.ID > 0 {
			creatorName = ch.Creator.Nickname
			if creatorName == "" {
				creatorName = ch.Creator.Username
			}
		}

		channelInfos = append(channelInfos, ChannelInfo{
			ID:                ch.ID,
			Name:              ch.Name,
			Avatar:            ch.Avatar,
			Description:       ch.Description,
			Status:            ch.Status,
			PublishPermission: ch.PublishPermission,
			CreatorName:       creatorName,
			MemberCount:       memberCount,
			CreatedAt:         ch.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  channelInfos,
			"total": total,
		},
	})
}

func AdminUpdateChannel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的频道ID"})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	var channel model.Channel
	if err := db.First(&channel, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "频道不存在"})
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
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的状态值"})
			return
		}
		updates["status"] = *req.Status
	}
	if req.PublishPermission != nil {
		if *req.PublishPermission != "creator_only" && *req.PublishPermission != "all_subscribers" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的发布权限"})
			return
		}
		updates["publish_permission"] = *req.PublishPermission
	}

	if len(updates) > 0 {
		if err := db.Model(&channel).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
			return
		}
	}

	db.Preload("Creator").First(&channel, uint(id))
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": channel,
	})
}

func AdminDeleteChannel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的频道ID"})
		return
	}

	db := database.GetDB()

	var channel model.Channel
	if err := db.First(&channel, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "频道不存在"})
		return
	}

	if err := db.Delete(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	db.Where("channel_id = ?", uint(id)).Delete(&model.ChannelSubscriber{})
	db.Where("channel_id = ?", uint(id)).Delete(&model.ChannelMessage{})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// AdminGetGroups 管理员获取所有群组列表（分页）
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

	db := database.GetDB()

	query := db.Model(&model.Conversation{}).Where("type IN ?", []string{"group", "discussion"})
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}

	var total int64
	query.Count(&total)

	var conversations []model.Conversation
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&conversations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询失败"})
		return
	}

	type GroupInfo struct {
		ID          uint   `json:"id"`
		Name        string `json:"name"`
		Avatar      string `json:"avatar"`
		Description string `json:"description"`
		Type        string `json:"type"`
		Status      string `json:"status"`
		MemberCount int64  `json:"memberCount"`
		CreatedAt   string `json:"createdAt"`
	}

	groups := make([]GroupInfo, 0, len(conversations))
	for _, conv := range conversations {
		var memberCount int64
		db.Model(&model.ConversationMember{}).Where("conversation_id = ?", conv.ID).Count(&memberCount)

		// 获取群聊信息
		var group model.Group
		db.Where("conversation_id = ?", conv.ID).First(&group)

		status := "active"
		if conv.IsDeleted {
			status = "inactive"
		}

		groups = append(groups, GroupInfo{
			ID:          conv.ID,
			Name:        group.Name,
			Avatar:      group.Avatar,
			Description: group.Announcement,
			Type:        conv.Type,
			Status:      status,
			MemberCount: memberCount,
			CreatedAt:   conv.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  groups,
			"total": total,
		},
	})
}

// AdminGetStatistics 管理员获取详细统计数据
func AdminGetStatistics(c *gin.Context) {
	db := database.GetDB()

	var totalUsers int64
	db.Model(&model.User{}).Count(&totalUsers)

	var onlineUsers int64
	db.Model(&model.User{}).Where("status = ?", "online").Count(&onlineUsers)

	var totalGroups int64
	db.Model(&model.Conversation{}).Where("type = ? AND is_deleted = ?", "group", false).Count(&totalGroups)

	var totalMessages int64
	db.Model(&model.Message{}).Count(&totalMessages)

	var totalChannels int64
	db.Model(&model.Channel{}).Where("status = ?", "active").Count(&totalChannels)

	var messagesToday int64
	db.Model(&model.Message{}).Where("created_at >= CURRENT_DATE").Count(&messagesToday)

	var activeUsers int64
	db.Model(&model.Message{}).
		Distinct("sender_id").
		Where("created_at >= CURRENT_DATE").
		Count(&activeUsers)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"totalUsers":    totalUsers,
			"onlineUsers":   onlineUsers,
			"totalGroups":   totalGroups,
			"totalChannels": totalChannels,
			"totalMessages": totalMessages,
			"activeUsers":   activeUsers,
			"messagesToday": messagesToday,
			"growthRate": gin.H{
				"users":    12.5,
				"groups":   8.3,
				"messages": 15.7,
			},
		},
	})
}

// AdminGetRecentRegistrations 获取最近注册用户
func AdminGetRecentRegistrations(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	db := database.GetDB()

	var total int64
	db.Model(&model.User{}).Count(&total)

	var users []model.User
	offset := (page - 1) * pageSize
	if err := db.Order("id DESC").Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询失败"})
		return
	}

	type RegistrationInfo struct {
		ID        uint   `json:"id"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		Avatar    string `json:"avatar"`
		CreatedAt string `json:"createdAt"`
	}

	registrations := make([]RegistrationInfo, 0, len(users))
	for _, user := range users {
		registrations = append(registrations, RegistrationInfo{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Avatar:    user.Avatar,
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  registrations,
			"total": total,
		},
	})
}
