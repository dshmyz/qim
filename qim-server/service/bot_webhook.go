package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// BotConfig 存储于 model.Bot.Config（JSON 文本列），控制 bot 的回复路由。
// 无需迁移：Config 已是 text 列，未识别字段忽略。
type BotConfig struct {
	Mode          string `json:"mode"`            // "internal_ai"(默认, 既有行为) | "external_webhook"
	WebhookURL    string `json:"webhook_url"`     // external_webhook 模式下的回调地址
	WebhookSecret string `json:"webhook_secret"`  // HMAC-SHA256 签名密钥，与 bot 访问令牌分离
}

// ParseBotConfig 解析 bot.Config JSON，失败或为空返回零值（Mode="" 视为 internal_ai）。
func ParseBotConfig(configJSON string) BotConfig {
	var cfg BotConfig
	if configJSON != "" {
		_ = json.Unmarshal([]byte(configJSON), &cfg)
	}
	return cfg
}

// IsExternalWebhook 是否走外部 agent webhook 路由。
func (c BotConfig) IsExternalWebhook() bool {
	return c.Mode == "external_webhook" && c.WebhookURL != ""
}

// BotWebhookPayload 外部 agent webhook 回调载荷。
// thread_id 即 QIM conversation_id，agent 据此回复同一会话。
// event 取值：
//   - "bot.message"：用户在 bot 会话中的文本/多媒体回复（默认）。
//   - "bot.card_action"：用户点击了 agent 先前发出的卡片按钮，ActionID/ActionValue 携带所选项。
type BotWebhookPayload struct {
	Event        string `json:"event"`
	BotID        uint   `json:"bot_id"`
	ThreadID     uint   `json:"thread_id"`
	MessageID    uint   `json:"message_id"`
	UserID       uint   `json:"user_id"`
	UserNickname string `json:"user_nickname"`
	UserAvatar   string `json:"user_avatar"`
	Content      string `json:"content"`
	MsgType      string `json:"msg_type"`
	Timestamp    string `json:"timestamp"`
	DeliveryID   string `json:"delivery_id"`
	// ActionID/ActionValue 仅 event=="bot.card_action" 时有意义：
	// 指明用户在 MessageID 这张卡片上点击的按钮 id 及其 value。
	ActionID    string `json:"action_id,omitempty"`
	ActionValue string `json:"action_value,omitempty"`
}

// SendBotWebhook 将用户回复转发到外部 agent 的 webhook。
// 复用 webhook_sender.go 的 sharedHTTPClient 连接池与 HMAC-SHA256 签名风格，
// 但不改动 SendRemind（scope 纪律）。webhookSecret 为空时不签名。
// 返回 error 表示失败，调用方据 delivery_id 记录审计。
func SendBotWebhook(webhookURL, webhookSecret string, payload BotWebhookPayload) error {
	if webhookURL == "" {
		return fmt.Errorf("webhook_url 未配置")
	}

	payload.Timestamp = time.Now().UTC().Format(time.RFC3339)
	payload.DeliveryID = uuid.New().String()
	if payload.Event == "" {
		payload.Event = "bot.message"
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化 webhook 载荷失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("构造 webhook 请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("X-QIM-Event", payload.Event)
	req.Header.Set("X-QIM-Timestamp", payload.Timestamp)
	req.Header.Set("X-QIM-Delivery", payload.DeliveryID)

	if webhookSecret != "" {
		mac := hmac.New(sha256.New, []byte(webhookSecret))
		mac.Write(bodyBytes)
		req.Header.Set("X-QIM-Signature", hex.EncodeToString(mac.Sum(nil)))
	}

	resp, err := sharedHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("webhook 调用失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("webhook 返回错误: HTTP %d, body: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
