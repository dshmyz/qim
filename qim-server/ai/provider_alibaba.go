package ai

import (
	"fmt"
	"log"
	"net/http"
)

// AlibabaProvider 阿里通义千问提供商
type AlibabaProvider struct {
	*BaseProvider
	config ProviderConfig
}

// NewAlibabaProvider 创建阿里提供商
func NewAlibabaProvider(config ProviderConfig) *AlibabaProvider {
	return &AlibabaProvider{
		BaseProvider: NewBaseProvider(),
		config:       config,
	}
}

func (p *AlibabaProvider) Name() string {
	return "alibaba"
}

func (p *AlibabaProvider) IsConfigured() bool {
	return p.config.IsSet()
}

func (p *AlibabaProvider) Chat(messages []Message) (string, error) {
	if !p.IsConfigured() {
		return "", fmt.Errorf("Alibaba API key is not configured")
	}

	log.Printf("[Alibaba] Making request with model: %s", p.config.Model)

	reqBody := map[string]interface{}{
		"model": p.config.Model,
		"input": map[string]interface{}{
			"messages": messages,
		},
		"parameters": map[string]interface{}{
			"temperature": p.config.ExtraParams["temperature"],
			"max_tokens":  p.config.ExtraParams["max_tokens"],
		},
	}

	resp, err := p.ExecuteWithRetry(func() (*http.Request, error) {
		req, _, err := CreateJSONRequest(
			"POST",
			p.config.BaseURL+"/chat/completions",
			p.config.APIKey,
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

	if output, ok := response["output"].(map[string]interface{}); ok {
		if text, ok := output["text"].(string); ok {
			return text, nil
		}
	}

	return "", fmt.Errorf("invalid Alibaba API response")
}

func (p *AlibabaProvider) ChatStream(messages []Message, onChunk func(chunk StreamChunk) error) error {
	if !p.IsConfigured() {
		return fmt.Errorf("Alibaba API key is not configured")
	}

	log.Printf("[Alibaba] Making streaming request with model: %s", p.config.Model)

	reqBody := map[string]interface{}{
		"model": p.config.Model,
		"input": map[string]interface{}{
			"messages": messages,
		},
		"parameters": map[string]interface{}{
			"temperature": p.config.ExtraParams["temperature"],
			"max_tokens":  p.config.ExtraParams["max_tokens"],
			"stream":      true,
		},
	}

	req, _, err := CreateJSONRequest(
		"POST",
		p.config.BaseURL+"/chat/completions",
		p.config.APIKey,
		reqBody,
		nil,
	)
	if err != nil {
		return err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return fmt.Errorf("Alibaba API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Alibaba API returned non-200 status: %d", resp.StatusCode)
	}

	return p.ReadJSONStream(resp.Body, func(chunk map[string]interface{}) error {
		if output, ok := chunk["output"].(map[string]interface{}); ok {
			if text, ok := output["text"].(string); ok && text != "" {
				return onChunk(StreamChunk{Content: text})
			}
		}
		return nil
	})
}
