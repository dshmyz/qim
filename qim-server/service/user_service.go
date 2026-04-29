package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"qim-server/cache"
	"qim-server/database"
	"qim-server/model"
)

var ErrUserNotFound = errors.New("user not found")

type UserService struct{}

func NewUserService() *UserService {
	return &UserService{}
}

func (s *UserService) GetDB() interface{} {
	return database.GetDB()
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

	db := database.GetDB()

	var user model.User
	if err := db.First(&user, userID).Error; err != nil {
		return nil, ErrUserNotFound
	}

	if jsonData, err := json.Marshal(&user); err == nil {
		cache.UserCache.Put(cacheKey, jsonData)
	}

	return &user, nil
}

func (s *UserService) GetUserByUsername(username string) (*model.User, error) {
	db := database.GetDB()

	var user model.User
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, ErrUserNotFound
	}

	return &user, nil
}

func (s *UserService) UpdateUserStatus(userID uint, status string) error {
	db := database.GetDB()

	result := db.Model(&model.User{}).Where("id = ?", userID).Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	var user model.User
	if err := db.First(&user, userID).Error; err == nil {
		cacheKey := fmt.Sprintf("user:%d", userID)
		if jsonData, err := json.Marshal(&user); err == nil {
			cache.UserCache.Put(cacheKey, jsonData)
		}
	} else {
		log.Printf("failed to fetch updated user for cache: %v", err)
	}

	return nil
}

func (s *UserService) SearchUsers(keyword string, limit int) ([]model.User, error) {
	db := database.GetDB()

	if limit <= 0 {
		limit = 20
	}

	var users []model.User
	query := db.Where("nickname LIKE ? OR username LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (s *UserService) UpdateUser(userID uint, updates map[string]interface{}) (*model.User, error) {
	db := database.GetDB()

	var user model.User
	if err := db.First(&user, userID).Error; err != nil {
		return nil, ErrUserNotFound
	}

	if err := db.Model(&user).Updates(updates).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *UserService) CreateUser(username, password, nickname, avatar string) (*model.User, error) {
	db := database.GetDB()

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

	if err := db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) IsUsernameExists(username string) (bool, error) {
	db := database.GetDB()

	var count int64
	if err := db.Model(&model.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *UserService) GetUsersByIDs(userIDs []uint) ([]model.User, error) {
	db := database.GetDB()

	var users []model.User
	if err := db.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}
