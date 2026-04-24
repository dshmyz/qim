package ai

import (
	"fmt"
	"log"
	"net/http"
)

// AnthropicProvider Anthropic Claude 提供商
type AnthropicProvider struct {
	*BaseProvider
	config ProviderConfig
}

// NewAnthropicProvider 创建 Anthropic 提供商
func NewAnthropicProvider(config ProviderConfig) *AnthropicProvider {
	return &AnthropicProvider{
		BaseProvider: NewBaseProvider(),
		config:       config,
	}
}

func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

func (p *AnthropicProvider) IsConfigured() bool {
	return p.config.IsSet()
}

func (p *AnthropicProvider) Chat(messages []Message) (string, error) {
	if !p.IsConfigured() {
		return "", fmt.Errorf("Anthropic API key is not configured")
	}

	log.Printf("[Anthropic] Making request with model: %s", p.config.Model)

	reqBody := map[string]interface{}{
		"model":       p.config.Model,
		"messages":    messages,
		"max_tokens":  p.config.ExtraParams["max_tokens"],
		"temperature": p.config.ExtraParams["temperature"],
	}

	resp, err := p.ExecuteWithRetry(func() (*http.Request, error) {
		req, _, err := CreateJSONRequest(
			"POST",
			p.config.BaseURL+"/messages",
			"",
			reqBody,
			map[string]string{
				"x-api-key":           p.config.APIKey,
				"anthropic-version":   "2023-06-01",
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

	if content, ok := response["content"].([]interface{}); ok && len(content) > 0 {
		if contentItem, ok := content[0].(map[string]interface{}); ok {
			if text, ok := contentItem["text"].(string); ok {
				return text, nil
			}
		}
	}

	return "", fmt.Errorf("invalid Anthropic API response")
}

func (p *AnthropicProvider) ChatStream(messages []Message, onChunk func(chunk StreamChunk) error) error {
	return fmt.Errorf("Anthropic provider does not support streaming in this implementation")
}
