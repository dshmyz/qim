package ai

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
)

// BaiduProvider 百度文心一言提供商
type BaiduProvider struct {
	*BaseProvider
	config ProviderConfig
}

// NewBaiduProvider 创建百度提供商
func NewBaiduProvider(config ProviderConfig) *BaiduProvider {
	return &BaiduProvider{
		BaseProvider: NewBaseProvider(),
		config:       config,
	}
}

func (p *BaiduProvider) Name() string {
	return "baidu"
}

func (p *BaiduProvider) IsConfigured() bool {
	return p.config.IsDualKeySet()
}

// getAccessToken 获取百度 access token
func (p *BaiduProvider) getAccessToken() (string, error) {
	params := url.Values{}
	params.Add("grant_type", "client_credentials")
	params.Add("client_id", p.config.APIKey)
	params.Add("client_secret", p.config.APISecret)

	url := p.config.BaseURL + "/oauth/2.0/token?" + params.Encode()

	resp, err := p.Client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get Baidu access token: %w", err)
	}
	defer resp.Body.Close()

	var response map[string]interface{}
	if err := parseJSONResponse(resp, &response); err != nil {
		return "", err
	}

	if accessToken, ok := response["access_token"].(string); ok {
		return accessToken, nil
	}

	return "", fmt.Errorf("failed to get Baidu access token")
}

func (p *BaiduProvider) Chat(messages []Message) (string, error) {
	if !p.IsConfigured() {
		return "", fmt.Errorf("Baidu API key or secret key is not configured")
	}

	token, err := p.getAccessToken()
	if err != nil {
		return "", err
	}

	log.Printf("[Baidu] Making request with model: %s", p.config.Model)

	reqBody := map[string]interface{}{
		"messages":    messages,
		"model":       p.config.Model,
		"temperature": p.config.ExtraParams["temperature"],
		"max_tokens":  p.config.ExtraParams["max_tokens"],
	}

	apiURL := p.config.BaseURL + "/rpc/2.0/ai_custom/v1/wenxinworkshop/chat/completions?access_token=" + token

	resp, err := p.ExecuteWithRetry(func() (*http.Request, error) {
		req, _, err := CreateJSONRequest(
			"POST",
			apiURL,
			"", // Baidu uses token in URL, not in header
			reqBody,
			nil,
		)
		return req, err
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var response map[string]interface{}
	if err := parseJSONResponse(resp, &response); err != nil {
		return "", err
	}

	if errMsg, ok := response["error_msg"].(string); ok {
		return "", fmt.Errorf("Baidu API error: %s", errMsg)
	}

	if result, ok := response["result"].(string); ok {
		return result, nil
	}

	return "", fmt.Errorf("invalid Baidu API response")
}

func (p *BaiduProvider) ChatStream(messages []Message, onChunk func(chunk StreamChunk) error) error {
	if !p.IsConfigured() {
		return fmt.Errorf("Baidu API key or secret key is not configured")
	}

	token, err := p.getAccessToken()
	if err != nil {
		return err
	}

	log.Printf("[Baidu] Making streaming request with model: %s", p.config.Model)

	reqBody := map[string]interface{}{
		"messages":    messages,
		"model":       p.config.Model,
		"temperature": p.config.ExtraParams["temperature"],
		"max_tokens":  p.config.ExtraParams["max_tokens"],
		"stream":      true,
	}

	apiURL := p.config.BaseURL + "/rpc/2.0/ai_custom/v1/wenxinworkshop/chat/completions?access_token=" + token

	req, _, err := CreateJSONRequest(
		"POST",
		apiURL,
		"",
		reqBody,
		nil,
	)
	if err != nil {
		return err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return fmt.Errorf("Baidu API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Baidu API returned non-200 status: %d", resp.StatusCode)
	}

	return p.ReadJSONStream(resp.Body, func(chunk map[string]interface{}) error {
		if result, ok := chunk["result"].(string); ok && result != "" {
			return onChunk(StreamChunk{Content: result})
		}
		if isEnd, ok := chunk["is_end"].(bool); ok && isEnd {
			finish := "stop"
			return onChunk(StreamChunk{Finish: &finish})
		}
		return nil
	})
}
