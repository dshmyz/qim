package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMergeMessageExtra 存疑标注合并进 Extra 时不得丢既有字段，脏数据不阻断。
func TestMergeMessageExtra(t *testing.T) {
	ann := QualityAnnotation{Status: "doubtful", Reason: "包含资料中不存在的事实", Confidence: 0.8, At: 1700000000}

	t.Run("保留既有字段", func(t *testing.T) {
		got, err := MergeMessageExtra(`{"knowledge_sources":[{"title":"文档"}],"tool_calls":[]}`,
			map[string]interface{}{"quality": ann})
		require.NoError(t, err)
		fields := ParseMessageExtraFields(got)
		assert.Contains(t, fields, "knowledge_sources", "既有字段不得丢失")
		q, ok := fields["quality"].(map[string]interface{})
		require.True(t, ok, "quality 应可解析: %s", got)
		assert.Equal(t, "doubtful", q["status"])
		assert.Equal(t, "包含资料中不存在的事实", q["reason"])
	})

	t.Run("空 Extra 可创建", func(t *testing.T) {
		got, err := MergeMessageExtra("", map[string]interface{}{"quality": ann})
		require.NoError(t, err)
		assert.Contains(t, got, `"doubtful"`)
	})

	t.Run("脏 Extra 不阻断", func(t *testing.T) {
		got, err := MergeMessageExtra(`{"broken":`, map[string]interface{}{"quality": ann})
		require.NoError(t, err)
		fields := ParseMessageExtraFields(got)
		require.NotNil(t, fields)
		assert.Contains(t, fields, "quality")
	})
}

// TestParseMessageExtraFields_Quality 白名单必须透出 quality（否则前端拿不到存疑标注）。
func TestParseMessageExtraFields_Quality(t *testing.T) {
	raw := `{"tool_calls":[{"id":"c1"}],"quality":{"status":"doubtful","reason":"x"},"internal_note":"不应透出"}`
	fields := ParseMessageExtraFields(raw)
	assert.Equal(t, "doubtful", fields["quality"].(map[string]interface{})["status"])
	assert.Contains(t, fields, "tool_calls")
	assert.NotContains(t, fields, "internal_note", "白名单外字段不得透出")
}
