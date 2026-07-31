package ai

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type fallbackProvider struct {
	BaseProvider
}

func (p *fallbackProvider) Name() string { return "fallback" }
func (p *fallbackProvider) Chat(messages []Message) (string, error) {
	system := ""
	if len(messages) > 0 {
		system = messages[0].Content
	}
	if strings.Contains(system, "forbidden_tool") {
		return `{"tool_call":{"name":"forbidden_tool","arguments":{}}}`, nil
	}
	if strings.Contains(system, "allowed_tool") {
		return `{"tool_call":{"name":"allowed_tool","arguments":{}}}`, nil
	}
	return "final", nil
}
func (p *fallbackProvider) ChatStream([]Message, func(StreamChunk) error) error { return nil }
func (p *fallbackProvider) ChatStreamWithContext(context.Context, []Message, func(StreamChunk) error) error {
	return nil
}
func (p *fallbackProvider) Embedding(string) ([]float32, error) { return nil, nil }
func (p *fallbackProvider) ChatWithTools([]Message, []ToolDef) (*ChatResponse, error) {
	return nil, fmt.Errorf("native tools unavailable")
}
func (p *fallbackProvider) IsConfigured() bool        { return true }
func (p *fallbackProvider) WithModel(string) Provider { return p }

type staticTool struct {
	name string
}

func (t staticTool) Name() string                       { return t.name }
func (t staticTool) Description() string                { return t.name }
func (t staticTool) Parameters() map[string]interface{} { return map[string]interface{}{} }
func (t staticTool) Execute(map[string]interface{}, *CallerContext) (interface{}, error) {
	if t.name == "forbidden_tool" {
		return nil, fmt.Errorf("forbidden tool executed")
	}
	return map[string]interface{}{"ok": true}, nil
}

func TestPromptEngineeringFallbackHonorsAllowedTools(t *testing.T) {
	svc := NewAIService(&AIConfig{
		Router: RouterConfig{Routes: map[TaskType]Route{TaskTypeChat: {Provider: "mock"}}},
	})
	svc.SetProviderForTesting("mock", &fallbackProvider{})
	registry := NewToolRegistry(nil)
	registry.RegisterTool(staticTool{name: "allowed_tool"})
	registry.RegisterTool(staticTool{name: "forbidden_tool"})
	svc.SetToolRegistry(registry)

	_, err := svc.GetCompletionWithToolsMultiStep(
		TaskTypeChat,
		[]Message{{Role: "system", Content: "system"}, {Role: "user", Content: "use a tool"}},
		&CallerContext{AllowedTools: []string{"allowed_tool"}},
		[]string{"allowed_tool"},
		1,
		nil,
	)
	if err != nil {
		t.Fatalf("fallback should not expose or execute disallowed tools: %v", err)
	}
}
