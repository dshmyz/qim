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

// TestSelectProviderEmbeddingNoRouteUsesProviderDefault 验证 embedding 任务未显式配置路由时
// 回退到 defaultTask（chat）路由，但**模型名必须返回空串**：此时 chat 路由的模型是文本模型，
// 绝不能经 WithModel 塞给 embedding 端点（会把 gpt-4o 之类发送到 /embeddings 导致 400/404）。
// 空模型让 Embed() 走 provider.Embedding() 使用自身配置的 embedding 模型。
// （对照：vision 回退仍返回 chat 模型——视觉模型可读图；embedding 是唯一必须丢弃 chat 模型的场景。）
func TestSelectProviderEmbeddingNoRouteUsesProviderDefault(t *testing.T) {
	router := NewModelRouter(RouterConfig{
		DefaultTask: TaskTypeChat,
		Routes: map[TaskType]Route{
			TaskTypeChat: {Provider: "chat-svc", Model: "gpt-4o-mini", Fallback: []string{"embed-svc"}},
		},
	})
	pool := map[string]Provider{
		"chat-svc":  &nameableStub{name: "chat-svc"},
		"embed-svc": &nameableStub{name: "embed-svc"},
	}

	provider, model, err := router.SelectProvider(pool, TaskTypeEmbedding)
	if err != nil {
		t.Fatalf("embedding 选路失败: %v", err)
	}
	if provider == nil {
		t.Fatal("embedding 应选中某个 provider")
	}
	if model != "" {
		t.Fatalf("embedding 未配路由时不应返回 chat 模型名，实际 %q", model)
	}
}

// TestSelectProviderEmbeddingExplicitRouteKeepsModel 验证显式配置了 embedding 路由时，
// 返回该路由配置的 embedding 模型（不受"回退到 chat 路由返回空"规则影响）。
func TestSelectProviderEmbeddingExplicitRouteKeepsModel(t *testing.T) {
	router := NewModelRouter(RouterConfig{
		DefaultTask: TaskTypeChat,
		Routes: map[TaskType]Route{
			TaskTypeChat:      {Provider: "chat-svc", Model: "gpt-4o-mini"},
			TaskTypeEmbedding: {Provider: "embed-svc", Model: "text-embedding-3-small"},
		},
	})
	pool := map[string]Provider{
		"chat-svc":  &nameableStub{name: "chat-svc"},
		"embed-svc": &nameableStub{name: "embed-svc"},
	}

	provider, model, err := router.SelectProvider(pool, TaskTypeEmbedding)
	if err != nil {
		t.Fatalf("embedding 选路失败: %v", err)
	}
	if provider.Name() != "embed-svc" || model != "text-embedding-3-small" {
		t.Fatalf("显式 embedding 路由应返回其模型 (%s/%s)，实际 %s/%s",
			"embed-svc", "text-embedding-3-small", provider.Name(), model)
	}
}
