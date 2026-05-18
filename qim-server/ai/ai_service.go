package ai

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
)

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
		log.Printf("[AI Service] %d AI providers initialized", len(svc.pool))
	}

	return svc
}

func (s *AIService) SetMCPServer(mcpServer *MCPServer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mcpServer = mcpServer
}

func (s *AIService) GetMCPServer() *MCPServer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mcpServer
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

	tools := mcpServer.ListTools()
	toolDefs := make([]ToolDef, 0, len(tools))
	for _, tool := range tools {
		name := tool["name"].(string)
		desc := tool["description"].(string)
		params := tool["parameters"].(map[string]interface{})
		toolDefs = append(toolDefs, ToolDef{
			Name:        name,
			Description: desc,
			Parameters:  params,
		})
	}

	log.Printf("[AI Service] 尝试使用 native function calling，工具数: %d", len(toolDefs))
	resp, err := provider.ChatWithTools(messages, toolDefs)
	if err != nil {
		log.Printf("[AI Service] Native function calling not supported, falling back to prompt engineering: %v", err)
		return s.getCompletionWithToolsPromptEngineering(taskType, messages, callerCtx)
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
		log.Printf("[AI Service] 执行工具: name=%s, args=%v", tc.Name, tc.Arguments)
		result, execErr := mcpServer.ExecuteTool(tc.Name, tc.Arguments, callerCtx)
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

	log.Printf("[AI Service] 工具调用 - 发送请求到 AI，工具数: %d", len(tools))
	reply, err := s.GetCompletion(taskType, newMessages)
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
	result, err := mcpServer.ExecuteTool(toolCall.Name, toolCall.Arguments, callerCtx)
	if err != nil {
		log.Printf("[AI Service] 工具执行失败: %v", err)
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

func (s *AIService) GetConfig() *AIConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

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

func (s *AIService) filterContent(content string) string {
	if len(content) > 10000 {
		content = content[:10000] + "..."
	}
	return content
}

func (s *AIService) IsConfigured() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.pool) > 0
}

func (s *AIService) Embed(text string) ([]float32, error) {
	provider, _, err := s.router.SelectProvider(s.pool, TaskTypeEmbedding)
	if err != nil {
		return nil, err
	}
	return provider.Embedding(text)
}