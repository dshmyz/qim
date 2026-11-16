package client

// 聊天窗口顶部（ChatHeader.vue / ChatHeaderActions.vue）对齐验证：
// 会话列表摊平字段映射、状态行内容、更多菜单动作的真实 HTTP 契约、截图。

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

// TestConvsFromAPIHeaderFields：后端 GetConversations 对 single 会话会摊平
// 对端 User 的 ip / signature / status / other_member_id，头部副标题与资料卡
// 全靠这些字段，之前 qapi.Conversation 漏接导致单聊副标题是假的。
func TestConvsFromAPIHeaderFields(t *testing.T) {
	list := []qapi.Conversation{{
		ID: qapi.ID("2"), Type: "single", Name: "张三", Avatar: "/a.png",
		IP: "10.0.1.23", Signature: "今天也要开心", Status: "online",
		OtherMemberID: 8, OtherMemberType: "user",
	}}
	got := convsFromAPI(list)
	c := got[0]
	if c.IP != "10.0.1.23" || c.Signature != "今天也要开心" || c.Status != "online" {
		t.Errorf("单聊副标题字段没接上: %+v", c)
	}
	if c.PeerID != 8 || c.PeerType != "user" || c.Avatar != "/a.png" {
		t.Errorf("对端身份字段没接上: %+v", c)
	}
	if !c.Online {
		t.Error("status=online 应同步 online 标志")
	}
}

// TestHeaderStatusRender：原版状态行只有「单聊 IP+签名」「群聊内联公告」，
// 没有「N 名成员」「单聊/群聊」这类自造文案；bot 会话带「AI 机器人」徽标。
func TestHeaderStatusRender(t *testing.T) {
	cases := []struct {
		name string
		conv conversation
		want []string
		dont []string
	}{
		{
			name: "单聊 IP 与签名",
			conv: conversation{ID: "2", Name: "张三", Kind: "single",
				IP: "10.0.1.23", Signature: "今天也要开心", Status: "online", PeerID: 8},
			want: []string{"张三", "10.0.1.23", "今天也要开心"},
			dont: []string{"单聊", "名成员"},
		},
		{
			name: "单聊无对端字段时状态行留空",
			conv: conversation{ID: "3", Name: "李四", Kind: "single"},
			want: []string{"李四"},
			dont: []string{"单聊", "名成员", "未知"},
		},
		{
			name: "群聊无公告不显示成员数",
			conv: conversation{ID: "106", Name: "项目群", Kind: "group",
				Members: []member{{UserID: 8, Name: "张三"}}},
			want: []string{"项目群"},
			dont: []string{"名成员", "群公告"},
		},
		{
			name: "群聊公告内联在头部",
			conv: conversation{ID: "106", Name: "项目群", Kind: "group",
				Announcement: "本周五 17:00 团建"},
			want: []string{"项目群", "本周五 17:00 团建"},
			dont: []string{"群公告", "名成员"},
		},
		{
			name: "bot 会话带 AI 机器人徽标",
			conv: conversation{ID: "9", Name: "写作助手", Kind: "bot"},
			want: []string{"写作助手", "AI 机器人"},
			dont: []string{"机器人会话"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &app{active: 0, loggedIn: true, selfID: "7", convs: []conversation{tc.conv}}
			tt := ui.NewTester(a.view, 1050, 703)
			tt.Frame()
			tt.Frame()
			for _, w := range tc.want {
				if !tt.HasText(w) {
					t.Errorf("头部缺 %q, texts=%q", w, tt.Texts())
				}
			}
			for _, d := range tc.dont {
				if tt.HasText(d) {
					t.Errorf("头部不该出现 %q, texts=%q", d, tt.Texts())
				}
			}
		})
	}
}

// TestSetGroupNameContract：更多菜单「修改群名称」→ PUT /groups/:convID
// body 只有 name，本地群名乐观更新。
func TestSetGroupNameContract(t *testing.T) {
	var mu sync.Mutex
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/v1/groups/106" {
			w.WriteHeader(404)
			return
		}
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		gotPath, gotBody = r.URL.Path, string(b)
		mu.Unlock()
		io.WriteString(w, `{"code":0,"message":"ok"}`)
	}))
	defer srv.Close()

	a := &app{
		loggedIn: true, client: qapi.New(srv.URL), ctx: context.Background(),
		convs: []conversation{{ID: "106", Name: "旧群名", Kind: "group"}},
	}
	a.active = 0
	a.openGroupNameEditor()
	if a.nameDraft != "旧群名" || !a.nameEditorOpen {
		t.Fatalf("编辑弹层未预填当前群名: draft=%q open=%v", a.nameDraft, a.nameEditorOpen)
	}
	a.nameDraft = "  新群名  "
	a.saveGroupName()
	a.mu.RLock()
	gotName, stillOpen := a.convs[0].Name, a.nameEditorOpen
	a.mu.RUnlock()
	if gotName != "新群名" || stillOpen {
		t.Errorf("乐观更新/关闭弹层失败: name=%q open=%v", gotName, stillOpen)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		p, b := gotPath, gotBody
		mu.Unlock()
		if p != "" {
			if p != "/api/v1/groups/106" {
				t.Fatalf("端点错: %s", p)
			}
			var req struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal([]byte(b), &req); err != nil {
				t.Fatalf("body 解析失败: %v (%s)", err, b)
			}
			if req.Name != "新群名" {
				t.Errorf("群名未 TrimSpace 或字段错: %q", req.Name)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("改名 PUT 请求未发出")
}

// TestDissolveGroupContract：更多菜单「解散群聊」→ DELETE /groups/:convID，
// 成功后清掉当前会话。
func TestDissolveGroupContract(t *testing.T) {
	var mu sync.Mutex
	got := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/groups/106":
			mu.Lock()
			got = true
			mu.Unlock()
			io.WriteString(w, `{"code":0,"message":"ok"}`)
		case r.URL.Path == "/api/v1/conversations":
			io.WriteString(w, `{"code":0,"data":[]}`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	a := &app{
		loggedIn: true, client: qapi.New(srv.URL), ctx: context.Background(),
		convs: []conversation{{ID: "106", Name: "项目群", Kind: "group"}},
		msgs:  []message{{Key: "m1", Text: "hi"}},
	}
	a.active = 0
	a.openDissolveConfirm()
	if !a.dissolveConfirmOpen {
		t.Fatal("确认框没打开")
	}
	a.dissolveGroup()
	if a.dissolveConfirmOpen {
		t.Error("确认框没关掉")
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		ok := got
		mu.Unlock()
		if ok {
			// 后台 goroutine 经 mutate 写 active/msgs，读取要走同一把锁，
			// 否则 -race 会报 DATA RACE（headless 下 mutate 直接执行）。
			a.mu.RLock()
			active, nmsg := a.active, len(a.msgs)
			a.mu.RUnlock()
			if active == -1 && nmsg == 0 {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	a.mu.RLock()
	active, nmsg := a.active, len(a.msgs)
	a.mu.RUnlock()
	t.Fatalf("解散链路未完成: got=%v active=%d msgs=%d", got, active, nmsg)
}

// TestGroupNameEditorRender：改群名弹层与解散确认框结构齐全。
func TestGroupNameEditorRender(t *testing.T) {
	a := &app{
		active: 0, loggedIn: true, selfID: "7",
		nameEditorOpen: true, nameDraft: "项目群",
		convs: []conversation{{ID: "106", Name: "项目群", Kind: "group"}},
	}
	tt := ui.NewTester(a.view, 1050, 703)
	tt.Frame()
	tt.Frame()
	for _, w := range []string{"修改群名称", "保存", "取消"} {
		if !tt.HasText(w) {
			t.Errorf("改名弹层缺 %q, texts=%q", w, tt.Texts())
		}
	}

	b := &app{
		active: 0, loggedIn: true, selfID: "7",
		dissolveConfirmOpen: true,
		convs:               []conversation{{ID: "106", Name: "项目群", Kind: "group"}},
	}
	tt2 := ui.NewTester(b.view, 1050, 703)
	tt2.Frame()
	tt2.Frame()
	for _, w := range []string{"确认解散群聊", "确定解散", "取消"} {
		if !tt2.HasText(w) {
			t.Errorf("解散确认框缺 %q, texts=%q", w, tt2.Texts())
		}
	}
}

// TestShotChatHeader 出三张头部截图：群聊（带公告 + 成员侧栏）、
// 单聊（IP/签名）、bot（AI 机器人徽标），供肉眼核对顶部对齐。
func TestShotChatHeader(t *testing.T) {
	msgs := []message{
		{Key: "day", Pill: "今天"},
		{Key: "m1", ServerID: "1", Sender: "张三", Text: "收到", At: "10:00", Stamp: time.Now().UnixMilli()},
	}
	cases := []struct {
		file string
		conv conversation
	}{
		{"/tmp/qim-header-group.png", conversation{
			ID: "106", Name: "项目群", Kind: "group", Color: "#3385ff",
			Announcement: "本周五 17:00 团建，请提前报名",
			Members: []member{
				{UserID: 7, Name: "我", Role: "owner"},
				{UserID: 8, Name: "张三"}, {UserID: 9, Name: "李四"},
			},
		}},
		{"/tmp/qim-header-single.png", conversation{
			ID: "2", Name: "张三", Kind: "single", Color: "#10b981",
			IP: "10.0.1.23", Signature: "今天也要开心", Status: "online", PeerID: 8,
		}},
		{"/tmp/qim-header-bot.png", conversation{
			ID: "9", Name: "写作助手", Kind: "bot", Color: "#7c3aed",
		}},
	}
	for _, tc := range cases {
		a := &app{
			active: 0, loggedIn: true, selfID: "7", userName: "我",
			convs: []conversation{tc.conv}, msgs: msgs,
		}
		tt := ui.NewTester(a.view, 1050, 703)
		tt.Frame()
		tt.Frame()
		write(t, tt, tc.file)
	}
}
