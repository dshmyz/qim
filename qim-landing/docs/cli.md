---
title: CLI 使用指南
---

# NUIM CLI 使用指南

## 概述

`nuim` 是 NUIM 的命令行客户端工具，基于 Cobra 构建的纯 HTTP 客户端。它不耦合服务端内部逻辑，只依赖 REST API 契约，适用于：

- **AI Agent 集成**：Claude Code、OpenCode 等 agent 通过 Bash 调用 CLI 在 NUIM 内收发消息
- **自动化脚本**：消息投递、任务管理、日历操作
- **运维排查**：快速查看会话消息、搜索历史

---

## 安装

### 从源码编译

```bash
cd qim-server
go build -o nuim ./cmd/nuim/
```

### 跨平台编译

```bash
# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o nuim-darwin-arm64 ./cmd/nuim/

# Linux
GOOS=linux GOARCH=amd64 go build -o nuim-linux-amd64 ./cmd/nuim/

# Windows
GOOS=windows GOARCH=amd64 go build -o nuim-windows-amd64.exe ./cmd/nuim/
```

### 自动更新

```bash
nuim update          # 从服务器下载最新版，SHA256 校验后替换
nuim update --insecure  # 跳过 SHA256 校验（不推荐）
```

---

## 配置

配置文件路径：`~/.nuim/config.json`

### 首次配置

```bash
# 设置服务器地址和 Bot 令牌
nuim config set --server http://localhost:8080 --token qbot_your_bot_token

# （可选）追加用户 JWT，用于以用户身份操作任务/日历
nuim config set --user-token eyJhbGciOi...
```

### 查看配置

```bash
nuim config show
# 输出（token 脱敏）：
# server_url:  http://localhost:8080
# bot_token:   qbot_abc...xyz
# user_token:  eyJhbGc...OiJ9
```

---

## 命令速查

### 身份验证

| 命令 | 说明 |
|------|------|
| `nuim login` | 交互式登录（用户名/密码），获取 JWT + refresh token |
| `nuim whoami` | 验证 user_token，显示当前用户信息 |
| `nuim version` | 显示 CLI 版本号 |
| `nuim update` | 自动更新 CLI 到最新版 |

### 消息操作

| 命令 | 说明 |
|------|------|
| `nuim send` | 发送消息（文本/Markdown/卡片） |
| `nuim messages list` | 拉取会话历史消息 |
| `nuim messages poll` | 轮询新消息（持续/单次） |
| `nuim messages actions` | 查看卡片点击事件 |
| `nuim messages edit` | 更新消息内容（卡片状态回写） |
| `nuim messages search` | 按关键词搜索消息 |
| `nuim stream` | 追加流式分段到已有消息 |
| `nuim stream-stdin` | 从 stdin 逐行读取，流式发送 |

### 任务与日历

| 命令 | 说明 |
|------|------|
| `nuim task list` | 列出待办任务 |
| `nuim task create` | 创建任务 |
| `nuim task update` | 更新任务状态 |
| `nuim event list` | 列出日历事件 |
| `nuim event create` | 创建日历事件 |
| `nuim event update` | 更新日历事件 |

### 会话管理

| 命令 | 说明 |
|------|------|
| `nuim conversations list` | 列出最近会话 |

### 工具命令

| 命令 | 说明 |
|------|------|
| `nuim completion bash/zsh/fish/powershell` | 生成 shell 补全脚本 |

---

## 使用示例

### 发送文本消息

```bash
# 向用户 alice 发送一条文本消息（自动创建/查找会话）
nuim send --to alice --content "你好，这是一条测试消息"

# 指定会话
nuim send --to alice --thread "项目讨论" --content "明天开会"
```

### 发送 Markdown 消息

```bash
nuim send --to alice --type markdown --content "**加粗** 和 `代码`"

# 从文件读取内容发送
nuim send --to alice --type markdown --content - < report.md
```

### 发送交互卡片

```bash
# 发送带按钮的交互卡片
nuim send --to alice --type card --content '{
  "title": "审批请求",
  "text": "用户 bob 申请加入群组「研发部」",
  "buttons": [
    {"id": "approve", "text": "批准", "style": "primary"},
    {"id": "reject", "text": "拒绝"}
  ]
}'

# 等待用户点击结果
nuim messages poll --thread alice --type card_action --once --output json
```

### 发送文件

```bash
# 上传文件并以 Markdown 链接发送（需要 user_token）
nuim send --to alice --file ./design.pdf
```

### 读取消息

```bash
# 拉取最近 50 条消息
nuim messages list --thread alice

# 只看最新 10 条
nuim messages list --thread alice --limit 10

# 过滤特定类型
nuim messages list --thread alice --type markdown

# 搜索消息
nuim messages search --keyword "部署"
```

### 轮询新消息（Agent 模式）

```bash
# 持续监听用户回复
nuim messages poll --thread alice

# 只等一次新消息就退出（适合 agent 一次性等待）
nuim messages poll --thread alice --once

# 等待卡片点击结果
nuim messages poll --thread alice --type card_action --once

# 自定义轮询间隔（默认 2s）
nuim messages poll --thread alice --interval 5s
```

### 流式消息（配合 AI Agent）

```bash
# 管道模式：Claude 回复 → NUIM 流式消息
claude -p "解释量子计算" | nuim stream-stdin --to alice

# 手动控制流式消息
# 1. 创建流式消息占位
msg_id=$(nuim send --to alice --type streaming --content "" --output id)

# 2. 逐段追加内容
nuim stream --message-id $msg_id --delta "第一段内容..."
nuim stream --message-id $msg_id --delta "第二段内容..."

# 3. 结束流式，转为 Markdown 渲染
nuim stream --message-id $msg_id --finish
```

### 任务管理

```bash
# 创建任务（需要 user_token）
nuim task create --title "修复登录 bug" --assignee alice

# 查看任务列表
nuim task list

# 更新任务状态
nuim task update --id 1 --status done
```

### 输出格式控制

```bash
# 默认：人类可读格式
nuim messages list --thread alice

# 原始 JSON（适合脚本解析）
nuim messages list --thread alice --output json

# 裸 ID（方便管道取值）
nuim send --to alice --content "hi" --output id
# 输出: 42
```

---

## Agent 集成模式

### Claude Code + Bash 集成

```bash
# 1. 配置 CLI
nuim config set --server http://localhost:8080 --token qbot_...

# 2. Agent 发消息给用户
nuim send --to user1 --type card --content '{"title":"选择","text":"你想怎么做？","buttons":[{"id":"a","text":"方案A"},{"id":"b","text":"方案B"}]}'

# 3. 轮询等待用户回复
nuim messages poll --thread user1 --type card_action --once --output json
# → {"id":42,"content":"{\"action_id\":\"a\"}","type":"card_action",...}

# 4. 根据用户选择继续处理
```

### Shell 脚本自动化

```bash
#!/bin/bash
# 批量通知脚本
for user in alice bob charlie; do
  nuim send --to "$user" --content "系统将于今晚 22:00 维护"
done
```

---

## 安全说明

- Bot 令牌以 `qbot_` 开头，在管理后台「Bot 运维」页面签发
- 配置文件 `~/.nuim/config.json` 权限为 600（仅所有者可读写）
- `nuim update` 默认进行 SHA256 校验，防止二进制篡改
- user_token 为 JWT，有过期时间，支持 refresh token 自动续期
