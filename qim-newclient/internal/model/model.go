// Package model 是 qim 原生客户端的数据模型与服务端数据归一化层。
// 它只依赖 internal/qapi，不含任何 UI（mygo/ui）与窗口状态，
// 因此可被 client 与测试独立复用。
package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"qim-newclient/internal/qapi"
)

// avatarPalette 头像底色按名字散列，同一个种子每次刷新颜色一致。
var avatarPalette = []string{
	"#3385ff", "#26b361", "#6366f1", "#14b8a6",
	"#f59e0b", "#ef4444", "#0ea5e9", "#8d6e63",
}

type Conversation struct {
	ID           string
	Name         string
	Sub          string
	Preview      string
	Time         string
	Unread       int
	Color        string
	Online       bool
	Kind         string // group / single / bot / discussion
	Stamp        int64  // 毫秒，排序用
	Muted        bool
	Pinned       bool
	GroupID      uint     // 群会话对应的 group 主键，公告/管理操作要用
	Announcement string   // 群公告，群会话才有
	Members      []Member // 进群后拉详情补充，@ 候选优先用它
	// 单聊头部：服务端在会话列表里摊平的对端信息（对齐 ChatHeader.vue 的
	// ip / signature / status / other_member_id），没有字段时副标题留空。
	Avatar    string
	IP        string
	Signature string
	Status    string // online / offline / busy
	PeerID    uint
	PeerType  string // user / bot / system / api
}

type Member struct {
	UserID uint
	Name   string
	Avatar string
	Role   string // owner / admin / Member
}

type Message struct {
	Key      string // ui.List 的稳定 key：服务端 id 或本地临时 id
	Mine     bool
	Sender   string
	SenderID string // 发送者用户 id（server_id）：头像右键「快速 @」/资料卡要用
	Avatar   string // 发送者头像 URL（qapi.Sender.Avatar），空则退回彩色首字
	Text     string
	At       string
	Pill     string
	Stamp    int64
	Pending  bool
	Failed   bool
	Read     bool // 自己发的消息：对方是否已读（后端 is_read + WS 回执）
	// 服务端消息 id（数字串）。撤回/删除/提醒/引用都只能用它；
	// 本地乐观消息为空，相关菜单项要禁用。
	ServerID    string
	QuoteID     string // 被引用的服务端消息 id
	QuoteSender string
	QuoteText   string
	AtMe        bool   // 这条消息 @ 了我（或 @ 所有人）
	Divider     bool   // 「以下为新消息」分隔线，不是真消息
	Recalled    bool   // 已撤回，不能再次撤回
	OrigText    string // 撤回前的原文（extra.original_content），重新编辑用；非撤回消息为空
	// P2 富媒体：kind 非空时 body 走图片/文件渲染，text 只是复述用的摘要。
	// localPath 是待上传（或刚上传完）的本地文件，用于发送中即时预览。
	Kind      string
	Att       qapi.Attachment
	LocalPath string
	// 消息渲染对齐（qim-client TextMessage / MergedForwardMessage / ShareMessage）：
	Sys  bool                // system 类型：居中灰胶囊，无头像无气泡
	MD   bool                // markdown 类型：整段按 markdown 渲染（标题/代码块/列表…）
	CD   *qapi.CardInfo      // card 类型：bot 交互卡片（只读展示）
	NW   *qapi.NewsInfo      // news 类型：资讯卡，可点链接
	MI   *qapi.MiniAppInfo   // miniApp 类型：小程序分享卡
	Segs []qapi.ContentSeg   // text 消息按 @ token 切好的片段（富文本渲染）
	MF   *qapi.MergedForward // merged_forward 解析结果
	SH   *qapi.ShareCard     // share 解析结果
	// 线格式原文（转发要保持原类型原 content）：
	RawType    string
	RawContent string
}

type MessageQuote struct {
	ID   string
	Name string
	Text string
}

func AvatarColor(seed string) string {
	if seed == "" {
		return avatarPalette[0]
	}
	var h uint32 = 2166136261
	for _, r := range seed {
		h = (h ^ uint32(r)) * 16777619
	}
	// 取高位，低位容易在短字符串上撞
	return avatarPalette[(h>>8)%uint32(len(avatarPalette))]
}

func KindLabel(kind string) string {
	switch kind {
	case "group":
		return "群聊"
	case "discussion":
		return "讨论组"
	case "bot":
		return "机器人"
	case "single":
		return "单聊"
	case "":
		return "未知"
	default:
		return kind
	}
}

func ConvTimeLabel(ms int64) string {
	if ms <= 0 {
		return ""
	}
	t := time.UnixMilli(ms)
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "刚刚"
	case diff < time.Hour:
		return fmt.Sprintf("%d分钟前", int(diff/time.Minute))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%d小时前", int(diff/time.Hour))
	case diff < 48*time.Hour:
		return "昨天"
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%d天前", int(diff/(24*time.Hour)))
	default:
		return t.Format("01-02")
	}
}

func StampLabel(ms int64) string {
	if ms <= 0 {
		return ""
	}
	return time.UnixMilli(ms).Format("15:04")
}

func ConvsFromAPI(list []qapi.Conversation) []Conversation {
	out := make([]Conversation, 0, len(list))
	for _, cv := range list {
		name := cv.DisplayName()
		// 预览对齐 formatMessagePreview：没有消息给「暂无消息」，
		// 群聊/讨论组在正文前拼发送者名（「张三: 收到」），单聊不拼。
		// 前缀只在消息真的带 sender 时加——原版按 sender 对象取值，
		// 缺字段时宁可不加，也不要出现「未知用户: 早」这种兜底名。
		preview := "暂无消息"
		if last := cv.Last(); last != nil {
			text := last.Preview()
			if text != "" {
				preview = text
				if cv.Type == "group" || cv.Type == "discussion" {
					if last.Sender != nil && last.Sender.Name != "" {
						preview = last.Sender.Name + ": " + text
					}
				}
			}
		}
		// 成员数只在群聊且成员已加载时显示（对齐
		// `Conversation.type === 'group' && Conversation.members`），
		// 由 convSubLabel 在渲染时算，这里不再重复。
		out = append(out, Conversation{
			ID:           cv.ID.String(),
			Name:         name,
			Preview:      preview,
			Time:         ConvTimeLabel(cv.Timestamp),
			Unread:       cv.UnreadCount,
			Color:        AvatarColor(cv.ID.String() + name),
			Online:       cv.Status == "online",
			Kind:         cv.Type,
			Stamp:        cv.Timestamp,
			Muted:        cv.Muted,
			Pinned:       cv.PinnedNow(),
			GroupID:      cv.GroupID,
			Announcement: cv.Announcement,
			Avatar:       cv.Avatar,
			IP:           cv.IP,
			Signature:    cv.Signature,
			Status:       cv.Status,
			PeerID:       cv.OtherMemberID,
			PeerType:     cv.OtherMemberType,
		})
	}
	return out
}

func ConvSubLabel(cv Conversation) string {
	if cv.Sub != "" {
		return cv.Sub
	}
	if cv.Kind == "group" && len(cv.Members) > 0 {
		return fmt.Sprintf("(%d人)", len(cv.Members))
	}
	return ""
}

func OneLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.ReplaceAll(s, "\t", " ")
}

func MembersFromAPI(list []qapi.ConvMember) []Member {
	out := make([]Member, 0, len(list))
	for _, m := range list {
		out = append(out, Member{
			UserID: m.UserID,
			Name:   m.DisplayName(),
			Avatar: m.User.Avatar,
			Role:   m.Role,
		})
	}
	return out
}

func MessageKey(id string, i int) string {
	if id != "" {
		return "m-" + id
	}
	return fmt.Sprintf("i-%d", i)
}

func RecalledLabel(mine bool, sender string) string {
	if mine {
		return "你撤回了一条消息"
	}
	name := sender
	if name == "" {
		name = "对方"
	}
	return name + " 撤回了一条消息"
}

func MessageFromAPI(m qapi.Message, selfID string, i int) Message {
	// Preview 已经把 @ 提及 token 解码成 "@名字"，这里不能再用原始 Content，
	// 否则气泡里会露出 @{mention:...} 这种内部格式。
	text := m.Preview()
	mine := m.IsMine(selfID)
	sys := m.Type == "system"
	if m.IsRecalled {
		// 对齐 qim-client：自己的消息显示「你撤回了一条消息」，别人的显示发送者名。
		text = RecalledLabel(mine, m.SenderName())
	}
	if text == "" {
		text = "[" + m.Type + "]"
	}

	// 富媒体 content 是 {"url":...,"id":...} 的 JSON，直接当正文会露出一串
	// 协议字段。这里转成附件结构 + 一句可读摘要（复制、引用、会话预览都用
	// 这句），气泡本体交给 imageBubble / fileBubble 渲染。
	kind, att := "", qapi.Attachment{}
	if qapi.IsMediaType(m.Type) && !m.IsRecalled {
		kind = strings.ToLower(m.Type)
		att = qapi.ParseAttachment(m.Content, &m)
		if att.IsZero() {
			kind = "" // 异常 content 退化成普通文本，别渲染一个空卡片
		} else {
			text = MediaSummary(kind, att)
			// 历史消息里可能只有 URL、没有 mime；渲染时以 kind 为准，
			// IsImage() 只用于本地选文件时的类型判断，不覆盖原字段。
		}
	}

	// 文本消息按 @ token 切片段（富文本渲染 @ 高亮 chip）；system 消息
	// 内容同样可能有 token，切片段无妨（胶囊里只显示纯文本）。
	var segs []qapi.ContentSeg
	if (m.Type == "text" || sys) && !m.IsRecalled && m.Content != "" {
		segs = qapi.DecodeMentionSegments(m.Content)
	}

	// 合并转发 / 分享卡片：解析失败时留在普通气泡里给可读短标签，
	// 别渲染一个空卡片。
	var mf *qapi.MergedForward
	if m.Type == "merged_forward" && !m.IsRecalled {
		mf = qapi.ParseMergedForward(m.Content)
	}
	var sh *qapi.ShareCard
	if m.Type == "share" && !m.IsRecalled {
		sh = qapi.ParseShareCard(m.Content)
	}
	// bot 卡片 / 资讯 / 小程序分享：只读展示，解析失败退化普通气泡。
	var cd *qapi.CardInfo
	if m.Type == "card" && !m.IsRecalled {
		cd = qapi.ParseCardInfo(m.Content)
	}
	var nw *qapi.NewsInfo
	if m.Type == "news" && !m.IsRecalled {
		nw = qapi.ParseNewsInfo(m.Content)
	}
	var mi *qapi.MiniAppInfo
	if m.Type == "miniApp" && !m.IsRecalled {
		mi = qapi.ParseMiniAppInfo(m.Content)
	}

	// 引用有两种形状：展开后的 quoted_message，或只有 quoted_message_id
	// （原消息被删/接口没 preload）。后者至少保留一条占位，不把引用弄丢。
	quoteID, quoteSender, quoteText := "", "", ""
	if q := m.QuotedMessage; q != nil {
		quoteID = q.ID.String()
		quoteSender = q.SenderName()
		quoteText = q.Preview()
		if quoteText == "" {
			quoteText = "[" + q.Type + "]"
		}
		if q.IsRecalled {
			quoteText = "该消息已撤回"
		} else if qapi.IsMediaType(q.Type) {
			if qa := qapi.ParseAttachment(q.Content, q); !qa.IsZero() {
				quoteText = MediaSummary(strings.ToLower(q.Type), qa)
			}
		}
	} else if m.QuotedMessageID.String() != "" {
		quoteID, quoteText = m.QuotedMessageID.String(), "原消息已不可见"
	}

	senderAvatar := ""
	senderID := m.SenderID.String()
	if m.Sender != nil {
		senderAvatar = m.Sender.Avatar
		if senderID == "" {
			senderID = m.Sender.ID.String()
		}
	}
	return Message{
		Key:         MessageKey(m.ID.String(), i),
		Mine:        mine,
		Sender:      m.SenderName(),
		SenderID:    senderID,
		Avatar:      senderAvatar,
		Text:        text,
		MD:          m.Type == "markdown" && !m.IsRecalled && m.Content != "",
		At:          StampLabel(m.Timestamp),
		Stamp:       m.Timestamp,
		Failed:      m.IsFailed,
		ServerID:    m.ID.String(),
		QuoteID:     quoteID,
		QuoteSender: quoteSender,
		QuoteText:   quoteText,
		AtMe:        !mine && m.MentionsMe(selfID),
		Recalled:    m.IsRecalled,
		OrigText:    m.RecallOriginal(),
		Kind:        kind,
		Att:         att,
		Sys:         sys,
		Segs:        segs,
		MF:          mf,
		SH:          sh,
		CD:          cd,
		NW:          nw,
		MI:          mi,
		RawType:     m.Type,
		RawContent:  m.Content,
		// 后端 is_read 对自己发的消息表示「对方已读」，对别人发的消息
		// 表示「我是否已读」，两种语义共用同一字段，这里只取前者。
		Read: mine && m.IsRead,
	}
}

func MediaSummary(kind string, att qapi.Attachment) string {
	label := map[string]string{
		"image": "[图片]", "file": "[文件]", "video": "[视频]", "audio": "[语音]",
	}[kind]
	if label == "" {
		label = "[附件]"
	}
	name := att.DisplayName()
	if size := qapi.FormatSize(att.Size); size != "" {
		return label + " " + name + " · " + size
	}
	return label + " " + name
}

func InsertUnreadDivider(msgs []Message, unread int) []Message {
	if unread <= 0 || len(msgs) == 0 {
		return msgs
	}
	at, firstReal, count := -1, -1, 0
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Pill != "" || msgs[i].Divider {
			continue
		}
		if firstReal < 0 || i < firstReal {
			firstReal = i
		}
		count++
		if count == unread {
			at = i
			break
		}
	}
	if at < 0 {
		at = firstReal
	}
	if at < 0 {
		return msgs
	}
	out := make([]Message, 0, len(msgs)+1)
	out = append(out, msgs[:at]...)
	out = append(out, Message{Key: "unread-divider", Divider: true, Text: "以下为新消息"})
	return append(out, msgs[at:]...)
}

func NeedsTimeDivider(prev, cur int64) bool {
	if prev <= 0 || cur <= 0 {
		return true
	}
	if cur-prev > 5*60*1000 {
		return true
	}
	pt, ct := time.UnixMilli(prev), time.UnixMilli(cur)
	py, pm, pd := pt.Date()
	cy, cm, cd := ct.Date()
	return py != cy || pm != cm || pd != cd
}

func TimeDividerLabel(ms int64) string {
	t := time.UnixMilli(ms)
	now := time.Now()
	y, m, d := t.Date()
	ny, nm, nd := now.Date()
	hm := t.Format("15:04")
	if y == ny && m == nm && d == nd {
		return hm
	}
	yy, ym, yd := now.AddDate(0, 0, -1).Date()
	if y == yy && m == ym && d == yd {
		return "昨天 " + hm
	}
	if delta := now.Sub(t); delta > 0 && delta < 7*24*time.Hour {
		weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
		return weekdays[int(t.Weekday())] + " " + hm
	}
	return t.Format("2006-01-02 15:04")
}

func MsgsFromAPI(list []qapi.Message, selfID string) []Message {
	out := make([]Message, 0, len(list)+2)
	var lastStamp int64
	for i, m := range list {
		if m.Timestamp > 0 && NeedsTimeDivider(lastStamp, m.Timestamp) {
			out = append(out, Message{
				Key:   fmt.Sprintf("time-%d", m.Timestamp),
				Pill:  TimeDividerLabel(m.Timestamp),
				Stamp: m.Timestamp,
			})
		}
		if m.Timestamp > 0 {
			lastStamp = m.Timestamp
		}
		out = append(out, MessageFromAPI(m, selfID, i))
	}
	return out
}

func PrependOlder(existing, older []Message) []Message {
	seen := make(map[string]bool, len(existing))
	hasOlderPill := false
	for _, m := range older {
		if m.Pill != "" {
			hasOlderPill = true
		}
	}
	for _, m := range existing {
		if m.Pill == "" && !m.Divider {
			seen[m.Key] = true
		}
	}
	out := make([]Message, 0, len(older)+len(existing))
	for _, m := range older {
		if m.Pill == "" && !m.Divider && seen[m.Key] {
			continue
		}
		out = append(out, m)
	}
	for i, m := range existing {
		if i == 0 && m.Pill != "" && hasOlderPill {
			continue
		}
		out = append(out, m)
	}
	return out
}

func OldestServerID(msgs []Message) string {
	best := uint64(0)
	for _, m := range msgs {
		if m.Pill != "" || !strings.HasPrefix(m.Key, "m-") {
			continue
		}
		id, err := strconv.ParseUint(strings.TrimPrefix(m.Key, "m-"), 10, 64)
		if err != nil {
			continue
		}
		if best == 0 || id < best {
			best = id
		}
	}
	if best == 0 {
		return ""
	}
	return strconv.FormatUint(best, 10)
}

func (cv Conversation) IsGroupish() bool {
	return cv.Kind == "group" || cv.Kind == "discussion"
}

func (c Conversation) Initial() string {
	return FirstRune(c.Name)
}

// FirstRune 取首个字符，空串时兜底成 "?"。
func FirstRune(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return "?"
	}
	return string(r[0])
}
