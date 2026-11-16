package client

// P1 测试：历史消息游标分页、草稿、失败重试、已读回执、
// 三路搜索编排，以及新增 UI 元素在 headless 渲染里确实存在。

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

func p1Reply(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "message": "ok", "data": data})
}

func p1Msg(id int) map[string]any {
	return map[string]any{
		"id": id, "conversation_id": 101, "sender_id": 7, "type": "text",
		"content": fmt.Sprintf("消息 %d", id), "created_at": "2026-11-14T09:30:00+08:00",
	}
}

func TestP1HistoryPaging(t *testing.T) {
	var olderCalls atomic.Int64
	var gotBefore string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/read"):
			p1Reply(w, map[string]any{})
		case strings.HasSuffix(r.URL.Path, "/messages") && r.Method == http.MethodGet:
			before := r.URL.Query().Get("before_id")
			var msgs []any
			if before == "" {
				for i := 51; i <= 100; i++ { // 满一页 → 还有更早的
					msgs = append(msgs, p1Msg(i))
				}
			} else {
				olderCalls.Add(1)
				gotBefore = before
				for i := 1; i <= 30; i++ { // 不足一页 → 翻到头
					msgs = append(msgs, p1Msg(i))
				}
			}
			p1Reply(w, map[string]any{
				"messages":   msgs,
				"pagination": map[string]any{"current_page": 1, "total_pages": 3, "total": 100, "page_size": 50},
			})
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	a := newTestApp(srv.URL)
	a.client.SetSession("tkn", "rtk")
	a.loggedIn, a.selfID = true, "7"
	a.convs = []conversation{{ID: "101", Name: "群"}}
	a.selectConversation(0)
	waitFor(t, "第一页历史", func() bool {
		return a.readBool(func() bool { return len(a.msgs) >= 50 && !a.loadingMsgs })
	})
	if !a.readBool(func() bool { return a.hasOlder && a.oldestID == "51" }) {
		t.Fatalf("第一页后应还有更早消息，游标应为 51: hasOlder=%v oldest=%q", a.hasOlder, a.oldestID)
	}

	a.mu.Lock()
	a.loadOlder()
	a.mu.Unlock()
	waitFor(t, "更早的一页", func() bool {
		return a.readBool(func() bool { return len(a.msgs) >= 80 && !a.loadingOlder })
	})
	if olderCalls.Load() != 1 || gotBefore != "51" {
		t.Errorf("before_id 应该用 51，实际 %q（调用 %d 次）", gotBefore, olderCalls.Load())
	}
	if a.readBool(func() bool { return a.hasOlder }) {
		t.Error("第二页不足 50 条，应该判定已翻到头")
	}
	// 51 条消息 + 1 个日期胶囊：胶囊不能因为两页同一天而重复。
	a.read(func() {
		if len(a.msgs) != 81 {
			t.Errorf("合并后消息数 = %d，期望 81（80 条 + 1 胶囊）", len(a.msgs))
		}
		pills := 0
		for _, m := range a.msgs {
			if m.Pill != "" {
				pills++
			}
		}
		if pills != 1 {
			t.Errorf("日期胶囊重复了 %d 个", pills)
		}
		if a.msgs[1].Key != "m-1" {
			t.Errorf("更早的消息没排在前面，首条 = %+v", a.msgs[1])
		}
		if a.oldestID != "1" {
			t.Errorf("游标没推进到 1: %q", a.oldestID)
		}
	})
}

func TestP1PrependOlderDedup(t *testing.T) {
	existing := []message{
		{Key: "day-2026-11-14", Pill: "2026-11-14"},
		{Key: "m-3", Text: "三"},
		{Key: "m-4", Text: "四"},
	}
	older := []message{
		{Key: "day-2026-11-14", Pill: "2026-11-14"},
		{Key: "m-1", Text: "一"},
		{Key: "m-3", Text: "边界重复"}, // 后端把边界那条又返回了一次
	}
	got := prependOlder(existing, older)
	wantKeys := []string{"day-2026-11-14", "m-1", "m-3", "m-4"}
	if len(got) != len(wantKeys) {
		t.Fatalf("合并结果 = %+v", got)
	}
	for i, k := range wantKeys {
		if got[i].Key != k {
			t.Errorf("第 %d 条 = %q，期望 %q", i, got[i].Key, k)
		}
	}
	if got[2].Text != "三" {
		t.Error("重复消息应该保留已有的那条，不能被旧页覆盖")
	}
}

func TestP1Drafts(t *testing.T) {
	a := &app{active: -1, convs: []conversation{{ID: "a"}, {ID: "b"}}}
	a.selectConversation(0) // 先进入一个会话，输入框才会存在
	a.input = "给 A 的草稿"
	a.selectConversation(1)
	if a.input != "" {
		t.Errorf("切到 B 应该是空输入框，实际 %q", a.input)
	}
	a.input = "给 B 的草稿"
	a.selectConversation(0)
	if a.input != "给 A 的草稿" {
		t.Errorf("切回 A 没恢复草稿: %q", a.input)
	}
	// 发出去之后草稿要清掉，否则切走再回来会重现
	a.loggedIn = false
	a.sendText(a.input)
	if a.draftFor("a") != "" {
		t.Errorf("发送后草稿还在: %q", a.draftFor("a"))
	}
}

func TestP1RetryFailed(t *testing.T) {
	a := &app{
		active: 0,
		convs:  []conversation{{ID: "101"}},
		msgs: []message{
			{Key: "local-1", Mine: true, Text: "发不出去的内容", Failed: true},
		},
		notice: "发送失败：xxx",
	}
	a.retryFailed("local-1")
	if len(a.msgs) != 1 {
		t.Fatalf("重试后应该只有一条: %+v", a.msgs)
	}
	if a.msgs[0].Text != "发不出去的内容" || a.msgs[0].Failed {
		t.Errorf("重试没重新上屏: %+v", a.msgs[0])
	}
	if a.notice != "" {
		t.Error("重试应该清掉旧的错误提示")
	}
}

func TestP1OpenRemoteConvKeepsDraft(t *testing.T) {
	a := &app{active: 0, convs: []conversation{{ID: "a"}}, input: "A 的草稿"}
	a.openRemoteConv(qapi.SearchConvHit{ID: qapi.ID("remote"), Name: "历史会话"}, "m-9")
	if a.draftFor("a") != "A 的草稿" {
		t.Errorf("往 convs 头部插会话后草稿串位了: %q", a.draftFor("a"))
	}
	if a.active < 0 || a.convs[a.active].ID != "remote" {
		t.Errorf("没选中新会话: active=%d convs=%+v", a.active, a.convs)
	}
	if a.highlight != "m-9" || a.search != "" {
		t.Errorf("跳转后高亮/搜索态不对: highlight=%q search=%q", a.highlight, a.search)
	}
}

func TestP1ReadReceipt(t *testing.T) {
	a := &app{
		active: 0,
		convs:  []conversation{{ID: "101"}},
		msgs: []message{
			{Key: "m-1", Mine: true},
			{Key: "m-2", Mine: true},
			{Key: "m-3", Mine: false},
		},
	}
	a.applyReadReceipt("101", []qapi.ID{"1"})
	if !a.msgs[0].Read || a.msgs[1].Read || a.msgs[2].Read {
		t.Errorf("只该把命中的我方消息标已读: %+v", a.msgs)
	}
	// 空 message_ids = 整个会话已读
	a.applyReadReceipt("101", nil)
	if !a.msgs[1].Read || a.msgs[2].Read {
		t.Errorf("空回执应把我方消息全标已读，但不能动对方消息: %+v", a.msgs)
	}
	// 不属于当前会话的回执不能误标
	a.msgs[1].Read = false
	a.applyReadReceipt("999", nil)
	if a.msgs[1].Read {
		t.Error("别的会话的回执不该改当前屏上的气泡")
	}
}

func TestP1ReadFlagFromHistory(t *testing.T) {
	var m qapi.Message
	if err := json.Unmarshal([]byte(`{"id":5,"sender_id":7,"type":"text","content":"好","is_read":true}`), &m); err != nil {
		t.Fatal(err)
	}
	if got := messageFromAPI(m, "7", 0); !got.Read {
		t.Error("自己发的消息 is_read=true 应该显示已读")
	}
	if got := messageFromAPI(m, "8", 0); got.Read {
		t.Error("别人发的消息 is_read 是「我读过」，不能当成已读回执")
	}
}

func TestP1SearchOrchestration(t *testing.T) {
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		switch r.URL.Path {
		case "/api/v1/conversations/search":
			p1Reply(w, []any{map[string]any{"id": 9, "type": "group", "name": "技术交流群"}})
		case "/api/v1/users/search":
			p1Reply(w, []any{map[string]any{"id": 8, "type": "user", "name": "张三", "username": "zs"}})
		case "/api/v1/messages/search":
			p1Reply(w, map[string]any{
				"list": []any{map[string]any{
					"id": 77, "conversation_id": 9, "sender_id": 8, "type": "text", "content": "构建挂了",
					"conversation_meta": map[string]any{"name": "技术交流群", "type": "group"},
				}},
				"total": 1,
			})
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	a := newTestApp(srv.URL)
	a.client.SetSession("tkn", "rtk")
	a.loggedIn = true
	a.mu.Lock()
	a.triggerSearch("构建")
	a.mu.Unlock()
	waitFor(t, "三路搜索结果", func() bool {
		return a.readBool(func() bool { return !a.searchBusy && len(a.searchHits.Messages) > 0 })
	})
	a.read(func() {
		if len(a.searchHits.Conversations) != 1 || len(a.searchHits.Users) != 1 {
			t.Errorf("三路结果不齐: %+v", a.searchHits)
		}
		if a.searchHits.MessageTotal != 1 || a.searchErr != "" {
			t.Errorf("消息总数/错误提示不对: total=%d err=%q", a.searchHits.MessageTotal, a.searchErr)
		}
	})

	// 输入被清空后，即使旧请求晚回来也不能把结果塞回界面。
	a.mu.Lock()
	a.triggerSearch("")
	a.mu.Unlock()
	time.Sleep(400 * time.Millisecond)
	if a.readBool(func() bool { return !a.searchHits.Empty() || a.searchBusy }) {
		t.Error("清空搜索后旧结果没被作废")
	}
}

// TestP1RenderShots 用 headless 渲染出 P1 的两个新界面。
// 真窗口的 Window.CapturePage 取的是 surface 当前像素：窗口被遮挡时
// 系统不派发新帧，会截到启动首帧（登录页）。headless 走同一套渲染引擎，
// 是验证布局的可靠手段，不受窗口可见性影响。
func TestP1RenderShots(t *testing.T) {
	msgs := []message{{Key: "day-2026-11-14", Pill: "2026-11-14"}}
	for i := 1; i <= 50; i++ {
		mine := i%3 == 0
		msgs = append(msgs, message{
			Key: fmt.Sprintf("m-%d", 895+i), Mine: mine, Sender: "用户一",
			Text: fmt.Sprintf("P1 分页测试消息 %02d", i), At: "16:57", Read: mine,
		})
	}
	a := &app{
		active: 0, loggedIn: true, userName: "原生客户端测试",
		connStatus: qapi.StatusConnected,
		hasOlder:   true, oldestID: "896",
		convs: []conversation{{
			ID: "106", Name: "用户一", Kind: "single",
			Preview: "P1 分页测试消息 52", Time: "16:57", Color: avatarColor("106用户一"),
		}},
		msgs: msgs,
	}
	tt := ui.NewTester(a.view, 1050, 703)
	tt.Frame()
	write(t, tt, "/tmp/qim-p1-chat.png")

	b := &app{
		active: -1, loggedIn: true, userName: "原生客户端测试",
		connStatus: qapi.StatusConnected,
		convs:      []conversation{{ID: "106", Name: "用户一", Preview: "P1 分页测试消息 52", Time: "16:57"}},
		search:     "P1", lastSearch: "P1",
		searchHits: qapi.SearchHits{
			Conversations: []qapi.SearchConvHit{{ID: qapi.ID("106"), Name: "用户一", Type: "single"}},
			Messages: []qapi.SearchMessageHit{
				{Message: qapi.Message{ID: qapi.ID("900"), ConversationID: qapi.ID("106"), Content: "P1 分页测试消息 05"}, ConvID: "106", ConvName: "用户一"},
				{Message: qapi.Message{ID: qapi.ID("901"), ConversationID: qapi.ID("106"), Content: "P1 分页测试消息 06"}, ConvID: "106", ConvName: "用户一"},
			},
			Users:        []qapi.SearchUserHit{{ID: qapi.ID("4"), Name: "用户一", Username: "user1"}},
			MessageTotal: 52,
		},
	}
	tt2 := ui.NewTester(b.view, 1050, 703)
	tt2.Frame()
	write(t, tt2, "/tmp/qim-p1-search.png")
}

func TestP1Views(t *testing.T) {
	a := &app{
		active: 0, loggedIn: true, hasOlder: true,
		convs: []conversation{{ID: "101", Name: "技术交流群"}},
		msgs:  []message{{Key: "m-1", Mine: true, Text: "你好", At: "09:30"}},
	}
	tt := ui.NewTester(a.view, 1200, 800)
	tt.Frame()
	for _, want := range []string{"加载更早消息", "技术交流群", "你好"} {
		if !tt.HasText(want) {
			t.Errorf("聊天界面缺 %q, texts=%q", want, tt.Texts())
		}
	}

	// 搜索态：三段分组标题都要在渲染树里
	b := &app{
		active: -1, loggedIn: true,
		convs:      []conversation{{ID: "101", Name: "技术交流群"}},
		search:     "技术",
		lastSearch: "技术",
		searchHits: qapi.SearchHits{
			Conversations: []qapi.SearchConvHit{{ID: qapi.ID("101"), Name: "技术交流群"}},
			Messages: []qapi.SearchMessageHit{{
				Message: qapi.Message{Content: "构建挂了"}, ConvID: "101", ConvName: "技术交流群",
			}},
			Users:        []qapi.SearchUserHit{{ID: qapi.ID("8"), Name: "张三", Username: "zs"}},
			MessageTotal: 1,
		},
	}
	tt2 := ui.NewTester(b.view, 1200, 800)
	tt2.Frame()
	for _, want := range []string{"会话 · 1", "聊天记录 · 1", "联系人 · 1", "构建挂了"} {
		if !tt2.HasText(want) {
			t.Errorf("搜索界面缺 %q, texts=%q", want, tt2.Texts())
		}
	}
}
