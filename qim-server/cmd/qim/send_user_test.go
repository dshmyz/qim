package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// 用户身份发送链路：--to 用户名 → /users/search 精确匹配 → 建/复用单聊 → 发消息。
// mock 三个用户 API 端点，且严格复刻服务端真实契约：
// CreateSingleConversation 只认 user_id（recipient_id 是绑定结构体里的死别名，
// 服务端不读、缺 user_id 直接 400"缺少必要参数"）——mock 不放水，防止契约再次漂移。
func TestSendAsUser_UsernameResolveAndSend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/users/search" && r.URL.Query().Get("q") == "alice":
			json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": []map[string]any{
				{"id": 7, "username": "alice", "name": "Alice"},
			}})
		case r.URL.Path == "/api/v1/conversations" && r.Method == http.MethodPost:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["type"] != "single" {
				t.Errorf("expected type=single, got %v", body["type"])
			}
			// 与 handler/conversation_handler.go 的 CreateSingleConversation 一致：
			// 缺 user_id → 400"缺少必要参数"
			uid, _ := body["user_id"].(float64)
			if uid != 7 {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]any{"code": 1001, "message": "缺少必要参数"})
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"id": 3}})
		case r.URL.Path == "/api/v1/conversations/3/messages" && r.Method == http.MethodPost:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["type"] != "markdown" {
				t.Errorf("expected type=markdown, got %v", body["type"])
			}
			if body["content"] != "hi" {
				t.Errorf("expected content=hi, got %v", body["content"])
			}
			json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"id": 99}})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	writeTestConfig(t, srv.URL, "qbot_x")

	if got := resolveUserConversationID("", "alice"); got != 3 {
		t.Fatalf("expected conversation id 3, got %d", got)
	}
	id, err := sendMessageAsUser(3, "hi", "markdown", 0)
	if err != nil {
		t.Fatalf("sendMessageAsUser: %v", err)
	}
	if id != 99 {
		t.Fatalf("expected message id 99, got %d", id)
	}
}

func TestSendAsUser_NumericConversationDirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 不允许触发搜索/建会话：直接传会话 ID 时只应发消息
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	writeTestConfig(t, srv.URL, "qbot_x")

	if got := resolveUserConversationID("42", ""); got != 42 {
		t.Fatalf("expected conversation id 42, got %d", got)
	}
}
