---
title: MCP 接入指南
---

# NUIM MCP 接入指南

## 概述

NUIM 提供标准的 **MCP (Model Context Protocol)** 服务器 `nuim-mcp`，支持 Claude Code、Cursor 等任何兼容 MCP 协议的 AI Agent 通过标准化接口在 NUIM 内收发消息。

### 与 CLI 的区别

| 特性 | `nuim` CLI | `nuim-mcp` MCP Server |
|------|-----------|----------------------|
| 协议 | 命令行 + HTTP | 标准 MCP (JSON-RPC) |
| 适用场景 | Bash 脚本、手动操作 | Claude Code、Cursor 等 MCP 客户端 |
| 传输模式 | N/A | stdio / Streamable HTTP |
| 认证方式 | 配置文件存储 token | stdio: 命令行参数; HTTP: 请求头 |
| 消息格式 | JSON lines | MCP tool call/result |

---

## 快速开始

### 1. 构建 nuim-mcp

```bash
cd qim-server
go build -o nuim-mcp ./cmd/nuim-mcp/
```

### 2. 获取 Bot 令牌

在管理后台「Bot 运维」页面创建 Bot 并签发令牌，或通过 API 创建：

```bash
# 令牌格式：qbot_xxxxxxxxxxxxxxxx
```

### 3. 配置 Claude Code

编辑 `~/.claude/settings.json`（全局）或项目下 `.claude/settings.json`：

```json
{
  "mcpServers": {
    "nuim": {
      "command": "/path/to/nuim-mcp",
      "args": ["--token", "qbot_your_token_here", "--server", "http://localhost:8080"]
    }
  }
}
```

配置完成后重启 Claude Code，即可使用 NUIM 工具。

### 4. 配置 Cursor

在 Cursor 设置中添加 MCP Server，或编辑 `.cursor/mcp.json`：

```json
{
  "mcpServers": {
    "nuim": {
      "command": "/path/to/nuim-mcp",
      "args": ["--token", "qbot_your_token_here", "--server", "http://localhost:8080"]
    }
  }
}
```

---

## 传输模式

### stdio 模式（默认，本地使用）

适用于本地 Claude Code / Cursor，通过标准输入输出通信：

```bash
nuim-mcp --token qbot_xxxx --server http://localhost:8080
```

- `--token`（必填）：Bot 访问令牌
- `--server`（可选）：NUIM 服务器地址，默认 `http://localhost:8080`
- `--user-token`（可选）：用户 JWT，用于任务管理、日历事件、消息搜索等需要用户身份的接口

#### 配置用户 JWT

如需使用任务、日历等需要用户身份的工具，需先获取用户 JWT：

```bash
# 通过 CLI 登录获取 token
nuim login
nuim whoami  # 查看当前 token

# 启动 MCP 时传入
nuim-mcp --token qbot_xxxx --user-token eyJhbGciOi... --server http://localhost:8080
```

### Streamable HTTP 模式（远程部署）

适用于远程部署，任意 MCP 客户端通过 HTTP 调用：

```bash
nuim-mcp --transport http --addr :8082 --server http://localhost:8080
```

- `--transport http`：启用 HTTP 传输
- `--addr`：监听地址，默认 `:8082`
- `--server`：NUIM 服务器地址
- 认证方式：每个请求通过 `Authorization: Bearer qbot_xxx` 头传入 Bot token
- 用户 JWT：通过 `X-QIM-User-Token` 头传入（可选，用于任务/日历等接口）
- 运行模式：Stateless（无会话持久化）

#### HTTP 模式客户端示例

```bash
# 列出工具
curl -X POST http://localhost:8082/mcp \
  -H "Authorization: Bearer qbot_xxxx" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'

# 调用 send_message（仅需 Bot token）
curl -X POST http://localhost:8082/mcp \
  -H "Authorization: Bearer qbot_xxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "send_message",
      "arguments": {
        "to_user_id": 1,
        "content": "Hello from MCP!",
        "msg_type": "text"
      }
    }
  }'

# 调用 create_task（需用户 JWT）
curl -X POST http://localhost:8082/mcp \
  -H "Authorization: Bearer qbot_xxxx" \
  -H "X-QIM-User-Token: eyJhbGciOi..." \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "create_task",
      "arguments": {
        "title": "修复登录 bug",
        "priority": "high"
      }
    }
  }'
```

---

## MCP 工具列表

nuim-mcp 注册了 13 个标准 MCP 工具，分为四组：

### 消息收发（Bot Token）

#### list_messages

列出指定会话的最近消息。首次进入会话时读取历史。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `thread_id` | uint64 | 是 | 会话 ID |
| `limit` | int | 否 | 最大返回条数 |

**返回**：每行一条 JSON 消息。

#### poll_messages

增量拉取 `after_id` 之后的新消息。用于感知用户回复。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `thread_id` | uint64 | 是 | 会话 ID |
| `after_id` | uint64 | 是 | 上次拿到的最大消息 ID |

**返回**：每行一条 JSON 消息。

#### send_message

向指定用户发送消息。`thread_id` 省略时自动创建/复用会话。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `to_user_id` | uint64 | 是 | 目标用户 ID |
| `thread_id` | uint64 | 否 | 会话 ID（省略则自动创建） |
| `content` | string | 是 | 消息内容 |
| `msg_type` | string | 否 | `text` / `markdown` / `card`，默认 `text` |

**返回**：`{"message_id": 42, "conversation_id": 7}`

**卡片消息**（`msg_type: "card"`）的 content 格式：

```json
{
  "title": "审批请求",
  "text": "用户申请加入群组",
  "buttons": [
    {"id": "approve", "text": "批准", "style": "primary"},
    {"id": "reject", "text": "拒绝"}
  ]
}
```

用户点击按钮后，通过 `poll_messages` 收到 `type: "card_action"` 的消息，其中 `content` 包含 `action_id` 字段标识点击了哪个按钮。

#### start_streaming_message

创建流式消息占位（用户端显示 typing 状态），返回 `message_id`。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `to_user_id` | uint64 | 是 | 目标用户 ID |
| `thread_id` | uint64 | 否 | 会话 ID |

**返回**：`{"message_id": 43, "conversation_id": 7}`

#### append_streaming_chunk

向流式消息追加一段内容增量。可多次调用。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `message_id` | uint64 | 是 | 流式消息 ID（来自 `start_streaming_message`） |
| `delta` | string | 是 | 追加的内容片段 |

**返回**：`{"ok": true}`

#### finish_streaming_message

收尾流式消息，将其转为最终 Markdown 渲染。调用后不再接受追加。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `message_id` | uint64 | 是 | 流式消息 ID |

**返回**：`{"ok": true}`

### 消息增强（Bot Token）

#### edit_message

更新一条已存在的 bot 消息内容，用于卡片状态回写、修正错误等场景。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `message_id` | uint64 | 是 | 消息 ID |
| `content` | string | 是 | 新消息内容 |
| `msg_type` | string | 否 | 消息类型（留空保持原类型） |

**返回**：`{"success": true}`

#### search_messages

按关键词搜索历史消息（需要用户 JWT）。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `keyword` | string | 是 | 搜索关键词 |
| `conversation_id` | uint64 | 否 | 限定会话 ID |
| `limit` | int | 否 | 最大返回条数 |

**返回**：每行一条搜索命中 JSON。

### 任务管理（用户 JWT）

#### list_tasks

列出当前用户的待办任务。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `status` | string | 否 | 过滤状态：`todo` / `doing` / `done` |
| `limit` | int | 否 | 最大返回条数，默认 50 |

**返回**：每行一条任务 JSON。

#### create_task

创建一条待办任务。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `title` | string | 是 | 任务标题 |
| `due_date` | string | 否 | 截止日期（YYYY-MM-DD） |
| `priority` | string | 否 | 优先级：`low` / `medium`（默认） / `high` |
| `description` | string | 否 | 任务描述 |

**返回**：`{"task_id": 123}`

#### update_task

更新一条待办任务的字段。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `task_id` | uint64 | 是 | 任务 ID |
| `status` | string | 否 | `todo` / `doing` / `done` |
| `priority` | string | 否 | `low` / `medium` / `high` |
| `title` | string | 否 | 新标题 |
| `due_date` | string | 否 | 新截止日期 |

**返回**：`{"success": true}`

### 日历事件（用户 JWT）

#### list_events

列出当前用户的日历事件。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `limit` | int | 否 | 最大返回条数，默认 50 |

**返回**：每行一条事件 JSON。

#### create_event

创建一条日历事件。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `title` | string | 是 | 事件标题 |
| `start` | string | 是 | 开始时间（RFC3339 格式，如 `2026-08-01T14:00:00+08:00`） |
| `end` | string | 是 | 结束时间（RFC3339 格式） |
| `reminder` | int | 否 | 提前提醒分钟数，0 表示不提醒 |
| `description` | string | 否 | 事件描述 |

**返回**：`{"event_id": 456}`

#### update_event

更新一条日历事件的字段。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `event_id` | uint64 | 是 | 事件 ID |
| `title` | string | 否 | 新标题 |
| `start` | string | 否 | 新开始时间 |
| `end` | string | 否 | 新结束时间 |
| `reminder` | int | 否 | 新提醒分钟数 |

**返回**：`{"success": true}`

---

## 典型集成流程

### 场景 1：发送消息并等待用户回复

```
1. send_message(to_user_id: 1, content: "请选择方案", msg_type: "card")
   → {"message_id": 42, "conversation_id": 7}

2. poll_messages(thread_id: 7, after_id: 42)
   → 等待用户点击...

3. 收到 card_action 消息，解析 action_id，继续处理
```

### 场景 2：流式回复（AI 逐步输出）

```
1. start_streaming_message(to_user_id: 1)
   → {"message_id": 43, "conversation_id": 7}

2. append_streaming_chunk(message_id: 43, delta: "首先，")
3. append_streaming_chunk(message_id: 43, delta: "我们需要...")
4. append_streaming_chunk(message_id: 43, delta: "具体步骤如下。")

5. finish_streaming_message(message_id: 43)
   → 消息转为 Markdown 渲染，用户端不再显示 typing
```

### 场景 3：读取历史 + 发送回复

```
1. list_messages(thread_id: 7, limit: 10)
   → 获取最近 10 条消息，了解上下文

2. send_message(to_user_id: 1, thread_id: 7, content: "根据以上讨论，我的建议是...")
```

---

## 安全说明

- Bot 令牌以 `qbot_` 开头，在管理后台「Bot 运维」页面签发
- stdio 模式下 token 通过命令行参数传递，仅本机可见
- HTTP 模式下 token 通过 `Authorization: Bearer` 头传递，建议配合 TLS 使用
- 所有 API 调用经过 Bot Auth 中间件验证，权限范围由 Bot 配置决定
- HTTP 模式为 Stateless，不存储会话状态，token 需每次请求携带

---

## 故障排查

| 问题 | 排查方式 |
|------|---------|
| Claude Code 无法连接 | 检查 `~/.claude/settings.json` 中路径和 token 是否正确 |
| HTTP 模式 401 | 检查 `Authorization: Bearer qbot_xxx` 头是否正确携带 |
| 消息发送失败 | 确认 `to_user_id` 存在，Bot 有发消息权限 |
| 流式消息不显示 | 确认 `start` → `append` → `finish` 三步完整调用 |
| poll 返回空 | 确认 `after_id` 设置正确（应为上次最大消息 ID） |
