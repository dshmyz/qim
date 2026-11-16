// Package qapi 是 qim 服务端的 Go 客户端。
//
// 它的存在意义只有一个：让 Go 成为客户端唯一的数据源。所有会话和消息
// 都由这个包直连 qim-server：登录、拉列表、拉历史、发消息、收推送，
// 原生界面只读它归一化后的结果，不再依赖任何网页层。
//
// 协议契约照 qim-client 的 src/api/*.ts 抄写，字段名保持后端原样的
// snake_case，不做重命名——多一层映射就多一处对不上的风险。
package qapi

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// normalizeStamp 把秒级时间戳统一成毫秒。
// 后端不同接口混用秒（time.Now().Unix()）和毫秒（前端 Date.now()），
// 在解析期一次抹平，UI 只认毫秒，免得每处显示都判断单位。
func normalizeStamp(v int64) int64 {
	if v <= 0 {
		return 0
	}
	if v < 1e12 {
		return v * 1000
	}
	return v
}

// parseWireTime 解析后端的 created_at / last_message_at。
// 正常是 RFC3339 字符串，但历史版本也返回过秒/毫秒数字，统一收下并返回毫秒。
func parseWireTime(raw json.RawMessage) int64 {
	if len(raw) == 0 || string(raw) == "null" {
		return 0
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
			if t, err := time.Parse(layout, s); err == nil {
				return t.UnixMilli()
			}
		}
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return normalizeStamp(n)
		}
		return 0
	}
	var n int64
	if err := json.Unmarshal(raw, &n); err == nil {
		return normalizeStamp(n)
	}
	return 0
}

// ID 兼容后端把 id 返回成数字或字符串两种情况。
// 网页侧到处 .toString() 就是这个原因；Go 里在解析期一次性抹平。
type ID string

func (i *ID) UnmarshalJSON(b []byte) error {
	s := string(b)
	switch s {
	case "null", `""`:
		*i = ""
		return nil
	}
	if s[0] == '"' {
		var str string
		if err := json.Unmarshal(b, &str); err != nil {
			return err
		}
		*i = ID(str)
		return nil
	}
	*i = ID(s)
	return nil
}

func (i ID) String() string { return string(i) }

func (i ID) Empty() bool { return i == "" }

// APIError 是服务端返回的业务错误。code != 0 即失败，
// 即使 HTTP 状态是 200——这是该后端的约定。
type APIError struct {
	Code    int
	Message string
	Detail  json.RawMessage
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("服务端错误 %d: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("服务端错误 %d", e.Code)
}

// IsTwoFactorRequired 判断是不是「需要二次验证」而不是登录失败。
// 网页侧靠 code 1002 + detail.two_factor_required 区分，并弹验证码框。
func (e *APIError) IsTwoFactorRequired() bool {
	if e.Code != 1002 || len(e.Detail) == 0 {
		return false
	}
	var d struct {
		TwoFactorRequired bool   `json:"two_factor_required"`
		Session           string `json:"session"`
	}
	if err := json.Unmarshal(e.Detail, &d); err != nil {
		return false
	}
	return d.TwoFactorRequired
}

// TwoFactorSession 取二次验证的会话标识，配合 /auth/2fa/verify 使用。
func (e *APIError) TwoFactorSession() string {
	var d struct {
		Session string `json:"session"`
	}
	_ = json.Unmarshal(e.Detail, &d)
	return d.Session
}

// User 是登录返回的用户对象。后端在不同接口里给的字段不太一致
// （nickname / name / username 都出现过），DisplayName 负责兜底。
type User struct {
	ID         ID     `json:"id"`
	Username   string `json:"username"`
	Nickname   string `json:"nickname"`
	Name       string `json:"name"`
	Avatar     string `json:"avatar"`
	Email      string `json:"email"`
	Status     string `json:"status"`
	RealName   string `json:"real_name"`
	Signature  string `json:"signature"`
	Phone      string `json:"phone"`
	Gender     string `json:"gender"`
	Type       string `json:"type"`
	IP         string `json:"ip"`
	Department string `json:"department"`
	Position   string `json:"position"`
	CreatedAt  string `json:"created_at"`
}

// DisplayName 按「昵称 → 名字 → 用户名」的优先级取显示名，
// 和网页侧 shell 里的取法保持一致。
func (u User) DisplayName() string {
	for _, s := range []string{u.Nickname, u.Name, u.Username} {
		if s != "" {
			return s
		}
	}
	if !u.ID.Empty() {
		return u.ID.String()
	}
	return "未登录"
}

// Session 是登录成功后持有的凭据集合。
type Session struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

// Sender 是消息里内嵌的发送者信息。
type Sender struct {
	ID     ID     `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// UnmarshalJSON 兼容后端把发送者名字放在 nickname / username 的情况。
// qim-server 的 model.User 用的是 nickname，前端归一化时才改成 name。
func (s *Sender) UnmarshalJSON(b []byte) error {
	type plain Sender
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	*s = Sender(p)
	if s.Name != "" {
		return nil
	}
	var w struct {
		Nickname string `json:"nickname"`
		Username string `json:"username"`
	}
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	for _, v := range []string{w.Nickname, w.Username} {
		if v != "" {
			s.Name = v
			return nil
		}
	}
	return nil
}

// Message 对应后端消息体。
// 字段与 qim-client src/api/message.ts 的接口一一对应。
type Message struct {
	ID             ID       `json:"id"`
	ConversationID ID       `json:"conversation_id"`
	SenderID       ID       `json:"sender_id"`
	Sender         *Sender  `json:"sender"`
	Type           string   `json:"type"`
	Content        string   `json:"content"`
	Extra          string   `json:"extra"`
	FileName       string   `json:"file_name"`
	FileSize       int64    `json:"file_size"`
	FileURL        string   `json:"file_url"`
	ThumbnailURL   string   `json:"thumbnail_url"`
	Timestamp      int64    `json:"timestamp"`
	IsRecalled     bool     `json:"isRecalled"`
	IsSelf         bool     `json:"isSelf"`
	IsRead         bool     `json:"isRead"`
	IsFailed       bool     `json:"isFailed"`
	QuotedMessage  *Message `json:"quotedMessage"`
	// 引用消息在响应里有两种形状：完整对象 quoted_message，或只有
	// quoted_message_id（被引用消息已删/接口没 preload）。后者至少能让
	// 气泡显示「引用了一条消息」而不是把引用弄丢。
	QuotedMessageID ID   `json:"quoted_message_id"`
	MentionUserIDs  []ID `json:"mention_user_ids"`
	IsAtMention     bool `json:"is_at_mention"`
}

// UnmarshalJSON 把后端的原始线格式与前端归一化格式都收进同一个结构。
//
// qim-server 的 BuildMessageResponse 返回的是 snake_case + RFC3339 时间：
// created_at / is_recalled / is_read / is_failed / quoted_message / sender.nickname，
// 而早期接口与网页内部模型用的是 timestamp / isRecalled / isSelf / sender.name。
// 两边都收，解析期归一化，UI 层就只有一种形状。
func (m *Message) UnmarshalJSON(b []byte) error {
	type plain Message
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	*m = Message(p)
	m.Timestamp = normalizeStamp(m.Timestamp)

	var w struct {
		CreatedAt   json.RawMessage `json:"created_at"`
		Recalled    *bool           `json:"is_recalled"`
		Read        *bool           `json:"is_read"`
		Failed      *bool           `json:"is_failed"`
		Quote       *Message        `json:"quoted_message"`
		FileName    string          `json:"file_name"`
		FileSize    int64           `json:"file_size"`
		FileURL     string          `json:"file_url"`
		Thumbnail   string          `json:"thumbnail_url"`
		IsStreaming bool            `json:"is_streaming"`
	}
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	if t := parseWireTime(w.CreatedAt); t > 0 {
		m.Timestamp = t
	}
	if w.Recalled != nil {
		m.IsRecalled = *w.Recalled
	}
	if w.Read != nil {
		m.IsRead = *w.Read
	}
	if w.Failed != nil {
		m.IsFailed = *w.Failed
	}
	if w.Quote != nil && m.QuotedMessage == nil {
		m.QuotedMessage = w.Quote
	}
	if w.FileName != "" {
		m.FileName = w.FileName
	}
	if w.FileSize != 0 {
		m.FileSize = w.FileSize
	}
	if w.FileURL != "" {
		m.FileURL = w.FileURL
	}
	if w.Thumbnail != "" {
		m.ThumbnailURL = w.Thumbnail
	}
	if w.IsStreaming {
		m.Type = "streaming"
	}
	return nil
}

// IsMine 判断这条消息是不是当前用户发的。
// 后端不下发 isSelf，只有 sender_id / sender.id，所以由客户端拿登录返回的
// user.id 比对；保留 isSelf 是为了兼容已经归一化过的数据源。
func (m *Message) IsMine(selfID string) bool {
	if m == nil {
		return false
	}
	if m.IsSelf {
		return true
	}
	if selfID == "" {
		return false
	}
	if m.SenderID.String() == selfID {
		return true
	}
	return m.Sender != nil && m.Sender.ID.String() == selfID
}

// SenderName 返回气泡上方要显示的名字，缺失时给个可读兜底。
func (m *Message) SenderName() string {
	if m == nil {
		return ""
	}
	if m.Sender != nil && m.Sender.Name != "" {
		return m.Sender.Name
	}
	if m.SenderID.String() != "" {
		return "用户 " + m.SenderID.String()
	}
	return "未知用户"
}

// Preview 返回会话列表里那一行摘要文本。
// 空内容（图片/文件消息）退化成类型名，避免列表出现空白行。
// RecallOriginal 从 Extra 里取撤回前的原文（extra.original_content）。
// 后端撤回时把原内容存进 Extra 供「撤回后重新编辑」；非撤回消息或解析
// 失败都返回空串。
func (m *Message) RecallOriginal() string {
	if m == nil || m.Extra == "" {
		return ""
	}
	var extra struct {
		OriginalContent string `json:"original_content"`
	}
	if err := json.Unmarshal([]byte(m.Extra), &extra); err != nil {
		return ""
	}
	return extra.OriginalContent
}

func (m *Message) Preview() string {
	if m == nil {
		return ""
	}
	switch m.Type {
	case "merged_forward":
		return MergedForwardLabel(m.Content)
	case "share":
		if name := ShareCardName(m.Content); name != "" {
			return "[分享] " + name
		}
		return "[分享]"
	case "system":
		text, _ := DecodeMentions(m.Content)
		return text
	case "markdown":
		// 对齐 qim-client：列表预览剥掉 markdown 语法符号，只留正文
		//（气泡内的完整渲染走客户端自己的 markdown 管线）。
		if m.Content != "" {
			text, _ := DecodeMentions(m.Content)
			return StripMarkdown(text)
		}
		return "[富文本]"
	case "card":
		// 对齐 messageDisplay：卡片预览优先 title/text，别露 JSON。
		if ci := ParseCardInfo(m.Content); ci != nil {
			if ci.Title != "" {
				return "[卡片] " + ci.Title
			}
			return "[卡片] " + ci.Text
		}
		return "[卡片]"
	case "news":
		if ni := ParseNewsInfo(m.Content); ni != nil && ni.Title != "" {
			return "[资讯] " + ni.Title
		}
		return "[资讯]"
	case "miniApp":
		if mi := ParseMiniAppInfo(m.Content); mi != nil {
			return "[小程序] " + mi.Label()
		}
		return "[小程序]"
	}
	if m.Content != "" {
		// @ 提及在 content 里是 token，列表预览也要解码成人看得懂的 "@名字"
		text, _ := DecodeMentions(m.Content)
		return text
	}
	switch m.Type {
	case "image":
		return "[图片]"
	case "file":
		return "[文件] " + m.FileName
	case "video":
		return "[视频]"
	case "audio":
		return "[语音]"
	case "markdown":
		return "[富文本]"
	case "card":
		return "[卡片]"
	case "news":
		return "[图文]"
	case "miniApp":
		return "[小程序]"
	}
	return ""
}

// Conversation 对应后端会话对象。
//
// 注意 lastMessage 在网页侧是 camelCase，而 unread_count 是 snake_case——
// 后端确实混用，所以这里两个都收，Last() 取有值的那个。
type Conversation struct {
	ID           ID       `json:"id"`
	Name         string   `json:"name"`
	Avatar       string   `json:"avatar"`
	Type         string   `json:"type"` // single / group / bot / discussion
	UnreadCount  int      `json:"unread_count"`
	Timestamp    int64    `json:"timestamp"`
	LastMessage  *Message `json:"lastMessage"`
	LastMessageS *Message `json:"last_message"`
	Muted        bool     `json:"muted"`
	Pinned       bool     `json:"pinned"`
	IsPinned     bool     `json:"is_pinned"`
	Status       string   `json:"status"`
	Announcement string   `json:"announcement"`
	OtherName    string   `json:"other_member_name"`
	// 单聊头部副标题 / 头像角标用：后端 GetConversations 对 single 会话
	// 会把对端 User 的 IP/Signature/Status 摊平到会话对象上（bot 会话没有）。
	IP              string       `json:"ip"`
	Signature       string       `json:"signature"`
	OtherMemberID   uint         `json:"other_member_id"`
	OtherMemberType string       `json:"other_member_type"`
	IsDeleted       bool         `json:"is_deleted"`
	GroupID         uint         `json:"group_id"`
	Members         []ConvMember `json:"members"`
}

// ConvMember 是会话成员（群聊详情里带）。@ 候选、成员列表、群公告都靠它。
// User 嵌套在成员里，昵称/头像都从 User 取；Role 决定是不是群主/管理员。
type ConvMember struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"` // owner / admin / member
	User   struct {
		ID       uint   `json:"id"`
		Nickname string `json:"nickname"`
		Username string `json:"username"`
		Avatar   string `json:"avatar"`
	} `json:"user"`
}

// DisplayName 返回成员在 @ 候选里要展示的名字，优先昵称。
func (m ConvMember) DisplayName() string {
	for _, s := range []string{m.User.Nickname, m.User.Username} {
		if s != "" {
			return s
		}
	}
	return "用户" + strconv.Itoa(int(m.UserID))
}

// ReadUser 是已读回执里的一条用户记录（GET /messages/:id/read-users）。
// 后端直接序列化 model.User，发送者本人不在列表里。
type ReadUser struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// DisplayName 优先昵称，回退用户名，再回退「用户+id」。
func (u ReadUser) DisplayName() string {
	for _, s := range []string{u.Nickname, u.Username} {
		if s != "" {
			return s
		}
	}
	return "用户" + strconv.Itoa(int(u.ID))
}

// MessageReadInfo 是已读详情响应：read_users + read_count + total_members。
type MessageReadInfo struct {
	ReadUsers    []ReadUser `json:"read_users"`
	ReadCount    int64      `json:"read_count"`
	TotalMembers int64      `json:"total_members"`
}

// UnmarshalJSON 兼容真实后端的会话结构。
//
// 两处关键差异：时间不是 timestamp 数字而是 last_message_at / created_at
// 的 RFC3339 字符串；置顶字段叫 is_pinned。解析后统一成 Timestamp（毫秒）。
func (c *Conversation) UnmarshalJSON(b []byte) error {
	type plain Conversation
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	*c = Conversation(p)
	c.Timestamp = normalizeStamp(c.Timestamp)

	var w struct {
		LastMessageAt json.RawMessage `json:"last_message_at"`
		CreatedAt     json.RawMessage `json:"created_at"`
		Pinned        *bool           `json:"is_pinned"`
	}
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	if c.Timestamp == 0 {
		if t := parseWireTime(w.LastMessageAt); t > 0 {
			c.Timestamp = t
		} else {
			c.Timestamp = parseWireTime(w.CreatedAt)
		}
	}
	if w.Pinned != nil {
		c.IsPinned = *w.Pinned
	}
	return nil
}

// DisplayName 返回侧栏要显示的名字。单聊后端可能只给 other_member_name。
func (c Conversation) DisplayName() string {
	for _, s := range []string{c.Name, c.OtherName} {
		if s != "" {
			return s
		}
	}
	if !c.ID.Empty() {
		return "会话 " + c.ID.String()
	}
	return "未命名会话"
}

// Last 返回最后一条消息（兼容两种命名）。
func (c Conversation) Last() *Message {
	if c.LastMessage != nil {
		return c.LastMessage
	}
	return c.LastMessageS
}

// PinnedNow 兼容 pinned / is_pinned 两种写法。
func (c Conversation) PinnedNow() bool { return c.Pinned || c.IsPinned }

// Notification 是通知中心里的一条通知（后端 model.Notification 直传）。
// 字段名保持 snake_case 原样，created_at 在解析期归一化成毫秒 Timestamp。
type Notification struct {
	ID            ID     `json:"id"`
	Type          string `json:"type"` // group_invitation / todo_assigned / system_message …
	Title         string `json:"title"`
	Content       string `json:"content"`
	Read          bool   `json:"read"`
	Priority      string `json:"priority"`
	ActionType    string `json:"action_type"` // accept_ignore / approve_reject / confirm_reschedule …
	ActionPayload string `json:"action_payload"`
	Pinned        bool   `json:"pinned"`
	Important     bool   `json:"important"`
	Handled       bool   `json:"handled"`
	// Timestamp 是 created_at（或历史版本的 timestamp 数字）归一化后的毫秒值。
	Timestamp int64 `json:"-"`
}

// UnmarshalJSON 兼容 created_at（RFC3339 字符串）与 timestamp（秒/毫秒数字）两种时间来源。
func (n *Notification) UnmarshalJSON(b []byte) error {
	type plain Notification
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	*n = Notification(p)
	var w struct {
		CreatedAt json.RawMessage `json:"created_at"`
		Stamp     int64           `json:"timestamp"`
	}
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	n.Timestamp = normalizeStamp(w.Stamp)
	if t := parseWireTime(w.CreatedAt); t > 0 {
		n.Timestamp = t
	}
	return nil
}

// ActionButtons 返回这条通知在通知中心里应显示的动作按钮。
// 与 qim-client NotificationCenter.vue 的 ACTION_BUTTONS 表一致；
// handled=true 表示已处理，不再出按钮。
func (n Notification) ActionButtons() []NotificationAction {
	if n.Handled {
		return nil
	}
	switch n.ActionType {
	case "accept_ignore":
		return []NotificationAction{{Value: "accept", Label: "接受"}, {Value: "ignore", Label: "忽略"}}
	case "confirm_reschedule":
		return []NotificationAction{{Value: "confirm", Label: "确认"}, {Value: "reschedule", Label: "延期"}}
	case "approve_reject":
		return []NotificationAction{{Value: "approve", Label: "同意"}, {Value: "reject", Label: "拒绝"}}
	case "todo_mass_confirm":
		return []NotificationAction{{Value: "confirm_create", Label: "确认创建"}, {Value: "todo_ignore", Label: "忽略"}}
	}
	return nil
}

// NotificationAction 是通知条目上的一个动作按钮。
type NotificationAction struct {
	Value string
	Label string
}

// NotificationPatch 是 PATCH /notifications/:id 的请求体。
// pinned / important 在后端是 toggle 语义：字段存在（传 true）即翻转。
type NotificationPatch struct {
	Read      *bool  `json:"read,omitempty"`
	Pinned    *bool  `json:"pinned,omitempty"`
	Important *bool  `json:"important,omitempty"`
	Action    string `json:"action,omitempty"`
}

// NotificationPage 是通知列表分页结果（data:{list,total,page}）。
type NotificationPage struct {
	List  []Notification `json:"list"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
}

// MessagePage 是历史消息分页结果。
type MessagePage struct {
	Messages   []Message `json:"messages"`
	Pagination struct {
		CurrentPage int `json:"current_page"`
		TotalPages  int `json:"total_pages"`
		Total       int `json:"total"`
		PageSize    int `json:"page_size"`
	} `json:"pagination"`
}

// SendRequest 是发送消息的请求体。
type SendRequest struct {
	Type            string `json:"type"`
	Content         string `json:"content"`
	QuotedMessageID int64  `json:"quoted_message_id,omitempty"`
	FileSize        int64  `json:"file_size,omitempty"`
	FileName        string `json:"file_name,omitempty"`
}

// TextRequest 是纯文本消息的快捷构造。
func TextRequest(text string) SendRequest {
	return SendRequest{Type: "text", Content: text}
}

// TextRequestWithQuote 构造一条引用回复。后端只认数字 id 且用
// quoted_message_id 字段名，非数字 id（本地乐观消息）直接退化成普通文本，
// 免得把整条消息发失败。
func TextRequestWithQuote(text, quotedID string) SendRequest {
	req := TextRequest(text)
	if n, err := strconv.ParseInt(strings.TrimSpace(quotedID), 10, 64); err == nil && n > 0 {
		req.QuotedMessageID = n
	}
	return req
}

// ---------- 频道（对齐 qim-client stores/channel.ts） ----------

// 频道发布/评论权限取值（与后端 handler 校验一致）。
const (
	ChannelPublishCreatorOnly  = "creator_only"    // 仅创建者可发
	ChannelPublishAllSubs      = "all_subscribers" // 所有订阅者可发
	ChannelCommentAllSubs      = "all_subscribers" // 所有订阅者可评论
	ChannelCommentDisabled     = "disabled"        // 关闭评论
	ChannelStatusActive        = "active"
	ChannelStatusPending       = "pending"
	ChannelStatusRejected      = "rejected"
	ChannelNotifChannelMessage = "channel_message" // 频道消息通知的 type
)

// Channel 频道。GET /channels 返回的是 Channel + 订阅/统计扩展字段的合成结构。
type Channel struct {
	ID                json.Number `json:"id"`
	Name              string      `json:"name"`
	Description       string      `json:"description"`
	Avatar            string      `json:"avatar"`
	CreatorID         json.Number `json:"creator_id"`
	Status            string      `json:"status"`
	PublishPermission string      `json:"publish_permission"`
	CommentPermission string      `json:"comment_permission"`
	IsDefault         bool        `json:"is_default"`
	CreatedAt         string      `json:"created_at"`
	Creator           *User       `json:"creator,omitempty"`

	// 合成扩展字段（GetChannels 专属）
	IsSubscribed bool            `json:"is_subscribed"`
	MessageCount int             `json:"message_count"`
	LastMessage  *ChannelMessage `json:"last_message,omitempty"`

	// 本地维护：未读数（WS channel_message 通知驱动，不来自后端）
	Unread int `json:"-"`
}

// IsCreator 判断当前用户是不是频道创建者（对齐 qim-client 的 String()===String() 约定）。
func (ch Channel) IsCreator(selfID string) bool {
	return selfID != "" && ch.CreatorID.String() == selfID
}

// CanPublish 当前用户能否在此频道发消息（创建者/管理员豁免由上层判断角色，
// 这里只按 publish_permission 与订阅状态给出一般结论）。
func (ch Channel) CanPublish(selfID string, subscribed bool) bool {
	if ch.IsCreator(selfID) || ch.PublishPermission == ChannelPublishAllSubs {
		return subscribed || ch.IsCreator(selfID)
	}
	return false
}

// CanComment 频道是否允许评论。
func (ch Channel) CanComment() bool {
	return ch.CommentPermission != ChannelCommentDisabled
}

// ChannelMessage 频道消息（广播式动态，不是会话消息）。
type ChannelMessage struct {
	ID        json.Number `json:"id"`
	ChannelID json.Number `json:"channel_id"`
	SenderID  json.Number `json:"sender_id"`
	Content   string      `json:"content"`
	Type      string      `json:"type"`
	CreatedAt string      `json:"created_at"`
	Sender    *User       `json:"sender,omitempty"`

	// 本地维护：点赞状态（GET /likes 或本地乐观更新）
	LikeCount int  `json:"-"`
	Liked     bool `json:"-"`
}

// ChannelComment 频道消息评论。
type ChannelComment struct {
	ID        json.Number `json:"id"`
	MessageID json.Number `json:"message_id"`
	UserID    json.Number `json:"user_id"`
	Content   string      `json:"content"`
	CreatedAt string      `json:"created_at"`
	User      *User       `json:"user,omitempty"`
}

// ChannelLikeInfo 点赞状态（POST/DELETE/GET /channels/messages/:id/likes 的 data）。
type ChannelLikeInfo struct {
	LikeCount int  `json:"like_count"`
	IsLiked   bool `json:"is_liked"`
}

// ChannelCreateRequest 创建频道请求体。
type ChannelCreateRequest struct {
	Name              string `json:"name"`
	Description       string `json:"description,omitempty"`
	Avatar            string `json:"avatar,omitempty"`
	PublishPermission string `json:"publish_permission,omitempty"`
	CommentPermission string `json:"comment_permission,omitempty"`
}

// ChannelCreated 创建频道响应 data（含审批状态：无需审批=approved，需审批=pending）。
type ChannelCreated struct {
	ID             json.Number `json:"id"`
	Name           string      `json:"name"`
	Status         string      `json:"status"`
	ApprovalStatus string      `json:"approval_status"`
}

// Department 组织架构树节点（GET /organization/tree）。
// employees 是完整 User 数组；subDepartments 递归。parentId 可能缺失。
type Department struct {
	ID             json.Number  `json:"id"`
	Name           string       `json:"name"`
	ExternalID     string       `json:"externalId"`
	ParentID       *json.Number `json:"parentId"`
	Level          int          `json:"level"`
	Path           string       `json:"path"`
	SortOrder      int          `json:"sortOrder"`
	SubDepartments []Department `json:"subDepartments"`
	Employees      []User       `json:"employees"`
}

// IDString 部门 id 的字符串形式。
func (d Department) IDString() string { return d.ID.String() }

// WalkDeptChildren 先序遍历子部门（不含自身）。
func (d Department) WalkDeptChildren(fn func(Department)) {
	for _, sub := range d.SubDepartments {
		fn(sub)
		sub.WalkDeptChildren(fn)
	}
}

// CollectDeptEmployees 收集本部门及全部子部门的员工（去重，按 id）。
func (d Department) CollectDeptEmployees() []User {
	seen := map[string]bool{}
	var out []User
	var walk func(Department)
	walk = func(dep Department) {
		for _, e := range dep.Employees {
			id := e.ID.String()
			if !seen[id] {
				seen[id] = true
				out = append(out, e)
			}
		}
		for _, sub := range dep.SubDepartments {
			walk(sub)
		}
	}
	walk(d)
	return out
}

// UserStatusItem 单个用户的连接状态（GET /users/status）。
// status: online / offline（后端仅这两种）；last_online 是 unix 秒，0 表示从未在线。
type UserStatusItem struct {
	UserID     json.Number `json:"user_id"`
	Status     string      `json:"status"`
	LastOnline int64       `json:"last_online"`
}

// Online 判断是否在线。
func (s UserStatusItem) Online() bool { return s.Status == "online" }

// UpdateMeRequest 更新个人资料（PUT /users/me）。后端仅对非空字段生效。
type UpdateMeRequest struct {
	Nickname  string `json:"nickname,omitempty"`
	Avatar    string `json:"avatar,omitempty"`
	Signature string `json:"signature,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Email     string `json:"email,omitempty"`
}
