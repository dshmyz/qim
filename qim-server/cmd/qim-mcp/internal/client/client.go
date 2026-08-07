// Package client 是 qim-mcp 与 QIM Bot API 之间的纯 HTTP 客户端。
// 与 cmd/qim CLI 同源：只依赖 REST 契约（GET/POST/PUT /api/v1/... + Bearer token），
// 不耦合 server 内部（service/handler），便于随 agent 侧独立分发。
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	ID       uint64 `json:"id"`
	Title    string `json:"title"`
	Start    string `json:"start"`
	End      string `json:"end"`
	Reminder int    `json:"reminder"`
}

// BotAPIClient 封装对 QIM Bot API 的调用，支持 Bot Token 和 User JWT 双认证。
type BotAPIClient struct {
	serverURL string
	token     string // Bot 令牌 (qbot_xxx)
	userToken string // 用户 JWT（用于任务/日历等需要用户身份的接口）
	http      *http.Client
}

// New 创建客户端。serverURL 形如 http://localhost:8080，token 为 qbot_ 开头的明文令牌。
// userToken 为可选的用户 JWT，用于调用需要用户身份的接口（任务、日历等）。
func New(serverURL, token, userToken string) *BotAPIClient {
	return &BotAPIClient{
		serverURL: serverURL,
		token:     token,
		userToken: userToken,
		http:      &http.Client{Timeout: 30 * time.Second},
	}
}

// ListMessages 拉会话消息。afterID=0 拉最近，>0 增量拉 afterID 之后的消息。
func (c *BotAPIClient) ListMessages(threadID, afterID uint64, limit int) ([]Message, error) {
	u := fmt.Sprintf("%s/api/v1/bot/messages?thread_id=%d&limit=%d", c.serverURL, threadID, clamp(limit))
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

// BotGroup 是 bot 已入群的群会话摘要。
type BotGroup struct {
	ConversationID uint64 `json:"conversation_id"`
	GroupName      string `json:"group_name"`
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

// SendMessage 发送一条消息，返回新消息 ID 与会话 ID。msgType: text|markdown|card|streaming。
func (c *BotAPIClient) SendMessage(toUserID, threadID uint64, content, msgType string) (msgID, convID uint64, err error) {
	payload := map[string]any{
		"to_user_id": toUserID,
		"content":    content,
		"msg_type":   msgType,
	}
	if threadID > 0 {
		payload["thread_id"] = threadID
	}
	body, _ := json.Marshal(payload)
	respBody, err := c.post(c.serverURL+"/api/v1/bot/messages", body)
	if err != nil {
		return 0, 0, err
	}
	var resp struct {
		Data struct {
			MessageID      uint64 `json:"message_id"`
			ConversationID uint64 `json:"conversation_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return 0, 0, fmt.Errorf("解析响应失败: %w", err)
	}
	if resp.Data.MessageID == 0 {
		return 0, 0, fmt.Errorf("响应缺少 message_id: %s", truncate(string(respBody), 300))
	}
	return resp.Data.MessageID, resp.Data.ConversationID, nil
}

// SendMessageToConversation 按会话(单聊/群聊)发送一条消息。返回新消息 ID 与会话 ID。
// conversationID 非 0 时走群聊/已建单聊会话，threadID 供单聊 thread 场景用。
func (c *BotAPIClient) SendMessageToConversation(conversationID uint64, content, msgType string) (msgID, convID uint64, err error) {
	payload := map[string]any{
		"conversation_id": conversationID,
		"content":         content,
		"msg_type":        msgType,
	}
	body, _ := json.Marshal(payload)
	respBody, err := c.post(c.serverURL+"/api/v1/bot/messages", body)
	if err != nil {
		return 0, 0, err
	}
	var resp struct {
		Data struct {
			MessageID      uint64 `json:"message_id"`
			ConversationID uint64 `json:"conversation_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return 0, 0, fmt.Errorf("解析响应失败: %w", err)
	}
	if resp.Data.MessageID == 0 {
		return 0, 0, fmt.Errorf("响应缺少 message_id: %s", truncate(string(respBody), 300))
	}
	return resp.Data.MessageID, resp.Data.ConversationID, nil
}

// StreamChunk 向流式消息追加一段 delta，或 finish=true 收尾。
func (c *BotAPIClient) StreamChunk(msgID uint64, delta string, finish bool) error {
	body, _ := json.Marshal(map[string]any{
		"content_delta": delta,
		"finish":        finish,
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
func (c *BotAPIClient) SearchMessages(keyword string, convID uint64, limit int) ([]SearchResultItem, error) {
	if c.userToken == "" {
		return nil, fmt.Errorf("搜索消息需要 userToken")
	}
	u := fmt.Sprintf("%s/api/v1/messages/search?keyword=%s&pageSize=%d",
		c.serverURL, url.QueryEscape(keyword), clamp(limit))
	if convID > 0 {
		u += fmt.Sprintf("&conv_id=%d", convID)
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

// User JWT 认证方法（用于 /api/v1/tasks/*、/api/v1/events/* 等）

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

func clamp(n int) int {
	if n <= 0 {
		return 50
	}
	if n > 200 {
		return 200
	}
	return n
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
