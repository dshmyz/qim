# QIM Agent CLI 使用说明（给 Claude Code / OpenCode）

你（agent）可以通过 Bash 调用 `qim` CLI 在 QIM 即时通讯系统里收发消息。人机共用同一套命令。

## 配置（一次性）

```bash
qim config set --server http://localhost:8080 --token qbot_<你的bot令牌>
qim config show   # 检查（token 脱敏）
```

令牌由 QIM 管理员在「我的机器人 - 配置 - 签发令牌」获取，仅显示一次。

## 读取用户消息（pull）

```bash
# 列出某会话的全部/增量消息，每行一个 JSON
qim messages list --thread <conv_id> [--after-id <上一条id>] [--limit 50]

# 持续轮询新消息（阻塞，每条新消息输出一行 JSON）
qim messages poll --thread <conv_id> [--interval 2s]
```

每条消息 JSON：`{id, conversation_id, sender_id, sender_type, sender_nickname, content, type, origin, created_at}`。
`sender_type=="bot"` 的是你自己发的，回复时跳过。

## 回复用户

```bash
# 一次性发消息，stdout 输出 message_id
qim send --to <user_id> --thread <conv_id> --type text|markdown|card --content "..."

# 流式回复（推荐）：建流式消息，stdin 逐行作为增量，EOF 结束转 markdown
echo "第一段" | qim stream-stdin --to <user_id> --thread <conv_id>
# 或配合 claude：
claude -p "用户说：xxx" | qim stream-stdin --to <user_id> --thread <conv_id>

# 手动分段
qim stream --message-id <id> --delta "..."   # 追加
qim stream --message-id <id> --finish        # 结束
```

## 卡片消息（结构化交互）

```bash
qim send --to <user_id> --thread <conv_id> --type card --content '{"title":"确认","text":"是否继续？","buttons":[{"id":"yes","text":"是"},{"id":"no","text":"否"}]}'
```

用户点按钮后，QIM 会以 webhook（event=bot.card_action）回调，或你下次 `messages poll` 时通过后续消息感知。

## 典型循环

```bash
qim messages poll --thread <conv_id> | while read msg; do
  sender=$(echo "$msg" | jq -r '.sender_type')
  [ "$sender" = "bot" ] && continue
  user_msg=$(echo "$msg" | jq -r '.content')
  user_id=$(echo "$msg" | jq -r '.sender_id')
  claude -p "回复用户：$user_msg" | qim stream-stdin --to "$user_id" --thread <conv_id>
done
```

完整可跑示例见同目录 `agent-loop.sh`（支持 `--mock` 免 API key 验证闭环）。
