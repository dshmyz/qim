package ai

import (
	"encoding/json"
	"fmt"
	"net/http"

	"qim-server/pkg/logger"
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

func (p *AlibabaProvider) WithModel(model string) Provider {
	newConfig := p.config
	newConfig.Model = model
	return &AlibabaProvider{
		BaseProvider: p.BaseProvider,
		config:       newConfig,
	}
}

func (p *AlibabaProvider) ChatWithTools(messages []Message, tools []ToolDef) (*ChatResponse, error) {
	return nil, fmt.Errorf("Alibaba provider does not support native function calling, use prompt engineering instead")
}

func (p *AlibabaProvider) Chat(messages []Message) (string, error) {
	if !p.IsConfigured() {
		return "", fmt.Errorf("Alibaba API key is not configured")
	}

	logger.WithModule("Alibaba").Debug("Making request", "model", p.config.Model)

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

	logger.WithModule("Alibaba").Debug("Making streaming request", "model", p.config.Model)

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

// Embedding 将文本转换为向量（使用阿里 OpenAI 兼容的 /v1/embeddings 接口）
func (p *AlibabaProvider) Embedding(text string) ([]float32, error) {
	if !p.IsConfigured() {
		return nil, fmt.Errorf("Alibaba API key is not configured")
	}

	type embeddingRequest struct {
		Model string `json:"model"`
		Input string `json:"input"`
	}

	type embeddingData struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	}

	type embeddingResponse struct {
		Data  []embeddingData `json:"data"`
		Model string          `json:"model"`
	}

	// 使用 embedding 专用模型（如果配置了的话），否则使用当前模型
	embeddingModel := p.config.Model
	if p.config.ExtraParams["embedding_model"] != nil {
		if m, ok := p.config.ExtraParams["embedding_model"].(string); ok && m != "" {
			embeddingModel = m
		}
	}

	reqBody := embeddingRequest{
		Model: embeddingModel,
		Input: text,
	}

	resp, err := p.ExecuteWithRetry(func() (*http.Request, error) {
		req, _, err := CreateJSONRequest(
			"POST",
			p.config.BaseURL+"/embeddings",
			p.config.APIKey,
			reqBody,
			nil,
		)
		return req, err
	})
	if err != nil {
		return nil, fmt.Errorf("Alibaba embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	var response embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Alibaba embedding response: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in Alibaba response")
	}

	logger.WithModule("Alibaba").Info("Embedding completed", "model", response.Model, "dimension", len(response.Data[0].Embedding))

	return response.Data[0].Embedding, nil
}
