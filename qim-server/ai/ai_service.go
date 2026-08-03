package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type AIService struct {
	config       *AIConfig
	factory      *ProviderFactory
	pool         map[string]Provider
	configPool   map[string]Provider // config.yaml 初始化的 pool 副本，DB 为空时兜底
	router       *ModelRouter
	mu           sync.RWMutex
	toolRegistry *ToolRegistry
	usageSink    func(taskType TaskType, provider, model string, usage *TokenUsage, durationMs int64) // 异步落库回调
}

func NewAIService(cfg *AIConfig) *AIService {
	svc := &AIService{
		config:     cfg,
		factory:    NewProviderFactory(),
		pool:       make(map[string]Provider),
		configPool: make(map[string]Provider),
		router:     NewModelRouter(cfg.Router),
	}

	for name, providerCfg := range cfg.AllProviders() {
		provider, err := svc.factory.CreateProviderByName(name, providerCfg)
		if err != nil {
			log.Printf("[AI Service] Warning: Failed to init provider %s: %v", name, err)
			continue
		}
		svc.pool[name] = provider
		svc.configPool[name] = provider
		log.Printf("[AI Service] Provider %s initialized", name)
	}

	if len(svc.pool) == 0 {
		log.Printf("[AI Service] Warning: No AI providers initialized")
	} else {
		log.Printf("[AI Service] %d AI providers initialized", len(svc.pool))
	}

	return svc
}

func (s *AIService) SetToolRegistry(toolRegistry *ToolRegistry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.toolRegistry = toolRegistry
}

// SetUsageSink 设置 token 用量异步落库回调。传 nil 则禁用落库。
func (s *AIService) SetUsageSink(fn func(taskType TaskType, provider, model string, usage *TokenUsage, durationMs int64)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.usageSink = fn
}

// SetProviderForTesting 仅供测试：绕过 factory 直接注入 provider（如 mock）到 pool，
// 用于在不依赖真实 LLM API 的情况下验证 tool calling 链路。
func (s *AIService) SetProviderForTesting(name string, p Provider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pool == nil {
		s.pool = make(map[string]Provider)
	}
	s.pool[name] = p
	if s.configPool == nil {
		s.configPool = make(map[string]Provider)
	}
	s.configPool[name] = p
}

func (s *AIService) GetToolRegistry() *ToolRegistry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.toolRegistry
}

// selectProvider 在读锁保护下快照 router 与 pool 再做路由选择，
// 避免与 ReloadProvidersFromDB / UpdateConfig 的写入发生并发 map 读写崩溃。
func (s *AIService) selectProvider(taskType TaskType, overrides ...Override) (Provider, string, error) {
	s.mu.RLock()
	router := s.router
	pool := s.pool
	s.mu.RUnlock()
	return router.SelectProvider(pool, taskType, overrides...)
}

func (s *AIService) GetCompletion(taskType TaskType, messages []Message, overrides ...Override) (string, error) {
	provider, modelName, err := s.selectProvider(taskType, overrides...)
	if err != nil {
		return "", err
	}
	filteredMessages := s.filterMessages(messages)
	start := time.Now()
	var result string
	var usage *TokenUsage

	// 优先走 UsageProvider 接口获取 token 用量
	if up, ok := provider.(UsageProvider); ok {
		var u *TokenUsage
		if modelName != "" {
			if upm, ok2 := provider.WithModel(modelName).(UsageProvider); ok2 {
				result, u, err = upm.ChatWithUsage(filteredMessages)
			} else {
				result, err = provider.WithModel(modelName).Chat(filteredMessages)
			}
		} else {
			result, u, err = up.ChatWithUsage(filteredMessages)
		}
		usage = u
	} else {
		if modelName != "" {
			result, err = provider.WithModel(modelName).Chat(filteredMessages)
		} else {
			result, err = provider.Chat(filteredMessages)
		}
	}

	duration := time.Since(start).Milliseconds()
	status := "success"
	if err != nil {
		status = "error"
	}
	log.Printf("[AI Usage] task=%s provider=%s model=%s duration=%dms status=%s",
		taskType, provider.Name(), modelName, duration, status)

	// 异步落库
	if usage != nil && s.usageSink != nil {
		s.usageSink(taskType, provider.Name(), modelName, usage, duration)
	}

	return result, err
}

func (s *AIService) GetCompletionStream(taskType TaskType, messages []Message, onChunk func(chunk StreamChunk) error, overrides ...Override) error {
	return s.GetCompletionStreamWithContext(context.Background(), taskType, messages, onChunk, overrides...)
}

// GetCompletionStreamWithContext 支持 context 取消的流式请求（用于超时控制）
func (s *AIService) GetCompletionStreamWithContext(ctx context.Context, taskType TaskType, messages []Message, onChunk func(chunk StreamChunk) error, overrides ...Override) error {
	provider, modelName, err := s.selectProvider(taskType, overrides...)
	if err != nil {
		return err
	}
	filteredMessages := s.filterMessages(messages)
	start := time.Now()
	if modelName != "" {
		err = provider.WithModel(modelName).ChatStreamWithContext(ctx, filteredMessages, onChunk)
	} else {
		err = provider.ChatStreamWithContext(ctx, filteredMessages, onChunk)
	}
	duration := time.Since(start).Milliseconds()
	status := "success"
	if err != nil {
		status = "error"
	}
	log.Printf("[AI Usage] task=%s provider=%s model=%s duration=%dms status=%s",
		taskType, provider.Name(), modelName, duration, status)
	return err
}

func (s *AIService) getCompletionWithToolsCore(taskType TaskType, messages []Message, callerCtx *CallerContext, allowed []string, overrides ...Override) (string, error) {
	s.mu.RLock()
	toolRegistry := s.toolRegistry
	s.mu.RUnlock()

	if toolRegistry == nil {
		return s.GetCompletion(taskType, messages, overrides...)
	}

	provider, modelName, err := s.selectProvider(taskType, overrides...)
	if err != nil {
		return "", err
	}

	tools := toolRegistry.ListTools()
	// allowed 非空时只注入白名单内的工具（群聊助手过滤掉运维工具）
	allowedSet := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		allowedSet[a] = true
	}
	toolDefs := make([]ToolDef, 0, len(tools))
	for _, tool := range tools {
		name := tool["name"].(string)
		if len(allowedSet) > 0 && !allowedSet[name] {
			continue
		}
		desc := tool["description"].(string)
		params := tool["parameters"].(map[string]interface{})
		toolDefs = append(toolDefs, ToolDef{
			Name:        name,
			Description: desc,
			Parameters:  params,
		})
	}

	log.Printf("[AI Service] 尝试使用 native function calling，工具数: %d", len(toolDefs))
	var resp *ChatResponse
	if modelName != "" {
		resp, err = provider.WithModel(modelName).ChatWithTools(messages, toolDefs)
	} else {
		resp, err = provider.ChatWithTools(messages, toolDefs)
	}
	if err != nil {
		log.Printf("[AI Service] Native function calling not supported, falling back to prompt engineering: %v", err)
		return s.getCompletionWithToolsPromptEngineering(taskType, messages, callerCtx, allowed, overrides...)
	}

	if len(resp.ToolCalls) == 0 {
		log.Printf("[AI Service] Native function calling - 无工具调用，直接返回回复")
		return resp.Content, nil
	}

	log.Printf("[AI Service] Native function calling - 检测到 %d 个工具调用", len(resp.ToolCalls))

	newMessages := make([]Message, len(messages))
	copy(newMessages, messages)

	newMessages = append(newMessages, Message{
		Role:      "assistant",
		Content:   resp.Content,
		ToolCalls: resp.ToolCalls,
	})

	for _, tc := range resp.ToolCalls {
		if len(allowedSet) > 0 && !allowedSet[tc.Name] {
			return "", fmt.Errorf("tool %s is not allowed", tc.Name)
		}
		log.Printf("[AI Service] 执行工具: name=%s, args=%v", tc.Name, tc.Arguments)
		result, execErr := toolRegistry.ExecuteTool(tc.Name, tc.Arguments, callerCtx)
		if execErr != nil {
			log.Printf("[AI Service] 工具执行失败: %v", execErr)
			return "", execErr
		}
		log.Printf("[AI Service] 工具执行成功: %v", result)

		resultJSON, _ := json.Marshal(result)
		newMessages = append(newMessages, Message{
			Role:       "tool",
			Content:    string(resultJSON),
			ToolCallID: tc.ID,
		})
	}

	log.Printf("[AI Service] Native function calling - 请求最终回复")
	var finalResp *ChatResponse
	if modelName != "" {
		finalResp, err = provider.WithModel(modelName).ChatWithTools(newMessages, toolDefs)
	} else {
		finalResp, err = provider.ChatWithTools(newMessages, toolDefs)
	}
	if err != nil {
		return "", err
	}

	return finalResp.Content, nil
}

// GetCompletionWithTools 注入全部 AI 工具进行 function calling（管理后台 AI 等用）。
func (s *AIService) GetCompletionWithTools(taskType TaskType, messages []Message, callerCtx *CallerContext, overrides ...Override) (string, error) {
	return s.getCompletionWithToolsCore(taskType, messages, callerCtx, nil, overrides...)
}

// GetCompletionWithToolsFiltered 只注入白名单工具进行 function calling（群聊助手用，
// 避免注入运维工具造成误调用）。
func (s *AIService) GetCompletionWithToolsFiltered(taskType TaskType, messages []Message, callerCtx *CallerContext, allowed []string, overrides ...Override) (string, error) {
	return s.getCompletionWithToolsCore(taskType, messages, callerCtx, allowed, overrides...)
}

// MaxReActSteps 多步推理最大循环轮数（防无限循环）。
const MaxReActSteps = 8

// ReActStepCallback 每步工具执行后的回调（可选），用于向调用方通报进度。
type ReActStepCallback func(step int, toolName string, args map[string]interface{}, result interface{}, err error)

// GetCompletionWithToolsMultiStep 多步 ReAct 循环：LLM 可连续调用工具直到不再发出 tool call
// 或达到 maxSteps 上限。每轮：LLM → tool calls → execute → 结果追加到 messages → 下一轮 LLM。
// 当 LLM 返回纯文本（无 tool call）时视为最终回答，循环结束。
// allowed 为空时注入全部工具；onStep 可为 nil。
func (s *AIService) GetCompletionWithToolsMultiStep(taskType TaskType, messages []Message, callerCtx *CallerContext, allowed []string, maxSteps int, onStep ReActStepCallback, overrides ...Override) (string, error) {
	s.mu.RLock()
	toolRegistry := s.toolRegistry
	s.mu.RUnlock()

	if toolRegistry == nil {
		return s.GetCompletion(taskType, messages, overrides...)
	}

	provider, modelName, err := s.selectProvider(taskType, overrides...)
	if err != nil {
		return "", err
	}

	// 构建工具定义
	tools := toolRegistry.ListTools()
	allowedSet := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		allowedSet[a] = true
	}
	toolDefs := make([]ToolDef, 0, len(tools))
	for _, tool := range tools {
		name := tool["name"].(string)
		if len(allowedSet) > 0 && !allowedSet[name] {
			continue
		}
		desc := tool["description"].(string)
		params := tool["parameters"].(map[string]interface{})
		toolDefs = append(toolDefs, ToolDef{Name: name, Description: desc, Parameters: params})
	}

	if maxSteps <= 0 {
		maxSteps = MaxReActSteps
	}

	// 工作副本，避免修改调用方原始 slice
	workMsgs := make([]Message, len(messages))
	copy(workMsgs, messages)

	callProvider := func(msgs []Message) (*ChatResponse, error) {
		if modelName != "" {
			return provider.WithModel(modelName).ChatWithTools(msgs, toolDefs)
		}
		return provider.ChatWithTools(msgs, toolDefs)
	}

	for step := 1; step <= maxSteps; step++ {
		resp, err := callProvider(workMsgs)
		if err != nil {
			// 首轮即失败时降级到 prompt engineering（与单轮行为一致）
			if step == 1 {
				log.Printf("[AI ReAct] Native function calling not supported, falling back: %v", err)
				return s.getCompletionWithToolsPromptEngineering(taskType, messages, callerCtx, allowed, overrides...)
			}
			return "", fmt.Errorf("react step %d provider error: %w", step, err)
		}

		// 无工具调用 → 最终回答
		if len(resp.ToolCalls) == 0 {
			log.Printf("[AI ReAct] 完成，共 %d 步", step-1)
			return resp.Content, nil
		}

		log.Printf("[AI ReAct] step=%d tool_calls=%d", step, len(resp.ToolCalls))

		// 追加 assistant 消息（含 tool_calls）
		workMsgs = append(workMsgs, Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// 逐个执行工具
		for _, tc := range resp.ToolCalls {
			if len(allowedSet) > 0 && !allowedSet[tc.Name] {
				execErr := fmt.Errorf("tool %s is not allowed", tc.Name)
				if onStep != nil {
					onStep(step, tc.Name, tc.Arguments, nil, execErr)
				}
				log.Printf("[AI ReAct] 工具执行失败: name=%s err=%v", tc.Name, execErr)
				errJSON, _ := json.Marshal(map[string]string{"error": execErr.Error()})
				workMsgs = append(workMsgs, Message{
					Role:       "tool",
					Content:    string(errJSON),
					ToolCallID: tc.ID,
				})
				continue
			}
			result, execErr := toolRegistry.ExecuteTool(tc.Name, tc.Arguments, callerCtx)
			if onStep != nil {
				onStep(step, tc.Name, tc.Arguments, result, execErr)
			}
			if execErr != nil {
				log.Printf("[AI ReAct] 工具执行失败: name=%s err=%v", tc.Name, execErr)
				// 将错误作为 tool 结果返回给 LLM，让它决定如何处理（而非直接中断）
				errJSON, _ := json.Marshal(map[string]string{"error": execErr.Error()})
				workMsgs = append(workMsgs, Message{
					Role:       "tool",
					Content:    string(errJSON),
					ToolCallID: tc.ID,
				})
				continue
			}
			resultJSON, _ := json.Marshal(result)
			workMsgs = append(workMsgs, Message{
				Role:       "tool",
				Content:    string(resultJSON),
				ToolCallID: tc.ID,
			})
		}
	}

	// 达到最大步数仍未结束，做最后一次无工具调用获取总结
	log.Printf("[AI ReAct] 达到最大步数 %d，请求最终总结", maxSteps)
	finalResp, err := callProvider(workMsgs)
	if err != nil {
		return "", fmt.Errorf("react final summary error: %w", err)
	}
	return finalResp.Content, nil
}

func (s *AIService) getCompletionWithToolsPromptEngineering(taskType TaskType, messages []Message, callerCtx *CallerContext, allowed []string, overrides ...Override) (string, error) {
	s.mu.RLock()
	toolRegistry := s.toolRegistry
	s.mu.RUnlock()

	if toolRegistry == nil {
		return s.GetCompletion(taskType, messages, overrides...)
	}

	tools := toolRegistry.ListTools()
	allowedSet := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		allowedSet[a] = true
	}
	filteredTools := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		name := tool["name"].(string)
		if len(allowedSet) > 0 && !allowedSet[name] {
			continue
		}
		filteredTools = append(filteredTools, tool)
	}

	toolsDesc := "你可以使用以下工具（如果用户请求涉及管理操作，请使用工具）：\n\n"
	for _, tool := range filteredTools {
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
{"tool_call": {"name": "工具名称", "arguments": {"参数名": "参数值"}}}

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

	log.Printf("[AI Service] 工具调用 - 发送请求到 AI，工具数: %d", len(filteredTools))
	reply, err := s.GetCompletion(taskType, newMessages, overrides...)
	if err != nil {
		log.Printf("[AI Service] 工具调用 - AI 请求失败: %v", err)
		return "", err
	}
	log.Printf("[AI Service] 工具调用 - AI 回复: %s", reply[:min(200, len(reply))])

	toolCall, err := parseToolCall(reply)
	if err != nil || toolCall == nil {
		log.Printf("[AI Service] 工具调用 - 未检测到工具调用")
		return reply, nil
	}

	log.Printf("[AI Service] 工具调用 - 检测到工具调用: name=%s, args=%v", toolCall.Name, toolCall.Arguments)
	if len(allowedSet) > 0 && !allowedSet[toolCall.Name] {
		return "", fmt.Errorf("tool %s is not allowed", toolCall.Name)
	}
	result, err := toolRegistry.ExecuteTool(toolCall.Name, toolCall.Arguments, callerCtx)
	if err != nil {
		log.Printf("[AI Service] 工具执行失败: %v", err)
		return "", err
	}

	newMessages = append(newMessages, Message{Role: "assistant", Content: reply})
	resultJSON, _ := json.Marshal(result)
	newMessages = append(newMessages, Message{Role: "user", Content: fmt.Sprintf("工具 %s 执行结果: %s\n请根据这个结果生成给用户的回复。", toolCall.Name, string(resultJSON))})

	finalReply, err := s.GetCompletion(taskType, newMessages, overrides...)
	if err != nil {
		return "", err
	}

	return finalReply, nil
}

type toolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

func parseToolCall(reply string) (*toolCall, error) {
	idx := strings.Index(reply, "{")
	if idx == -1 {
		return nil, nil
	}

	jsonStr := reply[idx:]
	if endIdx := strings.LastIndex(jsonStr, "}"); endIdx >= 0 {
		jsonStr = jsonStr[:endIdx+1]
	}

	var result struct {
		ToolCall *toolCall `json:"tool_call"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, nil
	}

	if result.ToolCall == nil || result.ToolCall.Name == "" {
		return nil, nil
	}

	return result.ToolCall, nil
}

func (s *AIService) UpdateConfig(cfg *AIConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = cfg
	for name, providerCfg := range cfg.AllProviders() {
		provider, err := s.factory.CreateProviderByName(name, providerCfg)
		if err != nil {
			log.Printf("[AI Service] Failed to update provider %s: %v", name, err)
			continue
		}
		s.pool[name] = provider
	}
	s.router = NewModelRouter(cfg.Router)
}

// DBProviderInfo 数据库 Provider 的纯数据描述，避免 ai 包依赖 model 包。
type DBProviderInfo struct {
	ID       uint
	Name     string // 唯一标识（用于 pool key）
	APIType  string // openai / anthropic / custom 等
	Endpoint string // BaseURL
	APIKey   string
	Models   []string // 支持的模型列表
	Enabled  bool
	Priority int
}

// ReloadProvidersFromDB 从数据库 Provider 列表重新加载 pool。
// 已启用的 Provider 会被加入 pool；未启用或加载失败的会被跳过。
// 语义：DB 为覆盖性数据源——当 DB 有已启用 Provider 时完全替换 pool；
// 当 DB 中没有已启用 Provider 时，保留现有 pool（通常是 config.yaml 配置）作为兜底。
func (s *AIService) ReloadProvidersFromDB(providers []DBProviderInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()

	newPool := make(map[string]Provider)
	for _, p := range providers {
		if !p.Enabled || p.APIKey == "" {
			continue
		}
		// 选择第一个模型作为默认模型
		model := ""
		if len(p.Models) > 0 {
			model = p.Models[0]
		}
		providerCfg := ProviderConfig{
			APIKey:  p.APIKey,
			Model:   model,
			BaseURL: p.Endpoint,
		}
		provider, err := s.factory.CreateProviderByName(p.APIType, providerCfg)
		if err != nil {
			log.Printf("[AI Service] Failed to load DB provider %s (type=%s): %v", p.Name, p.APIType, err)
			continue
		}
		// 使用小写 name 作为 pool key，与 config.yaml 中的 provider 命名保持一致
		poolKey := strings.ToLower(p.Name)
		newPool[poolKey] = provider
		log.Printf("[AI Service] DB provider %s loaded as %q (type=%s, model=%s)", p.Name, poolKey, p.APIType, model)
	}

	if len(newPool) > 0 {
		s.pool = newPool
		log.Printf("[AI Service] Reloaded %d providers from DB", len(newPool))
	} else {
		// DB 中没有已启用的 Provider，回退到 config.yaml 初始化的 pool
		s.pool = make(map[string]Provider)
		for k, v := range s.configPool {
			s.pool[k] = v
		}
		log.Printf("[AI Service] No enabled DB providers, restored config.yaml pool (%d)", len(s.pool))
	}
}

func (s *AIService) GetConfig() *AIConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *AIService) filterMessages(messages []Message) []Message {
	filtered := make([]Message, len(messages))
	for i, msg := range messages {
		filtered[i] = Message{
			Role:     msg.Role,
			Content:  s.filterContent(msg.Content),
			ImageURL: msg.ImageURL,
		}
	}
	return filtered
}

func (s *AIService) filterContent(content string) string {
	const maxRunes = 10000
	runes := []rune(content)
	if len(runes) <= maxRunes {
		return content
	}
	// 尝试在 maxRunes 范围内的最后一个句子边界处截断
	cutAt := maxRunes
	for i := maxRunes - 1; i >= 0; i-- {
		switch runes[i] {
		case '。', '.', '!', '！', '？', '?', '\n':
			cutAt = i + 1
			break
		}
		if cutAt != maxRunes {
			break
		}
	}
	return string(runes[:cutAt]) + "\n...(内容已截断)"
}

func (s *AIService) IsConfigured() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.pool) > 0
}

func (s *AIService) Embed(text string) ([]float32, error) {
	provider, _, err := s.selectProvider(TaskTypeEmbedding)
	if err != nil {
		return nil, err
	}
	return provider.Embedding(text)
}
