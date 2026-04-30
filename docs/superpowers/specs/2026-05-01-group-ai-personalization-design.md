# 群组 AI 助手个性化增强设计

**日期：** 2026-05-01
**状态：** 设计完成，待实现
**目标：** 扩展群组 AI 助手设置，支持人设风格、触发规则细化、知识库绑定等个性化配置

---

## 概述

将现有的群AI设置从4个基础配置项扩展为4个Tab的多模块配置面板，覆盖基础设置、人设风格、触发规则和知识库四大维度。

---

## UI 设计

### Tab 结构

```
┌──────────────────────────────────────┐
│ AI 助手设置                          │
├──────────────────────────────────────┤
│ [基础设置] [人设风格] [触发规则] [知识库] │
├──────────────────────────────────────┤
│                                      │
│  当前选中 Tab 的内容                  │
│                                      │
└──────────────────────────────────────┘
```

### Tab 1：基础设置

| 配置项 | 类型 | 说明 | 后端字段 |
|--------|------|------|----------|
| 启用 AI 助手 | 开关 | 已有 | `ai_enabled` |
| AI 助手名称 | 文本输入 | 已有 | `ai_assistant_name` |
| 上下文消息数 | 数字输入 | 已有 | 前端控制 |

### Tab 2：人设风格

| 配置项 | 类型 | 说明 | 后端字段 |
|--------|------|------|----------|
| 预设人设 | 卡片单选 | 专业严谨/轻松幽默/简洁高效/贴心助手/技术专家 | `ai_personality` |
| 自定义提示词 | 文本域 | 可输入自定义 system prompt，覆盖预设 | `ai_custom_prompt` |
| 回复语言 | 下拉选择 | 自动/中文/English/日本語 | `ai_language` |
| 回复长度 | 卡片单选 | 简短(1-2句)/适中(3-5句)/详细(不限) | `ai_max_length` |

### Tab 3：触发规则

| 配置项 | 类型 | 说明 | 后端字段 |
|--------|------|------|----------|
| 回复模式 | 下拉选择 | 仅被@时回复/智能判断/始终回复/关闭 | `ai_reply_mode` |
| @回复方式 | 开关 | 直接回复 vs @提问者后回复 | `ai_mention_reply_mode` |
| 防刷屏间隔 | 下拉选择 | 关闭/3分钟/5分钟/10分钟/15分钟 | `ai_anti_spam_interval` |
| 触发关键词 | 标签输入 | 指定关键词，只有包含这些关键词时才触发 | `ai_trigger_keywords` |

### Tab 4：知识库

| 配置项 | 类型 | 说明 | 后端字段 |
|--------|------|------|----------|
| 绑定文件列表 | 文件选择列表 | 从文件箱选择文档作为AI参考 | `group_documents` 关联表 |
| 绑定笔记列表 | 笔记选择列表 | 关联笔记作为知识库 | 复用现有笔记关联 |
| 自动学习 | 开关 | 是否从聊天记录中提取知识点（暂不实现） | `ai_auto_learn` |

---

## 数据模型变更

### Group 表新增字段

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `ai_personality` | string | `professional` | 人设：professional/casual/concise/friendly/technical |
| `ai_custom_prompt` | text | 空 | 自定义 system prompt |
| `ai_language` | string | `auto` | 回复语言：auto/zh/en/ja |
| `ai_max_length` | string | `medium` | 回复长度：short/medium/long |
| `ai_mention_reply_mode` | string | `mention` | @回复方式：direct/mention |
| `ai_anti_spam_interval` | int | 5 | 防刷屏间隔（分钟），0表示关闭 |
| `ai_trigger_keywords` | string | 空 | 触发关键词，逗号分隔 |
| `ai_auto_learn` | bool | false | 自动学习（预留） |

### 新增关联表：group_documents

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | uint | 主键 |
| `group_id` | uint | 群ID |
| `file_id` | uint | 文件ID |
| `created_at` | time | 创建时间 |

---

## 后端变更

### 1. 模型层
- `model.Group` 新增上述字段

### 2. API 层
- 修改 `UpdateGroupAISettings` handler，支持新增字段
- 新增 `GET /conversations/:id/ai-documents` 获取群绑定的文档列表
- 新增 `POST /conversations/:id/ai-documents` 绑定文档
- 新增 `DELETE /conversations/:id/ai-documents/:file_id` 解绑文档

### 3. 智能回复引擎
- 修改 `SmartReplyEngine`，在 system prompt 中注入人设、语言、长度等配置
- 修改 `handleAIMention`，支持防刷屏间隔和触发关键词过滤
- 修改 `BuildSystemPrompt`，将人设、语言、长度配置注入 prompt

### 4. Prompt 构建器
- 修改 `buildRules()` 方法，根据配置动态生成回复规则
- 新增方法注入自定义 prompt

---

## 前端变更

### 1. GroupAIPanel.vue 重构
- 改为 Tab 结构布局
- 四个 Tab 分别对应基础设置、人设风格、触发规则、知识库
- 每个 Tab 内部使用独立的子组件保持文件简洁

### 2. 新增子组件
- `AIPersonaSettings.vue` — 人设风格 Tab
- `AITriggerSettings.vue` — 触发规则 Tab
- `AIKnowledgeSettings.vue` — 知识库 Tab

### 3. 状态管理
- 使用 `reactive` 管理所有 AI 设置状态
- 统一 `saveSettings` 方法，合并所有 Tab 的配置一次性提交

---

## 执行顺序

```
Phase 1: 数据模型 + API（后端）
├── 1. Group 模型新增字段
├── 2. 迁移脚本/自动迁移
├── 3. 更新 UpdateGroupAISettings API
├── 4. 新增文档绑定 API（3个端点）
│
Phase 2: 前端组件重构
├── 5. 创建 AI 设置子组件（3个）
├── 6. 重构 GroupAIPanel.vue 为 Tab 结构
├── 7. 更新 ChatHeader/ChatWindow 传递新 props
│
Phase 3: 智能回复引擎增强
├── 8. Prompt 构建器注入人设/语言/长度
├── 9. SmartReplyEngine 支持防刷屏和触发关键词
│
Phase 4: 联调测试
├── 10. 前端设置保存 → 后端存储 → 前端读取验证
└── 11. AI 回复使用新配置验证
```
