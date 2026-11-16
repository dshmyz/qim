package client

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

// TestRecalledLabel 验证撤回占位文案对齐 qim-client：
// 自己的 →「你撤回了一条消息」；别人的 →「{发送者} 撤回了一条消息」，空名兜底「对方」。
func TestRecalledLabel(t *testing.T) {
	if got := recalledLabel(true, "张三"); got != "你撤回了一条消息" {
		t.Errorf("自己的撤回文案错: %q", got)
	}
	if got := recalledLabel(false, "张三"); got != "张三 撤回了一条消息" {
		t.Errorf("别人的撤回文案错: %q", got)
	}
	if got := recalledLabel(false, ""); got != "对方 撤回了一条消息" {
		t.Errorf("空名兜底错: %q", got)
	}
}

// TestMessageFromAPIRecallOriginal 验证归一化时保留撤回原文（extra.original_content），
// 非撤回消息 origText 为空。
func TestMessageFromAPIRecallOriginal(t *testing.T) {
	var m qapi.Message
	raw := `{"id":1,"type":"text","sender_id":7,"content":"[消息已撤回]","is_recalled":true,
		"extra":"{\"original_content\":\"撤回前的原话\",\"knowledge_sources\":[]}"}`
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	got := messageFromAPI(m, "7", 0)
	if !got.Recalled || got.OrigText != "撤回前的原话" {
		t.Errorf("撤回原文没保留: recalled=%v orig=%q", got.Recalled, got.OrigText)
	}
	if got.Text != "你撤回了一条消息" {
		t.Errorf("自己的撤回文案错: %q", got.Text)
	}

	var plain qapi.Message
	if err := json.Unmarshal([]byte(`{"id":2,"type":"text","content":"正常消息"}`), &plain); err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	nm := messageFromAPI(plain, "7", 1)
	if nm.OrigText != "" {
		t.Errorf("非撤回消息不该有原文: %q", nm.OrigText)
	}
}

// TestEditRecalledFillsInput 验证重新编辑把原文整体回填输入框。
func TestEditRecalledFillsInput(t *testing.T) {
	a := &app{input: "正在打字"}
	m := message{Recalled: true, OrigText: "撤回前的原话"}

	a.editRecalled(m)
	if a.input != "撤回前的原话" {
		t.Errorf("输入框没被回填: %q", a.input)
	}

	// 没有原文的撤回占位点不动
	a.input = "别动"
	a.editRecalled(message{Recalled: true})
	if a.input != "别动" {
		t.Errorf("无原文时不应改输入框: %q", a.input)
	}
}

// TestRecallEditRender headless 渲染：自己撤回且原文在的气泡下有「重新编辑」，
// 别人的撤回 / 无原文的没有按钮。
func TestRecallEditRender(t *testing.T) {
	a := &app{
		active:     0,
		loggedIn:   true,
		userName:   "我",
		selfID:     "7",
		connStatus: qapi.StatusConnected,
		convs:      []conversation{{ID: "106", Name: "单聊", Kind: "single"}},
		msgs: []message{
			{Key: "m-1", Mine: true, Recalled: true, OrigText: "撤回前的原话",
				Text: "你撤回了一条消息"},
			{Key: "m-2", Mine: true, Recalled: true, Text: "你撤回了一条消息"},
			{Key: "m-3", Mine: false, Sender: "张三", Recalled: true,
				Text: "张三 撤回了一条消息"},
		},
	}
	tt := ui.NewTester(a.view, 1050, 850)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("你撤回了一条消息") || !tt.HasText("张三 撤回了一条消息") {
		t.Errorf("撤回占位文案没渲染: %q", tt.Texts())
	}
	if !tt.HasText("✎ 重新编辑") {
		t.Errorf("重新编辑按钮没渲染: %q", tt.Texts())
	}
	// 按钮只有一个（m-1），m-2 无原文、m-3 非自己的都不给入口
	n := 0
	for _, s := range tt.Texts() {
		if strings.Contains(s, "重新编辑") {
			n++
		}
	}
	if n != 1 {
		t.Errorf("重新编辑按钮数量错: %d", n)
	}
	write(t, tt, "/tmp/qim-recall-edit.png")
}
