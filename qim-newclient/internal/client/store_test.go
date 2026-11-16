package client

// P0 持久化测试：钥匙串读写的内存替身、启动免登录、失效清理、
// 网络故障保留凭据、退出时通知服务端并清除本地凭据。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func newStoredApp(t *testing.T, url string, s savedSession) (*app, *memStore) {
	t.Helper()
	a := newTestApp(url)
	st := &memStore{}
	if err := st.Save(s); err != nil {
		t.Fatal(err)
	}
	a.store = st
	a.client.OnSessionChange(a.onSessionChange)
	return a, st
}

func jsonReply(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": code, "data": data})
}

func TestMemStoreRoundTrip(t *testing.T) {
	st := &memStore{}
	if _, err := st.Load(); err == nil {
		t.Fatal("空 store 应返回未找到")
	}
	s := savedSession{ServerURL: "http://127.0.0.1:8080", Token: "t", RefreshToken: "r", UserID: "7"}
	if err := st.Save(s); err != nil {
		t.Fatal(err)
	}
	got, err := st.Load()
	if err != nil || got.Token != "t" || got.ServerURL != s.ServerURL {
		t.Fatalf("读回不对: %+v %v", got, err)
	}
	if err := st.Clear(); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Load(); err == nil {
		t.Fatal("Clear 之后不该还能读到")
	}
}

func TestRestoreSessionValid(t *testing.T) {
	var convCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/conversations" {
			convCalls.Add(1)
			jsonReply(w, 0, map[string]any{"list": []any{}})
			return
		}
		jsonReply(w, 0, map[string]any{})
	}))
	defer srv.Close()

	s := savedSession{
		ServerURL: srv.URL, Token: "tkn_native", RefreshToken: "rtk_native",
		UserID: "7", UserName: "测试用户",
	}
	a, st := newStoredApp(t, srv.URL, s)
	a.restoreSession()
	waitFor(t, "恢复登录态", func() bool {
		return a.readBool(func() bool { return a.loggedIn })
	})
	a.read(func() {
		if a.userName != "测试用户" || a.selfID != "7" || a.loginServer != srv.URL {
			t.Errorf("恢复后的用户状态不对: %q / %q / %q", a.userName, a.selfID, a.loginServer)
		}
	})
	if tok, rt := a.client.Session(); tok != "tkn_native" || rt != "rtk_native" {
		t.Errorf("凭据没写进 client: %q / %q", tok, rt)
	}
	if _, set, _ := st.current(); !set {
		t.Error("有效凭据被误清")
	}
	if convCalls.Load() == 0 {
		t.Error("恢复时没有校验服务端")
	}
	a.logout()
}

func TestRestoreSessionAuthFailureClearsCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/refresh" {
			jsonReply(w, 1002, nil) // 刷新也失败 → 真的需要重新登录
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		jsonReply(w, 1002, nil)
	}))
	defer srv.Close()

	a, st := newStoredApp(t, srv.URL, savedSession{
		ServerURL: srv.URL, Token: "expired", RefreshToken: "expired_rt", UserID: "7",
	})
	a.restoreSession()
	waitFor(t, "失效凭据被清除", func() bool {
		_, set, _ := st.current()
		tok, _ := a.client.Session()
		return !set && tok == ""
	})
	if a.readBool(func() bool { return a.loggedIn }) {
		t.Error("凭据失效却进了已登录态")
	}
}

func TestRestoreSessionServerErrorKeepsCredentials(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		jsonReply(w, 5000, nil)
	}))
	defer srv.Close()

	a, st := newStoredApp(t, srv.URL, savedSession{
		ServerURL: srv.URL, Token: "tkn", RefreshToken: "rtk", UserID: "7",
	})
	a.restoreSession()
	waitFor(t, "校验请求已发出", func() bool { return calls.Load() >= 1 })
	time.Sleep(50 * time.Millisecond) // 等恢复协程处理完响应
	if _, set, _ := st.current(); !set {
		t.Error("服务端故障不该清掉登录态")
	}
	if a.readBool(func() bool { return a.loggedIn }) {
		t.Error("校验没通过却进了已登录态")
	}
	tok, _ := a.client.Session()
	if tok != "tkn" {
		t.Errorf("本地凭据被改动: %q", tok)
	}
}

func TestLogoutNotifiesServerAndClearsStore(t *testing.T) {
	var logoutCalls atomic.Int32
	var gotAuth atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/logout" {
			logoutCalls.Add(1)
			gotAuth.Store(r.Header.Get("Authorization"))
		}
		jsonReply(w, 0, map[string]any{})
	}))
	defer srv.Close()

	a, st := newStoredApp(t, srv.URL, savedSession{})
	a.client.SetSession("tkn_live", "rtk_live")
	a.rememberSession(srv.URL, "7", "测试用户")
	a.logout()
	waitFor(t, "服务端登出请求", func() bool { return logoutCalls.Load() == 1 })
	if got, _ := gotAuth.Load().(string); got != "Bearer tkn_live" {
		t.Errorf("登出没带 token 快照: %q", got)
	}
	if _, set, _ := st.current(); set {
		t.Error("退出后本地凭据没清")
	}
	if tok, _ := a.client.Session(); tok != "" {
		t.Errorf("退出后 token 还在: %q", tok)
	}
}
