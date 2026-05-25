package model

import (
	"time"

	"gorm.io/gorm"
)

type AlertRule struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	Name          string    `json:"name" gorm:"not null"`
	Metric        string    `json:"metric" gorm:"not null"` // cpu, memory, disk, network
	Condition     string    `json:"condition" gorm:"not null"` // gt, lt, eq
	Threshold     float64   `json:"threshold" gorm:"not null"`
	Duration      int       `json:"duration" gorm:"not null"` // 持续时间（秒）
	NotifyMethods string    `json:"notifyMethods" gorm:"type:text"` // JSON array
	NotifyTargets string    `json:"notifyTargets" gorm:"type:text"` // JSON array
	Enabled       bool      `json:"enabled" gorm:"default:true"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type AlertHistory struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	RuleID    uint           `json:"ruleId" gorm:"not null;index"`
	Metric    string         `json:"metric" gorm:"not null"`
	Value     float64        `json:"value" gorm:"not null"`
	Status    string         `json:"status" gorm:"not null"` // firing, resolved
	HandledAt *time.Time     `json:"handledAt"`
	HandlerID *uint          `json:"handlerId"`
	CreatedAt time.Time      `json:"createdAt"`
	AlertRule *AlertRule     `json:"alertRule" gorm:"foreignKey:RuleID"`
}

func (AlertRule) TableName() string {
	return "alert_rules"
}

func (AlertHistory) TableName() string {
	return "alert_history"
}

func AutoMigrateAlertTables(db *gorm.DB) error {
	return db.AutoMigrate(&AlertRule{}, &AlertHistory{})
}
