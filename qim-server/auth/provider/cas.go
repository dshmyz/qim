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
	ServerURL   string `json:"server_url"`
	ServiceURL  string `json:"service_url"`
	ValidateURL string `json:"validate_url"`
	UseProxy    bool   `json:"use_proxy"`
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

	userInfo := make(map[string]interface{})
	userInfo["username"] = casResponse.AuthenticationSuccess.User
	userInfo["attributes"] = casResponse.AuthenticationSuccess.Attributes

	return &AuthResult{
		Success:  true,
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

	userInfo := make(map[string]interface{})
	userInfo["username"] = casResponse.AuthenticationSuccess.User
	userInfo["attributes"] = casResponse.AuthenticationSuccess.Attributes
	userInfo["proxy_granting_ticket"] = casResponse.AuthenticationSuccess.ProxyGrantingTicket

	return &AuthResult{
		Success:  true,
		Message:  "代理认证成功",
		UserInfo: userInfo,
	}, nil
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
	User                string            `xml:"user"`
	Attributes          map[string]string `xml:"attributes"`
	ProxyGrantingTicket string            `xml:"proxyGrantingTicket"`
}

type CASAuthenticationFailure struct {
	Code        string `xml:"code,attr"`
	Description string `xml:",chardata"`
}
