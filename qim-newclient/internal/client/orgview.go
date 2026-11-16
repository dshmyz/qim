package client

// 侧边栏页签内容（对齐 qim-client Sidebar.vue 的 org / groups 两个
// content-section）：OrgTree.vue 递归部门树 + GroupList.vue 群聊列表。

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"qim-newclient/internal/qapi"
	"qim-newclient/internal/theme"
)

// ensureOrgTree 确保组织树数据就绪（org 页签与建群弹窗共用一份缓存）。
func (a *app) ensureOrgTree() {
	if a.orgLoaded || a.orgLoading {
		return
	}
	a.mutate(func() { a.orgLoading = true })
	if a.client == nil || !a.loggedIn {
		a.mutate(func() { a.orgLoading = false })
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
		defer cancel()
		depts, err := a.client.OrgTree(ctx)
		if err != nil {
			log.Printf("[org] 组织树加载失败: %v", err)
			a.mutate(func() {
				a.orgLoading = false
				a.notice = "组织架构加载失败：" + friendlyError(err)
			})
			return
		}
		a.mutate(func() {
			a.orgDepts = depts
			a.orgLoaded = true
			a.orgLoading = false
		})
	}()
}

// orgTabDeptExpanded 查组织架构页签里部门是否展开（默认全部收起，
// 对齐 OrgTree.vue 的 expandedDepartments 初始为空）。
func (a *app) orgTabDeptExpanded(deptID string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.orgTabExpanded[deptID]
}

// toggleOrgTabExpand 展开/收起组织架构页签里的部门。
func (a *app) toggleOrgTabExpand(deptID string) {
	a.mutate(func() {
		if a.orgTabExpanded == nil {
			a.orgTabExpanded = map[string]bool{}
		}
		a.orgTabExpanded[deptID] = !a.orgTabExpanded[deptID]
	})
}

// orgSidebar 组织架构页签：递归部门树 + 员工行。
// 单击员工 → 资料卡；双击 → 发起私聊（对齐 OrgTree 的 selectUser / openChat）。
func (a *app) orgSidebar(c *ui.Context) {
	if a.orgLoading {
		ui.Column(c).Grow(1).FillWidth().Center().Children(func() {
			ui.Text(c, "正在加载组织架构…").FontSize(13).TextColor(theme.Gray500)
		})
		return
	}
	if !a.orgLoaded && len(a.orgDepts) == 0 {
		// 对齐 OrgTree.vue 的 empty-org 占位
		ui.Column(c).Grow(1).FillWidth().Center().Gap(8).Padding(0, 24).Children(func() {
			ui.Text(c, "🏢").FontSize(28)
			ui.Text(c, "暂无组织架构").FontSize(14).FontWeight(600).TextColor(theme.Gray800)
			ui.Text(c, "暂无部门数据，请联系管理员配置组织架构").FontSize(12).TextColor(theme.Gray500).TextAlign(ui.Center)
		})
		return
	}
	depts := a.orgTabTree()
	if len(depts) == 0 {
		ui.Column(c).Grow(1).FillWidth().Center().Gap(6).Padding(0, 24).Children(func() {
			ui.Text(c, "🔍").FontSize(24)
			ui.Textf(c, "未找到匹配结果").FontSize(14).FontWeight(600).TextColor(theme.Gray800)
			ui.Textf(c, "没有找到与 %q 相关的员工或部门", strings.TrimSpace(a.search)).FontSize(12).TextColor(theme.Gray500).TextAlign(ui.Center)
		})
		return
	}
	// 部门树可能比侧栏高：包一层 Scroll，滚轮能滚到底（原来直接放
	// Column 里，超出部分直接被裁掉）。
	ui.Scroll(c).Grow(1).FillWidth().Children(func() {
		ui.Column(c).FillWidth().Padding(6, 8, 8, 8).Children(func() {
			a.orgTabRows(c, depts, 0)
		})
	})
}

// orgTabTree 当前页签的过滤树：搜索词走 filterDepts（员工名/部门名命中，
// 部门名命中保留全部员工），空词返回全树。
func (a *app) orgTabTree() []qapi.Department {
	q := strings.ToLower(strings.TrimSpace(a.search))
	a.mu.RLock()
	defer a.mu.RUnlock()
	if q == "" {
		return a.orgDepts
	}
	return filterDepts(a.orgDepts, q)
}

// orgTabRows 递归画部门树（对齐 OrgTree.vue：▼/▶ + 部门名 + 员工行）。
func (a *app) orgTabRows(c *ui.Context, depts []qapi.Department, level int) {
	for _, d := range depts {
		a.orgTabDeptRow(c, d, level)
		// 搜索时全部视为展开（对齐 OrgTree 的 watch searchQuery）
		if searching := strings.TrimSpace(a.search) != ""; searching || a.orgTabDeptExpanded(d.ID.String()) {
			a.orgTabRows(c, d.SubDepartments, level+1)
			for _, e := range d.Employees {
				a.orgTabEmpRow(c, e, level+1)
			}
		}
	}
}

// orgTabDeptRow 部门行：▶/▼ + 部门名 + 人数徽标。
func (a *app) orgTabDeptRow(c *ui.Context, d qapi.Department, level int) {
	id := d.ID.String()
	expanded := a.orgTabDeptExpanded(id)
	row := ui.Row(c).FillWidth().Padding(7, 8, 7, float32(level)*16+8).Gap(8).AlignItems(ui.Center)
	if row.Hovered() {
		row = row.Background(theme.Gray50)
	}
	row.Children(func() {
		arrow := "▶"
		if expanded {
			arrow = "▼"
		}
		ui.Box(c).Width(14).Center().Children(func() {
			ui.Text(c, arrow).FontSize(9).TextColor(theme.Gray500)
		})
		ui.Text(c, d.Name).Grow(1).FontSize(13).FontWeight(500).TextColor(theme.Gray800).
			MaxLines(1).Ellipsis("…")
		if n := len(d.CollectDeptEmployees()); n > 0 {
			ui.Box(c).Padding(1, 6).Radius(8).Background(theme.Gray100).Children(func() {
				ui.Textf(c, "%d", n).FontSize(9).TextColor(theme.Gray500)
			})
		}
	})
	// 整行点击切换展开（对齐 tree-node-content @click=toggleDepartment）
	if row.Clicked() {
		a.toggleOrgTabExpand(id)
	}
}

// orgTabEmpRow 员工行：头像 + 姓名 + 职位。单击开资料卡，双击发私聊。
func (a *app) orgTabEmpRow(c *ui.Context, u qapi.User, level int) {
	id := u.ID.String()
	name := u.DisplayName()
	row := ui.Row(c).FillWidth().Padding(5, 8, 5, float32(level)*16+22).Gap(8).AlignItems(ui.Center)
	if row.Hovered() {
		row = row.Background(theme.Gray50)
	}
	if row.Clicked() {
		a.openUserCard(id, name, u.Avatar)
	}
	if row.DoubleClicked() {
		a.startChatWith(qapi.SearchUserHit{ID: u.ID, Name: name, Username: u.Username})
	}
	// 对齐 qim-client OrgTree.vue 员工节点右键（userMenuItems）：
	// 查看资料 + 发起私聊，顺序与 MainContextMenus.vue 一致。
	row.ContextMenu(func(menu *ui.Menu) {
		if menu.Item("查看资料").Chosen() {
			a.openUserCard(id, name, u.Avatar)
		}
		if menu.Item("发起私聊").Chosen() {
			a.startChatWith(qapi.SearchUserHit{ID: u.ID, Name: name, Username: u.Username})
		}
	})
	row.Children(func() {
		av := ui.Box(c).Width(24).Height(24)
		av.Children(func() {
			a.avatar(c, name, u.Avatar, avatarStyle{Size: 24, Font: 10})
		})
		ui.Text(c, name).FontSize(13).TextColor(theme.Gray800).MaxLines(1).Ellipsis("…")
		// 在线点紧跟名字（树接口没有职位字段，职位留待后端补充后再画在行尾）
		st := a.statusOf(id)
		if st != "" {
			dot := theme.Gray300
			if st == "online" {
				dot = theme.Online
			}
			ui.Box(c).Width(7).Height(7).Radius(3.5).Background(dot)
		}
	})
}

// groupsSidebar 群聊列表页签（对齐 GroupList.vue）：本账号的群会话，
// 头像带 群/讨 徽标，点击进入会话，搜索词本地过滤群名。
func (a *app) groupsSidebar(c *ui.Context) {
	list := groupConvs(a.convs, strings.TrimSpace(a.search))
	if len(list) == 0 {
		ui.Column(c).Grow(1).FillWidth().Center().Children(func() {
			hint := "暂无群聊"
			if strings.TrimSpace(a.search) != "" {
				hint = "无匹配群聊"
			}
			ui.Text(c, hint).FontSize(13).TextColor(theme.Gray500)
		})
		return
	}
	// 用 ui.List：群多时可滚，且按会话 id 保位。
	a.groupsList.Key = func(row int) any { return a.convs[list[row]].ID }
	ui.Column(c).Grow(1).FillWidth().Padding(7, 8, 0, 8).Children(func() {
		ui.List(c, &a.groupsList, len(list), func(row int) {
			a.groupRow(c, list[row])
		}).Grow(1).FillWidth()
	})
}

// groupConvs 过滤出群会话（group / discussion），搜索词匹配群名；
// 返回的是 a.convs 的下标，保持原排序。
func groupConvs(conv []conversation, q string) []int {
	var out []int
	q = strings.ToLower(q)
	for i, cv := range conv {
		if cv.Kind != "group" && cv.Kind != "discussion" {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(cv.Name), q) {
			continue
		}
		out = append(out, i)
	}
	return out
}

// groupRow 一条群聊行：头像 + 群/讨 徽标 + 群名 + 成员数，点击进会话。
func (a *app) groupRow(c *ui.Context, i int) {
	conv := a.convs[i]
	badge, bcol := "群", theme.Primary
	if conv.Kind == "discussion" {
		badge, bcol = "讨", theme.Success
	}
	row := ui.Row(c).FillWidth().Gap(12).Padding(10, 12).AlignItems(ui.Center)
	if i == a.active {
		row = row.Background(theme.PrimaryBg)
	} else if row.Hovered() {
		row = row.Background(theme.Primary50)
	}
	row.Children(func() {
		a.avatar(c, conv.Name, "", avatarStyle{Size: 36, Font: 14, Bg: convColor(conv)})
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, conv.Name).FontSize(13).FontWeight(500).TextColor(theme.Gray800).
					MaxLines(1).Ellipsis("…")
				ui.Box(c).Padding(0, 5).Radius(3).Background(bcol).Children(func() {
					ui.Text(c, badge).FontSize(9).TextColor(theme.White)
				})
			})
			if n := len(conv.Members); n > 0 {
				ui.Textf(c, "%d 名成员", n).FontSize(11).TextColor(theme.Gray500)
			}
		})
		if conv.Unread > 0 {
			ui.Box(c).Height(18).Radius(9).Padding(0, 5).Background(theme.Primary).Center().Children(func() {
				ui.Textf(c, "%d", conv.Unread).TextColor(theme.White).FontSize(10).FontWeight(600)
			})
		}
	})
	if row.Clicked() {
		a.highlight = ""
		a.selectConversation(i)
	}
}
