# 消息已读回执功能 - 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 实现消息已读回执功能，包括后端已读列表分页查询、WebSocket 实时状态同步、前端 IntersectionObserver 自动标记已读、节流批量推送、虚拟滚动已读列表

**架构：** 基于现有 `MessageReadReceipt` 模型和已有 handler 进行增强。后端增加分页查询和批量已读状态 API，优化 WebSocket 已读广播。前端新增 `useReadReceipt` composable 管理节流和自动标记，拆分已读用户弹窗为子组件，在 `MessageItem` 中集成 `IntersectionObserver`

**技术栈：** Go + Gin + GORM + SQLite（后端），Vue 3 + TypeScript + Composition API（前端）

---

## 文件结构

### 后端文件（qim-server）

| 文件 | 操作 | 职责 |
|------|------|------|
| `qim-server/model/model.go` | 不变 | `MessageReadReceipt` 模型已存在 |
| `qim-server/handler/message_handler.go` | 修改 | 增强 `GetMessageReadUsers` 分页；优化 `MarkConversationAsRead` 批量写入；新增 `GetMessageReadStatus` |
| `qim-server/ws/ws.go` | 修改 | 增强 `handleReadMessage` 支持批量消息 ID |
| `qim-server/app/routes.go` | 修改 | 新增 `POST /messages/read-status` 路由 |
| `qim-server/app/init.go` | 修改 | 添加已读回执索引迁移 |

### 前端文件（qim-client）

| 文件 | 操作 | 职责 |
|------|------|------|
| `qim-client/src/composables/useReadReceipt.ts` | 创建 | 已读回执核心 composable |
| `qim-client/src/composables/useReadUsersModal.ts` | 创建 | 已读列表弹窗状态管理 |
| `qim-client/src/components/read-users/ReadUserItem.vue` | 创建 | 已读用户单行子组件 |
| `qim-client/src/components/read-users/ReadUsersList.vue` | 创建 | 已读列表容器（虚拟滚动） |
| `qim-client/src/components/chat/ReadUsersModal.vue` | 重构 | 弹窗容器，引用子组件 |
| `qim-client/src/components/message/MessageItem.vue` | 修改 | 集成 IntersectionObserver |
| `qim-client/src/components/chat/ChatWindow.vue` | 修改 | 集成 composable + WebSocket + ReadUsersModal |
| `qim-client/src/types/index.ts` | 修改 | 新增已读回执类型定义 |

---

## 后端实现

### 任务 1：增强 GetMessageReadUsers 分页查询

**文件：**
- 修改：`qim-server/handler/message_handler.go:668-718`

- [ ] **步骤 1：替换 GetMessageReadUsers 函数为分页版本**

```go
func GetMessageReadUsers(c *gin.Context) {
	userID, _ := c.Get("user_id")
	msgIDStr := c.Param("id")

	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息ID")
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page := 1
	pageSize := 20
	if p, e := strconv.Atoi(pageStr); e == nil && p > 0 {
		page = p
	}
	if ps, e := strconv.Atoi(pageSizeStr); e == nil && ps > 0 && ps <= 50 {
		pageSize = ps
	}
	offset := (page - 1) * pageSize

	db := database.GetDB()

	var msg model.Message
	if err := db.First(&msg, uint(msgID)).Error; err != nil {
		response.NotFound(c, "消息不存在")
		return
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", msg.ConversationID, userID).First(&member).Error; err != nil {
		response.Forbidden(c, "无权限访问")
		return
	}

	var total int64
	db.Model(&model.MessageReadReceipt{}).Where("message_id = ?", msgID).Count(&total)

	var readReceipts []model.MessageReadReceipt
	db.Where("message_id = ?", msgID).
		Preload("User").
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&readReceipts)

	var readUsers []map[string]interface{}
	for _, receipt := range readReceipts {
		if receipt.User != nil {
			name := receipt.User.Nickname
			if name == "" {
				name = receipt.User.Username
			}
			readUsers = append(readUsers, map[string]interface{}{
				"id":       receipt.User.ID,
				"username": receipt.User.Username,
				"nickname": name,
				"avatar":   receipt.User.Avatar,
				"read_at":  receipt.CreatedAt,
			})
		}
	}

	var totalMembers int64
	db.Model(&model.ConversationMember{}).Where("conversation_id = ?", msg.ConversationID).Count(&totalMembers)

	readCount := total
	if readCount == 0 {
		readCount = int64(len(readReceipts))
	}

	response.Success(c, gin.H{
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
		"read_users":    readUsers,
		"total_members": totalMembers,
		"read_count":    readCount,
		"unread_count":  totalMembers - readCount - 1,
	})
}
```

- [ ] **步骤 2：编译验证**

```bash
cd qim-server && go build ./...
```
预期：编译成功

- [ ] **步骤 3：Commit**

```bash
cd qim-server && git add handler/message_handler.go && git commit -m "feat: 已读用户列表增加分页查询支持"
```

---

### 任务 2：新增批量已读状态查询 API

**文件：**
- 修改：`qim-server/handler/message_handler.go`（末尾添加）
- 修改：`qim-server/app/routes.go`（添加路由）

- [ ] **步骤 1：在 message_handler.go 末尾添加 GetMessageReadStatus**

```go
func GetMessageReadStatus(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		MessageIDs []uint `json:"message_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if len(req.MessageIDs) == 0 {
		response.Success(c, gin.H{"statuses": []interface{}{}})
		return
	}

	db := database.GetDB()

	type ReadStatus struct {
		MessageID    uint `json:"message_id"`
		ReadCount    int  `json:"read_count"`
		TotalMembers int  `json:"total_members"`
		IsReadByMe   bool `json:"is_read_by_me"`
	}

	statuses := make([]ReadStatus, 0, len(req.MessageIDs))

	for _, msgID := range req.MessageIDs {
		var msg model.Message
		if err := db.First(&msg, msgID).Error; err != nil {
			continue
		}

		var readCount int64
		db.Model(&model.MessageReadReceipt{}).Where("message_id = ?", msgID).Count(&readCount)

		var totalMembers int64
		db.Model(&model.ConversationMember{}).Where("conversation_id = ?", msg.ConversationID).Count(&totalMembers)

		var readCountForMe int64
		db.Model(&model.MessageReadReceipt{}).Where("message_id = ? AND user_id = ?", msgID, userID).Count(&readCountForMe)

		statuses = append(statuses, ReadStatus{
			MessageID:    msgID,
			ReadCount:    int(readCount),
			TotalMembers: int(totalMembers),
			IsReadByMe:   readCountForMe > 0,
		})
	}

	response.Success(c, gin.H{"statuses": statuses})
}
```

- [ ] **步骤 2：在 routes.go 中添加路由**

在 `authed.GET("/messages/:id/quote-chain", handler.GetMessageQuoteChain)` 之后添加：

```go
authed.POST("/messages/read-status", handler.GetMessageReadStatus)
```

- [ ] **步骤 3：编译验证**

```bash
cd qim-server && go build ./...
```

- [ ] **步骤 4：Commit**

```bash
cd qim-server && git add handler/message_handler.go app/routes.go && git commit -m "feat: 新增批量已读状态查询 API"
```

---

### 任务 3：优化 MarkConversationAsRead 批量写入

**文件：**
- 修改：`qim-server/handler/message_handler.go:720-818`

- [ ] **步骤 1：替换 MarkConversationAsRead 函数**

```go
func MarkConversationAsRead(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	if strings.HasPrefix(convIDStr, "conv_") {
		convIDStr = strings.TrimPrefix(convIDStr, "conv_")
	}

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的会话ID")
		return
	}

	var req struct {
		MessageIDs []uint `json:"message_ids"`
	}
	c.ShouldBindJSON(&req)

	db := database.GetDB()

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&member).Error; err != nil {
		response.Forbidden(c, "无权限访问")
		return
	}

	query := db.Model(&model.Message{}).Where("conversation_id = ? AND sender_id != ?", uint(convID), userID)
	if len(req.MessageIDs) > 0 {
		query = query.Where("id IN ?", req.MessageIDs)
	}

	var messages []model.Message
	query.Find(&messages)

	if len(messages) == 0 {
		response.Success(c, gin.H{"marked_count": 0, "conversation_id": convID})
		return
	}

	tx := db.Begin()
	for _, msg := range messages {
		tx.Exec("INSERT OR IGNORE INTO message_reads (message_id, conversation_id, user_id, created_at) VALUES (?, ?, ?, ?)",
			msg.ID, msg.ConversationID, userID, time.Now())
	}

	tx.Model(&model.Message{}).
		Where("conversation_id = ? AND sender_id != ? AND is_read = false", uint(convID), userID).
		UpdateColumn("is_read", true)

	tx.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", uint(convID), userID).
		UpdateColumn("unread_count", 0)

	now := time.Now()
	tx.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", uint(convID), userID).
		UpdateColumn("last_read_at", now)

	tx.Commit()

	var conv model.Conversation
	db.First(&conv, uint(convID))

	messageIDs := make([]uint, len(messages))
	for i, msg := range messages {
		messageIDs[i] = msg.ID
	}

	if ws.GlobalHub != nil {
		readMsg := ws.WSMessage{
			Type: "read_receipt",
			Data: map[string]interface{}{
				"conversation_id": convID,
				"message_ids":     messageIDs,
				"user_id":         userID,
				"read_at":         now,
			},
		}
		jsonMsg, _ := json.Marshal(readMsg)

		if conv.Type == "single" {
			var otherMember model.ConversationMember
			db.Where("conversation_id = ? AND user_id != ?", uint(convID), userID).First(&otherMember)
			ws.GlobalHub.SendToUser(otherMember.UserID, jsonMsg)
		} else if conv.Type == "group" {
			senderIDs := make(map[uint]bool)
			for _, msg := range messages {
				senderIDs[msg.SenderID] = true
			}
			delete(senderIDs, userID.(uint))
			for senderID := range senderIDs {
				ws.GlobalHub.SendToUser(senderID, jsonMsg)
			}
		}
	}

	response.Success(c, gin.H{
		"marked_count":    len(messages),
		"conversation_id": convID,
	})
}
```

- [ ] **步骤 2：编译验证**

```bash
cd qim-server && go build ./...
```

- [ ] **步骤 3：Commit**

```bash
cd qim-server && git add handler/message_handler.go && git commit -m "perf: 优化标记已读为批量写入 + 完善 WebSocket 广播"
```

---

### 任务 4：增强 WebSocket handleReadMessage

**文件：**
- 修改：`qim-server/ws/ws.go:436-504`

- [ ] **步骤 1：替换 handleReadMessage 函数**

```go
func handleReadMessage(c *Client, data interface{}) {
	db := database.GetDB()

	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	convIDFloat, _ := msgData["conversation_id"].(float64)
	convID := uint(convIDFloat)

	var messageIDs []uint
	if msgIDFloat, ok := msgData["message_id"].(float64); ok {
		messageIDs = []uint{uint(msgIDFloat)}
	} else if ids, ok := msgData["message_ids"].([]interface{}); ok {
		for _, id := range ids {
			if floatID, ok := id.(float64); ok {
				messageIDs = append(messageIDs, uint(floatID))
			}
		}
	}

	if len(messageIDs) == 0 {
		return
	}

	db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, c.userID).
		Updates(map[string]interface{}{
			"unread_count": 0,
			"last_read_at": time.Now(),
		})

	for _, msgID := range messageIDs {
		var existing model.MessageReadReceipt
		err := db.Where("message_id = ? AND user_id = ?", msgID, c.userID).First(&existing).Error
		if err != nil {
			db.Create(&model.MessageReadReceipt{
				MessageID:      msgID,
				ConversationID: convID,
				UserID:         c.userID,
			})
		}
	}

	result := db.Model(&model.Message{}).
		Where("conversation_id = ? AND sender_id != ? AND id IN ? AND is_read = false", convID, c.userID, messageIDs).
		UpdateColumn("is_read", true)

	if result.RowsAffected > 0 {
		var conv model.Conversation
		db.First(&conv, convID)

		readMsg := WSMessage{
			Type: "read_receipt",
			Data: map[string]interface{}{
				"conversation_id": convID,
				"message_ids":     messageIDs,
				"user_id":         c.userID,
				"timestamp":       time.Now().Unix(),
			},
		}
		jsonMsg, _ := json.Marshal(readMsg)

		if conv.Type == "single" {
			var otherMember model.ConversationMember
			db.Where("conversation_id = ? AND user_id != ?", convID, c.userID).First(&otherMember)
			c.hub.SendToUser(otherMember.UserID, jsonMsg)
		} else if conv.Type == "group" {
			var messages []model.Message
			db.Where("id IN ?", messageIDs).Find(&messages)
			senderIDs := make(map[uint]bool)
			for _, msg := range messages {
				if msg.SenderID != c.userID {
					senderIDs[msg.SenderID] = true
				}
			}
			for senderID := range senderIDs {
				c.hub.SendToUser(senderID, jsonMsg)
			}
		}
	}
}
```

- [ ] **步骤 2：编译验证**

```bash
cd qim-server && go build ./...
```

- [ ] **步骤 3：Commit**

```bash
cd qim-server && git add ws/ws.go && git commit -m "feat: 增强 WebSocket 已读消息处理支持批量消息 ID"
```

---

## 前端实现

### 任务 5：新增类型定义

**文件：**
- 修改：`qim-client/src/types/index.ts`

- [ ] **步骤 1：在文件末尾添加**

```typescript
export interface ReadUser {
  id: number
  username: string
  nickname: string
  avatar: string
  read_at: string
}

export interface ReadUsersResponse {
  total: number
  page: number
  page_size: number
  read_users: ReadUser[]
  total_members: number
  read_count: number
  unread_count: number
}

export interface MessageReadStatus {
  message_id: number
  read_count: number
  total_members: number
  is_read_by_me: boolean
}
```

- [ ] **步骤 2：Commit**

```bash
cd qim-client && git add src/types/index.ts && git commit -m "feat: 添加已读回执 TypeScript 类型定义"
```

---

### 任务 6：创建 useReadReceipt composable

**文件：**
- 创建：`qim-client/src/composables/useReadReceipt.ts`

- [ ] **步骤 1：创建文件**

```typescript
import { ref, onUnmounted } from 'vue'

export function useReadReceipt(request: (url: string, options?: any) => Promise<any>) {
  const READ_THROTTLE_MS = 300

  const pendingReads = new Map<string, Set<number>>()
  let throttleTimer: number | null = null
  const trackedMessageIds = new Set<string>()

  const readStatusMap = ref<Map<number, { read_count: number; total_members: number }>>(new Map())

  const queueMarkRead = (conversationId: string, messageId: number) => {
    const key = `${conversationId}-${messageId}`
    if (trackedMessageIds.has(key)) return
    trackedMessageIds.add(key)

    if (!pendingReads.has(conversationId)) {
      pendingReads.set(conversationId, new Set())
    }
    pendingReads.get(conversationId)!.add(messageId)

    if (throttleTimer) clearTimeout(throttleTimer)
    throttleTimer = window.setTimeout(flushPendingReads, READ_THROTTLE_MS)
  }

  const flushPendingReads = async () => {
    if (pendingReads.size === 0) return

    const readsToFlush = new Map(pendingReads)
    pendingReads.clear()
    throttleTimer = null

    for (const [conversationId, messageIds] of readsToFlush) {
      const ids = Array.from(messageIds)
      if (ids.length === 0) continue

      try {
        await request(`/api/v1/conversations/${conversationId}/read`, {
          method: 'POST',
          body: JSON.stringify({ message_ids: ids }),
        })
      } catch (error) {
        console.error('批量标记已读失败:', error)
        for (const msgId of ids) {
          trackedMessageIds.delete(`${conversationId}-${msgId}`)
          queueMarkRead(conversationId, msgId)
        }
      }
    }
  }

  const handleReadReceipt = (data: any) => {
    const { message_ids, user_id } = data
    if (message_ids && Array.isArray(message_ids)) {
      const newMap = new Map(readStatusMap.value)
      for (const msgId of message_ids) {
        const existing = newMap.get(msgId)
        if (existing) {
          newMap.set(msgId, {
            read_count: existing.read_count + 1,
            total_members: existing.total_members,
          })
        }
      }
      readStatusMap.value = newMap
    }
  }

  const cleanup = () => {
    if (throttleTimer) {
      clearTimeout(throttleTimer)
      throttleTimer = null
    }
    flushPendingReads()
    pendingReads.clear()
    trackedMessageIds.clear()
  }

  onUnmounted(() => {
    cleanup()
  })

  return {
    readStatusMap,
    queueMarkRead,
    flushPendingReads,
    handleReadReceipt,
    cleanup,
  }
}
```

- [ ] **步骤 2：Commit**

```bash
cd qim-client && git add src/composables/useReadReceipt.ts && git commit -m "feat: 创建 useReadReceipt composable"
```

---

### 任务 7：创建 useReadUsersModal composable

**文件：**
- 创建：`qim-client/src/composables/useReadUsersModal.ts`

- [ ] **步骤 1：创建文件**

```typescript
import { ref } from 'vue'
import type { ReadUser, ReadUsersResponse } from '../types'

export function useReadUsersModal(request: (url: string, options?: any) => Promise<any>) {
  const PAGE_SIZE = 20

  const visible = ref(false)
  const currentMessageId = ref<number | null>(null)
  const readUsers = ref<ReadUser[]>([])
  const totalPages = ref(0)
  const currentPage = ref(1)
  const totalMembers = ref(0)
  const readCount = ref(0)
  const unreadCount = ref(0)
  const isLoading = ref(false)
  const hasMore = ref(true)

  const openModal = async (messageId: number) => {
    currentMessageId.value = messageId
    visible.value = true
    readUsers.value = []
    currentPage.value = 1
    hasMore.value = true
    await fetchReadUsers(1)
  }

  const closeModal = () => {
    visible.value = false
    currentMessageId.value = null
    readUsers.value = []
  }

  const fetchReadUsers = async (page: number = 1) => {
    if (!currentMessageId.value || isLoading.value) return

    isLoading.value = true
    try {
      const res = await request(
        `/api/v1/messages/${currentMessageId.value}/read-users?page=${page}&page_size=${PAGE_SIZE}`
      )

      if (res.code === 0) {
        const data = res.data as ReadUsersResponse

        if (page === 1) {
          readUsers.value = data.read_users || []
        } else {
          readUsers.value = [...readUsers.value, ...(data.read_users || [])]
        }

        totalPages.value = Math.ceil(data.total / PAGE_SIZE)
        currentPage.value = page
        totalMembers.value = data.total_members
        readCount.value = data.read_count
        unreadCount.value = data.unread_count
        hasMore.value = page < totalPages.value
      }
    } catch (error) {
      console.error('获取已读用户列表失败:', error)
    } finally {
      isLoading.value = false
    }
  }

  const loadMore = () => {
    if (hasMore.value && !isLoading.value) {
      fetchReadUsers(currentPage.value + 1)
    }
  }

  return {
    visible,
    currentMessageId,
    readUsers,
    totalPages,
    currentPage,
    totalMembers,
    readCount,
    unreadCount,
    isLoading,
    hasMore,
    openModal,
    closeModal,
    fetchReadUsers,
    loadMore,
  }
}
```

- [ ] **步骤 2：Commit**

```bash
cd qim-client && git add src/composables/useReadUsersModal.ts && git commit -m "feat: 创建 useReadUsersModal composable"
```

---

### 任务 8：创建 ReadUserItem 子组件

**文件：**
- 创建：`qim-client/src/components/read-users/ReadUserItem.vue`

- [ ] **步骤 1：创建文件**

```vue
<template>
  <div class="read-user-item">
    <img
      :src="avatarUrl"
      :alt="user.nickname || user.username"
      class="read-user-avatar"
    />
    <div class="read-user-info">
      <span class="read-user-name">{{ user.nickname || user.username }}</span>
      <span class="read-user-time">{{ formatReadTime(user.read_at) }}</span>
    </div>
    <i class="fas fa-check read-icon"></i>
  </div>
</template>

<script setup lang="ts">
import type { ReadUser } from '../../types'
import { computed } from 'vue'
import { getAvatarUrl } from '../../utils/avatar'

const props = defineProps<{
  user: ReadUser
  serverUrl?: string
}>()

const avatarUrl = computed(() => {
  if (props.user.avatar && props.user.avatar.startsWith('http')) {
    return props.user.avatar
  }
  return getAvatarUrl(
    props.user.avatar || '',
    props.user.nickname || props.user.username,
    props.serverUrl || ''
  )
})

const formatReadTime = (readAt: string) => {
  const date = new Date(readAt)
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}
</script>

<style scoped>
.read-user-item {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  background: var(--list-bg, #f8f9fa);
  border-radius: 8px;
  transition: all 0.2s;
}

.read-user-item:hover {
  background: var(--hover-color, #e9ecef);
}

.read-user-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  object-fit: cover;
  margin-right: 12px;
  flex-shrink: 0;
}

.read-user-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.read-user-name {
  font-size: 14px;
  color: var(--text-color, #333);
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.read-user-time {
  font-size: 11px;
  color: var(--text-secondary, #999);
}

.read-icon {
  color: #4caf50;
  font-size: 14px;
  flex-shrink: 0;
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
cd qim-client && git add src/components/read-users/ReadUserItem.vue && git commit -m "feat: 创建 ReadUserItem 子组件"
```

---

### 任务 9：创建 ReadUsersList 子组件（虚拟滚动）

**文件：**
- 创建：`qim-client/src/components/read-users/ReadUsersList.vue`

- [ ] **步骤 1：创建文件**

```vue
<template>
  <div class="read-users-list" ref="scrollContainer" @scroll="handleScroll">
    <div v-if="readUsers.length === 0 && !isLoading" class="empty-read">
      暂无已读用户
    </div>

    <div class="virtual-scroll-container" :style="{ height: totalHeight + 'px' }">
      <div
        class="virtual-scroll-items"
        :style="{ transform: `translateY(${offsetY}px)`, height: visibleHeight + 'px' }"
      >
        <ReadUserItem
          v-for="user in visibleUsers"
          :key="user.id"
          :user="user"
          :server-url="serverUrl"
        />
      </div>
    </div>

    <div v-if="isLoading && readUsers.length > 0" class="loading-more">
      <i class="fas fa-spinner fa-spin"></i>
      <span>加载中...</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import ReadUserItem from './ReadUserItem.vue'
import type { ReadUser } from '../../types'

const props = defineProps<{
  readUsers: ReadUser[]
  totalMembers: number
  isLoading: boolean
  hasMore: boolean
  serverUrl: string
}>()

const emit = defineEmits<{
  loadMore: []
}>()

const scrollContainer = ref<HTMLElement | null>(null)

const ITEM_HEIGHT = 56
const VISIBLE_COUNT = 10
const BUFFER_COUNT = 3

const totalHeight = computed(() => props.readUsers.length * ITEM_HEIGHT)
const visibleHeight = computed(() => (VISIBLE_COUNT + BUFFER_COUNT * 2) * ITEM_HEIGHT)

const scrollTop = ref(0)
const startIndex = ref(0)

const visibleUsers = computed(() => {
  const end = Math.min(startIndex.value + VISIBLE_COUNT + BUFFER_COUNT * 2, props.readUsers.length)
  return props.readUsers.slice(startIndex.value, end)
})

const offsetY = computed(() => startIndex.value * ITEM_HEIGHT)

const handleScroll = () => {
  if (!scrollContainer.value) return
  scrollTop.value = scrollContainer.value.scrollTop

  const newStartIndex = Math.max(0, Math.floor(scrollTop.value / ITEM_HEIGHT) - BUFFER_COUNT)
  startIndex.value = newStartIndex

  const scrollBottom = scrollContainer.value.scrollHeight - scrollContainer.value.scrollTop - scrollContainer.value.clientHeight
  if (scrollBottom < 200 && props.hasMore && !props.isLoading) {
    emit('loadMore')
  }
}

watch(() => props.readUsers.length, () => {
  startIndex.value = 0
  if (scrollContainer.value) {
    scrollContainer.value.scrollTop = 0
  }
})
</script>

<style scoped>
.read-users-list {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.empty-read {
  text-align: center;
  color: var(--text-secondary, #999);
  padding: 24px;
  font-size: 14px;
}

.virtual-scroll-container {
  position: relative;
  width: 100%;
}

.virtual-scroll-items {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  overflow: hidden;
}

.loading-more {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px;
  color: var(--text-secondary, #999);
  font-size: 13px;
}

.loading-more .fa-spinner {
  color: var(--primary-color, #3b82f6);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
cd qim-client && git add src/components/read-users/ReadUsersList.vue && git commit -m "feat: 创建 ReadUsersList 子组件（虚拟滚动）"
```

---

### 任务 10：重构 ReadUsersModal 使用子组件

**文件：**
- 修改：`qim-client/src/components/chat/ReadUsersModal.vue`

- [ ] **步骤 1：完整替换**

```vue
<template>
  <div v-if="visible" class="read-users-modal" @click="handleClose">
    <div class="read-users-content" @click.stop>
      <div class="read-users-header">
        <h3>已读人员 ({{ readCount }}/{{ Math.max(0, totalMembers - 1) }})</h3>
        <button class="close-btn" @click.stop="handleClose">&times;</button>
      </div>

      <ReadUsersList
        :read-users="readUsers"
        :total-members="totalMembers"
        :is-loading="isLoading"
        :has-more="hasMore"
        :server-url="serverUrl"
        @load-more="loadMore"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import ReadUsersList from '../read-users/ReadUsersList.vue'
import { useReadUsersModal } from '../../composables/useReadUsersModal'

const props = defineProps<{
  visible: boolean
  messageId: number | null
  serverUrl: string
  request: (url: string, options?: any) => Promise<any>
}>()

const emit = defineEmits<{
  close: []
}>()

const {
  readUsers,
  totalMembers,
  readCount,
  isLoading,
  hasMore,
  loadMore,
} = useReadUsersModal(props.request)

const handleClose = () => {
  emit('close')
}
</script>

<style scoped>
.read-users-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.read-users-content {
  background: var(--sidebar-bg);
  border-radius: 12px;
  width: 360px;
  max-height: 480px;
  display: flex;
  flex-direction: column;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  overflow: hidden;
}

.read-users-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background: var(--panel-bg);
  border-bottom: 1px solid var(--border-color);
}

.read-users-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color);
}

.close-btn {
  background: none;
  border: none;
  font-size: 24px;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0 4px;
  line-height: 1;
}

.close-btn:hover {
  color: var(--text-color);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
cd qim-client && git add src/components/chat/ReadUsersModal.vue && git commit -m "refactor: 重构 ReadUsersModal 使用子组件 + composable"
```

---

### 任务 11：在 MessageItem 中集成 IntersectionObserver

**文件：**
- 修改：`qim-client/src/components/message/MessageItem.vue`

- [ ] **步骤 1：修改 props 定义**

在现有 props 中添加：
```typescript
conversationId?: string
onMessageVisible?: (messageId: number) => void
```

- [ ] **步骤 2：添加 IntersectionObserver 逻辑**

在 script setup 中（isAIMessage 计算属性之后）添加：

```typescript
import { ref, onMounted, onBeforeUnmount } from 'vue'

const messageRef = ref<HTMLElement | null>(null)
const hasMarkedRead = ref(false)

onMounted(() => {
  if (!props.isSelf && !props.isRecalled && props.conversationId && !hasMarkedRead.value) {
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting && entry.intersectionRatio >= 0.5) {
            hasMarkedRead.value = true
            props.onMessageVisible?.(props.message.id)
            observer.unobserve(entry.target)
          }
        })
      },
      { threshold: 0.5, rootMargin: '0px 0px -50px 0px' }
    )

    if (messageRef.value) {
      observer.observe(messageRef.value)
    }

    onBeforeUnmount(() => {
      observer.disconnect()
    })
  }
})
```

- [ ] **步骤 3：在 template 根 div 上添加 ref**

将 `<div class="message-item" ...>` 改为 `<div ref="messageRef" class="message-item" ...>`

- [ ] **步骤 4：Commit**

```bash
cd qim-client && git add src/components/message/MessageItem.vue && git commit -m "feat: MessageItem 集成 IntersectionObserver 自动标记已读"
```

---

### 任务 12：在 ChatWindow 中集成已读回执系统

**文件：**
- 修改：`qim-client/src/components/chat/ChatWindow.vue`
- 修改：`qim-client/src/components/chat/MessageListView.vue`

- [ ] **步骤 1：在 ChatWindow script 中导入并初始化**

在现有 import 区域添加：
```typescript
import { useReadReceipt } from '../../composables/useReadReceipt'
import { useReadUsersModal } from '../../composables/useReadUsersModal'
import { addWsHandler } from '../../composables/useWebSocket'
import ReadUsersModal from './ReadUsersModal.vue'
```

在 `const request = useRequest()` 之后添加：
```typescript
const {
  readStatusMap,
  queueMarkRead,
  handleReadReceipt: handleWsReadReceipt,
  cleanup: cleanupReadReceipt,
} = useReadReceipt(request)

const {
  visible: readUsersModalVisible,
  currentMessageId: readUsersModalMessageId,
  openModal: openReadUsersModal,
  closeModal: closeReadUsersModal,
  readUsers: modalReadUsers,
  totalMembers: modalTotalMembers,
  readCount: modalReadCount,
  isLoading: modalIsLoading,
  hasMore: modalHasMore,
  loadMore: loadMoreReadUsers,
} = useReadUsersModal(request)
```

- [ ] **步骤 2：替换 showReadUsers 方法**

```typescript
const showReadUsers = async (message: any) => {
  await openReadUsersModal(message.id)
}
```

- [ ] **步骤 3：添加 WebSocket read_receipt 处理器**

在 `connectWebSocket` 的 messageHandlers 中添加：
```typescript
'read_receipt': (data: any) => {
  handleWsReadReceipt(data)
}
```

- [ ] **步骤 4：在 template 中添加 ReadUsersModal**

在 ChatWindow template 底部添加：
```vue
<ReadUsersModal
  :visible="readUsersModalVisible"
  :message-id="readUsersModalMessageId"
  :server-url="serverUrl"
  :request="request"
  @close="closeReadUsersModal"
/>
```

- [ ] **步骤 5：在 MessageListView 中传递回调**

修改 ChatWindow 中 MessageListView 的使用，添加：
```vue
:conversation-id="currentConversationId"
:on-message-visible="onMessageVisible"
```

其中 `onMessageVisible` 定义为：
```typescript
const onMessageVisible = (messageId: number) => {
  if (currentConversationId.value) {
    queueMarkRead(String(currentConversationId.value), messageId)
  }
}
```

- [ ] **步骤 6：修改 MessageListView.vue 传递回调到 MessageItem**

在 MessageListView 中找到 MessageItem 的渲染位置，传递 `onMessageVisible` 和 `conversationId`。

- [ ] **步骤 7：Commit**

```bash
cd qim-client && git add src/components/chat/ChatWindow.vue src/components/chat/MessageListView.vue && git commit -m "feat: ChatWindow 集成已读回执系统 + WebSocket 实时同步"
```

---

### 任务 13：添加数据库索引迁移

**文件：**
- 修改：`qim-server/app/init.go:40-70`

- [ ] **步骤 1：在 MigrateDB 末尾、migrateMiniApps 之前添加**

```go
if db.Migrator().HasTable("message_reads") {
	if !db.Migrator().HasIndex(&model.MessageReadReceipt{}, "MessageID") {
		db.Migrator().CreateIndex(&model.MessageReadReceipt{}, "MessageID")
	}
}
```

- [ ] **步骤 2：编译验证**

```bash
cd qim-server && go build ./...
```

- [ ] **步骤 3：Commit**

```bash
cd qim-server && git add app/init.go && git commit -m "feat: 添加已读回执表索引迁移"
```
