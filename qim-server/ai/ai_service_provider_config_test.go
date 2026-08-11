package ai

import (
	"context"
	"testing"
)

// streamUsageFakeProvider 测试桩：在流式回调中按预设顺序发送 chunk。
// 用于验证 ChatStreamWithProviderConfig 能截获最后一个 chunk 的 Usage 并上报 usageSink。
type streamUsageFakeProvider struct {
	chunks []StreamChunk
}

func (p *streamUsageFakeProvider) Name() string { return "stream-usage-fake" }
func (p *streamUsageFakeProvider) Chat([]Message) (string, error) {
	return "", nil
}
func (p *streamUsageFakeProvider) ChatStream([]Message, func(StreamChunk) error) error { return nil }
func (p *streamUsageFakeProvider) ChatStreamWithContext(_ context.Context, _ []Message, onChunk func(StreamChunk) error) error {
	for _, c := range p.chunks {
		if err := onChunk(c); err != nil {
			return err
		}
	}
	return nil
}
func (p *streamUsageFakeProvider) Embedding(string) ([]float32, error)                        { return nil, nil }
func (p *streamUsageFakeProvider) SupportsEmbedding() bool { return true }
func (p *streamUsageFakeProvider) ChatWithTools([]Message, []ToolDef) (*ChatResponse, error) { return nil, nil }
func (p *streamUsageFakeProvider) ChatStreamWithTools(context.Context, []Message, []ToolDef, func(StreamChunk) error) error {
	return ErrStreamingToolsNotSupported
}
func (p *streamUsageFakeProvider) IsConfigured() bool { return true }
func (p *streamUsageFakeProvider) WithModel(string) Provider {
	return p
}

func TestChatStreamWithProviderConfigReportsUsage(t *testing.T) {
	svc := NewAIService(&AIConfig{})
	svc.factory.createOverride = func(_ string, _ ProviderConfig) (Provider, error) {
		return &streamUsageFakeProvider{chunks: []StreamChunk{
			{Content: "hello"},
			{Usage: &StreamUsage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30}},
		}}, nil
	}

	var capturedUsage *TokenUsage
	var capturedProvider, capturedModel string
	var capturedDuration int64
	svc.SetUsageSink(func(_ TaskType, provider, model string, usage *TokenUsage, durationMs int64) {
		capturedProvider = provider
		capturedModel = model
		capturedUsage = usage
		capturedDuration = durationMs
	})

	err := svc.ChatStreamWithProviderConfig(
		context.Background(), TaskTypeChat,
		[]Message{{Role: "user", Content: "hi"}},
		"openai", ProviderConfig{Model: "gpt-4"},
		func(_ StreamChunk) error { return nil },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedUsage == nil {
		t.Fatal("usageSink 未被调用：流式 usage 应被截获并上报")
	}
	if capturedUsage.PromptTokens != 10 || capturedUsage.CompletionTokens != 20 || capturedUsage.TotalTokens != 30 {
		t.Fatalf("usage 数值不符: %+v", capturedUsage)
	}
	if capturedProvider != "stream-usage-fake" {
		t.Fatalf("provider 名不符: %s", capturedProvider)
	}
	if capturedModel != "gpt-4" {
		t.Fatalf("model 名不符: %s", capturedModel)
	}
	if capturedDuration < 0 {
		t.Fatalf("duration 不应为负: %d", capturedDuration)
	}
}

func TestChatStreamWithProviderConfigNoUsageWhenChunkHasNone(t *testing.T) {
	svc := NewAIService(&AIConfig{})
	svc.factory.createOverride = func(_ string, _ ProviderConfig) (Provider, error) {
		return &streamUsageFakeProvider{chunks: []StreamChunk{
			{Content: "hello"},
			{Content: " world"},
		}}, nil
	}

	sinkCalled := false
	svc.SetUsageSink(func(_ TaskType, _, _ string, _ *TokenUsage, _ int64) {
		sinkCalled = true
	})

	err := svc.ChatStreamWithProviderConfig(
		context.Background(), TaskTypeChat,
		[]Message{{Role: "user", Content: "hi"}},
		"openai", ProviderConfig{},
		func(_ StreamChunk) error { return nil },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sinkCalled {
		t.Fatal("chunk 无 Usage 时不应调用 usageSink")
	}
}
