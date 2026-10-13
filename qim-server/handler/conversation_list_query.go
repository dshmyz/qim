package handler

// conversationListQuerySQL 返回会话列表查询的基础 SQL（不含动态过滤段与排序分页段）。
//
// 设计说明（如实陈述，语义与原 COALESCE 版本完全等价）：
//
//  1. WHERE 过滤已隐藏会话用 (cs.is_hidden = false OR cs.is_hidden IS NULL)
//     替代 COALESCE(cs.is_hidden, false) = false。is_hidden 目前没有独立索引
//     （idx_user_conv 是 (user_id, conversation_id)，见 model.go），两种写法
//     现阶段执行计划相同；改写是防御性规范——若未来为该列建索引，函数包裹会
//     使其失效。LEFT JOIN 后 cs 可能为 NULL（会话从未被该用户访问），
//     IS NULL 分支保证"未隐藏"语义与原 COALESCE 等价。
//
//  2. SELECT/ORDER BY/游标谓词统一用 IFNULL 替代 COALESCE，语义等价，风格统一。
//     注意：函数包裹下的排序/游标比较本身走不了索引（会 filesort），这一点
//     改写前后相同；彻底消除需物化排序键（如 last_message_at NOT NULL 化
//     + 生成列/索引），见 conversation_list_query_test.go 注释。
//
// 该函数被 conversation_list_query_test.go 做字符串断言保护，
// 防止后续回退为 COALESCE 包裹形式（保持函数包裹零增长）。
func conversationListQuerySQL() string {
	return `
		SELECT
			cm.*,
			c.last_message_at,
			c.created_at,
			IFNULL(cs.is_pinned, false) as is_pinned,
			IFNULL(cs.is_hidden, false) as is_hidden
		FROM conversation_members cm
		LEFT JOIN conversations c ON c.id = cm.conversation_id
		LEFT JOIN conversation_sessions cs ON cs.conversation_id = cm.conversation_id AND cs.user_id = ?
		WHERE cm.user_id = ? AND (cs.is_hidden = false OR cs.is_hidden IS NULL)
	`
}

// conversationListCountQuerySQL 返回会话列表总数查询的基础 SQL。
// 与 conversationListQuerySQL 共用相同的 WHERE 过滤逻辑，
// 确保 COUNT(*) 结果与实际列表一致，且同样可走索引。
func conversationListCountQuerySQL() string {
	return `
		SELECT COUNT(*)
		FROM conversation_members cm
		LEFT JOIN conversations c ON c.id = cm.conversation_id
		LEFT JOIN conversation_sessions cs ON cs.conversation_id = cm.conversation_id AND cs.user_id = ?
		WHERE cm.user_id = ? AND (cs.is_hidden = false OR cs.is_hidden IS NULL)
	`
}
