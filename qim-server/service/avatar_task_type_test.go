package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
)

// TestAvatarTaskType 钉死分身的任务路由：只做「带图 → 视觉」最小路由，不做复杂度分级
//（自选模型门控优先）。带图 + 有视觉路由 → vision；带图 + 无视觉路由 → chat（不分级，
// 避免 base64 进 digest）；纯文本 → chat。
func TestAvatarTaskType(t *testing.T) {
	withVision := ai.NewAIService(&ai.AIConfig{Router: ai.RouterConfig{
		Routes: map[ai.TaskType]ai.Route{
			ai.TaskTypeVision: {Provider: "openai", Model: "vision-model"},
		},
	}})
	noRoutes := ai.NewAIService(&ai.AIConfig{})

	complexQ := "请分析一下这个方案的优缺点，并给出改进建议？"

	cases := []struct {
		name     string
		svc      *ai.AIService
		messages []ai.Message
		want     ai.TaskType
	}{
		{"纯文本走 chat 不分级", withVision, []ai.Message{{Role: "user", Content: complexQ}}, ai.TaskTypeChat},
		{"带图有视觉路由走 vision", withVision, []ai.Message{{Role: "user", Content: "这是什么", ImageURL: "data:image/png;base64,xxx"}}, ai.TaskTypeVision},
		{"多图 ImageURLs 同样走 vision", withVision, []ai.Message{{Role: "user", ImageURLs: []string{"data:image/png;base64,a"}}}, ai.TaskTypeVision},
		{"带图无视觉路由回退 chat 不分级", noRoutes, []ai.Message{{Role: "user", Content: complexQ, ImageURL: "data:image/png;base64,xxx"}}, ai.TaskTypeChat},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := avatarTaskType(tc.svc, tc.messages); got != tc.want {
				t.Fatalf("avatarTaskType() = %q, want %q", got, tc.want)
			}
		})
	}
}
