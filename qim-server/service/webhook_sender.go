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
	"sync"
	"text/template"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// sharedHTTPClient 复用 TCP 连接池，避免每次提醒调用都新建 http.Client
var sharedHTTPClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}

// templateCache 缓存已解析的 body_template，key 为模板字符串，value 为 *template.Template
// 相同模板只解析一次，后续调用直接复用
var templateCache sync.Map

// getCompiledTemplate 解析并缓存 body 模板；并发场景下相同模板可能被解析多次，但只会存储一份
func getCompiledTemplate(bodyTemplate string) (*template.Template, error) {
	if v, ok := templateCache.Load(bodyTemplate); ok {
		return v.(*template.Template), nil
	}
	tmpl, err := template.New("body").Parse(bodyTemplate)
	if err != nil {
		return nil, err
	}
	actual, _ := templateCache.LoadOrStore(bodyTemplate, tmpl)
	return actual.(*template.Template), nil
}

// WebhookConfig 消息提醒 webhook 配置
// 由管理员在后台系统配置页面填写，存储在 SystemConfig 表中（key=message_remind_webhook）
type WebhookConfig struct {
	Enabled        bool              `json:"enabled"`         // 启用开关
	SystemName     string            `json:"system_name"`     // 管理员配置的目标系统展示名称
	URL            string            `json:"url"`             // 外部系统接收地址
	Method         string            `json:"method"`          // HTTP 方法，默认 POST
	Secret         string            `json:"secret"`          // HMAC-SHA256 签名密钥，空则不签名
	TimeoutSeconds int               `json:"timeout_seconds"` // 超时秒数（1-30），默认 10
	Headers        map[string]string `json:"headers"`         // 附加自定义请求头
	BodyTemplate   string            `json:"body_template"`   // body 模板，Go template 语法
}

// RemindData 提醒数据，用于渲染 body_template 和构造 payload
type RemindData struct {
	MessageID             uint   // 消息 ID
	ConversationID        uint   // 会话 ID
	ConversationType      string // 会话类型（single/group）
	SenderID              uint   // 发送者 ID
	SenderUsername        string // 发送者用户名
	SenderNickname        string // 发送者昵称
	SenderEmail           string // 发送者邮箱
	RecipientID           uint   // 接收者 ID
	RecipientUsername     string // 接收者用户名
	RecipientNickname     string // 接收者昵称
	RecipientEmail        string // 接收者邮箱
	MessageContentPreview string // 消息内容前 100 字符
	MessageType           string // 消息类型
	MessageSentAt         string // 消息发送时间（RFC3339）
	MessageURL            string // 可点击跳转链接
}

// LoadWebhookConfig 从 SystemConfig 读取 webhook 配置
// 配置不存在或解析失败时返回 error，调用方应据此返回"提醒功能未配置"
func LoadWebhookConfig(db *gorm.DB) (*WebhookConfig, error) {
	var cfg model.SystemConfig
	if err := db.Where("config_key = ?", "message_remind_webhook").First(&cfg).Error; err != nil {
		return nil, err
	}
	var wc WebhookConfig
	if err := json.Unmarshal([]byte(cfg.Value), &wc); err != nil {
		return nil, fmt.Errorf("解析 webhook 配置失败: %w", err)
	}
	if wc.Method == "" {
		wc.Method = "POST"
	}
	if wc.TimeoutSeconds <= 0 {
		wc.TimeoutSeconds = 10
	}
	if wc.TimeoutSeconds > 30 {
		wc.TimeoutSeconds = 30
	}
	return &wc, nil
}

// SendRemind 发送提醒到外部系统
// 流程：渲染 body_template → 构造 HTTP 请求 → 设置签名/事件头 → 调用 → 校验响应状态码
// 返回 error 表示失败，调用方据此通过 WebSocket 推送失败回执给发送方
func SendRemind(cfg *WebhookConfig, data RemindData) error {
	deliveryID := uuid.New().String()
	timestamp := time.Now().UTC().Format(time.RFC3339)

	// 渲染 body 模板（已解析的模板会被缓存，避免每次调用重复 Parse）
	tmpl, err := getCompiledTemplate(cfg.BodyTemplate)
	if err != nil {
		return fmt.Errorf("body_template 解析失败: %w", err)
	}

	templateData := map[string]interface{}{
		"Event":                 "message.remind",
		"DeliveryID":            deliveryID,
		"Timestamp":             timestamp,
		"SenderID":              data.SenderID,
		"SenderUsername":        data.SenderUsername,
		"SenderNickname":        data.SenderNickname,
		"SenderEmail":           data.SenderEmail,
		"RecipientID":           data.RecipientID,
		"RecipientUsername":     data.RecipientUsername,
		"RecipientNickname":     data.RecipientNickname,
		"RecipientEmail":        data.RecipientEmail,
		"MessageID":             data.MessageID,
		"MessageContentPreview": truncateString(data.MessageContentPreview, 100),
		"MessageType":           data.MessageType,
		"MessageSentAt":         data.MessageSentAt,
		"MessageURL":            data.MessageURL,
		"ConversationID":        data.ConversationID,
		"ConversationType":      data.ConversationType,
	}

	var bodyBuf bytes.Buffer
	if err := tmpl.Execute(&bodyBuf, templateData); err != nil {
		return fmt.Errorf("body_template 渲染失败: %w", err)
	}
	bodyBytes := bodyBuf.Bytes()

	// 构造 HTTP 请求，使用 context 控制单次请求超时（复用共享 client 的连接池）
	// 兜底处理直接构造的 cfg（如测试或未走 LoadWebhookConfig 归一化的场景），避免 timeout 为 0 导致 context 立即过期
	timeoutSeconds := cfg.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 10
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, cfg.Method, cfg.URL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("构造请求失败: %w", err)
	}

	// 设置标准 headers
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("X-QIM-Event", "message.remind")
	req.Header.Set("X-QIM-Timestamp", timestamp)
	req.Header.Set("X-QIM-Delivery", deliveryID)
	// 设置自定义 headers（可能覆盖标准 headers，由管理员负责）
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	// HMAC-SHA256 签名（secret 非空时才发送）
	if cfg.Secret != "" {
		mac := hmac.New(sha256.New, []byte(cfg.Secret))
		mac.Write(bodyBytes)
		req.Header.Set("X-QIM-Signature", hex.EncodeToString(mac.Sum(nil)))
	}

	// 调用（复用共享 client 的连接池）
	resp, err := sharedHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("webhook 调用失败: %w", err)
	}
	defer resp.Body.Close()

	// 校验响应状态码，2xx 为成功，其他视为失败
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("webhook 返回错误: HTTP %d, body: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// truncateString 截断字符串到指定 rune 长度，超出部分用 "..." 替代
func truncateString(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
