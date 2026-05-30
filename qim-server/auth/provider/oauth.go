package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

type OAuthConfig struct {
	ClientID         string            `json:"client_id"`
	ClientSecret     string            `json:"client_secret"`
	AuthURL          string            `json:"auth_url"`
	TokenURL         string            `json:"token_url"`
	UserInfoURL      string            `json:"user_info_url"`
	RevokeURL        string            `json:"revoke_url"`
	RedirectURL      string            `json:"redirect_url"`
	Scope            string            `json:"scope"`
	AttributeMapping map[string]string `json:"attribute_mapping"`
}

// defaultOAuthAttributeMapping 默认的 OAuth 属性映射（兼容 Google/GitHub 等常见服务商）
var defaultOAuthAttributeMapping = map[string]string{
	"username": "login",
	"nickname": "name",
	"email":    "email",
	"phone":    "phone",
	"avatar":   "picture",
}

type OAuthProvider struct {
	name         string
	enabled      bool
	priority     int
	config       *OAuthConfig
	oauth2Config *oauth2.Config
}

func NewOAuthProvider(name string, enabled bool, priority int, configJSON string) (*OAuthProvider, error) {
	var config OAuthConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("解析OAuth配置失败: %w", err)
	}

	if config.AttributeMapping == nil {
		config.AttributeMapping = defaultOAuthAttributeMapping
	} else {
		for k, v := range defaultOAuthAttributeMapping {
			if _, exists := config.AttributeMapping[k]; !exists {
				config.AttributeMapping[k] = v
			}
		}
	}

	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Scopes:       []string{config.Scope},
		Endpoint: oauth2.Endpoint{
			AuthURL:  config.AuthURL,
			TokenURL: config.TokenURL,
		},
		RedirectURL: config.RedirectURL,
	}

	return &OAuthProvider{
		name:         name,
		enabled:      enabled,
		priority:     priority,
		config:       &config,
		oauth2Config: oauth2Config,
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

func (p *OAuthProvider) GetConfig() *OAuthConfig {
	return p.config
}

func (p *OAuthProvider) Authenticate(ctx context.Context, creds *Credentials) (*AuthResult, error) {
	return &AuthResult{
		Success: false,
		Message: "OAuth认证需要通过授权流程，请使用GetAuthURL获取授权地址",
	}, nil
}

func (p *OAuthProvider) GetAuthURL(state string) string {
	return p.oauth2Config.AuthCodeURL(state)
}

func (p *OAuthProvider) HandleCallback(ctx context.Context, code string) (*AuthResult, error) {
	token, err := p.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return &AuthResult{
			Success: false,
			Message: fmt.Sprintf("获取访问令牌失败: %v", err),
		}, nil
	}

	rawUserInfo, err := p.getUserInfo(ctx, token.AccessToken)
	if err != nil {
		return &AuthResult{
			Success: false,
			Message: fmt.Sprintf("获取用户信息失败: %v", err),
		}, nil
	}

	// 按映射配置转换为标准字段
	userInfo := p.mapAttributes(rawUserInfo)

	// 确定用户唯一标识：优先用映射后的 username，其次用 id/sub
	userID := getStringFromMap(userInfo, "username")
	if userID == "" {
		userID = getStringFromMap(rawUserInfo, "id", "sub")
	}

	return &AuthResult{
		Success:  true,
		UserID:   userID,
		Message:  "认证成功",
		UserInfo: userInfo,
	}, nil
}

// mapAttributes 将 OAuth 原始用户信息按映射配置转换为标准字段
func (p *OAuthProvider) mapAttributes(rawUserInfo map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for standardKey, sourceKey := range p.config.AttributeMapping {
		if val, ok := rawUserInfo[sourceKey]; ok && val != nil {
			result[standardKey] = val
		}
	}
	return result
}

func (p *OAuthProvider) getUserInfo(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", p.config.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取用户信息失败: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var userInfo map[string]interface{}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}

	return userInfo, nil
}

func (p *OAuthProvider) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	ts := p.oauth2Config.TokenSource(ctx, token)
	return ts.Token()
}

func (p *OAuthProvider) RevokeToken(ctx context.Context, token string) error {
	if p.config.RevokeURL == "" {
		return nil
	}

	form := url.Values{}
	form.Set("token", token)
	if p.config.ClientID != "" {
		form.Set("client_id", p.config.ClientID)
	}
	if p.config.ClientSecret != "" {
		form.Set("client_secret", p.config.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.RevokeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("创建撤销请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("撤销令牌请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("撤销令牌失败: HTTP %d", resp.StatusCode)
	}

	return nil
}

func (p *OAuthProvider) BuildRedirectURL(redirectURI string) string {
	u, _ := url.Parse(p.config.AuthURL)
	q := u.Query()
	q.Set("client_id", p.config.ClientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("scope", p.config.Scope)
	u.RawQuery = q.Encode()
	return u.String()
}

// getStringFromMap 从 map 中按多个候选 key 获取字符串值
func getStringFromMap(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := data[key]; ok {
			if s, ok := val.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}
