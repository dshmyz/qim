package ai

import (
	"testing"
)

type fakeProvider struct {
	Provider
	model string
}

func (f *fakeProvider) IsConfigured() bool { return true }
func (f *fakeProvider) Name() string       { return "fake" }

// unavailableProvider 模拟未配置（不可用）的 provider。
type unavailableProvider struct{ Provider }

func (u *unavailableProvider) IsConfigured() bool { return false }
func (u *unavailableProvider) Name() string       { return "unavailable" }

func TestRouterImplicitFallbackDropsRouteModel(t *testing.T) {
	cfg := RouterConfig{
		DefaultTask: TaskTypeChat,
		Routes: map[TaskType]Route{
			TaskTypeChat: {Provider: "primary", Model: "deepseek-v4-flash", Fallback: []string{"nonexistent"}},
		},
	}
	r := NewModelRouter(cfg)

	primary := &unavailableProvider{}
	backup := &fakeProvider{}
	pool := map[string]Provider{"primary": primary, "backup": backup}

	// 显式配置的 primary 不可用 → 隐式兜底选中 backup：
	// 原路由模型名不得携带过去（对兜底 provider 多半不存在，会 400/404）
	p, model, err := r.SelectProvider(pool, TaskTypeChat)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if p != Provider(backup) {
		t.Fatal("应选中 backup")
	}
	if model != "" {
		t.Fatalf("隐式兜底应返回空模型名让 provider 用自身配置，got %q", model)
	}
}

func TestRouterExplicitRouteKeepsModel(t *testing.T) {
	cfg := RouterConfig{
		DefaultTask: TaskTypeChat,
		Routes: map[TaskType]Route{
			TaskTypeChat: {Provider: "primary", Model: "deepseek-v4-flash"},
		},
	}
	r := NewModelRouter(cfg)
	primary := &fakeProvider{}
	pool := map[string]Provider{"primary": primary}

	_, model, err := r.SelectProvider(pool, TaskTypeChat)
	if err != nil {
		t.Fatal(err)
	}
	if model != "deepseek-v4-flash" {
		t.Fatalf("显式路由应保留模型名，got %q", model)
	}
}

func TestWithTemperature_DoesNotPolluteSharedConfig(t *testing.T) {
	p := &OpenAIProvider{}
	p.config.ExtraParams = map[string]interface{}{"temperature": 0.7, "max_tokens": 1000}

	cold := p.WithTemperature(0.1).(*OpenAIProvider)

	// 共享实例的温度不得被改写
	if v := p.config.ExtraParams["temperature"]; v != 0.7 {
		t.Fatalf("共享实例温度被污染: %v", v)
	}
	// 新实例温度生效
	if v := cold.config.ExtraParams["temperature"]; v != 0.1 {
		t.Fatalf("新实例温度未生效: %v", v)
	}
	// 其他参数保留
	if v := cold.config.ExtraParams["max_tokens"]; v != 1000 {
		t.Fatalf("其他参数应保留: %v", v)
	}
}

func TestDeterministicTemperature(t *testing.T) {
	if t2, ok := deterministicTemperature(TaskTypeIntent); !ok || t2 != 0.1 {
		t.Fatalf("intent 应为 0.1 低温，got %v ok=%v", t2, ok)
	}
	if t2, ok := deterministicTemperature(TaskTypeAnalysis); !ok || t2 != 0.1 {
		t.Fatalf("analysis 应为 0.1 低温，got %v ok=%v", t2, ok)
	}
	if _, ok := deterministicTemperature(TaskTypeChat); ok {
		t.Fatal("chat 对话不应强制低温")
	}
}
