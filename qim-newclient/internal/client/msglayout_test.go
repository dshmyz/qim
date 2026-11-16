package client

// 消息行逐项对齐专项：时间分隔规则、发送者/头像、气泡下方 meta、
// 撤回占位，以及三种会话的渲染截图。

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

const minuteMS = int64(60 * 1000)

// TestNeedsTimeDivider：对齐 useChatUtils.shouldShowTimeDivider 的
// 「第一条 / >5 分钟 / 跨天」三条件。
func TestNeedsTimeDivider(t *testing.T) {
	base := time.Date(2026, 11, 15, 9, 0, 0, 0, time.Local).UnixMilli()
	cases := []struct {
		name string
		prev int64
		cur  int64
		want bool
	}{
		{"第一条", 0, base, true},
		{"同分钟", base, base, false},
		{"4 分钟", base, base + 4*minuteMS, false},
		{"恰好 5 分钟", base, base + 5*minuteMS, false},
		{"超过 5 分钟", base, base + 5*minuteMS + 1, true},
		{"跨天但间隔很短", base - 2*60*minuteMS, base, true},
		{"时间戳缺失按第一条处理", base, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := needsTimeDivider(tc.prev, tc.cur); got != tc.want {
				t.Errorf("needsTimeDivider(%d,%d)=%v, 期望 %v", tc.prev, tc.cur, got, tc.want)
			}
		})
	}
}

// TestTimeDividerLabel：今天 / 昨天 / 本周 / 更早四档文案。
func TestTimeDividerLabel(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, time.Local)
	if got := timeDividerLabel(today.UnixMilli()); got != "09:30" {
		t.Errorf("今天 = %q, 期望 09:30", got)
	}
	yesterday := today.AddDate(0, 0, -1)
	if got := timeDividerLabel(yesterday.UnixMilli()); got != "昨天 09:30" {
		t.Errorf("昨天 = %q", got)
	}
	older := today.AddDate(0, 0, -30)
	if got := timeDividerLabel(older.UnixMilli()); !strings.Contains(got, "09:30") ||
		!strings.Contains(got, "2026-") {
		t.Errorf("更早 = %q, 应含完整日期与时间", got)
	}
}

// TestMsgsFromAPITimeDividers：一页里按 5 分钟规则插胶囊，而不是只按天分。
func TestMsgsFromAPITimeDividers(t *testing.T) {
	base := time.Date(2026, 11, 15, 9, 0, 0, 0, time.Local).UnixMilli()
	list := []qapi.Message{
		{ID: qapi.ID("1"), Timestamp: base, Content: "一"},
		{ID: qapi.ID("2"), Timestamp: base + minuteMS, Content: "二"},
		{ID: qapi.ID("3"), Timestamp: base + 10*minuteMS, Content: "三"},
	}
	got := msgsFromAPI(list, "7")
	pills := 0
	for _, m := range got {
		if m.Pill != "" {
			pills++
		}
	}
	if pills != 2 {
		t.Fatalf("胶囊数 = %d, 期望 2（开头 + 10 分钟间隔）: %+v", pills, got)
	}
	if len(got) != 5 || got[2].Text != "二" || got[4].Text != "三" {
		t.Errorf("消息顺序不对: %+v", got)
	}
}

// TestPrependOlderKeepsMiddleDividers：只去掉 existing 开头的胶囊，
// 中间按 5 分钟插入的胶囊必须保留。
func TestPrependOlderKeepsMiddleDividers(t *testing.T) {
	existing := []message{
		{Key: "time-100", Pill: "09:00"},
		{Key: "m-1", Text: "一"},
		{Key: "time-200", Pill: "09:20"},
		{Key: "m-2", Text: "二"},
	}
	older := []message{{Key: "time-50", Pill: "08:50"}, {Key: "m-0", Text: "零"}}
	got := prependOlder(existing, older)
	pills := 0
	for _, m := range got {
		if m.Pill != "" {
			pills++
		}
	}
	if pills != 2 {
		t.Errorf("合并后胶囊数 = %d, 期望 2: %+v", pills, got)
	}
	if got[0].Key != "time-50" {
		t.Errorf("开头应是更早那页的胶囊: %+v", got[0])
	}
}

// TestMessageFromAPIAvatar：发送者头像 URL 要带进界面数据，供真图渲染。
func TestMessageFromAPIAvatar(t *testing.T) {
	m := qapi.Message{
		ID: qapi.ID("9"), Timestamp: time.Now().UnixMilli(),
		Content: "带头像", Sender: &qapi.Sender{ID: qapi.ID("8"), Name: "张三", Avatar: "/uploads/a.png"},
	}
	got := messageFromAPI(m, "7", 0)
	if got.Avatar != "/uploads/a.png" || got.Sender != "张三" {
		t.Errorf("发送者映射不对: %+v", got)
	}
	if got.SenderID != "8" {
		t.Errorf("发送者 ID 没带进来（头像右键快速 @ 要用）: SenderID=%q", got.SenderID)
	}
}

// TestMessageFromAPISenderIDFallback：老接口只给顶层 sender_id、
// 内嵌 sender 没有 id 时，也要把 id 接上，否则快速 @ 生成一个 @0。
func TestMessageFromAPISenderIDFallback(t *testing.T) {
	got := messageFromAPI(qapi.Message{
		ID: "9", SenderID: qapi.ID("41"),
		Sender: &qapi.Sender{Name: "王五"},
	}, "7", 0)
	if got.SenderID != "41" {
		t.Errorf("顶层 sender_id 回退失败: SenderID=%q", got.SenderID)
	}
}

// TestAtMeBubbleAutoHighlight：@我消息收到即高亮（浅主色底），
// 不用等点 banner 或搜索跳转。对齐 .message-item.at-mention .message-bubble。
func TestAtMeBubbleAutoHighlight(t *testing.T) {
	msgs := []message{
		{Key: "m1", Sender: "张三", Text: "普通消息", At: "07:20"},
		{Key: "m2", Sender: "李四", Text: "@你 来看一下", At: "07:21", AtMe: true},
	}
	a := &app{
		active: 0, loggedIn: true, userName: "我",
		convs: []conversation{{ID: "101", Name: "项目群", Kind: "group"}},
		msgs:  msgs,
	}
	tt := ui.NewTester(a.view, 1050, 703)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("@我") {
		t.Errorf("@我徽标缺失: texts=%q", tt.Texts())
	}
	write(t, tt, "/tmp/qim-atme-highlight.png")
}

// TestMessageMetaRender：状态在气泡下方常显，时间文本仍存在
// （用 Opacity 隐藏，hover 才可见，对齐 .message-time）。
func TestMessageMetaRender(t *testing.T) {
	cases := []struct {
		name string
		msg  message
		want []string
	}{
		{"已读", message{Key: "m1", Mine: true, Text: "收到", At: "07:24", Read: true}, []string{"07:24", "已读"}},
		{"发送中", message{Key: "m2", Mine: true, Text: "在发", At: "07:25", Pending: true}, []string{"07:25", "发送中"}},
		{"失败可重试", message{Key: "m3", Mine: true, Text: "没发出", At: "07:26", Failed: true}, []string{"07:26", "发送失败", "重试"}},
		{"对方也有时间", message{Key: "m4", Sender: "张三", Text: "你好", At: "07:20"}, []string{"07:20"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &app{
				active: 0, loggedIn: true, userName: "我",
				convs: []conversation{{ID: "101", Name: "项目群", Kind: "group"}},
				msgs:  []message{tc.msg},
			}
			tt := ui.NewTester(a.view, 1050, 703)
			tt.Frame()
			tt.Frame()
			for _, w := range tc.want {
				if !tt.HasText(w) {
					t.Errorf("meta 缺 %q: texts=%q", w, tt.Texts())
				}
			}
		})
	}
}

// TestSenderNameOnlyInGroups：单聊 / bot 不显示发送者名（原版只在
// group/discussion 且非自己时显示），群聊显示。
func TestSenderNameOnlyInGroups(t *testing.T) {
	for _, tc := range []struct {
		kind string
		want bool
	}{{"group", true}, {"discussion", true}, {"single", false}, {"bot", false}} {
		a := &app{
			active: 0, loggedIn: true, userName: "我",
			convs: []conversation{{ID: "101", Name: "会话", Kind: tc.kind}},
			msgs:  []message{{Key: "m1", Sender: "颜麟育", Text: "项目进度", At: "07:20"}},
		}
		tt := ui.NewTester(a.view, 1050, 703)
		tt.Frame()
		tt.Frame()
		if got := tt.HasText("颜麟育"); got != tc.want {
			t.Errorf("kind=%s 发送者名显示=%v, 期望 %v: texts=%q", tc.kind, got, tc.want, tt.Texts())
		}
	}
}

// TestRecalledPillRender：撤回占位是灰胶囊，且与「重新编辑」同一行。
func TestRecalledPillRender(t *testing.T) {
	a := &app{
		active: 0, loggedIn: true, userName: "我",
		convs: []conversation{{ID: "101", Name: "项目群", Kind: "group"}},
		msgs: []message{{Key: "m1", Sender: "张三", At: "07:28", Recalled: true,
			Text: "张三 撤回了一条消息", OrigText: "原来的内容"}},
	}
	tt := ui.NewTester(a.view, 1050, 703)
	tt.Frame()
	tt.Frame()
	for _, w := range []string{"张三 撤回了一条消息", "重新编辑"} {
		if !tt.HasText(w) {
			t.Errorf("撤回占位缺 %q: texts=%q", w, tt.Texts())
		}
	}
}

// TestShotMessageLayout：群聊 / 单聊 / 自己消息三态截图。
func TestShotMessageLayout(t *testing.T) {
	msgs := []message{
		{Key: "time-1", Pill: "09:00"},
		{Key: "m1", Sender: "颜麟育", At: "09:00", Text: "@测试用户  基于通用知识，本周五例会（8月7日14:00）建议议程如下：\n1. 项目进展回顾\n2. 超时与阻塞项评审"},
		{Key: "m2", Mine: true, Sender: "我", At: "09:02", Read: true, Text: "收到，我下午先整理一版材料"},
		{Key: "m3", Sender: "测试用户", At: "09:03", Text: "好的"},
		{Key: "time-2", Pill: "09:10"}, // 中间的分隔，保证截图里能看见两侧横线
		{Key: "m4", Mine: true, Sender: "我", At: "09:11", Failed: true, Text: "这条没发出去"},
		{Key: "m5", Sender: "张三", At: "09:05", Recalled: true, Text: "张三 撤回了一条消息", OrigText: "撤回原文"},
	}
	for _, tc := range []struct {
		file string
		kind string
	}{
		{"/tmp/qim-msglayout-group.png", "group"},
		{"/tmp/qim-msglayout-single.png", "single"},
	} {
		a := &app{
			active: 0, loggedIn: true, userName: "我", selfID: "7",
			convs: []conversation{{ID: "101", Name: "技术交流群", Kind: tc.kind}},
			msgs:  msgs,
		}
		tt := ui.NewTester(a.view, 1050, 900)
		tt.Frame()
		tt.Frame()
		if !tt.HasText("09:00") {
			t.Errorf("%s 缺时间胶囊: texts=%q", tc.file, tt.Texts())
		}
		write(t, tt, tc.file)
	}

	// 引用消息：验证「发送者名 + 摘要」两行结构。
	q := &app{
		active: 0, loggedIn: true, userName: "我",
		convs: []conversation{{ID: "101", Name: "项目群", Kind: "group"}},
		msgs: []message{{Key: "m1", Sender: "用户一", At: "16:56", Text: "收到，回复这条",
			QuoteSender: "用户一", QuoteText: "这是被引用的原文"}},
	}
	tt := ui.NewTester(q.view, 1050, 703)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("用户一") || !tt.HasText("这是被引用的原文") {
		t.Errorf("引用块结构不对: texts=%q", tt.Texts())
	}
	write(t, tt, "/tmp/qim-msglayout-quote.png")
}
