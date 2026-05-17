package ai

import (
	"encoding/json"
	"fmt"
	"qim-server/pkg/logger"
	"strings"
	"sync"
)

type AIService struct {
	config    *AIConfig
	factory   *ProviderFactory
	provider  Provider
	mu        sync.RWMutex
	mcpServer *MCPServer
}

func NewAIService(cfg *AIConfig) *AIService {
	svc := &AIService{
		config:  cfg,
		factory: NewProviderFactory(),
	}

	if err := svc.updateProvider(cfg); err != nil {
		logger.WithModule("AI").Warn("Warning: Failed to initialize provider", "error", err)
	}

	return svc
}

// SetMCPServer 设置 MCP 服务器用于工具调用
func (s *AIService) SetMCPServer(mcpServer *MCPServer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mcpServer = mcpServer
}

// GetMCPServer 获取 MCP 服务器
func (s *AIService) GetMCPServer() *MCPServer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mcpServer
}

// GetCompletionWithTools 带工具调用的 AI 完成
func (s *AIService) GetCompletionWithTools(messages []Message, callerCtx *CallerContext) (reply string, err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.WithModule("AI").Error("PANIC in GetCompletionWithTools", "panic", r)
			reply = ""
			err = fmt.Errorf("panic in GetCompletionWithTools: %v", r)
		}
	}()

	s.mu.RLock()
	mcpServer := s.mcpServer
	provider := s.provider
	s.mu.RUnlock()

	if mcpServer == nil {
		return s.GetCompletion(messages)
	}

	// 构建工具定义列表
	tools := mcpServer.ListTools()
	toolDefs := make([]ToolDef, 0, len(tools))

	// 如果有 AllowedTools 限制，只使用指定的工具
	allowedMap := make(map[string]bool)
	for _, name := range callerCtx.AllowedTools {
		allowedMap[name] = true
	}
	filterTools := len(callerCtx.AllowedTools) > 0

	for _, tool := range tools {
		name := tool["name"].(string)
		if filterTools && !allowedMap[name] {
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

	// 尝试使用 native function calling
	logger.WithModule("AI").Info("尝试使用 native function calling", "toolCount", len(toolDefs), "totalTools", len(tools))
	resp, err := provider.ChatWithTools(messages, toolDefs)
	if err != nil {
		logger.WithModule("AI").Error("ChatWithTools 失败", "error", err)
		// 降级到 prompt engineering
		return s.getCompletionWithToolsPromptEngineering(messages, callerCtx)
	}
	logger.WithModule("AI").Info("ChatWithTools 返回", "content", resp.Content, "toolCalls", len(resp.ToolCalls))

	// 如果没有工具调用，直接返回
	if len(resp.ToolCalls) == 0 {
		logger.WithModule("AI").Info("Native function calling - 无工具调用，直接返回回复")
		return resp.Content, nil
	}

	logger.WithModule("AI").Info("Native function calling - 检测到工具调用", "count", len(resp.ToolCalls))

	// 复制消息历史用于后续调用
	newMessages := make([]Message, len(messages))
	copy(newMessages, messages)

	// 追加 assistant 消息（包含工具调用）
	newMessages = append(newMessages, Message{
		Role:      "assistant",
		Content:   resp.Content,
		ToolCalls: resp.ToolCalls,
	})

	// 执行所有工具调用
	for _, tc := range resp.ToolCalls {
		logger.WithModule("AI").Info("执行工具", "name", tc.Name, "args", tc.Arguments)
		// 参数为空时跳过，避免因空参数导致执行失败
		if len(tc.Arguments) == 0 {
			logger.WithModule("AI").Info("工具参数为空，跳过", "name", tc.Name)
			continue
		}
		result, execErr := mcpServer.ExecuteTool(tc.Name, tc.Arguments, callerCtx)
		if execErr != nil {
			logger.WithModule("AI").Error("工具执行失败", "error", execErr)
			return "", execErr
		}
		logger.WithModule("AI").Info("工具执行成功", "result", result)

		// 追加工具结果消息
		resultJSON, _ := json.Marshal(result)
		newMessages = append(newMessages, Message{
			Role:       "tool",
			Content:    string(resultJSON),
			ToolCallID: tc.ID,
			Name:       tc.Name,
		})
	}

	// 再次调用 LLM 生成最终回复，不再提供工具，强制 LLM 生成文本内容
	logger.WithModule("AI").Info("Native function calling - 请求最终回复", "messageCount", len(newMessages))
	finalResp, err := provider.ChatWithTools(newMessages, nil)
	if err != nil {
		logger.WithModule("AI").Error("Native function calling - 最终回复失败", "error", err)
		return "", err
	}
	logger.WithModule("AI").Info("Native function calling - 最终回复成功", "content", finalResp.Content[:min(200, len(finalResp.Content))])

	return finalResp.Content, nil
}

// getCompletionWithToolsPromptEngineering 使用 prompt engineering 方式实现工具调用（降级方案）
func (s *AIService) getCompletionWithToolsPromptEngineering(messages []Message, callerCtx *CallerContext) (string, error) {
	s.mu.RLock()
	mcpServer := s.mcpServer
	s.mu.RUnlock()

	if mcpServer == nil {
		return s.GetCompletion(messages)
	}

	// 构建工具列表描述
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

	// 添加使用说明到系统消息
	toolInstruction := toolsDesc + `如需调用工具，请严格按照以下 JSON 格式返回：
{"tool_call": {"name": "工具名称", "arguments": {"参数名": "参数值"}}}

如果不需要调用工具，直接输出回复内容。
注意：只在用户明确要求执行管理操作时才调用工具，普通聊天不要调用工具。`

	// 将工具说明注入到 system 消息中
	var newMessages []Message
	for _, msg := range messages {
		if msg.Role == "system" {
			newMessages = append(newMessages, Message{Role: "system", Content: msg.Content + "\n\n" + toolInstruction})
		} else {
			newMessages = append(newMessages, msg)
		}
	}

	// 第一次调用 AI
	logger.WithModule("AI").Info("工具调用 - 发送请求到 AI", "toolCount", len(tools))
	reply, err := s.GetCompletion(newMessages)
	if err != nil {
		logger.WithModule("AI").Error("工具调用 - AI 请求失败", "error", err)
		return "", err
	}
	logger.WithModule("AI").Info("工具调用 - AI 回复", "reply", reply[:min(200, len(reply))])

	// 检查是否包含工具调用
	toolCall, err := parseToolCall(reply)
	if err != nil || toolCall == nil {
		logger.WithModule("AI").Info("工具调用 - 未检测到工具调用")
		// 没有工具调用，直接返回回复
		return reply, nil
	}

	logger.WithModule("AI").Info("工具调用 - 检测到工具调用", "name", toolCall.Name, "args", toolCall.Arguments)

	// 执行工具
	result, err := mcpServer.ExecuteTool(toolCall.Name, toolCall.Arguments, callerCtx)
	if err != nil {
		logger.WithModule("AI").Error("工具执行失败", "error", err)
		return "", err
	}
	logger.WithModule("AI").Info("工具执行成功", "result", result)

	// 将工具结果追加到消息列表
	newMessages = append(newMessages, Message{Role: "assistant", Content: reply})
	resultJSON, _ := json.Marshal(result)
	newMessages = append(newMessages, Message{Role: "user", Content: fmt.Sprintf("工具 %s 执行结果: %s\n请根据这个结果生成给用户的回复。", toolCall.Name, string(resultJSON))})

	// 第二次调用 AI，基于工具结果生成最终回复
	finalReply, err := s.GetCompletion(newMessages)
	if err != nil {
		return "", err
	}

	return finalReply, nil
}

// toolCall 工具调用（内部使用）
type toolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// parseToolCall 从 AI 回复中解析工具调用
func parseToolCall(reply string) (*toolCall, error) {
	// 尝试提取 JSON
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
		return nil, nil // 不是有效的 JSON，当作普通回复
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

	if err := s.updateProvider(cfg); err != nil {
		logger.WithModule("AI").Error("Failed to update provider", "error", err)
	}
}

func (s *AIService) updateProvider(cfg *AIConfig) error {
	provider, err := s.factory.CreateProvider(cfg)
	if err != nil {
		return err
	}
	s.provider = provider
	return nil
}

func (s *AIService) GetConfig() *AIConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *AIService) GetCompletion(messages []Message) (string, error) {
	s.mu.RLock()
	provider := s.provider
	s.mu.RUnlock()

	if provider == nil {
		return "", fmt.Errorf("AI provider not initialized")
	}

	filteredMessages := s.filterMessages(messages)
	return provider.Chat(filteredMessages)
}

// GetCompletionStream 流式获取AI完成
func (s *AIService) GetCompletionStream(messages []Message, onChunk func(chunk StreamChunk) error) error {
	s.mu.RLock()
	provider := s.provider
	s.mu.RUnlock()

	if provider == nil {
		return fmt.Errorf("AI provider not initialized")
	}

	filteredMessages := s.filterMessages(messages)
	return provider.ChatStream(filteredMessages, onChunk)
}

// 过滤消息内容，防止恶意输入
func (s *AIService) filterMessages(messages []Message) []Message {
	filtered := make([]Message, len(messages))
	for i, msg := range messages {
		filtered[i] = Message{
			Role:    msg.Role,
			Content: s.filterContent(msg.Content),
		}
	}
	return filtered
}

// 过滤内容，移除潜在的恶意内容
func (s *AIService) filterContent(content string) string {
	if len(content) > 10000 {
		content = content[:10000] + "..."
	}
	return content
}

func (s *AIService) IsConfigured() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.provider == nil {
		return false
	}

	return s.provider.IsConfigured()
}

// Embed 将文本转换为向量（使用当前 Provider 的 Embedding 能力）
func (s *AIService) Embed(text string) ([]float32, error) {
	s.mu.RLock()
	provider := s.provider
	s.mu.RUnlock()

	if provider == nil {
		return nil, fmt.Errorf("AI provider not initialized")
	}

	return provider.Embedding(text)
}
