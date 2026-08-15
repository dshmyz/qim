package ai

import (
	"testing"
)

// TestAnthropicProviderMergesExtraParams 验证 createAnthropicProviderFromConfig 不再丢弃
// 调用方传入的 cfg.ExtraParams（修复 #4）：用户自选模型保存的 max_tokens/temperature
// 应透传到 anthropic provider，缺失时才补默认值（与 openai 系行为一致）。
func TestAnthropicProviderMergesExtraParams(t *testing.T) {
	f := NewProviderFactory()
	p := f.createAnthropicProviderFromConfig(ProviderConfig{
		APIKey:  "sk-a",
		Model:   "claude-3-5-sonnet",
		BaseURL: "https://api.anthropic.com/v1",
		ExtraParams: map[string]interface{}{
			"max_tokens":  2048,
			"temperature": 0.2,
		},
	})
	ap, ok := p.(*AnthropicProvider)
	if !ok {
		t.Fatalf("应返回 AnthropicProvider，实际 %T", p)
	}
	if ap.config.ExtraParams["max_tokens"] != 2048 {
		t.Fatalf("max_tokens 未透传: %v", ap.config.ExtraParams["max_tokens"])
	}
	if ap.config.ExtraParams["temperature"] != 0.2 {
		t.Fatalf("temperature 未透传: %v", ap.config.ExtraParams["temperature"])
	}
}

func TestAnthropicProviderFillsDefaults(t *testing.T) {
	f := NewProviderFactory()
	p := f.createAnthropicProviderFromConfig(ProviderConfig{APIKey: "sk-a"})
	ap, _ := p.(*AnthropicProvider)
	if ap.config.ExtraParams["max_tokens"] != 1000 {
		t.Fatalf("缺失 max_tokens 应补默认 1000: %v", ap.config.ExtraParams["max_tokens"])
	}
	if ap.config.ExtraParams["temperature"] != 0.7 {
		t.Fatalf("缺失 temperature 应补默认 0.7: %v", ap.config.ExtraParams["temperature"])
	}
}

// TestGetCompletionWithProviderConfigHonorsUserParams 验证走 provider-config 的非流式路径
// 把用户保存的参数递给 provider（factory.CreateProviderByName 消费 cfg.ExtraParams）。
func TestGetCompletionWithProviderConfigHonorsUserParams(t *testing.T) {
	svc := NewAIService(&AIConfig{})
	var got string
	gotCfg := ProviderConfig{}
	var seenMax, seenTemp interface{}
	svc.factory.createOverride = func(_ string, cfg ProviderConfig) (Provider, error) {
		got = cfg.Model
		gotCfg = cfg
		seenMax = cfg.ExtraParams["max_tokens"]
		seenTemp = cfg.ExtraParams["temperature"]
		return &streamUsageFakeProvider{}, nil
	}
	_, err := svc.GetCompletionWithProviderConfig(
		TaskTypeChat, []Message{{Role: "user", Content: "hi"}},
		"openai",
		ProviderConfig{Model: "gpt-4", ExtraParams: map[string]interface{}{
			"max_tokens": 512, "temperature": float64(0),
		}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "gpt-4" {
		t.Fatalf("model 不符: %s", got)
	}
	if seenMax != 512 {
		t.Fatalf("max_tokens 未透传: %v", seenMax)
	}
	if seenTemp != float64(0) {
		t.Fatalf("temperature=0 应透传（确定性输出）: %v", seenTemp)
	}
	if gotCfg.ExtraParams == nil {
		t.Fatal("ExtraParams 不应为 nil")
	}
}

// TestOpenAIProviderPropagatesEmbeddingBaseURL 验证 OpenAI 兼容 provider 的搭建路径都透传
// EmbeddingBaseURL（embedding 语义检索需要独立的 embedding 端点，否则会打到 chat baseURL 失败）。
// createGenericOpenAIProvider 是真实路径（CreateProviderByName 实际消费）；createOpenAIProvider
// 为遗留入口，两处必须一致，避免未来误用导致 embedding 失效。
func TestOpenAIProviderPropagatesEmbeddingBaseURL(t *testing.T) {
	f := NewProviderFactory()
	want := "https://embed.example.com/v3"

	// 真实路径：CreateProviderByName 的 openai 分支走 createGenericOpenAIProvider
	p, err := f.CreateProviderByName("openai", ProviderConfig{
		APIKey:           "sk-a",
		Model:            "gpt-4",
		BaseURL:          "https://api.example.com/v1",
		EmbeddingBaseURL: want,
	})
	if err != nil {
		t.Fatalf("CreateProviderByName err: %v", err)
	}
	op, ok := p.(*OpenAIProvider)
	if !ok {
		t.Fatalf("应为 *OpenAIProvider，实际 %T", p)
	}
	if op.config.EmbeddingBaseURL != want {
		t.Fatalf("createGenericOpenAIProvider 未透传 EmbeddingBaseURL: 期望 %q 实际 %q", want, op.config.EmbeddingBaseURL)
	}

	// 遗留入口 createOpenAIProvider 必须同样透传（当前漏传，本测试驱动其补上）
	legacy := f.createOpenAIProvider(&AIConfig{
		OpenAI: OpenAIConfig{APIKey: "sk-a", Model: "gpt-4", BaseURL: "https://api.example.com/v1", EmbeddingBaseURL: want},
	})
	lop, ok := legacy.(*OpenAIProvider)
	if !ok {
		t.Fatalf("createOpenAIProvider 应为 *OpenAIProvider，实际 %T", legacy)
	}
	if lop.config.EmbeddingBaseURL != want {
		t.Fatalf("createOpenAIProvider 未透传 EmbeddingBaseURL: 期望 %q 实际 %q", want, lop.config.EmbeddingBaseURL)
	}
}
