package provider

import (
	"context"
	"encoding/json"
	"fmt"
)

type OAuthConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AuthURL      string `json:"auth_url"`
	TokenURL     string `json:"token_url"`
	UserInfoURL  string `json:"user_info_url"`
	RedirectURL  string `json:"redirect_url"`
	Scope        string `json:"scope"`
}

type OAuthProvider struct {
	name     string
	enabled  bool
	priority int
	config   *OAuthConfig
}

func NewOAuthProvider(name string, enabled bool, priority int, configJSON string) (*OAuthProvider, error) {
	var config OAuthConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("解析OAuth配置失败: %w", err)
	}

	return &OAuthProvider{
		name:     name,
		enabled:  enabled,
		priority: priority,
		config:   &config,
	}, nil
}

func (p *OAuthProvider) Name() string {
	return p.name
}

func (p *OAuthProvider) GetType() string {
	return "redirect"
}

func (p *OAuthProvider) IsEnabled() bool {
	return p.enabled
}

func (p *OAuthProvider) Priority() int {
	return p.priority
}

func (p *OAuthProvider) Authenticate(ctx context.Context, creds *Credentials) (*AuthResult, error) {
	return &AuthResult{
		Success: false,
		Message: "OAuth认证需要通过授权流程，请使用GetAuthURL获取授权地址",
	}, nil
}

func (p *OAuthProvider) GetAuthURL(state string) string {
	return fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		p.config.AuthURL,
		p.config.ClientID,
		p.config.RedirectURL,
		p.config.Scope,
		state,
	)
}

func (p *OAuthProvider) HandleCallback(ctx context.Context, code string) (*AuthResult, error) {
	return &AuthResult{
		Success: false,
		Message: "OAuth回调处理需要安装oauth2库并实现完整流程",
	}, nil
}
