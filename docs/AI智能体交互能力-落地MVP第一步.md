# Agent↔用户消息闭环 · MVP 第一步落地方案

> 对应《AI智能体交互能力-头脑风暴.md》第七节 MVP 切法 #1：
> **纯文本往返：bot 身份 + 出站 + webhook 回调 + 基础治理（2-3 周）**

## 1. 目标与范围

**目标**：让外部 coding agent（Claude Code / OpenCode 等）能够
1. 主动给 QIM 用户推送消息（agent → QIM）
2. 收到用户在 QIM 内的回复（QIM → agent）

**本期范围（MVP 第一步）**
- bot 身份与 token 鉴权
- 出站 HTTP API（agent → QIM 用户，纯文本/markdown）
- 用户回复经 webhook 回调转发给 agent
- 基础治理：token scope、限流、审计、签名

**不在本期**
- 交互式卡片 / 按钮 / 富媒体（第二步）
- 群聊 bot、流式输出、多模态附件
- 审批队列 UI、主动摘要定时推送
- webhook 失败重试 / 死信队列（第二步）

---

## 2. 关键发现：现有可复用基建（大幅缩小工作量）

调研后确认 QIM **已有一套 bot 系统**，本方案以「扩展」为主，非从零搭建：

| 能力 | 现有实现 | 复用方式 |
|------|---------|---------|
| bot 身份 | `model.Bot`（model.go:305）+ 虚拟 `User{Type:"bot"}`（bot_creation_handler.go:200） | 直接复用，无需建表 |
| bot↔用户会话 | `model.BotConversation{BotID,UserID,ConversationID}`（model.go:343）+ `Conversation{Type:"bot"}` | 直接复用作为 thread |
| bot 创建+审批 | `CreateBot` + `ApprovalTypeBot` + `OperationLogService.LogUserOperation`（bot_creation_handler.go） | webhook 模式 bot 复用审批链 |
| 用户回复触发点 | `handleBotMessage`（message_service.go:219），由 `SendMessage` 在 `convType=="bot"` 时 `SafeGo` 异步调用（:159） | **在此分支**：webhook 模式转发给 agent，否则走现有内部 AI |
| 出站消息范式 | avatar `sendPrivateReply`（avatar_worker_pool.go:174）：建消息→更新 conv→unread+1→`ws.SendToConversation` | 出站 endpoint 照此范式实现 |
| 消息响应组装 | `buildMessageResponse`（message_service.go:956）已对 `Sender.Type=="bot"` 置 `is_ai_message=true` | 前端展示已兼容 |
| webhook 出站 HTTP | `webhook_sender.go`：`sharedHTTPClient` 连接池 + HMAC-SHA256 签名（`X-QIM-Signature`/`X-QIM-Event`/`X-QIM-Delivery`） | 复用签名风格与 client，**不改 `SendRemind`** |
| 非鉴权 token 中间件范式 | `NodeAuthMiddleware`（node_auth.go）：Header 密钥校验 | `BotAuthMiddleware` 仿此，改为 per-bot token |
| 限流 | `NewIPRateLimiter` + `RateLimitMiddleware`（routes.go:205） | 仿作 per-bot 令牌桶 |
| 审计 | `OperationLogService.LogUserOperation` + `OperationLogMiddleware` | 记录 bot 出站与 webhook 投递 |

**结论**：identity / 会话 / 创建审批 / 触发点 / 出站范式 / 签名 / 限流 / 审计 八项基建均已存在。本期净新增 = token 鉴权 + 出站 endpoint + 回调转发分支 + 治理参数化。

---

## 3. 方案分四块

### 一、Bot 身份 + Token 鉴权

**身份**：复用 `model.Bot` + 虚拟 `User{Type:"bot"}` + `BotConversation`，不新建身份表。

**Token 模型**（新增 `model/bot_token.go`）：
```go
type BotToken struct {
    ID         uint      `gorm:"primarykey"`
    BotID      uint      `gorm:"not null;index"`
    TokenHash  string    `gorm:"size:128;not null;uniqueIndex"` // sha256(token), 不存明文
    Name       string    `gorm:"size:64"`                        // 标注用途，如 "claude-code"
    CreatedAt  time.Time
    LastUsedAt *time.Time
    RevokedAt  gorm.DeletedAt `gorm:"index"`                     // 软删除=撤销
}
```

**鉴权中间件**（新增 `middleware/bot_auth.go`，仿 `NodeAuthMiddleware`）：
- 从 `Authorization: Bearer <token>` 取 token
- `sha256(token)` 查 `BotToken`，未撤销 → 预加载 `Bot`（含 `VirtualUserID`）
- 校验 `Bot.IsActive`，否则 403
- 注入 context：`bot_id`、`bot`(*model.Bot)、`virtual_user_id`
- 更新 `LastUsedAt`（异步，防爆破探测时不暴露时序）

**Bot.Config 扩展**（无需迁移，`Config` 已是 text JSON）：
```json
{
  "mode": "external_webhook",        // "internal_ai"(默认, 既有行为) | "external_webhook"
  "webhook_url": "https://agent/callback",
  "webhook_secret": "..."            // HMAC 签名密钥, 与 bot token 分离
}
```
通过既有 `UpdateMyBot` / 管理端更新 Config 透传。

**Token 签发端点**（注册在 `authed`，创建者或 system_admin）：
- `POST /api/v1/bots/:id/token` → 生成随机 token，存 hash，**明文仅此一次返回**
- `DELETE /api/v1/bots/:id/token/:tid` → 撤销

### 二、出站：agent → QIM 用户

**新增 endpoint**（`api` 组下，挂 `BotAuthMiddleware`，不走 `authed` JWT）：
```
POST /api/v1/bot/messages
Authorization: Bearer <bot_token>
{
  "to_user_id": 123,
  "content": "构建失败，请确认是否回滚？",
  "msg_type": "text",          // 可选, 默认 text; 支持 "markdown"
  "thread_id": 456             // 可选, 已有会话 conversation_id; 不传则建/取该 bot+user 的会话
}
→ 200 { "message_id": 789, "conversation_id": 456, "created_at": "..." }
```

**实现**（新增 `service/bot_messaging_service.go`）：
1. 从 context 取 bot，校验 `IsActive` + `VirtualUserID != nil`
2. `EnsureBotConversation(botID, toUserID)`：查 `BotConversation`，无则建 `Conversation{Type:"bot"}` + 两成员 + `BotConversation`（抽取自 `conversation_handler.go:642` 现有 bot 会话逻辑，避免重复）
3. 若传 `thread_id`，校验归属该 bot+user
4. 建 `Message{ConversationID, SenderID: *bot.VirtualUserID, Type: msgType, Content, Origin: "bot", IsRead: true}`
5. 更新 conv `last_message_id`/`last_message_at`
6. 给人类成员 `unread_count + 1`（bot 自身已读）
7. `ws.GlobalHub.SendToConversation(convID, excludeUserID=botVirtualUserID, {type:"new_message", data: buildMessageResponse(...)})`
8. 审计：`OperationLogService.LogUserOperation(c, "bot", "send_message")`，extra 记 to_user_id / message_id

> 范式与 avatar `sendPrivateReply` 一致；`Origin:"bot"` 为新增取值（`buildMessageResponse` 已通过 `Sender.Type=="bot"` 兼容展示）。

### 三、回调：用户回复 → agent

**改动 `service/message_service.go::handleBotMessage`**（唯一侵入点）：

在现有「查 BotConversation → Bot」之后、调内部 AI 之前，加分支：
```go
var cfg botConfig
json.Unmarshal([]byte(bot.Config), &cfg)

if cfg.Mode == "external_webhook" && cfg.WebhookURL != "" {
    // 异步转发用户消息到 agent webhook，不再调内部 AI
    utils.SafeGoWithLabel("bot-webhook", func() {
        s.forwardToWebhook(bot, cfg, userID, convID, content, msg)
    })
    return
}
// 以下保持现有内部 AI 流程不变
```

**转发实现**（新增 `service/bot_webhook.go::SendBotWebhook`）：
- 复用 `webhook_sender.go` 的 `sharedHTTPClient` 与 HMAC-SHA256 签名风格，**不改动 `SendRemind`**（scope 纪律）
- payload：
```json
{
  "event": "bot.message",
  "bot_id": 7,
  "thread_id": 456,           // = conversation_id, agent 据此回复同一会话
  "message_id": 1001,
  "user_id": 123,
  "user_nickname": "张三",
  "user_avatar": "/uploads/...",
  "content": "先别回滚，我看下日志",
  "msg_type": "text",
  "timestamp": "2026-07-23T10:00:00Z",
  "delivery_id": "uuid"       // 幂等键
}
```
- Headers：`X-QIM-Event: bot.message`、`X-QIM-Signature: <HMAC-SHA256(body, webhook_secret)>`、`X-QIM-Timestamp`、`X-QIM-Delivery`
- 失败：记审计（delivery_id + 错误），MVP **不重试**；可选在会话内插一条系统提示「消息投递失败」（默认关，避免噪音）

> `handleBotMessage` 已是 `SafeGo` 异步（message_service.go:159），转发再套一层 `SafeGo` 保证不阻塞；用户消息已在 `SendMessage` 中持久化（:140），转发只读不写主消息。

### 四、基础治理

| 维度 | 做法 |
|------|------|
| token scope | `BotAuthMiddleware` 注入 `bot_id`，出站只能以该 bot 身份发；token 与 webhook_secret 分离 |
| 限流 | bot 组挂独立 per-bot 令牌桶（仿 `NewIPRateLimiter`，key=bot_id，如 60 条/分钟/bot）；全局 IP 限流仍生效 |
| 审计 | 每次出站 + 每次 webhook 投递（成功/失败 + delivery_id）走 `OperationLogService` |
| 签名 | webhook HMAC-SHA256，agent 端可校验来源真实性 |
| 审批 | bot 创建已走 `ApprovalTypeBot`；`external_webhook` 模式同样需审批激活才能用 |
| 撤销 | token 软删除即时生效；bot `IsActive=false` 全局停用 |

---

## 4. 改动文件清单

**NEW**
- `qim-server/model/bot_token.go` — `BotToken` 模型
- `qim-server/middleware/bot_auth.go` — `BotAuthMiddleware`
- `qim-server/service/bot_messaging_service.go` — `EnsureBotConversation` + `SendOutbound`
- `qim-server/service/bot_webhook.go` — `SendBotWebhook`
- `qim-server/handler/bot_api_handler.go` — `SendMessage` / `IssueToken` / `RevokeToken`（+ 可选 `GetConversations`/`GetMessages` 供 agent 拉历史）

**MODIFY**
- `qim-server/service/message_service.go` — `handleBotMessage` 加 webhook 分支（唯一行为侵入点，`internal_ai` 默认分支不变）
- `qim-server/app/routes.go` — 注册 `botAPI` 组（`BotAuthMiddleware`）+ token 端点（`authed`）
- `qim-server/app/init.go` — `AutoMigrate` 模型列表加 `BotToken`
- `qim-server/handler/bot_creation_handler.go` — `CreateBotRequest.Config` 透传 `mode`/`webhook_url`/`webhook_secret`（已透传 Config，仅需文档化字段）；可选：创建并审批通过后自动签发首 token
- `qim-server/model/model.go` — `Message.Origin` 注释补 `'bot'`（仅注释；`buildMessageResponse` 已兼容）

---

## 5. 验收标准

1. agent 持 token `POST /api/v1/bot/messages` → 用户客户端收到 `new_message` WS 推送，会话列表出现该 bot 会话，未读 +1，消息以 bot 身份（头像/昵称）展示
2. 用户在 bot 会话内回复文本 → agent 配置的 `webhook_url` 收到 POST，`X-QIM-Signature` 校验通过，payload 含 `thread_id` / `message_id` / `user_id` / `content`
3. agent 用同一 `thread_id` 再次出站 → 消息落在同一会话（线程连续）
4. 无效 token → 401；非活跃 bot → 403；超限流 → 429
5. 审计日志可见每次出站与每次 webhook 投递结果（含 delivery_id）
6. 既有「内部 AI bot」行为不变（`mode` 默认 `internal_ai` 走原 `handleBotMessage` AI 分支）

---

## 6. 风险与边界

- **bot 给任意用户发消息的滥用风险**：MVP 靠 限流 + 审计 + 撤销 + 审批激活 兜底；后续可加「用户需先关注 bot」或管理员白名单
- **webhook 不可达**：MVP 仅记失败 + delivery_id，不重试；重试/死信列入第二步
- **与现有内部 AI bot 共存**：`mode` 字段区分，默认 `internal_ai` 不改既有行为
- **安全**：token 仅创建时明文返回一次，库内存 sha256；`webhook_secret` 与 bot 出站 token 分离，互不通用
- **多节点**：`SendToConversation` 已支持跨节点广播（`sendToUserToOtherNodes`），bot 出站天然兼容集群

---

## 7. 工时估算（2-3 周）

| 模块 | 人天 |
|------|------|
| Token 模型 + BotAuthMiddleware + 签发端点 | 2 |
| 出站 endpoint + BotMessagingService（EnsureBotConversation 抽取） | 2 |
| handleBotMessage webhook 分支 + SendBotWebhook | 1.5 |
| 治理：per-bot 限流 + 审计接入 + 签名校验 | 1.5 |
| 路由注册 + AutoMigrate + Config 字段文档化 | 0.5 |
| 联调（含一个 demo agent 脚本验证闭环） | 2 |
| 单测 + 边界（撤销 token、超限流、webhook 失败、thread_id 校验） | 2 |
| **合计** | **~13.5 人天** |

---

## 8. 后续衔接（第二步预览，不在本期）

- 交互式卡片（按钮/表单）→ 复用本闭环的 thread_id + 出站 endpoint
- webhook 失败重试 + 死信队列
- 群聊 bot（`BotConversation` 扩展 group_id）
- 主动摘要定时推送（接 `pkg/scheduler`）
- MCP 标准 adapter（让 agent 用标准 MCP 调 QIM 工具，与本闭环互补）
