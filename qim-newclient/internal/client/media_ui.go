package client

// 富媒体气泡：文本 / 图片 / 文件卡片。
// 图片位图由 mediacache.go 异步加载，这里只负责「有图就画、没图就占位」，
// 绝不在绘制帧里发网络请求。

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
	"qim-newclient/internal/theme"
)

const (
	// 对齐 qim-client ImageMessage.vue：图片 max-height 250，容器
	// max-width 100%（气泡最大 560）。之前 240×180 把长截图压得
	// 字都看不清，看着反而乱。
	imageMaxW = 400.0
	imageMaxH = 250.0
)

// messageBubble 是消息列表里两处（自己 / 对方）共用的气泡入口。
// 按 kind 分流，避免各自复制一套分支。
func (a *app) messageBubble(c *ui.Context, m message) {
	switch {
	case m.Recalled:
		a.recalledPill(c, m)
	case m.Sys:
		a.systemPill(c, m)
	case m.MD:
		a.mdBubble(c, m)
	case m.Kind == "image":
		a.imageBubble(c, m)
	case m.Kind == "file", m.Kind == "video", m.Kind == "audio":
		a.fileBubble(c, m)
	case m.MF != nil:
		a.mergedCard(c, m)
	case m.SH != nil:
		a.shareCard(c, m)
	case m.CD != nil:
		a.botCard(c, m)
	case m.NW != nil:
		a.newsCard(c, m)
	case m.MI != nil:
		a.miniAppCard(c, m)
	default:
		a.textBubble(c, m)
	}
}

// recalledPill 对齐 .recalled-message：灰胶囊 + 同行「重新编辑」，
// 而不是普通气泡下面再挂一个按钮。
func (a *app) recalledPill(c *ui.Context, m message) {
	box := ui.Box(c).Radius(16).Padding(8, 12).Background(ui.Hex("#f2f2f2")).Center()
	box.Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, m.Text).FontSize(12).TextColor(theme.Gray500)
			if m.OrigText != "" {
				edit := ui.Box(c).Height(20).Radius(4).Padding(0, 6).Center()
				if edit.Hovered() {
					edit = edit.Background(ui.Hex("#e3eefc"))
				}
				if edit.Clicked() {
					a.editRecalled(m)
				}
				edit.Children(func() {
					ui.Text(c, "✎ 重新编辑").FontSize(12).TextColor(theme.Primary)
				})
			}
		})
	})
	a.attachMessageMenu(a.highlightBubble(box, m), m)
}

func (a *app) textBubble(c *ui.Context, m message) {
	// 对方气泡 = --message-bubble-bg(--color-gray-100) #f8f8f8；
	// 自己气泡 = 主色混 88% 白 + 深字（见 selfBubbleBG）。
	bg, fg := theme.Gray100, theme.Gray800
	if m.Mine {
		bg, fg = selfBubbleBG(), selfBubbleFG()
	}
	var body ui.Element
	if len(m.Segs) > 0 {
		// 富文本渲染：@ 提及高亮 chip、链接可点击（对齐 qim-client TextMessage）。
		body = ui.RichText(c).Selectable().Padding(10, 14).Radius(12).
			FontSize(14).LineHeight(1.6).Background(bg).
			TextColor(fg).MaxWidth(560)
		body.Children(func() {
			for _, seg := range m.Segs {
				if seg.Mention != nil {
					a.mentionChip(c, seg.Text, seg.Mention.Target == "all", m.Mine)
					continue
				}
				a.richPlain(c, seg.Text, m.Mine)
			}
		})
	} else {
		body = ui.Text(c, m.Text).Selectable().Padding(10, 14).Radius(12).
			FontSize(14).LineHeight(1.6).Background(bg).
			TextColor(fg).MaxWidth(560)
	}
	bubble := a.highlightBubble(body, m)
	a.attachMessageMenu(bubble, m)
}

// mentionChip 在富文本段落里画一个 @ 提及 chip：@all 用警示色，
// 单人用主题色；自己气泡（主色底）上反色。
// mygo 富文本子元素不支持 padding/圆角，用背景高亮近似网页端的 chip。
func (a *app) mentionChip(c *ui.Context, text string, all, mine bool) {
	el := ui.Text(c, text)
	switch {
	case mine:
		// 浅主色气泡上用深蓝字 + 略深的 chip 底，别再反白。
		el = el.TextColor(selfBubbleLink()).Background(ui.Hex("#d3e4ff"))
	case all:
		el = el.TextColor(theme.WarningTx).Background(theme.WarningBg)
	default:
		el = el.TextColor(theme.Primary).Background(theme.Primary50)
	}
}

// richPlain 渲染一段普通文本：先抽 markdown 链接 [label](url)，再抽裸 URL，
// 剩下的按纯文本输出。经典表情 [名称] 保持线格式（网页端渲染 gif）。
func (a *app) richPlain(c *ui.Context, text string, mine bool) {
	linkFg := theme.Primary
	if mine {
		linkFg = selfBubbleLink() // 浅色气泡上用更深的蓝，保证可读
	}
	rest := text
	for rest != "" {
		label, url, before, after, ok := nextMarkdownLink(rest)
		if !ok {
			emitBareURLs(c, rest, linkFg)
			return
		}
		emitBareURLs(c, before, linkFg)
		ui.Link(c, label, url).TextColor(linkFg)
		rest = after
	}
}

// nextMarkdownLink 找第一个 [label](http...) 形式的链接。
func nextMarkdownLink(s string) (label, url, before, after string, ok bool) {
	open := strings.Index(s, "[")
	for open >= 0 {
		close1 := strings.Index(s[open:], "]")
		if close1 < 0 {
			break
		}
		close1 += open
		if close1+1 < len(s) && s[close1+1] == '(' {
			end := strings.Index(s[close1+2:], ")")
			if end >= 0 {
				u := s[close1+2 : close1+2+end]
				if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
					return s[open+1 : close1], u, s[:open], s[close1+2+end+1:], true
				}
			}
		}
		open = strings.Index(s[close1+1:], "[")
		if open >= 0 {
			open += close1 + 1
		}
	}
	return "", "", "", s, false
}

// emitBareURLs 把一段文本按裸 URL 切开输出，URL 变可点链接。
func emitBareURLs(c *ui.Context, text string, linkFg ui.Color) {
	for text != "" {
		i, j := nextURL(text)
		if i < 0 {
			ui.Text(c, text)
			return
		}
		if i > 0 {
			ui.Text(c, text[:i])
		}
		ui.Link(c, text[i:j], text[i:j]).TextColor(linkFg)
		text = text[j:]
	}
}

// nextURL 找第一个 http(s):// 且后面没有空白的区间；返回 [i,j) 或 -1。
func nextURL(s string) (int, int) {
	for _, scheme := range []string{"https://", "http://"} {
		i := 0
		for {
			k := strings.Index(s[i:], scheme)
			if k < 0 {
				break
			}
			i += k
			// 链接起点前不能是字母（避免把 "xhttp://" 当 URL）
			if i > 0 && isWordByte(s[i-1]) {
				i += len(scheme)
				continue
			}
			j := i + len(scheme)
			for j < len(s) {
				// 按 rune 前进：URL 边界含中文标点，按字节切会切碎多字节字符
				r, size := utf8.DecodeRuneInString(s[j:])
				if strings.ContainsRune(" \t\n\r，。；！？、）(！", r) {
					break
				}
				j += size
			}
			// 去尾部标点（同样按 rune）
			for j > i+len(scheme) {
				r, size := utf8.DecodeLastRuneInString(s[i:j])
				if !strings.ContainsRune(".,:;!?，。；！？", r) {
					break
				}
				j -= size
			}
			return i, j
		}
	}
	return -1, -1
}

func isWordByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}

// systemPill 是 system 类型消息的居中灰胶囊（对齐 SystemMessage.vue）：
// 无头像、无气泡，居中一行浅灰底小字。
func (a *app) systemPill(c *ui.Context, m message) {
	ui.Column(c).FillWidth().AlignItems(ui.Center).Padding(6, 0).Children(func() {
		ui.Box(c).Radius(12).Padding(4, 12).Background(theme.Gray100).Children(func() {
			ui.Text(c, m.Text).FontSize(12).TextColor(theme.Gray500).LineHeight(1.4)
		})
	})
}

// imageBubble 显示已解码位图；加载中/失败显示占位块，点占位块可重试。
// 点击图片本身走「打开 / 下载」，用系统默认程序查看原图。
func (a *app) imageBubble(c *ui.Context, m message) {
	src, errText, loading := a.imageState(m.Att, m.LocalPath)
	key := a.mediaKey(m.Att, m.LocalPath)

	// 原版容器 14px 圆角 + 淡阴影（box-shadow 0 6px 18px rgba(15,23,42,.06)），
	// 让图片卡片和白底消息区之间有清晰边界。
	frame := ui.Box(c).Radius(14).Background(theme.White).MaxWidth(imageMaxW+6).
		BorderWidth(1).BorderColor(theme.Gray200).Padding(3).
		Shadow(0, 6, 18, 0, ui.Hex("#0f172a10"))
	if frame.Clicked() {
		if errText != "" {
			a.invalidateMedia(key)
		} else if m.LocalPath == "" && !m.Att.IsZero() {
			a.openAttachment(m.Att)
		}
	}
	frame.Children(func() {
		if src != nil {
			w, h := ui.ImageSize(src)
			w, h = fitImageSize(w, h)
			ui.Image(c, src).Size(w, h).Radius(8)
			return
		}
		label := "图片加载中…"
		if errText != "" {
			label = "图片加载失败 · 点击重试"
		} else if !loading {
			label = "图片"
		}
		if errText != "" {
			label += "（" + shortErr(errText, 22) + "）"
		}
		holder := ui.Box(c).Width(200).Height(110).Radius(8).Background(theme.Gray100).Center()
		if holder.Clicked() {
			if errText != "" {
				a.invalidateMedia(key)
			} else if m.LocalPath == "" && !m.Att.IsZero() {
				a.openAttachment(m.Att)
			}
		}
		holder.Children(func() {
			ui.Text(c, label).FontSize(11).TextColor(theme.Gray500).Padding(0, 8)
		})
	})
	a.attachMessageMenu(frame, m)
}

// fitImageSize 等比缩到 imageMaxW×imageMaxH 内；小图最多放大 3 倍，免得 32×32 的
// 验证码缩略图在聊天区里几乎看不见。
func fitImageSize(w, h float32) (float32, float32) {
	if w <= 0 || h <= 0 {
		return 160, 110
	}
	scale := imageMaxW / w
	if s := imageMaxH / h; s < scale {
		scale = s
	}
	if scale > 3 {
		scale = 3
	}
	return w * scale, h * scale
}

// fileBubble 是附件卡片：类型图标 + 文件名 + 大小/状态，点击下载或打开。
func (a *app) fileBubble(c *ui.Context, m message) {
	att := m.Att
	path, busy := a.downloadState(att, m.LocalPath)
	bg, fg := theme.White, theme.Gray800
	if m.Mine {
		bg, fg = theme.PrimaryBg, theme.Gray800
	}
	name := att.DisplayName()
	sub := att.KindLabel()
	if size := qapi.FormatSize(att.Size); size != "" {
		sub += " · " + size
	}
	switch {
	case busy:
		sub += " · 下载中…"
	case path != "":
		sub += " · 点击打开"
	default:
		sub += " · 点击下载"
	}

	card := ui.Box(c).Width(258).Height(68).Radius(10).Background(bg).
		BorderWidth(1).BorderColor(theme.Gray200).Padding(0, 12)
	if card.Clicked() && m.LocalPath == "" {
		a.openAttachment(att)
	}
	card.Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Width(38).Height(38).Radius(8).Background(theme.Info).Center().Children(func() {
				ui.Text(c, iconLabel(m.Kind, att)).FontSize(11).TextColor(theme.White)
			})
			ui.Column(c).Grow(1).Gap(3).Children(func() {
				ui.Text(c, name).FontSize(13).TextColor(fg).MaxLines(1).Ellipsis("…")
				ui.Text(c, sub).FontSize(11).TextColor(theme.Gray500).MaxLines(1).Ellipsis("…")
			})
		})
	})
	a.attachMessageMenu(card, m)
}

func iconLabel(kind string, att qapi.Attachment) string {
	switch kind {
	case "video":
		return "视频"
	case "audio":
		return "语音"
	}
	label := att.KindLabel()
	// 三个字符以内（PDF / zip / 图片）原样放，中文长标签再截成两字，
	// 免得 38px 的图标块里挤成一团。
	if r := []rune(label); len(r) > 3 {
		return string(r[:2])
	}
	return label
}

func shortErr(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// mergedCard 是 merged_forward 消息的卡片（对齐 MergedForwardMessage.vue）：
// 头部「💬 聊天记录（N条）+ 查看聊天记录」，下面列前 3 条「发送者：预览」。
// 点头部打开记录弹窗看全部。
func (a *app) mergedCard(c *ui.Context, m message) {
	mf := m.MF
	bg, fg := theme.White, theme.Gray800
	if m.Mine {
		bg, fg = theme.PrimaryBg, theme.Gray800
	}
	card := ui.Box(c).Width(320).Radius(12).Background(bg).
		BorderWidth(1).BorderColor(theme.Gray200).Padding(12, 14)
	card.Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "💬").FontSize(15)
			ui.Text(c, mf.Title+"（"+strconv.Itoa(len(mf.Messages))+"条）").
				FontSize(13).FontWeight(600).TextColor(fg).MaxLines(1).Ellipsis("…")
			ui.Spacer(c)
			view := ui.Row(c).Gap(3).AlignItems(ui.Center)
			if view.Clicked() {
				a.openMergedRecord(mf)
			}
			view.Children(func() {
				ui.Text(c, "查看聊天记录").FontSize(11).TextColor(theme.Primary)
				ui.Text(c, "›").FontSize(11).TextColor(theme.Primary)
			})
		})
		ui.Column(c).FillWidth().Gap(4).Padding(8, 0, 0, 0).Children(func() {
			n := len(mf.Messages)
			if n > 3 {
				n = 3
			}
			for _, it := range mf.Messages[:n] {
				ui.Row(c).FillWidth().Gap(6).Children(func() {
					ui.Text(c, it.SenderName).FontSize(12).FontWeight(500).TextColor(theme.Gray600).MaxLines(1)
					ui.Text(c, qapi.ItemPreview(it)).FontSize(12).TextColor(theme.Gray500).MaxLines(1).Ellipsis("…")
				})
			}
			if len(mf.Messages) > 3 {
				ui.Text(c, fmt.Sprintf("… 共 %d 条，点击查看", len(mf.Messages))).
					FontSize(11).TextColor(theme.Gray500).Padding(2, 0)
			}
		})
	})
	a.attachMessageMenu(card, m)
}

// shareCard 是 share 消息的卡片（对齐 ShareMessage.vue 的通用形态）：
// 类型徽标 + 名称 + 摘要，点击看原文。
func (a *app) shareCard(c *ui.Context, m message) {
	sh := m.SH
	bg, fg := theme.White, theme.Gray800
	if m.Mine {
		bg, fg = theme.PrimaryBg, theme.Gray800
	}
	card := ui.Box(c).Width(280).Radius(12).Background(bg).
		BorderWidth(1).BorderColor(theme.Gray200).Padding(12, 14)
	if card.Clicked() && sh.Original != "" {
		a.openShareOriginal(sh)
	}
	card.Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Width(34).Height(34).Radius(8).Background(theme.Info).Center().Children(func() {
				ui.Text(c, shareGlyph(sh.Type)).FontSize(14)
			})
			ui.Column(c).Grow(1).Gap(3).Children(func() {
				ui.Text(c, sh.Name).FontSize(13).FontWeight(600).TextColor(fg).MaxLines(1).Ellipsis("…")
				ui.Text(c, qapi.ShareKindLabel(sh.Type)+" · 点击查看原文").FontSize(11).TextColor(theme.Gray500).MaxLines(1).Ellipsis("…")
			})
		})
		if sh.Content != "" && sh.Content != sh.Name {
			ui.Text(c, sh.Content).FontSize(12).TextColor(theme.Gray600).MaxLines(2).Padding(8, 0, 0, 0)
		}
	})
	a.attachMessageMenu(card, m)
}

// shareGlyph 是分享类型的徽标符号。
func shareGlyph(t string) string {
	switch t {
	case "file":
		return "📄"
	case "note":
		return "📝"
	case "sticky":
		return "🗒"
	default:
		return "💬"
	}
}

// botCard 是 card 类型（bot 交互卡片）的只读展示（对齐 CardMessage）：
// 标题 + 正文摘要 + 按钮回显。按钮的实际点击提交属 bot 会话能力，
// 这里不发起交互；服务端已派生过 card_action_id 时回显已选状态。
func (a *app) botCard(c *ui.Context, m message) {
	cd := m.CD
	bg, fg := theme.White, theme.Gray800
	if m.Mine {
		bg, fg = theme.PrimaryBg, theme.Gray800
	}
	card := ui.Box(c).Width(300).Radius(12).Background(bg).
		BorderWidth(1).BorderColor(theme.Gray200).Padding(12, 14)
	card.Children(func() {
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			if cd.Title != "" {
				ui.Text(c, cd.Title).FontSize(13).FontWeight(600).TextColor(fg).MaxLines(2)
			}
			if cd.Text != "" && cd.Text != cd.Title {
				ui.Text(c, cd.Text).FontSize(12).TextColor(theme.Gray600).LineHeight(1.5)
			}
			for _, b := range cd.Buttons {
				chosen := b.ID != "" && b.ID == cd.ActionID
				btnBg, btnFg := theme.Primary50, theme.Primary
				if chosen {
					btnBg, btnFg = theme.SuccessBg, theme.SuccessTx
				}
				label := b.Text
				if chosen {
					label = "✓ " + label
				}
				ui.Box(c).FillWidth().Height(28).Radius(6).Background(btnBg).Center().Children(func() {
					ui.Text(c, label).FontSize(12).TextColor(btnFg).MaxLines(1).Ellipsis("…")
				})
			}
		})
	})
	a.attachMessageMenu(card, m)
}

// newsCard 是 news 类型（资讯卡）：图 + 标题 + 摘要，点击开链接。
func (a *app) newsCard(c *ui.Context, m message) {
	nw := m.NW
	bg := theme.White
	if m.Mine {
		bg = theme.PrimaryBg
	}
	card := ui.Box(c).Width(300).Radius(12).Background(bg).
		BorderWidth(1).BorderColor(theme.Gray200).Padding(12, 14)
	if card.Clicked() && nw.URL != "" {
		_ = openURL(nw.URL)
	}
	card.Children(func() {
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			ui.Text(c, "📰 "+nw.Title).FontSize(13).FontWeight(600).TextColor(theme.Gray800).MaxLines(2)
			if nw.Summary != "" && nw.Summary != nw.Title {
				ui.Text(c, nw.Summary).FontSize(12).TextColor(theme.Gray500).MaxLines(3).LineHeight(1.5)
			}
			if nw.URL != "" {
				ui.Text(c, nw.URL).FontSize(11).TextColor(theme.Primary).MaxLines(1).Ellipsis("…")
			}
		})
	})
	a.attachMessageMenu(card, m)
}

// miniAppCard 是 miniApp 类型（小程序分享卡）的展示（对齐 MiniAppMessage）：
// 小程序图标块 + 名称 + 描述。真正的打开走小程序 loader，桌面端暂不支持。
func (a *app) miniAppCard(c *ui.Context, m message) {
	mi := m.MI
	bg := theme.White
	if m.Mine {
		bg = theme.PrimaryBg
	}
	card := ui.Box(c).Width(280).Radius(12).Background(bg).
		BorderWidth(1).BorderColor(theme.Gray200).Padding(12, 14)
	card.Children(func() {
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Width(34).Height(34).Radius(8).Background(theme.Success).Center().Children(func() {
					ui.Text(c, "🧩").FontSize(14)
				})
				ui.Column(c).Grow(1).Gap(2).Children(func() {
					ui.Text(c, mi.Label()).FontSize(13).FontWeight(600).TextColor(theme.Gray800).MaxLines(1).Ellipsis("…")
					ui.Text(c, "小程序").FontSize(11).TextColor(theme.Gray500)
				})
			})
			if mi.Desc != "" {
				ui.Text(c, mi.Desc).FontSize(12).TextColor(theme.Gray600).MaxLines(2).LineHeight(1.5)
			}
		})
	})
	a.attachMessageMenu(card, m)
}
