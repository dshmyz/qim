package service

import (
	"errors"
	"fmt"

	"github.com/dshmyz/qim/qim-server/model"
	"gorm.io/gorm"
)

// AI 消息反馈服务的哨兵错误。
var (
	ErrFeedbackNotFound     = errors.New("反馈不存在")
	ErrFeedbackInvalidParam = errors.New("无效的反馈参数")
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

// ErrFeedbackServiceNotReady / ErrFeedbackForbidden 反馈服务哨兵错误。
var (
	ErrFeedbackServiceNotReady = errors.New("反馈服务不可用")
	ErrFeedbackForbidden       = errors.New("无权对该消息反馈")
)
