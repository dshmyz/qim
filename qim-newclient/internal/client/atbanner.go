package client

// atBanner：@ 我提醒横幅（对齐 qim-client AtMentionBanner + navigate）。
// 消息列表里 atMe 且未被横幅消费过的消息计数，点击定位第一条并高亮。

import (
	"github.com/egoist/mygo/ui"
	"qim-newclient/internal/theme"
)

// unreadAtCount 是当前会话里还没被横幅消费的 @ 我消息数。对方的
// atMe 消息 read 恒为 false（read 字段只对已发消息有意义），所以用
// atSeen 记账：点一次消费一条，横幅随之消失。
func (a *app) unreadAtCount() int {
	n := 0
	for _, m := range a.msgs {
		if m.AtMe && !m.Mine && !a.atSeen[m.Key] {
			n++
		}
	}
	return n
}

// firstUnreadAt 返回第一条未消费的 @ 我消息的下标（含 pill/divider 的
// 原始列表行号，与 ListState 的 row 一致）。
func (a *app) firstUnreadAt() (int, message, bool) {
	for i, m := range a.msgs {
		if m.AtMe && !m.Mine && !a.atSeen[m.Key] {
			return i, m, true
		}
	}
	return -1, message{}, false
}

// atMentionBanner 渲染横幅，对齐 qim-client AtMentionBanner.vue：
// 通栏、padding 8/16、--primary-light 底 + 12% 主色下边框、18px @ 徽标、
// 13px 次要色文案（数字主色加粗）、右侧「点击查看 ⌄」。
// 它挂在 ChatWindow 根上（mainArea 顶层），横跨消息列与成员侧栏。
func (a *app) atMentionBanner(c *ui.Context) {
	n := a.unreadAtCount()
	if n == 0 {
		return
	}
	bar := ui.Row(c).FillWidth().Padding(8, 16).Gap(8).AlignItems(ui.Center).
		Background(theme.Primary50).BorderWidth(0, 0, 1, 0).BorderColor(ui.Hex("#3385ff2e"))
	if bar.Hovered() {
		bar = bar.Background(theme.PrimaryWash)
	}
	if bar.Clicked() {
		a.navigateToAtMention()
	}
	bar.Children(func() {
		ui.Box(c).Width(18).Height(18).Radius(4).Background(theme.PrimaryWash).Center().Children(func() {
			ui.Text(c, "@").FontSize(11).FontWeight(600).TextColor(theme.Primary)
		})
		ui.Row(c).Gap(0).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "你有 ").FontSize(13).TextColor(theme.Gray500)
			ui.Textf(c, "%d", n).FontSize(13).FontWeight(600).TextColor(theme.Primary)
			ui.Text(c, " 条未读的 @ 消息").FontSize(13).TextColor(theme.Gray500)
		})
		ui.Spacer(c)
		ui.Text(c, "点击查看").FontSize(12).TextColor(theme.Primary)
		ui.Icon(c, iconChevDown).FontSize(11).TextColor(theme.Primary)
	})
}

// navigateToAtMention 定位第一条未读的 @ 消息：滚到视口、描边高亮、
// 记账消费。对齐 ChatWindow.navigateToFirstAtMention（已读上报在进会话
// 时已统一发过，这里只管本地状态）。
func (a *app) navigateToAtMention() {
	idx, m, ok := a.firstUnreadAt()
	if !ok {
		return
	}
	a.list.ScrollIntoView(idx)
	a.highlight = m.Key
	if a.atSeen == nil {
		a.atSeen = map[string]bool{}
	}
	a.atSeen[m.Key] = true
}
