---
title: MCP 接入指南
---

# NUIM MCP 接入指南

## 概述

NUIM 提供标准的 **MCP (Model Context Protocol)** 服务器（分发产物类型为 `mcp`），支持 Claude Code、Cursor 等任何兼容 MCP 协议的 AI Agent 通过标准化接口在 NUIM 内收发消息、管理待办/日历/笔记。

`mcp` 的参数命名与 `cli` 对齐（`to`/`conversation`、`type`、`reply_to`、`file`、`as_user`、`due`/`desc`、`after_id`/`limit`、`keyword`/`conv`、`message_id`/`delta`/`finish`/`abort`），底层调用同一套 REST API。

> 会话目标统一用 `conversation`（会话 ID 或名称）。`thread` 仍作为弃用别名保留，读取/发送/流式工具都接受，但新接入请直接用 `conversation`。

### 与 CLI 的区别

| 特性 | `cli` | `mcp` MCP Server |
|------|-----------|----------------------|
| 协议 | 命令行 + HTTP | 标准 MCP (JSON-RPC) |
| 适用场景 | Bash 脚本、手动操作 | Claude Code、Cursor 等 MCP 客户端 |
| 传输模式 | N/A | stdio / Streamable HTTP |
| 认证方式 | 配置文件存储 token | stdio: 命令行参数; HTTP: 请求头 |
| 能力面 | 全部命令 | 与 CLI 对齐的 24 个工具 |
| 轮询编排 | `agent loop` 常驻 | 由 MCP 客户端自行编排（工具无状态） |

### 文档占位符说明

本文档使用以下占位符，请按实际环境替换：

| 占位符 | 含义 | 示例 |
|--------|------|------|
| `<SERVER>` | NUIM 服务器地址（含协议与端口） | `http://localhost:8080` 或 `https://qim.example.com` |
| `<二进制名>` | mcp 产物二进制名，即管理后台发布 mcp 版本时**上传的文件名**；install.sh 会自动按此名安装到 `~/.local/bin` | `qim-mcp` |
| `<cli二进制名>` | cli 产物二进制名，同理为发布 cli 版本时上传的文件名；用于登录获取用户令牌 | `qim-cli`（详见 [CLI 使用指南](./cli.md)） |
| `qbot_xxxx` / `qbot_your_token_here` | Bot 访问令牌，在管理后台「Bot 运维」页面签发，固定以 `qbot_` 开头 | `qbot_3f9a2b1c8d7e...` |
| `qusr_xxxx` / `eyJhbGciOi...` | 用户令牌：`cli login` 的 JWT 或客户端「设置 → 开发者设置」签发的 `qusr_` 长期令牌 | `qusr_9a8b7c...` |

> 提示：二进制名由管理后台发布版本时上传的原始文件名决定，**不是固定值**。可按团队约定命名（如 `qim-mcp`、`qim-cli`），发布后 install.sh 与下载链接均按该名落地。

---

## 快速开始

### 1. 安装 mcp

`mcp` 走与 `cli` 相同的分发管线，可直接从服务器下载预编译二进制（文件名为上传时设定的名字）。

**方式一：安装脚本一键安装（推荐）**

将 `<SERVER>` 替换为你的服务器地址：

```bash
SERVER=<SERVER> bash -c "$(curl -fsSL <SERVER>/install.sh)"
```

> 脚本会自动检测平台、下载对应预编译二进制、校验 SHA256（按上传文件名）、安装到 `~/.local/bin`（自动加入 PATH），并打印 Claude Code / Cursor 配置指引。

**方式二：curl 直接下载**

适用于想手动放置二进制的场景（`<二进制名>` 为管理后台发布时上传的文件名）：

```bash
# macos/arm64 例如；其他平台把 os/arch 换成 darwin/linux/windows × amd64/arm64
curl -fsSL "<SERVER>/api/v1/cli/download?os=darwin&arch=arm64&product=mcp" -o <二进制名>
chmod +x <二进制名>
sudo mv <二进制名> /usr/local/bin/
```

查看当前平台可用的最新版本与校验值：

```bash
curl -fsSL "<SERVER>/api/v1/cli/version?product=mcp"
```

> 下载地址按平台区分（os/arch），返回的 `sha256` 以该版本上传的二进制文件名为 key。
> 版本由管理后台「CLI 版本」页以「产物类型 = mcp」发布，发布后即可下载。

**方式三：源码构建**

```bash
cd qim-server
go build -o mcp ./cmd/qim-mcp/
```

### 2. 获取令牌

- **Bot 令牌**：在管理后台「Bot 运维」页面创建 Bot 并签发令牌，格式 `qbot_xxxxxxxx`。
- **用户令牌**（可选）：`cli login` 的 JWT，或客户端「设置 → 开发者设置」签发的 `qusr_` 长期令牌。
  待办、日历、笔记、消息搜索、`whoami`、`list_conversations`、`file`、`as_user` 需要它。

### 3. 配置 Claude Code

编辑 `~/.claude/settings.json`（全局）或项目下 `.claude/settings.json`：

```json
{
  "mcpServers": {
    "nuim": {
      "command": "<二进制名>",
      "args": [
        "--token", "qbot_your_token_here",
        "--server", "http://localhost:8080",
        "--user-token", "qusr_your_user_token"
      ]
    }
  }
}
```

配置完成后重启 Claude Code，即可使用 NUIM 工具。不需要用户态能力时可省略 `--user-token`。

### 4. 配置 Cursor

在 Cursor 设置中添加 MCP Server，或编辑 `.cursor/mcp.json`：

```json
{
  "mcpServers": {
    "nuim": {
      "command": "<二进制名>",
      "args": [
        "--token", "qbot_your_token_here",
        "--server", "http://localhost:8080",
        "--user-token", "qusr_your_user_token"
      ]
    }
  }
}
```

---

## 传输模式

### stdio 模式（默认，本地使用）

适用于本地 Claude Code / Cursor，通过标准输入输出通信：

```bash
<二进制名> --token qbot_xxxx --server http://localhost:8080 [--user-token qusr_xxxx]
```

- `--token`（必填）：Bot 访问令牌
- `--server`（可选）：NUIM 服务器地址，默认 `http://localhost:8080`
- `--user-token`（可选）：用户令牌，用于待办、日历、笔记、消息搜索、`whoami`、`list_conversations`、`file`、`as_user`

#### 配置用户令牌

如需使用用户态工具，需先获取用户令牌：

```bash
# 通过 cli 产物登录获取 token
<cli二进制名> login
<cli二进制名> whoami  # 查看当前用户

# 启动 mcp 时传入
<二进制名> --token qbot_xxxx --user-token eyJhbGciOi... --server http://localhost:8080
```

### Streamable HTTP 模式（远程部署）

适用于远程部署，任意 MCP 客户端通过 HTTP 调用：

```bash
<二进制名> --transport http --addr :8082 --server http://localhost:8080
```

- `--transport http`：启用 HTTP 传输
- `--addr`：监听地址，默认 `:8082`
- `--server`：NUIM 服务器地址
- 认证方式：每个请求通过 `Authorization: Bearer qbot_xxx` 头传入 Bot token
- 用户令牌：通过 `X-QIM-User-Token` 头传入（可选，用于用户态接口）
- 运行模式：Stateless（无会话持久化）

#### 生产部署建议

远程 HTTP 模式暴露的是**无会话的公开 MCP 端口**，对外提供前请务必加固：

1. **通过反向代理暴露并启用 TLS**（如 Nginx + Let's Encrypt），**不要**直接裸露 HTTP 端口。示例 Nginx：

   ```nginx
   server {
     listen 443 ssl;
     server_name mcp.example.com;
     ssl_certificate     /etc/letsencrypt/live/mcp.example.com/fullchain.pem;
     ssl_certificate_key /etc/letsencrypt/live/mcp.example.com/privkey.pem;

     location /mcp {
       proxy_pass http://127.0.0.1:8082/mcp;
       proxy_set_header Host $host;
       proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
       proxy_read_timeout 120s;
     }
   }
   ```

2. **限制访问范围**：远程 MCP 对能触达该地址的客户端开放，建议用防火墙/IP 白名单或 VPN 限制到企业内网。
3. **Bot 令牌即权限边界**：每个请求的 `Authorization: Bearer` 令牌决定了该连接可用哪些工具，妥善保管、按需签发/吊销。
4. **部署进程**：建议用 systemd 托管，随系统启动并自动重启；`--addr` 建议绑定 `127.0.0.1:8082`（仅本机），由 Nginx 负责对外。

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
        "to": "1",
        "content": "Hello from MCP!",
        "type": "text"
      }
    }
  }'

# 调用 create_task（需用户令牌）
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

mcp 服务器注册了 24 个标准 MCP 工具，按功能分组：

- **身份与发现**：`whoami` `get_bot_identity` `list_group_conversations` `list_bot_conversations` `list_conversations`
- **消息收发**：`list_messages` `poll_messages` `list_actions` `send_message` `edit_message`
- **流式回复**：`start_streaming_message` `stream_message`
- **搜索**：`search_messages` `search_notes`
- **任务**（需用户令牌）：`list_tasks` `create_task` `update_task`
- **日历**（需用户令牌）：`list_events` `create_event` `update_event`
- **笔记**（需用户令牌）：`list_notes` `get_note` `create_note` `update_note`

### 身份与发现

#### whoami

返回当前用户令牌对应的身份（需用户令牌），等价 CLI `whoami`。

**返回**：`{"id": 14, "username": "alice", "nickname": "爱丽丝", "avatar": "..."}`

#### get_bot_identity

返回当前 bot 令牌对应的身份。`virtual_user_id` 用于识别群消息里的 `@{mention:<id>}` 是否指向自己。

**返回**：`{"id": 8, "name": "E2EBot", "type": "assistant", "virtual_user_id": 15, "is_active": true}`

#### list_group_conversations

列出该 bot 已入群的群会话，等价 CLI `groups`。返回每行一条 JSON：`{"conversation_id":5,"group_name":"项目组A"}`。

#### list_bot_conversations

列出该 bot 关联的全部会话（群聊 + 1:1 私聊）。返回每行一条 JSON：`{"conversation_id":5,"conv_type":"group","name":"项目组A"}`。

#### list_conversations

列出当前**用户**视角的最近会话（需用户令牌），等价 CLI `conversations list`。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `limit` | int | 否 | 最多返回条数，默认 50 |

**返回**：每行一条 JSON，含 `conversation_id` / `conv_type` / `name` / `unread_count` / `last_message`（单聊会回退到 `other_member_name`）。

### 读取消息

#### list_messages

拉取指定会话的一页消息。与 CLI `messages list` 同源：按 id 升序，用 `after_id` 翻页。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `conversation` | string | 是 | 会话 ID 或会话名 |
| `after_id` | uint64 | 否 | 只返回 id 大于该值的消息 |
| `limit` | int | 否 | 最多返回条数（1-100，默认 50） |
| `type` | string | 否 | `text` / `markdown` / `card` / `card_action` / `streaming` |

**返回**：每行一条 JSON 消息：`{id, conversation_id, sender_id, sender_type, sender_nickname, content, type, origin, created_at}`。

#### poll_messages

增量拉取 `after_id` 之后的新消息。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `conversation` | string | 是 | 会话 ID 或会话名 |
| `after_id` | uint64 | 是 | 上次拿到的最大消息 id |
| `limit` | int | 否 | 最多返回条数（1-100，默认 50） |
| `type` | string | 否 | 按消息类型过滤；水位线仍按整批推进 |

**返回**：`{"after_id": 1038, "messages": [...]}`。`after_id` 为本批最大消息 id，下次轮询原样回传。

#### list_actions

查看卡片按钮点击事件（等价 CLI `messages actions`，即 `type=card_action` 的消息）。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `conversation` | string | 是 | 会话 ID 或会话名 |
| `after_id` | uint64 | 否 | 只返回 id 大于该值的消息 |
| `limit` | int | 否 | 最多返回条数（1-100，默认 50） |

**返回**：每行一条 JSON 消息，`action_id` 标识用户点了哪个按钮。

### 发送消息

#### send_message

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `to` | string | 否 | 目标用户 ID 或用户名（单聊）；与 `conversation` 二选一 |
| `conversation` | string | 否 | 会话 ID 或会话名（群聊/已建单聊）；省略时用 `to` 发单聊 |
| `content` | string | 否 | 消息内容；`content` 与 `file` 至少一个 |
| `type` | string | 否 | `text` / `markdown` / `card`，默认 `text` |
| `reply_to` | uint64 | 否 | 引用回复的消息 ID |
| `file` | string | 否 | 本地文件路径（需用户令牌）；上传后以 markdown 链接发送 |
| `as_user` | bool | 否 | 以本人身份发送（需用户令牌），等价 CLI `--as-user` |

**返回**：`{"message_id": 42, "conversation_id": 7}`

**卡片消息**（`type: "card"`）的 content 格式：

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

用户点击按钮后，通过 `list_actions` / `poll_messages` 收到 `type: "card_action"` 的消息。

> `file` 是 mcp 进程所在主机的路径。stdio 模式即调用方本机；远程 HTTP 模式指 MCP 服务所在主机。

#### edit_message

更新一条已存在的 bot 消息（卡片状态回写等）。参数：`message_id`、`content`、`type`（可选，留空保持原类型）。

**返回**：`{"success": true}`

### 流式回复

#### start_streaming_message

创建流式消息占位（用户端显示 typing 状态），返回 `{"message_id": 43, "conversation_id": 7}`。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `to` | string | 否 | 目标用户 ID 或用户名 |
| `conversation` | string | 否 | 会话 ID 或会话名，优先于 `to` |

`to` 与 `conversation` 至少一个。

#### stream_message

等价 CLI `stream`。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `message_id` | uint64 | 是 | 流式消息 ID（来自 `start_streaming_message`） |
| `delta` | string | 否 | 追加的内容片段 |
| `finish` | bool | 否 | 为 true 时收尾并转 markdown |
| `abort` | bool | 否 | 为 true 时丢弃占位（判定无需回复时用） |

`delta`、`finish`、`abort` 至少一个。**返回**：`{"ok": true}`。

### 搜索

#### search_messages

按关键词搜索历史消息（需用户令牌）。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `keyword` | string | 是 | 搜索关键词 |
| `conv` | string | 否 | 限定会话 ID（数字） |
| `limit` | int | 否 | 最多返回条数，默认 20 |

#### search_notes

按语义检索当前用户的笔记（需用户令牌）。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `query` | string | 是 | 检索问题 |
| `top_k` | int | 否 | 返回条数，默认 5 |

### 任务（用户令牌）

#### list_tasks

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `status` | string | 否 | `todo` / `doing` / `done` |
| `limit` | int | 否 | 最多返回条数，默认 50 |

#### create_task

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `title` | string | 是 | 任务标题 |
| `due` | string | 否 | 截止日期 `YYYY-MM-DD` |
| `priority` | string | 否 | `low` / `medium`（默认） / `high` |
| `desc` | string | 否 | 任务描述 |

**返回**：`{"task_id": 123}`

#### update_task

参数：`task_id`（必填）、`status`、`priority`、`title`、`due`（均可选，至少一个）。**返回**：`{"success": true}`

### 日历事件（用户令牌）

#### list_events

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `limit` | int | 否 | 最多返回条数，默认 50 |

#### create_event

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `title` | string | 是 | 事件标题 |
| `start` | string | 是 | 开始时间，如 `2026-08-01 14:00` 或 RFC3339 |
| `end` | string | 是 | 结束时间 |
| `reminder` | int | 否 | 提前提醒分钟数（0 = 不提醒） |
| `desc` | string | 否 | 事件描述 |

**返回**：`{"event_id": 456}`

#### update_event

只传需要修改的字段；本工具会先读取现有事件再合并（服务端要求 title/start/end 全量），不会误清空其他字段。

参数：`event_id`（必填）、`title`、`start`、`end`、`reminder`、`desc`（均可选，至少一个）。
`reminder=0` 表示取消提醒，省略表示不改。**返回**：`{"success": true}`

### 笔记（用户令牌）

#### list_notes

列出当前用户的正式笔记（自动排除便签）。参数：`limit`（可选）。

#### get_note

按 id 获取单条笔记全文。参数：`note_id`。

#### create_note

参数：`title`（必填）、`content`（可选）。**返回**：`{"note_id": 12}`。

#### update_note

只传需要修改的字段；本工具会先读取现有笔记再合并，不会清空未传字段。

参数：`note_id`（必填）、`title`、`content`（均可选，至少一个）。**返回**：`{"success": true}`。

---

## 典型集成流程

### 场景 1：发送卡片并等待用户点击

```
1. send_message(to: "1", content: "{\"title\":\"请选择方案\",\"text\":\"...\",\"buttons\":[...]}", type: "card")
   → {"message_id": 42, "conversation_id": 7}

2. poll_messages(conversation: "7", after_id: 42, type: "card_action")
   → {"after_id": 55, "messages": [ {type: "card_action", action_id: "confirm", ...} ]}
```

### 场景 2：流式回复（AI 逐步输出）

```
1. start_streaming_message(to: "1")
   → {"message_id": 43, "conversation_id": 7}

2. stream_message(message_id: 43, delta: "首先，")
3. stream_message(message_id: 43, delta: "我们需要...")
4. stream_message(message_id: 43, delta: "具体步骤如下。")
5. stream_message(message_id: 43, finish: true)

（若判断无需回复）stream_message(message_id: 43, abort: true)
```

### 场景 3：读取历史 + 发送回复

```
1. get_bot_identity() → 记下 virtual_user_id（识别 @ 自己）
2. list_messages(conversation: "7", limit: 10) → 了解上下文
3. send_message(conversation: "7", content: "根据以上讨论，我的建议是...", type: "markdown")
```

---

## 安全说明

- Bot 令牌以 `qbot_` 开头，在管理后台「Bot 运维」页面签发
- 用户令牌以 `qusr_` 开头（或 `cli login` 的 JWT），仅能访问该用户自己的待办/日历/笔记
- stdio 模式下 token 通过命令行参数传递，仅本机可见
- HTTP 模式下 token 通过 `Authorization` / `X-QIM-User-Token` 头传递，建议配合 TLS 使用
- 所有 API 调用经过 Bot Auth / 用户 Auth 中间件验证，权限范围由令牌决定
- HTTP 模式为 Stateless，不存储会话状态，token 需每次请求携带

---

## 故障排查

| 问题 | 排查方式 |
|------|---------|
| Claude Code 无法连接 | 检查 `~/.claude/settings.json` 中路径和 token 是否正确 |
| HTTP 模式 400/401 | 检查 `Authorization: Bearer qbot_xxx` 头是否正确携带 |
| 待办/笔记工具报 userToken | 配置 `--user-token`（stdio）或 `X-QIM-User-Token` 头（HTTP） |
| 消息发送失败 | 确认 `to` 对应的用户/会话存在，Bot 有发消息权限 |
| 流式消息不显示 | 确认 `start_streaming_message` → `stream_message(delta)` → `stream_message(finish)` 完整调用 |
| poll 返回空 | 确认 `after_id` 为上次返回的 `after_id`（本批最大消息 id） |
| 群内 @ 识别不到自己 | 用 `get_bot_identity` 取 `virtual_user_id`，与消息里的 `@{mention:<id>}` 比对 |
