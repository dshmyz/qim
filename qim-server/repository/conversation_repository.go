package repository

import (
	"context"
	"time"

	"github.com/dshmyz/qim/qim-server/model"

	"gorm.io/gorm"
)

type conversationRepository struct {
	*baseRepository[model.Conversation]
	db *gorm.DB
}

func NewConversationRepository(db *gorm.DB) ConversationRepository {
	return &conversationRepository{
		baseRepository: &baseRepository[model.Conversation]{db: db},
		db:             db,
	}
}

func (r *conversationRepository) FindByUserID(ctx context.Context, userID uint) ([]*model.Conversation, error) {
	var convMembers []model.ConversationMember
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Preload("Conversation").
		Preload("Conversation.LastMessage").
		Preload("Conversation.Members").
		Preload("Conversation.Members.User").
		Find(&convMembers).Error
	if err != nil {
		return nil, err
	}

	conversations := make([]*model.Conversation, 0, len(convMembers))
	for _, cm := range convMembers {
		conversations = append(conversations, &cm.Conversation)
	}
	return conversations, nil
}

func (r *conversationRepository) FindSingleConversation(ctx context.Context, userID1, userID2 uint) (*model.Conversation, error) {
	var conv model.Conversation
	err := r.db.WithContext(ctx).
		Joins("JOIN conversation_members cm1 ON cm1.conversation_id = conversations.id").
		Joins("JOIN conversation_members cm2 ON cm2.conversation_id = conversations.id").
		Where("conversations.type = ?", "single").
		Where("cm1.user_id = ?", userID1).
		Where("cm2.user_id = ?", userID2).
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *conversationRepository) AddMember(ctx context.Context, conversationID, userID uint, role string) error {
	member := &model.ConversationMember{
		ConversationID: conversationID,
		UserID:         userID,
		Role:           role,
		JoinedAt:       time.Now(),
	}
	return r.db.WithContext(ctx).Create(member).Error
}

func (r *conversationRepository) RemoveMember(ctx context.Context, conversationID, userID uint) error {
	return r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Delete(&model.ConversationMember{}).Error
}

func (r *conversationRepository) UpdateMemberRole(ctx context.Context, conversationID, userID uint, role string) error {
	return r.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("role", role).Error
}

func (r *conversationRepository) IsMember(ctx context.Context, conversationID, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *conversationRepository) GetMembers(ctx context.Context, conversationID uint) ([]model.ConversationMember, error) {
	var members []model.ConversationMember
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Preload("User").
		Find(&members).Error
	return members, err
}

func (r *conversationRepository) SetMute(ctx context.Context, conversationID, userID uint, muted bool) (*model.ConversationMember, error) {
	var member model.ConversationMember
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		First(&member).Error; err != nil {
		return nil, err
	}

	member.Muted = muted
	if err := r.db.WithContext(ctx).Save(&member).Error; err != nil {
		return nil, err
	}

	return &member, nil
}

func (r *conversationRepository) WithTx(tx *gorm.DB) BaseRepository[model.Conversation] {
	return &conversationRepository{
		baseRepository: &baseRepository[model.Conversation]{db: tx},
		db:             tx,
	}
}
