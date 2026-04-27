# 全场景AI体验优化设计方案

**日期**: 2026-04-27
**状态**: 设计完成,待审查
**范围**: 群聊AI增强、单聊AI增强、全局AI功能、AI交互优化

---

## 概述

基于现有AI基础设施(多AI提供商、MCP工具调用、SmartReplyEngine),全面提升QIM即时通讯系统的AI用户体验,覆盖群聊、单聊、全局搜索三大场景。

---

## 模块一: 群聊AI增强

### 1.1 @AI触发回复

**功能**: 群聊中@AI助手,触发AI智能回复

**实现方案**:
- 前端: 监听消息内容中的`@AI`或`@助手`模式
- 后端: 在`smart_reply_handler.go`中新增`handleAIMention()`逻辑
- 支持自定义AI助手名称(群设置中配置)

**数据流**:
```
用户发送"@AI 今天天气如何" 
→ 消息WebSocket推送到服务器
→ SmartReplyEngine检测到@AI模式
→ 调用AIService.GetCompletionWithTools()
→ 流式返回AI回复
→ 前端渲染AI消息(带AI标识)
```

### 1.2 智能自动回复

**功能**: 基于现有SmartReplyEngine,AI判断是否需要自动参与群聊讨论

**优化点**:
- 意图检测增强: 识别问题、求助、决策讨论等高价值场景
- 回复策略配置: 群设置中可配置AI回复频率(始终/仅被@/智能判断/关闭)
- 防刷屏机制: 同一话题5分钟内只回复一次

### 1.3 AI消息特殊标识

**功能**: AI消息带明显的AI标签和专属样式

**UI设计**:
- 消息左侧显示AI徽章(🤖 AI Assistant)
- 消息背景使用渐变色(区别于普通用户)
- 消息右上角显示"由AI生成"小标签
- hover时显示"复制"、"重新生成"等操作按钮

**组件**: `AIMessageBadge`

### 1.4 群聊日报优化

**功能**: 现有日报功能UI优化

**改进**:
- 支持手动触发"生成今日总结"
- 日报消息可折叠/展开
- 支持导出为Markdown/PDF
- 日报消息增加交互: 点击待办事项可标记完成

---

## 模块二: 单聊AI增强

### 2.1 快捷指令按钮

**功能**: 输入框旁固定AI快捷操作栏

**预设指令**:
- 📝 总结对话 - 总结当前会话内容
- 🌐 翻译 - 翻译选中文本(支持多语言)
- ✍️ 改写 - 改写输入框中的文本
- ✨ 润色 - 润色文本语气和表达
- 🔍 代码审查 - 审查代码片段

**UI设计**:
- 固定在MessageInput组件上方
- 横向滚动按钮栏(移动端自适应)
- 按钮hover显示功能说明tooltip
- 支持用户自定义添加/删除快捷指令

**组件**: `AIQuickActions`、`AIQuickActionItem`

### 2.2 上下文记忆

**功能**: AI记住当前会话的对话历史

**实现**:
- 前端维护`opsMessagesHistory`数组(现有代码已有基础)
- 后端通过`messages`参数传递历史上下文
- 支持"清空上下文"按钮
- 显示当前上下文消息数/token估算

**组件**: `AIContextBar`

### 2.3 AI助手快捷入口

**功能**: 从现有AIAssistantApp优化为可嵌入任意会话

**改进**:
- AI助手不再独立页面,而是作为聊天窗口的一个模式
- 聊天窗口header增加"切换AI助手"按钮
- AI模式与普通聊天模式平滑切换

---

## 模块三: 全局AI功能

### 3.1 会话智能摘要

**功能**: 一键生成长会话/群聊摘要

**触发方式**:
- 聊天窗口顶部"生成摘要"按钮
- 右键会话列表项
- 快捷键 `Ctrl+Shift+S`

**摘要内容结构**:
```
📋 会话摘要 - [会话名称]
⏰ 时间范围: [选择的时间段]

🔥 核心话题
1. 话题一 (讨论热度: 高)
2. 话题二 (讨论热度: 中)

✅ 重要决策
- 决策一 (决策人: XXX)
- 决策二 (决策人: XXX)

📌 待办事项
- [ ] 待办一 (负责人: XXX)
- [ ] 待办二 (负责人: XXX)

💬 关键发言
- [用户A]: 重要观点摘要
- [用户B]: 重要观点摘要
```

**UI组件**: `AISummaryPanel`

### 3.2 智能消息搜索

**功能**: 语义搜索,理解用户意图

**搜索能力**:
- 自然语言: "上周关于项目的讨论"、"张三说过的方案"
- 多条件组合: 发送者 + 时间范围 + 关键词
- 搜索结果按相关性排序
- 高亮匹配内容
- 点击结果直接跳转到消息位置

**UI组件**: `AISearchInput`、`AISearchResults`

**后端API**: `POST /api/v1/ai/search`

---

## 模块四: AI交互与错误处理

### 4.1 右键菜单AI操作

**功能**: 消息右键菜单新增AI操作

**菜单项**:
- 🤖 AI总结这条消息
- 🌐 翻译为中文
- ✍️ 改写文本
- ✨ 润色表达
- 🔍 搜索相关内容

**支持多选消息后统一处理**

### 4.2 快捷键支持

| 快捷键 | 功能 |
|--------|------|
| `Ctrl+K` / `Cmd+K` | 全局唤起AI快捷面板 |
| `Ctrl+Shift+S` | 快速总结当前会话 |
| `Esc` | 关闭AI面板 |

**组件**: `AIKeyboardShortcuts`

### 4.3 错误处理与降级

**原则**: AI功能应该是增强,不是阻碍

**具体措施**:
- AI服务不可用: 显示"AI服务暂时不可用,请稍后再试"(非技术错误)
- 请求超时: 自动重试3次,指数退避(1s, 2s, 4s)
- 流式输出中断: 显示"生成中断,点击重试"
- Token超限: 自动截断并提示"内容过长,已截断前5000字符"
- 所有AI操作支持取消按钮

**组件**: `AIErrorBoundary`

### 4.4 加载状态优化

**改进**:
- AI处理中显示阶段提示: "正在分析中..." → "正在生成回复..."
- 长时间处理显示进度条(估算)
- 支持"取消"按钮中断请求
- 完成后显示"生成完成"短暂提示

**组件**: `AILoadingIndicator`

---

## 技术架构

### 前端组件清单

| 组件名 | 文件路径 | 功能 |
|--------|----------|------|
| AIMessageBadge | `components/ai/AIMessageBadge.vue` | AI消息标识 |
| AIQuickActions | `components/ai/AIQuickActions.vue` | 快捷指令栏 |
| AIQuickActionItem | `components/ai/AIQuickActionItem.vue` | 单个快捷按钮 |
| AIContextBar | `components/ai/AIContextBar.vue` | 上下文状态条 |
| AISummaryPanel | `components/ai/AISummaryPanel.vue` | 智能摘要面板 |
| AISearchInput | `components/ai/AISearchInput.vue` | 智能搜索输入 |
| AISearchResults | `components/ai/AISearchResults.vue` | 搜索结果展示 |
| GlobalAIButton | `components/ai/GlobalAIButton.vue` | 全局AI入口 |
| AIMessageContextMenu | `components/ai/AIMessageContextMenu.vue` | AI右键菜单 |
| AIKeyboardShortcuts | `composables/useAIKeyboardShortcuts.ts` | 快捷键管理 |
| AIErrorBoundary | `components/ai/AIErrorBoundary.vue` | 错误边界 |
| AILoadingIndicator | `components/ai/AILoadingIndicator.vue` | 加载指示器 |
| GroupAIPanel | `components/ai/GroupAIPanel.vue` | 群聊AI设置 |

### 后端API清单

| 端点 | 方法 | 功能 |
|------|------|------|
| `/api/v1/ai/completion/stream` | POST | 流式AI完成(已有) |
| `/api/v1/ai/summary` | POST | 生成会话摘要 |
| `/api/v1/ai/search` | POST | 语义搜索消息 |
| `/api/v1/ai/translate` | POST | 翻译文本 |
| `/api/v1/ai/rewrite` | POST | 改写文本 |
| `/api/v1/ai/polish` | POST | 润色文本 |
| `/api/v1/conversations/{id}/ai-settings` | PUT | 更新群聊AI设置 |

### 数据模型变更

**Message表字段修改**:
- `Content`字段: 从`type:text`(64KB)改为`type:mediumtext`(16MB) — 支持长AI回复存储

**Group表新增字段**:
- `ai_enabled` (已有) - 是否启用AI
- `ai_reply_mode` - AI回复模式(always/mention_only/smart/off)
- `ai_assistant_name` - AI助手名称(默认"AI助手")

**Conversation表新增字段**:
- `ai_context_messages` - AI上下文消息数(默认10)

---

## AI输出长度控制方案

### 问题背景

AI回复可能非常长(群聊日报5-10KB,代码审查/翻译可能更长)。现有`Message.Content`为`TEXT`类型(64KB限制),且`AIService`仅过滤输入内容,未限制输出长度,存在以下风险:
1. 超长AI回复可能导致数据库保存失败
2. 前端渲染大量内容影响性能
3. 用户阅读体验差

### 三层防护方案

#### 第一层: 数据库升级

将`Message.Content`字段从`TEXT`升级为`MEDIUMTEXT`(16MB),确保极端情况下不会因超长内容导致保存失败:

```go
// model/model.go - Message结构体
Content string `json:"content" gorm:"type:mediumtext;not null"` // 从 text 改为 mediumtext
```

同时需要创建数据库迁移脚本:

```sql
ALTER TABLE messages MODIFY COLUMN content MEDIUMTEXT NOT NULL;
```

#### 第二层: AI输出长度控制

在`AIService`中新增`filterOutput`方法,根据消息类型动态限制输出长度:

```go
// ai/ai_service.go
func (s *AIService) filterOutput(content string, msgType string) string {
    limits := map[string]int{
        "ai_reply":     3000,  // 普通AI回复
        "ai_summary":   5000,  // 会话摘要
        "ai_translate": 2000,  // 翻译
        "ai_rewrite":   2000,  // 改写
        "ai_polish":    2000,  // 润色
        "ai_daily":     8000,  // 群聊日报
    }
    
    limit, ok := limits[msgType]
    if !ok {
        limit = 3000 // 默认3000字符
    }
    
    if len(content) > limit {
        return content[:limit] + "\n\n---\n*内容过长已截断,完整内容可导出查看*"
    }
    return content
}
```

**调用时机**: 在AI回复保存到数据库之前,消息Handler中调用:

```go
// handler/smart_reply_handler.go
reply, err := e.aiService.GetCompletion(messages)
if err != nil {
    // 错误处理
}

// 截断输出
reply = e.aiService.filterOutput(reply, "ai_reply")

// 保存到数据库
msg := model.Message{
    Content: reply,
    // ...
}
```

#### 第三层: 前端UX优化

长AI消息折叠显示,默认只显示前3行,支持展开/收起:

**UI设计**:
```
┌─────────────────────────────────────────────────┐
│ 🤖 AI 助手                              [收起] │
├─────────────────────────────────────────────────┤
│ 今天团队讨论了以下三个主要议题：                  │
│ 1. 项目进度更新 - 前端已完成80%...              │
│ 2. 技术方案评审 - 推荐使用Vue 3...              │
│ ...                                              │
│                                                  │
│ [展开全部] (共5,234字符)                        │
└─────────────────────────────────────────────────┘
```

**展开后**:
```
┌─────────────────────────────────────────────────┐
│ 🤖 AI 助手                              [收起] │
├─────────────────────────────────────────────────┤
│ (完整内容...)                                    │
│ ...                                              │
│ ---                                              │
│ *内容过长已截断,完整内容可导出查看*             │
│                                                  │
│ [导出为Markdown] [复制完整内容]                 │
└─────────────────────────────────────────────────┘
```

**组件实现要点**:
```vue
<!-- components/ai/AIMessageContent.vue -->
<template>
  <div class="ai-message-content">
    <div v-if="!isExpanded" class="preview">
      {{ truncatedContent }}
    </div>
    <div v-else class="full-content" v-html="renderMarkdown(content)"></div>
    
    <div v-if="isTruncated" class="ai-message-footer">
      <button v-if="!isExpanded" @click="isExpanded = true">
        展开全部 (共{{ contentLength }}字符)
      </button>
      <div v-else class="ai-actions">
        <button @click="exportMarkdown">导出为Markdown</button>
        <button @click="copyContent">复制完整内容</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

const props = defineProps<{
  content: string
  maxLength?: number
}>()

const isExpanded = ref(false)
const previewLines = 3

const truncatedContent = computed(() => {
  const lines = props.content.split('\n').slice(0, previewLines)
  return lines.join('\n') + '...'
})

const isTruncated = computed(() => {
  return props.content.length > (props.maxLength || 500)
})

const contentLength = computed(() => props.content.length)
</script>
```

**样式要点**:
- 预览模式: 固定高度 + `overflow: hidden`
- 展开模式: 完整渲染,支持内部滚动(如果内容极长)
- 添加平滑过渡动画(`transition: max-height 0.3s ease`)
- 深色/浅色主题适配

### 核心原则

1. **不依赖数据库报错** — 在保存前主动截断并友好提示
2. **多层防护** — 数据库扩容 + 后端限制 + 前端折叠
3. **用户体验优先** — 长内容可展开完整查看,不丢失信息
4. **一致性** — 所有AI消息组件统一使用`AIMessageContent`处理长度

---

## 实现优先级

### Phase 1 (核心功能)
1. @AI触发回复
2. AI消息特殊标识
3. 快捷指令按钮
4. 右键菜单AI操作

### Phase 2 (增强功能)
1. 会话智能摘要
2. 智能消息搜索
3. 上下文记忆优化
4. 错误处理与降级

### Phase 3 (高级功能)
1. 全局AI快捷入口
2. 快捷键支持
3. 群聊日报优化
4. 自定义快捷指令

---

## 风险评估

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| AI API调用成本高 | 中 | 缓存常见问答、限制调用频率 |
| 流式输出不稳定 | 中 | 重试机制、降级到非流式 |
| 用户体验不一致 | 低 | 统一设计语言、充分测试 |
| 性能影响 | 低 | 异步处理、懒加载组件 |

---

## 后续迭代方向

1. AI语音交互(语音转文字+AI回复+文字转语音)
2. AI图片生成和理解
3. AI会议纪要自动生成
4. AI智能推荐回复(类似Gmail Smart Reply)
5. 多模态AI(支持图片、文件内容理解)
