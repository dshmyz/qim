package service

import (
	"context"
	"fmt"

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
type SendMessageTool struct {
	messageService *MessageService
}

func NewSendMessageTool(messageService *MessageService) *SendMessageTool {
	return &SendMessageTool{messageService: messageService}
}

func (t *SendMessageTool) Name() string { return "send_message" }

func (t *SendMessageTool) Description() string {
	return "在当前会话中发送一条消息。需要传入消息内容。如果用户没有指定会话，则使用当前打开的会话。"
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

	_, err := t.messageService.SendMessage(convID, userID, "text", content, nil)
	if err != nil {
		return nil, fmt.Errorf("发送消息失败: %w", err)
	}

	return map[string]interface{}{"sent": true, "conversation_id": convID}, nil
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
func RegisterUserTools(registry *ai.ToolRegistry, taskSvc *TaskService, msgSvc *MessageService, searchGraph *UnifiedSearchGraph, summaryGraph *SummaryGraph) {
	if taskSvc != nil {
		registry.RegisterTool(NewCreateTaskTool(taskSvc))
		registry.RegisterTool(NewListTasksTool(taskSvc))
	}
	if msgSvc != nil {
		registry.RegisterTool(NewSendMessageTool(msgSvc))
	}
	if searchGraph != nil {
		registry.RegisterTool(NewSearchKnowledgeTool(searchGraph))
	}
	if summaryGraph != nil {
		registry.RegisterTool(NewSummarizeConversationTool(summaryGraph))
	}
}
