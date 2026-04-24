package ai

import (
	"fmt"
	"log"
	"net/http"
)

// TencentProvider 腾讯混元大模型提供商
type TencentProvider struct {
	*BaseProvider
	config ProviderConfig
}

// NewTencentProvider 创建腾讯提供商
func NewTencentProvider(config ProviderConfig) *TencentProvider {
	return &TencentProvider{
		BaseProvider: NewBaseProvider(),
		config:       config,
	}
}

func (p *TencentProvider) Name() string {
	return "tencent"
}

func (p *TencentProvider) IsConfigured() bool {
	return p.config.IsDualKeySet()
}

func (p *TencentProvider) Chat(messages []Message) (string, error) {
	if !p.IsConfigured() {
		return "", fmt.Errorf("Tencent Secret ID or Secret Key is not configured")
	}

	log.Printf("[Tencent] Making request with model: %s", p.config.Model)

	reqBody := map[string]interface{}{
		"Model":       p.config.Model,
		"Messages":    messages,
		"Temperature": p.config.ExtraParams["temperature"],
		"MaxTokens":   p.config.ExtraParams["max_tokens"],
	}

	resp, err := p.ExecuteWithRetry(func() (*http.Request, error) {
		req, _, err := CreateJSONRequest(
			"POST",
			p.config.BaseURL+"/v1/chat/completions",
			p.config.APIKey,
			reqBody,
			map[string]string{
				"X-TC-Action":  "ChatCompletions",
				"X-TC-Version": "2023-09-01",
			},
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

	if choices, ok := response["Choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["Message"].(map[string]interface{}); ok {
				if content, ok := message["Content"].(string); ok {
					return content, nil
				}
			}
		}
	}

	return "", fmt.Errorf("invalid Tencent API response")
}

func (p *TencentProvider) ChatStream(messages []Message, onChunk func(chunk StreamChunk) error) error {
	return fmt.Errorf("Tencent provider does not support streaming")
}
