package ai

import (
	"encoding/json"
	"fmt"
	"net/http"

	"qim-server/pkg/logger"
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

func (p *TencentProvider) WithModel(model string) Provider {
	newConfig := p.config
	newConfig.Model = model
	return &TencentProvider{
		BaseProvider: p.BaseProvider,
		config:       newConfig,
	}
}

func (p *TencentProvider) ChatWithTools(messages []Message, tools []ToolDef) (*ChatResponse, error) {
	return nil, fmt.Errorf("Tencent provider does not support native function calling, use prompt engineering instead")
}

func (p *TencentProvider) Chat(messages []Message) (string, error) {
	if !p.IsConfigured() {
		return "", fmt.Errorf("Tencent Secret ID or Secret Key is not configured")
	}

	logger.WithModule("Tencent").Debug("Making request", "model", p.config.Model)

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

// Embedding 将文本转换为向量（使用腾讯 OpenAI 兼容的 /v1/embeddings 接口）
func (p *TencentProvider) Embedding(text string) ([]float32, error) {
	if !p.IsConfigured() {
		return nil, fmt.Errorf("Tencent Secret ID or Secret Key is not configured")
	}

	type embeddingData struct {
		Embedding []float32 `json:"Embedding"`
		Index     int       `json:"Index"`
	}

	type embeddingResponse struct {
		Response struct {
			Data  []embeddingData `json:"Data"`
			Model string          `json:"Model"`
			Usage struct {
				PromptTokens int `json:"PromptTokens"`
				TotalTokens  int `json:"TotalTokens"`
			} `json:"Usage"`
		} `json:"Response"`
	}

	// 使用 embedding 专用模型（如果配置了的话），否则使用当前模型
	embeddingModel := p.config.Model
	if p.config.ExtraParams["embedding_model"] != nil {
		if m, ok := p.config.ExtraParams["embedding_model"].(string); ok && m != "" {
			embeddingModel = m
		}
	}

	reqBody := map[string]interface{}{
		"Model": embeddingModel,
		"Input": map[string]interface{}{
			"Text": []string{text},
		},
	}

	resp, err := p.ExecuteWithRetry(func() (*http.Request, error) {
		req, _, err := CreateJSONRequest(
			"POST",
			p.config.BaseURL+"/",
			p.config.APIKey,
			reqBody,
			map[string]string{
				"X-TC-Action":  "GetEmbedding",
				"X-TC-Version": "2023-09-01",
			},
		)
		return req, err
	})
	if err != nil {
		return nil, fmt.Errorf("Tencent embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	var response embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Tencent embedding response: %w", err)
	}

	if len(response.Response.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in Tencent response")
	}

	logger.WithModule("Tencent").Info("Embedding completed", "model", response.Response.Model, "dimension", len(response.Response.Data[0].Embedding))

	return response.Response.Data[0].Embedding, nil
}
