package ai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

	// IsConfigured 检查提供商是否已正确配置
	IsConfigured() bool
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

// ExecuteWithRetry 执行 HTTP 请求并自动重试
func (bp *BaseProvider) ExecuteWithRetry(createRequest func() (*http.Request, error)) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 1; attempt <= bp.MaxRetries; attempt++ {
		req, reqErr := createRequest()
		if reqErr != nil {
			return nil, fmt.Errorf("failed to create request: %w", reqErr)
		}

		resp, err = bp.Client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			log.Printf("[AI Service] Request successful on attempt %d", attempt)
			return resp, nil
		}

		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}
		log.Printf("[AI Service] Request failed (attempt %d/%d): err=%v, status=%d",
			attempt, bp.MaxRetries, err, statusCode)

		if attempt < bp.MaxRetries {
			sleepDuration := time.Duration(attempt) * time.Second
			log.Printf("[AI Service] Retrying in %v", sleepDuration)
			time.Sleep(sleepDuration)
		}
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", bp.MaxRetries, err)
}

// ReadSSEStream 读取 SSE 格式的流并解析数据
func (bp *BaseProvider) ReadSSEStream(body io.ReadCloser, processData func(data string) error) error {
	defer body.Close()
	reader := bufio.NewScanner(body)

	for reader.Scan() {
		line := reader.Text()
		log.Printf("[AI Service] Received SSE line: %q", line)

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			if strings.TrimSpace(data) == "[DONE]" {
				log.Printf("[AI Service] Received stream end marker")
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
