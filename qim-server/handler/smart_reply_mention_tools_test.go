package handler

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShouldUseToolsForMention 钉死：只有 command 意图走带工具路径，其他意图走流式纯文本。
func TestShouldUseToolsForMention(t *testing.T) {
	tests := []struct {
		name   string
		intent *ai.MessageIntent
		want   bool
	}{
		{"command 走带工具", &ai.MessageIntent{Type: "command", Confidence: 0.8}, true},
		{"chat 走流式", &ai.MessageIntent{Type: "chat"}, false},
		{"query 走流式", &ai.MessageIntent{Type: "query"}, false},
		{"alert 走流式", &ai.MessageIntent{Type: "alert"}, false},
		{"todo 走流式", &ai.MessageIntent{Type: "todo"}, false},
		{"nil 走流式", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ShouldUseToolsForMention(tt.intent))
		})
	}
}

// TestIntentDetectorCommandTriggersTools 端到端钉死：@AI 管理操作指令（踢人/加人/禁言等）
// 被 IntentDetector 识别为 command 意图，从而走带工具路径；普通聊天/咨询不走。
// 这条测试保护"群聊AI助手代管成员"能力：一旦 @AI 路径被改回不带工具的流式，
// 或 IntentDetector 不再识别管理操作词，测试会失败。
func TestIntentDetectorCommandTriggersTools(t *testing.T) {
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
		assert.True(t, ShouldUseToolsForMention(intent),
			"Detect(%q) 应识别为 command 走带工具，实际 type=%s", c, intent.Type)
	}

	nonCommands := []string{
		"今天天气不错",
		"你好呀",
		"这个功能怎么用",
	}
	for _, c := range nonCommands {
		intent, err := detector.Detect(c, 1, 1)
		require.NoError(t, err, "Detect(%q)", c)
		assert.False(t, ShouldUseToolsForMention(intent),
			"Detect(%q) 不应走带工具，实际 type=%s", c, intent.Type)
	}
}
