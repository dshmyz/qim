#!/usr/bin/env bash
# agent-loop.sh - QIM agent 闭环示例
#
# 轮询某会话的新消息，把每条用户消息喂给 Claude Code（或 --mock 回显），
# 回复经 `qim stream-stdin` 流式发回 QIM，气泡在客户端实时增长。
#
# 用法：
#   ./agent-loop.sh --thread <conv_id> [--mock] [--interval 2]
#
# 前置：
#   1. go build -o qim ./cmd/qim  （或 go install ./cmd/qim）
#   2. qim config set --server http://localhost:8080 --token qbot_...
#   3. 在 QIM 里向 bot 发消息触发会话，记下 conv_id（即 thread_id）
#
# --mock：不调 claude，回复 "echo: <用户消息>"，无需 API key 即可验证闭环。
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
if [[ -z "$THREAD" ]]; then echo "--thread 必填" >&2; exit 2; fi

command -v qim >/dev/null 2>&1 || { echo "未找到 qim，请先 go build -o qim ./cmd/qim 并放入 PATH" >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "需要 jq" >&2; exit 1; }
if [[ "$MOCK" -eq 0 ]]; then
  command -v claude >/dev/null 2>&1 || { echo "未找到 claude CLI（--mock 可免验证闭环）" >&2; exit 1; }
fi

echo "[agent-loop] 轮询会话 $THREAD（mock=$MOCK）..." >&2

# qim messages poll 持续输出新消息，每行一个 JSON
qim messages poll --thread "$THREAD" --interval "${INTERVAL}s" 2>/dev/null | while IFS= read -r line; do
  sender_type=$(echo "$line" | jq -r '.sender_type')
  [[ "$sender_type" == "bot" ]] && continue   # 跳过自己发的
  sender_id=$(echo "$line" | jq -r '.sender_id')
  content=$(echo "$line" | jq -r '.content')

  echo "[agent-loop] <- 用户($sender_id): $content" >&2

  if [[ "$MOCK" -eq 1 ]]; then
    # mock：直接流式回显，验证闭环 + 流式气泡
    printf 'echo: %s\n' "$content" | qim stream-stdin --to "$sender_id" --thread "$THREAD"
  else
    # 真实：把用户消息喂给 claude，输出流式回 QIM
    claude -p "你在 QIM 里作为 bot 回复用户。用户说：$content" | qim stream-stdin --to "$sender_id" --thread "$THREAD"
  fi
  echo "[agent-loop] -> 已流式回复" >&2
done
