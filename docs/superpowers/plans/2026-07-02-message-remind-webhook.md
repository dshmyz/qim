# 消息提醒 Webhook 实施计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 补全右键消息"发送提醒"功能，通过 webhook 调用外部系统（企业微信/飞书/Slack 等机器人）实现真正的提醒触达，含 HMAC 签名、失败回执、防滥用。

**架构：** 复用 SystemConfig 表存储 webhook 配置；后端收到提醒请求后异步调用外部 webhook，通过 WebSocket 推送结果回执给发送方；前端只做菜单触发和结果展示。

**技术栈：** Go + Gin + GORM + text/template + WebSocket + Vue 3 + Element Plus

---

## 阶段总览

| 阶段 | 任务 | 优先级 |
|------|------|--------|
| 一 | T1 webhook 配置存储与 seed | P0 |
| 一 | T2 webhook sender 服务实现 | P0 |
| 一 | T3 RemindMessage handler 改造 | P0 |
| 二 | T4 前端监听 remind_result 事件 | P1 |
| 二 | T5 管理后台 webhook 配置 UI | P1 |

---

## 阶段一：P0 后端核心实现

### 任务 T1：webhook 配置存储与 seed

**问题：** 需要一个地方存储 webhook 配置（URL/method/secret/headers/body_template 等）。

**方案：** 复用 SystemConfig 表，新增一条 key=`message_remind_webhook`、type=`json` 的记录。不新建表。

**配置结构：**
```json
{
  "enabled": false,
  "url": "",
  "method": "POST",
  "secret": "",
  "timeout_seconds": 10,
  "headers": {"Content-Type": "application/json"},
  "body_template": ""
}
```

| 字段 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `enabled` | 是 | false | 启用开关 |
| `url` | 是 | - | 外部系统接收地址 |
| `method` | 否 | POST | HTTP 方法 |
| `secret` | 否 | - | HMAC 签名密钥，空则不签名 |
| `timeout_seconds` | 否 | 10 | 超时秒数（1-30） |
| `headers` | 否 | {} | 附加自定义请求头 |
| `body_template` | 是 | - | body 模板，支持 Go template 语法 |

**文件：**
- 修改：`qim-server/app/init.go`（新增 seedMessageRemindWebhook 函数）

**关键代码：**

在 `seedFileUploadConfig` 旁新增：
```go
// seedMessageRemindWebhook 初始化消息提醒 webhook 配置
func seedMessageRemindWebhook(db *gorm.DB) {
    defaultCfg := model.SystemConfig{
        ConfigKey: "message_remind_webhook",
        Value:     `{"enabled":false,"url":"","method":"POST","secret":"","timeout_seconds":10,"headers":{"Content-Type":"application/json"},"body_template":""}`,
        Type:      "json",
        Desc:      "消息提醒 webhook 配置（管理员配置）",
    }
    db.Where("config_key = ?", defaultCfg.ConfigKey).FirstOrCreate(&defaultCfg)
    logger.WithModule("Migrate").Info("消息提醒 webhook 配置初始化完成")
}
```

在 init 流程中调用（参考 `seedFileUploadConfig` 的调用位置，确保在 SystemConfig 表 AutoMigrate 之后）。

- [ ] **步骤 1：在 app/init.go 中新增 seedMessageRemindWebhook 函数**
- [ ] **步骤 2：在 init 流程中调用该函数**
- [ ] **步骤 3：启动服务验证 SystemConfig 表生成默认记录**
- [ ] **步骤 4：Commit**

```bash
git add qim-server/app/init.go
git commit -m "feat(remind): 初始化消息提醒 webhook 默认配置"
```

---

### 任务 T2：webhook sender 服务实现

**问题：** 需要一个独立的 service 负责 webhook 调用，含配置加载、payload 构造、模板渲染、HMAC 签名、HTTP 调用。

**文件：**
- 创建：`qim-server/service/webhook_sender.go`
- 创建：`qim-server/service/webhook_sender_test.go`

**关键代码：**

`webhook_sender.go`：
```go
package service

import (
    "bytes"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
    "text/template"
    "time"

    "github.com/dshmyz/qim/qim-server/model"
    "github.com/google/uuid"
    "gorm.io/gorm"
)

// WebhookConfig 消息提醒 webhook 配置
type WebhookConfig struct {
    Enabled        bool              `json:"enabled"`
    URL            string            `json:"url"`
    Method         string            `json:"method"`
    Secret         string            `json:"secret"`
    TimeoutSeconds int               `json:"timeout_seconds"`
    Headers        map[string]string `json:"headers"`
    BodyTemplate   string            `json:"body_template"`
}

// RemindData 提醒数据
type RemindData struct {
    MessageID             uint
    ConversationID        uint
    ConversationType      string
    SenderID              uint
    SenderUsername        string
    SenderNickname        string
    SenderEmail           string
    RecipientID           uint
    RecipientUsername     string
    RecipientNickname     string
    RecipientEmail        string
    MessageContentPreview string
    MessageType           string
    MessageSentAt         string
    MessageURL            string
}

// LoadWebhookConfig 从 SystemConfig 读取 webhook 配置
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

// SendRemind 发送提醒到外部系统，返回 error 表示失败
func SendRemind(cfg *WebhookConfig, data RemindData) error {
    deliveryID := uuid.New().String()
    timestamp := time.Now().UTC().Format(time.RFC3339)

    // 渲染 body 模板
    tmpl, err := template.New("body").Parse(cfg.BodyTemplate)
    if err != nil {
        return fmt.Errorf("body_template 解析失败: %w", err)
    }

    templateData := map[string]interface{}{
        "Event":                  "message.remind",
        "DeliveryID":             deliveryID,
        "Timestamp":              timestamp,
        "SenderID":               data.SenderID,
        "SenderUsername":         data.SenderUsername,
        "SenderNickname":         data.SenderNickname,
        "SenderEmail":            data.SenderEmail,
        "RecipientID":            data.RecipientID,
        "RecipientUsername":      data.RecipientUsername,
        "RecipientNickname":      data.RecipientNickname,
        "RecipientEmail":         data.RecipientEmail,
        "MessageID":              data.MessageID,
        "MessageContentPreview":  truncateString(data.MessageContentPreview, 100),
        "MessageType":            data.MessageType,
        "MessageSentAt":          data.MessageSentAt,
        "MessageURL":             data.MessageURL,
        "ConversationID":         data.ConversationID,
        "ConversationType":       data.ConversationType,
    }

    var bodyBuf bytes.Buffer
    if err := tmpl.Execute(&bodyBuf, templateData); err != nil {
        return fmt.Errorf("body_template 渲染失败: %w", err)
    }
    bodyBytes := bodyBuf.Bytes()

    // 构造 HTTP 请求
    req, err := http.NewRequest(cfg.Method, cfg.URL, bytes.NewReader(bodyBytes))
    if err != nil {
        return fmt.Errorf("构造请求失败: %w", err)
    }

    // 设置 headers
    req.Header.Set("Content-Type", "application/json; charset=utf-8")
    req.Header.Set("X-QIM-Event", "message.remind")
    req.Header.Set("X-QIM-Timestamp", timestamp)
    req.Header.Set("X-QIM-Delivery", deliveryID)
    for k, v := range cfg.Headers {
        req.Header.Set(k, v)
    }

    // HMAC 签名
    if cfg.Secret != "" {
        mac := hmac.New(sha256.New, []byte(cfg.Secret))
        mac.Write(bodyBytes)
        req.Header.Set("X-QIM-Signature", hex.EncodeToString(mac.Sum(nil)))
    }

    // 调用
    client := &http.Client{Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second}
    resp, err := client.Do(req)
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

func truncateString(s string, n int) string {
    runes := []rune(s)
    if len(runes) <= n {
        return s
    }
    return string(runes[:n]) + "..."
}
```

**单元测试** `webhook_sender_test.go`：
```go
package service

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestSendRemind_Success(t *testing.T) {
    var receivedBody string
    var receivedSignature string
    var receivedEvent string
    var receivedDelivery string
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        buf := make([]byte, 1024)
        n, _ := r.Body.Read(buf)
        receivedBody = string(buf[:n])
        receivedSignature = r.Header.Get("X-QIM-Signature")
        receivedEvent = r.Header.Get("X-QIM-Event")
        receivedDelivery = r.Header.Get("X-QIM-Delivery")
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    cfg := &WebhookConfig{
        Enabled:      true,
        URL:          server.URL,
        Method:       "POST",
        Secret:       "test-secret",
        Headers:      map[string]string{"X-Custom": "test"},
        BodyTemplate: `{"text":"{{.SenderNickname}} 提醒你：{{.MessageContentPreview}}"}`,
    }
    data := RemindData{
        SenderNickname:        "Alice",
        MessageContentPreview: "会议纪要见附件",
    }

    err := SendRemind(cfg, data)
    if err != nil {
        t.Fatalf("expected success, got error: %v", err)
    }
    if receivedEvent != "message.remind" {
        t.Errorf("expected event message.remind, got %s", receivedEvent)
    }
    if receivedDelivery == "" {
        t.Error("expected non-empty delivery ID")
    }
    if receivedSignature == "" {
        t.Error("expected non-empty signature")
    }
    if !strings.Contains(receivedBody, "Alice") {
        t.Errorf("expected body to contain Alice, got %s", receivedBody)
    }
}

func TestSendRemind_HTTPError(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusInternalServerError)
        w.Write([]byte("internal error"))
    }))
    defer server.Close()

    cfg := &WebhookConfig{
        Enabled:      true,
        URL:          server.URL,
        Method:       "POST",
        BodyTemplate: `{"text":"test"}`,
    }
    err := SendRemind(cfg, RemindData{})
    if err == nil {
        t.Fatal("expected error for HTTP 500")
    }
    if !strings.Contains(err.Error(), "HTTP 500") {
        t.Errorf("expected HTTP 500 in error, got %v", err)
    }
}

func TestSendRemind_TemplateError(t *testing.T) {
    cfg := &WebhookConfig{
        Enabled:      true,
        URL:          "http://localhost",
        Method:       "POST",
        BodyTemplate: `{{.InvalidSyntax`,
    }
    err := SendRemind(cfg, RemindData{})
    if err == nil {
        t.Fatal("expected template parse error")
    }
}

func TestTruncateString(t *testing.T) {
    if got := truncateString("hello", 10); got != "hello" {
        t.Errorf("expected hello, got %s", got)
    }
    if got := truncateString("hello world", 5); got != "hello..." {
        t.Errorf("expected hello..., got %s", got)
    }
}
```

- [ ] **步骤 1：编写失败的测试**（创建 webhook_sender_test.go）
- [ ] **步骤 2：运行测试验证失败**

运行：`cd qim-server && go test ./service/ -run TestSendRemind -v`
预期：FAIL

- [ ] **步骤 3：编写实现代码**（创建 webhook_sender.go）
- [ ] **步骤 4：运行测试验证通过**

运行：`cd qim-server && go test ./service/ -run TestSendRemind -v`
预期：PASS

- [ ] **步骤 5：Commit**

```bash
git add qim-server/service/webhook_sender.go qim-server/service/webhook_sender_test.go
git commit -m "feat(remind): 实现 webhook sender 服务，含模板渲染和 HMAC 签名"
```

---

### 任务 T3：RemindMessage handler 改造

**问题：** `qim-server/handler/message_handler.go:795-821` 的 `RemindMessage` 是空壳，校验通过后直接返回"提醒已发送"，没有真正调用 webhook。

**方案：** 保留并扩展校验逻辑（单聊、未读、超 1 小时、非 bot、频率限制），读取 webhook 配置，异步调用 webhook，通过 WebSocket 推送结果回执。

**文件：**
- 修改：`qim-server/handler/message_handler.go`（RemindMessage 重写）

**关键代码：**

在 `message_handler.go` 顶部增加：
```go
import (
    "encoding/json"
    "os"
    "sync"
    "time"
    "github.com/dshmyz/qim/qim-server/service"
    "github.com/dshmyz/qim/qim-server/utils"
)

var remindRateLimiter sync.Map // key: msgID(uint), value: time.Time
```

重写 `RemindMessage`（第 795-821 行）：
```go
func RemindMessage(c *gin.Context) {
    userID, _ := c.Get("user_id")
    msgIDStr := c.Param("id")

    msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
    if err != nil {
        response.BadRequest(c, "无效的消息ID")
        return
    }

    msgSvc := di.GlobalContainer.MessageService
    db := di.GlobalContainer.DB

    msg, err := msgSvc.GetMessageByID(uint(msgID))
    if err != nil {
        response.NotFound(c, "消息不存在")
        return
    }

    // 校验：是发送者本人
    if msg.SenderID != userID.(uint) {
        response.Forbidden(c, "无权限发送提醒")
        return
    }

    // 校验：单聊
    var conv model.Conversation
    if err := db.First(&conv, msg.ConversationID).Error; err != nil {
        response.InternalServerError(c, "会话不存在")
        return
    }
    if conv.Type != "single" {
        response.BadRequest(c, "仅支持单聊消息提醒")
        return
    }

    // 校验：消息未读
    if msg.IsRead {
        response.BadRequest(c, "消息已被读，无需提醒")
        return
    }

    // 校验：超 1 小时
    if time.Since(msg.CreatedAt) < time.Hour {
        response.BadRequest(c, "消息发送未满 1 小时")
        return
    }

    // 校验：非 bot 消息
    if msg.Sender.Type == "bot" {
        response.BadRequest(c, "不支持机器人消息提醒")
        return
    }

    // 频率限制：同一消息每小时最多 1 次
    if last, ok := remindRateLimiter.Load(msg.ID); ok {
        if time.Since(last.(time.Time)) < time.Hour {
            response.BadRequest(c, "该消息已提醒过，请 1 小时后再试")
            return
        }
    }

    // 读取 webhook 配置
    webhookCfg, err := service.LoadWebhookConfig(db)
    if err != nil || !webhookCfg.Enabled {
        response.BadRequest(c, "提醒功能未配置或未启用")
        return
    }

    // 查询接收者（单聊中除发送者外的成员）
    var recipient model.User
    if err := db.Where("id IN (?) AND id != ?",
        db.Model(&model.ConversationMember{}).Select("user_id").Where("conversation_id = ?", msg.ConversationID),
        userID.(uint),
    ).First(&recipient).Error; err != nil {
        response.InternalServerError(c, "接收者不存在")
        return
    }

    // 构造 MessageURL
    baseURL := os.Getenv("QIM_BASE_URL")
    messageURL := ""
    if baseURL != "" {
        messageURL = fmt.Sprintf("%s/chat?conv=%d&msg=%d", baseURL, msg.ConversationID, msg.ID)
    }

    data := service.RemindData{
        MessageID:             msg.ID,
        ConversationID:        msg.ConversationID,
        ConversationType:      conv.Type,
        SenderID:              msg.Sender.ID,
        SenderUsername:        msg.Sender.Username,
        SenderNickname:        msg.Sender.Nickname,
        SenderEmail:           msg.Sender.Email,
        RecipientID:           recipient.ID,
        RecipientUsername:     recipient.Username,
        RecipientNickname:     recipient.Nickname,
        RecipientEmail:        recipient.Email,
        MessageContentPreview: msg.Content,
        MessageType:           msg.Type,
        MessageSentAt:         msg.CreatedAt.Format(time.RFC3339),
        MessageURL:            messageURL,
    }

    // 记录频率限制
    remindRateLimiter.Store(msg.ID, time.Now())

    // 立即返回，异步调用 webhook
    response.Success(c, gin.H{"message": "提醒发送中"})

    // 异步调用 webhook，结果通过 WebSocket 回执
    senderID := userID.(uint)
    utils.SafeGoWithLabel("webhook-remind", func() {
        hub := di.GlobalContainer.Hub
        var result []byte

        if err := service.SendRemind(webhookCfg, data); err != nil {
            logger.WithModule("Remind").Error("webhook 调用失败",
                "message_id", msg.ID, "error", err)
            result, _ = json.Marshal(map[string]interface{}{
                "type": "remind_result",
                "data": map[string]interface{}{
                    "message_id": msg.ID,
                    "success":    false,
                    "error":      err.Error(),
                    "timestamp":  time.Now().Format(time.RFC3339),
                },
            })
        } else {
            result, _ = json.Marshal(map[string]interface{}{
                "type": "remind_result",
                "data": map[string]interface{}{
                    "message_id": msg.ID,
                    "success":    true,
                    "timestamp":  time.Now().Format(time.RFC3339),
                },
            })
        }
        hub.SendToUser(senderID, result)
    })
}
```

**验证点：**
- 保留前端现有 `canSendReminder` 判断（message 是自己的 + 未读 + 超 1 小时 + 非 bot），后端做相同校验作为防御
- 单聊校验后端新增（前端 ChatWindow.vue:1692 已有，后端兜底）
- 频率限制内存存储，重启清零（可接受）

- [ ] **步骤 1：改造 RemindMessage handler**
- [ ] **步骤 2：验证编译通过**

运行：`cd qim-server && go build ./...`

- [ ] **步骤 3：手动测试（未配置 webhook 场景）**

配置 enabled=false，调用 API，预期返回"提醒功能未配置或未启用"

- [ ] **步骤 4：手动测试（配置 webhook 场景）**

配置企微机器人 URL，调用 API，预期：
- 立即返回"提醒发送中"
- 企微群收到消息
- WebSocket 收到 remind_result success=true

- [ ] **步骤 5：手动测试（失败回执）**

配置错误 URL，调用 API，预期：
- 立即返回"提醒发送中"
- WebSocket 收到 remind_result success=false + error 信息

- [ ] **步骤 6：手动测试（频率限制）**

同一消息 1 小时内调用两次，预期第二次返回"该消息已提醒过，请 1 小时后再试"

- [ ] **步骤 7：Commit**

```bash
git add qim-server/handler/message_handler.go
git commit -m "feat(remind): RemindMessage 真正实现，异步调用 webhook 并回执结果"
```

---

## 阶段二：P1 前端实现

### 任务 T4：前端监听 remind_result 事件

**问题：** 前端收到"提醒已发送"后，无法感知 webhook 调用是否真正成功，需要监听 WebSocket 推送的 remind_result 事件。

**文件：**
- 修改：`qim-client/src/views/Main.vue`（WebSocket onmessage 分支增加 remind_result）
- 修改：`qim-client/src/components/chat/ChatWindow.vue`（sendMessageReminder 改为"提醒发送中"提示）

**关键代码：**

`Main.vue` WebSocket onmessage 处理（参考现有 `new_message` 分支位置）：
```typescript
// 在 WebSocket onmessage 的 switch/if 分支中增加
if (data.type === 'remind_result') {
  const result = data.data
  if (result.success) {
    $message.success('提醒已送达外部系统')
  } else {
    $message.error(`提醒发送失败：${result.error || '未知错误'}`)
  }
  return
}
```

`ChatWindow.vue` sendMessageReminder（第 1705-1729 行）调整提示文案：
```typescript
const sendMessageReminder = async () => {
  if (!selectedMessage.value) {
    closeMessageContextMenu()
    return
  }

  const message = selectedMessage.value

  try {
    const response = await request(`/api/v1/messages/${message.id}/remind`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      }
    })

    if (response.code === 0) {
      $message.info('提醒发送中，结果稍后通知')  // 改为 info，真正结果通过 WebSocket 推送
    } else {
      $message.error('发送提醒失败: ' + response.message)
    }
  } catch (error) {
    logger.error('发送提醒失败:', error)
    $message.error('发送提醒失败: ' + error.message)
  }
}
```

- [ ] **步骤 1：在 Main.vue WebSocket onmessage 中增加 remind_result 分支**
- [ ] **步骤 2：调整 ChatWindow.vue sendMessageReminder 提示文案**
- [ ] **步骤 3：手动测试提醒成功和失败两种场景的提示**
- [ ] **步骤 4：Commit**

```bash
git add qim-client/src/views/Main.vue qim-client/src/components/chat/ChatWindow.vue
git commit -m "feat(remind): 前端监听 remind_result 事件，区分发送中和最终结果"
```

---

### 任务 T5：管理后台 webhook 配置 UI

**问题：** 管理员需要一个 UI 配置 webhook（URL/method/secret/headers/body_template 等）。

**方案：** 在 qim-admin 系统配置页面增加 webhook 配置区，含启用开关、各字段输入、常用系统模板预设按钮、测试按钮。

**文件：**
- 修改：`qim-admin/src/views/SystemManagement/SystemConfig.vue`（或对应系统配置页面）
- 修改：`qim-admin/src/stores/systemConfig.ts`（加 webhook 配置的读写）

**UI 设计：**

```
┌─ 消息提醒 Webhook ─────────────────────────────────┐
│                                                      │
│  启用：[ ] 开启                                       │
│                                                      │
│  请求地址：[__________________________________]      │
│  请求方法：[POST ▼]                                   │
│  超时秒数：[10    ]                                   │
│  签名密钥：[**********] （HMAC-SHA256，可选）         │
│                                                      │
│  请求头：                                            │
│  [Content-Type    ] [application/json    ] [删除]   │
│  [+ 添加请求头]                                      │
│                                                      │
│  请求体模板：                                        │
│  ┌──────────────────────────────────────────┐       │
│  │ {"text":"{{.SenderNickname}} 提醒你：     │       │
│  │   {{.MessageContentPreview}}"}            │       │
│  └──────────────────────────────────────────┘       │
│                                                      │
│  快速填充模板：                                      │
│  [企业微信] [飞书] [Slack] [自定义]                  │
│                                                      │
│  可用变量：                                          │
│  {{.SenderNickname}} 发送者昵称                      │
│  {{.RecipientNickname}} 接收者昵称                   │
│  {{.MessageContentPreview}} 消息内容前100字          │
│  {{.MessageURL}} 消息链接                            │
│  ... (完整列表见侧边栏)                              │
│                                                      │
│  [测试发送] [保存]                                    │
└──────────────────────────────────────────────────────┘
```

**快速填充模板：**

| 系统 | body_template |
|------|---------------|
| 企业微信 | `{"msgtype":"text","text":{"content":"{{.SenderNickname}} 提醒你查看 QIM 消息：{{.MessageContentPreview}}"}}` |
| 飞书 | `{"msg_type":"text","content":{"text":"{{.SenderNickname}} 提醒你查看 QIM 消息：{{.MessageContentPreview}}"}}` |
| Slack | `{"text":"{{.SenderNickname}} 提醒你查看 QIM 消息：{{.MessageContentPreview}}"}` |
| 自定义 | 空 |

**关键代码（Vue 组件片段）：**

```vue
<template>
  <div class="webhook-config-section">
    <h3>消息提醒 Webhook</h3>

    <el-form label-width="100px">
      <el-form-item label="启用">
        <el-switch v-model="config.enabled" />
      </el-form-item>

      <el-form-item label="请求地址">
        <el-input v-model="config.url" placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx" />
      </el-form-item>

      <el-form-item label="请求方法">
        <el-select v-model="config.method" style="width: 120px">
          <el-option label="POST" value="POST" />
          <el-option label="PUT" value="PUT" />
        </el-select>
      </el-form-item>

      <el-form-item label="超时秒数">
        <el-input-number v-model="config.timeout_seconds" :min="1" :max="30" />
      </el-form-item>

      <el-form-item label="签名密钥">
        <el-input v-model="config.secret" type="password" placeholder="HMAC-SHA256 密钥（可选）" show-password />
      </el-form-item>

      <el-form-item label="请求头">
        <div v-for="(header, index) in headerList" :key="index" class="header-row">
          <el-input v-model="header.key" placeholder="Header 名" style="width: 200px" />
          <el-input v-model="header.value" placeholder="Header 值" style="width: 300px" />
          <el-button type="danger" @click="removeHeader(index)">删除</el-button>
        </div>
        <el-button @click="addHeader">+ 添加请求头</el-button>
      </el-form-item>

      <el-form-item label="请求体模板">
        <el-input
          v-model="config.body_template"
          type="textarea"
          :rows="6"
          placeholder='{"text":"{{.SenderNickname}} 提醒你：{{.MessageContentPreview}}"}'
        />
      </el-form-item>

      <el-form-item label="快速填充">
        <el-button @click="applyTemplate('wechat')">企业微信</el-button>
        <el-button @click="applyTemplate('feishu')">飞书</el-button>
        <el-button @click="applyTemplate('slack')">Slack</el-button>
        <el-button @click="applyTemplate('custom')">自定义</el-button>
      </el-form-item>

      <el-form-item label="可用变量">
        <div class="variables-help">
          <code>{{.SenderNickname}}</code> 发送者昵称
          <code>{{.RecipientNickname}}</code> 接收者昵称
          <code>{{.MessageContentPreview}}</code> 消息内容前 100 字符
          <code>{{.MessageURL}}</code> 消息跳转链接
          <code>{{.SenderEmail}}</code> 发送者邮箱
          <code>{{.RecipientEmail}}</code> 接收者邮箱
        </div>
      </el-form-item>

      <el-form-item>
        <el-button type="primary" @click="testWebhook">测试发送</el-button>
        <el-button type="success" @click="saveConfig">保存</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getSystemConfig, updateSystemConfig } from '@/api/system'

const config = reactive({
  enabled: false,
  url: '',
  method: 'POST',
  secret: '',
  timeout_seconds: 10,
  headers: {} as Record<string, string>,
  body_template: ''
})

const headerList = ref<Array<{key: string, value: string}>>([])

const templates = {
  wechat: '{"msgtype":"text","text":{"content":"{{.SenderNickname}} 提醒你查看 QIM 消息：{{.MessageContentPreview}}"}}',
  feishu: '{"msg_type":"text","content":{"text":"{{.SenderNickname}} 提醒你查看 QIM 消息：{{.MessageContentPreview}}"}}',
  slack: '{"text":"{{.SenderNickname}} 提醒你查看 QIM 消息：{{.MessageContentPreview}}"}',
  custom: ''
}

const applyTemplate = (type: keyof typeof templates) => {
  config.body_template = templates[type]
}

const addHeader = () => {
  headerList.value.push({ key: '', value: '' })
}

const removeHeader = (index: number) => {
  headerList.value.splice(index, 1)
}

const syncHeaders = () => {
  const headers: Record<string, string> = {}
  headerList.value.forEach(h => {
    if (h.key) headers[h.key] = h.value
  })
  config.headers = headers
}

const loadConfig = async () => {
  try {
    const result = await getSystemConfig()
    if (result.message_remind_webhook) {
      const webhookCfg = typeof result.message_remind_webhook === 'string'
        ? JSON.parse(result.message_remind_webhook)
        : result.message_remind_webhook
      Object.assign(config, webhookCfg)
      headerList.value = Object.entries(config.headers).map(([key, value]) => ({ key, value }))
    }
  } catch (e) {
    ElMessage.error('加载配置失败')
  }
}

const saveConfig = async () => {
  syncHeaders()
  try {
    await updateSystemConfig({
      message_remind_webhook: JSON.stringify(config)
    })
    ElMessage.success('保存成功')
  } catch (e) {
    ElMessage.error('保存失败')
  }
}

const testWebhook = async () => {
  syncHeaders()
  try {
    await updateSystemConfig({
      message_remind_webhook: JSON.stringify(config)
    })
    ElMessage.info('配置已保存，请在 QIM 客户端右键消息测试')
  } catch (e) {
    ElMessage.error('保存失败')
  }
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.webhook-config-section {
  margin-top: 20px;
  padding: 20px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
}
.header-row {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}
.variables-help {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  font-size: 12px;
  color: var(--text-color-secondary);
}
.variables-help code {
  background: var(--bg-color);
  padding: 2px 6px;
  border-radius: 3px;
  font-family: monospace;
}
</style>
```

- [ ] **步骤 1：在 qim-admin 系统配置页面增加 webhook 配置区**
- [ ] **步骤 2：实现启用开关、URL、method、超时、密钥输入**
- [ ] **步骤 3：实现请求头 key-value 列表编辑**
- [ ] **步骤 4：实现请求体模板文本域**
- [ ] **步骤 5：实现快速填充模板按钮（企微/飞书/Slack/自定义）**
- [ ] **步骤 6：实现可用变量说明**
- [ ] **步骤 7：实现保存和测试按钮**
- [ ] **步骤 8：手动测试完整流程（配置 → 保存 → 客户端触发 → 收到提醒）**
- [ ] **步骤 9：Commit**

```bash
git add qim-admin/src/
git commit -m "feat(remind): 管理后台增加 webhook 配置 UI，含模板预设和变量说明"
```

---

## Webhook 规范

### 1. QIM → 外部系统：请求

**请求头：**
| Header | 说明 | 是否必需 |
|--------|------|---------|
| `Content-Type` | 固定 `application/json; charset=utf-8` | 是 |
| `X-QIM-Signature` | HMAC-SHA256 签名（hex），空 secret 时不发 | 否 |
| `X-QIM-Event` | 固定 `message.remind` | 是 |
| `X-QIM-Timestamp` | 发送时间戳（RFC3339） | 是 |
| `X-QIM-Delivery` | 投递唯一 ID（UUID），用于幂等 | 是 |

**签名算法：**
```
signature = hex( HMAC-SHA256(secret, body_raw_bytes) )
```

**模板变量：**
| 变量 | 说明 |
|------|------|
| `.Event` | 固定 `message.remind` |
| `.DeliveryID` | 投递唯一 ID |
| `.Timestamp` | 发送时间 |
| `.SenderID` / `.SenderUsername` / `.SenderNickname` / `.SenderEmail` | 发送者信息 |
| `.RecipientID` / `.RecipientUsername` / `.RecipientNickname` / `.RecipientEmail` | 接收者信息 |
| `.MessageID` | 消息 ID |
| `.MessageContentPreview` | 消息内容前 100 字符 |
| `.MessageType` | 消息类型 |
| `.MessageSentAt` | 消息发送时间 |
| `.MessageURL` | 可点击跳转链接 |
| `.ConversationID` / `.ConversationType` | 会话信息 |

### 2. 外部系统 → QIM：响应

**硬性要求：** HTTP 状态码 2xx 表示成功，其他视为失败。

### 3. 外部系统建议约定

- 用 `X-QIM-Delivery` 做幂等去重
- 收到请求后先返回 200，异步处理业务逻辑
- 用 secret 验签，拒绝伪造请求
- 检查 `X-QIM-Timestamp` 与当前时间差是否在 5 分钟内

### 4. 外部系统实现示例（Python Flask）

```python
from flask import Flask, request, abort
import hmac, hashlib

app = Flask(__name__)
SECRET = "your-hmac-secret-key"

@app.route('/qim-remind', methods=['POST'])
def qim_remind():
    # 1. 验签
    signature = request.headers.get('X-QIM-Signature', '')
    body = request.get_data()
    expected = hmac.new(SECRET.encode(), body, hashlib.sha256).hexdigest()
    if not hmac.compare_digest(signature, expected):
        abort(401)

    # 2. 幂等检查
    delivery_id = request.headers.get('X-QIM-Delivery')
    if is_processed(delivery_id):
        return {'ok': True}

    # 3. 解析 payload
    payload = request.json
    recipient = payload['data']['recipient']
    message_preview = payload['data']['message']['content_preview']

    # 4. 调用钉钉/企微/邮件/短信等渠道通知 recipient
    send_notification(recipient, message_preview)

    return {'ok': True}
```

---

## 防滥用

| 限制 | 规则 | 实现 |
|------|------|------|
| 同一消息 | 每 1 小时最多提醒 1 次 | `sync.Map` 记录 messageID → 上次提醒时间 |
| 同一用户 | 每分钟最多发起 5 次提醒 | （可选，当前未实现，单聊场景滥用风险低） |

重启清零（可接受，用户大不了再点一次）。

---

## 验证清单

| 场景 | 验证点 |
|------|--------|
| 未配置 webhook | 返回"提醒功能未配置或未启用" |
| 配置已启用 + 企微机器人 | 收到企微群消息，内容含发送者昵称和消息预览 |
| 配置已启用 + 飞书机器人 | 收到飞书群消息 |
| 配置已启用 + Slack | 收到 Slack 消息 |
| webhook URL 错误 | 回执"webhook 调用失败: ..." |
| webhook 返回 500 | 回执"webhook 返回错误: HTTP 500" |
| 同一消息 1 小时内重复提醒 | 返回"该消息已提醒过，请 1 小时后再试" |
| 群聊消息提醒 | 返回"仅支持单聊消息提醒" |
| 消息已读 | 返回"消息已被读，无需提醒" |
| 消息未满 1 小时 | 返回"消息发送未满 1 小时" |
| bot 消息 | 返回"不支持机器人消息提醒" |
| HMAC 签名验证 | 外部系统用 secret 验签通过 |
| 幂等性 | 同一 delivery_id 重复投递，外部系统只处理一次 |
| 前端提醒中提示 | "提醒发送中，结果稍后通知" |
| 前端成功回执 | "提醒已送达外部系统" |
| 前端失败回执 | "提醒发送失败：xxx" |

---

## 取舍说明

- **不做自动重试**：失败回执让用户感知，用户可手动再点一次（受频率限制）。未来若需要自动重试，goroutine 中加重试逻辑即可。
- **不做预设适配器**：custom 模板覆盖企微/飞书/Slack 等主流系统（鉴权都是 URL 拼参数），钉钉加签场景暂不支持（可用关键字鉴权替代）。未来若强需求再加 `adapter_type` 字段扩展。
- **频率限制内存存储**：重启清零，对提醒场景可接受。
- **配置存 SystemConfig**：复用现有基础设施，无需新表。
- **单聊限制**：当前仅支持单聊消息提醒（与前端 canSendReminder 逻辑一致），群聊场景 @提及有独立的未读计数机制。

---

## 环境变量

新增一个可选环境变量：
- `QIM_BASE_URL`：用于构造 `MessageURL`，形如 `https://qim.example.com`。未配置时 MessageURL 为空字符串。

---

## 依赖关系

- T1 是基础，T2 和 T3 依赖 T1
- T2 是基础，T3 依赖 T2
- T4 独立（仅前端改动）
- T5 依赖 T1（配置存储已就绪）

**建议执行顺序：** T1 → T2 → T3 → T4 → T5
