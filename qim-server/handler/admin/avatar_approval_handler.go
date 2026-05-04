package admin

import (
	"net/http"
	"strconv"
	"time"

	"qim-server/database"
	"qim-server/model"

	"github.com/gin-gonic/gin"
)

// AvatarApprovalItem 审批列表项
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

// ListPendingAvatars 获取待审批的分身列表
func ListPendingAvatars(c *gin.Context) {
	db := database.GetDB()

	status := c.DefaultQuery("status", model.ApprovalStatusPending) // pending, approved, rejected, all

	var configs []model.AvatarConfig
	query := db.Model(&model.AvatarConfig{}).Preload("User")

	if status != "all" {
		query = query.Where("approval_status = ?", status)
	}

	if err := query.Order("applied_at DESC").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取审批列表失败"})
		return
	}

	// 组装审批列表项
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
		// 填充用户信息
		if config.User.ID != 0 {
			item.UserName = config.User.Nickname
			item.UserAvatar = config.User.Avatar
		} else {
			// 如果 Preload 没有加载到，手动查询
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

// ApproveAvatar 通过分身申请
func ApproveAvatar(c *gin.Context) {
	avatarID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的分身 ID"})
		return
	}
	db := database.GetDB()

	// 获取当前管理员 ID
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

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "审批通过", "data": config})
}

// RejectAvatarRequest 拒绝分身请求
type RejectAvatarRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// RejectAvatar 拒绝分身申请
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

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已拒绝", "data": config})
}
