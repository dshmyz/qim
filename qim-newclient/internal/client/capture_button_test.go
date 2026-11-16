package client

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

// TestTitleStripNoDebugChips 顶栏对齐原版 Sidebar/layout：只有 QIM logo、
// 拖动区和三色灯，不再有「● 连接状态」「📷 截图」这两个本端调试入口。
func TestTitleStripNoDebugChips(t *testing.T) {
	a := &app{active: -1, loggedIn: true, connStatus: qapi.StatusConnected}
	tt := ui.NewTester(a.view, 1366, 850)
	tt.Frame()
	if !tt.HasText("QIM") {
		t.Errorf("顶栏缺 QIM logo, texts=%q", tt.Texts())
	}
	for _, bad := range []string{"📷 截图", "已连接", "连接中"} {
		if tt.HasText(bad) {
			t.Errorf("顶栏不该再有调试入口 %q, texts=%q", bad, tt.Texts())
		}
	}
}

// TestTitleStripNoBottomLine 顶栏下方不该多出横线：此前整行 Shadow 在
// y=40 的左侧 360px 画了一条 #f2f2f2 灰线，原版该处是纯白。
func TestTitleStripNoBottomLine(t *testing.T) {
	a := &app{active: -1, loggedIn: true, connStatus: qapi.StatusConnected}
	tt := ui.NewTester(a.view, 1050, 703)
	tt.Frame()
	tt.Frame()
	for _, x := range []int{10, 200, 350} {
		r, g, b, _ := tt.Image().At(x, 40).RGBA()
		if v := (int(r>>8) + int(g>>8) + int(b>>8)) / 3; v < 250 {
			t.Errorf("顶栏下方 x=%d 出现多余横线, 亮度=%d", x, v)
		}
	}
}

// TestSidebarPlusMenu 「+」对齐原版 actionMenu：左键弹菜单（创建群聊 /
// 创建频道），而不是直接蹦出建群弹窗。
func TestSidebarPlusMenu(t *testing.T) {
	a := &app{active: -1, loggedIn: true, userName: "管理员"}
	tt := ui.NewTester(a.view, 1366, 850)
	tt.Frame()

	// 侧栏头右侧的「+」：60(导航) + 300 - 20(padding) - 14(半个按钮) = 326。
	tt.ClickAt(326, 68)
	tt.Frame()
	for _, want := range []string{"创建群聊", "创建频道"} {
		if !tt.HasText(want) {
			t.Fatalf("菜单缺 %q, texts=%q", want, tt.Texts())
		}
	}
	write(t, tt, "/tmp/qim-sidebar-menu.png")
	if err := tt.Click("创建群聊"); err != nil {
		t.Fatalf("点「创建群聊」失败: %v", err)
	}
	tt.Frame()
	if !a.orgOpen || a.orgMode != "create" {
		t.Errorf("建群弹窗没打开: open=%v mode=%q", a.orgOpen, a.orgMode)
	}
}

// TestShotSidebarTop 侧栏顶部截图：头像 + 昵称 + 通知角标（99+ 封顶）+ 加号。
func TestShotSidebarTop(t *testing.T) {
	a := &app{
		active:       -1,
		loggedIn:     true,
		userName:     "管理员",
		connStatus:   qapi.StatusConnected,
		unreadNotifs: 120,
		convs: []conversation{
			{ID: "1", Name: "技术交流群", Kind: "group", Preview: "张三: 收到", Time: "刚刚", Unread: 3},
		},
	}
	tt := ui.NewTester(a.view, 1366, 850)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("99+") {
		t.Errorf("通知角标没封顶成 99+, texts=%q", tt.Texts())
	}
	write(t, tt, "/tmp/qim-sidebar-top.png")
}

// TestLogoutMovedToSettings 退出登录从侧栏头部移到设置页账号区，
// 对齐原版 settingsMenuItems 的位置；侧栏头部不再有「退出」按钮。
func TestLogoutMovedToSettings(t *testing.T) {
	a := &app{active: -1, loggedIn: true, connStatus: qapi.StatusConnected, settingsOpen: true}
	tt := ui.NewTester(a.view, 1366, 850)
	tt.Frame()
	if !tt.HasText("退出登录") {
		t.Errorf("设置页缺「退出登录」, texts=%q", tt.Texts())
	}
	if !tt.HasText("连接状态") {
		t.Errorf("设置页缺「连接状态」, texts=%q", tt.Texts())
	}

	// 侧栏头部：只剩头像 + 昵称 + 铃铛 + 加号。
	b := &app{active: -1, loggedIn: true, userName: "管理员"}
	tb := ui.NewTester(b.view, 1366, 850)
	tb.Frame()
	for _, bad := range []string{"退出", "已连接"} {
		if tb.HasText(bad) {
			t.Errorf("侧栏头部不该有 %q, texts=%q", bad, tb.Texts())
		}
	}
	if !tb.HasText("管理员") {
		t.Errorf("侧栏头部缺当前用户昵称, texts=%q", tb.Texts())
	}
}
