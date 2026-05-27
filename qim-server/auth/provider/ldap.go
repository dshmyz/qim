package provider

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"
)

type LDAPConfig struct {
	Server           string            `json:"server"`
	Port             int               `json:"port"`
	BindDN           string            `json:"bind_dn"`
	BindPassword     string            `json:"bind_password"`
	BaseDN           string            `json:"base_dn"`
	Filter           string            `json:"filter"`
	UseTLS           bool              `json:"use_tls"`
	AttributeMapping map[string]string `json:"attribute_mapping"`
}

// defaultLDAPAttributeMapping 默认的 LDAP 属性映射，将 LDAP 属性名映射为标准字段名
var defaultLDAPAttributeMapping = map[string]string{
	"username": "uid",
	"nickname": "cn",
	"email":    "mail",
	"phone":    "telephonenumber",
	"avatar":   "jpegphoto",
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

	if config.Port == 0 {
		config.Port = 389
	}

	if config.Filter == "" {
		config.Filter = "(uid=%s)"
	}

	if config.AttributeMapping == nil {
		config.AttributeMapping = defaultLDAPAttributeMapping
	} else {
		for k, v := range defaultLDAPAttributeMapping {
			if _, exists := config.AttributeMapping[k]; !exists {
				config.AttributeMapping[k] = v
			}
		}
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

	l, err := p.connect()
	if err != nil {
		return &AuthResult{
			Success: false,
			Message: fmt.Sprintf("连接LDAP服务器失败: %v", err),
		}, nil
	}
	defer l.Close()

	if p.config.BindDN != "" && p.config.BindPassword != "" {
		if err := l.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
			return &AuthResult{
				Success: false,
				Message: fmt.Sprintf("LDAP管理员绑定失败: %v", err),
			}, nil
		}
	}

	// 根据映射配置收集需要请求的 LDAP 属性
	ldapAttrs := []string{"dn"}
	for _, attr := range p.config.AttributeMapping {
		ldapAttrs = append(ldapAttrs, strings.ToLower(attr))
	}

	filter := fmt.Sprintf(p.config.Filter, ldap.EscapeFilter(creds.Username))
	searchRequest := ldap.NewSearchRequest(
		p.config.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter,
		ldapAttrs,
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		return &AuthResult{
			Success: false,
			Message: fmt.Sprintf("LDAP搜索失败: %v", err),
		}, nil
	}

	if len(sr.Entries) == 0 {
		return &AuthResult{
			Success: false,
			Message: "用户不存在",
		}, nil
	}

	if len(sr.Entries) > 1 {
		return &AuthResult{
			Success: false,
			Message: "找到多个用户记录",
		}, nil
	}

	userDN := sr.Entries[0].DN

	if err := l.Bind(userDN, creds.Password); err != nil {
		return &AuthResult{
			Success: false,
			Message: "用户名或密码错误",
		}, nil
	}

	// 先收集原始 LDAP 属性
	rawAttrs := make(map[string]string)
	for _, attr := range sr.Entries[0].Attributes {
		if len(attr.Values) > 0 {
			rawAttrs[strings.ToLower(attr.Name)] = attr.Values[0]
		}
	}

	// 按映射配置转换为标准字段
	userInfo := p.mapAttributes(rawAttrs)
	userInfo["username"] = creds.Username
	userInfo["dn"] = userDN

	return &AuthResult{
		Success:  true,
		UserID:   userDN,
		Message:  "认证成功",
		UserInfo: userInfo,
	}, nil
}

// mapAttributes 将 LDAP 原始属性按映射配置转换为标准字段
func (p *LDAPProvider) mapAttributes(rawAttrs map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for standardKey, ldapAttr := range p.config.AttributeMapping {
		if val, ok := rawAttrs[strings.ToLower(ldapAttr)]; ok && val != "" {
			result[standardKey] = val
		}
	}
	return result
}

func (p *LDAPProvider) TestConnection() error {
	l, err := p.connect()
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer l.Close()

	if p.config.BindDN != "" && p.config.BindPassword != "" {
		if err := l.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
			return fmt.Errorf("绑定失败: %w", err)
		}
	}

	return nil
}

func (p *LDAPProvider) connect() (*ldap.Conn, error) {
	addr := fmt.Sprintf("%s:%d", p.config.Server, p.config.Port)

	if p.config.UseTLS {
		return ldap.DialTLS("tcp", addr, &tls.Config{
			InsecureSkipVerify: true,
		})
	}

	return ldap.Dial("tcp", addr)
}
