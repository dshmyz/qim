// Package aiprompt 提供 AI 系统提示词里「人设 + 回复规则」的单一来源。
//
// 群聊助手（service/smart_reply_graph.go）与私聊/群聊 AI（handler/prompt_builder.go）
// 曾各自复制一份「5 种性格人设 + 语言规则 + 长度规则 + 通用回复规则」，改一处漏三处
//（例如品牌名 QIM 就曾差点漏改）。本包把这几段抽成共享函数，两个调用方接入同一份，
// 保证口径一致。品牌名统一走 productname.Name。
//
// 长度档位枚举（存于 GroupAIConfig.MaxLength / 分身 ReplyStrategy.MaxReplyLength）：
//
//	short     简短
//	medium    适中（默认）
//	very_long 较长（6-10 句，新增）
//	long      详细（不限）
package aiprompt

import (
	"fmt"
	"github.com/dshmyz/qim/qim-server/pkg/productname"
	"strings"
	"time"
)

// CurrentTimeLine 返回统一格式的「当前时间」注入行，例如：【当前时间】2026-08-09 01:07 (日)。
// 全项目各 AI system prompt 统一走此单一来源，保证口径一致（与 prompt 段共用 aiprompt 包思路一致）。
func CurrentTimeLine() string {
	return FormatTimeLine(time.Now())
}

// FormatTimeLine 将指定时间格式化为统一的「当前时间」注入行（不含行尾换行）。
// 供需要注入特定时刻（而非当前时刻）的调用方使用，格式与 CurrentTimeLine 完全一致。
func FormatTimeLine(t time.Time) string {
	weekdays := []string{"日", "一", "二", "三", "四", "五", "六"}
	return fmt.Sprintf("【当前时间】%s (%s)", t.Format("2006-01-02 15:04"), weekdays[t.Weekday()])
}

// BuildPersona 返回性格对应的人设文案（不带行尾换行）。
// 支持 casual / concise / friendly / technical，其余一律返回专业严谨默认人设。
func BuildPersona(personality string) string {
	prefix := "你是 " + productname.Name + " 企业即时通讯系统中"
	switch personality {
	case "casual":
		return prefix + "的 AI 助手，性格轻松幽默。在回答中可以适当使用表情和emoji，语气活泼。"
	case "concise":
		return prefix + "的 AI 助手，风格简洁高效。回答直奔主题，不废话，只说重点。"
	case "friendly":
		return prefix + "的 AI 助手，性格温暖亲切。回答要有耐心，语气友善，像一个贴心的伙伴。"
	case "technical":
		return prefix + "的技术专家 AI 助手。回答要有技术深度，关注细节，必要时提供代码示例和技术方案。"
	default:
		return prefix + "的智能助手，风格专业严谨。回答要专业、客观、有条理。"
	}
}

// LanguageRule 返回语言规则文案；lang 为 zh / en，其余返回空串（不注入语言约束）。
func LanguageRule(lang string) string {
	switch lang {
	case "zh":
		return "请使用中文回答"
	case "en":
		return "Please answer in English"
	default:
		return ""
	}
}

// LengthRule 返回长度规则文案；maxLength 为 short / medium / very_long / long，其余返回空串。
func LengthRule(maxLength string) string {
	switch maxLength {
	case "short":
		return "回答要简短，控制在50字以内"
	case "medium":
		return "回答适中，控制在150字以内"
	case "very_long":
		return "回答较详细，控制在400字以内"
	case "long":
		return "回答详细，可以展开说明"
	default:
		return ""
	}
}

// ReplyRules 返回通用回复规则（知识库优先 + 兜底说明 + 不手工 @ 提及）。
// 这部分与语言/长度无关，各场景通用。
func ReplyRules() []string {
	return []string{
		"优先使用知识库中的内容回答",
		"如果知识库中没有相关内容，使用你的通用知识回答，但明确说明\"以下回答基于通用知识，建议核实\"",
		"回复中不要使用 @用户名 或 @任何人 的格式，不要 @ 提及任何群成员，系统会自动处理提及；直接称呼对方名字即可，不要加 @ 前缀",
	}
}

// JoinRules 将规则切片拼接为「\n- rule」列表块，追加到现有输出。
func JoinRules(sb *strings.Builder, rules []string) {
	for _, r := range rules {
		sb.WriteString("\n- " + r)
	}
}
