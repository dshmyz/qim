package client

// -probe-actions：对真实 qim-server 跑一遍 P1 第二批的所有写操作。
// 与 -probe-api 一样不开窗口、不初始化 mygo，验证的是协议而不是界面。
//
//	go run . -probe-actions -server http://127.0.0.1:8080 -user native_probe -pass '...'
//
// 会在第一个会话里留下几条探针消息并撤回/删除其中一部分，因此只应对
// 本地开发库使用，不要指向生产环境。

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"qim-newclient/internal/qapi"
)

var probeActions = flag.Bool("probe-actions", false, "只跑引用/@提及/撤回/删除/提醒的真实接口验证（不开窗口），跑完退出")

func runActionProbe() int {
	server := strings.TrimSpace(*flagServer)
	user := strings.TrimSpace(*flagUser)
	pass := *flagPass
	if pass == "" {
		pass = os.Getenv("QIM_PASS")
	}
	if server == "" || user == "" || pass == "" {
		fmt.Println("用法: -probe-actions -server http://host:port -user <用户名> -pass <密码>")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	c := qapi.New(server)
	fmt.Printf("服务端: %s\n", c.BaseURL())
	fmt.Println("注意: 该探针会在第一个会话写入并撤回/删除测试消息，仅用于本地开发库。")
	fmt.Println()

	sess, err := c.Login(ctx, user, pass)
	if err != nil {
		fmt.Printf("[登录] ✗ %v\n", err)
		return 1
	}
	fmt.Printf("[登录] ✓ %s id=%s\n", sess.User.DisplayName(), sess.User.ID)

	convs, err := c.Conversations(ctx)
	if err != nil || len(convs) == 0 {
		fmt.Printf("[会话] ✗ 没有可用会话: %v\n", err)
		return 1
	}
	conv := convs[0]
	fmt.Printf("[会话] ✓ 使用 [%s] %s (type=%s)\n", conv.ID, conv.DisplayName(), conv.Type)

	// 发送消息后不假设响应体形状，统一从最新一页历史里按正文回查 id。
	send := func(content string, req qapi.SendRequest) (*qapi.Message, error) {
		if _, err := c.SendMessage(ctx, conv.ID, req); err != nil {
			return nil, err
		}
		page, err := c.Messages(ctx, conv.ID, 1, 10)
		if err != nil {
			return nil, err
		}
		for i := len(page.Messages) - 1; i >= 0; i-- {
			if page.Messages[i].Content == content {
				return &page.Messages[i], nil
			}
		}
		return nil, fmt.Errorf("发送成功但历史里找不到这条消息")
	}
	latest := func(id string) *qapi.Message {
		page, err := c.Messages(ctx, conv.ID, 1, 10)
		if err != nil {
			return nil
		}
		for i := range page.Messages {
			if page.Messages[i].ID.String() == id {
				return &page.Messages[i]
			}
		}
		return nil
	}

	stamp := time.Now().Format("15:04:05")
	base := "原生动作探针 " + stamp
	m1, err := send(base, qapi.TextRequest(base))
	if err != nil {
		fmt.Printf("[1/6 发消息] ✗ %v\n", err)
		return 1
	}
	fmt.Printf("[1/6 发消息] ✓ id=%s\n", m1.ID)

	quoteText := "引用回复 " + stamp
	m2, err := send(quoteText, qapi.TextRequestWithQuote(quoteText, m1.ID.String()))
	if err != nil {
		fmt.Printf("[2/6 引用回复] ✗ %v\n", err)
		return 1
	}
	if m2.QuotedMessageID.String() != m1.ID.String() && (m2.QuotedMessage == nil || m2.QuotedMessage.ID.String() != m1.ID.String()) {
		fmt.Printf("[2/6 引用回复] ✗ 后端没回带引用关系: id=%s quoted_id=%q quoted_msg=%v\n",
			m2.ID, m2.QuotedMessageID, m2.QuotedMessage)
		return 1
	}
	fmt.Printf("[2/6 引用回复] ✓ id=%s → 引用 %s\n", m2.ID, m1.ID)

	mentionText := "动作探针 " + qapi.EncodeMentionAll() + " 请查收 " + stamp
	m3, err := send(mentionText, qapi.TextRequest(mentionText))
	if err != nil {
		fmt.Printf("[3/6 @提及] ✗ %v\n", err)
		return 1
	}
	if decoded, mentions := qapi.DecodeMentions(m3.Content); len(mentions) == 0 || !strings.Contains(decoded, "@所有人") {
		fmt.Printf("[3/6 @提及] ✗ 后端回显的 content 解不出提及: %q\n", m3.Content)
		return 1
	}
	fmt.Printf("[3/6 @提及] ✓ id=%s content 解码后=%q\n", m3.ID, func() string {
		d, _ := qapi.DecodeMentions(m3.Content)
		return d
	}())

	if _, err := c.RecallMessage(ctx, m2.ID.String()); err != nil {
		fmt.Printf("[4/6 撤回] ✗ %v\n", err)
		return 1
	}
	if got := latest(m2.ID.String()); got == nil || !got.IsRecalled {
		fmt.Printf("[4/6 撤回] ✗ 历史里没标记为已撤回: %+v\n", got)
		return 1
	}
	fmt.Printf("[4/6 撤回] ✓ 历史里 is_recalled=true\n")

	// 提醒默认要求消息发出满 1 小时。先在历史里找一条够老、自己发的未读
	// 消息验证成功路径；找不到就退回刚发的消息，验证业务条件拦截的路径。
	remindID := m1.ID.String()
	if page, err := c.Messages(ctx, conv.ID, 1, 50); err == nil {
		for i := range page.Messages {
			cand := &page.Messages[i]
			if cand.IsMine(sess.User.ID.String()) && !cand.IsRead && !cand.IsRecalled &&
				cand.Timestamp > 0 && time.Since(time.UnixMilli(cand.Timestamp)) >= time.Hour {
				remindID = cand.ID.String()
				break
			}
		}
	}
	if err := c.RemindMessage(ctx, remindID); err != nil {
		// 业务错误（未满 60 分钟 / 已读 / 提醒渠道未启用）都说明请求
		// 已经到达后端并被正确解析，协议是通的；网络错误才算失败。
		var ae *qapi.APIError
		if asAPIError(err, &ae) {
			fmt.Printf("[5/6 提醒] ✓ 接口已通，后端按业务条件拦截: %v\n", err)
		} else {
			fmt.Printf("[5/6 提醒] ✗ 网络或协议错误: %v\n", err)
			return 1
		}
	} else {
		fmt.Printf("[5/6 提醒] ✓ id=%s 请求已被接受\n", remindID)
	}

	if err := c.DeleteMessage(ctx, m3.ID.String()); err != nil {
		fmt.Printf("[6/6 删除] ✗ %v\n", err)
		return 1
	}
	if got := latest(m3.ID.String()); got != nil {
		fmt.Printf("[6/6 删除] ✗ 删除后仍能拉到 id=%s\n", m3.ID)
		return 1
	}
	fmt.Printf("[6/6 删除] ✓ 已从我的历史里移除\n")

	fmt.Println("\n结论: 引用 / @提及 token / 撤回 / 删除 / 提醒 的真实协议全部对齐。")
	return 0
}
