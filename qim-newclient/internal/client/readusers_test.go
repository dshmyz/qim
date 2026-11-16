package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

// TestOpenReadUsersLoads 验证已读详情链路：打开弹层 → 异步 GET
// /messages/:id/read-users → 弹层展示 read_users 与计数；关掉后丢弃迟到的结果。
func TestOpenReadUsersLoads(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/messages/55/read-users") {
			io := strings.NewReader(`{"code":0,"message":"ok","data":{"read_users":[` +
				`{"id":8,"username":"zhangsan","nickname":"张三","avatar":""},` +
				`{"id":9,"username":"lisi","nickname":"李四","avatar":""}],` +
				`"read_count":2,"total_members":5}}`)
			_, _ = io.WriteTo(w)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	a := &app{
		loggedIn: true,
		client:   qapi.New(srv.URL),
		ctx:      context.Background(),
	}
	m := message{Key: "m-1", Mine: true, ServerID: "55"}

	a.openReadUsers(m)
	a.mu.Lock()
	open0, busy0 := a.readUsersOpen, a.readUsersBusy
	a.mu.Unlock()
	if !open0 || !busy0 {
		t.Fatalf("弹层未进入加载态: open=%v busy=%v", open0, busy0)
	}

	deadline := time.Now().Add(3 * time.Second)
	var loaded bool
	for time.Now().Before(deadline) {
		a.mu.Lock()
		done := !a.readUsersBusy && a.readUsersInfo != nil
		a.mu.Unlock()
		if done {
			loaded = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !loaded {
		a.mu.Lock()
		busy, info := a.readUsersBusy, a.readUsersInfo
		a.mu.Unlock()
		t.Fatalf("已读列表未加载完成: busy=%v info=%v", busy, info)
	}
	a.mu.Lock()
	nUsers, total := len(a.readUsersInfo.ReadUsers), a.readUsersInfo.TotalMembers
	a.mu.Unlock()
	if nUsers != 2 || total != 5 {
		t.Errorf("解析错: users=%d total=%d", nUsers, total)
	}

	a.closeReadUsers()
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.readUsersOpen || a.readUsersInfo != nil {
		t.Errorf("关闭未清理: open=%v info=%v", a.readUsersOpen, a.readUsersInfo)
	}
}

// TestReadUsersStaleDrop 验证关掉弹层后，迟到的响应不会重新写入状态。
func TestReadUsersStaleDrop(t *testing.T) {
	a := &app{
		loggedIn: true,
		client:   qapi.New("http://127.0.0.1:1"), // 不会真连上，等响应时弹层已关
		ctx:      context.Background(),
	}
	a.openReadUsers(message{Key: "m-1", Mine: true, ServerID: "55"})
	a.closeReadUsers()
	time.Sleep(100 * time.Millisecond) // 给失败路径留时间走完
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.readUsersOpen || a.readUsersBusy || a.readUsersInfo != nil {
		t.Errorf("关闭后状态被迟到结果污染: open=%v busy=%v info=%v",
			a.readUsersOpen, a.readUsersBusy, a.readUsersInfo)
	}
}

// TestReadUsersPickerRender headless 渲染已读弹层：标题、用户行、汇总、按钮齐全。
func TestReadUsersPickerRender(t *testing.T) {
	a := &app{
		active:        0,
		loggedIn:      true,
		readUsersOpen: true,
		readUsersInfo: &qapi.MessageReadInfo{
			ReadUsers: []qapi.ReadUser{
				{ID: 8, Nickname: "张三"},
				{ID: 9, Nickname: "李四"},
			},
			ReadCount:    2,
			TotalMembers: 5,
		},
		convs: []conversation{{ID: "106", Name: "项目群", Kind: "group"}},
	}
	tt := ui.NewTester(a.view, 1050, 850)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{"已读详情", "张三", "李四", "已读 2 人 / 共 5 人，未读 3 人", "关闭"} {
		if !tt.HasText(want) {
			t.Errorf("已读弹层缺 %q, texts=%q", want, tt.Texts())
		}
	}
	write(t, tt, "/tmp/qim-read-users.png")
}
