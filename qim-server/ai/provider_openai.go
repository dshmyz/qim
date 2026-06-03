package ai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/dshmyz/qim/qim-server/pkg/logger"
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

func (p *OpenAIProvider) WithModel(model string) Provider {
	newConfig := p.config
	newConfig.Model = model
	return &OpenAIProvider{
		BaseProvider: p.BaseProvider,
		config:       newConfig,
	}
}

func (p *OpenAIProvider) Chat(messages []Message) (string, error) {
	if !p.IsConfigured() {
		return "", fmt.Errorf("OpenAI API key is not configured")
	}

	logger.WithModule("OpenAI").Debug("Making request", "model", p.config.Model)

	reqBody := ChatCompletionRequest{
		Model:    p.config.Model,
		Messages: messages,
		MaxTokens: func() int {
			if v, ok := p.config.ExtraParams["max_tokens"].(int); ok {
				return v
			}
			return 4096
		}(),
		Temperature: func() float64 {
			if v, ok := p.config.ExtraParams["temperature"].(float64); ok {
				return v
			}
			return 0.7
		}(),
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

	logger.WithModule("OpenAI").Info("Request completed",
		"prompt_tokens", response.Usage.PromptTokens,
		"completion_tokens", response.Usage.CompletionTokens,
		"total_tokens", response.Usage.TotalTokens)

	return response.Choices[0].Message.Content, nil
}

func (p *OpenAIProvider) ChatStream(messages []Message, onChunk func(chunk StreamChunk) error) error {
	if !p.IsConfigured() {
		return fmt.Errorf("OpenAI API key is not configured")
	}

	logger.WithModule("OpenAI").Debug("Making streaming request", "model", p.config.Model)

	reqBody := struct {
		Model       string    `json:"model"`
		Messages    []Message `json:"messages"`
		MaxTokens   int       `json:"max_tokens,omitempty"`
		Temperature float64   `json:"temperature,omitempty"`
		Stream      bool      `json:"stream"`
	}{
		Model:    p.config.Model,
		Messages: messages,
		MaxTokens: func() int {
			if v, ok := p.config.ExtraParams["max_tokens"].(int); ok {
				return v
			}
			return 4096
		}(),
		Temperature: func() float64 {
			if v, ok := p.config.ExtraParams["temperature"].(float64); ok {
				return v
			}
			return 0.7
		}(),
		Stream: true,
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
			logger.WithModule("OpenAI").Error("Failed to unmarshal stream data", "error", err)
			return nil
		}

		if len(chunk.Choices) > 0 {
			sc := StreamChunk{
				Content: chunk.Choices[0].Delta.Content,
				Finish:  chunk.Choices[0].FinishReason,
			}
			if sc.Content != "" || sc.Finish != nil {
				logger.WithModule("OpenAI").Debug("Sending chunk", "content", sc.Content, "finish", sc.Finish)
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

	logger.WithModule("OpenAI").Info("Embedding completed", "model", response.Model, "prompt_tokens", response.Usage.PromptTokens)

	return response.Data[0].Embedding, nil
}

func (p *OpenAIProvider) ChatWithTools(messages []Message, tools []ToolDef) (*ChatResponse, error) {
	if !p.IsConfigured() {
		return nil, fmt.Errorf("OpenAI API key is not configured")
	}

	logger.WithModule("OpenAI").Debug("Making request with tools", "model", p.config.Model, "tools_count", len(tools))

	reqBody := struct {
		Model       string    `json:"model"`
		Messages    []Message `json:"messages"`
		MaxTokens   int       `json:"max_tokens,omitempty"`
		Temperature float64   `json:"temperature,omitempty"`
		Tools       []struct {
			Type     string `json:"type"`
			Function struct {
				Name        string                 `json:"name"`
				Description string                 `json:"description"`
				Parameters  map[string]interface{} `json:"parameters"`
			} `json:"function"`
		} `json:"tools,omitempty"`
	}{
		Model:    p.config.Model,
		Messages: messages,
		MaxTokens: func() int {
			if v, ok := p.config.ExtraParams["max_tokens"].(int); ok {
				return v
			}
			return 4096
		}(),
		Temperature: func() float64 {
			if v, ok := p.config.ExtraParams["temperature"].(float64); ok {
				return v
			}
			return 0.7
		}(),
	}

	if len(tools) > 0 {
		reqBody.Tools = make([]struct {
			Type     string `json:"type"`
			Function struct {
				Name        string                 `json:"name"`
				Description string                 `json:"description"`
				Parameters  map[string]interface{} `json:"parameters"`
			} `json:"function"`
		}, len(tools))
		for i, t := range tools {
			reqBody.Tools[i].Type = "function"
			reqBody.Tools[i].Function.Name = t.Name
			reqBody.Tools[i].Function.Description = t.Description
			reqBody.Tools[i].Function.Parameters = t.Parameters
		}
	}

	// 调试日志：打印请求体
	reqBodyJSON, _ := json.Marshal(reqBody)
	if len(reqBodyJSON) > 5000 {
		logger.WithModule("OpenAI").Debug("Request body (truncated)", "body", string(reqBodyJSON[:5000]))
	} else {
		logger.WithModule("OpenAI").Debug("Request body", "body", string(reqBodyJSON))
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
		return nil, err
	}
	defer resp.Body.Close()

	// 调试：打印非 200 响应
	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		logger.WithModule("OpenAI").Error("Non-200 response", "status", resp.Status, "body", string(bodyBytes))
		return nil, fmt.Errorf("OpenAI API returned status %d", resp.StatusCode)
	}
	var response struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string          `json:"name"`
						Arguments json.RawMessage `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls,omitempty"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode OpenAI response: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in OpenAI response")
	}

	chatResp := &ChatResponse{
		Content: response.Choices[0].Message.Content,
	}

	for _, tc := range response.Choices[0].Message.ToolCalls {
		var args map[string]interface{}
		if err := json.Unmarshal(tc.Function.Arguments, &args); err != nil {
			// 某些模型返回的是 JSON 字符串而非 JSON 对象，需要二次解析
			var argStr string
			if strErr := json.Unmarshal(tc.Function.Arguments, &argStr); strErr == nil && argStr != "" {
				if err2 := json.Unmarshal([]byte(argStr), &args); err2 != nil {
					logger.WithModule("OpenAI").Error("Failed to unmarshal tool call arguments (both raw and string)", "error", err)
					args = make(map[string]interface{})
				}
			} else {
				logger.WithModule("OpenAI").Error("Failed to unmarshal tool call arguments", "error", err)
				args = make(map[string]interface{})
			}
		}
		chatResp.ToolCalls = append(chatResp.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}

	logger.WithModule("OpenAI").Info("Request completed",
		"prompt_tokens", response.Usage.PromptTokens,
		"completion_tokens", response.Usage.CompletionTokens,
		"total_tokens", response.Usage.TotalTokens,
		"tool_calls", len(chatResp.ToolCalls))

	return chatResp, nil
}
