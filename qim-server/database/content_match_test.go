package database

import (
	"strings"
	"testing"
)

func TestContentMatch(t *testing.T) {
	origD := D
	defer func() { D = origD }()

	mysqlWithNgram := &mysqlDialect{supportsFulltext: true}
	mysqlWithoutNgram := &mysqlDialect{supportsFulltext: false}
	sqlite := &sqliteDialect{}

	tests := []struct {
		name      string
		dialect   Dialect
		keyword   string
		wantMatch bool
	}{
		// ngram 可用：≥2 rune 的中英文关键词均走 MATCH
		{name: "chinese keyword with ngram", dialect: mysqlWithNgram, keyword: "项目排期", wantMatch: true},
		{name: "english keyword with ngram", dialect: mysqlWithNgram, keyword: "golang", wantMatch: true},
		{name: "mixed keyword with ngram", dialect: mysqlWithNgram, keyword: "hello项目", wantMatch: true},
		// 单字关键词在 ngram_token_size=2 下切不出 gram，必须 LIKE 兜底
		{name: "single chinese char downgrades", dialect: mysqlWithNgram, keyword: "会", wantMatch: false},
		{name: "single ascii char downgrades", dialect: mysqlWithNgram, keyword: "a", wantMatch: false},
		// 库不支持 ngram 全文：一律 LIKE
		{name: "mysql without ngram", dialect: mysqlWithoutNgram, keyword: "项目", wantMatch: false},
		{name: "sqlite always like", dialect: sqlite, keyword: "项目", wantMatch: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			D = tt.dialect
			cond, args := ContentMatch(tt.keyword)
			if tt.wantMatch {
				if !strings.Contains(cond, "MATCH(messages.content)") {
					t.Fatalf("期望 MATCH 分支，实际条件: %s", cond)
				}
				// 布尔模式短语包裹：args[0] = "kw"（防 + - ( ) " 等操作符注入）
				if len(args) != 1 {
					t.Fatalf("MATCH 参数数量错误: %v", args)
				}
				phrase, ok := args[0].(string)
				if !ok || !strings.HasPrefix(phrase, "\"") || !strings.HasSuffix(phrase, "\"") {
					t.Fatalf("MATCH 参数必须为短语包裹形态: %v", args)
				}
				if strings.Trim(phrase, "\"") != tt.keyword {
					t.Fatalf("短语内容错误: %v", args)
				}
			} else {
				if !strings.Contains(cond, "LIKE ?") {
					t.Fatalf("期望 LIKE 分支，实际条件: %s", cond)
				}
				if len(args) != 1 || args[0] != "%"+tt.keyword+"%" {
					t.Fatalf("LIKE 参数错误: %v", args)
				}
			}
		})
	}
}

// 布尔模式操作符注入：含操作符的关键词必须被短语包裹中性化，
// 且短语内的双引号剥除（嵌入引号会截断短语）。
func TestContentMatch_BooleanOperatorSafety(t *testing.T) {
	origD := D
	defer func() { D = origD }()
	D = &mysqlDialect{supportsFulltext: true}

	cond, args := ContentMatch(`C++说"你好"`)
	if !strings.Contains(cond, "MATCH") {
		t.Fatalf("应走 MATCH 分支")
	}
	phrase := args[0].(string)
	inner := strings.Trim(phrase, `"`)
	if strings.Contains(inner, `"`) {
		t.Fatalf("引号未剥除，布尔表达式会被截断: %q", phrase)
	}
	if !strings.Contains(inner, "C++") {
		t.Fatalf("操作符 + 不应被剥除（短语内是字面量）: %q", phrase)
	}
}
