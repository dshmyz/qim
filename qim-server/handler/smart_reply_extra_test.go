package handler

import (
	"encoding/json"
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPersistAIMessageExtra_MergesToolCallsAndSources
// 群助手回复结束时，工具调用 + 命中的知识来源应合并写进消息 Extra（JSON），
// 供刷新/REST 回放后工具卡片与「知识来源」徽章仍可见。
func TestPersistAIMessageExtra_MergesToolCallsAndSources(t *testing.T) {
	t.Run("工具调用与知识来源合并写入", func(t *testing.T) {
		msg := &model.Message{}
		e := &SmartReplyEngine{}
		e.persistAIMessageExtra(func() *model.Message { return msg }, []ToolCallRecord{{
			ID: "call_1", ToolLabel: "计算器", Status: "ok",
		}}, []service.KnowledgeSource{{Title: "Q3 规划", Score: 0.92}})

		require.NotEmpty(t, msg.Extra, "有内容时应写入 Extra")
		var extra map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(msg.Extra), &extra))

		tcs, ok := extra["tool_calls"].([]interface{})
		require.True(t, ok, "应包含 tool_calls")
		require.Len(t, tcs, 1)

		ks, ok := extra["knowledge_sources"].([]interface{})
		require.True(t, ok, "应包含 knowledge_sources")
		require.Len(t, ks, 1)
		first := ks[0].(map[string]interface{})
		assert.Equal(t, "Q3 规划", first["title"])
		assert.Equal(t, 0.92, first["score"])
	})

	t.Run("无内容时不写 Extra 且不调用 getMsg", func(t *testing.T) {
		msg := &model.Message{}
		e := &SmartReplyEngine{}
		var getMsgCalled bool
		e.persistAIMessageExtra(func() *model.Message { getMsgCalled = true; return msg }, nil, nil)
		assert.Empty(t, msg.Extra, "全空时不应写 Extra")
		assert.False(t, getMsgCalled, "全空时不应触发 getMsg()（惰性：避免懒创建空消息）")
	})

	t.Run("仅知识来源", func(t *testing.T) {
		msg := &model.Message{}
		e := &SmartReplyEngine{}
		e.persistAIMessageExtra(func() *model.Message { return msg }, nil, []service.KnowledgeSource{{Title: "会议纪要", Score: 0.88}})
		require.NotEmpty(t, msg.Extra)
		var extra map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(msg.Extra), &extra))
		_, hasTK := extra["tool_calls"]
		assert.False(t, hasTK, "无工具调用时不应写 tool_calls")
		ks, ok := extra["knowledge_sources"].([]interface{})
		require.True(t, ok)
		require.Len(t, ks, 1)
	})

	t.Run("nil 消息安全", func(t *testing.T) {
		e := &SmartReplyEngine{}
		e.persistAIMessageExtra(func() *model.Message { return nil }, []ToolCallRecord{{ID: "x"}}, nil) // 不应 panic
	})
}
