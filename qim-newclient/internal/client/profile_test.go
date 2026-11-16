package client

// B 组单测：资料编辑 / 用户卡 / 组织树选人建群 / 在线状态 / 渲染。

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"qim-newclient/internal/qapi"
)

// profileSrvState fake server 的可变状态与请求计数。
type profileSrvState struct {
	mu         sync.Mutex
	failPut    bool
	putReq     qapi.UpdateMeRequest
	putCalls   int
	meCalls    int
	userCalls  int
	orgCalls   int
	lastGroup  string
	lastIDs    []uint
	groupCalls int
}

// serveProfile 起一个 B 组 fake server。
func serveProfile(t *testing.T, st *profileSrvState) *qapi.Client {
	t.Helper()
	mux := http.NewServeMux()
	// 自己资料
	mux.HandleFunc("/api/v1/users/me", func(w http.ResponseWriter, r *http.Request) {
		st.mu.Lock()
		defer st.mu.Unlock()
		switch r.Method {
		case http.MethodGet:
			st.meCalls++
			writeJSON(w, map[string]any{"code": 0, "data": map[string]any{
				"id": 1, "username": "me", "nickname": "Me", "email": "me@x.com",
				"gender": "male", "signature": "hi", "created_at": "2026-01-02T00:00:00Z",
			}})
		case http.MethodPut:
			st.putCalls++
			var req qapi.UpdateMeRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			st.putReq = req
			if st.failPut {
				writeJSON(w, map[string]any{"code": 500, "message": "boom"})
				return
			}
			nick := req.Nickname
			if nick == "" {
				nick = "Me"
			}
			writeJSON(w, map[string]any{"code": 0, "data": map[string]any{
				"id": 1, "username": "me", "nickname": nick, "signature": req.Signature,
			}})
		}
	})
	// 他人详情
	mux.HandleFunc("/api/v1/users/", func(w http.ResponseWriter, r *http.Request) {
		st.mu.Lock()
		defer st.mu.Unlock()
		st.userCalls++
		writeJSON(w, map[string]any{"code": 0, "data": map[string]any{
			"id": 2, "username": "bob", "nickname": "Bob", "email": "bob@x.com",
			"status": "online", "signature": "sig", "type": "user",
		}})
	})
	// 用户状态批量
	mux.HandleFunc("/api/v1/users/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"code": 0, "data": []map[string]any{
			{"user_id": 2, "status": "online", "last_online": 0},
			{"user_id": 3, "status": "offline", "last_online": 100},
		}})
	})
	// 组织树
	mux.HandleFunc("/api/v1/organization/tree", func(w http.ResponseWriter, r *http.Request) {
		st.mu.Lock()
		defer st.mu.Unlock()
		st.orgCalls++
		writeJSON(w, map[string]any{"code": 0, "data": map[string]any{
			"departments": []map[string]any{
				{"id": 10, "name": "研发部", "level": 1, "subDepartments": []map[string]any{
					{"id": 11, "name": "前端组", "level": 2, "employees": []map[string]any{
						{"id": 2, "username": "bob", "nickname": "Bob"},
					}},
				}, "employees": []map[string]any{
					{"id": 3, "username": "carol", "nickname": "Carol"},
				}},
			},
		}})
	})
	// 建群
	mux.HandleFunc("/api/v1/conversations", func(w http.ResponseWriter, r *http.Request) {
		st.mu.Lock()
		defer st.mu.Unlock()
		if r.Method == http.MethodPost {
			st.groupCalls++
			var req struct {
				Type    string `json:"type"`
				Name    string `json:"name"`
				Members []uint `json:"member_ids"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			st.lastGroup = req.Name
			st.lastIDs = req.Members
			writeJSON(w, map[string]any{"code": 0, "data": map[string]any{
				"id": 77, "type": "group", "name": req.Name,
			}})
			return
		}
		// GET 会话列表（建群成功后的刷新）
		writeJSON(w, map[string]any{"code": 0, "data": []map[string]any{
			{"id": 77, "type": "group", "name": st.lastGroup, "unread": 0},
		}})
	})
	mux.HandleFunc("/api/v1/conversations/77", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"code": 0, "data": map[string]any{
			"id": 77, "type": "group", "name": "G", "unread": 0,
		}})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return qapi.New(srv.URL)
}

// waitForProfile 轮询直到 cond 为真（最多 3 秒），失败即 Fatal。
func waitForProfile(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("超时等待: %s", what)
}

// TestOpenSelfProfile 打开资料弹层拉 me + 部门反查。
func TestOpenSelfProfile(t *testing.T) {
	st := &profileSrvState{}
	a := newChannelTestApp()
	a.client = serveProfile(t, st)
	a.openSelfProfile()
	waitForProfile(t, "me 加载", func() bool {
		a.mu.RLock()
		defer a.mu.RUnlock()
		return a.me != nil && a.me.Username == "me"
	})
	if st.meCalls == 0 {
		t.Errorf("应请求 me: %d", st.meCalls)
	}
}

// TestSaveSelfProfile 保存昵称与签名。
func TestSaveSelfProfile(t *testing.T) {
	st := &profileSrvState{}
	a := newChannelTestApp()
	a.client = serveProfile(t, st)
	a.me = &qapi.User{ID: "1", Username: "me", Nickname: "Me", Signature: "old"}
	a.profNick = "NewNick"
	a.profSign = "new sig"
	a.profileOpen = true
	a.saveSelfProfile()
	waitForProfile(t, "保存完成", func() bool {
		a.mu.RLock()
		defer a.mu.RUnlock()
		return a.profileOpen == false && a.me != nil && a.me.Nickname == "NewNick"
	})
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.putReq.Nickname != "NewNick" || st.putReq.Signature != "new sig" {
		t.Errorf("PUT body 不符: %+v", st.putReq)
	}
}

// TestSaveSelfProfileFail 保存失败保持弹层打开。
func TestSaveSelfProfileFail(t *testing.T) {
	st := &profileSrvState{failPut: true}
	a := newChannelTestApp()
	a.client = serveProfile(t, st)
	a.me = &qapi.User{ID: "1", Nickname: "Me", Signature: "old"}
	a.profNick = "X"
	a.profileOpen = true
	a.saveSelfProfile()
	waitForProfile(t, "失败回执", func() bool {
		st.mu.Lock()
		defer st.mu.Unlock()
		return st.putCalls == 1
	})
	a.mu.RLock()
	if !a.profileOpen || a.profileSaving {
		t.Error("失败应保持打开且结束 saving")
	}
	a.mu.RUnlock()
}

// TestOpenUserCard 用户卡：详情覆盖 + 状态缓存 + 部门反查。
func TestOpenUserCard(t *testing.T) {
	st := &profileSrvState{}
	a := newChannelTestApp()
	a.client = serveProfile(t, st)
	a.openUserCard("2", "兜底名", "")
	waitForProfile(t, "用户卡加载", func() bool {
		a.mu.RLock()
		defer a.mu.RUnlock()
		return !a.userCardBusy && a.userCardUser != nil && a.userCardUser.Nickname == "Bob"
	})
	a.mu.RLock()
	dept, Status := a.userCardDept, a.statusMap["2"]
	a.mu.RUnlock()
	if dept != "前端组" {
		t.Errorf("部门反查错: %q", dept)
	}
	if Status != "online" {
		t.Errorf("状态缓存错: %q", Status)
	}
}

// TestApplyUserStatus WS 状态更新进缓存。
func TestApplyUserStatus(t *testing.T) {
	a := newChannelTestApp()
	a.applyUserStatus("5", "online")
	if a.statusOf("5") != "online" {
		t.Error("WS 状态应进缓存")
	}
	a.applyUserStatus("5", "offline")
	if a.statusOf("5") != "offline" {
		t.Error("状态应可翻转为 offline")
	}
}

// TestFetchMemberStatuses 批量拉取进缓存。
func TestFetchMemberStatuses(t *testing.T) {
	a := newChannelTestApp()
	a.client = serveProfile(t, &profileSrvState{})
	a.fetchMemberStatuses([]string{"2", "3"})
	waitForProfile(t, "状态拉取", func() bool {
		a.mu.RLock()
		defer a.mu.RUnlock()
		return a.statusMap["2"] == "online" && a.statusMap["3"] == "offline"
	})
}

// TestOrgTreeSelect 组织树选人：单选/部门全选/搜索过滤/去重。
func TestOrgTreeSelect(t *testing.T) {
	st := &profileSrvState{}
	a := newChannelTestApp()
	a.client = serveProfile(t, st)
	a.openCreateGroup()
	waitForProfile(t, "组织树加载", func() bool {
		a.mu.RLock()
		defer a.mu.RUnlock()
		return a.orgLoaded
	})
	tree := a.orgFilteredTree()
	if len(tree) != 1 || tree[0].Name != "研发部" {
		t.Fatalf("树结构错: %+v", tree)
	}
	// 单选 Bob（前端组）
	bob := tree[0].SubDepartments[0].Employees[0]
	a.toggleOrgMember(bob)
	if a.orgSelectedMembers()[0].Nickname != "Bob" {
		t.Error("Bob 应被选中")
	}
	// 再选一次 → 取消
	a.toggleOrgMember(bob)
	if len(a.orgSelectedMembers()) != 0 {
		t.Error("再点应取消")
	}
	// 部门全选（研发部含子部门 = Carol + Bob）
	a.toggleOrgDept(tree[0])
	if got := a.orgSelectedMembers(); len(got) != 2 {
		t.Fatalf("部门全选应 2 人: %+v", got)
	}
	// toggleDept 再次 → 全反选
	a.toggleOrgDept(tree[0])
	if len(a.orgSelectedMembers()) != 0 {
		t.Error("再次全选应清空")
	}
	// 搜索过滤：carol 命中 → 保留研发部，但只留 Carol
	a.orgSearch = "carol"
	filtered := a.orgFilteredTree()
	if len(filtered) != 1 || len(filtered[0].Employees) != 1 || filtered[0].Employees[0].Username != "carol" {
		t.Errorf("搜索过滤错: %+v", filtered)
	}
	// 搜索过滤：命中子部门员工 bob
	a.orgSearch = "bob"
	filtered = a.orgFilteredTree()
	if len(filtered) == 0 || len(filtered[0].SubDepartments) != 1 {
		t.Errorf("bob 过滤应保留前端组: %+v", filtered)
	}
	// 部门名命中保留全部员工（子部门不命中仍被过滤，对齐 qim-client）
	a.orgSearch = "研发部"
	filtered = a.orgFilteredTree()
	if len(filtered) == 0 || len(filtered[0].Employees) != 1 || len(filtered[0].SubDepartments) != 0 {
		t.Errorf("部门名命中应保留员工并过滤子部门: %+v", filtered)
	}
}

// TestSubmitCreateGroup 建群请求 body 与成功后关闭并选中会话。
func TestSubmitCreateGroup(t *testing.T) {
	st := &profileSrvState{}
	a := newChannelTestApp()
	a.client = serveProfile(t, st)
	a.orgOpen = true
	a.orgLoaded = true
	a.orgName = "探针群"
	a.orgSel = map[string]bool{"2": true, "3": true}
	a.orgUserByID = map[string]qapi.User{
		"2": {ID: "2", Nickname: "Bob"},
		"3": {ID: "3", Nickname: "Carol"},
	}
	a.submitCreateGroup()
	waitForProfile(t, "建群完成", func() bool {
		a.mu.RLock()
		defer a.mu.RUnlock()
		return !a.orgOpen
	})
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.lastGroup != "探针群" || len(st.lastIDs) != 2 {
		t.Errorf("建群请求不符: name=%q ids=%v", st.lastGroup, st.lastIDs)
	}
	waitForProfile(t, "会话选中", func() bool {
		a.mu.RLock()
		defer a.mu.RUnlock()
		return a.active == 0 && len(a.convs) == 1
	})
}

// TestFindDeptOfUser 纯函数：多层树反查部门名。
func TestFindDeptOfUser(t *testing.T) {
	tree := []qapi.Department{
		{ID: "10", Name: "研发部", SubDepartments: []qapi.Department{
			{ID: "11", Name: "前端组", Employees: []qapi.User{{ID: "2"}}},
		}, Employees: []qapi.User{{ID: "3"}}},
	}
	if got := findDeptOfUser(tree, "2"); got != "前端组" {
		t.Errorf("2 应在前端组: %q", got)
	}
	if got := findDeptOfUser(tree, "3"); got != "研发部" {
		t.Errorf("3 应在研发部: %q", got)
	}
	if got := findDeptOfUser(tree, "9"); got != "" {
		t.Errorf("9 应找不到: %q", got)
	}
}

// TestCollectDeptEmployees 部门收集含子部门且去重。
func TestCollectDeptEmployees(t *testing.T) {
	d := qapi.Department{ID: "10", Employees: []qapi.User{{ID: "3"}},
		SubDepartments: []qapi.Department{{ID: "11", Employees: []qapi.User{{ID: "2"}, {ID: "3"}}}}}
	got := d.CollectDeptEmployees()
	if len(got) != 2 {
		t.Fatalf("应去重为 2 人: %+v", got)
	}
}

// TestProfileModalRender 自我资料弹层渲染。
func TestProfileModalRender(t *testing.T) {
	a := newChannelTestApp()
	a.profileOpen = true
	a.me = &qapi.User{ID: "1", Username: "me", Nickname: "Me", Email: "m@x.com",
		Gender: "male", Signature: "hi", CreatedAt: "2026-01-02T00:00:00Z"}
	a.meDept = "研发部"
	tt := ui.NewTester(a.view, 1050, 850)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{"@me", "账号", "部门", "研发部", "手机", "邮箱", "性别", "加入时间", "保存", "取消"} {
		if !tt.HasText(want) {
			t.Errorf("资料弹层缺 %q, texts=%q", want, tt.Texts())
		}
	}
	write(t, tt, "/tmp/qim-profile.png")
}

// TestUserCardRender 用户资料小卡渲染。
func TestUserCardRender(t *testing.T) {
	a := newChannelTestApp()
	a.userCardOpen = true
	a.userCardUser = &qapi.User{ID: "2", Username: "bob", Nickname: "Bob", Email: "b@x.com", Type: "user"}
	a.userCardDept = "前端组"
	a.statusMap = map[string]string{"2": "online"}
	tt := ui.NewTester(a.view, 1050, 850)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{"Bob", "在线", "@", "前端组", "b@x.com", "@ TA", "发私聊"} {
		if !tt.HasText(want) {
			t.Errorf("用户卡缺 %q, texts=%q", want, tt.Texts())
		}
	}
	write(t, tt, "/tmp/qim-usercard.png")
}

// TestCreateGroupModalRender 建群弹窗渲染（树 + 胶囊 + 计数）。
func TestCreateGroupModalRender(t *testing.T) {
	a := newChannelTestApp()
	a.orgOpen = true
	a.orgLoaded = true
	a.orgDepts = []qapi.Department{{
		ID: "10", Name: "研发部", Level: 1,
		Employees: []qapi.User{{ID: "3", Nickname: "Carol", Username: "carol"}},
		SubDepartments: []qapi.Department{{ID: "11", Name: "前端组", Level: 2,
			Employees: []qapi.User{{ID: "2", Nickname: "Bob", Username: "bob"}}}},
	}}
	a.orgSel = map[string]bool{"2": true}
	a.orgUserByID = map[string]qapi.User{"2": {ID: "2", Nickname: "Bob"}}
	a.orgName = "探针群"
	tt := ui.NewTester(a.view, 1050, 850)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{"创建群聊", "群聊名称", "群聊成员", "已选择 1 人", "研发部", "前端组", "Bob", "Carol", "创建 (1)"} {
		if !tt.HasText(want) {
			t.Errorf("建群弹窗缺 %q, texts=%q", want, tt.Texts())
		}
	}
	write(t, tt, "/tmp/qim-creategroup.png")
}

// TestGenderAndStatusLabels 文案辅助。
func TestGenderAndStatusLabels(t *testing.T) {
	if genderLabel("male") != "男" || genderLabel("female") != "女" || genderLabel("") != "保密" {
		t.Error("genderLabel 不符")
	}
	if statusText("online", "user") != "在线" || statusText("", "bot") != "AI 助手" || statusText("offline", "user") != "离线" {
		t.Error("statusText 不符")
	}
	if shortDate("2026-01-02T00:00:00Z") != "2026-01-02" || shortDate("") != "—" {
		t.Error("shortDate 不符")
	}
	if orDefault("", "未设置") != "未设置" || orDefault("x", "未设置") != "x" {
		t.Error("orDefault 不符")
	}
	if fmt.Sprint(strings.Contains("abc", "b")) != "true" {
		t.Error("sanity")
	}
}
