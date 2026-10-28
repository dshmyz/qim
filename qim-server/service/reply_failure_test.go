package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBuildFailureReply 失败三分法话术单一来源：
// 看不了 / 查不到 / 用不了 各自给出不同的、可操作的建议。
func TestBuildFailureReply(t *testing.T) {
	t.Run("看不了-过大", func(t *testing.T) {
		got := BuildFailureReply(FailureMediaUnreadable, FailureDetail{
			MediaType: "图片", Name: "团建照片.png", TooLarge: true,
		})
		assert.Contains(t, got, "团建照片.png")
		assert.Contains(t, got, "太大")
		assert.Contains(t, got, "压缩", "应给出可操作建议")
	})

	t.Run("看不了-读不出", func(t *testing.T) {
		got := BuildFailureReply(FailureMediaUnreadable, FailureDetail{
			MediaType: "文件", Name: "报告.doc",
		})
		assert.Contains(t, got, "报告.doc")
		assert.Contains(t, got, "读不出来")
		assert.NotContains(t, got, "太大", "格式失败不应说成体积问题")
	})

	t.Run("查不到-点名失败工具", func(t *testing.T) {
		got := BuildFailureReply(FailureNoResult, FailureDetail{FailedTools: []string{"查询任务", "日历"}})
		assert.Contains(t, got, "查询任务")
		assert.Contains(t, got, "日历")
		assert.Contains(t, got, "线索", "应提示补充线索")
	})

	t.Run("查不到-无结果", func(t *testing.T) {
		got := BuildFailureReply(FailureNoResult, FailureDetail{})
		assert.Contains(t, got, "换个更具体的问法")
		assert.NotContains(t, got, "服务异常", "不应对用户声称服务故障")
	})

	t.Run("用不了", func(t *testing.T) {
		got := BuildFailureReply(FailureModelUnavailable, FailureDetail{})
		assert.Contains(t, got, "服务异常")
		assert.Contains(t, got, "请联系管理员", "应给出升级路径")
	})

	t.Run("未知类型退化为查不到", func(t *testing.T) {
		got := BuildFailureReply(FailureKind("unknown"), FailureDetail{})
		assert.Equal(t, BuildFailureReply(FailureNoResult, FailureDetail{}), got)
	})
}

// TestBuildMediaUnreadablePrompt 给模型看的提示与用户可见话术同源，且要求不编造内容。
func TestBuildMediaUnreadablePrompt(t *testing.T) {
	got := BuildMediaUnreadablePrompt("你引用了", "图片", "截图.png", true, "如对方另有问题请照常回答。")
	assert.Contains(t, got, "你引用了")
	assert.Contains(t, got, "图片消息「截图.png」")
	assert.Contains(t, got, "太大", "结论应来自统一话术")
	assert.Contains(t, got, "不要猜测或编造")
	assert.Contains(t, got, "如对方另有问题请照常回答")

	// 无文件名时不应出现空书名号
	noName := BuildMediaUnreadablePrompt("用户发送了", "语音", "", false, "")
	assert.NotContains(t, noName, "「」")
	assert.Contains(t, noName, "一条语音消息")
}
