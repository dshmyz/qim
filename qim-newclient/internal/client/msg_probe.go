package client

// -probe-msg：消息渲染对齐的互操作探针。
// 原生侧按真协议发 merged_forward / share / 带 @token 的文本，再回读校验
// 后端存储的类型与 content（网页端将按同一契约渲染）。不依赖 UI。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"qim-newclient/internal/qapi"
)

func runMsgProbe() int {
	server := strings.TrimSpace(*flagServer)
	user := strings.TrimSpace(*flagUser)
	pass := *flagPass
	if pass == "" {
		pass = os.Getenv("QIM_PASS")
	}
	if server == "" || user == "" || pass == "" {
		fmt.Println("用法: -probe-msg -server http://host:port -user <用户名> -pass <密码>")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	c := qapi.New(server)
	fmt.Printf("服务端: %s\n", c.BaseURL())

	if _, err := c.Login(ctx, user, pass); err != nil {
		fmt.Printf("[登录] ✗ %v\n", err)
		return 1
	}
	fmt.Println("[登录] ✓")

	convs, err := c.Conversations(ctx)
	if err != nil || len(convs) == 0 {
		fmt.Printf("[会话] ✗ 拉会话失败或为空: err=%v n=%d\n", err, len(convs))
		return 1
	}
	target := convs[0]
	fmt.Printf("[会话] ✓ 目标 %s(%s)\n", target.DisplayName(), target.ID)

	// 1. 合并转发真协议
	items := []qapi.MergedForwardItem{
		{ID: "p1", Type: "text", Content: "探针第一条", SenderName: "probe", Timestamp: time.Now().UnixMilli()},
		{ID: "p2", Type: "image", Content: `{"url":"/static/uploads/x.png"}`, SenderName: "probe", Timestamp: time.Now().UnixMilli()},
	}
	payload := qapi.BuildMergedForwardPayload("探针的聊天记录", items)
	sent, err := c.SendMessage(ctx, target.ID, qapi.SendRequest{Type: "merged_forward", Content: payload})
	if err != nil {
		fmt.Printf("[合并转发] ✗ 发送失败: %v\n", err)
		return 1
	}
	if sent == nil || sent.Type != "merged_forward" {
		fmt.Printf("[合并转发] ✗ 回执类型不符: %+v\n", sent)
		return 1
	}
	fmt.Println("[合并转发] ✓ 已按 merged_forward 类型入库")

	// 2. share 卡片
	share := map[string]any{
		"type": "message", "id": "p3", "name": "转发消息",
		"content":         "转发了消息: 探针分享",
		"originalContent": `{"version":1,"title":"t","messages":[]}`,
	}
	shareJSON, _ := json.Marshal(share)
	if _, err := c.SendMessage(ctx, target.ID, qapi.SendRequest{Type: "share", Content: string(shareJSON)}); err != nil {
		fmt.Printf("[分享] ✗ 发送失败: %v\n", err)
		return 1
	}
	fmt.Println("[分享] ✓ 已按 share 类型入库")

	// 3. 回读校验：最近消息里两条都在，类型与解析正确
	page, err := c.Messages(ctx, target.ID, 1, 20)
	if err != nil {
		fmt.Printf("[回读] ✗ %v\n", err)
		return 1
	}
	var mfOK, shOK bool
	for _, m := range page.Messages {
		switch m.Type {
		case "merged_forward":
			if mf := qapi.ParseMergedForward(m.Content); mf != nil && mf.Title == "探针的聊天记录" && len(mf.Messages) == 2 {
				if qapi.ItemPreview(mf.Messages[1]) == "[图片]" {
					mfOK = true
				}
			}
		case "share":
			if sc := qapi.ParseShareCard(m.Content); sc != nil && sc.Name == "转发消息" && sc.Original != "" {
				shOK = true
			}
		}
	}
	if !mfOK {
		fmt.Println("[回读] ✗ merged_forward 回读/解析不符")
		return 1
	}
	if !shOK {
		fmt.Println("[回读] ✗ share 回读/解析不符")
		return 1
	}
	fmt.Println("[回读] ✓ merged_forward 与 share 均可往返")

	// 4. 收尾：撤掉两条探针消息，不留垃圾
	cleanCtx, cc := context.WithTimeout(context.Background(), 20*time.Second)
	defer cc()
	page2, err := c.Messages(cleanCtx, target.ID, 1, 20)
	if err == nil {
		for _, m := range page2.Messages {
			if m.Type == "merged_forward" || m.Type == "share" {
				_ = c.DeleteMessage(cleanCtx, m.ID.String())
			}
		}
	}
	fmt.Println("[清理] ✓ 探针消息已删除")
	return 0
}
