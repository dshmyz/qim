package client

// C 组杂项渲染截图：设置页（帮助/检查更新入口）+ 反馈弹层 + 更新弹窗。

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

func TestShotCGroup(t *testing.T) {
	a := &app{
		active:   -1,
		loggedIn: true,
		userName: "探针账号",
		convs: []conversation{
			{ID: "1", Name: "技术交流群", Sub: "(9人)", Preview: "测试用户:收到", Time: "2小时前", Color: "#3385ff", Online: true},
		},
	}
	a.active = 0
	a.settingsOpen = true
	tt := ui.NewTester(a.view, 1366, 850)
	tt.Frame()
	tt.Frame()
	write(t, tt, "/tmp/qim-settings-c.png")

	// 反馈弹层（提交视图）
	a.settingsOpen = false
	a.openFeedback()
	a.fbType = "bug"
	a.fbContent = "输入框在群聊里换行发送后会丢失最后一个字符，麻烦看一下"
	tt.Frame()
	write(t, tt, "/tmp/qim-feedback.png")

	// 我的反馈列表
	a.switchFbList()
	a.fbList = []qapi.UserFeedback{
		{ID: 7, Type: "bug", Content: "输入框在群聊里换行发送后会丢失最后一个字符", Status: "resolved", Reply: "已修复，下个版本发布", HandlerName: "管理员", CreatedAt: "2026-11-15T10:30:00+08:00"},
		{ID: 6, Type: "feature", Content: "希望加夜间模式", Status: "pending", CreatedAt: "2026-11-14T09:00:00Z"},
	}
	tt.Frame()
	write(t, tt, "/tmp/qim-feedback-list.png")

	// 检查更新弹窗（发现新版本）
	a.closeFeedback()
	a.updOpen = true
	a.updHasNew = true
	a.updResult = "发现新版本 v3.0.0"
	a.updInfo = &qapi.ClientVersion{Version: "3.0.0", UpdateNotes: "大版本更新：全新 UI 与性能优化", DownloadURL: "https://example.com/dl"}
	tt.Frame()
	write(t, tt, "/tmp/qim-update.png")
}
