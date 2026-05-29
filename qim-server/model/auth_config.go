package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	AuthProviderProtocolLDAP  = "ldap"
	AuthProviderProtocolOAuth = "oauth"
	AuthProviderProtocolCAS   = "cas"
)

var ValidAuthProviderProtocols = map[string]bool{
	AuthProviderProtocolLDAP:  true,
	AuthProviderProtocolOAuth: true,
	AuthProviderProtocolCAS:   true,
}

type AuthProvider struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	Name        string         `json:"name" gorm:"uniqueIndex;size:50;not null"`
	Protocol    string         `json:"protocol" gorm:"size:20;not null;default:'ldap'"`
	Type        string         `json:"type" gorm:"size:20;not null"`
	Enabled     bool           `json:"enabled" gorm:"default:true"`
	Priority    int            `json:"priority" gorm:"default:100"`
	Config      string         `json:"config" gorm:"type:text"`
	DisplayName string         `json:"display_name" gorm:"size:100"`
	Icon        string         `json:"icon" gorm:"size:200"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

type OrgSyncConfig struct {
	ID             uint           `json:"id" gorm:"primarykey"`
	Name           string         `json:"name" gorm:"uniqueIndex;size:50;not null"`
	Enabled        bool           `json:"enabled" gorm:"default:true"`
	SyncType       string         `json:"sync_type" gorm:"size:20;not null"`
	Schedule       string         `json:"schedule" gorm:"size:100"`
	Config         string         `json:"config" gorm:"type:text"`
	LastSyncAt     *time.Time     `json:"last_sync_at"`
	LastSyncStatus string         `json:"last_sync_status" gorm:"size:20"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

type OrgSyncLog struct {
	ID           uint           `json:"id" gorm:"primarykey"`
	ConfigID     uint           `json:"config_id" gorm:"not null;index"`
	Status       string         `json:"status" gorm:"size:20;not null"`
	StartedAt    time.Time      `json:"started_at" gorm:"not null"`
	FinishedAt   *time.Time     `json:"finished_at"`
	Stats        string         `json:"stats" gorm:"type:text"`
	ErrorMessage string         `json:"error_message" gorm:"type:text"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}
