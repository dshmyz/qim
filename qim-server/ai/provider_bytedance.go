package ai

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// BytedanceProvider 字节跳动豆包提供商
type BytedanceProvider struct {
	*BaseProvider
	config ProviderConfig
}

// NewBytedanceProvider 创建字节跳动提供商
func NewBytedanceProvider(config ProviderConfig) *BytedanceProvider {
	return &BytedanceProvider{
		BaseProvider: NewBaseProvider(),
		config:       config,
	}
}

func (p *BytedanceProvider) Name() string {
	return "bytedance"
}

func (p *BytedanceProvider) IsConfigured() bool {
	return p.config.IsSet()
}

func (p *BytedanceProvider) ChatWithTools(messages []Message, tools []ToolDef) (*ChatResponse, error) {
	return nil, fmt.Errorf("Bytedance provider does not support native function calling, use prompt engineering instead")
}

func (p *BytedanceProvider) Chat(messages []Message) (string, error) {
	if !p.IsConfigured() {
		return "", fmt.Errorf("Bytedance API key is not configured")
	}

	log.Printf("[Bytedance] Making request with model: %s", p.config.Model)

	reqBody := map[string]interface{}{
		"model":       p.config.Model,
		"messages":    messages,
		"temperature": p.config.ExtraParams["temperature"],
		"max_tokens":  p.config.ExtraParams["max_tokens"],
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

	if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content, nil
				}
			}
		}
	}

	return "", fmt.Errorf("invalid Bytedance API response")
}

func (p *BytedanceProvider) ChatStream(messages []Message, onChunk func(chunk StreamChunk) error) error {
	if !p.IsConfigured() {
		return fmt.Errorf("Bytedance API key is not configured")
	}

	log.Printf("[Bytedance] Making streaming request with model: %s", p.config.Model)

	reqBody := map[string]interface{}{
		"model":       p.config.Model,
		"messages":    messages,
		"temperature": p.config.ExtraParams["temperature"],
		"max_tokens":  p.config.ExtraParams["max_tokens"],
		"stream":      true,
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
		return fmt.Errorf("Bytedance API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Bytedance API returned non-200 status: %d", resp.StatusCode)
	}

	return p.ReadJSONStream(resp.Body, func(chunk map[string]interface{}) error {
		var streamChunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}

		chunkJSON, _ := json.Marshal(chunk)
		if err := json.Unmarshal(chunkJSON, &streamChunk); err != nil {
			return nil
		}

		if len(streamChunk.Choices) > 0 && streamChunk.Choices[0].Delta.Content != "" {
			return onChunk(StreamChunk{Content: streamChunk.Choices[0].Delta.Content})
		}
		return nil
	})
}

// Embedding 将文本转换为向量（使用字节跳动 OpenAI 兼容的 /v1/embeddings 接口）
func (p *BytedanceProvider) Embedding(text string) ([]float32, error) {
	if !p.IsConfigured() {
		return nil, fmt.Errorf("Bytedance API key is not configured")
	}

	type embeddingData struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	}

	type embeddingResponse struct {
		Data  []embeddingData `json:"data"`
		Model string          `json:"model"`
		Usage struct {
			PromptTokens int `json:"prompt_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}

	// 使用 embedding 专用模型（如果配置了的话），否则使用当前模型
	embeddingModel := p.config.Model
	if p.config.ExtraParams["embedding_model"] != nil {
		if m, ok := p.config.ExtraParams["embedding_model"].(string); ok && m != "" {
			embeddingModel = m
		}
	}

	reqBody := map[string]interface{}{
		"model": embeddingModel,
		"input": text,
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
		return nil, fmt.Errorf("Bytedance embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	var response embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Bytedance embedding response: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in Bytedance response")
	}

	log.Printf("[Bytedance] Embedding completed, model=%s, dimension=%d", response.Model, len(response.Data[0].Embedding))

	return response.Data[0].Embedding, nil
}
