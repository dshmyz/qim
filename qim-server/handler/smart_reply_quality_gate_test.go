package handler

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/service"
)

// verdictStubProvider 质量门测试桩：Chat 固定返回核验 verdict JSON。
// WithModel 返回自身（ai_provider-stub 陷阱：带 model 路由时经 WithModel 调用，不覆写会丢桩）。
type verdictStubProvider struct {
	ai.BaseProvider
	chatFn func(messages []ai.Message) (string, error)
}

func (p *verdictStubProvider) Name() string { return "stub" }
func (p *verdictStubProvider) Chat(messages []ai.Message) (string, error) {
	return p.chatFn(messages)
}
func (p *verdictStubProvider) ChatStream([]ai.Message, func(ai.StreamChunk) error) error { return nil }
func (p *verdictStubProvider) ChatStreamWithContext(context.Context, []ai.Message, func(ai.StreamChunk) error) error {
	return nil
}
func (p *verdictStubProvider) Embedding(string) ([]float32, error) { return nil, nil }
func (p *verdictStubProvider) SupportsEmbedding() bool             { return false }
func (p *verdictStubProvider) IsConfigured() bool                  { return true }
func (p *verdictStubProvider) WithModel(string) ai.Provider        { return p }
func (p *verdictStubProvider) ChatWithTools([]ai.Message, []ai.ToolDef) (*ai.ChatResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// newQualityGateStubAI 构造只含分析路由的 AIService，chatFn 捕获 judge 收到的 prompt 供断言。
func newQualityGateStubAI(t *testing.T, chatFn func(messages []ai.Message) (string, error)) (*ai.AIService, *[]string) {
	t.Helper()
	var prompts []string
	stub := &verdictStubProvider{chatFn: func(messages []ai.Message) (string, error) {
		if len(messages) > 0 {
			prompts = append(prompts, messages[0].Content)
		}
		return chatFn(messages)
	}}
	svc := ai.NewAIService(&ai.AIConfig{
		Router: ai.RouterConfig{Routes: map[ai.TaskType]ai.Route{
			ai.TaskTypeAnalysis: {Provider: "stub"},
			ai.TaskTypeChat:     {Provider: "stub"},
		}},
	})
	svc.SetProviderForTesting("stub", stub)
	return svc, &prompts
}

const failingVerdictJSON = `{"relevant":true,"supported":false,"hallucinate":true,"confidence":0.8,"reason":"回复包含资料中不存在的具体事实"}`
const passingVerdictJSON = `{"relevant":true,"supported":true,"hallucinate":false,"confidence":0.9,"reason":"ok"}`

func metricsDelta(t *testing.T, fn func()) (advisory, skipped, rejected uint64) {
	t.Helper()
	before := service.GlobalAIReplyMetrics.Snapshot()
	fn()
	after := service.GlobalAIReplyMetrics.Snapshot()
	return after.QualityAdvisory - before.QualityAdvisory,
		after.QualitySkipped - before.QualitySkipped,
		after.QualityRejected - before.QualityRejected
}

// TestRunQualityGateSkipsEmptyEvidence 空 evidence 快速通道：judge 无判据时跳过核验，
// 不产生「可用资料为空」类噪声 verdict。
func TestRunQualityGateSkipsEmptyEvidence(t *testing.T) {
	chatFnCalled := false
	svc, _ := newQualityGateStubAI(t, func([]ai.Message) (string, error) {
		chatFnCalled = true
		return failingVerdictJSON, nil
	})
	e := &SmartReplyEngine{aiService: svc, qualityGateEnabled: true}
	input := &service.SmartReplyContext{Message: "随便聊聊", OriginalContent: "随便聊聊"}

	advisory, skipped, rejected := metricsDelta(t, func() {
		verdict, skip, err := e.runQualityGate(input, "一条普通闲聊回复", nil)
		if err != nil || verdict != nil || !skip {
			t.Fatalf("期望跳过核验, got verdict=%v skip=%v err=%v", verdict, skip, err)
		}
	})
	if chatFnCalled {
		t.Fatal("空 evidence 不应调用 judge")
	}
	if skipped != 1 || advisory != 0 || rejected != 0 {
		t.Fatalf("计数错误: advisory=%d skipped=%d rejected=%d", advisory, skipped, rejected)
	}

	// 有工具结果时不算空 evidence，照常核验。
	svc2, prompts := newQualityGateStubAI(t, func([]ai.Message) (string, error) {
		return failingVerdictJSON, nil
	})
	e2 := &SmartReplyEngine{aiService: svc2, qualityGateEnabled: true}
	verdict, skip, err := e2.runQualityGate(input, "回复", []string{"工具查询结果: 明日有会"})
	if skip || err != nil || verdict == nil {
		t.Fatalf("有工具结果应照常核验, got skip=%v err=%v", skip, err)
	}
	if len(*prompts) != 1 || !strings.Contains((*prompts)[0], "工具查询结果: 明日有会") {
		t.Fatalf("judge prompt 应包含工具结果: %q", *prompts)
	}
}

// TestValidateReplyBeforeSendAdvisory @AI 默认旁路放行：核验不通过也发送正文，
// 仅记 QualityAdvisory + 失败案例，verdict 理由作为 qualityNote 随 Extra 落库。
func TestValidateReplyBeforeSendAdvisory(t *testing.T) {
	svc, prompts := newQualityGateStubAI(t, func([]ai.Message) (string, error) {
		return failingVerdictJSON, nil
	})
	e := &SmartReplyEngine{aiService: svc, qualityGateEnabled: true, mentionGateBlock: false}
	// KnowledgeCtx 非空 → 走真实核验分支（不触发空 evidence 跳过）。
	input := &service.SmartReplyContext{Message: "问题", KnowledgeCtx: "知识库资料内容"}

	var advisory, skipped, rejected uint64
	var send bool
	var note string
	advisory, skipped, rejected = metricsDelta(t, func() {
		send, note = e.validateReplyBeforeSend(input, "AI回复正文", nil)
	})
	if !send {
		t.Fatal("@AI 旁路模式应放行发送")
	}
	if note != "回复包含资料中不存在的具体事实" {
		t.Fatalf("qualityNote 应为 verdict 理由, got=%q", note)
	}
	if advisory != 1 || skipped != 0 || rejected != 0 {
		t.Fatalf("计数错误: advisory=%d skipped=%d rejected=%d", advisory, skipped, rejected)
	}
	// evidence 同源 sanity：judge 必须看到生成侧的知识上下文。
	if len(*prompts) != 1 || !strings.Contains((*prompts)[0], "知识库资料内容") {
		t.Fatalf("judge prompt 应包含生成侧知识上下文: %q", *prompts)
	}
}

// TestValidateReplyBeforeSendMentionBlock mention 阻断配置开启时恢复旧行为：拦截发送。
func TestValidateReplyBeforeSendMentionBlock(t *testing.T) {
	svc, _ := newQualityGateStubAI(t, func([]ai.Message) (string, error) {
		return failingVerdictJSON, nil
	})
	e := &SmartReplyEngine{
		aiService: svc, qualityGateEnabled: true, mentionGateBlock: true,
		messageSender: &WebSocketMessageSender{}, // hub 为 nil 时 NotifyReplyDecision 为 no-op
	}
	input := &service.SmartReplyContext{Message: "问题", KnowledgeCtx: "知识库资料内容"}

	var advisory, rejected uint64
	var send bool
	advisory, _, rejected = metricsDelta(t, func() {
		send, _ = e.validateReplyBeforeSend(input, "AI回复正文", nil)
	})
	if send {
		t.Fatal("阻断模式应拦截发送")
	}
	if rejected != 1 || advisory != 0 {
		t.Fatalf("计数错误: advisory=%d rejected=%d", advisory, rejected)
	}
}

// TestValidateReplyBeforeSendPasses 核验通过照常放行，无旁路计数。
func TestValidateReplyBeforeSendPasses(t *testing.T) {
	svc, _ := newQualityGateStubAI(t, func([]ai.Message) (string, error) {
		return passingVerdictJSON, nil
	})
	e := &SmartReplyEngine{aiService: svc, qualityGateEnabled: true}
	input := &service.SmartReplyContext{Message: "问题", KnowledgeCtx: "知识库资料内容"}

	var advisory, skipped, rejected uint64
	var send bool
	advisory, skipped, rejected = metricsDelta(t, func() {
		send, _ = e.validateReplyBeforeSend(input, "AI回复正文", nil)
	})
	if !send {
		t.Fatal("核验通过应放行")
	}
	if advisory != 0 || skipped != 0 || rejected != 0 {
		t.Fatalf("核验通过不应产生任何质量门计数: advisory=%d skipped=%d rejected=%d", advisory, skipped, rejected)
	}
}
