package ai

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// streamMockProvider 模拟支持流式 tool-call 的 OpenAI 兼容 Provider：
// 第一个回合流式吐一个工具调用（arguments 分两片），第二个回合流式吐最终答案。
// 用调用次数区分回合，模拟 ReAct「工具回合 → final 回合」。
type streamMockProvider struct {
	BaseProvider
	calls int
}

func (p *streamMockProvider) Name() string { return "stream-mock" }
func (p *streamMockProvider) Chat(messages []Message) (string, error) {
	return "fallback", nil
}
func (p *streamMockProvider) ChatStream([]Message, func(StreamChunk) error) error { return nil }
func (p *streamMockProvider) ChatStreamWithContext(context.Context, []Message, func(StreamChunk) error) error {
	return nil
}
func (p *streamMockProvider) Embedding(string) ([]float32, error) { return nil, nil }
func (p *streamMockProvider) ChatWithTools([]Message, []ToolDef) (*ChatResponse, error) {
	return nil, nil
}
func (p *streamMockProvider) IsConfigured() bool        { return true }
func (p *streamMockProvider) WithModel(string) Provider { return p }

// ChatStreamWithTools 覆盖 BaseProvider 默认（默认返 ErrStreamingToolsNotSupported），
// 模拟真实流式 tool-call 解析。
func (p *streamMockProvider) ChatStreamWithTools(_ context.Context, _ []Message, _ []ToolDef, onChunk func(chunk StreamChunk) error) error {
	p.calls++
	if p.calls == 1 {
		// 工具回合：无正文，只吐 tool_calls 增量（arguments 分两片模拟 OpenAI 分片语义）
		stop := "tool_calls"
		if err := onChunk(StreamChunk{ToolCalls: []ToolCallDelta{{Index: 0, ID: "call_1", Name: "sum_tool", Arguments: `{"a":1`}}}); err != nil {
			return err
		}
		return onChunk(StreamChunk{ToolCalls: []ToolCallDelta{{Index: 0, Arguments: `,"b":2}`}}, Finish: &stop})
	}
	// final 回合：逐 token 吐正文
	stop := "stop"
	for _, piece := range []string{"答", "案", "是", "3"} {
		if err := onChunk(StreamChunk{Content: piece}); err != nil {
			return err
		}
	}
	return onChunk(StreamChunk{Finish: &stop})
}

type sumTool struct{}

func (sumTool) Name() string                       { return "sum_tool" }
func (sumTool) Description() string                { return "sum" }
func (sumTool) Parameters() map[string]interface{} { return map[string]interface{}{} }
func (sumTool) Execute(args map[string]interface{}, _ *CallerContext) (interface{}, error) {
	return float64(args["a"].(float64)) + float64(args["b"].(float64)), nil
}

func TestGetCompletionWithToolsStreamMultiStep(t *testing.T) {
	svc := NewAIService(&AIConfig{
		Router: RouterConfig{Routes: map[TaskType]Route{TaskTypeChat: {Provider: "mock"}}},
	})
	mock := &streamMockProvider{}
	svc.SetProviderForTesting("mock", mock)
	registry := NewToolRegistry(nil)
	registry.RegisterTool(sumTool{})
	svc.SetToolRegistry(registry)

	var streamed []string
	var steps []string
	onChunk := func(chunk StreamChunk) error {
		if chunk.Content != "" {
			streamed = append(streamed, chunk.Content)
		}
		return nil
	}
	onStep := func(_ int, callID, phase, name string, args map[string]interface{}, _ interface{}, _ error) {
		steps = append(steps, phase+"@"+name)
	}

	err := svc.GetCompletionWithToolsStreamMultiStep(
		TaskTypeChat,
		[]Message{{Role: "user", Content: "1+2=?"}},
		&CallerContext{UserID: 1},
		[]string{"sum_tool"},
		3,
		onStep,
		onChunk,
	)
	if err != nil {
		t.Fatalf("streaming react failed: %v", err)
	}

	// final 答案逐 token 流出
	got := strings.Join(streamed, "")
	if got != "答案是3" {
		t.Fatalf("expected streamed final answer '答案是3', got %q", got)
	}
	// 工具回合执行：start/end 均触发
	joinedSteps := strings.Join(steps, "")
	if !strings.Contains(joinedSteps, "start@sum_tool") || !strings.Contains(joinedSteps, "end@sum_tool") {
		t.Fatalf("expected tool start/end steps, got %v", steps)
	}
	// 工具被真实执行（累计结果回喂）→ 共享 mock 已到 final 回合
	if mock.calls != 2 {
		t.Fatalf("expected 2 provider calls (tool round + final round), got %d", mock.calls)
	}
}

func TestGetCompletionWithToolsStreamMultiStepFallsBackWhenUnsupported(t *testing.T) {
	// fallbackProvider 不覆盖 ChatStreamWithTools → 继承 BaseProvider 默认返回
	// ErrStreamingToolsNotSupported，流式引擎应在首回合识别并向上透传，供调用方降级。
	svc := NewAIService(&AIConfig{
		Router: RouterConfig{Routes: map[TaskType]Route{TaskTypeChat: {Provider: "mock"}}},
	})
	svc.SetProviderForTesting("mock", &fallbackProvider{})
	registry := NewToolRegistry(nil)
	registry.RegisterTool(staticTool{name: "allowed_tool"})
	svc.SetToolRegistry(registry)

	err := svc.GetCompletionWithToolsStreamMultiStep(
		TaskTypeChat,
		[]Message{{Role: "user", Content: "hi"}},
		&CallerContext{UserID: 1},
		nil,
		2,
		nil,
		func(StreamChunk) error { return nil },
	)
	if err == nil || !errors.Is(err, ErrStreamingToolsNotSupported) {
		t.Fatalf("expected ErrStreamingToolsNotSupported for non-streaming provider, got %v", err)
	}
}
