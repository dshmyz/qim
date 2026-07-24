package model

import (
	"time"

	"gorm.io/gorm"
)

// BotToken 外部 agent 调用 Bot API 的访问令牌。
// 明文 token 仅在签发时返回一次，库内只存 sha256 哈希（TokenHash）。
// 撤销通过软删除（DeletedAt）实现，即时生效。
type BotToken struct {
	ID         uint           `json:"id" gorm:"primarykey"`
	BotID      uint           `json:"bot_id" gorm:"not null;index"`
	TokenHash  string         `json:"-" gorm:"size:128;not null;uniqueIndex"` // sha256(明文 token)
	Name       string         `json:"name" gorm:"size:64"`                    // 用途标注，如 "claude-code"
	CreatedAt  time.Time      `json:"created_at"`
	LastUsedAt *time.Time     `json:"last_used_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"` // 软删除 = 撤销
	Bot        Bot            `json:"bot,omitempty" gorm:"foreignkey:BotID"`
}
