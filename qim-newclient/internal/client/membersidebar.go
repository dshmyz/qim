package client

// 群成员侧栏，对齐 qim-client MemberSidebar.vue：
// 仅群聊/讨论组显示；180px 展开（头部折叠钮 + 搜索钮 + 搜索框 + 成员列表），
// 折叠后 48px 竖排头像栏；成员行有在线点、群主/管理徽标，
// 单击资料卡、双击发私聊、右键菜单（@提及 / 发私聊）。

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"qim-newclient/internal/qapi"
	"qim-newclient/internal/theme"
)

// memberSidebar 取代会话信息栏：原 convInfoPanel 的「类型/ID/未读」
// 诊断行 qim-client 并没有，移除后布局与原版一致（composer 仍通栏）。
func (a *app) memberSidebar(c *ui.Context) {
	conv := a.currentConv()
	// qim-client showMemberSidebar：仅 group / discussion。
	if conv.Kind != "group" && conv.Kind != "discussion" {
		return
	}
	if a.membersCollapsed {
		a.memberRail(c, conv)
		return
	}
	ui.Column(c).Width(180).FillHeight().Background(theme.White).
		Shadow(-1, 0, 2, 0, theme.Shadow).Children(func() {
		// 头部一行：收起钮 + 「群成员 (N)」+ 搜索钮。
		// 对齐 MemberSidebar.vue 的 .members-header（padding 8/12、h3 13/500）：
		// 之前把标题单独放第二行，顶部比原版多了一行。
		members := filterMembers(conv.Members, a.memberSearch)
		n := len(members)
		ui.Row(c).FillWidth().Padding(8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			btn := ui.Box(c).Width(26).Height(26).Radius(5).Center()
			if btn.Hovered() {
				btn = btn.Background(theme.Gray100)
			}
			if btn.Clicked() {
				a.mutate(func() { a.membersCollapsed = true })
			}
			btn.Children(func() {
				ui.Icon(c, iconChevLeft).FontSize(12).TextColor(theme.Gray500)
			})
			ui.Textf(c, "群成员 (%d)", n).FontSize(13).FontWeight(500).TextColor(theme.Gray800)
			ui.Spacer(c)
			sbtn := ui.Box(c).Width(26).Height(26).Radius(5).Center()
			sbg, sfg := theme.White, theme.Gray500
			if a.memberSearchOpen {
				sbg, sfg = theme.Primary50, theme.Primary
			}
			if sbtn.Hovered() {
				sbg = theme.Gray100
			}
			if sbtn.Clicked() {
				a.mutate(func() {
					a.memberSearchOpen = !a.memberSearchOpen
					if !a.memberSearchOpen {
						a.memberSearch = ""
					}
				})
			}
			sbtn = sbtn.Background(sbg)
			sbtn.Children(func() {
				ui.Icon(c, iconSearch).FontSize(12).TextColor(sfg)
			})
		})
		// 搜索框（展开时才可打开）
		if a.memberSearchOpen {
			ui.Row(c).FillWidth().Padding(6, 8).Children(func() {
				ui.TextInput(c, &a.memberSearch).Placeholder("搜索群成员...").FillWidth().FontSize(12).
					Radius(5).BorderWidth(1).BorderColor(theme.Gray200).Background(theme.White).Padding(6, 10)
			})
		}
		// 成员列表
		if n == 0 {
			ui.Column(c).Grow(1).FillWidth().Center().Children(func() {
				hint := "群成员加载中…"
				if strings.TrimSpace(a.memberSearch) != "" {
					hint = "无匹配成员"
				}
				ui.Text(c, hint).FontSize(12).TextColor(theme.Gray500)
			})
			return
		}
		a.memberList.Key = func(i int) any { return members[i].UserID }
		a.memberList.FollowEnd = false
		ui.List(c, &a.memberList, n, func(i int) {
			a.memberRow(c, conv.ID, members[i])
		})
	})
}

// memberRail 折叠态 48px：竖排成员头像（前 12 个），点击展开。
func (a *app) memberRail(c *ui.Context, conv conversation) {
	ui.Column(c).Width(48).FillHeight().AlignItems(ui.Center).Gap(4).Padding(6, 0, 8, 0).
		Background(theme.White).Shadow(-1, 0, 2, 0, theme.Shadow).Children(func() {
		btn := ui.Box(c).Width(26).Height(26).Radius(5).Center()
		if btn.Hovered() {
			btn = btn.Background(theme.Gray100)
		}
		if btn.Clicked() {
			a.mutate(func() { a.membersCollapsed = false })
		}
		btn.Children(func() {
			ui.Icon(c, iconChevRight).FontSize(12).TextColor(theme.Gray500)
		})
		for i, m := range conv.Members {
			if i >= 12 {
				break
			}
			av := ui.Box(c).Width(28).Height(28)
			if av.Clicked() {
				a.openUserCard(fmt.Sprintf("%d", m.UserID), m.Name, m.Avatar)
			}
			av.Children(func() {
				a.avatar(c, m.Name, m.Avatar, avatarStyle{Size: 28, Font: 11})
			})
		}
	})
}

// filterMembers 按搜索词过滤成员（昵称前缀/包含，大小写不敏感）。
func filterMembers(ms []member, q string) []member {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return ms
	}
	var out []member
	for _, m := range ms {
		if strings.Contains(strings.ToLower(m.Name), q) {
			out = append(out, m)
		}
	}
	return out
}

// memberRow 展开态成员行：头像(在线点) + 名字(角色徽标)，单击资料卡，
// 双击发私聊，右键 @提及 / 发私聊。
func (a *app) memberRow(c *ui.Context, convID string, m member) {
	uid := fmt.Sprintf("%d", m.UserID)
	row := ui.Row(c).FillWidth().Padding(6, 10).Gap(8).AlignItems(ui.Center)
	if row.Hovered() {
		row = row.Background(theme.Gray100)
	}
	if row.Clicked() {
		a.openUserCard(uid, m.Name, m.Avatar)
	}
	if row.DoubleClicked() {
		a.chatWithMember(m)
	}
	row.ContextMenu(func(menu *ui.Menu) {
		// 顺序对齐 qim-client MemberContextMenu.vue：查看资料 → @ 提及
		// → 发起私聊。设管理员/转让群主/移除群聊依赖后端群成员管理契约，
		// 当前 qapi 未提供，故不摆点了没反应的空项（也不留悬空分隔线）。
		if menu.Item("查看资料").Chosen() {
			a.openUserCard(uid, m.Name, m.Avatar)
		}
		if menu.Item("@ 提及").Chosen() {
			a.insertMention(m)
		}
		if menu.Item("发起私聊").Chosen() {
			a.chatWithMember(m)
		}
	})
	row.Children(func() {
		// 头像 + 右下角在线点（MemberSidebar member-avatar-wrapper）：
		// 原版头像 28px，状态点 8px + 1.5px 白边（这里用白底 8 / 内点 5 近似）。
		ui.Box(c).Width(28).Height(28).Children(func() {
			a.avatar(c, m.Name, m.Avatar, avatarStyle{Size: 28, Font: 11})
			if st := a.statusOf(uid); st != "" {
				dot := theme.Gray300
				if st == "online" {
					dot = theme.Online
				}
				ui.Box(c).Width(8).Height(8).Radius(4).Background(theme.White).Absolute().Right(0).Bottom(0).Center().Children(func() {
					ui.Box(c).Width(5).Height(5).Radius(2.5).Background(dot)
				})
			}
		})
		ui.Column(c).Grow(1).Gap(1).Children(func() {
			ui.Row(c).FillWidth().Gap(4).AlignItems(ui.Center).Children(func() {
				ui.Text(c, m.Name).FontSize(12).TextColor(theme.Gray800).MaxLines(1).Ellipsis("…")
				// 角色标签对齐 MemberSidebar.vue：纯色文字（群主金/管理蓝），无底色
				switch m.Role {
				case "owner":
					ui.Text(c, "群主").FontSize(10).TextColor(ui.Hex("#d4a017"))
				case "admin":
					ui.Text(c, "管理").FontSize(10).TextColor(ui.Hex("#2196f3"))
				}
			})
		})
	})
}

// chatWithMember 从成员侧栏发私聊（复用 startChatWith 管线）。
func (a *app) chatWithMember(m member) {
	a.startChatWith(qapi.SearchUserHit{
		ID:   qapi.ID(fmt.Sprintf("%d", m.UserID)),
		Name: m.Name,
	})
}
