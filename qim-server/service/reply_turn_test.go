package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/stretchr/testify/assert"
)

// TestPlanGroupTurn 决策矩阵：模式 / 工具 / 投递 / 核验 / 状态全由纯函数决定。
// 这是群助手行为的单一来源，改动路由规则必须同步本表。
func TestPlanGroupTurn(t *testing.T) {
	groupTools := []string{"search_knowledge", "list_tasks", "search_messages"}
	base := TurnInput{
		Question:           "我周三有什么会",
		GroupTools:         groupTools,
		VisionDisableTools: true,
		QualityGateEnabled: true,
	}
	withIntent := func(typ, question string) TurnInput {
		in := base
		in.Intent = &ai.MessageIntent{Type: typ}
		in.Question = question
		return in
	}
	withImage := func(in TurnInput) TurnInput {
		in.HasImage = true
		return in
	}

	cases := []struct {
		name         string
		in           TurnInput
		wantMode     TurnMode
		wantTools    bool
		wantDelivery TurnDelivery
		wantGate     TurnGate
		wantSkipKnow bool
		wantExtGuide bool
		wantStatus   []string
	}{
		{
			name:       "带图 + 闲聊 → 纯视觉单轮",
			in:         withImage(withIntent("chat", "图里是什么")),
			wantMode:   TurnVisionOnly,
			wantStatus: []string{"正在看图片…"},
		},
		{
			name:       "带图 + 查询 → 两段式（先识图再执行）",
			in:         withImage(withIntent("query", "把图里的待办建一下")),
			wantMode:   TurnVisionThenAct,
			wantTools:  true,
			wantStatus: []string{"正在看图片…", "正在处理…"},
		},
		{
			name:         "带图 + 管理指令 → 两段式且跳过知识注入",
			in:           withImage(withIntent("command", "看下这张图再把张三踢了")),
			wantMode:     TurnVisionThenAct,
			wantTools:    true,
			wantSkipKnow: true,
			wantStatus:   []string{"正在看图片…", "正在处理…"},
		},
		{
			name: "带图但显式关闭 vision_disable_tools → 回退旧行为（带图挂工具）",
			in: func() TurnInput {
				in := withImage(withIntent("query", "图里是什么"))
				in.VisionDisableTools = false
				return in
			}(),
			wantMode:  TurnToolReAct,
			wantTools: true,
		},
		{
			name:       "无图小闲聊 → 直答不挂工具",
			in:         withIntent("chat", "你好"),
			wantMode:   TurnDirectChat,
			wantStatus: nil,
		},
		{
			name:      "无图但长问句（误判 chat）→ 仍保留工具",
			in:        withIntent("chat", "帮我总结一下这个群最近都在聊什么话题"),
			wantMode:  TurnToolReAct,
			wantTools: true,
		},
		{
			name:      "无图 + 查询 → 带工具 ReAct",
			in:        withIntent("query", "我周三有什么会"),
			wantMode:  TurnToolReAct,
			wantTools: true,
		},
		{
			name: "无工具作用域 + 查询 → 退化为直答（避免空白名单=全部放行）",
			in: func() TurnInput {
				in := withIntent("query", "我周三有什么会")
				in.GroupTools = nil
				return in
			}(),
			wantMode: TurnDirectChat,
		},
		{
			name: "无工具作用域 + 带图查询 → 退化为纯视觉单轮",
			in: func() TurnInput {
				in := withImage(withIntent("query", "图里有什么"))
				in.GroupTools = nil
				return in
			}(),
			wantMode:   TurnVisionOnly,
			wantStatus: []string{"正在看图片…"},
		},
		{
			name:      "意图缺失 → 保守走带工具",
			in:        base,
			wantMode:  TurnToolReAct,
			wantTools: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := PlanGroupTurn(tc.in)
			assert.Equal(t, tc.wantMode, got.Mode, "模式")
			assert.Equal(t, tc.wantTools, got.WantsTools(), "是否挂工具")
			assert.Equal(t, tc.wantExtGuide, got.ExternalGuide, "外部工具指引")
			assert.Equal(t, tc.wantSkipKnow, got.SkipKnowledge, "跳过知识注入")
			assert.Equal(t, tc.wantStatus, got.Status, "状态文案")
			if tc.wantTools {
				assert.Equal(t, groupTools, got.Tools, "工具集应等于群作用域工具")
			} else {
				assert.Empty(t, got.Tools, "不挂工具时不得放行任何工具")
			}
		})
	}
}

// TestPlanGroupTurn_GateAndDelivery 核验策略决定投递策略：
// 只有「核验可阻断」才需要缓冲，旁路/关闭时一律边流边发。
func TestPlanGroupTurn_GateAndDelivery(t *testing.T) {
	base := TurnInput{Question: "我周三有什么会", GroupTools: []string{"list_tasks"}}

	cases := []struct {
		name         string
		mutate       func(*TurnInput)
		wantGate     TurnGate
		wantDelivery TurnDelivery
	}{
		{"默认旁路 → 异步核验 + 边流边发", func(*TurnInput) {}, GateAsync, DeliveryLive},
		{"质量门关闭 → 不核验 + 边流边发", func(in *TurnInput) { in.QualityGateEnabled = false }, GateOff, DeliveryLive},
		{"阻断模式 → 发送前核验 + 缓冲", func(in *TurnInput) { in.MentionGateBlock = true }, GateBlock, DeliveryBuffered},
		{"auto 自动回复 → 恒阻断 + 缓冲", func(in *TurnInput) { in.Auto = true }, GateBlock, DeliveryBuffered},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := base
			in.QualityGateEnabled = true
			tc.mutate(&in)
			got := PlanGroupTurn(in)
			assert.Equal(t, tc.wantGate, got.Gate, "核验策略")
			assert.Equal(t, tc.wantDelivery, got.Delivery, "投递策略")
		})
	}
}

// TestPlanGroupTurn_ExternalGuide 外部工具指引只在「外部 MCP 启用且确有工具」时注入，
// 避免提示词声称能调不存在的工具。
func TestPlanGroupTurn_ExternalGuide(t *testing.T) {
	in := TurnInput{
		Intent:               &ai.MessageIntent{Type: "query"},
		Question:             "算一下 3.5 × 7",
		GroupTools:           nil, // 无任何工具
		ExternalToolsEnabled: true,
		VisionDisableTools:   true,
		QualityGateEnabled:   true,
	}
	got := PlanGroupTurn(in)
	assert.False(t, got.ExternalGuide, "无工具时不得注入外部工具指引")

	in.GroupTools = []string{"mcp_demo_calculator"}
	got = PlanGroupTurn(in)
	assert.True(t, got.ExternalGuide, "有外部工具且已启用时应注入指引")
}
