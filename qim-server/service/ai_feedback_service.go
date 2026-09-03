package service

import (
	"errors"
	"fmt"

	"github.com/dshmyz/qim/qim-server/model"
	pkgErr "github.com/dshmyz/qim/qim-server/pkg/errors"
	"gorm.io/gorm"
)

// AI 消息反馈服务的哨兵错误。
var (
	ErrFeedbackNotFound     = pkgErr.NotFoundError("反馈不存在")
	ErrFeedbackInvalidParam = pkgErr.BadRequestError("无效的反馈参数")
)

// AIFeedbackService 用户对 AI 回复的显式反馈（👍/👎）。
// 主观反馈与 ai_reply_metrics 自动度量互补，是质量闭环的用户侧信号源。
type AIFeedbackService struct {
	db *gorm.DB
}

// NewAIFeedbackService 创建反馈服务。
func NewAIFeedbackService(db *gorm.DB) *AIFeedbackService {
	return &AIFeedbackService{db: db}
}

// SetFeedback 设置/修改/撤销（rating=0）用户对消息的反馈。
// 校验消息存在且用户在其会话内（防止对不可见消息刷反馈）。
func (s *AIFeedbackService) SetFeedback(userID, messageID uint, rating int) error {
	if s.db == nil {
		return ErrFeedbackServiceNotReady
	}
	if rating != 0 && rating != 1 && rating != -1 {
		return ErrFeedbackInvalidParam
	}

	var msg model.Message
	if err := s.db.Select("id, conversation_id").First(&msg, messageID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrFeedbackNotFound
		}
		return fmt.Errorf("查询消息失败: %w", err)
	}
	var member model.ConversationMember
	if err := s.db.Where("conversation_id = ? AND user_id = ?", msg.ConversationID, userID).
		First(&member).Error; err != nil {
		return ErrFeedbackForbidden
	}

	if rating == 0 {
		res := s.db.Where("message_id = ? AND user_id = ?", messageID, userID).
			Delete(&model.AIMessageFeedback{})
		if res.Error != nil {
			return fmt.Errorf("撤销反馈失败: %w", res.Error)
		}
		return nil
	}

	// upsert：存在则改 rating，否则创建
	var fb model.AIMessageFeedback
	if err := s.db.Where("message_id = ? AND user_id = ?", messageID, userID).
		First(&fb).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fb = model.AIMessageFeedback{MessageID: messageID, UserID: userID, Rating: rating}
			if err := s.db.Create(&fb).Error; err != nil {
				return fmt.Errorf("保存反馈失败: %w", err)
			}
			return nil
		}
		return fmt.Errorf("查询反馈失败: %w", err)
	}
	fb.Rating = rating
	if err := s.db.Model(&model.AIMessageFeedback{}).Where("id = ?", fb.ID).
		Update("rating", rating).Error; err != nil {
		return fmt.Errorf("更新反馈失败: %w", err)
	}
	return nil
}

// GetFeedback 查询用户对某条消息的当前反馈（0=无）。供客户端恢复选中态。
func (s *AIFeedbackService) GetFeedback(userID, messageID uint) (int, error) {
	if s.db == nil {
		return 0, ErrFeedbackServiceNotReady
	}
	var fb model.AIMessageFeedback
	if err := s.db.Where("message_id = ? AND user_id = ?", messageID, userID).
		First(&fb).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return fb.Rating, nil
}

// GetFeedbackBatch 批量查询用户对多条消息的反馈（一次 IN 查询）。
// 客户端打开会话时每条 AI 回复各查一次形成 N+1，列表层批量拉取替代。
// 返回 map[messageID]rating，未反馈的消息不在 map 中（调用方按 0 处理）。
func (s *AIFeedbackService) GetFeedbackBatch(userID uint, messageIDs []uint) (map[uint]int, error) {
	if s.db == nil {
		return nil, ErrFeedbackServiceNotReady
	}
	result := make(map[uint]int, len(messageIDs))
	if len(messageIDs) == 0 {
		return result, nil
	}
	var rows []model.AIMessageFeedback
	if err := s.db.Where("user_id = ? AND message_id IN ?", userID, messageIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		result[r.MessageID] = r.Rating
	}
	return result, nil
}

// ErrFeedbackServiceNotReady / ErrFeedbackForbidden 反馈服务哨兵错误。
var (
	ErrFeedbackServiceNotReady = pkgErr.InternalError("反馈服务不可用")
	// 保持既有 HTTP 语义（handler 原映射为 400）
	ErrFeedbackForbidden = pkgErr.BadRequestError("无权对该消息反馈")
)
