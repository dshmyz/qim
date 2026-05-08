package service

import (
	"qim-server/model"

	"gorm.io/gorm"
)

type GroupDocumentService struct {
	db *gorm.DB
}

func NewGroupDocumentService(db *gorm.DB) *GroupDocumentService {
	return &GroupDocumentService{db: db}
}

func (s *GroupDocumentService) GetDocumentsByGroup(groupID uint) ([]model.GroupDocument, error) {
	var docs []model.GroupDocument
	err := s.db.Where("group_id = ?", groupID).Order("created_at DESC").Find(&docs).Error
	return docs, err
}

func (s *GroupDocumentService) GetDocumentByID(id uint) (*model.GroupDocument, error) {
	var doc model.GroupDocument
	err := s.db.First(&doc, id).Error
	return &doc, err
}

func (s *GroupDocumentService) CreateDocument(doc *model.GroupDocument) error {
	return s.db.Create(doc).Error
}

func (s *GroupDocumentService) UpdateDocument(doc *model.GroupDocument) error {
	return s.db.Save(doc).Error
}

func (s *GroupDocumentService) DeleteDocument(id uint) error {
	return s.db.Delete(&model.GroupDocument{}, id).Error
}
