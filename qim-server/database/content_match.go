package database

// ContentMatch 返回"消息内容包含关键词"的 WHERE 片段与参数，是消息搜索的匹配单源。
// SupportsFulltext() 为真（MySQL 上以 ngram 解析器探测为准）且关键词 ≥2 rune 时
// 走 FULLTEXT MATCH——ngram 对中英文统一按 2-gram 切分，中文可正常命中；
// 其余情况降级 LIKE：单字关键词在 ngram_token_size=2 下切不出 gram，MATCH 恒无结果。
func ContentMatch(keyword string) (string, []any) {
	if D != nil && D.SupportsFulltext() && len([]rune(keyword)) >= 2 {
		return "MATCH(messages.content) AGAINST(? IN BOOLEAN MODE)", []any{keyword}
	}
	return "messages.content LIKE ?", []any{"%" + keyword + "%"}
}
