package client

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestExtractCodeBlocks(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "带语言标识的单块",
			src:  "看这段：\n```go\nfunc main() {}\n```\n结束",
			want: []string{"func main() {}"},
		},
		{
			name: "无语言标识",
			src:  "```\nhello\nworld\n```",
			want: []string{"hello\nworld"},
		},
		{
			name: "多块用空行连接由调用方处理",
			src:  "```js\nlet a = 1\n```\n中间\n```python\nprint(a)\n```",
			want: []string{"let a = 1", "print(a)"},
		},
		{
			name: "无围栏返回空",
			src:  "普通文本 `inline` 不算代码块",
			want: nil,
		},
		{
			name: "围栏必须有换行",
			src:  "```go func(){}```",
			want: nil,
		},
		{
			name: "CRLF 也支持",
			src:  "```go\r\npackage main\r\n```",
			want: []string{"package main"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractCodeBlocks(tc.src)
			if len(got) != len(tc.want) {
				t.Fatalf("提取块数 = %d, 想要 %d: %q", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("块[%d] = %q, 想要 %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestCodeBlocksForCopy(t *testing.T) {
	m := message{RawContent: "前文\n```go\na := 1\n```\n后文", Text: "前文 [代码] 后文"}
	if got := codeBlocksForCopy(m); got != "a := 1" {
		t.Errorf("复制内容 = %q, 想要 %q", got, "a := 1")
	}
	// 回退到 Text（本地构造的消息没有 RawContent）
	m = message{Text: "```js\nconst x = 2\n```"}
	if got := codeBlocksForCopy(m); got != "const x = 2" {
		t.Errorf("回退复制内容 = %q, 想要 %q", got, "const x = 2")
	}
	if hasCodeBlock(message{RawContent: "没有代码"}) {
		t.Error("普通文本不应触发复制代码菜单")
	}
	if !hasCodeBlock(m) {
		t.Error("含围栏的消息应触发复制代码菜单")
	}
}

// 把一行高亮结果拼回去，必须与原行完全一致（不能吞字符/加字符）。
func TestHighlightCodeLineRoundTrip(t *testing.T) {
	lines := []string{
		`func main() {`,
		`	// 这是注释`,
		`	s := "hello \"world\"" // trailing`,
		`	n := 0x1F + 42`,
		`	if err != nil { return }`,
		`	fmt.Println(s, n)`,
		`	{"id": 42, "ok": true}`,
		`# Python comment`,
		`def greet(name):`,
		`	SELECT * FROM msg WHERE id = 7;`,
		``,
	}
	for _, line := range lines {
		spans := highlightCodeLine(line, ui.Hex("#404040"))
		var rebuilt strings.Builder
		for _, s := range spans {
			rebuilt.WriteString(s.Text)
		}
		if rebuilt.String() != line {
			t.Errorf("往返不一致: 原=%q 重建=%q", line, rebuilt.String())
		}
	}
}

func TestHighlightCodeLineColors(t *testing.T) {
	find := func(line, frag string) ui.Color {
		for _, s := range highlightCodeLine(line, themeGray800()) {
			if strings.Contains(s.Text, frag) {
				return s.Color
			}
		}
		t.Fatalf("在 %q 里找不到片段 %q", line, frag)
		return ui.Color{}
	}
	if got := find(`func main() {`, "func"); got != hlKeyword {
		t.Errorf("func 应为关键字色, got %v", got)
	}
	if got := find(`func main() {`, "main"); got != hlFunction {
		t.Errorf("main 应为函数名色, got %v", got)
	}
	if got := find(`	s := "hello"`, `"hello"`); got != hlString {
		t.Errorf("字符串应为字符串色, got %v", got)
	}
	if got := find(`	// trailing`, "// trailing"); got != hlComment {
		t.Errorf("注释应为注释色, got %v", got)
	}
	if got := find(`	n := 42`, "42"); got != hlAttr {
		t.Errorf("数字应为数字色, got %v", got)
	}
	if got := find(`	{"id": 42}`, `"id"`); got != hlTag {
		t.Errorf("JSON 属性名应为标签色, got %v", got)
	}
	if got := find(`	n := len(s)`, "len"); got != hlBuiltin {
		t.Errorf("内置函数 len 应为内置色, got %v", got)
	}
	if got := find(`	fmt.Println(s)`, "Println"); got != hlFunction {
		t.Errorf("方法名 Println 应为函数名色, got %v", got)
	}
}

func themeGray800() ui.Color { return ui.Hex("#404040") }
