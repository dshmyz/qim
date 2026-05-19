# AI 多模型调度 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 实现 ModelRouter 和 Provider Pool，支持 AI 任务按类型路由到不同 provider/model，包含 failover 和用户覆盖。

**架构：** 新增 `ModelRouter` 作为路由层，`AIService` 从单 Provider 改为 Provider Pool（`map[string]Provider`），所有方法增加 `TaskType` 参数。EinoChatModel 绑定 TaskType 让 Graph 也支持多模型。

**技术栈：** Go, Eino, Gin

---

### 任务 1：定义 TaskType、Route 等核心类型

**文件：**
- 修改：`qim-server/ai/types.go`

- 在 `ai/types.go` 中新增 TaskType 常量、Route、RouterConfig、Override 结构体

- [ ] **步骤 1：在 types.go 中新增类型定义**

```go
// TaskType 任务类型
type TaskType string

const (
    TaskTypeChat         TaskType = "chat"
    TaskTypeIntent       TaskType = "intent_recognition"
    TaskTypeAnalysis     TaskType = "analysis"
    TaskTypeEmbedding    TaskType = "embedding"
    TaskTypeToolCalling  TaskType = "tool_calling"
    TaskTypeSearch       TaskType = "search"
    TaskTypeDigest       TaskType = "digest"
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

- [ ] **步骤 2：编译验证**

运行：`cd qim-server && go build ./ai/...`
预期：编译成功

- [ ] **步骤 3：Commit**

```bash
git add qim-server/ai/types.go
git commit -m "feat(ai): add TaskType, Route, RouterConfig, Override types"
```

---

### 任务 2：实现 ModelRouter

**文件：**
- 创建：`qim-server/ai/router.go`

- [ ] **步骤 1：创建 router.go 实现 ModelRouter**

```go
package ai

import "fmt"

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

func (r *ModelRouter) SelectProvider(
    pool map[string]Provider,
    taskType TaskType,
    overrides ...Override,
) (Provider, string, error) {
    for _, ov := range overrides {
        if ov.TaskType == taskType && ov.Provider != "" {
            if p, ok := pool[ov.Provider]; ok && p.IsConfigured() {
                return p, ov.Model, nil
            }
        }
    }

    route, ok := r.routes[taskType]
    if !ok {
        route = r.routes[r.defaultTask]
    }

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

- [ ] **步骤 2：编译验证**

运行：`cd qim-server && go build ./ai/...`
预期：编译成功

- [ ] **步骤 3：Commit**

```bash
git add qim-server/ai/router.go
git commit -m "feat(ai): implement ModelRouter with failover support"
```

---

### 任务 3：扩展 ProviderConfig + ProviderFactory

**文件：**
- 修改：`qim-server/ai/provider_config.go`
- 修改：`qim-server/ai/provider_factory.go`

- [ ] **步骤 1：为每个 ProviderConfig 结构体添加 ToProviderConfig 方法**

在 `ai/provider_config.go` 中为 `OpenAIConfig`、`BaiduConfig` 等添加转换方法：

```go
func (c OpenAIConfig) ToProviderConfig() ProviderConfig {
    return ProviderConfig{
        APIKey:  c.APIKey,
        Model:   c.Model,
        BaseURL: c.BaseURL,
        ExtraParams: map[string]interface{}{
            "embedding_model": c.EmbeddingModel,
        },
    }
}

func (c BaiduConfig) ToProviderConfig() ProviderConfig {
    return ProviderConfig{
        APIKey:    c.APIKey,
        APISecret: c.SecretKey,
        Model:     c.Model,
        BaseURL:   c.BaseURL,
    }
}

func (c AlibabaConfig) ToProviderConfig() ProviderConfig {
    return ProviderConfig{
        APIKey:  c.APIKey,
        Model:   c.Model,
        BaseURL: c.BaseURL,
    }
}

func (c TencentConfig) ToProviderConfig() ProviderConfig {
    return ProviderConfig{
        APIKey:    c.SecretID,
        APISecret: c.SecretKey,
        Model:     c.Model,
        BaseURL:   c.BaseURL,
    }
}

func (c BytedanceConfig) ToProviderConfig() ProviderConfig {
    return ProviderConfig{
        APIKey:  c.APIKey,
        Model:   c.Model,
        BaseURL: c.BaseURL,
    }
}

func (c AnthropicConfig) ToProviderConfig() ProviderConfig {
    return ProviderConfig{
        APIKey:  c.APIKey,
        Model:   c.Model,
        BaseURL: c.BaseURL,
    }
}
```

- [ ] **步骤 2：在 ProviderFactory 中新增 CreateProviderByName 方法**

在 `ai/provider_factory.go` 中为 `openai` 创建一个通用的 helper 方法，供多个名称共用：

```go
func (f *ProviderFactory) CreateProviderByName(name string, cfg ProviderConfig) (Provider, error) {
    switch name {
    case "openai", "deepseek":
        return f.createGenericOpenAIProvider(name, cfg), nil
    case "anthropic":
        return f.createAnthropicProviderFromConfig(cfg), nil
    default:
        return nil, fmt.Errorf("unsupported provider: %s", name)
    }
}

func (f *ProviderFactory) createGenericOpenAIProvider(name string, cfg ProviderConfig) Provider {
    extraParams := cfg.ExtraParams
    if extraParams == nil {
        extraParams = make(map[string]interface{})
    }
    if _, ok := extraParams["max_tokens"]; !ok {
        extraParams["max_tokens"] = 1000
    }
    if _, ok := extraParams["temperature"]; !ok {
        extraParams["temperature"] = 0.7
    }
    return NewOpenAIProvider(ProviderConfig{
        APIKey:  cfg.APIKey,
        Model:   cfg.Model,
        BaseURL: cfg.BaseURL,
        ExtraParams: extraParams,
    })
}

func (f *ProviderFactory) createAnthropicProviderFromConfig(cfg ProviderConfig) Provider {
    return NewAnthropicProvider(ProviderConfig{
        APIKey:  cfg.APIKey,
        Model:   cfg.Model,
        BaseURL: cfg.BaseURL,
        ExtraParams: map[string]interface{}{
            "max_tokens":  1000,
            "temperature": 0.7,
        },
    })
}
```

- [ ] **步骤 3：编译验证**

运行：`cd qim-server && go build ./ai/...`
预期：编译成功

- [ ] **步骤 4：Commit**

```bash
git add qim-server/ai/provider_config.go qim-server/ai/provider_factory.go
git commit -m "feat(ai): add CreateProviderByName and ToProviderConfig"
```

---

### 任务 4：重构 AIService — Provider Pool + Router

**文件：**
- 修改：`qim-server/ai/ai_service.go`
- 修改：`qim-server/ai/types.go`（更新 AIConfig）

- [ ] **步骤 1：更新 AIConfig 添加 Router 和 AllProviders 方法**

在 `ai/types.go` 中修改 `AIConfig`：

```go
type AIConfig struct {
    Router      RouterConfig      `yaml:"router"`
    MaxTokens   int               `yaml:"max_tokens"`
    Temperature float64           `yaml:"temperature"`
    OpenAI      OpenAIConfig      `yaml:"openai"`
    Baidu       BaiduConfig       `yaml:"baidu"`
    Alibaba     AlibabaConfig     `yaml:"alibaba"`
    Tencent     TencentConfig     `yaml:"tencent"`
    Bytedance   BytedanceConfig   `yaml:"bytedance"`
    Anthropic   AnthropicConfig   `yaml:"anthropic"`
    DeepSeek    OpenAIConfig      `yaml:"deepseek"`
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
```

- [ ] **步骤 2：重构 AIService 结构体和方法**

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
        log.Printf("[AI Service] Provider %s initialized", name)
    }

    if len(svc.pool) == 0 {
        log.Printf("[AI Service] Warning: No AI providers initialized")
    } else {
        log.Printf("[Service] %d AI providers initialized", len(svc.pool))
    }

    return svc
}

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

func (s *AIService) GetCompletionWithTools(taskType TaskType, messages []Message, callerCtx *CallerContext, overrides ...Override) (string, error) {
    s.mu.RLock()
    mcpServer := s.mcpServer
    s.mu.RUnlock()

    if mcpServer == nil {
        return s.GetCompletion(taskType, messages, overrides...)
    }

    provider, _, err := s.router.SelectProvider(s.pool, taskType, overrides...)
    if err != nil {
        return "", err
    }

    // 构建工具定义列表（代码不变，使用传入的 provider 替换 s.provider）
    tools := mcpServer.ListTools()
    toolDefs := make([]ToolDef, 0, len(tools))
    for _, tool := range tools {
        name := tool["name"].(string)
        desc := tool["description"].(string)
        params := tool["parameters"].(map[string]interface{})
        toolDefs = append(toolDefs, ToolDef{Name: name, Description: desc, Parameters: params})
    }

    // 尝试 native function calling
    log.Printf("[AI Service] 尝试 native function calling，工具数: %d", len(toolDefs))
    resp, err := provider.ChatWithTools(messages, toolDefs)
    if err != nil {
        log.Printf("[AI Service] Native function calling not supported, falling back to prompt engineering: %v", err)
        return s.getCompletionWithToolsPromptEngineering(taskType, messages, callerCtx)
    }

    if len(resp.ToolCalls) == 0 {
        return resp.Content, nil
    }

    newMessages := make([]Message, len(messages))
    copy(newMessages, messages)
    newMessages = append(newMessages, Message{Role: "assistant", Content: resp.Content, ToolCalls: resp.ToolCalls})

    for _, tc := range resp.ToolCalls {
        log.Printf("[AI Service] 执行工具: name=%s, args=%v", tc.Name, tc.Arguments)
        result, execErr := mcpServer.ExecuteTool(tc.Name, tc.Arguments, callerCtx)
        if execErr != nil {
            return "", execErr
        }
        resultJSON, _ := json.Marshal(result)
        newMessages = append(newMessages, Message{Role: "tool", Content: string(resultJSON), ToolCallID: tc.ID})
    }

    finalResp, err := provider.ChatWithTools(newMessages, toolDefs)
    if err != nil {
        return "", err
    }
    return finalResp.Content, nil
}

func (s *AIService) getCompletionWithToolsPromptEngineering(taskType TaskType, messages []Message, callerCtx *CallerContext) (string, error) {
    s.mu.RLock()
    mcpServer := s.mcpServer
    s.mu.RUnlock()

    if mcpServer == nil {
        return s.GetCompletion(taskType, messages)
    }

    tools := mcpServer.ListTools()
    toolsDesc := "你可以使用以下工具（如果用户请求涉及管理操作，请使用工具）：\n\n"
    for _, tool := range tools {
        name := tool["name"].(string)
        desc := tool["description"].(string)
        params := tool["parameters"].(map[string]interface{})
        toolsDesc += fmt.Sprintf("工具: %s\n说明: %s\n", name, desc)
        toolsDesc += "参数:\n"
        for pname, pinfo := range params {
            if pmap, ok := pinfo.(map[string]interface{}); ok {
                req := ""
                if pmap["required"] == true {
                    req = " (必填)"
                }
                toolsDesc += fmt.Sprintf("  - %s: %s%s\n", pname, pmap["description"], req)
            }
        }
        toolsDesc += "\n"
    }

    toolInstruction := toolsDesc + `如需调用工具，请严格按照以下 JSON 格式返回：
{"tool_call": {"name":"工具名称", "arguments": {"参数名":"参数值"}}}

如果不需要调用工具，直接输出回复内容。
注意：只在用户明确要求执行管理操作时才调用工具，普通聊天不要调用工具。`

    var newMessages []Message
    for _, msg := range messages {
        if msg.Role == "system" {
            newMessages = append(newMessages, Message{Role: "system", Content: msg.Content + "\n\n" + toolInstruction})
        } else {
            newMessages = append(newMessages, msg)
        }
    }

    log.Printf("[AI Service] 工具调用 - 发送请求到 AI，工具数: %d", len(tools))
    reply, err := s.GetCompletion(taskType, newMessages)
    if err != nil {
        return "", err
    }

    toolCall, err := parseToolCall(reply)
    if err != nil || toolCall == nil {
        return reply, nil
    }

    log.Printf("[AI Service] 工具调用 - 检测到工具调用: name=%s, args=%v", toolCall.Name, toolCall.Arguments)
    result, err := mcpServer.ExecuteTool(toolCall.Name, toolCall.Arguments, callerCtx)
    if err != nil {
        return "", err
    }

    newMessages = append(newMessages, Message{Role: "assistant", Content: reply})
    resultJSON, _ := json.Marshal(result)
    newMessages = append(newMessages, Message{Role: "user", Content: fmt.Sprintf("工具 %s 执行结果: %s\n请根据这个结果生成给用户的回复。", toolCall.Name, string(resultJSON))})

    finalReply, err := s.GetCompletion(taskType, newMessages)
    if err != nil {
        return "", err
    }
    return finalReply, nil
}

func (s *AIService) IsConfigured() bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return len(s.pool) > 0
}

// Embed 使用 embedding 路由
func (s *AIService) Embed(text string) ([]float32, error) {
    provider, _, err := s.router.SelectProvider(s.pool, TaskTypeEmbedding)
    if err != nil {
        return nil, err
    }
    return provider.Embedding(text)
}
```

- [ ] **步骤 3：编译验证**

运行：`cd qim-server && go build ./ai/...`
预期：编译成功

- [ ] **步骤 4：Commit**

```bash
git add qim-server/ai/ai_service.go qim-server/ai/types.go
git commit -m "feat(ai): refactor AIService with provider pool and router"
```

---

### 任务 5：更新 config.yaml 和配置加载

**文件：**
- 修改：`qim-server/config.yaml`

- [ ] **步骤 1：更新 config.yaml 的 AI 配置结构**

将 ai 配置改为新格式：

```yaml
ai:
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
  
  max_tokens: 1000
  temperature: 0.7
  
  openai:
    api_key: "sk-rds-e4ou93yis8whzi8680k7js4oxxqgr4my"
    model: "qwen-flash"
    base_url: "http://0yc4r0gn901.copilot.rds.aliyuncs.com/compatible-mode/v1"
    embedding_model: "text-embedding-v4"
  baidu:
    api_key: ""
    secret_key: ""
    model: "ERNIE-Bot-4.0"
    base_url: "https://aip.baidubce.com"
  alibaba:
    api_key: ""
    api_secret: ""
    model: "qwen-plus"
    base_url: "https://dashscope.aliyuncs.com/api/v1"
  tencent:
    secret_id: ""
    secret_key: ""
    model: "hunyuan-pro"
    base_url: "https://hunyuan.tencentcloudapi.com"
  bytedance:
    api_key: ""
    model: "doubao-pro-1.0"
    base_url: "https://ark.cn-beijing.volces.com/api/v3"
  anthropic:
    api_key: "YOUR-AI-API-KEY-HERE"
    model: "claude-3-haiku"
    base_url: "http://0yc4r0gn901.copilot.rds.aliyuncs.com/apps/anthropic"
```

- [ ] **步骤 2：编译验证**

运行：`cd qim-server && go build ./...`
预期：编译成功

- [ ] **步骤 3：Commit**

```bash
git add qim-server/config.yaml
git commit -m "feat(ai): update config.yaml with routing table"
```

---

### 任务 6：适配 EinoChatModel — 绑定 TaskType

**文件：**
- 修改：`qim-server/service/eino_chat_model.go`

- [ ] **步骤 1：为 EinoChatModel 增加 taskType 字段**

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
```

- [ ] **步骤 2：更新 Generate 和 Stream 方法调用**

```go
func (m *EinoChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
    aiMessages := einoMessagesToAIMessages(input)
    callerCtx := &ai.CallerContext{UserID: m.userID}
    reply, err := m.aiService.GetCompletionWithTools(m.taskType, aiMessages, callerCtx, m.overrides...)
    if err != nil {
        return nil, err
    }
    return &schema.Message{Role: schema.Assistant, Content: reply}, nil
}

func (m *EinoChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
    aiMessages := einoMessagesToAIMessages(input)
    
    for _, msg := range aiMessages {
        log.Printf("[EinoChatModel] [%s]: %s", msg.Role, msg.Content)
    }

    sr, sw := schema.Pipe[*schema.Message](0)

    go func() {
        defer sw.Close()
        err := m.aiService.GetCompletionStream(m.taskType, aiMessages, func(chunk ai.StreamChunk) error {
            msg := &schema.Message{Role: schema.Assistant, Content: chunk.Content}
            sw.Send(msg, nil)
            return nil
        }, m.overrides...)
        if err != nil {
            log.Printf("[EinoChatModel] Stream error: %v", err)
            sw.Send(nil, err)
        }
    }()

    return sr, nil
}
```

- [ ] **步骤 3：更新所有 Graph 的 NewEinoChatModel 调用**

修改以下文件中创建 EinoChatModel 的位置，传入对应的 taskType：

- `service/summary_graph.go`: `NewEinoChatModel(g.aiService, ai.TaskTypeAnalysis, 0)`
- `service/text_process_graph.go`: `NewEinoChatModel(g.aiService, ai.TaskTypeAnalysis, 0)`
- `service/unified_search_graph.go`: `NewEinoChatModel(g.aiService, ai.TaskTypeSearch, 0)`
- `service/smart_digest_graph.go`: `NewEinoChatModel(g.aiService, ai.TaskTypeDigest, 0)`
- `service/smart_reply_graph.go`: `NewEinoChatModel(g.aiService, ai.TaskTypeChat, 0)`
- `service/avatar_reply_graph.go`: `NewEinoChatModel(g.aiService, ai.TaskTypeChat, 0)`

- [ ] **步骤 4：编译验证**

运行：`cd qim-server && go build ./service/...`
预期：编译成功

- [ ] **步骤 5：Commit**

```bash
git add qim-server/service/eino_chat_model.go qim-server/service/summary_graph.go qim-server/service/text_process_graph.go qim-server/service/unified_search_graph.go qim-server/service/smart_digest_graph.go qim-server/service/smart_reply_graph.go qim-server/service/avatar_reply_graph.go
git commit -m "feat(ai): bind TaskType to EinoChatModel and all graphs"
```

---

### 任务 7：适配 Handler 层

**文件：**
- 修改：`qim-server/handler/ai_handler.go`
- 修改：`qim-server/handler/ai_text_handler.go`
- 修改：`qim-server/handler/ai_search_handler.go`
- 修改：`qim-server/handler/ai_summary_handler.go`

- [ ] **步骤 1：更新 ai_handler.go 基本调用**

在 `ai_handler.go` 中搜索 `h.aiService.GetCompletion`、`h.aiService.GetCompletionStream`、`h.aiService.IsConfigured` 等调用，增加 TaskType 参数：

```go
// GetCompletion
result, err := h.aiService.GetCompletion(ai.TaskTypeChat, req.Messages)

// GetCompletionStream
err := h.aiService.GetCompletionStream(ai.TaskTypeChat, req.Messages, func(chunk ai.StreamChunk) error {

// IsConfigured（无需 taskType，保持原样）
if !h.aiService.IsConfigured() {
```

- [ ] **步骤 2：更新 ai_summary_handler.go**

```go
// 调用 AI 生成摘要时使用 TaskTypeAnalysis
result, err := h.aiService.GetCompletion(ai.TaskTypeAnalysis, messages)
```

- [ ] **步骤 3：更新 ai_text_handler.go**

```go
// 翻译、改写、润色都使用 TaskTypeAnalysis
result, err := h.aiService.GetCompletion(ai.TaskTypeAnalysis, messages)
```

- [ ] **步骤 4：更新 ai_search_handler.go**

```go
// 搜索使用 TaskTypeSearch
result, err := h.aiService.GetCompletion(ai.TaskTypeSearch, messages)
```

- [ ] **步骤 5：编译验证**

运行：`cd qim-server && go build ./handler/...`
预期：编译成功

- [ ] **步骤 6：Commit**

```bash
git add qim-server/handler/ai_handler.go qim-server/handler/ai_text_handler.go qim-server/handler/ai_search_handler.go qim-server/handler/ai_summary_handler.go
git commit -m "feat(ai): adapt handlers to pass TaskType to AIService"
```

---

### 任务 8：扩展使用量统计（AIUsageLog）

**文件：**
- 修改：`qim-server/model/model.go`
- 修改：`qim-server/handler/ai_usage_handler.go`
- 修改：`qim-server/ai/ai_service.go`（插入统计逻辑）

- [ ] **步骤 1：扩展 AIUsageLog 模型**

在 `model/model.go` 中找到 `AIUsageLog`，增加字段：

```go
type AIUsageLog struct {
    ID        uint      `gorm:"primaryKey"`
    UserID    uint      `gorm:"index"`
    Provider  string    `gorm:"size:64"`
    Model     string    `gorm:"size:128"`
    TaskType  string    `gorm:"size:64"`
    TokensIn  int
    TokensOut int
    Duration  int64
    Status    string    `gorm:"size:32"`
    CreatedAt time.Time
}
```

- [ ] **步骤 2：在 AI 调用点插入统一统计逻辑**

在 `ai/ai_service.go` 的方法中增加记录调用：

```go
// GetCompletion 示例
func (s *AIService) GetCompletion(taskType TaskType, messages []Message, overrides ...Override) (string, error) {
    provider, modelName, err := s.router.SelectProvider(s.pool, taskType, overrides...)
    if err != nil {
        return "", err
    }
    filteredMessages := s.filterMessages(messages)
    start := time.Now()
    result, err := provider.Chat(filteredMessages)
    
    // 记录使用量
    go func() {
        status := "success"
        if err != nil {
            status = "error"
        }
        log.Printf("[AI Usage] task=%s provider=%s model=%s duration=%dms status=%s",
            taskType, provider.Name(), modelName, time.Since(start).Milliseconds(), status)
    }()
    
    return result, err
}
```

- [ ] **步骤 3：编译验证**

运行：`cd qim-server && go build ./...`
预期：编译成功

- [ ] **步骤 4：Commit**

```bash
git add qim-server/model/model.go qim-server/ai/ai_service.go
git commit -m "feat(ai): extend AIUsageLog with provider/model/taskType fields"
```

---

### 任务 9：支持用户 Override

**文件：**
- 修改：`qim-server/model/model.go`（UserAIConfig 增加 Overrides）
- 修改：`qim-server/handler/user_ai_config_handler.go`（API 支持 overrides）
- 修改：`qim-server/service/ai_config_service.go`（获取用户 overrides）

- [ ] **步骤 1：UserAIConfig 增加 Overrides 字段**

```go
type UserAIConfig struct {
    ID               uint           `gorm:"primaryKey"`
    UserID           uint           `gorm:"uniqueIndex"`
    ConfigName       string         `gorm:"size:128"`
    Provider         string         `gorm:"size:64"`
    ModelName        string         `gorm:"size:128"`
    BaseURL          string         `gorm:"size:512"`
    APIKeyEncrypted  string         `gorm:"size:512"`
    Temperature      float64        `gorm:"default:0.7"`
    MaxTokens        int            `gorm:"default:1000"`
    IsVerified       bool           `gorm:"default:false"`
    Overrides        string         `gorm:"type:json"` // JSON 序列化的 []ai.Override
    CreatedAt        time.Time
    UpdatedAt        time.Time
}
```

- [ ] **步骤 2：在 ai_config_service.go 中新增 GetUserOverrides 方法**

```go
func (s *AIConfigService) GetUserOverrides(userID uint) []ai.Override {
    config, err := s.GetByUserID(userID)
    if err != nil || config == nil || config.Overrides == "" {
        return nil
    }
    var overrides []ai.Override
    if err := json.Unmarshal([]byte(config.Overrides), &overrides); err != nil {
        log.Printf("[AIConfigService] Failed to unmarshal overrides for user %d: %v", userID, err)
        return nil
    }
    return overrides
}
```

- [ ] **步骤 3：在 user_ai_config_handler.go 中增加 overrides 字段支持**

在请求体和响应体中增加 overrides 字段：

```go
type CreateConfigRequest struct {
    ConfigName string        `json:"config_name"`
    Provider   string        `json:"provider"`
    APIKey     string        `json:"api_key"`
    ModelName  string        `json:"model_name"`
    BaseURL    string        `json:"base_url"`
    Overrides  []ai.Override `json:"overrides,omitempty"`
}

// Handler 中在创建/更新时序列化 Overrides
overridesJSON, _ := json.Marshal(req.Overrides)
```

- [ ] **步骤 4：Handler 集成 — 在 ai_handler.go 中使用用户 overrides**

```go
// GetCompletion 中获取用户 overrides
userID := c.GetUint("user_id")
userOverrides := h.aiConfigService.GetUserOverrides(userID)
result, err := h.aiService.GetCompletion(ai.TaskTypeChat, req.Messages, userOverrides...)
```

- [ ] **步骤 5：编译验证**

运行：`cd qim-server && go build ./...`
预期：编译成功

- [ ] **步骤 6：Commit**

```bash
git add qim-server/model/model.go qim-server/service/ai_config_service.go qim-server/handler/user_ai_config_handler.go
git commit -m "feat(ai): support user-level model overrides"
```

---

### 任务 10：服务启动验证

- [ ] **步骤 1：启动服务并验证启动日志**

运行：`cd qim-server && go run .`
预期：启动日志中应看到：
- `[AI Service] Provider openai initialized providers: openai, anthropic...`（多个 provider）
- `[AI Service] ModelRouter initialized with 7 routes`
- 原有 API 正常响应

- [ ] **步骤 2：测试基础 API**

```bash
curl -X POST http://localhost:8080/api/v1/ai/completion \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"Hello"}]}'
```

预期：正常返回 AI 回复

- [ ] **步骤 3：测试流式 API**

```bash
curl -X POST http://localhost:8080/api/v1/ai/completion/stream \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"Hello"}]}'
```

预期：正常返回 SSE 流