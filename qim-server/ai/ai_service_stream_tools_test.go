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
// preamble 非空时工具回合先吐一段前置草稿正文再吐 tool_calls，
// 模拟部分模型调工具前输出思考文本的行为。
type streamMockProvider struct {
	BaseProvider
	calls    int
	preamble string
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

func (p *streamMockProvider) SupportsEmbedding() bool { return true }
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
		// 工具回合：可选拈权草稿正文，再吐 tool_calls 增量（arguments 分两片模拟 OpenAI 分片语义）
		if p.preamble != "" {
			if err := onChunk(StreamChunk{Content: p.preamble}); err != nil {
				return err
			}
		}
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
		context.Background(),
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
		context.Background(),
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

// TestStreamMultiStepToolRoundPreambleRetracted 验证工具回合前置草稿的回撤帧协议：
// 草稿先按正文逐 token 实时流出（保证流式打字感），回合终了发现跟着 tool_calls，
// 引擎补发一帧 MoveToReasoning 携带草稿原文——前端据此把草稿从答案区切走移入思考区，
// 答案区最终只剩 final 回合内容。
func TestStreamMultiStepToolRoundPreambleRetracted(t *testing.T) {
	svc := NewAIService(&AIConfig{
		Router: RouterConfig{Routes: map[TaskType]Route{TaskTypeChat: {Provider: "mock"}}},
	})
	mock := &streamMockProvider{preamble: "我先查一下待办…"}
	svc.SetProviderForTesting("mock", mock)
	registry := NewToolRegistry(nil)
	registry.RegisterTool(sumTool{})
	svc.SetToolRegistry(registry)

	var content strings.Builder
	var reasoningFrames []string
	onChunk := func(chunk StreamChunk) error {
		if chunk.Content != "" {
			content.WriteString(chunk.Content)
		}
		if chunk.MoveToReasoning != "" {
			reasoningFrames = append(reasoningFrames, chunk.MoveToReasoning)
		}
		return nil
	}

	err := svc.GetCompletionWithToolsStreamMultiStep(
		context.Background(),
		TaskTypeChat,
		[]Message{{Role: "user", Content: "我有哪些待办？"}},
		&CallerContext{UserID: 1},
		[]string{"sum_tool"},
		3,
		nil,
		onChunk,
	)
	if err != nil {
		t.Fatalf("streaming react failed: %v", err)
	}

	draft := "我先查一下待办…"
	// 草稿确实先按正文流出了（live 打字），再被回撤
	if !strings.HasPrefix(content.String(), draft) {
		t.Fatalf("expected preamble %q streamed as content first, got %q", draft, content.String())
	}
	// 回撤帧恰好一帧、原文精确匹配
	if len(reasoningFrames) != 1 || reasoningFrames[0] != draft {
		t.Fatalf("expected exactly one MoveToReasoning frame carrying %q, got %v", draft, reasoningFrames)
	}
	// 模拟前端回撤（按原文从答案区切除）后，答案区只剩 final 回合内容
	answer := strings.Replace(content.String(), draft, "", 1)
	if answer != "答案是3" {
		t.Fatalf("expected answer '答案是3' after retracting preamble, got %q", answer)
	}
}
