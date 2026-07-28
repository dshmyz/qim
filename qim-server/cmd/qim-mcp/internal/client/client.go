// Package client 是 qim-mcp 与 QIM Bot API 之间的纯 HTTP 客户端。
// 与 cmd/qim CLI 同源：只依赖 REST 契约（GET/POST /api/v1/bot/... + Bearer token），
// 不耦合 server 内部（service/handler），便于随 agent 侧独立分发。
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Message 对齐 Bot API `GET /bot/messages` 返回的消息形态（与 cmd/qim CLI 一致）。
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

// BotAPIClient 封装对 QIM Bot API 的调用，每个请求带 Bot 令牌。
type BotAPIClient struct {
	serverURL string
	token     string
	http      *http.Client
}

// New 创建客户端。serverURL 形如 http://localhost:8080，token 为 qbot_ 开头的明文令牌。
func New(serverURL, token string) *BotAPIClient {
	return &BotAPIClient{
		serverURL: serverURL,
		token:     token,
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

// SendMessage 发送一条消息，返回新消息 ID 与会话 ID。msgType: text|markdown|card|streaming。
// toUserID 为目标人类用户；threadID>0 复用既有会话，=0 由服务端自动建/复用会话。
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

func (c *BotAPIClient) get(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)
	return c.do(req)
}

func (c *BotAPIClient) post(url string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	c.setAuth(req)
	return c.do(req)
}

func (c *BotAPIClient) setAuth(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
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
