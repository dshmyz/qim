package client

// 会话列表右键动作的真实接口验证（-probe-convlist）：
// ① 建临时群 → 会话列表能读到它，且 members 带回来（侧栏据此显示「(N人)」）；
// ② DELETE /conversations/:id 把它从列表隐藏（不影响群本身）；
// ③ 重新建群后以群主身份 POST /groups/:id/exit，确认后端拦下群主退群；
// ④ 解散临时群收尾。
//
// 这里验证的是「客户端调的路径 / 方法 / 语义」与后端一致，纯展示逻辑
// （相对时间、预览拼发送者名、草稿）由单测覆盖。

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"qim-newclient/internal/qapi"
)

var probeConvList = flag.Bool("probe-convlist", false,
	"只跑会话列表链路：建群→列表回读→移除会话→群主退群被拦→解散（不开窗口），跑完退出")

func runConvListProbe() int {
	server := strings.TrimSpace(*flagServer)
	user := strings.TrimSpace(*flagUser)
	pass := *flagPass
	if pass == "" {
		pass = os.Getenv("QIM_PASS")
	}
	if server == "" || user == "" || pass == "" {
		fmt.Println("用法: -probe-convlist -server http://host:port -user <用户名> -pass <密码>")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	c := qapi.New(server)
	fmt.Printf("服务端: %s\n", c.BaseURL())

	sess, err := c.Login(ctx, user, pass)
	if err != nil {
		fmt.Printf("[登录] ✗ %v\n", err)
		return 1
	}
	fmt.Printf("[登录] ✓ %s id=%s\n", sess.User.DisplayName(), sess.User.ID)

	conv, err := c.CreateGroup(ctx, "会话列表探针临时群", nil)
	if err != nil {
		fmt.Printf("[建群] ✗ %v\n", err)
		return 1
	}
	convID := conv.ID
	cleanup := func() {
		if err := c.DissolveGroup(context.Background(), convID); err != nil {
			fmt.Printf("[清理] ✗ 解散失败 %s: %v\n", convID, err)
		} else {
			fmt.Printf("[清理] ✓ 已解散 %s\n", convID)
		}
	}
	fmt.Printf("[建群] ✓ [%s] %s\n", convID, conv.DisplayName())

	// ② 列表里能读到，且 members 有值（侧栏成员数的唯一来源）。
	list, err := c.Conversations(ctx)
	if err != nil {
		fmt.Printf("[列表] ✗ %v\n", err)
		cleanup()
		return 1
	}
	var found *qapi.Conversation
	for i := range list {
		if list[i].ID.String() == convID.String() {
			found = &list[i]
			break
		}
	}
	if found == nil {
		fmt.Printf("[列表] ✗ 新建的群没出现在会话列表\n")
		cleanup()
		return 1
	}
	if len(found.Members) == 0 {
		fmt.Printf("[列表] ✗ 群会话没带 members，侧栏成员数会缺失\n")
		cleanup()
		return 1
	}
	fmt.Printf("[列表] ✓ 出现在列表，members=%d，type=%s\n", len(found.Members), found.Type)

	// ③ DELETE /conversations/:id 只做 per-user 隐藏，群本身还在。
	if err := c.DeleteConversation(ctx, convID); err != nil {
		fmt.Printf("[移除会话] ✗ %v\n", err)
		cleanup()
		return 1
	}
	list2, err := c.Conversations(ctx)
	if err != nil {
		fmt.Printf("[移除会话] ✗ 回读列表失败: %v\n", err)
		cleanup()
		return 1
	}
	for i := range list2 {
		if list2[i].ID.String() == convID.String() {
			fmt.Printf("[移除会话] ✗ 列表里还在\n")
			cleanup()
			return 1
		}
	}
	fmt.Printf("[移除会话] ✓ 已从列表隐藏（群未解散）\n")

	// ④ 群主退群必须被后端拦下（提示转让或解散），证明路径与错误处理都通。
	err = c.ExitGroup(ctx, convID)
	if err == nil {
		fmt.Printf("[退群] ✗ 群主退群竟然成功了\n")
		cleanup()
		return 1
	}
	if !strings.Contains(err.Error(), "群主") {
		fmt.Printf("[退群] ✗ 错误信息不含群主提示: %v\n", err)
		cleanup()
		return 1
	}
	fmt.Printf("[退群] ✓ 群主被拦下: %v\n", err)

	cleanup()
	fmt.Println("\n结论: 会话列表字段（members）/ 移除会话 / 退群路径与后端语义一致。")
	return 0
}
