package client

// 这个文件是一个纯命令行探针：不创建窗口、不碰 mygo，
// 只验证「Go 能否独立跑完一次真实会话」。
//
//	go run . -probe-api -server http://localhost:8080 -user grace -pass ******
//
// 它跑完这条链路就退出，输出每一步的真实结果：
// 登录 → 拉会话列表 → 拉历史消息 → 连 WS 等推送。
//
// 为什么值得单独做一个探针：在这之前，Go 手里有 token 但一次请求都没发过，
// 所有数据都由网页报上来。这条链路能不能通，是「Go 作为唯一数据源」
// 这个架构成立与否的唯一硬指标，和 UI 做得多漂亮无关。

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"qim-newclient/internal/qapi"
)

var (
	probeAPI = flag.Bool("probe-api", false, "只跑服务端接口连通性验证（不开窗口），跑完退出")
	// -server / -user / -pass 同时给桌面端用：命令行带上就自动登录，
	// 不填则停在原生登录页。探针复用同一组参数，避免两套地址配置。
	flagServer  = flag.String("server", "http://localhost:8080", "服务端地址，如 http://localhost:8080")
	flagUser    = flag.String("user", "", "登录用户名（填了且与 -pass 同时存在时自动登录）")
	flagPass    = flag.String("pass", "", "登录密码（也可用环境变量 QIM_PASS 传，避免进 shell 历史）")
	probeWSWait = flag.Duration("ws-wait", 8*time.Second, "连上 WS 后等待推送的时间")
)

// runAPIProbe 是 -probe-api 的实现。返回值直接作为进程退出码。
func runAPIProbe() int {
	server := strings.TrimSpace(*flagServer)
	user := strings.TrimSpace(*flagUser)
	pass := *flagPass
	if pass == "" {
		pass = os.Getenv("QIM_PASS")
	}

	if server == "" || user == "" || pass == "" {
		fmt.Println("用法: -probe-api -server http://host:port -user <用户名> -pass <密码>")
		fmt.Println("     密码也可用环境变量 QIM_PASS 传入")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	ok := func(step string) { fmt.Printf("  ✓ %s\n", step) }
	bad := func(step string, err error) {
		fmt.Printf("  ✗ %s: %v\n", step, err)
	}

	c := qapi.New(server)
	fmt.Printf("服务端: %s\n\n", c.BaseURL())

	// 1. 登录
	fmt.Println("[1/4] 登录")
	sess, err := c.Login(ctx, user, pass)
	if err != nil {
		var ae *qapi.APIError
		if asAPIError(err, &ae) && ae.IsTwoFactorRequired() {
			fmt.Printf("  ! 该账号开了二次验证，需要验证码（session=%s）\n", ae.TwoFactorSession())
			fmt.Println("    探针暂时不处理 2FA，请先用未开 2FA 的账号验证")
			return 3
		}
		bad("登录失败", err)
		return 1
	}
	ok(fmt.Sprintf("登录成功 用户=%q id=%s token=%d 字节 refresh=%v",
		sess.User.DisplayName(), sess.User.ID, len(sess.Token), sess.RefreshToken != ""))

	// 2. 会话列表
	fmt.Println("[2/4] 会话列表")
	convs, err := c.Conversations(ctx)
	if err != nil {
		bad("拉会话失败", err)
		return 1
	}
	ok(fmt.Sprintf("拿到 %d 个会话", len(convs)))
	if len(convs) == 0 {
		fmt.Println("  ! 会话列表为空，后面的步骤无法继续（这个账号可能还没加入任何会话）")
		return 1
	}
	limit := len(convs)
	if limit > 5 {
		limit = 5
	}
	for _, cv := range convs[:limit] {
		preview := ""
		if m := cv.Last(); m != nil {
			preview = m.Preview()
		}
		fmt.Printf("    - [%s] %s  未读=%d  %s\n", cv.ID, cv.Name, cv.UnreadCount, truncRunes(preview, 24))
	}

	// 3. 历史消息
	fmt.Println("[3/4] 历史消息")
	first := convs[0]
	page, err := c.Messages(ctx, first.ID, 1, 20)
	if err != nil {
		bad("拉消息失败", err)
		return 1
	}
	ok(fmt.Sprintf("「%s」共 %d 条历史，本页 %d 条",
		first.Name, page.Pagination.Total, len(page.Messages)))
	tail := page.Messages
	if len(tail) > 3 {
		tail = tail[len(tail)-3:]
	}
	for _, m := range tail {
		who := "对方"
		// 真实后端不下发 isSelf，只有 sender_id / sender.id，
		// 必须拿登录返回的 user.id 比对，否则自己的消息也会被打印成「对方」。
		if m.IsMine(sess.User.ID.String()) {
			who = "我"
		}
		fmt.Printf("    - %s %s: %s\n", who, m.Type, truncRunes(m.Preview(), 30))
	}

	// 4. WebSocket
	fmt.Println("[4/4] WebSocket 推送")
	rt := qapi.NewRealtime(c)
	defer rt.Stop()

	eventKinds := map[string]int{}
	order := make([]string, 0, 8)
	rt.OnAny(func(e qapi.Event) {
		if _, seen := eventKinds[e.Type]; !seen {
			order = append(order, e.Type)
		}
		eventKinds[e.Type]++
	})

	ready := make(chan struct{}, 1)
	rt.OnStatus(func(s qapi.Status) {
		if s == qapi.StatusConnected {
			select {
			case ready <- struct{}{}:
			default:
			}
		}
	})

	wsCtx, wsCancel := context.WithTimeout(ctx, *probeWSWait+5*time.Second)
	defer wsCancel()
	rt.Start(wsCtx)

	select {
	case <-ready:
		ok("认证成功，长连接已就绪")
	case <-time.After(10 * time.Second):
		fmt.Println("  ✗ 10 秒内没有收到 auth_success")
		fmt.Println("    可能原因：token 无效、后端 WS 路径不是 /api/v1/ws、或认证帧格式不对")
		return 1
	}

	fmt.Printf("  等待推送 %s ……\n", *probeWSWait)
	time.Sleep(*probeWSWait)

	if len(order) == 0 {
		fmt.Println("  · 这段时间没有收到推送（正常，取决于有没有新消息）")
	} else {
		for _, k := range order {
			fmt.Printf("  · 收到 %s × %d\n", k, eventKinds[k])
		}
	}
	if rt.Status() != qapi.StatusConnected {
		fmt.Printf("  ! 结束时连接状态是 %v，可能在反复重连\n", rt.Status())
	}

	fmt.Println()
	fmt.Println("结论: Go 侧可以独立完成 登录 → 会话 → 消息 → 长连接 全链路。")
	fmt.Println("      纯原生桌面端已经用这份客户端作为唯一数据源（go run . -server ... -user ... -pass ...）。")
	return 0
}

// asAPIError 是 errors.As 的简写，避免为了一个断言引入额外 import。
func asAPIError(err error, target **qapi.APIError) bool {
	if e, ok := err.(*qapi.APIError); ok {
		*target = e
		return true
	}
	return false
}

func truncRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func init() {
	// 探针在 flag.Parse 之后、mygo 初始化之前执行，所以这里只做参数校验的兜底。
	_ = log.Flags()
}
