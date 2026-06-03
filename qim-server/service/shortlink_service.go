package service

import (
	"github.com/dshmyz/qim/qim-server/model"

	"gorm.io/gorm"
)

type ShortLinkService struct {
	db *gorm.DB
}

func NewShortLinkService(db *gorm.DB) *ShortLinkService {
	return &ShortLinkService{db: db}
}

func (s *ShortLinkService) GetShortLink(code string) (*model.ShortLink, error) {
	var link model.ShortLink
	err := s.db.Where("code = ?", code).First(&link).Error
	return &link, err
}

func (s *ShortLinkService) CreateShortLink(link *model.ShortLink) error {
	return s.db.Create(link).Error
}

func (s *ShortLinkService) GetLinks(page, pageSize int) ([]model.ShortLink, int64, error) {
	var links []model.ShortLink
	var total int64

	s.db.Model(&model.ShortLink{}).Count(&total)
	err := s.db.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&links).Error
	return links, total, err
}

func (s *ShortLinkService) DeleteLink(id uint) error {
	return s.db.Delete(&model.ShortLink{}, id).Error
}
