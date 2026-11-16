package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

// newAIFakeServer 模拟 /ai/sidebar/stream 的 SSE 响应。
func newAIFakeServer(t *testing.T, pieces []string, errFrame string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ai/sidebar/stream" {
			t.Errorf("AI 请求路径 = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, p := range pieces {
			b, _ := json.Marshal(map[string]string{"content": p})
			w.Write([]byte("data: " + string(b) + "\n\n"))
		}
		if errFrame != "" {
			b, _ := json.Marshal(map[string]string{"error": errFrame})
			w.Write([]byte("data: " + string(b) + "\n\n"))
			return
		}
		w.Write([]byte("data: {\"finish\":\"stop\"}\n\n"))
	}))
}

func TestShotAISidebarEmpty(t *testing.T) {
	a := &app{
		loggedIn: true, userName: "我", selfID: "7", active: 0,
		convs:  []conversation{{ID: "101", Name: "技术交流群", Kind: "group"}},
		aiOpen: true,
	}
	tt := ui.NewTester(a.view, 1050, 900)
	tt.Frame()
	tt.Frame()
	for _, text := range []string{"AI 助手", "有什么可以帮你？", "总结讨论", "设置提醒", "搜索信息", "查看待办", "Enter 发送 · Shift+Enter 换行"} {
		if !tt.HasText(text) {
			t.Errorf("空态缺 %q; texts=%q", text, tt.Texts())
		}
	}
	write(t, tt, "/tmp/qim-ai-empty.png")
}

func TestShotAISidebarChat(t *testing.T) {
	a := &app{
		loggedIn: true, userName: "我", selfID: "7", active: 0,
		convs:  []conversation{{ID: "101", Name: "技术交流群", Kind: "group"}},
		aiOpen: true,
		aiTurns: []aiTurn{
			{Role: "user", Text: "帮我总结一下讨论要点"},
			{Role: "assistant", Text: "## 讨论要点\n\n- 项目进展回顾\n- 超时项评审\n\n**结论**：下午先整理一版材料。"},
		},
	}
	tt := ui.NewTester(a.view, 1050, 900)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("帮我总结一下讨论要点") || !tt.HasText("讨论要点") {
		t.Errorf("对话区内容不对: texts=%q", tt.Texts())
	}
	write(t, tt, "/tmp/qim-ai-chat.png")
}

func TestAISidebarToggleAndScope(t *testing.T) {
	a := &app{loggedIn: true, active: 0,
		convs: []conversation{{ID: "101", Name: "技术交流群"}}}
	if a.aiOpen {
		t.Fatal("初始不该打开")
	}
	a.aiToggle()
	if !a.aiOpen {
		t.Fatal("切换后应打开")
	}
	if got := a.aiContextName(); got != "技术交流群" {
		t.Errorf("上下文名 = %q", got)
	}
	a.aiToggleScope()
	if !a.aiScopeCross || a.aiContextName() != "" {
		t.Errorf("自由对话状态不对: cross=%v name=%q", a.aiScopeCross, a.aiContextName())
	}
	a.aiClose()
	if a.aiOpen {
		t.Error("关闭后不该还打开")
	}
}

// TestAISendStream 用假 SSE 服务端验证整条异步链路：
// 发消息 → 逐帧追加到助手气泡 → busy 复位 → 文本完整。
func TestAISendStream(t *testing.T) {
	srv := newAIFakeServer(t, []string{"讨论", "要点", "已完成"}, "")
	defer srv.Close()

	a := &app{loggedIn: true, active: 0, ctx: context.Background(),
		convs:  []conversation{{ID: "101", Name: "技术交流群"}},
		client: qapi.New(srv.URL)}
	a.aiSend("帮我总结")
	waitFor(t, "AI 流结束", func() bool {
		a.mu.RLock()
		defer a.mu.RUnlock()
		return !a.aiBusy
	})

	a.mu.RLock()
	defer a.mu.RUnlock()
	if len(a.aiTurns) != 2 {
		t.Fatalf("气泡数 = %d, 想要 2", len(a.aiTurns))
	}
	if a.aiTurns[0].Text != "帮我总结" {
		t.Errorf("用户气泡 = %q", a.aiTurns[0].Text)
	}
	if got := a.aiTurns[1].Text; got != "讨论要点已完成" {
		t.Errorf("助手气泡 = %q", got)
	}
	if a.aiTurns[1].Streaming {
		t.Error("结束后不该还在 streaming")
	}
}
