---
title: 功能介绍
order: 1
---

# NUIM 功能介绍

NUIM 是一款集成 AI 能力的现代化企业通讯解决方案，为团队提供一站式协作平台。

---

## 消息通信

### 基础消息

- **发送文本消息**：向指定会话发送文本内容
- **消息搜索**：按关键词、时间范围、发送者搜索历史消息
- **消息撤回**：撤回已发送的消息（限时效内）
- **@ 提及**：@指定成员或 @全体成员

### 会话管理

- **私聊**：一对一直接沟通
- **群聊**：多人协作空间，支持群公告、群文件
- **频道**：围绕特定主题的开放讨论空间
- **会话置顶**：重要会话置顶显示
- **已读回执**：查看消息阅读状态

```bash
# 发送消息示例
nuim send --to "张三" --content "你好"

# 搜索历史消息
nuim messages search --keyword "项目计划"
```

---

## 自定义机器人

NUIM 支持创建自定义机器人，用于自动化回复、智能助手等场景。

### 机器人类型

| 类型 | 说明 |
|------|------|
| 系统助手（system） | 系统内置机器人，提供帮助信息等基础功能 |
| AI 助手（assistant） | 基于大模型的智能助手，可自定义系统提示词 |
| 自定义机器人（custom） | 连接第三方服务或本地模型的自定义机器人 |
| 群聊助手（group_assistant） | 群聊级别的 AI 助手，关联指定群聊 |

### 配置入口

在管理后台的 **机器人管理** 页面可创建和管理机器人。

### 配置参数

创建机器人时支持以下配置项：

| 参数 | 类型 | 说明 |
|------|------|------|
| `name` | string | 机器人名称（必填） |
| `description` | string | 机器人描述 |
| `type` | string | 机器人类型：`assistant` / `custom` / `system` / `group_assistant` |
| `avatar` | string | 机器人头像 URL |
| `config` | JSON | 机器人配置，不同类型配置不同 |
| `use_system_config` | bool | 是否使用系统配置（默认 true） |
| `user_config_id` | uint | 关联的用户 AI 配置 ID |
| `is_active` | bool | 是否启用 |

#### AI 助手配置

在「机器人管理」页面创建时，选择 **AI 助手** 类型，系统会自动关联 AI 配置。可在个人设置的 **AI 配置** 页面管理系统提示词和模型参数。

#### 自定义机器人配置

支持通过 `custom_model_url` 连接第三方模型服务：

```json
{
  "type": "custom",
  "custom_model_url": "http://localhost:8000/chat"
}
```

### 模板机器人

系统预置了多个模板机器人，可直接在「机器人管理」页面一键创建：

- **AI 助手**：通用智能问答
- **代码助手**：编程辅助
- **翻译助手**：多语言翻译
- **系统助手**：系统帮助

---

## 分身（Avatar）

分身是 NUIM 的核心 AI 能力，可以让 AI 学习你的风格，在你离线时代为回复。

### 功能概述

- **自动学习人设**：分析历史对话，学习你的表达风格和角色定位
- **智能触发**：根据触发规则自动在会话中代为回复
- **知识范围**：控制分身可以回答的知识领域
- **回复策略**：仿真人延迟，避免机械感

### 配置入口

在个人设置的 **分身设置** 页面可开启和配置分身。

### 配置参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `enabled` | bool | 是否启用分身 |
| `name` | string | 分身名称，默认「我的分身」 |
| `activate_by_default` | bool | 是否在所有会话默认激活 |
| `auto_learned_persona` | text | 自动学习生成的人设（系统维护） |
| `custom_persona_addon` | text | 自定义人设补充，如「回复简洁直接」 |
| `knowledge_scope` | object | 知识范围配置 |
| `trigger_rules` | object | 触发规则配置 |
| `reply_strategy` | object | 回复策略配置 |
| `model_config_id` | uint | 使用的 AI 模型配置 |
| `takeover_cooldown` | int | 接管冷却时间（分钟），默认 10 |

### 触发规则

控制分身何时自动回复：

| 模式 | 说明 |
|------|------|
| `mention` | 仅在被 @时回复（群聊默认） |
| `offline` | 离线时自动回复 |
| `keyword` | 命中关键词时回复 |
| `all` | 所有消息都回复 |
| `smart` | 智能判断是否回复 |

#### 触发规则示例

```json
{
  "mode": "mention",
  "keywords": ["紧急", "urgent"],
  "timeRanges": [
    { "dayOfWeek": [1, 2, 3, 4, 5], "startHour": 18, "endHour": 9 }
  ],
  "excludedConversations": [5, 12]
}
```

### 回复策略

控制分身的回复行为：

| 参数 | 说明 |
|------|------|
| `replyDelay` | 仿真人打字延迟（秒） |
| `maxReplyLength` | 最大回复长度：`short` / `medium` / `long` |
| `groupReplyTarget` | 群聊回复目标：`group`（群内）/ `private`（私聊） |
| `confidenceThreshold` | 置信度阈值（0-1），低于阈值不回复 |
| `disclaimerStyle` | 免责声明样式：`badge` / `footer` / `both` |
| `replyOutOfScope` | 是否回复知识范围外的消息 |

### 会话级控制

在会话详情页可独立控制分身开关：

- **开启/关闭**：在指定会话中启用或禁用分身
- **临时接管**：让分身临时接管会话，指定时长后自动恢复

---

## 审批工作流

系统内置审批功能，用于敏感操作的审核：

- **分身审批**：开启分身功能需管理员审批
- **机器人审批**：创建自定义机器人需管理员审批
- **频道审批**：创建公开频道需管理员审批

审批类型包括：`avatar`（分身）、`bot`（机器人）、`channel`（频道）、`group_ai`（群聊 AI 助手）。

---

## 组织架构

### 成员管理

- **部门结构**：支持多级部门组织
- **成员信息**：查看成员名片、在线状态

### 认证方式

支持多种身份认证：
- **OAuth/OIDC/CAS**：对接企业统一认证
- **多因素认证（TOTP）**：两步验证

---

## AI Agent 集成

NUIM 提供 MCP（Model Context Protocol）协议支持，可与主流 AI Agent 框架集成。

### 接入方式

| 协议 | 说明 |
|------|------|
| `stdio` | 本地 AI Agent（Claude Code / Cursor）通过标准输入输出接入 |
| `streamable HTTP` | 远程部署的 MCP 客户端通过 HTTP 接入 |

```bash
# stdio 模式（本地）
<mcp二进制名> --token qbot_xxx

# HTTP 模式（远程）
<mcp二进制名> --transport http --addr :8082 --server http://localhost:8080
```

更多详情请参考 [CLI 使用指南](/docs/cli) 和 [MCP 接入指南](/docs/mcp)。
