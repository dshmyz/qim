package qapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// reply 写一个标准的 {code,message,data} 响应体。
func reply(w http.ResponseWriter, code int, message string, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code": code, "message": message, "data": data,
	})
}

func TestLoginOK(t *testing.T) {
	var gotBody map[string]any
	var gotVer, gotPlat string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login" {
			t.Errorf("路径不对: %s", r.URL.Path)
		}
		gotVer = r.Header.Get("X-App-Version")
		gotPlat = r.Header.Get("X-App-Platform")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		reply(w, 0, "ok", map[string]any{
			"token":         "tkn_1",
			"refresh_token": "rtk_1",
			"user":          map[string]any{"id": 7, "username": "grace", "avatar": "a.png"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL)
	s, err := c.Login(context.Background(), "grace", "pw123")
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	if s.Token != "tkn_1" || s.RefreshToken != "rtk_1" {
		t.Fatalf("凭据解析错: %+v", s)
	}
	if s.User.ID != "7" {
		t.Fatalf("数字 id 没转成字符串: %q", s.User.ID)
	}
	if s.User.DisplayName() != "grace" {
		t.Fatalf("显示名错: %q", s.User.DisplayName())
	}
	if gotBody["username"] != "grace" || gotBody["version"] == nil {
		t.Fatalf("请求体不对: %+v", gotBody)
	}
	if gotVer == "" || gotPlat == "" {
		t.Fatalf("缺版本/平台头: ver=%q plat=%q", gotVer, gotPlat)
	}
	if tok, _ := c.Session(); tok != "tkn_1" {
		t.Fatalf("token 没写进 client: %q", tok)
	}
}

func TestLoginTwoFactor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    1002,
			"message": "需要二次验证",
			"detail":  map[string]any{"two_factor_required": true, "session": "sess_9"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL)
	_, err := c.Login(context.Background(), "grace", "pw")
	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatalf("应为 APIError，实际 %T %v", err, err)
	}
	if !ae.IsTwoFactorRequired() {
		t.Fatal("没识别出二次验证")
	}
	if ae.TwoFactorSession() != "sess_9" {
		t.Fatalf("会话码错: %q", ae.TwoFactorSession())
	}
}

func TestLoginBadPassword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reply(w, 1001, "用户名或密码错误", nil)
	}))
	defer srv.Close()

	c := New(srv.URL)
	_, err := c.Login(context.Background(), "grace", "bad")
	var ae *APIError
	if !errors.As(err, &ae) || ae.Code != 1001 {
		t.Fatalf("错误码没透出: %v", err)
	}
	if ae.IsTwoFactorRequired() {
		t.Fatal("密码错不该判成二次验证")
	}
}

func TestConversationsMixedTypes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/conversations" {
			t.Errorf("路径不对: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tkn_x" {
			t.Errorf("Authorization 头不对: %q", got)
		}
		// 故意混用：id 数字、id 字符串、lastMessage camelCase、last_message snake_case
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"message":"ok","data":[
			{"id":101,"name":"技术交流群","type":"group","unread_count":3,"timestamp":1700000000,
			 "lastMessage":{"id":1,"type":"text","content":"hello","conversation_id":101}},
			{"id":"c2","name":"系统通知","type":"single","unread_count":0,"timestamp":1699999999,
			 "last_message":{"id":"m2","type":"image","content":"","conversation_id":"c2"}}
		]}`)
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.SetSession("tkn_x", "")

	convs, err := c.Conversations(context.Background())
	if err != nil {
		t.Fatalf("拉会话失败: %v", err)
	}
	if len(convs) != 2 {
		t.Fatalf("数量错: %d", len(convs))
	}
	if convs[0].ID != "101" || convs[0].UnreadCount != 3 {
		t.Fatalf("第一条解析错: %+v", convs[0])
	}
	if convs[0].Last() == nil || convs[0].Last().Preview() != "hello" {
		t.Fatalf("lastMessage 没解出来: %+v", convs[0].Last())
	}
	if convs[1].ID != "c2" || convs[1].Last() == nil || convs[1].Last().Preview() != "[图片]" {
		t.Fatalf("第二条解析错: %+v / %+v", convs[1], convs[1].Last())
	}
}

func TestMessagesPaging(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/conversations/101/messages") {
			t.Errorf("路径不对: %s", r.URL.Path)
		}
		if r.URL.Query().Get("page") != "2" || r.URL.Query().Get("page_size") != "50" {
			t.Errorf("分页参数不对: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"data":{"messages":[
			{"id":11,"conversation_id":101,"sender_id":7,"type":"text","content":"在吗","timestamp":1700000001,"isSelf":true},
			{"id":12,"conversation_id":101,"sender_id":8,"type":"text","content":"在","timestamp":1700000002}
		],"pagination":{"current_page":2,"total_pages":5,"total":230,"page_size":50}}}`)
	}))
	defer srv.Close()

	c := New(srv.URL)
	p, err := c.Messages(context.Background(), "101", 2, 50)
	if err != nil {
		t.Fatalf("拉消息失败: %v", err)
	}
	if len(p.Messages) != 2 || p.Pagination.Total != 230 {
		t.Fatalf("分页解析错: %+v", p)
	}
	if p.Messages[0].ConversationID != "101" || !p.Messages[0].IsSelf {
		t.Fatalf("消息字段错: %+v", p.Messages[0])
	}
}

func TestSendMessageBody(t *testing.T) {
	var got SendRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("方法不对: %s", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		reply(w, 0, "ok", map[string]any{"id": 99, "content": got.Content, "type": "text"})
	}))
	defer srv.Close()

	c := New(srv.URL)
	m, err := c.SendMessage(context.Background(), "101", TextRequest("你好"))
	if err != nil {
		t.Fatalf("发送失败: %v", err)
	}
	if got.Type != "text" || got.Content != "你好" {
		t.Fatalf("请求体错: %+v", got)
	}
	if m.ID != "99" {
		t.Fatalf("返回消息解析错: %+v", m)
	}
}

func TestRefreshOn401(t *testing.T) {
	var refreshed int
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/refresh":
			refreshed++
			reply(w, 0, "ok", map[string]any{"token": "tkn_new", "refresh_token": "rtk_new"})
		case "/api/v1/conversations":
			calls++
			// 第一次装作 token 过期，刷新之后才放行
			if r.Header.Get("Authorization") != "Bearer tkn_new" {
				reply(w, 401, "token 已过期", nil)
				return
			}
			reply(w, 0, "ok", []Conversation{})
		default:
			t.Errorf("意外路径: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.SetSession("tkn_old", "rtk_old")

	if _, err := c.Conversations(context.Background()); err != nil {
		t.Fatalf("自动刷新后仍失败: %v", err)
	}
	if refreshed != 1 {
		t.Fatalf("应刷新 1 次，实际 %d", refreshed)
	}
	if calls != 2 {
		t.Fatalf("应请求 2 次（401 + 重放），实际 %d", calls)
	}
	if tok, rt := c.Session(); tok != "tkn_new" || rt != "rtk_new" {
		t.Fatalf("新凭据没写回: %q %q", tok, rt)
	}
}

func TestRealtimeAuthAndPush(t *testing.T) {
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	gotAuth := make(chan string, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ws" {
			t.Errorf("WS 路径不对: %s", r.URL.Path)
			return
		}
		if r.URL.Query().Get("version") == "" || r.URL.Query().Get("platform") == "" {
			t.Errorf("缺 version/platform 查询参数: %s", r.URL.RawQuery)
		}
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("升级失败: %v", err)
			return
		}
		defer conn.Close()

		// 首帧必须是 auth，且带 token
		var first struct {
			Type string `json:"type"`
			Data struct {
				Token string `json:"token"`
			} `json:"data"`
		}
		if err := conn.ReadJSON(&first); err != nil {
			t.Errorf("读首帧失败: %v", err)
			return
		}
		gotAuth <- first.Type + ":" + first.Data.Token
		if first.Type != "auth" {
			t.Errorf("首帧不是 auth: %+v", first)
			return
		}

		_ = conn.WriteJSON(map[string]any{"type": "auth_success", "data": map[string]any{"user_id": 7}})
		_ = conn.WriteJSON(map[string]any{
			"type": "new_message",
			"data": map[string]any{"id": 555, "conversation_id": 101, "content": "服务端推来的"},
		})
		// 留一点时间让客户端读完再断
		time.Sleep(600 * time.Millisecond)
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.SetSession("tkn_ws", "")

	rt := NewRealtime(c)
	defer rt.Stop()

	msgs := make(chan Message, 1)
	connected := make(chan Status, 4)
	rt.On("new_message", func(e Event) {
		var m Message
		if err := e.Raw(&m); err != nil {
			t.Errorf("解析推送失败: %v", err)
			return
		}
		msgs <- m
	})
	rt.OnStatus(func(s Status) { connected <- s })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rt.Start(ctx)

	select {
	case a := <-gotAuth:
		if a != "auth:tkn_ws" {
			t.Fatalf("认证帧不对: %q", a)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("服务端没收到认证帧")
	}

	select {
	case m := <-msgs:
		if m.ID != "555" || m.ConversationID != "101" || m.Content != "服务端推来的" {
			t.Fatalf("推送内容错: %+v", m)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("没收到推送消息")
	}

	select {
	case s := <-connected:
		if s != StatusConnecting && s != StatusConnected {
			t.Fatalf("状态回调异常: %v", s)
		}
	case <-time.After(time.Second):
		t.Fatal("没有状态回调")
	}
}

func TestRealtimeNeedsLogin(t *testing.T) {
	c := New("http://127.0.0.1:1")
	rt := NewRealtime(c)
	defer rt.Stop()
	if rt.Status() != StatusDisconnected {
		t.Fatalf("初始状态应为未连接: %v", rt.Status())
	}
	if err := rt.Send("ping", nil); err == nil {
		t.Fatal("未连接时 Send 应报错")
	}
}

// 真实 qim-server 的线格式：data.list 包一层、时间是 RFC3339 字符串、
// 单聊名字在 other_member_name、置顶叫 is_pinned。
func TestConversationsRealWireShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"data":{"list":[
			{"id":7,"type":"single","other_member_name":"张三","unread_count":2,
			 "last_message_at":"2026-11-14T10:00:00+08:00","is_pinned":true,
			 "last_message":{"id":31,"content":"在吗","created_at":"2026-11-14T10:00:00+08:00"}}
		],"total":1,"page":1,"page_size":20,"has_more":false}}`)
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.SetSession("tkn", "")
	convs, err := c.Conversations(context.Background())
	if err != nil {
		t.Fatalf("真实结构解析失败: %v", err)
	}
	if len(convs) != 1 {
		t.Fatalf("应解出 1 个会话，实际 %d", len(convs))
	}
	cv := convs[0]
	if cv.ID != "7" || cv.DisplayName() != "张三" || cv.UnreadCount != 2 {
		t.Fatalf("会话字段错: %+v", cv)
	}
	if !cv.PinnedNow() {
		t.Error("is_pinned 没被识别")
	}
	if cv.Timestamp == 0 || cv.Last() == nil || cv.Last().Preview() != "在吗" {
		t.Fatalf("时间/最后消息没解出来: %+v / %+v", cv, cv.Last())
	}
	if got := time.UnixMilli(cv.Timestamp).Format("2006-01-02"); got != "2026-11-14" {
		t.Errorf("last_message_at 解析错: %s", got)
	}
}

// 后端消息不下发 isSelf / timestamp，靠 sender_id 与 created_at 归一化。
func TestMessageRealWireShape(t *testing.T) {
	var m Message
	if err := json.Unmarshal([]byte(`{
		"id":31,"conversation_id":7,"sender_id":7,"type":"text","content":"在吗",
		"created_at":"2026-11-14T10:00:00+08:00","is_read":true,"is_recalled":false,
		"sender":{"id":7,"nickname":"张三","username":"zhangsan","avatar":"a.png"}
	}`), &m); err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if m.SenderName() != "张三" {
		t.Errorf("发送者名字错: %q", m.SenderName())
	}
	if !m.IsMine("7") || m.IsMine("8") {
		t.Error("isSelf 判定错")
	}
	if !m.IsRead || m.IsRecalled {
		t.Error("snake_case 状态位没归一化")
	}
	if time.UnixMilli(m.Timestamp).Format("15:04") != "10:00" {
		t.Errorf("created_at 解析错: %d", m.Timestamp)
	}
}

func TestClientDefaultHeaders(t *testing.T) {
	c := New("http://127.0.0.1:1")
	if c.Version != "2.0.39" {
		t.Errorf("版本号应与 Electron 客户端一致，实际 %q", c.Version)
	}
	switch runtime.GOOS {
	case "darwin":
		if c.Platform != "macos" {
			t.Errorf("darwin 应上报 macos，实际 %q", c.Platform)
		}
	case "linux", "windows":
		if c.Platform != runtime.GOOS {
			t.Errorf("平台上报错: %q", c.Platform)
		}
	}
}
