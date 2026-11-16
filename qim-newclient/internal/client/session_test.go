package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/gorilla/websocket"

	"qim-newclient/internal/qapi"
)

// fakeBackend 按真实 qim-server 的线格式回包：data.list 包一层、
// created_at 是 RFC3339、sender 里是 nickname、不下发 isSelf。
// 用这个形状测，避免「假服务端能跑、真后端全空」。
type fakeBackend struct {
	sends   atomic.Int64
	reads   atomic.Int64
	lastMsg atomic.Value // string
}

func (f *fakeBackend) handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	reply := func(v any) { _ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "message": "ok", "data": v}) }

	switch {
	case r.URL.Path == "/api/v1/auth/login":
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["password"] != "pw" {
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 1001, "message": "密码错误"})
			return
		}
		reply(map[string]any{
			"token": "tkn_native", "refresh_token": "rtk_native",
			"user": map[string]any{"id": 7, "username": "grace", "nickname": "测试用户"},
		})

	case r.URL.Path == "/api/v1/conversations":
		reply(map[string]any{
			"list": []any{
				map[string]any{
					"id": 101, "type": "group", "name": "技术交流群", "unread_count": 3,
					"last_message_at": "2026-11-14T10:00:00+08:00",
					"last_message":    map[string]any{"id": 900, "content": "早", "created_at": "2026-11-14T10:00:00+08:00"},
				},
				map[string]any{
					"id": "c2", "type": "single", "other_member_name": "张三", "unread_count": 0,
					"last_message_at": "2026-11-14T09:00:00+08:00",
				},
			},
			"total": 2,
		})

	case strings.HasSuffix(r.URL.Path, "/messages") && r.Method == http.MethodGet:
		reply(map[string]any{
			"messages": []any{
				map[string]any{
					"id": 1, "conversation_id": 101, "sender_id": 8, "type": "text", "content": "早上好",
					"created_at": "2026-11-14T09:30:00+08:00", "is_read": true,
					"sender": map[string]any{"id": 8, "nickname": "张三"},
				},
				map[string]any{
					"id": 2, "conversation_id": 101, "sender_id": 7, "type": "text", "content": "早，构建挂了",
					"created_at": "2026-11-14T09:31:00+08:00",
					"sender":     map[string]any{"id": 7, "nickname": "测试用户"},
				},
			},
			"pagination": map[string]any{"current_page": 1, "total_pages": 1, "total": 2, "page_size": 50},
		})

	case strings.HasSuffix(r.URL.Path, "/messages") && r.Method == http.MethodPost:
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.sends.Add(1)
		if s, ok := body["content"].(string); ok {
			f.lastMsg.Store(s)
		}
		reply(map[string]any{
			"id": 3, "conversation_id": 101, "sender_id": 7, "type": "text",
			"content": body["content"], "created_at": "2026-11-14T10:05:00+08:00",
		})

	case strings.HasSuffix(r.URL.Path, "/read"):
		f.reads.Add(1)
		reply(map[string]any{})

	case r.URL.Path == "/api/v1/ws":
		f.serveWS(w, r)

	default:
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 404, "message": "not found"})
	}
}

func (f *fakeBackend) serveWS(w http.ResponseWriter, r *http.Request) {
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	var first struct {
		Type string `json:"type"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := conn.ReadJSON(&first); err != nil || first.Type != "auth" {
		return
	}
	_ = conn.WriteJSON(map[string]any{"type": "auth_success", "data": map[string]any{"user_id": 7}})
	_ = conn.WriteJSON(map[string]any{
		"type": "new_message",
		"data": map[string]any{
			"id": 4, "conversation_id": 101, "sender_id": 8, "type": "text",
			"content": "这是服务端推来的新消息", "created_at": "2026-11-14T10:06:00+08:00",
			"sender": map[string]any{"id": 8, "nickname": "张三"},
		},
	})
	// 保持连接，让 StatusConnected 有一个稳定窗口
	time.Sleep(1200 * time.Millisecond)
}

func newTestApp(url string) *app {
	ctx, cancel := context.WithCancel(context.Background())
	return &app{
		active:      -1,
		client:      qapi.New(url),
		ctx:         ctx,
		cancel:      cancel,
		loginServer: url,
		loginUser:   "grace",
		loginPass:   "pw",
	}
}

// read / readBool：headless 模式下读界面状态，和后台网络回调的写入互斥。
func (a *app) read(fn func()) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	fn()
}

func (a *app) readBool(fn func() bool) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return fn()
}

// waitFor 轮询异步状态，避免用固定 Sleep 造成偶发失败。
func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("超时等待：%s", what)
}

// TestNativeSessionFlow 验证纯原生客户端的完整数据流：
// 登录 → 会话列表 → 历史消息 → 发送 → WS 推送 → 退出登录。
func TestNativeSessionFlow(t *testing.T) {
	f := &fakeBackend{}
	srv := httptest.NewServer(http.HandlerFunc(f.handler))
	defer srv.Close()

	a := newTestApp(srv.URL)
	ctx := a.ctx

	// 1. 登录：token 归 Client，界面只拿到显示名和自己的 id
	ok, err := a.login(ctx, srv.URL, "grace", "pw", "")
	if err != nil || !ok {
		t.Fatalf("登录失败: ok=%v err=%v", ok, err)
	}
	if !a.loggedIn || a.selfID != "7" || a.userName != "测试用户" {
		t.Fatalf("登录态没写进 app: %+v", a)
	}
	if tok, _ := a.client.Session(); tok != "tkn_native" {
		t.Fatalf("token 没进 Client: %q", tok)
	}

	// 2. 会话列表：真实后端的 data.list 结构
	if err := a.refreshConversations(ctx); err != nil {
		t.Fatalf("拉会话失败: %v", err)
	}
	if len(a.convs) != 2 || a.convs[0].ID != "101" || a.convs[0].Unread != 3 {
		t.Fatalf("会话列表错: %+v", a.convs)
	}
	if a.convs[1].Name != "张三" || a.convs[0].Preview != "早" {
		t.Fatalf("显示名/预览没解出来: %+v", a.convs)
	}

	// 3. 选中会话：拉历史 + 已读，mine 靠 sender_id 判断
	a.selectConversation(0)
	waitFor(t, "历史消息加载完", func() bool {
		return a.readBool(func() bool { return len(a.msgs) >= 2 && !a.loadingMsgs })
	})
	a.read(func() {
		if a.convs[0].Unread != 0 {
			t.Error("选中后未读没清零")
		}
		// 列表里可能混有跨天时间胶囊和「以下为新消息」分隔线，
		// 按服务端 id 找真消息，别写死下标。
		byKey := make(map[string]message)
		for _, m := range a.msgs {
			byKey[m.Key] = m
		}
		first, second := byKey["m-1"], byKey["m-2"]
		if first.Mine || !second.Mine {
			t.Errorf("自己/对方判断错: %+v", a.msgs)
		}
		if first.Sender != "张三" {
			t.Errorf("发送者昵称错: %q", first.Sender)
		}
	})
	waitFor(t, "已读请求", func() bool { return f.reads.Load() == 1 })

	// 4. 发送：乐观上屏 → REST → 换上服务端 id
	a.sendText("  你好，原生客户端  ")
	waitFor(t, "发送请求", func() bool { return f.sends.Load() == 1 })
	if got, _ := f.lastMsg.Load().(string); got != "你好，原生客户端" {
		t.Errorf("发送内容没 trim: %q", got)
	}
	waitFor(t, "发送回执落到界面", func() bool {
		return a.readBool(func() bool {
			return len(a.msgs) >= 3 && !a.msgs[len(a.msgs)-1].Pending
		})
	})
	a.read(func() {
		last := a.msgs[len(a.msgs)-1]
		if last.Failed || last.Key != "m-3" {
			t.Errorf("发送状态/去重 key 错: %+v", last)
		}
	})

	// 5. WS：连接状态 + 推送追加到当前会话
	a.startRealtime()
	t.Cleanup(func() {
		if a.rt != nil {
			a.rt.Stop()
		}
	})
	waitFor(t, "WS 认证成功", func() bool {
		return a.readBool(func() bool { return a.connStatus == qapi.StatusConnected })
	})
	waitFor(t, "推送消息上屏", func() bool {
		return a.readBool(func() bool {
			return len(a.msgs) >= 4 && a.msgs[len(a.msgs)-1].Text == "这是服务端推来的新消息"
		})
	})
	a.read(func() {
		if a.msgs[len(a.msgs)-1].Mine {
			t.Error("别人推来的消息不该标成自己发的")
		}
		if a.convs[0].Preview != "这是服务端推来的新消息" {
			t.Errorf("侧栏预览没更新: %q", a.convs[0].Preview)
		}
	})

	// 6. 退出：清凭据、清界面数据
	a.logout()
	a.read(func() {
		if a.loggedIn || len(a.convs) != 0 || len(a.msgs) != 0 || a.active != -1 {
			t.Errorf("退出后状态没清干净: %+v", a)
		}
	})
	if tok, _ := a.client.Session(); tok != "" {
		t.Errorf("退出后 token 还在: %q", tok)
	}
}

// TestLoginTwoFactorStep 验证 2FA 不是当成登录失败，而是切到验证码步骤。
func TestLoginTwoFactorStep(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    1002,
			"message": "需要二次验证",
			"detail":  map[string]any{"two_factor_required": true, "session": "sess_9"},
		})
	}))
	defer srv.Close()

	a := newTestApp(srv.URL)
	ok, err := a.login(a.ctx, srv.URL, "grace", "pw", "")
	if err != nil || ok {
		t.Fatalf("2FA 不该算失败也不该算登录成功: ok=%v err=%v", ok, err)
	}
	if !a.need2FA || a.twoFASession != "sess_9" || a.loginBusy {
		t.Fatalf("没切到验证码步骤: %+v", a)
	}

	tt := ui.NewTester(a.view, 900, 700)
	tt.Frame()
	if !tt.HasText("二次验证") {
		t.Errorf("登录页没切出验证码输入框, texts %q", tt.Texts())
	}
}
