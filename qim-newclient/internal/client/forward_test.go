package client

// 转发 / 合并转发对齐 qim-client 的「转发」与多选「合并转发」：
// 右键转发单条、多选合并转发、目标会话选择浮层、复用 SendMessage 落到真实后端。

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

// TestForwardSendsToTarget 验证转发真的把内容 POST 到「目标会话」端点，
// 且内容原样透传（不走当前活跃会话，不改界面）。
func TestForwardSendsToTarget(t *testing.T) {
	var mu sync.Mutex
	var gotPath, gotBody string
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/messages") {
			b, _ := io.ReadAll(r.Body)
			mu.Lock()
			gotPath = r.URL.Path
			gotBody = string(b)
			mu.Unlock()
			atomic.AddInt32(&hits, 1)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":1}}`))
	}))
	defer srv.Close()

	a := &app{
		loggedIn: true,
		client:   qapi.New(srv.URL),
		ctx:      context.Background(),
		convs: []conversation{
			{ID: "100", Name: "当前会话", Kind: "group"},
			{ID: "200", Name: "目标会话", Kind: "single"},
		},
	}

	a.sendForward("200", nil, "把这条转过去")

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&hits) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if atomic.LoadInt32(&hits) == 0 {
		t.Fatal("转发请求未发出")
	}
	mu.Lock()
	path, body := gotPath, gotBody
	mu.Unlock()
	if path != "/api/v1/conversations/200/messages" {
		t.Fatalf("转发目标端点错：%s，期望 /api/v1/conversations/200/messages", path)
	}
	var req struct {
		Type    string `json:"type"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("请求体解析失败: %v (%s)", err, body)
	}
	if req.Type != "text" || req.Content != "把这条转过去" {
		t.Fatalf("转发内容不符: type=%q content=%q，期望 text / 把这条转过去", req.Type, req.Content)
	}
}

// TestMergeForwardPayload 验证合并转发打包成 merged_forward 真协议
// （version 1 + title + 每条 id/type/content/senderName/timestamp 原样保留）。
func TestMergeForwardPayload(t *testing.T) {
	a := &app{
		loggedIn: true,
		active:   0,
		convs:    []conversation{{ID: "100", Name: "项目群", Kind: "group"}},
		msgs: []message{
			{Key: "k1", ServerID: "11", Sender: "张三", Text: "在吗", RawType: "text", RawContent: "在吗", Stamp: 1000, Mine: false},
			{Key: "k2", ServerID: "12", Sender: "我", Text: "在的", RawType: "text", RawContent: "在的", Stamp: 2000, Mine: true},
			{Key: "k3", ServerID: "13", Sender: "李四", Text: "[图片] a.png", Kind: "image",
				RawType: "image", RawContent: `{"url":"/s/x.png"}`, Stamp: 3000, Mine: false},
		},
	}
	a.selected = map[string]bool{"k1": true, "k2": true, "k3": true}

	a.openMergeForward()

	if a.forwardTitle != "合并转发" || !a.forwardOpen {
		t.Fatalf("面板未打开: title=%q open=%v", a.forwardTitle, a.forwardOpen)
	}
	if a.forwardReq == nil || a.forwardReq.Type != "merged_forward" {
		t.Fatalf("应产出 merged_forward 请求: %+v", a.forwardReq)
	}
	mf := qapi.ParseMergedForward(a.forwardReq.Content)
	if mf == nil {
		t.Fatalf("payload 校验失败: %s", a.forwardReq.Content)
	}
	if mf.Title != "项目群 的聊天记录" || len(mf.Messages) != 3 {
		t.Fatalf("payload 不符: title=%q n=%d", mf.Title, len(mf.Messages))
	}
	if mf.Messages[2].Type != "image" || mf.Messages[2].Content != `{"url":"/s/x.png"}` {
		t.Fatalf("图片条目未保留原类型原 content: %+v", mf.Messages[2])
	}
}

// TestForwardRequestFor 验证单条转发按类型挑请求（对齐 buildForwardedMessage）。
func TestForwardRequestFor(t *testing.T) {
	// 文本 → [转发] 前缀
	req := forwardRequestFor(message{RawType: "text", Text: "你好", RawContent: "你好"})
	if req == nil || req.Type != "text" || req.Content != "[转发] 你好" {
		t.Fatalf("文本转发不符: %+v", req)
	}
	// 图片 → 原类型原 content
	req = forwardRequestFor(message{Kind: "image", RawType: "image", Text: "[图片] a.png", RawContent: `{"url":"/s/a.png"}`})
	if req == nil || req.Type != "image" || req.Content != `{"url":"/s/a.png"}` {
		t.Fatalf("图片转发不符: %+v", req)
	}
	// 合并转发原样透传
	req = forwardRequestFor(message{MF: &qapi.MergedForward{Version: 1, Title: "t"}, RawType: "merged_forward", RawContent: `{"version":1}`})
	if req == nil || req.Type != "merged_forward" {
		t.Fatalf("合并转发透传不符: %+v", req)
	}
	// share 卡片包一层 message share
	req = forwardRequestFor(message{SH: &qapi.ShareCard{Name: "便签"}, RawType: "share", RawContent: `{"type":"sticky"}`, Text: "内容"})
	if req == nil || req.Type != "share" {
		t.Fatalf("share 转发不符: %+v", req)
	}
	var sc map[string]any
	if err := json.Unmarshal([]byte(req.Content), &sc); err != nil || sc["type"] != "message" {
		t.Fatalf("share 包装不符: %s (%v)", req.Content, err)
	}
	if sc["originalContent"] != `{"type":"sticky"}` {
		t.Fatalf("share originalContent 应为原 content: %v", sc["originalContent"])
	}
}

// TestForwardPickerRender 验证转发浮层在 headless 下真的渲染出目标会话与标题。
func TestForwardPickerRender(t *testing.T) {
	a := &app{
		loggedIn: true,
		convs: []conversation{
			{ID: "100", Name: "项目群", Kind: "group"},
			{ID: "200", Name: "张三", Kind: "single"},
			{ID: "300", Name: "产品讨论组", Kind: "discussion"},
		},
	}
	a.openForward("待转发内容", "转发消息")

	tt := ui.NewTester(a.view, 1050, 720)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{"转发消息", "选择目标会话", "项目群", "张三", "产品讨论组"} {
		if !tt.HasText(want) {
			t.Errorf("转发面板缺 %q, texts=%q", want, tt.Texts())
		}
	}
}
