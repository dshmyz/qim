package model

import (
	"time"

	"gorm.io/gorm"
)

type CrashLog struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       *uint     `json:"userId"`
	Platform     string    `json:"platform" gorm:"not null"`
	AppVersion   string    `json:"appVersion" gorm:"not null"`
	DeviceModel  string    `json:"deviceModel"`
	OSVersion    string    `json:"osVersion"`
	ErrorStack   string    `json:"errorStack" gorm:"type:text"`
	ErrorMessage string    `json:"errorMessage" gorm:"type:text"`
	Extra        string    `json:"extra" gorm:"type:text"`
	CreatedAt    time.Time `json:"createdAt"`
}

type UserFeedback struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    *uint          `json:"userId"`
	Type      string         `json:"type" gorm:"not null"`
	Content   string         `json:"content" gorm:"type:text;not null"`
	Status    string         `json:"status" gorm:"default:pending"`
	Priority  string         `json:"priority" gorm:"default:normal"`
	Screenshot string        `json:"screenshot"`
	Reply     string         `json:"reply" gorm:"type:text"`
	HandlerID *uint          `json:"handlerId"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (CrashLog) TableName() string {
	return "crash_logs"
}

func (UserFeedback) TableName() string {
	return "user_feedbacks"
}

func AutoMigrateClientTables(db *gorm.DB) error {
	return db.AutoMigrate(&CrashLog{}, &UserFeedback{})
}
