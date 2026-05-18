# AI 多模型调度设计

## 概述

当前系统使用**单 Provider 模式**——`AIService` 持有一个 `Provider` 实例，所有 AI 任务（聊天、embedding、工具调用等）都使用同一个模型。本设计的目标是支持**多模型调度**，允许不同任务类型使用不同的 AI provider/model，并支持用户/群组级覆盖。

## 需求

1. **按任务类型路由**：不同 AI 任务（聊天、意图识别、分析、embedding 等）可以配置不同的 provider 和 model
2. **全局路由表 + 用户/群组覆盖**：既有系统级默认路由，也允许用户/群组按需覆盖
3. **Failover/降级**：当主 provider 不可用时，自动切换到备选 provider
4. **使用量统计**：扩展现有 `AIUsageLog`，记录每次调用的 provider、model、任务类型、耗时等

## 架构设计

### 整体结构

```
现在（单 Provider）               改造后（ModelRouter）
┌───────────────────┐            ┌──────────────────────────────┐
│    AIService      │            │        AIService             │
│  ┌─────────────┐  │            │  ┌────────────────────────┐  │
│  │  Provider   │  │            │  │    ModelRouter          │  │
│  │ (当前一个)   │  │            │  │  ┌──────────────────┐  │  │
│  └─────────────┘  │            │  │  │ chat → openai     │  │  │
└───────────────────┘            │  │  │ intent → deepseek │  │  │
                                  │  │  │ analysis → claude │  │  │
                                  │  │  │ embedding → qwen │  │  │
                                  │  │  │ fallback: ...    │  │  │
                                  │  │  └──────────────────┘  │  │
                                  │  └────────────────────────┘  │
                                  │  ┌────────────────────────┐  │
                                  │  │  Provider Pool          │  │
                                  │  │  map[string]Provider    │  │
                                  │  │  ├─ "openai"   → P1     │  │
                                  │  │  ├─ "deepseek" → P2     │  │
                                  │  │  ├─ "claude"   → P3     │  │
                                  │  │  └─ "qwen"     → P4     │  │
                                  │  └────────────────────────┘  │
                                  └──────────────────────────────┘
```

### 新增/修改的核心代码

#### 1. `ai/types.go` — 新增 TaskType 和 Route 类型定义

```go
// TaskType 任务类型
type TaskType string

const (
    TaskTypeChat           TaskType = "chat"
    TaskTypeIntent         TaskType = "intent_recognition"
    TaskTypeAnalysis       TaskType = "analysis"
    TaskTypeEmbedding      TaskType = "embedding"
    TaskTypeToolCalling    TaskType = "tool_calling"
    TaskTypeSearch         TaskType = "search"
    TaskTypeDigest         TaskType = "digest"
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
```

#### 2. `ai/router.go` — 新增 ModelRouter

```go
package ai

import "fmt"

// ModelRouter 模型路由器
type ModelRouter struct {
    routes      map[TaskType]Route
    defaultTask TaskType
}

func NewModelRouter(cfg RouterConfig) *ModelRouter {
    if cfg.DefaultTask == "" {
        cfg.DefaultTask = TaskTypeChat
    }
    if cfg.Routes == nil {
        cfg.Routes = make(map[TaskType]Route)
    }
    return &ModelRouter{
        routes:      cfg.Routes,
        defaultTask: cfg.DefaultTask,
    }
}

// SelectProvider 根据任务类型选择 provider
// 先检查 overrides，再查全局路由表，最后尝试 fallback
func (r *ModelRouter) SelectProvider(
    pool map[string]Provider,
    taskType TaskType,
    overrides ...Override,
) (Provider, string, error) {
    // 1. 检查 overrides
    for _, ov := range overrides {
        if ov.TaskType == taskType && ov.Provider != "" {
            if p, ok := pool[ov.Provider]; ok && p.IsConfigured() {
                return p, ov.Model, nil
            }
        }
    }

    // 2. 查全局路由表
    route, ok := r.routes[taskType]
    if !ok {
        route = r.routes[r.defaultTask]
    }

    // 3. 尝试主 provider + fallback
    candidates := []string{route.Provider}
    candidates = append(candidates, route.Fallback...)

    var lastErr error
    for _, name := range candidates {
        provider, ok := pool[name]
        if !ok || !provider.IsConfigured() {
            lastErr = fmt.Errorf("provider %s not configured", name)
            continue
        }
        return provider, route.Model, nil
    }

    return nil, "", fmt.Errorf("all providers unavailable for %s: %w", taskType, lastErr)
}
```

#### 3. `ai/ai_service.go` — AIService 改造

核心变化：
- `provider` 字段 → `pool map[string]Provider`
- 新增 `router *ModelRouter`
- 所有方法增加 `taskType` 参数
- 初始化时遍历所有配置，创建 provider 池

```go
type AIService struct {
    config    *AIConfig
    factory   *ProviderFactory
    pool      map[string]Provider
    router    *ModelRouter
    mu        sync.RWMutex
    mcpServer *MCPServer
}

func NewAIService(cfg *AIConfig) *AIService {
    svc := &AIService{
        config:  cfg,
        factory: NewProviderFactory(),
        pool:    make(map[string]Provider),
        router:  NewModelRouter(cfg.Router),
    }

    for name, providerCfg := range cfg.AllProviders() {
        provider, err := svc.factory.CreateProviderByName(name, providerCfg)
        if err != nil {
            log.Printf("[AI Service] Warning: Failed to init provider %s: %v", name, err)
            continue
        }
        svc.pool[name] = provider
    }

    return svc
}

// 方法签名变化示例
func (s *AIService) GetCompletion(taskType TaskType, messages []Message, overrides ...Override) (string, error) {
    provider, _, err := s.router.SelectProvider(s.pool, taskType, overrides...)
    if err != nil {
        return "", err
    }
    filteredMessages := s.filterMessages(messages)
    return provider.Chat(filteredMessages)
}

func (s *AIService) GetCompletionStream(taskType TaskType, messages []Message, onChunk func(chunk StreamChunk) error, overrides ...Override) error {
    provider, _, err := s.router.SelectProvider(s.pool, taskType, overrides...)
    if err != nil {
        return err
    }
    filteredMessages := s.filterMessages(messages)
    return provider.ChatStream(filteredMessages, onChunk)
}

// Embed 使用专门的 embedding 路由
func (s *AIService) Embed(text string) ([]float32, error) {
    provider, _, err := s.router.SelectProvider(s.pool, TaskTypeEmbedding)
    if err != nil {
        return nil, err
    }
    return provider.Embedding(text)
}
```

#### 4. `ai/types.go` — AIConfig 扩展

```go
type AIConfig struct {
    // 移除顶层的 Provider 字段（改为 router.routes 驱动）
    Router      RouterConfig    `yaml:"router"`
    MaxTokens   int             `yaml:"max_tokens"`
    Temperature float64         `yaml:"temperature"`
    OpenAI      OpenAIConfig    `yaml:"openai"`
    Baidu       BaiduConfig     `yaml:"baidu"`
    Alibaba     AlibabaConfig   `yaml:"alibaba"`
    Tencent     TencentConfig   `yaml:"tencent"`
    Bytedance   BytedanceConfig `yaml:"bytedance"`
    Anthropic   AnthropicConfig `yaml:"anthropic"`
    DeepSeek    OpenAIConfig    `yaml:"deepseek"`   // 新增（兼容 OpenAI 格式）
}

// AllProviders 返回所有已配置的 provider（有 API key 的）
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
    // ... 其他 provider 同理
    return providers
}
```

#### 5. `ai/provider_factory.go` — 扩展工厂

```go
// CreateProviderByName 根据 provider 名称创建
func (f *ProviderFactory) CreateProviderByName(name string, cfg ProviderConfig) (Provider, error) {
    switch name {
    case "openai":
        return f.createOpenAIProviderFromConfig(cfg), nil
    case "deepseek":
        return f.createOpenAIProviderFromConfig(cfg), nil  // DeepSeek 兼容 OpenAI 格式
    case "anthropic":
        return f.createAnthropicProviderFromConfig(cfg), nil
    // ... 其他 provider
    default:
        return nil, fmt.Errorf("unsupported provider: %s", name)
    }
}
```

#### 6. `service/eino_chat_model.go` — Graph 层适配

```go
type EinoChatModel struct {
    aiService *ai.AIService
    taskType  ai.TaskType
    userID    uint
    overrides []ai.Override
}

func NewEinoChatModel(aiService *ai.AIService, taskType ai.TaskType, userID uint) *EinoChatModel {
    return &EinoChatModel{
        aiService: aiService,
        taskType:  taskType,
        userID:    userID,
    }
}

func (m *EinoChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
    aiMessages := einoMessagesToAIMessages(input)
    callerCtx := &ai.CallerContext{UserID: m.userID}
    reply, err := m.aiService.GetCompletionWithTools(m.taskType, aiMessages, callerCtx, m.overrides...)
    if err != nil {
        return nil, err
    }
    return &schema.Message{Role: schema.Assistant, Content: reply}, nil
}
```

### 配置示例（config.yaml）

```yaml
ai:
  # 全局路由表
  router:
    default_task: "chat"
    routes:
      chat:
        provider: "openai"
        model: "qwen-flash"
      intent_recognition:
        provider: "openai"
        model: "deepseek-chat"
        fallback: ["alibaba"]
      analysis:
        provider: "anthropic"
        model: "claude-3-haiku"
        fallback: ["openai"]
      embedding:
        provider: "openai"
        model: "text-embedding-v4"
      tool_calling:
        provider: "openai"
        model: "qwen-flash"
      search:
        provider: "openai"
        model: "qwen-flash"
      digest:
        provider: "anthropic"
        model: "claude-3-haiku"
  
  # Provider 凭证（所有配置过的都会在启动时初始化到 pool 中）
  openai:
    api_key: "sk-xxx"
    model: "qwen-flash"
    base_url: "http://xxx"
    embedding_model: "text-embedding-v4"
  deepseek:
    api_key: "sk-yyy"
    model: "deepseek-chat"
    base_url: "https://api.deepseek.com"
  anthropic:
    api_key: "sk-zzz"
    model: "claude-3-haiku"
    base_url: "http://xxx/apps/anthropic"
```

### Failover 流程

```
SelectProvider(TaskTypeIntent, overrides)
  → 检查 overrides（用户级）→ 无匹配
  → 查全局路由表 → intent_recognition → provider: "deepseek", fallback: ["alibaba"]
  → 尝试 deepseek → IsConfigured() == false → 尝试 alibaba
  → alibaba 可用 → 返回 alibaba provider
  → 如果全部不可用 → 返回 error
```

### 使用量统计

扩展现有 `AIUsageLog` 模型：

```go
type AIUsageLog struct {
    ID        uint      `gorm:"primaryKey"`
    UserID    uint      `gorm:"index"`
    Provider  string    // 实际使用的 provider
    Model     string    // 实际使用的模型
    TaskType  string    // 任务类型
    TokensIn  int
    TokensOut int
    Duration  int64     // 耗时（毫秒）
    Status    string    // "success" | "fallback" | "error"
    CreatedAt time.Time
}
```

记录时机：在 `AIService` 的方法中统一拦截，使用闭包模式：

```go
func (s *AIService) recordUsage(taskType TaskType, providerName string) func(tokensIn, tokensOut int, err error) {
    start := time.Now()
    return func(tokensIn, tokensOut int, err error) {
        status := "success"
        if err != nil {
            status = "error"
        }
        // 写入 AIUsageLog
        log.Printf("[AI Usage] task=%s provider=%s tokens=%d/%d duration=%dms status=%s",
            taskType, providerName, tokensIn, tokensOut,
            time.Since(start).Milliseconds(), status)
    }
}
```

### 用户/群组覆盖

用户可以通过 `UserAIConfig` 配置自己的模型覆盖：

```go
// model/model.go
type UserAIConfig struct {
    // ... 已有字段
    Overrides []Override `json:"overrides" gorm:"type:json"`
}
```

优先级别：**用户 Override > 全局路由表**

Handler 层集成：

```go
func (h *AIHandler) GetCompletion(c *gin.Context) {
    userID := c.GetUint("user_id")
    userOverrides := h.aiConfigService.GetUserOverrides(userID)
    result, err := h.aiService.GetCompletion(
        TaskTypeChat, req.Messages, userOverrides...,
    )
}
```

### handler 调用方适配

所有 handler 调用 `AIService` 的方法需要增加 `taskType` 参数：

| Handler 路由 | TaskType |
|---|---|
| `POST /completion` | `chat` |
| `POST /completion/stream` | `chat` |
| `POST /summary` | `analysis` |
| `POST /translate` | `analysis` |
| `POST /rewrite` | `analysis` |
| `POST /polish` | `analysis` |
| `POST /search` | `search` |
| `GET /digest` | `digest` |
| `POST /configs/my/{id}/test` | 使用默认 chat |

### 向后兼容

- 不影响前端 API 接口格式
- `POST /api/v1/ai/completion` 的请求体不变，无需前端改动
- 用户可以在不感知多模型的情况下继续使用 AI 功能

## 变更文件清单

| 文件 | 变更类型 | 说明 |
|---|---|---|
| `ai/types.go` | 修改 | 新增 TaskType、Route、RouterConfig、Override 等类型 |
| `ai/router.go` | 新增 | ModelRouter 实现 |
| `ai/ai_service.go` | 修改 | 改为 provider pool + router；方法加 taskType 参数 |
| `ai/provider.go` | 修改 | Provider 接口不变（无需改动） |
| `ai/provider_factory.go` | 修改 | 新增 `CreateProviderByName` 方法 |
| `ai/provider_config.go` | 修改 | 新增 `ToProviderConfig` 转换方法 |
| `config/config.go` | 修改 | 无变化（AIConfig 变化自动继承） |
| `config.yaml` | 修改 | ai 配置结构变化 |
| `handler/ai_handler.go` | 修改 | 调用时传入 taskType |
| `handler/ai_text_handler.go` | 修改 | 同上 |
| `handler/ai_search_handler.go` | 修改 | 同上 |
| `handler/ai_summary_handler.go` | 修改 | 同上 |
| `service/eino_chat_model.go` | 修改 | 增加 taskType 绑定 |
| `service/summary_graph.go` | 修改 | 创建 EinoChatModel 时指定 taskType |
| `service/text_process_graph.go` | 修改 | 同上 |
| `service/unified_search_graph.go` | 修改 | 同上 |
| `service/smart_digest_graph.go` | 修改 | 同上 |
| `service/smart_reply_graph.go` | 修改 | 同上 |
| `service/avatar_reply_graph.go` | 修改 | 同上 |
| `model/model.go` | 修改 | AIUsageLog 扩展；UserAIConfig 增加 Overrides |
| `di/container.go` | 修改 | AIService 初始化无变化（自动兼容） |