package model

import "time"

// UserSetting 用户个人设置（通用 key-value）
// 轻量偏好类配置：quick_replies / input_preferences / theme_preferences 等
// 强类型/有外键关系的领域数据仍用专门表（如 ai_configs），不塞这里
type UserSetting struct {
	ID           uint      `json:"id" gorm:"primarykey"`
	UserID       uint      `json:"user_id" gorm:"not null;uniqueIndex:idx_user_setting_key,priority:1"`
	SettingKey   string    `json:"setting_key" gorm:"size:100;not null;uniqueIndex:idx_user_setting_key,priority:2"`
	SettingValue string    `json:"setting_value" gorm:"type:text"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (UserSetting) TableName() string {
	return "user_settings"
}
