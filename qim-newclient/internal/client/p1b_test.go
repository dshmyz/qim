package client

// P1 第二批：引用回复、撤回/删除/提醒、未读分隔线、多选、@ 提及，
// 以及这些新增元素在 headless 渲染里确实画得出来。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

func TestP1BUnreadDivider(t *testing.T) {
	msgs := []message{
		{Key: "day-1", Pill: "2026-11-14"},
		{Key: "m-1", Text: "旧消息"},
		{Key: "m-2", Text: "未读 1"},
		{Key: "m-3", Text: "未读 2"},
	}
	got := insertUnreadDivider(msgs, 2)
	if len(got) != 5 || !got[2].Divider || got[2].Text != "以下为新消息" {
		t.Fatalf("分隔线位置不对: %+v", got)
	}
	if got[3].Key != "m-2" || got[4].Key != "m-3" {
		t.Fatalf("分隔线应该在第一条未读之前: %+v", got)
	}
	// 未读数比本页消息还多：分隔线放到时间胶囊之后、最早一条消息之前
	all := insertUnreadDivider(msgs, 99)
	if len(all) != 5 || !all[1].Divider || all[2].Key != "m-1" {
		t.Fatalf("未读数溢出时位置不对: %+v", all)
	}
	if out := insertUnreadDivider(msgs, 0); len(out) != len(msgs) {
		t.Error("未读为 0 不该插分隔线")
	}
	// 翻页时分隔线不能被当成普通消息去重掉，也不能复制造出第二条
	older := []message{{Key: "day-0", Pill: "2026-11-13"}, {Key: "m-0", Text: "更早"}}
	merged := prependOlder(got, older)
	dividers := 0
	for _, m := range merged {
		if m.Divider {
			dividers++
		}
	}
	// 两页合并后只保留最前面那条时间胶囊，existing 开头的胶囊会被去掉，
	// 所以是 6 条（不是 7）；未读分隔线仍然恰好 1 条。
	if dividers != 1 || len(merged) != 6 {
		t.Fatalf("翻页后分隔线丢了或重复: %d 条, %+v", len(merged), merged)
	}
}

func TestP1BMessageFromAPIQuoteAndMention(t *testing.T) {
	raw := `{
		"id":901,"conversation_id":106,"sender_id":8,"type":"text",
		"content":"hi ` + qapi.EncodeMention("7", "小明") + `",
		"created_at":"2026-11-14T18:00:00+08:00",
		"quoted_message":{"id":896,"content":"原文","sender":{"id":8,"nickname":"张三"}}
	}`
	var api qapi.Message
	if err := json.Unmarshal([]byte(raw), &api); err != nil {
		t.Fatal(err)
	}
	m := messageFromAPI(api, "7", 0)
	if m.ServerID != "901" || m.QuoteID != "896" || m.QuoteSender != "张三" || m.QuoteText != "原文" {
		t.Fatalf("引用字段不对: %+v", m)
	}
	if strings.Contains(m.Text, "@{mention") || !strings.Contains(m.Text, "@小明") {
		t.Fatalf("@ token 没解码: %q", m.Text)
	}
	if !m.AtMe {
		t.Error("被 @ 的消息应该打标记")
	}
	// 只有 id、没有展开对象的引用也要留一条占位
	var onlyID qapi.Message
	_ = json.Unmarshal([]byte(`{"id":902,"quoted_message_id":896}`), &onlyID)
	got := messageFromAPI(onlyID, "7", 0)
	if got.QuoteID != "896" || got.QuoteText != "原消息已不可见" {
		t.Fatalf("只有 id 的引用没兜底: %+v", got)
	}
	// 自己发的消息不该显示 "@我"
	self := messageFromAPI(api, "8", 0)
	if self.AtMe {
		t.Error("自己发的消息不该算 @我")
	}
}

func TestP1BQuoteSend(t *testing.T) {
	// 只把解析出来的标量存进共享快照：直接共享 map 会让 handler 的
	// 回包序列化与测试断言并发读同一个 map，-race 会报数据竞争。
	type snapshot struct {
		content any
		quoted  any
		seen    bool
	}
	var mu sync.Mutex
	var snap snapshot
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/messages"):
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			content := body["content"]
			mu.Lock()
			snap = snapshot{content: content, quoted: body["quoted_message_id"], seen: true}
			mu.Unlock()
			p1Reply(w, map[string]any{
				"id": 902, "conversation_id": 101, "sender_id": 7,
				"type": "text", "content": content,
				"created_at": "2026-11-14T18:00:00+08:00",
			})
		default:
			p1Reply(w, map[string]any{"messages": []any{}})
		}
	}))
	defer srv.Close()

	a := newTestApp(srv.URL)
	a.client.SetSession("tkn", "rtk")
	a.loggedIn, a.selfID = true, "7"
	a.convs = []conversation{{ID: "101", Name: "用户一", Kind: "single"}}
	a.active = 0
	a.msgs = []message{{Key: "m-896", ServerID: "896", Sender: "用户一", Text: "原文", Stamp: time.Now().UnixMilli()}}
	a.setQuote(a.msgs[0])
	if a.quote == nil || a.quote.ID != "896" {
		t.Fatal("引用没挂到输入框")
	}
	a.sendText("  收到  ")
	waitFor(t, "引用发送请求", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return snap.seen
	})
	mu.Lock()
	s := snap
	mu.Unlock()
	if s.content != "收到" {
		t.Errorf("正文没 trim: %v", s.content)
	}
	if id, _ := s.quoted.(float64); id != 896 {
		t.Errorf("quoted_message_id 没带上: %v", s.quoted)
	}
	if a.readBool(func() bool { return a.quote != nil }) {
		t.Error("发送后引用条没清掉")
	}
	waitFor(t, "回执换成服务端 id", func() bool {
		return a.readBool(func() bool { return !a.msgs[len(a.msgs)-1].Pending })
	})
	a.read(func() {
		last := a.msgs[len(a.msgs)-1]
		if last.QuoteID != "896" || last.QuoteText != "原文" || last.Key != "m-902" {
			t.Errorf("乐观气泡的引用/回执不对: %+v", last)
		}
	})
}

func TestP1BMessageActions(t *testing.T) {
	var mu sync.Mutex
	calls := make(map[string]int)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls[r.Method+" "+r.URL.Path]++
		mu.Unlock()
		p1Reply(w, map[string]any{})
	}))
	defer srv.Close()

	a := newTestApp(srv.URL)
	a.client.SetSession("tkn", "rtk")
	a.loggedIn, a.selfID = true, "7"
	a.convs = []conversation{{ID: "101", Name: "用户一", Kind: "single"}}
	a.active = 0
	now := time.Now().UnixMilli()
	a.msgs = []message{
		{Key: "m-901", ServerID: "901", Mine: true, Text: "我发的", Stamp: now},
		{Key: "m-902", ServerID: "902", Text: "对方发的", Stamp: now},
		{Key: "local-1", Mine: true, Text: "没发出去", Stamp: now},
	}

	if !a.canRecall(a.msgs[0]) {
		t.Error("2 分钟内自己发的消息应该可撤回")
	}
	if a.canRecall(a.msgs[1]) || a.canRecall(a.msgs[2]) {
		t.Error("别人的消息 / 本地乐观消息不该可撤回")
	}
	old := a.msgs[0]
	old.Stamp = time.Now().Add(-3 * time.Minute).UnixMilli()
	if a.canRecall(old) {
		t.Error("超过时限不该可撤回")
	}
	// 提醒还要求消息发出满 1 小时（后端 messageRemindTime 默认 3600s）
	oldEnough := a.msgs[0]
	oldEnough.Stamp = time.Now().Add(-2 * time.Hour).UnixMilli()
	if !a.canRemind(oldEnough) {
		t.Error("满 1 小时、自己发的未读单聊消息应该可提醒")
	}
	if a.canRemind(a.msgs[0]) || a.canRemind(a.msgs[1]) {
		t.Error("刚发的消息 / 别人的消息不该可提醒")
	}
	// 群聊里不该出现提醒
	a.convs[0].Kind = "group"
	if a.canRemind(oldEnough) {
		t.Error("群聊不支持消息提醒")
	}
	a.convs[0].Kind = "single"

	a.recallMessage(a.msgs[0])
	a.remindMessage(a.msgs[0])
	a.deleteMessages([]string{"m-901"})
	waitFor(t, "三个操作都发出", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(calls) == 3
	})
	mu.Lock()
	want := map[string]int{
		"POST /api/v1/messages/901/recall": 1,
		"POST /api/v1/messages/901/remind": 1,
		"DELETE /api/v1/messages/901":      1,
	}
	for k, v := range want {
		if calls[k] != v {
			t.Errorf("调用 %s = %d 次", k, calls[k])
		}
	}
	mu.Unlock()
	waitFor(t, "删除后气泡消失", func() bool {
		return a.readBool(func() bool { return len(a.msgs) == 2 })
	})
}

func TestP1BSelectionAndMentionInput(t *testing.T) {
	// 末尾 @token 的识别：前面必须是空白，邮箱里的 @ 不算
	if _, q, ok := activeMention("大家好 @张"); !ok || q != "张" {
		t.Errorf("末尾 @ 没识别: q=%q ok=%v", q, ok)
	}
	if _, _, ok := activeMention("mail@example.com"); ok {
		t.Error("邮箱里的 @ 不该触发候选")
	}
	if _, _, ok := activeMention("@张 已选好"); ok {
		t.Error("token 后面有空格就不该再算正在输入")
	}

	a := &app{active: 0, convs: []conversation{{ID: "101"}}, input: "大家好 @张"}
	a.applyMentionHit(qapi.SearchUserHit{ID: qapi.ID("41"), Name: "张三", Username: "zs"})
	if a.input != "大家好 @张三 " {
		t.Fatalf("选中后输入框不对: %q", a.input)
	}
	if token := a.mentionPicks["@张三"]; !strings.Contains(token, "@{mention:41") {
		t.Fatalf("没记下发送用的 token: %q", token)
	}
	got := a.encodeMentions("大家好 @张三 看下")
	if !strings.Contains(got, "@{mention:41") || strings.Contains(got, "@张三") {
		t.Fatalf("发送前没换成 token: %q", got)
	}
	// 草稿切走再切回来，@ 选择不能丢（否则草稿里的 @名字 只是普通文本）
	a.saveDraft("101", a.input)
	a.restoreMentionPicks("101")
	if a.mentionPicks["@张三"] == "" {
		t.Error("草稿里的 @ 选择没恢复")
	}

	a.msgs = []message{
		{Key: "m-1", Mine: true, Text: "第一条", Sender: "我"},
		{Key: "m-2", Text: "第二条", Sender: "张三"},
	}
	a.startSelectMode("m-1")
	if !a.selectMode || len(a.selectedKeys()) != 1 {
		t.Fatal("多选没进入")
	}
	a.toggleSelected("m-2")
	if got := a.selectedText(); !strings.Contains(got, "第一条") || !strings.Contains(got, "第二条") {
		t.Errorf("复制内容不对: %q", got)
	}
	removeMessageKeys(&a.msgs, map[string]bool{"m-1": true})
	if len(a.msgs) != 1 || a.msgs[0].Key != "m-2" {
		t.Fatalf("批量删除后列表不对: %+v", a.msgs)
	}
	a.exitSelectMode()
	if a.selectMode || a.selected != nil {
		t.Error("退出多选没清状态")
	}
}

func TestP1BRenderShots(t *testing.T) {
	now := time.Now().UnixMilli()
	msgs := []message{
		{Key: "day-2026-11-14", Pill: "2026-11-14"},
		{Key: "m-896", ServerID: "896", Sender: "用户一", Text: "这是被引用的原文", At: "16:50", Stamp: now},
		{Key: "unread-divider", Divider: true, Text: "以下为新消息"},
		{Key: "m-897", ServerID: "897", Sender: "用户一", Text: "你好 @我", At: "16:55", Stamp: now, AtMe: true},
		{Key: "m-898", ServerID: "898", Mine: true, Sender: "我", Text: "收到，回复这条",
			At: "16:56", Stamp: now, QuoteID: "896", QuoteSender: "用户一", QuoteText: "这是被引用的原文"},
	}
	a := &app{
		active: 0, loggedIn: true, userName: "原生客户端测试",
		connStatus: qapi.StatusConnected, selfID: "7",
		convs: []conversation{{ID: "106", Name: "用户一", Kind: "single",
			Preview: "收到，回复这条", Time: "16:56", Color: avatarColor("106用户一")}},
		msgs:       msgs,
		selectMode: true,
		selected:   map[string]bool{"m-897": true},
		quote:      &messageQuote{ID: "898", Name: "我", Text: "收到，回复这条"},
	}
	tt := ui.NewTester(a.view, 1050, 703)
	tt.Frame()
	// 引用块现在按原版拆成「发送者名」+「摘要」两行，
	// 不再是旧的一行「用户一：这是被引用的原文」。
	for _, want := range []string{"以下为新消息", "@我", "用户一", "这是被引用的原文", "回复 我", "已选 1 条消息"} {
		if !tt.HasText(want) {
			t.Errorf("界面缺 %q, texts=%q", want, tt.Texts())
		}
	}
	write(t, tt, "/tmp/qim-p1b-chat.png")

	// @ 候选面板：末尾输入 @ 时弹出，选中后写下可读文本
	b := &app{
		active: 0, loggedIn: true, userName: "原生客户端测试",
		connStatus: qapi.StatusConnected,
		convs:      []conversation{{ID: "106", Name: "用户一", Kind: "single", Color: avatarColor("106")}},
		msgs:       []message{{Key: "m-1", ServerID: "1", Text: "你好", At: "16:50", Stamp: now}},
		input:      "帮我看下 @张",
		mentionHits: []qapi.SearchUserHit{
			{ID: qapi.ID("41"), Name: "张三", Username: "zs"},
			{ID: qapi.ID("42"), Name: "张四", Username: "zsi"},
		},
	}
	tt2 := ui.NewTester(b.view, 1050, 703)
	tt2.Frame()
	for _, want := range []string{"@所有人", "张三", "张四"} {
		if !tt2.HasText(want) {
			t.Errorf("候选面板缺 %q, texts=%q", want, tt2.Texts())
		}
	}
	write(t, tt2, "/tmp/qim-p1b-mention.png")
}
