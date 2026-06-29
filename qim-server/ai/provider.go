package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"strings"
	"time"
)

// Provider 统一的 AI 提供商接口
type Provider interface {
	// Name 返回提供商名称
	Name() string

	// Chat 发送聊天请求并返回完整的回复
	Chat(messages []Message) (string, error)

	// ChatStream 发送聊天请求并以流式方式返回回复（返回 JSON 编码的 StreamChunk）
	ChatStream(messages []Message, onChunk func(chunk StreamChunk) error) error

	// ChatStreamWithContext 支持 context 取消的流式请求（用于超时控制）
	ChatStreamWithContext(ctx context.Context, messages []Message, onChunk func(chunk StreamChunk) error) error

	// Embedding 将文本转换为向量
	Embedding(text string) ([]float32, error)

	// ChatWithTools 带 function calling 的聊天
	ChatWithTools(messages []Message, tools []ToolDef) (*ChatResponse, error)

	// IsConfigured 检查提供商是否已正确配置
	IsConfigured() bool

	// WithModel 返回使用指定 model 的 Provider副本（共享 HTTP 连接池）
	WithModel(model string) Provider
}

// BaseProvider 提供所有提供商共用的基础功能
type BaseProvider struct {
	Client     *http.Client
	MaxRetries int
}

// NewBaseProvider 创建基础提供商
func NewBaseProvider() *BaseProvider {
	return &BaseProvider{
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
		MaxRetries: 2,
	}
}

// ChatStreamWithContext 默认实现：返回未实现错误
// 每个具体 Provider 应覆盖此方法以提供带 context 取消的流式支持
func (bp *BaseProvider) ChatStreamWithContext(ctx context.Context, messages []Message, onChunk func(chunk StreamChunk) error) error {
	return fmt.Errorf("ChatStreamWithContext not implemented")
}

// ExecuteWithRetry 执行 HTTP 请求并自动重试
func (bp *BaseProvider) ExecuteWithRetry(createRequest func() (*http.Request, error)) (*http.Response, error) {
	var resp *http.Response
	var err error
	var lastStatusCode int

	for attempt := 1; attempt <= bp.MaxRetries; attempt++ {
		req, reqErr := createRequest()
		if reqErr != nil {
			return nil, fmt.Errorf("failed to create request: %w", reqErr)
		}

		resp, err = bp.Client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			logger.WithModule("AI").Info("Request successful on attempt", "attempt", attempt)
			return resp, nil
		}

		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
			// 非 200 响应必须关闭 Body 防止资源泄漏
			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
			}
		}
		lastStatusCode = statusCode
		logger.WithModule("AI").Error("Request failed",
			"attempt", attempt,
			"maxRetries", bp.MaxRetries,
			"error", err,
			"status", statusCode,
		)

		if attempt < bp.MaxRetries {
			sleepDuration := time.Duration(attempt) * time.Second
			logger.WithModule("AI").Info("Retrying", "sleepDuration", sleepDuration)
			time.Sleep(sleepDuration)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("request failed after %d retries: %w", bp.MaxRetries, err)
	}

	// 读取最后一次非 200 响应的响应体
	if lastStatusCode != 0 && resp != nil {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		logger.WithModule("AI").Error("Last non-200 response body", "status", lastStatusCode, "body", string(bodyBytes))
	}

	return nil, fmt.Errorf("request failed after %d retries, last status: %d", bp.MaxRetries, lastStatusCode)
}

// ReadSSEStream 读取 SSE 格式的流并解析数据
func (bp *BaseProvider) ReadSSEStream(body io.ReadCloser, processData func(data string) error) error {
	defer body.Close()
	reader := bufio.NewScanner(body)

	for reader.Scan() {
		line := reader.Text()
		logger.WithModule("AI").Debug("Received SSE line", "line", line)

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			if strings.TrimSpace(data) == "[DONE]" {
				logger.WithModule("AI").Debug("Received stream end marker")
				break
			}

			if err := processData(data); err != nil {
				return err
			}
		}
	}

	return reader.Err()
}

// ReadJSONStream 读取 JSON 格式的流（每行一个 JSON 对象）
func (bp *BaseProvider) ReadJSONStream(body io.ReadCloser, processChunk func(chunk map[string]interface{}) error) error {
	defer body.Close()
	decoder := json.NewDecoder(body)

	for {
		var chunk map[string]interface{}
		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode stream chunk: %w", err)
		}

		if err := processChunk(chunk); err != nil {
			return err
		}
	}

	return nil
}

// parseJSONResponse 解析 JSON 响应
func parseJSONResponse(resp *http.Response, v interface{}) error {
	return json.NewDecoder(resp.Body).Decode(v)
}

// CreateJSONRequest 创建 JSON 请求的辅助函数
func CreateJSONRequest(method, url, apiKey string, body interface{}, headers map[string]string) (*http.Request, []byte, error) {
	reqJSON, err := json.Marshal(body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqJSON))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return req, reqJSON, nil
}

// CreateJSONRequestWithContext 创建带 context 的 JSON HTTP 请求（支持超时取消）
func CreateJSONRequestWithContext(ctx context.Context, method, url, apiKey string, body interface{}, headers map[string]string) (*http.Request, []byte, error) {
	reqJSON, err := json.Marshal(body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(reqJSON))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return req, reqJSON, nil
}

// ToolDef 工具定义（用于 function calling）
type ToolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall 工具调用
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"-"`
}

// MarshalJSON 自定义序列化，OpenAI 格式要求：
// {"id": "...", "type": "function", "function": {"name": "...", "arguments": "..."}}
func (tc ToolCall) MarshalJSON() ([]byte, error) {
	argsJSON, err := json.Marshal(tc.Arguments)
	if err != nil {
		argsJSON = []byte("{}")
	}
	return json.Marshal(map[string]interface{}{
		"id":   tc.ID,
		"type": "function",
		"function": map[string]interface{}{
			"name":      tc.Name,
			"arguments": string(argsJSON),
		},
	})
}

// ChatResponse 聊天响应（包含工具调用）
type ChatResponse struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}
