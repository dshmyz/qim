package client

// B 组对齐（qim-client SelfProfileModal / UserProfileCard / CreateGroupModal+OrgTreePicker）：
//   1. 自我资料弹层：Hero 渐变头（72px 可点击头像 + 昵称 + @账号 + 签名输入）+
//      信息区（账号/部门/手机/邮箱/性别/加入时间）+ 底部保存/取消。
//   2. 用户资料小卡：248px 卡片，头部 40px 头像+在线状态点 + 名字 + 状态文字，
//      信息行（@账号 / 部门·职位 / 邮箱 / 手机 / 签名），动作按钮（@TA / 发私聊）。
//      群成员行单击打开（对齐 MemberSidebar.vue 的 handleMemberClick）。
//   3. 建群弹窗：群名输入 + 成员选择（搜索过滤组织树 + 已选胶囊栏 +
//      部门/员工两级树形复选，已在群禁选——建群场景无已在群）。
//   4. 在线状态：statusMap 缓存 user_id → online/offline，批量接口拉取，
//      WS user_status_changed 实时更新。

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/model"
	"qim-newclient/internal/qapi"
	"qim-newclient/internal/theme"
)

// ===================== 自我资料 =====================

// openSelfProfile 拉最新的自己（GET /users/me）然后打开弹层。
func (a *app) openSelfProfile() {
	a.mutate(func() {
		a.profileOpen = true
		a.profileBusy = true
	})
	if a.client == nil || !a.loggedIn {
		a.mutate(func() { a.profileBusy = false })
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()
		u, err := a.client.Me(ctx)
		if err != nil {
			log.Printf("[profile] 拉取个人信息失败: %v", err)
			a.mutate(func() {
				a.profileBusy = false
				a.notice = "加载个人信息失败：" + friendlyError(err)
			})
			return
		}
		// 部门名：从组织树反查（找不到就空着显示「未设置」）
		dept := a.lookupDeptName(u.ID.String())
		a.mutate(func() {
			a.me = u
			a.meDept = dept
			a.profileBusy = false
		})
	}()
}

// lookupDeptName 在组织树里找 user 所属部门名（加载树如果还没加载）。
func (a *app) lookupDeptName(userID string) string {
	if len(a.orgDepts) == 0 {
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()
		if depts, err := a.client.OrgTree(ctx); err == nil {
			a.mutate(func() {
				a.orgDepts = depts
				a.orgLoaded = true
			})
		}
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return findDeptOfUser(a.orgDepts, userID)
}

// findDeptOfUser 纯函数：在树里找 user 所在的最深部门名。
func findDeptOfUser(depts []qapi.Department, userID string) string {
	for _, d := range depts {
		for _, e := range d.Employees {
			if e.ID.String() == userID {
				return d.Name
			}
		}
		if sub := findDeptOfUser(d.SubDepartments, userID); sub != "" {
			return sub
		}
	}
	return ""
}

// closeSelfProfile 关闭自我资料弹层并清草稿。
func (a *app) closeSelfProfile() {
	a.mutate(func() {
		a.profileOpen = false
		a.profNick = ""
		a.profSign = ""
		a.profileBusy = false
		a.profileSaving = false
	})
}

// saveSelfProfile 保存昵称/签名（PUT /users/me），成功更新本地 me。
func (a *app) saveSelfProfile() {
	if a.client == nil || !a.loggedIn || a.profileSaving {
		return
	}
	a.mu.Lock()
	me := a.me
	nick, sign := a.profNick, a.profSign
	a.mu.Unlock()
	if me == nil {
		return
	}
	req := qapi.UpdateMeRequest{}
	if strings.TrimSpace(nick) != "" && nick != me.Nickname {
		req.Nickname = nick
	}
	if sign != me.Signature {
		req.Signature = sign
	}
	if req.Nickname == "" && req.Signature == "" {
		a.closeSelfProfile()
		return
	}
	a.mutate(func() { a.profileSaving = true })
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()
		u, err := a.client.UpdateMe(ctx, req)
		if err != nil {
			log.Printf("[profile] 保存失败: %v", err)
			a.mutate(func() {
				a.profileSaving = false
				a.notice = "保存资料失败：" + friendlyError(err)
			})
			return
		}
		a.mutate(func() {
			a.me = u
			a.profileSaving = false
			a.profileOpen = false
		})
	}()
}

// changeSelfAvatar 弹原生文件框选头像 → 上传 → PUT /users/me。
// mygo.Dialog.Open 必须在非 UI 线程调用（同 pickAttachment 的注释）。
func (a *app) changeSelfAvatar() {
	if a.client == nil || !a.loggedIn || a.avatarBusy {
		return
	}
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:       "选择头像图片",
			ButtonLabel: "确定",
			Parent:      a.win,
			Filters: []mygo.FileFilter{{
				Name:       "图片",
				Extensions: []string{"png", "jpg", "jpeg", "gif", "webp"},
			}},
		})
		if err != nil || len(paths) == 0 || paths[0] == "" {
			return
		}
		a.mutate(func() { a.avatarBusy = true })
		ctx, cancel := context.WithTimeout(a.ctx, 60*time.Second)
		defer cancel()
		att, err := a.client.Upload(ctx, paths[0])
		if err != nil {
			a.mutate(func() {
				a.avatarBusy = false
				a.notice = "上传头像失败：" + friendlyError(err)
			})
			return
		}
		u, err := a.client.UpdateMe(ctx, qapi.UpdateMeRequest{Avatar: att.URL})
		if err != nil {
			a.mutate(func() {
				a.avatarBusy = false
				a.notice = "更新头像失败：" + friendlyError(err)
			})
			return
		}
		a.mutate(func() {
			a.me = u
			a.avatarBusy = false
		})
	}()
}

// ===================== 用户资料小卡 =====================

// openUserCard 打开用户资料小卡：先显示兜底信息（名字/状态占位），
// 再后台拉完整详情（GET /users/:id）+ 在线状态 + 部门名。
func (a *app) openUserCard(userID, fallbackName, fallbackAvatar string) {
	if userID == "" || userID == a.selfID {
		// 自己点自己 → 打开自我资料
		a.openSelfProfile()
		return
	}
	a.mutate(func() {
		a.userCardOpen = true
		a.userCardBusy = true
		a.userCardUser = &qapi.User{ID: qapi.ID(userID), Nickname: fallbackName, Avatar: fallbackAvatar}
		a.userCardDept = ""
		a.userCardPos = ""
	})
	if a.client == nil || !a.loggedIn {
		a.mutate(func() { a.userCardBusy = false })
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()
		u, err := a.client.GetUser(ctx, qapi.ID(userID))
		if err != nil {
			// 详情拉不到也把卡留着（兜底名字还在），只标完成
			log.Printf("[profile] 用户详情失败: %v", err)
			a.mutate(func() { a.userCardBusy = false })
			return
		}
		// /users/:id 响应自带 department/position；空时退回组织树反查
		dept, pos := u.Department, u.Position
		if dept == "" {
			dept, pos = a.lookupDeptAndPosition(u.ID.String())
		}
		st := u.Status
		if st == "" {
			st = "offline"
		}
		a.mutate(func() {
			a.userCardUser = u
			a.userCardDept = dept
			a.userCardPos = pos
			a.userCardBusy = false
			if a.statusMap == nil {
				a.statusMap = map[string]string{}
			}
			a.statusMap[u.ID.String()] = st
		})
	}()
}

// lookupDeptAndPosition 从组织树反查部门名与职位。
func (a *app) lookupDeptAndPosition(userID string) (string, string) {
	if !a.orgLoaded && len(a.orgDepts) == 0 {
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()
		if depts, err := a.client.OrgTree(ctx); err == nil {
			a.mutate(func() {
				a.orgDepts = depts
				a.orgLoaded = true
			})
		}
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	name, pos := "", ""
	var walk func([]qapi.Department)
	walk = func(depts []qapi.Department) {
		for _, d := range depts {
			for _, e := range d.Employees {
				if e.ID.String() == userID {
					name = d.Name
				}
			}
			walk(d.SubDepartments)
		}
	}
	walk(a.orgDepts)
	return name, pos
}

// closeUserCard 关闭用户卡。
func (a *app) closeUserCard() {
	a.mutate(func() {
		a.userCardOpen = false
		a.userCardUser = nil
		a.userCardBusy = false
	})
}

// mentionFromUserCard 从用户卡把 @TA 塞进输入框（保留原有 @ 入口）。
func (a *app) mentionFromUserCard() {
	a.mu.Lock()
	u := a.userCardUser
	a.mu.Unlock()
	if u == nil {
		return
	}
	a.closeUserCard()
	uid := 0
	fmt.Sscanf(u.ID.String(), "%d", &uid)
	a.insertMention(member{UserID: uint(uid), Name: u.DisplayName()})
}

// chatFromUserCard 从用户卡发起/跳转私聊。
func (a *app) chatFromUserCard() {
	a.mu.Lock()
	u := a.userCardUser
	a.mu.Unlock()
	if u == nil {
		return
	}
	a.closeUserCard()
	a.startChatWith(qapi.SearchUserHit{ID: u.ID, Name: u.DisplayName(), Username: u.Username})
}

// applyUserStatus WS user_status_changed 更新本地状态缓存。
func (a *app) applyUserStatus(userID, status string) {
	if userID == "" || status == "" {
		return
	}
	a.mutate(func() {
		if a.statusMap == nil {
			a.statusMap = map[string]string{}
		}
		a.statusMap[userID] = status
	})
}

// statusOf 查某用户的连接状态，未知返回 ""（不画点）。
func (a *app) statusOf(userID string) string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.statusMap[userID]
}

// fetchMemberStatuses 批量拉群成员在线状态（进群会话时调用，失败静默）。
func (a *app) fetchMemberStatuses(ids []string) {
	if a.client == nil || !a.loggedIn || len(ids) == 0 {
		return
	}
	go func() {
		nums := make([]uint, 0, len(ids))
		for _, id := range ids {
			if n, err := strconv.ParseUint(strings.TrimSpace(id), 10, 64); err == nil && n > 0 {
				nums = append(nums, uint(n))
			}
		}
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()
		list, err := a.client.UsersStatus(ctx, nums)
		if err != nil {
			return
		}
		a.mutate(func() {
			if a.statusMap == nil {
				a.statusMap = map[string]string{}
			}
			for _, it := range list {
				a.statusMap[it.UserID.String()] = it.Status
			}
		})
	}()
}

// ===================== 建群（组织树选人） =====================

// openCreateGroup 打开建群弹窗并确保组织树已加载。
func (a *app) openCreateGroup() {
	a.mutate(func() {
		a.orgOpen = true
		a.orgMode = "create"
		a.orgName = ""
		a.orgSearch = ""
		if a.orgSel == nil {
			a.orgSel = map[string]bool{}
		}
	})
	a.ensureOrgTree()
}

// openInviteMembers 打开邀请成员弹窗（复用组织树选人，orgMode=invite）。
// 对齐 qim-client ChatHeaderActions 的邀请成员入口。
func (a *app) openInviteMembers() {
	conv := a.currentConv()
	if conv.ID == "" {
		return
	}
	a.mutate(func() {
		a.orgOpen = true
		a.orgMode = "invite"
		a.inviteConv = conv.ID
		a.orgName = ""
		a.orgSearch = ""
		a.orgSel = map[string]bool{}
	})
	a.ensureOrgTree()
}

// closeCreateGroup 关闭弹窗并清选择（建群/邀请共用）。
func (a *app) closeCreateGroup() {
	a.mutate(func() {
		a.orgOpen = false
		a.orgMode = ""
		a.inviteConv = ""
		a.orgName = ""
		a.orgSearch = ""
		a.orgSel = map[string]bool{}
		a.orgBusy = false
	})
}

// toggleOrgExpand 展开/收起部门。
func (a *app) toggleOrgExpand(deptID string) {
	a.mutate(func() {
		if a.orgExpanded == nil {
			a.orgExpanded = map[string]bool{}
		}
		a.orgExpanded[deptID] = !a.orgExpanded[deptID]
	})
}

// orgDeptExpanded 查部门是否展开（默认收起，顶层默认展开对齐 qim-client）。
func (a *app) orgDeptExpanded(deptID string, topLevel bool) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	v, ok := a.orgExpanded[deptID]
	if ok {
		return v
	}
	return topLevel // 顶层默认展开，减少点击
}

// orgSelectedMembers 返回按 id 排序的已选成员（稳定渲染）。
func (a *app) orgSelectedMembers() []qapi.User {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if len(a.orgSel) == 0 {
		return nil
	}
	ids := make([]string, 0, len(a.orgSel))
	for id := range a.orgSel {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]qapi.User, 0, len(ids))
	if m := a.orgUserByID; m != nil {
		for _, id := range ids {
			if u, ok := m[id]; ok {
				out = append(out, u)
			}
		}
	}
	return out
}

// toggleOrgMember 选中/取消一个员工。
func (a *app) toggleOrgMember(u qapi.User) {
	a.mutate(func() {
		id := u.ID.String()
		if a.orgSel == nil {
			a.orgSel = map[string]bool{}
		}
		if a.orgUserByID == nil {
			a.orgUserByID = map[string]qapi.User{}
		}
		if a.orgSel[id] {
			delete(a.orgSel, id)
		} else {
			a.orgSel[id] = true
			a.orgUserByID[id] = u
		}
	})
}

// toggleOrgDept 全选/反选一个部门（含子部门，对齐 OrgTreePicker.toggleDept）。
func (a *app) toggleOrgDept(d qapi.Department) {
	emps := d.CollectDeptEmployees()
	a.mu.Lock()
	all := true
	for _, e := range emps {
		if !a.orgSel[e.ID.String()] {
			all = false
			break
		}
	}
	a.mu.Unlock()
	a.mutate(func() {
		if a.orgSel == nil {
			a.orgSel = map[string]bool{}
		}
		if a.orgUserByID == nil {
			a.orgUserByID = map[string]qapi.User{}
		}
		for _, e := range emps {
			id := e.ID.String()
			if all {
				delete(a.orgSel, id)
			} else {
				a.orgSel[id] = true
				a.orgUserByID[id] = e
			}
		}
	})
}

// orgDeptSelCount 部门内已选人数 / 总人数（给行尾徽标）。
func (a *app) orgDeptSelCount(d qapi.Department) (int, int) {
	emps := d.CollectDeptEmployees()
	a.mu.RLock()
	defer a.mu.RUnlock()
	n := 0
	for _, e := range emps {
		if a.orgSel[e.ID.String()] {
			n++
		}
	}
	return n, len(emps)
}

// orgFilteredTree 按搜索词本地过滤树（保留部门上下文：部门名命中保留全部员工）。
func (a *app) orgFilteredTree() []qapi.Department {
	a.mu.RLock()
	q := strings.ToLower(strings.TrimSpace(a.orgSearch))
	a.mu.RUnlock()
	if q == "" {
		a.mu.RLock()
		defer a.mu.RUnlock()
		return a.orgDepts
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return filterDepts(a.orgDepts, q)
}

// filterDepts 纯函数过滤（对齐 OrgTreePicker.filterDepartment）。
func filterDepts(depts []qapi.Department, q string) []qapi.Department {
	var out []qapi.Department
	for _, d := range depts {
		nameMatch := strings.Contains(strings.ToLower(d.Name), q)
		var emps []qapi.User
		for _, e := range d.Employees {
			for _, f := range []string{e.DisplayName(), e.Username, e.Email} {
				if strings.Contains(strings.ToLower(f), q) {
					emps = append(emps, e)
					break
				}
			}
		}
		subs := filterDepts(d.SubDepartments, q)
		if nameMatch || len(emps) > 0 || len(subs) > 0 {
			if nameMatch {
				emps = d.Employees
			}
			out = append(out, qapi.Department{
				ID: d.ID, Name: d.Name, Level: d.Level,
				SubDepartments: subs, Employees: emps,
			})
		}
	}
	return out
}

// submitOrgMembers 弹窗提交分发：邀请模式走 AddGroupMembers，否则建群。
func (a *app) submitOrgMembers() {
	if a.orgMode == "invite" {
		a.submitInviteMembers()
		return
	}
	a.submitCreateGroup()
}

// submitInviteMembers 把选中成员加进当前群（POST /groups/:id/members），
// 成功后关弹窗并刷新该会话的成员列表。
func (a *app) submitInviteMembers() {
	if a.client == nil || !a.loggedIn || a.orgBusy || a.inviteConv == "" {
		return
	}
	a.mu.Lock()
	sel := a.orgSel
	convID := a.inviteConv
	a.mu.Unlock()
	if len(sel) == 0 {
		return
	}
	ids := make([]qapi.ID, 0, len(sel))
	for id := range sel {
		ids = append(ids, qapi.ID(strings.TrimSpace(id)))
	}
	a.mutate(func() { a.orgBusy = true })
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
		defer cancel()
		if err := a.client.AddGroupMembers(ctx, convID, ids); err != nil {
			log.Printf("[org] 邀请成员失败: %v", err)
			a.mutate(func() {
				a.orgBusy = false
				a.notice = "邀请成员失败：" + friendlyError(err)
			})
			return
		}
		a.closeCreateGroup()
		a.refreshConversationDetail(convID)
	}()
}

// submitCreateGroup 创建群聊：选中成员 id 列表 → CreateGroup。
// 成功后刷新会话列表并选中新会话。
func (a *app) submitCreateGroup() {
	if a.client == nil || !a.loggedIn || a.orgBusy {
		return
	}
	a.mu.Lock()
	name := strings.TrimSpace(a.orgName)
	sel := a.orgSel
	a.mu.Unlock()
	if name == "" || len(sel) == 0 {
		return
	}
	ids := make([]uint, 0, len(sel))
	for id := range sel {
		if n, err := strconv.ParseUint(strings.TrimSpace(id), 10, 64); err == nil && n > 0 {
			ids = append(ids, uint(n))
		}
	}
	if len(ids) == 0 {
		return
	}
	a.mutate(func() { a.orgBusy = true })
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
		defer cancel()
		conv, err := a.client.CreateGroup(ctx, name, ids)
		if err != nil {
			log.Printf("[org] 建群失败: %v", err)
			a.mutate(func() {
				a.orgBusy = false
				a.notice = "创建群聊失败：" + friendlyError(err)
			})
			return
		}
		// closeCreateGroup 内部自带 mutate，不能再嵌套进 mutate（a.mu 不可重入）
		a.closeCreateGroup()
		a.refreshConversationsSelect(conv.ID.String())
	}()
}

// refreshConversationsSelect 刷新会话列表并选中指定会话（建群成功后调用）。
func (a *app) refreshConversationsSelect(convID string) {
	if a.client == nil || !a.loggedIn {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
		defer cancel()
		if err := a.refreshConversations(ctx); err != nil {
			return
		}
		a.mutate(func() {
			if i := a.findConv(convID); i >= 0 {
				a.selectConversation(i)
			}
		})
	}()
}

// ===================== UI：自我资料弹层 =====================

// selfProfileModal 自我资料弹层，布局对齐 SelfProfileModal.vue：
// 400px 弹窗 + 遮罩；Hero 渐变头（72px 可点击头像 / 昵称 / @账号 / 签名输入）+
// 信息区（每行 30px 图标块 + label/value）+ 底部保存（主色）/取消。
func (a *app) selfProfileModal(c *ui.Context) {
	if !a.profileOpen {
		return
	}
	ui.Box(c).FillWidth().FillHeight().Absolute().Left(0).Top(0).
		Background(ui.Hex("#00000073")).Children(func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Column(c).Width(400).MaxHeight(640).Radius(16).Background(theme.White).
				AlignItems(ui.Center).Children(func() {
				// ---- Hero 头部（主色底，对齐 .self-hero）----
				ui.Column(c).FillWidth().Padding(32, 24, 20, 24).Gap(8).AlignItems(ui.Center).
					Background(theme.Primary).Children(func() {
					// 72px 圆头像：点击换头像（上传中转圈省略）
					avWrap := ui.Box(c).Width(72).Height(72).Radius(36).Background(ui.Hex("#ffffff88")).
						BorderWidth(3, 3, 3, 3).BorderColor(ui.Hex("#ffffffcc")).Center()
					if avWrap.Clicked() {
						a.changeSelfAvatar()
					}
					avWrap.Children(func() {
						a.avatarContent(c, 72, 36)
					})
					// 昵称：可编辑草稿输入（居中白字）
					ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
						ui.Spacer(c)
						nick := a.draftNick()
						ui.TextInput(c, &nick).FillWidth().FontSize(16).FontWeight(700).
							TextColor(theme.White).Background(ui.Transparent).
							Placeholder("昵称")
						a.profNick = nick
						ui.Spacer(c)
					})
					if a.me != nil {
						ui.Text(c, "@"+a.me.Username).FontSize(12).TextColor(ui.Hex("#ffffffb3"))
					}
					// 签名输入（对齐 .self-signature-input：半透明白底居中）
					ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
						ui.Spacer(c)
						sign := a.profSign
						ui.TextInput(c, &sign).Width(280).Height(28).MinHeight(28).Radius(8).
							FontSize(12).TextColor(theme.White).
							Background(ui.Hex("#ffffff1a")).
							Placeholder("写一句签名吧…")
						a.profSign = sign
						ui.Spacer(c)
					})
				})
				// ---- 信息区 ----
				ui.Column(c).FillWidth().Grow(1).Padding(16, 20).Gap(2).AlignItems(ui.Start).Children(func() {
					if a.profileBusy {
						ui.Text(c, "加载中…").FontSize(12).TextColor(theme.Gray500).Padding(20, 0)
						return
					}
					if a.me == nil {
						ui.Text(c, "资料加载失败").FontSize(12).TextColor(theme.Gray500).Padding(20, 0)
						return
					}
					dept := a.meDept
					emptyCol := theme.Gray300
					a.infoRow(c, "@", "账号", a.me.Username, theme.Gray800)
					a.infoRow(c, "❋", "部门", orDefault(dept, "未设置"), emptyCol)
					a.infoRow(c, "☎", "手机", orDefault(a.me.Phone, "未绑定"), emptyCol)
					a.infoRow(c, "✉", "邮箱", orDefault(a.me.Email, "未绑定"), emptyCol)
					a.infoRow(c, "⚙", "性别", genderLabel(a.me.Gender), theme.Gray800)
					a.infoRow(c, "⌚", "加入时间", shortDate(a.me.CreatedAt), theme.Gray500)
				})
				// ---- 操作栏 ----
				ui.Row(c).FillWidth().Padding(16, 20).Gap(8).
					BorderWidth(1, 0, 0, 0).BorderColor(theme.Gray100).Children(func() {
					saveBtn := ui.Box(c).Grow(1).Height(38).Radius(10).Background(theme.Primary).Center()
					saveDisabled := a.profileSaving || a.avatarBusy
					if saveBtn.Clicked() && !saveDisabled {
						a.saveSelfProfile()
					}
					saveCol := theme.White
					if saveDisabled {
						saveCol = ui.Hex("#ffffff99")
					}
					saveBtn.Children(func() {
						label := "保存"
						if a.profileSaving {
							label = "保存中…"
						} else if a.avatarBusy {
							label = "头像上传中…"
						}
						ui.Text(c, label).FontSize(13).FontWeight(600).TextColor(saveCol)
					})
					cancelBtn := ui.Box(c).Grow(1).Height(38).Radius(10).Background(theme.Gray100).Center()
					if cancelBtn.Clicked() {
						a.closeSelfProfile()
					}
					cancelBtn.Children(func() {
						ui.Text(c, "取消").FontSize(13).FontWeight(600).TextColor(theme.Gray800)
					})
				})
			})
		})
	})
}

// draftNick 取昵称草稿：没动过就回填当前昵称。
func (a *app) draftNick() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.profNick != "" {
		return a.profNick
	}
	if a.me != nil {
		return a.me.DisplayName()
	}
	return ""
}

// avatarContent 画资料弹层里的头像内容：真图或首字。
func (a *app) avatarContent(c *ui.Context, size, radius float32) {
	if a.me != nil && a.me.Avatar != "" {
		if src, _, _ := a.imageState(qapi.Attachment{URL: a.me.Avatar}, ""); src != nil {
			ui.Image(c, src).Size(size, size).Radius(radius)
			return
		}
	}
	name := ""
	if a.me != nil {
		name = a.me.DisplayName()
	}
	if name == "" {
		name = a.displayName()
	}
	ui.Text(c, model.FirstRune(name)).TextColor(theme.White).FontWeight(600).FontSize(size * 0.4)
}

// infoRow 一行信息（对齐 .info-item）：30px 图标块 + label/value 竖排。
func (a *app) infoRow(c *ui.Context, icon, label, value string, valueCol ui.Color) {
	ui.Row(c).FillWidth().Padding(10, 12).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Width(30).Height(30).Radius(8).Background(theme.Gray100).Center().Children(func() {
			ui.Text(c, icon).FontSize(12).TextColor(theme.Gray600)
		})
		ui.Column(c).Grow(1).Gap(1).AlignItems(ui.Start).Children(func() {
			ui.Text(c, label).FontSize(10).TextColor(theme.Gray500)
			ui.Text(c, value).FontSize(13).TextColor(valueCol).MaxLines(2).Ellipsis("…")
		})
	})
}

// ===================== UI：用户资料小卡 =====================

// userCardOverlay 用户资料小卡，布局对齐 UserProfileCard.vue：
// 248px 卡片：头部 [40px 头像+状态点 | 名字 + 状态文字]、信息行、动作按钮。
func (a *app) userCardOverlay(c *ui.Context) {
	if !a.userCardOpen || a.userCardUser == nil {
		return
	}
	u := *a.userCardUser
	status := a.statusOf(u.ID.String())
	if status == "" && u.Status != "" {
		status = u.Status
	}
	ui.Box(c).FillWidth().FillHeight().Absolute().Left(0).Top(0).
		Background(ui.Hex("#00000040")).Children(func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Column(c).Width(248).Radius(12).Background(theme.White).Padding(14, 14, 12, 14).
				AlignItems(ui.Start).Shadow(0, 8, 28, 0, theme.Shadow).Children(func() {
				// 头部
				ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Width(40).Height(40).Children(func() {
						a.avatar(c, u.DisplayName(), u.Avatar, avatarStyle{Size: 40, Font: 15, Weight: 400})
						// 状态点（右下角，绿=在线 灰=离线）
						if status != "" {
							dotCol := theme.Gray300
							if status == "online" {
								dotCol = theme.Online
							}
							ui.Box(c).Width(10).Height(10).Radius(5).Background(dotCol).
								BorderWidth(2, 2, 2, 2).BorderColor(theme.White).
								Absolute().Right(-1).Bottom(-1)
						}
					})
					ui.Column(c).Grow(1).Gap(2).AlignItems(ui.Start).Children(func() {
						ui.Text(c, u.DisplayName()).FontSize(14).FontWeight(600).TextColor(theme.Gray800).
							MaxLines(1).Ellipsis("…")
						ui.Text(c, statusText(status, u.Type)).FontSize(11).TextColor(theme.Gray500)
					})
				})
				// 信息行（对齐 infoRows：@账号 / 部门·职位 / 邮箱 / 签名）
				ui.Column(c).FillWidth().Padding(10, 0, 2, 0).Gap(6).AlignItems(ui.Start).Children(func() {
					if u.Username != "" {
						a.cardInfoRow(c, "@", u.Username)
					}
					org := strings.TrimSpace(a.userCardDept)
					if a.userCardPos != "" {
						if org != "" {
							org += " · "
						}
						org += a.userCardPos
					}
					if org != "" {
						a.cardInfoRow(c, "❋", org)
					}
					if u.Email != "" {
						a.cardInfoRow(c, "✉", u.Email)
					}
					if a.userCardBusy {
						ui.Text(c, "加载详情中…").FontSize(11).TextColor(theme.Gray500).Padding(2, 0)
					}
					if sig := u.Signature; sig != "" {
						a.cardInfoRow(c, "❝", sig)
					}
				})
				// 动作按钮（对齐 actions slot）
				ui.Row(c).FillWidth().Gap(8).Padding(4, 0, 0, 0).Children(func() {
					atBtn := ui.Box(c).Grow(1).Height(30).Radius(8).Background(theme.Gray100).Center()
					if atBtn.Clicked() {
						a.mentionFromUserCard()
					}
					atBtn.Children(func() {
						ui.Text(c, "@ TA").FontSize(12).TextColor(theme.Gray800)
					})
					chatBtn := ui.Box(c).Grow(1).Height(30).Radius(8).Background(theme.Primary).Center()
					if chatBtn.Clicked() {
						a.chatFromUserCard()
					}
					chatBtn.Children(func() {
						ui.Text(c, "发私聊").FontSize(12).TextColor(theme.White)
					})
				})
			})
		})
	})
}

// cardInfoRow 用户卡的一行信息（icon + text）。
func (a *app) cardInfoRow(c *ui.Context, icon, text string) {
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, icon).FontSize(11).TextColor(theme.Gray500).Width(14)
		ui.Text(c, text).FontSize(12).TextColor(theme.Gray800).MaxLines(2).Ellipsis("…").Grow(1)
	})
}

// ===================== UI：建群弹窗（组织树选人） =====================

// createGroupModal 建群弹窗，布局对齐 CreateGroupModal.vue：
// header 标题+关闭；群名输入；成员选择区（label+已选 n 人+清空 → 搜索框 →
// 已选胶囊栏 → 组织树：部门行 [▶|☐|名|n/m]、员工行 [☐|头像|名|职位]）；
// footer 取消 / 创建(n)。
func (a *app) createGroupModal(c *ui.Context) {
	if !a.orgOpen {
		return
	}
	modalMask(c, func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Column(c).Width(460).Height(560).Radius(12).Background(theme.White).
				Children(func() {
					// header
					ui.Row(c).FillWidth().Height(52).Padding(0, 16).AlignItems(ui.Center).
						BorderWidth(0, 0, 1, 0).BorderColor(theme.Gray100).Children(func() {
						title := "创建群聊"
						if a.orgMode == "invite" {
							title = "邀请成员"
						}
						ui.Text(c, title).FontSize(15).FontWeight(600).TextColor(theme.Gray800)
						ui.Spacer(c)
						closeBtn := ui.Box(c).Width(26).Height(26).Radius(13).Center()
						if closeBtn.Clicked() {
							a.closeCreateGroup()
						}
						closeBtn.Children(func() {
							ui.Text(c, "✕").FontSize(13).TextColor(theme.Gray500)
						})
					})
					// 群名（仅创建模式）
					if a.orgMode != "invite" {
						ui.Row(c).FillWidth().Padding(12, 16, 4, 16).Gap(10).AlignItems(ui.Center).Children(func() {
							ui.Text(c, "群聊名称").FontSize(12).TextColor(theme.Gray500).Width(60)
							name := a.orgName
							ui.TextInput(c, &name).Grow(1).Height(34).Radius(8).Background(theme.Gray100).
								Padding(0, 10).FontSize(13).TextColor(theme.Gray800).Placeholder("请输入群聊名称")
							a.orgName = name
						})
					}
					// 成员选择头
					selCount := 0
					a.mu.RLock()
					selCount = len(a.orgSel)
					a.mu.RUnlock()
					ui.Row(c).FillWidth().Padding(8, 16, 4, 16).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "群聊成员").FontSize(12).TextColor(theme.Gray500)
						ui.Spacer(c)
						if selCount > 0 {
							ui.Textf(c, "已选择 %d 人", selCount).FontSize(11).TextColor(theme.Primary)
							clearBtn := ui.Box(c).Padding(2, 6).Radius(6).Children(func() {
								ui.Text(c, "清空").FontSize(11).TextColor(theme.Gray500)
							})
							if clearBtn.Clicked() {
								a.mutate(func() { a.orgSel = map[string]bool{} })
							}
						}
					})
					// 搜索框
					ui.Row(c).FillWidth().Padding(4, 16).Children(func() {
						ui.Box(c).FillWidth().Height(30).Radius(8).Background(theme.Gray100).Padding(0, 10).Children(func() {
							q := a.orgSearch
							ui.TextInput(c, &q).FillWidth().Height(30).MinHeight(30).FontSize(12).
								TextColor(theme.Gray800).Background(ui.Transparent).Placeholder("搜索成员...")
							a.orgSearch = q
						})
					})
					// 已选胶囊栏（对齐 SelectedMembersBar）
					if selCount > 0 {
						ui.Row(c).FillWidth().Padding(4, 16).Gap(6).Children(func() {
							for _, u := range a.orgSelectedMembers() {
								a.selectedChip(c, u)
							}
						})
					}
					// 组织树（滚动）
					ui.Column(c).FillWidth().Grow(1).Padding(0, 8).Children(func() {
						switch {
						case a.orgLoading:
							ui.Column(c).FillWidth().Center().Padding(32, 0).Children(func() {
								ui.Text(c, "组织架构加载中…").FontSize(12).TextColor(theme.Gray500)
							})
						default:
							depts := a.orgFilteredTree()
							if len(depts) == 0 {
								msg := "暂无组织架构"
								if strings.TrimSpace(a.orgSearch) != "" {
									msg = "没有找到匹配的成员"
								}
								ui.Column(c).FillWidth().Center().Padding(32, 0).Children(func() {
									ui.Text(c, msg).FontSize(12).TextColor(theme.Gray500)
								})
							} else {
								a.orgDeptRows(c, depts, 0, true)
							}
						}
					})
					// footer
					ui.Row(c).FillWidth().Height(56).Padding(0, 16).Gap(10).AlignItems(ui.Center).
						BorderWidth(1, 0, 0, 0).BorderColor(theme.Gray100).Children(func() {
						cancelBtn := ui.Box(c).Grow(1).Height(36).Radius(8).Background(theme.Gray100).Center()
						if cancelBtn.Clicked() {
							a.closeCreateGroup()
						}
						cancelBtn.Children(func() {
							ui.Text(c, "取消").FontSize(13).TextColor(theme.Gray800)
						})
						createDisabled := a.orgBusy || (a.orgMode != "invite" && strings.TrimSpace(a.orgName) == "") || selCount == 0
						createBtn := ui.Box(c).Grow(1).Height(36).Radius(8).Background(func() ui.Color {
							if createDisabled {
								return theme.Gray200
							}
							return theme.Primary
						}()).Center()
						if createBtn.Clicked() && !createDisabled {
							a.submitOrgMembers()
						}
						createBtn.Children(func() {
							col := theme.White
							if createDisabled {
								col = theme.Gray500
							}
							label := "创建"
							if a.orgMode == "invite" {
								label = "添加"
							}
							if a.orgBusy {
								if a.orgMode == "invite" {
									label = "添加中…"
								} else {
									label = "创建中…"
								}
							}
							ui.Textf(c, "%s (%d)", label, selCount).FontSize(13).FontWeight(600).TextColor(col)
						})
					})
				})
		})
	})
}

// orgDeptRows 递归画部门与员工行（对齐 OrgTreePicker 的节点结构）。
func (a *app) orgDeptRows(c *ui.Context, depts []qapi.Department, level int, top bool) {
	for _, d := range depts {
		a.orgDeptRow(c, d, level, top)
		// 展开才画子级（搜索时自动视为展开）
		expanded := a.orgDeptExpanded(d.ID.String(), top)
		if searching := strings.TrimSpace(a.orgSearch) != ""; searching || expanded {
			a.orgDeptRows(c, d.SubDepartments, level+1, false)
			for _, e := range d.Employees {
				a.orgEmpRow(c, e, level+1)
			}
		}
	}
}

// orgDeptRow 部门行：[▶|☐|部门名|n/m 徽标]，缩进 level*18+8。
func (a *app) orgDeptRow(c *ui.Context, d qapi.Department, level int, top bool) {
	id := d.ID.String()
	expanded := a.orgDeptExpanded(id, top)
	selN, total := a.orgDeptSelCount(d)
	row := ui.Row(c).FillWidth().Padding(7, 10, 7, float32(level)*18+8).Gap(8).AlignItems(ui.Center)
	if row.Hovered() {
		row = row.Background(theme.Gray50)
	}
	row.Children(func() {
		// toggle
		arrow := "▶"
		if expanded {
			arrow = "▼"
		}
		tg := ui.Box(c).Width(14).Center().Children(func() {
			ui.Text(c, arrow).FontSize(9).TextColor(theme.Gray500)
		})
		if tg.Clicked() {
			a.toggleOrgExpand(id)
		}
		a.smallCheckbox(c, total > 0 && selN == total)
		ui.Text(c, d.Name).Grow(1).FontSize(13).FontWeight(500).TextColor(theme.Gray800).
			MaxLines(1).Ellipsis("…")
		ui.Box(c).Padding(1, 6).Radius(8).Background(theme.Gray100).Children(func() {
			ui.Textf(c, "%d/%d", selN, total).FontSize(9).TextColor(theme.Gray500)
		})
	})
	// 整行点击切换展开（对齐 dept-name @click=toggleExpand）
	if row.Clicked() {
		a.toggleOrgExpand(id)
	}
}

// orgEmpRow 员工行：[占位|☐|头像|姓名|职位]，缩进随层级。
func (a *app) orgEmpRow(c *ui.Context, u qapi.User, level int) {
	id := u.ID.String()
	checked := false
	a.mu.RLock()
	checked = a.orgSel[id]
	a.mu.RUnlock()
	row := ui.Row(c).FillWidth().Padding(6, 10, 6, float32(level)*18+8).Gap(8).AlignItems(ui.Center)
	if row.Hovered() {
		row = row.Background(theme.Gray50)
	}
	if row.Clicked() {
		a.toggleOrgMember(u)
	}
	row.Children(func() {
		a.smallCheckbox(c, checked)
		a.avatar(c, u.DisplayName(), "", avatarStyle{Size: 22, Font: 10, Weight: 400})
		ui.Text(c, u.DisplayName()).Grow(1).FontSize(13).TextColor(theme.Gray800).MaxLines(1).Ellipsis("…")
		if u.Email != "" {
			ui.Text(c, u.Email).FontSize(10).TextColor(theme.Gray500).MaxLines(1).Ellipsis("…")
		}
	})
}

// smallCheckbox 12px 自绘复选框（主色填充 + ✓）。
func (a *app) smallCheckbox(c *ui.Context, checked bool) {
	box := ui.Box(c).Width(15).Height(15).Radius(3).Children(func() {
		if checked {
			ui.Text(c, "✓").FontSize(10).TextColor(theme.White)
		}
	})
	if checked {
		box = box.Background(theme.Primary)
	} else {
		box = box.BorderWidth(1, 1, 1, 1).BorderColor(theme.Gray300).Background(theme.White)
	}
}

// selectedChip 已选成员胶囊（头像+名+可移除，对齐 SelectedMembersBar）。
func (a *app) selectedChip(c *ui.Context, u qapi.User) {
	ui.Row(c).Padding(3, 8, 3, 4).Gap(4).Radius(14).Background(theme.Gray100).AlignItems(ui.Center).Children(func() {
		a.avatar(c, u.DisplayName(), "", avatarStyle{Size: 18, Font: 9, Weight: 400})
		ui.Text(c, u.DisplayName()).FontSize(11).TextColor(theme.Gray800).MaxLines(1).Ellipsis("…")
		rm := ui.Box(c).Width(14).Center().Children(func() {
			ui.Text(c, "✕").FontSize(9).TextColor(theme.Gray500)
		})
		if rm.Clicked() {
			a.toggleOrgMember(u)
		}
	})
}

// ===================== 小工具 =====================

// orDefault 空值兜底。
func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

// genderLabel 性别显示。
func genderLabel(g string) string {
	switch g {
	case "male":
		return "男"
	case "female":
		return "女"
	default:
		return "保密"
	}
}

// shortDate 时间字符串取日期部分。
func shortDate(s string) string {
	if s == "" {
		return "—"
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Format("2006-01-02")
	}
	return s[:10]
}

// statusText 状态文案（bot/system 显示「AI 助手」，对齐 UserProfileCard）。
func statusText(status, typ string) string {
	if typ == "bot" || typ == "system" {
		return "AI 助手"
	}
	switch status {
	case "online":
		return "在线"
	case "busy":
		return "忙碌"
	default:
		return "离线"
	}
}

// resetProfile 登录/登出时清空 B 组全部状态。
func (a *app) resetProfile() {
	a.me = nil
	a.meDept = ""
	a.profileOpen = false
	a.profileBusy = false
	a.profileSaving = false
	a.profNick = ""
	a.profSign = ""
	a.avatarBusy = false
	a.userCardOpen = false
	a.userCardBusy = false
	a.userCardUser = nil
	a.userCardDept = ""
	a.userCardPos = ""
	a.orgOpen = false
	a.orgLoading = false
	a.orgDepts = nil
	a.orgLoaded = false
	a.orgExpanded = nil
	a.orgName = ""
	a.orgSearch = ""
	a.orgSel = map[string]bool{}
	a.orgUserByID = nil
	a.orgBusy = false
	a.statusMap = map[string]string{}
}
