package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dshmyz/qim/qim-server/cache"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/repository"

	"gorm.io/gorm"
)

var ErrUserNotFound = errors.New("user not found")

type UserService struct {
	db       *gorm.DB
	userRepo repository.UserRepository
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{
		db:       db,
		userRepo: repository.NewUserRepository(db),
	}
}

func (s *UserService) WithRepo(repo repository.UserRepository) *UserService {
	return &UserService{
		db:       s.db,
		userRepo: repo,
	}
}

func (s *UserService) GetUser(userID uint) (*model.User, error) {
	cacheKey := fmt.Sprintf("user:%d", userID)

	if data, ok := cache.UserCache.Get(cacheKey); ok {
		if jsonData, ok := data.([]byte); ok {
			var user model.User
			if err := json.Unmarshal(jsonData, &user); err == nil {
				return &user, nil
			}
		}
	}

	user, err := s.userRepo.FindByID(context.Background(), userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if jsonData, err := json.Marshal(user); err == nil {
		cache.UserCache.Put(cacheKey, jsonData)
	}

	return user, nil
}

func (s *UserService) GetUserByUsername(username string) (*model.User, error) {
	user, err := s.userRepo.FindByUsername(context.Background(), username)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *UserService) UpdateUserStatus(userID uint, status string) error {
	if err := s.userRepo.UpdateStatus(context.Background(), userID, status); err != nil {
		return err
	}

	user, err := s.userRepo.FindByID(context.Background(), userID)
	if err == nil {
		cacheKey := fmt.Sprintf("user:%d", userID)
		if jsonData, err := json.Marshal(user); err == nil {
			cache.UserCache.Put(cacheKey, jsonData)
		}
	} else {
		logger.WithModule("UserService").Error("failed to fetch updated user for cache", "error", err)
	}

	return nil
}

func (s *UserService) SearchUsers(keyword string, limit int) ([]model.User, error) {
	if limit <= 0 {
		limit = 20
	}

	users, err := s.userRepo.Search(context.Background(), keyword, limit)
	if err != nil {
		return nil, err
	}

	result := make([]model.User, len(users))
	for i, u := range users {
		result[i] = *u
	}
	return result, nil
}

func (s *UserService) UpdateUser(userID uint, updates map[string]interface{}) (*model.User, error) {
	user, err := s.userRepo.FindByID(context.Background(), userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if err := s.db.Model(user).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 更新用户缓存：删除旧缓存，下次获取时会从数据库加载最新数据
	cacheKey := fmt.Sprintf("user:%d", userID)
	cache.UserCache.Delete(cacheKey)

	// 重新获取更新后的用户信息并缓存
	updatedUser, err := s.userRepo.FindByID(context.Background(), userID)
	if err == nil {
		if jsonData, err := json.Marshal(updatedUser); err == nil {
			cache.UserCache.Put(cacheKey, jsonData)
		}
	}

	return updatedUser, nil
}

func (s *UserService) CreateUser(username, password, nickname, avatar string) (*model.User, error) {
	nick := nickname
	if nick == "" {
		nick = username
	}

	av := avatar
	if av == "" {
		av = ""
	}

	user := &model.User{
		Username: username,
		Nickname: nick,
		Avatar:   av,
		Status:   "online",
	}

	if err := s.userRepo.Create(context.Background(), user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) IsUsernameExists(username string) (bool, error) {
	_, err := s.userRepo.FindByUsername(context.Background(), username)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (s *UserService) GetUsersByIDs(userIDs []uint) ([]model.User, error) {
	if len(userIDs) == 0 {
		return []model.User{}, nil
	}

	usersMap := make(map[uint]model.User, len(userIDs))
	missingIDs := make([]uint, 0, len(userIDs))

	// 检查缓存
	for _, id := range userIDs {
		cacheKey := fmt.Sprintf("user:%d", id)
		if data, ok := cache.UserCache.Get(cacheKey); ok {
			if jsonData, ok := data.([]byte); ok {
				var user model.User
				if err := json.Unmarshal(jsonData, &user); err == nil {
					usersMap[id] = user
					continue
				}
			}
		}
		missingIDs = append(missingIDs, id)
	}

	// 批量查询缺失的用户
	if len(missingIDs) > 0 {
		dbUsers, err := s.userRepo.FindByIDs(context.Background(), missingIDs)
		if err != nil {
			return nil, err
		}

		// 更新缓存并收集结果
		for _, user := range dbUsers {
			usersMap[user.ID] = *user
			if jsonData, err := json.Marshal(user); err == nil {
				cacheKey := fmt.Sprintf("user:%d", user.ID)
				cache.UserCache.Put(cacheKey, jsonData)
			}
		}
	}

	// 按原始 ID 顺序返回结果
	result := make([]model.User, 0, len(userIDs))
	for _, id := range userIDs {
		if user, ok := usersMap[id]; ok {
			result = append(result, user)
		}
	}

	return result, nil
}

func (s *UserService) GetUserRoles(userID uint) ([]string, error) {
	var userRoles []model.UserRole
	if err := s.db.Where("user_id = ?", userID).Find(&userRoles).Error; err != nil {
		return nil, err
	}
	roleNames := make([]string, 0, len(userRoles))
	for _, ur := range userRoles {
		roleNames = append(roleNames, ur.Role)
	}
	return roleNames, nil
}

func (s *UserService) GetSystemUser() *model.User {
	var systemUser model.User
	if err := s.db.Where("type = ?", "system").First(&systemUser).Error; err != nil {
		return nil
	}
	return &systemUser
}

func (s *UserService) GetSystemUserID() uint {
	systemUser := s.GetSystemUser()
	if systemUser != nil {
		return systemUser.ID
	}
	return 0
}

// GetDefaultAIAssistant 获取或创建默认 AI 助手用户（用于单聊/通用场景）
func (s *UserService) GetDefaultAIAssistant() (*model.User, error) {
	var existingBot model.Bot
	err := s.db.Where("type = ? AND group_id IS NULL", model.BotTypeAssistant).First(&existingBot).Error
	if err == nil && existingBot.VirtualUserID != nil {
		var user model.User
		if err := s.db.First(&user, *existingBot.VirtualUserID).Error; err == nil {
			return &user, nil
		}
	}

	var existingUser model.User
	if err := s.db.Where("username = ?", "bot_ai_assistant").First(&existingUser).Error; err == nil {
		bot := model.Bot{
			Name:          "AI助手",
			Description:   "通用 AI 助手",
			Type:          model.BotTypeAssistant,
			IsActive:      true,
			VirtualUserID: &existingUser.ID,
		}
		if createErr := s.db.Where("type = ? AND virtual_user_id = ?", model.BotTypeAssistant, existingUser.ID).FirstOrCreate(&bot).Error; createErr != nil {
			logger.WithModule("GetDefaultAIAssistant").Error("创建 Bot 记录失败", "error", createErr)
		}
		return &existingUser, nil
	}

	aiUser := model.User{
		Username: "bot_ai_assistant",
		Nickname: "AI助手",
		Type:     "bot",
		Status:   "online",
	}
	if err := s.db.Create(&aiUser).Error; err != nil {
		return nil, fmt.Errorf("创建默认 AI 助手用户失败: %w", err)
	}

	bot := model.Bot{
		Name:          "AI助手",
		Description:   "通用 AI 助手",
		Type:          model.BotTypeAssistant,
		IsActive:      true,
		VirtualUserID: &aiUser.ID,
	}
	if err := s.db.Create(&bot).Error; err != nil {
		logger.WithModule("GetDefaultAIAssistant").Error("创建 Bot 记录失败", "error", err)
	}

	return &aiUser, nil
}

// EnsureGroupAIAssistant 为指定群创建或获取 AI 助手用户记录
func (s *UserService) EnsureGroupAIAssistant(groupID uint, assistantName string) (*model.User, error) {
	// 查找已存在的群聊 AI 助手
	var existingBot model.Bot
	err := s.db.Where("group_id = ? AND type = ?", groupID, model.BotTypeGroupAssistant).First(&existingBot).Error
	if err == nil && existingBot.VirtualUserID != nil {
		var user model.User
		if err := s.db.First(&user, *existingBot.VirtualUserID).Error; err == nil {
			return &user, nil
		}
	}

	// 创建新的 AI 助手用户
	username := fmt.Sprintf("bot_group_assistant_%d", groupID)
	if assistantName == "" {
		assistantName = "AI助手"
	}
	aiUser := model.User{
		Username: username,
		Nickname: assistantName,
		Type:     "bot",
		Status:   "online",
	}
	if err := s.db.Create(&aiUser).Error; err != nil {
		return nil, fmt.Errorf("创建 AI 助手用户失败: %w", err)
	}

	// 创建关联 Bot 记录
	bot := model.Bot{
		Name:          assistantName,
		Description:   fmt.Sprintf("群 %d 的 AI 助手", groupID),
		Type:          model.BotTypeGroupAssistant,
		IsActive:      true,
		GroupID:       &groupID,
		VirtualUserID: &aiUser.ID,
	}
	if err := s.db.Create(&bot).Error; err != nil {
		logger.WithModule("EnsureGroupAIAssistant").Error("创建 Bot 记录失败", "error", err)
	}

	// 确保 AI 助手加入群的 conversation_members
	var conv model.Conversation
	s.db.Where("type = ? AND id IN (SELECT conversation_id FROM `groups` WHERE id = ?)", "group", groupID).First(&conv)
	if conv.ID > 0 {
		var member model.ConversationMember
		if err := s.db.Where("conversation_id = ? AND user_id = ?", conv.ID, aiUser.ID).First(&member).Error; err != nil {
			member = model.ConversationMember{
				ConversationID: conv.ID,
				UserID:         aiUser.ID,
				Role:           "member",
				JoinedAt:       time.Now(),
			}
			s.db.Create(&member)
			logger.WithModule("EnsureGroupAIAssistant").Info("AI 助手已加入群", "aiUserID", aiUser.ID, "groupID", groupID)
		}
	}

	logger.WithModule("EnsureGroupAIAssistant").Info("为群创建 AI 助手用户", "groupID", groupID, "aiUserID", aiUser.ID, "name", assistantName)
	return &aiUser, nil
}
