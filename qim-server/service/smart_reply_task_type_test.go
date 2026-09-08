package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
)

// TestResolveQuotedImageTaskType 钉死群 @AI 提及路径的任务路由：
//   - 无被引用图片：按触发消息复杂度分级（复杂 -> digest，与 /ai/completion、bot 回复同一条规则）
//   - 被引用图片 + 有视觉路由：走 TaskTypeVision
//   - 被引用图片 + 无视觉路由：降级 QuotedFailed 并回退 chat（降级文本不做复杂度分级）
func TestResolveQuotedImageTaskType(t *testing.T) {
	withVision := ai.NewAIService(&ai.AIConfig{Router: ai.RouterConfig{
		Routes: map[ai.TaskType]ai.Route{
			ai.TaskTypeVision: {Provider: "openai", Model: "vision-model"},
		},
	}})
	noRoutes := ai.NewAIService(&ai.AIConfig{})

	complexQ := "请分析一下这个方案的优缺点，并给出改进建议？"
	graphWithVision := NewSmartReplyGraph(withVision, nil, nil, nil, nil, nil)
	graphNoRoutes := NewSmartReplyGraph(noRoutes, nil, nil, nil, nil, nil)

	cases := []struct {
		name               string
		graph              *SmartReplyGraph
		input              *SmartReplyContext
		want               ai.TaskType
		wantQuotedDegraded bool // 被引用图片是否被降级为 QuotedFailed
	}{
		{
			name:  "纯文本简单问题走 chat",
			graph: graphWithVision,
			input: &SmartReplyContext{Message: "你好"},
			want:  ai.TaskTypeChat,
		},
		{
			name:  "纯文本复杂问题走 digest",
			graph: graphWithVision,
			input: &SmartReplyContext{Message: complexQ},
			want:  ai.TaskTypeDigest,
		},
		{
			name:  "被引用图片有视觉路由走 vision",
			graph: graphWithVision,
			input: &SmartReplyContext{Message: complexQ, Quoted: &QuotedContext{Kind: QuotedImage, Name: "a.png", ImageURL: "data:image/png;base64,xxx"}},
			want:  ai.TaskTypeVision,
		},
		{
			name:               "被引用图片无视觉路由降级 QuotedFailed 回 chat",
			graph:              graphNoRoutes,
			input:              &SmartReplyContext{Message: complexQ, Quoted: &QuotedContext{Kind: QuotedImage, Name: "a.png", ImageURL: "data:image/png;base64,xxx"}},
			want:               ai.TaskTypeChat,
			wantQuotedDegraded: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.graph.resolveQuotedImageTaskType(tc.input)
			if got != tc.want {
				t.Fatalf("resolveQuotedImageTaskType() = %q, want %q", got, tc.want)
			}
			degraded := tc.input.Quoted != nil && tc.input.Quoted.Kind == QuotedFailed
			if degraded != tc.wantQuotedDegraded {
				t.Fatalf("Quoted 降级状态 = %v, want %v", degraded, tc.wantQuotedDegraded)
			}
		})
	}
}
