package handler

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitReplyChunks(t *testing.T) {
	// 空串 → 无分块
	assert.Empty(t, splitReplyChunks(""))

	// 短文本不切
	in := "今天上海多云，26℃。"
	out := splitReplyChunks(in)
	require.Len(t, out, 1)
	assert.Equal(t, in, out[0])

	// 拼接回原样（不丢字）
	long := "第一句要点。第二句要点。第三句要点，继续第四句。第五句结尾。" + strings.Repeat("字", 300)
	out = splitReplyChunks(long)
	assert.NotEmpty(t, out)
	joined := ""
	for _, c := range out {
		joined += c
	}
	assert.Equal(t, long, joined, "分块拼接应等于原文，不得丢字")
}

// TestFriendlyToolLabel 校验工具标签是「动作名词」而非进行时态：工具调用总发生在
// feedback 闭包返回之后（即已结束），若标签仍带「正在…」在结束后卡片会显得奇怪，
// 因此标签应为中性名词，完成/失败由 status + 前端徽标体现。
func TestFriendlyToolLabel(t *testing.T) {
	cases := []struct {
		tool string
		want string
	}{
		{tool: "group_management", want: "群管理操作"},
		{tool: "user_management", want: "用户管理"},
		{tool: "group_summary", want: "群聊总结"},
		{tool: "search_messages", want: "群消息搜索"},
		{tool: "create_group_task", want: "创建群待办"},
		{tool: "system_notification", want: "系统通知"},
		{tool: "mcp_demo_calculator", want: "计算"},
		{tool: "mcp_weather_get_weather", want: "查询天气"},
		{tool: "mcp_search_svc_query", want: "查询"},
		{tool: "translate", want: "翻译"},
		{tool: "gen_image", want: "生成图片"},
		{tool: "parse_pdf", want: "处理文档"},
		{tool: "mcp_unknown_service", want: "外部服务"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, friendlyToolLabel(c.tool), "tool=%s", c.tool)
	}
	// 任一标签都不得是进行时「正在…」
	for _, tt := range []string{"group_management", "mcp_demo_calculator", "mcp_demo_get_weather", "other_tool"} {
		assert.NotContains(t, friendlyToolLabel(tt), "正在", "tool=%s 不应为进行时态标签", tt)
	}
}
