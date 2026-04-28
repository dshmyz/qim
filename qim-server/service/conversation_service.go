package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"qim-server/cache"
	"qim-server/database"
	"qim-server/model"
	"qim-server/ws"

	"gorm.io/gorm"
)

var ErrConversationNotFound = errors.New("conversation not found")
var ErrConversationForbidden = errors.New("access forbidden")
var ErrNotConversationOwner = errors.New("only owner can perform this action")

type ConversationService struct{}

func NewConversationService() *ConversationService {
	return &ConversationService{}
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
	db := database.GetDB()

	var convMembers []model.ConversationMember
	if err := db.Where("user_id = ?", userID).Preload("Conversation").Preload("Conversation.LastMessage").Preload("Conversation.Members").Preload("Conversation.Members.User").Find(&convMembers).Error; err != nil {
		return nil, err
	}

	var conversations []ConversationWithPin
	for _, cm := range convMembers {
		var session model.ConversationSession
		db.Where("user_id = ? AND conversation_id = ?", userID, cm.Conversation.ID).FirstOrCreate(&session, model.ConversationSession{
			IsPinned:      false,
			LastVisitedAt: time.Now(),
		})

		convWithPin := ConversationWithPin{
			Conversation: cm.Conversation,
			IsPinned:     session.IsPinned,
		}

		if cm.Conversation.Type == "single" {
			for _, m := range cm.Conversation.Members {
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

		conversations = append(conversations, convWithPin)
	}

	return conversations, nil
}

func (s *ConversationService) GetConversation(convID uint) (*model.Conversation, error) {
	db := database.GetDB()

	var conv model.Conversation
	if err := db.Preload("Members").Preload("Members.User").First(&conv, convID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}

	return &conv, nil
}

func (s *ConversationService) GetConversationWithAccessCheck(convID, userID uint) (*model.Conversation, error) {
	db := database.GetDB()

	var conv model.Conversation
	if err := db.Preload("Members").Preload("Members.User").First(&conv, convID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", conv.ID, userID).First(&member).Error; err != nil {
		return nil, ErrConversationForbidden
	}

	return &conv, nil
}

func (s *ConversationService) CreateSingleConversation(userID1, userID2 uint) (*model.Conversation, error) {
	db := database.GetDB()

	var existingConv model.Conversation
	err := db.Raw(`
		SELECT c.* FROM conversations c
		JOIN conversation_members cm1 ON c.id = cm1.conversation_id
		JOIN conversation_members cm2 ON c.id = cm2.conversation_id
		WHERE c.type = 'single'
		AND cm1.user_id = ? AND cm2.user_id = ?
	`, userID1, userID2).Scan(&existingConv).Error

	if err == nil && existingConv.ID > 0 {
		db.Preload("Members").Preload("Members.User").First(&existingConv, existingConv.ID)
		return &existingConv, nil
	}

	if userID1 == userID2 {
		var targetUser model.User
		if err := db.First(&targetUser, userID1).Error; err != nil {
			return nil, err
		}

		conv := model.Conversation{
			Type: "single",
		}
		if err := db.Create(&conv).Error; err != nil {
			return nil, err
		}

		if err := db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID1, Role: "member"}).Error; err != nil {
			return nil, err
		}

		db.Preload("Members").Preload("Members.User").First(&conv, conv.ID)
		return &conv, nil
	}

	var targetUser model.User
	if err := db.First(&targetUser, userID2).Error; err != nil {
		return nil, err
	}

	conv := model.Conversation{
		Type: "single",
	}
	if err := db.Create(&conv).Error; err != nil {
		return nil, err
	}

	if err := db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID1, Role: "member"}).Error; err != nil {
		return nil, err
	}
	if err := db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID2, Role: "member"}).Error; err != nil {
		return nil, err
	}

	db.Preload("Members").Preload("Members.User").First(&conv, conv.ID)
	return &conv, nil
}

func (s *ConversationService) CreateGroupConversation(name string, creatorID uint, memberIDs []uint, avatar string) (*model.Conversation, error) {
	db := database.GetDB()

	conv := model.Conversation{
		Type: "group",
	}
	if err := db.Create(&conv).Error; err != nil {
		return nil, err
	}

	// 创建群聊记录
	group := model.Group{
		ConversationID:   conv.ID,
		GroupType:        "group",
		Name:             name,
		Avatar:           avatar,
		CreatorID:        creatorID,
		InvitePermission: "owner_admin",
		AIEnabled:        false,
	}
	if err := db.Create(&group).Error; err != nil {
		return nil, err
	}

	if err := db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: creatorID, Role: "owner"}).Error; err != nil {
		return nil, err
	}

	for _, mid := range memberIDs {
		if mid != creatorID {
			if err := db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: mid, Role: "member"}).Error; err != nil {
				return nil, err
			}
		}
	}

	db.Preload("Members").Preload("Members.User").First(&conv, conv.ID)
	return &conv, nil
}

func (s *ConversationService) CreateDiscussionConversation(name string, creatorID uint, memberIDs []uint, avatar string) (*model.Conversation, error) {
	db := database.GetDB()

	conv := model.Conversation{
		Type: "discussion",
	}
	if err := db.Create(&conv).Error; err != nil {
		return nil, err
	}

	// 创建群聊记录
	group := model.Group{
		ConversationID:   conv.ID,
		GroupType:        "discussion",
		Name:             name,
		Avatar:           avatar,
		CreatorID:        creatorID,
		InvitePermission: "owner_admin",
		AIEnabled:        false,
	}
	if err := db.Create(&group).Error; err != nil {
		return nil, err
	}

	if err := db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: creatorID, Role: "owner"}).Error; err != nil {
		return nil, err
	}

	for _, mid := range memberIDs {
		if mid != creatorID {
			if err := db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: mid, Role: "member"}).Error; err != nil {
				return nil, err
			}
		}
	}

	db.Preload("Members").Preload("Members.User").First(&conv, conv.ID)
	return &conv, nil
}

func (s *ConversationService) UpdateConversation(convID uint, updates map[string]interface{}) error {
	db := database.GetDB()

	result := db.Model(&model.Conversation{}).Where("id = ?", convID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrConversationNotFound
	}

	return nil
}

func (s *ConversationService) DeleteConversation(convID, userID uint) error {
	db := database.GetDB()

	var conv model.Conversation
	if err := db.First(&conv, convID).Error; err != nil {
		return ErrConversationNotFound
	}

	if conv.Type != "group" && conv.Type != "discussion" {
		return errors.New("only group or discussion can be deleted")
	}

	var currentMember model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, userID).First(&currentMember).Error; err != nil {
		return ErrConversationForbidden
	}

	if currentMember.Role != "owner" {
		return ErrNotConversationOwner
	}

	tx := db.Begin()

	// 获取群聊信息并更新
	var group model.Group
	if err := tx.Where("conversation_id = ?", convID).First(&group).Error; err == nil {
		group.Name = "[已解散] " + group.Name
		if err := tx.Save(&group).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Where("conversation_id = ?", convID).Delete(&model.ConversationMember{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	conv.IsDeleted = true
	if err := tx.Save(&conv).Error; err != nil {
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

	db := database.GetDB()

	var members []model.ConversationMember
	if err := db.Where("conversation_id = ?", convID).Preload("User").Find(&members).Error; err != nil {
		return nil, err
	}

	if jsonData, err := json.Marshal(&members); err == nil {
		cache.ConversationMemberCache.Put(cacheKey, jsonData)
	}

	return members, nil
}

func (s *ConversationService) IsConversationMember(convID, userID uint) (bool, error) {
	db := database.GetDB()

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, userID).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (s *ConversationService) UpdateMemberRole(convID, userID uint, role string) error {
	db := database.GetDB()

	result := db.Model(&model.ConversationMember{}).Where("conversation_id = ? AND user_id = ?", convID, userID).Update("role", role)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("member not found")
	}

	return nil
}

func (s *ConversationService) SetConversationMute(convID, userID uint, muted bool) (*model.ConversationMember, error) {
	db := database.GetDB()

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, userID).First(&member).Error; err != nil {
		return nil, ErrConversationForbidden
	}

	member.Muted = muted
	if err := db.Save(&member).Error; err != nil {
		return nil, err
	}

	return &member, nil
}

func (s *ConversationService) SetConversationPin(convID, userID uint, isPinned bool) (*model.ConversationSession, error) {
	db := database.GetDB()

	var session model.ConversationSession
	result := db.Where("user_id = ? AND conversation_id = ?", userID, convID).First(&session)

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
		if err := db.Create(&session).Error; err != nil {
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
		if err := db.Save(&session).Error; err != nil {
			return nil, err
		}
	}

	return &session, nil
}
