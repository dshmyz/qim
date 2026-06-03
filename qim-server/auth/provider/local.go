package provider

import (
	"context"
	"errors"
	"strconv"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type LocalProvider struct {
	enabled  bool
	priority int
	db       *gorm.DB
}

func NewLocalProvider(enabled bool, priority int) *LocalProvider {
	return &LocalProvider{
		enabled:  enabled,
		priority: priority,
		db:       database.GetDB(),
	}
}

func (p *LocalProvider) Name() string {
	return "local"
}

func (p *LocalProvider) GetType() string {
	return "direct"
}

func (p *LocalProvider) IsEnabled() bool {
	return p.enabled
}

func (p *LocalProvider) Priority() int {
	return p.priority
}

func (p *LocalProvider) Authenticate(ctx context.Context, creds *Credentials) (*AuthResult, error) {
	if creds.Username == "" || creds.Password == "" {
		return &AuthResult{
			Success: false,
			Message: "用户名和密码不能为空",
		}, nil
	}

	var user model.User
	if err := p.db.Where("username = ?", creds.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &AuthResult{
				Success: false,
				Message: "用户不存在",
			}, nil
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(creds.Password)); err != nil {
		return &AuthResult{
			Success: false,
			Message: "密码错误",
		}, nil
	}

	if (user.Type == "bot_assistant" || user.Type == "bot_avatar") || user.Type == "system" || user.Type == "api" {
		return &AuthResult{
			Success: false,
			Message: "该账户类型不支持登录",
		}, nil
	}

	return &AuthResult{
		Success: true,
		UserID:  strconv.FormatUint(uint64(user.ID), 10),
		UserInfo: map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"nickname": user.Nickname,
			"email":    user.Email,
			"phone":    user.Phone,
			"avatar":   user.Avatar,
		},
		Message: "认证成功",
	}, nil
}
