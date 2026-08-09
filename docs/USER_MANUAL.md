# QIM 详细使用手册

> 本文档面向 QIM 的**终端用户**与**开发者/集成方**，把散落在各篇专项文档中的功能用说明统一汇总。
> 内容依据截至 2026-08 的 `qim-server/service`、`qim-client` 与 `docs/` 现状编写。
> 相关专项文档：[CLI 使用指南](./CLI使用指南.md) · [MCP 接入指南](./MCP接入指南.md) ·
> [群助手知识库与记忆对照](./群助手知识库与记忆对照.md)。

---

## 目录

**Part A · 终端用户篇**
- [1. 登录与会话](#1-登录与会话)
- [2. 消息通信](#2-消息通信)
- [3. 应用中心](#3-应用中心)
- [4. AI 助手与群助手](#4-ai-助手与群助手)
- [5. AI 分身 / Avatar](#5-ai-分身--avatar)
- [6. 群文件空间与知识库](#6-群文件空间与知识库)
- [7. 通知与提醒](#7-通知与提醒)

**Part B · 开发/集成篇**
- [8. 机器人与 Bot](#8-机器人与-bot)
- [9. Webhook 接入](#9-webhook-接入)
- [10. QIM CLI](#10-qim-cli)
- [11. MCP Server](#11-mcp-server)
- [12. 外部 Agent 消息闭环](#12-外部-agent-消息闭环)

---

# Part A · 终端用户篇

## 1. 登录与会话

### 1.1 登录

1. 启动应用，输入用户名、密码。
2. 可选：勾选「记住密码」（Electron `safeStorage` 加密存储），下次自动登录。
3. 若后端不在本机，可在登录界面点击「设置服务器地址」修改（默认 `http://localhost:8080`）。
4. 点击「登录」进入主界面。

> 企业账密可开启**双因素认证**（后端为模拟实现，非生产级）。管理员可在管理后台配置。

### 1.2 会话类型

| 类型 | 说明 | 入口 |
|------|------|------|
| **单聊 / 私聊** | 一对一加密沟通 | 左侧栏「组织架构」→ 双击或「发起私聊」 |
| **群聊** | 多人协作空间，支持群公告、群文件、管理员 | 左侧栏「+」→「创建群聊」 |
| **讨论组** | 扁平化讨论，所有成员平等参与 | 「+」→「创建讨论组」 |
| **频道** | 订阅制信息发布，富文本 Markdown 渲染、审批机制 | 「+」→「频道」 |

### 1.3 会话管理

- **置顶**：把重要会话固定在列表顶部。
- **免打扰 / 隐藏**：减少干扰或隐藏会话。
- **消息搜索 / 漫游**：按关键词、日期、类型筛选，多端同步。

---

## 2. 消息通信

### 2.1 消息类型

| 类型 | 说明 |
|------|------|
| 📝 文本 | URL 自动识别与链接转换 |
| 🖼️ 图片 | 大图预览 |
| 📎 文件 | 上传、下载、另存 |
| 🔗 分享 | 笔记、便签、文件分享到用户/群聊 |
| 📱 小程序卡片 | 内嵌应用卡片 |
| 📰 资讯卡片 | 富文本资讯 |
| 🤖 Bot 卡片 | 带按钮/动作的交互卡片，支持状态回写 |

### 2.2 常用操作

- **引用/回复**：鼠标悬停消息 → 回复，上下文清晰。
- **撤回/编辑**：2 分钟内撤回，撤回后可重新编辑回填。
- **@ 提及**：`@` 指定成员或 `@全体成员`。
- **已读回执**：群聊显示已读人数，单聊显示已读状态。
- **语音/视频**：WebRTC 音视频通话、屏幕共享（注意：禁止对 AI 助手发起屏幕共享）。

---

## 3. 应用中心

左侧栏「应用」进入，包含：

| 应用 | 说明 |
|------|------|
| 📊 统计报表 | 消息趋势、文件分布、任务完成率可视化 |
| 📅 日历 | 月视图，事件管理，支持提醒 |
| 📝 笔记 | Markdown 编辑器，实时预览，可分享 |
| 📌 便签 | 快捷笔记，多色分类 |
| 📁 文件管理 | 个人文件、群文件空间、上传下载、分级权限 |
| ✅ 任务管理 | 看板视图，待办/进行中/已完成 |
| 🤖 AI 助手 | 智能对话机器人 |
| 🔗 短链接 | URL 缩短与管理 |

**分享内容**：在笔记/便签/文件中找到内容 → 点击「分享」→ 选择用户或群聊 → 确认。

---

## 4. AI 助手与群助手

### 4.1 AI 助手（系统）

- 入口：左侧栏「应用」→「AI 助手」→ 选择机器人对话。
- 能力：多模型对话、智能摘要、智能搜索、4xx 快速失败 + 合理重试。

### 4.2 群聊 AI 助手

群聊中可启用 AI 助手代答，支持三种触发方式：

1. **`@AI` 触发**：在群聊中 `@AI` + 提问，群助手优先响应。
2. **关键词门控**：配置关键词，消息命中关键词时自动回复。
3. **代管成员**：群助手可托管指定成员，在该成员被 @ 时代答。

**群级记忆与知识库**：
- **群知识库**：群显式上传的权威文档（`GroupDocumentService`），回答定位"事实是什么"。
- **群记忆**：从群对话自动提炼的共识/约定/偏好（`GroupMemoryService`），回答定位"这个群默认怎么做事"。
- 两者独立存储、刻意隔离。详见 [群助手知识库与记忆对照](./群助手知识库与记忆对照.md)。

### 4.3 帮我回复（草稿模式）

收到消息后可使用「帮我回复」，系统生成草稿，复用分身流式生成能力，可编辑后发送。

---

## 5. AI 分身 / Avatar

分身是 QIM 的核心 AI 能力——让 AI 学习你的风格，在你离线时代为回复。

### 5.1 开启与配置

- 入口：个人设置的「分身设置」。
- 配置项：
  - **人设**：分析历史对话自动学习你的表达风格和角色定位。
  - **触发规则**：根据规则在合适时机代为回复。
  - **知识范围 / 知识库开关**：控制分身可以回答的知识领域。
    - `notes` 开关：是否读取**主人笔记**作为知识来源。
    - `tasks` 开关：是否读取任务作为上下文。
  - **回复策略**：仿真人延迟，避免机械感。
  - **模型**：可绑定自定义模型（门控与生成共用同一模型，详见记忆/模型门控约束）。

### 5.2 分身记忆

- 分身有自己的记忆（`AvatarMemoryService`），从与主人的对话中提炼风格与偏好。
- **三态会话激活**、**置信度门控**、**任务检索**、Notes/Tasks 知识开关、记忆写入路径等治理能力已落地。
- 支持**手动接管**：在分身正要回复时，你可以主动接管会话。

### 5.3 触发决策

群聊/单聊中判断是否由分身代答由 `DecideGroupAIReply` 等触发决策逻辑控制（群助手/分身共用治理框架）。

---

## 6. 群文件空间与知识库

### 6.1 文件空间

- **个人文件空间**：你自己的上传/下载、排序、消息中的文件引用。
- **群文件空间**：群成员共享，支持上传、下载、排序、消息引用。
- **权限体系**：集中式文件权限校验 + 群文件空间 scope 隔离；`scope` 隔离个人空间、群文件面板、历史文件。

### 6.2 群知识库（AI 专用）

- 群知识库面向群助手/分身检索，与普通"群文件"是两套：群知识库走 `GroupDocumentService`（上传 → 解析分块 → 向量化），供 AI 召回。
- 公共文档由管理后台「文档管理」维护，可多版本管理。

---

## 7. 通知与提醒

- **待办提醒**：任务/事件定时触发通知。管理后台「参数配置」可调提醒门槛、重复冷却等。
- **通知中心**：系统聚合通知；点频道消息可跳转。
- **调度能力**：任务、日历、提醒、通知可后台调度触发。

---

# Part B · 开发/集成篇

## 8. 机器人与 Bot

### 8.1 机器人类型

| 类型 | 说明 |
|------|------|
| 系统助手（`system`） | 系统内置机器人 |
| AI 助手（`assistant`） | 基于大模型的智能助手，可自定义系统提示词 |
| 自定义机器人（`custom`） | 连接第三方服务或本地模型 |
| 群聊助手（`group_assistant`） | 群聊级 AI 助手，关联指定群 |

### 8.2 创建机器人

- 入口：**管理后台 → 机器人管理**。
- 关键配置项：`name`（必填）、`description`、`type`、`avatar`、`config`(JSON)、`is_active`。
- **模板机器人**：系统预置 AI 助手、代码助手、翻译助手、系统助手，可一键创建。
- **模型来源**：`use_system_config=true` 用系统默认模型（推荐）；否则可绑定创建者「我的模型配置」`user_config_id` 的自定义 provider。

### 8.3 Bot 会话与配对

- 每个 Bot 对应一个**虚拟用户**与 `bot_token`（格式 `qbot_xxxxxxxxxxxxxxxx`），用于外部调用。
- Bot 与用户发起私聊时建立 1:1 会话配对（`bot_conversations`）。

> ⚠️ **删除机器人必须走后端 `BotService.DeleteBot`**：它会一并清理群内残留的成员记录与 1:1 配对。直接删库会导致群里残留已删 bot、私聊 404。

### 8.4 Bot 消息

- Bot 可发送文本 / Markdown / 图片 / 文件 / **交互卡片**。
- 交互卡片：带按钮，用户点击后卡片动作可**幂等处理**、**状态回写**（重新渲染）。
- Bot 流式回复：先建占位消息再逐段追加（见 [CLI 流式](./CLI使用指南.md) / [MCP 流式](#11-mcp-server)）。

---

## 9. Webhook 接入

`external_webhook` 模式的 Bot 可把用户回复推送到你的 webhook 回调地址。

### 9.1 Bot 配置（`config` JSON）

```json
{
  "mode": "external_webhook",
  "webhook_url": "https://your-agent.example.com/hook",
  "webhook_secret": "your_hmac_secret",
  "use_creator_notes": false
}
```

- `mode: external_webhook` — 外部 agent bot（身份判定依据）。
- `webhook_url` 为空 = **纯 pull 模式**（不推 webhook，靠 `GET /bot/messages` 拉取）。
- `webhook_secret` — HMAC-SHA256 签名密钥（与 `bot_token` 分离）。
- `use_creator_notes` — `internal_ai` 模式下是否读创建者笔记作为知识库。

### 9.2 回调载荷（Payload）

`POST` 到 `webhook_url`，请求头：

| 头 | 说明 |
|----|------|
| `X-QIM-Event` | 事件类型 |
| `X-QIM-Timestamp` | UTC RFC3339 时间戳 |
| `X-QIM-Delivery` | 投递唯一 ID |
| `X-QIM-Signature` | `HMAC-SHA256(body, secret)`（有 secret 时） |

事件（`event`）：
- `bot.message`：用户在 bot 会话中的文本/多媒体回复（默认）。
- `bot.card_action`：用户点击了 agent 发出的卡片按钮。

载荷字段（关键）：

```json
{
  "event": "bot.message",
  "bot_id": 123,
  "thread_id": 456,
  "message_id": 789,
  "user_id": 1,
  "user_nickname": "alice",
  "user_avatar": "",
  "content": "你好，agent",
  "msg_type": "text",
  "timestamp": "2026-08-09T00:00:00Z",
  "delivery_id": "uuid",
  "action_id": "",        // 仅 bot.card_action
  "action_value": "",     // 仅 bot.card_action
  "group_context": ""     // 群聊外部 agent 被 @ 时附带群记忆+知识库
}
```

- `thread_id` 即 QIM `conversation_id`，agent 据此回复同一会话。

### 9.3 可靠投递（Outbox）

- 用户回复 **先落 outbox 表再异步投递**，失败**指数退避重试**（30s → 2m → 10m → 1h）。
- 超过 `MaxAttempts=4` 次进入**死信**（`dead`），保留 `last_error` 供排查。
- 管理后台 **Bot 运维 → Webhook 投递监控** 可查看投递状态，支持**手动重投**（`done` 不可重投）。

---

## 10. QIM CLI

`qim` 是基于 Cobra 的纯 HTTP 命令行客户端，可用于 AI Agent 集成、自动化脚本、运维排查。
**完整命令与示例见 [docs/CLI使用指南.md](./CLI使用指南.md)**，这里给出要点。

### 10.1 安装与配置

```bash
cd qim-server && go build -o qim ./cmd/qim/
qim config set --server http://localhost:8080 --token qbot_your_bot_token
# 可选：以用户身份操作任务/日历/笔记
qim config set --user-token <user_jwt>
```

### 10.2 常用命令速览

```bash
qim send --to alice --content "你好"              # 发文本（自动建会话）
qim send --to alice --type markdown --content - < report.md   # Markdown/文件
qim send --to alice --type card --content '{...}' # 交互卡片
qim messages list --thread alice                   # 拉历史
qim messages poll --thread alice --once            # 等一条新消息（agent 模式）
qim stream-stdin --to alice                        # 管道流式
qim task create --title "修复bug" --assignee alice # 任务
qim note create --title "部署备忘" --content "..." # 笔记（长期记忆）

# 身份
qim login        # 交互式登录（拿 JWT+refresh）
qim whoami
```

> 笔记是**用户级私有数据**，走登录用户 JWT，与 `qbot_xxx` 身份相互独立。

---

## 11. MCP Server

`qim-mcp` 是标准 MCP Server（stdio adapter + Streamable HTTP transport），对外暴露 IM 工具给 Claude/Cursor。
**完整接入步骤、传输模式、安全说明见 [docs/MCP接入指南.md](./MCP接入指南.md)**。

### 11.1 快速开始

1. 构建：`cd qim-server && go build -o qim-mcp ./cmd/qim-mcp/`
2. 取 Bot 令牌：`qbot_xxxxxxxxxxxxxxxx`
3. 在 Claude Code / Cursor 里以 stdio 模式（或 Streamable HTTP 远程模式）接入。

### 11.2 工具列表

| 工具 | 说明 |
|------|------|
| `list_messages` | 列会话消息 |
| `poll_messages` | 轮询新消息 |
| `send_message` | 发送消息 |
| `start_streaming_message` / `append_streaming_chunk` / `finish_streaming_message` | 流式消息 |
| `list_notes` / `get_note` / `create_note` / `update_note` / `search_notes` | 笔记（需 user JWT） |

### 11.3 典型场景

- **场景 1**：`send_message` 发送 → `poll_messages` 等待用户回复。
- **场景 2**：`start_streaming_message` → 逐步 `append_streaming_chunk` → `finish_streaming_message`（AI 实时输出）。
- **场景 3**：`list_messages` 读历史 → `send_message` 回复。

---

## 12. 外部 Agent 消息闭环

外部 Agent（CLI / MCP / Bot webhook）↔ QIM 用户之间形成完整消息闭环：

```
用户发消息 ──► QIM 后端 ──► (webhook 推送给 agent) / (agent pull /bot/messages)
agent 回复 ──► send_message / 流式 ──► QIM 会话 ──► 用户实时看到
```

- **闭环起点**：用户在 bot 会话中回复，或点击 agent 发来的卡片。
- **流式收尾**：agent 可在断线后补齐流式消息；支持**纯 pull 模式**解决 pull-mode agent 消息黑洞。
- **卡片动作**：幂等处理 + 状态回写，保证重复点击不产生重复副作用。

---

## 附：参考文档索引

| 主题 | 文档 |
|------|------|
| 全部功能概览 | [根 README](../README.md) |
| CLI | [CLI使用指南](./CLI使用指南.md) |
| MCP | [MCP接入指南](./MCP接入指南.md) |
| 群助手记忆/知识库 | [群助手知识库与记忆对照](./群助手知识库与记忆对照.md) |
| 后端 API | [qim-server/API.md](../qim-server/API.md) |
| 发布记录 | [docs/releases/](../docs/releases/) |
