package ai

import "testing"

func TestIsComplexQuery(t *testing.T) {
	cases := []struct {
		name  string
		q     string
		want  bool
	}{
		{"空串", "", false},
		{"简单问候", "你好", false},
		{"短问题", "几点下班", false},
		{"长问题触发", "我需要根据过去三个季度的销售数据、团队产能和外部市场环境，制定一个能落地的明年增长规划，并给出分阶段的执行顺序和风险预案，同时评估每个阶段的投入产出比", true},
		{"推理词", "为什么数据同步会失败？", true},
		{"多问句", "这个接口为什么慢？是网络还是代码问题？", true},
		{"分析词", "帮我对这两份方案做对比分析", true},
		{"代码词", "这段代码的 bug 在哪", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsComplexQuery(tc.q)
			if got != tc.want {
				t.Errorf("IsComplexQuery(%q) = %v, want %v", tc.q, got, tc.want)
			}
		})
	}
}

func TestChatTaskType_UsesLastUserMessage(t *testing.T) {
	svc := &AIService{}
	// 最后一条 user 消息复杂 → digest
	if got := svc.ChatTaskType([]Message{{Role: "system", Content: "x"}, {Role: "user", Content: "为什么这么慢"}}); got != TaskTypeDigest {
		t.Errorf("复杂问题应路由 digest, got %s", got)
	}
	// 简单 → chat
	if got := svc.ChatTaskType([]Message{{Role: "user", Content: "你好"}}); got != TaskTypeChat {
		t.Errorf("简单问题应路由 chat, got %s", got)
	}
	// 无 user 消息 → 默认 chat
	if got := svc.ChatTaskType([]Message{{Role: "system", Content: "x"}}); got != TaskTypeChat {
		t.Errorf("无 user 消息应默认 chat, got %s", got)
	}
}
