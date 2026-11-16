// 输入区与发送动作。
// 从 main.go 按职责拆出，逻辑未改动。
package client

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"qim-newclient/internal/theme"
	"strings"
)

// composer 底部输入区（全宽），对齐 qim-client MessageInput.vue / ChatToolbar.vue：
// 工具栏（30x30 按钮 / gap 8）→ 斜杠首次引导条 → 白底 composer（120 高，
// 聚焦主色边框）→ input-actions 一行（左快捷键提示 / 右发送按钮）。
func (a *app) composer(c *ui.Context) {
	col := ui.Column(c).FillWidth().Background(theme.White).BorderWidth(1, 0, 0, 0).BorderColor(theme.Gray200)
	// 文件悬停在输入区时整块高亮（对齐 qim-client 的 .drag-over）；
	// 原版是虚线边框，mygo 没有 dashed，用主色实线 + 浅主色底近似。
	if col.FileDragOver() {
		col = col.Background(theme.PrimarySoft).BorderWidth(2).BorderColor(theme.Primary).Radius(8)
	}
	// 文件拖进输入区直接进待发送队列（对齐 qim-client handleDrop）。
	a.takeDropped(col)
	col.Children(func() {
		// 工具栏顺序对齐 ChatToolbar.vue：通话(带下拉) / 屏幕 / 表情 /
		// 文件 / 图片 / 代码块 / 小程序 / AI / 右端消息管理。
		// 通话、屏幕、代码块、小程序、AI、消息管理按需求搁置，这里只保留
		// 与网页端一致的占位与按钮态，不伪造交互。
		ui.Row(c).FillWidth().Height(40).Padding(0, 15).AlignItems(ui.Center).Gap(8).Children(func() {
			a.toolButton(c, iconPhone, theme.Gray600, false, true, nil)
			a.toolButton(c, iconMonitor, theme.Gray600, false, false, nil)
			a.toolButton(c, iconSmile, smileColor(a.emojiOpen), a.emojiOpen, false, func() { a.toggleEmoji() })
			a.mediaToolButton(c, iconClip, "file")
			a.mediaToolButton(c, iconImage, "image")
			a.toolButton(c, iconCode, theme.Gray600, false, false, nil)
			a.toolButton(c, iconGrid, theme.Gray600, false, false, nil)
			a.toolButton(c, iconBot, ui.Hex("#10a37f"), false, false, nil)
			ui.Spacer(c)
			a.toolButton(c, iconClock, theme.Gray600, false, false, nil)
		})
		// 斜杠命令首次引导：关闭 / 试一下都会记住，重启不再出现。
		a.slashHint(c)
		a.selectionBar(c)
		a.quoteBar(c)
		a.pasteBar(c)
		a.quickPanel(c)
		a.mentionPanel(c)
		// composer 统一容器：浅色边框 + 白底；聚焦时边框转主色
		// （对齐 .composer:focus-within）。
		ui.Row(c).FillWidth().Padding(0, 15).Children(func() {
			box := ui.Box(c).FillWidth().Height(120).Radius(8).
				BorderWidth(1).BorderColor(theme.Gray200).Background(theme.White)
			box.Children(func() {
				ta := ui.TextArea(c, &a.input).Placeholder("输入消息...").FillWidth().FillHeight().
					FontSize(14).BorderWidth(0).Background(theme.White).Padding(10, 12).AutoFocus().
					// 粘贴截图直接进待发送队列（对齐 qim-client handlePaste）：
					// 剪贴板有图片就拦截，文本照常插入。
					OnPaste(a.takePaste).
					OnTextKey(func(k ui.TextKey) bool {
						if k.Key != ui.KeyEnter || k.Mods != 0 {
							return false // Shift+Enter 交给输入框换行
						}
						// 命令面板打开时 Enter 采纳第一条短语，不发原命令文本。
						if a.quickOpen {
							if hits := a.filteredQuickReplies(); len(hits) > 0 {
								a.applyQuickReply(hits[0])
								k.SetText(a.input)
							}
							return true
						}
						a.submitComposer(k.Text())
						k.SetText("")
						return true
					})
				if ta.Focused() || box.Focused() {
					box.BorderColor(theme.Primary)
				}
			})
		})
		// input-actions：与输入框分离的一行，发送按钮在右侧而不是浮在框内。
		ui.Row(c).FillWidth().Padding(8, 15, 12, 15).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, "按 Enter 发送，Shift+Enter 换行").FontSize(12).TextColor(theme.Gray500).Opacity(0.6)
			ui.Spacer(c)
			canSend := a.canSendComposer()
			btn := ui.Box(c).Width(75).Height(32).Radius(4).Center().Disabled(!canSend)
			if canSend {
				btn = btn.Background(theme.Primary)
				if btn.Hovered() {
					btn = btn.Background(theme.PrimaryStrong)
				}
			} else {
				btn = btn.Background(theme.Gray200)
			}
			if btn.Clicked() {
				if a.quickOpen {
					if hits := a.filteredQuickReplies(); len(hits) > 0 {
						a.applyQuickReply(hits[0])
					}
				} else {
					a.submitComposer(a.input)
					a.input = ""
				}
			}
			btn.Children(func() {
				fg := theme.White
				if !canSend {
					fg = theme.Gray500
				}
				ui.Text(c, "发送").FontSize(14).TextColor(fg)
			})
		})
	})
}

// smileColor 是表情按钮的激活态前景色（对齐 .toolbar-btn 的 hover/active）。
func smileColor(active bool) ui.Color {
	if active {
		return theme.Primary
	}
	return theme.Gray600
}

// toolButton 是统一的工具栏按钮：30x30 点击区、radius 4、hover 浅底、
// 16px 图标；action 为 nil 时只占位保持布局（功能按需求搁置）。
func (a *app) toolButton(c *ui.Context, icon *ui.SVG, fg ui.Color, active, chevron bool, action func()) {
	width := float32(30)
	if chevron {
		width = 40
	}
	btn := ui.Box(c).Width(width).Height(30).Radius(4).Center()
	switch {
	case active:
		btn = btn.Background(theme.Primary50)
	case btn.Hovered() && action != nil:
		btn = btn.Background(theme.Gray100)
	}
	if btn.Clicked() && action != nil {
		action()
	}
	btn.Children(func() {
		ui.Row(c).Gap(2).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icon).FontSize(16).TextColor(fg)
			if chevron {
				ui.Icon(c, iconChevDown).FontSize(10).TextColor(theme.Gray500)
			}
		})
	})
}

// canSendComposer 对齐原版 send-btn 的 :disabled 条件：
// 没有文字且没有待发送附件时禁用。
func (a *app) canSendComposer() bool {
	return strings.TrimSpace(a.input) != "" || len(a.pendingFiles) > 0
}

// submitComposer 是 Enter 与发送按钮共用的提交路径：待发送附件先发
// （图片/文件各成一条），输入框文字随后正常发。命令面板由调用方处理。
func (a *app) submitComposer(text string) {
	if len(a.pendingFiles) > 0 {
		a.sendPendingFiles()
	}
	if t := strings.TrimSpace(text); t != "" {
		a.sendText(t)
	}
}

// selectionBar 是多选模式下的操作条，替换掉平时的发送区提示。
func (a *app) selectionBar(c *ui.Context) {
	if !a.selectMode {
		return
	}
	ui.Row(c).FillWidth().Height(38).Padding(0, 15).Gap(10).AlignItems(ui.Center).
		Background(theme.Primary50).Children(func() {
		ui.Text(c, fmt.Sprintf("已选 %d 条消息", len(a.selectedKeys()))).
			FontSize(12).TextColor(theme.Gray800)
		ui.Spacer(c)
		smallAction(c, "复制", func() { a.copyMessageText(a.selectedText()) })
		smallAction(c, "合并转发", func() { a.openMergeForward() })
		smallAction(c, "删除", func() { a.deleteMessages(a.selectedKeys()) })
		smallAction(c, "取消", func() { a.exitSelectMode() })
	})
}

func smallAction(c *ui.Context, label string, fn func()) {
	btn := ui.Box(c).Height(24).Radius(5).Padding(0, 10).Background(theme.White).
		BorderWidth(1).BorderColor(theme.Gray200).Center()
	if btn.Clicked() {
		fn()
	}
	btn.Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(theme.Gray800)
	})
}
