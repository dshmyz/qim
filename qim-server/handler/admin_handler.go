package handler

import (
	"net/http"
	"strconv"

	"qim-server/database"
	"qim-server/model"

	"github.com/gin-gonic/gin"
)

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

		status := "active"
		if conv.IsDeleted {
			status = "inactive"
		}

		groups = append(groups, GroupInfo{
			ID:          conv.ID,
			Name:        conv.Name,
			Avatar:      conv.Avatar,
			Description: conv.Announcement,
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
