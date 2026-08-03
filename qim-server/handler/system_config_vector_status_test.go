package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMergeRuntimeFlags_VectorEnabledTrue 验证向量库可用时 vector_enabled=true。
// 这个字段不存数据库（是基础设施状态），由 handler 运行时注入，
// 让前端在不调额外接口的情况下知道笔记/分身等 RAG 能力是否可用。
func TestMergeRuntimeFlags_VectorEnabledTrue(t *testing.T) {
	in := map[string]interface{}{"enableAI": true, "messageRecallTime": 120}
	out := mergeRuntimeFlags(in, true)
	assert.Equal(t, true, out["enableAI"], "原字段保留")
	assert.Equal(t, 120, out["messageRecallTime"], "原字段保留")
	assert.Equal(t, true, out["vector_enabled"], "vector_enabled 应为 true")
}

// TestMergeRuntimeFlags_VectorEnabledFalse 验证向量库未配时 vector_enabled=false。
// 前端据此显示"知识库开关无效"提示，避免用户误以为开关生效。
func TestMergeRuntimeFlags_VectorEnabledFalse(t *testing.T) {
	in := map[string]interface{}{"enableAI": true}
	out := mergeRuntimeFlags(in, false)
	assert.Equal(t, false, out["vector_enabled"], "vector_enabled 应为 false")
}

// TestMergeRuntimeFlags_OverwriteExisting 若 result 已含 vector_enabled（理论不会，但防御）应覆盖。
func TestMergeRuntimeFlags_OverwriteExisting(t *testing.T) {
	in := map[string]interface{}{"vector_enabled": "stale"}
	out := mergeRuntimeFlags(in, true)
	assert.Equal(t, true, out["vector_enabled"], "应覆盖为运行时真实值")
}
