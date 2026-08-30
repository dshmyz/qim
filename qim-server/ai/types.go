package ai

import "encoding/json"

// ErrStreamingToolsNotSupported 标记当前 Provider 不支持流式 tool-call（如 Anthropic）。
// 流式 ReAct 引擎首回合收到此错误时，让调用方降级到非流式 GetCompletionWithToolsMultiStep，
// 功能不回归；非首回合出现则视为硬错误返回。
var ErrStreamingToolsNotSupported = &streamingToolsUnsupportedError{}

type streamingToolsUnsupportedError struct{}

func (e *streamingToolsUnsupportedError) Error() string {
	return "streaming tools not supported by provider"
}
func (e *streamingToolsUnsupportedError) Is(target error) bool {
	_, ok := target.(*streamingToolsUnsupportedError)
	return ok
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ImageURL   string     `json:"image_url,omitempty"`
	ImageURLs  []string   `json:"-"` // 多图：分身批量多模态用。多个 base64 data URL，序列化见 MarshalJSON
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// Alias 是 Message 的字段镜像类型（不带 MarshalJSON 方法），供 MarshalJSON 在匿名结构里
// 嵌入以复用除 Content 外的全部字段序列化，同时避免因方法重入导致的无限递归。
type Alias Message

func (m Message) MarshalJSON() ([]byte, error) {
	// 多图优先（ImageURLs），其次单图（ImageURL，向后兼容），均输出 OpenAI content 数组格式。
	images := m.ImageURLs
	if len(images) == 0 && m.ImageURL != "" {
		images = []string{m.ImageURL}
	}
	if len(images) > 0 {
		content := make([]interface{}, 0, len(images)+1)
		content = append(content, map[string]string{"type": "text", "text": m.Content})
		for _, url := range images {
			content = append(content, map[string]interface{}{
				"type": "image_url",
				"image_url": map[string]interface{}{
					"url": url,
				},
			})
		}
		aux := struct {
			Role    string        `json:"role"`
			Content []interface{} `json:"content"`
		}{
			Role:    m.Role,
			Content: content,
		}
		return json.Marshal(aux)
	}
	aux := struct {
		Alias
		Content interface{} `json:"content"`
	}{
		Alias:   Alias(m),
		Content: m.Content,
	}
	if m.Content == "" && len(m.ToolCalls) > 0 {
		aux.Content = nil
	}
	return json.Marshal(aux)
}

type ChatCompletionRequest struct {
	Model            string    `json:"model"`
	Messages         []Message `json:"messages"`
	MaxTokens        int       `json:"max_tokens,omitempty"`
	Temperature      float64   `json:"temperature,omitempty"`
	FrequencyPenalty float64   `json:"frequency_penalty,omitempty"`
}

// extraFloat 从 ExtraParams 读取一个 float 参数，缺失时返回 def。
// 供各 provider 统一解析 frequency_penalty 等可选生成参数。
func extraFloat(extra map[string]interface{}, key string, def float64) float64 {
	if v, ok := extra[key].(float64); ok {
		return v
	}
	return def
}

type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int     `json:"index"`
		Message Message `json:"message"`
		Finish  string  `json:"finish_reason"`
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
	Error   *string      `json:"error,omitempty"`
	// Pending 非 nil 时表示本次流中产生了待用户确认的敏感工具调用（如 send_message）。
	// 仅供 SSE 下行给前端渲染确认条；ReAct 引擎内部不会设置此字段，由 handler 的
	// 进度回调在工具执行后填入。普通流式/工具增量路径恒为 nil。
	Pending *PendingSend `json:"pending,omitempty"`
	// ToolEvent 工具调用进度事件（侧边栏 SSE 通道）：start=running / end=ok|error。
	// bot 会话同类信息走 ai_tool_call WS 事件；侧边栏只有 SSE 通道，故帧内携带。
	// 前端按 tool_call_id 跨帧 upsert 成工具轨迹（与 ToolCallTrace 同构）。
	ToolEvent *ToolEvent `json:"tool_event,omitempty"`
	// ToolCalls 流式 tool-call 回合的增量：OpenAI 兼容流把 function.arguments 以分片
	// JSON 字符串多 chunk 发送，逐片透传由调用方（流式 ReAct 引擎）按 index 跨 chunk 累积，
	// 回合终了再整体 unmarshal。仅流式 tool-call 路径使用，普通流式恒为空。
	ToolCalls []ToolCallDelta `json:"tool_calls,omitempty"`
}

// ToolEvent 工具调用进度事件载荷（侧边栏 SSE）。
// 与 bot 会话的 ToolCallRecord 对齐：前端按 id upsert，渲染复用 ToolCallTrace。
type ToolEvent struct {
	Step       int                    `json:"step"`
	ToolCallID string                 `json:"tool_call_id,omitempty"`
	ToolName   string                 `json:"tool_name"`
	Label      string                 `json:"label"`
	Status     string                 `json:"status"` // running | ok | error
	Args       map[string]interface{} `json:"args,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

// PendingSend 侧边栏「待确认发送」卡片载荷：工具生成待确认请求后经 SSE 推给前端，
// 前端据此渲染确认条（目标会话 + 内容预览 + 确认/取消按钮），确认后调
// POST /ai/pending-actions/:id/confirm 真正执行。
type PendingSend struct {
	ID                   uint   `json:"id"`
	TargetConversationID uint   `json:"target_conversation_id"`
	TargetName           string `json:"target_name"`
	Preview              string `json:"preview"`
}

// ToolCallDelta 流式 tool-call 的一条增量：Index 标识同一流内第几个 tool call
// （稳定键），ID/Name/Arguments 分片累积——ID/Name 通常只在首个分片出现，
// Arguments 为从 0 拼接的原始 JSON 字符串。
type ToolCallDelta struct {
	Index     int    `json:"index"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// TokenUsage Token 用量统计
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// UsageProvider 可选接口：Provider 若实现此接口，Chat 时可返回 token 用量。
// 未实现的 Provider 仍走原 Chat() 路径，usage 为 nil。
type UsageProvider interface {
	ChatWithUsage(messages []Message) (string, *TokenUsage, error)
}

type StreamUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type AIConfig struct {
	Router      RouterConfig    `yaml:"router"`
	MaxTokens   int             `yaml:"max_tokens"`
	Temperature float64         `yaml:"temperature"`
	OpenAI      OpenAIConfig    `yaml:"openai"`
	Baidu       BaiduConfig     `yaml:"baidu"`
	Alibaba     AlibabaConfig   `yaml:"alibaba"`
	Tencent     TencentConfig   `yaml:"tencent"`
	Bytedance   BytedanceConfig `yaml:"bytedance"`
	Anthropic   AnthropicConfig `yaml:"anthropic"`
	DeepSeek    OpenAIConfig    `yaml:"deepseek"`
}

type OpenAIConfig struct {
	APIKey           string `yaml:"api_key"`
	Model            string `yaml:"model"`
	BaseURL          string `yaml:"base_url"`
	EmbeddingBaseURL string `yaml:"embedding_base_url"`
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
	TaskTypeVision      TaskType = "vision"
)

// Route 路由规则
type Route struct {
	Provider string   `yaml:"provider" json:"provider"`
	Model    string   `yaml:"model" json:"model"`
	Fallback []string `yaml:"fallback" json:"fallback"`
}

// RouterConfig 路由配置
type RouterConfig struct {
	DefaultTask TaskType           `yaml:"default_task" json:"defaultTask"`
	Routes      map[TaskType]Route `yaml:"routes" json:"routes"`
}

// Override 覆盖规则（用户/群组级）
type Override struct {
	TaskType TaskType `json:"task_type"`
	Provider string   `json:"provider"`
	Model    string   `json:"model"`
}

func (c *AIConfig) AllProviders() map[string]ProviderConfig {
	providers := make(map[string]ProviderConfig)
	if c.OpenAI.APIKey != "" {
		providers["openai"] = c.OpenAI.ToProviderConfig()
	}
	if c.Anthropic.APIKey != "" {
		providers["anthropic"] = c.Anthropic.ToProviderConfig()
	}
	if c.DeepSeek.APIKey != "" {
		providers["deepseek"] = c.DeepSeek.ToProviderConfig()
	}
	if c.Baidu.APIKey != "" {
		providers["baidu"] = c.Baidu.ToProviderConfig()
	}
	if c.Alibaba.APIKey != "" {
		providers["alibaba"] = c.Alibaba.ToProviderConfig()
	}
	if c.Tencent.SecretID != "" {
		providers["tencent"] = c.Tencent.ToProviderConfig()
	}
	if c.Bytedance.APIKey != "" {
		providers["bytedance"] = c.Bytedance.ToProviderConfig()
	}
	return providers
}
