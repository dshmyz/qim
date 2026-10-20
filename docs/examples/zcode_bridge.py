#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
QIM ↔ zcode bridge 示例：把 QIM 群聊/私聊的消息桥接给 zcode，回复写回 QIM。

架构（QIM 侧零改动，复用外部 agent bot 基建）：

    QIM 群/私聊 ──pull(Bot API)──> 本进程 ──HTTP──> zcode bridge
         ↑                                              │
         └────────────── 回写 POST /bot/messages ←──────┘

环境变量：
    QIM_SERVER     QIM 服务地址（默认 http://localhost:8080）
    QIM_BOT_TOKEN  外部 agent bot 的 qbot_ 令牌（机器人配置 → Token 管理签发）
    ZCODE_URL      zcode bridge 的 HTTP 端点，如 http://127.0.0.1:8765/api/chat
    STATE_FILE     水位文件（默认 ~/.qim/zcode-bridge-state.json），重启不重放
    POLL_INTERVAL  轮询间隔秒数（默认 2）

zcode 侧约定（唯一需要按真实协议修改的地方见 call_zcode）：
    POST {ZCODE_URL}  body: {"session_id": "<QIM会话ID>", "message": "【发送者】 内容"}
    期望响应: {"reply": "..."}（reply 为空则不回写）

运行前提：机器人已创建（模式「外部 Agent」、webhook_url 留空）并拉进群；
本进程与 `qim agent loop` 互斥——同一 bot 只能有一个消费者，否则双份回复。

注意：若 zcode bridge 实际是 stdin/stdout 形态（每条消息起进程），不需要本脚本，
直接 `qim agent loop --command '<zcode命令>'` 即可。
"""

import json
import os
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

QIM_SERVER = os.environ.get("QIM_SERVER", "http://localhost:8080").rstrip("/")
QIM_BOT_TOKEN = os.environ.get("QIM_BOT_TOKEN", "")
ZCODE_URL = os.environ.get("ZCODE_URL", "http://127.0.0.1:8765/api/chat")
STATE_FILE = os.environ.get("STATE_FILE", os.path.expanduser("~/.qim/zcode-bridge-state.json"))
POLL_INTERVAL = float(os.environ.get("POLL_INTERVAL", "2"))

# QIM 的 @ 提及 token：@{mention:<虚拟用户ID>|<urlencode姓名>}，@all 为 @{mention:all}
MENTION_RE = re.compile(r"@\{mention:(all|[1-9]\d*)(?:\|([^}]*))?\}")


def qim_get(path):
    req = urllib.request.Request(
        QIM_SERVER + path, headers={"Authorization": "Bearer " + QIM_BOT_TOKEN}
    )
    with urllib.request.urlopen(req, timeout=10) as resp:
        return json.load(resp)["data"]


def qim_post(path, body):
    req = urllib.request.Request(
        QIM_SERVER + path,
        data=json.dumps(body).encode("utf-8"),
        headers={"Authorization": "Bearer " + QIM_BOT_TOKEN,
                 "Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=30) as resp:
        return json.load(resp)


def call_zcode(session_id: str, message: str) -> str:
    """调用 zcode bridge——接入真实协议时只改这个函数。

    约定（示例假设，按 zcode 文档调整字段名/路径/鉴权头）：
        POST {ZCODE_URL}  {"session_id": ..., "message": ...}  →  {"reply": ...}
    session_id 用 QIM 会话 ID：同一会话在 zcode 侧共享上下文，不同会话互不串味。
    """
    req = urllib.request.Request(
        ZCODE_URL,
        data=json.dumps({"session_id": session_id, "message": message}).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=120) as resp:
        return (json.load(resp).get("reply") or "").strip()


def decode_mentions(content: str) -> str:
    """@{mention:40|%E6%9C%AC...} → @本地Claude助手，供外部系统阅读。"""
    def repl(m):
        raw = m.group(2) or ""
        return "@" + urllib.parse.unquote(raw) if raw else "@某人"
    return MENTION_RE.sub(repl, content)


def is_mentioned(content: str, virtual_user_id: int) -> bool:
    """群里仅响应 @ 本 bot（私聊没有 @ token，天然全回）。"""
    return any(m.group(1) == str(virtual_user_id) for m in MENTION_RE.finditer(content))


def load_state():
    try:
        with open(STATE_FILE) as f:
            return json.load(f)
    except FileNotFoundError:
        return {"last_id": {}, "virtual_user_id": 0}


def save_state(state):
    os.makedirs(os.path.dirname(STATE_FILE), exist_ok=True)
    tmp = STATE_FILE + ".tmp"
    with open(tmp, "w") as f:
        json.dump(state, f, ensure_ascii=False)
    os.replace(tmp, STATE_FILE)  # 原子写：中断不留半份状态


def discover_conversations(state):
    """自动发现 bot 关联的会话；旧版 server 无 /bot/conversations 时降级 /bot/groups。"""
    try:
        convs = qim_get("/api/v1/bot/conversations")
        out = {c["conversation_id"]: c.get("conv_type", "group") for c in convs}
    except urllib.error.HTTPError as e:
        if e.code != 404:
            raise
        groups = qim_get("/api/v1/bot/groups")
        out = {g["conversation_id"]: "group" for g in groups}
    if not out and state.get("virtual_user_id"):
        # 会话列表为空但身份在：冷启动时先记住身份，等首个会话出现
        pass
    return out


def poll_once(state):
    convs = discover_conversations(state)
    vu = state.get("virtual_user_id") or 0

    for conv_id, conv_type in convs.items():
        key = str(conv_id)
        after = state["last_id"].get(key, 0)
        msgs = qim_get(f"/api/v1/bot/messages?thread_id={conv_id}&after_id={after}&limit=100")
        items = (msgs or {}).get("messages", [])
        if not items:
            continue

        if key not in state["last_id"]:
            # 冷启动跳到最新：bridge 转发的是增量，不回放历史（否则旧消息全部涌向 zcode）
            state["last_id"][key] = max(m["id"] for m in items)
            continue

        for m in items:
            if m["id"] <= after:
                continue
            # 防回声环：bot 发的（含本 bridge 回写的）一律跳过
            if m.get("sender_type") == "bot" or m.get("origin") == "bot":
                continue
            content = m.get("content", "")
            if conv_type == "group" and not is_mentioned(content, vu):
                continue  # 群里仅响应 @ 本 bot；私聊每条都转发
            text = decode_mentions(content)
            sender = m.get("sender_nickname") or f"user#{m.get('sender_id')}"
            reply = call_zcode(key, f"【{sender}】 {text}")
            if reply:
                qim_post("/api/v1/bot/messages", {
                    "conversation_id": conv_id,
                    "content": reply,
                    "msg_type": "markdown",
                })
            after = max(after, m["id"])

        state["last_id"][key] = after

    save_state(state)


def main():
    if not QIM_BOT_TOKEN:
        sys.exit("缺 QIM_BOT_TOKEN：机器人配置 → Token 管理签发（仅显示一次）")

    state = load_state()
    if not state.get("virtual_user_id"):
        me = qim_get("/api/v1/bot/me")
        state["virtual_user_id"] = me.get("virtual_user_id") or 0
        save_state(state)
    print(f"zcode bridge 启动：bot 虚拟用户 {state['virtual_user_id']}，zcode={ZCODE_URL}")

    while True:
        try:
            poll_once(state)
        except urllib.error.HTTPError as e:
            print(f"warn: HTTP {e.code}: {e.reason}", file=sys.stderr)
        except Exception as e:  # 网络抖动不退出，下轮重试
            print(f"warn: {e}", file=sys.stderr)
        time.sleep(POLL_INTERVAL)


if __name__ == "__main__":
    main()
