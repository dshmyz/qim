package handler

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitReplyChunks(t *testing.T) {
	// 空串 → 无分块
	assert.Empty(t, splitReplyChunks(""))

	// 短文本不切
	in := "今天上海多云，26℃。"
	out := splitReplyChunks(in)
	require.Len(t, out, 1)
	assert.Equal(t, in, out[0])

	// 拼接回原样（不丢字）
	long := "第一句要点。第二句要点。第三句要点，继续第四句。第五句结尾。" + strings.Repeat("字", 300)
	out = splitReplyChunks(long)
	assert.NotEmpty(t, out)
	joined := ""
	for _, c := range out {
		joined += c
	}
	assert.Equal(t, long, joined, "分块拼接应等于原文，不得丢字")
}
