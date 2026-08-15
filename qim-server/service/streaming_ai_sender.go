package service

import (
	"encoding/json"
	"strings"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

// ToolCallRecord 一条工具调用的结构化记录：前端据此渲染独立的工具调用卡片，
// 不拼进 markdown 正文。tool_label 为面向用户的中文动作名词（如「查询天气」「计算」），
// args 为调用参数摘要，status 标记状态（"running" 进行中 | "ok" 已完成 | "error" 失败），
// 由前端渲染进行态/✓/失败。ID 为同一工具调用的稳定标识（start/end 一致），前端据此把
// 进行态实时更新为终态；仅 WS 实时通道使用，回放（Extra）为终态静态快照。
//
// 该类型下沉到 service 包是为了让 service 层（如专属机器人回复路径）能复用流式 +
// 工具调用基建，而无需 import handler（避免循环依赖）。handler 包通过 type 别名
// `type ToolCallRecord = service.ToolCallRecord` 保持零改动引用同一类型。
type ToolCallRecord struct {
	ID        string                 `json:"id,omitempty"`
	ToolName  string                 `json:"tool_name,omitempty"` // 原始工具名（如 mcp_server_search），供前端 fallback 展示
	ToolLabel string                 `json:"tool_label"`
	Args      map[string]interface{} `json:"args,omitempty"`
	Status    string                 `json:"status,omitempty"` // "running" | "ok" | "error"
}

// StreamingAISender 抽象 handler 层的流式 AI 消息发送 + 工具调用事件推送能力，
// 供 service 层（专属机器人回复等）在不 import handler 的前提下复用群 @AI 同款基建。
// 实现方：handler.WebSocketMessageSender（签名天然匹配本接口）。
type StreamingAISender interface {
	// SendStreamingAIMessage 采用懒创建：延迟到首个非空正文块或首个工具调用才落库空消息。
	// 返回 (sendChunk, getMsg, finish)：
	//   sendChunk: 追加正文块、实时广播流式 new_message（is_streaming=true）
	//   getMsg:    幂等获取/创建当前消息（供工具回调锚定 message_id）
	//   finish:    收尾落库 + broadcastNewMessage（更新会话最后消息/未读/广播）
	SendStreamingAIMessage(conversationID uint, assistantName string) (func(string) error, func() *model.Message, func() *model.Message, error)
	// SendToolCallEvent 把一条工具调用作为独立 WS 事件推给会话（type=ai_tool_call），
	// 前端按 message_id 把卡片关联到对应流式 AI 消息。实时推送与 Extra 持久化分离。
	SendToolCallEvent(conversationID uint, msgID uint, record ToolCallRecord)
}

// NewToolCallFeedback 构造带工具回复的 ReAct 进度回调（群 @AI、专属机器人等带工具路径共用）。
//
// 工具调用会触发两次：phase="start" 推一条 status=running 的 ai_tool_call 事件，前端把该
// 行渲染成「进行中」动画态，回应工具执行那几秒的过程反馈；phase="end" 推终态事件（ok/error）
// 并把终态记录收进 toolCalls 供结束时写 Extra 持久化。同一工具调用的 start/end 共享
// toolCallID（即 ai_service 传入的 tc.ID），前端据此按 ID 把进行态更新为终态而非重复追加。
//
// toolTitles / toolDescriptions 为外部 MCP 工具的标题和描述映射（name → value），
// 由 MCPClientGateway 在注册时收集。标题优先于描述，描述优先于内置关键词映射。
func NewToolCallFeedback(sender StreamingAISender, conversationID uint, getMsg func() *model.Message, toolCalls *[]ToolCallRecord, toolTitles, toolDescriptions map[string]string) ai.ReActStepCallback {
	return func(_ int, toolCallID, phase, tool string, args map[string]interface{}, result interface{}, execErr error) {
		if phase == "start" {
			rec := ToolCallRecord{ID: toolCallID, ToolName: tool, ToolLabel: resolveToolLabel(tool, toolTitles, toolDescriptions), Args: args, Status: "running"}
			if msg := getMsg(); msg != nil {
				sender.SendToolCallEvent(conversationID, msg.ID, rec)
			}
			return
		}
		// end：收集终态记录用于持久化，并实时推送终态事件（前端按 ID 覆盖进行态行）
		status := "ok"
		if execErr != nil || result == nil {
			status = "error"
		}
		rec := ToolCallRecord{ID: toolCallID, ToolName: tool, ToolLabel: resolveToolLabel(tool, toolTitles, toolDescriptions), Args: args, Status: status}
		*toolCalls = append(*toolCalls, rec)
		if msg := getMsg(); msg != nil {
			sender.SendToolCallEvent(conversationID, msg.ID, rec)
		}
	}
}

// resolveToolLabel 解析工具调用的显示标签。优先级：MCP Title > MCP Description > 内置关键词映射。
func resolveToolLabel(tool string, toolTitles, toolDescriptions map[string]string) string {
	// 标题最优先（来自 MCP Tool.Title 或 Annotations.Title）
	if toolTitles != nil {
		if title, ok := toolTitles[tool]; ok && title != "" {
			return title
		}
	}
	// 描述次优先（来自 MCP Tool.Description）
	if toolDescriptions != nil {
		if desc, ok := toolDescriptions[tool]; ok && desc != "" {
			return desc
		}
	}
	return FriendlyToolLabel(tool)
}

// FriendlyToolLabel 把内部工具名映射为面向用户的中文动作名词（表意的工具名，不带
// 进行时态）。内置群管理工具（group_management/user_management/...）与外部 mcp_*
// 工具都走这里；mcp_* 工具始终提取可读名（如「查询 Stock price」「Fmt」），
// 不再退化为无意义的「外部服务」。
//
// 调用总是发生在工具执行结束后（feedback 闭包在工具返回后才触发），因此标签用
// 动作名词而非「正在…」进行时；完成/失败由 status + 前端状态徽标体现，避免结束后
// 卡片仍显示「正在 XX」的奇怪语义。
func FriendlyToolLabel(tool string) string {
	switch {
	// 内置群管理工具
	case strings.Contains(tool, "group_management"):
		return "群管理操作"
	case strings.Contains(tool, "user_management"):
		return "用户管理"
	case strings.Contains(tool, "group_summary"):
		return "群聊总结"
	case strings.Contains(tool, "search_messages"):
		return "群消息搜索"
	case strings.Contains(tool, "create_group_task"):
		return "创建群待办"
	case strings.Contains(tool, "system_notification"):
		return "系统通知"
	// 用户侧工具
	case strings.Contains(tool, "list_tasks"), strings.Contains(tool, "create_user_task"):
		return "任务管理"
	case strings.Contains(tool, "search_knowledge"):
		return "知识搜索"
	case strings.Contains(tool, "summarize_conversation"):
		return "会话总结"
	case strings.Contains(tool, "send_message"):
		return "发送消息"
	// 外部 MCP 工具（mcp_<conn>_<tool>）
	case strings.Contains(tool, "calculator"), strings.Contains(tool, "calc"):
		return "计算"
	case strings.Contains(tool, "weather"):
		return "查询天气"
	case strings.Contains(tool, "search"), strings.Contains(tool, "query"):
		return "查询"
	case strings.Contains(tool, "translate"):
		return "翻译"
	case strings.Contains(tool, "image"), strings.Contains(tool, "img"):
		return "生成图片"
	case strings.Contains(tool, "pdf"), strings.Contains(tool, "doc"):
		return "处理文档"
	default:
		// 外部 MCP 工具名格式为 mcp_<conn>_<tool>，始终提取 <tool> 部分作为可读标签，
		// 不再退化为无意义的「外部服务」。这样即使工具名不匹配任何已知关键词，用户也能
		// 看到工具原名（如 fmt、get_info），而非千篇一律的「外部服务」。
		if strings.HasPrefix(tool, "mcp_") {
			trimmed := tool[len("mcp_"):]
			if idx := strings.Index(trimmed, "_"); idx >= 0 {
				name := trimmed[idx+1:]
				if name != "" {
					return friendlyExternalToolName(name)
				}
				// tool 部分为空（如 mcp_conn_），用连接名做标签
				conn := trimmed[:idx]
				if conn != "" {
					return "外部工具（" + formatToolSuffix(conn) + "）"
				}
			}
		}
		return "外部工具"
	}
}

// friendlyExternalToolName 将外部 MCP 工具的 snake_case 名称转为人类可读标签。
// 常见动作动词翻译为中文，其余保留原名格式化显示。
// 例如 get_stock_price → "查询股价"，send_email → "发送邮件"，my_custom_func → "My custom func"。
func friendlyExternalToolName(name string) string {
	// 常见动作前缀 → 中文标签映射（按最长前缀匹配）
	prefixMap := []struct{ prefix, label string }{
		{"get_", "查询"},
		{"fetch_", "获取"},
		{"search_", "搜索"},
		{"lookup_", "查询"},
		{"find_", "查找"},
		{"create_", "创建"},
		{"send_", "发送"},
		{"delete_", "删除"},
		{"update_", "更新"},
		{"list_", "列举"},
		{"query_", "查询"},
		{"calculate_", "计算"},
		{"generate_", "生成"},
		{"download_", "下载"},
		{"upload_", "上传"},
	}

	for _, m := range prefixMap {
		if strings.HasPrefix(name, m.prefix) {
			suffix := name[len(m.prefix):]
			return m.label + " " + formatToolSuffix(suffix)
		}
	}

	// 无匹配前缀时，把 snake_case 转成可读格式（首字母大写）
	return formatToolSuffix(name)
}

// formatToolSuffix 把 snake_case / kebab-case 后缀转为 Title Case 可读文本。
// 例如 stock_price → "Stock price"，fmt-get → "Fmt get"，data → "Data"。
func formatToolSuffix(s string) string {
	if s == "" {
		return "外部工具"
	}
	// 替换下划线和连字符为空格，首字母大写
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.TrimSpace(s)
	if len(s) > 0 {
		s = strings.ToUpper(s[:1]) + s[1:]
	}
	return s
}

// PersistAIMessageExtra 把工具调用记录 + 命中的知识来源合并持久化到消息 Extra（JSON），
// 供回放/REST 回放/刷新后工具卡片与「知识来源」徽章仍可见。实时推送由 feedback 闭包
// 已发（ai_tool_call 事件），此处只管落库快照。无 tool_calls 且无 sources 时为 no-op。
// getMsg 返回 nil（消息从未创建）时安全跳过，不 panic。
func PersistAIMessageExtra(getMsg func() *model.Message, toolCalls []ToolCallRecord, sources []KnowledgeSource) {
	extra := map[string]interface{}{}
	if len(toolCalls) > 0 {
		extra["tool_calls"] = toolCalls
	}
	if len(sources) > 0 {
		extra["knowledge_sources"] = sources
	}
	if len(extra) == 0 {
		return
	}
	msg := getMsg()
	if msg == nil {
		return
	}
	if b, err := json.Marshal(extra); err == nil {
		msg.Extra = string(b)
	} else {
		logger.WithModule("StreamingAI").Error("序列化消息 Extra 失败", "error", err)
	}
}
