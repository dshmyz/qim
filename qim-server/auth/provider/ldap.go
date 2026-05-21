package provider

import (
	"context"
	"encoding/json"
	"fmt"
)

type LDAPConfig struct {
	Server   string `json:"server"`
	Port     int    `json:"port"`
	BindDN   string `json:"bind_dn"`
	BindPass string `json:"bind_pass"`
	BaseDN   string `json:"base_dn"`
	Filter   string `json:"filter"`
	UseTLS   bool   `json:"use_tls"`
}

type LDAPProvider struct {
	name     string
	enabled  bool
	priority int
	config   *LDAPConfig
}

func NewLDAPProvider(name string, enabled bool, priority int, configJSON string) (*LDAPProvider, error) {
	var config LDAPConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("解析LDAP配置失败: %w", err)
	}

	return &LDAPProvider{
		name:     name,
		enabled:  enabled,
		priority: priority,
		config:   &config,
	}, nil
}

func (p *LDAPProvider) Name() string {
	return p.name
}

func (p *LDAPProvider) GetType() string {
	return "direct"
}

func (p *LDAPProvider) IsEnabled() bool {
	return p.enabled
}

func (p *LDAPProvider) Priority() int {
	return p.priority
}

func (p *LDAPProvider) Authenticate(ctx context.Context, creds *Credentials) (*AuthResult, error) {
	if creds.Username == "" || creds.Password == "" {
		return &AuthResult{
			Success: false,
			Message: "用户名和密码不能为空",
		}, nil
	}

	return &AuthResult{
		Success: false,
		Message: "LDAP认证需要安装go-ldap库并实现连接逻辑",
	}, nil
}

func (p *LDAPProvider) TestConnection() error {
	return fmt.Errorf("LDAP连接测试需要安装go-ldap库")
}
