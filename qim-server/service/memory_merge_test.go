package service

import (
	"context"
	"strings"
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
)

// mergeKindProvider 固定返回指定 kind 的三分类测试桩，用于不调真实 LLM 验证 inferMemoryMergeKind。
type mergeKindProvider struct{ kind string }

var _ ai.Provider = (*mergeKindProvider)(nil)

func (p *mergeKindProvider) Name() string { return "merge-kind" }
func (p *mergeKindProvider) Chat(_ []ai.Message) (string, error) {
	return `{"kind": "` + p.kind + `"}`, nil
}
func (p *mergeKindProvider) ChatStream(_ []ai.Message, _ func(ai.StreamChunk) error) error { return nil }
func (p *mergeKindProvider) ChatStreamWithContext(_ context.Context, _ []ai.Message, _ func(ai.StreamChunk) error) error {
	return nil
}
func (p *mergeKindProvider) Embedding(_ string) ([]float32, error) { return nil, nil }
func (p *mergeKindProvider) SupportsEmbedding() bool               { return true }
func (p *mergeKindProvider) ChatWithTools(_ []ai.Message, _ []ai.ToolDef) (*ai.ChatResponse, error) {
	return &ai.ChatResponse{Content: "{}"}, nil
}
func (p *mergeKindProvider) ChatStreamWithTools(_ context.Context, _ []ai.Message, _ []ai.ToolDef, _ func(ai.StreamChunk) error) error {
	return ai.ErrStreamingToolsNotSupported
}
func (p *mergeKindProvider) IsConfigured() bool { return true }
func (p *mergeKindProvider) WithModel(_ string) ai.Provider {
	return p
}

// TestInferMemoryMergeKind_Mapping 锁定三分类 LLM 返回字符串到枚举的映射（R1 统一判定）。
func TestInferMemoryMergeKind_Mapping(t *testing.T) {
	cases := []struct {
		kind string
		want MemoryMergeKind
	}{
		{"conflict", MemoryMergeConflict},
		{"duplicate", MemoryMergeDuplicate},
		{"new", MemoryMergeNew},
		{"unknown", MemoryMergeNew},
	}
	for _, c := range cases {
		svc := ai.NewAIService(&ai.AIConfig{})
		svc.SetProviderForTesting("merge-kind", &mergeKindProvider{kind: c.kind})
		got, err := inferMemoryMergeKind(svc, "新", "旧")
		if err != nil {
			t.Fatalf("kind=%s: 不应报错, got %v", c.kind, err)
		}
		if got != c.want {
			t.Fatalf("kind=%s: 期望 %v, got %v", c.kind, c.want, got)
		}
	}
}

// TestInferMemoryMergeKind_NilAndUnparseable 锁定安全回退：aiService=nil 或无法解析时退化为新增（不误并不误删）。
func TestInferMemoryMergeKind_NilAndUnparseable(t *testing.T) {
	if got, err := inferMemoryMergeKind(nil, "新", "旧"); err != nil || got != MemoryMergeNew {
		t.Fatalf("nil svc 应返回 New 且无错, got %v err=%v", got, err)
	}
	svc := ai.NewAIService(&ai.AIConfig{})
	svc.SetProviderForTesting("merge-kind", &mergeKindProvider{kind: ""})
	got, err := inferMemoryMergeKind(svc, "新", "旧")
	if err != nil {
		t.Fatalf("不可解析不应报错, got %v", err)
	}
	if got != MemoryMergeNew {
		t.Fatalf("不可解析应退化为 New, got %v", got)
	}
}

// TestMergedImportance_KeepsHigher 锁定重要度合并取 max（防止复述把旧档位 5 降成新档位 3）。
func TestMergedImportance_KeepsHigher(t *testing.T) {
	if got := mergedImportance(map[string]string{"importance": "5.0"}, 3); got != 5.0 {
		t.Fatalf("旧档位更高应保留 5，got %v", got)
	}
	if got := mergedImportance(map[string]string{"importance": "2.0"}, 4); got != 4.0 {
		t.Fatalf("新档位更高应取 4，got %v", got)
	}
	// 旧 metadata 缺失/坏值 → 回退新档位
	if got := mergedImportance(nil, 4); got != 4.0 {
		t.Fatalf("缺 old 应回退新档位 4，got %v", got)
	}
	if got := mergedImportance(map[string]string{"importance": "abc"}, 4); got != 4.0 {
		t.Fatalf("坏值应回退新档位 4，got %v", got)
	}
}

// TestGroupMemoryCtxText_WithSender 锁定群记忆正文注入附带发言人（P2-consume）。
func TestGroupMemoryCtxText_WithSender(t *testing.T) {
	results := []SearchResult{
		{Content: "项目改用 PostgreSQL", Metadata: map[string]string{"sender_name": "李四"}},
		{Content: "每周五下午发版", Metadata: map[string]string{"sender_name": "王五"}},
		{Content: "旧兼容记忆（无 sender）", Metadata: map[string]string{}},
		{Content: "", Metadata: map[string]string{"sender_name": "空"}, Score: 0.9}, // 空内容应跳过
	}
	out := GroupMemoryCtxText(results)
	if !strings.Contains(out, "• [李四] 项目改用 PostgreSQL") {
		t.Fatalf("应附发言人李四, got: %q", out)
	}
	if !strings.Contains(out, "• [王五] 每周五下午发版") {
		t.Fatalf("应附发言人王五, got: %q", out)
	}
	if !strings.Contains(out, "• 旧兼容记忆（无 sender）") {
		t.Fatalf("无 sender 应退化为纯内容行, got: %q", out)
	}
	if strings.Contains(out, "空") || strings.Count(out, "•") != 3 {
		t.Fatalf("空内容应被跳过且行数=3, got: %q", out)
	}
}

// TestMemoryResultsToSources_SuppressesLowImportance 锁定 R2：群记忆重要度≤2 不进「知识来源」徽章。
func TestMemoryResultsToSources_SuppressesLowImportance(t *testing.T) {
	results := []SearchResult{
		{Content: "高重要度记忆", Score: 0.9, DocID: "m1", Metadata: map[string]string{"importance": "5.0"}},
		{Content: "低重要度记忆", Score: 0.8, DocID: "m2", Metadata: map[string]string{"importance": "2.0"}},
		{Content: "缺失重要度记忆", Score: 0.7, DocID: "m3", Metadata: map[string]string{}},
	}
	out := memoryResultsToSources(results, 0)
	if len(out) != 2 {
		t.Fatalf("期望高重要度+缺失重要度共 2 条进徽章, got %d", len(out))
	}
	for _, ks := range out {
		if ks.ID == "m2" {
			t.Fatalf("低重要度记忆不应进徽章: %+v", ks)
		}
	}
}