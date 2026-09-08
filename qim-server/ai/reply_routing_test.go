package ai

import "testing"

// TestResolveReplyTaskType 钉死对话回复统一任务路由的优先级：
// 视觉（带图）优先于复杂度分级；未配置视觉路由时带图回退 chat 且不做复杂度分级
// （避免把图片 base64 塞给 digest 思考模型）；纯文本按复杂度分级（复杂 -> digest）。
func TestResolveReplyTaskType(t *testing.T) {
	withVision := NewAIService(&AIConfig{Router: RouterConfig{
		Routes: map[TaskType]Route{
			TaskTypeVision: {Provider: "openai", Model: "vision-model"},
		},
	}})
	noRoutes := NewAIService(&AIConfig{})

	complexQ := "请分析一下这个方案的优缺点，并给出改进建议？"
	imageMsg := Message{Role: "user", Content: complexQ, ImageURL: "data:image/png;base64,xxx"}

	cases := []struct {
		name     string
		svc      *AIService
		messages []Message
		want     TaskType
	}{
		{"纯文本简单问题走 chat", withVision, []Message{{Role: "user", Content: "你好"}}, TaskTypeChat},
		{"纯文本复杂问题走 digest", withVision, []Message{{Role: "user", Content: complexQ}}, TaskTypeDigest},
		{"带图优先视觉路由", withVision, []Message{imageMsg}, TaskTypeVision},
		{"带图多图 ImageURLs 同样走视觉", withVision, []Message{{Role: "user", ImageURLs: []string{"data:image/png;base64,a", "data:image/png;base64,b"}}}, TaskTypeVision},
		{"未配视觉路由带图回退 chat 不分级", noRoutes, []Message{imageMsg}, TaskTypeChat},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.svc.ResolveReplyTaskType(tc.messages); got != tc.want {
				t.Fatalf("ResolveReplyTaskType() = %q, want %q", got, tc.want)
			}
		})
	}
}
