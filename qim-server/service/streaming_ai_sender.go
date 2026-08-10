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
func NewToolCallFeedback(sender StreamingAISender, conversationID uint, getMsg func() *model.Message, toolCalls *[]ToolCallRecord) ai.ReActStepCallback {
	return func(_ int, toolCallID, phase, tool string, args map[string]interface{}, result interface{}, execErr error) {
		if phase == "start" {
			rec := ToolCallRecord{ID: toolCallID, ToolLabel: FriendlyToolLabel(tool), Args: args, Status: "running"}
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
		rec := ToolCallRecord{ID: toolCallID, ToolLabel: FriendlyToolLabel(tool), Args: args, Status: status}
		*toolCalls = append(*toolCalls, rec)
		if msg := getMsg(); msg != nil {
			sender.SendToolCallEvent(conversationID, msg.ID, rec)
		}
	}
}

// FriendlyToolLabel 把内部工具名映射为面向用户的中文动作名词（表意的工具名，不带
// 进行时态）。内置群管理工具（group_management/user_management/...）与外部 mcp_*
// 工具都走这里；未命中的退化为通用「外部服务」。
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
		return "外部服务"
	}
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
