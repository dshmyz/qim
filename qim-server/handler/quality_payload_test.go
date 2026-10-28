package handler

import (
	"encoding/json"
	"testing"

	"github.com/dshmyz/qim/qim-server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// quality_payload_test.go 钉死两条 WS 载荷契约：客户端（AIQualityBadge / Main.vue）
// 直接依赖这些字段名与"只推送变化字段"的语义。

func TestQualityUpdatePayload(t *testing.T) {
	ann := service.QualityAnnotation{Status: "doubtful", Reason: "回复包含资料中不存在的具体事实", Confidence: 0.8, At: 1793266800}
	got := qualityUpdatePayload(7, 98, ann)

	assert.Equal(t, uint(7), got["id"])
	assert.Equal(t, uint(98), got["conversation_id"])
	assert.Equal(t, ann, got["quality"])
	assert.NotContains(t, got, "content", "增量更新不得携带正文，避免覆盖已流出的内容")

	raw, err := json.Marshal(got)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"status":"doubtful"`)
	assert.Contains(t, string(raw), `"reason":"回复包含资料中不存在的具体事实"`)
}

func TestReplyStartedPayload(t *testing.T) {
	base := replyStartedPayload(98, "")
	assert.Equal(t, uint(98), base["conversation_id"])
	assert.NotContains(t, base, "status", "无阶段文案时不得下发 status（保持旧的「思考中」渲染）")

	withStatus := replyStartedPayload(98, "正在看图片…")
	assert.Equal(t, "正在看图片…", withStatus["status"])
}
