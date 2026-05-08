package service

import (
	"qim-server/model"

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
