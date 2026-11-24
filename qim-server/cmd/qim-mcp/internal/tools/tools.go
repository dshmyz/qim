// Package tools 把 QIM API 的操作封装为标准 MCP 工具。
// agent（Claude Code/Cursor 等）经 MCP 调用这些工具即可在 QIM 内收发消息、
// 管理待办、安排日历、读写笔记，无需手搓轮询脚本。
//
// 参数命名与 cmd/qim CLI 对齐：to/thread/conversation/type/reply_to/file/as_user、
// due/desc、after_id/limit、keyword/conv、message_id/delta/finish/abort 等。
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/dshmyz/qim/qim-server/cmd/qim-mcp/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Adapter 持有 API 客户端，方法作为 MCP tool handler。
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
		Description: "拉取指定会话的一页消息（thread 可为会话 ID 或会话名）。与 CLI `messages list` 同源：按 id 升序，用 after_id 翻页；需要最新水位线时从返回结果里取最大 id。可用 type 过滤消息类型。返回每行一条 JSON 消息。",
	}, a.listMessages)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "poll_messages",
		Description: "增量拉取会话中 after_id 之后的新消息（thread 可为会话 ID 或会话名）。返回 {after_id, messages}：after_id 为本批最大消息 id，下次轮询直接回传。可用 type 过滤，但水位线始终按整批消息推进，不会因过滤而卡住。",
	}, a.pollMessages)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_actions",
		Description: "查看卡片按钮点击事件（等价 CLI `messages actions`，即 type=card_action 的消息）。用于 agent 发交互卡片后确认用户点了哪个按钮。返回每行一条 JSON 消息，action_id 字段标识按钮。",
	}, a.listActions)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_group_conversations",
		Description: "列出该 agent 已入群的群会话（conversation_id + 群名），等价 CLI `groups`。要在群里主动发言前先调用本工具发现 conversation_id。返回每行一条 JSON。",
	}, a.listGroupConversations)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_bot_conversations",
		Description: "列出本 bot 关联的全部会话：群聊（conv_type=group）+ 与用户的 1:1 私聊（conv_type=bot）。用于发现用户私聊的 conversation_id。返回每行一条 JSON。",
	}, a.listBotConversations)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_conversations",
		Description: "列出当前登录用户最近的会话（用户视角，等价 CLI `conversations list`，需要用户令牌）。含会话类型、未读数、最后一条消息。与 list_bot_conversations（bot 视角）不同。",
	}, a.listConversations)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "send_message",
		Description: "发送一条消息。单聊用 to（用户 ID 或用户名），可带 thread（会话 ID 或名）；群聊/已建会话用 conversation（会话 ID，优先于 to/thread）。type 可选 text|markdown|card（默认 text）；reply_to 引用回复；file 上传本地文件并以 markdown 链接发送；as_user=true 以本人身份发送。file 与 as_user 需要用户令牌。card 的 content 为 {\"title\":\"...\",\"text\":\"...\",\"buttons\":[{\"id\":\"confirm\",\"text\":\"确认\"}]}。返回 {message_id, conversation_id}。",
	}, a.sendMessage)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "start_streaming_message",
		Description: "创建一条流式消息占位（typing），返回 {message_id, conversation_id}。目标用 to/thread 或 conversation 指定。随后用 stream_message 追加/收尾。",
	}, a.startStreaming)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "stream_message",
		Description: "向流式消息追加 delta 或收尾（等价 CLI `stream`）。delta 追加内容；finish=true 收尾并转 markdown；abort=true 丢弃占位（判定无需回复时用，避免留下空气泡）。",
	}, a.streamMessage)

	// --- 消息增强 ---

	mcp.AddTool(s, &mcp.Tool{
		Name:        "edit_message",
		Description: "更新一条已存在的 bot 消息内容（卡片状态回写、修正错误等）。type 可选，留空保持原类型。返回 {success:true}。",
	}, a.editMessage)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_messages",
		Description: "按关键词搜索历史消息（需要用户令牌）。keyword 为搜索词，conv 可选限定会话 ID，limit 控制返回条数（默认 20）。返回每行一条搜索命中 JSON。",
	}, a.searchMessages)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "whoami",
		Description: "返回当前用户令牌对应的身份（等价 CLI `whoami`，需要用户令牌）：id/username/nickname。",
	}, a.whoami)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_bot_identity",
		Description: "返回当前 bot 令牌对应的身份：id/name/type/virtual_user_id。virtual_user_id 用于识别群消息里的 @{mention:<id>} 是否指向自己。",
	}, a.getBotIdentity)

	// --- 任务管理 ---

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_tasks",
		Description: "列出当前用户的待办任务（需要用户令牌）。status 可选过滤（todo|doing|done），limit 控制条数。返回每行一条任务 JSON。",
	}, a.listTasks)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_task",
		Description: "创建一条待办任务（需要用户令牌）。title 必填；due 格式 YYYY-MM-DD；priority 可选 low|medium|high（默认 medium）；desc 可选。返回 {task_id}。",
	}, a.createTask)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "update_task",
		Description: "更新一条待办任务的部分字段（需要用户令牌）。status 可选 todo|doing|done，priority 可选 low|medium|high，title/due 可选。返回 {success:true}。",
	}, a.updateTask)

	// --- 日历事件 ---

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_events",
		Description: "列出当前用户的日历事件（需要用户令牌）。limit 控制返回条数。返回每行一条事件 JSON。",
	}, a.listEvents)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_event",
		Description: "创建一条日历事件（需要用户令牌）。start/end 支持 CLI 本地时间写法（如 2026-08-01 14:00）或 RFC3339；reminder 为提前提醒分钟数（0=不提醒）；desc 可选。返回 {event_id}。",
	}, a.createEvent)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "update_event",
		Description: "更新日历事件的部分字段（需要用户令牌）。只传需要改的字段；服务端要求 title/start/end 全量，本工具会先读取现有事件合并，不会误清空其他字段。reminder=0 表示取消提醒，省略表示不改。返回 {success:true}。",
	}, a.updateEvent)

	// --- 笔记管理 ---

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_notes",
		Description: "列出当前用户的正式笔记（需要用户令牌，等价 CLI `note list`，会排除便签）。用于了解用户积累的长期知识。返回每行一条笔记 JSON。",
	}, a.listNotes)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_note",
		Description: "按 id 获取当前用户的单条笔记全文（需要用户令牌）。返回该笔记完整 JSON，含 title/content/tags/summary。",
	}, a.getNote)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_note",
		Description: "创建一条笔记（需要用户令牌）。title 必填，content 可选。返回 {note_id}。",
	}, a.createNote)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "update_note",
		Description: "更新一条已有笔记的部分字段（需要用户令牌）。只传需要改的 title/content；本工具会先读取现有笔记合并，不会把未传的字段清空。返回 {success:true}。",
	}, a.updateNote)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_notes",
		Description: "按语义检索当前用户的笔记（需要用户令牌），返回与 query 最相关的若干条命中（含 title/content/score）。",
	}, a.searchNotes)
}

// --- 参数类型 ---

type listMessagesParams struct {
	Thread  string `json:"thread" jsonschema:"会话 ID 或会话名"`
	AfterID uint64 `json:"after_id,omitempty" jsonschema:"只返回 id 大于该值的消息"`
	Limit   int    `json:"limit,omitempty" jsonschema:"最多返回条数（1-100，默认 50）"`
	Type    string `json:"type,omitempty" jsonschema:"按消息类型过滤：text|markdown|card|card_action|streaming"`
}

type pollMessagesParams struct {
	Thread  string `json:"thread" jsonschema:"会话 ID 或会话名"`
	AfterID uint64 `json:"after_id" jsonschema:"上次拿到的最大消息 id"`
	Limit   int    `json:"limit,omitempty" jsonschema:"最多返回条数（1-100，默认 50）"`
	Type    string `json:"type,omitempty" jsonschema:"按消息类型过滤；水位线仍按整批推进"`
}

type listActionsParams struct {
	Thread  string `json:"thread" jsonschema:"会话 ID 或会话名"`
	AfterID uint64 `json:"after_id,omitempty" jsonschema:"只返回 id 大于该值的消息"`
	Limit   int    `json:"limit,omitempty" jsonschema:"最多返回条数（1-100，默认 50）"`
}

type listConversationsParams struct {
	Limit int `json:"limit,omitempty" jsonschema:"最多返回条数（默认 50）"`
}

type sendMessageParams struct {
	To           string `json:"to,omitempty" jsonschema:"目标用户 ID 或用户名（单聊）；与 conversation 二选一"`
	Thread       string `json:"thread,omitempty" jsonschema:"会话 ID 或会话名（单聊）；省略时自动建/复用会话"`
	Conversation string `json:"conversation,omitempty" jsonschema:"会话 ID（群聊/已建单聊），优先于 to/thread"`
	ReplyTo      uint64 `json:"reply_to,omitempty" jsonschema:"引用回复的消息 ID"`
	Type         string `json:"type,omitempty" jsonschema:"消息类型 text|markdown|card（默认 text）"`
	Content      string `json:"content,omitempty" jsonschema:"消息正文；card 时为按钮卡片 JSON。content 与 file 至少一个"`
	File         string `json:"file,omitempty" jsonschema:"本地文件路径（需要用户令牌）"`
	AsUser       bool   `json:"as_user,omitempty" jsonschema:"以本人身份发送（需要用户令牌）"`
}

type startStreamingParams struct {
	To           string `json:"to,omitempty" jsonschema:"目标用户 ID 或用户名"`
	Thread       string `json:"thread,omitempty" jsonschema:"会话 ID 或会话名"`
	Conversation string `json:"conversation,omitempty" jsonschema:"会话 ID，优先于 to/thread"`
}

type streamMessageParams struct {
	MessageID uint64 `json:"message_id" jsonschema:"流式消息 ID"`
	Delta     string `json:"delta,omitempty" jsonschema:"追加内容"`
	Finish    bool   `json:"finish,omitempty" jsonschema:"为 true 时收尾并转 markdown"`
	Abort     bool   `json:"abort,omitempty" jsonschema:"为 true 时丢弃流式占位（无需回复时）"`
}

type editMessageParams struct {
	MessageID uint64 `json:"message_id"`
	Content   string `json:"content"`
	Type      string `json:"type,omitempty"`
}

type searchMessagesParams struct {
	Keyword string `json:"keyword"`
	Conv    string `json:"conv,omitempty" jsonschema:"限定会话 ID（数字）"`
	Limit   int    `json:"limit,omitempty" jsonschema:"最多返回条数（默认 20）"`
}

type listTasksParams struct {
	Status string `json:"status,omitempty" jsonschema:"todo|doing|done"`
	Limit  int    `json:"limit,omitempty"`
}

type createTaskParams struct {
	Title    string `json:"title"`
	Due      string `json:"due,omitempty" jsonschema:"截止日期 YYYY-MM-DD"`
	Priority string `json:"priority,omitempty" jsonschema:"low|medium|high（默认 medium）"`
	Desc     string `json:"desc,omitempty"`
}

type updateTaskParams struct {
	TaskID   uint64 `json:"task_id"`
	Status   string `json:"status,omitempty"`
	Priority string `json:"priority,omitempty"`
	Title    string `json:"title,omitempty"`
	Due      string `json:"due,omitempty" jsonschema:"截止日期 YYYY-MM-DD"`
}

type listEventsParams struct {
	Limit int `json:"limit,omitempty"`
}

type createEventParams struct {
	Title    string `json:"title"`
	Start    string `json:"start" jsonschema:"开始时间，如 2026-08-01 14:00 或 RFC3339"`
	End      string `json:"end" jsonschema:"结束时间，如 2026-08-01 15:00 或 RFC3339"`
	Reminder int    `json:"reminder,omitempty" jsonschema:"提前提醒分钟数（0 = 不提醒）"`
	Desc     string `json:"desc,omitempty"`
}

type updateEventParams struct {
	EventID  uint64 `json:"event_id"`
	Title    string `json:"title,omitempty"`
	Start    string `json:"start,omitempty" jsonschema:"开始时间，如 2026-08-01 14:00 或 RFC3339"`
	End      string `json:"end,omitempty"`
	Reminder *int   `json:"reminder,omitempty" jsonschema:"提前提醒分钟数；省略 = 不改，0 = 取消提醒"`
	Desc     string `json:"desc,omitempty"`
}

type listNotesParams struct {
	Limit int `json:"limit,omitempty"`
}

type getNoteParams struct {
	NoteID uint64 `json:"note_id"`
}

type createNoteParams struct {
	Title   string `json:"title"`
	Content string `json:"content,omitempty"`
}

type updateNoteParams struct {
	NoteID  uint64  `json:"note_id"`
	Title   *string `json:"title,omitempty" jsonschema:"新标题；省略 = 不改"`
	Content *string `json:"content,omitempty" jsonschema:"新正文；省略 = 不改"`
}

type searchNotesParams struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k,omitempty"`
}

// --- Handlers ---

func (a *Adapter) listMessages(ctx context.Context, req *mcp.CallToolRequest, p listMessagesParams) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(p.Thread) == "" {
		return nil, nil, fmt.Errorf("list_messages 需传入 thread（会话 ID 或会话名）")
	}
	msgs, err := a.api.ListMessages(p.Thread, p.AfterID, p.Limit)
	if err != nil {
		return nil, nil, fmt.Errorf("list_messages 失败: %w", err)
	}
	return textResult(messagesToLines(filterMessages(msgs, p.Type))), nil, nil
}

func (a *Adapter) pollMessages(ctx context.Context, req *mcp.CallToolRequest, p pollMessagesParams) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(p.Thread) == "" {
		return nil, nil, fmt.Errorf("poll_messages 需传入 thread（会话 ID 或会话名）")
	}
	if p.AfterID == 0 {
		return nil, nil, fmt.Errorf("poll_messages 需传入 after_id（上次最大消息 id）")
	}
	limit := p.Limit
	if limit <= 0 {
		limit = 100 // 与 CLI messages poll 的批量一致
	}
	msgs, err := a.api.ListMessages(p.Thread, p.AfterID, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("poll_messages 失败: %w", err)
	}
	var maxID uint64
	for _, m := range msgs {
		if m.ID > maxID {
			maxID = m.ID
		}
	}
	if maxID == 0 {
		maxID = p.AfterID
	}
	result := struct {
		AfterID  uint64           `json:"after_id"`
		Messages []client.Message `json:"messages"`
	}{AfterID: maxID, Messages: filterMessages(msgs, p.Type)}
	if result.Messages == nil {
		result.Messages = []client.Message{}
	}
	line, _ := json.Marshal(result)
	return textResult(string(line)), nil, nil
}

func (a *Adapter) listActions(ctx context.Context, req *mcp.CallToolRequest, p listActionsParams) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(p.Thread) == "" {
		return nil, nil, fmt.Errorf("list_actions 需传入 thread（会话 ID 或会话名）")
	}
	msgs, err := a.api.ListMessages(p.Thread, p.AfterID, p.Limit)
	if err != nil {
		return nil, nil, fmt.Errorf("list_actions 失败: %w", err)
	}
	return textResult(messagesToLines(filterMessages(msgs, "card_action"))), nil, nil
}

// listGroupConversationsParams 无入参，仅占位（go-sdk 要求 handler 必带 input 参数）。
type listGroupConversationsParams struct{}

func (a *Adapter) listGroupConversations(ctx context.Context, req *mcp.CallToolRequest, p listGroupConversationsParams) (*mcp.CallToolResult, any, error) {
	groups, err := a.api.ListBotGroups()
	if err != nil {
		return nil, nil, fmt.Errorf("list_group_conversations 失败: %w", err)
	}
	var lines []string
	for _, g := range groups {
		lines = append(lines, fmt.Sprintf(`{"conversation_id":%d,"group_name":%q}`, g.ConversationID, g.GroupName))
	}
	if len(lines) == 0 {
		lines = append(lines, "(该 agent 尚未被拉入任何群)")
	}
	return textResult(strings.Join(lines, "\n")), nil, nil
}

// listBotConversationsParams 无入参，仅占位。
type listBotConversationsParams struct{}

func (a *Adapter) listBotConversations(ctx context.Context, req *mcp.CallToolRequest, p listBotConversationsParams) (*mcp.CallToolResult, any, error) {
	convs, err := a.api.ListBotConversations()
	if err != nil {
		return nil, nil, fmt.Errorf("list_bot_conversations 失败: %w", err)
	}
	var lines []string
	for _, c := range convs {
		lines = append(lines, fmt.Sprintf(`{"conversation_id":%d,"conv_type":%q,"name":%q}`, c.ConversationID, c.ConvType, c.Name))
	}
	if len(lines) == 0 {
		lines = append(lines, "(该 agent 还没有任何会话：被拉进群或与用户建立私聊后才会出现)")
	}
	return textResult(strings.Join(lines, "\n")), nil, nil
}

func (a *Adapter) listConversations(ctx context.Context, req *mcp.CallToolRequest, p listConversationsParams) (*mcp.CallToolResult, any, error) {
	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}
	convs, err := a.api.ListUserConversations(limit)
	if err != nil {
		return nil, nil, fmt.Errorf("list_conversations 失败: %w", err)
	}
	var lines []string
	for _, c := range convs {
		name := c.Name
		if name == "" {
			name = c.OtherMemberName
		}
		last := ""
		if c.LastMessage != nil {
			last = c.LastMessage.Content
		}
		line, _ := json.Marshal(struct {
			ConversationID uint64 `json:"conversation_id"`
			ConvType       string `json:"conv_type"`
			Name           string `json:"name"`
			UnreadCount    int    `json:"unread_count,omitempty"`
			LastMessage    string `json:"last_message,omitempty"`
		}{c.ID, c.Type, name, c.UnreadCount, last})
		lines = append(lines, string(line))
	}
	if len(lines) == 0 {
		lines = append(lines, "(当前用户还没有任何会话)")
	}
	return textResult(strings.Join(lines, "\n")), nil, nil
}

func (a *Adapter) sendMessage(ctx context.Context, req *mcp.CallToolRequest, p sendMessageParams) (*mcp.CallToolResult, any, error) {
	msgType := p.Type
	if msgType == "" {
		msgType = "text"
	}
	content := p.Content
	if p.File != "" {
		url, name, err := a.api.UploadFile(p.File)
		if err != nil {
			return nil, nil, fmt.Errorf("send_message 上传文件失败: %w（file 需为 qim-mcp 进程可读的本地路径；远程 HTTP 模式指 MCP 服务所在主机）", err)
		}
		link := fmt.Sprintf("📎 [%s](%s)", name, url)
		if content == "" {
			content = link
		} else {
			content += "\n\n" + link
		}
		if msgType == "text" {
			msgType = "markdown"
		}
	}
	if content == "" {
		return nil, nil, fmt.Errorf("send_message 需提供 content 或 file")
	}

	if p.AsUser {
		convID, err := a.resolveUserConversation(p.Conversation, p.To)
		if err != nil {
			return nil, nil, fmt.Errorf("send_message 失败: %w", err)
		}
		id, err := a.api.SendAsUser(convID, content, msgType, p.ReplyTo)
		if err != nil {
			return nil, nil, fmt.Errorf("send_message 失败: %w", err)
		}
		return textResult(fmt.Sprintf(`{"message_id":%d,"conversation_id":%d}`, id, convID)), nil, nil
	}

	id, convID, err := a.api.SendMessage(client.SendParams{
		To:           p.To,
		Thread:       p.Thread,
		Conversation: p.Conversation,
		Content:      content,
		MsgType:      msgType,
		ReplyTo:      p.ReplyTo,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("send_message 失败: %w", err)
	}
	return textResult(fmt.Sprintf(`{"message_id":%d,"conversation_id":%d}`, id, convID)), nil, nil
}

// resolveUserConversation 解析以本人身份发送的目标会话：
// conversation 为数字会话 ID；否则按 to（数字用户 ID 或用户名）建/复用单聊。
func (a *Adapter) resolveUserConversation(conversation, to string) (uint64, error) {
	if conversation != "" {
		id, err := strconv.ParseUint(conversation, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("as_user 下 conversation 需为会话 ID（数字）: %s", conversation)
		}
		return id, nil
	}
	if strings.TrimSpace(to) == "" {
		return 0, fmt.Errorf("as_user 需要 to（用户 ID 或用户名）或 conversation（会话 ID）")
	}
	var recipientID uint64
	if id, ok := parseNumeric(to); ok {
		recipientID = id
	} else {
		var err error
		recipientID, err = a.api.SearchUserID(to)
		if err != nil {
			return 0, err
		}
	}
	return a.api.CreateSingleConversation(recipientID)
}

func (a *Adapter) startStreaming(ctx context.Context, req *mcp.CallToolRequest, p startStreamingParams) (*mcp.CallToolResult, any, error) {
	if p.To == "" && p.Conversation == "" {
		return nil, nil, fmt.Errorf("start_streaming_message 需要 to（用户）或 conversation（会话 ID）")
	}
	id, convID, err := a.api.SendMessage(client.SendParams{
		To:           p.To,
		Thread:       p.Thread,
		Conversation: p.Conversation,
		Content:      "",
		MsgType:      "streaming",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("start_streaming_message 失败: %w", err)
	}
	return textResult(fmt.Sprintf(`{"message_id":%d,"conversation_id":%d}`, id, convID)), nil, nil
}

func (a *Adapter) streamMessage(ctx context.Context, req *mcp.CallToolRequest, p streamMessageParams) (*mcp.CallToolResult, any, error) {
	if p.MessageID == 0 {
		return nil, nil, fmt.Errorf("stream_message 需传入 message_id")
	}
	if !p.Finish && !p.Abort && p.Delta == "" {
		return nil, nil, fmt.Errorf("stream_message 需提供 delta、finish 或 abort 之一")
	}
	if err := a.api.StreamChunk(p.MessageID, p.Delta, p.Finish, p.Abort); err != nil {
		return nil, nil, fmt.Errorf("stream_message 失败: %w", err)
	}
	return textResult(`{"ok":true}`), nil, nil
}

func (a *Adapter) editMessage(ctx context.Context, req *mcp.CallToolRequest, p editMessageParams) (*mcp.CallToolResult, any, error) {
	if err := a.api.EditMessage(p.MessageID, p.Content, p.Type); err != nil {
		return nil, nil, fmt.Errorf("edit_message 失败: %w", err)
	}
	return textResult(`{"success":true}`), nil, nil
}

func (a *Adapter) searchMessages(ctx context.Context, req *mcp.CallToolRequest, p searchMessagesParams) (*mcp.CallToolResult, any, error) {
	if p.Keyword == "" {
		return nil, nil, fmt.Errorf("search_messages 需传入 keyword")
	}
	items, err := a.api.SearchMessages(p.Keyword, p.Conv, p.Limit)
	if err != nil {
		return nil, nil, fmt.Errorf("search_messages 失败: %w", err)
	}
	return textResult(marshalLines(items)), nil, nil
}

// whoamiParams 无入参，仅占位。
type whoamiParams struct{}

func (a *Adapter) whoami(ctx context.Context, req *mcp.CallToolRequest, p whoamiParams) (*mcp.CallToolResult, any, error) {
	u, err := a.api.WhoamiUser()
	if err != nil {
		return nil, nil, fmt.Errorf("whoami 失败: %w", err)
	}
	line, _ := json.Marshal(u)
	return textResult(string(line)), nil, nil
}

// getBotIdentityParams 无入参，仅占位。
type getBotIdentityParams struct{}

func (a *Adapter) getBotIdentity(ctx context.Context, req *mcp.CallToolRequest, p getBotIdentityParams) (*mcp.CallToolResult, any, error) {
	b, err := a.api.WhoamiBot()
	if err != nil {
		return nil, nil, fmt.Errorf("get_bot_identity 失败: %w", err)
	}
	line, _ := json.Marshal(b)
	return textResult(string(line)), nil, nil
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
	return textResult(marshalLines(tasks)), nil, nil
}

func (a *Adapter) createTask(ctx context.Context, req *mcp.CallToolRequest, p createTaskParams) (*mcp.CallToolResult, any, error) {
	if p.Title == "" {
		return nil, nil, fmt.Errorf("create_task 需传入 title")
	}
	priority := p.Priority
	if priority == "" {
		priority = "medium"
	}
	id, err := a.api.CreateTask(p.Title, p.Due, priority, p.Desc)
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
	if p.Due != "" {
		fields["due_date"] = p.Due
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
	return textResult(marshalLines(events)), nil, nil
}

func (a *Adapter) createEvent(ctx context.Context, req *mcp.CallToolRequest, p createEventParams) (*mcp.CallToolResult, any, error) {
	if p.Title == "" || p.Start == "" || p.End == "" {
		return nil, nil, fmt.Errorf("create_event 需传入 title/start/end")
	}
	start, err := client.LocalToRFC3339(p.Start)
	if err != nil {
		return nil, nil, fmt.Errorf("create_event start 格式错误: %w", err)
	}
	end, err := client.LocalToRFC3339(p.End)
	if err != nil {
		return nil, nil, fmt.Errorf("create_event end 格式错误: %w", err)
	}
	id, err := a.api.CreateEvent(p.Title, start, end, p.Reminder, p.Desc)
	if err != nil {
		return nil, nil, fmt.Errorf("create_event 失败: %w", err)
	}
	return textResult(fmt.Sprintf(`{"event_id":%d}`, id)), nil, nil
}

func (a *Adapter) updateEvent(ctx context.Context, req *mcp.CallToolRequest, p updateEventParams) (*mcp.CallToolResult, any, error) {
	if p.Title == "" && p.Start == "" && p.End == "" && p.Reminder == nil && p.Desc == "" {
		return nil, nil, fmt.Errorf("update_event 至少指定一个要修改的字段")
	}
	cur, err := a.api.GetEvent(p.EventID)
	if err != nil {
		return nil, nil, fmt.Errorf("update_event 读取现有事件失败: %w", err)
	}
	title, start, end := cur.Title, cur.Start, cur.End
	reminder := cur.Reminder
	desc := cur.Description
	if p.Title != "" {
		title = p.Title
	}
	if p.Start != "" {
		if start, err = client.LocalToRFC3339(p.Start); err != nil {
			return nil, nil, fmt.Errorf("update_event start 格式错误: %w", err)
		}
	}
	if p.End != "" {
		if end, err = client.LocalToRFC3339(p.End); err != nil {
			return nil, nil, fmt.Errorf("update_event end 格式错误: %w", err)
		}
	}
	if p.Reminder != nil {
		reminder = *p.Reminder
	}
	if p.Desc != "" {
		desc = p.Desc
	}
	fields := map[string]any{
		"title":    title,
		"start":    start,
		"end":      end,
		"reminder": reminder,
	}
	if desc != "" {
		fields["description"] = desc
	}
	if err := a.api.UpdateEvent(p.EventID, fields); err != nil {
		return nil, nil, fmt.Errorf("update_event 失败: %w", err)
	}
	return textResult(`{"success":true}`), nil, nil
}

func (a *Adapter) listNotes(ctx context.Context, req *mcp.CallToolRequest, p listNotesParams) (*mcp.CallToolResult, any, error) {
	limit := p.Limit
	if limit == 0 {
		limit = 50
	}
	notes, err := a.api.ListNotes(limit)
	if err != nil {
		return nil, nil, fmt.Errorf("list_notes 失败: %w", err)
	}
	return textResult(marshalLines(notes)), nil, nil
}

func (a *Adapter) getNote(ctx context.Context, req *mcp.CallToolRequest, p getNoteParams) (*mcp.CallToolResult, any, error) {
	note, err := a.api.GetNote(p.NoteID)
	if err != nil {
		return nil, nil, fmt.Errorf("get_note 失败: %w", err)
	}
	line, _ := json.Marshal(note)
	return textResult(string(line)), nil, nil
}

func (a *Adapter) createNote(ctx context.Context, req *mcp.CallToolRequest, p createNoteParams) (*mcp.CallToolResult, any, error) {
	if p.Title == "" {
		return nil, nil, fmt.Errorf("create_note 需传入 title")
	}
	id, err := a.api.CreateNote(p.Title, p.Content)
	if err != nil {
		return nil, nil, fmt.Errorf("create_note 失败: %w", err)
	}
	return textResult(fmt.Sprintf(`{"note_id":%d}`, id)), nil, nil
}

func (a *Adapter) updateNote(ctx context.Context, req *mcp.CallToolRequest, p updateNoteParams) (*mcp.CallToolResult, any, error) {
	if p.Title == nil && p.Content == nil {
		return nil, nil, fmt.Errorf("update_note 至少提供 title 或 content 之一")
	}
	cur, err := a.api.GetNote(p.NoteID)
	if err != nil {
		return nil, nil, fmt.Errorf("update_note 读取现有笔记失败: %w", err)
	}
	title, content := cur.Title, cur.Content
	if p.Title != nil {
		title = *p.Title
	}
	if p.Content != nil {
		content = *p.Content
	}
	if err := a.api.UpdateNote(p.NoteID, title, content); err != nil {
		return nil, nil, fmt.Errorf("update_note 失败: %w", err)
	}
	return textResult(`{"success":true}`), nil, nil
}

func (a *Adapter) searchNotes(ctx context.Context, req *mcp.CallToolRequest, p searchNotesParams) (*mcp.CallToolResult, any, error) {
	if p.Query == "" {
		return nil, nil, fmt.Errorf("search_notes 需传入 query")
	}
	topK := p.TopK
	if topK == 0 {
		topK = 5
	}
	results, err := a.api.SearchNotesSemantic(p.Query, topK)
	if err != nil {
		return nil, nil, fmt.Errorf("search_notes 失败: %w", err)
	}
	return textResult(marshalLines(results)), nil, nil
}

// --- Helpers ---

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

func messagesToLines(msgs []client.Message) string {
	return marshalLines(msgs)
}

func marshalLines[T any](items []T) string {
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	for i := range items {
		line, _ := json.Marshal(items[i])
		b.Write(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// filterMessages 按 type 过滤；空 type 不过滤。
func filterMessages(msgs []client.Message, typ string) []client.Message {
	if typ == "" {
		return msgs
	}
	filtered := make([]client.Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Type == typ {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// parseNumeric 判断字符串是否为纯数字 ID。
func parseNumeric(s string) (uint64, bool) {
	if s == "" {
		return 0, false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	id, err := strconv.ParseUint(s, 10, 64)
	return id, err == nil
}
