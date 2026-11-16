package qapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAIStreamSidebar(t *testing.T) {
	var gotReq AISidebarRequest
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ai/sidebar/stream" {
			t.Errorf("路径不对: %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Errorf("请求体不是 JSON: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		// 每帧标准 SSE：data: <json>\n\n；中间夹一行注释不能被误解析。
		for _, frame := range []string{
			`{"content":"你"}`,
			`{"content":"好"}`,
			": ping",
			`{"content":"！"}`,
			`{"finish":"stop"}`,
		} {
			w.Write([]byte("data: " + frame + "\n\n"))
		}
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.SetSession("tok-ai", "")
	var pieces []string
	var finished bool
	err := c.AIStreamSidebar(context.Background(), AISidebarRequest{
		Message: "你好", Scope: "current", ConversationID: 42,
	}, func(ch AIStreamChunk) {
		pieces = append(pieces, ch.Content)
		if ch.Finish != nil {
			finished = true
		}
	})
	if err != nil {
		t.Fatalf("流失败: %v", err)
	}
	if got := strings.Join(pieces, ""); got != "你好！" {
		t.Errorf("拼接内容 = %q, 想要 你好！", got)
	}
	if !finished {
		t.Error("没收到 finish 帧")
	}
	if gotAuth != "Bearer tok-ai" {
		t.Errorf("认证头 = %q", gotAuth)
	}
	if gotReq.Message != "你好" || gotReq.Scope != "current" || gotReq.ConversationID != 42 {
		t.Errorf("请求体 = %+v", gotReq)
	}
}

func TestAIStreamSidebarErrorFrame(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("data: {\"content\":\"前半段\"}\n\n"))
		w.Write([]byte("data: {\"error\":\"AI请求失败：配额不足\"}\n\n"))
		// 错误帧后不应再有内容；即便有，客户端也必须已停止。
		w.Write([]byte("data: {\"content\":\"不该收到\"}\n\n"))
	}))
	defer srv.Close()

	var pieces []string
	var aiErr string
	err := New(srv.URL).AIStreamSidebar(context.Background(), AISidebarRequest{Message: "x"},
		func(ch AIStreamChunk) {
			pieces = append(pieces, ch.Content)
			if ch.Error != nil {
				aiErr = *ch.Error
			}
		})
	if err != nil {
		t.Fatalf("错误帧不算传输失败: %v", err)
	}
	if aiErr != "AI请求失败：配额不足" {
		t.Errorf("错误文案 = %q", aiErr)
	}
	if strings.Join(pieces, "") != "前半段" {
		t.Errorf("错误帧后还在收内容: %v", pieces)
	}
}

func TestAIStreamSidebarDisabled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"code":1003,"message":"AI 功能已关闭"}`))
	}))
	defer srv.Close()

	err := New(srv.URL).AIStreamSidebar(context.Background(), AISidebarRequest{Message: "x"}, nil)
	e, ok := err.(*APIError)
	if !ok || e.Code != 1003 || e.Message != "AI 功能已关闭" {
		t.Fatalf("错误没透传: %v", err)
	}
}

func TestAISuggestedPrompts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ai/suggested-prompts" {
			t.Errorf("路径不对: %s", r.URL.Path)
		}
		w.Write([]byte(`{"code":0,"message":"success","data":{"prompts":["总结讨论","查看待办"]}}`))
	}))
	defer srv.Close()

	got, err := New(srv.URL).AISuggestedPrompts(context.Background())
	if err != nil {
		t.Fatalf("拉提示失败: %v", err)
	}
	if len(got) != 2 || got[0] != "总结讨论" || got[1] != "查看待办" {
		t.Errorf("提示 = %v", got)
	}
}
