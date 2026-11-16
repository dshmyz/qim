package client

// 核心聊天补齐截图：@ 我横幅、card/news/miniApp 占位卡、待发送文件预览条。

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestShotCoreChat(t *testing.T) {
	a := &app{active: 0, loggedIn: true, atSeen: map[string]bool{}}
	a.convs = []conversation{{ID: "1", Name: "产品讨论组", Sub: "(6人)", Preview: "聊天记录", Time: "刚刚", Color: "#3385ff", Kind: "discussion"}}

	at1 := messageFromAPI(mustMsg(`{"id":"61","type":"text","content":"@{mention:7|我} 这条 @ 消息还没看，点横幅跳过来","sender":{"id":"8","name":"李四"},"timestamp":1700000001000}`), "7", 0)
	at1.AtMe = true
	at2 := messageFromAPI(mustMsg(`{"id":"62","type":"text","content":"@{mention:all|所有人} 下午评审","sender":{"id":"8","name":"李四"},"timestamp":1700000002000}`), "7", 0)
	at2.AtMe = true

	card := messageFromAPI(mustMsg(`{"id":"63","type":"card","content":`+quoteJSON(`{"title":"下单确认","text":"订单 #123 等待确认，请在 24 小时内处理。","buttons":[{"id":"ok","text":"确认订单"},{"id":"no","text":"取消"}]}`)+`,"sender":{"id":"9","name":"bot"},"timestamp":1700000003000}`), "7", 0)
	news := messageFromAPI(mustMsg(`{"id":"64","type":"news","content":`+quoteJSON(`{"title":"QIM 2.0 发布","summary":"全新原生客户端上线，支持 markdown 渲染与文件网盘。","url":"https://example.com/blog/qim2"}`)+`,"sender":{"id":"8","name":"李四"},"timestamp":1700000004000}`), "7", 0)
	mini := messageFromAPI(mustMsg(`{"id":"65","type":"miniApp","content":`+quoteJSON(`{"id":"wx1","display_name":"青雀便签","description":"随手记一笔记，多端同步。"}`)+`,"sender":{"id":"8","name":"李四"},"timestamp":1700000005000}`), "7", 0)

	a.msgs = []message{at1, at2, card, news, mini}
	tt := ui.NewTester(a.view, 1200, 1500)
	tt.Frame()
	tt.Frame()
	write(t, tt, "/tmp/qim-corechat.png")

	// 待发送文件预览条（composer 上方）
	tmp := t.TempDir()
	p1 := filepath.Join(tmp, "qim-paste-1.png")
	p2 := filepath.Join(tmp, "需求文档.pdf")
	_ = os.WriteFile(p1, []byte("png"), 0o644)
	_ = os.WriteFile(p2, []byte("pdf"), 0o644)
	a.pendingFiles = []string{p1, p2}
	tt.Frame()
	tt.Frame()
	write(t, tt, "/tmp/qim-corechat-pastebar.png")
	a.clearPendingFiles()
}
