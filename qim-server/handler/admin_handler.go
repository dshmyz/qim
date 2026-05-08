package handler

import (
	"net/http"
	"strconv"

	"qim-server/di"
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
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  roles,
			"total": total,
		},
	})
}

func CreateRole(c *gin.Context) {
	var req struct {
		UserID uint   `json:"user_id" binding:"required"`
		Role   string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	adminSvc := di.GlobalContainer.AdminService
	userSvc := di.GlobalContainer.UserService

	_, err := userSvc.GetUser(req.UserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	userRole, err := adminSvc.CreateRole(req.UserID, req.Role)
	if err != nil {
		if err == gorm.ErrDuplicatedKey {
			c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "用户已有此角色"})
			return
		}
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

	adminSvc := di.GlobalContainer.AdminService
	userRole, err := adminSvc.GetRole(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "角色不存在"})
		return
	}

	if req.Role != "" {
		userRole.Role = req.Role
		adminSvc.UpdateRole(userRole)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": userRole,
	})
}

func DeleteRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的角色ID"})
		return
	}

	adminSvc := di.GlobalContainer.AdminService
	if _, err := adminSvc.GetRole(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "角色不存在"})
		return
	}

	adminSvc.DeleteRole(uint(id))

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询失败"})
		return
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

	adminSvc := di.GlobalContainer.AdminService
	if err := adminSvc.DeleteGroup(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "群组不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  users,
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

	adminSvc := di.GlobalContainer.AdminService
	channels, total, err := adminSvc.GetChannels(page, pageSize, keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  channels,
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

	adminSvc := di.GlobalContainer.AdminService
	if _, err := adminSvc.GetChannel(uint(id)); err != nil {
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
		if err := adminSvc.UpdateChannel(uint(id), updates); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
			return
		}
	}

	channel, _ := adminSvc.GetChannel(uint(id))
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

	adminSvc := di.GlobalContainer.AdminService
	if err := adminSvc.DeleteChannel(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "频道不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询失败"})
		return
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
	adminSvc := di.GlobalContainer.AdminService
	stats, err := adminSvc.GetStatistics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
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
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  registrations,
			"total": total,
		},
	})
}
