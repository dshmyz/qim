package aiprompt

import (
	"github.com/dshmyz/qim/qim-server/pkg/productname"
	"regexp"
	"strings"
	"testing"
)

func TestCurrentTimeLine(t *testing.T) {
	got := CurrentTimeLine()
	// 形如：【当前时间】2026-08-09 01:07 (日)
	re := regexp.MustCompile(`^【当前时间】\d{4}-\d{2}-\d{2} \d{2}:\d{2} \([一二三四五六日]\)$`)
	if !re.MatchString(got) {
		t.Errorf("CurrentTimeLine 格式不正确: %q", got)
	}
	if !strings.HasPrefix(got, "【当前时间】") {
		t.Errorf("应以【当前时间】开头: %q", got)
	}
}

func TestBuildPersona(t *testing.T) {
	// 各性格返回对应人设，且都带当前品牌名
	cases := map[string]string{
		"casual":    "性格轻松幽默",
		"concise":   "风格简洁高效",
		"friendly":  "性格温暖亲切",
		"technical": "技术专家",
	}
	for p, want := range cases {
		got := BuildPersona(p)
		if !strings.Contains(got, productname.Name) {
			t.Errorf("BuildPersona(%q) 应包含品牌名 %q: %q", p, productname.Name, got)
		}
		if !strings.Contains(got, want) {
			t.Errorf("BuildPersona(%q) 应包含 %q: %q", p, want, got)
		}
	}
	// 默认/空值回退到专业严谨人设
	fallback := BuildPersona("")
	if !strings.Contains(fallback, "专业严谨") {
		t.Errorf("默认人设应含专业严谨: %q", fallback)
	}
	fallback2 := BuildPersona("whatever")
	if fallback2 != fallback {
		t.Errorf("未知性格应回退默认人设: %q vs %q", fallback2, fallback)
	}
}

func TestLanguageRule(t *testing.T) {
	if got := LanguageRule("zh"); got != "请使用中文回答" {
		t.Errorf("zh: %q", got)
	}
	if got := LanguageRule("en"); got != "Please answer in English" {
		t.Errorf("en: %q", got)
	}
	if got := LanguageRule("fr"); got != "" {
		t.Errorf("未知语言应返回空串: %q", got)
	}
}

func TestLengthRule(t *testing.T) {
	cases := map[string]string{
		"short":     "50字以内",
		"medium":    "150字以内",
		"very_long": "400字以内",
		"long":      "可以展开说明",
	}
	for v, want := range cases {
		got := LengthRule(v)
		if !strings.Contains(got, want) {
			t.Errorf("LengthRule(%q) 应包含 %q: %q", v, want, got)
		}
	}
	if got := LengthRule("unknown"); got != "" {
		t.Errorf("未知长度应返回空串: %q", got)
	}
}

func TestReplyRules(t *testing.T) {
	rules := ReplyRules()
	if len(rules) != 3 {
		t.Fatalf("应返回 3 条通用规则, got %d: %v", len(rules), rules)
	}
	joined := strings.Join(rules, "|")
	if !strings.Contains(joined, "优先使用知识库中的内容回答") {
		t.Errorf("缺知识库优先规则: %v", rules)
	}
	if !strings.Contains(joined, "通用知识") {
		t.Errorf("缺兜底说明规则: %v", rules)
	}
	if !strings.Contains(joined, "@") {
		t.Errorf("缺 @ 提及约束规则: %v", rules)
	}
}
