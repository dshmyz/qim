package client

// paste：待发送文件队列（对齐 qim-client pendingFiles + PendingFilesPreview）。
// 两个入口：
//   1. 输入框 Cmd+V 粘贴截图 —— OnPaste 回调里读剪贴板 PNG 存临时文件；
//   2. 文件拖进 composer —— 每帧查容器元素的 DroppedFiles。
// 文件先进 pendingFiles，输入框上方出预览条（名称/大小/✕ 移除），
// Enter 发送（逐条按扩展名定 image/file 类型），与文字互不影响。

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
	"qim-newclient/internal/theme"
)

// takePaste 是输入框的 OnPaste 回调：剪贴板里有图片就把它收进待发送
// 队列并拦截这次粘贴（不让图片当乱码文本插进输入框）；没有图片返回
// false，文本照常粘贴。对齐 qim-client ChatWindow.handlePaste。
func (a *app) takePaste(text string) bool {
	data := a.svc.ReadClipboardData(transfer.PNG)
	if len(data.Items()) == 0 {
		return false
	}
	png, err := data.Read(transfer.PNG)
	if err != nil || len(png) == 0 {
		return false
	}
	path := filepath.Join(os.TempDir(), fmt.Sprintf("qim-paste-%d.png", time.Now().UnixNano()))
	if err := os.WriteFile(path, png, 0o644); err != nil {
		return false
	}
	a.pendingFiles = append(a.pendingFiles, path)
	return true
}

// takeDropped 是 composer 容器每帧查的文件拖放：路径进同一个待发送队列。
func (a *app) takeDropped(e ui.Element) {
	for _, p := range e.DroppedFiles() {
		if p != "" {
			a.pendingFiles = append(a.pendingFiles, p)
		}
	}
}

// removePendingFile 移除待发送队列里的一个文件；临时粘贴文件顺带删掉。
func (a *app) removePendingFile(i int) {
	if i < 0 || i >= len(a.pendingFiles) {
		return
	}
	path := a.pendingFiles[i]
	a.pendingFiles = append(a.pendingFiles[:i], a.pendingFiles[i+1:]...)
	if strings.HasPrefix(filepath.Base(path), "qim-paste-") {
		_ = os.Remove(path)
	}
}

// clearPendingFiles 退出会话/重置时清空队列，临时文件一并删除。
func (a *app) clearPendingFiles() {
	for i := range a.pendingFiles {
		a.removePendingFile(i)
	}
}

// pendingKind 按扩展名定消息类型：图片后缀发 image，其余发 file。
// 对齐 qim-client 的 uploadAndSendFile（截图 png 也走图片消息）。
func pendingKind(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp":
		return "image"
	}
	return "file"
}

// sendPendingFiles 把队列逐条发出去（不含输入框文字；qim-client 同样
// 分开发），全部清空。引用只挂第一条。发送走 sendAttachment 的既定
// 管线（乐观占位+上传）。
func (a *app) sendPendingFiles() {
	convID := a.convs[a.active].ID
	var quote *messageQuote
	if len(a.pendingFiles) > 0 && a.quote != nil {
		quote = a.quote
		a.quote = nil
	}
	for _, path := range a.pendingFiles {
		a.sendAttachment(pendingKind(path), path, convID, quote, qapi.Attachment{})
		quote = nil
	}
	a.pendingFiles = nil
}

// pasteBar 是输入框上方的待发送预览条（对齐 PendingFilesPreview）。
func (a *app) pasteBar(c *ui.Context) {
	if len(a.pendingFiles) == 0 {
		return
	}
	ui.Column(c).FillWidth().Padding(6, 15, 0, 15).Gap(4).Children(func() {
		for i, path := range a.pendingFiles {
			name := filepath.Base(path)
			size := ""
			if st, err := os.Stat(path); err == nil {
				size = qapi.FormatSize(st.Size())
			}
			row := ui.Row(c).FillWidth().Height(40).Radius(8).Background(theme.Gray50).
				Padding(0, 12).Gap(10).AlignItems(ui.Center)
			row.Children(func() {
				ui.Box(c).Width(26).Height(26).Radius(6).Background(theme.Info).Center().Children(func() {
					glyph := "📄"
					if pendingKind(path) == "image" {
						glyph = "🖼"
					}
					ui.Text(c, glyph).FontSize(12)
				})
				ui.Column(c).Grow(1).Gap(1).Children(func() {
					ui.Text(c, name).FontSize(12).TextColor(theme.Gray800).MaxLines(1).Ellipsis("…")
					ui.Text(c, size).FontSize(10).TextColor(theme.Gray500)
				})
				del := ui.Box(c).Width(22).Height(22).Radius(4).Center()
				if del.Clicked() {
					a.removePendingFile(i)
				}
				del.Children(func() {
					ui.Icon(c, iconClose).FontSize(13).TextColor(theme.Gray500)
				})
			})
		}
		ui.Text(c, "按 Enter 发送附件，Shift+Enter 只发文字").FontSize(10).TextColor(theme.Gray500).Padding(2, 2)
	})
}
