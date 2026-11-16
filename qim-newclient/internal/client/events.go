package client

// WS 事件 → 界面状态。事件 data 是原始 JSON：不同事件形状差别很大，
// 每个 handler 只解自己需要的字段，坏帧只记日志，不能带崩读循环。

import (
	"context"
	"log"
	"strconv"
	"strings"

	"qim-newclient/internal/qapi"
)

// registerRealtimeHandlers 把后端的事件名接到原生状态上。
// selfID 在登录完成时固定，回调不去读会被 logout 覆盖的 app 字段。
func (a *app) registerRealtimeHandlers(rt *qapi.Realtime, ctx context.Context, selfID string) {
	rt.On("new_message", func(e qapi.Event) {
		var m qapi.Message
		if err := e.Raw(&m); err != nil {
			log.Printf("[ws] 无法解析 new_message: %v", err)
			return
		}
		a.applyIncoming(m, selfID)
	})

	// new_notification：应用内通知中心通道（后端留痕 + 推送）。
	// 推的是完整 Notification 对象，插到面板列表头并累计未读徽标。
	rt.On("new_notification", func(e qapi.Event) {
		var n qapi.Notification
		if err := e.Raw(&n); err != nil {
			log.Printf("[ws] 无法解析 new_notification: %v", err)
			return
		}
		a.applyNewNotification(n)
	})

	// notification：旧通道，目前仅频道消息推送用它（WSMessage.Type="notification"，
	// Data 是 Notification 对象且其 Type=channel_message）。别的类型忽略。
	rt.On("notification", func(e qapi.Event) {
		var n qapi.Notification
		if err := e.Raw(&n); err != nil {
			log.Printf("[ws] 无法解析 notification: %v", err)
			return
		}
		if n.Type == qapi.ChannelNotifChannelMessage {
			a.applyChannelNotification(n)
		}
	})

	// 会话本身或群成员变了：字段逐个建模不划算，直接重拉列表。
	for _, typ := range []string{
		"conversation_updated", "conversation_update", "new_conversation",
		"group_invitation", "added_to_group", "group_member_left",
		"group_member_joined", "group_member_role_updated",
		"group_owner_transferred", "group_announcement_updated",
	} {
		rt.On(typ, func(e qapi.Event) { go a.refreshConversations(ctx) })
	}

	// message_recalled 推的是完整消息对象（content 已被改成「[消息已撤回]」）。
	rt.On("message_recalled", func(e qapi.Event) {
		var m qapi.Message
		if err := e.Raw(&m); err != nil {
			log.Printf("[ws] 无法解析 message_recalled: %v", err)
			return
		}
		a.applyRecalled(m)
	})

	// message_deleted 只带 {message_id, conversation_id}。
	rt.On("message_deleted", func(e qapi.Event) {
		var d struct {
			MessageID qapi.ID `json:"message_id"`
		}
		if err := e.Raw(&d); err != nil {
			log.Printf("[ws] 无法解析 message_deleted: %v", err)
			return
		}
		a.removeMessage(d.MessageID.String())
	})

	// message_updated 有两种 payload：正文更新带 content，质量核验标注
	// 只带 id + quality。后者没有正文，不能用空串把气泡覆盖掉。
	rt.On("message_updated", func(e qapi.Event) {
		var probe struct {
			Content *string `json:"content"`
		}
		if err := e.Raw(&probe); err != nil {
			log.Printf("[ws] 无法解析 message_updated: %v", err)
			return
		}
		if probe.Content == nil {
			return
		}
		var m qapi.Message
		if err := e.Raw(&m); err != nil {
			log.Printf("[ws] 无法解析 message_updated 正文: %v", err)
			return
		}
		a.applyUpdated(m, selfID)
	})

	// 已读回执：把对方读过的消息在我方气泡上标「已读」。
	rt.On("message_read", func(e qapi.Event) {
		var d struct {
			ConversationID qapi.ID   `json:"conversation_id"`
			UserID         qapi.ID   `json:"user_id"`
			MessageIDs     []qapi.ID `json:"message_ids"`
		}
		if err := e.Raw(&d); err != nil {
			log.Printf("[ws] 无法解析 message_read: %v", err)
			return
		}
		log.Printf("[ws] 已读回执 会话=%s 用户=%s 消息数=%d",
			d.ConversationID, d.UserID, len(d.MessageIDs))
		a.applyReadReceipt(d.ConversationID.String(), d.MessageIDs)
	})

	// 下面两类 P0 只把分发链接通并校验 payload：在线状态需要会话里带
	// 对方 user_id、系统配置要等设置页，都还不具备 UI 载体。

	rt.On("user_status_changed", func(e qapi.Event) {
		var d struct {
			UserID qapi.ID `json:"user_id"`
			Status string  `json:"status"`
		}
		if err := e.Raw(&d); err != nil {
			log.Printf("[ws] 无法解析 user_status_changed: %v", err)
			return
		}
		log.Printf("[ws] 用户状态 用户=%s 状态=%s", d.UserID, d.Status)
		a.applyUserStatus(d.UserID.String(), d.Status)
	})

	rt.On("system_config_updated", func(e qapi.Event) {
		var d map[string]any
		if err := e.Raw(&d); err != nil {
			log.Printf("[ws] 无法解析 system_config_updated: %v", err)
			return
		}
		log.Printf("[ws] 系统配置更新 %d 项", len(d))
	})

	// 服务端 WS 缓冲区溢出时发 sync_hint，消息已落库但没有推过来，
	// 必须按 after_id 增量拉回，成功后才 acknowledge_sync。
	rt.On("sync_hint", func(e qapi.Event) {
		var d struct {
			Reason string `json:"reason"`
		}
		_ = e.Raw(&d)
		log.Printf("[ws] 收到同步提示 reason=%q，开始离线补偿", d.Reason)
		go a.recoverMissed(ctx, rt, selfID)
	})
}

// applyReadReceipt 把对方读过这件事落到我发出的消息上。
// message_ids 为空视为「整个会话都被读了」，把当前会话里我方消息全标已读。
func (a *app) applyReadReceipt(convID string, ids []qapi.ID) {
	keys := make(map[string]bool, len(ids))
	for _, id := range ids {
		keys[messageKey(id.String(), 0)] = true
	}
	a.mutate(func() {
		if a.active < 0 || a.active >= len(a.convs) || a.convs[a.active].ID != convID {
			return // 消息不在屏上，等下次拉历史时由 is_read 带回来
		}
		for i := range a.msgs {
			if !a.msgs[i].Mine || a.msgs[i].Pill != "" {
				continue
			}
			if len(keys) == 0 || keys[a.msgs[i].Key] {
				a.msgs[i].Read = true
			}
		}
	})
}

// applyRecalled 把已显示的气泡换成撤回占位。
// 推送里是完整消息对象（content 已被服务端改成占位、原内容进 extra），
// 自己撤回的保留原文供「重新编辑」。
func (a *app) applyRecalled(m qapi.Message) {
	key := messageKey(m.ID.String(), 0)
	orig := m.RecallOriginal()
	a.mutate(func() {
		mine := m.IsMine(a.selfID)
		for i := range a.msgs {
			if a.msgs[i].Key == key {
				a.msgs[i].Text = recalledLabel(mine, m.SenderName())
				a.msgs[i].Recalled = true
				a.msgs[i].OrigText = orig
				// 富媒体撤回后不能再显示原来的图片/文件卡片
				a.msgs[i].Kind = ""
				a.msgs[i].Att = qapi.Attachment{}
				a.msgs[i].LocalPath = ""
				break
			}
		}
		if j := a.findConv(m.ConversationID.String()); j >= 0 && j == a.active {
			a.convs[j].Preview = recalledLabel(mine, m.SenderName())
		}
	})
}

// removeMessage 从当前会话里删掉一条消息。
func (a *app) removeMessage(id string) {
	if id == "" {
		return
	}
	key := messageKey(id, 0)
	a.mutate(func() {
		for i := range a.msgs {
			if a.msgs[i].Key == key {
				a.msgs = append(a.msgs[:i], a.msgs[i+1:]...)
				return
			}
		}
	})
}

// applyUpdated 就地替换气泡正文；流式中间帧不进消息列表。
func (a *app) applyUpdated(m qapi.Message, selfID string) {
	if m.Type == "streaming" {
		return
	}
	key := messageKey(m.ID.String(), 0)
	incoming := messageFromAPI(m, selfID, 0)
	a.mutate(func() {
		for i := range a.msgs {
			if a.msgs[i].Key == key {
				a.msgs[i].Text = incoming.Text
				return
			}
		}
	})
}

// recoverMissed 处理离线消息补偿：当前会话按本地最大消息 id 增量拉，
// 其它会话没有本地消息缓存，退化为重拉列表刷新预览和未读。
// 任一步失败都不 ack，服务端会在 3s 后重发 sync_hint 重试。
func (a *app) recoverMissed(ctx context.Context, rt *qapi.Realtime, selfID string) {
	if a.client == nil {
		return
	}
	var convID, afterID string
	a.mutate(func() {
		if a.active < 0 || a.active >= len(a.convs) {
			return
		}
		convID = a.convs[a.active].ID
		for i := len(a.msgs) - 1; i >= 0; i-- {
			k := a.msgs[i].Key
			if !strings.HasPrefix(k, "m-") {
				continue
			}
			candidate := strings.TrimPrefix(k, "m-")
			if _, err := strconv.ParseUint(candidate, 10, 64); err == nil {
				afterID = candidate
				break
			}
		}
	})

	if convID != "" && afterID != "" {
		page, err := a.client.MessagesAfter(ctx, qapi.ID(convID), afterID, 50)
		if err != nil {
			log.Printf("[ws] 离线补偿失败，等待服务端重发 sync_hint: %v", err)
			return
		}
		fresh := msgsFromAPI(page.Messages, selfID)
		a.mutate(func() {
			if a.active < 0 || a.active >= len(a.convs) || a.convs[a.active].ID != convID {
				return
			}
			for _, m := range fresh {
				if m.Pill != "" || a.hasMessage(m.Key) {
					continue
				}
				a.msgs = append(a.msgs, m)
			}
		})
		log.Printf("[ws] 离线补偿 会话=%s 服务端返回 %d 条", convID, len(page.Messages))
	}

	if err := a.refreshConversations(ctx); err != nil {
		log.Printf("[ws] 离线补偿刷新会话失败: %v", err)
		return
	}
	if err := rt.Send("acknowledge_sync", nil); err != nil {
		log.Printf("[ws] 发送同步确认失败: %v", err)
	}
}
