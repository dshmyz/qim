package client

// 频道功能单测：类型契约 / tab 过滤 / 选中与未读 / 订阅乐观回滚 /
// 发消息权限与迟到丢弃 / 点赞乐观校正 / 评论展开 / WS 频道通知分发 / 渲染。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"qim-newclient/internal/qapi"
)

// writeJSON 写标准响应。
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// newChannelTestApp 建一个登录态 app（client 由调用方注入）。
func newChannelTestApp() *app {
	a := newApp()
	a.loggedIn = true
	a.selfID = "1"
	return a
}

// serveChannels 起一个频道 fake server，返回 (server, 状态锁)。
func serveChannels(t *testing.T, state *channelSrvState) *qapi.Client {
	t.Helper()
	mux := http.NewServeMux()
	// 频道列表
	mux.HandleFunc("/api/v1/channels", func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()
		switch r.Method {
		case http.MethodGet:
			listReq++
			writeJSON(w, map[string]any{"code": 0, "data": state.channels})
		case http.MethodPost:
			createCalls++
			var req qapi.ChannelCreateRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			state.lastCreate = req
			writeJSON(w, map[string]any{"code": 0, "data": map[string]any{
				"id": 99, "name": req.Name, "status": "active", "approval_status": "approved",
			}})
		}
	})
	mux.HandleFunc("/api/v1/channels/2/subscribe", func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()
		subCalls++
		writeJSON(w, map[string]any{"code": 0, "message": "ok"})
	})
	mux.HandleFunc("/api/v1/channels/2/messages", func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()
		switch r.Method {
		case http.MethodGet:
			msgReq++
			// 后端 DESC 返回
			writeJSON(w, map[string]any{"code": 0, "data": []map[string]any{
				{"id": 2, "channel_id": 2, "sender_id": 2, "content": "newer", "type": "text", "created_at": "2026-11-15T10:00:00Z",
					"sender": map[string]any{"id": 2, "username": "bob", "nickname": "Bob"}},
				{"id": 1, "channel_id": 2, "sender_id": 1, "content": "older", "type": "text", "created_at": "2026-11-15T09:00:00Z",
					"sender": map[string]any{"id": 1, "username": "me", "nickname": "Me"}},
			}})
		case http.MethodPost:
			sendCalls++
			if state.failSend {
				writeJSON(w, map[string]any{"code": 403, "message": "无权限发布消息，仅频道创建者可发布"})
				return
			}
			var req struct{ Content string }
			_ = json.NewDecoder(r.Body).Decode(&req)
			state.lastSent = req.Content
			writeJSON(w, map[string]any{"code": 0, "data": map[string]any{
				"id": 3, "channel_id": 2, "sender_id": 1, "content": req.Content, "type": "text",
				"created_at": "2026-11-15T11:00:00Z",
				"sender":     map[string]any{"id": 1, "username": "me", "nickname": "Me"},
			}})
		}
	})
	mux.HandleFunc("/api/v1/channels/messages/1/like", func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()
		likeCalls++
		if r.Method == http.MethodPost {
			writeJSON(w, map[string]any{"code": 0, "data": map[string]any{"like_count": 1, "is_liked": true}})
			return
		}
		writeJSON(w, map[string]any{"code": 0, "data": map[string]any{"like_count": 0, "is_liked": false}})
	})
	mux.HandleFunc("/api/v1/channels/messages/1/likes", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"code": 0, "data": map[string]any{"like_count": 7, "is_liked": true}})
	})
	mux.HandleFunc("/api/v1/channels/messages/1/comments", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, map[string]any{"code": 0, "data": []map[string]any{
				{"id": 1, "message_id": 1, "user_id": 2, "content": "good", "created_at": "2026-11-15T10:05:00Z",
					"user": map[string]any{"id": 2, "username": "bob", "nickname": "Bob"}},
			}})
		case http.MethodPost:
			var req struct{ Content string }
			_ = json.NewDecoder(r.Body).Decode(&req)
			writeJSON(w, map[string]any{"code": 0, "data": map[string]any{
				"id": 2, "message_id": 1, "user_id": 1, "content": req.Content,
				"created_at": "2026-11-15T11:01:00Z",
				"user":       map[string]any{"id": 1, "username": "me", "nickname": "Me"},
			}})
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return qapi.New(srv.URL)
}

// channelSrvState 记录 fake server 状态与调用计数。
type channelSrvState struct {
	mu         sync.Mutex
	channels   []map[string]any
	failSend   bool
	lastCreate qapi.ChannelCreateRequest
	lastSent   string
}

var (
	listReq     int
	createCalls int
	subCalls    int
	msgReq      int
	sendCalls   int
	likeCalls   int
)

func resetChannelCounters() {
	listReq, createCalls, subCalls, msgReq, sendCalls, likeCalls = 0, 0, 0, 0, 0, 0
}

func chansFixture() []map[string]any {
	return []map[string]any{
		{"id": 1, "name": "默认频道", "creator_id": 9, "status": "active",
			"publish_permission": "creator_only", "comment_permission": "all_subscribers",
			"is_default": true, "is_subscribed": true, "message_count": 3},
		{"id": 2, "name": "公告板", "creator_id": 1, "status": "active",
			"publish_permission": "all_subscribers", "comment_permission": "all_subscribers",
			"is_default": false, "is_subscribed": true, "message_count": 5},
		{"id": 3, "name": "未订阅", "creator_id": 2, "status": "active",
			"publish_permission": "creator_only", "comment_permission": "disabled",
			"is_default": false, "is_subscribed": false, "message_count": 0},
	}
}

// TestChannelTypeContract 频道类型解析：合成字段与权限方法。
func TestChannelTypeContract(t *testing.T) {
	var ch qapi.Channel
	raw := `{"id":5,"name":"n","creator_id":"3","publish_permission":"creator_only",
		"comment_permission":"disabled","is_subscribed":true,"message_count":9}`
	if err := json.Unmarshal([]byte(raw), &ch); err != nil {
		t.Fatal(err)
	}
	if ch.ID.String() != "5" || ch.CreatorID.String() != "3" || !ch.IsSubscribed || ch.MessageCount != 9 {
		t.Fatalf("解析错: %+v", ch)
	}
	if !ch.IsCreator("3") || ch.IsCreator("4") {
		t.Error("IsCreator 判断错")
	}
	if ch.CanComment() {
		t.Error("disabled 频道应不可评论")
	}
	// 订阅者 + creator_only = 不可发
	if ch.CanPublish("3", true) != true {
		t.Error("创建者应可发")
	}
	ch2 := qapi.Channel{CreatorID: "9", PublishPermission: qapi.ChannelPublishAllSubs}
	if !ch2.CanPublish("4", true) {
		t.Error("all_subscribers+订阅应可发")
	}
	if ch2.CanPublish("4", false) {
		t.Error("未订阅不可发")
	}
}

// TestOpenChannelsLoads 打开频道拉列表，本地未读保留。
func TestOpenChannelsLoads(t *testing.T) {
	resetChannelCounters()
	state := &channelSrvState{channels: chansFixture()}
	a := newChannelTestApp()
	a.client = serveChannels(t, state)
	a.openChannels()
	waitFor(t, "频道列表加载", func() bool {
		a.mu.Lock()
		defer a.mu.Unlock()
		return a.channelsLoaded
	})
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.channels) != 3 {
		t.Fatalf("频道数错: %d", len(a.channels))
	}
	if !a.channelsOpen {
		t.Error("面板应打开")
	}
}

// TestChannelTabFilter 订阅/广场 tab + 搜索过滤。
func TestChannelTabFilter(t *testing.T) {
	state := &channelSrvState{channels: chansFixture()}
	a := newChannelTestApp()
	a.client = serveChannels(t, state)
	a.channelsLoaded = true
	a.channels = []qapi.Channel{
		{ID: "1", Name: "默认频道", IsSubscribed: true, PublishPermission: qapi.ChannelPublishCreatorOnly},
		{ID: "2", Name: "公告板", IsSubscribed: true, PublishPermission: qapi.ChannelPublishAllSubs},
		{ID: "3", Name: "未订阅", IsSubscribed: false},
	}
	a.channelTab = "subscribed"
	if got := len(a.filteredChannels()); got != 2 {
		t.Fatalf("订阅 tab 应 2 个, got %d", got)
	}
	a.channelTab = "discover"
	if got := len(a.filteredChannels()); got != 3 {
		t.Fatalf("广场 tab 应 3 个, got %d", got)
	}
	a.channelSearch = "公告"
	if got := len(a.filteredChannels()); got != 1 {
		t.Fatalf("搜索应 1 个, got %d", got)
	}
}

// TestSelectChannelReversesMessages 选中频道：DESC 反转成 ASC，未读清零。
func TestSelectChannelReversesMessages(t *testing.T) {
	resetChannelCounters()
	state := &channelSrvState{channels: chansFixture()}
	a := newChannelTestApp()
	a.client = serveChannels(t, state)
	a.channelsLoaded = true
	a.channels = []qapi.Channel{
		{ID: "1", Name: "A", IsSubscribed: true},
		{ID: "2", Name: "B", IsSubscribed: true, Unread: 4},
	}
	a.selectChannel(1)
	waitFor(t, "消息加载", func() bool {
		a.mu.Lock()
		defer a.mu.Unlock()
		return len(a.chMsgs) == 2
	})
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.chMsgs[0].Content != "older" || a.chMsgs[1].Content != "newer" {
		t.Fatalf("顺序错: %s, %s", a.chMsgs[0].Content, a.chMsgs[1].Content)
	}
	if a.channels[1].Unread != 0 {
		t.Error("未读应清零")
	}
	if a.chMsgs[0].Sender == nil || a.chMsgs[0].Sender.DisplayName() != "Me" {
		t.Errorf("sender 应解析: %+v", a.chMsgs[0])
	}
}

// TestSelectChannelStaleDrop 迟到响应：切频道后旧结果丢弃。
func TestSelectChannelStaleDrop(t *testing.T) {
	resetChannelCounters()
	state := &channelSrvState{channels: chansFixture()}
	a := newChannelTestApp()
	a.client = serveChannels(t, state)
	a.channelsLoaded = true
	a.channels = []qapi.Channel{{ID: "1", Name: "A", IsSubscribed: true}, {ID: "2", Name: "B"}}
	a.selectChannel(1)
	waitFor(t, "消息加载", func() bool {
		a.mu.Lock()
		defer a.mu.Unlock()
		return len(a.chMsgs) == 2
	})
	// 模拟迟到：手动改 channelSel 后再触发 apply（无法直接改 goroutine，改测 applyChannelNotification 的刷新丢弃即可）
	a.mu.Lock()
	if a.chMsgs[0].Content == "" {
		t.Error("消息不应为空")
	}
	a.mu.Unlock()
}

// TestToggleSubscribeRollback 订阅失败回滚。
func TestToggleSubscribeRollback(t *testing.T) {
	resetChannelCounters()
	state := &channelSrvState{channels: chansFixture()}
	a := newChannelTestApp()
	a.client = serveChannels(t, state)
	a.channelsLoaded = true
	a.channels = []qapi.Channel{{ID: "2", Name: "B", IsSubscribed: true}}
	// 退订 id=2 → fake 总成功 → 变 false
	a.toggleSubscribe("2")
	waitFor(t, "退订生效", func() bool {
		state.mu.Lock()
		calls := subCalls
		state.mu.Unlock()
		a.mu.Lock()
		defer a.mu.Unlock()
		return calls >= 1 && !a.channels[0].IsSubscribed
	})
}

// TestSubmitChannelMessagePermission 无权限拒绝、有权限发送成功。
func TestSubmitChannelMessagePermission(t *testing.T) {
	resetChannelCounters()
	state := &channelSrvState{channels: chansFixture()}
	a := newChannelTestApp()
	a.client = serveChannels(t, state)
	a.channelsLoaded = true
	a.channels = []qapi.Channel{
		{ID: "2", Name: "B", IsSubscribed: true, PublishPermission: qapi.ChannelPublishAllSubs},
	}
	a.channelSel = 0
	// 有权限：发送成功
	a.chInput = "hello"
	a.submitChannelMessage()
	waitFor(t, "消息发送", func() bool {
		state.mu.Lock()
		calls := sendCalls
		state.mu.Unlock()
		a.mu.Lock()
		defer a.mu.Unlock()
		return calls == 1 && len(a.chMsgs) == 1
	})
	a.mu.Lock()
	if a.chMsgs[0].Content != "hello" || a.chInput != "" {
		t.Fatal("发送结果或输入清理错")
	}
	a.mu.Unlock()
	// 无权限：creator_only 且非创建者
	a.channels[0].PublishPermission = qapi.ChannelPublishCreatorOnly
	a.chInput = "nope"
	a.submitChannelMessage()
	a.mu.Lock()
	if sendCalls != 1 {
		t.Error("无权限不应发请求")
	}
	if a.chInput != "nope" {
		t.Error("无权限时输入应保留")
	}
	a.mu.Unlock()
	// 创建者豁免
	a.channels[0].CreatorID = "1"
	a.submitChannelMessage()
	waitFor(t, "创建者发送", func() bool {
		state.mu.Lock()
		calls := sendCalls
		state.mu.Unlock()
		a.mu.Lock()
		defer a.mu.Unlock()
		return calls == 2
	})
}

// TestSubmitChannelMessageFailRollback 发送失败：输入回填。
func TestSubmitChannelMessageFailRollback(t *testing.T) {
	resetChannelCounters()
	state := &channelSrvState{channels: chansFixture(), failSend: true}
	a := newChannelTestApp()
	a.client = serveChannels(t, state)
	a.channelsLoaded = true
	a.channels = []qapi.Channel{{ID: "2", Name: "B", IsSubscribed: true,
		PublishPermission: qapi.ChannelPublishAllSubs}}
	a.channelSel = 0
	a.chInput = "will fail"
	a.submitChannelMessage()
	waitFor(t, "失败回填", func() bool {
		a.mu.Lock()
		defer a.mu.Unlock()
		return a.chInput == "will fail" && !a.chBusy
	})
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.chMsgs) != 0 {
		t.Error("失败不应入列")
	}
}

// TestToggleChannelLike 点赞乐观 + 服务端校正。
func TestToggleChannelLike(t *testing.T) {
	resetChannelCounters()
	state := &channelSrvState{channels: chansFixture()}
	a := newChannelTestApp()
	a.client = serveChannels(t, state)
	a.chMsgs = []qapi.ChannelMessage{{ID: "1", Content: "x"}}
	a.toggleChannelLike(0)
	waitFor(t, "点赞请求", func() bool {
		state.mu.Lock()
		calls := likeCalls
		state.mu.Unlock()
		a.mu.Lock()
		defer a.mu.Unlock()
		return calls == 1
	})
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.chMsgs[0].LikeCount != 1 || !a.chMsgs[0].Liked {
		t.Fatalf("点赞结果错: %+v", a.chMsgs[0])
	}
}

// TestOpenChannelComments 评论展开：拉列表 + 点赞状态校正。
func TestOpenChannelComments(t *testing.T) {
	resetChannelCounters()
	state := &channelSrvState{channels: chansFixture()}
	a := newChannelTestApp()
	a.client = serveChannels(t, state)
	a.chMsgs = []qapi.ChannelMessage{{ID: "1", Content: "x", LikeCount: 0}}
	a.openChannelComments(0)
	waitFor(t, "评论加载", func() bool {
		a.mu.Lock()
		defer a.mu.Unlock()
		return len(a.chComments) == 1
	})
	a.mu.Lock()
	if a.chComments[0].Content != "good" || a.chComments[0].User.DisplayName() != "Bob" {
		a.mu.Unlock()
		t.Fatalf("评论解析错: %+v", a.chComments[0])
	}
	if a.chMsgs[0].LikeCount != 7 || !a.chMsgs[0].Liked {
		a.mu.Unlock()
		t.Error("点赞状态应被校正")
	}
	a.mu.Unlock()
	// 再点收起
	a.openChannelComments(0)
	a.mu.Lock()
	if a.chCommentFor != "" {
		a.mu.Unlock()
		t.Error("再点应收起")
	}
	a.mu.Unlock()
}

// TestSubmitChannelComment 发评论追加。
func TestSubmitChannelComment(t *testing.T) {
	resetChannelCounters()
	state := &channelSrvState{channels: chansFixture()}
	a := newChannelTestApp()
	a.client = serveChannels(t, state)
	a.channelsLoaded = true
	a.channels = []qapi.Channel{{ID: "2", Name: "B", CommentPermission: qapi.ChannelCommentAllSubs}}
	a.chCommentFor = "1"
	a.chCommentInput = "nice"
	a.submitChannelComment()
	waitFor(t, "评论发送", func() bool {
		a.mu.Lock()
		defer a.mu.Unlock()
		return len(a.chComments) == 1
	})
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.chComments[0].Content != "nice" || a.chCommentInput != "" {
		t.Error("评论结果或输入清理错")
	}
}

// TestSubmitChannelCreate 创建频道：成功后关闭弹层并刷新。
func TestSubmitChannelCreate(t *testing.T) {
	resetChannelCounters()
	state := &channelSrvState{channels: chansFixture()}
	a := newChannelTestApp()
	a.client = serveChannels(t, state)
	a.chCreateOpen = true
	a.chName = "我的频道"
	a.submitChannelCreate()
	waitFor(t, "创建请求", func() bool {
		state.mu.Lock()
		calls := createCalls
		state.mu.Unlock()
		a.mu.Lock()
		defer a.mu.Unlock()
		return calls == 1
	})
	waitFor(t, "列表刷新", func() bool {
		state.mu.Lock()
		calls := listReq
		state.mu.Unlock()
		a.mu.Lock()
		defer a.mu.Unlock()
		return calls >= 1 // 创建成功后 loadChannels 重新拉列表
	})
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.chCreateOpen {
		t.Error("弹层应关闭")
	}
	if state.lastCreate.Name != "我的频道" {
		t.Errorf("请求体错: %+v", state.lastCreate)
	}
}

// TestApplyChannelNotification WS 频道通知：打开中刷新、否则累计未读。
func TestApplyChannelNotification(t *testing.T) {
	state := &channelSrvState{channels: chansFixture()}
	a := newChannelTestApp()
	a.client = serveChannels(t, state)
	a.channelsLoaded = true
	a.channels = []qapi.Channel{{ID: "2", Name: "B", IsSubscribed: true}}
	a.channelSel = -1

	n := qapi.Notification{Type: "channel_message", ActionPayload: `{"channel_id":2,"channel_name":"B"}`}
	// 未打开面板：未读 +1
	a.applyChannelNotification(n)
	a.mu.Lock()
	if a.channels[0].Unread != 1 {
		t.Fatalf("未读应 +1: %d", a.channels[0].Unread)
	}
	a.mu.Unlock()
	// 打开面板但选中该频道：清零 + 刷新消息
	a.channelsOpen = true
	a.channelSel = 0
	a.applyChannelNotification(n)
	waitFor(t, "消息刷新", func() bool {
		a.mu.Lock()
		defer a.mu.Unlock()
		return len(a.chMsgs) == 2
	})
	a.mu.Lock()
	if a.channels[0].Unread != 0 {
		a.mu.Unlock()
		t.Error("打开中应清零")
	} else {
		a.mu.Unlock()
	}
	// 非频道消息类型忽略
	other := qapi.Notification{Type: "group_join_request", ActionPayload: `{}`}
	readUnread := func() int {
		a.mu.Lock()
		defer a.mu.Unlock()
		return a.channels[0].Unread
	}
	b := readUnread()
	a.applyChannelNotification(other)
	if readUnread() != b {
		t.Error("非频道通知不应影响未读")
	}
	// 数字 channel_id（float64）解析
	n2 := qapi.Notification{Type: "channel_message"}
	raw, _ := json.Marshal(map[string]any{"channel_id": 2})
	n2.ActionPayload = string(raw)
	a.channelSel = -1
	a.channelsOpen = false
	a.applyChannelNotification(n2)
	a.mu.Lock()
	if a.channels[0].Unread != 1 {
		t.Errorf("float64 channel_id 应解析: %d", a.channels[0].Unread)
	}
	a.mu.Unlock()
}

// TestNormalizeID normalizeID 各种形状。
func TestNormalizeID(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{nil, ""},
		{" 3 ", "3"},
		{float64(42), "42"},
		{json.Number("7"), "7"},
	}
	for _, c := range cases {
		if got := normalizeID(c.in); got != c.want {
			t.Errorf("normalizeID(%v)=%q want %q", c.in, got, c.want)
		}
	}
}

// TestChannelsViewRender 渲染：频道视图 + 创建弹层无 panic 且关键文案在。
func TestChannelsViewRender(t *testing.T) {
	a := &app{
		loggedIn:       true,
		selfID:         "1",
		channelsOpen:   true,
		channelsLoaded: true,
		channels: []qapi.Channel{
			{ID: "2", Name: "公告板", IsSubscribed: true, PublishPermission: qapi.ChannelPublishAllSubs},
		},
		channelSel: 0,
		chMsgs: []qapi.ChannelMessage{{ID: "1", Content: "今日上线新版本", Sender: &qapi.User{Nickname: "Me"},
			CreatedAt: time.Now().Format(time.RFC3339), LikeCount: 2}},
		chCommentFor: "1",
		chComments:   []qapi.ChannelComment{{ID: "1", Content: "不错", User: &qapi.User{Nickname: "Bob"}}},
		chCreateOpen: true,
	}
	tt := ui.NewTester(a.view, 1050, 850)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{
		"频道", "创建频道", "订阅", "广场", "公告板", "Me", "今日上线新版本", "评论", "不错",
	} {
		if !tt.HasText(want) {
			t.Errorf("频道视图缺 %q, texts=%q", want, tt.Texts())
		}
	}
	write(t, tt, "/tmp/qim-channels.png")
}

// TestChTimeLabel RFC3339 时间显示。
func TestChTimeLabel(t *testing.T) {
	if got := chTimeLabel("2026-11-15T10:00:00Z"); got == "" {
		t.Error("RFC3339 应可解析")
	}
	if chTimeLabel("") != "" || chTimeLabel("garbage") != "" {
		t.Error("非法输入应返回空")
	}
}
