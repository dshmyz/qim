package client

// A 组聊天体验渲染截图：/quick 面板 + emoji 面板（默认 tab / 经典 tab）。

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestShotQuickEmoji(t *testing.T) {
	a := &app{
		active:   -1,
		loggedIn: true,
		convs: []conversation{
			{ID: "1", Name: "技术交流群", Sub: "(9人)", Preview: "测试用户:收到", Time: "2小时前", Color: "#3385ff", Online: true},
			{ID: "2", Name: "张三", Preview: "好的", Time: "2天前", Color: "#26b361", Online: true},
		},
	}
	a.active = 0
	a.convs[0].Unread = 0
	a.msgs = []message{
		{Key: "m-1", Sender: "测试用户", Text: "你好，这是原生客户端渲染的一条消息。", At: "09:30"},
		{Key: "m-2", Mine: true, Text: "收到，数据来自 qapi。", At: "09:31"},
	}

	// 1. /quick 面板
	a.input = "/quick"
	tt := ui.NewTester(a.view, 1366, 850)
	tt.Frame()
	tt.Frame()
	write(t, tt, "/tmp/qim-quick.png")
	a.closeQuick()
	a.input = ""

	// 2. emoji 面板（默认 tab）
	a.emojiOpen = true
	tt.Frame()
	write(t, tt, "/tmp/qim-emoji.png")

	// 3. emoji 面板（经典 tab）
	a.emojiTab = "classic"
	tt.Frame()
	write(t, tt, "/tmp/qim-emoji-classic.png")
}
