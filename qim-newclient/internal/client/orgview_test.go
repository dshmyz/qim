package client

// 侧边栏页签 / 组织架构树 / 群聊列表 / 群成员侧栏 的单测与截图。

import (
	"encoding/json"
	"testing"

	"github.com/egoist/mygo/ui"
	"qim-newclient/internal/qapi"
)

func TestFilterMembers(t *testing.T) {
	ms := []member{{UserID: 1, Name: "张三"}, {UserID: 2, Name: "李四"}, {UserID: 3, Name: "Zhang San EN"}}
	if got := len(filterMembers(ms, "")); got != 3 {
		t.Fatalf("空词应全量，got %d", got)
	}
	got := filterMembers(ms, "zhang")
	if len(got) != 1 || got[0].UserID != 3 {
		t.Fatalf("大小写不敏感过滤失败: %+v", got)
	}
	if len(filterMembers(ms, "王五")) != 0 {
		t.Fatal("不匹配应返回空")
	}
}

func TestGroupConvs(t *testing.T) {
	convs := []conversation{
		{ID: "1", Kind: "single", Name: "张三"},
		{ID: "2", Kind: "group", Name: "产品研发群"},
		{ID: "3", Kind: "discussion", Name: "讨论A"},
		{ID: "4", Kind: "bot", Name: "机器人"},
	}
	got := groupConvs(convs, "")
	if len(got) != 2 || convs[got[0]].ID != "2" || convs[got[1]].ID != "3" {
		t.Fatalf("应只含群/讨论组: %v", got)
	}
	got = groupConvs(convs, "产品")
	if len(got) != 1 || convs[got[0]].ID != "2" {
		t.Fatalf("按名过滤失败: %v", got)
	}
}

func TestOrgTabExpand(t *testing.T) {
	a := &app{}
	if a.orgTabDeptExpanded("d1") {
		t.Fatal("默认应收起")
	}
	a.toggleOrgTabExpand("d1")
	if !a.orgTabDeptExpanded("d1") {
		t.Fatal("点击后应展开")
	}
	a.toggleOrgTabExpand("d1")
	if a.orgTabDeptExpanded("d1") {
		t.Fatal("再点应收起")
	}
}

func TestOpenInviteMembers(t *testing.T) {
	a := &app{active: 0}
	a.convs = []conversation{{ID: "9", Kind: "group", Name: "研发群"}}
	a.openInviteMembers()
	if !a.orgOpen || a.orgMode != "invite" || a.inviteConv != "9" {
		t.Fatalf("邀请模式状态不对: open=%v mode=%q conv=%q", a.orgOpen, a.orgMode, a.inviteConv)
	}
	if a.orgSel == nil {
		t.Fatal("orgSel 应初始化")
	}
	a.closeCreateGroup()
	if a.orgMode != "" || a.orgOpen {
		t.Fatal("关闭应清 mode")
	}
}

func TestInviteSubmitNoClient(t *testing.T) {
	a := &app{}
	a.openInviteMembers() // 无会话时应直接返回
	if a.orgOpen {
		t.Fatal("无会话不应打开")
	}
	a.orgOpen = true
	a.orgMode = "invite"
	a.inviteConv = "9"
	a.orgSel = map[string]bool{"3": true}
	a.submitInviteMembers() // 无 client 不 panic、状态不误清
	if !a.orgOpen {
		t.Fatal("无 client 不应关闭弹窗")
	}
}

func TestShotSidebarTabs(t *testing.T) {
	newShota := func(active int) *app {
		a := &app{active: active, loggedIn: true}
		a.convs = []conversation{
			{ID: "1", Name: "张三", Preview: "在吗", Time: "10:24", Color: "#3385ff", Kind: "single"},
			{ID: "2", Name: "产品研发群", Sub: "(6人)", Preview: "好", Time: "09:12", Color: "#7c5cff", Kind: "group", Unread: 3},
			{ID: "3", Name: "设计讨论组", Preview: "稿子好了", Time: "昨天", Color: "#10b981", Kind: "discussion"},
		}
		a.convs[1].Members = []member{{UserID: 7, Name: "我", Role: "owner"}, {UserID: 8, Name: "李四"}}
		a.orgDepts = []qapi.Department{
			{ID: json.Number("10"), Name: "研发部", SubDepartments: []qapi.Department{
				{ID: json.Number("11"), Name: "客户端组", Employees: []qapi.User{
					{ID: qapi.ID(json.Number("7")), Nickname: "我"},
					{ID: qapi.ID(json.Number("8")), Nickname: "李四"},
				}},
			}, Employees: []qapi.User{{ID: qapi.ID(json.Number("9")), Nickname: "王五"}}},
			{ID: json.Number("20"), Name: "设计部", Employees: []qapi.User{{ID: qapi.ID(json.Number("12")), Nickname: "赵六"}}},
		}
		a.orgLoaded = true
		a.statusMap = map[string]string{"8": "online"}
		return a
	}
	// mygo 的 ElementTransition 在 headless Tester 里时钟冻结，背景色停在
	// 首帧值——所以每个状态都要在 NewTester 之前设好，一段一个 tester。
	setOrg := func(a *app) {
		a.tab = "org"
		a.orgTabExpanded = map[string]bool{"10": true, "11": true}
	}
	a := newShota(0)
	setOrg(a)
	tt := ui.NewTester(a.view, 1200, 900)
	tt.Frame()
	tt.Frame()
	write(t, tt, "/tmp/qim-tab-org.png")

	a = newShota(0)
	a.tab = "groups"
	tt = ui.NewTester(a.view, 1200, 900)
	tt.Frame()
	tt.Frame()
	write(t, tt, "/tmp/qim-tab-groups.png")

	a = newShota(1) // 产品研发群，成员侧栏展开
	tt = ui.NewTester(a.view, 1200, 900)
	tt.Frame()
	tt.Frame()
	write(t, tt, "/tmp/qim-membersidebar.png")

	a = newShota(1)
	a.membersCollapsed = true
	tt = ui.NewTester(a.view, 1200, 900)
	tt.Frame()
	tt.Frame()
	write(t, tt, "/tmp/qim-membersidebar-collapsed.png")

	a = newShota(0) // 单聊：不出现成员侧栏
	tt = ui.NewTester(a.view, 1200, 900)
	tt.Frame()
	tt.Frame()
	write(t, tt, "/tmp/qim-single-no-sidebar.png")
}
