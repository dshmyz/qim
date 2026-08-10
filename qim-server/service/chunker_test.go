package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSplitBySize_Overlap 相邻块间应有重叠上下文，前一块尾部内容应出现在下一块开头。
func TestSplitBySize_Overlap(t *testing.T) {
	// 6 段各 50 字，maxSize=110 → 切多块，overlap 在尾部
	var paras []string
	for i := 0; i < 6; i++ {
		paras = append(paras, strings.Repeat("ABCDE", 10)) // 50 字
	}
	text := strings.Join(paras, "\n\n")
	chunks := SplitBySize(text, 110)
	require.Greater(t, len(chunks), 1, "应被切分多块")
	// 相邻块应有重叠
	for i := 0; i+1 < len(chunks); i++ {
		tail := chunks[i][len(chunks[i])-30:]
		assert.Contains(t, chunks[i+1], tail, "块 %d 尾部应重叠到块 %d", i, i+1)
	}
}

// TestChunkDocument_Overlap 入库分块后，相邻块应包含重叠上下文。
func TestChunkDocument_Overlap(t *testing.T) {
	// 构造一个无标题但有多段的长文档（会走 SplitBySize 路径）
	var paras []string
	for i := 0; i < 10; i++ {
		paras = append(paras, strings.Repeat("字", 100))
	}
	text := strings.Join(paras, "\n\n")
	chunks := ChunkDocument(text, 300)
	require.Greater(t, len(chunks), 1, "长文档应被切分")
	// 相邻块应有重叠
	for i := 0; i+1 < len(chunks); i++ {
		tail := chunks[i].Content[len(chunks[i].Content)-20:]
		assert.Contains(t, chunks[i+1].Content, tail, "块 %d 尾部应重叠到块 %d", i, i+1)
	}
}

// TestSplitBySize_SentenceBoundary 重叠区域应尽量在句子边界断开，而非字符中间。
func TestSplitBySize_SentenceBoundary(t *testing.T) {
	// 每段都是完整句子（句号结尾），验证重叠在句子边界处
	s1 := strings.Repeat("第一句内容。", 8)   // ~96 字
	s2 := strings.Repeat("第二段文字。", 8)   // ~96 字
	s3 := strings.Repeat("第三段落完。", 8)   // ~96 字
	text := s1 + "\n\n" + s2 + "\n\n" + s3
	chunks := SplitBySize(text, 150)
	require.Greater(t, len(chunks), 1, "应被切分")
	// 第 2 块开头不应是半截句子（不应以"第"字开头接上一个不完整句）
	if len(chunks) > 1 {
		assert.NotRegexp(t, `^[^。！？\n]{20,}`, chunks[1],
			"第 2 块开头不应有超过 20 字的无标点前缀（应在句子边界重叠）")
	}
}
