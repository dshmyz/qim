package qapi

import "testing"

// StripMarkdown：对齐 qim-client 的 stripMarkdown——预览、复制、引用等
// 纯文本场合不露语法符号。

func TestStripMarkdownFence(t *testing.T) {
	src := "前置说明\n```xml\n<note><title>测试</title></note>\n```\n后置说明"
	got := StripMarkdown(src)
	want := "前置说明\n[代码]\n后置说明"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestStripMarkdownInline(t *testing.T) {
	got := StripMarkdown("**加粗** 与 *斜体*、~~删除~~、`code` 混排")
	want := "加粗 与 斜体、删除、code 混排"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestStripMarkdownLinkImage(t *testing.T) {
	got := StripMarkdown("看 [需求说明](https://docs.qq.com/doc) 和 ![截图](https://a.com/b.png)")
	want := "看 需求说明 和 [图片]"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestStripMarkdownBlocks(t *testing.T) {
	src := "# 标题一\n## 标题二\n> 引用一句\n- 列表项\n1. 有序项\n正文"
	got := StripMarkdown(src)
	want := "标题一\n标题二\n引用一句\n列表项\n有序项\n正文"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestStripMarkdownOneLine(t *testing.T) {
	got := StripMarkdownOneLine("第一行\n第二行\n```go\ncode()\n```")
	want := "第一行 第二行 [代码]"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestStripMarkdownEmpty(t *testing.T) {
	if StripMarkdown("") != "" {
		t.Fatal("empty in, empty out")
	}
}
