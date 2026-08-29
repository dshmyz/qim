package service

// ai_tool_scopes.go — 各 AI 入口的工具白名单单一来源。
// 此前三份白名单散落在本包两个文件与 handler 包，能力自述与实际放行靠注释约定对齐；
// 收敛到本文件后，调整某入口的工具面只需改这里一处。
// 注意：改白名单前先确认对应工具已在 RegisterUserTools / 群工具 / MCP 网关注册，
// 且工具自身按 CallerContext 做数据边界（见 botAllowedTools 的隐私注释）。

// SidebarAllowedTools 侧边栏 AI（元对话）可调用的工具白名单。
// 单一来源：buildSidebarSystemPrompt 用它注入能力自述，streamCompletionWithTools 用它作为
// 实际 allowlist，保证「侧边栏 AI 自述的能力」与「它真实能调用的工具」严格一致，避免漂移。
// send_message 在此入口为确认制：执行时生成待确认请求，用户点击确认后才真正发出
// （见 AIPendingActionService），不同于 bot 1:1 入口的直接排除。
var SidebarAllowedTools = []string{
	"create_user_task",
	"list_tasks",
	"send_message",
	"search_knowledge",
	"summarize_conversation",
	"list_calendar_events",
	"create_calendar_event",
	"search_files",
}

// botAllowedTools 专属机器人 1:1 会话可调用的工具白名单。
// 按 talker scope（CallerContext.UserID = 和 bot 对话的用户），不暴露创建者私有数据：
// list_tasks / create_user_task / search_knowledge 均按 ctx.UserID 检索，别人和 bot 对话
// 只能读到他自己的任务/知识。不含 send_message（防 bot 代用户向其他会话发消息，滥用风险）；
// 不含 summarize_conversation（群场景导向，1:1 价值有限）。
var botAllowedTools = []string{
	"list_tasks",
	"create_user_task",
	"search_knowledge",
}

// groupAssistantToolWhitelist 群 @AI 内置群管理工具白名单。
// 实际放行 = 本清单 +（若开启）外部 MCP 工具，见 SmartReplyGraph.groupAssistantAllowedTools。
var groupAssistantToolWhitelist = []string{
	"group_management", "create_group_task", "search_messages", "group_summary", "system_notification",
}
