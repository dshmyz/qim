package client

import (
	"image/png"
	"os"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestShot 把界面渲染成 PNG，便于肉眼检查布局。
// 渲染两次：未选会话 → /tmp/qim-shot.png，选中会话 → /tmp/qim-chat.png。
func TestShot(t *testing.T) {
	a := &app{
		active:   -1,
		loggedIn: true,
		convs: []conversation{
			{ID: "1", Name: "技术交流群", Sub: "(9人)", Preview: "测试用户:{\"foo\":\"bar\"}", Time: "2小时前", Unread: 3, Color: "#3385ff", Online: true},
			{ID: "2", Name: "系统", Preview: "【系统通知】新版本 v2.0...", Time: "2天前", Color: "#26b361", Online: true},
			{ID: "3", Name: "写作助手", Preview: "你好！我是AI助手，有什么...", Time: "07-02", Color: "#6366f1", Online: true},
			{ID: "4", Name: "系统助手", Preview: "暂无消息", Time: "07-02", Color: "#14b8a6"},
		},
	}
	tt := ui.NewTester(a.view, 1366, 850)
	tt.Frame()
	write(t, tt, "/tmp/qim-shot.png")
	a.active = 0
	a.convs[0].Unread = 0
	a.msgs = []message{
		{Key: "m-1", Sender: "测试用户", Text: "你好，这是原生客户端渲染的一条消息。", At: "09:30"},
		{Key: "m-2", Mine: true, Text: "收到，数据来自 qapi，不再有网页层。", At: "09:31"},
	}
	tt.Frame()
	write(t, tt, "/tmp/qim-chat.png")
}

// TestShotLogin 把原生登录页也存一张，确认表单、错误提示位置正常。
func TestShotLogin(t *testing.T) {
	a := &app{active: -1, loginServer: "http://localhost:8080"}
	tt := ui.NewTester(a.view, 1366, 850)
	tt.Frame()
	write(t, tt, "/tmp/qim-login.png")
}

func write(t *testing.T, tt *ui.Tester, path string) {
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, tt.Image()); err != nil {
		t.Fatal(err)
	}
}
