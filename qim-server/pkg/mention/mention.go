// Package mention 提供 @ 提及的解析、序列化与展示工具。
//
// content 中提及以 token 内嵌：
//   @{mention:<id>|<urlencoded name>}   @ 单人
//   @{mention:all|所有人}               @ 所有人
//
// content 是唯一事实源。MentionUserIDs 等派生信息读取时实时计算，不落库。
package mention

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// tokenRegex 匹配 content 中的 mention token。
//   group 1: target — "all" 或纯数字 user id
//   group 2: name   — urlencoded 显示名（可缺失）
var tokenRegex = regexp.MustCompile(`@\{mention:(all|[1-9]\d*)(?:\|([^}]*))?\}`)

// Mention 表示一次提及。
type Mention struct {
	Target string // "all" 或 user id 字符串
	UserID uint   // 0 表示 @all
	All    bool
	Name   string // decode 后的显示名
	Start  int    // token 在 content 中的字节偏移
	End    int
}

// Parse 解析 content 中的所有 mention token。
func Parse(content string) []Mention {
	matches := tokenRegex.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]Mention, 0, len(matches))
	for _, m := range matches {
		start, end := m[0], m[1]
		targetStart, targetEnd := m[2], m[3]
		nameStart, nameEnd := m[4], m[5]

		target := content[targetStart:targetEnd]
		var name string
		if nameStart >= 0 && nameEnd > nameStart {
			if decoded, err := url.QueryUnescape(content[nameStart:nameEnd]); err == nil {
				name = decoded
			} else {
				name = content[nameStart:nameEnd]
			}
		}

		ment := Mention{
			Target: target,
			Name:   name,
			Start:  start,
			End:    end,
		}
		if target == "all" {
			ment.All = true
		} else {
			if id, err := strconv.ParseUint(target, 10, 64); err == nil {
				ment.UserID = uint(id)
			}
		}
		out = append(out, ment)
	}
	return out
}

// ExtractUserIDs 将 mentions 展开为用户 ID 列表。
// @all 展开为 allMemberIDs；普通 mention 取 UserID。
// 结果去重、排除 0、排除 excludeUserID（通常为发送者自己）。
func ExtractUserIDs(mentions []Mention, allMemberIDs []uint, excludeUserID uint) []uint {
	seen := make(map[uint]struct{}, len(mentions)+len(allMemberIDs))
	result := make([]uint, 0, len(mentions)+len(allMemberIDs))

	add := func(id uint) {
		if id == 0 {
			return
		}
		if id == excludeUserID {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}

	for _, m := range mentions {
		if m.All {
			for _, id := range allMemberIDs {
				add(id)
			}
		} else {
			add(m.UserID)
		}
	}
	return result
}

// IsMentioned 判断 userID 是否被提及（不展开 @all）。
// 用于 @all 触发场景外的单点判断。
func IsMentioned(mentions []Mention, userID uint) bool {
	for _, m := range mentions {
		if !m.All && m.UserID == userID {
			return true
		}
	}
	return false
}

// IsAllMentioned 判断是否包含 @all。
func IsAllMentioned(mentions []Mention) bool {
	for _, m := range mentions {
		if m.All {
			return true
		}
	}
	return false
}

// Encode 生成 @ 单人的 token。
func Encode(userID uint, name string) string {
	return "@{mention:" + strconv.FormatUint(uint64(userID), 10) + "|" + url.QueryEscape(name) + "}"
}

// EncodeAll 生成 @ 所有人 的 token。
func EncodeAll() string {
	return "@{mention:all|" + url.QueryEscape("所有人") + "}"
}

// Decode 将 content 中的 mention token 替换为 "@姓名" 纯文本。
// token 缺失 name 时：@all 显示为 "@所有人"，@id 显示为原 token。
func Decode(content string) string {
	return tokenRegex.ReplaceAllStringFunc(content, func(token string) string {
		sub := tokenRegex.FindStringSubmatch(token)
		if len(sub) < 3 {
			return token
		}
		target, encodedName := sub[1], sub[2]
		if target == "all" {
			if encodedName == "" {
				return "@所有人"
			}
			if name, err := url.QueryUnescape(encodedName); err == nil {
				return "@" + name
			}
			return "@" + encodedName
		}
		if encodedName == "" {
			return token
		}
		if name, err := url.QueryUnescape(encodedName); err == nil {
			return "@" + name
		}
		return "@" + encodedName
	})
}

// HasAnyMention 判断 content 是否包含任何 mention token。
func HasAnyMention(content string) bool {
	return tokenRegex.MatchString(content)
}

// StripTokens 移除 content 中的所有 mention token，保留其余文本。
func StripTokens(content string) string {
	return tokenRegex.ReplaceAllString(content, "")
}

// DisplayName 返回单个 mention 的展示文本（不含 @ 前缀的 name 部分）。
// 用于 UI chip 渲染。
func (m Mention) DisplayName() string {
	if m.Name != "" {
		return m.Name
	}
	if m.All {
		return "所有人"
	}
	if m.UserID > 0 {
		return "用户" + strconv.FormatUint(uint64(m.UserID), 10)
	}
	return ""
}

// AtText 返回 mention 的展示文本（含 @ 前缀）。
func (m Mention) AtText() string {
	return "@" + m.DisplayName()
}

// ensure strings imported（用于未来可能的 strings 工具扩展）
var _ = strings.TrimSpace
