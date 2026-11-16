package client

// 频道（对齐 qim-client stores/channel.ts + ChannelSidebar/MessageCard）：
// 左栏频道列表（订阅/广场两 tab + 搜索 + 创建），右侧消息卡片流（点赞/评论）。
// 频道消息是广播式动态：发送后订阅者收到 type=channel_message 的通知（WS 事件名
// 是 "notification"，与 new_notification 不同），未读由通知驱动，本地维护。

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"

	"qim-newclient/internal/qapi"
)

// openChannels 打开整屏频道视图：首次进入拉频道列表。
func (a *app) openChannels() {
	a.mutate(func() {
		a.settingsOpen = false
		a.notifsOpen = false
		a.channelsOpen = true
	})
	if a.channelsLoaded {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()
		if err := a.loadChannels(ctx); err != nil {
			a.setNotice("加载频道失败：" + friendlyError(err))
		}
	}()
}

// closeChannels 关闭频道视图；选中态与消息保留，回来还在原地。
func (a *app) closeChannels() {
	a.mutate(func() { a.channelsOpen = false })
}

// resetChannels 登录/登出时清空频道状态。
func (a *app) resetChannels() {
	a.channelsOpen = false
	a.channelsLoaded = false
	a.channels = nil
	a.channelTab = "subscribed"
	a.channelSearch = ""
	a.channelSel = -1
	a.chMsgs = nil
	a.chMsgsLoading = false
	a.chInput = ""
	a.chBusy = false
	a.chCreateOpen = false
	a.chName = ""
	a.chDesc = ""
	a.chCommentFor = ""
	a.chComments = nil
	a.chCommentInput = ""
	a.chCommentsLoading = false
	a.chCommentBusy = false
}

// loadChannels 拉频道列表。保留本地未读计数（后端不存频道未读）。
func (a *app) loadChannels(ctx context.Context) error {
	if a.client == nil {
		return nil
	}
	list, err := a.client.ListChannels(ctx)
	if err != nil {
		return err
	}
	a.mutate(func() {
		oldUnread := make(map[string]int, len(a.channels))
		for _, ch := range a.channels {
			oldUnread[ch.ID.String()] = ch.Unread
		}
		a.channels = list
		for i := range a.channels {
			if u, ok := oldUnread[a.channels[i].ID.String()]; ok {
				a.channels[i].Unread = u
			}
		}
		a.channelsLoaded = true
		// 选中项还原；列表为空则清空选中。
		if a.channelSel >= len(a.channels) {
			a.channelSel = -1
		}
	})
	log.Printf("[app] 频道列表 %d 个", len(list))
	return nil
}

// filteredChannels 按当前 tab + 搜索词过滤频道。
func (a *app) filteredChannels() []qapi.Channel {
	q := strings.ToLower(strings.TrimSpace(a.channelSearch))
	var out []qapi.Channel
	for _, ch := range a.channels {
		if a.channelTab == "subscribed" && !ch.IsSubscribed {
			continue
		}
		if q != "" &&
			!strings.Contains(strings.ToLower(ch.Name), q) &&
			!strings.Contains(strings.ToLower(ch.Description), q) {
			continue
		}
		out = append(out, ch)
	}
	return out
}

// totalChannelUnread 所有频道的未读总数（侧栏入口徽标用）。
func (a *app) totalChannelUnread() int {
	sum := 0
	for _, ch := range a.channels {
		sum += ch.Unread
	}
	return sum
}

// selectChannel 选中频道：清未读、拉消息（后端 DESC → 反转 ASC）。
func (a *app) selectChannel(i int) {
	if i < 0 || i >= len(a.channels) {
		return
	}
	a.mutate(func() {
		a.channelSel = i
		a.channels[i].Unread = 0
		a.chMsgs = nil
		a.chMsgsLoading = true
		a.chCommentFor = ""
		a.chComments = nil
	})
	chID := a.channels[i].ID
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
		defer cancel()
		list, err := a.client.ChannelMessages(ctx, qapi.ID(chID.String()))
		a.mutate(func() {
			// 迟到响应：用户已切到别的频道就丢弃。
			if a.channelSel < 0 || a.channels[a.channelSel].ID.String() != chID.String() {
				return
			}
			a.chMsgsLoading = false
			if err != nil {
				a.notice = "加载频道消息失败：" + friendlyError(err)
				return
			}
			// 后端 created_at DESC，反转成时间正序渲染。
			for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
				list[i], list[j] = list[j], list[i]
			}
			a.chMsgs = list
		})
	}()
}

// selChannel 当前选中的频道（越界返回 nil）。
func (a *app) selChannel() *qapi.Channel {
	if a.channelSel < 0 || a.channelSel >= len(a.channels) {
		return nil
	}
	return &a.channels[a.channelSel]
}

// canPublishIn 当前用户能否在选中频道发消息（对齐 qim-client：创建者豁免）。
func (a *app) canPublishIn(ch *qapi.Channel) bool {
	if ch == nil {
		return false
	}
	if ch.IsCreator(a.selfID) {
		return true
	}
	return ch.PublishPermission == qapi.ChannelPublishAllSubs && ch.IsSubscribed
}

// setChannelTab 切订阅/广场。
func (a *app) setChannelTab(tab string) {
	a.mutate(func() { a.channelTab = tab })
}

// setChannelSearch 更新搜索词。
func (a *app) setChannelSearch(q string) {
	a.mutate(func() { a.channelSearch = q })
}

// toggleSubscribe 订阅/退订当前显示的频道。乐观更新 + 失败回滚。
func (a *app) toggleSubscribe(chID string) {
	idx := a.channelIndexByID(chID)
	if idx < 0 || a.client == nil || !a.loggedIn {
		return
	}
	ch := &a.channels[idx]
	if ch.IsDefault && ch.IsSubscribed {
		a.setNotice("默认频道不可退订")
		return
	}
	want := !ch.IsSubscribed
	a.mutate(func() { a.channels[idx].IsSubscribed = want })
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()
		var err error
		if want {
			err = a.client.SubscribeChannel(ctx, qapi.ID(ch.ID.String()))
		} else {
			err = a.client.UnsubscribeChannel(ctx, qapi.ID(ch.ID.String()))
		}
		if err != nil {
			a.mutate(func() {
				if i := a.channelIndexByID(chID); i >= 0 {
					a.channels[i].IsSubscribed = !want
				}
			})
			a.setNotice(friendlyError(err))
		}
	}()
}

// channelIndexByID 按 id 找频道下标，找不到返回 -1。
func (a *app) channelIndexByID(id string) int {
	for i := range a.channels {
		if a.channels[i].ID.String() == id {
			return i
		}
	}
	return -1
}

// submitChannelMessage 发布频道消息。有权限才发；用服务端返回值入列（含 Sender）。
func (a *app) submitChannelMessage() {
	text := strings.TrimSpace(a.chInput)
	ch := a.selChannel()
	if text == "" || ch == nil || a.chBusy || !a.loggedIn || a.client == nil {
		return
	}
	if !a.canPublishIn(ch) {
		a.setNotice("无权限发布消息：仅频道创建者或开启订阅者发布时才可发")
		return
	}
	chID := ch.ID
	a.mutate(func() {
		a.chBusy = true
		a.chInput = ""
	})
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
		defer cancel()
		msg, err := a.client.SendChannelMessage(ctx, qapi.ID(chID.String()), text, "text")
		a.mutate(func() {
			a.chBusy = false
			if err != nil {
				a.notice = "发布失败：" + friendlyError(err)
				a.chInput = text // 输入回填，别让用户重打
				return
			}
			// 迟到响应：已切频道就丢弃。
			cur := a.selChannel()
			if cur == nil || cur.ID.String() != chID.String() {
				return
			}
			if msg != nil {
				a.chMsgs = append(a.chMsgs, *msg)
			}
		})
	}()
}

// toggleChannelLike 点赞/取消点赞消息 i。乐观翻转，用服务端返回值校正
// （注意：重复点赞时后端返回 code=0 但不带 data，此时保持乐观值）。
func (a *app) toggleChannelLike(i int) {
	if i < 0 || i >= len(a.chMsgs) || a.client == nil || !a.loggedIn {
		return
	}
	msg := a.chMsgs[i]
	want := !msg.Liked
	msgID := msg.ID
	a.mutate(func() {
		a.chMsgs[i].Liked = want
		if want {
			a.chMsgs[i].LikeCount++
		} else if a.chMsgs[i].LikeCount > 0 {
			a.chMsgs[i].LikeCount--
		}
	})
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()
		var info *qapi.ChannelLikeInfo
		var err error
		if want {
			info, err = a.client.LikeChannelMessage(ctx, qapi.ID(msgID.String()))
		} else {
			info, err = a.client.UnlikeChannelMessage(ctx, qapi.ID(msgID.String()))
		}
		a.mutate(func() {
			if i >= len(a.chMsgs) || a.chMsgs[i].ID.String() != msgID.String() {
				return
			}
			if err != nil {
				// 回滚
				a.chMsgs[i].Liked = !want
				if want {
					if a.chMsgs[i].LikeCount > 0 {
						a.chMsgs[i].LikeCount--
					}
				} else {
					a.chMsgs[i].LikeCount++
				}
				a.notice = friendlyError(err)
				return
			}
			// 服务端权威值校正（幂等空响应 info 为零值时不动乐观值）
			if info != nil && (info.LikeCount > 0 || info.IsLiked) {
				a.chMsgs[i].LikeCount = info.LikeCount
				a.chMsgs[i].Liked = info.IsLiked
			}
		})
	}()
}

// openChannelComments 展开消息 i 的评论：拉评论列表 + 点赞状态。
func (a *app) openChannelComments(i int) {
	if i < 0 || i >= len(a.chMsgs) || a.client == nil {
		return
	}
	msgID := a.chMsgs[i].ID
	a.mutate(func() {
		if a.chCommentFor == msgID.String() {
			a.chCommentFor = "" // 再点收起
			return
		}
		a.chCommentFor = msgID.String()
		a.chComments = nil
		a.chCommentInput = ""
		a.chCommentsLoading = true
	})
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
		defer cancel()
		comments, cerr := a.client.ChannelMessageComments(ctx, qapi.ID(msgID.String()))
		likes, lerr := a.client.ChannelMessageLikes(ctx, qapi.ID(msgID.String()))
		a.mutate(func() {
			if a.chCommentFor != msgID.String() {
				return
			}
			a.chCommentsLoading = false
			if cerr != nil {
				a.notice = "加载评论失败：" + friendlyError(cerr)
				return
			}
			a.chComments = comments
			// 点赞状态只在请求成功时校正
			if lerr == nil && likes != nil {
				for j := range a.chMsgs {
					if a.chMsgs[j].ID.String() == msgID.String() {
						a.chMsgs[j].LikeCount = likes.LikeCount
						a.chMsgs[j].Liked = likes.IsLiked
						break
					}
				}
			}
		})
	}()
}

// submitChannelComment 给展开中的消息发评论。
func (a *app) submitChannelComment() {
	text := strings.TrimSpace(a.chCommentInput)
	if text == "" || a.chCommentFor == "" || a.chCommentBusy || a.client == nil || !a.loggedIn {
		return
	}
	ch := a.selChannel()
	if ch == nil || !ch.CanComment() {
		a.setNotice("该频道已关闭评论")
		return
	}
	msgID := a.chCommentFor
	a.mutate(func() {
		a.chCommentBusy = true
		a.chCommentInput = ""
	})
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()
		cm, err := a.client.CommentChannelMessage(ctx, qapi.ID(msgID), text)
		a.mutate(func() {
			a.chCommentBusy = false
			if err != nil {
				a.notice = "评论失败：" + friendlyError(err)
				a.chCommentInput = text
				return
			}
			if a.chCommentFor == msgID && cm != nil {
				a.chComments = append(a.chComments, *cm)
			}
		})
	}()
}

// openChannelCreate 打开创建频道弹层。
func (a *app) openChannelCreate() {
	a.mutate(func() {
		a.chCreateOpen = true
		a.chName = ""
		a.chDesc = ""
	})
}

func (a *app) closeChannelCreate() {
	a.mutate(func() { a.chCreateOpen = false })
}

// submitChannelCreate 创建频道。默认权限：仅创建者可发 + 订阅者可评论（与后端默认一致）。
func (a *app) submitChannelCreate() {
	name := strings.TrimSpace(a.chName)
	if name == "" || a.chBusy || a.client == nil || !a.loggedIn {
		return
	}
	desc := strings.TrimSpace(a.chDesc)
	a.mutate(func() { a.chBusy = true })
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
		defer cancel()
		created, err := a.client.CreateChannel(ctx, qapi.ChannelCreateRequest{
			Name:              name,
			Description:       desc,
			PublishPermission: qapi.ChannelPublishCreatorOnly,
			CommentPermission: qapi.ChannelCommentAllSubs,
		})
		a.mutate(func() {
			a.chBusy = false
			if err != nil {
				a.notice = "创建频道失败：" + friendlyError(err)
				return
			}
			a.chCreateOpen = false
			// 创建成功自动订阅并刷新列表。
			go func() {
				sctx, scancel := context.WithTimeout(a.ctx, 10*time.Second)
				defer scancel()
				if created != nil {
					_ = a.client.SubscribeChannel(sctx, qapi.ID(created.ID.String()))
				}
				if err := a.loadChannels(sctx); err == nil && created != nil {
					a.mutate(func() {
						if i := a.channelIndexByID(created.ID.String()); i >= 0 {
							a.selectChannel(i)
						}
					})
				}
			}()
			if created != nil && created.Status == qapi.ChannelStatusPending {
				a.notice = "频道已创建，等待管理员审批后可用"
			}
		})
	}()
}

// applyChannelNotification 处理 WS "notification" 事件里的频道消息通知：
// 正打开该频道 → 刷新消息；否则累计该频道未读（对齐 qim-client Main.vue）。
func (a *app) applyChannelNotification(n qapi.Notification) {
	if n.Type != qapi.ChannelNotifChannelMessage {
		return
	}
	var payload struct {
		ChannelID   any    `json:"channel_id"`
		ChannelName string `json:"channel_name"`
	}
	if n.ActionPayload != "" {
		_ = json.Unmarshal([]byte(n.ActionPayload), &payload)
	}
	chID := normalizeID(payload.ChannelID)
	if chID == "" {
		return
	}
	// 正打开的频道：刷新消息（权威数据源），未读清零。
	a.mutate(func() {
		i := a.channelIndexByID(chID)
		if i < 0 {
			return // 频道列表没加载过：下次 openChannels 会拉全量
		}
		if a.channelsOpen && a.channelSel == i {
			a.channels[i].Unread = 0
		} else {
			a.channels[i].Unread++
		}
	})
	if a.channelsOpen && a.channelSel >= 0 && a.channelSel < len(a.channels) &&
		a.channels[a.channelSel].ID.String() == chID {
		go func() {
			ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
			defer cancel()
			list, err := a.client.ChannelMessages(ctx, qapi.ID(chID))
			a.mutate(func() {
				if err != nil {
					return
				}
				for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
					list[i], list[j] = list[j], list[i]
				}
				a.chMsgs = list
			})
		}()
	}
}

// normalizeID 把 any（数字/字符串/浮点）统一成十进制字符串。
func normalizeID(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return t.String()
	default:
		b, _ := json.Marshal(t)
		return strings.Trim(string(b), `"`)
	}
}

// chTimeLabel 把频道消息/评论的 created_at（RFC3339）转成显示文案。
func chTimeLabel(s string) string {
	if s == "" {
		return ""
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return convTimeLabel(t.UnixMilli())
	}
	return ""
}
