---
name: "frontend-api-standards"
description: "QIM 前端项目 API 请求和用户信息获取规范。Invoke when 实现新功能需要调用 API 或获取用户信息时，确保代码一致性。"
---

# 前端 API 请求规范

本规范总结了 QIM 前端项目中 API 请求、用户信息获取、状态管理的标准做法，确保代码一致性和可维护性。

## 1. API 请求规范

### 1.1 使用 useRequest composable

**必须**使用项目提供的 `useRequest` composable，**禁止**直接使用 `fetch` 或 `axios`。

```typescript
// ✅ 正确做法
import { useRequest } from '../../composables/useRequest'

const { request, serverUrl, getToken } = useRequest()

// 发送消息
const response = await request(`/api/v1/conversations/${conversationId}/messages`, {
  method: 'POST',
  body: JSON.stringify({
    type: 'miniApp',
    content: JSON.stringify(miniApp)
  })
})

// ❌ 错误做法 - 不要直接使用 fetch
const response = await fetch(`${serverUrl.value}/api/v1/...`, {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`
  },
  body: JSON.stringify(data)
})
```

### 1.2 useRequest 提供的功能

`useRequest` composable 已内置以下功能，无需重复实现：

- **Token 自动注入**：自动从 localStorage 获取 token 并添加到 Authorization header
- **Content-Type 智能设置**：FormData 类型不设置 Content-Type
- **超时控制**：默认 30 秒超时，可自定义
- **错误处理**：
  - 401 自动抛出 UNAUTHORIZED
  - 403 权限不足提示
  - 429 请求频繁警告
  - 其他错误自动处理
- **URL 参数处理**：支持 `params` 配置项

### 1.3 标准请求模式

```typescript
import { useRequest } from '../../composables/useRequest'
import type { ApiResponse } from '../../composables/useRequest'

const { request } = useRequest()

// GET 请求
const loadConversations = async () => {
  const response = await request<ApiResponse>('/api/v1/conversations', {
    method: 'GET',
    params: { status: 'active' }
  })
  
  if (response.code === 0) {
    return response.data
  }
  throw new Error(response.message || '获取失败')
}

// POST 请求
const sendMessage = async (conversationId: string, content: string, type = 'text') => {
  const response = await request<ApiResponse>(`/api/v1/conversations/${conversationId}/messages`, {
    method: 'POST',
    body: JSON.stringify({ content, type })
  })
  
  if (response.code === 0) {
    return response.data
  }
  throw new Error(response.message || '发送失败')
}

// 带超时控制的请求
const uploadFile = async (formData: FormData) => {
  const response = await request<ApiResponse>('/api/v1/upload', {
    method: 'POST',
    body: formData,
    timeout: 60000  // 60 秒超时
  })
  
  return response.data
}
```

## 2. 用户信息获取规范

### 2.1 使用 useCurrentUser composable

**必须**使用 `useCurrentUser` composable 获取当前用户信息，**禁止**直接从 localStorage 读取和解析。

```typescript
// ✅ 正确做法
import { useCurrentUser } from '../../composables/useCurrentUser'

const { currentUser } = useCurrentUser()

// 获取用户 ID
const userId = currentUser.value?.id

// 获取用户昵称
const nickname = currentUser.value?.nickname || currentUser.value?.username

// ❌ 错误做法 - 不要直接从 localStorage 读取
const userStr = localStorage.getItem('user')
const currentUser = userStr ? JSON.parse(userStr) : null
```

### 2.2 useCurrentUser 提供的功能

- **自动同步**：用户信息变更时自动同步到 userProfile
- **角色解析**：自动解析 `isAdmin` 状态
- **头像处理**：提供 `getProfileAvatar(serverUrl)` 处理头像 URL
- **刷新机制**：提供 `refreshUser()` 从服务器刷新用户信息

## 3. 状态管理规范

### 3.1 使用 chatStore 管理聊天状态

**必须**使用 `useChatStore` 管理消息列表、会话列表等状态。

```typescript
// ✅ 正确做法
import { useChatStore } from '../../stores/chat'

const chatStore = useChatStore()

// 获取当前会话 ID
const conversationId = chatStore.currentConversationId

// 添加消息到本地列表
chatStore.receiveMessage(conversationId, newMessage, true)

// 标记会话已读
chatStore.markConversationRead(conversationId)

// ❌ 错误做法 - 不要直接操作 messages ref
messages.value.push(newMessage)
```

### 3.2 会话 ID 获取

```typescript
// ✅ 正确做法 - 从 store 获取
const chatStore = useChatStore()
const conversationId = chatStore.currentConversationId

// ✅ 正确做法 - 从 composable 获取
const { currentConversationId } = useConversation()

// ❌ 错误做法 - 不要从 props 层层传递或从 localStorage 读取
const conversationId = props.currentConversationId
```

## 4. 消息发送标准流程

### 4.1 标准消息发送流程

```typescript
import { useRequest } from '../../composables/useRequest'
import { useChatStore } from '../../stores/chat'
import { useCurrentUser } from '../../composables/useCurrentUser'

const { request } = useRequest()
const chatStore = useChatStore()
const { currentUser } = useCurrentUser()

const handleSendMessage = async (messageData: any) => {
  const conversationId = chatStore.currentConversationId
  if (!conversationId) {
    $message.warning('请先选择一个聊天会话')
    return
  }

  try {
    // 1. 发送请求到后端
    const response = await request(`/api/v1/conversations/${conversationId}/messages`, {
      method: 'POST',
      body: JSON.stringify({
        type: messageData.type,
        content: messageData.content
      })
    })

    // 2. 处理响应
    if (response.code === 0) {
      // 3. 构建消息对象
      const newMessage = {
        id: response.data.id?.toString() || Date.now().toString(),
        content: response.data.content,
        sender: {
          id: response.data.sender?.id?.toString() || currentUser.value?.id?.toString() || '',
          name: response.data.sender?.nickname || response.data.sender?.username || currentUser.value?.nickname || currentUser.value?.username || '',
          avatar: response.data.sender?.avatar || currentUser.value?.avatar || ''
        },
        timestamp: new Date().getTime(),
        type: response.data.type || messageData.type,
        isSelf: true,
        isRead: false,
        conversationId: conversationId,
        ...messageData.extraData  // 额外数据如 miniAppData, newsData 等
      }

      // 4. 更新本地消息列表
      chatStore.receiveMessage(conversationId, newMessage as any, true)

      // 5. 滚动到底部
      nextTick(() => {
        chatWindowRef.value?.scrollToBottom()
      })

      $message.success('发送成功')
    } else {
      throw new Error(response.message || '发送失败')
    }
  } catch (error) {
    console.error('发送消息失败:', error)
    $message.error('发送失败，请重试')
  }
}
```

## 5. 组件通信规范

### 5.1 Props 传递原则

- **简单状态**：使用 props 和 v-model
- **全局状态**：使用 composable 或 store，避免多层 props 传递
- **回调函数**：使用 emit 事件，**不要**通过 props 传递回调函数

```vue
<!-- ✅ 正确做法 - 使用 emit -->
<MiniAppManager
  v-model:showMiniAppList="showMiniAppList"
  @send="handleSendMessage"
/>

<!-- ❌ 错误做法 - 不要通过 props 传递回调 -->
<MiniAppManager
  :on-select="handleSendMessage"
/>
```

### 5.2 事件命名规范

- 使用 kebab-case 命名事件：`@send-message`, `@update:value`
- 使用 camelCase 命名方法：`handleSendMessage`, `handleUpdateValue`

## 6. 错误处理规范

### 6.1 统一错误处理

```typescript
try {
  const response = await request('/api/v1/...')
  
  if (response.code === 0) {
    // 成功处理
  } else {
    // 业务错误
    let errorMessage = '操作失败'
    if (response.code === 401) {
      errorMessage = '登录已过期，请重新登录'
      // 处理 session expired
    } else if (response.code === 403) {
      errorMessage = '权限不足'
    } else if (response.message) {
      errorMessage = response.message
    }
    $message.error(errorMessage)
  }
} catch (error) {
  console.error('请求失败:', error)
  $message.error('网络错误，请重试')
}
```

## 7. 示例：完整的小程序发送功能

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { useRequest } from '../../composables/useRequest'
import { useChatStore } from '../../stores/chat'
import { useCurrentUser } from '../../composables/useCurrentUser'

const { request } = useRequest()
const chatStore = useChatStore()
const { currentUser } = useCurrentUser()

const handleSendMiniApp = async (miniApp: any) => {
  const conversationId = chatStore.currentConversationId
  
  if (!conversationId) {
    $message.warning('请先选择一个聊天会话')
    return
  }

  try {
    const response = await request(`/api/v1/conversations/${conversationId}/messages`, {
      method: 'POST',
      body: JSON.stringify({
        type: 'miniApp',
        content: JSON.stringify(miniApp)
      })
    })

    if (response.code === 0) {
      const newMessage = {
        id: response.data.id?.toString() || Date.now().toString(),
        content: response.data.content || JSON.stringify(miniApp),
        sender: {
          id: response.data.sender?.id?.toString() || currentUser.value?.id?.toString() || '',
          name: response.data.sender?.nickname || response.data.sender?.username || currentUser.value?.nickname || currentUser.value?.username || '',
          avatar: response.data.sender?.avatar || currentUser.value?.avatar || ''
        },
        timestamp: new Date().getTime(),
        type: 'miniApp',
        isSelf: true,
        isRead: false,
        conversationId: conversationId,
        miniAppData: miniApp
      }

      chatStore.receiveMessage(conversationId, newMessage as any, true)
      $message.success(`小程序 "${miniApp.name}" 已发送`)
    } else {
      throw new Error(response.message || '发送失败')
    }
  } catch (error) {
    console.error('发送小程序消息失败:', error)
    $message.error('发送失败，请重试')
  }
}
</script>
```

## 8. 禁止事项

### 8.1 禁止直接使用 fetch

```typescript
// ❌ 禁止
const response = await fetch(`${serverUrl}/api/v1/...`, {
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('token')}`
  }
})

// ✅ 应该使用
const response = await request('/api/v1/...', { method: 'POST' })
```

### 8.2 禁止直接读取 localStorage

```typescript
// ❌ 禁止
const user = JSON.parse(localStorage.getItem('user') || '{}')
const token = localStorage.getItem('token')
const conversationId = localStorage.getItem('currentConversationId')

// ✅ 应该使用
const { currentUser } = useCurrentUser()
const { getToken } = useRequest()
const conversationId = chatStore.currentConversationId
```

### 8.3 禁止通过 props 传递回调函数

```vue
<!-- ❌ 禁止 -->
<ChildComponent :on-click="handleClick" />

<!-- ✅ 应该使用 emit -->
<ChildComponent @click="handleClick" />
```

## 总结

遵循本规范可以确保：
1. **代码一致性**：所有组件使用相同的 API 请求和用户信息获取方式
2. **易于维护**：集中管理 token、超时、错误处理等逻辑
3. **避免混乱**：统一的状态管理避免多处读取 localStorage 导致的不一致
4. **提高质量**：自动处理认证、超时、错误等常见问题
