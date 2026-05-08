package service

import (
	"qim-server/model"

	"gorm.io/gorm"
)

type SystemConfigService struct {
	db *gorm.DB
}

func NewSystemConfigService(db *gorm.DB) *SystemConfigService {
	return &SystemConfigService{db: db}
}

func (s *SystemConfigService) GetConfig(key string) (*model.SystemConfig, error) {
	var config model.SystemConfig
	err := s.db.Where("key = ?", key).First(&config).Error
	return &config, err
}

func (s *SystemConfigService) GetAllConfigs() ([]model.SystemConfig, error) {
	var configs []model.SystemConfig
	err := s.db.Find(&configs).Error
	return configs, err
}

func (s *SystemConfigService) UpdateConfig(config *model.SystemConfig) error {
	return s.db.Save(config).Error
}

func (s *SystemConfigService) CreateConfig(config *model.SystemConfig) error {
	return s.db.Create(config).Error
}
