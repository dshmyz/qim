#!/usr/bin/env bash
# agent-loop.sh - QIM agent 闭环示例（支持卡片往返）
#
# 轮询会话新消息，分两种事件处理：
#   - 用户文本  -> Claude 回复（或 --mock 发一张确认卡片）-> 流式回 QIM
#   - 用户点击卡片 -> card_action -> Claude 执行对应动作 -> 流式回 QIM
#
# 用法：
#   ./agent-loop.sh --thread <conv_id> [--mock] [--interval 2]
#
# 前置：
#   1. go build -o qim ./cmd/qim  （或 go install ./cmd/qim）
#   2. qim config set --server http://localhost:8080 --token qbot_...
#   3. 在 QIM 里向 bot 发消息触发会话，记下 conv_id（即 thread_id）
#
# --mock 全往返（无需 claude/API key）：
#   1. 用户发任意文本 -> bot 回一张确认卡片（[确认][取消]）
#   2. 用户点卡片按钮 -> bot 流式回复「已执行：{按钮文本}」
#   验证：发卡 -> 点击 -> card_action 拉取 -> 流式回复，闭环畅通且不产生 webhook 死信
#   （纯 pull 模式 bot：webhook_url 空，点击事件靠 GET /bot/messages 拉到，不经 outbox）。
# 真实模式：需安装 claude CLI 并登录。
set -euo pipefail

THREAD=""
MOCK=0
INTERVAL=2
while [[ $# -gt 0 ]]; do
  case "$1" in
    --thread) THREAD="$2"; shift 2;;
    --mock) MOCK=1; shift;;
    --interval) INTERVAL="$2"; shift 2;;
    *) echo "未知参数: $1" >&2; exit 2;;
  esac
done
[[ -n "$THREAD" ]] || { echo "--thread 必填" >&2; exit 2; }

command -v qim >/dev/null 2>&1 || { echo "未找到 qim，请先 go build -o qim ./cmd/qim 并放入 PATH" >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "需要 jq" >&2; exit 1; }
# CLAUDE 可覆盖：如 host 有 ANTHROPIC_BASE_URL 干扰，用 CLAUDE="env -i HOME=\$HOME PATH=\$PATH claude" ./agent-loop.sh
CLAUDE="${CLAUDE:-claude}"
if [[ "$MOCK" -eq 0 ]]; then
  command -v claude >/dev/null 2>&1 || { echo "未找到 claude CLI（--mock 可免验证闭环）" >&2; exit 1; }
fi

# 发一张确认卡片，输出 message_id
send_card() {
  local to="$1" title="$2"
  local card
  card=$(jq -nc --arg t "$title" \
    '{title:$t,buttons:[{id:"confirm",text:"确认",style:"primary",value:"confirm"},{id:"cancel",text:"取消",value:"cancel"}]}')
  qim send --to "$to" --thread "$THREAD" --type card --content "$card"
}

# 流式回一段文本（按行追加，EOF finish）
stream_reply() {
  local to="$1" text="$2"
  printf '%s\n' "$text" | qim stream-stdin --to "$to" --thread "$THREAD"
}

echo "[agent-loop] 轮询会话 $THREAD（mock=$MOCK）..." >&2

# 注意：不抑制 poll 的 stderr，限流(429)/网络错误会打到日志，便于排障。
# 限流时 poll 返回空，loop 静默空转——调大 --interval 或放宽 bot 限流（默认 60/min）。
qim messages poll --thread "$THREAD" --interval "${INTERVAL}s" | while IFS= read -r line; do
  sender_type=$(echo "$line" | jq -r '.sender_type')
  [[ "$sender_type" == "bot" ]] && continue   # 跳过自己发的（含自己发的卡片）
  sender_id=$(echo "$line" | jq -r '.sender_id')
  msg_type=$(echo "$line" | jq -r '.type')
  content=$(echo "$line" | jq -r '.content')

  # 卡片点击：card_action（pull-mode agent 靠此拉到用户点击事件，不走 webhook）
  if [[ "$msg_type" == "card_action" ]]; then
    action_id=$(echo "$content" | jq -r '.action_id // empty')
    action_text=$(echo "$content" | jq -r '.action_text // .action_id // "操作"')
    echo "[agent-loop] <- 用户($sender_id) 点击卡片: $action_text ($action_id)" >&2
    if [[ "$MOCK" -eq 1 ]]; then
      stream_reply "$sender_id" "已为你执行：${action_text}"
    else
      $CLAUDE -p "用户在 QIM 卡片上点了「$action_text」（action_id=$action_id）。请据此执行对应操作并简短回复。" \
        | qim stream-stdin --to "$sender_id" --thread "$THREAD"
    fi
    echo "[agent-loop] -> 已处理点击" >&2
    continue
  fi

  # 普通文本
  echo "[agent-loop] <- 用户($sender_id): $content" >&2
  if [[ "$MOCK" -eq 1 ]]; then
    # mock：发一张确认卡片，演示 发卡 -> 点击 -> card_action 往返
    send_card "$sender_id" "确认执行「$content」?"
    echo "[agent-loop] -> 已发卡片" >&2
  else
    # 真实：把用户消息喂给 claude，输出流式回 QIM
    $CLAUDE -p "你在 QIM 里作为 bot 回复用户。用户说：$content" \
      | qim stream-stdin --to "$sender_id" --thread "$THREAD"
    echo "[agent-loop] -> 已流式回复" >&2
  fi
done
