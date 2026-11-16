package client

// 富媒体发送：原生文件对话框 → multipart 上传 → image/file 消息。
// 乐观上屏的那一刻就用本地文件直接预览图片，不等上传返回；
// 上传或发送失败都留在气泡上标红，点「重试」会跳过已成功的那一步。

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
	"qim-newclient/internal/theme"
)

// pickAttachment 在 goroutine 里弹原生文件选择框。
// mygo.Dialog.Open 内部会 postMain 再等结果，必须避开 UI 绘制线程，
// 否则主线程等自己投递的任务会死锁。
func (a *app) pickAttachment(kind, convID string) {
	opts := mygo.OpenDialogOptions{
		Title:       "选择文件",
		ButtonLabel: "发送",
		Parent:      a.win,
	}
	if kind == "image" {
		opts.Title = "选择图片"
		opts.Filters = []mygo.FileFilter{{
			Name:       "图片",
			Extensions: []string{"png", "jpg", "jpeg", "gif", "webp", "bmp"},
		}}
	} else {
		opts.Filters = []mygo.FileFilter{{
			Name:       "常用文件",
			Extensions: []string{"pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "txt", "md", "zip", "png", "jpg", "mp4", "mp3"},
		}}
	}
	paths, err := mygo.Dialog.Open(opts)
	if err != nil {
		a.setNotice("选择文件失败：" + friendlyError(err))
		return
	}
	if len(paths) == 0 || paths[0] == "" {
		return
	}
	a.sendAttachment(kind, paths[0], convID, nil, qapi.Attachment{})
}

// sendAttachment 插入一条乐观富媒体消息，然后在后台上传 + 发送。
// uploaded 非空表示重试时附件已经在服务端了，不要重复上传。
func (a *app) sendAttachment(kind, path, convID string, quote *messageQuote, uploaded qapi.Attachment) {
	if convID == "" || path == "" {
		return
	}
	att := uploaded
	if att.URL == "" {
		att = localAttachment(path)
	}
	text := mediaSummary(kind, att)
	a.nextID++
	key := fmt.Sprintf("local-%d", a.nextID)
	now := time.Now()
	msg := message{
		Key: key, Mine: true, Kind: kind, Att: att, LocalPath: path,
		Text: text, At: now.Format("15:04"), Stamp: now.UnixMilli(), Pending: true,
	}
	if quote != nil {
		msg.QuoteID, msg.QuoteSender, msg.QuoteText = quote.ID, quote.Name, quote.Text
	}
	a.mutate(func() {
		// 用户可能已经在选文件期间切走会话：仍然把消息发出去，
		// 但不往当前会话列表里插别人的气泡。
		if a.active >= 0 && a.active < len(a.convs) && a.convs[a.active].ID == convID {
			a.msgs = append(a.msgs, msg)
			a.convs[a.active].Preview = text
			a.convs[a.active].Time = now.Format("15:04")
			a.convs[a.active].Stamp = now.UnixMilli()
		}
		a.quote = nil
	})
	if a.client == nil || !a.loggedIn {
		a.mutate(func() { a.updateLocal(key, false, false) })
		return
	}
	quotedID := ""
	if quote != nil {
		quotedID = quote.ID
	}
	go a.submitAttachment(a.ctx, convID, key, kind, quotedID, path, att)
}

// localAttachment 只从本地文件读元数据，不读文件内容：预览由缓存层按
// localPath 自己读，避免这里把大文件整个读进内存。
func localAttachment(path string) qapi.Attachment {
	name := filepath.Base(path)
	att := qapi.Attachment{
		Name:     name,
		MimeType: mime.TypeByExtension(strings.ToLower(filepath.Ext(name))),
	}
	if st, err := os.Stat(path); err == nil {
		att.Size = st.Size()
	}
	return att
}

func (a *app) submitAttachment(parent context.Context, convID, key, kind, quotedID, path string, att qapi.Attachment) {
	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	client := a.client
	if client == nil {
		a.failLocal(key, "发送失败：未登录")
		return
	}

	// 第一阶段：上传。已上传过（重试）就跳过，避免同一张图在服务端留两份。
	if att.ID == "" || att.URL == "" {
		up, err := client.Upload(ctx, path)
		if err != nil {
			a.failLocal(key, "上传失败："+friendlyError(err))
			return
		}
		att = up
		a.mutate(func() {
			for i := range a.msgs {
				if a.msgs[i].Key == key {
					a.msgs[i].Att = att
					a.msgs[i].Text = mediaSummary(kind, att)
					break
				}
			}
		})
	}

	// 第二阶段：发消息。content 是后端约定的 JSON 字符串（不是纯文本）。
	m, err := client.SendMessage(ctx, qapi.ID(convID), qapi.MediaRequest(kind, att, quotedID))
	if err != nil {
		a.failLocal(key, "发送失败："+friendlyError(err))
		return
	}
	a.mutate(func() {
		a.updateLocal(key, false, false)
		if m != nil && m.ID.String() != "" {
			a.rekeyLocal(key, messageKey(m.ID.String(), 0), m.Timestamp)
		}
	})
	// 粘贴截图的临时文件发送成功后就地删除；失败保留（重试还要用）。
	if base := filepath.Base(path); strings.HasPrefix(base, "qim-paste-") {
		_ = os.Remove(path)
	}
}

func (a *app) failLocal(key, why string) {
	a.mutate(func() {
		a.updateLocal(key, false, true)
		a.notice = why
	})
}

// resendAttachment 重发失败的富媒体消息：保留本地预览与已上传的附件，
// 只重走没成功的那一步。
func (a *app) resendAttachment(m message) {
	convID := a.currentConv().ID
	if convID == "" {
		return
	}
	a.nextID++
	key := fmt.Sprintf("local-%d", a.nextID)
	now := time.Now()
	retry := m
	retry.Key, retry.Pending, retry.Failed = key, true, false
	retry.ServerID, retry.Recalled = "", false
	retry.At, retry.Stamp = now.Format("15:04"), now.UnixMilli()
	a.msgs = append(a.msgs, retry) // 同步插入，旧气泡已在 retryFailed 里摘掉
	if a.client == nil || !a.loggedIn {
		a.updateLocal(key, false, false)
		return
	}
	go a.submitAttachment(a.ctx, convID, key, retry.Kind, retry.QuoteID, retry.LocalPath, retry.Att)
}

// mediaToolButton 把工具栏里原本纯装饰的回形针/图片图标变成真按钮。
func (a *app) mediaToolButton(c *ui.Context, icon *ui.SVG, kind string) {
	// 与 toolButton 同规格（30x30 / radius 4 / hover 浅底），保证工具栏一行对齐。
	btn := ui.Box(c).Width(30).Height(30).Radius(4).Center()
	if btn.Hovered() {
		btn = btn.Background(theme.Gray100)
	}
	if btn.Clicked() && a.loggedIn && a.active >= 0 && a.active < len(a.convs) {
		convID := a.convs[a.active].ID
		go a.pickAttachment(kind, convID)
	}
	btn.Children(func() {
		ui.Icon(c, icon).FontSize(16).TextColor(theme.Gray600)
	})
}
