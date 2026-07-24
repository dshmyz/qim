# QIM MCP Server 使用说明（给 Claude Code / Cursor 等支持 MCP 的 agent）

你（agent）可以通过 MCP（Model Context Protocol）直接调用 QIM 的消息工具，无需手搓 Bash 轮询脚本。
本 MCP server 经 stdio 与你交换 JSON-RPC，底层调用 QIM Bot API（Bearer token 鉴权）。

## 注册（一次性）

把 `mcp-config.json` 的内容合并进你的 MCP 客户端配置（Claude Code 的 `.mcp.json` / Claude Desktop 配置）：

```json
{
  "mcpServers": {
    "qim": {
      "command": "qim-mcp",
      "args": ["--server", "http://localhost:8080", "--token", "qbot_<你的bot令牌>"]
    }
  }
}
```

令牌由 QIM 用户在客户端「我的机器人 - 配置 - 签发令牌」获取（仅显示一次）。
`--server` 指向 QIM 服务端地址。配置后重启客户端，工具即出现在你的工具列表里。

## 工具清单

### 读取消息

- `list_messages(thread_id, limit?)`：列出会话最近消息，每行一条 JSON。
- `poll_messages(thread_id, after_id)`：增量拉 `after_id` 之后的新消息。轮询时传入上次拿到的最大消息 id。

每条消息 JSON：`{id, conversation_id, sender_id, sender_type, sender_nickname, content, type, origin, created_at}`。
`sender_type=="bot"` 的是你自己发的，回复时跳过。

### 发送消息

- `send_message(to_user_id, content, msg_type?, thread_id?)`：向指定用户发一条消息，返回 `{"message_id":...,"conversation_id":...}`。
  `msg_type` 可选 `text|markdown|card`（默认 text）；`card` 时 content 为按钮卡片 JSON。
  `thread_id` 省略时自动建/复用会话；返回的 `conversation_id` 可作后续 list/poll 的 `thread_id`。

### 流式回复（边生成边推送）

- `start_streaming_message(to_user_id, thread_id?)`：创建流式消息占位，返回 `{"message_id":...,"conversation_id":...}`。
- `append_streaming_chunk(message_id, delta)`：追加一段内容增量，可多次调用。
- `finish_streaming_message(message_id)`：收尾，转为最终 markdown 渲染。

典型流式循环：`start` -> 生成中多次 `append_chunk` -> `finish`。`message_id` 需在三次调用间传递。

## 卡片消息（结构化交互）

```
send_message(to_user_id=<id>, content={"title":"确认","text":"是否继续？","buttons":[{"id":"yes","text":"是"},{"id":"no","text":"否"}]}, msg_type="card")
```

用户点按钮后，QIM 以 webhook（event=bot.card_action）回调你的服务端；若你未配 webhook，下次 `poll_messages` 时通过后续消息感知。

## 典型循环

1. `send_message(to_user_id=<用户id>, content=<开场白>)` 拿到 `conversation_id`，记下作 `thread_id`。
   或用 `list_messages(thread_id)` 读既有会话历史，记下最大 id。
2. 循环 `poll_messages(thread_id, after_id=<上次最大id>)` 感知用户新消息。
3. 跳过 `sender_type=="bot"` 的（自己发的）。
4. 生成回复：
   - 一次性：`send_message(to_user_id=<id>, thread_id=<conv_id>, content=<回复>, msg_type="markdown")`。
   - 流式：`start_streaming_message(to_user_id=<id>, thread_id=<conv_id>)` -> 分段 `append_streaming_chunk` -> `finish_streaming_message`（用户实时看到打字效果）。
5. 更新 after_id 为最新 id，回到第 2 步。

## 与 qim CLI 的关系

两者底层调同一套 Bot API，二选一即可：
- **MCP（本工具）**：标准协议，支持 MCP 的 agent 即插即用，推荐。
- **qim CLI**：Bash 驱动，适合不能装 MCP 二进制或需 shell 编排的场景，见 `../qim/examples/CLAUDE.md`。
