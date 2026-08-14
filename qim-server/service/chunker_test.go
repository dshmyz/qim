package service

import (
	"strings"
	"testing"
	"unicode/utf8"

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
	// 每段约 80 rune（句号结尾），3 段共 240+ > maxSize=150，验证重叠在句子边界处
	s1 := strings.Repeat("第一句详细内容测试一下。", 5)   // 55 rune
	s2 := strings.Repeat("第二段文字充分描述。", 5)        // 45 rune
	s3 := strings.Repeat("第三段落完整结束语。", 5)        // 45 rune
	text := s1 + "\n\n" + s2 + "\n\n" + s3
	chunks := SplitBySize(text, 150)
	require.Greater(t, len(chunks), 1, "应被切分")
	// 第 2 块开头不应是半截句子（不应有超过 20 字的无标点前缀）
	if len(chunks) > 1 {
		assert.NotRegexp(t, `^[^。！？\n]{20,}`, chunks[1],
			"第 2 块开头不应有超过 20 字的无标点前缀（应在句子边界重叠）")
	}
}

// TestSplitBySize_CJKNoInvalidUTF8 验证纯中文长文本切块后每块都是合法 UTF-8、且不超 maxSize。
// 回归：overlap 用字节切片（tail[len(tail)-overlap:]）会从多字节汉字中间截断，产生非法 UTF-8
// 残片；且 overlap 前补可把近 maxSize 的块顶超预算而不重新切分。
func TestSplitBySize_CJKNoInvalidUTF8(t *testing.T) {
	var paras []string
	for i := 0; i < 20; i++ {
		paras = append(paras, strings.Repeat("中文字符内容测试", 10)) // 30 字/段
	}
	text := strings.Join(paras, "\n\n")
	const maxSize = 80
	chunks := SplitBySize(text, maxSize)
	require.Greater(t, len(chunks), 1, "长中文文本应被切分多块")

	for i, chunk := range chunks {
		assert.True(t, utf8.ValidString(chunk), "块 %d 含非法 UTF-8 字节（overlap 从汉字中间切断）", i)
		// 单块内的有效字符数不应超过 maxSize（overlap 前补不得让它超预算）
		assert.LessOrEqual(t, utf8.RuneCountInString(chunk), maxSize,
			"块 %d 有效字符数 %d 超过 maxSize %d（overlap 顶超预算未重新切分）", i, utf8.RuneCountInString(chunk), maxSize)
	}
}
