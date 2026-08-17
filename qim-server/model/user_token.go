package model

import (
	"time"

	"gorm.io/gorm"
)

// UserToken 用户签发的长期访问令牌（供 qim CLI / qim-mcp 等外部工具以本人身份调用用户 API）。
// 明文 token 仅在签发时返回一次，库内只存 sha256 哈希（TokenHash）。
// 撤销通过软删除（DeletedAt）实现，即时生效。
type UserToken struct {
	ID         uint           `json:"id" gorm:"primarykey"`
	UserID     uint           `json:"user_id" gorm:"not null;index"`
	TokenHash  string         `json:"-" gorm:"size:128;not null;uniqueIndex"` // sha256(明文 token)
	Name       string         `json:"name" gorm:"size:64"`                    // 用途标注，如 "cli"、"qim-mcp"
	CreatedAt  time.Time      `json:"created_at"`
	LastUsedAt *time.Time     `json:"last_used_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"` // 软删除 = 撤销
}
