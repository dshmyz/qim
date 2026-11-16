package qapi

// P0 契约测试：退避曲线、错误码语义、认证路径不重放、增量消息、
// 登出 token 快照、refresh 轮换后的回调。

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNextBackoff(t *testing.T) {
	want := []time.Duration{
		2 * time.Second, 4 * time.Second, 8 * time.Second,
		16 * time.Second, 30 * time.Second, 30 * time.Second,
	}
	cur := time.Second
	for i, w := range want {
		cur = nextBackoff(cur)
		if cur != w {
			t.Fatalf("第 %d 次退避 = %v，期望 %v", i+1, cur, w)
		}
	}
}

func TestIsUnauthorized(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"1001 参数错误/密码错", &APIError{Code: 1001, Message: "密码错误"}, false},
		{"1002 二次验证", &APIError{Code: 1002, Detail: json.RawMessage(`{"two_factor_required":true,"session":"s"}`)}, false},
		{"1002 未授权", &APIError{Code: 1002, Message: "未授权"}, true},
		{"HTTP 401", &APIError{Code: 401}, true},
		{"非业务错误", errors.New("connection refused"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isUnauthorized(tc.err); got != tc.want {
				t.Fatalf("isUnauthorized = %v，期望 %v", got, tc.want)
			}
		})
	}
}

// 登录失败即便返回 1002 也不能触发刷新：否则上一次登录残留的
// refresh_token 会被拿去换新 token，把旧会话带进新账号的登录流程。
func TestAuthEndpointsNeverRefresh(t *testing.T) {
	var refreshCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/refresh" {
			refreshCalls.Add(1)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 1002, "message": "需要二次验证",
			"detail": map[string]any{"two_factor_required": true, "session": "sess_9"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.SetSession("old-token", "old-refresh")
	_, err := c.Login(context.Background(), "grace", "pw")
	var ae *APIError
	if !errors.As(err, &ae) || !ae.IsTwoFactorRequired() {
		t.Fatalf("应识别成二次验证: %v", err)
	}
	if refreshCalls.Load() != 0 {
		t.Fatalf("登录失败触发了 %d 次 token 刷新", refreshCalls.Load())
	}
}

// 真正的业务码 1002 未授权要能触发刷新重放；之前只认 1001，
// token 过期时客户端会直接卡死而不是自动续期。
func TestUnauthorized1002RefreshesAndReplays(t *testing.T) {
	var refreshCalls, convCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/refresh":
			refreshCalls.Add(1)
			reply(w, 0, "ok", map[string]any{"token": "tkn_new", "refresh_token": "rtk_new"})
		case "/api/v1/conversations":
			if convCalls.Add(1) == 1 {
				reply(w, 1002, "未授权", nil)
				return
			}
			reply(w, 0, "ok", map[string]any{"list": []any{
				map[string]any{"id": 1, "name": "技术交流群"},
			}})
		default:
			t.Errorf("意外路径 %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.SetSession("tkn_old", "rtk_old")
	convs, err := c.Conversations(context.Background())
	if err != nil {
		t.Fatalf("刷新后重放失败: %v", err)
	}
	if len(convs) != 1 || convs[0].DisplayName() != "技术交流群" {
		t.Fatalf("重放结果不对: %+v", convs)
	}
	if refreshCalls.Load() != 1 || convCalls.Load() != 2 {
		t.Fatalf("刷新 %d 次 / 会话请求 %d 次，期望 1 / 2", refreshCalls.Load(), convCalls.Load())
	}
	tok, rt := c.Session()
	if tok != "tkn_new" || rt != "rtk_new" {
		t.Fatalf("新凭据没写回 client: %q / %q", tok, rt)
	}
}

func TestMessagesAfterQuery(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		reply(w, 0, "ok", map[string]any{"messages": []any{}})
	}))
	defer srv.Close()

	c := New(srv.URL)
	if _, err := c.MessagesAfter(context.Background(), ID("101"), "900", 0); err != nil {
		t.Fatalf("增量拉取失败: %v", err)
	}
	if !strings.Contains(gotQuery, "after_id=900") || !strings.Contains(gotQuery, "page_size=50") {
		t.Fatalf("查询参数不对: %q", gotQuery)
	}
}

// 退出登录会立刻清空本地凭据，登出请求必须用调用方传进来的 token 快照。
func TestLogoutUsesSnapshottedToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/logout" || r.Method != http.MethodPost {
			t.Errorf("登出路径/方法不对: %s %s", r.Method, r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		reply(w, 0, "ok", map[string]any{})
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.SetSession("tkn_live", "rtk_live")
	token, _ := c.Session()
	c.SetSession("", "") // 模拟界面层已经清掉凭据
	if err := c.Logout(context.Background(), token); err != nil {
		t.Fatalf("登出失败: %v", err)
	}
	if gotAuth != "Bearer tkn_live" {
		t.Fatalf("登出没带上 token 快照: %q", gotAuth)
	}
}

// refresh 会轮换 refresh_token，回调必须拿到新值，持久化才不会写旧凭据。
func TestSessionChangeCallbackOnRefresh(t *testing.T) {
	type pair struct{ token, refresh string }
	changes := make(chan pair, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/refresh":
			reply(w, 0, "ok", map[string]any{"token": "tkn_new", "refresh_token": "rtk_new"})
		case "/api/v1/conversations":
			reply(w, 1002, "未授权", nil)
		default:
			t.Errorf("意外路径 %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.SetSession("tkn_old", "rtk_old")
	c.OnSessionChange(func(token, refresh string) {
		changes <- pair{token, refresh}
	})
	_, _ = c.Conversations(context.Background())

	select {
	case p := <-changes:
		if p.token != "tkn_new" || p.refresh != "rtk_new" {
			t.Fatalf("回调拿到旧凭据: %+v", p)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("refresh 后没有触发凭据变化回调")
	}
}
