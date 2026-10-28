package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// reply_turn_exec_test.go 覆盖 ExecuteTurn* 的集成缝隙（不是决策纯函数）：
// 工具是否真的传下去、降级路径是否走对、失败是否原样上抛。

// stubTool 最小工具实现：只为让注册表里有一个真实的工具定义可断言。
type stubTool struct{ name string }

func (s stubTool) Name() string        { return s.name }
func (s stubTool) Description() string { return "stub tool" }
func (s stubTool) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}
func (s stubTool) Execute(map[string]interface{}, *ai.CallerContext) (interface{}, error) {
	return "ok", nil
}

// execTurnProvider 可配置行为的 provider：分别记录视觉通道与工具通道的调用。
type execTurnProvider struct {
	visionCalls []recordedCall
	toolStream  []recordedCall
	toolChat    []recordedCall

	visionErr     error
	visionReply   string
	actionReply   string
	toolStreamErr error
}

var _ ai.Provider = (*execTurnProvider)(nil)

func (p *execTurnProvider) Name() string { return "exec-turn" }
func (p *execTurnProvider) Chat(messages []ai.Message) (string, error) {
	p.visionCalls = append(p.visionCalls, newRecordedCall(messages, nil))
	return p.visionReply, p.visionErr
}
func (p *execTurnProvider) ChatStream([]ai.Message, func(ai.StreamChunk) error) error { return nil }
func (p *execTurnProvider) ChatStreamWithContext(_ context.Context, messages []ai.Message, onChunk func(ai.StreamChunk) error) error {
	p.visionCalls = append(p.visionCalls, newRecordedCall(messages, nil))
	if p.visionErr != nil {
		return p.visionErr
	}
	return onChunk(ai.StreamChunk{Content: p.visionReply})
}
func (p *execTurnProvider) Embedding(string) ([]float32, error) { return nil, nil }
func (p *execTurnProvider) SupportsEmbedding() bool             { return false }
func (p *execTurnProvider) ChatWithTools(messages []ai.Message, tools []ai.ToolDef) (*ai.ChatResponse, error) {
	p.toolChat = append(p.toolChat, newRecordedCall(messages, tools))
	return &ai.ChatResponse{Content: p.actionReply}, nil
}
func (p *execTurnProvider) ChatStreamWithTools(_ context.Context, messages []ai.Message, tools []ai.ToolDef, onChunk func(ai.StreamChunk) error) error {
	p.toolStream = append(p.toolStream, newRecordedCall(messages, tools))
	if p.toolStreamErr != nil {
		return p.toolStreamErr
	}
	return onChunk(ai.StreamChunk{Content: p.actionReply})
}
func (p *execTurnProvider) IsConfigured() bool           { return true }
func (p *execTurnProvider) WithModel(string) ai.Provider { return p }

func toolNames(tools []ai.ToolDef) []string {
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		names = append(names, t.Name)
	}
	return names
}

// newExecTurnGraph 构造带/不带视觉路由的 graph；注册两个真实工具供断言。
func newExecTurnGraph(t *testing.T, withVisionRoute bool) (*SmartReplyGraph, *execTurnProvider, *model.Conversation) {
	t.Helper()
	db := setupServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.SystemConfig{}))
	prevDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prevDB })

	conv := model.Conversation{Type: "group"}
	require.NoError(t, db.Create(&conv).Error)

	routes := map[ai.TaskType]ai.Route{ai.TaskTypeChat: {Provider: "openai", Model: "chat-model"}}
	if withVisionRoute {
		routes[ai.TaskTypeVision] = ai.Route{Provider: "openai", Model: "vision-model"}
	}
	aiSvc := ai.NewAIService(&ai.AIConfig{Router: ai.RouterConfig{Routes: routes}})
	prov := &execTurnProvider{visionReply: "图里是一个任务清单", actionReply: "已为你创建 2 个待办"}
	aiSvc.SetProviderForTesting("openai", prov)

	registry := ai.NewToolRegistry(nil)
	// 第三个工具不在计划里：用来证明"白名单为空=全部放行"的兜底不会掩盖计划未生效。
	for _, name := range []string{"list_tasks", "search_knowledge", "group_management"} {
		require.NoError(t, registry.RegisterTool(stubTool{name: name}))
	}
	aiSvc.SetToolRegistry(registry)

	g := &SmartReplyGraph{db: db, aiService: aiSvc, thresholdSvc: NewAiThresholdService(db)}
	return g, prov, &conv
}

func execImageInput(convID uint) *SmartReplyContext {
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

func execVisionPlan() ReplyTurnPlan {
	return PlanGroupTurn(TurnInput{
		Intent:             &ai.MessageIntent{Type: "query"},
		Question:           "把图里的待办建一下",
		HasImage:           true,
		GroupTools:         []string{"list_tasks", "search_knowledge"},
		VisionDisableTools: true,
		QualityGateEnabled: true,
	})
}

// TestExecuteTurn_SecondSegmentCarriesPlannedTools 两段式第二段必须真的带上计划里的工具
// 定义（此前测试里工具表为空，只验证了"走了工具通道"，等于没验证能力）。
func TestExecuteTurn_SecondSegmentCarriesPlannedTools(t *testing.T) {
	g, prov, conv := newExecTurnGraph(t, true)

	streamed, err := g.ExecuteTurnStream(context.Background(), execVisionPlan(), execImageInput(conv.ID), nil, nil,
		func(ai.StreamChunk) error { return nil })
	require.NoError(t, err)
	require.True(t, streamed)

	require.Len(t, prov.visionCalls, 1)
	assert.True(t, prov.visionCalls[0].hasImage, "第一段必须带图")
	assert.Empty(t, prov.visionCalls[0].tools, "第一段不得带工具")

	require.Len(t, prov.toolStream, 1, "第二段应走流式工具通道")
	assert.False(t, prov.toolStream[0].hasImage, "第二段不得再带图")
	assert.ElementsMatch(t, []string{"list_tasks", "search_knowledge"}, toolNames(prov.toolStream[0].tools),
		"第二段的工具定义必须来自回合计划（注册表里的 group_management 不得混入）")

	// 提示词里的能力自述必须与实际放行的工具一致（声明 ≠ 放行会让模型假调工具）。
	prompt := systemPromptOf(prov.toolStream[0].messages)
	assert.Contains(t, prompt, "list_tasks", "提示词应声明计划内的工具")
	assert.Contains(t, prompt, "search_knowledge")
}

// systemPromptOf 拼接一次调用里所有 system 消息，用于断言能力自述。
func systemPromptOf(messages []ai.Message) string {
	var sb strings.Builder
	for _, m := range messages {
		if m.Role == "system" {
			sb.WriteString(m.Content)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// TestExecuteTurn_VisionOnlyPromptClaimsNoTools 纯视觉单轮必须既不挂工具、也不在提示词里
// 声称有工具（否则模型会产出假工具调用或声称"我查了"）。
func TestExecuteTurn_VisionOnlyPromptClaimsNoTools(t *testing.T) {
	g, prov, conv := newExecTurnGraph(t, true) // 注册表里确有 list_tasks / search_knowledge

	plan := PlanGroupTurn(TurnInput{
		Intent:             &ai.MessageIntent{Type: "chat"},
		Question:           "图里是什么",
		HasImage:           true,
		GroupTools:         []string{"list_tasks", "search_knowledge"},
		VisionDisableTools: true,
		QualityGateEnabled: true,
	})
	require.Equal(t, TurnVisionOnly, plan.Mode)

	_, err := g.ExecuteTurnStream(context.Background(), plan, execImageInput(conv.ID), nil, nil,
		func(ai.StreamChunk) error { return nil })
	require.NoError(t, err)

	require.Len(t, prov.visionCalls, 1)
	prompt := systemPromptOf(prov.visionCalls[0].messages)
	assert.NotContains(t, prompt, "list_tasks", "视觉单轮不得声明工具能力")
	assert.NotContains(t, prompt, "search_knowledge")
}

// TestExecuteTurn_NoVisionRouteFallsBackToSingleSegment 未配置视觉路由时：
// 图片在 prepareInput 里降级为"看不到"文字，执行退回单段带工具，不得再打视觉模型。
func TestExecuteTurn_NoVisionRouteFallsBackToSingleSegment(t *testing.T) {
	g, prov, conv := newExecTurnGraph(t, false)

	streamed, err := g.ExecuteTurnStream(context.Background(), execVisionPlan(), execImageInput(conv.ID), nil, nil,
		func(ai.StreamChunk) error { return nil })
	require.NoError(t, err)
	require.True(t, streamed)

	assert.Empty(t, prov.visionCalls, "没有视觉路由时不得调用视觉通道（图片 base64 发给纯文本模型必 400）")
	require.Len(t, prov.toolStream, 1, "应退回单段带工具执行")
	assert.False(t, prov.toolStream[0].hasImage, "降级后不得再携带图片")

	var joined strings.Builder
	for _, m := range prov.toolStream[0].messages {
		joined.WriteString(m.Content)
	}
	assert.Contains(t, joined.String(), "没有可用的视觉模型", "应把降级原因交给模型如实说明")
}

// TestExecuteTurn_VisionPhaseErrorPropagates 第一段识图失败：错误原样上抛，不得继续调工具。
func TestExecuteTurn_VisionPhaseErrorPropagates(t *testing.T) {
	g, prov, conv := newExecTurnGraph(t, true)
	prov.visionErr = errors.New("vision upstream 500")

	streamed, err := g.ExecuteTurnStream(context.Background(), execVisionPlan(), execImageInput(conv.ID), nil, nil,
		func(ai.StreamChunk) error { return nil })
	assert.True(t, streamed, "视觉段已开始流出，调用方不应再整回合降级重跑")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vision upstream 500")
	assert.Empty(t, prov.toolStream, "识图失败不得继续执行动作")
	assert.Empty(t, prov.toolChat)
}

// TestExecuteTurnStream_SecondSegmentUnsupportedAppendsNonStream 第二段 Provider 不支持
// 流式 tool-call：内部降级非流式并把正文补发，不回到调用方的整回合降级（否则识图段会被重复生成）。
func TestExecuteTurnStream_SecondSegmentUnsupportedAppendsNonStream(t *testing.T) {
	g, prov, conv := newExecTurnGraph(t, true)
	prov.toolStreamErr = ai.ErrStreamingToolsNotSupported

	var out strings.Builder
	streamed, err := g.ExecuteTurnStream(context.Background(), execVisionPlan(), execImageInput(conv.ID), nil, nil,
		func(ch ai.StreamChunk) error { out.WriteString(ch.Content); return nil })
	require.NoError(t, err)
	assert.True(t, streamed)

	require.Len(t, prov.toolStream, 1, "先尝试流式工具通道")
	require.Len(t, prov.toolChat, 1, "不支持流式时内部降级非流式")
	require.Len(t, prov.visionCalls, 1, "识图段不得被重复调用")

	body := out.String()
	assert.Contains(t, body, "图里是一个任务清单", "识图正文应已上屏")
	assert.Contains(t, body, "已为你创建 2 个待办", "降级后的执行结果应补发")
	assert.Equal(t, 1, strings.Count(body, "图里是一个任务清单"), "识图正文不得重复")
}
