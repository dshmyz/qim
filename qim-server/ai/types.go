package ai

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int     `json:"index"`
		Message Message `json:"message"`
		Finish  string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type StreamChunk struct {
	Content string       `json:"content"`
	Finish  *string      `json:"finish,omitempty"`
	Usage   *StreamUsage `json:"usage,omitempty"`
}

type StreamUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type AIConfig struct {
	Provider    string          `yaml:"provider"`
	MaxTokens   int             `yaml:"max_tokens"`
	Temperature float64         `yaml:"temperature"`
	OpenAI      OpenAIConfig    `yaml:"openai"`
	Baidu       BaiduConfig     `yaml:"baidu"`
	Alibaba     AlibabaConfig   `yaml:"alibaba"`
	Tencent     TencentConfig   `yaml:"tencent"`
	Bytedance   BytedanceConfig `yaml:"bytedance"`
	Anthropic   AnthropicConfig `yaml:"anthropic"`
}

type OpenAIConfig struct {
	APIKey         string `yaml:"api_key"`
	Model          string `yaml:"model"`
	BaseURL        string `yaml:"base_url"`
	EmbeddingModel string `yaml:"embedding_model"`
}

type BaiduConfig struct {
	APIKey    string `yaml:"api_key"`
	SecretKey string `yaml:"secret_key"`
	Model     string `yaml:"model"`
	BaseURL   string `yaml:"base_url"`
}

type AlibabaConfig struct {
	APIKey    string `yaml:"api_key"`
	APISecret string `yaml:"api_secret"`
	Model     string `yaml:"model"`
	BaseURL   string `yaml:"base_url"`
}

type TencentConfig struct {
	SecretID  string `yaml:"secret_id"`
	SecretKey string `yaml:"secret_key"`
	Model     string `yaml:"model"`
	BaseURL   string `yaml:"base_url"`
}

type BytedanceConfig struct {
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
	BaseURL string `yaml:"base_url"`
}

type AnthropicConfig struct {
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
	BaseURL string `yaml:"base_url"`
}

// TaskType 任务类型
type TaskType string

const (
	TaskTypeChat        TaskType = "chat"
	TaskTypeIntent      TaskType = "intent_recognition"
	TaskTypeAnalysis    TaskType = "analysis"
	TaskTypeEmbedding   TaskType = "embedding"
	TaskTypeToolCalling TaskType = "tool_calling"
	TaskTypeSearch      TaskType = "search"
	TaskTypeDigest      TaskType = "digest"
)

// Route 路由规则
type Route struct {
	Provider string   `yaml:"provider"`
	Model    string   `yaml:"model"`
	Fallback []string `yaml:"fallback"`
}

// RouterConfig 路由配置
type RouterConfig struct {
	DefaultTask TaskType           `yaml:"default_task"`
	Routes      map[TaskType]Route `yaml:"routes"`
}

// Override 覆盖规则（用户/群组级）
type Override struct {
	TaskType TaskType `json:"task_type"`
	Provider string   `json:"provider"`
	Model    string   `json:"model"`
}
