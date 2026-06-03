package service

import (
	"context"
	"fmt"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/repository"

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

type TransferOwnerParams struct {
	ConversationID uint
	CurrentOwnerID uint
	NewOwnerID     uint
	CurrentMember  *model.ConversationMember
	TargetMember   *model.ConversationMember
	Group          *model.Group
}

func (s *GroupService) TransferOwner(params *TransferOwnerParams) error {
	tx := s.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("开启事务失败: %w", tx.Error)
	}

	params.CurrentMember.Role = "admin"
	if err := tx.Save(params.CurrentMember).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新原群主角色失败: %w", err)
	}

	params.TargetMember.Role = "owner"
	if err := tx.Save(params.TargetMember).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新新群主角色失败: %w", err)
	}

	if err := tx.Model(params.Group).Update("creator_id", params.TargetMember.UserID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新群创建者失败: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}
