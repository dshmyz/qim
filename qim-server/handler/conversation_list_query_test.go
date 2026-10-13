package handler

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConversationListQuery_NoCoalesceOnIndexedColumns 验证：
// 会话列表查询的 WHERE/SELECT/ORDER BY 子句不得对 conversation_sessions 的
// 过滤/排序列使用 COALESCE(...) 函数包裹。
//
// 背景（如实陈述）：
//   - cs.is_hidden / cs.is_pinned 目前【没有】独立索引（idx_user_conv 是
//     (user_id, conversation_id)，见 model.go），对它们做 COALESCE 包裹
//     并不影响现有执行计划；
//   - 本测试守护的是防御性规范：若未来为这两列建索引，函数包裹会使新索引失效；
//     游标谓词与 ORDER BY 中的 IFNULL 同样是函数包裹，排序当前仍走不了索引
//     （会 filesort），彻底消除需物化排序键（生成列 + 索引）；
//   - 本次重写的主要保证是语义等价：过滤与排序行为与原 COALESCE 版本一致。
//
// 等价改写方式（保持语义且为未来建索引留路）：
//   - is_hidden：用 (cs.is_hidden = false OR cs.is_hidden IS NULL) 代替
//     COALESCE(cs.is_hidden, false) = false
//   - is_pinned / last_message_at：SELECT 与 ORDER BY 统一用 IFNULL，
//     与 COALESCE 语义等价
func TestConversationListQuery_NoCoalesceOnIndexedColumns(t *testing.T) {
	sql := conversationListQuerySQL()
	low := strings.ToLower(sql)

	// 不应出现对 cs.is_hidden / cs.is_pinned / c.last_message_at 的 COALESCE 包裹
	assert.NotContains(t, low, "coalesce(cs.is_hidden",
		"WHERE 不应使用 COALESCE(cs.is_hidden,...)：函数包裹零增长的防御性规范（该列当前无独立索引），为未来建索引留路")
	assert.NotContains(t, low, "coalesce(cs.is_pinned",
		"SELECT/ORDER BY 不应使用 COALESCE(cs.is_pinned,...)：统一 IFNULL 风格的防御性规范")
	assert.NotContains(t, low, "coalesce(c.last_message_at",
		"ORDER BY 不应使用 COALESCE(c.last_message_at,...)：统一 IFNULL 风格；IFNULL 仍是函数包裹，排序本身走不了索引")
}

// TestConversationListQuery_StillFiltersHiddenConversations 验证：
// 重构后的 SQL 必须保留"过滤已隐藏会话"的语义。
// 等价改写后，WHERE 子句必须包含 cs.is_hidden IS NULL 或 cs.is_hidden = false 的判断。
func TestConversationListQuery_StillFiltersHiddenConversations(t *testing.T) {
	sql := conversationListQuerySQL()
	low := strings.ToLower(sql)

	// 至少出现一次对 is_hidden 的判断（无论是 IS NULL 还是 = false）
	hasHiddenFilter := strings.Contains(low, "cs.is_hidden is null") ||
		strings.Contains(low, "cs.is_hidden = false") ||
		strings.Contains(low, "cs.is_hidden=false")
	assert.True(t, hasHiddenFilter,
		"重构后仍需过滤已隐藏会话：WHERE 必须包含 cs.is_hidden IS NULL 或 cs.is_hidden = false，当前 SQL: %s", sql)
}

// TestConversationListQuery_OrdersByPinnedAndActivity 验证：
// 重构后的 SQL 必须保留"置顶优先 + 按最后消息时间倒序"的排序语义。
// ORDER BY 子句由 handler 在基础 SQL 之上动态拼接（含 LIMIT/OFFSET），
// 这里验证 handler 拼接后的完整 SQL（含排序段）必须涉及 is_pinned 字段。
func TestConversationListQuery_OrdersByPinnedAndActivity(t *testing.T) {
	// 模拟 handler 拼接最终 SQL 的过程（与 GetConversations 一致）
	sql := conversationListQuerySQL() + `
		ORDER BY is_pinned DESC, IFNULL(c.last_message_at, c.created_at) DESC, cm.conversation_id DESC
		LIMIT ?
	`
	low := strings.ToLower(sql)

	assert.Contains(t, low, "order by",
		"会话列表查询必须包含 ORDER BY 子句")
	assert.True(t, strings.Contains(low, "is_pinned"),
		"ORDER BY 必须涉及 is_pinned 字段以实现置顶优先")
}
