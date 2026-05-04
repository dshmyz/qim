# AI 助手模块优化设计文档

## 一、背景与目标

### 1.1 问题概述

qim-client 的 AI 助手模块存在以下问题：

**P0 严重问题（功能不可用）**：

- BotChatView 没有实际的 AI 对话能力
- ModelConfigFormModal 的 emit 用法错误
- handleEditBot 是空操作

**P1 架构问题**：

- API 调用方式不统一（三种不同的 HTTP 调用方式）
- 类型定义重复（Bot 和 Message 在多个文件重复定义）
- 使用 alert/confirm 而非项目共享组件

**功能重复**：

- Markdown 渲染重复（3 处）
- 思考中指示器重复（3 处）
- 复制/导出功能重复（2 处）
- AI 请求处理模式重复（前端 5 处，后端 4 处）

### 1.2 优化目标

1. **功能完整性**：让 Bot 对话功能真正可用
2. **架构一致性**：统一 API 调用、类型定义、组件使用
3. **代码复用**：消除重复代码，提高可维护性
4. **用户体验**：支持流式响应、对话历史持久化

***

## 二、技术方案

### 2.1 Bot 对话机制选择

经过分析，后端已有完整的 Bot 会话机制：

| API 端点                                           | 功能           |
| ------------------------------------------------ | ------------ |
| `POST /api/v1/conversations/bot`                 | 创建或获取 Bot 会话 |
| `GET /api/v1/conversations/:id/messages`         | 获取会话历史消息     |
| `POST /api/v1/conversations/:id/messages/stream` | 发送流式消息       |

**优势**：

- ✅ 自动保存消息历史到数据库
- ✅ 自动关联 Bot 配置
- ✅ 自动加载历史消息作为上下文
- ✅ 支持跨设备同步

**发现的问题**：

1. `StreamMessage` 函数（message\_handler.go:497-510）没有使用 Bot 的系统提示词，需要修复
2. `sender_id = 0` 表示 Bot 消息存在歧义，无法区分不同 Bot

### 2.2 用户类型设计

**问题**：当前 `sender_id = 0` 表示 Bot 消息，存在以下问题：

- 无法区分不同 Bot
- 无法区分 Bot 消息和系统消息
- Preload Sender 时无法加载 Bot 信息

**解决方案**：在用户表添加用户类型字段，为每个 Bot 创建虚拟用户。

```go
type User struct {
    ID           uint
    Username     string
    Nickname     string
    Avatar       string
    Type         string     // 'user' | 'bot' | 'system' | 'api'
    Status       string     // 'active' | 'inactive' | 'suspended'
    // ...
}
```

**用户类型说明**：

| 类型         | 用途     | 示例            |
| ---------- | ------ | ------------- |
| **user**   | 普通用户   | alice、bob     |
| **bot**    | 机器人    | AI助手、翻译官      |
| **system** | 系统用户   | 系统通知、群聊日报     |
| **api**    | API 用户 | 第三方集成、Webhook |

**数据示例**：

| id | type   | username        | nickname | avatar              |
| -- | ------ | --------------- | -------- | ------------------- |
| 1  | user   | alice           | 爱丽丝      | /avatar/1.jpg       |
| 2  | user   | bob             | 鲍勃       | /avatar/2.jpg       |
| 3  | bot    | bot\_assistant  | AI助手     | /bot/assistant.png  |
| 4  | bot    | bot\_translator | 翻译官      | /bot/translator.png |
| 5  | system | system          | 系统       | /system.png         |
| 6  | api    | api\_webhook    | Webhook  | /api.png            |

**说明**：

- ID 使用数据库自增，保证唯一性
- 通过 `type` 字段区分用户类型
- Bot 和用户的 ID 可能混在一起，但通过 `type` 字段可以清晰区分

**各类型使用场景**：

1. **user** - 普通用户
   - 正常的 IM 用户
   - 可以发送消息、加入群组、创建 Bot 等
2. **bot** - 机器人用户
   - AI 对话机器人
   - 群聊 AI 助手
   - 自动回复机器人
3. **system** - 系统用户
   - 系统通知（"系统维护通知"）
   - 群聊日报（当前 `sender_id = 0` 的场景）
   - 审批通知（"您的 Bot 已通过审批"）
4. **api** - API 用户
   - 第三方系统集成
   - Webhook 回调
   - 自动化脚本

**实现逻辑**：

1. 创建 Bot 时同步创建虚拟用户：

```go
// 创建 Bot
bot := model.Bot{Name: "AI助手", Avatar: "/bot/assistant.png", ...}
db.Create(&bot)

// 创建虚拟用户（ID 自动生成）
virtualUser := model.User{
    Username: fmt.Sprintf("bot_%d", bot.ID),
    Nickname: bot.Name,
    Avatar:   bot.Avatar,
    Type:     "bot",
    Status:   "active",
}
db.Create(&virtualUser)  // ID 自动生成

// 更新 Bot 关联虚拟用户
bot.VirtualUserID = virtualUser.ID
db.Save(&bot)
```

1. 保存 Bot 消息时使用虚拟用户 ID：

```go
botReply := model.Message{
    ConversationID: conversationID,
    SenderID:       bot.VirtualUserID,  // 虚拟用户 ID
    Type:           "markdown",
    Content:        reply,
}
```

1. 查询时 Preload 自动加载 Bot 信息：

```go
db.Preload("Sender").Find(&messages)
// Sender 会自动填充虚拟用户信息（nickname、avatar、type）
```

1. 查询时通过 type 字段区分：

```go
// 获取所有真实用户
db.Where("type = ?", "user").Find(&users)

// 获取所有 Bot
db.Where("type = ?", "bot").Find(&bots)

// 统计真实用户数量
db.Model(&User{}).Where("type = ?", "user").Count(&count)
```

**职责划分**：

| 表     | 职责                           |
| ----- | ---------------------------- |
| users | 消息显示（nickname、avatar）、用户类型区分 |
| bots  | Bot 配置管理（系统提示词、审批状态等）        |

**注意事项**：

1. 用户统计排除非真实用户：

```go
db.Model(&User{}).Where("type = ?", "user").Count(&count)
```

1. 用户搜索排除非真实用户：

```go
db.Where("type = ? AND nickname LIKE ?", "user", "%"+keyword+"%").Find(&users)
```

1. 权限控制：

```go
var typePermissions = map[string][]string{
    "user":   {"send_message", "create_group", "create_bot"},
    "bot":    {"send_message", "read_conversation"},
    "system": {"send_notification", "broadcast"},
    "api":    {"read_messages", "send_webhook"},
}
```

1. 创建系统用户：系统启动时自动创建 ID=1 的系统用户（如果不存在）

### 2.3 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                    AI 助手模块优化架构                        │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐      ┌──────────────┐                     │
│  │ ChatCenter   │─────▶│ useBotChat   │                     │
│  │   (容器)      │      │  (对话逻辑)   │                     │
│  └──────────────┘      └──────┬───────┘                     │
│         │                      │                              │
│         ▼                      ▼                              │
│  ┌──────────────┐      ┌──────────────┐                     │
│  │BotChatView   │      │ useAIStream  │                     │
│  │  (UI 展示)    │      │  (流式处理)   │                     │
│  └──────────────┘      └──────┬───────┘                     │
│         │                      │                              │
│         ▼                      ▼                              │
│  ┌──────────────┐      ┌──────────────┐                     │
│  │MarkdownRenderer│    │  Bot 会话 API │                     │
│  │ThinkingIndicator│    │ /conversations│                    │
│  └──────────────┘      └──────────────┘                     │
└─────────────────────────────────────────────────────────────┘
```

***

## 三、详细设计

### 3.1 新增文件

#### 3.1.1 前端新增文件

| 文件路径                                         | 功能描述                 |
| -------------------------------------------- | -------------------- |
| `src/composables/useBotChat.ts`              | Bot 对话核心逻辑           |
| `src/composables/useAIStream.ts`             | SSE 流式响应处理           |
| `src/types/bot.ts`                           | Bot 相关类型定义（统一现有重复类型） |
| `src/components/shared/MarkdownRenderer.vue` | 统一的 Markdown 渲染组件    |
| `src/utils/clipboard.ts`                     | 复制和导出工具函数            |

#### 3.1.2 后端新增文件

| 文件路径                        | 功能描述         |
| --------------------------- | ------------ |
| `handler/ai_text_common.go` | 通用 AI 文本处理函数 |
| `handler/message_utils.go`  | 消息格式化工具      |

### 3.2 核心组件实现

#### 3.2.1 useBotChat.ts

```typescript
// src/composables/useBotChat.ts

import { ref } from 'vue'
import { useRequest } from './useRequest'
import { useAIStream } from './useAIStream'
import type { BotMessage, Bot } from '@/types/bot'

export function useBotChat(botId: number) {
  const { get, post } = useRequest()
  const { stream } = useAIStream()
  
  const conversationId = ref<number | null>(null)
  const messages = ref<BotMessage[]>([])
  const isGenerating = ref(false)
  const error = ref<string | null>(null)
  
  async function initConversation() {
    const response = await post('/api/v1/conversations/bot', { bot_id: botId })
    if (response?.data) {
      conversationId.value = response.data.id
      await loadMessages()
    }
  }
  
  async function loadMessages() {
    if (!conversationId.value) return
    const response = await get(`/api/v1/conversations/${conversationId.value}/messages`)
    if (response?.data) {
      messages.value = response.data.map(formatMessage)
    }
  }
  
  async function sendMessage(content: string) {
    if (!conversationId.value || isGenerating.value) return
    
    isGenerating.value = true
    error.value = null
    
    messages.value.push({
      id: Date.now(),
      role: 'user',
      content,
      timestamp: new Date()
    })
    
    const assistantMsg: BotMessage = {
      id: Date.now() + 1,
      role: 'assistant',
      content: '',
      timestamp: new Date(),
      isStreaming: true
    }
    messages.value.push(assistantMsg)
    
    try {
      await stream({
        url: `/api/v1/conversations/${conversationId.value}/messages/stream`,
        body: { content },
        onChunk: (chunk) => {
          assistantMsg.content += chunk
        },
        onComplete: () => {
          assistantMsg.isStreaming = false
          isGenerating.value = false
        },
        onError: (err) => {
          assistantMsg.content = '抱歉，AI 服务暂时不可用'
          assistantMsg.isStreaming = false
          isGenerating.value = false
          error.value = err.message
        }
      })
    } catch (e: any) {
      error.value = e.message
      isGenerating.value = false
    }
  }
  
  function clearMessages() {
    messages.value = []
  }
  
  return {
    conversationId,
    messages,
    isGenerating,
    error,
    initConversation,
    loadMessages,
    sendMessage,
    clearMessages
  }
}
```

#### 3.2.2 useAIStream.ts

```typescript
// src/composables/useAIStream.ts

import { ref } from 'vue'
import { getToken } from './useRequest'

interface StreamOptions {
  url: string
  body: Record<string, any>
  onChunk: (content: string) => void
  onComplete: () => void
  onError: (error: Error) => void
}

export function useAIStream() {
  const abortController = ref<AbortController | null>(null)
  
  async function stream(options: StreamOptions): Promise<void> {
    const token = getToken()
    abortController.value = new AbortController()
    
    try {
      const response = await fetch(options.url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(options.body),
        signal: abortController.value.signal
      })
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`)
      }
      
      const reader = response.body?.getReader()
      if (!reader) throw new Error('No reader available')
      
      const decoder = new TextDecoder()
      let buffer = ''
      
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        
        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''
        
        for (const line of lines) {
          if (line.startsWith('data: ')) {
            const data = line.slice(6)
            if (data.trim() === '') continue
            
            try {
              const chunk = JSON.parse(data)
              if (chunk.content) {
                options.onChunk(chunk.content)
              }
              if (chunk.finish === 'stop') {
                options.onComplete()
                return
              }
            } catch {
              // 忽略解析错误
            }
          }
        }
      }
      
      options.onComplete()
    } catch (e: any) {
      if (e.name === 'AbortError') {
        return
      }
      options.onError(e)
    }
  }
  
  function abort() {
    abortController.value?.abort()
  }
  
  return { stream, abort }
}
```

#### 3.2.3 MarkdownRenderer.vue

```vue
<!-- src/components/shared/MarkdownRenderer.vue -->
<template>
  <div class="markdown-content" @click="handleLinkClick">
    <div v-html="renderedContent"></div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { marked } from 'marked'
import { sanitizeMarkdown } from '@/utils/sanitize'

const props = defineProps<{
  content: string
}>()

const renderedContent = computed(() => {
  if (!props.content) return ''
  try {
    const result = marked(props.content)
    return sanitizeMarkdown(typeof result === 'string' ? result : String(result))
  } catch {
    return props.content.replace(/\n/g, '<br>')
  }
})

const handleLinkClick = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  const link = target.closest('a')
  if (link && window.electron?.shell?.openExternal) {
    event.preventDefault()
    const href = link.getAttribute('href')
    if (href) window.electron.shell.openExternal(href)
  }
}
</script>
```

#### 3.2.4 clipboard.ts

```typescript
// src/utils/clipboard.ts

export async function copyToClipboard(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    return false
  }
}

export function exportAsMarkdown(content: string, filename: string = 'content.md'): void {
  const blob = new Blob([content], { type: 'text/markdown' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}
```

### 3.3 后端修复

#### 3.3.1 修复 StreamMessage 添加系统提示词

```go
// handler/message_handler.go - StreamMessage 函数修改

// 在加载历史消息之前，添加系统提示词
var bot model.Bot
if err := db.First(&bot, botConv.BotID).Error; err != nil {
    log.Printf("[StreamMessage] 查找机器人信息失败: %v", err)
    close(responseChan)
    doneChan <- true
    return
}

// 解析 Bot 配置获取系统提示词
systemPrompt := "你是一个智能助手，帮助用户解决问题。"
if bot.Config != "" {
    var botConfig map[string]interface{}
    if err := json.Unmarshal([]byte(bot.Config), &botConfig); err == nil {
        if prompt, ok := botConfig["system_prompt"].(string); ok && prompt != "" {
            systemPrompt = prompt
        }
    }
}

var aiMessages []ai.Message
// 添加系统提示词
aiMessages = append(aiMessages, ai.Message{
    Role:    "system",
    Content: systemPrompt,
})

// 加载历史消息
var messages []model.Message
db.Where("conversation_id = ?", convID).Order("created_at ASC").Limit(20).Find(&messages)

for _, msg := range messages {
    role := "user"
    if msg.SenderID == 0 {
        role = "assistant"
    }
    aiMessages = append(aiMessages, ai.Message{
        Role:    role,
        Content: msg.Content,
    })
}
```

#### 3.3.2 通用 AI 文本处理函数

```go
// handler/ai_text_common.go

package handler

import (
    "errors"
    "qim-server/ai"
)

type AITextRequest struct {
    Text         string
    TaskType     string
    SystemPrompt string
    Options      map[string]interface{}
}

type AITextResponse struct {
    Result     string
    SourceLang string
    TargetLang string
}

func (h *AIHandler) processAIText(req AITextRequest) (*AITextResponse, error) {
    if !h.aiService.IsConfigured() {
        return nil, errors.New("AI服务未配置")
    }

    messages := []ai.Message{
        {Role: "system", Content: req.SystemPrompt},
        {Role: "user", Content: req.Text},
    }

    result, err := h.aiService.GetCompletion(messages)
    if err != nil {
        return nil, err
    }

    result = h.aiService.FilterOutput(result, "ai_"+req.TaskType)

    return &AITextResponse{Result: result}, nil
}
```

***

## 四、修改文件清单

### 4.1 前端修改

| 文件                                                | 修改内容                                  |
| ------------------------------------------------- | ------------------------------------- |
| `src/components/apps/ai/ChatCenter.vue`           | 使用 useBotChat 替代本地消息管理                |
| `src/components/apps/ai/BotChatView.vue`          | 复用 MarkdownRenderer、ThinkingIndicator |
| `src/components/ai/AIMessageContent.vue`          | 使用 MarkdownRenderer                   |
| `src/components/message/StreamingMessage.vue`     | 使用 MarkdownRenderer                   |
| `src/components/ai/AISummaryPanel.vue`            | 使用 MarkdownRenderer 和 clipboard 工具    |
| `src/composables/useAIActions.ts`                 | 重构为通用请求处理                             |
| `src/composables/useBots.ts`                      | 统一使用 useRequest                       |
| `src/components/apps/ai/ModelConfigFormModal.vue` | 修复 emit 错误                            |
| `src/types/ai.ts`                                 | 更新 AI\_PROVIDERS 默认模型                 |

### 4.2 后端修改

| 文件                              | 修改内容                    |
| ------------------------------- | ----------------------- |
| `handler/message_handler.go`    | StreamMessage 添加系统提示词支持 |
| `handler/ai_text_handler.go`    | 使用通用函数重构                |
| `handler/ai_summary_handler.go` | 使用消息格式化工具               |
| `handler/ai_search_handler.go`  | 使用消息格式化工具               |

***

## 五、实施计划

### 5.1 阶段一：P0 问题修复（核心功能）

1. **后端修复 StreamMessage**
   - 添加系统提示词支持
   - 测试流式对话功能
2. **前端实现 useBotChat**
   - 实现 Bot 会话初始化
   - 实现消息加载和发送
   - 实现 SSE 流式处理
3. **前端修复 ModelConfigFormModal**
   - 修复 emit 用法错误
4. **前端实现 handleEditBot**
   - 实现编辑机器人功能

### 5.2 阶段二：P1 问题修复（架构一致性）

1. **统一 API 调用方式**
   - useBots.ts 改用 useRequest
   - useModelConfigs.ts 改用 useRequest
2. **统一类型定义**
   - 创建 types/bot.ts
   - 删除重复定义
3. **替换 alert/confirm**
   - 使用项目共享组件
4. **抽取复用组件**
   - MarkdownRenderer.vue
   - clipboard.ts
   - 重构 useAIActions

### 5.3 阶段三：后端优化

1. **抽取通用函数**
   - ai\_text\_common.go
   - message\_utils.go
2. **重构现有代码**
   - ai\_text\_handler.go
   - ai\_summary\_handler.go
   - ai\_search\_handler.go

***

## 六、验收标准

### 6.1 功能验收

- [ ] Bot 对话功能可用，能正常发送消息并收到流式回复
- [ ] 对话历史持久化，刷新页面后仍可查看
- [ ] Bot 使用正确的系统提示词
- [ ] 模型配置保存功能正常
- [ ] 编辑机器人功能正常

### 6.2 架构验收

- [ ] 所有 API 调用统一使用 useRequest
- [ ] 类型定义无重复
- [ ] 无 alert/confirm 调用
- [ ] Markdown 渲染组件复用
- [ ] 思考指示器组件复用

### 6.3 代码质量

- [ ] TypeScript 编译无错误
- [ ] ESLint 检查通过
- [ ] 无重复代码

***

## 七、风险与应对

| 风险                        | 影响 | 应对措施              |
| ------------------------- | -- | ----------------- |
| 后端 StreamMessage 修改影响现有功能 | 高  | 充分测试，保留原有逻辑作为降级方案 |
| SSE 连接不稳定                 | 中  | 添加重连机制和错误提示       |
| 对话历史过多影响性能                | 中  | 后端限制加载条数，前端支持分页   |

***

## 八、附录

### 8.1 Bot 会话数据存储详解

#### 8.1.1 数据表结构

**1. users 表（用户和 Bot）**

```go
type User struct {
    ID           uint       // 用户/Bot ID
    Username     string     // 用户名
    Nickname     string     // 昵称（Bot 显示名称）
    Avatar       string     // 头像
    Type         string     // 'user' | 'bot'
    // ...
}
```

**Bot 虚拟用户特点**：

- `type = 'bot'`
- 每个对应一个 Bot 记录
- 只存储显示信息（nickname、avatar）

***

**2. bots 表（Bot 配置）**

```go
type Bot struct {
    ID              uint
    Name            string     // Bot 名称
    Avatar          string     // Bot 头像
    Description     string
    Type            string     // 'system', 'custom', 'ai'
    Config          string     // JSON 配置（系统提示词等）
    IsActive        bool
    ApprovalStatus  string
    CreatorID       uint
    VirtualUserID   *uint      // 关联虚拟用户 ID
}
```

**职责**：Bot 配置管理、审批流程、对话逻辑

***

**3. conversations 表（会话基本信息）**

```go
type Conversation struct {
    ID            uint       // 会话 ID
    Type          string     // 会话类型：'single', 'group', 'bot'
    IsDeleted     bool
    LastMessageID *uint
    LastMessageAt *time.Time
    // ...
}
```

**Bot 会话特点**：`type = 'bot'`

***

**4. bot\_conversations 表（Bot 与会话关联）**

```go
type BotConversation struct {
    ID             uint    // 关联 ID
    BotID          uint    // Bot ID
    UserID         uint    // 用户 ID
    ConversationID uint    // 会话 ID
}
```

***

**5. messages 表（所有消息）**

```go
type Message struct {
    ID              uint
    ConversationID  uint
    SenderID        uint       // 用户 ID 或虚拟用户 ID
    Type            string     // 'text', 'markdown'
    Content         string
    // ...
    Sender          User       // 关联 users 表
}
```

**消息区分**：

- 用户消息：`SenderID = 用户ID`，`Sender.Type = 'user'`
- Bot 消息：`SenderID = 虚拟用户ID`，`Sender.Type = 'bot'`

***

#### 8.1.2 数据流向

```
┌─────────────────────────────────────────────────────────────┐
│                    Bot 会话数据流向                          │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  1. 创建 Bot                                                 │
│     POST /api/v1/bots { name, avatar, config, ... }         │
│     ↓                                                         │
│     ┌──────────────────────────────────────┐                │
│     │ ① 创建 Bot 记录                      │                │
│     │ ② 创建虚拟用户（type='bot'）         │                │
│     │ ③ 更新 Bot.VirtualUserID            │                │
│     └──────────────────────────────────────┘                │
│                                                               │
│  2. 创建会话                                                 │
│     POST /api/v1/conversations/bot { bot_id }               │
│     ↓                                                         │
│     ┌──────────────────────────────────────┐                │
│     │ 检查 bot_conversations 是否存在      │                │
│     │ WHERE bot_id = ? AND user_id = ?     │                │
│     └──────────────────────────────────────┘                │
│     ↓                                                         │
│     存在 → 返回已有会话                                       │
│     不存在 → 创建新会话                                       │
│                                                               │
│  3. 发送消息（流式）                                         │
│     POST /api/v1/conversations/:id/messages/stream          │
│     { content: "用户消息" }                                  │
│     ↓                                                         │
│     ┌──────────────────────────────────────┐                │
│     │ ① 保存用户消息（sender_id = 用户ID） │                │
│     │ ② 查询 Bot 配置                      │                │
│     │ ③ 加载历史消息作为上下文             │                │
│     │ ④ 调用 AI API 生成回复               │                │
│     │ ⑤ 保存 Bot 回复                      │                │
│     │    sender_id = Bot.VirtualUserID     │                │
│     └──────────────────────────────────────┘                │
│     ↓                                                         │
│     SSE 流式返回 Bot 回复                                     │
│                                                               │
│  4. 查询消息                                                 │
│     GET /api/v1/conversations/:id/messages                  │
│     ↓                                                         │
│     db.Preload("Sender").Find(&messages)                    │
│     ↓                                                         │
│     返回消息列表                                              │
│     - 用户消息：Sender.Type = 'user'                         │
│     - Bot 消息：Sender.Type = 'bot'                          │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

***

#### 8.1.3 关键代码位置

**创建 Bot 和虚拟用户**：

- 接口：`POST /api/v1/bots`
- 代码：`handler/bot_creation_handler.go` → `CreateBot` 函数

**Bot 回复保存（流式）**：

- 接口：`POST /api/v1/conversations/:id/messages/stream`
- 代码：`handler/message_handler.go:529-535`

```go
botReply := model.Message{
    ConversationID: uint(convIDUint),
    SenderID:       bot.VirtualUserID,  // 虚拟用户 ID
    Type:           "markdown",
    Content:        fullResponse,
}
db.Create(&botReply)
```

***

#### 8.1.4 查询示例

```sql
-- 获取某个用户与某个 Bot 的会话
SELECT bc.*, c.*, b.name as bot_name
FROM bot_conversations bc
JOIN conversations c ON bc.conversation_id = c.id
JOIN bots b ON bc.bot_id = b.id
WHERE bc.user_id = ? AND bc.bot_id = ?;

-- 获取会话的所有消息（包含用户和 Bot）
SELECT m.*, u.nickname as sender_name, u.avatar as sender_avatar, u.type as sender_type
FROM messages m
JOIN users u ON m.sender_id = u.id
WHERE m.conversation_id = ?
ORDER BY m.created_at ASC;

-- 获取所有 Bot 用户
SELECT * FROM users WHERE type = 'bot';

-- 统计某个 Bot 的对话数量
SELECT COUNT(DISTINCT bc.user_id) as user_count,
       COUNT(m.id) as message_count
FROM bot_conversations bc
JOIN messages m ON bc.conversation_id = m.conversation_id
JOIN users u ON m.sender_id = u.id AND u.type = 'bot'
WHERE bc.bot_id = ?;
```

***

#### 8.1.5 数据迁移

现有数据迁移步骤：

```sql
-- 1. 添加用户类型字段
ALTER TABLE users ADD COLUMN type VARCHAR(20) DEFAULT 'user';
ALTER TABLE users ADD COLUMN status VARCHAR(20) DEFAULT 'active';
ALTER TABLE users ADD INDEX idx_type (type);

-- 2. 更新现有用户的类型
UPDATE users SET type = 'user' WHERE type IS NULL OR type = '';

-- 3. 创建系统用户（如果不存在）
INSERT INTO users (username, nickname, avatar, type, status, created_at, updated_at)
SELECT 'system', '系统', '/system.png', 'system', 'active', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM users WHERE type = 'system');

-- 4. 为现有 Bot 创建虚拟用户
INSERT INTO users (username, nickname, avatar, type, status, created_at, updated_at)
SELECT 
    CONCAT('bot_', id),
    name,
    avatar,
    'bot',
    'active',
    NOW(),
    NOW()
FROM bots
WHERE virtual_user_id IS NULL;

-- 5. 更新 Bot 表关联虚拟用户
UPDATE bots b
JOIN users u ON u.username = CONCAT('bot_', b.id) AND u.type = 'bot'
SET b.virtual_user_id = u.id
WHERE b.virtual_user_id IS NULL;

-- 6. 更新消息表的 sender_id（Bot 消息）
UPDATE messages m
JOIN bot_conversations bc ON m.conversation_id = bc.conversation_id
JOIN bots b ON bc.bot_id = b.id
SET m.sender_id = b.virtual_user_id
WHERE m.sender_id = 0;

-- 7. 更新 system_messages 表的 sender_id（系统消息）
UPDATE system_messages sm
SET sm.sender_id = (SELECT id FROM users WHERE type = 'system' LIMIT 1)
WHERE sm.sender_id = 0;
```

**迁移验证**：

```sql
-- 验证用户类型分布
SELECT type, COUNT(*) FROM users GROUP BY type;

-- 验证 Bot 虚拟用户
SELECT b.id, b.name, b.virtual_user_id, u.nickname
FROM bots b
LEFT JOIN users u ON b.virtual_user_id = u.id;

-- 验证消息发送者
SELECT m.id, m.sender_id, u.type, u.nickname
FROM messages m
JOIN users u ON m.sender_id = u.id
ORDER BY m.id DESC
LIMIT 10;
```

***

#### 8.1.6 注意事项

1. **用户统计排除 Bot**：
   ```go
   db.Model(&User{}).Where("type = ?", "user").Count(&count)
   ```
2. **用户搜索排除 Bot**：
   ```go
   db.Where("type = ? AND nickname LIKE ?", "user", "%"+keyword+"%").Find(&users)
   ```
3. **Bot ID 分配**：建议虚拟用户 ID 从 10000 开始，避免与用户 ID 冲突
4. **历史消息加载**：后端默认加载最近 20 条消息作为 AI 上下文

### 8.2 相关文件参考

- 后端 Bot 会话创建：`handler/conversation_handler.go:162-186`
- 后端流式消息处理：`handler/message_handler.go:470-566`
- 后端非流式消息处理：`handler/misc_handler.go:202-306`
- 数据模型定义：`model/model.go:55-65` (Conversation), `model/model.go:119-134` (Message), `model/model.go:240-249` (BotConversation)

