package client

import (
	"context"
	"testing"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

// view 在没有窗口的 headless tester 里跑，点击和输入都按真人的路径来。
func TestView(t *testing.T) {
	a := &app{
		active:   -1,
		loggedIn: true, // 未登录会画登录页；这里测的是主界面
		convs: []conversation{
			{ID: "101", Name: "技术交流群", Preview: "测试用户", Time: "2小时前", Unread: 3, Color: "#3b82f6"},
			{ID: "102", Name: "张三", Preview: "收到", Time: "07-02", Color: "#10b981"},
		},
	}
	tt := ui.NewTester(a.view, 1080, 700)
	if !tt.HasText("QIM") {
		t.Errorf("nav rail logo missing, texts %q", tt.Texts())
	}
	if !tt.HasText("选择一个会话开始聊天") {
		t.Errorf("placeholder missing, texts %q", tt.Texts())
	}
	// 点击会话打开聊天；client 为 nil 时不发网络请求，只走本地状态
	if err := tt.Click("张三"); err != nil {
		t.Fatal(err)
	}
	if a.active != 1 {
		t.Errorf("active = %d, want 1", a.active)
	}
	if a.convs[1].Unread != 0 {
		t.Errorf("选中后未读没清零: %d", a.convs[1].Unread)
	}
	if !tt.HasText("暂无消息，发一条试试") {
		t.Errorf("空消息态没显示, texts %q", tt.Texts())
	}

	// 输入框 AutoFocus，直接输入并按 Enter 发送（TextArea 的 OnTextKey）
	tt.Type("测试消息")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if len(a.msgs) == 0 || a.msgs[len(a.msgs)-1].Text != "测试消息" {
		t.Errorf("last message %v, want 测试消息", a.msgs)
	}
	if a.input != "" {
		t.Errorf("input not cleared: %q", a.input)
	}
}

func TestLoginView(t *testing.T) {
	a := &app{
		active: -1,
		client: qapi.New("http://127.0.0.1:1"),
		ctx:    context.Background(),
	}
	tt := ui.NewTester(a.view, 900, 700)
	tt.Frame()
	for _, want := range []string{"QIM 青雀", "服务器地址", "账号", "密码", "登录"} {
		if !tt.HasText(want) {
			t.Errorf("登录页缺 %q, texts %q", want, tt.Texts())
		}
	}
	// 空表单提交要给出明确报错，不能静默无反应
	a.submitLogin()
	if a.loginErr == "" {
		t.Error("空表单应提示填写服务器地址/账号")
	}
}

func TestSearchFiltersConversations(t *testing.T) {
	a := &app{
		active: -1,
		convs: []conversation{
			{ID: "1", Name: "技术交流群", Preview: "构建挂了"},
			{ID: "2", Name: "系统通知", Preview: "版本更新"},
		},
	}
	if got := len(a.visibleConvs()); got != 2 {
		t.Fatalf("未搜索时应显示 2 条，实际 %d", got)
	}
	a.search = "构建"
	got := a.visibleConvs()
	if len(got) != 1 || a.convs[got[0]].Name != "技术交流群" {
		t.Errorf("搜索预览没命中: %v", got)
	}
	a.search = "系统"
	got = a.visibleConvs()
	if len(got) != 1 || a.convs[got[0]].Name != "系统通知" {
		t.Errorf("搜索名称没命中: %v", got)
	}
}
