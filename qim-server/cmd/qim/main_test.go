package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockBotServer 起一个 mock Bot API 后端，记录收到的请求，返回固定载荷。
type mockBotServer struct {
	t           *testing.T
	lastURL     string
	lastAuth    string
	lastBody    string
	sendRespID  int64
	streamCalls int
}

func newMockBotServer(t *testing.T) (*httptest.Server, *mockBotServer) {
	t.Helper()
	m := &mockBotServer{t: t, sendRespID: 42}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.lastURL = r.URL.String()
		m.lastAuth = r.Header.Get("Authorization")
		if r.Body != nil {
			b, _ := io.ReadAll(r.Body)
			m.lastBody = string(b)
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/bot/messages" && r.Method == http.MethodGet:
			// messages list：返回一条用户消息 + 一条 bot 消息
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"messages": []message{
				{ID: 10, ConversationID: 1, SenderID: 2, SenderType: "user", SenderNickname: "U", Content: "hi", Type: "text", Origin: "user", CreatedAt: "2026-07-24T10:00:00Z"},
				{ID: 11, ConversationID: 1, SenderID: 3, SenderType: "bot", SenderNickname: "B", Content: "reply", Type: "markdown", Origin: "bot", CreatedAt: "2026-07-24T10:00:01Z"},
			}}})
		case r.URL.Path == "/api/v1/bot/messages" && r.Method == http.MethodPost:
			m.streamCalls = 0
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"message_id": m.sendRespID, "conversation_id": 1}})
		case strings.HasPrefix(r.URL.Path, "/api/v1/bot/messages/") && strings.HasSuffix(r.URL.Path, "/stream"):
			m.streamCalls++
			json.NewEncoder(w).Encode(map[string]any{"data": nil, "message": "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, m
}

// writeTestConfig 写一个临时配置指向 mock server，返回清理函数。
func writeTestConfig(t *testing.T, serverURL, token string) {
	t.Helper()
	dir := t.TempDir()
	orig := configPath()
	// 通过覆盖 configPath 不易（包级函数），改用环境变量驱动：configPath 用 UserHomeDir。
	// 这里直接把 HOME 指向临时目录。
	t.Setenv("HOME", dir)
	cfg := config{ServerURL: serverURL, BotToken: token}
	b, _ := json.MarshalIndent(cfg, "", "  ")
	require_NoErr(t, os.MkdirAll(filepath.Join(dir, configDir), 0o700))
	require_NoErr(t, os.WriteFile(filepath.Join(dir, configDir, "config.json"), b, 0o600))
	_ = orig
}

func require_NoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMessagesList_ParsesAndSendsAuth(t *testing.T) {
	srv, m := newMockBotServer(t)
	writeTestConfig(t, srv.URL, "qbot_test_token")

	msgs, err := fetchMessages(1, 0, 50)
	require_NoErr(t, err)
	if len(msgs) != 2 {
		t.Fatalf("want 2 messages, got %d", len(msgs))
	}
	if msgs[0].Content != "hi" || msgs[0].SenderType != "user" {
		t.Fatalf("unexpected first msg: %+v", msgs[0])
	}
	if msgs[1].SenderType != "bot" {
		t.Fatalf("want bot sender, got %s", msgs[1].SenderType)
	}
	// 鉴权头 + 查询参数
	if m.lastAuth != "Bearer qbot_test_token" {
		t.Fatalf("want Bearer token, got %q", m.lastAuth)
	}
	if !strings.Contains(m.lastURL, "thread_id=1") || !strings.Contains(m.lastURL, "limit=50") {
		t.Fatalf("unexpected url: %s", m.lastURL)
	}
	// after_id 增量
	_, err = fetchMessages(1, 10, 50)
	require_NoErr(t, err)
	if !strings.Contains(m.lastURL, "after_id=10") {
		t.Fatalf("want after_id=10 in url: %s", m.lastURL)
	}
}

func TestSend_ReturnsMessageID(t *testing.T) {
	srv, m := newMockBotServer(t)
	writeTestConfig(t, srv.URL, "qbot_test_token")

	id, err := sendMessage(2, 1, "hello", "markdown", 0)
	require_NoErr(t, err)
	if id != 42 {
		t.Fatalf("want message_id 42, got %d", id)
	}
	// 请求体含字段
	var body map[string]any
	require_NoErr(t, json.Unmarshal([]byte(m.lastBody), &body))
	if body["to_user_id"] != float64(2) || body["msg_type"] != "markdown" || body["content"] != "hello" {
		t.Fatalf("unexpected body: %s", m.lastBody)
	}
}

func TestStreamChunk_SendsDeltaAndFinish(t *testing.T) {
	srv, m := newMockBotServer(t)
	writeTestConfig(t, srv.URL, "qbot_test_token")

	require_NoErr(t, streamChunk(42, "chunk1", false))
	var body map[string]any
	require_NoErr(t, json.Unmarshal([]byte(m.lastBody), &body))
	if body["content_delta"] != "chunk1" || body["finish"] != false {
		t.Fatalf("unexpected stream body: %s", m.lastBody)
	}
	// finish
	require_NoErr(t, streamChunk(42, "", true))
	require_NoErr(t, json.Unmarshal([]byte(m.lastBody), &body))
	if body["finish"] != true {
		t.Fatalf("want finish=true: %s", m.lastBody)
	}
}

func TestStreamStdin_PipesDeltasThenFinish(t *testing.T) {
	srv, m := newMockBotServer(t)
	writeTestConfig(t, srv.URL, "qbot_test_token")

	// 验证 stream-stdin 的核心契约：建流式消息（POST /messages），
	// 随后每个 delta 一次 stream 调用，EOF 一次 finish。
	// stdin 的逐行读取由 bufio.Scanner 承担（标准库），此处直接验证 streamChunk 计数。
	msgID, err := sendMessage(2, 1, "", "streaming", 0)
	require_NoErr(t, err)
	if msgID != 42 {
		t.Fatalf("want 42, got %d", msgID)
	}
	for _, delta := range []string{"line1\n", "line2\n"} {
		require_NoErr(t, streamChunk(msgID, delta, false))
	}
	require_NoErr(t, streamChunk(msgID, "", true))
	if m.streamCalls != 3 { // 2 delta + 1 finish
		t.Fatalf("want 3 stream calls, got %d", m.streamCalls)
	}
}
