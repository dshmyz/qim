package service

import (
	"context"
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// reply_turn_regression_test.go 钉住两个曾经真实存在的缺陷：
//  1. 回合计划的 SkipKnowledge 必须在 prepareInput 之前生效，否则管理指令仍会跑检索；
//  2. 计划未授权任何工具时不得把「空白名单」当成「全部放行」下发给模型。

type recordingKnowledge struct{ calls int }

func (r *recordingKnowledge) BuildContext(string, uint) string { r.calls++; return "" }
func (r *recordingKnowledge) BuildContextWithSources(string, uint, int) (string, []KnowledgeSource) {
	r.calls++
	return "【知识库】无关内容", []KnowledgeSource{{Title: "无关文档", Source: "knowledge"}}
}

// TestPreparedHistory_CommandSkipsKnowledgeRetrieval 管理指令（SkipKnowledge=true）不得触发
// 知识/记忆检索：prepareInput 依赖 input.SkipKnowledge 决定是否检索，计划必须在此之前落定。
func TestPreparedHistory_CommandSkipsKnowledgeRetrieval(t *testing.T) {
	db := setupServiceTestDB(t)
	prevDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prevDB })

	conv := model.Conversation{Type: "group"}
	require.NoError(t, db.Create(&conv).Error)
	require.NoError(t, db.Create(&model.Group{
		ConversationID: conv.ID, GroupType: "group", Name: "测试群", CreatorID: 1,
	}).Error)

	kb := &recordingKnowledge{}
	g := NewSmartReplyGraph(ai.NewAIService(&ai.AIConfig{}), db, kb, nil, nil, nil)

	plan := PlanGroupTurn(TurnInput{
		Intent:             &ai.MessageIntent{Type: "command"},
		Question:           "把张三踢出群",
		GroupTools:         []string{"group_management"},
		VisionDisableTools: true,
		QualityGateEnabled: true,
	})
	require.True(t, plan.SkipKnowledge, "前置：管理指令计划应跳过知识注入")

	input := &SmartReplyContext{ConversationID: conv.ID, UserID: 1, Message: "把张三踢出群", IsAIMention: true}
	_, _, err := g.preparedHistory(input, plan)
	require.NoError(t, err)

	assert.Zero(t, kb.calls, "管理指令不得触发知识检索（SkipKnowledge 必须在 prepareInput 之前生效）")
	assert.Empty(t, input.KnowledgeCtx, "管理指令不应注入知识上下文")
}

// TestExecuteTurn_EmptyPlannedToolsExposesNothing 计划里没有工具（群作用域为空）时，
// ReAct 不得把注册表里的工具全部下发给模型：空白名单必须解释为「一个都不给」。
func TestExecuteTurn_EmptyPlannedToolsExposesNothing(t *testing.T) {
	g, prov, conv := newExecTurnGraph(t, true) // 注册表里确有 list_tasks / search_knowledge / group_management

	plan := PlanGroupTurn(TurnInput{
		Intent:             &ai.MessageIntent{Type: "query"},
		Question:           "我周三有什么会",
		GroupTools:         nil, // 群作用域为空
		VisionDisableTools: true,
		QualityGateEnabled: true,
	})
	require.Equal(t, TurnDirectChat, plan.Mode, "群作用域无工具时应退化为直答，而不是进空 ReAct")
	require.False(t, plan.WantsTools(), "前置：计划未授权任何工具")

	_, err := g.ExecuteTurn(context.Background(), plan,
		&SmartReplyContext{ConversationID: conv.ID, UserID: 1, Message: "我周三有什么会"})
	require.NoError(t, err)

	assert.Empty(t, prov.toolChat, "未授权工具时不得进入带工具通道")
	assert.Empty(t, prov.toolStream, "未授权工具时不得进入流式带工具通道")
}

// TestExecuteTurn_DefenseAgainstEmptyAllowlist 即使绕开计划直接构造 ToolReAct 空工具，
// 执行层也不得把空 allowlist 交给 ReAct（纵深防御）。
func TestExecuteTurn_DefenseAgainstEmptyAllowlist(t *testing.T) {
	g, prov, conv := newExecTurnGraph(t, true)
	plan := ReplyTurnPlan{Mode: TurnToolReAct, Tools: nil, Delivery: DeliveryLive, Gate: GateOff}

	streamed, err := g.ExecuteTurnStream(context.Background(), plan,
		&SmartReplyContext{ConversationID: conv.ID, UserID: 1, Message: "我周三有什么会"}, nil, nil,
		func(ai.StreamChunk) error { return nil })
	require.NoError(t, err)
	require.True(t, streamed)
	assert.Empty(t, prov.toolStream, "空工具计划不得进入 ReAct（空 allowlist = 全部放行）")
}
