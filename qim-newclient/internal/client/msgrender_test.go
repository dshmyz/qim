package client

// 消息渲染对齐（qim-client TextMessage / SystemMessage / MergedForwardMessage /
// ShareMessage）：system 胶囊、@ chip 与链接富文本、合并转发卡片与记录弹窗、
// share 卡片与原文弹窗、列表预览标签。

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

// TestNextURL 裸 URL 识别的边界：尾部标点不算、前缀字母不算。
func TestNextURL(t *testing.T) {
	i, j := nextURL("看这个 https://example.com/a?b=1。怎么样")
	if i < 0 {
		t.Fatal("未识别 URL")
	}
	if got := "看这个 https://example.com/a?b=1。怎么样"[i:j]; got != "https://example.com/a?b=1" {
		t.Fatalf("URL 边界错: %q", got)
	}
	if i, _ := nextURL("xhttps://a.com"); i >= 0 {
		t.Fatal("字母前缀不应算 URL")
	}
	if i, _ := nextURL("没有链接"); i >= 0 {
		t.Fatal("误判 URL")
	}
}

// TestNextMarkdownLink markdown 链接抽取。
func TestNextMarkdownLink(t *testing.T) {
	label, url, before, after, ok := nextMarkdownLink("前[文档](https://docs.qq.com)后")
	if !ok || label != "文档" || url != "https://docs.qq.com" || before != "前" || after != "后" {
		t.Fatalf("解析错: %q %q %q %q %v", label, url, before, after, ok)
	}
	// 非链接的方括号不动
	_, _, _, _, ok = nextMarkdownLink("数组[1](元)")
	if ok {
		t.Fatal("误判 markdown 链接")
	}
}

// TestPreviewSpecialTypes 会话列表预览：merged_forward / share / system 不露 JSON。
func TestPreviewSpecialTypes(t *testing.T) {
	mf := qapi.Message{Type: "merged_forward", Content: `{"version":1,"title":"项目群的聊天记录","messages":[{"id":"1","type":"text","content":"hi","senderName":"张三","timestamp":1}]}`}
	if got := mf.Preview(); got != "[聊天记录] 项目群的聊天记录（1条）" {
		t.Fatalf("merged_forward 预览错: %q", got)
	}
	sh := qapi.Message{Type: "share", Content: `{"type":"sticky","name":"购物清单"}`}
	if got := sh.Preview(); got != "[分享] 购物清单" {
		t.Fatalf("share 预览错: %q", got)
	}
	sys := qapi.Message{Type: "system", Content: "张三 加入了群聊"}
	if got := sys.Preview(); got != "张三 加入了群聊" {
		t.Fatalf("system 预览错: %q", got)
	}
}

// TestSystemMessagePill system 消息渲染成居中胶囊：无发送者名、无普通气泡。
func TestSystemMessagePill(t *testing.T) {
	m := messageFromAPI(qapi.Message{
		ID: "9", Type: "system", Content: "张三 加入了群聊", Timestamp: 1000,
	}, "7", 0)
	if !m.Sys || m.Text != "张三 加入了群聊" {
		t.Fatalf("system 识别错: Sys=%v text=%q", m.Sys, m.Text)
	}

	a := &app{active: 0, loggedIn: true, client: nil}
	a.convs = []conversation{{ID: "1", Name: "群", Kind: "group"}}
	a.msgs = []message{m, {Key: "m-1", Sender: "张三", Text: "大家好", At: "10:00"}}

	tt := ui.NewTester(a.view, 900, 600)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("张三 加入了群聊") {
		t.Fatalf("胶囊文本未渲染: %q", tt.Texts())
	}
}

// TestTextRichRender @ 提及 chip + 链接在 headless 下渲染出可读文本。
func TestTextRichRender(t *testing.T) {
	content := `看看 @{mention:5|李四} 和 [文档](https://docs.qq.com/x) 还有 https://example.com/go`
	m := messageFromAPI(qapi.Message{
		ID: "1", Type: "text", Content: content, SenderID: "8", Sender: &qapi.Sender{ID: "8", Name: "李四"}, Timestamp: 1000,
	}, "7", 0)
	if len(m.Segs) != 3 {
		t.Fatalf("分段数错: %d (%+v)", len(m.Segs), m.Segs)
	}
	if m.Segs[1].Mention == nil || m.Segs[1].Text != "@李四" {
		t.Fatalf("提及段错: %+v", m.Segs[1])
	}

	a := &app{active: 0, loggedIn: true}
	a.convs = []conversation{{ID: "1", Name: "张三", Kind: "single"}}
	a.msgs = []message{m}
	tt := ui.NewTester(a.view, 900, 600)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{"@李四", "看看", "文档", "还有", "https://example.com/go"} {
		if !tt.HasText(want) {
			t.Errorf("富文本缺 %q, texts=%q", want, tt.Texts())
		}
	}
	if tt.HasText("@{mention") {
		t.Fatal("token 泄漏到界面")
	}
}

// TestMergedForwardCardRender 合并转发卡片：标题+条数+前3条预览，
// 点「查看聊天记录」开记录弹窗。
func TestMergedForwardCardRender(t *testing.T) {
	raw := `{"version":1,"title":"项目群的聊天记录","messages":[` +
		`{"id":"1","type":"text","content":"在吗","senderName":"张三","timestamp":1700000000000},` +
		`{"id":"2","type":"image","content":"{\"url\":\"/s/a.png\"}","senderName":"我","timestamp":1700000001000},` +
		`{"id":"3","type":"text","content":"收到","senderName":"李四","timestamp":1700000002000},` +
		`{"id":"4","type":"text","content":"第四条","senderName":"王五","timestamp":1700000003000}]}`
	m := messageFromAPI(qapi.Message{ID: "20", Type: "merged_forward", Content: raw,
		SenderID: "8", Timestamp: 1000}, "7", 0)
	if m.MF == nil {
		t.Fatal("merged_forward 未解析")
	}

	a := &app{active: 0, loggedIn: true}
	a.convs = []conversation{{ID: "1", Name: "群", Kind: "group"}}
	a.msgs = []message{m}
	tt := ui.NewTester(a.view, 900, 600)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{"项目群的聊天记录（4条）", "查看聊天记录", "张三", "在吗", "共 4 条"} {
		if !tt.HasText(want) {
			t.Errorf("卡片缺 %q, texts=%q", want, tt.Texts())
		}
	}
	if tt.HasText(`{"url"`) {
		t.Fatal("图片 JSON 泄漏")
	}

	// 打开记录弹窗：4 条全列
	a.openMergedRecord(m.MF)
	tt.Frame()
	for _, want := range []string{"💬 项目群的聊天记录", "第四条", "王五"} {
		if !tt.HasText(want) {
			t.Errorf("记录弹窗缺 %q", want)
		}
	}
	a.closeMergedRecord()
	tt.Frame()
	if tt.HasText("王五") {
		t.Fatal("弹窗关闭后仍显示")
	}
}

// TestShareCardRender share 卡片 + 原文弹窗。
func TestShareCardRender(t *testing.T) {
	raw := `{"type":"sticky","id":"42","name":"购物清单","content":"分享了便签: 购物清单","originalContent":"买牛奶、鸡蛋"}`
	m := messageFromAPI(qapi.Message{ID: "21", Type: "share", Content: raw,
		SenderID: "8", Timestamp: 1000}, "7", 0)
	if m.SH == nil {
		t.Fatal("share 未解析")
	}

	a := &app{active: 0, loggedIn: true}
	a.convs = []conversation{{ID: "1", Name: "张三", Kind: "single"}}
	a.msgs = []message{m}
	tt := ui.NewTester(a.view, 900, 600)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{"购物清单", "便签 · 点击查看原文", "分享了便签: 购物清单"} {
		if !tt.HasText(want) {
			t.Errorf("卡片缺 %q, texts=%q", want, tt.Texts())
		}
	}
	if tt.HasText("originalContent") {
		t.Fatal("JSON 字段泄漏")
	}

	a.openShareOriginal(m.SH)
	tt.Frame()
	if !tt.HasText("买牛奶、鸡蛋") || !tt.HasText("便签分享") {
		t.Fatalf("原文弹窗缺内容: %q", tt.Texts())
	}
	a.closeShareOriginal()
}

// TestMergedForwardShapeInvalid 形状不对的 merged_forward 退化为普通气泡。
func TestMergedForwardShapeInvalid(t *testing.T) {
	m := messageFromAPI(qapi.Message{ID: "22", Type: "merged_forward",
		Content: `{"version":2,"title":"x","messages":[]}`, Timestamp: 1000}, "7", 0)
	if m.MF != nil {
		t.Fatal("坏 payload 不应解析")
	}
	if m.Text != "[聊天记录] x（0条）" && m.Text == "" {
		t.Fatalf("兜底文本缺失: %q", m.Text)
	}
}
