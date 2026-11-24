// Package client 是 qim-mcp 与 QIM API 之间的纯 HTTP 客户端。
// 与 cmd/qim CLI 同源：只依赖 REST 契约（GET/POST/PUT /api/v1/... + Bearer token），
// 不耦合 server 内部（service/handler），便于随 agent 侧独立分发。
//
// 参数语义与 CLI 对齐：目标用户/会话既可以是数字 ID，也可以是名称；
// 时间接受 CLI 的本地时间写法并转 RFC3339；用户态操作用 userToken。
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Message 对齐 Bot API 返回的消息形态（与 cmd/qim CLI 一致）。
type Message struct {
	ID             uint64 `json:"id"`
	ConversationID uint64 `json:"conversation_id"`
	SenderID       uint64 `json:"sender_id"`
	SenderType     string `json:"sender_type"`
	SenderNickname string `json:"sender_nickname"`
	Content        string `json:"content"`
	Type           string `json:"type"`
	Origin         string `json:"origin"`
	CreatedAt      string `json:"created_at"`
}

// SearchResultItem 消息搜索结果中的单条命中。
type SearchResultItem struct {
	ID         uint64 `json:"id"`
	Content    string `json:"content"`
	Type       string `json:"type"`
	ConvID     uint64 `json:"conversation_id"`
	SenderName string `json:"sender_name"`
	CreatedAt  string `json:"created_at"`
}

// Task 对齐任务 API 返回的待办形态。
type Task struct {
	ID       uint64  `json:"id"`
	Title    string  `json:"title"`
	Status   string  `json:"status"`
	Priority string  `json:"priority"`
	DueDate  *string `json:"due_date"`
}

// Event 对齐日历事件 API 返回的形态。
type Event struct {
	ID          uint64 `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Start       string `json:"start"`
	End         string `json:"end"`
	AllDay      bool   `json:"all_day"`
	Reminder    int    `json:"reminder"`
}

// BotGroup 是 bot 已入群的群会话摘要。
type BotGroup struct {
	ConversationID uint64 `json:"conversation_id"`
	GroupName      string `json:"group_name"`
}

// BotConversationSummary 是 bot 关联的全部会话摘要（群聊 + 1:1 私聊）。
type BotConversationSummary struct {
	ConversationID uint64 `json:"conversation_id"`
	ConvType       string `json:"conv_type"` // group | bot
	Name           string `json:"name"`      // 群名 或 对端用户昵称（可能为空）
}

// UserConversation 是用户视角的最近会话（对齐 CLI conversations list）。
type UserConversation struct {
	ID              uint64 `json:"id"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	UnreadCount     int    `json:"unread_count"`
	OtherMemberName string `json:"other_member_name"`
	LastMessage     *struct {
		Content string `json:"content"`
	} `json:"last_message"`
}

// BotIdentity 是当前 bot 令牌对应的身份（GET /bot/me）。
type BotIdentity struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Type          string `json:"type"`
	VirtualUserID *uint  `json:"virtual_user_id"`
	IsActive      bool   `json:"is_active"`
}

// UserIdentity 是当前用户令牌对应的身份（GET /users/me）。
type UserIdentity struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// Note 对齐笔记 API 返回的形态（与 cmd/qim CLI 一致）。
type Note struct {
	ID        uint64 `json:"id"`
	UserID    uint64 `json:"user_id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Type      string `json:"type"`
	Tags      string `json:"tags"` // JSON 数组字符串
	Summary   string `json:"summary"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// NoteSearchResult 笔记语义检索的命中项。
type NoteSearchResult struct {
	Content string  `json:"content"`
	Score   float64 `json:"score"`
	Title   string  `json:"title"`
	NoteID  string  `json:"note_id"`
}

// SendParams 汇总一条出站消息的全部目标与内容参数，避免超过四个位置参数。
type SendParams struct {
	To           string // 用户 ID 或用户名（单聊）
	Thread       string // 已弃用：会话 ID 或会话名，等价 Conversation
	Conversation string // 会话 ID 或会话名（群聊/已建单聊），优先于 To/Thread
	Content      string
	MsgType      string
	ReplyTo      uint64
}

// BotAPIClient 封装对 QIM API 的调用，支持 Bot Token 和 User Token 双认证。
type BotAPIClient struct {
	serverURL string
	token     string // Bot 令牌 (qbot_xxx)
	userToken string // 用户令牌（qusr_ 长期令牌或 JWT，用于任务/日历等需要用户身份的接口）
	http      *http.Client
}

// New 创建客户端。serverURL 形如 http://localhost:8080，token 为 qbot_ 开头的明文令牌。
// userToken 为可选的用户令牌，用于调用需要用户身份的接口。
func New(serverURL, token, userToken string) *BotAPIClient {
	return &BotAPIClient{
		serverURL: serverURL,
		token:     token,
		userToken: userToken,
		http:      &http.Client{Timeout: 30 * time.Second},
	}
}

// --- 消息 ---

// ListMessages 拉会话消息。thread 可以是会话 ID 或会话名（与服务端 thread_name 解析一致）。
// afterID=0 拉最近一页（按 id 升序，与 CLI messages list 同源），>0 增量拉 afterID 之后的消息。
func (c *BotAPIClient) ListMessages(thread string, afterID uint64, limit int) ([]Message, error) {
	if thread == "" {
		return nil, fmt.Errorf("缺少 thread（会话 ID 或会话名）")
	}
	u := fmt.Sprintf("%s/api/v1/bot/messages?limit=%d", c.serverURL, clamp(limit))
	if id, ok := parseNumeric(thread); ok {
		u += fmt.Sprintf("&thread_id=%d", id)
	} else {
		u += "&thread_name=" + url.QueryEscape(thread)
	}
	if afterID > 0 {
		u += fmt.Sprintf("&after_id=%d", afterID)
	}
	body, err := c.get(u)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data struct {
			Messages []Message `json:"messages"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	return resp.Data.Messages, nil
}

// ListBotGroups 列出该 bot 已入群的群会话（conversation_id + 群名），供主动群发前发现群。
func (c *BotAPIClient) ListBotGroups() ([]BotGroup, error) {
	body, err := c.get(c.serverURL + "/api/v1/bot/groups")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data []BotGroup `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	return resp.Data, nil
}

// ListBotConversations 列出该 bot 关联的全部会话（群聊 + 1:1 私聊）。
func (c *BotAPIClient) ListBotConversations() ([]BotConversationSummary, error) {
	body, err := c.get(c.serverURL + "/api/v1/bot/conversations")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data []BotConversationSummary `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	return resp.Data, nil
}

// ResolveConversation 把会话 ID 或会话名解析为会话 ID（bot 视角）。
// 名称在 /api/v1/bot/conversations 里精确匹配（群名或 1:1 对端昵称）。
func (c *BotAPIClient) ResolveConversation(nameOrID string) (uint64, error) {
	if id, ok := parseNumeric(nameOrID); ok {
		return id, nil
	}
	body, err := c.get(c.serverURL + "/api/v1/bot/conversations")
	if err != nil {
		return 0, err
	}
	var resp struct {
		Data []BotConversationSummary `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("解析会话列表失败: %w", err)
	}
	matched := make([]BotConversationSummary, 0, 1)
	for _, conv := range resp.Data {
		if conv.Name == nameOrID {
			matched = append(matched, conv)
		}
	}
	switch len(matched) {
	case 0:
		return 0, fmt.Errorf("未找到会话 %q（可用 list_group_conversations / list_bot_conversations 查看）", nameOrID)
	case 1:
		return matched[0].ConversationID, nil
	default:
		return 0, fmt.Errorf("会话名 %q 不唯一（匹配 %d 个），请改用会话 ID", nameOrID, len(matched))
	}
}

// ListUserConversations 列出当前用户的最近会话（对齐 CLI conversations list）。
func (c *BotAPIClient) ListUserConversations(limit int) ([]UserConversation, error) {
	if c.userToken == "" {
		return nil, fmt.Errorf("列出用户会话需要 userToken")
	}
	if limit <= 0 {
		limit = 50
	}
	u := fmt.Sprintf("%s/api/v1/conversations?page_size=%d", c.serverURL, limit)
	body, err := c.userGet(u)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data struct {
			List []UserConversation `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析会话列表失败: %w", err)
	}
	return resp.Data.List, nil
}

// SendMessage 按 CLI send 的语义发送一条消息，返回新消息 ID 与会话 ID。
// conversation（或弃用的 thread）非空时按会话发送（群聊/已建单聊），优先于 to。
// 会话目标既可以是数字 ID，也可以是名称，名称会先在 bot 会话列表里精确解析。
func (c *BotAPIClient) SendMessage(p SendParams) (msgID, convID uint64, err error) {
	msgType := p.MsgType
	if msgType == "" {
		msgType = "text"
	}
	payload := map[string]any{
		"content":  p.Content,
		"msg_type": msgType,
	}
	if p.ReplyTo > 0 {
		payload["reply_to_id"] = p.ReplyTo
	}
	target := p.Conversation
	if target != "" {
		id, err := c.ResolveConversation(target)
		if err != nil {
			return 0, 0, err
		}
		payload["conversation_id"] = id
	} else if p.Thread != "" {
		// 弃用别名：先按会话解析；解析不到时保留旧的 1:1 语义，
		// 把名称交给服务端按用户名/昵称解析（thread_name）。
		if id, err := c.ResolveConversation(p.Thread); err == nil {
			payload["conversation_id"] = id
		} else if p.To != "" {
			setTargetUser(payload, p.To)
			if id, ok := parseNumeric(p.Thread); ok {
				payload["thread_id"] = id
			} else {
				payload["thread_name"] = p.Thread
			}
		} else {
			setTargetUser(payload, p.Thread)
		}
	} else {
		if p.To == "" {
			return 0, 0, fmt.Errorf("缺少 to（用户 ID 或用户名）或 conversation（会话 ID 或名称）")
		}
		setTargetUser(payload, p.To)
	}
	body, _ := json.Marshal(payload)
	respBody, err := c.post(c.serverURL+"/api/v1/bot/messages", body)
	if err != nil {
		return 0, 0, err
	}
	return parseSendResponse(respBody)
}

// SendAsUser 以当前用户身份（userToken）向会话发消息：POST /conversations/:id/messages。
func (c *BotAPIClient) SendAsUser(convID uint64, content, msgType string, replyTo uint64) (uint64, error) {
	if c.userToken == "" {
		return 0, fmt.Errorf("以本人身份发送需要 userToken")
	}
	if msgType == "" {
		msgType = "text"
	}
	payload := map[string]any{
		"type":    msgType,
		"content": content,
	}
	if replyTo > 0 {
		payload["quoted_message_id"] = replyTo
	}
	body, _ := json.Marshal(payload)
	respBody, err := c.userPost(fmt.Sprintf("%s/api/v1/conversations/%d/messages", c.serverURL, convID), body)
	if err != nil {
		return 0, err
	}
	var resp struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return 0, fmt.Errorf("解析响应失败: %w", err)
	}
	if resp.Data.ID == 0 {
		return 0, fmt.Errorf("响应缺少 message_id: %s", truncate(string(respBody), 300))
	}
	return resp.Data.ID, nil
}

// StreamChunk 向流式消息追加一段 delta，finish=true 收尾，abort=true 丢弃占位。
func (c *BotAPIClient) StreamChunk(msgID uint64, delta string, finish, abort bool) error {
	body, _ := json.Marshal(map[string]any{
		"content_delta": delta,
		"finish":        finish,
		"abort":         abort,
	})
	u := fmt.Sprintf("%s/api/v1/bot/messages/%d/stream", c.serverURL, msgID)
	_, err := c.post(u, body)
	return err
}

// EditMessage 更新一条已存在的 bot 消息。
func (c *BotAPIClient) EditMessage(msgID uint64, content, msgType string) error {
	payload := map[string]any{"content": content}
	if msgType != "" {
		payload["msg_type"] = msgType
	}
	body, _ := json.Marshal(payload)
	u := fmt.Sprintf("%s/api/v1/bot/messages/%d", c.serverURL, msgID)
	_, err := c.put(u, body)
	return err
}

// SearchMessages 按关键词搜索消息（需 userToken）。
func (c *BotAPIClient) SearchMessages(keyword, conv string, limit int) ([]SearchResultItem, error) {
	if c.userToken == "" {
		return nil, fmt.Errorf("搜索消息需要 userToken")
	}
	u := fmt.Sprintf("%s/api/v1/messages/search?keyword=%s&pageSize=%d",
		c.serverURL, url.QueryEscape(keyword), clampSearch(limit))
	if conv != "" {
		if _, err := strconv.ParseUint(conv, 10, 64); err != nil {
			return nil, fmt.Errorf("conv 需为会话 ID（数字）: %s", conv)
		}
		u += "&conv_id=" + conv
	}
	body, err := c.userGet(u)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data struct {
			List []SearchResultItem `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析搜索结果失败: %w", err)
	}
	return resp.Data.List, nil
}

// WhoamiBot 返回当前 bot 令牌对应的身份（含 virtual_user_id，用于识别群内 @）。
func (c *BotAPIClient) WhoamiBot() (*BotIdentity, error) {
	body, err := c.get(c.serverURL + "/api/v1/bot/me")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data BotIdentity `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析 bot 身份失败: %w", err)
	}
	return &resp.Data, nil
}

// WhoamiUser 返回当前用户令牌对应的身份（对齐 CLI whoami）。
func (c *BotAPIClient) WhoamiUser() (*UserIdentity, error) {
	if c.userToken == "" {
		return nil, fmt.Errorf("查询用户身份需要 userToken")
	}
	body, err := c.userGet(c.serverURL + "/api/v1/users/me")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data UserIdentity `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析用户身份失败: %w", err)
	}
	return &resp.Data, nil
}

// UploadFile 以用户身份 multipart 上传本地文件，返回 (url, name)。
// 与 CLI --file 一致：用户态上传，再把链接拼进消息正文。
func (c *BotAPIClient) UploadFile(path string) (string, string, error) {
	if c.userToken == "" {
		return "", "", fmt.Errorf("上传文件需要 userToken")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return "", "", fmt.Errorf("构建上传表单失败: %w", err)
	}
	if _, err := io.Copy(fw, f); err != nil {
		return "", "", fmt.Errorf("读取文件失败: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", "", fmt.Errorf("构建上传表单失败: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.serverURL+"/api/v1/upload", &buf)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	c.setUserAuth(req)
	respBody, err := c.do(req)
	if err != nil {
		return "", "", err
	}
	var resp struct {
		Data struct {
			URL  string `json:"url"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", "", fmt.Errorf("解析上传响应失败: %w", err)
	}
	if resp.Data.URL == "" {
		return "", "", fmt.Errorf("上传响应缺少 url: %s", truncate(string(respBody), 300))
	}
	name := resp.Data.Name
	if name == "" {
		name = filepath.Base(path)
	}
	return resp.Data.URL, name, nil
}

// SearchUserID 按用户名精确匹配用户 ID（对齐 CLI --as-user 的用户名解析）。
func (c *BotAPIClient) SearchUserID(name string) (uint64, error) {
	if c.userToken == "" {
		return 0, fmt.Errorf("按用户名解析用户需要 userToken")
	}
	body, err := c.userGet(c.serverURL + "/api/v1/users/search?q=" + url.QueryEscape(name) + "&scope=users")
	if err != nil {
		return 0, err
	}
	var resp struct {
		Data []struct {
			ID       uint64 `json:"id"`
			Username string `json:"username"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("解析搜索响应失败: %w", err)
	}
	for _, u := range resp.Data {
		if u.Username == name {
			return u.ID, nil
		}
	}
	return 0, fmt.Errorf("未找到用户: %s（请确认用户名，或改用数字用户 ID）", name)
}

// CreateSingleConversation 创建/复用与目标用户的单聊，返回会话 ID。
func (c *BotAPIClient) CreateSingleConversation(recipientID uint64) (uint64, error) {
	if c.userToken == "" {
		return 0, fmt.Errorf("创建单聊需要 userToken")
	}
	body, _ := json.Marshal(map[string]any{"type": "single", "user_id": recipientID})
	respBody, err := c.userPost(c.serverURL+"/api/v1/conversations", body)
	if err != nil {
		return 0, err
	}
	var resp struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return 0, fmt.Errorf("解析创建单聊响应失败: %w", err)
	}
	if resp.Data.ID == 0 {
		return 0, fmt.Errorf("创建单聊失败: %s", truncate(string(respBody), 300))
	}
	return resp.Data.ID, nil
}

// --- Task API（需 userToken）---

// ListTasks 列出用户待办。
func (c *BotAPIClient) ListTasks(status string, limit int) ([]Task, error) {
	if c.userToken == "" {
		return nil, fmt.Errorf("任务操作需要 userToken")
	}
	body, err := c.userGet(c.serverURL + "/api/v1/tasks")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data []Task `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析任务列表失败: %w", err)
	}
	tasks := resp.Data
	if status != "" {
		filtered := make([]Task, 0)
		for _, t := range tasks {
			if t.Status == status {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}
	if len(tasks) > limit {
		tasks = tasks[:limit]
	}
	return tasks, nil
}

// CreateTask 创建待办，返回新任务 ID。
func (c *BotAPIClient) CreateTask(title, dueDate, priority, description string) (uint64, error) {
	if c.userToken == "" {
		return 0, fmt.Errorf("任务操作需要 userToken")
	}
	payload := map[string]any{"title": title, "priority": priority}
	if dueDate != "" {
		payload["due_date"] = dueDate
	}
	if description != "" {
		payload["description"] = description
	}
	body, _ := json.Marshal(payload)
	respBody, err := c.userPost(c.serverURL+"/api/v1/tasks", body)
	if err != nil {
		return 0, err
	}
	var resp struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return 0, fmt.Errorf("解析创建任务响应失败: %w", err)
	}
	return resp.Data.ID, nil
}

// UpdateTask 更新待办的字段。
func (c *BotAPIClient) UpdateTask(id uint64, fields map[string]any) error {
	if c.userToken == "" {
		return fmt.Errorf("任务操作需要 userToken")
	}
	if len(fields) == 0 {
		return fmt.Errorf("至少指定一个要修改的字段")
	}
	body, _ := json.Marshal(fields)
	u := fmt.Sprintf("%s/api/v1/tasks/%d", c.serverURL, id)
	_, err := c.userPut(u, body)
	return err
}

// --- Event API（需 userToken）---

// ListEvents 列出用户日历事件。
func (c *BotAPIClient) ListEvents(limit int) ([]Event, error) {
	if c.userToken == "" {
		return nil, fmt.Errorf("日历操作需要 userToken")
	}
	body, err := c.userGet(c.serverURL + "/api/v1/events")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data []Event `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析事件列表失败: %w", err)
	}
	events := resp.Data
	if len(events) > limit {
		events = events[:limit]
	}
	return events, nil
}

// GetEvent 获取单条日历事件（供 update_event 先读后合并）。
func (c *BotAPIClient) GetEvent(id uint64) (*Event, error) {
	if c.userToken == "" {
		return nil, fmt.Errorf("日历操作需要 userToken")
	}
	body, err := c.userGet(fmt.Sprintf("%s/api/v1/events/%d", c.serverURL, id))
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data Event `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析事件失败: %w", err)
	}
	return &resp.Data, nil
}

// CreateEvent 创建日历事件，返回新事件 ID。start/end 为 RFC3339 格式。
func (c *BotAPIClient) CreateEvent(title, start, end string, reminder int, description string) (uint64, error) {
	if c.userToken == "" {
		return 0, fmt.Errorf("日历操作需要 userToken")
	}
	payload := map[string]any{
		"title":    title,
		"start":    start,
		"end":      end,
		"reminder": reminder,
	}
	if description != "" {
		payload["description"] = description
	}
	body, _ := json.Marshal(payload)
	respBody, err := c.userPost(c.serverURL+"/api/v1/events", body)
	if err != nil {
		return 0, err
	}
	var resp struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return 0, fmt.Errorf("解析创建事件响应失败: %w", err)
	}
	return resp.Data.ID, nil
}

// UpdateEvent 更新日历事件的字段。
func (c *BotAPIClient) UpdateEvent(id uint64, fields map[string]any) error {
	if c.userToken == "" {
		return fmt.Errorf("日历操作需要 userToken")
	}
	if len(fields) == 0 {
		return fmt.Errorf("至少指定一个要修改的字段")
	}
	body, _ := json.Marshal(fields)
	u := fmt.Sprintf("%s/api/v1/events/%d", c.serverURL, id)
	_, err := c.userPut(u, body)
	return err
}

// --- Note API（需 userToken）---

// ListNotes 列出当前用户的正式笔记（type=note，排除便签），与 CLI note list 一致。
func (c *BotAPIClient) ListNotes(limit int) ([]Note, error) {
	if c.userToken == "" {
		return nil, fmt.Errorf("笔记操作需要 userToken")
	}
	body, err := c.userGet(c.serverURL + "/api/v1/notes")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data []Note `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析笔记列表失败: %w", err)
	}
	notes := make([]Note, 0, len(resp.Data))
	for _, n := range resp.Data {
		if n.Type != "" && n.Type != "note" {
			continue
		}
		notes = append(notes, n)
	}
	if len(notes) > limit {
		notes = notes[:limit]
	}
	return notes, nil
}

// GetNote 获取单条笔记。
func (c *BotAPIClient) GetNote(noteID uint64) (*Note, error) {
	if c.userToken == "" {
		return nil, fmt.Errorf("笔记操作需要 userToken")
	}
	u := fmt.Sprintf("%s/api/v1/notes/%d", c.serverURL, noteID)
	body, err := c.userGet(u)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data Note `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析笔记失败: %w", err)
	}
	return &resp.Data, nil
}

// CreateNote 创建一条笔记，返回新笔记 ID。
func (c *BotAPIClient) CreateNote(title, content string) (uint64, error) {
	if c.userToken == "" {
		return 0, fmt.Errorf("笔记操作需要 userToken")
	}
	payload := map[string]any{"title": title, "content": content}
	body, _ := json.Marshal(payload)
	respBody, err := c.userPost(c.serverURL+"/api/v1/notes", body)
	if err != nil {
		return 0, err
	}
	var resp struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return 0, fmt.Errorf("解析创建笔记响应失败: %w", err)
	}
	if resp.Data.ID == 0 {
		return 0, fmt.Errorf("响应缺少笔记 id: %s", truncate(string(respBody), 300))
	}
	return resp.Data.ID, nil
}

// UpdateNote 更新笔记的标题/正文。
func (c *BotAPIClient) UpdateNote(noteID uint64, title, content string) error {
	if c.userToken == "" {
		return fmt.Errorf("笔记操作需要 userToken")
	}
	payload := map[string]any{
		"title":   title,
		"content": content,
	}
	body, _ := json.Marshal(payload)
	u := fmt.Sprintf("%s/api/v1/notes/%d", c.serverURL, noteID)
	_, err := c.userPut(u, body)
	return err
}

// SearchNotesSemantic 按语义检索当前用户笔记（返回最相关的若干条命中）。
func (c *BotAPIClient) SearchNotesSemantic(query string, topK int) ([]NoteSearchResult, error) {
	if c.userToken == "" {
		return nil, fmt.Errorf("笔记操作需要 userToken")
	}
	if topK <= 0 {
		topK = 5
	}
	payload := map[string]any{"query": query, "top_k": topK}
	body, _ := json.Marshal(payload)
	respBody, err := c.userPost(c.serverURL+"/api/v1/notes/search", body)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data struct {
			Results []NoteSearchResult `json:"results"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("解析笔记检索结果失败: %w", err)
	}
	return resp.Data.Results, nil
}

// --- 底层 HTTP 方法 ---

// Bot Token 认证方法（用于 /api/v1/bot/* 路由）

func (c *BotAPIClient) get(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.setBotAuth(req)
	return c.do(req)
}

func (c *BotAPIClient) post(url string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	c.setBotAuth(req)
	return c.do(req)
}

func (c *BotAPIClient) put(url string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	c.setBotAuth(req)
	return c.do(req)
}

// User Token 认证方法（用于 /api/v1/tasks/*、/api/v1/events/* 等）

func (c *BotAPIClient) userGet(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.setUserAuth(req)
	return c.do(req)
}

func (c *BotAPIClient) userPost(url string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	c.setUserAuth(req)
	return c.do(req)
}

func (c *BotAPIClient) userPut(url string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	c.setUserAuth(req)
	return c.do(req)
}

func (c *BotAPIClient) setBotAuth(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

func (c *BotAPIClient) setUserAuth(req *http.Request) {
	if c.userToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.userToken)
	}
}

func (c *BotAPIClient) do(req *http.Request) ([]byte, error) {
	// 自报 CLI/MCP 身份：服务端客户端版本门槛（message_version_gate）按此豁免——
	// qim login 的 user_token 与桌面客户端 JWT 同型，无法按令牌区分。
	req.Header.Set("X-Client-App", "cli")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	return body, nil
}

// setTargetUser 按“数字 ID / 名称”写入 to_user_id 或 to_user_name。
func setTargetUser(payload map[string]any, to string) {
	if id, ok := parseNumeric(to); ok {
		payload["to_user_id"] = id
		return
	}
	payload["to_user_name"] = to
}

// parseSendResponse 解析 bot/用户发送接口的 message_id + conversation_id。
func parseSendResponse(body []byte) (uint64, uint64, error) {
	var resp struct {
		Data struct {
			MessageID      uint64 `json:"message_id"`
			ConversationID uint64 `json:"conversation_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, 0, fmt.Errorf("解析响应失败: %w", err)
	}
	if resp.Data.MessageID == 0 {
		return 0, 0, fmt.Errorf("响应缺少 message_id: %s", truncate(string(body), 300))
	}
	return resp.Data.MessageID, resp.Data.ConversationID, nil
}

// LocalToRFC3339 把 CLI 支持的本地时间写法转成 RFC3339（带本地时区偏移）。
// 支持 "2006-01-02 15:04"、":15:04:05"、"2006-01-02T15:04" 与 RFC3339。
func LocalToRFC3339(s string) (string, error) {
	layouts := []string{"2006-01-02 15:04", "2006-01-02 15:04:05", "2006-01-02T15:04", time.RFC3339}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t.Format(time.RFC3339), nil
		}
	}
	return "", fmt.Errorf("无法解析 %q（示例: 2026-08-01 14:00 或 RFC3339）", s)
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

// clamp 与服务端/CLI 保持一致：<=0 或 >100 一律回落 50（服务端对 >100 也是重置为 50）。
func clamp(n int) int {
	if n <= 0 || n > 100 {
		return 50
	}
	return n
}

// clampSearch 搜索结果默认 20；上限交给服务端处理（>100 会被服务端重置为默认值）。
func clampSearch(n int) int {
	if n <= 0 {
		return 20
	}
	return n
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
