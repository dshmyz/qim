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

func (s *BlacklistService) GetBlacklist(page, pageSize int, keyword string) ([]model.Blacklist, int64, error) {
	var blacklist []model.Blacklist
	var total int64

	query := s.db.Model(&model.Blacklist{}).Preload("User")
	if keyword != "" {
		query = query.Joins("JOIN users ON users.id = blacklist.user_id").
			Where("users.username LIKE ? OR users.nickname LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&blacklist).Error
	return blacklist, total, err
}

func (s *BlacklistService) AddToBlacklist(blacklist *model.Blacklist) error {
	var existing model.Blacklist
	err := s.db.Where("user_id = ?", blacklist.UserID).First(&existing).Error
	if err == nil {
		return gorm.ErrDuplicatedKey
	}
	return s.db.Create(blacklist).Error
}

func (s *BlacklistService) RemoveFromBlacklist(id uint) error {
	return s.db.Delete(&model.Blacklist{}, id).Error
}

func (s *BlacklistService) IsUserBlacklisted(userID uint) bool {
	var count int64
	s.db.Model(&model.Blacklist{}).Where("user_id = ?", userID).Count(&count)
	return count > 0
}
