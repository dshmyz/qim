package database

import "strings"

// ContentMatch 返回"消息内容包含关键词"的 WHERE 片段与参数，是消息搜索的匹配单源。
// SupportsFulltext() 为真（MySQL 上以 ngram 解析器探测为准）且关键词 ≥2 rune 时
// 走 FULLTEXT MATCH——ngram 对中英文统一按 2-gram 切分，中文可正常命中；
// 其余情况降级 LIKE：单字关键词在 ngram_token_size=2 下切不出 gram，MATCH 恒无结果。
//
// 关键词以短语形式（"kw"）送入 BOOLEAN MODE：+ - > < ( ) ~ * " @ 在布尔模式里是
// 操作符，原样拼接会让 C++ / 含引号句子等输入形成畸形布尔表达式，静默返回错误
// 结果；短语包裹使其按字面子串匹配，与 LIKE 语义对齐。短语内的双引号剥除
// （布尔模式的短语没有转义语法，嵌入引号会截断短语）。
func ContentMatch(keyword string) (string, []any) {
	if D != nil && D.SupportsFulltext() && len([]rune(keyword)) >= 2 {
		phrase := strings.ReplaceAll(keyword, `"`, " ")
		return "MATCH(messages.content) AGAINST(? IN BOOLEAN MODE)", []any{"\"" + phrase + "\""}
	}
	return "messages.content LIKE ?", []any{"%" + keyword + "%"}
}
