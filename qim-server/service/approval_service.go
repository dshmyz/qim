package service

import (
	"encoding/json"
	"net/http"
	"qim-server/database"
	"qim-server/model"
	"qim-server/ws"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ApprovalService struct {
	db *gorm.DB
}

func NewApprovalService() *ApprovalService {
	return &ApprovalService{
		db: database.GetDB(),
	}
}

type ApprovalAction string

const (
	ApprovalActionApproved ApprovalAction = "approved"
	ApprovalActionRejected ApprovalAction = "rejected"
	ApprovalActionEnabled  ApprovalAction = "enabled"
)

type ApprovalNotification struct {
	EntityName   string
	EntityType   string
	UserID       uint
	Action       ApprovalAction
	Reason       string
	ExtraContext map[string]any
}

func (s *ApprovalService) ListApprovals(entityType string, status string) ([]model.ApprovalListItem, int64, error) {
	var items []model.ApprovalListItem

	switch entityType {
	case model.ApprovalTypeAvatar:
		return s.listAvatarApprovals(status)
	case model.ApprovalTypeBot:
		return s.listBotApprovals(status)
	case model.ApprovalTypeChannel:
		return s.listChannelApprovals(status)
	default:
		return items, 0, nil
	}
}

func (s *ApprovalService) listAvatarApprovals(status string) ([]model.ApprovalListItem, int64, error) {
	var configs []model.AvatarConfig
	query := s.db.Model(&model.AvatarConfig{}).Preload("User")

	if status != "all" {
		query = query.Where("approval_status = ?", status)
	}

	if err := query.Order("applied_at DESC").Find(&configs).Error; err != nil {
		return nil, 0, err
	}

	items := make([]model.ApprovalListItem, 0, len(configs))
	for _, config := range configs {
		item := model.ApprovalListItem{
			ID:             config.ID,
			Type:           model.ApprovalTypeAvatar,
			CreatorID:      config.UserID,
			Name:           config.Name,
			Description:    "用户分身功能",
			ApprovalStatus: config.ApprovalStatus,
			AppliedAt:      config.AppliedAt,
			ApprovedAt:     config.ApprovedAt,
			RejectReason:   config.RejectReason,
			CreatedAt:      config.CreatedAt,
		}
		if config.User.ID != 0 {
			item.CreatorName = config.User.Nickname
			item.CreatorAvatar = config.User.Avatar
		} else {
			var user model.User
			if err := s.db.Where("id = ?", config.UserID).First(&user).Error; err == nil {
				item.CreatorName = user.Nickname
				item.CreatorAvatar = user.Avatar
			}
		}
		items = append(items, item)
	}

	return items, int64(len(items)), nil
}

func (s *ApprovalService) listBotApprovals(status string) ([]model.ApprovalListItem, int64, error) {
	var bots []model.Bot
	query := s.db.Model(&model.Bot{}).Where("creator_id != 0")

	if status != "all" {
		query = query.Where("approval_status = ?", status)
	}

	if err := query.Order("created_at DESC").Find(&bots).Error; err != nil {
		return nil, 0, err
	}

	items := make([]model.ApprovalListItem, 0, len(bots))
	for _, bot := range bots {
		item := model.ApprovalListItem{
			ID:             bot.ID,
			Type:           model.ApprovalTypeBot,
			CreatorID:      bot.CreatorID,
			Name:           bot.Name,
			Description:    bot.Description,
			ApprovalStatus: bot.ApprovalStatus,
			RejectReason:   bot.RejectReason,
			CreatedAt:      bot.CreatedAt,
		}
		var creator model.User
		if err := s.db.Where("id = ?", bot.CreatorID).First(&creator).Error; err == nil {
			item.CreatorName = creator.Nickname
			item.CreatorAvatar = creator.Avatar
		}
		var count int64
		s.db.Model(&model.Bot{}).Where("creator_id = ?", bot.CreatorID).Count(&count)
		item.Extra = map[string]any{
			"bot_type":          bot.Type,
			"creator_bot_count": count,
		}
		items = append(items, item)
	}

	return items, int64(len(items)), nil
}

func (s *ApprovalService) listChannelApprovals(status string) ([]model.ApprovalListItem, int64, error) {
	return []model.ApprovalListItem{}, 0, nil
}

func (s *ApprovalService) Approve(entityType string, id uint, adminID uint) error {
	now := time.Now()

	switch entityType {
	case model.ApprovalTypeAvatar:
		return s.approveAvatar(id, adminID, &now)
	case model.ApprovalTypeBot:
		return s.approveBot(id, adminID, &now)
	case model.ApprovalTypeChannel:
		return s.approveChannel(id, adminID, &now)
	default:
		return gorm.ErrRecordNotFound
	}
}

func (s *ApprovalService) approveAvatar(id uint, adminID uint, now *time.Time) error {
	var config model.AvatarConfig
	if err := s.db.Where("id = ? AND approval_status = ?", id, model.ApprovalStatusPending).First(&config).Error; err != nil {
		return err
	}

	if err := s.db.Model(&config).Updates(map[string]interface{}{
		"approval_status": model.ApprovalStatusApproved,
		"enabled":         true,
		"approved_at":     now,
		"approved_by":     adminID,
	}).Error; err != nil {
		return err
	}

	s.SendNotification(ApprovalNotification{
		EntityName: "分身功能",
		EntityType: model.ApprovalTypeAvatar,
		UserID:     config.UserID,
		Action:     ApprovalActionApproved,
	})

	return nil
}

func (s *ApprovalService) approveBot(id uint, adminID uint, now *time.Time) error {
	var bot model.Bot
	if err := s.db.Where("id = ? AND approval_status = ?", id, model.ApprovalStatusPending).First(&bot).Error; err != nil {
		return err
	}

	if err := s.db.Model(&bot).Updates(map[string]interface{}{
		"approval_status": model.ApprovalStatusApproved,
		"is_active":       true,
		"approved_at":     now,
		"approved_by":     adminID,
	}).Error; err != nil {
		return err
	}

	s.SendNotification(ApprovalNotification{
		EntityName: bot.Name,
		EntityType: model.ApprovalTypeBot,
		UserID:     bot.CreatorID,
		Action:     ApprovalActionApproved,
		ExtraContext: map[string]any{
			"bot_name": bot.Name,
		},
	})

	return nil
}

func (s *ApprovalService) approveChannel(id uint, adminID uint, now *time.Time) error {
	return nil
}

func (s *ApprovalService) Reject(entityType string, id uint, adminID uint, reason string) error {
	now := time.Now()

	switch entityType {
	case model.ApprovalTypeAvatar:
		return s.rejectAvatar(id, adminID, &now, reason)
	case model.ApprovalTypeBot:
		return s.rejectBot(id, adminID, &now, reason)
	case model.ApprovalTypeChannel:
		return s.rejectChannel(id, adminID, &now, reason)
	default:
		return gorm.ErrRecordNotFound
	}
}

func (s *ApprovalService) rejectAvatar(id uint, adminID uint, now *time.Time, reason string) error {
	var config model.AvatarConfig
	if err := s.db.Where("id = ? AND approval_status = ?", id, model.ApprovalStatusPending).First(&config).Error; err != nil {
		return err
	}

	if err := s.db.Model(&config).Updates(map[string]interface{}{
		"approval_status": model.ApprovalStatusRejected,
		"enabled":         false,
		"reject_reason":   reason,
		"approved_at":     now,
		"approved_by":     adminID,
	}).Error; err != nil {
		return err
	}

	s.SendNotification(ApprovalNotification{
		EntityName: "分身功能",
		EntityType: model.ApprovalTypeAvatar,
		UserID:     config.UserID,
		Action:     ApprovalActionRejected,
		Reason:     reason,
	})

	return nil
}

func (s *ApprovalService) rejectBot(id uint, adminID uint, now *time.Time, reason string) error {
	var bot model.Bot
	if err := s.db.Where("id = ? AND approval_status = ?", id, model.ApprovalStatusPending).First(&bot).Error; err != nil {
		return err
	}

	if err := s.db.Model(&bot).Updates(map[string]interface{}{
		"approval_status": model.ApprovalStatusRejected,
		"is_active":       false,
		"reject_reason":   reason,
		"approved_at":     now,
		"approved_by":     adminID,
	}).Error; err != nil {
		return err
	}

	s.SendNotification(ApprovalNotification{
		EntityName: bot.Name,
		EntityType: model.ApprovalTypeBot,
		UserID:     bot.CreatorID,
		Action:     ApprovalActionRejected,
		Reason:     reason,
		ExtraContext: map[string]any{
			"bot_name": bot.Name,
		},
	})

	return nil
}

func (s *ApprovalService) rejectChannel(id uint, adminID uint, now *time.Time, reason string) error {
	return nil
}

func (s *ApprovalService) EnableAvatar(userID uint, adminID uint) error {
	now := time.Now()

	var config model.AvatarConfig
	err := s.db.Where("user_id = ?", userID).First(&config).Error

	if err != nil {
		config = model.AvatarConfig{
			UserID:          userID,
			Name:            "我的分身",
			Enabled:         true,
			ApprovalStatus:  model.ApprovalStatusApproved,
			ApprovedAt:      &now,
			ApprovedBy:      &adminID,
			UseSystemConfig: true,
		}
		if err := s.db.Create(&config).Error; err != nil {
			return err
		}
	} else {
		if err := s.db.Model(&config).Updates(map[string]interface{}{
			"approval_status": model.ApprovalStatusApproved,
			"enabled":         true,
			"approved_at":     &now,
			"approved_by":     adminID,
		}).Error; err != nil {
			return err
		}
	}

	s.SendNotification(ApprovalNotification{
		EntityName: "分身功能",
		EntityType: model.ApprovalTypeAvatar,
		UserID:     userID,
		Action:     ApprovalActionEnabled,
	})

	return nil
}

func (s *ApprovalService) SendNotification(n ApprovalNotification) {
	var title, content string
	priority := "normal"

	switch n.EntityType {
	case model.ApprovalTypeAvatar:
		switch n.Action {
		case ApprovalActionApproved:
			title = "分身功能已开通"
			content = "您的分身功能申请已通过审批，现在可以启用分身功能了。"
			priority = "important"
		case ApprovalActionEnabled:
			title = "分身功能已开通"
			content = "管理员已为您开通分身功能，现在可以启用分身功能了。"
			priority = "important"
		case ApprovalActionRejected:
			title = "分身功能申请被拒绝"
			if n.Reason != "" {
				content = "您的分身功能申请被拒绝，原因：" + n.Reason
			} else {
				content = "您的分身功能申请被拒绝。"
			}
		}

	case model.ApprovalTypeBot:
		botName := ""
		if n.ExtraContext != nil {
			if name, ok := n.ExtraContext["bot_name"].(string); ok {
				botName = name
			}
		}
		switch n.Action {
		case ApprovalActionApproved:
			title = "Bot 审批通过"
			if botName != "" {
				content = "您创建的机器人「" + botName + "」已通过审批，现在可以使用了。"
			} else {
				content = "您创建的机器人已通过审批，现在可以使用了。"
			}
			priority = "important"
		case ApprovalActionRejected:
			title = "Bot 审批被拒绝"
			if botName != "" {
				if n.Reason != "" {
					content = "您创建的机器人「" + botName + "」被拒绝，原因：" + n.Reason
				} else {
					content = "您创建的机器人「" + botName + "」被拒绝。"
				}
			} else {
				if n.Reason != "" {
					content = "您创建的机器人被拒绝，原因：" + n.Reason
				} else {
					content = "您创建的机器人被拒绝。"
				}
			}
		}
	}

	notification := model.Notification{
		UserID:   n.UserID,
		Type:     string(n.EntityType) + "_approval",
		Title:    title,
		Content:  content,
		Priority: priority,
	}

	if err := s.db.Create(&notification).Error; err != nil {
		return
	}

	if ws.GlobalHub != nil {
		notificationMsg := ws.WSMessage{
			Type: "notification",
			Data: notification,
		}
		jsonMsg, _ := json.Marshal(notificationMsg)
		ws.GlobalHub.SendToUser(n.UserID, jsonMsg)
	}
}

type ApprovalHandler struct {
	service *ApprovalService
}

func NewApprovalHandler() *ApprovalHandler {
	return &ApprovalHandler{
		service: NewApprovalService(),
	}
}

func (h *ApprovalHandler) RegisterRoutes(r *gin.RouterGroup) {
	approvals := r.Group("/approvals")
	{
		approvals.GET("", h.List)
		approvals.POST("/:type/:id/approve", h.Approve)
		approvals.POST("/:type/:id/reject", h.Reject)
		approvals.POST("/avatar/enable", h.EnableAvatar)
	}
}

func (h *ApprovalHandler) List(c *gin.Context) {
	entityType := c.DefaultQuery("type", "all")
	status := c.DefaultQuery("status", model.ApprovalStatusPending)

	var allItems []model.ApprovalListItem
	var total int64

	if entityType == "all" {
		for _, t := range []string{model.ApprovalTypeAvatar, model.ApprovalTypeBot} {
			items, count, err := h.service.ListApprovals(t, status)
			if err == nil {
				allItems = append(allItems, items...)
				total += count
			}
		}
	} else {
		var err error
		allItems, total, err = h.service.ListApprovals(entityType, status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取审批列表失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":     allItems,
			"total":    total,
			"page":     1,
			"pageSize": len(allItems),
		},
	})
}

func (h *ApprovalHandler) Approve(c *gin.Context) {
	entityType := c.Param("type")
	idStr := c.Param("id")

	id, err := json.Number(idStr).Int64()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的 ID"})
		return
	}

	adminIDAny, _ := c.Get("user_id")
	adminID := adminIDAny.(uint)

	if err := h.service.Approve(entityType, uint(id), adminID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "记录不存在或无需审批"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "审批通过"})
}

func (h *ApprovalHandler) Reject(c *gin.Context) {
	entityType := c.Param("type")
	idStr := c.Param("id")

	id, err := json.Number(idStr).Int64()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的 ID"})
		return
	}

	adminIDAny, _ := c.Get("user_id")
	adminID := adminIDAny.(uint)

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请填写拒绝原因"})
		return
	}

	if err := h.service.Reject(entityType, uint(id), adminID, req.Reason); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "记录不存在或无需审批"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已拒绝"})
}

func (h *ApprovalHandler) EnableAvatar(c *gin.Context) {
	var req struct {
		UserID uint `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	adminIDAny, _ := c.Get("user_id")
	adminID := adminIDAny.(uint)

	if err := h.service.EnableAvatar(req.UserID, adminID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "启用失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "分身已启用"})
}
