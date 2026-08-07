package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCompactSources 覆盖分身回复「依据来源」下发前的压缩：nil 安全、按类型+标题去重、
// snippet 截断到 80 个字节-rune（保证 WS 载荷有界）。
func TestCompactSources(t *testing.T) {
	p := &AvatarWorkerPool{}

	// 空 / nil → 返回 nil，避免下发空 sources 字段
	assert.Nil(t, p.compactSources(nil))
	assert.Nil(t, p.compactSources([]AvatarSource{}))

	// 去重：同 type+title 仅保留一条；不同 type 同 title 不视为重复
	in := []AvatarSource{
		{Type: "note", Title: "会议", Snippet: "a"},
		{Type: "note", Title: "会议", Snippet: "b"}, // 重复（同 type+title）
		{Type: "group", Title: "会议", Snippet: "c"}, // 不同 type，保留
	}
	out := p.compactSources(in)
	assert.Len(t, out, 2, "同 type+title 应去重")
	assert.Equal(t, "a", out[0].Snippet, "保留首次命中的 snippet")
	assert.Equal(t, "group", out[1].Type)

	// snippet 截断到 80 个字符（按 rune，兼容中文）
	long := strings.Repeat("依", 200)
	trunc := p.compactSources([]AvatarSource{{Type: "memory", Snippet: long}})
	assert.Len(t, []rune(trunc[0].Snippet), 80, "snippet 应截断到 80 字符")
}
