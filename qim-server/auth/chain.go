package auth

import (
	"context"
	"sort"

	"qim-server/auth/provider"
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
			continue
		}

		if result.Success {
			return result, p.Name(), nil
		}
	}

	return &provider.AuthResult{
		Success: false,
		Message: "所有认证方式均失败",
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
