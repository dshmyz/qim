package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// 会话目标收敛：--conversation 支持数字 ID 或名称，--thread 只是弃用别名。

func TestResolveConversationID_Numeric(t *testing.T) {
	if got := resolveConversationID("42"); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
}

func TestResolveConversationID_ByName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/bot/conversations":
			json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": []map[string]any{
				{"conversation_id": 98, "conv_type": "group", "name": "产品群"},
				{"conversation_id": 7, "conv_type": "bot", "name": "爱丽丝"},
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	writeTestConfig(t, srv.URL, "qbot_x")

	if got := resolveConversationID("产品群"); got != 98 {
		t.Fatalf("expected group conversation 98, got %d", got)
	}
	if got := resolveConversationID("爱丽丝"); got != 7 {
		t.Fatalf("expected direct conversation 7, got %d", got)
	}
}

func TestResolveThreadArg_FallsBackToServerName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// 没有名为 alice 的 bot 会话：--thread 应保留原值，交给服务端 thread_name 解析。
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": []map[string]any{
			{"conversation_id": 98, "conv_type": "group", "name": "产品群"},
		}})
	}))
	t.Cleanup(srv.Close)
	writeTestConfig(t, srv.URL, "qbot_x")

	if got := resolveThreadArg("", "alice"); got != "alice" {
		t.Fatalf("expected fallback to server thread_name, got %q", got)
	}
	if got := resolveThreadArg("", "98"); got != "98" {
		t.Fatalf("expected numeric passthrough, got %q", got)
	}
}

func TestSendMessageToConversation_ConversationOnly(t *testing.T) {
	srv, m := newMockBotServer(t)
	writeTestConfig(t, srv.URL, "qbot_x")

	id, err := sendMessageToConversation(98, "hi", "markdown", 0)
	require_NoErr(t, err)
	if id != 42 {
		t.Fatalf("expected message id 42, got %d", id)
	}

	var body map[string]any
	require_NoErr(t, json.Unmarshal([]byte(m.lastBody), &body))
	if body["conversation_id"] != float64(98) {
		t.Fatalf("want conversation_id=98, body: %s", m.lastBody)
	}
	if _, ok := body["to_user_id"]; ok {
		t.Fatalf("conversation 发送不应带 to_user_id: %s", m.lastBody)
	}
	if _, ok := body["thread_id"]; ok {
		t.Fatalf("conversation 发送不应带 thread_id: %s", m.lastBody)
	}
}
