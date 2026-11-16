// 通知中心。
// 从 main.go 按职责拆出，逻辑未改动。
package client

import (
	"github.com/egoist/mygo/ui"
	"qim-newclient/internal/qapi"
	"qim-newclient/internal/theme"
)

// notifTabs 是通知中心的过滤 tab 定义（对齐 qim-client filterTabs）。
type notifTab struct {
	value string
	label string
}

func (a *app) notifTabCounts() map[string]int {
	all := len(a.notifs)
	unread, important := 0, 0
	for _, n := range a.notifs {
		if !n.Read {
			unread++
		}
		if n.Important {
			important++
		}
	}
	return map[string]int{"all": all, "unread": unread, "important": important}
}

// filteredNotifs 按 tab 过滤通知，置顶的排前面。
func (a *app) filteredNotifs() []qapi.Notification {
	out := make([]qapi.Notification, 0, len(a.notifs))
	for _, n := range a.notifs {
		switch a.notifsTab {
		case "unread":
			if n.Read {
				continue
			}
		case "important":
			if !n.Important {
				continue
			}
		}
		out = append(out, n)
	}
	// 稳定排序：置顶在前，其余保持时间倒序原样
	j := 0
	for i := range out {
		if out[i].Pinned {
			out[j], out[i] = out[i], out[j]
			j++
		}
	}
	return out
}

// notificationCenter 是通知中心面板（对齐 qim-client NotificationCenter.vue）：
// 半透明遮罩 + 居中卡片，头部一键已读/清空，tabs 全部/未读/重要，
// 列表分页加载，条目支持点击已读、置顶、重要、删除与动作按钮。
func (a *app) notificationCenter(c *ui.Context) {
	if !a.notifsOpen {
		return
	}
	counts := a.notifTabCounts()
	hasNotifs := len(a.notifs) > 0
	hasUnread := counts["unread"] > 0
	tabs := []notifTab{{"all", "全部"}, {"unread", "未读"}, {"important", "重要"}}

	modalMask(c, func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Column(c).Width(460).Height(560).Radius(10).Background(theme.White).
				Children(func() {
					// 头部：标题 + 一键已读 + 清空 + 关闭
					ui.Row(c).FillWidth().Height(48).Padding(0, 16).Gap(10).AlignItems(ui.Center).
						BorderWidth(0, 0, 1, 0).BorderColor(theme.Gray200).Children(func() {
						ui.Text(c, "通知中心").FontWeight(600).FontSize(15).TextColor(theme.Gray800)
						ui.Spacer(c)
						btnHeaderAction(c, "一键已读", !hasUnread, func() { a.markAllNotifsRead() })
						btnHeaderAction(c, "清空", !hasNotifs, func() { a.clearNotifs() })
						x := ui.Box(c).Width(24).Height(24).Radius(4).Center()
						if x.Clicked() {
							a.closeNotifs()
						}
						x.Children(func() {
							ui.Icon(c, iconClose).FontSize(14).TextColor(theme.Gray500)
						})
					})
					// tabs：全部 / 未读 / 重要
					ui.Row(c).FillWidth().Height(40).Padding(0, 16).Gap(18).AlignItems(ui.Center).
						BorderWidth(0, 0, 1, 0).BorderColor(theme.Gray200).Children(func() {
						for _, t := range tabs {
							active := a.notifsTab == t.value
							row := ui.Row(c).Height(40).Gap(5).AlignItems(ui.Center).Children(func() {
								col := theme.Gray500
								if active {
									col = theme.Primary
								}
								ui.Text(c, t.label).FontSize(13).FontWeight(600).TextColor(col)
								if t.value != "all" && counts[t.value] > 0 {
									ui.Box(c).Height(16).Radius(8).Padding(0, 5).Background(theme.Primary).
										Center().Children(func() {
										ui.Textf(c, "%d", counts[t.value]).FontSize(10).TextColor(theme.White)
									})
								}
							})
							if active {
								_ = row // mygo 无下划线修饰，主色文字即激活态
							}
							// 点击整块切 tab
							if row.Clicked() {
								a.setNotifsTab(t.value)
							}
						}
					})
					// 列表区
					if a.notifsBusy {
						ui.Column(c).Grow(1).FillWidth().Center().Children(func() {
							ui.Text(c, "正在加载通知…").FontSize(13).TextColor(theme.Gray500)
						})
						return
					}
					list := a.filteredNotifs()
					if len(list) == 0 {
						hint := "暂无通知"
						switch a.notifsTab {
						case "unread":
							hint = "暂无未读通知"
						case "important":
							hint = "暂无重要通知"
						}
						ui.Column(c).Grow(1).FillWidth().Center().Gap(12).Children(func() {
							ui.Icon(c, iconBell).FontSize(48).TextColor(theme.Gray300)
							ui.Text(c, hint).FontSize(13).TextColor(theme.Gray500)
						})
						return
					}
					ui.Column(c).Grow(1).FillWidth().Children(func() {
						a.notifList.Key = func(i int) any { return list[i].ID.String() }
						ui.List(c, &a.notifList, len(list), func(i int) {
							a.notifRow(c, list[i])
						})
						// 底部加载更多（不足一页 = 已经到头，隐藏）
						if a.notifsHasMore {
							ui.Row(c).FillWidth().Height(40).Center().Children(func() {
								btn := ui.Box(c).Radius(11).Padding(4, 14).Background(theme.Gray100).Center()
								if btn.Clicked() {
									go a.loadNotifsPage(a.notifsPage + 1)
								}
								btn.Children(func() {
									ui.Text(c, "加载更多").FontSize(12).TextColor(theme.Gray500)
								})
							})
						}
					})
				})
		})
	})
}

// btnHeaderAction 是通知中心头部的文字按钮（一键已读/清空），disabled 时置灰。
func btnHeaderAction(c *ui.Context, label string, disabled bool, onClick func()) {
	btn := ui.Box(c).Height(26).Radius(6).Padding(0, 10).Background(theme.Gray100).Center()
	col := theme.Gray800
	if disabled {
		col = theme.Gray300
	}
	if !disabled && btn.Clicked() {
		onClick()
	}
	btn.Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(col)
	})
}

// ---------- 频道视图（对齐 qim-client ChannelSidebar + ChannelDetailNew） ----------

// notifRow 是通知中心里的一条通知，对齐 qim-client notification-item：
// 未读蓝点 + 标题（重要星标）+ 内容两行 + 时间（已处理徽标），
// hover 时右侧浮出工具钮（置顶/重要/删除），底部是动作按钮行。
func (a *app) notifRow(c *ui.Context, n qapi.Notification) {
	row := ui.Column(c).FillWidth().Padding(10, 16, 10, 16).Gap(6).
		BorderWidth(0, 0, 1, 0).BorderColor(theme.Gray50)
	if row.Hovered() {
		row = row.Background(theme.Gray50)
	}
	row.Children(func() {
		// 第一行：未读点 + 标题 + 工具钮（工具钮始终渲染但未 hover 时淡色）
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			if !n.Read {
				ui.Box(c).Width(7).Height(7).Radius(3.5).Background(theme.Primary)
			}
			ui.Text(c, n.Title).FontSize(13).FontWeight(600).TextColor(theme.Gray800).
				MaxLines(1).Ellipsis("…")
			if n.Important {
				ui.Icon(c, iconStar).FontSize(12).TextColor(theme.Warning)
			}
			ui.Spacer(c)
			a.notifTool(c, iconPin, n.Pinned, func() { a.toggleNotifPin(n.ID.String()) })
			a.notifTool(c, iconStar, n.Important, func() { a.toggleNotifImportant(n.ID.String()) })
			a.notifTool(c, iconClose, false, func() { a.deleteNotif(n.ID.String()) })
		})
		// 内容
		if n.Content != "" {
			ui.Text(c, n.Content).FontSize(12).TextColor(theme.Gray500).MaxLines(2).LineHeight(1.5)
		}
		// 时间 + 已处理徽标
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, convTimeLabel(n.Timestamp)).FontSize(11).TextColor(theme.Gray500)
			if n.Handled {
				ui.Box(c).Padding(1, 6).Radius(3).Background(theme.Gray100).Children(func() {
					ui.Text(c, "已处理").FontSize(10).TextColor(theme.Gray500)
				})
			}
		})
		// 动作按钮（未处理且 action_type 有映射）
		if btns := n.ActionButtons(); len(btns) > 0 {
			ui.Row(c).FillWidth().Gap(8).Padding(2, 0, 0, 0).Children(func() {
				for _, b := range btns {
					btn := ui.Box(c).Height(26).Radius(6).Padding(0, 14).Center()
					primary := b.Value == "accept" || b.Value == "confirm" || b.Value == "approve" || b.Value == "confirm_create"
					if primary {
						btn = btn.Background(theme.Primary)
					} else {
						btn = btn.Background(theme.Gray100)
					}
					if btn.Clicked() {
						a.handleNotifAction(n.ID.String(), b.Value)
					}
					col := theme.Gray800
					if primary {
						col = theme.White
					}
					btn.Children(func() {
						ui.Text(c, b.Label).FontSize(12).TextColor(col)
					})
				}
			})
		}
	})
	// 点击条目 = 标已读
	if row.Clicked() {
		a.clickNotif(n.ID.String())
	}
}

// notifTool 是通知条目右侧的小工具钮（置顶/重要/删除），active 时主色。
func (a *app) notifTool(c *ui.Context, icon *ui.SVG, active bool, onClick func()) {
	btn := ui.Box(c).Width(22).Height(22).Radius(4).Center()
	col := theme.Gray300
	if active {
		col = theme.Primary
	} else if btn.Hovered() {
		col = theme.Gray600
		btn = btn.Background(theme.Gray100)
	}
	if btn.Clicked() {
		onClick()
	}
	btn.Children(func() {
		ui.Icon(c, icon).FontSize(13).TextColor(col)
	})
}
