package ai

import "testing"

// nameableStub 可命名、默认已配置的 Provider 桩，用于路由选择测试。
// 嵌入 BaseProvider 复用 Embedding/ChatWithTools 等默认实现，仅需补齐核心方法。
type nameableStub struct {
	BaseProvider
	name string
}

func (p *nameableStub) Name() string                                              { return p.name }
func (p *nameableStub) IsConfigured() bool                                        { return true }
func (p *nameableStub) WithModel(string) Provider                                 { return p }
func (p *nameableStub) Chat([]Message) (string, error)                            { return "", nil }
func (p *nameableStub) ChatStream([]Message, func(StreamChunk) error) error       { return nil }
func (p *nameableStub) ChatWithTools([]Message, []ToolDef) (*ChatResponse, error) { return nil, nil }
func (p *nameableStub) Embedding(string) ([]float32, error)                       { return nil, nil }

// TestSelectProviderVisionExplicit 验证显式配置「视觉理解」路由时，
// 视觉任务命中该路由的 Provider / 模型，而非回退到 defaultTask。
func TestSelectProviderVisionExplicit(t *testing.T) {
	router := NewModelRouter(RouterConfig{
		DefaultTask: TaskTypeChat,
		Routes: map[TaskType]Route{
			TaskTypeChat:   {Provider: "chat-svc", Model: "gpt-4o-mini"},
			TaskTypeVision: {Provider: "vision-svc", Model: "gpt-4o"},
		},
	})
	pool := map[string]Provider{
		"chat-svc":   &nameableStub{name: "chat-svc"},
		"vision-svc": &nameableStub{name: "vision-svc"},
	}

	if !router.HasExplicitRoute(TaskTypeVision) {
		t.Fatal("显式配置视觉路由后 HasExplicitRoute 应为 true")
	}

	provider, model, err := router.SelectProvider(pool, TaskTypeVision)
	if err != nil {
		t.Fatalf("选路失败: %v", err)
	}
	if provider.Name() != "vision-svc" || model != "gpt-4o" {
		t.Fatalf("视觉任务应命中 vision 路由 (%s/%s)，实际 %s/%s",
			"vision-svc", "gpt-4o", provider.Name(), model)
	}
}

// TestSelectProviderVisionFallsBackToDefault 验证未显式配置视觉路由时，
// 视觉任务回退到 defaultTask 的 Provider / 模型，HasExplicitRoute 为 false。
func TestSelectProviderVisionFallsBackToDefault(t *testing.T) {
	router := NewModelRouter(RouterConfig{
		DefaultTask: TaskTypeChat,
		Routes: map[TaskType]Route{
			TaskTypeChat: {Provider: "chat-svc", Model: "gpt-4o-mini"},
		},
	})
	pool := map[string]Provider{
		"chat-svc": &nameableStub{name: "chat-svc"},
	}

	if router.HasExplicitRoute(TaskTypeVision) {
		t.Fatal("未配置视觉路由时 HasExplicitRoute 应为 false")
	}

	provider, model, err := router.SelectProvider(pool, TaskTypeVision)
	if err != nil {
		t.Fatalf("回退选路失败: %v", err)
	}
	if provider.Name() != "chat-svc" || model != "gpt-4o-mini" {
		t.Fatalf("视觉任务未配置时应回退到 defaultTask (%s/%s)，实际 %s/%s",
			"chat-svc", "gpt-4o-mini", provider.Name(), model)
	}
}
