package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ApprovalService struct {
	db *gorm.DB
}

func NewApprovalService(db *gorm.DB) *ApprovalService {
	return &ApprovalService{
		db: db,
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

func (s *ApprovalService) ListApprovals(entityType string, status string, page int, pageSize int) ([]model.ApprovalListItem, int64, error) {
	var items []model.ApprovalListItem
	var total int64
	var err error

	switch entityType {
	case model.ApprovalTypeAvatar:
		items, total, err = s.listAvatarApprovals(status)
	case model.ApprovalTypeBot:
		items, total, err = s.listBotApprovals(status)
	case model.ApprovalTypeChannel:
		items, total, err = s.listChannelApprovals(status)
	case model.ApprovalTypeGroupAI:
		items, total, err = s.listGroupAIApprovals(status)
	case "all", "":
		return s.listAllApprovals(status, page, pageSize)
	default:
		return items, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}

	return paginateApprovals(items, total, page, pageSize), total, nil
}

// listAllApprovals 使用快照字段单表分页查询，无需回查源表
func (s *ApprovalService) listAllApprovals(status string, page, pageSize int) ([]model.ApprovalListItem, int64, error) {
	query := s.db.Model(&model.Approval{})
	if status != "all" && status != "" {
		query = query.Where("status = ?", status)
	} else {
		query = query.Where("status != ?", model.ApprovalStatusNone)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var approvals []model.Approval
	offset := (page - 1) * pageSize
	if err := query.Order("applied_at DESC").Offset(offset).Limit(pageSize).Find(&approvals).Error; err != nil {
		return nil, 0, err
	}

	items := make([]model.ApprovalListItem, 0, len(approvals))
	for _, a := range approvals {
		item := model.ApprovalListItem{
			ID:             a.ID,
			Type:           a.TargetType,
			CreatorID:      a.AppliedBy,
			CreatorName:    a.CreatorName,
			CreatorAvatar:  a.CreatorAvatar,
			Name:           a.TargetName,
			Description:    a.TargetDescription,
			ApprovalStatus: a.Status,
			AppliedAt:      &a.AppliedAt,
			ApprovedAt:     a.ApprovedAt,
			RejectReason:   a.RejectReason,
			CreatedAt:      a.CreatedAt,
		}
		if a.ExtraJSON != "" && a.ExtraJSON != "{}" {
			var extra map[string]any
			if err := json.Unmarshal([]byte(a.ExtraJSON), &extra); err == nil {
				item.Extra = extra
			}
		}
		items = append(items, item)
	}
	return items, total, nil
}

// paginateApprovals 对内存中的审批列表做分页切片；page/pageSize 非法时返回全部。
func paginateApprovals(items []model.ApprovalListItem, total int64, page int, pageSize int) []model.ApprovalListItem {
	if page <= 0 || pageSize <= 0 {
		return items
	}
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []model.ApprovalListItem{}
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

func (s *ApprovalService) listAvatarApprovals(status string) ([]model.ApprovalListItem, int64, error) {
	var approvals []model.Approval
	query := s.db.Model(&model.Approval{}).Where("target_type = ?", model.ApprovalTypeAvatar)

	if status != "all" {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("applied_at DESC").Find(&approvals).Error; err != nil {
		return nil, 0, err
	}

	items := make([]model.ApprovalListItem, 0, len(approvals))
	for _, approval := range approvals {
		var config model.AvatarConfig
		if err := s.db.Where("id = ?", approval.TargetID).First(&config).Error; err != nil {
			continue
		}

		item := model.ApprovalListItem{
			ID:             approval.ID,
			Type:           model.ApprovalTypeAvatar,
			CreatorID:      approval.AppliedBy,
			Name:           config.Name,
			Description:    "用户分身功能",
			ApprovalStatus: approval.Status,
			AppliedAt:      &approval.AppliedAt,
			ApprovedAt:     approval.ApprovedAt,
			RejectReason:   approval.RejectReason,
			CreatedAt:      approval.CreatedAt,
		}

		var user model.User
		if err := s.db.Where("id = ?", approval.AppliedBy).First(&user).Error; err == nil {
			item.CreatorName = user.Nickname
			item.CreatorAvatar = user.Avatar
		}
		items = append(items, item)
	}

	return items, int64(len(items)), nil
}

func (s *ApprovalService) listBotApprovals(status string) ([]model.ApprovalListItem, int64, error) {
	var approvals []model.Approval
	query := s.db.Model(&model.Approval{}).Where("target_type = ?", model.ApprovalTypeBot)

	if status != "all" {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("applied_at DESC").Find(&approvals).Error; err != nil {
		return nil, 0, err
	}

	items := make([]model.ApprovalListItem, 0, len(approvals))
	for _, approval := range approvals {
		var bot model.Bot
		if err := s.db.Where("id = ?", approval.TargetID).First(&bot).Error; err != nil {
			continue
		}

		item := model.ApprovalListItem{
			ID:             approval.ID,
			Type:           model.ApprovalTypeBot,
			CreatorID:      approval.AppliedBy,
			Name:           bot.Name,
			Description:    bot.Description,
			ApprovalStatus: approval.Status,
			AppliedAt:      &approval.AppliedAt,
			ApprovedAt:     approval.ApprovedAt,
			RejectReason:   approval.RejectReason,
			CreatedAt:      approval.CreatedAt,
		}

		var creator model.User
		if err := s.db.Where("id = ?", approval.AppliedBy).First(&creator).Error; err == nil {
			item.CreatorName = creator.Nickname
			item.CreatorAvatar = creator.Avatar
		}
		item.Extra = map[string]any{
			"bot_type": bot.Type,
		}
		items = append(items, item)
	}

	return items, int64(len(items)), nil
}

func (s *ApprovalService) listChannelApprovals(status string) ([]model.ApprovalListItem, int64, error) {
	var approvals []model.Approval
	query := s.db.Model(&model.Approval{}).Where("target_type = ?", model.ApprovalTypeChannel)

	if status != "all" && status != "" {
		query = query.Where("status = ?", status)
	} else {
		query = query.Where("status != ?", model.ApprovalStatusNone)
	}

	if err := query.Order("applied_at DESC").Find(&approvals).Error; err != nil {
		return nil, 0, err
	}

	items := make([]model.ApprovalListItem, 0, len(approvals))
	for _, approval := range approvals {
		var channel model.Channel
		if err := s.db.Where("id = ?", approval.TargetID).First(&channel).Error; err != nil {
			continue
		}

		item := model.ApprovalListItem{
			ID:             approval.ID,
			Type:           model.ApprovalTypeChannel,
			CreatorID:      approval.AppliedBy,
			Name:           channel.Name,
			Description:    channel.Description,
			ApprovalStatus: approval.Status,
			AppliedAt:      &approval.AppliedAt,
			ApprovedAt:     approval.ApprovedAt,
			RejectReason:   approval.RejectReason,
			CreatedAt:      approval.CreatedAt,
		}

		var creator model.User
		if err := s.db.Where("id = ?", approval.AppliedBy).First(&creator).Error; err == nil {
			item.CreatorName = creator.Nickname
			item.CreatorAvatar = creator.Avatar
		}

		item.Extra = map[string]any{
			"channel_id":         channel.ID,
			"publish_permission": channel.PublishPermission,
		}
		items = append(items, item)
	}

	return items, int64(len(items)), nil
}

func (s *ApprovalService) listGroupAIApprovals(status string) ([]model.ApprovalListItem, int64, error) {
	var approvals []model.Approval
	query := s.db.Model(&model.Approval{}).Where("target_type = ?", model.ApprovalTypeGroupAI)

	if status != "all" {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("applied_at DESC").Find(&approvals).Error; err != nil {
		return nil, 0, err
	}

	items := make([]model.ApprovalListItem, 0, len(approvals))
	for _, approval := range approvals {
		var group model.Group
		if err := s.db.Where("id = ?", approval.TargetID).First(&group).Error; err != nil {
			continue
		}

		aiConfig := group.GetAIConfig()
		if status != model.ApprovalStatusPending && !aiConfig.Enabled {
			continue
		}

		item := model.ApprovalListItem{
			ID:             approval.ID,
			Type:           model.ApprovalTypeGroupAI,
			CreatorID:      approval.AppliedBy,
			Name:           group.Name,
			Description:    "群聊AI助手",
			ApprovalStatus: approval.Status,
			AppliedAt:      &approval.AppliedAt,
			ApprovedAt:     approval.ApprovedAt,
			RejectReason:   approval.RejectReason,
			CreatedAt:      approval.CreatedAt,
		}

		var creator model.User
		if err := s.db.Where("id = ?", approval.AppliedBy).First(&creator).Error; err == nil {
			item.CreatorName = creator.Nickname
			item.CreatorAvatar = creator.Avatar
		}

		item.Extra = map[string]any{
			"group_id":        group.ID,
			"conversation_id": group.ConversationID,
			"assistant_name":  aiConfig.AssistantName,
		}
		items = append(items, item)
	}

	return items, int64(len(items)), nil
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
	case model.ApprovalTypeGroupAI:
		return s.approveGroupAI(id, adminID, &now)
	default:
		return gorm.ErrRecordNotFound
	}
}

func (s *ApprovalService) approveAvatar(id uint, adminID uint, now *time.Time) error {
	var approval model.Approval
	if err := s.db.Where("id = ? AND status = ? AND target_type = ?", id, model.ApprovalStatusPending, model.ApprovalTypeAvatar).First(&approval).Error; err != nil {
		return err
	}

	tx := s.db.Begin()

	if err := tx.Model(&approval).Updates(map[string]interface{}{
		"status":      model.ApprovalStatusApproved,
		"approved_at": now,
		"approved_by": adminID,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	var config model.AvatarConfig
	if err := tx.Where("id = ?", approval.TargetID).First(&config).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&config).Update("enabled", true).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	s.SendNotification(ApprovalNotification{
		EntityName: "分身功能",
		EntityType: model.ApprovalTypeAvatar,
		UserID:     approval.AppliedBy,
		Action:     ApprovalActionApproved,
	})

	return nil
}

func (s *ApprovalService) approveBot(id uint, adminID uint, now *time.Time) error {
	var approval model.Approval
	if err := s.db.Where("id = ? AND status = ? AND target_type = ?", id, model.ApprovalStatusPending, model.ApprovalTypeBot).First(&approval).Error; err != nil {
		return err
	}

	tx := s.db.Begin()

	if err := tx.Model(&approval).Updates(map[string]interface{}{
		"status":      model.ApprovalStatusApproved,
		"approved_at": now,
		"approved_by": adminID,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	var bot model.Bot
	if err := tx.Where("id = ?", approval.TargetID).First(&bot).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&bot).Update("is_active", true).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	s.SendNotification(ApprovalNotification{
		EntityName: bot.Name,
		EntityType: model.ApprovalTypeBot,
		UserID:     approval.AppliedBy,
		Action:     ApprovalActionApproved,
		ExtraContext: map[string]any{
			"bot_name": bot.Name,
		},
	})

	return nil
}

func (s *ApprovalService) approveChannel(id uint, adminID uint, now *time.Time) error {
	var approval model.Approval
	if err := s.db.Where("id = ? AND status = ? AND target_type = ?", id, model.ApprovalStatusPending, model.ApprovalTypeChannel).First(&approval).Error; err != nil {
		return err
	}

	if err := s.db.Model(&approval).Updates(map[string]interface{}{
		"status":      model.ApprovalStatusApproved,
		"approved_at": now,
		"approved_by": adminID,
	}).Error; err != nil {
		return err
	}

	var channel model.Channel
	if err := s.db.Where("id = ?", approval.TargetID).First(&channel).Error; err != nil {
		return err
	}

	s.SendNotification(ApprovalNotification{
		EntityName: channel.Name,
		EntityType: model.ApprovalTypeChannel,
		UserID:     approval.AppliedBy,
		Action:     ApprovalActionApproved,
		ExtraContext: map[string]any{
			"channel_id": channel.ID,
		},
	})

	return nil
}

func (s *ApprovalService) approveGroupAI(id uint, adminID uint, now *time.Time) error {
	var approval model.Approval
	if err := s.db.Where("id = ? AND status = ? AND target_type = ?", id, model.ApprovalStatusPending, model.ApprovalTypeGroupAI).First(&approval).Error; err != nil {
		return err
	}

	tx := s.db.Begin()

	if err := tx.Model(&approval).Updates(map[string]interface{}{
		"status":      model.ApprovalStatusApproved,
		"approved_at": now,
		"approved_by": adminID,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	var group model.Group
	if err := tx.Where("id = ?", approval.TargetID).First(&group).Error; err != nil {
		tx.Rollback()
		return err
	}

	aiConfig := group.GetAIConfig()
	aiConfig.Enabled = true
	group.SetAIConfig(aiConfig)

	if err := tx.Model(&group).Update("ai_config", group.AIConfigJSON).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	userSvc := NewUserService(s.db)
	assistantName := aiConfig.AssistantName
	if assistantName == "" {
		assistantName = "AI助手"
	}
	if aiUser, err := userSvc.EnsureGroupAIAssistant(group.ID, assistantName); err == nil {
		content := fmt.Sprintf("%s 已加入群聊，开始为大家服务", assistantName)
		NotifyMembersJoined(s.db, group.ConversationID, aiUser.ID, content, []model.User{*aiUser})
	} else {
		logger.WithModule("ApprovalService").Error("创建群助手账号失败", "groupID", group.ID, "error", err)
	}

	s.SendNotification(ApprovalNotification{
		EntityName: group.Name,
		EntityType: model.ApprovalTypeGroupAI,
		UserID:     approval.AppliedBy,
		Action:     ApprovalActionApproved,
		ExtraContext: map[string]any{
			"group_name": group.Name,
		},
	})

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
	case model.ApprovalTypeGroupAI:
		return s.rejectGroupAI(id, adminID, &now, reason)
	default:
		return gorm.ErrRecordNotFound
	}
}

func (s *ApprovalService) rejectAvatar(id uint, adminID uint, now *time.Time, reason string) error {
	var approval model.Approval
	if err := s.db.Where("id = ? AND status = ? AND target_type = ?", id, model.ApprovalStatusPending, model.ApprovalTypeAvatar).First(&approval).Error; err != nil {
		return err
	}

	tx := s.db.Begin()

	if err := tx.Model(&approval).Updates(map[string]interface{}{
		"status":        model.ApprovalStatusRejected,
		"reject_reason": reason,
		"approved_at":   now,
		"approved_by":   adminID,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	var config model.AvatarConfig
	if err := tx.Where("id = ?", approval.TargetID).First(&config).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&config).Update("enabled", false).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	s.SendNotification(ApprovalNotification{
		EntityName: "分身功能",
		EntityType: model.ApprovalTypeAvatar,
		UserID:     approval.AppliedBy,
		Action:     ApprovalActionRejected,
		Reason:     reason,
	})

	return nil
}

func (s *ApprovalService) rejectBot(id uint, adminID uint, now *time.Time, reason string) error {
	var approval model.Approval
	if err := s.db.Where("id = ? AND status = ? AND target_type = ?", id, model.ApprovalStatusPending, model.ApprovalTypeBot).First(&approval).Error; err != nil {
		return err
	}

	tx := s.db.Begin()

	if err := tx.Model(&approval).Updates(map[string]interface{}{
		"status":        model.ApprovalStatusRejected,
		"reject_reason": reason,
		"approved_at":   now,
		"approved_by":   adminID,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	var bot model.Bot
	if err := tx.Where("id = ?", approval.TargetID).First(&bot).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&bot).Update("is_active", false).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	s.SendNotification(ApprovalNotification{
		EntityName: bot.Name,
		EntityType: model.ApprovalTypeBot,
		UserID:     approval.AppliedBy,
		Action:     ApprovalActionRejected,
		Reason:     reason,
		ExtraContext: map[string]any{
			"bot_name": bot.Name,
		},
	})

	return nil
}

func (s *ApprovalService) rejectChannel(id uint, adminID uint, now *time.Time, reason string) error {
	var approval model.Approval
	if err := s.db.Where("id = ? AND status = ? AND target_type = ?", id, model.ApprovalStatusPending, model.ApprovalTypeChannel).First(&approval).Error; err != nil {
		return err
	}

	if err := s.db.Model(&approval).Updates(map[string]interface{}{
		"status":        model.ApprovalStatusRejected,
		"reject_reason": reason,
		"approved_at":   now,
		"approved_by":   adminID,
	}).Error; err != nil {
		return err
	}

	var channel model.Channel
	if err := s.db.Where("id = ?", approval.TargetID).First(&channel).Error; err != nil {
		return err
	}

	s.SendNotification(ApprovalNotification{
		EntityName: channel.Name,
		EntityType: model.ApprovalTypeChannel,
		UserID:     approval.AppliedBy,
		Action:     ApprovalActionRejected,
		ExtraContext: map[string]any{
			"channel_id":    channel.ID,
			"reject_reason": reason,
		},
	})

	return nil
}

func (s *ApprovalService) rejectGroupAI(id uint, adminID uint, now *time.Time, reason string) error {
	var approval model.Approval
	if err := s.db.Where("id = ? AND status = ? AND target_type = ?", id, model.ApprovalStatusPending, model.ApprovalTypeGroupAI).First(&approval).Error; err != nil {
		return err
	}

	tx := s.db.Begin()

	if err := tx.Model(&approval).Updates(map[string]interface{}{
		"status":        model.ApprovalStatusRejected,
		"reject_reason": reason,
		"approved_at":   now,
		"approved_by":   adminID,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	var group model.Group
	if err := tx.Where("id = ?", approval.TargetID).First(&group).Error; err != nil {
		tx.Rollback()
		return err
	}

	aiConfig := group.GetAIConfig()
	aiConfig.Enabled = false
	group.SetAIConfig(aiConfig)

	if err := tx.Model(&group).Update("ai_config", group.AIConfigJSON).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	s.SendNotification(ApprovalNotification{
		EntityName: group.Name,
		EntityType: model.ApprovalTypeGroupAI,
		UserID:     approval.AppliedBy,
		Action:     ApprovalActionRejected,
		Reason:     reason,
		ExtraContext: map[string]any{
			"group_name": group.Name,
		},
	})

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
			UseSystemConfig: true,
		}
		if err := s.db.Create(&config).Error; err != nil {
			return err
		}

		creatorName, creatorAvatar := s.fetchUserSnapshot(userID)
		approval := model.Approval{
			TargetType:        model.ApprovalTypeAvatar,
			TargetID:          config.ID,
			Status:            model.ApprovalStatusApproved,
			AppliedAt:         now,
			AppliedBy:         userID,
			ApprovedAt:        &now,
			ApprovedBy:        &adminID,
			TargetName:        config.Name,
			TargetDescription: "用户分身功能",
			CreatorName:       creatorName,
			CreatorAvatar:     creatorAvatar,
			ExtraJSON:         "{}",
		}
		if err := s.db.Create(&approval).Error; err != nil {
			return err
		}
	} else {
		if err := s.db.Model(&config).Update("enabled", true).Error; err != nil {
			return err
		}

		// 查找该分身最新的审批记录（可能有多条，如先拒绝再申请）
		var existing model.Approval
		findErr := s.db.Where("target_type = ? AND target_id = ?", model.ApprovalTypeAvatar, config.ID).
			Order("created_at DESC").
			First(&existing).Error

		if findErr == gorm.ErrRecordNotFound {
			// 审批记录不存在（如审批系统上线前创建的配置），补建一条已通过记录
			creatorName, creatorAvatar := s.fetchUserSnapshot(userID)
			approval := model.Approval{
				TargetType:        model.ApprovalTypeAvatar,
				TargetID:          config.ID,
				Status:            model.ApprovalStatusApproved,
				AppliedAt:         now,
				AppliedBy:         userID,
				ApprovedAt:        &now,
				ApprovedBy:        &adminID,
				TargetName:        config.Name,
				TargetDescription: "用户分身功能",
				CreatorName:       creatorName,
				CreatorAvatar:     creatorAvatar,
				ExtraJSON:         "{}",
			}
			if err := s.db.Create(&approval).Error; err != nil {
				return err
			}
		} else if findErr != nil {
			return findErr
		} else {
			// 已有审批记录，只更新最新一条
			if err := s.db.Model(&existing).Updates(map[string]interface{}{
				"status":      model.ApprovalStatusApproved,
				"approved_at": &now,
				"approved_by": adminID,
			}).Error; err != nil {
				return err
			}
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

func (s *ApprovalService) CreateApproval(targetType string, targetID uint, appliedBy uint) error {
	now := time.Now()

	name, desc, extraJSON := s.fetchTargetSnapshot(targetType, targetID)
	creatorName, creatorAvatar := s.fetchUserSnapshot(appliedBy)

	approval := model.Approval{
		TargetType:        targetType,
		TargetID:          targetID,
		Status:            model.ApprovalStatusPending,
		AppliedAt:         now,
		AppliedBy:         appliedBy,
		TargetName:        name,
		TargetDescription: desc,
		CreatorName:       creatorName,
		CreatorAvatar:     creatorAvatar,
		ExtraJSON:         extraJSON,
	}

	return s.db.Create(&approval).Error
}

// fetchTargetSnapshot 按 target_type 查源表，返回名称、描述和 extra JSON
func (s *ApprovalService) fetchTargetSnapshot(targetType string, targetID uint) (name, desc, extraJSON string) {
	switch targetType {
	case model.ApprovalTypeAvatar:
		var config model.AvatarConfig
		if err := s.db.Where("id = ?", targetID).First(&config).Error; err == nil {
			return config.Name, "用户分身功能", "{}"
		}
	case model.ApprovalTypeBot:
		var bot model.Bot
		if err := s.db.Where("id = ?", targetID).First(&bot).Error; err == nil {
			extra, _ := json.Marshal(map[string]any{"bot_type": bot.Type})
			return bot.Name, bot.Description, string(extra)
		}
	case model.ApprovalTypeChannel:
		var channel model.Channel
		if err := s.db.Where("id = ?", targetID).First(&channel).Error; err == nil {
			extra, _ := json.Marshal(map[string]any{
				"channel_id":         channel.ID,
				"publish_permission": channel.PublishPermission,
			})
			return channel.Name, channel.Description, string(extra)
		}
	case model.ApprovalTypeGroupAI:
		var group model.Group
		if err := s.db.Where("id = ?", targetID).First(&group).Error; err == nil {
			aiConfig := group.GetAIConfig()
			extra, _ := json.Marshal(map[string]any{
				"group_id":        group.ID,
				"conversation_id": group.ConversationID,
				"assistant_name":  aiConfig.AssistantName,
			})
			return group.Name, "群聊AI助手", string(extra)
		}
	}
	return "", "", "{}"
}

// fetchUserSnapshot 返回申请人的昵称和头像
func (s *ApprovalService) fetchUserSnapshot(userID uint) (name, avatar string) {
	var user model.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err == nil {
		return user.Nickname, user.Avatar
	}
	return "", ""
}

func (s *ApprovalService) GetApproval(targetType string, targetID uint) (*model.Approval, error) {
	var approval model.Approval
	err := s.db.Where("target_type = ? AND target_id = ?", targetType, targetID).Order("created_at DESC").First(&approval).Error
	return &approval, err
}

// ReapplyApproval 重新申请审批：将已有审批记录重置为 pending 状态
func (s *ApprovalService) ReapplyApproval(approvalID uint, appliedBy uint) error {
	now := time.Now()
	return s.db.Model(&model.Approval{}).Where("id = ?", approvalID).Updates(map[string]interface{}{
		"status":        model.ApprovalStatusPending,
		"applied_at":    now,
		"applied_by":    appliedBy,
		"approved_at":   nil,
		"approved_by":   nil,
		"reject_reason": "",
	}).Error
}

func (s *ApprovalService) SendNotification(notification ApprovalNotification) {
	if ws.GlobalHub != nil {
		msg, _ := json.Marshal(map[string]interface{}{
			"type":          "approval_notification",
			"entity_name":   notification.EntityName,
			"entity_type":   notification.EntityType,
			"user_id":       notification.UserID,
			"action":        string(notification.Action),
			"reason":        notification.Reason,
			"extra_context": notification.ExtraContext,
			"created_at":    time.Now().Unix(),
		})
		ws.GlobalHub.Broadcast <- msg
	}
}

func (s *ApprovalService) IsApprovalEnabled(entityType string) bool {
	var config model.ApprovalConfig
	if err := s.db.Where("type = ?", entityType).First(&config).Error; err != nil || !config.Enabled {
		return false
	}
	return true
}

func (s *ApprovalService) HandleApprovalRequest(c *gin.Context, entityType string, targetID uint, userID uint) {
	var config model.ApprovalConfig
	if err := s.db.Where("type = ?", entityType).First(&config).Error; err != nil || !config.Enabled {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "无需审批",
		})
		return
	}

	err := s.CreateApproval(entityType, targetID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "申请提交失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "申请已提交，等待审批",
	})
}

// ApprovalHandler 审批管理处理器
type ApprovalHandler struct {
	service *ApprovalService
}

func NewApprovalHandler(service *ApprovalService) *ApprovalHandler {
	return &ApprovalHandler{service: service}
}

func (h *ApprovalHandler) RegisterRoutes(router *gin.RouterGroup) {
	approvals := router.Group("/approvals")
	{
		approvals.GET("", h.List)
		approvals.POST("/:type/:id/approve", h.Approve)
		approvals.POST("/:type/:id/reject", h.Reject)
		approvals.GET("/configs", h.GetConfigs)
		approvals.PUT("/configs/:type", h.UpdateConfig)
	}
}

func (h *ApprovalHandler) List(c *gin.Context) {
	entityType := c.Query("type")
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	items, total, err := h.service.ListApprovals(entityType, status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": items, "total": total}})
}

func (h *ApprovalHandler) Approve(c *gin.Context) {
	entityType := c.Param("type")
	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 32)
	adminID := c.GetUint("user_id")

	if err := h.service.Approve(entityType, uint(id), adminID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "审批通过"})
}

func (h *ApprovalHandler) Reject(c *gin.Context) {
	entityType := c.Param("type")
	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 32)
	adminID := c.GetUint("user_id")

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误"})
		return
	}

	if err := h.service.Reject(entityType, uint(id), adminID, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已拒绝"})
}

func (h *ApprovalHandler) GetConfigs(c *gin.Context) {
	var configs []model.ApprovalConfig
	if err := h.service.db.Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": configs})
}

func (h *ApprovalHandler) UpdateConfig(c *gin.Context) {
	entityType := c.Param("type")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误"})
		return
	}

	var config model.ApprovalConfig
	if err := h.service.db.Where("type = ?", entityType).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			config = model.ApprovalConfig{Type: entityType}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
			return
		}
	}

	config.Enabled = req.Enabled
	if err := h.service.db.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}
