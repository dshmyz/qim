package service

import (
	"context"
	"strings"
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
)

// scriptedMemoProvider 按提示文本内容返回对应 JSON 的测试桩：
//   - 判定提示（含"请以 JSON 返回，形如"）→ 返回 remember/importance 判定 JSON
//   - 反射提示（含"记忆的可迁移范围"）→ 返回结构化反射 JSON（含 scope/type）
//
// 用于在不依赖真实 LLM 时验证 reflectConsolidated 会把反射出的 scope/type 正确带回 ref
// （回归该处曾缺一行赋值，导致落库 scope 恒空而按 global 处理）。
type scriptedMemoProvider struct {
	verdictJSON    string
	reflectionJSON string
}

var _ ai.Provider = (*scriptedMemoProvider)(nil)

func (p *scriptedMemoProvider) Name() string { return "scripted-memo" }
func (p *scriptedMemoProvider) Chat(messages []ai.Message) (string, error) {
	var content strings.Builder
	for _, m := range messages {
		content.WriteString(m.Content)
		content.WriteString("\n")
	}
	body := content.String()
	switch {
	case strings.Contains(body, "请以 JSON 返回，形如"):
		return p.verdictJSON, nil
	case strings.Contains(body, "记忆的可迁移范围"):
		return p.reflectionJSON, nil
	}
	return "{}", nil
}
func (p *scriptedMemoProvider) ChatStream(messages []ai.Message, onChunk func(chunk ai.StreamChunk) error) error {
	return nil
}
func (p *scriptedMemoProvider) ChatStreamWithContext(ctx context.Context, messages []ai.Message, onChunk func(chunk ai.StreamChunk) error) error {
	return nil
}
func (p *scriptedMemoProvider) Embedding(text string) ([]float32, error) {
	return nil, nil
}
func (p *scriptedMemoProvider) SupportsEmbedding() bool { return true }
func (p *scriptedMemoProvider) ChatWithTools(messages []ai.Message, tools []ai.ToolDef) (*ai.ChatResponse, error) {
	return &ai.ChatResponse{Content: "{}"}, nil
}
func (p *scriptedMemoProvider) ChatStreamWithTools(ctx context.Context, messages []ai.Message, tools []ai.ToolDef, onChunk func(ai.StreamChunk) error) error {
	return ai.ErrStreamingToolsNotSupported
}
func (p *scriptedMemoProvider) IsConfigured() bool { return true }
func (p *scriptedMemoProvider) WithModel(model string) ai.Provider {
	return p
}

func newScriptedMemoAIService(p *scriptedMemoProvider) *ai.AIService {
	svc := ai.NewAIService(&ai.AIConfig{})
	svc.SetProviderForTesting("scripted-memo", p)
	return svc
}

func TestReflectConsolidated_PropagatesConversationScope(t *testing.T) {
	svc := newScriptedMemoAIService(&scriptedMemoProvider{
		verdictJSON:    `{"remember": true, "importance": 4}`,
		reflectionJSON: `{"summary":"和张三约好周五晚上一起吃饭","facts":["周五晚上一起吃饭"],"themes":["约定"],"entities":["张三"],"type":"event","scope":"conversation"}`,
	})

	ref, verdict, err := reflectConsolidated(svc, "和张三约好周五晚上一起吃饭", nil, nil, nil)
	if err != nil {
		t.Fatalf("reflectConsolidated err: %v", err)
	}
	if !verdict.ShouldRemember {
		t.Fatal("应当判定为值得记")
	}
	if ref.Scope != "conversation" {
		t.Fatalf("反射出的 scope 未传播到 ref: got %q, want conversation", ref.Scope)
	}
	if ref.Type != "event" {
		t.Fatalf("反射出的 type 未传播到 ref: got %q, want event", ref.Type)
	}
	if ref.Summary != "和张三约好周五晚上一起吃饭" {
		t.Fatalf("反射出的 summary 未传播到 ref: got %q", ref.Summary)
	}
}

func TestReflectConsolidated_PropagatesGlobalScope(t *testing.T) {
	svc := newScriptedMemoAIService(&scriptedMemoProvider{
		verdictJSON:    `{"remember": true, "importance": 4}`,
		reflectionJSON: `{"summary":"项目 Alpha 上线定在下周一","facts":["项目 Alpha 下周一定上线"],"themes":["项目"],"entities":["Alpha"],"type":"fact","scope":"global"}`,
	})

	ref, _, err := reflectConsolidated(svc, "项目 Alpha 上线定在下周一", nil, nil, nil)
	if err != nil {
		t.Fatalf("reflectConsolidated err: %v", err)
	}
	if ref.Scope != "global" {
		t.Fatalf("反射出的 scope 未传播到 ref: got %q, want global", ref.Scope)
	}
}