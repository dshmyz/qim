package service

import (
	"qim-server/model"

	"gorm.io/gorm"
)

type BlacklistService struct {
	db *gorm.DB
}

func NewBlacklistService(db *gorm.DB) *BlacklistService {
	return &BlacklistService{db: db}
}

func (s *BlacklistService) GetBlacklist() ([]model.Blacklist, error) {
	var blacklist []model.Blacklist
	err := s.db.Order("created_at DESC").Find(&blacklist).Error
	return blacklist, err
}

func (s *BlacklistService) AddToBlacklist(blacklist *model.Blacklist) error {
	return s.db.Create(blacklist).Error
}

func (s *BlacklistService) RemoveFromBlacklist(id uint) error {
	return s.db.Delete(&model.Blacklist{}, id).Error
}
