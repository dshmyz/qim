package service

import (
	"context"
	"strings"
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordedCall 记录一次模型调用的关键特征：是否带图、是否带工具、消息内容。
type recordedCall struct {
	messages []ai.Message
	tools    []ai.ToolDef
	hasImage bool
}

func newRecordedCall(messages []ai.Message, tools []ai.ToolDef) recordedCall {
	call := recordedCall{messages: messages, tools: tools}
	for _, m := range messages {
		if m.ImageURL != "" || len(m.ImageURLs) > 0 {
			call.hasImage = true
		}
	}
	return call
}

// turnRecordingProvider 记录各调用通道，并按通道返回可区分的文本，供两段式顺序断言。
type turnRecordingProvider struct {
	visionCalls []recordedCall // 纯流式/纯补全（无工具）
	toolCalls   []recordedCall // 带工具（流式或非流式）
	visionReply string
	actionReply string
}

var _ ai.Provider = (*turnRecordingProvider)(nil)

func (p *turnRecordingProvider) Name() string { return "turn-recording" }
func (p *turnRecordingProvider) Chat(messages []ai.Message) (string, error) {
	p.visionCalls = append(p.visionCalls, newRecordedCall(messages, nil))
	return p.visionReply, nil
}
func (p *turnRecordingProvider) ChatStream([]ai.Message, func(ai.StreamChunk) error) error {
	return nil
}
func (p *turnRecordingProvider) ChatStreamWithContext(_ context.Context, messages []ai.Message, onChunk func(ai.StreamChunk) error) error {
	p.visionCalls = append(p.visionCalls, newRecordedCall(messages, nil))
	return onChunk(ai.StreamChunk{Content: p.visionReply})
}
func (p *turnRecordingProvider) Embedding(string) ([]float32, error) { return nil, nil }
func (p *turnRecordingProvider) SupportsEmbedding() bool             { return false }
func (p *turnRecordingProvider) ChatWithTools(messages []ai.Message, tools []ai.ToolDef) (*ai.ChatResponse, error) {
	p.toolCalls = append(p.toolCalls, newRecordedCall(messages, tools))
	return &ai.ChatResponse{Content: p.actionReply}, nil
}
func (p *turnRecordingProvider) ChatStreamWithTools(_ context.Context, messages []ai.Message, tools []ai.ToolDef, onChunk func(ai.StreamChunk) error) error {
	p.toolCalls = append(p.toolCalls, newRecordedCall(messages, tools))
	return onChunk(ai.StreamChunk{Content: p.actionReply})
}
func (p *turnRecordingProvider) IsConfigured() bool           { return true }
func (p *turnRecordingProvider) WithModel(string) ai.Provider { return p }

// newTurnTestGraph 构造带视觉路由的 graph + 记录型 provider + 一个群会话。
func newTurnTestGraph(t *testing.T) (*SmartReplyGraph, *turnRecordingProvider, *model.Conversation) {
	t.Helper()
	db := setupServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.SystemConfig{}))
	prevDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prevDB })

	conv := model.Conversation{Type: "group"}
	require.NoError(t, db.Create(&conv).Error)

	aiSvc := ai.NewAIService(&ai.AIConfig{Router: ai.RouterConfig{
		Routes: map[ai.TaskType]ai.Route{
			ai.TaskTypeVision: {Provider: "openai", Model: "vision-model"},
			ai.TaskTypeChat:   {Provider: "openai", Model: "chat-model"},
		},
	}})
	prov := &turnRecordingProvider{visionReply: "图里是一张任务清单", actionReply: "已为你创建 2 个待办"}
	aiSvc.SetProviderForTesting("openai", prov)
	// 注册工具表：否则带工具路径会整体降级为普通补全，测不到"第二段真的走工具通道"。
	aiSvc.SetToolRegistry(ai.NewToolRegistry(nil))

	g := &SmartReplyGraph{db: db, aiService: aiSvc, thresholdSvc: NewAiThresholdService(db)}
	return g, prov, &conv
}

func imageTurnInput(convID uint) *SmartReplyContext {
	return &SmartReplyContext{
		ConversationID: convID,
		UserID:         1,
		Message:        "把图里的待办建一下",
		IsAIMention:    true,
		Quoted: &QuotedContext{
			Kind:     QuotedImage,
			Name:     "a.png",
			ImageURL: "data:image/png;base64,xxx",
		},
	}
}

func visionPlan() ReplyTurnPlan {
	return PlanGroupTurn(TurnInput{
		Intent:             &ai.MessageIntent{Type: "query"},
		Question:           "把图里的待办建一下",
		HasImage:           true,
		GroupTools:         []string{"search_knowledge", "list_tasks"},
		VisionDisableTools: true,
		QualityGateEnabled: true,
	})
}

// TestExecuteTurn_VisionOnly 带图不加行动意图：单次视觉调用，不挂工具。
func TestExecuteTurn_VisionOnly(t *testing.T) {
	g, prov, conv := newTurnTestGraph(t)
	plan := PlanGroupTurn(TurnInput{
		Intent:             &ai.MessageIntent{Type: "chat"},
		Question:           "图里是什么",
		HasImage:           true,
		GroupTools:         []string{"list_tasks"},
		VisionDisableTools: true,
		QualityGateEnabled: true,
	})
	require.Equal(t, TurnVisionOnly, plan.Mode)

	var got strings.Builder
	streamed, err := g.ExecuteTurnStream(context.Background(), plan, imageTurnInput(conv.ID), nil, nil,
		func(ch ai.StreamChunk) error { got.WriteString(ch.Content); return nil })
	require.NoError(t, err)
	assert.True(t, streamed)
	assert.Equal(t, "图里是一张任务清单", got.String())
	require.Len(t, prov.visionCalls, 1, "视觉单轮只应调用一次模型")
	assert.True(t, prov.visionCalls[0].hasImage, "视觉调用必须携带图片")
	assert.Empty(t, prov.toolCalls, "视觉单轮不得走带工具通道")
}

// TestExecuteTurn_VisionThenAct 带图且有行动意图：第一段识图（带图、无工具），
// 第二段执行（无图、有工具），两段正文都上屏。
func TestExecuteTurn_VisionThenAct(t *testing.T) {
	g, prov, conv := newTurnTestGraph(t)
	plan := visionPlan()
	require.Equal(t, TurnVisionThenAct, plan.Mode)

	var got strings.Builder
	var statuses []string
	onStatus := func(s string) { statuses = append(statuses, s) }
	streamed, err := g.ExecuteTurnStream(context.Background(), plan, imageTurnInput(conv.ID), onStatus, nil,
		func(ch ai.StreamChunk) error { got.WriteString(ch.Content); return nil })
	require.NoError(t, err)
	assert.True(t, streamed)

	require.Len(t, prov.visionCalls, 1, "第一段应有一次视觉调用")
	assert.True(t, prov.visionCalls[0].hasImage, "第一段必须携带图片")

	require.Len(t, prov.toolCalls, 1, "第二段应有一次带工具调用")
	assert.False(t, prov.toolCalls[0].hasImage, "第二段不得再携带图片（图片与 tools 不同请求）")
	assert.NotEmpty(t, plan.Tools, "计划应为第二段准备工具白名单")

	assert.Contains(t, got.String(), "图里是一张任务清单", "识图结果应上屏")
	assert.Contains(t, got.String(), "已为你创建 2 个待办", "执行结果应续写在同一气泡")
	assert.Equal(t, []string{"正在看图片…", "正在处理…"}, statuses, "两段各下发一次状态")
}

// TestExecuteTurn_VisionThenActNonStream 非流式两段式（auto 路径）：
// 识图结果 + 执行结果拼接返回，顺序与流式一致。
func TestExecuteTurn_VisionThenActNonStream(t *testing.T) {
	g, prov, conv := newTurnTestGraph(t)
	reply, err := g.ExecuteTurn(context.Background(), visionPlan(), imageTurnInput(conv.ID))
	require.NoError(t, err)
	assert.Contains(t, reply, "图里是一张任务清单")
	assert.Contains(t, reply, "已为你创建 2 个待办")
	require.Len(t, prov.visionCalls, 1)
	require.Len(t, prov.toolCalls, 1)
	assert.False(t, prov.toolCalls[0].hasImage, "执行段不得携带图片")
}
