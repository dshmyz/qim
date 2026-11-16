// 频道页与频道动态。
// 从 main.go 按职责拆出，逻辑未改动。
package client

import (
	"github.com/egoist/mygo/ui"
	"qim-newclient/internal/qapi"
	"qim-newclient/internal/theme"
	"strings"
)

// channelsView 整屏频道视图：左栏频道列表（搜索 + 订阅/广场 + 创建），
// 右侧选中频道的消息卡片流（点赞/评论）与发布输入。
func (a *app) channelsView(c *ui.Context) {
	ui.Column(c).Fill().Background(theme.Gray50).Children(func() {
		// 头部：返回 + 标题 + 创建频道
		ui.Row(c).FillWidth().Height(56).Padding(0, 20).Gap(12).AlignItems(ui.Center).
			Background(theme.White).BorderWidth(0, 0, 1, 0).BorderColor(theme.Gray200).Children(func() {
			back := ui.Box(c).Height(32).Radius(6).Padding(0, 12).Center()
			if back.Clicked() {
				a.closeChannels()
			}
			back.Children(func() {
				ui.Text(c, "← 返回").FontSize(13).TextColor(theme.Primary)
			})
			ui.Text(c, "频道").FontWeight(600).FontSize(16).TextColor(theme.Gray800)
			ui.Spacer(c)
			btnHeaderAction(c, "+ 创建频道", false, a.openChannelCreate)
		})
		// 主体：左列表 + 右详情
		ui.Row(c).Grow(1).FillWidth().Children(func() {
			a.channelSidebar(c)
			a.channelDetail(c)
		})
		a.channelCreateModal(c)
	})
}

// channelSidebar 左栏：搜索框 + 订阅/广场 tabs + 频道列表。
func (a *app) channelSidebar(c *ui.Context) {
	ui.Column(c).Width(250).FillHeight().Background(theme.White).
		BorderWidth(0, 1, 0, 0).BorderColor(theme.Gray200).Children(func() {
		// 搜索框
		ui.Box(c).FillWidth().Padding(12, 12, 4, 12).Children(func() {
			ui.Box(c).FillWidth().Height(30).Radius(8).Background(theme.Gray100).Padding(0, 10).Children(func() {
				ui.TextArea(c, &a.channelSearch).Placeholder("搜索频道...").FillWidth().Height(30).MinHeight(30).
					FontSize(12).TextColor(theme.Gray800).Background(ui.Transparent)
			})
		})
		// tabs：订阅 / 广场
		unreadSubs := 0
		for _, ch := range a.channels {
			if ch.IsSubscribed {
				unreadSubs += ch.Unread
			}
		}
		ui.Row(c).FillWidth().Height(38).Padding(0, 12).Gap(16).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(theme.Gray200).Children(func() {
			for _, t := range []struct{ key, label string }{{"subscribed", "订阅"}, {"discover", "广场"}} {
				active := a.channelTab == t.key
				row := ui.Row(c).Height(38).Gap(5).AlignItems(ui.Center).Children(func() {
					col := theme.Gray500
					if active {
						col = theme.Primary
					}
					ui.Text(c, t.label).FontSize(13).FontWeight(600).TextColor(col)
					if t.key == "subscribed" && unreadSubs > 0 {
						ui.Box(c).Height(16).Radius(8).Padding(0, 5).Background(theme.Primary).
							Center().Children(func() {
							ui.Textf(c, "%d", unreadSubs).FontSize(10).TextColor(theme.White)
						})
					}
				})
				if row.Clicked() {
					a.setChannelTab(t.key)
				}
			}
		})
		// 列表
		if a.channelsLoaded {
			list := a.filteredChannels()
			if len(list) == 0 {
				hint := "暂无订阅的频道\n去广场看看"
				if a.channelTab == "discover" {
					hint = "暂无频道\n点右上角创建一个"
				}
				ui.Column(c).Grow(1).FillWidth().Center().Gap(10).Children(func() {
					ui.Icon(c, iconChannel).FontSize(40).TextColor(theme.Gray300)
					ui.Text(c, hint).FontSize(12).TextColor(theme.Gray500).TextAlign(ui.Center)
				})
			} else {
				ui.Column(c).Grow(1).FillWidth().Children(func() {
					for i := range list {
						a.channelRow(c, list[i])
					}
				})
			}
		} else {
			ui.Column(c).Grow(1).FillWidth().Center().Children(func() {
				ui.Text(c, "正在加载频道…").FontSize(13).TextColor(theme.Gray500)
			})
		}
	})
}

// channelRow 频道列表行：名称 + 未读徽标 + 消息数，选中高亮。
func (a *app) channelRow(c *ui.Context, ch qapi.Channel) {
	selected := a.channelSel >= 0 && a.channelSel < len(a.channels) &&
		a.channels[a.channelSel].ID.String() == ch.ID.String()
	bg := ui.Transparent
	if selected {
		bg = theme.Primary50
	}
	row := ui.Row(c).FillWidth().Height(44).Padding(0, 12).Gap(8).Background(bg).AlignItems(ui.Center)
	if row.Clicked() {
		if i := a.channelIndexByID(ch.ID.String()); i >= 0 {
			a.selectChannel(i)
		}
	}
	row.Children(func() {
		// 头像：首字色块
		name := ch.Name
		a.avatar(c, name, "", avatarStyle{Size: 28, Font: 12, Weight: 400, Radius: 6})
		ui.Column(c).Grow(1).Children(func() {
			ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, name).FontSize(13).FontWeight(600).TextColor(theme.Gray800).
					MaxLines(1).Ellipsis("…").Grow(1)
				if ch.Unread > 0 {
					ui.Box(c).Height(16).Radius(8).Padding(0, 5).Background(theme.Primary).
						Center().Children(func() {
						ui.Textf(c, "%d", ch.Unread).FontSize(10).TextColor(theme.White)
					})
				}
			})
			ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
				tag := "广播"
				if ch.PublishPermission != qapi.ChannelPublishCreatorOnly {
					tag = "协作"
				}
				ui.Text(c, tag).FontSize(10).TextColor(theme.Gray500)
				if ch.MessageCount > 0 {
					ui.Textf(c, "%d 条动态", ch.MessageCount).FontSize(10).TextColor(theme.Gray500)
				}
			})
		})
	})
}

// channelDetail 右侧详情：频道头 + 消息卡片流 + 发布输入/只读提示。
func (a *app) channelDetail(c *ui.Context) {
	ui.Column(c).Grow(1).FillHeight().Children(func() {
		ch := a.selChannel()
		if ch == nil {
			ui.Column(c).Grow(1).FillWidth().Center().Gap(14).Children(func() {
				ui.Icon(c, iconChannel).FontSize(64).TextColor(theme.Gray300)
				ui.Text(c, "选择一个频道").FontSize(14).TextColor(theme.Gray500)
			})
			return
		}
		if a.chMsgsLoading {
			ui.Column(c).Grow(1).FillWidth().Center().Children(func() {
				ui.Text(c, "正在加载消息…").FontSize(13).TextColor(theme.Gray500)
			})
			return
		}
		// 消息流
		ui.Column(c).Grow(1).FillWidth().Padding(16, 20).Gap(12).Children(func() {
			if len(a.chMsgs) == 0 {
				ui.Column(c).Grow(1).FillWidth().Center().Gap(10).Children(func() {
					ui.Text(c, "频道还没有动态").FontSize(13).TextColor(theme.Gray500)
				})
			}
			for i := range a.chMsgs {
				a.channelMsgCard(c, i)
			}
		})
		// 发布区：有权限显示输入框；订阅者无权限显示只读提示
		if a.canPublishIn(ch) {
			ui.Row(c).FillWidth().Height(56).Padding(0, 20).Gap(10).AlignItems(ui.Center).
				Background(theme.White).BorderWidth(1, 0, 0, 0).BorderColor(theme.Gray200).Children(func() {
				ui.Box(c).Grow(1).Height(36).Radius(18).Background(theme.Gray100).Padding(0, 14).Children(func() {
					ui.TextArea(c, &a.chInput).Placeholder("发布动态...").FillWidth().Height(36).MinHeight(36).
						FontSize(13).TextColor(theme.Gray800).Background(ui.Transparent)
				})
				send := ui.Box(c).Height(34).Radius(8).Padding(0, 16).Background(theme.Primary).Center()
				disabled := strings.TrimSpace(a.chInput) == "" || a.chBusy
				if send.Clicked() && !disabled {
					a.submitChannelMessage()
				}
				send.Children(func() {
					col := theme.White
					if disabled {
						col = theme.WhiteOverlay
					}
					ui.Text(c, "发布").FontSize(13).TextColor(col)
				})
			})
		} else if ch.IsSubscribed {
			ui.Row(c).FillWidth().Height(44).Center().Background(theme.White).
				BorderWidth(1, 0, 0, 0).BorderColor(theme.Gray200).Children(func() {
				ui.Text(c, "仅频道创建者可发布动态").FontSize(12).TextColor(theme.Gray500)
			})
		} else {
			ui.Row(c).FillWidth().Height(44).Center().Background(theme.White).
				BorderWidth(1, 0, 0, 0).BorderColor(theme.Gray200).Children(func() {
				ui.Text(c, "订阅该频道后可查看与互动").FontSize(12).TextColor(theme.Gray500)
			})
		}
	})
}

// channelMsgCard 一条频道消息卡片：作者、内容、时间、点赞与评论操作、评论展开区。
func (a *app) channelMsgCard(c *ui.Context, i int) {
	m := &a.chMsgs[i]
	sender := "未知用户"
	if m.Sender != nil {
		sender = m.Sender.DisplayName()
	}
	mine := m.SenderID.String() == a.selfID

	ui.Box(c).FillWidth().Radius(10).Background(theme.White).Padding(12, 14).Children(func() {
		// 头部：头像 + 作者 + 时间
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			a.avatar(c, sender, "", avatarStyle{Size: 30, Font: 12, Weight: 400})
			ui.Text(c, sender).FontSize(13).FontWeight(600).TextColor(theme.Gray800)
			if mine {
				ui.Box(c).Height(16).Radius(4).Padding(0, 5).Background(theme.Primary50).Center().Children(func() {
					ui.Text(c, "我").FontSize(10).TextColor(theme.Primary)
				})
			}
			ui.Text(c, chTimeLabel(m.CreatedAt)).FontSize(11).TextColor(theme.Gray500)
		})
		// 内容
		ui.Text(c, m.Content).FontSize(14).TextColor(theme.Gray800).FillWidth().Padding(0, 0, 0, 38)
		// 操作行：点赞 / 评论
		ui.Row(c).FillWidth().Padding(0, 0, 0, 38).Gap(14).AlignItems(ui.Center).Children(func() {
			likeBtn := ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
				col := theme.Gray500
				if m.Liked {
					col = ui.Hex("#e0245e")
				}
				ui.Text(c, "♥").FontSize(13).TextColor(col)
				ui.Textf(c, "%d", m.LikeCount).FontSize(12).TextColor(col)
			})
			if likeBtn.Clicked() {
				a.toggleChannelLike(i)
			}
			commentBtn := ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
				col := theme.Gray500
				if a.chCommentFor == m.ID.String() {
					col = theme.Primary
				}
				ui.Text(c, "💬").FontSize(12).TextColor(col)
				ui.Text(c, "评论").FontSize(12).TextColor(col)
			})
			if commentBtn.Clicked() {
				a.openChannelComments(i)
			}
		})
		// 评论展开区
		if a.chCommentFor == m.ID.String() {
			a.channelCommentsBlock(c, m)
		}
	})
}

// channelCommentsBlock 消息卡片下的评论列表与评论输入。
func (a *app) channelCommentsBlock(c *ui.Context, m *qapi.ChannelMessage) {
	ui.Column(c).FillWidth().Padding(0, 0, 0, 38).Gap(8).Children(func() {
		if a.chCommentsLoading {
			ui.Text(c, "正在加载评论…").FontSize(12).TextColor(theme.Gray500)
			return
		}
		if len(a.chComments) == 0 {
			ui.Text(c, "还没有评论").FontSize(12).TextColor(theme.Gray500)
		}
		for _, cm := range a.chComments {
			name := "未知用户"
			if cm.User != nil {
				name = cm.User.DisplayName()
			}
			ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
				a.avatar(c, name, "", avatarStyle{Size: 20, Font: 9, Weight: 400})
				ui.Text(c, name).FontSize(12).FontWeight(600).TextColor(theme.Gray800)
				ui.Text(c, cm.Content).FontSize(12).TextColor(theme.Gray800).Grow(1).MaxLines(3).Ellipsis("…")
				ui.Text(c, chTimeLabel(cm.CreatedAt)).FontSize(10).TextColor(theme.Gray500)
			})
		}
		// 评论输入（频道允许评论时）
		if a.selChannel() != nil && a.selChannel().CanComment() {
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Grow(1).Height(28).Radius(14).Background(theme.Gray100).Padding(0, 10).Children(func() {
					ui.TextArea(c, &a.chCommentInput).Placeholder("写下你的评论...").FillWidth().Height(28).MinHeight(28).
						FontSize(12).TextColor(theme.Gray800).Background(ui.Transparent)
				})
				send := ui.Box(c).Height(26).Radius(13).Padding(0, 12).Background(theme.Primary).Center()
				disabled := strings.TrimSpace(a.chCommentInput) == "" || a.chCommentBusy
				if send.Clicked() && !disabled {
					a.submitChannelComment()
				}
				send.Children(func() {
					ui.Text(c, "发送").FontSize(11).TextColor(theme.White)
				})
			})
		}
	})
}

// channelCreateModal 创建频道弹层：名称 / 描述（头像 URL 略——原生客户端用首字头像）。
func (a *app) channelCreateModal(c *ui.Context) {
	if !a.chCreateOpen {
		return
	}
	modalMask(c, func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Column(c).Width(420).Radius(10).Background(theme.White).Children(func() {
				ui.Row(c).FillWidth().Height(48).Padding(0, 16).AlignItems(ui.Center).
					BorderWidth(0, 0, 1, 0).BorderColor(theme.Gray200).Children(func() {
					ui.Text(c, "创建频道").FontWeight(600).FontSize(15).TextColor(theme.Gray800)
					ui.Spacer(c)
					x := ui.Box(c).Width(24).Height(24).Radius(4).Center()
					if x.Clicked() {
						a.closeChannelCreate()
					}
					x.Children(func() {
						ui.Icon(c, iconClose).FontSize(14).TextColor(theme.Gray500)
					})
				})
				ui.Column(c).FillWidth().Padding(16, 16).Gap(12).Children(func() {
					ui.Text(c, "频道名称 *").FontSize(12).TextColor(theme.Gray500)
					ui.Box(c).FillWidth().Height(34).Radius(8).Background(theme.Gray100).Padding(0, 10).Children(func() {
						ui.TextArea(c, &a.chName).Placeholder("输入频道名称").FillWidth().Height(34).MinHeight(34).
							FontSize(13).TextColor(theme.Gray800).Background(ui.Transparent)
					})
					ui.Text(c, "频道描述").FontSize(12).TextColor(theme.Gray500)
					ui.Box(c).FillWidth().Height(64).Radius(8).Background(theme.Gray100).Padding(6, 10).Children(func() {
						ui.TextArea(c, &a.chDesc).Placeholder("输入频道描述（可选）").FillWidth().Height(52).MinHeight(52).
							FontSize(13).TextColor(theme.Gray800).Background(ui.Transparent)
					})
					ui.Row(c).FillWidth().Height(44).Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Spacer(c)
						cancel := ui.Box(c).Height(32).Radius(8).Padding(0, 14).Background(theme.Gray100).Center()
						if cancel.Clicked() {
							a.closeChannelCreate()
						}
						cancel.Children(func() {
							ui.Text(c, "取消").FontSize(13).TextColor(theme.Gray800)
						})
						ok := ui.Box(c).Height(32).Radius(8).Padding(0, 14).Background(theme.Primary).Center()
						if ok.Clicked() && strings.TrimSpace(a.chName) != "" && !a.chBusy {
							a.submitChannelCreate()
						}
						ok.Children(func() {
							label := "创建"
							if a.chBusy {
								label = "创建中…"
							}
							ui.Text(c, label).FontSize(13).TextColor(theme.White)
						})
					})
				})
			})
		})
	})
}
