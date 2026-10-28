package handler

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// planFor 用最小输入跑一次回合计划（群作用域只放群管理工具，避免依赖 DB 白名单配置）。
func planFor(intent *ai.MessageIntent, question string) service.ReplyTurnPlan {
	return service.PlanGroupTurn(service.TurnInput{
		Intent:             intent,
		Question:           question,
		GroupTools:         []string{"group_management"},
		VisionDisableTools: true,
		QualityGateEnabled: true,
	})
}

// commandPlan 构造一个「管理指令」回合计划：带工具、跳过知识注入。
// 供群管理端到端测试直接驱动 ExecuteTurn（模拟 @AI 发起管理操作）。
func commandPlan(g *service.SmartReplyGraph, input *service.SmartReplyContext) service.ReplyTurnPlan {
	return service.PlanGroupTurn(service.TurnInput{
		Intent:             &ai.MessageIntent{Type: "command"},
		Question:           input.Message,
		GroupTools:         g.GroupTools(),
		VisionDisableTools: true,
		QualityGateEnabled: true,
	})
}

// TestIntentDetectorCommandDrivesPlan 端到端钉死：@AI 管理操作指令（踢人/加人/禁言等）
// 被 IntentDetector 识别为 command 意图，从而进入带工具路径并跳过知识注入；
// 普通寒暄则走直答、不挂工具。
// 这条测试保护"群聊 AI 助手代管成员"能力：一旦管理指令不再进工具路径，
// 或 IntentDetector 不再识别管理操作词，测试会失败。
func TestIntentDetectorCommandDrivesPlan(t *testing.T) {
	detector := ai.NewIntentDetector(nil)

	commands := []string{
		"把张三踢出群",
		"移除李四",
		"禁言王五",
		"邀请赵六加入",
		"设置管理员钱七",
		"取消管理员孙八",
	}
	for _, c := range commands {
		intent, err := detector.Detect(c, 1, 1)
		require.NoError(t, err, "Detect(%q)", c)
		plan := planFor(intent, c)
		assert.True(t, plan.WantsTools(), "Detect(%q) 应进入带工具路径，实际 mode=%s", c, plan.Mode)
		assert.True(t, plan.SkipKnowledge, "Detect(%q) 管理指令应跳过知识注入", c)
	}

	smallTalk := []string{"今天天气不错", "你好呀", "这个功能怎么用"}
	for _, c := range smallTalk {
		intent, err := detector.Detect(c, 1, 1)
		require.NoError(t, err, "Detect(%q)", c)
		plan := planFor(intent, c)
		assert.False(t, plan.SkipKnowledge, "Detect(%q) 非管理指令不应跳过知识注入", c)
		assert.NotEqual(t, service.TurnVisionThenAct, plan.Mode, "Detect(%q) 无图不应进入两段式", c)
	}
}
