package service

import "testing"

func TestExtractGraphEntities_Codes(t *testing.T) {
	text := "产品需求文档 PRD-2024-001 需要跟进，另外 BUG-123 已经修复。V1.2-3 是版本号。"
	ents := extractGraphEntities(text)
	want := map[string]bool{"PRD-2024-001": true, "BUG-123": true}
	got := map[string]bool{}
	for _, e := range ents {
		got[e] = true
	}
	for k := range want {
		if !got[k] {
			t.Errorf("extractGraphEntities 未抽到 %q，got %v", k, ents)
		}
	}
}

func TestExtractGraphEntities_Empty(t *testing.T) {
	if ents := extractGraphEntities(""); ents != nil && len(ents) != 0 {
		t.Errorf("空文本应返回空，got %v", ents)
	}
	if ents := extractGraphEntities("普通中文没有编码 token"); len(ents) != 0 {
		t.Errorf("无编码 token 应返回空，got %v", ents)
	}
}

func TestExtractGraphEntities_Dedupe(t *testing.T) {
	text := "PRD-2024-001 和 PRD-2024-001 是同一个需求"
	ents := extractGraphEntities(text)
	count := 0
	for _, e := range ents {
		if e == "PRD-2024-001" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("应去重为 1 个 PRD-2024-001，got %d", count)
	}
}

func TestContainsStr(t *testing.T) {
	if !containsStr([]string{"a", "b"}, "b") {
		t.Error("containsStr 应命中 b")
	}
	if containsStr([]string{"a", "b"}, "c") {
		t.Error("containsStr 不应命中 c")
	}
}
