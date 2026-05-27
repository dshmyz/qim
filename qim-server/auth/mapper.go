package auth

import (
	"encoding/json"

	"qim-server/database"
	"qim-server/model"

	"gorm.io/gorm"
)

type UserMapper struct {
	db *gorm.DB
}

func NewUserMapper() *UserMapper {
	return &UserMapper{
		db: database.GetDB(),
	}
}

func (m *UserMapper) MapOrCreateUser(externalUserID string, providerName string, userInfo map[string]interface{}) (*model.User, error) {
	var mapping model.ExternalUserMapping
	err := m.db.Where("provider_name = ? AND external_user_id = ?", providerName, externalUserID).
		First(&mapping).Error

	if err == nil {
		var user model.User
		if err := m.db.First(&user, mapping.UserID).Error; err != nil {
			return nil, err
		}
		return &user, nil
	}

	username := m.getString(userInfo, "username", "uid", "cn")
	nickname := m.getString(userInfo, "nickname", "username", "cn")
	email := m.getString(userInfo, "email", "mail")
	phone := m.getString(userInfo, "phone", "telephonenumber", "mobile")
	avatar := m.getString(userInfo, "avatar", "jpegphoto", "photo")

	user := &model.User{
		Username: username,
		Nickname: nickname,
		Email:    email,
		Phone:    phone,
		Avatar:   avatar,
		Status:   "offline",
		Type:     "user",
	}

	if user.Username == "" {
		user.Username = externalUserID
	}
	if user.Nickname == "" {
		user.Nickname = user.Username
	}

	if err := m.db.Create(user).Error; err != nil {
		return nil, err
	}

	externalData, _ := json.Marshal(userInfo)
	mapping = model.ExternalUserMapping{
		UserID:           user.ID,
		ProviderName:     providerName,
		ExternalUserID:   externalUserID,
		ExternalUsername: user.Username,
		ExternalData:     string(externalData),
	}

	if err := m.db.Create(&mapping).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (m *UserMapper) getString(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := data[key]; ok {
			if str, ok := val.(string); ok && str != "" {
				return str
			}
		}
	}
	return ""
}
