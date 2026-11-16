// 转发 / 已读 / 公告 / 群名 / 解散等弹层。
// 从 main.go 按职责拆出，逻辑未改动。
package client

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"qim-newclient/internal/qapi"
	"qim-newclient/internal/theme"
)

// forwardPicker 是转发 / 合并转发的目标会话选择浮层（对齐 qim-client 的 ShareModal）。
// 半透明遮罩 + 居中卡片，列出全部会话，点一列即转发到该会话。
// 用 Absolute 全屏覆盖，挂在 view 根的最后，保证盖在三个栏之上。
func (a *app) forwardPicker(c *ui.Context) {
	if !a.forwardOpen {
		return
	}
	n := len(a.convs)
	modalMask(c, func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Column(c).Width(360).MaxHeight(440).Radius(10).Background(theme.White).
				Padding(16, 16).Gap(10).AlignItems(ui.Start).Children(func() {
				ui.Text(c, a.forwardTitle).FontSize(14).TextColor(theme.Gray800).FontWeight(500)
				ui.Text(c, "选择目标会话").FontSize(12).TextColor(theme.Gray500)
				ui.Column(c).FillWidth().Grow(1).Children(func() {
					if n == 0 {
						ui.Column(c).FillWidth().Center().Children(func() {
							ui.Text(c, "没有可转发的会话").FontSize(12).TextColor(theme.Gray500)
						})
						return
					}
					a.forwardList.Key = func(i int) any { return a.convs[i].ID }
					ui.List(c, &a.forwardList, n, func(i int) {
						a.forwardRow(c, a.convs[i])
					})
				})
				cancel := ui.Box(c).Height(30).Radius(6).Padding(0, 14).Background(theme.Gray100).Center()
				if cancel.Clicked() {
					a.closeForward()
				}
				cancel.Children(func() {
					ui.Text(c, "取消").FontSize(12).TextColor(theme.Gray600)
				})
			})
		})
	})
}

// readUsersPicker 是已读详情弹层（对齐 qim-client 的 ReadUsersModal）：
// 列出已读用户（首字头像 + 名字），底部给「已读 x / 共 y 人」汇总，
// 未读数 = total_members - read_count。右键自己的消息「查看已读」打开。
func (a *app) readUsersPicker(c *ui.Context) {
	if !a.readUsersOpen {
		return
	}
	modalMask(c, func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Column(c).Width(340).MaxHeight(460).Radius(10).Background(theme.White).
				Padding(16, 16).Gap(10).AlignItems(ui.Start).Children(func() {
				ui.Text(c, "已读详情").FontSize(14).TextColor(theme.Gray800).FontWeight(500)
				switch {
				case a.readUsersBusy:
					ui.Column(c).FillWidth().Center().Padding(20, 0).Children(func() {
						ui.Text(c, "加载中…").FontSize(12).TextColor(theme.Gray500)
					})
				case a.readUsersInfo == nil:
					ui.Column(c).FillWidth().Center().Padding(20, 0).Children(func() {
						ui.Text(c, "没有已读数据").FontSize(12).TextColor(theme.Gray500)
					})
				default:
					info := a.readUsersInfo
					if len(info.ReadUsers) == 0 {
						ui.Column(c).FillWidth().Center().Padding(20, 0).Children(func() {
							ui.Text(c, "还没有人已读").FontSize(12).TextColor(theme.Gray500)
						})
					} else {
						ui.Column(c).FillWidth().Grow(1).Children(func() {
							for _, u := range info.ReadUsers {
								a.readUserRow(c, u)
							}
						})
					}
					unread := info.TotalMembers - info.ReadCount
					ui.Text(c, fmt.Sprintf("已读 %d 人 / 共 %d 人，未读 %d 人",
						info.ReadCount, info.TotalMembers, unread)).
						FontSize(12).TextColor(theme.Gray500)
				}
				cancel := ui.Box(c).Height(30).Radius(6).Padding(0, 14).Background(theme.Gray100).Center()
				if cancel.Clicked() {
					a.closeReadUsers()
				}
				cancel.Children(func() {
					ui.Text(c, "关闭").FontSize(12).TextColor(theme.Gray600)
				})
			})
		})
	})
}

// readUserRow 是已读列表里的一行用户：彩色首字头像 + 名字。
func (a *app) readUserRow(c *ui.Context, u qapi.ReadUser) {
	name := u.DisplayName()
	ui.Row(c).FillWidth().Padding(6, 4).Gap(8).AlignItems(ui.Center).Children(func() {
		a.avatar(c, name, "", avatarStyle{Size: 24, Font: 10, Weight: 400})
		ui.Text(c, name).FontSize(13).TextColor(theme.Gray800).MaxLines(1).Ellipsis("…")
	})
}

// announcementEditor 是群公告编辑弹层（对齐 qim-client 的编辑群公告对话框）：
// 遮罩 + 居中卡片 + TextArea 草稿，保存走 saveAnnouncement（乐观更新+失败回滚）。
func (a *app) announcementEditor(c *ui.Context) {
	if !a.annEditorOpen {
		return
	}
	modalMask(c, func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Column(c).Width(420).Radius(10).Background(theme.White).
				Padding(16, 16).Gap(10).AlignItems(ui.Start).Children(func() {
				ui.Text(c, "编辑群公告").FontSize(14).TextColor(theme.Gray800).FontWeight(500)
				ui.TextArea(c, &a.annDraft).Placeholder("输入群公告内容...").
					FillWidth().Height(120).FontSize(13)
				ui.Row(c).FillWidth().Gap(8).Children(func() {
					ui.Spacer(c)
					cancel := ui.Box(c).Height(30).Radius(6).Padding(0, 14).Background(theme.Gray100).Center()
					if cancel.Clicked() {
						a.closeAnnouncementEditor()
					}
					cancel.Children(func() {
						ui.Text(c, "取消").FontSize(12).TextColor(theme.Gray600)
					})
					save := ui.Box(c).Height(30).Radius(6).Padding(0, 14).Background(theme.Primary).Center()
					if save.Clicked() {
						a.saveAnnouncement()
					}
					save.Children(func() {
						ui.Text(c, "保存").FontSize(12).TextColor(theme.White)
					})
				})
			})
		})
	})
}

// groupNameEditor 是头部「更多 → 修改群名称」的弹层，复用公告编辑弹层版式。
func (a *app) groupNameEditor(c *ui.Context) {
	if !a.nameEditorOpen {
		return
	}
	modalMask(c, func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Column(c).Width(420).Radius(10).Background(theme.White).
				Padding(16, 16).Gap(12).AlignItems(ui.Start).Children(func() {
				ui.Text(c, "修改群名称").FontSize(14).TextColor(theme.Gray800).FontWeight(500)
				ui.TextInput(c, &a.nameDraft).Placeholder("输入群聊名称...").
					FillWidth().Height(34).FontSize(13).Radius(6).
					BorderWidth(1).BorderColor(theme.Gray200).Padding(0, 10)
				ui.Row(c).FillWidth().Gap(8).Children(func() {
					ui.Spacer(c)
					cancel := ui.Box(c).Height(30).Radius(6).Padding(0, 14).Background(theme.Gray100).Center()
					if cancel.Clicked() {
						a.closeGroupNameEditor()
					}
					cancel.Children(func() {
						ui.Text(c, "取消").FontSize(12).TextColor(theme.Gray600)
					})
					save := ui.Box(c).Height(30).Radius(6).Padding(0, 14).Background(theme.Primary).Center()
					if save.Clicked() {
						a.saveGroupName()
					}
					save.Children(func() {
						ui.Text(c, "保存").FontSize(12).TextColor(theme.White)
					})
				})
			})
		})
	})
}

// dissolveConfirm 是「解散群聊」的二次确认（对齐 qim-client ConfirmDialog）。
func (a *app) dissolveConfirm(c *ui.Context) {
	if !a.dissolveConfirmOpen {
		return
	}
	modalMask(c, func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Column(c).Width(380).Radius(10).Background(theme.White).
				Padding(16, 16).Gap(12).AlignItems(ui.Start).Children(func() {
				ui.Text(c, "确认解散群聊").FontSize(15).FontWeight(600).TextColor(theme.Gray800)
				ui.Text(c, "确定要解散此群聊吗？解散后所有消息和成员数据将被删除。").
					FontSize(13).TextColor(theme.Gray600).MaxLines(2).LineHeight(1.5)
				ui.Row(c).FillWidth().Gap(8).Children(func() {
					ui.Spacer(c)
					cancel := ui.Box(c).Height(30).Radius(6).Padding(0, 14).Background(theme.Gray100).Center()
					if cancel.Clicked() {
						a.closeDissolveConfirm()
					}
					cancel.Children(func() {
						ui.Text(c, "取消").FontSize(12).TextColor(theme.Gray600)
					})
					ok := ui.Box(c).Height(30).Radius(6).Padding(0, 14).Background(theme.DangerSolid).Center()
					if ok.Clicked() {
						a.dissolveGroup()
					}
					ok.Children(func() {
						ui.Text(c, "确定解散").FontSize(12).TextColor(theme.White)
					})
				})
			})
		})
	})
}

// convConfirm 是会话右键「退出群聊 / 移除会话」的二次确认（对齐
// qim-client 的 QMessageBox.confirm）：标题说明动作，正文带会话名，
// 确认按钮用危险色。
func (a *app) convConfirm(c *ui.Context) {
	if a.convConfirmKind == "" {
		return
	}
	title, body, okLabel := "移除会话", "确定要从列表中移除「"+a.convConfirmName+"」吗？只会从你的列表里隐藏，不影响其他人。", "移除"
	if a.convConfirmKind == "exit" {
		title, okLabel = "确认退出群聊", "退出"
		body = "确定要退出群聊「" + a.convConfirmName + "」吗？退出后将不再收到该群消息。"
	}
	modalMask(c, func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Column(c).Width(380).Radius(10).Background(theme.White).
				Padding(16, 16).Gap(12).AlignItems(ui.Start).Children(func() {
				ui.Text(c, title).FontSize(15).FontWeight(600).TextColor(theme.Gray800)
				ui.Text(c, body).FontSize(13).TextColor(theme.Gray600).MaxLines(3).LineHeight(1.5)
				ui.Row(c).FillWidth().Gap(8).Children(func() {
					ui.Spacer(c)
					cancel := ui.Box(c).Height(30).Radius(6).Padding(0, 14).Background(theme.Gray100).Center()
					if cancel.Clicked() {
						a.closeConvConfirm()
					}
					cancel.Children(func() {
						ui.Text(c, "取消").FontSize(12).TextColor(theme.Gray600)
					})
					ok := ui.Box(c).Height(30).Radius(6).Padding(0, 14).Background(theme.DangerSolid).Center()
					if ok.Clicked() {
						a.confirmConvAction()
					}
					ok.Children(func() {
						ui.Text(c, okLabel).FontSize(12).TextColor(theme.White)
					})
				})
			})
		})
	})
}

// forwardRow 是转发面板里一个目标会话：头像 + 名称，点击即转发并关闭。
func (a *app) forwardRow(c *ui.Context, conv conversation) {
	row := ui.Row(c).FillWidth().Padding(8, 10).Gap(8).AlignItems(ui.Center)
	if row.Hovered() {
		row = row.Background(theme.Primary50)
	}
	if row.Clicked() {
		a.doForward(conv.ID, conv.Name)
	}
	row.Children(func() {
		a.avatar(c, conv.Name, "", avatarStyle{Size: 28, Font: 12, Weight: 400})
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, conv.Name).FontSize(13).TextColor(theme.Gray800).MaxLines(1).Ellipsis("…")
			if lbl := kindLabel(conv.Kind); lbl != "" {
				ui.Text(c, lbl).FontSize(11).TextColor(theme.Gray500)
			}
		})
	})
}

// quoteBar 是输入框上方的引用条，发送后由 sendText 取走并清空。
func (a *app) quoteBar(c *ui.Context) {
	q := a.quote
	if q == nil {
		return
	}
	ui.Row(c).FillWidth().Padding(0, 15, 4, 15).Children(func() {
		ui.Row(c).FillWidth().Radius(6).Background(theme.Primary50).Padding(6, 10).Gap(8).
			AlignItems(ui.Center).Children(func() {
			ui.Box(c).Width(3).Height(28).Radius(1).Background(theme.Primary)
			ui.Column(c).Grow(1).Gap(2).Children(func() {
				ui.Text(c, "回复 "+q.Name).FontSize(11).TextColor(theme.Primary)
				ui.Text(c, q.Text).FontSize(12).TextColor(theme.Gray800).MaxLines(1).Ellipsis("…")
			})
			x := ui.Box(c).Width(20).Height(20).Radius(4).Center()
			if x.Clicked() {
				a.clearQuote()
			}
			x.Children(func() {
				ui.Icon(c, iconClose).FontSize(12).TextColor(theme.Gray500)
			})
		})
	})
}
