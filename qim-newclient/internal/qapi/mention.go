package qapi

// @ 提及在 content 里不是普通文本，而是内嵌 token：
//
//	@{mention:<id>|<urlencoded name>}   单人或
//	@{mention:all|所有人}                @ 所有人
//
// 规则来自 qim-server/pkg/mention，content 是唯一事实源（MentionUserIDs
// 只是读取时实时算出来的派生字段，不落库）。所以客户端要：
//   - 显示时把 token 换成 "@名字"，别把大括号原样给用户看；
//   - 发送时把用户选中的联系人编码回 token，后端才会真的发 @ 通知。
//
// 之前只把 content 当纯文本渲染，@ 消息会显示成一串乱码。

import (
	"net/url"
	"regexp"
	"strings"
)

var mentionToken = regexp.MustCompile(`@\{mention:(all|[1-9]\d*)(?:\|([^}]*))?\}`)

// Mention 是解码后的一次提及。Target 是 "all" 或用户 id 字符串。
type Mention struct {
	Target string
	Name   string
}

// DecodeMentions 把 content 里的 token 换成可读的 "@名字"。
// 后端可能省略名字（老数据），这时退化为 "@用户<id>" 而不是空白。
func DecodeMentions(content string) (string, []Mention) {
	matches := mentionToken.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content, nil
	}
	var b strings.Builder
	b.Grow(len(content))
	last := 0
	out := make([]Mention, 0, len(matches))
	for _, m := range matches {
		start, end := m[0], m[1]
		target := content[m[2]:m[3]]
		name := ""
		if m[4] >= 0 && m[5] > m[4] {
			if decoded, err := url.QueryUnescape(content[m[4]:m[5]]); err == nil {
				name = decoded
			} else {
				name = content[m[4]:m[5]]
			}
		}
		if name == "" {
			if target == "all" {
				name = "所有人"
			} else {
				name = "用户" + target
			}
		}
		b.WriteString(content[last:start])
		b.WriteString("@")
		b.WriteString(name)
		out = append(out, Mention{Target: target, Name: name})
		last = end
	}
	b.WriteString(content[last:])
	return b.String(), out
}

// ContentSeg 是把 content 按 @ token 切开后的片段：Mention 非 nil 表示
// 这段是一个 @ 提及（Text 已是 "@名字"），否则是普通文本。
type ContentSeg struct {
	Text    string
	Mention *Mention
}

// DecodeMentionSegments 与 DecodeMentions 同源，但保留片段边界，
// 供富文本渲染把 @ 提及做成高亮 chip 而不是混在正文里。
func DecodeMentionSegments(content string) []ContentSeg {
	matches := mentionToken.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		if content == "" {
			return nil
		}
		return []ContentSeg{{Text: content}}
	}
	out := make([]ContentSeg, 0, len(matches)*2+1)
	last := 0
	for _, m := range matches {
		start, end := m[0], m[1]
		target := content[m[2]:m[3]]
		name := ""
		if m[4] >= 0 && m[5] > m[4] {
			if decoded, err := url.QueryUnescape(content[m[4]:m[5]]); err == nil {
				name = decoded
			} else {
				name = content[m[4]:m[5]]
			}
		}
		if name == "" {
			if target == "all" {
				name = "所有人"
			} else {
				name = "用户" + target
			}
		}
		if start > last {
			out = append(out, ContentSeg{Text: content[last:start]})
		}
		out = append(out, ContentSeg{Text: "@" + name, Mention: &Mention{Target: target, Name: name}})
		last = end
	}
	if last < len(content) {
		out = append(out, ContentSeg{Text: content[last:]})
	}
	return out
}

// EncodeMention 生成单人提及 token。名字用 QueryEscape，和后端 url.QueryUnescape 对称。
func EncodeMention(userID, name string) string {
	return "@{mention:" + userID + "|" + url.QueryEscape(name) + "}"
}

// EncodeMentionAll 生成 @ 所有人 token。
func EncodeMentionAll() string {
	return "@{mention:all|" + url.QueryEscape("所有人") + "}"
}

// MentionsMe 判断这条消息是不是 @ 了我（或 @ 了所有人）。
// HTTP 历史接口会直接给 is_at_mention；WS 推送与搜索结果没有这个字段，
// 就从 content 里的 token 和 mention_user_ids 自己算，两条路都要走。
func (m *Message) MentionsMe(selfID string) bool {
	if m == nil {
		return false
	}
	if m.IsAtMention {
		return true
	}
	for _, id := range m.MentionUserIDs {
		if id.String() == selfID {
			return true
		}
	}
	_, mentions := DecodeMentions(m.Content)
	for _, mt := range mentions {
		if mt.Target == "all" || (selfID != "" && mt.Target == selfID) {
			return true
		}
	}
	return false
}
