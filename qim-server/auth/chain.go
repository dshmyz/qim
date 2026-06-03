package auth

import (
	"context"
	"sort"

	"github.com/dshmyz/qim/qim-server/auth/provider"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

type AuthChain struct {
	directProviders   []provider.AuthProvider
	redirectProviders []provider.AuthProvider
}

func NewAuthChain() *AuthChain {
	return &AuthChain{
		directProviders:   make([]provider.AuthProvider, 0),
		redirectProviders: make([]provider.AuthProvider, 0),
	}
}

func (ac *AuthChain) RegisterProvider(p provider.AuthProvider) {
	if p.GetType() == "direct" {
		ac.directProviders = append(ac.directProviders, p)
	} else {
		ac.redirectProviders = append(ac.redirectProviders, p)
	}
}

func (ac *AuthChain) SortProviders() {
	sort.Slice(ac.directProviders, func(i, j int) bool {
		return ac.directProviders[i].Priority() < ac.directProviders[j].Priority()
	})
}

func (ac *AuthChain) AuthenticateDirect(ctx context.Context, creds *provider.Credentials) (*provider.AuthResult, string, error) {
	for _, p := range ac.directProviders {
		if !p.IsEnabled() {
			continue
		}

		result, err := p.Authenticate(ctx, creds)
		if err != nil {
			logger.WithModule("Auth").Warn("认证提供者执行异常", "provider", p.Name(), "error", err)
			continue
		}

		if result.Success {
			logger.WithModule("Auth").Info("认证成功", "provider", p.Name(), "username", creds.Username)
			return result, p.Name(), nil
		}

		logger.WithModule("Auth").Info("认证被拒绝", "provider", p.Name(), "username", creds.Username, "reason", result.Message)
	}

	logger.WithModule("Auth").Warn("所有认证提供者均失败", "username", creds.Username)
	return &provider.AuthResult{
		Success: false,
		Message: "用户名或密码错误",
	}, "", nil
}

func (ac *AuthChain) GetRedirectProviders() []provider.AuthProvider {
	result := make([]provider.AuthProvider, 0)
	for _, p := range ac.redirectProviders {
		if p.IsEnabled() {
			result = append(result, p)
		}
	}
	return result
}
