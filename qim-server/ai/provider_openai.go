package ai

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// OpenAIProvider OpenAI 兼容的 API 提供商
type OpenAIProvider struct {
	*BaseProvider
	config ProviderConfig
}

// NewOpenAIProvider 创建 OpenAI 提供商
func NewOpenAIProvider(config ProviderConfig) *OpenAIProvider {
	return &OpenAIProvider{
		BaseProvider: NewBaseProvider(),
		config:       config,
	}
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) IsConfigured() bool {
	return p.config.IsSet()
}

func (p *OpenAIProvider) Chat(messages []Message) (string, error) {
	if !p.IsConfigured() {
		return "", fmt.Errorf("OpenAI API key is not configured")
	}

	log.Printf("[OpenAI] Making request with model: %s", p.config.Model)

	reqBody := ChatCompletionRequest{
		Model:       p.config.Model,
		Messages:    messages,
		MaxTokens:   p.config.ExtraParams["max_tokens"].(int),
		Temperature: p.config.ExtraParams["temperature"].(float64),
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

	var response ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode OpenAI response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in OpenAI response")
	}

	log.Printf("[OpenAI] Request completed, usage: prompt_tokens=%d, completion_tokens=%d, total_tokens=%d",
		response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)

	return response.Choices[0].Message.Content, nil
}

func (p *OpenAIProvider) ChatStream(messages []Message, onChunk func(chunk StreamChunk) error) error {
	if !p.IsConfigured() {
		return fmt.Errorf("OpenAI API key is not configured")
	}

	log.Printf("[OpenAI] Making streaming request with model: %s", p.config.Model)

	reqBody := struct {
		Model       string    `json:"model"`
		Messages    []Message `json:"messages"`
		MaxTokens   int       `json:"max_tokens,omitempty"`
		Temperature float64   `json:"temperature,omitempty"`
		Stream      bool      `json:"stream"`
	}{
		Model:       p.config.Model,
		Messages:    messages,
		MaxTokens:   p.config.ExtraParams["max_tokens"].(int),
		Temperature: p.config.ExtraParams["temperature"].(float64),
		Stream:      true,
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
		return fmt.Errorf("OpenAI API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OpenAI API returned non-200 status: %d", resp.StatusCode)
	}

	return p.ReadSSEStream(resp.Body, func(data string) error {
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			log.Printf("[OpenAI] Failed to unmarshal stream data: %v", err)
			return nil
		}

		if len(chunk.Choices) > 0 {
			sc := StreamChunk{
				Content: chunk.Choices[0].Delta.Content,
				Finish:  chunk.Choices[0].FinishReason,
			}
			if sc.Content != "" || sc.Finish != nil {
				log.Printf("[OpenAI] Sending chunk: %q, finish: %v", sc.Content, sc.Finish)
				return onChunk(sc)
			}
		}
		return nil
	})
}

// Embedding 将文本转换为向量（使用 OpenAI 兼容的 /v1/embeddings 接口）
func (p *OpenAIProvider) Embedding(text string) ([]float32, error) {
	if !p.IsConfigured() {
		return nil, fmt.Errorf("OpenAI API key is not configured")
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
		return nil, fmt.Errorf("OpenAI embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	var response embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode OpenAI embedding response: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in OpenAI response")
	}

	log.Printf("[OpenAI] Embedding completed, model=%s, prompt_tokens=%d", response.Model, response.Usage.PromptTokens)

	return response.Data[0].Embedding, nil
}
