package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
)

// ListTasksTool 查询用户任务列表。
type ListTasksTool struct {
	taskService *TaskService
}

func NewListTasksTool(taskService *TaskService) *ListTasksTool {
	return &ListTasksTool{taskService: taskService}
}

func (t *ListTasksTool) Name() string { return "list_tasks" }

func (t *ListTasksTool) Description() string {
	return "查询当前用户的任务/待办列表，可指定状态筛选。返回任务标题、截止日期、优先级、状态等。"
}

func (t *ListTasksTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"status": map[string]interface{}{
			"type":        "string",
			"description": "状态筛选：todo、done、all（可选，默认 all）",
			"required":    false,
		},
	}
}

func (t *ListTasksTool) Execute(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	if t.taskService == nil {
		return nil, fmt.Errorf("task service not available")
	}

	var userID uint
	if ctx != nil {
		userID = ctx.UserID
	}
	if userID == 0 {
		return nil, fmt.Errorf("需要登录后才能查询任务")
	}

	tasks, err := t.taskService.GetTasks(userID)
	if err != nil {
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}

	statusFilter := "all"
	if s, ok := params["status"].(string); ok && s != "" {
		statusFilter = s
	}

	var result []map[string]interface{}
	for _, task := range tasks {
		if statusFilter != "all" && task.Status != statusFilter {
			continue
		}
		result = append(result, map[string]interface{}{
			"id":       task.ID,
			"title":    task.Title,
			"due_date": task.DueDate,
			"priority": task.Priority,
			"status":   task.Status,
		})
	}

	return map[string]interface{}{"tasks": result, "count": len(result)}, nil
}

// SendMessageTool 让 AI 代替用户发送消息到指定会话。
// 当 CallerContext.ConfirmTools 命中本工具时为确认制：不直接发送，而是生成待确认
// 记录（AIPendingActionService），由用户在客户端点击确认后才真正发出。
// 直接发送的旧路径保留给未启用确认制的入口与测试。
type SendMessageTool struct {
	messageService *MessageService
	pendings       *AIPendingActionService
}

func NewSendMessageTool(messageService *MessageService, pendings *AIPendingActionService) *SendMessageTool {
	return &SendMessageTool{messageService: messageService, pendings: pendings}
}

func (t *SendMessageTool) Name() string { return "send_message" }

func (t *SendMessageTool) Description() string {
	return "在当前会话中发送一条消息。需要传入消息内容。如果用户没有指定会话，则使用当前打开的会话。" +
		"注意：发送前会生成待确认请求，需用户在确认条上点击「确认发送」后才会真正发出；请勿重复调用。"
}

func (t *SendMessageTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"content": map[string]interface{}{
			"type":        "string",
			"description": "要发送的消息文本",
			"required":    true,
		},
		"conversation_id": map[string]interface{}{
			"type":        "string",
			"description": "目标会话 ID（可选，默认使用当前会话）",
			"required":    false,
		},
	}
}

func (t *SendMessageTool) Execute(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	if t.messageService == nil {
		return nil, fmt.Errorf("message service not available")
	}

	content, ok := params["content"].(string)
	if !ok || content == "" {
		return nil, fmt.Errorf("content is required")
	}

	var userID uint
	var convID uint
	if ctx != nil {
		userID = ctx.UserID
		convID = ctx.ConversationID
	}

	if cid, ok := params["conversation_id"].(string); ok && cid != "" {
		if _, err := fmt.Sscanf(cid, "%d", &convID); err != nil {
			return nil, fmt.Errorf("conversation_id 格式错误")
		}
	}

	if userID == 0 {
		return nil, fmt.Errorf("需要登录后才能发送消息")
	}
	if convID == 0 {
		return nil, fmt.Errorf("未指定会话，无法发送消息")
	}

	// 确认制入口（如侧边栏 AI）：模型不得自行决定代发，先落待确认记录，
	// 结果文本引导模型向用户复述内容与目标，等用户在确认条上操作。
	if t.pendings != nil && confirmToolRequired(ctx, "send_message") {
		record, err := t.pendings.CreatePendingSend(userID, ctx.ConversationID, convID, content)
		if err != nil {
			return nil, fmt.Errorf("生成待确认发送请求失败: %w", err)
		}
		info := ai.PendingSend{
			ID:                   record.ID,
			TargetConversationID: record.TargetConversationID,
			TargetName:           record.TargetName,
			Preview:              truncatePreview(content, 200),
		}
		return map[string]interface{}{
			"status":  "pending_confirmation",
			"pending": info,
			"note":    "尚未发送。请向用户复述将要发送的内容与目标会话，并请用户在下方确认条点击「确认发送」或「取消」。不要再次调用本工具。",
		}, nil
	}

	_, err := t.messageService.SendMessage(convID, userID, "text", content, nil)
	if err != nil {
		return nil, fmt.Errorf("发送消息失败: %w", err)
	}

	return map[string]interface{}{"sent": true, "conversation_id": convID}, nil
}

// confirmToolRequired 判断工具是否在 CallerContext.ConfirmTools 中（大小写不敏感，
// 与 isToolAllowed 的白名单匹配口径一致）。
func confirmToolRequired(ctx *ai.CallerContext, toolName string) bool {
	if ctx == nil {
		return false
	}
	for _, name := range ctx.ConfirmTools {
		if strings.EqualFold(name, toolName) {
			return true
		}
	}
	return false
}

// PendingSendFromResult 从确认制工具的返回值中提取待确认发送载荷。
// handler（侧边栏 SSE pending 帧、bot 会话确认卡）与测试共用此解析。
func PendingSendFromResult(result interface{}) *ai.PendingSend {
	m, ok := result.(map[string]interface{})
	if !ok {
		return nil
	}
	if status, _ := m["status"].(string); status != "pending_confirmation" {
		return nil
	}
	info, ok := m["pending"].(ai.PendingSend)
	if !ok || info.ID == 0 {
		return nil
	}
	return &info
}

// truncatePreview 截断内容预览（按 rune），供确认条与模型结果共用。
func truncatePreview(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

// SearchKnowledgeTool 让 AI 搜索知识库/笔记/历史消息。
type SearchKnowledgeTool struct {
	unifiedSearchGraph *UnifiedSearchGraph
}

func NewSearchKnowledgeTool(graph *UnifiedSearchGraph) *SearchKnowledgeTool {
	return &SearchKnowledgeTool{unifiedSearchGraph: graph}
}

func (t *SearchKnowledgeTool) Name() string { return "search_knowledge" }

func (t *SearchKnowledgeTool) Description() string {
	return "搜索用户的知识库、笔记和历史消息，用于回答需要上下文的问题。传入查询关键词即可。"
}

func (t *SearchKnowledgeTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"query": map[string]interface{}{
			"type":        "string",
			"description": "搜索关键词或问题",
			"required":    true,
		},
		"conversation_id": map[string]interface{}{
			"type":        "string",
			"description": "限定搜索的会话 ID（可选）",
			"required":    false,
		},
	}
}

func (t *SearchKnowledgeTool) Execute(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	if t.unifiedSearchGraph == nil {
		return nil, fmt.Errorf("knowledge search not available")
	}

	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required")
	}

	var userID, convID uint
	if ctx != nil {
		userID = ctx.UserID
		convID = ctx.ConversationID
	}

	if cid, ok := params["conversation_id"].(string); ok && cid != "" {
		fmt.Sscanf(cid, "%d", &convID)
	}

	if userID == 0 {
		return nil, fmt.Errorf("需要登录后才能搜索")
	}
	if convID > 0 {
		if err := requireConversationMember(userID, convID); err != nil {
			return nil, err
		}
	}

	result, err := t.unifiedSearchGraph.Execute(context.Background(), &UnifiedSearchInput{
		Query:          query,
		UserID:         userID,
		ConversationID: convID,
	})
	if err != nil {
		return nil, fmt.Errorf("搜索失败: %w", err)
	}

	var sources []map[string]interface{}
	for _, src := range result.Sources {
		sources = append(sources, map[string]interface{}{
			"type":    src.Type,
			"title":   src.Title,
			"content": src.Content,
		})
	}
	return map[string]interface{}{"answer": result.Answer, "sources": sources}, nil
}

// SummarizeConversationTool 让 AI 总结会话内容。
type SummarizeConversationTool struct {
	summaryGraph *SummaryGraph
}

func NewSummarizeConversationTool(graph *SummaryGraph) *SummarizeConversationTool {
	return &SummarizeConversationTool{summaryGraph: graph}
}

func (t *SummarizeConversationTool) Name() string { return "summarize_conversation" }

func (t *SummarizeConversationTool) Description() string {
	return "总结指定会话的内容。如果不指定会话，则总结当前打开的会话。支持按时间范围总结。"
}

func (t *SummarizeConversationTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"conversation_id": map[string]interface{}{
			"type":        "string",
			"description": "会话 ID（可选，默认当前会话）",
			"required":    false,
		},
		"time_range": map[string]interface{}{
			"type":        "string",
			"description": "时间范围：1h、today、7d（可选，默认 today）",
			"required":    false,
		},
	}
}

func (t *SummarizeConversationTool) Execute(params map[string]interface{}, ctx *ai.CallerContext) (interface{}, error) {
	if t.summaryGraph == nil {
		return nil, fmt.Errorf("summary graph not available")
	}

	var userID, convID uint
	if ctx != nil {
		userID = ctx.UserID
		convID = ctx.ConversationID
	}

	if cid, ok := params["conversation_id"].(string); ok && cid != "" {
		fmt.Sscanf(cid, "%d", &convID)
	}

	if userID == 0 {
		return nil, fmt.Errorf("需要登录后才能总结")
	}
	if convID == 0 {
		return nil, fmt.Errorf("未指定会话，无法总结")
	}
	if err := requireConversationMember(userID, convID); err != nil {
		return nil, err
	}

	timeRange := "today"
	if tr, ok := params["time_range"].(string); ok && tr != "" {
		timeRange = tr
	}

	result, err := t.summaryGraph.Execute(context.Background(), &SummaryInput{
		ConversationID: convID,
		TimeRange:      timeRange,
		UserID:         userID,
	})
	if err != nil {
		return nil, fmt.Errorf("总结失败: %w", err)
	}

	return map[string]interface{}{
		"summary":        result.Summary,
		"messages_count": result.MessagesCount,
		"active_members": result.ActiveMembers,
	}, nil
}

func requireConversationMember(userID, convID uint) error {
	var count int64
	if err := database.GetDB().Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("校验会话权限失败: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("无权访问该会话")
	}
	return nil
}

// RegisterUserTools 把用户侧 AI 工具注册到进程内工具注册表。
// pendings 为 send_message 的待确认执行服务（nil 时该工具退回直接发送，仅供测试场景）。
func RegisterUserTools(registry *ai.ToolRegistry, taskSvc *TaskService, msgSvc *MessageService, searchGraph *UnifiedSearchGraph, summaryGraph *SummaryGraph, pendings *AIPendingActionService) {
	if taskSvc != nil {
		registry.RegisterTool(NewCreateUserTaskTool(taskSvc))
		registry.RegisterTool(NewListTasksTool(taskSvc))
	}
	if msgSvc != nil {
		registry.RegisterTool(NewSendMessageTool(msgSvc, pendings))
	}
	if searchGraph != nil {
		registry.RegisterTool(NewSearchKnowledgeTool(searchGraph))
	}
	if summaryGraph != nil {
		registry.RegisterTool(NewSummarizeConversationTool(summaryGraph))
	}
}
