package client

// /quick 快捷回复与 emoji 表情面板（A 组聊天体验）测试。

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

// TestQuickTriggerText 触发态判定：/quick 本身、/quick+空格+过滤词命中；
// /quickly 之类不命中。
func TestQuickTriggerText(t *testing.T) {
	cases := []struct {
		in    string
		query string
		ok    bool
	}{
		{"/quick", "", true},
		{"/quick ", "", true},
		{"/quick 好", "好", true},
		{"/quick 收", "收", true},
		{"  /quick", "", true},
		{"/quickly", "", false},
		{"/quick abc", "abc", true},
		{"hello /quick", "", false},
		{"", "", false},
		{"/", "", true}, // 引导条「试一下」插入裸 /
		{"/ 好", "好", true},
		{"/note", "", false}, // 未实现的命令不误触发
	}
	for _, tc := range cases {
		q, ok := quickTriggerText(tc.in)
		if ok != tc.ok || (ok && q != tc.query) {
			t.Errorf("quickTriggerText(%q) = (%q,%v), 期望 (%q,%v)", tc.in, q, ok, tc.query, tc.ok)
		}
	}
}

// TestParseQuickReplies 未设置/非数组/空数组 → 默认短语；自定义数组原样保留。
func TestParseQuickReplies(t *testing.T) {
	// nil 响应（请求失败）
	if got := parseQuickReplies(nil); len(got) == 0 || got[0] != "收到" {
		t.Fatalf("nil 应兜底默认: %v", got)
	}
	// has_value=false
	v := &qapi.UserSettingValue{Value: []byte("null")}
	if got := parseQuickReplies(v); len(got) != len(defaultQuickReplies) {
		t.Fatalf("未设置应兜底默认: %v", got)
	}
	// 空数组 → 兜底
	v = &qapi.UserSettingValue{Value: []byte("[]"), HasValue: true}
	if got := parseQuickReplies(v); len(got) != len(defaultQuickReplies) {
		t.Fatalf("空数组应兜底默认: %v", got)
	}
	// 自定义数组
	v = &qapi.UserSettingValue{Value: []byte(`["嗯","马上来"]`), HasValue: true}
	got := parseQuickReplies(v)
	if len(got) != 2 || got[0] != "嗯" {
		t.Fatalf("自定义应保留: %v", got)
	}
	// 空串被剔除
	v = &qapi.UserSettingValue{Value: []byte(`["", "嗯"]`), HasValue: true}
	if got := parseQuickReplies(v); len(got) != 1 || got[0] != "嗯" {
		t.Fatalf("空串应剔除: %v", got)
	}
}

// TestQuickTriggerFetch 输入 /quick → 面板打开并从服务端拉自定义短语；
// 输入离开触发态 → 面板关闭。
func TestQuickTriggerFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"data":{"value":["项目词-甲","项目词-乙"],"has_value":true}}`)
	}))
	defer srv.Close()

	a := &app{
		loggedIn: true,
		client:   qapi.New(srv.URL),
		ctx:      context.Background(),
	}
	a.input = "/quick"
	a.maybeTriggerSlash()
	if !a.quickOpen || !a.quickBusy {
		t.Fatalf("触发后应 open+busy: open=%v busy=%v", a.quickOpen, a.quickBusy)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.Lock()
		busy := a.quickBusy
		a.mu.Unlock()
		if !busy {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	a.mu.Lock()
	items := a.quickItems
	a.mu.Unlock()
	if len(items) != 2 || items[0] != "项目词-甲" {
		t.Fatalf("自定义短语没拉到: %v", items)
	}

	// 输入离开触发态 → 关闭
	a.mu.Lock()
	a.input = "普通消息"
	a.mu.Unlock()
	a.maybeTriggerSlash()
	if a.quickOpen || a.quickBusy || a.quickItems != nil {
		t.Fatalf("离开触发态应全清: %+v", a)
	}
}

// TestQuickFilterApply 过滤命中与采纳：输入被整体替换为短语+空格，面板关闭。
func TestQuickFilterApply(t *testing.T) {
	a := &app{loggedIn: true}
	a.input = "/quick 谢"
	a.maybeTriggerSlash() // 无 client：直接用默认短语
	a.mu.Lock()
	open := a.quickOpen
	a.mu.Unlock()
	if !open {
		t.Fatal("无 client 也应开面板（默认短语兜底）")
	}
	hits := a.filteredQuickReplies()
	if len(hits) != 2 || hits[0] != "谢谢" || hits[1] != "感谢" {
		t.Fatalf("过滤结果不符: %v", hits)
	}
	a.applyQuickReply(hits[0])
	if a.input != "谢谢 " {
		t.Fatalf("采纳后输入应为「谢谢 」: %q", a.input)
	}
	if a.quickOpen || a.quickItems != nil {
		t.Fatalf("采纳后应关面板: %+v", a)
	}
}

// TestQuickPanelRender headless 渲染：触发 /quick 后面板出现短语与提示。
func TestQuickPanelRender(t *testing.T) {
	a := &app{
		loggedIn: true,
		convs:    []conversation{{ID: "100", Name: "张三", Kind: "single"}},
		active:   0,
	}
	a.input = "/quick"
	tt := ui.NewTester(a.view, 1050, 720)
	tt.Frame()
	tt.Frame()
	// 面板高度封顶 260px，List 虚拟化只渲染首屏可见的短语
	for _, want := range []string{"快速回复", "收到", "好的", "点击或回车插入"} {
		if !tt.HasText(want) {
			t.Errorf("快捷回复面板缺 %q, texts=%q", want, tt.Texts())
		}
	}
}

// TestClassicMarker 经典表情标记格式。
func TestClassicMarker(t *testing.T) {
	if classicMarker("呲牙") != "[呲牙]" || classicMarker("K歌") != "[K歌]" {
		t.Fatal("classicMarker 格式错")
	}
	if len(classicEmojis) != 163 {
		t.Fatalf("经典表情数量应 163（对齐 qim-client）: %d", len(classicEmojis))
	}
	if len(unicodeEmojis) == 0 {
		t.Fatal("unicode 表情不应为空")
	}
}

// TestEmojiInsertToggle 点表情追加输入且面板不自动关；toggle 关闭后 tab 复位。
func TestEmojiInsertToggle(t *testing.T) {
	a := &app{}
	a.emojiOpen = true
	a.emojiTab = "classic"
	a.insertEmoji(classicMarker("呲牙"))
	if a.input != "[呲牙]" || !a.emojiOpen {
		t.Fatalf("插入后输入=%q open=%v", a.input, a.emojiOpen)
	}
	a.insertEmoji("😊")
	if a.input != "[呲牙]😊" {
		t.Fatalf("连续插入应追加: %q", a.input)
	}
	a.toggleEmoji() // 关
	if a.emojiOpen || a.emojiTab != "" {
		t.Fatalf("关闭后应复位: open=%v tab=%q", a.emojiOpen, a.emojiTab)
	}
	a.toggleEmoji() // 开
	if !a.emojiOpen || a.emojiTab != "" {
		t.Fatalf("再开应回默认 tab: open=%v tab=%q", a.emojiOpen, a.emojiTab)
	}
}

// TestEmojiPanelRender headless 渲染：默认 tab 有 unicode 网格与底部两个 tab。
func TestEmojiPanelRender(t *testing.T) {
	a := &app{loggedIn: true, convs: []conversation{{ID: "100", Name: "张三", Kind: "single"}}, active: 0}
	a.emojiOpen = true
	tt := ui.NewTester(a.view, 1050, 720)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{"默认表情", "经典表情", "😊", "👍"} {
		if !tt.HasText(want) {
			t.Errorf("emoji 面板缺 %q, texts=%q", want, tt.Texts())
		}
	}
	// 切经典 tab：显示 [名称] 网格（列表首屏）
	a.emojiTab = "classic"
	tt.Frame()
	for _, want := range []string{"[呲牙]", "[微笑]", "[惊讶]"} {
		if !tt.HasText(want) {
			t.Errorf("经典 tab 缺 %q, texts=%q", want, tt.Texts())
		}
	}
}

// TestClearOpsResetsQuick 切会话时 quick/emoji 状态一并清掉。
func TestClearOpsResetsQuick(t *testing.T) {
	a := &app{}
	a.quickOpen = true
	a.quickBusy = true
	a.quickItems = []string{"嗯"}
	a.emojiOpen = true
	a.emojiTab = "classic"
	a.clearMessageOps()
	if a.quickOpen || a.quickBusy || a.quickItems != nil || a.emojiOpen || a.emojiTab != "" {
		t.Fatalf("clearMessageOps 应复位 A 组状态: %+v", a)
	}
	_ = strings.TrimSpace("")
}
