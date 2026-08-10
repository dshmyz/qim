package service

import (
	"fmt"
	"regexp"
	"strings"
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
func SplitBySize(text string, maxSize int) []string {
	if len(text) <= maxSize {
		return []string{text}
	}

	const overlap = 150

	paragraphs := strings.Split(text, "\n\n")
	var chunks []string
	current := ""

	for _, p := range paragraphs {
		if len(current)+len(p) > maxSize && current != "" {
			chunks = append(chunks, current)
			// 取前一块尾部 overlap 字符作为下一块的上下文开头
			tail := current
			if len(tail) > overlap {
				tail = tail[len(tail)-overlap:]
				if idx := strings.LastIndexAny(tail, "。！？\n"); idx > 0 {
					tail = tail[idx+1:]
				}
			}
			current = tail + "\n\n" + p
		} else {
			if current != "" {
				current += "\n\n"
			}
			current += p
		}
	}

	if current != "" {
		chunks = append(chunks, current)
	}

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
