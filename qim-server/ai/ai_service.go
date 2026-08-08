package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

// aiLog 把 AI 日志路由到 module="AI" 的目标（ai.log），同时仍打到 qim.log/stdout。
// 模块名沿用 ai 包既有约定（ai/provider.go 同样用 WithModule("AI")）。
var aiLog = logger.WithModule("AI")

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
			aiLog.Error("provider init failed", "name", name, "error", err)
			continue
		}
		svc.pool[name] = provider
		svc.configPool[name] = provider
		aiLog.Info("provider initialized", "name", name)
	}

	if len(svc.pool) == 0 {
		aiLog.Warn("no AI providers initialized")
	} else {
		aiLog.Info("AI providers initialized", "count", len(svc.pool))
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

// HasVisionRoute 报告是否显式配置了可用的视觉（多模态）任务路由。
// 未配置时 TaskTypeVision 会回退到 defaultTask（纯文本 chat 模型），
// 把图片 base64 发给它必然 400。调用方（如群 AI 引用图片路径）据此决定
// 是走多模态识别，还是把被引用图片降级为"当前模型不支持看图"的提示语。
func (s *AIService) HasVisionRoute() bool {
	s.mu.RLock()
	router := s.router
	s.mu.RUnlock()
	if router == nil {
		return false
	}
	return router.HasExplicitRoute(TaskTypeVision)
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
	// 与 GetCompletionStreamWithContext 一致：截获到 usage 时上报。
	status := "success"
	if err != nil {
		status = "error"
	}
	aiLog.Info("ai usage",
		"task", taskType, "provider", provider.Name(), "model", modelName,
		"durationMs", duration, "status", status)

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
	aiLog.Info("ai usage",
		"task", taskType, "provider", provider.Name(), "model", modelName,
		"durationMs", duration, "status", status)
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

	aiLog.Info("try native function calling", "tools", len(toolDefs))
	var resp *ChatResponse
	if modelName != "" {
		resp, err = provider.WithModel(modelName).ChatWithTools(messages, toolDefs)
	} else {
		resp, err = provider.ChatWithTools(messages, toolDefs)
	}
	if err != nil {
		aiLog.Warn("native function calling not supported, falling back to prompt engineering", "error", err)
		return s.getCompletionWithToolsPromptEngineering(taskType, messages, callerCtx, allowed, overrides...)
	}

	if len(resp.ToolCalls) == 0 {
		aiLog.Info("native function calling - no tool call, direct reply")
		return resp.Content, nil
	}

	aiLog.Info("native function calling detected tool calls", "count", len(resp.ToolCalls))

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
		aiLog.Info("executing tool", "name", tc.Name, "args", tc.Arguments)
		result, execErr := toolRegistry.ExecuteTool(tc.Name, tc.Arguments, callerCtx)
		if execErr != nil {
			aiLog.Error("tool execution failed", "name", tc.Name, "error", execErr)
			return "", execErr
		}
		aiLog.Info("tool execution succeeded", "name", tc.Name, "result", result)

		resultJSON, _ := json.Marshal(result)
		newMessages = append(newMessages, Message{
			Role:       "tool",
			Content:    string(resultJSON),
			ToolCallID: tc.ID,
		})
	}

	aiLog.Info("native function calling - requesting final reply")
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

// ReActStepCallback 每步工具调用的进度回调（可选），用于向调用方通报进度。
// 同一工具调用会触发两次：phase="start" 在工具执行前（用于实时「正在调用」反馈），
// phase="end" 在工具执行后（携带结果/错误）。toolCallID 在同一调用的 start/end 上一致，
// 供前端按 ID 把进行态更新为终态而非重复追加。toolCallID 通常取 LLM 返回的 tc.ID。
type ReActStepCallback func(step int, toolCallID string, phase string, toolName string, args map[string]interface{}, result interface{}, err error)

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
				aiLog.Warn("react native function calling not supported, falling back", "error", err)
				return s.getCompletionWithToolsPromptEngineering(taskType, messages, callerCtx, allowed, overrides...)
			}
			return "", fmt.Errorf("react step %d provider error: %w", step, err)
		}

		// 无工具调用 → 最终回答
		if len(resp.ToolCalls) == 0 {
			aiLog.Info("react completed", "steps", step-1)
			return resp.Content, nil
		}

		aiLog.Info("react step", "step", step, "toolCalls", len(resp.ToolCalls))

		// 追加 assistant 消息（含 tool_calls）
		workMsgs = append(workMsgs, Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// 逐个执行工具；每个调用先推 start（进行态）再推 end（终态），callID 供前端
		// 按 ID 把进行态更新为终态（而非重复追加）。
		for i, tc := range resp.ToolCalls {
			callID := tc.ID
			if callID == "" {
				callID = fmt.Sprintf("step%d_%d", step, i)
			}
			if len(allowedSet) > 0 && !allowedSet[tc.Name] {
				execErr := fmt.Errorf("tool %s is not allowed", tc.Name)
				if onStep != nil {
					onStep(step, callID, "end", tc.Name, tc.Arguments, nil, execErr)
				}
				aiLog.Warn("react tool execution failed", "name", tc.Name, "error", execErr)
				errJSON, _ := json.Marshal(map[string]string{"error": execErr.Error()})
				workMsgs = append(workMsgs, Message{
					Role:       "tool",
					Content:    string(errJSON),
					ToolCallID: tc.ID,
				})
				continue
			}
			if onStep != nil {
				onStep(step, callID, "start", tc.Name, tc.Arguments, nil, nil)
			}
			result, execErr := toolRegistry.ExecuteTool(tc.Name, tc.Arguments, callerCtx)
			if onStep != nil {
				onStep(step, callID, "end", tc.Name, tc.Arguments, result, execErr)
			}
			if execErr != nil {
				aiLog.Warn("react tool execution failed", "name", tc.Name, "error", execErr)
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
	aiLog.Info("react reached max steps, requesting final summary", "maxSteps", maxSteps)
	finalResp, err := callProvider(workMsgs)
	if err != nil {
		return "", fmt.Errorf("react final summary error: %w", err)
	}
	return finalResp.Content, nil
}

// toolCallAccumulator 流式回合内按 Index 累积的一条 tool call（跨 chunk 拼接原始
// arguments JSON 字符串）。回合终了把拼接好的 arguments 整体 unmarshal 成 map。
type toolCallAccumulator struct {
	ID       string
	Name     string
	ArgStr   strings.Builder
}

// GetCompletionWithToolsStreamMultiStep 真·流式 ReAct：与 GetCompletionWithToolsMultiStep
// 同构（LLM 可连续调工具直到不再发 tool call），区别在于每回合这次 LLM 调用走流式
// （ChatStreamWithTools），从而：
//   - final 回合（模型不再调工具→纯文本）→ 内容 delta **逐 token 实时**交给 onChunk =
//     真·逐字打字；
//   - 工具回合 → 工具调用以 ToolCallDelta 增量累积成完整 ToolCall 后执行，工具事件仍经
//     onStep 推（start/end），与现役工具卡片完全兼容。
//
// 工具调用是需完整参数才能执行的离散事件，故工具回合前的前置正文（模型推理过程）会被
// 实时送出——主流 OpenAI 兼容模型在调用工具前不输出正文，此场景极少；当其为最终答案时会
// 即时逐 token 呈现（正是目标）。首回合若 Provider 不支持流式 tool-call，返回
// ErrStreamingToolsNotSupported，调用方据此降级到非流式 GetCompletionWithToolsMultiStep。
// onChunk 收到 final 回合的内容 delta；调用返回 nil 表示 final 内容已全部流出。
func (s *AIService) GetCompletionWithToolsStreamMultiStep(taskType TaskType, messages []Message, callerCtx *CallerContext, allowed []string, maxSteps int, onStep ReActStepCallback, onChunk func(chunk StreamChunk) error, overrides ...Override) error {
	s.mu.RLock()
	toolRegistry := s.toolRegistry
	s.mu.RUnlock()

	if toolRegistry == nil {
		return s.GetCompletionStream(taskType, messages, onChunk, overrides...)
	}

	provider, modelName, err := s.selectProvider(taskType, overrides...)
	if err != nil {
		return err
	}

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

	workMsgs := make([]Message, len(messages))
	copy(workMsgs, messages)

	// callProviderStream 执行本轮流式调用，onChunk 侧实时累积内容与工具增量。
	callProviderStream := func(msgs []Message, onChunk func(chunk StreamChunk) error) error {
		if modelName != "" {
			return provider.WithModel(modelName).ChatStreamWithTools(context.Background(), msgs, toolDefs, onChunk)
		}
		return provider.ChatStreamWithTools(context.Background(), msgs, toolDefs, onChunk)
	}

	for step := 1; step <= maxSteps; step++ {
		var bufContent strings.Builder
		var toolCallsByIndex = make(map[int]*toolCallAccumulator)

		err := callProviderStream(workMsgs, func(chunk StreamChunk) error {
			if chunk.Content != "" {
				bufContent.WriteString(chunk.Content)
				// final 回合实时转发：逐 token 打字。工具回合罕见的前置正文亦会流出（见函数注释取舍）。
				if chunkErr := onChunk(StreamChunk{Content: chunk.Content}); chunkErr != nil {
					return chunkErr
				}
			}
			for _, tc := range chunk.ToolCalls {
				acc := toolCallsByIndex[tc.Index]
				if acc == nil {
					acc = &toolCallAccumulator{}
					toolCallsByIndex[tc.Index] = acc
				}
				if tc.ID != "" {
					acc.ID = tc.ID
				}
				if tc.Name != "" {
					acc.Name = tc.Name
				}
				acc.ArgStr.WriteString(tc.Arguments)
			}
			return nil
		})
		if err != nil {
			// 首回合即不支持流式 tool-call → 让调用方降级到非流式路径
			if step == 1 && errors.Is(err, ErrStreamingToolsNotSupported) {
				return ErrStreamingToolsNotSupported
			}
			return fmt.Errorf("react stream step %d provider error: %w", step, err)
		}

		// 组装本回合 tool calls（arguments 是跨 chunk 拼接的原始 JSON，回合终了整体解析）
		toolCalls := make([]ToolCall, 0, len(toolCallsByIndex))
		for idx := 0; idx < len(toolCallsByIndex); idx++ {
			acc := toolCallsByIndex[idx]
			if acc == nil {
				continue
			}
			var args map[string]interface{}
			// 解析失败以空 map 兜底，避免 nil 序列化问题
			if err := json.Unmarshal([]byte(acc.ArgStr.String()), &args); err != nil {
				args = map[string]interface{}{}
			}
			callID := acc.ID
			if callID == "" {
				callID = fmt.Sprintf("step%d_%d", step, idx)
			}
			toolCalls = append(toolCalls, ToolCall{ID: callID, Name: acc.Name, Arguments: args})
		}

		// 无工具调用 → final 回合：内容已逐 token 流出，结束
		if len(toolCalls) == 0 {
			aiLog.Info("react(stream) completed", "steps", step-1)
			return nil
		}

		aiLog.Info("react(stream) step", "step", step, "toolCalls", len(toolCalls))

		// 追加 assistant 消息（含 tool_calls 与实时流出的前置正文，供模型上下文一致）
		workMsgs = append(workMsgs, Message{
			Role:      "assistant",
			Content:   bufContent.String(),
			ToolCalls: toolCalls,
		})

		for _, tc := range toolCalls {
			if len(allowedSet) > 0 && !allowedSet[tc.Name] {
				execErr := fmt.Errorf("tool %s is not allowed", tc.Name)
				if onStep != nil {
					onStep(step, tc.ID, "end", tc.Name, tc.Arguments, nil, execErr)
				}
				aiLog.Warn("react(stream) tool not allowed", "name", tc.Name)
				errJSON, _ := json.Marshal(map[string]string{"error": execErr.Error()})
				workMsgs = append(workMsgs, Message{Role: "tool", Content: string(errJSON), ToolCallID: tc.ID})
				continue
			}
			if onStep != nil {
				onStep(step, tc.ID, "start", tc.Name, tc.Arguments, nil, nil)
			}
			result, execErr := toolRegistry.ExecuteTool(tc.Name, tc.Arguments, callerCtx)
			if onStep != nil {
				onStep(step, tc.ID, "end", tc.Name, tc.Arguments, result, execErr)
			}
			if execErr != nil {
				aiLog.Warn("react(stream) tool execution failed", "name", tc.Name, "error", execErr)
				errJSON, _ := json.Marshal(map[string]string{"error": execErr.Error()})
				workMsgs = append(workMsgs, Message{Role: "tool", Content: string(errJSON), ToolCallID: tc.ID})
				continue
			}
			resultJSON, _ := json.Marshal(result)
			workMsgs = append(workMsgs, Message{Role: "tool", Content: string(resultJSON), ToolCallID: tc.ID})
		}
	}

	// 达到最大步数仍未结束：做最后一次无工具调用获取总结并逐 token 流出
	aiLog.Info("react(stream) reached max steps, requesting final summary", "maxSteps", maxSteps)
	return s.GetCompletionStreamWithContext(context.Background(), taskType, workMsgs, onChunk, overrides...)
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

	aiLog.Info("tool prompt - sending request to AI", "tools", len(filteredTools))
	reply, err := s.GetCompletion(taskType, newMessages, overrides...)
	if err != nil {
		aiLog.Error("tool prompt - AI request failed", "error", err)
		return "", err
	}
	aiLog.Info("tool prompt - AI reply", "reply", reply[:min(200, len(reply))])

	toolCall, err := parseToolCall(reply)
	if err != nil || toolCall == nil {
		aiLog.Info("tool prompt - no tool call detected")
		return reply, nil
	}

	aiLog.Info("tool prompt - tool call detected", "name", toolCall.Name, "args", toolCall.Arguments)
	if len(allowedSet) > 0 && !allowedSet[toolCall.Name] {
		return "", fmt.Errorf("tool %s is not allowed", toolCall.Name)
	}
	result, err := toolRegistry.ExecuteTool(toolCall.Name, toolCall.Arguments, callerCtx)
	if err != nil {
		aiLog.Error("tool execution failed", "name", toolCall.Name, "error", err)
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
			aiLog.Error("failed to update provider", "name", name, "error", err)
			continue
		}
		s.pool[name] = provider
	}
	s.router = NewModelRouter(cfg.Router)
}

// UpdateRouter 用给定 RouterConfig 重建运行时 router（不重建 provider pool、不触碰 config.yaml）。
// 用于管理后台「AI 模型路由」保存后热更，无需重启。
// 注意：仅替换路由表，provider pool 仍由 config.yaml / DB 供应商管理。
func (s *AIService) UpdateRouter(rc RouterConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.config == nil {
		s.config = &AIConfig{}
	}
	s.config.Router = rc
	s.router = NewModelRouter(rc)
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
			aiLog.Error("failed to load DB provider", "name", p.Name, "type", p.APIType, "error", err)
			continue
		}
		// 使用小写 name 作为 pool key，与 config.yaml 中的 provider 命名保持一致
		poolKey := strings.ToLower(p.Name)
		newPool[poolKey] = provider
		aiLog.Info("DB provider loaded", "name", p.Name, "poolKey", poolKey, "type", p.APIType, "model", model)
	}

	if len(newPool) > 0 {
		s.pool = newPool
		aiLog.Info("reloaded providers from DB", "count", len(newPool))
	} else {
		// DB 中没有已启用的 Provider，回退到 config.yaml 初始化的 pool
		s.pool = make(map[string]Provider)
		for k, v := range s.configPool {
			s.pool[k] = v
		}
		aiLog.Warn("no enabled DB providers, restored config.yaml pool", "count", len(s.pool))
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

// GetCompletionWithProviderConfig 用显式传入的 ProviderConfig 临时创建 provider 完成一次非流式对话，
// 而不改动全局 pool。用于"用户自定义模型配置"（如分身自选模型、bot 自选模型）这类 per-call 凭据，
// 避免把每个用户的 API key 永久注册进共享 pool 造成生命周期/串味问题。
// name 为 provider 类型名（openai/anthropic/alibaba 等，见 ProviderFactory.CreateProviderByName）。
// 注：非流式 Chat 不可取消，但各 provider 的底层 http.Client 已设超时（BaseProvider 120s、
// anthropic 60s），不会永久挂起，故此处不需要额外 goroutine+select 包装——保持线性、可被调用方
// 直接 await，避免为不存在的"无限挂起"引入并发与孤儿 goroutine。
func (s *AIService) GetCompletionWithProviderConfig(taskType TaskType, messages []Message, name string, cfg ProviderConfig) (string, error) {
	provider, err := s.factory.CreateProviderByName(name, cfg)
	if err != nil {
		return "", fmt.Errorf("create custom provider %s: %w", name, err)
	}
	if !provider.IsConfigured() {
		return "", fmt.Errorf("custom provider %s not configured", name)
	}

	filtered := s.filterMessages(messages)
	start := time.Now()
	var result string
	var usage *TokenUsage
	if up, ok := provider.(UsageProvider); ok {
		if cfg.Model != "" {
			if upm, ok2 := provider.WithModel(cfg.Model).(UsageProvider); ok2 {
				result, usage, err = upm.ChatWithUsage(filtered)
			} else {
				result, err = provider.WithModel(cfg.Model).Chat(filtered)
			}
		} else {
			result, usage, err = up.ChatWithUsage(filtered)
		}
	} else {
		if cfg.Model != "" {
			result, err = provider.WithModel(cfg.Model).Chat(filtered)
		} else {
			result, err = provider.Chat(filtered)
		}
	}

	duration := time.Since(start).Milliseconds()
	status := "success"
	if err != nil {
		status = "error"
	}
	aiLog.Info("ai usage",
		"task", taskType, "provider", provider.Name(), "model", cfg.Model,
		"source", "custom", "durationMs", duration, "status", status)

	if usage != nil && s.usageSink != nil {
		s.usageSink(taskType, provider.Name(), cfg.Model, usage, duration)
	}
	return result, err
}

// ChatStreamWithProviderConfig 流式版本的 GetCompletionWithProviderConfig（供 ExecuteStream/草稿模式使用）。
func (s *AIService) ChatStreamWithProviderConfig(ctx context.Context, taskType TaskType, messages []Message, name string, cfg ProviderConfig, onChunk func(chunk StreamChunk) error) error {
	provider, err := s.factory.CreateProviderByName(name, cfg)
	if err != nil {
		return fmt.Errorf("create custom provider %s: %w", name, err)
	}
	if !provider.IsConfigured() {
		return fmt.Errorf("custom provider %s not configured", name)
	}

	filtered := s.filterMessages(messages)
	// 包装 onChunk 截获 Usage。流式响应中 usage 通常在最后一个 chunk 携带，
	// 取最后一个非 nil 的 Usage（部分 provider 在 finish chunk 重复发送累计 usage）。
	var capturedUsage *StreamUsage
	wrappedOnChunk := func(chunk StreamChunk) error {
		if chunk.Usage != nil {
			capturedUsage = chunk.Usage
		}
		return onChunk(chunk)
	}

	start := time.Now()
	if cfg.Model != "" {
		err = provider.WithModel(cfg.Model).ChatStreamWithContext(ctx, filtered, wrappedOnChunk)
	} else {
		err = provider.ChatStreamWithContext(ctx, filtered, wrappedOnChunk)
	}
	duration := time.Since(start).Milliseconds()
	status := "success"
	if err != nil {
		status = "error"
	}
	aiLog.Info("ai usage",
		"task", taskType, "provider", provider.Name(), "model", cfg.Model,
		"source", "custom", "durationMs", duration, "status", status)

	// 与 GetCompletionWithProviderConfig 一致：截获到 usage 时上报。
	if capturedUsage != nil && s.usageSink != nil {
		s.usageSink(taskType, provider.Name(), cfg.Model, &TokenUsage{
			PromptTokens:     capturedUsage.PromptTokens,
			CompletionTokens: capturedUsage.CompletionTokens,
			TotalTokens:      capturedUsage.TotalTokens,
		}, duration)
	}
	return err
}
