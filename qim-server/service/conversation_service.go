package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"qim-server/cache"
	"qim-server/model"
	"qim-server/repository"
	"qim-server/ws"

	"gorm.io/gorm"
)

var ErrConversationNotFound = errors.New("conversation not found")
var ErrConversationForbidden = errors.New("access forbidden")
var ErrNotConversationOwner = errors.New("only owner can perform this action")

type ConversationService struct {
	db        *gorm.DB
	convRepo  repository.ConversationRepository
	userRepo  repository.UserRepository
	groupRepo repository.GroupRepository
}

func NewConversationService(db *gorm.DB) *ConversationService {
	return &ConversationService{
		db:        db,
		convRepo:  repository.NewConversationRepository(db),
		userRepo:  repository.NewUserRepository(db),
		groupRepo: repository.NewGroupRepository(db),
	}
}

func (s *ConversationService) GetDB() *gorm.DB {
	return s.db
}

type ConversationWithPin struct {
	model.Conversation
	IsPinned        bool   `json:"is_pinned"`
	IP              string `json:"ip,omitempty"`
	Status          string `json:"status,omitempty"`
	Signature       string `json:"signature,omitempty"`
	OtherMemberID   uint   `json:"other_member_id,omitempty"`
	OtherMemberName string `json:"other_member_name,omitempty"`
}

func (s *ConversationService) GetConversations(userID uint) ([]ConversationWithPin, error) {
	ctx := context.Background()

	conversations, err := s.convRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var result []ConversationWithPin
	for _, conv := range conversations {
		var session model.ConversationSession
		s.db.Where("user_id = ? AND conversation_id = ?", userID, conv.ID).
			FirstOrCreate(&session, model.ConversationSession{
				IsPinned:      false,
				LastVisitedAt: time.Now(),
			})

		convWithPin := ConversationWithPin{
			Conversation: *conv,
			IsPinned:     session.IsPinned,
		}

		if conv.Type == "single" {
			for _, m := range conv.Members {
				if m.UserID != userID && m.UserID > 0 {
					convWithPin.IP = m.User.IP
					convWithPin.Status = m.User.Status
					convWithPin.Signature = m.User.Signature
					convWithPin.OtherMemberID = m.User.ID
					convWithPin.OtherMemberName = m.User.Nickname
					break
				}
			}
		}

		result = append(result, convWithPin)
	}

	return result, nil
}

func (s *ConversationService) GetConversation(convID uint) (*model.Conversation, error) {
	ctx := context.Background()

	conv, err := s.convRepo.FindByID(ctx, convID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}

	return conv, nil
}

func (s *ConversationService) GetConversationWithAccessCheck(convID, userID uint) (*model.Conversation, error) {
	ctx := context.Background()

	conv, err := s.convRepo.FindByID(ctx, convID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}

	isMember, err := s.convRepo.IsMember(ctx, convID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrConversationForbidden
	}

	return conv, nil
}

func (s *ConversationService) CreateSingleConversation(userID1, userID2 uint) (*model.Conversation, error) {
	ctx := context.Background()

	existingConv, err := s.convRepo.FindSingleConversation(ctx, userID1, userID2)
	if err == nil && existingConv != nil {
		return existingConv, nil
	}

	if userID1 == userID2 {
		conv := &model.Conversation{Type: "single"}
		if err := s.convRepo.Create(ctx, conv); err != nil {
			return nil, err
		}

		if err := s.convRepo.AddMember(ctx, conv.ID, userID1, "member"); err != nil {
			return nil, err
		}

		return s.convRepo.FindByID(ctx, conv.ID)
	}

	_, err = s.userRepo.FindByID(ctx, userID2)
	if err != nil {
		return nil, err
	}

	conv := &model.Conversation{Type: "single"}
	if err := s.convRepo.Create(ctx, conv); err != nil {
		return nil, err
	}

	if err := s.convRepo.AddMember(ctx, conv.ID, userID1, "member"); err != nil {
		return nil, err
	}
	if err := s.convRepo.AddMember(ctx, conv.ID, userID2, "member"); err != nil {
		return nil, err
	}

	return s.convRepo.FindByID(ctx, conv.ID)
}

func (s *ConversationService) CreateGroupConversation(name string, creatorID uint, memberIDs []uint, avatar string) (*model.Conversation, error) {
	ctx := context.Background()

	conv := &model.Conversation{Type: "group"}
	if err := s.convRepo.Create(ctx, conv); err != nil {
		return nil, err
	}

	group := &model.Group{
		ConversationID:   conv.ID,
		GroupType:        "group",
		Name:             name,
		Avatar:           avatar,
		CreatorID:        creatorID,
		InvitePermission: "owner_admin",
	}
	if err := s.groupRepo.Create(ctx, group); err != nil {
		return nil, err
	}

	if err := s.convRepo.AddMember(ctx, conv.ID, creatorID, "owner"); err != nil {
		return nil, err
	}

	for _, mid := range memberIDs {
		if mid != creatorID {
			if err := s.convRepo.AddMember(ctx, conv.ID, mid, "member"); err != nil {
				return nil, err
			}
		}
	}

	return s.convRepo.FindByID(ctx, conv.ID)
}

func (s *ConversationService) CreateDiscussionConversation(name string, creatorID uint, memberIDs []uint, avatar string) (*model.Conversation, error) {
	ctx := context.Background()

	conv := &model.Conversation{Type: "discussion"}
	if err := s.convRepo.Create(ctx, conv); err != nil {
		return nil, err
	}

	group := &model.Group{
		ConversationID:   conv.ID,
		GroupType:        "discussion",
		Name:             name,
		Avatar:           avatar,
		CreatorID:        creatorID,
		InvitePermission: "owner_admin",
	}
	if err := s.groupRepo.Create(ctx, group); err != nil {
		return nil, err
	}

	if err := s.convRepo.AddMember(ctx, conv.ID, creatorID, "owner"); err != nil {
		return nil, err
	}

	for _, mid := range memberIDs {
		if mid != creatorID {
			if err := s.convRepo.AddMember(ctx, conv.ID, mid, "member"); err != nil {
				return nil, err
			}
		}
	}

	return s.convRepo.FindByID(ctx, conv.ID)
}

func (s *ConversationService) UpdateConversation(convID uint, updates map[string]interface{}) error {
	ctx := context.Background()

	_, err := s.convRepo.FindByID(ctx, convID)
	if err != nil {
		return ErrConversationNotFound
	}

	result := s.db.WithContext(ctx).Model(&model.Conversation{}).Where("id = ?", convID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrConversationNotFound
	}

	return nil
}

func (s *ConversationService) DeleteConversation(convID, userID uint) error {
	ctx := context.Background()

	conv, err := s.convRepo.FindByID(ctx, convID)
	if err != nil {
		return ErrConversationNotFound
	}

	if conv.Type != "group" && conv.Type != "discussion" {
		return errors.New("only group or discussion can be deleted")
	}

	isMember, err := s.convRepo.IsMember(ctx, convID, userID)
	if err != nil || !isMember {
		return ErrConversationForbidden
	}

	members, err := s.convRepo.GetMembers(ctx, convID)
	if err != nil {
		return err
	}

	isOwner := false
	for _, m := range members {
		if m.UserID == userID && m.Role == "owner" {
			isOwner = true
			break
		}
	}
	if !isOwner {
		return ErrNotConversationOwner
	}

	tx := s.db.Begin()

	group, err := s.groupRepo.FindByConversationID(ctx, convID)
	if err == nil {
		group.Name = "[已解散] " + group.Name
		if err := tx.Save(group).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Where("conversation_id = ?", convID).Delete(&model.ConversationMember{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	conv.IsDeleted = true
	if err := tx.Save(conv).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	if ws.GlobalHub != nil {
		dissolveMsg := ws.WSMessage{
			Type: "conversation_deleted",
			Data: map[string]interface{}{
				"conversation_id": conv.ID,
				"message":         "群聊已被解散",
			},
		}
		jsonMsg, _ := json.Marshal(dissolveMsg)
		ws.GlobalHub.SendToConversation(convID, 0, jsonMsg)
	}

	return nil
}

func (s *ConversationService) GetConversationMembers(convID uint) ([]model.ConversationMember, error) {
	cacheKey := fmt.Sprintf("conv_members:%d", convID)

	if data, ok := cache.ConversationMemberCache.Get(cacheKey); ok {
		if jsonData, ok := data.([]byte); ok {
			var members []model.ConversationMember
			if err := json.Unmarshal(jsonData, &members); err == nil {
				return members, nil
			}
		}
	}

	ctx := context.Background()
	members, err := s.convRepo.GetMembers(ctx, convID)
	if err != nil {
		return nil, err
	}

	if jsonData, err := json.Marshal(&members); err == nil {
		cache.ConversationMemberCache.Put(cacheKey, jsonData)
	}

	return members, nil
}

func (s *ConversationService) IsConversationMember(convID, userID uint) (bool, error) {
	ctx := context.Background()
	return s.convRepo.IsMember(ctx, convID, userID)
}

func (s *ConversationService) UpdateMemberRole(convID, userID uint, role string) error {
	ctx := context.Background()
	return s.convRepo.UpdateMemberRole(ctx, convID, userID, role)
}

func (s *ConversationService) SetConversationMute(convID, userID uint, muted bool) (*model.ConversationMember, error) {
	ctx := context.Background()

	isMember, err := s.convRepo.IsMember(ctx, convID, userID)
	if err != nil || !isMember {
		return nil, ErrConversationForbidden
	}

	member, err := s.convRepo.SetMute(ctx, convID, userID, muted)
	if err != nil {
		return nil, err
	}

	return member, nil
}

func (s *ConversationService) SetConversationPin(convID, userID uint, isPinned bool) (*model.ConversationSession, error) {
	var session model.ConversationSession
	result := s.db.Where("user_id = ? AND conversation_id = ?", userID, convID).First(&session)

	if result.Error != nil {
		session = model.ConversationSession{
			UserID:         userID,
			ConversationID: convID,
			IsPinned:       isPinned,
			LastVisitedAt:  time.Now(),
		}
		if isPinned {
			now := time.Now()
			session.PinnedAt = &now
		}
		if err := s.db.Create(&session).Error; err != nil {
			return nil, err
		}
	} else {
		session.IsPinned = isPinned
		if isPinned {
			now := time.Now()
			session.PinnedAt = &now
		} else {
			session.PinnedAt = nil
		}
		if err := s.db.Save(&session).Error; err != nil {
			return nil, err
		}
	}

	return &session, nil
}

func (s *ConversationService) GetMember(convID, userID uint) (*model.ConversationMember, error) {
	ctx := context.Background()
	var member model.ConversationMember
	err := s.db.WithContext(ctx).Where("conversation_id = ? AND user_id = ?", convID, userID).First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

func (s *ConversationService) CreateMember(member *model.ConversationMember) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Create(member).Error
}

func (s *ConversationService) RemoveMember(convID, userID uint) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Where("conversation_id = ? AND user_id = ?", convID, userID).Delete(&model.ConversationMember{}).Error
}

func (s *ConversationService) GetMembersExcept(convID, excludeUserID uint) ([]model.ConversationMember, error) {
	ctx := context.Background()
	var members []model.ConversationMember
	err := s.db.WithContext(ctx).Where("conversation_id = ? AND user_id != ?", convID, excludeUserID).Find(&members).Error
	return members, err
}

func (s *ConversationService) GetMembersByRoles(convID uint, roles []string) ([]model.ConversationMember, error) {
	ctx := context.Background()
	var members []model.ConversationMember
	err := s.db.WithContext(ctx).Where("conversation_id = ? AND role IN ?", convID, roles).Find(&members).Error
	return members, err
}

func (s *ConversationService) GetMembersWithUser(convID uint) ([]model.ConversationMember, error) {
	ctx := context.Background()
	var members []model.ConversationMember
	err := s.db.WithContext(ctx).Where("conversation_id = ?", convID).Preload("User").Find(&members).Error
	return members, err
}

func (s *ConversationService) UpdateMember(member *model.ConversationMember) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Save(member).Error
}

func (s *ConversationService) IncrementUnreadCount(convID, excludeUserID uint) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id != ?", convID, excludeUserID).
		UpdateColumn("unread_count", gorm.Expr("unread_count + 1")).Error
}
