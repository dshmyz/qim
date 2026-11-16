package client

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestToggleNotify 验证 P4 设置页的通知开关：toggleNotify 翻转 notifyOnNew。
func TestToggleNotify(t *testing.T) {
	a := &app{notifyOnNew: true}
	a.toggleNotify()
	if a.notifyOnNew {
		t.Errorf("toggleNotify 应关掉通知")
	}
	a.toggleNotify()
	if !a.notifyOnNew {
		t.Errorf("toggleNotify 应重新打开通知")
	}
}

// TestSettingsView 验证 P4 设置页整屏渲染：标题、通知开关（开）、账号、关于都在。
func TestSettingsView(t *testing.T) {
	a := &app{
		loggedIn:     true,
		settingsOpen: true,
		userName:     "我",
		loginServer:  "http://127.0.0.1:8080",
		notifyOnNew:  true,
	}
	tt := ui.NewTester(a.view, 1050, 703)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{
		"设置", "通知", "新消息桌面通知", "开",
		"账号", "关于", "qim-newclient 0.1.0", "http://127.0.0.1:8080",
	} {
		if !tt.HasText(want) {
			t.Errorf("设置页缺 %q, texts=%q", want, tt.Texts())
		}
	}
}
