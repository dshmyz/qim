package service

import (
	"github.com/dshmyz/qim/qim-server/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserSettingService 用户个人设置（通用 key-value）
// 适合轻量偏好类配置（quick_replies、theme_preferences 等）。
// 强类型/有外键关系的领域数据用专门表，不塞这里。
type UserSettingService struct {
	db *gorm.DB
}

func NewUserSettingService(db *gorm.DB) *UserSettingService {
	return &UserSettingService{db: db}
}

// GetOrDefault 读取某项设置；不存在时返回 defaultValue（不写入 DB）
func (s *UserSettingService) GetOrDefault(userID uint, key string, defaultValue string) (string, error) {
	var setting model.UserSetting
	err := s.db.Where("user_id = ? AND setting_key = ?", userID, key).First(&setting).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return defaultValue, nil
		}
		return "", err
	}
	return setting.SettingValue, nil
}

// Upsert 新增或更新某项设置（整存整取）
func (s *UserSettingService) Upsert(userID uint, key string, value string) error {
	setting := model.UserSetting{
		UserID:       userID,
		SettingKey:   key,
		SettingValue: value,
	}
	// OnConflict 走唯一索引 (user_id, setting_key)，命中则更新 setting_value
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "setting_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"setting_value", "updated_at"}),
	}).Create(&setting).Error
}

// Delete 删除某项设置（仅限本人的）
func (s *UserSettingService) Delete(userID uint, key string) error {
	return s.db.Where("user_id = ? AND setting_key = ?", userID, key).
		Delete(&model.UserSetting{}).Error
}
