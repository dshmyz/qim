# QIM MCP Server 使用说明（给 Claude Code / Cursor 等支持 MCP 的 agent）

你（agent）可以通过 MCP（Model Context Protocol）直接调用 QIM 的工具，无需手搓 Bash 轮询脚本。
本 MCP server 的参数命名与 `qim` CLI 对齐：`to`/`thread`/`conversation`、`type`、`reply_to`、
`file`、`as_user`、`due`/`desc`、`after_id`/`limit`、`keyword`/`conv`、`message_id`/`delta`/`finish`/`abort`。
支持两种传输模式，底层均调用 QIM API（Bearer token 鉴权）：

- **stdio**（默认）：本地子进程，经 stdin/stdout 交换 JSON-RPC，适合本地 Claude Code / Cursor。
- **StreamableHTTP**：部署为独立 HTTP 服务，agent 经 HTTP 调用，适合远程部署、多客户端共享。

## 注册（一次性）

把 `mcp-config.json` 的内容合并进你的 MCP 客户端配置（Claude Code 的 `.mcp.json` / Claude Desktop 配置）：

```json
{
  "mcpServers": {
    "qim": {
      "command": "qim-mcp",
      "args": ["--server", "http://localhost:8080", "--token", "qbot_<bot令牌>", "--user-token", "qusr_<用户令牌>"]
    }
  }
}
```

- `--token`：Bot 令牌，用户在客户端「我的机器人 - 配置 - 签发令牌」获取（仅显示一次）。
- `--user-token`：可选。`qim login` 的 JWT，或客户端「设置 → 开发者设置」签发的 `qusr_` 长期令牌。
  待办、日历、笔记、消息搜索、`whoami`、`list_conversations`、`file`、`as_user` 都需要它。

`--server` 指向 QIM 服务端地址。配置后重启客户端，工具即出现在你的工具列表里。

### 远程部署（StreamableHTTP）

将 qim-mcp 部署为 HTTP 服务后，agent 无需本地安装二进制，经 HTTP 连接：

```bash
# 启动（bot token 由请求 Authorization 头传入，用户 token 由 X-QIM-User-Token 头传入）
qim-mcp --transport http --addr :8082 --server http://localhost:8080
```

```json
{
  "mcpServers": {
    "qim": {
      "type": "url",
      "url": "http://your-server:8082/mcp",
      "headers": {
        "Authorization": "Bearer qbot_<bot令牌>",
        "X-QIM-User-Token": "qusr_<用户令牌>"
      }
    }
  }
}
```

HTTP 模式为 stateless（无会话持久化），适合部署在服务器上供多个 agent 共享。

## 工具清单

### 身份与发现

- `whoami()`：当前用户身份（**需 user token**），等价 CLI `qim whoami`。
- `get_bot_identity()`：当前 bot 身份，含 `virtual_user_id`。群里判断 `@{mention:<id>}` 是否指向自己必须用它。
- `list_group_conversations()`：bot 已入群的群会话，等价 CLI `qim groups`。
- `list_bot_conversations()`：bot 关联的全部会话（群 + 1:1 私聊）。
- `list_conversations(limit?)`：当前**用户**视角的最近会话（**需 user token**），等价 CLI `qim conversations list`，
  含未读数与最后一条消息。

### 读取消息

- `list_messages(thread, after_id?, limit?, type?)`：拉一页消息。`thread` 可为会话 ID 或会话名；
  `type` 可选 `text|markdown|card|card_action|streaming`。与 CLI `messages list` 同源：按 id 升序，
  用 `after_id` 翻页，需要最新水位线时取返回结果里的最大 id。
- `poll_messages(thread, after_id, limit?, type?)`：增量拉取。返回
  `{"after_id": <本批最大消息 id>, "messages": [...]}`，下次轮询把 `after_id` 原样回传。
  即使 `type` 过滤掉部分消息，水位线仍按整批推进，不会卡住。
- `list_actions(thread, after_id?, limit?)`：卡片按钮点击事件（等价 CLI `qim messages actions`），
  即 `type=card_action` 的消息，`action_id` 标识用户点了哪个按钮。

每条消息 JSON：`{id, conversation_id, sender_id, sender_type, sender_nickname, content, type, origin, created_at}`。
`sender_type=="bot"` 的是你自己发的，回复时跳过。

### 发送消息

- `send_message(to?, thread?, conversation?, reply_to?, type?, content?, file?, as_user?)`：
  - 单聊：`to`（用户 ID 或用户名），`thread` 可选（会话 ID 或名；省略自动建/复用）。
  - 群聊/已建会话：`conversation`（会话 ID），优先于 `to`/`thread`。
  - `type` 可选 `text|markdown|card`（默认 text）。
  - `reply_to` 引用回复某条消息。
  - `file` 上传本地文件并以 markdown 链接发送（**需 user token**；路径须是 qim-mcp 进程可读的本地路径）。
  - `as_user=true` 以本人身份发送（**需 user token**），与 CLI `send --as-user` 一致。
  - 返回 `{message_id, conversation_id}`。后续 list/poll 用 `conversation_id` 作 `thread`。
- `edit_message(message_id, content, type?)`：更新已发出的 bot 消息（卡片状态回写等）。

何时用 card：当需要用户在几个明确选项中做决策（确认/取消、批准/拒绝、选择方案）时，优先用 card。
`content` 形如 `{"title":"标题","text":"说明","buttons":[{"id":"confirm","text":"确认"},{"id":"cancel","text":"取消"}]}`。
用户点击后通过 `list_actions` / `poll_messages` 收到 `type=card_action` 的消息，含 `action_id`。

### 流式回复（边生成边推送）

- `start_streaming_message(to?, thread?, conversation?)`：创建流式占位，返回 `{message_id, conversation_id}`。
- `stream_message(message_id, delta?, finish?, abort?)`：等价 CLI `qim stream`。
  - `delta` 追加一段内容，可多次调用。
  - `finish=true` 收尾，转为最终 markdown。
  - `abort=true` 丢弃占位（你判断无需回复时用，避免留下空气泡）。

典型循环：`start_streaming_message` → 多次 `stream_message(delta=...)` → `stream_message(finish=true)`。

### 搜索

- `search_messages(keyword, conv?, limit?)`：按关键词搜索历史消息（**需 user token**）；`conv` 限定会话 ID。
- `search_notes(query, top_k?)`：语义检索用户笔记（**需 user token**）。

### 待办

- `list_tasks(status?, limit?)`：列出待办（**需 user token**）。
- `create_task(title, due?, priority?, desc?)`：`due` 为 `YYYY-MM-DD`，`priority` 为 `low|medium|high`。
- `update_task(task_id, status?, priority?, title?, due?)`：只传要改的字段。

### 日历

- `list_events(limit?)`
- `create_event(title, start, end, reminder?, desc?)`：`start`/`end` 支持 `2026-08-01 14:00` 本地时间或 RFC3339；
  `reminder` 为提前提醒分钟数（0 = 不提醒）。
- `update_event(event_id, title?, start?, end?, reminder?, desc?)`：只传要改的字段；
  服务端要求全量，本工具会先读取现有事件合并，不会误清空其他字段。`reminder=0` 表示取消提醒，省略表示不改。

### 笔记

- `list_notes(limit?)`：正式笔记（自动排除便签），等价 CLI `qim note list`。
- `get_note(note_id)` / `create_note(title, content?)`
- `update_note(note_id, title?, content?)`：只传要改的字段；本工具会先读取现有笔记合并，不会清空未传字段。

## 典型循环

1. `get_bot_identity()` 拿到 `virtual_user_id`；`list_group_conversations()` / `list_bot_conversations()` 发现会话。
2. 用 `list_messages(thread=<conversation_id>)` 读取历史，从返回消息里取最大 id 作为起点。
3. 循环 `poll_messages(thread=<id>, after_id=<上次返回的 after_id>)` 感知新消息。
4. 跳过 `sender_type=="bot"` 的（自己发的）。
5. 生成回复：
   - 一次性：`send_message(conversation=<conv_id>, content=<回复>, type="markdown")`。
   - 流式：`start_streaming_message(conversation=<conv_id>)` → `stream_message(delta=...)` → `stream_message(finish=true)`。
6. 把 `poll_messages` 返回的 `after_id` 记下来，回到第 3 步。

## 与 qim CLI 的关系

两者底层调同一套 API，参数命名与能力面对齐，二选一即可：
- **MCP（本工具）**：标准协议，支持 MCP 的 agent 即插即用，推荐。
- **qim CLI**：Bash 驱动，适合不能装 MCP 二进制或需 shell 编排的场景，见 `cmd/qim/examples/CLAUDE.md`。

注意：`file` 参数是 qim-mcp **进程所在主机**的路径。stdio 模式即你的本机；远程 HTTP 模式指 MCP 服务所在主机，
此时建议改用 `content` 传下载 URL。用户令牌通过 `--user-token`（stdio）或 `X-QIM-User-Token`（HTTP）配置。
