package provider

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type CASConfig struct {
	ServerURL        string            `json:"server_url"`
	ServiceURL       string            `json:"service_url"`
	ValidateURL      string            `json:"validate_url"`
	UseProxy         bool              `json:"use_proxy"`
	AttributeMapping map[string]string `json:"attribute_mapping"`
}

// defaultCASAttributeMapping 默认的 CAS 属性映射
var defaultCASAttributeMapping = map[string]string{
	"username": "username",
	"nickname": "displayName",
	"email":    "mail",
	"phone":    "phone",
	"avatar":   "avatar",
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

	if config.ValidateURL == "" {
		config.ValidateURL = fmt.Sprintf("%s/serviceValidate", config.ServerURL)
	}

	if config.AttributeMapping == nil {
		config.AttributeMapping = defaultCASAttributeMapping
	} else {
		for k, v := range defaultCASAttributeMapping {
			if _, exists := config.AttributeMapping[k]; !exists {
				config.AttributeMapping[k] = v
			}
		}
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
	return fmt.Sprintf("%s/login?service=%s", p.config.ServerURL, url.QueryEscape(p.config.ServiceURL))
}

func (p *CASProvider) GetLogoutURL() string {
	return fmt.Sprintf("%s/logout", p.config.ServerURL)
}

func (p *CASProvider) ValidateTicket(ctx context.Context, ticket string) (*AuthResult, error) {
	validateURL := fmt.Sprintf("%s?service=%s&ticket=%s",
		p.config.ValidateURL,
		url.QueryEscape(p.config.ServiceURL),
		url.QueryEscape(ticket),
	)

	resp, err := http.Get(validateURL)
	if err != nil {
		return &AuthResult{
			Success: false,
			Message: fmt.Sprintf("CAS票据验证请求失败: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &AuthResult{
			Success: false,
			Message: fmt.Sprintf("读取CAS响应失败: %v", err),
		}, nil
	}

	var casResponse CASServiceResponse
	if err := xml.Unmarshal(body, &casResponse); err != nil {
		return &AuthResult{
			Success: false,
			Message: fmt.Sprintf("解析CAS响应失败: %v", err),
		}, nil
	}

	if casResponse.AuthenticationFailure != nil {
		return &AuthResult{
			Success: false,
			Message: fmt.Sprintf("CAS认证失败: %s", casResponse.AuthenticationFailure.Description),
		}, nil
	}

	if casResponse.AuthenticationSuccess == nil {
		return &AuthResult{
			Success: false,
			Message: "CAS响应格式错误",
		}, nil
	}

	casUser := casResponse.AuthenticationSuccess.User
	attrs := casResponse.AuthenticationSuccess.ParseAttributes()
	userInfo := p.mapAttributes(casUser, attrs)

	return &AuthResult{
		Success:  true,
		UserID:   casUser,
		Message:  "认证成功",
		UserInfo: userInfo,
	}, nil
}

func (p *CASProvider) ValidateProxyTicket(ctx context.Context, ticket string, targetService string) (*AuthResult, error) {
	if !p.config.UseProxy {
		return &AuthResult{
			Success: false,
			Message: "未启用代理模式",
		}, nil
	}

	validateURL := fmt.Sprintf("%s/proxyValidate?service=%s&ticket=%s&targetService=%s",
		p.config.ServerURL,
		url.QueryEscape(p.config.ServiceURL),
		url.QueryEscape(ticket),
		url.QueryEscape(targetService),
	)

	resp, err := http.Get(validateURL)
	if err != nil {
		return &AuthResult{
			Success: false,
			Message: fmt.Sprintf("CAS代理票据验证请求失败: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &AuthResult{
			Success: false,
			Message: fmt.Sprintf("读取CAS响应失败: %v", err),
		}, nil
	}

	var casResponse CASServiceResponse
	if err := xml.Unmarshal(body, &casResponse); err != nil {
		return &AuthResult{
			Success: false,
			Message: fmt.Sprintf("解析CAS响应失败: %v", err),
		}, nil
	}

	if casResponse.AuthenticationFailure != nil {
		return &AuthResult{
			Success: false,
			Message: fmt.Sprintf("CAS代理认证失败: %s", casResponse.AuthenticationFailure.Description),
		}, nil
	}

	if casResponse.AuthenticationSuccess == nil {
		return &AuthResult{
			Success: false,
			Message: "CAS响应格式错误",
		}, nil
	}

	casUser := casResponse.AuthenticationSuccess.User
	attrs := casResponse.AuthenticationSuccess.ParseAttributes()
	userInfo := p.mapAttributes(casUser, attrs)
	userInfo["proxy_granting_ticket"] = casResponse.AuthenticationSuccess.ProxyGrantingTicket

	return &AuthResult{
		Success:  true,
		UserID:   casUser,
		Message:  "代理认证成功",
		UserInfo: userInfo,
	}, nil
}

// mapAttributes 将 CAS 原始属性按映射配置转换为标准字段
func (p *CASProvider) mapAttributes(casUser string, casAttrs map[string]string) map[string]interface{} {
	result := make(map[string]interface{})

	for standardKey, sourceKey := range p.config.AttributeMapping {
		if standardKey == "username" && sourceKey == "username" {
			result["username"] = casUser
			continue
		}
		if val, ok := casAttrs[sourceKey]; ok && val != "" {
			result[standardKey] = val
		}
	}

	// 确保 username 始终有值
	if _, ok := result["username"]; !ok {
		result["username"] = casUser
	}

	return result
}

func (p *CASProvider) BuildServiceURL(callbackPath string) string {
	if strings.HasPrefix(callbackPath, "http") {
		return callbackPath
	}
	return fmt.Sprintf("%s%s", p.config.ServiceURL, callbackPath)
}

type CASServiceResponse struct {
	XMLName               xml.Name                  `xml:"http://www.yale.edu/tp/cas serviceResponse"`
	AuthenticationSuccess *CASAuthenticationSuccess `xml:"authenticationSuccess"`
	AuthenticationFailure *CASAuthenticationFailure `xml:"authenticationFailure"`
}

type CASAuthenticationSuccess struct {
	User                string `xml:"user"`
	ProxyGrantingTicket string `xml:"proxyGrantingTicket"`
	InnerXML            string `xml:",innerxml"`
}

func (s *CASAuthenticationSuccess) ParseAttributes() map[string]string {
	attrs := make(map[string]string)
	if s.InnerXML == "" {
		return attrs
	}

	// 包裹时声明 cas 命名空间，避免 <cas:user> 等带前缀元素导致解析失败
	wrapper := `<root xmlns:cas="http://www.yale.edu/tp/cas">` + s.InnerXML + `</root>`
	decoder := xml.NewDecoder(strings.NewReader(wrapper))
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			// 跳过 wrapper 根元素和 user/proxyGrantingTicket，对其余子元素 DecodeElement 提取值
			if t.Name.Local == "root" || t.Name.Local == "user" || t.Name.Local == "proxyGrantingTicket" {
				continue
			}
			// CAS 3.0 attributes 块本身跳过，其子元素会被单独处理
			if t.Name.Local == "attributes" {
				continue
			}
			// CAS 3.0 <cas:attribute name="displayName">张三</cas:attribute> 格式
			if t.Name.Local == "attribute" {
				var attrName string
				for _, a := range t.Attr {
					if a.Name.Local == "name" {
						attrName = a.Value
						break
					}
				}
				if attrName == "" {
					continue
				}
				var value string
				if err := decoder.DecodeElement(&value, &t); err == nil && value != "" {
					attrs[attrName] = value
				}
				continue
			}
			// 扁平格式：<displayName>张三</displayName>
			var value string
			if err := decoder.DecodeElement(&value, &t); err == nil && value != "" {
				attrs[t.Name.Local] = value
			}
		}
	}
	return attrs
}

type CASAuthenticationFailure struct {
	Code        string `xml:"code,attr"`
	Description string `xml:",chardata"`
}
