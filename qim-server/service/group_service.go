package service

import (
	"context"

	"qim-server/model"
	"qim-server/repository"

	"gorm.io/gorm"
)

type GroupService struct {
	repo repository.GroupRepository
	db   *gorm.DB
}

func NewGroupService(db *gorm.DB) *GroupService {
	return &GroupService{
		repo: repository.NewGroupRepository(db),
		db:   db,
	}
}

func (s *GroupService) GetGroupByConversationID(conversationID uint) (*model.Group, error) {
	ctx := context.Background()
	var group model.Group
	err := s.db.WithContext(ctx).Where("conversation_id = ?", conversationID).First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *GroupService) UpdateGroup(group *model.Group) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Save(group).Error
}

func (s *GroupService) UpdateGroupWithTx(tx *gorm.DB, group *model.Group) error {
	return tx.Save(group).Error
}

func (s *GroupService) GetGroupByConversationIDWithTx(tx *gorm.DB, conversationID uint) (*model.Group, error) {
	var group model.Group
	err := tx.Where("conversation_id = ?", conversationID).First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}
