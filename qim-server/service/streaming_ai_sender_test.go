package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStreamingAISender 测试用 StreamingAISender 替身，记录推送的工具调用事件。
type mockStreamingAISender struct {
	events []mockToolEvent
}

type mockToolEvent struct {
	convID uint
	msgID  uint
	record ToolCallRecord
}

func (m *mockStreamingAISender) SendStreamingAIMessage(conversationID uint, assistantName string) (func(string) error, func() *model.Message, func() *model.Message, error) {
	return func(string) error { return nil }, func() *model.Message { return nil }, func() *model.Message { return nil }, nil
}

func (m *mockStreamingAISender) SendToolCallEvent(conversationID uint, msgID uint, record ToolCallRecord) {
	m.events = append(m.events, mockToolEvent{convID: conversationID, msgID: msgID, record: record})
}

func (m *mockStreamingAISender) NotifyReplyStarted(conversationID uint, assistantName string) {
	// 测试替身：开始事件不落库不记录，仅保证接口合规。
}

// TestFriendlyToolLabel 验证内部工具名到中文动作名词的映射。
func TestFriendlyToolLabel(t *testing.T) {
	cases := []struct{ tool, want string }{
		// 内置工具
		{"group_management", "群管理操作"},
		{"user_management", "用户管理"},
		{"search_messages", "群消息搜索"},
		{"create_group_task", "创建群待办"},
		{"list_tasks", "任务管理"},
		{"create_user_task", "任务管理"},
		{"search_knowledge", "知识搜索"},
		{"summarize_conversation", "会话总结"},
		{"send_message", "发送消息"},
		// 内置工具含已知关键词
		{"mcp_test_calculator", "计算"},
		{"mcp_weather_lookup", "查询天气"},
		{"mcp_translate_v2", "翻译"},
		// 外部 MCP 工具名：mcp_<conn>_<tool>，提取 <tool> 并做可读化
		{"mcp_server_get_stock_price", "查询 Stock price"},
		{"mcp_myserver_send_email", "发送 Email"},
		{"mcp_api_fetch_data", "获取 Data"},
		{"mcp_service_lookup_user", "查询 User"},
		{"mcp_tool_custom_func", "Custom func"},
		// tool 部分为空时用连接名做标签，不再退化为「外部服务」
		{"mcp_conn_", "外部工具（Conn）"},
		// 含连字符的工具名也能正确格式化
		{"mcp_server_fmt_get_info", "Fmt get info"},
		{"mcp_server_my-tool_run", "My tool run"},
		// 无 mcp_ 前缀的未知工具 → 通用「外部工具」
		{"unknown_tool_xyz", "外部工具"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, FriendlyToolLabel(c.tool), "tool=%s", c.tool)
	}
}

// TestNewToolCallFeedback_StartEndPhases 验证 start 推 running 事件、end 推终态事件并收集记录。
func TestNewToolCallFeedback_StartEndPhases(t *testing.T) {
	sender := &mockStreamingAISender{}
	msg := &model.Message{ID: 42}
	getMsg := func() *model.Message { return msg }
	var toolCalls []ToolCallRecord

	fb := NewToolCallFeedback(sender, 7, getMsg, &toolCalls, nil, nil)

	// start 阶段：推一条 running 事件
	fb(1, "call_1", "start", "list_tasks", map[string]interface{}{"status": "todo"}, nil, nil)
	require.Len(t, sender.events, 1, "start 应推 1 条事件")
	assert.Equal(t, uint(7), sender.events[0].convID)
	assert.Equal(t, uint(42), sender.events[0].msgID)
	assert.Equal(t, "running", sender.events[0].record.Status)
	assert.Equal(t, "call_1", sender.events[0].record.ID)
	assert.Equal(t, "list_tasks", sender.events[0].record.ToolName)
	assert.Equal(t, "任务管理", sender.events[0].record.ToolLabel)
	assert.Empty(t, toolCalls, "start 阶段不应收集终态记录")

	// end 阶段（成功）：推 ok 终态事件 + 收集记录
	fb(1, "call_1", "end", "list_tasks", map[string]interface{}{"status": "todo"}, map[string]interface{}{"count": 3}, nil)
	require.Len(t, sender.events, 2, "end 应再推 1 条事件")
	assert.Equal(t, "ok", sender.events[1].record.Status)
	assert.Equal(t, "list_tasks", sender.events[1].record.ToolName)
	require.Len(t, toolCalls, 1, "end 应收集 1 条终态记录")
	assert.Equal(t, "ok", toolCalls[0].Status)
	assert.Equal(t, "list_tasks", toolCalls[0].ToolName)
}

// TestNewToolCallFeedback_EndWithError 验证 end 阶段执行出错时 status=error。
func TestNewToolCallFeedback_EndWithError(t *testing.T) {
	sender := &mockStreamingAISender{}
	msg := &model.Message{ID: 99}
	getMsg := func() *model.Message { return msg }
	var toolCalls []ToolCallRecord

	fb := NewToolCallFeedback(sender, 1, getMsg, &toolCalls, nil, nil)
	fb(1, "call_x", "end", "search_knowledge", nil, nil, ai.ErrStreamingToolsNotSupported)

	require.Len(t, sender.events, 1)
	assert.Equal(t, "error", sender.events[0].record.Status)
	require.Len(t, toolCalls, 1)
	assert.Equal(t, "error", toolCalls[0].Status)
}

// TestNewToolCallFeedback_NilMsgSkipsEvent 验证 getMsg 返回 nil 时不 panic 也不推事件。
func TestNewToolCallFeedback_NilMsgSkipsEvent(t *testing.T) {
	sender := &mockStreamingAISender{}
	getMsg := func() *model.Message { return nil }
	var toolCalls []ToolCallRecord

	fb := NewToolCallFeedback(sender, 1, getMsg, &toolCalls, nil, nil)
	fb(1, "call_1", "start", "list_tasks", nil, nil, nil) // 不应 panic
	fb(1, "call_1", "end", "list_tasks", nil, map[string]interface{}{"x": 1}, nil)

	// start 时 msg==nil -> 不推事件；end 时 msg==nil -> 不推事件但仍收集记录
	assert.Empty(t, sender.events, "msg==nil 时不应推事件")
	require.Len(t, toolCalls, 1, "end 阶段仍应收集终态记录")
}

// TestPersistAIMessageExtra_ServiceLevel 直接测试 service 层 PersistAIMessageExtra。
func TestPersistAIMessageExtra_ServiceLevel(t *testing.T) {
	t.Run("工具调用与知识来源合并写入", func(t *testing.T) {
		msg := &model.Message{}
		PersistAIMessageExtra(func() *model.Message { return msg }, []ToolCallRecord{{
			ID: "call_1", ToolLabel: "知识搜索", Status: "ok",
		}}, []KnowledgeSource{{Title: "Q3 规划", Score: 0.92}})
		require.NotEmpty(t, msg.Extra)
	})

	t.Run("无内容时不写 Extra 且不调用 getMsg", func(t *testing.T) {
		msg := &model.Message{}
		var called bool
		PersistAIMessageExtra(func() *model.Message { called = true; return msg }, nil, nil)
		assert.Empty(t, msg.Extra)
		assert.False(t, called, "全空时不应触发 getMsg")
	})

	t.Run("nil 消息安全", func(t *testing.T) {
		PersistAIMessageExtra(func() *model.Message { return nil }, []ToolCallRecord{{ID: "x"}}, nil)
	})
}

// TestResolveToolLabel_TitlePriority 验证 resolveToolLabel 的优先级：Title > Description > 内置映射。
func TestResolveToolLabel_TitlePriority(t *testing.T) {
	titles := map[string]string{"mcp_demo_create_rule": "创建规则"}
	descs := map[string]string{"mcp_demo_create_rule": "创建一条阻断规则"}

	// 标题最优先
	assert.Equal(t, "创建规则", resolveToolLabel("mcp_demo_create_rule", titles, descs))
	// 无标题时用描述
	assert.Equal(t, "创建一条阻断规则", resolveToolLabel("mcp_demo_create_rule", nil, descs))
	// 都没有时 fallback 到内置映射
	assert.Equal(t, "任务管理", resolveToolLabel("list_tasks", nil, nil))
}
