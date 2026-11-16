package client

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui/richtext"
)

func TestMarkdownRunAttrs(t *testing.T) {
	src := "**结论**：接口对齐 *v2 协议*，~~旧字段~~ 全部下线。`code` 与 [链接](https://a.com)\n\n# 标题\n\n- [x] 完成\n- [ ] 待办\n\n> 引用\n\n```go\nfmt.Println(1)\n```"
	for i, p := range richtext.ParseMarkdown(src).Paragraphs() {
		var parts []string
		for _, r := range p.Runs {
			a := r.Attributes
			tag := ""
			if a.Weight >= 600 {
				tag += "B"
			}
			if a.Italic {
				tag += "I"
			}
			if a.Strikethrough {
				tag += "S"
			}
			if a.Code {
				tag += "C"
			}
			if a.Link != "" {
				tag += "L(" + a.Link + ")"
			}
			parts = append(parts, tag+"«"+r.Text+"»")
		}
		st := p.Style
		t.Logf("para %d style={H:%d List:%d Checked:%v Ind:%d Quote:%v Code:%v} runs=%v",
			i, st.Heading, st.List, st.Checked, st.Indent, st.Quote, st.CodeBlock, strings.Join(parts, " "))
	}
}
