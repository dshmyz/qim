package ai

import (
	"context"
	"fmt"
	"testing"
)

// embedRecorderProvider 记录 WithModel 传入的模型，供验证 Embed 用 router 的 embedding.model。
type embedRecorderProvider struct {
	model string
}

func (p *embedRecorderProvider) Name() string { return "embed-fake" }
func (p *embedRecorderProvider) Chat([]Message) (string, error) { return "", nil }
func (p *embedRecorderProvider) ChatStream([]Message, func(StreamChunk) error) error { return nil }
func (p *embedRecorderProvider) ChatStreamWithContext(context.Context, []Message, func(StreamChunk) error) error {
	return nil
}
func (p *embedRecorderProvider) Embedding(string) ([]float32, error) {
	return []float32{float32(len(p.model))}, nil
}
func (p *embedRecorderProvider) SupportsEmbedding() bool { return true }
func (p *embedRecorderProvider) ChatWithTools([]Message, []ToolDef) (*ChatResponse, error) {
	return nil, nil
}
func (p *embedRecorderProvider) ChatStreamWithTools(context.Context, []Message, []ToolDef, func(StreamChunk) error) error {
	return ErrStreamingToolsNotSupported
}
func (p *embedRecorderProvider) IsConfigured() bool { return true }
func (p *embedRecorderProvider) WithModel(model string) Provider {
	p.model = model
	return p
}

// TestEmbed_UsesRouterEmbeddingModel
// Embed 应使用 router.embedding 路由配置的模型（经 WithModel 传入 provider），
// 而不是 provider 内部的自有模型。统一配置入口，消除双轨制误导。
func TestEmbed_UsesRouterEmbeddingModel(t *testing.T) {
	svc := NewAIService(&AIConfig{
		Router: RouterConfig{
			DefaultTask: TaskTypeChat,
			Routes: map[TaskType]Route{
				TaskTypeEmbedding: {Provider: "embed-fake", Model: "text-embed-3"},
			},
		},
	})

	fake := &embedRecorderProvider{}
	svc.SetProviderForTesting("embed-fake", fake)

	vec, err := svc.Embed("hello")
	if err != nil {
		t.Fatalf("Embed 失败: %v", err)
	}
	if fake.model != "text-embed-3" {
		t.Errorf("Embed 应使用 router 的 embedding 模型 text-embed-3，got %q", fake.model)
	}
	// 返回向量长度应基于传入的模型名（证明 Embedding 用的是 router 模型）
	if len(vec) != 1 || vec[0] != float32(len("text-embed-3")) {
		t.Errorf("Embedding 应基于 router 模型计算，got %v", vec)
	}
}

// TestEmbed_SkipsNonEmbeddingProvider
// embedding 路由配到不支持 embedding 的 provider 时，应回退/报错而非调用其 Embedding。
func TestEmbed_SkipsNonEmbeddingProvider(t *testing.T) {
	svc := NewAIService(&AIConfig{
		Router: RouterConfig{
			DefaultTask: TaskTypeChat,
			Routes: map[TaskType]Route{
				TaskTypeEmbedding: {Provider: "no-embed", Model: "x"},
			},
		},
	})
	// 只注入一个不支持 embedding 的 provider
	svc.SetProviderForTesting("no-embed", &nonEmbedProvider{})

	_, err := svc.Embed("hello")
	if err == nil {
		t.Fatal("embedding 路由指向不支持 embedding 的 provider 时应返回错误")
	}
}

// nonEmbedProvider 不支持 embedding 的 provider（模拟 Anthropic）。
type nonEmbedProvider struct{}

func (p *nonEmbedProvider) Name() string { return "no-embed" }
func (p *nonEmbedProvider) Chat([]Message) (string, error) { return "", nil }
func (p *nonEmbedProvider) ChatStream([]Message, func(StreamChunk) error) error { return nil }
func (p *nonEmbedProvider) ChatStreamWithContext(context.Context, []Message, func(StreamChunk) error) error {
	return nil
}
func (p *nonEmbedProvider) Embedding(string) ([]float32, error) {
	return nil, fmt.Errorf("not implemented")
}
func (p *nonEmbedProvider) SupportsEmbedding() bool { return false }
func (p *nonEmbedProvider) ChatWithTools([]Message, []ToolDef) (*ChatResponse, error) { return nil, nil }
func (p *nonEmbedProvider) ChatStreamWithTools(context.Context, []Message, []ToolDef, func(StreamChunk) error) error {
	return ErrStreamingToolsNotSupported
}
func (p *nonEmbedProvider) IsConfigured() bool { return true }
func (p *nonEmbedProvider) WithModel(string) Provider { return p }
