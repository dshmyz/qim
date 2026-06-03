package repository

import (
	"context"

	"github.com/dshmyz/qim/qim-server/model"

	"gorm.io/gorm"
)

type groupRepository struct {
	*baseRepository[model.Group]
	db *gorm.DB
}

func NewGroupRepository(db *gorm.DB) GroupRepository {
	return &groupRepository{
		baseRepository: &baseRepository[model.Group]{db: db},
		db:             db,
	}
}

func (r *groupRepository) FindByConversationID(ctx context.Context, conversationID uint) (*model.Group, error) {
	var group model.Group
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Preload("Conversation").
		Preload("Conversation.Members").
		Preload("Conversation.Members.User").
		First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *groupRepository) FindByCreatorID(ctx context.Context, creatorID uint) ([]*model.Group, error) {
	var groups []*model.Group
	err := r.db.WithContext(ctx).
		Where("creator_id = ?", creatorID).
		Preload("Conversation").
		Find(&groups).Error
	return groups, err
}

func (r *groupRepository) UpdateAnnouncement(ctx context.Context, id uint, announcement string) error {
	return r.db.WithContext(ctx).
		Model(&model.Group{}).
		Where("id = ?", id).
		Update("announcement", announcement).Error
}

func (r *groupRepository) AddMember(ctx context.Context, groupID, userID uint) error {
	var group model.Group
	if err := r.db.WithContext(ctx).First(&group, groupID).Error; err != nil {
		return err
	}

	return r.db.WithContext(ctx).Create(&model.ConversationMember{
		ConversationID: group.ConversationID,
		UserID:         userID,
		Role:           "member",
	}).Error
}

func (r *groupRepository) RemoveMember(ctx context.Context, groupID, userID uint) error {
	var group model.Group
	if err := r.db.WithContext(ctx).First(&group, groupID).Error; err != nil {
		return err
	}

	return r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", group.ConversationID, userID).
		Delete(&model.ConversationMember{}).Error
}

func (r *groupRepository) WithTx(tx *gorm.DB) BaseRepository[model.Group] {
	return &groupRepository{
		baseRepository: &baseRepository[model.Group]{db: tx},
		db:             tx,
	}
}
