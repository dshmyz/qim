# AI 管控与审批流设计文档

> 日期: 2026-04-28
> 状态: 待实现
> 关联计划: /docs/superpowers/plans/

## 概述

为 QIM 即时通讯系统的 AI 功能建立完整的管控体系，实现**统一 AI 助手 + 审批制自建 Bot** 的混合模式。管理员可管控 AI 功能启用范围、审批用户创建的 Bot、管理使用配额；用户可使用系统 Bot、模板 Bot，也可提交自建 Bot 申请。

---

## 核心概念

### Bot 分类

| 类型 | 创建者 | 审批要求 | 可见范围 |
|------|--------|----------|----------|
| 系统 Bot | 管理员在后台创建 | 无需审批 | 所有用户 |
| 模板 Bot | 管理员在后台配置模板 | 无需审批 | 所有用户 |
| 用户自建 Bot | 用户在前端创建 | 需管理员审批 | 审批通过后对所有用户可见 |

### 用户角色

- **管理员**：管理 AI 助手、审批 Bot、配置系统参数、查看审计日志
- **普通用户**：使用 AI 助手、创建 Bot（需审批）、管理自己的 Bot

---

## 模块设计

### 模块一：管理员审批面板

#### 1.1 页面结构

在现有 `qim-admin/src/views/AIAssistant.vue` 中增加 Tab 切换：

- **Tab 1: AI 助手管理** — 现有功能，管理系统 Bot（名称、头像、描述、提示词、启停）
- **Tab 2: Bot 审批** — 新增，处理用户提交的自建 Bot 申请

#### 1.2 审批列表

表格列：

| 列名 | 数据字段 | 说明 |
|------|----------|------|
| 申请人 | `creator_name`, `creator_avatar` | 显示用户名和头像 |
| Bot 名称 | `name`, `avatar` | Bot 名称和头像 |
| Bot 类型 | `type` | AI 机器人 / 自定义机器人 |
| 描述 | `description` | 简短描述 |
| 已创建数 | `creator_bot_count` | 该用户已创建的 Bot 总数 |
| 创建时间 | `created_at` | 提交时间 |
| 操作 | — | 查看提示词、通过、拒绝 |

#### 1.3 审批操作

- **查看提示词**：弹窗显示系统提示词、AI 提供商、模型配置等详情
- **通过**：调用 `PATCH /api/v1/admin/bot-approvals/{id}/approve`，Bot 状态变为 `active`，`approval_status` 变为 `approved`
- **拒绝**：调用 `PATCH /api/v1/admin/bot-approvals/{id}/reject`，可选填写拒绝原因，`approval_status` 变为 `rejected`，`reject_reason` 写入原因

#### 1.4 筛选与分页

- 筛选：全部 / 待审批 / 已通过 / 已拒绝
- 分页：与现有 AI 助手列表一致，支持 10/20/50/100 条

---

### 模块二：用户创建 Bot 增强

#### 2.1 创建流程

用户点击"创建机器人"后，先选择创建方式：

```
┌──────────────────────────────────────┐
│  创建机器人                           │
├──────────────────────────────────────┤
│  ● 使用模板（推荐）                   │
│    从管理员配置的模板中选择，直接可用  │
│                                      │
│  ○ 自定义机器人                       │
│    配置自己的 Prompt 和模型，需审批   │
└──────────────────────────────────────┘
```

#### 2.2 模板选择

选择"使用模板"后，展示模板列表：

- 调用 `GET /api/v1/bots/templates` 获取管理员配置的模板
- 每个模板展示：名称、图标、描述
- 选择模板后直接创建，Bot 状态为 `active`，`is_template = true`

#### 2.3 自定义创建

选择"自定义机器人"后，进入现有表单：

- 机器人名称（必填）
- 描述（必填）
- 机器人类型：AI 机器人 / 自定义机器人
- AI 提供商选择（当类型为 AI 时）
- 自定义模型地址（当提供商为自定义时）
- 自定义 Webhook 地址（当类型为自定义时）
- 头像 URL（可选）

提交后状态为 `pending`，等待管理员审批。

#### 2.4 数量限制

- 系统配置 `max_bots_per_user`（默认 5）控制每个用户最多可创建的 Bot 数量
- 创建前调用 `GET /api/v1/bots/my-count` 检查
- 达到上限时按钮置灰，提示"已达到创建上限（X个），如需更多请联系管理员"
- 后端同样做校验，防止绕过前端
- 管理员不受数量限制

---

### 模块三：用户侧"我的 Bot"

#### 3.1 入口

在 AI 助手页面增加 Tab 导航：

```
[ 可用机器人 ] [ 我的机器人 ] [ 创建机器人 ]
```

#### 3.2 我的机器人列表

展示用户自己创建的所有 Bot：

- **已通过**：正常使用，显示"可用"标签
- **待审批**：灰色不可用，显示"待审批"标签
- **已拒绝**：显示"已拒绝"标签 + 拒绝原因

支持操作：
- 查看/编辑（仅待审批和已拒绝状态可编辑）
- 删除（释放配额）

---

### 模块四：管理员 AI 管控

#### 4.1 AI 功能全局开关

在系统配置页面增加 AI 相关配置：

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `ai_enabled` | bool | true | 全局 AI 功能开关 |
| `max_bots_per_user` | int | 5 | 每个用户最多可创建的 Bot 数量 |
| `ai_allowed_roles` | string[] | ["all"] | 允许使用 AI 的角色列表 |
| `default_daily_limit` | int | 0 | 默认每日调用次数限制（0=不限制） |

#### 4.2 用户级 AI 控制

在用户管理页面，每个用户增加 AI 相关操作：

- 启用/禁用该用户的 AI 功能
- 设置该用户的每日调用次数限制（覆盖全局默认值）

#### 4.3 AI 使用审计

新增审计日志页面，记录：

- 用户 ID、用户名
- 使用的 Bot ID、Bot 名称
- 调用时间
- 调用类型（聊天/运维模式）
- 消息内容摘要（前 50 字符）

支持按用户、时间范围、Bot 筛选。

---

## 数据库变更

### 5.1 `bots` 表新增字段

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `approval_status` | varchar(20) | 'approved' | 审批状态：pending / approved / rejected |
| `creator_id` | uint | 0 | 创建者用户 ID（0=系统创建） |
| `creator_name` | varchar(100) | '' | 创建者用户名 |
| `reject_reason` | text | NULL | 拒绝原因 |
| `is_template` | bool | false | 是否为模板 Bot |

### 5.2 `ai_configs` 表新增字段

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `ai_enabled` | bool | true | 该用户是否可以使用 AI 功能 |
| `daily_limit` | int | 0 | 每日调用次数限制（0=不限制） |

### 5.3 新增 `system_configs` 配置项

在系统配置表中增加：

- `max_bots_per_user` (int, default=5)
- `ai_enabled` (bool, default=true)
- `ai_allowed_roles` (json, default=["all"])
- `default_daily_limit` (int, default=0)

### 5.4 新增 `ai_usage_logs` 表（审计日志）

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | uint | 主键 |
| `user_id` | uint | 用户 ID |
| `bot_id` | uint | Bot ID |
| `message_preview` | varchar(100) | 消息内容摘要 |
| `call_type` | varchar(20) | 调用类型 |
| `created_at` | timestamp | 调用时间 |

---

## API 设计

### 6.1 用户侧 API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/bots` | 获取可用 Bot 列表（系统 + 已审批 + 模板） |
| GET | `/api/v1/bots/templates` | 获取模板 Bot 列表 |
| GET | `/api/v1/bots/my` | 获取我创建的 Bot 列表 |
| GET | `/api/v1/bots/my-count` | 获取我已创建的 Bot 数量 |
| POST | `/api/v1/bots` | 创建 Bot |
| PUT | `/api/v1/bots/{id}` | 更新我的 Bot |
| DELETE | `/api/v1/bots/{id}` | 删除我的 Bot |

### 6.2 管理员 API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/admin/bot-approvals` | 获取待审批 Bot 列表 |
| PATCH | `/api/v1/admin/bot-approvals/{id}/approve` | 通过 Bot 申请 |
| PATCH | `/api/v1/admin/bot-approvals/{id}/reject` | 拒绝 Bot 申请 |
| GET | `/api/v1/admin/ai-usage-logs` | 获取 AI 使用审计日志 |
| PATCH | `/api/v1/admin/users/{id}/ai-config` | 设置用户 AI 配置 |

### 6.3 系统配置 API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/admin/system-config` | 获取系统配置（含 AI 相关） |
| PUT | `/api/v1/admin/system-config` | 更新系统配置（含 AI 相关） |

---

## 前端组件结构

### 7.1 Admin 端

```
qim-admin/src/views/
  AIAssistant.vue          # 改造：增加 Tab（AI 助手管理 / Bot 审批）
    ├── AIAssistantList    # 现有：AI 助手列表（提取为子组件）
    └── BotApprovalPanel   # 新增：Bot 审批面板

qim-admin/src/api/
  aiBots.ts                # 改造：增加审批相关 API
```

### 7.2 Client 端

```
qim-client/src/components/apps/
  AIAssistantApp.vue       # 改造：增加 Tab 导航和模板选择
    ├── BotSelection       # 提取：可用机器人列表
    ├── MyBotsPanel        # 新增：我的机器人面板
    ├── CreateBotModal     # 提取：创建机器人模态框
    │   ├── TemplateSelector   # 新增：模板选择器
    │   └── CustomBotForm      # 提取：自定义 Bot 表单

qim-client/src/composables/
  useBots.ts               # 新增：Bot 相关逻辑（创建、审批状态查询等）
```

---

## 数据流

### 8.1 Bot 创建与审批流

```
用户选择创建方式
  ├─ 使用模板 → GET /api/v1/bots/templates → 选择模板 → POST /api/v1/bots (is_template=true) → 直接可用
  └─ 自定义 → POST /api/v1/bots (approval_status=pending) → 等待审批
                ↓
         管理员审批面板收到通知
                ↓
         管理员审核
           ├─ 通过 → PATCH /api/v1/admin/bot-approvals/{id}/approve → 对用户可见
           └─ 拒绝 → PATCH /api/v1/admin/bot-approvals/{id}/reject → 通知用户（含拒绝原因）
```

### 8.2 Bot 列表查询流

```
GET /api/v1/bots
  ├─ 返回所有系统 Bot (creator_id=0, approval_status=approved)
  ├─ 返回所有模板 Bot (is_template=true)
  └─ 返回所有已审批通过的自建 Bot (approval_status=approved)
```

### 8.3 用户 AI 功能校验流

```
用户尝试使用 AI
  ↓
GET /api/v1/users/me/ai-config
  ├─ ai_enabled=false → 提示"AI 功能已被禁用"
  ├─ daily_limit>0 且今日已达上限 → 提示"今日 AI 调用次数已达上限"
  └─ 校验通过 → 正常使用
```

---

## 错误处理

| 场景 | 前端提示 | 后端行为 |
|------|----------|----------|
| 创建数量超限 | "已达到创建上限，请联系管理员" | 返回 400 + 错误码 `BOT_LIMIT_EXCEEDED` |
| AI 功能被禁用 | "AI 功能暂不可用" | 返回 403 + 错误码 `AI_DISABLED` |
| 每日调用超限 | "今日 AI 调用次数已达上限" | 返回 429 + 错误码 `AI_DAILY_LIMIT_EXCEEDED` |
| 审批不存在的 Bot | — | 返回 404 |
| 非管理员操作审批接口 | — | 返回 403 |

---

## 安全考虑

1. **权限控制**：所有 `/api/v1/admin/*` 接口需验证管理员角色
2. **Prompt 注入防护**：用户自定义 Prompt 需经过敏感词过滤
3. **数据隔离**：用户只能查看和操作自己创建的 Bot
4. **审计完整性**：审计日志只允许追加，不允许修改和删除
