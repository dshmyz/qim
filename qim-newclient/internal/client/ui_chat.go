// 聊天头部与消息列表渲染。
// 从 main.go 按职责拆出，逻辑未改动。
package client

import (
	"github.com/egoist/mygo/ui"
	"qim-newclient/internal/theme"
	"strconv"
)

// chatHeader 聊天区顶部，逐项对齐 qim-client ChatHeader.vue：
// 56px 通栏 / padding 0 20 / 白底 / 1px 底边框；左侧 header-info 是
// 48px 头像（带右下角标）+ gap 12 的文本列（名称 14/500 + 状态行），
// 右侧 ChatHeaderActions 是 gap 8 的 32px 图标按钮。
// 注意：header 挂在 mainArea 顶层，横跨消息列和群成员侧栏（ChatBody 才分栏）。
func (a *app) chatHeader(c *ui.Context) {
	conv := a.convs[a.active]
	ui.Row(c).FillWidth().Height(56).AlignItems(ui.Center).Padding(0, 20).
		Background(theme.White).BorderWidth(0, 0, 1, 0).BorderColor(theme.Gray200).Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			a.headerAvatar(c, conv)
			ui.Column(c).Gap(2).Children(func() {
				// header-name：名字可截断，bot 会话右侧跟「AI 机器人」紫色徽标。
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, conv.Name).FontSize(14).FontWeight(500).TextColor(theme.Gray800).
						MaxLines(1).Ellipsis("…")
					if conv.Kind == "bot" {
						ui.Box(c).Radius(9).Padding(2, 8).Background(theme.AIPurpleBg).
							BorderWidth(1).BorderColor(theme.AIPurpleBd).Children(func() {
							ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
								ui.Icon(c, iconBot).FontSize(10).TextColor(theme.AIPurple)
								ui.Text(c, "AI 机器人").FontSize(11).FontWeight(500).TextColor(theme.AIPurple)
							})
						})
					}
				})
				a.headerStatus(c, conv)
			})
		})
		// 只留一个 Spacer：之前尾部还有一个 Spacer，两个平分剩余空间，
		// 把右侧图标推到了窗口中间，离右边缘差了 100 多像素。
		dragSpacer(c)
		a.headerActions(c, conv)
	})
}

// headerAvatar 是 48px 圆头像 + 右下角标；单聊/bot 且拿得到对端 id 时
// 点击打开资料卡（对齐 ChatHeader.vue 的 avatar-wrapper.clickable）。
func (a *app) headerAvatar(c *ui.Context, conv conversation) {
	clickable := (conv.Kind == "single" || conv.Kind == "bot") && conv.PeerID > 0
	box := ui.Box(c).Width(48).Height(48)
	if clickable && box.Clicked() {
		a.openUserCard(strconv.FormatUint(uint64(conv.PeerID), 10), conv.Name, conv.Avatar)
	}
	box.Children(func() {
		// 对齐 Avatar.vue 的 :src：服务端给了头像就走图片缓存，
		// 没给（或还没下载完）退回和会话列表一致的彩色首字。
		a.avatar(c, conv.Name, conv.Avatar, avatarStyle{Size: 48, Font: 20, Bg: convColor(conv)})
		a.headerBadge(c, conv)
	})
}

// headerBadge 复刻 utils/user.ts 的 buildConversationBadge：
// group 蓝色群成员图标、discussion 绿色双气泡、bot 蓝色机器人、
// system 橙色齿轮、api 绿色插头；普通真人用在线状态点。角标直径约
// 头像的 35%（17px），外圈 2px 白环。
func (a *app) headerBadge(c *ui.Context, conv conversation) {
	var svg *ui.SVG
	var bg ui.Color
	switch {
	case conv.Kind == "group":
		svg, bg = iconUsers, theme.Primary
	case conv.Kind == "discussion":
		svg, bg = iconComments, theme.Online
	case conv.Kind == "bot" || conv.PeerType == "bot":
		svg, bg = iconBot, theme.Primary
	case conv.PeerType == "system":
		svg, bg = iconSettings, ui.Hex("#faad14")
	case conv.PeerType == "api":
		svg, bg = iconSettings, theme.Online
	}
	if svg == nil && conv.Status == "" {
		return
	}
	ui.Box(c).Width(17).Height(17).Radius(8.5).Background(theme.White).Absolute().
		Right(0).Bottom(0).Center().Children(func() {
		if svg != nil {
			ui.Box(c).Width(15).Height(15).Radius(7.5).Background(bg).Center().Children(func() {
				ui.Icon(c, svg).FontSize(9).TextColor(theme.White)
			})
			return
		}
		dot := theme.Gray300
		switch conv.Status {
		case "online":
			dot = theme.Online
		case "busy":
			dot = ui.Hex("#ff4d4f")
		}
		ui.Box(c).Width(13).Height(13).Radius(6.5).Background(dot)
	})
}

// headerStatus 是名称下的 .header-status：单聊显示对端 IP + 个性签名
// （后端在会话列表里摊平，没有就整行留空），群聊只在有公告时显示
// 一行浅底喇叭胶囊。原版没有「N 名成员」「单聊」这类自造副标题。
func (a *app) headerStatus(c *ui.Context, conv conversation) {
	if conv.Kind == "single" {
		if conv.IP == "" && conv.Signature == "" {
			return
		}
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			if conv.IP != "" {
				ui.Text(c, conv.IP).FontSize(11).TextColor(theme.Gray500).MaxLines(1)
			}
			if conv.Signature != "" {
				ui.Text(c, conv.Signature).FontSize(12).TextColor(theme.Gray500).
					MaxLines(1).Ellipsis("…").MaxWidth(360)
			}
		})
		return
	}
	if conv.IsGroupish() && conv.Announcement != "" {
		ui.Box(c).Radius(4).Background(theme.Gray100).Padding(2, 6).Children(func() {
			ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconVolume).FontSize(11).TextColor(theme.Gray500)
				ui.Text(c, conv.Announcement).FontSize(12).TextColor(theme.Gray500).
					MaxLines(1).Ellipsis("…").MaxWidth(360)
			})
		})
	}
}

// headerActions 复刻 ChatHeaderActions.vue：群文件仅 group（讨论组没有）、
// 邀请成员和「更多」仅 group/discussion；单聊的更多按钮依赖 AI 分身开关，
// AI 相关功能按约定搁置，故单聊不显示操作区。
func (a *app) headerActions(c *ui.Context, conv conversation) {
	if !conv.IsGroupish() {
		return
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		if conv.Kind == "group" {
			a.headerIcon(c, iconFolder, a.openFiles)
		}
		a.headerIcon(c, iconUserPlus, a.openInviteMembers)
		more := ui.Box(c).Width(32).Height(32).Radius(6).Center()
		if more.Hovered() {
			more = more.Background(theme.Primary50)
		}
		more = more.PopoverMenuButton(func(m *ui.Menu) {
			if m.Item("修改群名称").Chosen() {
				a.openGroupNameEditor()
			}
			if m.Item("编辑群公告").Chosen() {
				a.openAnnouncementEditor()
			}
			if a.isGroupOwner() {
				m.Separator()
				if m.Item("解散群聊").Chosen() {
					a.openDissolveConfirm()
				}
			}
		})
		more.Children(func() {
			ui.Icon(c, iconMore).FontSize(14).TextColor(theme.Gray600)
		})
	})
}

// headerIcon 是 .header-icon：32×32、圆角 6、14px 图标、hover 浅主色底。
func (a *app) headerIcon(c *ui.Context, svg *ui.SVG, onClick func()) {
	btn := ui.Box(c).Width(32).Height(32).Radius(6).Center()
	if btn.Hovered() {
		btn = btn.Background(theme.Primary50)
	}
	if btn.Clicked() {
		onClick()
	}
	btn.Children(func() {
		ui.Icon(c, svg).FontSize(14).TextColor(theme.Gray600)
	})
}

// isGroupOwner 判断当前用户是不是本群群主（原版「解散群聊」仅群主可见）。
func (a *app) isGroupOwner() bool {
	if a.active < 0 || a.active >= len(a.convs) {
		return false
	}
	conv := a.convs[a.active]
	selfUID, err := strconv.Atoi(a.selfID)
	if err != nil {
		return false
	}
	for _, m := range conv.Members {
		if m.UserID == uint(selfUID) {
			return m.Role == "owner"
		}
	}
	return false
}

// messages 消息列表，对齐 qim-client MessageItem.vue / TimeDivider.vue：
// 40px 圆头像（两侧都有）+ 群聊才显示发送者名 + 气泡 + 气泡**下方**的
// .message-meta（时间 hover 才显，发送状态常显）。
func (a *app) messages(c *ui.Context) {
	if len(a.msgs) == 0 {
		hint := "暂无消息，发一条试试"
		if a.loadingMsgs {
			hint = "正在加载消息…"
		}
		ui.Column(c).Grow(1).FillWidth().Center().Children(func() {
			ui.Text(c, hint).FontSize(13).TextColor(theme.Gray500)
		})
		return
	}
	conv := a.currentConv()
	groupish := conv.Kind == "group" || conv.Kind == "discussion"
	// 列表外套一层 Column，顶部放「加载更早」按钮。
	// mygo 的 ListState 没有滚动到顶回调，所以用显式按钮而不是无限滚动。
	// a.list.Key 让插入更早消息后仍停在原来的那一条上。
	ui.Column(c).Grow(1).FillWidth().Children(func() {
		a.olderButton(c)
		a.list.Key = func(i int) any { return a.msgs[i].Key }
		a.list.FollowEnd = true
		ui.List(c, &a.list, len(a.msgs), func(i int) {
			m := a.msgs[i]
			if m.Pill != "" {
				timeDivider(c, m.Pill)
				return
			}
			if m.Divider {
				unreadDivider(c)
				return
			}
			// system 消息行级拦截：居中灰胶囊，无头像无发送者无气泡
			// （对齐 qim-client SystemMessage.vue）。
			if m.Sys {
				a.systemPill(c, m)
				return
			}
			// 原版 .message-item{align-items:flex-start}：头像顶部对齐
			// 「昵称 + 气泡」这一列的顶部，不是垂直居中在气泡旁边。
			// 之前 Row 默认居中，图片这种高气泡会把头像压到正中间。
			row := ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Start)
			row.Children(func() {
				if a.selectMode {
					a.selectBox(c, m)
				}
				if m.Mine {
					ui.Spacer(c)
					ui.Column(c).MaxWidth(560).AlignItems(ui.End).Gap(4).Children(func() {
						quoteBlock(c, m)
						a.messageBubble(c, m)
						a.messageMeta(c, m, row.Hovered())
					})
					a.messageAvatar(c, a.userName, a.selfAvatar(), nil)
				} else {
					a.messageAvatar(c, m.Sender, m.Avatar, &m)
					ui.Column(c).MaxWidth(560).AlignItems(ui.Start).Gap(4).Children(func() {
						if groupish {
							ui.Row(c).Gap(6).AlignItems(ui.Center).Padding(0, 0, 0, 4).Children(func() {
								ui.Text(c, m.Sender).FontSize(12).TextColor(theme.Gray500)
								if m.AtMe {
									atMeBadge(c)
								}
							})
						} else if m.AtMe {
							atMeBadge(c)
						}
						quoteBlock(c, m)
						a.messageBubble(c, m)
						a.messageMeta(c, m, row.Hovered())
					})
					ui.Spacer(c)
				}
			})
			// 行和气泡都要挂：Selectable 文本自带编辑菜单，会抢走最内层的右键。
			a.attachMessageMenu(row, m)
		}).Grow(1).Padding(16, 20).Gap(16).Background(theme.White)
	})
}

// timeDivider 对齐 TimeDivider.vue：左右两条 1px 灰线 + 中间灰胶囊，
// gap 12 / 胶囊 padding 4x12 / radius 12 / 12px 次要色。
func timeDivider(c *ui.Context, label string) {
	ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Grow(1).Height(1).Background(theme.Gray200)
		ui.Box(c).Radius(12).Padding(4, 12).Background(theme.Gray200).Children(func() {
			ui.Text(c, label).FontSize(12).TextColor(theme.Gray500)
		})
		ui.Box(c).Grow(1).Height(1).Background(theme.Gray200)
	})
}

// messageAvatar 是 40px 圆头像：服务端给了 URL 就走图片缓存，
// 否则退回按名字生成的彩色首字（对齐 shared/Avatar.vue 的 src 回退）。
// m 非 nil 时（对方消息）挂右键「快速 @」菜单，对齐 qim-client
// MessageItem.vue 的 handleAvatarContextMenu → AvatarQuickMenu。
func (a *app) messageAvatar(c *ui.Context, name, avatar string, m *message) {
	box := ui.Box(c).Width(40).Height(40).Shrink(0)
	if m != nil && a.currentConv().IsGroupish() && !m.Mine && m.SenderID != "" {
		box.ContextMenu(func(menu *ui.Menu) {
			if menu.Item("快速 @").Chosen() {
				uid, _ := strconv.ParseUint(m.SenderID, 10, 64)
				a.insertMention(member{UserID: uint(uid), Name: name, Avatar: avatar})
			}
		})
	}
	box.Children(func() {
		a.avatar(c, name, avatar, avatarStyle{Size: 40, Font: 15, Border: true, Shrink: true})
	})
}

// selfAvatar 是自己消息头像的 URL（登录资料里的 avatar）。
func (a *app) selfAvatar() string {
	if a.me != nil {
		return a.me.Avatar
	}
	return ""
}

// messageMeta 对齐 .message-meta：在气泡下方，时间只在整行 hover 时
// 显现（用 Opacity 而不是条件渲染，headless 也能验证文本存在），
// 发送中 / 发送失败（带重试）/ 已读常显，自己消息靠父列右对齐。
// 原版样式：gap 8、margin-top 4、只 padding-left 10（右对齐时贴气泡右缘）；
// 时间 xxxs=11px、已读 tiny=10px 且用次要文字色 #999（绿色只用于失败态）。
func (a *app) messageMeta(c *ui.Context, m message, hovered bool) {
	status, statusColor := "", theme.Muted // #999，对齐 .message-read-status
	switch {
	case m.Failed:
		status, statusColor = "发送失败", theme.Danger // #f56c6c
	case m.Pending:
		status = "发送中"
	case m.Read:
		status, statusColor = "已读", ui.Hex("#4caf50") // .read 绿
	}
	if m.At == "" && status == "" {
		return
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(0, 0, 0, 10).Children(func() {
		if m.At != "" {
			op := float32(0)
			if hovered {
				op = 0.6
			}
			ui.Text(c, m.At).FontSize(11).TextColor(theme.Gray800).Opacity(op)
		}
		if status == "" {
			return
		}
		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			ui.Text(c, status).FontSize(10).TextColor(statusColor).Opacity(0.8)
			if m.Failed {
				retry := ui.Box(c).Height(18).Radius(3).Padding(0, 6).
					Background(ui.Hex("#fef0f0")).Center()
				if retry.Clicked() {
					a.retryFailed(m.Key)
				}
				retry.Children(func() {
					ui.Text(c, "重试").FontSize(10).TextColor(theme.Danger)
				})
			}
		})
	})
}

// quoteBlock 对齐 .quoted-message-preview：浅底 + 左侧 4px 主色竖条，
// 第一行加粗发送者名，第二行是缩进的引用摘要（最多两行）。
func quoteBlock(c *ui.Context, m message) {
	if m.QuoteText == "" {
		return
	}
	ui.Box(c).MaxWidth(560).Radius(4).Padding(10).
		Background(theme.Gray100).BorderWidth(0, 0, 0, 4).BorderColor(theme.Primary).
		Children(func() {
			if m.QuoteSender != "" {
				ui.Text(c, m.QuoteSender).FontSize(12).FontWeight(600).
					TextColor(theme.Gray800).Padding(0, 0, 5, 0)
			}
			// 原版引用内容左侧还有一条 2px、30% 透明度的主色细线
			// （.quoted-message-preview-content::before），这里用左边框近似。
			ui.Text(c, m.QuoteText).FontSize(12).TextColor(theme.Gray600).
				MaxLines(2).Ellipsis("…").Padding(0, 0, 0, 10).
				BorderWidth(0, 0, 0, 2).BorderColor(ui.Hex("#3385ff4d"))
		})
}

func atMeBadge(c *ui.Context) {
	ui.Box(c).Radius(4).Padding(1, 5).Background(ui.Hex("#ffeded")).Children(func() {
		ui.Text(c, "@我").FontSize(10).TextColor(ui.Hex("#d64545"))
	})
}

func unreadDivider(c *ui.Context) {
	line := ui.Hex("#f3c9c9")
	ui.Row(c).FillWidth().Gap(10).Padding(2, 0).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Grow(1).Height(1).Background(line)
		ui.Text(c, "以下为新消息").FontSize(11).TextColor(ui.Hex("#d64545"))
		ui.Box(c).Grow(1).Height(1).Background(line)
	})
}

func (a *app) selectBox(c *ui.Context, m message) {
	on := a.selected[m.Key]
	bg := theme.White
	if on {
		bg = theme.Primary
	}
	box := ui.Box(c).Width(18).Height(18).Radius(4).
		BorderWidth(1).BorderColor(theme.Primary).Background(bg).Center()
	if box.Clicked() {
		a.toggleSelected(m.Key)
	}
	box.Children(func() {
		if on {
			ui.Text(c, "✓").FontSize(11).TextColor(theme.White)
		}
	})
}

// olderButton 是消息列表顶部的翻页入口。已翻到头就整个隐藏，
// 不留下一个点了没反应的按钮。
func (a *app) olderButton(c *ui.Context) {
	if !a.hasOlder && !a.loadingOlder {
		return
	}
	label := "加载更早消息"
	if a.loadingOlder {
		label = "正在加载…"
	}
	ui.Row(c).FillWidth().Height(34).Center().Children(func() {
		btn := ui.Box(c).Radius(11).Padding(4, 12).Background(theme.Gray100).Center()
		if btn.Clicked() && !a.loadingOlder {
			a.loadOlder()
		}
		btn.Children(func() {
			ui.Text(c, label).FontSize(11).TextColor(theme.Gray500)
		})
	})
}

// highlightBubble 处理两种气泡高亮：
//   - @我消息（m.AtMe）：对齐 qim-client `.message-item.at-mention
//     .message-bubble` 的浅主色底 + 3px 左主色边框，收到即高亮；
//   - 搜索 / @横幅跳转命中（key==a.highlight）：2px 主色全边框，临时定位用。
//
// 跳转高亮优先级更高（覆盖 @我），因为它是用户主动定位的那一条。
func (a *app) highlightBubble(e ui.Element, m message) ui.Element {
	if m.AtMe && !m.Mine {
		e = e.Background(theme.PrimaryBg).BorderWidth(0, 0, 0, 3).BorderColor(theme.Primary)
	}
	if m.Key != "" && m.Key == a.highlight {
		e = e.BorderWidth(2).BorderColor(theme.Primary)
	}
	return e
}

// 自己消息气泡对齐 qim-client .message-item.self .message-bubble：
// color-mix(primary, white 88%) → 浅主色底 + 深字，不是深蓝底白字。
// 网页端深色气泡只在 elegant-dark 主题下出现；发送失败也用同一底色，
// 状态只由气泡下方的红色 meta 表达（原版没有失败气泡灰化）。
func selfBubbleBG() ui.Color   { return theme.PrimaryBg }
func selfBubbleFG() ui.Color   { return theme.Gray800 }
func selfBubbleLink() ui.Color { return ui.Hex("#1d4ed8") }

// convColor 给没来得及填头像底色的会话一个稳定兜底。
// 搜索结果临时插入的会话、headless 测试构造的会话都可能没有 color，
// 而 ui.Hex("") 会直接 panic，所以渲染前统一过一遍。
func convColor(c conversation) string {
	if c.Color != "" {
		return c.Color
	}
	return avatarColor(c.ID + c.Name)
}

// currentConv 返回当前选中的会话。
func (a *app) currentConv() conversation {
	if a.active >= 0 && a.active < len(a.convs) {
		return a.convs[a.active]
	}
	return conversation{}
}
