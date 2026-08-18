package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSnippetJudgeWindow(t *testing.T) {
	// 短正文原样返回
	assert.Equal(t, "短内容", snippetJudgeWindow("短内容", "q", 800))

	// 超长 + query 命中：窗口应包含 query 且加省略号
	long := strings.Repeat("无关心内容甲乙丙丁", 40) + "项目截止日期是3月15日" + strings.Repeat("后续无关内容子丑寅卯", 40)
	out := snippetJudgeWindow(long, "项目截止日期是3月15日", 200)
	assert.Len(t, []rune(out), 202, "窗口应约等于 maxRunes + 两侧省略号")
	assert.Contains(t, out, "项目截止日期是3月15日", "窗口应包含 query 命中处")
	assert.True(t, strings.HasPrefix(out, "…"), "窗口非从开头开始应有前置省略号")
	assert.True(t, strings.HasSuffix(out, "…"), "窗口未到结尾应有后置省略号")

	// query 未命中 → 回退取开头（含省略号结尾）
	out2 := snippetJudgeWindow(strings.Repeat("内容内容", 300), "完全不存在的词", 100)
	assert.Len(t, []rune(out2), 101)
	assert.True(t, strings.HasSuffix(out2, "…"))
	assert.False(t, strings.HasPrefix(out2, "…"), "取开头时不应有前置省略号")

	// query 命中在开头附近：窗口应从 0 开始（无前置省略号）
	head := "项目截止" + strings.Repeat("后续内容填充填充填充填充填充填充填充填充填充", 60)
	out3 := snippetJudgeWindow(head, "项目截止", 100)
	assert.False(t, strings.HasPrefix(out3, "…"), "query 在开头时窗口应从 0 开始")
	assert.Contains(t, out3, "项目截止")

	// 空 query → anchor 空 → 按取开头处理（不 panic）
	out4 := snippetJudgeWindow(strings.Repeat("内容内容", 300), "", 100)
	assert.Len(t, []rune(out4), 101)

	// 上下文感知 query（"历史行...\n当前提问：XXX"）：锚点应取提问部分而非开头的发送者名
	ctxQ := "Alice: 项目进度\nBob: 讨论中\n当前提问：项目截止日期是3月15日"
	long2 := strings.Repeat("填充内容甲乙丙丁", 3) + "项目截止日期是3月15日" + strings.Repeat("后续内容子丑寅卯", 30)
	out5 := snippetJudgeWindow(long2, ctxQ, 100)
	assert.Contains(t, out5, "项目截止日期是3月15日", "上下文感知 query 应锚定提问部分而非发送者名")
	assert.False(t, strings.HasPrefix(out5, "…"), "提问锚点命中在开头附近时窗口应从 0 开始")
}
