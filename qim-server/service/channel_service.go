package service

import (
	"errors"
	"fmt"

	"github.com/dshmyz/qim/qim-server/model"

	"gorm.io/gorm"
)

type ChannelService struct {
	db *gorm.DB
}

func NewChannelService(db *gorm.DB) *ChannelService {
	return &ChannelService{db: db}
}

func (s *ChannelService) GetChannels() ([]model.Channel, error) {
	var channels []model.Channel
	err := s.db.Order("sort_order ASC, created_at DESC").Find(&channels).Error
	return channels, err
}

func (s *ChannelService) GetChannelByID(id uint) (*model.Channel, error) {
	var channel model.Channel
	err := s.db.First(&channel, id).Error
	return &channel, err
}

func (s *ChannelService) CreateChannel(channel *model.Channel) error {
	return s.db.Create(channel).Error
}

func (s *ChannelService) UpdateChannel(channel *model.Channel) error {
	return s.db.Save(channel).Error
}

func (s *ChannelService) DeleteChannel(id uint) error {
	return s.db.Delete(&model.Channel{}, id).Error
}

// EnsureChannelUsable 校验频道是否处于可发布消息的状态。
// 仅 active 频道可用；pending/rejected/inactive 等状态均不允许发布，
// 这是审批流程的关键保护层（与列表过滤、订阅校验共同构成多层防护）。
func (s *ChannelService) EnsureChannelUsable(channel *model.Channel) error {
	switch channel.Status {
	case model.ChannelStatusActive:
		return nil
	case model.ChannelStatusPending:
		return errors.New("频道正在审批中，暂不可发布消息")
	case model.ChannelStatusRejected:
		return errors.New("频道已被拒绝，暂不可发布消息")
	case "inactive":
		return errors.New("频道已停用，暂不可发布消息")
	default:
		return fmt.Errorf("频道不可用，暂不可发布消息")
	}
}
