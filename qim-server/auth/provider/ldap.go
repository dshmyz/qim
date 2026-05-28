package provider

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"

	"qim-server/pkg/logger"

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
	log := logger.WithModule("LDAP")

	if creds.Username == "" || creds.Password == "" {
		log.Warn("认证失败: 用户名或密码为空")
		return &AuthResult{
			Success: false,
			Message: "用户名和密码不能为空",
		}, nil
	}

	addr := fmt.Sprintf("%s:%d", p.config.Server, p.config.Port)
	log.Info("开始LDAP认证", "username", creds.Username, "server", addr, "tls", p.config.UseTLS)

	l, err := p.connect()
	if err != nil {
		log.Error("连接LDAP服务器失败", "server", addr, "error", err)
		return &AuthResult{
			Success: false,
			Message: fmt.Sprintf("连接LDAP服务器失败: %v", err),
		}, nil
	}
	defer l.Close()

	log.Info("LDAP连接成功", "server", addr)

	if p.config.BindDN != "" && p.config.BindPassword != "" {
		if err := l.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
			log.Error("LDAP管理员绑定失败", "bindDN", p.config.BindDN, "error", err)
			return &AuthResult{
				Success: false,
				Message: fmt.Sprintf("LDAP管理员绑定失败: %v", err),
			}, nil
		}
		log.Info("LDAP管理员绑定成功", "bindDN", p.config.BindDN)
	}

	// 根据映射配置收集需要请求的 LDAP 属性
	ldapAttrs := []string{"dn"}
	for _, attr := range p.config.AttributeMapping {
		ldapAttrs = append(ldapAttrs, strings.ToLower(attr))
	}

	filter := fmt.Sprintf(p.config.Filter, ldap.EscapeFilter(creds.Username))
	log.Info("LDAP搜索用户", "filter", filter, "baseDN", p.config.BaseDN)

	searchRequest := ldap.NewSearchRequest(
		p.config.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter,
		ldapAttrs,
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		log.Error("LDAP搜索失败", "filter", filter, "error", err)
		return &AuthResult{
			Success: false,
			Message: fmt.Sprintf("LDAP搜索失败: %v", err),
		}, nil
	}

	if len(sr.Entries) == 0 {
		log.Warn("LDAP搜索结果为空", "username", creds.Username, "filter", filter, "baseDN", p.config.BaseDN)
		return &AuthResult{
			Success: false,
			Message: "用户不存在",
		}, nil
	}

	if len(sr.Entries) > 1 {
		log.Warn("LDAP搜索到多个用户", "username", creds.Username, "count", len(sr.Entries))
		return &AuthResult{
			Success: false,
			Message: "找到多个用户记录",
		}, nil
	}

	userDN := sr.Entries[0].DN
	log.Info("找到LDAP用户", "username", creds.Username, "dn", userDN)

	if err := l.Bind(userDN, creds.Password); err != nil {
		log.Warn("LDAP用户绑定失败(密码错误)", "username", creds.Username, "dn", userDN, "error", err)
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

	log.Info("LDAP认证成功", "username", creds.Username, "dn", userDN)
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
	log := logger.WithModule("LDAP")

	if p.config.UseTLS {
		log.Info("使用TLS连接LDAP", "addr", addr)
		conn, err := ldap.DialTLS("tcp", addr, &tls.Config{
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Error("TLS连接LDAP失败", "addr", addr, "error", err)
			return nil, fmt.Errorf("TLS连接 %s 失败: %w", addr, err)
		}
		return conn, nil
	}

	log.Info("连接LDAP", "addr", addr)
	conn, err := ldap.Dial("tcp", addr)
	if err != nil {
		log.Error("连接LDAP失败", "addr", addr, "error", err)
		return nil, fmt.Errorf("连接 %s 失败: %w", addr, err)
	}
	return conn, nil
}
