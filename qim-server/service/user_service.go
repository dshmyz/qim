package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"qim-server/cache"
	"qim-server/model"
	"qim-server/repository"

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

func (s *UserService) GetDB() *gorm.DB {
	return s.db
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
		log.Printf("failed to fetch updated user for cache: %v", err)
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

	return user, nil
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
	var users []model.User
	for _, id := range userIDs {
		user, err := s.userRepo.FindByID(context.Background(), id)
		if err != nil {
			continue
		}
		users = append(users, *user)
	}
	return users, nil
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
