package qapi

// P1 契约测试：历史消息游标分页、三路搜索接口的参数与返回形状、
// 发起单聊的请求体和两种 id 命名。这些地方后端和前端约定不一致，
// 靠测试锁住，免得以后改客户端时又凭印象猜。

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func p1Reply(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "message": "ok", "data": data})
}

func TestMessagesBeforeUsesCursor(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RequestURI()
		p1Reply(w, map[string]any{
			"messages": []any{
				map[string]any{"id": 12, "conversation_id": 101, "type": "text", "content": "更早的一条"},
			},
			"pagination": map[string]any{"current_page": 1, "total_pages": 3, "total": 120, "page_size": 50},
		})
	}))
	defer srv.Close()

	c := New(srv.URL)
	page, err := c.MessagesBefore(context.Background(), ID("101"), "42", 50)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/v1/conversations/101/messages?before_id=42&page_size=50" {
		t.Fatalf("游标参数不对: %s", gotPath)
	}
	if len(page.Messages) != 1 || page.Messages[0].Content != "更早的一条" {
		t.Fatalf("消息没解出来: %+v", page.Messages)
	}
	if page.Pagination.Total != 120 || page.Pagination.TotalPages != 3 {
		t.Fatalf("pagination 没解出来: %+v", page.Pagination)
	}
}

func TestSearchEndpoints(t *testing.T) {
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.RequestURI())
		switch r.URL.Path {
		case "/api/v1/conversations/search":
			p1Reply(w, []any{map[string]any{"id": 9, "type": "group", "name": "技术交流群"}})
		case "/api/v1/users/search":
			p1Reply(w, []any{map[string]any{"id": 8, "type": "user", "name": "张三", "username": "zhangsan"}})
		case "/api/v1/messages/search":
			p1Reply(w, map[string]any{
				"list": []any{map[string]any{
					"id": 77, "conversation_id": 9, "sender_id": 8, "type": "text", "content": "构建挂了",
					"created_at":        "2026-11-14T09:31:00+08:00",
					"conversation_meta": map[string]any{"name": "技术交流群", "type": "group"},
				}},
				"total": 1, "page": 1,
			})
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	c := New(srv.URL)
	convs, err := c.SearchConversations(context.Background(), "技术 群")
	if err != nil || len(convs) != 1 || convs[0].ID.String() != "9" {
		t.Fatalf("会话搜索: %+v %v", convs, err)
	}
	users, err := c.SearchUsers(context.Background(), "张")
	if err != nil || len(users) != 1 || users[0].Label() != "张三" {
		t.Fatalf("用户搜索: %+v %v", users, err)
	}
	hits, total, err := c.SearchMessages(context.Background(), "构建", "", 1, 20)
	if err != nil || total != 1 || len(hits) != 1 {
		t.Fatalf("消息搜索: %+v total=%d %v", hits, total, err)
	}
	if hits[0].Message.Content != "构建挂了" || hits[0].Message.Timestamp == 0 {
		t.Errorf("消息本体没归一化: %+v", hits[0].Message)
	}
	if hits[0].ConvID != "9" || hits[0].ConvName != "技术交流群" {
		t.Errorf("conversation_meta 没解出来: %+v", hits[0])
	}

	// 参数名是后端实测的：会话用 query、用户用 q+scope、消息用 keyword+pageSize（驼峰）
	want := []string{
		"/api/v1/conversations/search?query=%E6%8A%80%E6%9C%AF+%E7%BE%A4",
		"/api/v1/users/search?q=%E5%BC%A0&scope=users",
		"/api/v1/messages/search?keyword=%E6%9E%84%E5%BB%BA&page=1&pageSize=20",
	}
	for i, w := range want {
		if i >= len(calls) || calls[i] != w {
			t.Errorf("第 %d 个请求 = %q，期望 %q（全部: %v）", i+1, at(calls, i), w, calls)
		}
	}
}

func at(s []string, i int) string {
	if i < len(s) {
		return s[i]
	}
	return ""
}

func TestSearchMessageBadFrameSkipped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		p1Reply(w, map[string]any{
			"list": []any{
				map[string]any{"id": 1, "content": "好的"},
				"这不是对象",
			},
			"total": 2,
		})
	}))
	defer srv.Close()
	c := New(srv.URL)
	hits, total, err := c.SearchMessages(context.Background(), "好", "", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || total != 2 {
		t.Fatalf("坏帧应该跳过而不是毁掉整页: hits=%d total=%d", len(hits), total)
	}
}

func TestCreateSingleConversation(t *testing.T) {
	for _, tc := range []struct {
		name string
		data any
		id   string
	}{
		{"id 命名", map[string]any{"id": 106}, "106"},
		{"conversation_id 命名", map[string]any{"conversation_id": 107}, "107"},
		{"只回 code:0", map[string]any{}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body map[string]any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(raw, &body)
				if r.Method != http.MethodPost || r.URL.Path != "/api/v1/conversations" {
					w.WriteHeader(404)
					return
				}
				p1Reply(w, tc.data)
			}))
			defer srv.Close()
			c := New(srv.URL)
			id, err := c.CreateSingleConversation(context.Background(), "8")
			if err != nil {
				t.Fatal(err)
			}
			if id != tc.id {
				t.Errorf("会话 id = %q，期望 %q", id, tc.id)
			}
			if body["type"] != "single" || body["recipient_id"] != float64(8) {
				t.Errorf("请求体不对: %+v", body)
			}
		})
	}

	c := New("http://127.0.0.1:1")
	if _, err := c.CreateSingleConversation(context.Background(), "不是数字"); err == nil {
		t.Error("非数字用户 id 应该直接报错，不该发请求")
	}
}
