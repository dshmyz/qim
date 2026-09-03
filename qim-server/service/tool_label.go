package service

// userToolLabels 用户侧工具精确名 → 展示标签。
// bot 1:1 工具卡片（FriendlyToolLabel）与侧边栏工具事件（handler.toolDisplayName）共用，
// 单一来源，避免同一工具两处标签不一致。新增工具只需在此加一行。
var userToolLabels = map[string]string{
	"create_user_task":       "创建任务",
	"list_tasks":             "查询任务",
	"send_message":           "发送消息",
	"search_knowledge":       "搜索知识库",
	"summarize_conversation": "总结会话",
	"list_calendar_events":   "查询日程",
	"create_calendar_event":  "创建日程",
	"search_files":           "搜索文件",
}

// UserToolLabel 精确工具名 → 展示标签；未命中返回 false（由调用方按各自语境兜底）。
func UserToolLabel(tool string) (string, bool) {
	label, ok := userToolLabels[tool]
	return label, ok
}
