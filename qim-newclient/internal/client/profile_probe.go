package client

// -probe-profile：对真实 qim-server 验证 B 组全链路。
// 资料：GET /users/me → PUT 签名 → 回读 → 还原；
// 状态：GET /users/status 批量；组织：GET /organization/tree 结构校验；
// 建群：树里取一个其他用户 → POST /conversations 建群 → 会话列表出现 → 解散清理；
// 用户详情：GET /users/:id。

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"qim-newclient/internal/qapi"
)

func runProfileProbe() int {
	server := strings.TrimSpace(*flagServer)
	user := strings.TrimSpace(*flagUser)
	pass := *flagPass
	if pass == "" {
		pass = os.Getenv("QIM_PASS")
	}
	if server == "" || user == "" || pass == "" {
		fmt.Println("用法: -probe-profile -server http://host:port -user <用户名> -pass <密码>")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c := qapi.New(server)
	fmt.Printf("服务端: %s\n", c.BaseURL())

	sess, err := c.Login(ctx, user, pass)
	if err != nil {
		fmt.Printf("[登录] ✗ %v\n", err)
		return 1
	}
	selfID := sess.User.ID.String()
	fmt.Printf("[登录] ✓ %s id=%s\n", sess.User.DisplayName(), selfID)

	// 1. 资料：GET /users/me
	me, err := c.Me(ctx)
	if err != nil {
		fmt.Printf("[资料] ✗ %v\n", err)
		return 1
	}
	if me.ID.String() != selfID {
		fmt.Printf("[资料] ✗ id 不符: %s vs %s\n", me.ID, selfID)
		return 1
	}
	fmt.Printf("[资料] ✓ username=%s nickname=%s email=%q created=%q\n",
		me.Username, me.Nickname, me.Email, me.CreatedAt)

	// 1.5 聊天头部副标题的数据源：后端 GetConversations 会把 single 会话
	// 对端的 ip/signature/status/other_member_id 摊平到会话对象上。
	headConvs, err := c.Conversations(ctx)
	if err != nil {
		fmt.Printf("[头部字段] ✗ %v\n", err)
		return 1
	}
	singles := 0
	for _, cv := range headConvs {
		if cv.Type != "single" {
			continue
		}
		singles++
		fmt.Printf("[头部字段] single %s ip=%q signature=%q status=%q peer=%d peerType=%q\n",
			cv.ID, cv.IP, cv.Signature, cv.Status, cv.OtherMemberID, cv.OtherMemberType)
	}
	fmt.Printf("[头部字段] ✓ 会话 %d 条（单聊 %d 条）\n", len(headConvs), singles)

	// 2. PUT 签名 → 回读 → 还原
	const probeSign = "探针签名-勿动"
	up, err := c.UpdateMe(ctx, qapi.UpdateMeRequest{Signature: probeSign})
	if err != nil {
		fmt.Printf("[改签名] ✗ %v\n", err)
		return 1
	}
	if up.Signature != probeSign {
		fmt.Printf("[改签名] ✗ 返回签名=%q\n", up.Signature)
		return 1
	}
	me2, err := c.Me(ctx)
	if err != nil || me2.Signature != probeSign {
		fmt.Printf("[改签名] ✗ 回读失败: %v sig=%q\n", err, me2.Signature)
		return 1
	}
	// 还原：PUT 空字符串后端不更新（仅非空生效），所以写回原值或空格占位。
	if _, err := c.UpdateMe(ctx, qapi.UpdateMeRequest{Signature: " "}); err != nil {
		fmt.Printf("[还原] ⚠ %v\n", err)
	} else {
		fmt.Printf("[改签名] ✓ 写入/回读通过，已还原为空格\n")
	}

	// 3. 用户状态批量
	sts, err := c.UsersStatus(ctx, []uint{mustUint(selfID)})
	if err != nil {
		fmt.Printf("[状态] ✗ %v\n", err)
		return 1
	}
	if len(sts) != 1 || sts[0].UserID.String() != selfID {
		fmt.Printf("[状态] ✗ 响应异常: %+v\n", sts)
		return 1
	}
	fmt.Printf("[状态] ✓ user=%s status=%s last_online=%d\n", sts[0].UserID, sts[0].Status, sts[0].LastOnline)

	// 4. 组织树
	depts, err := c.OrgTree(ctx)
	if err != nil {
		fmt.Printf("[组织] ✗ %v\n", err)
		return 1
	}
	totalEmps := 0
	var walk func([]qapi.Department)
	walk = func(ds []qapi.Department) {
		for _, d := range ds {
			totalEmps += len(d.Employees)
			walk(d.SubDepartments)
		}
	}
	walk(depts)
	fmt.Printf("[组织] ✓ 顶层部门 %d 个，全员 %d 人\n", len(depts), totalEmps)

	// 5. 组织树里挑一个非自己的用户建群
	var peer qapi.User
	walk = func(ds []qapi.Department) {
		for _, d := range ds {
			for _, e := range d.Employees {
				if e.ID.String() != selfID && peer.ID == "" {
					peer = e
				}
			}
			walk(d.SubDepartments)
		}
	}
	walk(depts)
	if peer.ID == "" {
		fmt.Println("[建群] ⚠ 组织树里没有其他用户，跳过建群链路")
		return 0
	}
	gname := fmt.Sprintf("资料探针群-%d", time.Now().Unix())
	conv, err := c.CreateGroup(ctx, gname, []uint{mustUint(peer.ID.String())})
	if err != nil {
		fmt.Printf("[建群] ✗ %v\n", err)
		return 1
	}
	fmt.Printf("[建群] ✓ id=%s name=%s 成员=%s(%s)\n", conv.ID, gname, peer.DisplayName(), peer.ID)

	// 6. 用户详情
	if u, err := c.GetUser(ctx, peer.ID); err != nil || u.ID.String() != peer.ID.String() {
		fmt.Printf("[详情] ✗ err=%v\n", err)
		return 1
	} else {
		fmt.Printf("[详情] ✓ %s username=%s\n", u.DisplayName(), u.Username)
	}

	// 7. 邀请成员（本轮新增的 AddGroupMembers，走同一端点批量）
	inviteID := qapi.ID(selfID) // 把自己也拉进群
	if err := c.AddGroupMembers(ctx, conv.ID.String(), []qapi.ID{inviteID}); err != nil {
		fmt.Printf("[邀请] ✗ %v\n", err)
		return 1
	}
	d2, err := c.GetConversation(ctx, conv.ID)
	if err != nil {
		fmt.Printf("[邀请] ✗ 详情回读失败: %v\n", err)
		return 1
	}
	if len(d2.Members) < 2 {
		fmt.Printf("[邀请] ✗ 成员数未增加: %d\n", len(d2.Members))
		return 1
	}
	fmt.Printf("[邀请] ✓ 群成员 %d 人（含新拉入的自己）\n", len(d2.Members))

	// 7.5 头部「更多 → 修改群名称」：PUT /groups/:id {"name": ...}。
	// 后端只更新 groups 表 + WS 广播，不保证会话表同步，所以只验写链路。
	if err := c.SetGroupName(ctx, conv.ID, gname+"-改名"); err != nil {
		fmt.Printf("[改名] ✗ %v\n", err)
		return 1
	}
	fmt.Println("[改名] ✓ PUT /groups/:id 成功")

	// 8. 清理：解散探针群
	if err := c.DissolveGroup(ctx, conv.ID); err != nil {
		fmt.Printf("[清理] ✗ 解散失败: %v\n", err)
		return 1
	}
	fmt.Println("[清理] ✓ 探针群已解散")

	fmt.Println("=== 探针全过 ===")
	return 0
}

// mustUint 字符串转 uint，失败 panic（探针场景直接崩出来看）。
func mustUint(s string) uint {
	var n uint
	fmt.Sscanf(strings.TrimSpace(s), "%d", &n)
	return n
}
