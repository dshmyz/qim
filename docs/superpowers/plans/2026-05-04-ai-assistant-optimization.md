# AI 助手模块优化实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 修复 AI 助手模块的核心问题，实现 Bot 对话功能，统一架构和消除重复代码

**架构：** 
- 后端：添加用户类型字段，为 Bot 创建虚拟用户，修复 StreamMessage 系统提示词
- 前端：实现 useBotChat 使用现有 Bot 会话 API，复用 Markdown 渲染和 ThinkingIndicator 组件

**技术栈：** Go (Gin/GORM), Vue 3 (TypeScript), MySQL/SQLite

---

## 文件结构

### 后端文件

| 文件 | 职责 | 操作 |
|------|------|------|
| `model/model.go` | 添加 User.Type 和 User.Status 字段，Bot.VirtualUserID 字段 | 修改 |
| `handler/bot_creation_handler.go` | 创建 Bot 时同步创建虚拟用户 | 修改 |
| `handler/message_handler.go` | StreamMessage 添加系统提示词支持 | 修改 |
| `handler/misc_handler.go` | HandleBotMessage 使用虚拟用户 ID | 修改 |
| `handler/conversation_handler.go` | GetOrCreateBotConversation 返回虚拟用户信息 | 修改 |
| `app/init.go` | 系统启动时创建系统用户 | 修改 |
| `ddl_mysql.sql` | 添加 type、status、virtual_user_id 字段 | 修改 |
| `ddl_sqlite.sql` | 添加 type、status、virtual_user_id 字段 | 修改 |

### 前端文件

| 文件 | 职责 | 操作 |
|------|------|------|
| `src/composables/useBotChat.ts` | Bot 对话核心逻辑 | 创建 |
| `src/composables/useAIStream.ts` | SSE 流式响应处理 | 创建 |
| `src/types/bot.ts` | Bot 相关类型定义 | 创建 |
| `src/components/shared/MarkdownRenderer.vue` | 统一的 Markdown 渲染组件 | 创建 |
| `src/utils/clipboard.ts` | 复制和导出工具函数 | 创建 |
| `src/components/apps/ai/ChatCenter.vue` | 使用 useBotChat 替代本地消息管理 | 修改 |
| `src/components/apps/ai/BotChatView.vue` | 复用 MarkdownRenderer、ThinkingIndicator | 修改 |
| `src/components/ai/AIMessageContent.vue` | 使用 MarkdownRenderer | 修改 |
| `src/components/message/StreamingMessage.vue` | 使用 MarkdownRenderer | 修改 |
| `src/components/ai/AISummaryPanel.vue` | 使用 MarkdownRenderer 和 clipboard 工具 | 修改 |
| `src/composables/useAIActions.ts` | 重构为通用请求处理 | 修改 |
| `src/composables/useBots.ts` | 统一使用 useRequest | 修改 |
| `src/components/apps/ai/ModelConfigFormModal.vue` | 修复 emit 错误 | 修改 |
| `src/types/ai.ts` | 更新 AI_PROVIDERS 默认模型 | 修改 |

---

## 任务分解

---

### 任务 1：后端 - 添加用户类型字段

**文件：**
- 修改：`model/model.go:15-35`
- 修改：`ddl_mysql.sql:10-20`
- 修改：`ddl_sqlite.sql:10-20`

- [ ] **步骤 1：修改 User 模型添加 type 和 status 字段**

```go
// model/model.go - User 结构体
type User struct {
	ID           uint           `json:"id" gorm:"primarykey"`
	Username     string         `json:"username" gorm:"uniqueIndex;size:50;not null"`
	PasswordHash string         `json:"-" gorm:"size:255;not null"`
	Nickname     string         `json:"nickname" gorm:"size:100"`
	Avatar       string         `json:"avatar" gorm:"size:500"`
	Type         string         `json:"type" gorm:"size:20;default:'user';index"`  // 'user' | 'bot' | 'system' | 'api'
	Status       string         `json:"status" gorm:"size:20;default:'active'"`    // 'active' | 'inactive' | 'suspended'
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}
```

- [ ] **步骤 2：修改 Bot 模型添加 VirtualUserID 字段**

```go
// model/model.go - Bot 结构体
type Bot struct {
	ID              uint           `json:"id" gorm:"primarykey"`
	Name            string         `json:"name" gorm:"size:100;not null"`
	Avatar          string         `json:"avatar" gorm:"size:500"`
	Description     string         `json:"description" gorm:"type:text"`
	Type            string         `json:"type" gorm:"size:20;not null"`  // 'system', 'custom', 'ai'
	Config          string         `json:"config" gorm:"type:text"`
	IsActive        bool           `json:"is_active" gorm:"default:true"`
	ApprovalStatus  string         `json:"approval_status" gorm:"size:20;default:'pending'"`
	CreatorID       uint           `json:"creator_id"`
	CreatorName     string         `json:"creator_name" gorm:"size:100"`
	VirtualUserID   *uint          `json:"virtual_user_id"`  // 关联虚拟用户 ID
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}
```

- [ ] **步骤 3：更新 MySQL DDL**

```sql
-- ddl_mysql.sql - users 表
CREATE TABLE IF NOT EXISTS `users` (
  `id` INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  `username` VARCHAR(50) NOT NULL UNIQUE,
  `password_hash` VARCHAR(255) NOT NULL,
  `nickname` VARCHAR(100),
  `avatar` VARCHAR(500),
  `type` VARCHAR(20) DEFAULT 'user',
  `status` VARCHAR(20) DEFAULT 'active',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME,
  INDEX `idx_users_deleted_at` (`deleted_at`),
  INDEX `idx_users_type` (`type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ddl_mysql.sql - bots 表添加 virtual_user_id
ALTER TABLE bots ADD COLUMN virtual_user_id INT UNSIGNED NULL;
ALTER TABLE bots ADD INDEX idx_bots_virtual_user_id (virtual_user_id);
```

- [ ] **步骤 4：更新 SQLite DDL**

```sql
-- ddl_sqlite.sql - users 表
CREATE TABLE IF NOT EXISTS `users` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `username` VARCHAR(50) NOT NULL UNIQUE,
  `password_hash` VARCHAR(255) NOT NULL,
  `nickname` VARCHAR(100),
  `avatar` VARCHAR(500),
  `type` VARCHAR(20) DEFAULT 'user',
  `status` VARCHAR(20) DEFAULT 'active',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_users_deleted_at` ON `users`(`deleted_at`);
CREATE INDEX IF NOT EXISTS `idx_users_type` ON `users`(`type`);

-- ddl_sqlite.sql - bots 表添加 virtual_user_id
ALTER TABLE bots ADD COLUMN virtual_user_id INTEGER;
CREATE INDEX IF NOT EXISTS `idx_bots_virtual_user_id` ON `bots`(`virtual_user_id`);
```

- [ ] **步骤 5：Commit**

```bash
git add model/model.go ddl_mysql.sql ddl_sqlite.sql
git commit -m "feat: add user type and bot virtual_user_id fields"
```

---

### 任务 2：后端 - 创建系统用户和 Bot 虚拟用户

**文件：**
- 修改：`app/init.go:70-85`
- 修改：`handler/bot_creation_handler.go:60-120`

- [ ] **步骤 1：在 app/init.go 添加系统用户创建逻辑**

```go
// app/init.go - 在 AutoMigrate 后添加
func initSystemUser() {
	db := database.GetDB()

	var count int64
	db.Model(&model.User{}).Where("type = ?", "system").Count(&count)
	if count > 0 {
		return
	}

	systemUser := model.User{
		Username: "system",
		Nickname: "系统",
		Avatar:   "/system.png",
		Type:     "system",
		Status:   "active",
	}
	if err := db.Create(&systemUser).Error; err != nil {
		log.Printf("[Init] 创建系统用户失败: %v", err)
	} else {
		log.Printf("[Init] 创建系统用户成功: ID=%d", systemUser.ID)
	}
}

// 在 InitDB() 函数末尾调用
func InitDB() {
	// ... AutoMigrate ...
	
	initSystemUser()
}
```

- [ ] **步骤 2：修改 CreateBot 创建虚拟用户**

```go
// handler/bot_creation_handler.go - CreateBot 函数
func CreateBot(c *gin.Context) {
	userID, _ := c.Get("user_id")
	db := database.GetDB()

	// ... 现有的验证逻辑 ...

	var req CreateBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 创建 Bot
	bot := model.Bot{
		Name:           req.Name,
		Description:    req.Description,
		Type:           req.Type,
		Config:         req.Config,
		IsActive:       false,
		ApprovalStatus: "pending",
		CreatorID:      userID.(uint),
		CreatorName:    getUserName(db, userID.(uint)),
	}

	if req.Avatar != "" {
		bot.Avatar = req.Avatar
	}

	// 开启事务
	tx := db.Begin()

	// 创建 Bot
	if err := tx.Create(&bot).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	// 创建虚拟用户
	virtualUser := model.User{
		Username: fmt.Sprintf("bot_%d", bot.ID),
		Nickname: bot.Name,
		Avatar:   bot.Avatar,
		Type:     "bot",
		Status:   "active",
	}
	if err := tx.Create(&virtualUser).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建虚拟用户失败"})
		return
	}

	// 更新 Bot 的 VirtualUserID
	bot.VirtualUserID = &virtualUser.ID
	if err := tx.Save(&bot).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新Bot失败"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"id":                bot.ID,
			"name":              bot.Name,
			"virtual_user_id":   virtualUser.ID,
			"approval_status":   bot.ApprovalStatus,
		},
	})
}
```

- [ ] **步骤 3：Commit**

```bash
git add app/init.go handler/bot_creation_handler.go
git commit -m "feat: create virtual user when creating bot"
```

---

### 任务 3：后端 - 修复 StreamMessage 添加系统提示词

**文件：**
- 修改：`handler/message_handler.go:470-566`

- [ ] **步骤 1：修改 StreamMessage 函数添加系统提示词**

```go
// handler/message_handler.go - StreamMessage 函数
func StreamMessage(c *gin.Context, convID uint, content string, responseChan chan string, doneChan chan bool) {
	db := database.GetDB()

	// 查找 BotConversation
	var botConv model.BotConversation
	if err := db.Where("conversation_id = ?", convID).First(&botConv).Error; err != nil {
		log.Printf("[StreamMessage] 查找 BotConversation 失败: %v", err)
		close(responseChan)
		doneChan <- true
		return
	}

	// 查找 Bot 信息
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

	// 构建消息列表
	var aiMessages []ai.Message
	aiMessages = append(aiMessages, ai.Message{
		Role:    "system",
		Content: systemPrompt,
	})

	// 加载历史消息
	var messages []model.Message
	db.Where("conversation_id = ?", convID).Order("created_at ASC").Limit(20).Find(&messages)

	for _, msg := range messages {
		role := "user"
		if msg.SenderID == 0 || (bot.VirtualUserID != nil && msg.SenderID == *bot.VirtualUserID) {
			role = "assistant"
		}
		aiMessages = append(aiMessages, ai.Message{
			Role:    role,
			Content: msg.Content,
		})
	}

	// 添加当前用户消息
	aiMessages = append(aiMessages, ai.Message{
		Role:    "user",
		Content: content,
	})

	// ... 后续的流式处理逻辑 ...
}
```

- [ ] **步骤 2：修改 Bot 回复保存使用虚拟用户 ID**

```go
// handler/message_handler.go - StreamMessage 函数中保存回复的部分
botReply := model.Message{
	ConversationID: convID,
	SenderID:       *bot.VirtualUserID,  // 使用虚拟用户 ID
	Type:           "markdown",
	Content:        fullResponse,
}
db.Create(&botReply)
```

- [ ] **步骤 3：Commit**

```bash
git add handler/message_handler.go
git commit -m "fix: add system prompt support to StreamMessage"
```

---

### 任务 4：后端 - 修改 HandleBotMessage 使用虚拟用户

**文件：**
- 修改：`handler/misc_handler.go:202-306`

- [ ] **步骤 1：修改 HandleBotMessage 使用虚拟用户 ID**

```go
// handler/misc_handler.go - HandleBotMessage 函数
func HandleBotMessage(c *gin.Context) {
	// ... 现有逻辑 ...

	// 查找 Bot
	var bot model.Bot
	if err := db.First(&bot, botID).Error; err != nil {
		return
	}

	// 检查是否有虚拟用户
	if bot.VirtualUserID == nil {
		log.Printf("[HandleBotMessage] Bot 没有虚拟用户: %d", botID)
		return
	}

	// ... 生成回复逻辑 ...

	// 保存 Bot 回复
	msg := model.Message{
		ConversationID: convID,
		SenderID:       *bot.VirtualUserID,  // 使用虚拟用户 ID
		Type:           "markdown",
		Content:        reply,
	}
	db.Create(&msg)

	// ... 后续逻辑 ...
}
```

- [ ] **步骤 2：Commit**

```bash
git add handler/misc_handler.go
git commit -m "fix: use virtual user id in HandleBotMessage"
```

---

### 任务 5：前端 - 创建类型定义文件

**文件：**
- 创建：`src/types/bot.ts`

- [ ] **步骤 1：创建 Bot 类型定义文件**

```typescript
// src/types/bot.ts

export interface Bot {
  id: number
  name: string
  avatar?: string
  description?: string
  type: 'system' | 'custom' | 'ai'
  config?: BotConfig
  approvalStatus: 'pending' | 'approved' | 'rejected'
  creatorId: number
  creatorName: string
  virtualUserId?: number
  isActive: boolean
  createdAt: string
  updatedAt: string
}

export interface BotConfig {
  systemPrompt?: string
  temperature?: number
  maxTokens?: number
  model?: string
}

export interface BotMessage {
  id: number
  conversationId: number
  senderId: number
  senderType: 'user' | 'bot' | 'system' | 'api'
  sender?: {
    id: number
    nickname: string
    avatar?: string
    type: string
  }
  type: 'text' | 'markdown'
  content: string
  timestamp: Date
  isStreaming?: boolean
}

export interface BotConversation {
  id: number
  botId: number
  userId: number
  conversationId: number
  createdAt: string
}
```

- [ ] **步骤 2：Commit**

```bash
git add src/types/bot.ts
git commit -m "feat: add bot type definitions"
```

---

### 任务 6：前端 - 创建 useAIStream composable

**文件：**
- 创建：`src/composables/useAIStream.ts`

- [ ] **步骤 1：创建 useAIStream.ts**

```typescript
// src/composables/useAIStream.ts

import { ref } from 'vue'

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
    const token = localStorage.getItem('token')
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

- [ ] **步骤 2：Commit**

```bash
git add src/composables/useAIStream.ts
git commit -m "feat: add useAIStream composable for SSE streaming"
```

---

### 任务 7：前端 - 创建 useBotChat composable

**文件：**
- 创建：`src/composables/useBotChat.ts`

- [ ] **步骤 1：创建 useBotChat.ts**

```typescript
// src/composables/useBotChat.ts

import { ref, onMounted } from 'vue'
import { useRequest } from './useRequest'
import { useAIStream } from './useAIStream'
import type { BotMessage } from '@/types/bot'

export function useBotChat(botId: number) {
  const { get, post } = useRequest()
  const { stream } = useAIStream()

  const conversationId = ref<number | null>(null)
  const messages = ref<BotMessage[]>([])
  const isGenerating = ref(false)
  const error = ref<string | null>(null)

  async function initConversation() {
    try {
      const response = await post('/api/v1/conversations/bot', { bot_id: botId })
      if (response?.data) {
        conversationId.value = response.data.id
        await loadMessages()
      }
    } catch (e: any) {
      error.value = e.message || '初始化会话失败'
    }
  }

  async function loadMessages() {
    if (!conversationId.value) return

    try {
      const response = await get(`/api/v1/conversations/${conversationId.value}/messages`)
      if (response?.data) {
        messages.value = response.data.map((msg: any) => ({
          id: msg.id,
          conversationId: msg.conversation_id,
          senderId: msg.sender_id,
          senderType: msg.sender?.type || 'user',
          sender: msg.sender,
          type: msg.type,
          content: msg.content,
          timestamp: new Date(msg.created_at),
          isStreaming: false
        }))
      }
    } catch (e: any) {
      error.value = e.message || '加载消息失败'
    }
  }

  async function sendMessage(content: string) {
    if (!conversationId.value || isGenerating.value) return

    isGenerating.value = true
    error.value = null

    messages.value.push({
      id: Date.now(),
      conversationId: conversationId.value,
      senderId: 0,
      senderType: 'user',
      type: 'text',
      content,
      timestamp: new Date()
    })

    const assistantMsg: BotMessage = {
      id: Date.now() + 1,
      conversationId: conversationId.value!,
      senderId: 0,
      senderType: 'bot',
      type: 'markdown',
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

  onMounted(() => {
    initConversation()
  })

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

- [ ] **步骤 2：Commit**

```bash
git add src/composables/useBotChat.ts
git commit -m "feat: add useBotChat composable for bot conversation"
```

---

### 任务 8：前端 - 创建 MarkdownRenderer 组件

**文件：**
- 创建：`src/components/shared/MarkdownRenderer.vue`

- [ ] **步骤 1：创建 MarkdownRenderer.vue**

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

<style scoped>
.markdown-content {
  line-height: 1.6;
  word-wrap: break-word;
}

.markdown-content :deep(pre) {
  background: #f6f8fa;
  padding: 16px;
  border-radius: 6px;
  overflow-x: auto;
}

.markdown-content :deep(code) {
  background: #f6f8fa;
  padding: 2px 6px;
  border-radius: 3px;
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 0.9em;
}

.markdown-content :deep(pre code) {
  background: transparent;
  padding: 0;
}

.markdown-content :deep(a) {
  color: #0366d6;
  text-decoration: none;
}

.markdown-content :deep(a:hover) {
  text-decoration: underline;
}

.markdown-content :deep(ul),
.markdown-content :deep(ol) {
  padding-left: 2em;
}

.markdown-content :deep(blockquote) {
  border-left: 4px solid #dfe2e5;
  padding-left: 16px;
  color: #6a737d;
  margin: 0;
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/shared/MarkdownRenderer.vue
git commit -m "feat: add MarkdownRenderer component"
```

---

### 任务 9：前端 - 创建 clipboard 工具函数

**文件：**
- 创建：`src/utils/clipboard.ts`

- [ ] **步骤 1：创建 clipboard.ts**

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

- [ ] **步骤 2：Commit**

```bash
git add src/utils/clipboard.ts
git commit -m "feat: add clipboard utility functions"
```

---

### 任务 10：前端 - 修改 ChatCenter 使用 useBotChat

**文件：**
- 修改：`src/components/apps/ai/ChatCenter.vue`

- [ ] **步骤 1：修改 ChatCenter.vue 使用 useBotChat**

```vue
<!-- src/components/apps/ai/ChatCenter.vue -->
<template>
  <div class="chat-center">
    <BotList v-if="!selectedBot" :bots="bots" @select="selectBot" />
    <BotChatView
      v-else
      :bot="selectedBot"
      :messages="messages"
      :thinking="isGenerating"
      @send="handleSendMessage"
      @back="selectedBot = null"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import BotList from './BotList.vue'
import BotChatView from './BotChatView.vue'
import { useBotChat } from '@/composables/useBotChat'
import { useBots } from '@/composables/useBots'
import type { Bot } from '@/types/bot'

const { bots, fetchBots } = useBots()
const selectedBot = ref<Bot | null>(null)

const {
  messages,
  isGenerating,
  sendMessage,
  initConversation
} = useBotChat(selectedBot.value?.id || 0)

watch(selectedBot, (newBot) => {
  if (newBot) {
    initConversation()
  }
})

function selectBot(bot: Bot) {
  selectedBot.value = bot
}

async function handleSendMessage(content: string) {
  await sendMessage(content)
}

fetchBots()
</script>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/apps/ai/ChatCenter.vue
git commit -m "feat: integrate useBotChat in ChatCenter"
```

---

### 任务 11：前端 - 修改 BotChatView 复用组件

**文件：**
- 修改：`src/components/apps/ai/BotChatView.vue`

- [ ] **步骤 1：修改 BotChatView.vue 复用 MarkdownRenderer 和 ThinkingIndicator**

```vue
<!-- src/components/apps/ai/BotChatView.vue -->
<template>
  <div class="bot-chat-view">
    <div class="chat-header">
      <button class="back-btn" @click="$emit('back')">
        <i class="fas fa-arrow-left" />
      </button>
      <div class="bot-info">
        <img v-if="bot?.avatar" :src="bot.avatar" class="bot-avatar" />
        <span class="bot-name">{{ bot?.name }}</span>
      </div>
    </div>

    <div class="messages" ref="messagesContainer">
      <div v-for="msg in messages" :key="msg.id" :class="['message', msg.senderType]">
        <div v-if="msg.senderType === 'user'" class="content user-content">
          {{ msg.content }}
        </div>
        <MarkdownRenderer v-else :content="msg.content" />
      </div>
      <ThinkingIndicator v-if="thinking" />
    </div>

    <div class="input-area">
      <textarea
        v-model="input"
        :placeholder="`向 ${bot?.name} 提问...`"
        @keydown.enter.exact="handleSend"
        @keydown.enter.shift.exact.prevent="input += '\n'"
        rows="1"
      />
      <button @click="handleSend" :disabled="!input.trim() || thinking">
        <i class="fas fa-paper-plane" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, watch } from 'vue'
import MarkdownRenderer from '@/components/shared/MarkdownRenderer.vue'
import ThinkingIndicator from '@/components/message/ThinkingIndicator.vue'
import type { Bot, BotMessage } from '@/types/bot'

const props = defineProps<{
  bot: Bot | null
  messages: BotMessage[]
  thinking: boolean
}>()

const emit = defineEmits<{
  send: [content: string]
  back: []
}>()

const input = ref('')
const messagesContainer = ref<HTMLElement | null>(null)

function handleSend() {
  if (!input.value.trim() || props.thinking) return
  emit('send', input.value.trim())
  input.value = ''
}

watch(() => props.messages.length, () => {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
    }
  })
})
</script>

<style scoped>
.bot-chat-view {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.chat-header {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #e5e7eb;
}

.back-btn {
  background: none;
  border: none;
  font-size: 16px;
  cursor: pointer;
  padding: 8px;
  margin-right: 8px;
}

.bot-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.bot-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
}

.bot-name {
  font-weight: 500;
}

.messages {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

.message {
  margin-bottom: 12px;
}

.message.user {
  text-align: right;
}

.user-content {
  display: inline-block;
  background: #3b82f6;
  color: white;
  padding: 8px 12px;
  border-radius: 12px;
  max-width: 70%;
  word-wrap: break-word;
}

.message.bot {
  text-align: left;
}

.message.bot :deep(.markdown-content) {
  background: #f3f4f6;
  padding: 12px 16px;
  border-radius: 12px;
  max-width: 70%;
  display: inline-block;
  text-align: left;
}

.input-area {
  display: flex;
  padding: 12px 16px;
  border-top: 1px solid #e5e7eb;
  gap: 8px;
}

.input-area textarea {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  resize: none;
  font-size: 14px;
  line-height: 1.5;
}

.input-area textarea:focus {
  outline: none;
  border-color: #3b82f6;
}

.input-area button {
  padding: 8px 16px;
  background: #3b82f6;
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
}

.input-area button:disabled {
  background: #9ca3af;
  cursor: not-allowed;
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/apps/ai/BotChatView.vue
git commit -m "feat: refactor BotChatView to reuse MarkdownRenderer and ThinkingIndicator"
```

---

### 任务 12：前端 - 修改 ModelConfigFormModal 修复 emit 错误

**文件：**
- 修改：`src/components/apps/ai/ModelConfigFormModal.vue`

- [ ] **步骤 1：修改 ModelConfigFormModal.vue 修复 emit 用法**

```vue
<!-- src/components/apps/ai/ModelConfigFormModal.vue -->
<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  visible: boolean
  config?: any
}>()

const emit = defineEmits<{
  close: []
  save: [data: any]
}>()

const loading = ref(false)
const error = ref<string | null>(null)
const form = ref({
  name: '',
  provider: 'openai',
  apiKey: '',
  model: '',
  baseUrl: ''
})

function handleSubmit() {
  if (!form.value.name || !form.value.apiKey || !form.value.model) {
    error.value = '请填写必填字段'
    return
  }

  loading.value = true
  error.value = null

  emit('save', { ...form.value })
}

function handleClose() {
  emit('close')
}

function setLoading(val: boolean) {
  loading.value = val
}

function setError(val: string | null) {
  error.value = val
}

defineExpose({
  setLoading,
  setError
})
</script>
```

- [ ] **步骤 2：修改父组件处理 save 事件**

```vue
<!-- src/components/apps/ai/MyModelConfigs.vue -->
<template>
  <ModelConfigFormModal
    :visible="showModal"
    :config="editingConfig"
    @close="showModal = false"
    @save="handleSave"
    ref="formModalRef"
  />
</template>

<script setup lang="ts">
import { ref } from 'vue'
import ModelConfigFormModal from './ModelConfigFormModal.vue'

const formModalRef = ref<InstanceType<typeof ModelConfigFormModal> | null>(null)

async function handleSave(data: any) {
  try {
    await createConfig(data)
    formModalRef.value?.setLoading(false)
    showModal.value = false
  } catch (e: any) {
    formModalRef.value?.setLoading(false)
    formModalRef.value?.setError(e.message || '保存失败')
  }
}
</script>
```

- [ ] **步骤 3：Commit**

```bash
git add src/components/apps/ai/ModelConfigFormModal.vue src/components/apps/ai/MyModelConfigs.vue
git commit -m "fix: correct emit usage in ModelConfigFormModal"
```

---

### 任务 13：前端 - 修改 useBots 统一使用 useRequest

**文件：**
- 修改：`src/composables/useBots.ts`

- [ ] **步骤 1：修改 useBots.ts 使用 useRequest**

```typescript
// src/composables/useBots.ts

import { ref } from 'vue'
import { useRequest } from './useRequest'
import type { Bot } from '@/types/bot'

export function useBots() {
  const { get, post, put, del } = useRequest()

  const bots = ref<Bot[]>([])
  const templates = ref<Bot[]>([])
  const myBots = ref<Bot[]>([])
  const botCount = ref(0)
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchBots() {
    loading.value = true
    error.value = null
    try {
      const response = await get('/api/v1/bots')
      if (response?.data) {
        bots.value = response.data
      }
    } catch (e: any) {
      error.value = e.message || '获取机器人列表失败'
    } finally {
      loading.value = false
    }
  }

  async function fetchTemplates() {
    loading.value = true
    error.value = null
    try {
      const response = await get('/api/v1/bots/templates')
      if (response?.data) {
        templates.value = response.data
      }
    } catch (e: any) {
      error.value = e.message || '获取模板列表失败'
    } finally {
      loading.value = false
    }
  }

  async function fetchMyBots() {
    loading.value = true
    error.value = null
    try {
      const response = await get('/api/v1/bots/my')
      if (response?.data) {
        myBots.value = response.data
      }
    } catch (e: any) {
      error.value = e.message || '获取我的机器人失败'
    } finally {
      loading.value = false
    }
  }

  async function fetchBotCount() {
    try {
      const response = await get('/api/v1/bots/my-count')
      if (response?.data) {
        botCount.value = response.data.count
      }
    } catch (e: any) {
      error.value = e.message || '获取数量失败'
    }
  }

  async function createBot(data: Partial<Bot>) {
    loading.value = true
    error.value = null
    try {
      const response = await post('/api/v1/bots', data)
      return response?.data
    } catch (e: any) {
      error.value = e.message || '创建机器人失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function updateBot(id: number, data: Partial<Bot>) {
    loading.value = true
    error.value = null
    try {
      const response = await put(`/api/v1/bots/${id}`, data)
      return response?.data
    } catch (e: any) {
      error.value = e.message || '更新机器人失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function deleteBot(id: number) {
    loading.value = true
    error.value = null
    try {
      await del(`/api/v1/bots/${id}`)
    } catch (e: any) {
      error.value = e.message || '删除机器人失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    bots,
    templates,
    myBots,
    botCount,
    loading,
    error,
    fetchBots,
    fetchTemplates,
    fetchMyBots,
    fetchBotCount,
    createBot,
    updateBot,
    deleteBot
  }
}
```

- [ ] **步骤 2：Commit**

```bash
git add src/composables/useBots.ts
git commit -m "refactor: use useRequest in useBots"
```

---

### 任务 14：前端 - 更新 AI_PROVIDERS 默认模型

**文件：**
- 修改：`src/types/ai.ts`

- [ ] **步骤 1：更新 AI_PROVIDERS 默认模型**

```typescript
// src/types/ai.ts

export const AI_PROVIDERS = [
  {
    id: 'openai',
    name: 'OpenAI',
    icon: 'openai.png',
    defaultModel: 'gpt-4o-mini',
    models: ['gpt-4o-mini', 'gpt-4o', 'gpt-4-turbo', 'gpt-3.5-turbo']
  },
  {
    id: 'anthropic',
    name: 'Anthropic Claude',
    icon: 'anthropic.png',
    defaultModel: 'claude-sonnet-4-20250514',
    models: ['claude-sonnet-4-20250514', 'claude-3-5-sonnet-20241022', 'claude-3-opus-20240229']
  },
  {
    id: 'deepseek',
    name: 'DeepSeek',
    icon: 'deepseek.png',
    defaultModel: 'deepseek-chat',
    models: ['deepseek-chat', 'deepseek-coder']
  },
  {
    id: 'qwen',
    name: '阿里通义千问',
    icon: 'qwen.png',
    defaultModel: 'qwen-turbo',
    models: ['qwen-turbo', 'qwen-plus', 'qwen-max']
  },
  {
    id: 'hunyuan',
    name: '腾讯混元',
    icon: 'hunyuan.png',
    defaultModel: 'hunyuan-lite',
    models: ['hunyuan-lite', 'hunyuan-standard', 'hunyuan-pro']
  },
  {
    id: 'doubao',
    name: '字节豆包',
    icon: 'doubao.png',
    defaultModel: 'doubao-pro-32k',
    models: ['doubao-pro-32k', 'doubao-pro-128k', 'doubao-lite-32k']
  }
]
```

- [ ] **步骤 2：Commit**

```bash
git add src/types/ai.ts
git commit -m "feat: update AI_PROVIDERS with latest models and add DeepSeek"
```

---

## 验收测试

### 后端测试

1. **用户类型测试**
   - 创建新用户，验证 type 默认为 'user'
   - 创建 Bot，验证虚拟用户创建成功
   - 查询用户列表，验证 type 字段正确

2. **Bot 对话测试**
   - 创建 Bot 会话
   - 发送消息，验证流式响应
   - 验证 Bot 回复使用虚拟用户 ID
   - 验证系统提示词生效

### 前端测试

1. **Bot 对话测试**
   - 选择 Bot 进入对话
   - 发送消息，验证流式显示
   - 验证 Markdown 正确渲染
   - 刷新页面，验证历史消息加载

2. **组件复用测试**
   - 验证 MarkdownRenderer 在多个组件中正常工作
   - 验证 ThinkingIndicator 正确显示

---

## 执行选项

计划已完成并保存到 `docs/superpowers/plans/2026-05-04-ai-assistant-optimization.md`。两种执行方式：

**1. 子代理驱动（推荐）** - 每个任务调度一个新的子代理，任务间进行审查，快速迭代

**2. 内联执行** - 在当前会话中使用 executing-plans 执行任务，批量执行并设有检查点

选哪种方式？
