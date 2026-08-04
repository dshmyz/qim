// Package tools 把 QIM Bot API 的操作封装为标准 MCP 工具。
// agent（Claude Code/Cursor 等）经 MCP 调用这些工具即可在 QIM 内收发消息、
// 管理待办、安排日历，无需手搓轮询脚本。
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dshmyz/qim/qim-server/cmd/qim-mcp/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Adapter 持有 Bot API 客户端，方法作为 MCP tool handler。
type Adapter struct {
	api *client.BotAPIClient
}

// New 创建 Adapter。
func New(api *client.BotAPIClient) *Adapter { return &Adapter{api: api} }

// Register 在 MCP server 上注册全部工具。
func Register(s *mcp.Server, a *Adapter) {
	// --- 消息收发（Bot Token）---

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_messages",
		Description: "列出指定 QIM 会话（thread）的最近消息。用于首次进入会话时读取历史。返回每行一条 JSON 消息。",
	}, a.listMessages)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "poll_messages",
		Description: "增量拉取指定会话中 after_id 之后的新消息。用于感知用户回复，建议轮询时传入上次拿到的最大消息 id。返回每行一条 JSON 消息。",
	}, a.pollMessages)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "send_message",
		Description: "向指定用户发送一条消息（thread_id 省略时自动建/复用会话）。msg_type 可选 text|markdown|card（默认 text）；card 时 content 为按钮卡片 JSON。返回 {message_id, conversation_id}。后续 list/poll 用 conversation_id 作 thread_id。\n\n何时用 card：当需要用户在几个明确选项中做决策（确认/取消、批准/拒绝、选择方案）时，优先用 card 而非让用户用文字回复。card 的 content 形如 {\"title\":\"标题\",\"text\":\"说明\",\"buttons\":[{\"id\":\"confirm\",\"text\":\"确认\"},{\"id\":\"cancel\",\"text\":\"取消\"}]}，每个 button 需 id 和 text，可选 style(value=\"primary\")。用户点击后你会通过 poll/list 收到 type=card_action 的消息，含 action_id 字段标识点了哪个按钮。",
	}, a.sendMessage)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "start_streaming_message",
		Description: "在指定用户会话中创建一条流式消息（typing 占位），返回 {message_id, conversation_id}。随后用 append_streaming_chunk 追加内容，最后用 finish_streaming_message 收尾。thread_id 省略时自动建/复用会话。",
	}, a.startStreaming)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "append_streaming_chunk",
		Description: "向 start_streaming_message 返回的流式消息追加一段内容增量（delta）。可多次调用。",
	}, a.appendChunk)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "finish_streaming_message",
		Description: "收尾一条流式消息，将其转为最终 markdown 渲染。调用后该消息不再接受追加。",
	}, a.finishStreaming)

	// --- 消息增强 ---

	mcp.AddTool(s, &mcp.Tool{
		Name:        "edit_message",
		Description: "更新一条已存在的 bot 消息内容（用于卡片状态回写、修正错误等场景）。msg_type 可选，留空保持原类型。返回 {success:true}。",
	}, a.editMessage)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_messages",
		Description: "按关键词搜索历史消息（需要用户 JWT）。keyword 为搜索词，conversation_id 可选限定会话，limit 控制返回条数。返回每行一条搜索命中 JSON。",
	}, a.searchMessages)

	// --- 任务管理 ---

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_tasks",
		Description: "列出当前用户的待办任务（需要用户 JWT）。status 可选过滤状态（todo|doing|done），limit 控制条数。返回每行一条任务 JSON。",
	}, a.listTasks)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_task",
		Description: "创建一条待办任务（需要用户 JWT）。priority 可选 low|medium|high（默认 medium），due_date 格式 YYYY-MM-DD。返回 {task_id}。",
	}, a.createTask)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "update_task",
		Description: "更新一条待办任务的字段（需要用户 JWT）。status 可选 todo|doing|done，priority 可选 low|medium|high。返回 {success:true}。",
	}, a.updateTask)

	// --- 日历事件 ---

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_events",
		Description: "列出当前用户的日历事件（需要用户 JWT）。limit 控制返回条数。返回每行一条事件 JSON。",
	}, a.listEvents)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_event",
		Description: "创建一条日历事件（需要用户 JWT）。start/end 为 RFC3339 格式时间字符串（如 2026-08-01T14:00:00+08:00），reminder 为提前提醒分钟数（0=不提醒）。返回 {event_id}。",
	}, a.createEvent)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "update_event",
		Description: "更新一条日历事件的字段（需要用户 JWT）。返回 {success:true}。",
	}, a.updateEvent)
}

// --- 参数类型 ---

type listMessagesParams struct {
	ThreadID uint64 `json:"thread_id"`
	Limit    int    `json:"limit,omitempty"`
}

type pollMessagesParams struct {
	ThreadID uint64 `json:"thread_id"`
	AfterID  uint64 `json:"after_id"`
}

type sendMessageParams struct {
	ToUserID uint64 `json:"to_user_id"`
	ThreadID uint64 `json:"thread_id,omitempty"`
	Content  string `json:"content"`
	MsgType  string `json:"msg_type,omitempty"`
}

type startStreamingParams struct {
	ToUserID uint64 `json:"to_user_id"`
	ThreadID uint64 `json:"thread_id,omitempty"`
}

type appendChunkParams struct {
	MessageID uint64 `json:"message_id"`
	Delta     string `json:"delta"`
}

type finishStreamingParams struct {
	MessageID uint64 `json:"message_id"`
}

type editMessageParams struct {
	MessageID uint64 `json:"message_id"`
	Content   string `json:"content"`
	MsgType   string `json:"msg_type,omitempty"`
}

type searchMessagesParams struct {
	Keyword        string `json:"keyword"`
	ConversationID uint64 `json:"conversation_id,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

type listTasksParams struct {
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type createTaskParams struct {
	Title       string `json:"title"`
	DueDate     string `json:"due_date,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Description string `json:"description,omitempty"`
}

type updateTaskParams struct {
	TaskID   uint64 `json:"task_id"`
	Status   string `json:"status,omitempty"`
	Priority string `json:"priority,omitempty"`
	Title    string `json:"title,omitempty"`
	DueDate  string `json:"due_date,omitempty"`
}

type listEventsParams struct {
	Limit int `json:"limit,omitempty"`
}

type createEventParams struct {
	Title       string `json:"title"`
	Start       string `json:"start"`
	End         string `json:"end"`
	Reminder    int    `json:"reminder,omitempty"`
	Description string `json:"description,omitempty"`
}

type updateEventParams struct {
	EventID  uint64 `json:"event_id"`
	Title    string `json:"title,omitempty"`
	Start    string `json:"start,omitempty"`
	End      string `json:"end,omitempty"`
	Reminder int    `json:"reminder,omitempty"`
}

// --- Handlers ---

func (a *Adapter) listMessages(ctx context.Context, req *mcp.CallToolRequest, p listMessagesParams) (*mcp.CallToolResult, any, error) {
	msgs, err := a.api.ListMessages(p.ThreadID, 0, p.Limit)
	if err != nil {
		return nil, nil, fmt.Errorf("list_messages 失败: %w", err)
	}
	return textResult(messagesToLines(msgs)), nil, nil
}

func (a *Adapter) pollMessages(ctx context.Context, req *mcp.CallToolRequest, p pollMessagesParams) (*mcp.CallToolResult, any, error) {
	if p.AfterID == 0 {
		return nil, nil, fmt.Errorf("poll_messages 需传入 after_id（上次最大消息 id）")
	}
	msgs, err := a.api.ListMessages(p.ThreadID, p.AfterID, 200)
	if err != nil {
		return nil, nil, fmt.Errorf("poll_messages 失败: %w", err)
	}
	return textResult(messagesToLines(msgs)), nil, nil
}

func (a *Adapter) sendMessage(ctx context.Context, req *mcp.CallToolRequest, p sendMessageParams) (*mcp.CallToolResult, any, error) {
	msgType := p.MsgType
	if msgType == "" {
		msgType = "text"
	}
	id, convID, err := a.api.SendMessage(p.ToUserID, p.ThreadID, p.Content, msgType)
	if err != nil {
		return nil, nil, fmt.Errorf("send_message 失败: %w", err)
	}
	return textResult(fmt.Sprintf(`{"message_id":%d,"conversation_id":%d}`, id, convID)), nil, nil
}

func (a *Adapter) startStreaming(ctx context.Context, req *mcp.CallToolRequest, p startStreamingParams) (*mcp.CallToolResult, any, error) {
	id, convID, err := a.api.SendMessage(p.ToUserID, p.ThreadID, "", "streaming")
	if err != nil {
		return nil, nil, fmt.Errorf("start_streaming_message 失败: %w", err)
	}
	return textResult(fmt.Sprintf(`{"message_id":%d,"conversation_id":%d}`, id, convID)), nil, nil
}

func (a *Adapter) appendChunk(ctx context.Context, req *mcp.CallToolRequest, p appendChunkParams) (*mcp.CallToolResult, any, error) {
	if err := a.api.StreamChunk(p.MessageID, p.Delta, false); err != nil {
		return nil, nil, fmt.Errorf("append_streaming_chunk 失败: %w", err)
	}
	return textResult(`{"ok":true}`), nil, nil
}

func (a *Adapter) finishStreaming(ctx context.Context, req *mcp.CallToolRequest, p finishStreamingParams) (*mcp.CallToolResult, any, error) {
	if err := a.api.StreamChunk(p.MessageID, "", true); err != nil {
		return nil, nil, fmt.Errorf("finish_streaming_message 失败: %w", err)
	}
	return textResult(`{"ok":true}`), nil, nil
}

func (a *Adapter) editMessage(ctx context.Context, req *mcp.CallToolRequest, p editMessageParams) (*mcp.CallToolResult, any, error) {
	if err := a.api.EditMessage(p.MessageID, p.Content, p.MsgType); err != nil {
		return nil, nil, fmt.Errorf("edit_message 失败: %w", err)
	}
	return textResult(`{"success":true}`), nil, nil
}

func (a *Adapter) searchMessages(ctx context.Context, req *mcp.CallToolRequest, p searchMessagesParams) (*mcp.CallToolResult, any, error) {
	items, err := a.api.SearchMessages(p.Keyword, p.ConversationID, p.Limit)
	if err != nil {
		return nil, nil, fmt.Errorf("search_messages 失败: %w", err)
	}
	var b strings.Builder
	for _, item := range items {
		line, _ := json.Marshal(item)
		b.Write(line)
		b.WriteByte('\n')
	}
	return textResult(b.String()), nil, nil
}

func (a *Adapter) listTasks(ctx context.Context, req *mcp.CallToolRequest, p listTasksParams) (*mcp.CallToolResult, any, error) {
	limit := p.Limit
	if limit == 0 {
		limit = 50
	}
	tasks, err := a.api.ListTasks(p.Status, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("list_tasks 失败: %w", err)
	}
	var b strings.Builder
	for _, t := range tasks {
		line, _ := json.Marshal(t)
		b.Write(line)
		b.WriteByte('\n')
	}
	return textResult(b.String()), nil, nil
}

func (a *Adapter) createTask(ctx context.Context, req *mcp.CallToolRequest, p createTaskParams) (*mcp.CallToolResult, any, error) {
	priority := p.Priority
	if priority == "" {
		priority = "medium"
	}
	id, err := a.api.CreateTask(p.Title, p.DueDate, priority, p.Description)
	if err != nil {
		return nil, nil, fmt.Errorf("create_task 失败: %w", err)
	}
	return textResult(fmt.Sprintf(`{"task_id":%d}`, id)), nil, nil
}

func (a *Adapter) updateTask(ctx context.Context, req *mcp.CallToolRequest, p updateTaskParams) (*mcp.CallToolResult, any, error) {
	fields := make(map[string]any)
	if p.Status != "" {
		fields["status"] = p.Status
	}
	if p.Priority != "" {
		fields["priority"] = p.Priority
	}
	if p.Title != "" {
		fields["title"] = p.Title
	}
	if p.DueDate != "" {
		fields["due_date"] = p.DueDate
	}
	if err := a.api.UpdateTask(p.TaskID, fields); err != nil {
		return nil, nil, fmt.Errorf("update_task 失败: %w", err)
	}
	return textResult(`{"success":true}`), nil, nil
}

func (a *Adapter) listEvents(ctx context.Context, req *mcp.CallToolRequest, p listEventsParams) (*mcp.CallToolResult, any, error) {
	limit := p.Limit
	if limit == 0 {
		limit = 50
	}
	events, err := a.api.ListEvents(limit)
	if err != nil {
		return nil, nil, fmt.Errorf("list_events 失败: %w", err)
	}
	var b strings.Builder
	for _, e := range events {
		line, _ := json.Marshal(e)
		b.Write(line)
		b.WriteByte('\n')
	}
	return textResult(b.String()), nil, nil
}

func (a *Adapter) createEvent(ctx context.Context, req *mcp.CallToolRequest, p createEventParams) (*mcp.CallToolResult, any, error) {
	id, err := a.api.CreateEvent(p.Title, p.Start, p.End, p.Reminder, p.Description)
	if err != nil {
		return nil, nil, fmt.Errorf("create_event 失败: %w", err)
	}
	return textResult(fmt.Sprintf(`{"event_id":%d}`, id)), nil, nil
}

func (a *Adapter) updateEvent(ctx context.Context, req *mcp.CallToolRequest, p updateEventParams) (*mcp.CallToolResult, any, error) {
	fields := make(map[string]any)
	if p.Title != "" {
		fields["title"] = p.Title
	}
	if p.Start != "" {
		fields["start"] = p.Start
	}
	if p.End != "" {
		fields["end"] = p.End
	}
	if p.Reminder >= 0 {
		fields["reminder"] = p.Reminder
	}
	if err := a.api.UpdateEvent(p.EventID, fields); err != nil {
		return nil, nil, fmt.Errorf("update_event 失败: %w", err)
	}
	return textResult(`{"success":true}`), nil, nil
}

// --- Helpers ---

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

func messagesToLines(msgs []client.Message) string {
	if len(msgs) == 0 {
		return ""
	}
	var b strings.Builder
	for i := range msgs {
		line, _ := json.Marshal(msgs[i])
		b.Write(line)
		b.WriteByte('\n')
	}
	return b.String()
}
