package service

import (
	"qim-server/model"

	"gorm.io/gorm"
)

type VersionService struct {
	db *gorm.DB
}

func NewVersionService(db *gorm.DB) *VersionService {
	return &VersionService{db: db}
}

func (s *VersionService) GetVersions() ([]model.ClientVersion, error) {
	var versions []model.ClientVersion
	err := s.db.Order("created_at DESC").Find(&versions).Error
	return versions, err
}

func (s *VersionService) GetVersionByID(id uint) (*model.ClientVersion, error) {
	var version model.ClientVersion
	err := s.db.First(&version, id).Error
	return &version, err
}

func (s *VersionService) CreateVersion(version *model.ClientVersion) error {
	return s.db.Create(version).Error
}

func (s *VersionService) UpdateVersion(version *model.ClientVersion) error {
	return s.db.Save(version).Error
}

func (s *VersionService) DeleteVersion(id uint) error {
	return s.db.Delete(&model.ClientVersion{}, id).Error
}
