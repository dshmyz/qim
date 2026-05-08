package service

import (
	"qim-server/model"

	"gorm.io/gorm"
)

type AIProviderService struct {
	db *gorm.DB
}

func NewAIProviderService(db *gorm.DB) *AIProviderService {
	return &AIProviderService{db: db}
}

func (s *AIProviderService) GetProviders() ([]model.AIProvider, error) {
	var providers []model.AIProvider
	err := s.db.Order("created_at DESC").Find(&providers).Error
	return providers, err
}

func (s *AIProviderService) GetProviderByID(id uint) (*model.AIProvider, error) {
	var provider model.AIProvider
	err := s.db.First(&provider, id).Error
	return &provider, err
}

func (s *AIProviderService) CreateProvider(provider *model.AIProvider) error {
	return s.db.Create(provider).Error
}

func (s *AIProviderService) UpdateProvider(provider *model.AIProvider) error {
	return s.db.Save(provider).Error
}

func (s *AIProviderService) DeleteProvider(id uint) error {
	return s.db.Delete(&model.AIProvider{}, id).Error
}
