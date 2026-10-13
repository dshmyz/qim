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
				if len(args) != 1 || args[0] != tt.keyword {
					t.Fatalf("MATCH 参数错误: %v", args)
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
