package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"qim-server/pkg/logger"
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

func (p *AnthropicProvider) WithModel(model string) Provider {
	newConfig := p.config
	newConfig.Model = model
	return &AnthropicProvider{
		BaseProvider: p.BaseProvider,
		config:       newConfig,
	}
}

func (p *AnthropicProvider) Chat(messages []Message) (string, error) {
	if !p.IsConfigured() {
		return "", fmt.Errorf("Anthropic API key is not configured")
	}

	logger.WithModule("Anthropic").Debug("Making request", "model", p.config.Model)

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
				"x-api-key":         p.config.APIKey,
				"anthropic-version": "2023-06-01",
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

// Embedding 将文本转换为向量（Anthropic 目前不直接支持 Embedding，使用 OpenAI 兼容方式）
func (p *AnthropicProvider) Embedding(text string) ([]float32, error) {
	return nil, fmt.Errorf("Anthropic provider does not support Embedding API")
}

// ChatWithTools 带 function calling 的聊天
func (p *AnthropicProvider) ChatWithTools(messages []Message, tools []ToolDef) (*ChatResponse, error) {
	if !p.IsConfigured() {
		return nil, fmt.Errorf("Anthropic provider not configured")
	}

	logger.WithModule("Anthropic").Debug("Making ChatWithTools request", "model", p.config.Model, "tools", len(tools))

	anthropicMessages := make([]map[string]interface{}, len(messages))
	for i, m := range messages {
		anthropicMessages[i] = map[string]interface{}{
			"role":    m.Role,
			"content": m.Content,
		}
	}

	req := map[string]interface{}{
		"model":      p.config.Model,
		"messages":   anthropicMessages,
		"max_tokens": 4096,
	}

	if len(tools) > 0 {
		anthropicTools := make([]map[string]interface{}, len(tools))
		for i, t := range tools {
			anthropicTools[i] = map[string]interface{}{
				"name":         t.Name,
				"description":  t.Description,
				"input_schema": t.Parameters,
			}
		}
		req["tools"] = anthropicTools
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	httpReq, err := http.NewRequest("POST", p.config.BaseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct {
			Type  string                 `json:"type"`
			Text  string                 `json:"text"`
			Name  string                 `json:"name"`
			Input map[string]interface{} `json:"input"`
			ID    string                 `json:"id"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	chatResp := &ChatResponse{}
	for _, c := range result.Content {
		if c.Type == "text" {
			chatResp.Content += c.Text
		} else if c.Type == "tool_use" {
			if chatResp.ToolCalls == nil {
				chatResp.ToolCalls = []ToolCall{}
			}
			chatResp.ToolCalls = append(chatResp.ToolCalls, ToolCall{
				ID:        c.ID,
				Name:      c.Name,
				Arguments: c.Input,
			})
		}
	}

	return chatResp, nil
}
