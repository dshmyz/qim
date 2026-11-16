package client

// 服务端数据 → 原生界面数据的转换层。纯函数已下沉到 internal/model，
// 这里只保留与 app 状态耦合的逻辑，以及供业务代码沿用的转发包装。

import (
	"qim-newclient/internal/model"
	"qim-newclient/internal/qapi"
	"qim-newclient/internal/theme"
	"strings"

	"github.com/egoist/mygo/ui"
)

// 纯数据转换转发到 internal/model：client 只保留与 app 状态耦合的方法。
// 这里保留原函数名，业务代码与测试无需改动。
func avatarColor(seed string) string                       { return model.AvatarColor(seed) }
func kindLabel(kind string) string                         { return model.KindLabel(kind) }
func convTimeLabel(ms int64) string                        { return model.ConvTimeLabel(ms) }
func stampLabel(ms int64) string                           { return model.StampLabel(ms) }
func convsFromAPI(list []qapi.Conversation) []conversation { return model.ConvsFromAPI(list) }
func convSubLabel(cv conversation) string                  { return model.ConvSubLabel(cv) }
func oneLine(s string) string                              { return model.OneLine(s) }
func membersFromAPI(list []qapi.ConvMember) []member       { return model.MembersFromAPI(list) }
func messageKey(id string, i int) string                   { return model.MessageKey(id, i) }
func recalledLabel(mine bool, sender string) string        { return model.RecalledLabel(mine, sender) }
func messageFromAPI(m qapi.Message, selfID string, i int) message {
	return model.MessageFromAPI(m, selfID, i)
}
func mediaSummary(kind string, att qapi.Attachment) string { return model.MediaSummary(kind, att) }
func insertUnreadDivider(msgs []message, unread int) []message {
	return model.InsertUnreadDivider(msgs, unread)
}
func needsTimeDivider(prev, cur int64) bool { return model.NeedsTimeDivider(prev, cur) }
func timeDividerLabel(ms int64) string      { return model.TimeDividerLabel(ms) }
func msgsFromAPI(list []qapi.Message, selfID string) []message {
	return model.MsgsFromAPI(list, selfID)
}
func prependOlder(existing, older []message) []message { return model.PrependOlder(existing, older) }
func oldestServerID(msgs []message) string             { return model.OldestServerID(msgs) }

// statusColor 是标题栏/聊天头部的连接状态圆点颜色。
func (a *app) statusColor() ui.Color {
	switch a.connStatus {
	case qapi.StatusConnected:
		return theme.Online
	case qapi.StatusConnecting:
		return theme.Warning
	default:
		return theme.Gray500
	}
}

// convDraft 返回会话里还没发出去的草稿：当前打开的会话读输入框实时内容，
// 其他会话读切换时存下的草稿。侧栏预览用它显示「[草稿] …」。
func (a *app) convDraft(convID string) string {
	if convID == "" {
		return ""
	}
	if a.active >= 0 && a.active < len(a.convs) && a.convs[a.active].ID == convID {
		return strings.TrimSpace(a.input)
	}
	return strings.TrimSpace(a.drafts[convID])
}

// visibleConvs 返回过滤后的会话下标。搜索命中名称或最后一条消息预览。
func (a *app) visibleConvs() []int {
	q := strings.ToLower(strings.TrimSpace(a.search))
	idx := make([]int, 0, len(a.convs))
	for i := range a.convs {
		if q == "" || strings.Contains(strings.ToLower(a.convs[i].Name+" "+a.convs[i].Preview), q) {
			idx = append(idx, i)
		}
	}
	return idx
}
func (a *app) emptyListHint() string {
	if strings.TrimSpace(a.search) != "" {
		return "没有匹配的会话"
	}
	if a.loadingMsgs {
		return "正在加载会话…"
	}
	return "暂无会话"
}

// findConv 按服务端 id 找会话下标。
func (a *app) findConv(id string) int {
	for i := range a.convs {
		if a.convs[i].ID == id {
			return i
		}
	}
	return -1
}

// hasMessage 判断消息是否已经在列表里。WS 推送和发送回执可能重复到达，
// 去重靠服务端 id，避免同一条消息出现两遍。
func (a *app) hasMessage(key string) bool {
	for i := range a.msgs {
		if a.msgs[i].Key == key {
			return true
		}
	}
	return false
}
