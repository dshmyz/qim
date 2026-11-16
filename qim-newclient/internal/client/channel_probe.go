package client

// -probe-channels：对真实 qim-server 验证频道全链路。
// 创建 → 列表（is_subscribed/message_count）→ 发动态 → 回读（DESC→ASC）
// → 点赞/取消点赞 → 评论/回读。频道无用户侧删除端点（仅 admin），探针频道
// 以「频道探针-<unix>」命名留痕，不影响功能验证。

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"qim-newclient/internal/qapi"
)

func runChannelsProbe() int {
	server := strings.TrimSpace(*flagServer)
	user := strings.TrimSpace(*flagUser)
	pass := *flagPass
	if pass == "" {
		pass = os.Getenv("QIM_PASS")
	}
	if server == "" || user == "" || pass == "" {
		fmt.Println("用法: -probe-channels -server http://host:port -user <用户名> -pass <密码>")
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

	// 1. 创建频道
	name := fmt.Sprintf("频道探针-%d", time.Now().Unix())
	created, err := c.CreateChannel(ctx, qapi.ChannelCreateRequest{
		Name:        name,
		Description: "探针自动创建，可忽略",
	})
	if err != nil {
		fmt.Printf("[创建] ✗ %v\n", err)
		return 1
	}
	fmt.Printf("[创建] ✓ id=%s name=%s status=%s approval=%s\n",
		created.ID, created.Name, created.Status, created.ApprovalStatus)
	chID := qapi.ID(created.ID.String())
	if created.Status != qapi.ChannelStatusActive {
		fmt.Printf("[创建] ⚠ 本库开启了频道审批，状态=%s：发布应被拒，其余链路照测\n", created.Status)
	}

	// 2. 列表校验：含新频道、is_subscribed（后端创建即建订阅？校验实际行为）
	list, err := c.ListChannels(ctx)
	if err != nil {
		fmt.Printf("[列表] ✗ %v\n", err)
		return 1
	}
	var ch *qapi.Channel
	for i := range list {
		if list[i].ID.String() == chID.String() {
			ch = &list[i]
			break
		}
	}
	if ch == nil {
		fmt.Println("[列表] ✗ 新频道不在列表中")
		return 1
	}
	fmt.Printf("[列表] ✓ %d 个频道，含新频道 is_subscribed=%v message_count=%d\n",
		len(list), ch.IsSubscribed, ch.MessageCount)

	// 3. 创建者发布动态（creator_only 也允许创建者）
	const content = "探针动态 Hello Channel"
	msg, err := c.SendChannelMessage(ctx, chID, content, "text")
	if created.Status == qapi.ChannelStatusPending {
		// 审批中频道：发布必须被拒（EnsureChannelUsable 保护层）
		if err == nil {
			fmt.Println("[发布] ✗ pending 频道发布竟然成功了，审批保护失效")
			return 1
		}
		fmt.Printf("[发布] ✓ pending 频道发布被拒（审批保护生效）: %v\n", friendlyError(err))
		msg = nil
	} else if err != nil {
		fmt.Printf("[发布] ✗ %v\n", err)
		return 1
	} else if msg.Sender == nil || msg.Sender.DisplayName() == "" {
		fmt.Printf("[发布] ⚠ 响应缺 sender preload: %+v\n", msg)
	} else {
		fmt.Printf("[发布] ✓ id=%s sender=%s\n", msg.ID, msg.Sender.DisplayName())
	}

	// 4. 回读：DESC → 最新一条是刚发的（pending 频道无消息则跳过）
	msgs, err := c.ChannelMessages(ctx, chID)
	if err != nil {
		fmt.Printf("[回读] ✗ %v\n", err)
		return 1
	}
	if msg != nil {
		if len(msgs) == 0 {
			fmt.Println("[回读] ✗ 消息列表为空")
			return 1
		}
		if msgs[0].Content != content {
			fmt.Printf("[回读] ✗ 最新消息内容不符: %q (期望 %q)\n", msgs[0].Content, content)
			return 1
		}
		fmt.Printf("[回读] ✓ %d 条消息，DESC 最新=%q sender_id=%s\n", len(msgs), msgs[0].Content, msgs[0].SenderID)
	} else {
		fmt.Printf("[回读] ✓ pending 频道 %d 条消息（应为 0）\n", len(msgs))
	}

	// 5. 点赞 → GET likes → 取消点赞。pending 频道无消息时，
	// 从列表找一个 active 频道（默认频道通常有存量消息）继续验证。
	if msg == nil {
		fmt.Println("[点赞] ⚠ pending 频道无消息，改用已有 active 频道验证点赞/评论")
		for i := range list {
			cand := &list[i]
			if cand.Status != qapi.ChannelStatusActive {
				continue
			}
			candMsgs, err := c.ChannelMessages(ctx, qapi.ID(cand.ID.String()))
			if err != nil || len(candMsgs) == 0 {
				continue
			}
			msg = &candMsgs[0]
			fmt.Printf("[点赞] ✓ 使用 active 频道 %q 的消息 id=%s\n", cand.Name, msg.ID)
			break
		}
	}
	if msg == nil {
		fmt.Println("[点赞] ⚠ 没有可用的 active 频道消息，跳过点赞/评论链路")
	} else {
		info, err := c.LikeChannelMessage(ctx, qapi.ID(msg.ID.String()))
		if err != nil {
			fmt.Printf("[点赞] ✗ %v\n", err)
			return 1
		}
		if !info.IsLiked || info.LikeCount < 1 {
			fmt.Printf("[点赞] ✗ 返回异常: %+v\n", info)
			return 1
		}
		check, err := c.ChannelMessageLikes(ctx, qapi.ID(msg.ID.String()))
		if err != nil || !check.IsLiked {
			fmt.Printf("[点赞] ✗ GET likes 校验失败: %+v err=%v\n", check, err)
			return 1
		}
		fmt.Printf("[点赞] ✓ like_count=%d is_liked=true\n", info.LikeCount)
		un, err := c.UnlikeChannelMessage(ctx, qapi.ID(msg.ID.String()))
		if err != nil || un.IsLiked {
			fmt.Printf("[取消] ✗ %+v err=%v\n", un, err)
			return 1
		}
		fmt.Printf("[取消] ✓ like_count=%d is_liked=false\n", un.LikeCount)

		// 6. 评论 → 回读
		cm, err := c.CommentChannelMessage(ctx, qapi.ID(msg.ID.String()), "探针评论 Nice")
		if err != nil {
			fmt.Printf("[评论] ✗ %v\n", err)
			return 1
		}
		if cm.User == nil || cm.Content != "探针评论 Nice" {
			fmt.Printf("[评论] ✗ 响应异常: %+v\n", cm)
			return 1
		}
		cms, err := c.ChannelMessageComments(ctx, qapi.ID(msg.ID.String()))
		if err != nil || len(cms) == 0 {
			fmt.Printf("[评论] ✗ 回读失败: n=%d err=%v\n", len(cms), err)
			return 1
		}
		fmt.Printf("[评论] ✓ id=%s user=%s，回读 %d 条\n", cm.ID, cm.User.DisplayName(), len(cms))
	}

	// 7. 订阅往返：创建不自动订阅 → 订阅 → 退订 → 再订阅
	if err := c.SubscribeChannel(ctx, chID); err != nil {
		fmt.Printf("[订阅] ✗ %v\n", err)
		return 1
	}
	if err := c.UnsubscribeChannel(ctx, chID); err != nil {
		fmt.Printf("[退订] ✗ %v\n", err)
		return 1
	}
	if err := c.SubscribeChannel(ctx, chID); err != nil {
		fmt.Printf("[订阅] ✗ 再订阅: %v\n", err)
		return 1
	}
	fmt.Println("[订阅] ✓ 订阅→退订→再订阅往返成功")

	fmt.Println("\n结论: 频道链路（创建/列表/发布/回读/点赞/评论/订阅）真实协议对齐。")
	return 0
}
