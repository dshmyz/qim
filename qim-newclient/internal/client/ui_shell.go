// 整体布局：导航、侧栏、会话行、主区域。
// 从 main.go 按职责拆出，逻辑未改动。
package client

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"qim-newclient/internal/model"
	"qim-newclient/internal/qapi"
	"qim-newclient/internal/theme"
	"strings"
	"time"
)

func (a *app) view(c *ui.Context) {
	// OnPaste 回调签名里没有 Context，剪贴板读取服务在 build 时捕获。
	a.svc = c.Services()
	a.maybeTriggerMention()
	a.maybeTriggerSlash()
	// 原版结构：40px 全局拖动条（QIM + macOS 三色灯）在所有栏之上，
	// 下方未登录时是整幅登录页，登录后才是三栏。
	ui.Column(c).Fill().Children(func() {
		a.titleStrip(c)
		if !a.loggedIn {
			a.loginView(c)
			return
		}
		if a.settingsOpen {
			a.settingsView(c)
			return
		}
		if a.channelsOpen {
			a.channelsView(c)
			return
		}
		if a.filesOpen {
			a.filesView(c)
			return
		}
		ui.Row(c).Grow(1).FillWidth().Children(func() {
			a.navRail(c)
			a.sidebar(c)
			a.mainArea(c)
		})
		// 转发 / 合并转发目标选择浮层：Absolute 全屏覆盖三个栏。
		a.forwardPicker(c)
		a.announcementEditor(c)
		a.groupNameEditor(c)
		a.dissolveConfirm(c)
		a.convConfirm(c)
		a.readUsersPicker(c)
		a.notificationCenter(c)
		a.selfProfileModal(c)
		a.userCardOverlay(c)
		a.createGroupModal(c)
		a.emojiPanel(c)
		a.feedbackModal(c)
		a.updateDialog(c)
		a.mergedRecordModal(c)
		a.shareOriginalModal(c)
		// AI 助手：绝对定位在右侧和三栏之上，后绘制保证盖住聊天区。
		a.aiSidebar(c)
		a.aiBall(c)
	})
}

// dragSpacer 是顶栏/头部的空白拖拽区。
// 坑：ui.Spacer 本质上是空 Box（coreBox.Grow(1)），在 AlignItems(ui.Center)
// 的 Row 里交叉轴高度会被压成 0；命中判定 Rect.Contains 用 y < Y+H，
// 零高度区域永远收不到指针，DragWindow 于是形同虚设——表现就是
// “header 拖不动”。FillHeight 让它吃满父 Row 高度，拖拽区才真正可点。
func dragSpacer(c *ui.Context) ui.Element {
	return ui.Spacer(c).FillHeight().DragWindow()
}

// titleStrip 顶部 40px 拖动条：左侧 60px 放 QIM logo，右上角是
// macOS 三色窗口控制灯（16px 圆点、间距 12、距右 21，均取自原版实测）。
func (a *app) titleStrip(c *ui.Context) {
	// 原版顶栏下方没有可见横线（截图实测 y=40 整行纯白）。阴影只属于
	// 右上角 .window-controls，不能铺在整条顶栏上——那样会在左侧 360px
	// 多出一条灰线，且右侧没有，看起来像界面断裂。
	ui.Row(c).FillWidth().Height(40).AlignItems(ui.Center).Background(theme.White).Children(func() {
		ui.Box(c).Width(60).Height(40).Center().DragWindow().Children(func() {
			ui.Text(c, "QIM").FontWeight(600).FontSize(14).LetterSpacing(0.5).TextColor(theme.Gray800)
		})
		// 原版顶栏只有 logo + 拖动区 + 三色灯：连接状态与截图入口都是
		// 本端调试用的，已从这里移除（连接状态看设置页，截图走快捷键）。
		dragSpacer(c)
		// 三色灯：16px 圆点、间距 12、距右 21
		ui.Row(c).Gap(12).Padding(0, 21, 0, 0).AlignItems(ui.Center).Children(func() {
			a.lamp(c, theme.Amber, iconLampMinus, func() { a.win.Minimize() })
			a.lamp(c, theme.Green, iconLampSquare, func() { a.win.ToggleMaximize() })
			a.lamp(c, theme.Red, iconLampClose, func() { a.win.Close() })
		})
	})
}

// lamp 一个窗口控制圆点：16px 圆、白色小图形、150ms 过渡。
func (a *app) lamp(c *ui.Context, color ui.Color, icon *ui.SVG, onClick func()) {
	b := ui.Box(c).Width(16).Height(16).Radius(8).Background(color).Center()
	if b.Clicked() && onClick != nil {
		onClick()
	}
	b.Children(func() {
		ui.Icon(c, icon).FontSize(9).TextColor(theme.White)
	})
}

// navRail 最左侧垂直图标栏，对齐 SideOptions.vue：
// 宽 60、项 44×44 圆角 8、图标 18px、项间距 24（原版实测 pitch 68）、
// 首项距顶 22、底部留 16。
func (a *app) navRail(c *ui.Context) {
	ui.Column(c).Width(60).FillHeight().AlignItems(ui.Center).Gap(24).Padding(22, 0, 16, 0).
		BorderWidth(0, 1, 0, 0).BorderColor(theme.Gray50).Children(func() {
		a.navItem(c, iconChat, a.tabOrRecent() == "recent", func() { a.switchTab("recent") })
		a.navItem(c, iconContacts, a.tabOrRecent() == "org", func() { a.switchTab("org") })
		a.navItem(c, iconUsers, a.tabOrRecent() == "groups", func() { a.switchTab("groups") })
		a.navItem(c, iconApps, false, nil) // 应用页签：定位是应用/小程序，按需求搁置
		a.navItem(c, iconChannel, false, func() { a.openChannels() })
		a.navItem(c, iconFolder, false, func() { a.openFiles() })
		ui.Spacer(c)
		// 底部皮肤 + 设置，自带 gap 8
		ui.Column(c).Gap(8).Children(func() {
			a.navItem(c, iconPalette, false, nil)
			a.navItem(c, iconSettings, false, func() { a.settingsOpen = true })
		})
	})
}

// navItem 画一个导航图标项，active 时是主色 44×44 圆角块（对齐 .option-item.active）；
// 非 active 项 hover 时淡蓝底 + 图标变主色，带 150ms 颜色过渡。onClick 非空时点击触发。
func (a *app) navItem(c *ui.Context, icon *ui.SVG, active bool, onClick func()) {
	item := ui.Box(c).Width(44).Height(44).Radius(8).Center()
	col := theme.Gray500
	if active {
		item = item.Background(theme.Primary)
		col = theme.White
	} else if item.Hovered() {
		item = item.Background(theme.Primary50)
		col = theme.Primary
	}
	if item.Clicked() && onClick != nil {
		onClick()
	}
	item.Transition(ui.ElementTransition{Duration: 150 * time.Millisecond, Colors: true}).
		Children(func() {
			ui.Icon(c, icon).FontSize(18).TextColor(col)
		})
}

// tabOrRecent 取当前侧边栏页签，空串回落 recent。
func (a *app) tabOrRecent() string {
	if a.tab == "" {
		return "recent"
	}
	return a.tab
}

// switchTab 切换侧边栏页签；切走会话页时清掉搜索跳转高亮。
// 切到组织架构时确保组织树已加载（复用建群弹窗的缓存）。
func (a *app) switchTab(tab string) {
	a.mutate(func() {
		a.tab = tab
		a.highlight = ""
	})
	if tab == "org" {
		a.ensureOrgTree()
	}
}

// sidebarAction 是侧栏头部右上角的图标按钮，对齐 Sidebar.vue 的 .icon-btn：
// 28×28、圆角 6、14px 图标、文字色 gray-800、默认 opacity .7，
// 悬停不透明并铺 primary-50 底。badge 在图标之后画（通知未读角标）。
func (a *app) sidebarAction(c *ui.Context, svg *ui.SVG, onClick func(), badge func()) {
	btn := ui.Box(c).Width(28).Height(28).Radius(6).Center()
	if btn.Hovered() {
		btn = btn.Background(theme.Primary50).Opacity(1)
	} else {
		btn = btn.Opacity(0.7)
	}
	if btn.Clicked() && onClick != nil {
		onClick()
	}
	btn.Children(func() {
		ui.Icon(c, svg).FontSize(14).TextColor(theme.Gray800)
		if badge != nil {
			badge()
		}
	})
}

// sidebar 会话列表侧边栏，对齐 Sidebar.vue：宽 300、头部 56、搜索区 padding 8。
// 内容按 navRail 页签切换（recent 会话 / org 组织架构树 / groups 群聊列表）。
func (a *app) sidebar(c *ui.Context) {
	// 对齐 .sidebar：background var(--sidebar-bg)、box-shadow var(--shadow-sm)，
	// 没有右侧边框（分隔靠阴影）。
	ui.Column(c).Width(300).FillHeight().Shadow(0, 1, 3, 0, ui.Hex("#0000001a")).Children(func() {
		// 顶部：当前用户 + 通知 + 新建（高度 56、padding 0 20、分割线 6% 黑）
		ui.Row(c).FillWidth().Height(56).Padding(0, 20).Gap(12).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(ui.Hex("#0000000f")).Children(func() {
			// 自己头像：点击打开自我资料弹层（对齐 qim-client Sidebar 头像入口）。
			selfAv := ui.Box(c).Width(40).Height(40).Radius(20).Background(theme.Warning).Center()
			if selfAv.Clicked() {
				a.openSelfProfile()
			}
			selfAv.Children(func() {
				if a.me != nil && a.me.Avatar != "" {
					if src, _, _ := a.imageState(qapi.Attachment{URL: a.me.Avatar}, ""); src != nil {
						ui.Image(c, src).Size(40, 40).Radius(20)
						return
					}
				}
				ui.Text(c, model.FirstRune(a.displayName())).TextColor(theme.White).FontWeight(500).FontSize(14)
			})
			// 名字 flex:1 吃掉剩余宽度并省略，右侧按钮不收缩——对齐
			// Sidebar.vue 的 .user-name / .header-actions，长昵称不会把
			// 铃铛和「+」挤成一团。
			ui.Text(c, a.displayName()).Grow(1).FontWeight(500).FontSize(14).TextColor(theme.Gray800).
				MaxLines(1).Ellipsis("…")
			// header-actions：gap 8，两个 28×28 图标按钮。原版只有通知与「+」，
			// 没有退出按钮——退出登录在设置页账号区（对齐 settingsMenuItems）。
			ui.Row(c).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
				a.sidebarAction(c, iconBell, func() { a.openNotifs() }, func() {
					if a.unreadNotifs > 0 {
						label := fmt.Sprintf("%d", a.unreadNotifs)
						if a.unreadNotifs > 99 {
							label = "99+"
						}
						ui.Box(c).MinWidth(16).Height(16).Radius(8).Padding(0, 4).Background(theme.Danger).
							Absolute().Right(-4).Top(-4).Center().Children(func() {
							ui.Text(c, label).FontSize(10).TextColor(theme.White).FontWeight(600)
						})
					}
				})
				// 「+」对齐原版 actionMenu：左键弹菜单，不再直接建群。
				plus := ui.Box(c).Width(28).Height(28).Radius(6).Center()
				if plus.Hovered() {
					plus = plus.Background(theme.Primary50).Opacity(1)
				} else {
					plus = plus.Opacity(0.7)
				}
				plus = plus.PopoverMenuButton(func(m *ui.Menu) {
					if m.Item("创建群聊").Chosen() {
						a.openCreateGroup()
					}
					if m.Item("创建频道").Chosen() {
						a.openChannelCreate()
					}
				})
				plus.Children(func() {
					ui.Icon(c, iconPlus).FontSize(14).TextColor(theme.Gray800)
				})
			})
		})
		// 搜索框：外层 padding 8px，输入框 8/12、圆角 6、字号 13；
		// 占位文案与清空按钮对齐 Sidebar.vue（右侧 8px、✕ 显示条件为有关键词）。
		ph := "搜索用户、群组、聊天记录..."
		if a.tabOrRecent() != "recent" {
			ph = "搜索..."
		}
		ui.Row(c).FillWidth().Padding(8, 8).Shadow(0, 1, 2, 0, ui.Hex("#0000000d")).Children(func() {
			ui.Box(c).FillWidth().Children(func() {
				ti := ui.TextInput(c, &a.search).Placeholder(ph).FillWidth().FontSize(13).
					Radius(6).BorderWidth(1).BorderColor(theme.Gray200).Background(theme.White).Padding(8, 12)
				if ti.Focused() {
					ti = ti.BorderColor(theme.Primary)
				}
				if a.search != "" {
					clear := ui.Box(c).Width(18).Height(18).Center().Absolute().Right(8).Top(8)
					if clear.Clicked() {
						a.mutate(func() { a.search = "" })
					}
					clear.Children(func() {
						ui.Icon(c, iconClose).FontSize(12).TextColor(theme.Gray500)
					})
				}
			})
		})
		// 会话页签才发服务端搜索（带去抖与结果作废序号）。
		if a.tabOrRecent() == "recent" {
			a.maybeTriggerSearch()
			if strings.TrimSpace(a.search) != "" {
				a.searchResults(c)
				return
			}
		}
		switch a.tabOrRecent() {
		case "org":
			a.orgSidebar(c)
			return
		case "groups":
			a.groupsSidebar(c)
			return
		}
		// 列表为空时也给个明确状态，别让人以为界面卡死了。
		if len(a.convs) == 0 {
			ui.Column(c).Grow(1).FillWidth().Center().Children(func() {
				ui.Text(c, a.emptyListHint()).FontSize(13).TextColor(theme.Gray500)
			})
			return
		}
		// 会话列表：容器左右 8 + 行内 20（原版头像 x=88、角标右缘 332），
		// 顶部 7（搜索区自带 8 底距，原版首行头像 y=165）。
		// 用 ui.List 而不是手写 for：只构建可视行，滚轮 / 触控板 / 滚动条
		// 都能滚（会话一多就不会撑破侧栏底部）。Key 用会话 id，新消息把
		// 会话挪到上面时列表位置跟着条目走，不会跳。
		vis := a.visibleConvs()
		a.convList.Key = func(row int) any { return a.convs[vis[row]].ID }
		ui.Column(c).Grow(1).FillWidth().Padding(7, 8, 0, 8).Children(func() {
			ui.List(c, &a.convList, len(vis), func(row int) {
				a.convRow(c, vis[row])
			}).Grow(1).FillWidth()
		})
	})
}

// convRow 一条会话，对齐 ConversationList.vue：
// padding 12/20、gap 12、头像 40（真图 + 类型角标）、名称 14/500、
// 名称后成员数 12、预览 12（草稿走 warning 色）、时间 11、角标 18。
// 右侧 meta 是独立一列（免打扰图标 / 时间 / 未读角标自上而下）。
func (a *app) convRow(c *ui.Context, i int) {
	conv := a.convs[i]
	row := ui.Row(c).FillWidth().Gap(12).Padding(12, 20).AlignItems(ui.Center).Children(func() {
		// 头像：服务端有图就用真图，没有退化彩色字首；右下角标按会话类型/
		// 在线状态画（和聊天头部复用同一个 helper）。
		ui.Box(c).Width(40).Height(40).Children(func() {
			a.avatar(c, conv.Name, conv.Avatar, avatarStyle{Size: 40, Font: 15, Bg: convColor(conv)})
			a.headerBadge(c, conv)
		})
		// 名称 + 成员数 / 预览：1.6 行高（原版行 pitch 69 的来源）
		ui.Column(c).Grow(1).Gap(4).Children(func() {
			ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
				// 名称吃掉剩余宽度并省略，成员数不参与收缩——否则长群名
				// 会把「(2人)」挤成上下两行（截图反馈的乱源）。
				ui.Text(c, conv.Name).Grow(1).FontWeight(500).FontSize(14).LineHeight(1.6).
					TextColor(theme.Gray800).MaxLines(1).Ellipsis("…")
				if sub := convSubLabel(conv); sub != "" {
					ui.Text(c, sub).Shrink(0).TextColor(theme.Gray500).FontSize(12).LineHeight(1.6)
				}
			})
			// 预览：有草稿优先显示「[草稿] …」（原版 has-draft 用 warning 色 +
			// 铅笔图标），否则最后一条消息。原版没有「置顶」标签，靠排序体现。
			if draft := a.convDraft(conv.ID); draft != "" {
				ui.Row(c).FillWidth().Gap(4).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconPencil).FontSize(10).TextColor(theme.Warning)
					ui.Text(c, "[草稿] "+oneLine(draft)).FontSize(12).LineHeight(1.6).
						TextColor(theme.Warning).MaxLines(1).Ellipsis("…")
				})
			} else {
				ui.Text(c, conv.Preview).TextColor(theme.Gray500).FontSize(12).LineHeight(1.6).MaxLines(1).Ellipsis("…")
			}
		})
		// 免打扰图标 → 时间 → 未读角标（自上而下，右对齐）
		ui.Column(c).AlignItems(ui.End).Gap(4).Children(func() {
			if conv.Muted {
				ui.Icon(c, iconBellSlash).FontSize(12).TextColor(theme.Muted)
			}
			ui.Text(c, conv.Time).TextColor(theme.Gray500).FontSize(11)
			if conv.Unread > 0 {
				label := fmt.Sprintf("%d", conv.Unread)
				if conv.Unread > 99 {
					label = "99+"
				}
				ui.Box(c).Height(18).MinWidth(18).Radius(9).Padding(0, 5).Background(theme.Primary).Center().Children(func() {
					ui.Text(c, label).TextColor(theme.White).FontSize(10).FontWeight(600)
				})
			}
		})
	})
	if i == a.active {
		row = row.Background(theme.PrimaryBg)
	} else if row.Hovered() {
		row = row.Background(theme.Primary50)
	}
	row = row.Transition(ui.ElementTransition{Duration: 150 * time.Millisecond, Colors: true})
	// 右键菜单对齐 conversationMenuItems：置顶 / 免打扰 / 退出群聊 / 移除会话。
	// mygo 菜单没有 danger 样式，「移除会话」只能靠分隔线与文案体现。
	row.ContextMenu(func(menu *ui.Menu) {
		pinLabel := "置顶"
		if conv.Pinned {
			pinLabel = "取消置顶"
		}
		if menu.Item(pinLabel).Chosen() {
			a.togglePin(i)
		}
		muteLabel := "免打扰"
		if conv.Muted {
			muteLabel = "取消免打扰"
		}
		if menu.Item(muteLabel).Chosen() {
			a.toggleMute(i)
		}
		if conv.IsGroupish() {
			if menu.Item("退出群聊").Chosen() {
				a.askConvAction("exit", conv.ID, conv.Name)
			}
		}
		menu.Separator()
		if menu.Item("移除会话").Chosen() {
			a.askConvAction("remove", conv.ID, conv.Name)
		}
	})
	if row.Clicked() {
		a.highlight = "" // 手动切会话就撤掉搜索跳转的高亮
		a.selectConversation(i)
	}
}

// mainArea 右侧聊天区：头部 + 内容（占位或聊天窗口）。
// 聊天态 = 上半部（聊天列 + 成员面板）+ 底部全宽输入区，对齐 messages.png。
func (a *app) mainArea(c *ui.Context) {
	ui.Column(c).Grow(1).FillHeight().Children(func() {
		if a.active < 0 {
			a.panelHeader(c)
			a.placeholder(c)
			return
		}
		// 对齐 qim-client ChatWindow.vue：ChatHeader / AtMentionBanner 是
		// .chat-window 的直接子级，横跨「消息列 + 群成员侧栏」；只有 ChatBody
		// 才把两者分成左右两栏。之前把 header 塞进消息列，右侧操作图标会被
		// 成员侧栏挤到窗口中间，侧栏自身也顶到了窗口最上方。
		a.chatHeader(c)
		if a.notice != "" {
			a.noticeBar(c)
		}
		a.atMentionBanner(c)
		ui.Row(c).Grow(1).FillWidth().Children(func() {
			ui.Column(c).Grow(1).FillHeight().Children(func() {
				a.messages(c)
			})
			a.memberSidebar(c)
		})
		a.composer(c)
	})
}
