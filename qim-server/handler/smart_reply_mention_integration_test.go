package handler

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// smart_reply_mention_integration_test.go 覆盖 @AI 提及的整条链路（此前只能手点验证）：
// 意图 → 回合计划（带图两段式）→ 识图流式上屏 → 带工具执行 → 交付 → 异步核验存疑 → 标注推送与落库。
// 发送端已抽成 AIResponseSender，这里注入假实现收集实际下发的状态/正文/标注。

const integrationReplyMsgID = 9001

type integrationTool struct{ name string }

func (t integrationTool) Name() string        { return t.name }
func (t integrationTool) Description() string { return "集成测试工具" }
func (t integrationTool) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}
func (t integrationTool) Execute(map[string]interface{}, *ai.CallerContext) (interface{}, error) {
	return "任务清单：周五复盘会", nil
}

// integrationProvider 模拟两段式所需的模型行为：
// 视觉段回识图正文；工具段第一步返回 tool_call，第二步返回最终答复；Chat 用作核验 judge。
type integrationProvider struct {
	mu sync.Mutex

	visionCalls  int
	visionImage  bool
	visionEnd    time.Time // 识图段真正结束（provider 返回）的时刻，用于验证"首字早于段结束"
	toolSteps    int
	toolImage    bool
	toolNames    []string
	judgePrompts []string
}

var _ ai.Provider = (*integrationProvider)(nil)

func (p *integrationProvider) Name() string { return "integration" }
func (p *integrationProvider) Chat(messages []ai.Message) (string, error) {
	p.mu.Lock()
	if len(messages) > 0 {
		p.judgePrompts = append(p.judgePrompts, messages[0].Content)
	}
	p.mu.Unlock()
	// 核验判定：不通过（触发存疑标注）
	return `{"relevant":true,"supported":false,"hallucinate":true,"confidence":0.8,"reason":"回复包含资料中不存在的具体事实"}`, nil
}
func (p *integrationProvider) ChatStream([]ai.Message, func(ai.StreamChunk) error) error { return nil }
func (p *integrationProvider) ChatStreamWithContext(_ context.Context, messages []ai.Message, onChunk func(ai.StreamChunk) error) error {
	p.mu.Lock()
	p.visionCalls++
	p.visionImage = messagesContainImage(messages)
	p.mu.Unlock()

	if err := onChunk(ai.StreamChunk{Content: "图里是一张成员列表"}); err != nil {
		return err
	}
	// 拉开两帧间隔：只有"边流边发"的实现才可能在视觉段结束前把首帧送达客户端
	time.Sleep(60 * time.Millisecond)
	if err := onChunk(ai.StreamChunk{Content: "（共 3 人）"}); err != nil {
		return err
	}
	p.mu.Lock()
	p.visionEnd = time.Now()
	p.mu.Unlock()
	return nil
}
func (p *integrationProvider) ChatWithTools([]ai.Message, []ai.ToolDef) (*ai.ChatResponse, error) {
	return &ai.ChatResponse{Content: "已为你创建 2 个待办"}, nil
}
func (p *integrationProvider) ChatStreamWithTools(_ context.Context, messages []ai.Message, tools []ai.ToolDef, onChunk func(ai.StreamChunk) error) error {
	p.mu.Lock()
	p.toolSteps++
	step := p.toolSteps
	p.toolImage = messagesContainImage(messages)
	p.toolNames = make([]string, 0, len(tools))
	for _, t := range tools {
		p.toolNames = append(p.toolNames, t.Name)
	}
	p.mu.Unlock()

	if step == 1 {
		return onChunk(ai.StreamChunk{ToolCalls: []ai.ToolCallDelta{
			{Index: 0, ID: "call_1", Name: "list_tasks", Arguments: "{}"},
		}})
	}
	return onChunk(ai.StreamChunk{Content: "已为你创建 2 个待办"})
}
func (p *integrationProvider) Embedding(string) ([]float32, error) { return nil, nil }
func (p *integrationProvider) SupportsEmbedding() bool             { return false }
func (p *integrationProvider) IsConfigured() bool                  { return true }
func (p *integrationProvider) WithModel(string) ai.Provider        { return p }

func messagesContainImage(messages []ai.Message) bool {
	for _, m := range messages {
		if m.ImageURL != "" || len(m.ImageURLs) > 0 {
			return true
		}
	}
	return false
}

// fakeAISender 记录实际下发的状态文案 / 正文块 / 存疑标注。
type fakeAISender struct {
	mu       sync.Mutex
	started  int
	statuses []string
	chunks   []string
	quality  []service.QualityAnnotation
	toolEvts int
	finished bool
	msg      *model.Message
	// firstChunkAt 首个正文块送达客户端的时刻：用于验证"首字在流中即上屏"（而不是整段缓冲）。
	firstChunkAt time.Time
	decisions    []SmartReplyDecision
}

var _ AIResponseSender = (*fakeAISender)(nil)

func (f *fakeAISender) NotifyReplyStarted(uint, string) {
	f.mu.Lock()
	f.started++
	f.mu.Unlock()
}
func (f *fakeAISender) NotifyReplyStartedStatus(_ uint, _ string, status string) {
	f.mu.Lock()
	f.statuses = append(f.statuses, status)
	f.mu.Unlock()
}
func (f *fakeAISender) NotifyReplyDecision(_ uint, decision SmartReplyDecision) {
	f.mu.Lock()
	f.decisions = append(f.decisions, decision)
	f.mu.Unlock()
}
func (f *fakeAISender) SendAIMessage(uint, string, string, map[string]interface{}) error {
	return nil
}
func (f *fakeAISender) SendStreamingAIMessage(convID uint, _ string) (func(string) error, func() *model.Message, func() *model.Message, error) {
	msg := &model.Message{ID: integrationReplyMsgID, ConversationID: convID, Type: "text", Origin: "assistant"}
	f.mu.Lock()
	f.msg = msg
	f.mu.Unlock()
	send := func(c string) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.firstChunkAt.IsZero() {
			f.firstChunkAt = time.Now()
		}
		f.chunks = append(f.chunks, c)
		msg.Content += c
		return nil
	}
	return send,
		func() *model.Message { return msg },
		func() *model.Message { f.mu.Lock(); f.finished = true; f.mu.Unlock(); return msg },
		nil
}
func (f *fakeAISender) SendToolCallEvent(uint, uint, service.ToolCallRecord) {
	f.mu.Lock()
	f.toolEvts++
	f.mu.Unlock()
}
func (f *fakeAISender) BroadcastQualityUpdate(_ uint, _ uint, ann service.QualityAnnotation) {
	f.mu.Lock()
	f.quality = append(f.quality, ann)
	f.mu.Unlock()
}

func (f *fakeAISender) snapshot() (started int, statuses, chunks []string, quality []service.QualityAnnotation, toolEvts int, finished bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.started, append([]string(nil), f.statuses...), append([]string(nil), f.chunks...),
		append([]service.QualityAnnotation(nil), f.quality...), f.toolEvts, f.finished
}

func (f *fakeAISender) firstChunkTime() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.firstChunkAt
}

func (f *fakeAISender) decisionSnapshot() []SmartReplyDecision {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]SmartReplyDecision(nil), f.decisions...)
}

// stubQuotedReader 让 prepareInput 能把被引用图片读成 data URL（真实路径，不绕过）。
type stubQuotedReader struct{}

func (stubQuotedReader) ExtractTextForContext(uint) (string, string, error) {
	return "", "", assert.AnError
}
func (stubQuotedReader) ImageURLForContext(uint) (string, string, error) {
	return "a.png", "data:image/png;base64,xxx", nil
}

func waitForQuality(t *testing.T, sender *fakeAISender, within time.Duration) service.QualityAnnotation {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		_, _, _, quality, _, _ := sender.snapshot()
		if len(quality) > 0 {
			return quality[0]
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s 内未收到核验存疑标注", within)
	return service.QualityAnnotation{}
}

// integrationHarness 统一装配：内存库（强制单连接——sqlite :memory: 每个连接是独立库，
// 并发取到新连接会报 no such table）、假 provider/发送端、图片自引用消息与固定回复行。
type integrationHarness struct {
	db     *gorm.DB
	user   model.User
	conv   model.Conversation
	img    model.Message
	prov   *integrationProvider
	sender *fakeAISender
	engine *SmartReplyEngine
}

func newIntegrationHarness(t *testing.T, mentionGateBlock bool) *integrationHarness {
	t.Helper()
	db := setupHandlerTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	prevDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prevDB })

	user := model.User{Username: "asker", Nickname: "提问者"}
	require.NoError(t, db.Create(&user).Error)
	conv := model.Conversation{Type: "group"}
	require.NoError(t, db.Create(&conv).Error)

	// 触发消息本身是图片（自引用），内容格式与线上一致：{"id":文件ID,...,"name":...}
	img := model.Message{
		ID: 800, ConversationID: conv.ID, SenderID: user.ID, Type: "image",
		Content: `{"id":123,"url":"/static/a.png","name":"a.png"}`,
	}
	require.NoError(t, db.Create(&img).Error)
	// 流式回复将落在这一行（固定 ID 便于断言落库结果）
	require.NoError(t, db.Create(&model.Message{
		ID: integrationReplyMsgID, ConversationID: conv.ID, SenderID: user.ID,
		Type: "text", Content: "", Origin: "assistant",
	}).Error)

	prov := &integrationProvider{}
	aiSvc := ai.NewAIService(&ai.AIConfig{Router: ai.RouterConfig{Routes: map[ai.TaskType]ai.Route{
		ai.TaskTypeChat:     {Provider: "stub"},
		ai.TaskTypeVision:   {Provider: "stub"},
		ai.TaskTypeAnalysis: {Provider: "stub"},
	}}})
	aiSvc.SetProviderForTesting("stub", prov)
	registry := ai.NewToolRegistry(nil)
	require.NoError(t, registry.RegisterTool(integrationTool{name: "list_tasks"}))
	aiSvc.SetToolRegistry(registry)

	graph := service.NewSmartReplyGraph(aiSvc, db, nil, nil, nil, nil)
	graph.SetQuotedFileReader(stubQuotedReader{})

	sender := &fakeAISender{}
	engine := &SmartReplyEngine{
		aiService:          aiSvc,
		intentDetector:     ai.NewIntentDetector(nil),
		smartReplyGraph:    graph,
		messageSender:      sender,
		qualityGateEnabled: true,
		mentionGateBlock:   mentionGateBlock,
	}
	return &integrationHarness{db: db, user: user, conv: conv, img: img, prov: prov, sender: sender, engine: engine}
}

// TestAIMentionImageTwoStageAnnotatesQuality @AI + 图片 + 管理指令：
// 两段式（识图 → 带工具执行）→ 正文实时上屏 → 异步核验存疑 → 广播标注并落库。
func TestAIMentionImageTwoStageAnnotatesQuality(t *testing.T) {
	h := newIntegrationHarness(t, false) // 旁路：正文先发，核验事后标注
	e, sender, prov, db := h.engine, h.sender, h.prov, h.db
	user, conv, img := h.user, h.conv, h.img

	question := "把图里的人踢出群" // 命中 command 规则 → 带图两段式
	e.handleAIMentionWithGraph(user.ID, conv.ID, question, question, "AI助手", &img)

	started, statuses, chunks, _, toolEvts, finished := sender.snapshot()

	// —— 回合状态：首字前两段文案 ——
	assert.GreaterOrEqual(t, started, 1, "应推送「思考中」占位")
	assert.Equal(t, []string{"正在看图片…", "正在处理…"}, statuses, "两段式应按段下发状态")

	// —— 正文：识图结果与执行结果都在同一气泡里 ——
	body := strings.Join(chunks, "")
	assert.Contains(t, body, "图里是一张成员列表", "识图正文应实时上屏")
	assert.Contains(t, body, "（共 3 人）")
	assert.Contains(t, body, "已为你创建 2 个待办", "执行结果应续写")
	assert.True(t, finished, "流式消息应收尾")

	// —— 异步核验 → 存疑标注广播 ——
	ann := waitForQuality(t, sender, 3*time.Second)
	assert.Equal(t, "doubtful", ann.Status)
	assert.Contains(t, ann.Reason, "资料中不存在", "标注原因应来自核验判定")

	// —— 两段式：图片只进视觉段，工具只进执行段 ——
	prov.mu.Lock()
	assert.Equal(t, 1, prov.visionCalls, "识图段应调用一次")
	assert.True(t, prov.visionImage, "识图段必须携带图片")
	assert.False(t, prov.toolImage, "执行段不得再携带图片")
	assert.Equal(t, 2, prov.toolSteps, "执行段应为 工具调用 + 最终答复 两步")
	// 首字时延：首帧必须在识图段结束前就已送达（否则说明又退回了"整段缓冲后一次性发"）
	assert.False(t, prov.visionEnd.IsZero(), "识图段应已结束")
	assert.False(t, sender.firstChunkTime().IsZero(), "应收到正文块")
	assert.True(t, sender.firstChunkTime().Before(prov.visionEnd),
		"首字应在识图段结束前上屏（流式直发），实际首字=%v 段结束=%v", sender.firstChunkTime(), prov.visionEnd)
	assert.ElementsMatch(t, []string{"list_tasks"}, prov.toolNames, "执行段应携带计划内工具")
	assert.NotEmpty(t, prov.judgePrompts, "应发生一次质量核验")
	prov.mu.Unlock()
	assert.GreaterOrEqual(t, toolEvts, 1, "工具调用应有卡片事件")

	// —— 标注落库：既有 Extra（工具卡）保留，quality 合并进去 ——
	var saved model.Message
	require.NoError(t, db.First(&saved, integrationReplyMsgID).Error)
	assert.Contains(t, saved.Extra, "quality", "标注应写入消息 Extra")
	assert.Contains(t, saved.Extra, "doubtful")
	assert.Contains(t, saved.Extra, "tool_calls", "合并标注不得覆盖工具调用记录")
}

// TestAIMentionGateBlockWithholdsReply 阻断模式（ai.reply_quality_gate_mention_block=1）：
// 正文必须缓冲到核验结束，不通过则一个字都不发，只把原因通知给用户。
func TestAIMentionGateBlockWithholdsReply(t *testing.T) {
	h := newIntegrationHarness(t, true) // DeliveryBuffered

	question := "把图里的人踢出群"
	h.engine.handleAIMentionWithGraph(h.user.ID, h.conv.ID, question, question, "AI助手", &h.img)

	_, statuses, chunks, quality, _, _ := h.sender.snapshot()
	assert.Equal(t, []string{"正在看图片…", "正在处理…"}, statuses, "阶段状态仍应下发")
	assert.Empty(t, chunks, "阻断模式下未通过核验的正文不得上屏")
	assert.Empty(t, quality, "阻断路径不走异步存疑标注（正文根本没发）")

	decisions := h.sender.decisionSnapshot()
	require.Len(t, decisions, 1, "阻断时应通知用户原因，而不是静默")
	assert.Equal(t, "quality_rejected", decisions[0].ReasonCode)
}
