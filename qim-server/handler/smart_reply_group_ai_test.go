package handler

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
)

func TestGroupAIKeywordMatches(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		keywords string
		want     bool
	}{
		{"空关键词视为无限制", "随便聊聊", "", true},
		{"命中第一个", "今天天气不错", "天气,会议", true},
		{"命中第二个", "下午有个会议", "天气,会议", true},
		{"大小写不敏感", "please Report now", "report", true},
		{"未命中", "今天吃啥", "天气,会议", false},
		{"关键词带空格被 trim", "今天天气", "  天气  , 会议", true},
		{"空片段不误命中", "无关键字", "天气,,", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, groupAIKeywordMatches(c.content, c.keywords))
		})
	}
}

func TestGroupAIMentionsAI(t *testing.T) {
	cases := []struct {
		name          string
		content       string
		assistantName string
		want          bool
	}{
		{"明文 @AI 大写", "@AI 帮我看下", "AI助手", true},
		{"明文 @AI 小写", "帮我 @ai 看下", "AI助手", true},
		{"明文 @助手名", "问下@小助 今天如何", "小助", true},
		{"mention token 命中助手名", "@{mention:7|xiaozhu} 你好", "xiaozhu", true},
		{"mention token 名为 ai（大小写不敏感）", "@{mention:7|ai} 你好", "AI助手", true},
		{"未提及", "今天大家讨论一下", "AI助手", false},
		{"@ 其他人不算", "@{mention:8|张三} 你好", "AI助手", false},
		{"空 mention 名不误判", "@{mention:8|} 你好", "AI助手", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, groupAIMentionsAI(c.content, c.assistantName))
		})
	}
}

func TestIsAIAssistantMentionName(t *testing.T) {
	assert.False(t, isAIAssistantMentionName("", "AI助手"), "空名不命中")
	assert.True(t, isAIAssistantMentionName("AI助手", "AI助手"), "精确匹配")
	assert.True(t, isAIAssistantMentionName("ai", "AI助手"), "ai 大小写不敏感")
	assert.True(t, isAIAssistantMentionName("AI", "AI"), "自定义名精确")
	assert.False(t, isAIAssistantMentionName("张三", "AI助手"), "他人不命中")
}

func TestExtractAIQuestion(t *testing.T) {
	cases := []struct {
		name          string
		content       string
		assistantName string
		want          string
	}{
		{"明文 @AI 提问", "@AI 今天天气如何", "AI助手", "今天天气如何"},
		{"mention token 提问", "@{mention:7|xiaozhu} 帮我查一下", "xiaozhu", "帮我查一下"},
		{"明文 @助手名 提问", "@小助 今天呢", "小助", "今天呢"},
		{"无提及返回原文", "今天天气如何", "AI助手", "今天天气如何"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, extractAIQuestion(c.content, c.assistantName))
		})
	}
}

func TestDecideGroupAIReply(t *testing.T) {
	// action 字符串化，便于表内直观对照
	actionName := func(a GroupAIReplyAction) string {
		switch a {
		case GroupAISkipReply:
			return "Skip"
		case GroupAIMentionReply:
			return "Mention"
		case GroupAIAutoReply:
			return "Auto"
		}
		return "?"
	}

	cases := []struct {
		name          string
		enabled       bool
		replyMode     string
		keywords      string
		content       string
		assistantName string
		antiSpam      bool
		want          GroupAIReplyAction
	}{
		{"未启用直接跳过", false, "always", "", "@AI 帮我", "AI助手", false, GroupAISkipReply},
		{"mention_only + @AI 提及 → 直接回复", true, "mention_only", "", "@AI 帮我", "AI助手", false, GroupAIMentionReply},
		{"mention_only + 无提及 → 跳过", true, "mention_only", "天气", "今天天气", "AI助手", false, GroupAISkipReply},
		{"always + 无关键词 + 无提及 → 自动", true, "always", "", "随便聊聊", "AI助手", false, GroupAIAutoReply},
		{"always + 关键词命中 → 自动", true, "always", "天气,会议", "今天天气不错", "AI助手", false, GroupAIAutoReply},
		{"always + 关键词未命中 → 跳过", true, "always", "天气", "今天吃啥", "AI助手", false, GroupAISkipReply},
		// 钉死注释/代码不符点：always+配置关键词时，不含关键词的 @AI 提及也会被关键词门控跳过
		// 修复后：@AI 提及优先于关键词门控，即使关键词未命中也直接回复
		{"always + 关键词未命中时 @AI 提及 → 直接回复（提及绕过关键词门控）", true, "always", "天气", "@AI 帮我", "AI助手", false, GroupAIMentionReply},
		{"always + 关键词命中且 @AI 提及 → 直接回复", true, "always", "天气", "@AI 今天天气", "AI助手", false, GroupAIMentionReply},
		{"smart + 无关键词 + 无提及 → 自动", true, "smart", "", "这个问题怎么解", "AI助手", false, GroupAIAutoReply},
		{"off + @AI 提及 → 直接回复（提及优先于 off）", true, "off", "", "@AI 帮我", "AI助手", false, GroupAIMentionReply},
		{"off + 无提及 → 跳过", true, "off", "", "随便聊聊", "AI助手", false, GroupAISkipReply},
		// 反刷屏优先级：即使 @AI 提及，命中反刷屏窗口也跳过
		{"mention_only + @AI 提及但反刷屏命中 → 跳过", true, "mention_only", "", "@AI 帮我", "AI助手", true, GroupAISkipReply},
		{"always + 关键词命中但反刷屏命中 → 跳过", true, "always", "天气", "今天天气", "AI助手", true, GroupAISkipReply},
		{"大小写不敏感关键词命中 → 自动", true, "always", "Report", "please report", "AI助手", false, GroupAIAutoReply},
		{"mention_only + 自定义助手名明文提及 → 直接回复", true, "mention_only", "", "@xiaozhu 帮我", "xiaozhu", false, GroupAIMentionReply},
		{"mention_only + 自定义助手名 token 提及 → 直接回复", true, "mention_only", "", "@{mention:7|xiaozhu} 帮我", "xiaozhu", false, GroupAIMentionReply},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := model.GroupAIConfig{
				Enabled:         c.enabled,
				ReplyMode:       c.replyMode,
				TriggerKeywords: c.keywords,
			}
			got := DecideGroupAIReply(cfg, c.content, c.assistantName, c.antiSpam)
			if got != c.want {
				t.Errorf("DecideGroupAIReply got=%s want=%s", actionName(got), actionName(c.want))
			}
		})
	}
}
