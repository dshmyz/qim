package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

// TestCanEditAnnouncement 验证公告编辑权限判断：
// 群主/管理员可以，普通成员不可以，单聊不可以；成员未加载时按可编辑处理
// （入口先显示，由后端最终校验）。
func TestCanEditAnnouncement(t *testing.T) {
	a := &app{selfID: "7"}
	mk := func(role string) []member {
		return []member{
			{UserID: 7, Name: "我", Role: role},
			{UserID: 8, Name: "张三", Role: "member"},
		}
	}
	cases := []struct {
		kind    string
		members []member
		want    bool
	}{
		{"group", mk("owner"), true},
		{"group", mk("admin"), true},
		{"group", mk("member"), false},
		{"discussion", mk("owner"), true},
		{"single", mk("owner"), false},
		{"bot", mk("owner"), false},
		// 成员未加载（详情还没回来）：群聊类显示入口，后端兜底
		{"group", nil, true},
		{"single", nil, false},
	}
	for _, tc := range cases {
		a.convs = []conversation{{ID: "106", Kind: tc.kind, Members: tc.members}}
		a.active = 0
		if got := a.canEditAnnouncement(); got != tc.want {
			t.Errorf("kind=%s role成员表=%v: got %v, want %v", tc.kind, tc.members, got, tc.want)
		}
	}
}

// TestSaveAnnouncementOK 验证保存公告：乐观更新本地 + PUT 到
// /groups/:convID/announcement（body announcement 字段）+ 编辑弹层关闭。
func TestSaveAnnouncementOK(t *testing.T) {
	var mu sync.Mutex
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || !strings.HasSuffix(r.URL.Path, "/groups/106/announcement") {
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
		loggedIn: true,
		client:   qapi.New(srv.URL),
		ctx:      context.Background(),
		convs:    []conversation{{ID: "106", Name: "项目群", Kind: "group"}},
	}
	a.active = 0
	a.annDraft = "  新公告内容  "
	a.annEditorOpen = true

	a.saveAnnouncement()

	a.mu.Lock()
	got := a.convs[0].Announcement
	a.mu.Unlock()
	if got != "新公告内容" {
		t.Errorf("乐观更新未生效: %q", got)
	}
	a.mu.Lock()
	open, draft := a.annEditorOpen, a.annDraft
	a.mu.Unlock()
	if open || draft != "" {
		t.Errorf("编辑弹层未关闭重置: open=%v draft=%q", open, draft)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		p, b := gotPath, gotBody
		mu.Unlock()
		if p != "" {
			if !strings.HasSuffix(p, "/groups/106/announcement") {
				t.Fatalf("端点错: %s，期望 /groups/106/announcement", p)
			}
			var req struct {
				Announcement string `json:"announcement"`
			}
			if err := json.Unmarshal([]byte(b), &req); err != nil {
				t.Fatalf("body 解析失败: %v (%s)", err, b)
			}
			if req.Announcement != "新公告内容" {
				t.Errorf("公告内容错（应 TrimSpace）: %q", req.Announcement)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("公告 PUT 请求未发出")
}

// TestAnnouncementEditorRender headless 渲染编辑弹层：标题、草稿内容、按钮齐全。
func TestAnnouncementEditorRender(t *testing.T) {
	a := &app{
		active:        0,
		loggedIn:      true,
		selfID:        "7",
		annEditorOpen: true,
		annDraft:      "本周五 17:00 团建",
		convs: []conversation{
			{ID: "106", Name: "项目群", Kind: "group", Announcement: "旧公告"},
		},
	}
	tt := ui.NewTester(a.view, 1050, 850)
	tt.Frame()
	tt.Frame()
	// TextArea 的草稿值在 headless 文本导出里不是 text 节点（与消息输入框同款行为），
	// 这里只断言弹层结构与按钮；草稿绑定由 TestSaveAnnouncementOK 覆盖。
	for _, want := range []string{"编辑群公告", "保存", "取消"} {
		if !tt.HasText(want) {
			t.Errorf("公告编辑弹层缺 %q, texts=%q", want, tt.Texts())
		}
	}
	write(t, tt, "/tmp/qim-announcement-editor.png")
}

// TestSaveAnnouncementRollback 验证无权限保存失败时回滚：以服务端详情为准
// 恢复原公告（重拉详情，而不是本地猜测旧值）。
func TestSaveAnnouncementRollback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/groups/106/announcement"):
			// 模拟非管理员被后端拒绝
			io.WriteString(w, `{"code":403,"message":"只有群主或管理员可以设置群公告"}`)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/conversations/106"):
			io.WriteString(w, `{"code":0,"message":"ok","data":{"id":106,"type":"group","name":"项目群","announcement":"旧公告"}}`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	a := &app{
		loggedIn: true,
		client:   qapi.New(srv.URL),
		ctx:      context.Background(),
		convs:    []conversation{{ID: "106", Name: "项目群", Kind: "group"}},
	}
	a.active = 0
	a.annDraft = "越权写入"
	a.annEditorOpen = true

	a.saveAnnouncement()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.Lock()
		v := a.convs[0].Announcement
		a.mu.Unlock()
		if v == "旧公告" {
			return // 回滚成功
		}
		time.Sleep(20 * time.Millisecond)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	t.Fatalf("公告未回滚到服务端值: %q", a.convs[0].Announcement)
}
