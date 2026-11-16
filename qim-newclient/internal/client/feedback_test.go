package client

// C 组杂项（问题反馈 / 检查更新）主包测试。

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

// TestIsNewerVersion 点分段比较：任何一段更大即更新。
func TestIsNewerVersion(t *testing.T) {
	cases := []struct {
		newV, curV string
		want       bool
	}{
		{"2.1.0", "2.0.39", true},
		{"3.0", "2.0.39", true},
		{"2.0.40", "2.0.39", true},
		{"2.0.39", "2.0.39", false},
		{"2.0.38", "2.0.39", false},
		{"v2.1.0", "2.0.39", true},
		{"1.9.9", "2.0.39", false},
	}
	for _, tc := range cases {
		if got := isNewerVersion(tc.newV, tc.curV); got != tc.want {
			t.Errorf("isNewerVersion(%q,%q)=%v 期望 %v", tc.newV, tc.curV, got, tc.want)
		}
	}
}

// TestFbLabels 类型/状态/时间格式化。
func TestFbLabels(t *testing.T) {
	if fbTypeLabel("bug") != "Bug反馈" || fbTypeLabel("feature") != "功能建议" || fbTypeLabel("other") != "其他" {
		t.Fatal("类型文案错")
	}
	if fbStatusLabel("pending") != "待处理" || fbStatusLabel("resolved") != "已解决" || fbStatusLabel("nope") != "nope" {
		t.Fatal("状态文案错")
	}
	if got := fbTimeLabel("2026-11-15T10:30:00+08:00"); got != "11-15 10:30" {
		t.Fatalf("时间格式错: %q", got)
	}
	if got := fbTimeLabel("not-a-time"); got == "" {
		t.Fatal("非法时间应兜底非空")
	}
}

// TestCanSubmitFeedback 内容去空白 ≥10 字才可提交。
func TestCanSubmitFeedback(t *testing.T) {
	a := &app{fbContent: "太短"}
	if a.canSubmitFeedback() {
		t.Fatal("短内容不应可提交")
	}
	a.fbContent = "  这条反馈内容足够长可以提交了  "
	if !a.canSubmitFeedback() {
		t.Fatal("长内容应可提交")
	}
}

// TestSubmitFeedbackFlow 提交成功：表单字段正确、成功后切列表视图并刷新。
func TestSubmitFeedbackFlow(t *testing.T) {
	var gotType, gotContent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/feedbacks") && r.Method == http.MethodPost:
			_ = r.ParseMultipartForm(1 << 20)
			gotType = r.FormValue("type")
			gotContent = r.FormValue("content")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":0,"data":{"id":1}}`))
		case strings.HasSuffix(r.URL.Path, "/my-feedbacks"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":0,"data":{"list":[{"id":1,"type":"bug","content":"x","status":"pending","createdAt":"2026-11-15T10:00:00Z"}],"total":1}}`))
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":0,"data":null}`))
		}
	}))
	defer srv.Close()

	a := &app{loggedIn: true, client: qapi.New(srv.URL), ctx: context.Background()}
	a.openFeedback()
	a.fbType = "bug"
	a.fbContent = "这条反馈内容足够长可以提交了"
	a.submitFeedback()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.Lock()
		done := !a.fbBusy && a.fbView == "list"
		a.mu.Unlock()
		if done {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if gotType != "bug" || gotContent != "这条反馈内容足够长可以提交了" {
		t.Fatalf("表单字段错: type=%q content=%q", gotType, gotContent)
	}
	a.mu.Lock()
	view, content, busy := a.fbView, a.fbContent, a.fbBusy
	a.mu.Unlock()
	if view != "list" || content != "" || busy {
		t.Fatalf("成功后应切列表+清内容: view=%q content=%q busy=%v", view, content, busy)
	}
}

// TestCheckUpdateFlow 有新版本：弹窗打开、hasNew 置位、结果文案正确。
func TestCheckUpdateFlow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"list":[{"version":"3.0.0","updateNotes":"大版本更新","downloadUrl":"https://example.com/dl"}],"total":1}}`))
	}))
	defer srv.Close()

	a := &app{loggedIn: true, client: qapi.New(srv.URL), ctx: context.Background()}
	a.checkUpdate()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.Lock()
		checking := a.updChecking
		a.mu.Unlock()
		if !checking {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.updOpen || !a.updHasNew || a.updInfo == nil {
		t.Fatalf("应发现新版本: open=%v hasNew=%v info=%+v", a.updOpen, a.updHasNew, a.updInfo)
	}
	if a.updResult != "发现新版本 v3.0.0" {
		t.Fatalf("文案错: %q", a.updResult)
	}
}

// TestCheckUpdateLatest 无记录/同版本 → 已是最新。
func TestCheckUpdateLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"list":[],"total":0}}`))
	}))
	defer srv.Close()

	a := &app{loggedIn: true, client: qapi.New(srv.URL), ctx: context.Background()}
	a.checkUpdate()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.Lock()
		checking := a.updChecking
		a.mu.Unlock()
		if !checking {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.updHasNew || a.updResult != "当前已是最新版本" {
		t.Fatalf("无版本记录应显示已最新: hasNew=%v result=%q", a.updHasNew, a.updResult)
	}
}

// TestFeedbackModalRender headless 渲染提交视图与列表视图。
func TestFeedbackModalRender(t *testing.T) {
	a := &app{
		loggedIn: true,
		convs:    []conversation{{ID: "1", Name: "张三", Kind: "single"}},
		active:   0,
	}
	a.openFeedback()
	tt := ui.NewTester(a.view, 1050, 720)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{"问题反馈", "反馈类型", "Bug反馈", "功能建议", "至少 10 个字", "点击上传截图", "我的反馈", "提交反馈"} {
		if !tt.HasText(want) {
			t.Errorf("反馈弹层缺 %q, texts=%q", want, tt.Texts())
		}
	}
	// 切列表视图：显示记录 + 状态标签
	a.fbList = []qapi.UserFeedback{
		{ID: 7, Type: "bug", Content: "输入框换行会丢字", Status: "resolved", Reply: "已修复", HandlerName: "管理员", CreatedAt: "2026-11-15T10:30:00+08:00"},
	}
	a.fbView = "list"
	tt.Frame()
	for _, want := range []string{"我的反馈", "Bug反馈", "11-15 10:30", "输入框换行会丢字", "已解决", "返回提交"} {
		if !tt.HasText(want) {
			t.Errorf("反馈列表缺 %q, texts=%q", want, tt.Texts())
		}
	}
}

// TestUpdateDialogRender headless 渲染检查更新弹窗（发现新版本态）。
func TestUpdateDialogRender(t *testing.T) {
	a := &app{
		loggedIn: true,
		convs:    []conversation{{ID: "1", Name: "张三", Kind: "single"}},
		active:   0,
	}
	a.updOpen = true
	a.updHasNew = true
	a.updResult = "发现新版本 v3.0.0"
	a.updInfo = &qapi.ClientVersion{Version: "3.0.0", UpdateNotes: "大版本更新", DownloadURL: "https://example.com/dl"}
	tt := ui.NewTester(a.view, 1050, 720)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{"检查更新", "发现新版本 v3.0.0", "大版本更新", "前往下载", "关闭"} {
		if !tt.HasText(want) {
			t.Errorf("更新弹窗缺 %q, texts=%q", want, tt.Texts())
		}
	}
}

// TestOpenURLGuard openURL 只放行 http/https。
func TestOpenURLGuard(t *testing.T) {
	if err := openURL("file:///etc/passwd"); err != nil {
		t.Fatalf("非 http(s) 链接应静默忽略: %v", err)
	}
	if err := openURL("  "); err != nil {
		t.Fatalf("空链接应静默忽略: %v", err)
	}
	_ = io.Discard
}
