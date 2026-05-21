package provider

import (
	"context"
	"encoding/json"
	"fmt"
)

type CASConfig struct {
	ServerURL    string `json:"server_url"`
	ServiceURL   string `json:"service_url"`
	ValidateURL  string `json:"validate_url"`
	UseProxy     bool   `json:"use_proxy"`
}

type CASProvider struct {
	name     string
	enabled  bool
	priority int
	config   *CASConfig
}

func NewCASProvider(name string, enabled bool, priority int, configJSON string) (*CASProvider, error) {
	var config CASConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("解析CAS配置失败: %w", err)
	}

	return &CASProvider{
		name:     name,
		enabled:  enabled,
		priority: priority,
		config:   &config,
	}, nil
}

func (p *CASProvider) Name() string {
	return p.name
}

func (p *CASProvider) GetType() string {
	return "redirect"
}

func (p *CASProvider) IsEnabled() bool {
	return p.enabled
}

func (p *CASProvider) Priority() int {
	return p.priority
}

func (p *CASProvider) Authenticate(ctx context.Context, creds *Credentials) (*AuthResult, error) {
	return &AuthResult{
		Success: false,
		Message: "CAS认证需要通过CAS服务器，请使用GetLoginURL获取登录地址",
	}, nil
}

func (p *CASProvider) GetLoginURL() string {
	return fmt.Sprintf("%s/login?service=%s", p.config.ServerURL, p.config.ServiceURL)
}

func (p *CASProvider) ValidateTicket(ctx context.Context, ticket string) (*AuthResult, error) {
	return &AuthResult{
		Success: false,
		Message: "CAS票据验证需要安装cas库并实现完整流程",
	}, nil
}
