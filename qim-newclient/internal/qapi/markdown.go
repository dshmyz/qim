package qapi

import (
	"regexp"
	"strings"
)

// StripMarkdown 把 markdown 源剥成可读纯文本：围栏代码块折成「[代码]」，
// 标题/引用/列表前缀和行内语法符号（**粗体**、*斜体*、~~删除线~~、
// `行内代码`）去掉，图片折成「[图片]」，链接只留 label。
// 对齐 qim-client 的 stripMarkdown，用于会话列表预览、复制全文、引用
// 摘要等纯文本场合；气泡内的完整渲染走 mygo richtext，不经这里。
func StripMarkdown(src string) string {
	if src == "" {
		return ""
	}
	// 围栏代码块先整体折成一行标记，避免里面的 #、* 等被当普通语法剥掉。
	s := reFence.ReplaceAllString(src, "[代码]")
	// 图片在链接之前处理（![alt](url) 含链接的形状）。
	s = reMDImage.ReplaceAllString(s, "[图片]")
	s = reMDLink.ReplaceAllString(s, "$1")
	s = reMDCode.ReplaceAllString(s, "$1")
	s = reMDBold.ReplaceAllString(s, "$1")
	s = reMDItalic.ReplaceAllString(s, "$1")
	s = reMDStrike.ReplaceAllString(s, "$1")

	out := make([]string, 0, 8)
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimRight(line, " \t")
		line = reMDHeading.ReplaceAllString(line, "")
		line = reMDQuote.ReplaceAllString(line, "")
		line = reMDBullet.ReplaceAllString(line, "$1")
		line = reMDNumber.ReplaceAllString(line, "$1")
		if line == "" && (len(out) == 0 || out[len(out)-1] == "") {
			continue // 折叠连续空行
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// StripMarkdownOneLine 是给会话列表预览用的单行版本：换行折成空格。
func StripMarkdownOneLine(src string) string {
	return strings.Join(strings.Fields(StripMarkdown(src)), " ")
}

var (
	reFence     = regexp.MustCompile("(?s)```[^\n]*\n?.*?```|```[^\n]*$")
	reMDImage   = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]*)\)`)
	reMDLink    = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	reMDCode    = regexp.MustCompile("`([^`\n]+)`")
	reMDBold    = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)
	reMDItalic  = regexp.MustCompile(`\*([^*\n]+)\*`)
	reMDStrike  = regexp.MustCompile(`~~([^~\n]+)~~`)
	reMDHeading = regexp.MustCompile(`^#{1,6}\s+`)
	reMDQuote   = regexp.MustCompile(`^>\s?`)
	reMDBullet  = regexp.MustCompile(`^(\s*)[-*+]\s+`)
	reMDNumber  = regexp.MustCompile(`^(\s*)\d+[.)]\s+`)
)
