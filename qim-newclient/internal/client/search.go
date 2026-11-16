package client

// 三路搜索：会话（本地过滤 + 服务端补充）、聊天记录、联系人。
// 服务端三个接口的参数和返回形状都不一样，差异收在 internal/qapi/search.go，
// 这里只管编排、去抖、结果展示和点击跳转。

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
	"qim-newclient/internal/theme"
)

// maybeTriggerSearch 在 view 里比较这一帧和上一帧的搜索词。
// 输入框自己不提供 on-change 回调，靠绑定值的变化起一次后台请求。
func (a *app) maybeTriggerSearch() {
	if a.search == a.lastSearch {
		return
	}
	a.lastSearch = a.search
	a.triggerSearch(a.search)
}

// triggerSearch 让新请求作废旧结果。seq 是原子的：打字快时
// 旧请求即使晚回来也覆盖不了新关键词的结果。
func (a *app) triggerSearch(raw string) {
	seq := atomic.AddUint64(&a.searchSeq, 1)
	q := strings.TrimSpace(raw)
	a.searchHits = qapi.SearchHits{}
	a.searchErr = ""
	a.searchBusy = q != ""
	if q == "" || a.client == nil || !a.loggedIn {
		a.searchBusy = false
		return
	}
	go a.runSearch(a.ctx, a.client, q, seq)
}

func (a *app) runSearch(parent context.Context, client *qapi.Client, q string, seq uint64) {
	// 简单去抖：连续输入时只在停下来的那一次真正发请求。
	time.Sleep(280 * time.Millisecond)
	if atomic.LoadUint64(&a.searchSeq) != seq {
		return
	}
	ctx, cancel := context.WithTimeout(parent, 12*time.Second)
	defer cancel()

	convs, errConv := client.SearchConversations(ctx, q)
	users, errUser := client.SearchUsers(ctx, q)
	msgs, total, errMsg := client.SearchMessages(ctx, q, "", 1, 20)
	if atomic.LoadUint64(&a.searchSeq) != seq {
		return
	}

	hits := qapi.SearchHits{
		Conversations: convs,
		Users:         users,
		Messages:      msgs,
		MessageTotal:  total,
	}
	// 三路是独立的：某一路失败（比如旧版本后端没有该接口）不该让
	// 整个搜索白屏，只把失败的那路记进提示。
	var failed []string
	if errConv != nil {
		failed = append(failed, "会话")
	}
	if errUser != nil {
		failed = append(failed, "联系人")
	}
	if errMsg != nil {
		failed = append(failed, "聊天记录")
		log.Printf("[search] 搜索失败 %q: %v", q, errMsg)
	}
	errText := ""
	if len(failed) > 0 {
		errText = strings.Join(failed, "、") + "搜索暂不可用"
	}

	a.mutate(func() {
		if atomic.LoadUint64(&a.searchSeq) != seq {
			return
		}
		a.searchHits = hits
		a.searchBusy = false
		a.searchErr = errText
	})
}

// searchResults 是搜索态下的侧栏列表。会话段仍用本地 convRow
// （点一下就能切，交互和平时一致），服务端结果补在下面两段。
// 结果可能多到超出一屏，外面套 Scroll 保证能滚。
func (a *app) searchResults(c *ui.Context) {
	ui.Scroll(c).Grow(1).FillWidth().Children(func() {
		ui.Column(c).FillWidth().Children(func() {
			if a.searchBusy {
				ui.Text(c, "正在搜索…").FontSize(12).TextColor(theme.Gray500).Padding(8, 20)
			}
			if a.searchErr != "" {
				ui.Text(c, a.searchErr).FontSize(11).TextColor(theme.ErrorText).Padding(4, 20)
			}

			local := a.visibleConvs()
			if len(local) > 0 {
				a.searchSection(c, fmt.Sprintf("会话 · %d", len(local)))
				for _, i := range local {
					a.convRow(c, i)
				}
			}
			// 服务端能搜到不在最近列表里的历史会话，补一个轻量行。
			for _, h := range a.searchHits.Conversations {
				if a.findConv(h.ID.String()) < 0 {
					a.remoteConvRow(c, h)
				}
			}
			if len(a.searchHits.Messages) > 0 {
				a.searchSection(c, fmt.Sprintf("聊天记录 · %d", a.searchHits.MessageTotal))
				for _, h := range a.searchHits.Messages {
					a.messageHitRow(c, h)
				}
			}
			if len(a.searchHits.Users) > 0 {
				a.searchSection(c, fmt.Sprintf("联系人 · %d", len(a.searchHits.Users)))
				for _, h := range a.searchHits.Users {
					a.userHitRow(c, h)
				}
			}
			if len(local) == 0 && a.searchHits.Empty() && !a.searchBusy {
				// Scroll 里没有剩余高度可 Grow，居中换成上下留白。
				ui.Box(c).FillWidth().Padding(24, 20).Children(func() {
					ui.Text(c, "没有找到相关结果").FillWidth().TextAlign(ui.Center).
						FontSize(13).TextColor(theme.Gray500)
				})
			}
		})
	})
}

func (a *app) searchSection(c *ui.Context, title string) {
	ui.Row(c).FillWidth().Padding(10, 20, 4, 20).Children(func() {
		ui.Text(c, title).FontSize(11).FontWeight(600).TextColor(theme.Gray500)
	})
}

// hitRow 是搜索结果行的统一底座：左侧圆头像、右侧两行文字，可点击。
func (a *app) hitRow(c *ui.Context, seed, title, sub string) bool {
	row := ui.Row(c).FillWidth().Gap(10).Padding(8, 20).AlignItems(ui.Center)
	if row.Hovered() {
		row = row.Background(theme.Primary50)
	}
	row.Children(func() {
		a.avatar(c, title, "", avatarStyle{Size: 30, Font: 12, Weight: 400, Bg: avatarColor(seed)})
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(13).TextColor(theme.Gray800).MaxLines(1).Ellipsis("…")
			ui.Text(c, sub).FontSize(11).TextColor(theme.Gray500).MaxLines(1).Ellipsis("…")
		})
	})
	return row.Clicked()
}

func (a *app) remoteConvRow(c *ui.Context, h qapi.SearchConvHit) {
	if a.hitRow(c, h.ID.String()+h.Name, h.Name, "历史会话 · "+kindLabel(h.Type)) {
		a.openRemoteConv(h, "")
	}
}

func (a *app) messageHitRow(c *ui.Context, h qapi.SearchMessageHit) {
	name := h.ConvName
	if name == "" {
		name = "会话 " + h.ConvID
	}
	preview := h.Message.Preview()
	if preview == "" {
		preview = "[" + h.Message.Type + "]"
	}
	if a.hitRow(c, h.ConvID, name, preview) {
		a.openRemoteConv(qapi.SearchConvHit{
			ID: h.Message.ConversationID, Name: name, Type: h.ConvType,
		}, "m-"+h.Message.ID.String())
	}
}

func (a *app) userHitRow(c *ui.Context, h qapi.SearchUserHit) {
	sub := "@" + h.Username
	if h.Username == "" {
		sub = "点击发起单聊"
	}
	if a.hitRow(c, h.ID.String(), h.Label(), sub) {
		a.startChatWith(h)
	}
}

// openRemoteConv 打开一个不在本地会话列表里的会话。
// 先按 id 查一次，命中就直接切；没命中就临时插入一条，
// 下次会话列表刷新会被服务端数据替换掉。
func (a *app) openRemoteConv(h qapi.SearchConvHit, highlight string) {
	id := h.ID.String()
	if id == "" {
		return
	}
	// 先把当前会话的草稿存好并清掉 active：下面可能往 convs 头部插一条，
	// 下标一错位，selectConversation 会把草稿记到别的会话上。
	if a.active >= 0 && a.active < len(a.convs) {
		a.saveDraft(a.convs[a.active].ID, a.input)
		a.active = -1
	}
	i := a.findConv(id)
	if i < 0 {
		name := h.Name
		if name == "" {
			name = "会话 " + id
		}
		a.convs = append([]conversation{{
			ID: id, Name: name, Kind: h.Type, Color: avatarColor(id + name),
		}}, a.convs...)
		i = 0
	}
	a.selectConversation(i)
	a.highlight = highlight
	a.search = ""
	a.lastSearch = ""
	a.searchHits = qapi.SearchHits{}
}

// startChatWith 用搜索到的联系人开一段单聊，成功后重拉列表并自动选中。
func (a *app) startChatWith(h qapi.SearchUserHit) {
	if a.client == nil {
		return
	}
	uid := h.ID.String()
	if uid == "" {
		return
	}
	client, ctx := a.client, a.ctx
	a.setNotice("正在打开与 " + h.Label() + " 的会话…")
	go func() {
		id, err := client.CreateSingleConversation(ctx, uid)
		if err != nil {
			a.setNotice("发起会话失败：" + friendlyError(err))
			return
		}
		if err := a.refreshConversations(ctx); err != nil {
			a.setNotice("发起会话失败：" + friendlyError(err))
			return
		}
		a.mutate(func() {
			if i := a.findConv(id); i >= 0 {
				a.selectConversation(i)
			}
			a.notice = ""
			a.search = ""
			a.lastSearch = ""
			a.searchHits = qapi.SearchHits{}
		})
	}()
}
