package handler

import (
	"testing"
)

func TestGroupAIMentionsAI_BarePatternBoundary(t *testing.T) {
	cases := []struct {
		content string
		want    bool
		desc    string
	}{
		{"@AI 你好", true, "行首明文 @AI 应命中"},
		{"大家看 @AI 助手怎么说", true, "@ 前是空格应命中"},
		{"看看这个：@AI，回答下", true, "@ 前是标点应命中"},
		{"我的邮箱是 xxx@AI-support.com", false, "邮箱 local-part 内的 @AI 不得命中"},
		{"联系 foo@AIsupport", false, "单词内嵌 @AI 不得命中"},
		{"发送给 team@ai.example.com", false, "小写邮箱同样不得命中"},
		{"普通消息没有提及", false, "无 @ 不命中"},
	}
	for _, tt := range cases {
		if got := groupAIMentionsAI(tt.content, ""); got != tt.want {
			t.Errorf("%s: groupAIMentionsAI(%q)=%v want %v", tt.desc, tt.content, got, tt.want)
		}
	}
}

func TestGroupAIMentionsAI_AssistantNameBoundary(t *testing.T) {
	// 自定义助手名同样受边界保护
	if groupAIMentionsAI("发到 bot@智能助手群组", "智能助手") {
		t.Error("邮箱内嵌助手名不得命中")
	}
	if !groupAIMentionsAI("请 @智能助手 总结", "智能助手") {
		t.Error("明文 @助手名 应命中")
	}
}

func TestExtractAIQuestion_BarePatternBoundary(t *testing.T) {
	// 邮箱场景不得切割出尾巴当问题
	if got := extractAIQuestion("我的邮箱是 xxx@AI-support.com 谢谢", ""); got != "我的邮箱是 xxx@AI-support.com 谢谢" {
		t.Errorf("邮箱内容不应被切割，got %q", got)
	}
	if got := extractAIQuestion("@AI 今天天气如何", ""); got != "今天天气如何" {
		t.Errorf("行首 @AI 应提取问题，got %q", got)
	}
}
