package service

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Chunk 文本切片结果
type Chunk struct {
	Content string
	Title   string
}

// SplitMarkdownByHeading 按 Markdown 标题切片文本
func SplitMarkdownByHeading(text string) []Chunk {
	re := regexp.MustCompile(`(?m)^#{1,6}\s+(.+)$`)

	var chunks []Chunk
	matches := re.FindAllStringIndex(text, -1)

	if len(matches) == 0 {
		return []Chunk{{Content: text, Title: ""}}
	}

	for i, match := range matches {
		var content string
		title := text[match[0]:match[1]]

		if i+1 < len(matches) {
			content = text[match[0]:matches[i+1][0]]
		} else {
			content = text[match[0]:]
		}

		chunks = append(chunks, Chunk{
			Content: content,
			Title:   strings.TrimLeft(title, "# "),
		})
	}

	return chunks
}

// SplitBySize 按最大字符数切分文本，相邻块间保留 overlap 字符的上下文重叠，
// 避免完整句子被切断后两边都丢失完整语义。
//
// 实现要点（对比旧版）：
//   - 全程按 rune（unicode 码点）切分长度与 slice，杜绝从多字节汉字中间截断产生非法 UTF-8；
//   - 单块有效字符数恒不大于 maxSize：段落自身超长（含 overlap 后超限）时内部按 maxSize 再切，
//     避免 overlap 前补把近 maxSize 的块顶超预算法超上限。
func SplitBySize(text string, maxSize int) []string {
	if utf8.RuneCountInString(text) <= maxSize {
		return []string{text}
	}

	const overlap = 150

	paragraphs := strings.Split(text, "\n\n")
	var chunks []string
	current := []rune{}

	// emit 输出当前块；若其有效字符数超 maxSize（段落自身超长），内部按 maxSize 切分，
	// 保证每条输出都不超预算。
	emit := func(c []rune) {
		if len(c) == 0 {
			return
		}
		if len(c) <= maxSize {
			chunks = append(chunks, string(c))
			return
		}
		for len(c) > 0 {
			n := min(maxSize, len(c))
			chunks = append(chunks, string(c[:n]))
			c = c[n:]
		}
	}

	// overlapTail 返回 c 的尾部最多 overlap 个 rune，并尽量在句子边界后断开（与旧版
	// LastIndexAny 语义一致，但基于 rune）。
	overlapTail := func(c []rune) []rune {
		if len(c) <= overlap {
			return c
		}
		tail := c[len(c)-overlap:]
		for i := len(tail) - 1; i > 0; i-- {
			switch tail[i-1] {
			case '。', '！', '？', '\n':
				return tail[i:]
			}
		}
		return tail
	}

	for _, p := range paragraphs {
		parunes := []rune(p)
		if len(current) == 0 {
			current = parunes
			continue
		}
		if len(current)+len(parunes)+2 <= maxSize {
			current = append(current, '\n', '\n')
			current = append(current, parunes...)
			continue
		}
		// 放不下下一段：flush 当前块
		old := current
		emit(old)
		// 新块以前段尾部 overlap 开头，再接下一段；若再加上下一段仍超限，则丢弃 overlap
		// 让下一段自身内部切分（保证不超 maxSize）。
		current = append(overlapTail(old), '\n', '\n')
		current = append(current, parunes...)
		if len(current) > maxSize {
			current = parunes
		}
	}
	emit(current)
	return chunks
}

// ChunkDocument 将文档内容切分为合适大小的块
func ChunkDocument(content string, maxChunkSize int) []Chunk {
	// 先尝试按标题切片
	chunks := SplitMarkdownByHeading(content)

	// 如果单块太大，进一步按段落切分
	var finalChunks []Chunk
	for _, chunk := range chunks {
		if len(chunk.Content) > maxChunkSize {
			subChunks := SplitBySize(chunk.Content, maxChunkSize)
			for i, sub := range subChunks {
				finalChunks = append(finalChunks, Chunk{
					Content: sub,
					Title:   chunk.Title + fmt.Sprintf(" (part %d)", i+1),
				})
			}
		} else {
			finalChunks = append(finalChunks, chunk)
		}
	}

	return finalChunks
}
