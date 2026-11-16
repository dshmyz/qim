package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

// notifFixture 是测试里的一条原始通知 JSON（后端 model.Notification 直传形状）。
func notifFixture(id int, title string, read bool) string {
	return `{"id":` + strconv.Itoa(id) + `,"user_id":1,"type":"system_message","title":"` + title +
		`","content":"` + title + ` 的内容","read":` + boolJS(read) +
		`,"priority":"normal","action_type":"","action_payload":"","pinned":false,` +
		`"important":false,"handled":false,"created_at":"2026-11-15T09:30:00Z"}`
}

func boolJS(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// TestNotificationParse 验证 qapi.Notification 解析：数字 id、created_at 归一化毫秒。
func TestNotificationParse(t *testing.T) {
	var n qapi.Notification
	raw := `{"id":12,"title":"入群申请","content":"张三申请加入项目群","read":false,` +
		`"action_type":"approve_reject","action_payload":"{\"conversation_id\":7,\"user_id\":42}",` +
		`"pinned":false,"important":true,"handled":false,"created_at":"2026-11-15T09:30:00Z"}`
	if err := json.Unmarshal([]byte(raw), &n); err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if n.ID.String() != "12" {
		t.Errorf("id=%q", n.ID)
	}
	if n.Timestamp != time.Date(2026, 11, 15, 9, 30, 0, 0, time.UTC).UnixMilli() {
		t.Errorf("timestamp=%d", n.Timestamp)
	}
	if !n.Important || n.Read {
		t.Errorf("flags read=%v important=%v", n.Read, n.Important)
	}
	if btns := n.ActionButtons(); len(btns) != 2 || btns[0].Value != "approve" {
		t.Errorf("approve_reject 按钮 = %+v", btns)
	}
	// handled=true 不出按钮
	n.Handled = true
	if btns := n.ActionButtons(); len(btns) != 0 {
		t.Errorf("handled 后仍有按钮 %+v", btns)
	}
}

// TestNotifActionButtonsTable 覆盖 qim-client ACTION_BUTTONS 的全部映射。
func TestNotifActionButtonsTable(t *testing.T) {
	cases := map[string][]string{
		"accept_ignore":      {"accept", "ignore"},
		"confirm_reschedule": {"confirm", "reschedule"},
		"approve_reject":     {"approve", "reject"},
		"todo_mass_confirm":  {"confirm_create", "todo_ignore"},
		"unknown_type":       {},
	}
	for at, want := range cases {
		n := qapi.Notification{ActionType: at}
		btns := n.ActionButtons()
		got := make([]string, len(btns))
		for i, b := range btns {
			got[i] = b.Value
		}
		if len(got) != len(want) {
			t.Fatalf("%s: got %v want %v", at, got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("%s: got %v want %v", at, got, want)
			}
		}
	}
}

// TestOpenNotifsLoads 验证通知中心加载链路：打开 → GET /notifications →
// 列表写入、未读数、分页 hasMore 全部正确。
func TestOpenNotifsLoads(t *testing.T) {
	var mu sync.Mutex
	var seenPage string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/notifications") {
			mu.Lock()
			seenPage = r.URL.Query().Get("page") + "/" + r.URL.Query().Get("pageSize")
			mu.Unlock()
			// pageSize 参数名必须是 camelCase（handler 只认这个）
			if r.URL.Query().Get("pageSize") == "" {
				t.Errorf("缺少 pageSize 参数")
			}
			io := strings.NewReader(`{"code":0,"message":"ok","data":{"list":[` +
				notifFixture(1, "第一条", false) + "," +
				notifFixture(2, "第二条", true) +
				`],"total":2,"page":1}}`)
			_, _ = io.WriteTo(w)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	a := &app{
		loggedIn:  true,
		client:    qapi.New(srv.URL),
		ctx:       context.Background(),
		notifsTab: "all",
	}

	a.openNotifs()
	a.mu.Lock()
	open0, busy0 := a.notifsOpen, a.notifsBusy
	a.mu.Unlock()
	if !open0 || !busy0 {
		t.Fatalf("面板未进入加载态: open=%v busy=%v", open0, busy0)
	}

	deadline := time.Now().Add(3 * time.Second)
	var loaded bool
	for time.Now().Before(deadline) {
		a.mu.Lock()
		done := !a.notifsBusy && len(a.notifs) == 2
		a.mu.Unlock()
		if done {
			loaded = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !loaded {
		a.mu.Lock()
		busy, n := a.notifsBusy, len(a.notifs)
		a.mu.Unlock()
		t.Fatalf("通知未加载完成: busy=%v n=%d", busy, n)
	}
	mu.Lock()
	paged := seenPage
	mu.Unlock()
	if paged != "1/50" {
		t.Errorf("分页参数 = %q", paged)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.unreadNotifs != 1 {
		t.Errorf("unreadNotifs=%d", a.unreadNotifs)
	}
	if a.notifsHasMore {
		t.Errorf("不足一页却 hasMore")
	}
	if a.notifs[0].Title != "第一条" {
		t.Errorf("list[0].Title=%q", a.notifs[0].Title)
	}
}

// TestNotifsStaleDrop 验证关掉面板后迟到的响应不会写入状态。
func TestNotifsStaleDrop(t *testing.T) {
	a := &app{
		loggedIn:  true,
		client:    qapi.New("http://127.0.0.1:1"), // 不会连上
		ctx:       context.Background(),
		notifsTab: "all",
	}
	a.openNotifs()
	a.closeNotifs()
	time.Sleep(100 * time.Millisecond)
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.notifsOpen || a.notifsBusy || len(a.notifs) != 0 {
		t.Errorf("关闭后被迟到结果污染: open=%v busy=%v n=%d", a.notifsOpen, a.notifsBusy, len(a.notifs))
	}
}

// TestClickNotifMarksRead 验证点击未读通知 → 乐观置已读 → PUT read 成功保持。
func TestClickNotifMarksRead(t *testing.T) {
	var readCalls int
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/notifications/3/read") {
			mu.Lock()
			readCalls++
			mu.Unlock()
			_, _ = strings.NewReader(`{"code":0,"message":"标记已读成功"}`).WriteTo(w)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	a := &app{
		loggedIn:  true,
		client:    qapi.New(srv.URL),
		ctx:       context.Background(),
		notifsTab: "all",
		notifs: []qapi.Notification{
			{ID: "2", Title: "已读过的", Read: true},
			{ID: "3", Title: "未读的", Read: false},
		},
		unreadNotifs: 1,
	}
	a.clickNotif("3")
	waitFor(t, "标已读请求", func() bool { mu.Lock(); defer mu.Unlock(); return readCalls == 1 })
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.notifs[1].Read || a.unreadNotifs != 0 {
		t.Errorf("read=%v unread=%d", a.notifs[1].Read, a.unreadNotifs)
	}
	// 已读的再点不会发请求
	a.clickNotif("2")
	time.Sleep(80 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if readCalls != 1 {
		t.Errorf("已读条目被重复标记: calls=%d", readCalls)
	}
}

// TestClickNotifRollback 验证标已读失败时回滚成未读。
func TestClickNotifRollback(t *testing.T) {
	a := &app{
		loggedIn:     true,
		client:       qapi.New("http://127.0.0.1:1"),
		ctx:          context.Background(),
		notifsTab:    "all",
		notifs:       []qapi.Notification{{ID: "3", Title: "未读的", Read: false}},
		unreadNotifs: 1,
	}
	a.clickNotif("3")
	time.Sleep(150 * time.Millisecond) // 连接失败走完回滚路径
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.notifs[0].Read || a.unreadNotifs != 1 {
		t.Errorf("未回滚: read=%v unread=%d", a.notifs[0].Read, a.unreadNotifs)
	}
}

// TestMarkAllNotifsRead 验证一键已读成功路径（全部置已读、未读数清零）。
func TestMarkAllNotifsRead(t *testing.T) {
	var called bool
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/notifications/read-all") {
			mu.Lock()
			called = true
			mu.Unlock()
			_, _ = strings.NewReader(`{"code":0,"message":"ok"}`).WriteTo(w)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	a := &app{
		loggedIn:  true,
		client:    qapi.New(srv.URL),
		ctx:       context.Background(),
		notifsTab: "all",
		notifs: []qapi.Notification{
			{ID: "1", Read: false},
			{ID: "2", Read: false},
		},
		unreadNotifs: 2,
	}
	a.markAllNotifsRead()
	waitFor(t, "一键已读请求", func() bool { mu.Lock(); defer mu.Unlock(); return called })
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.unreadNotifs != 0 || !a.notifs[0].Read || !a.notifs[1].Read {
		t.Errorf("一键已读未生效: unread=%d", a.unreadNotifs)
	}
}

// TestClearNotifsRollback 验证清空失败时整表回滚。
func TestClearNotifsRollback(t *testing.T) {
	a := &app{
		loggedIn:     true,
		client:       qapi.New("http://127.0.0.1:1"),
		ctx:          context.Background(),
		notifsTab:    "all",
		notifs:       []qapi.Notification{{ID: "1", Read: false}},
		unreadNotifs: 1,
	}
	a.clearNotifs()
	time.Sleep(150 * time.Millisecond)
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.notifs) != 1 || a.unreadNotifs != 1 {
		t.Errorf("清空失败未回滚: n=%d unread=%d", len(a.notifs), a.unreadNotifs)
	}
}

// TestDeleteNotifRollback 验证删除失败时插回原位置。
func TestDeleteNotifRollback(t *testing.T) {
	a := &app{
		loggedIn:  true,
		client:    qapi.New("http://127.0.0.1:1"),
		ctx:       context.Background(),
		notifsTab: "all",
		notifs: []qapi.Notification{
			{ID: "1", Title: "A", Read: true},
			{ID: "2", Title: "B", Read: false},
			{ID: "3", Title: "C", Read: true},
		},
		unreadNotifs: 1,
	}
	a.deleteNotif("2")
	a.mu.Lock()
	if len(a.notifs) != 2 {
		t.Fatalf("删除未生效: n=%d", len(a.notifs))
	}
	a.mu.Unlock()
	time.Sleep(150 * time.Millisecond)
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.notifs) != 3 || a.notifs[1].Title != "B" {
		t.Errorf("删除失败未插回原位: %+v", a.notifs)
	}
	if a.unreadNotifs != 1 {
		t.Errorf("回滚后未读数=%d", a.unreadNotifs)
	}
}

// TestNotifToggleFlag 验证置顶/重要 toggle 的乐观翻转（fake server 接受 PATCH）。
func TestNotifToggleFlag(t *testing.T) {
	var mu sync.Mutex
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "/notifications/1") {
			buf := make([]byte, 256)
			n, _ := r.Body.Read(buf)
			mu.Lock()
			gotBody = string(buf[:n])
			mu.Unlock()
			_, _ = strings.NewReader(`{"code":0,"message":"操作成功","data":{"pinned":true}}`).WriteTo(w)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	a := &app{
		loggedIn:  true,
		client:    qapi.New(srv.URL),
		ctx:       context.Background(),
		notifsTab: "all",
		notifs:    []qapi.Notification{{ID: "1", Pinned: false}},
	}
	a.toggleNotifPin("1")
	waitFor(t, "PATCH 请求", func() bool { mu.Lock(); defer mu.Unlock(); return gotBody != "" })
	mu.Lock()
	body := gotBody
	mu.Unlock()
	if !strings.Contains(body, `"pinned"`) {
		t.Errorf("PATCH body=%s", body)
	}
	a.mu.Lock()
	pinned := a.notifs[0].Pinned
	a.mu.Unlock()
	if !pinned {
		t.Errorf("置顶未乐观翻转")
	}

	a.toggleNotifImportant("1")
	time.Sleep(80 * time.Millisecond)
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.notifs[0].Important {
		t.Errorf("重要未乐观翻转")
	}
}

// TestHandleNotifAction 验证通用动作（accept/ignore/…）走 PATCH 分发并置 handled。
func TestHandleNotifAction(t *testing.T) {
	var mu sync.Mutex
	var gotAction string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "/notifications/5") {
			var body struct {
				Action string `json:"action"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			mu.Lock()
			gotAction = body.Action
			mu.Unlock()
			io := strings.NewReader(`{"code":0,"message":"操作成功","data":{"notification":{"id":5,"handled":true,"read":true}}}`)
			_, _ = io.WriteTo(w)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	a := &app{
		loggedIn:  true,
		client:    qapi.New(srv.URL),
		ctx:       context.Background(),
		notifsTab: "all",
		notifs: []qapi.Notification{
			{ID: "5", ActionType: "accept_ignore", Read: false},
		},
		unreadNotifs: 1,
	}
	a.handleNotifAction("5", "accept")
	waitFor(t, "PATCH 动作", func() bool { mu.Lock(); defer mu.Unlock(); return gotAction == "accept" })
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.notifs[0].Handled || !a.notifs[0].Read || a.unreadNotifs != 0 {
		t.Errorf("动作未落地: handled=%v read=%v unread=%d", a.notifs[0].Handled, a.notifs[0].Read, a.unreadNotifs)
	}
}

// TestHandleApproveAction 验证 approve_reject 特殊分支：同意入群走
// POST /groups/:cid/members 而不是 PATCH action。
func TestHandleApproveAction(t *testing.T) {
	var mu sync.Mutex
	var joined bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/groups/7/members") {
			var body struct {
				MemberIDs []uint `json:"member_ids"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			// 服务端契约：member_ids 是数字数组（qim-server group_handler.go）
			if len(body.MemberIDs) != 1 || body.MemberIDs[0] != 42 {
				t.Errorf("member_ids=%v", body.MemberIDs)
			}
			mu.Lock()
			joined = true
			mu.Unlock()
			_, _ = strings.NewReader(`{"code":0,"message":"ok"}`).WriteTo(w)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	a := &app{
		loggedIn:  true,
		client:    qapi.New(srv.URL),
		ctx:       context.Background(),
		notifsTab: "all",
		notifs: []qapi.Notification{
			{ID: "9", ActionType: "approve_reject",
				ActionPayload: `{"conversation_id":7,"user_id":42}`},
		},
	}
	a.handleNotifAction("9", "approve")
	waitFor(t, "同意入群请求", func() bool { mu.Lock(); defer mu.Unlock(); return joined })
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.notifs[0].Handled {
		t.Errorf("同意后未置 handled")
	}
}

// TestApplyNewNotification 验证 WS 插队：去重、未读数、面板关闭时列表不插。
func TestApplyNewNotification(t *testing.T) {
	a := &app{notifsTab: "all"}
	// 面板没开、没加载过：只累计，不插列表（避免打开前有半截列表）
	a.applyNewNotification(qapi.Notification{ID: "1", Read: false})
	a.mu.Lock()
	n := len(a.notifs)
	a.mu.Unlock()
	if n != 0 {
		t.Fatalf("面板未打开就插入了列表 n=%d", n)
	}
	// 打开并加载过一页
	a.mu.Lock()
	a.notifsOpen = true
	a.notifsPage = 1
	a.notifs = []qapi.Notification{{ID: "0", Read: true}}
	a.mu.Unlock()
	a.applyNewNotification(qapi.Notification{ID: "1", Read: false})
	a.applyNewNotification(qapi.Notification{ID: "1", Read: false}) // 去重
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.notifs) != 2 || a.notifs[0].ID.String() != "1" {
		t.Errorf("插队结果: %+v", a.notifs)
	}
	if a.unreadNotifs != 1 {
		t.Errorf("unread=%d", a.unreadNotifs)
	}
}

// TestNotifTabFilter 验证 tab 过滤与置顶排序。
func TestNotifTabFilter(t *testing.T) {
	a := &app{
		notifsTab: "all",
		notifs: []qapi.Notification{
			{ID: "1", Read: true},
			{ID: "2", Read: false, Important: true},
			{ID: "3", Read: false, Pinned: true},
			{ID: "4", Read: true, Important: true},
		},
	}
	all := a.filteredNotifs()
	if len(all) != 4 || all[0].ID.String() != "3" {
		t.Errorf("置顶未排前: %+v", all)
	}
	a.mu.Lock()
	a.notifsTab = "unread"
	a.mu.Unlock()
	unread := a.filteredNotifs()
	if len(unread) != 2 || unread[0].ID.String() != "3" {
		t.Errorf("unread 过滤: %+v", unread)
	}
	a.mu.Lock()
	a.notifsTab = "important"
	a.mu.Unlock()
	imp := a.filteredNotifs()
	if len(imp) != 2 || imp[0].ID.String() != "2" {
		t.Errorf("important 过滤: %+v", imp)
	}
}

// TestNotificationCenterRender headless 渲染通知中心面板：
// 头部按钮、tabs、条目内容、动作按钮齐全，未读徽标可见。
func TestNotificationCenterRender(t *testing.T) {
	a := &app{
		loggedIn:   true,
		selfID:     "1",
		active:     0,
		convs:      []conversation{{ID: "106", Name: "项目群", Kind: "group"}},
		notifsOpen: true,
		notifsTab:  "all",
		notifs: []qapi.Notification{
			{ID: "1", Title: "入群申请", Content: "张三申请加入项目群", ActionType: "approve_reject",
				Pinned: true, Timestamp: time.Now().UnixMilli()},
			{ID: "2", Title: "系统通知", Content: "您的账号已成功登录", Read: true,
				Important: true, Timestamp: time.Now().UnixMilli()},
		},
		unreadNotifs: 1,
	}
	tt := ui.NewTester(a.view, 1050, 850)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{
		"通知中心", "一键已读", "清空", "全部", "未读", "重要",
		"入群申请", "张三申请加入项目群", "同意", "拒绝", "系统通知",
	} {
		if !tt.HasText(want) {
			t.Errorf("通知面板缺 %q, texts=%q", want, tt.Texts())
		}
	}
	write(t, tt, "/tmp/qim-notifications.png")
}

// TestNotifBellBadgeRender headless 渲染侧栏铃铛 + 未读徽标。
func TestNotifBellBadgeRender(t *testing.T) {
	a := &app{
		loggedIn:     true,
		selfID:       "1",
		active:       0,
		convs:        []conversation{{ID: "106", Name: "项目群", Kind: "group"}},
		unreadNotifs: 120, // >99 封顶
	}
	tt := ui.NewTester(a.view, 1050, 850)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("99+") {
		t.Errorf("未读徽标缺 99+, texts=%q", tt.Texts())
	}
	write(t, tt, "/tmp/qim-bell-badge.png")
}
