package admin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"qim-server/database"
	"qim-server/model"
	"qim-server/ws"

	"github.com/gin-gonic/gin"
)

type AvatarApprovalItem struct {
	ID             uint       `json:"id"`
	UserID         uint       `json:"user_id"`
	UserName       string     `json:"user_name"`
	UserAvatar     string     `json:"user_avatar"`
	Name           string     `json:"name"`
	ApprovalStatus string     `json:"approval_status"`
	AppliedAt      *time.Time `json:"applied_at"`
	RejectReason   string     `json:"reject_reason"`
	CreatedAt      time.Time  `json:"created_at"`
}

func ListPendingAvatars(c *gin.Context) {
	db := database.GetDB()

	status := c.DefaultQuery("status", model.ApprovalStatusPending)

	var configs []model.AvatarConfig
	query := db.Model(&model.AvatarConfig{}).Preload("User")

	if status != "all" {
		query = query.Where("approval_status = ?", status)
	}

	if err := query.Order("applied_at DESC").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取审批列表失败"})
		return
	}

	items := make([]AvatarApprovalItem, 0, len(configs))
	for _, config := range configs {
		item := AvatarApprovalItem{
			ID:             config.ID,
			UserID:         config.UserID,
			Name:           config.Name,
			ApprovalStatus: config.ApprovalStatus,
			AppliedAt:      config.AppliedAt,
			RejectReason:   config.RejectReason,
			CreatedAt:      config.CreatedAt,
		}
		if config.User.ID != 0 {
			item.UserName = config.User.Nickname
			item.UserAvatar = config.User.Avatar
		} else {
			var user model.User
			if err := db.Where("id = ?", config.UserID).First(&user).Error; err == nil {
				item.UserName = user.Nickname
				item.UserAvatar = user.Avatar
			}
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":     items,
			"total":    len(items),
			"page":     1,
			"pageSize": len(items),
		},
	})
}

func ApproveAvatar(c *gin.Context) {
	avatarID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的分身 ID"})
		return
	}
	db := database.GetDB()

	adminIDAny, _ := c.Get("user_id")
	adminID := adminIDAny.(uint)

	var config model.AvatarConfig
	if err := db.Where("id = ? AND approval_status = ?", uint(avatarID), model.ApprovalStatusPending).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "分身不存在或无需审批"})
		return
	}

	now := time.Now()
	if err := db.Model(&config).Updates(map[string]interface{}{
		"approval_status": model.ApprovalStatusApproved,
		"enabled":         true,
		"approved_at":     &now,
		"approved_by":     adminID,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "审批失败"})
		return
	}

	sendAvatarApprovalNotification(config.UserID, "approved", "")

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "审批通过", "data": config})
}

type RejectAvatarRequest struct {
	Reason string `json:"reason" binding:"required"`
}

func RejectAvatar(c *gin.Context) {
	avatarID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的分身 ID"})
		return
	}
	db := database.GetDB()

	var config model.AvatarConfig
	if err := db.Where("id = ? AND approval_status = ?", uint(avatarID), model.ApprovalStatusPending).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "分身不存在或无需审批"})
		return
	}

	var req RejectAvatarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误，拒绝原因不能为空"})
		return
	}

	if err := db.Model(&config).Updates(map[string]interface{}{
		"approval_status": model.ApprovalStatusRejected,
		"enabled":         false,
		"reject_reason":   req.Reason,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "操作失败"})
		return
	}

	sendAvatarApprovalNotification(config.UserID, "rejected", req.Reason)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已拒绝", "data": config})
}

type EnableAvatarRequest struct {
	UserID uint `json:"user_id" binding:"required"`
}

func EnableAvatarByAdmin(c *gin.Context) {
	var req EnableAvatarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	adminIDAny, _ := c.Get("user_id")
	adminID := adminIDAny.(uint)

	var config model.AvatarConfig
	err := db.Where("user_id = ?", req.UserID).First(&config).Error

	now := time.Now()

	if err != nil {
		config = model.AvatarConfig{
			UserID:          req.UserID,
			Name:            "我的分身",
			Enabled:         true,
			ApprovalStatus:  model.ApprovalStatusApproved,
			ApprovedAt:      &now,
			ApprovedBy:      &adminID,
			UseSystemConfig: true,
		}
		if err := db.Create(&config).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建分身配置失败"})
			return
		}
	} else {
		if err := db.Model(&config).Updates(map[string]interface{}{
			"approval_status": model.ApprovalStatusApproved,
			"enabled":         true,
			"approved_at":     &now,
			"approved_by":     adminID,
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "启用分身失败"})
			return
		}
	}

	sendAvatarApprovalNotification(req.UserID, "enabled", "")

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "分身已启用", "data": config})
}

func sendAvatarApprovalNotification(userID uint, action string, reason string) {
	db := database.GetDB()

	var title, content string
	var priority string = "normal"

	switch action {
	case "approved":
		title = "分身功能已开通"
		content = "您的分身功能申请已通过审批，现在可以启用分身功能了。"
		priority = "important"
	case "enabled":
		title = "分身功能已开通"
		content = "管理员已为您开通分身功能，现在可以启用分身功能了。"
		priority = "important"
	case "rejected":
		title = "分身功能申请被拒绝"
		if reason != "" {
			content = "您的分身功能申请被拒绝，原因：" + reason
		} else {
			content = "您的分身功能申请被拒绝。"
		}
	}

	notification := model.Notification{
		UserID:   userID,
		Type:     "avatar_approval",
		Title:    title,
		Content:  content,
		Priority: priority,
	}

	if err := db.Create(&notification).Error; err != nil {
		return
	}

	if ws.GlobalHub != nil {
		notificationMsg := ws.WSMessage{
			Type: "notification",
			Data: notification,
		}
		jsonMsg, _ := json.Marshal(notificationMsg)
		ws.GlobalHub.SendToUser(userID, jsonMsg)
	}
}
