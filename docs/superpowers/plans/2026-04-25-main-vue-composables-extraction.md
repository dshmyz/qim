# Main.vue Composables 提取实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将 Main.vue 中约 6000 行的 script 部分拆分为 8 个独立的 composables，使 Main.vue 精简到约 800 行。

**架构：** 按职责将 Main.vue 中的逻辑拆分为 8 个 composables，每个 composable 管理特定领域的状态和逻辑。Main.vue 作为组合层，调用这些 composables 并将状态和函数传递给模板。

**技术栈：** Vue 3 Composition API, TypeScript

---

### 任务 1：创建 composables 目录和 useNotifications

**文件：**
- 创建：`src/composables/useNotifications.ts`
- 修改：`src/views/Main.vue` - 添加导入和替换

- [ ] **步骤 1：创建 useNotifications.ts**

```typescript
import { ref, computed } from 'vue'

export function useNotifications() {
  const notifications = ref<any[]>([])
  const unreadNotificationCount = ref(0)
  const showNotificationCenter = ref(false)

  const filteredNotifications = computed(() => {
    return notifications.value.filter(n => !n.read)
  })

  function handleNotificationCenter() {
    showNotificationCenter.value = true
    unreadNotificationCount.value = 0
  }

  function handleNewNotification(notification: any) {
    console.log('收到新通知:', notification)
    const newNotification = {
      id: notification.id || Date.now().toString(),
      title: notification.title || '新通知',
      content: notification.content || '',
      timestamp: notification.timestamp || Date.now(),
      read: false,
      type: notification.type || 'system',
      data: notification.data || {}
    }
    notifications.value = [newNotification, ...notifications.value]
    unreadNotificationCount.value++
  }

  function handleNotificationClick(notification: any) {
    if (notification.type === 'message' && notification.data?.conversationId) {
      // 切换到消息会话
    } else if (notification.type === 'group' && notification.data?.groupId) {
      // 切换到群组
    }
  }

  function markAllNotificationsAsRead() {
    notifications.value.forEach(n => n.read = true)
    unreadNotificationCount.value = 0
  }

  return {
    notifications,
    unreadNotificationCount,
    showNotificationCenter,
    filteredNotifications,
    handleNotificationCenter,
    handleNewNotification,
    handleNotificationClick,
    markAllNotificationsAsRead
  }
}
```

- [ ] **步骤 2：在 Main.vue 中添加导入**

在 Main.vue 的 `<script setup>` 部分添加：

```typescript
import { useNotifications } from '@/composables/useNotifications'
```

- [ ] **步骤 3：在 Main.vue 中使用 composable**

在 Main.vue 中添加：

```typescript
const {
  notifications,
  unreadNotificationCount,
  showNotificationCenter,
  filteredNotifications,
  handleNotificationCenter,
  handleNewNotification,
  handleNotificationClick,
  markAllNotificationsAsRead
} = useNotifications()
```

- [ ] **步骤 4：删除 Main.vue 中的重复代码**

删除 Main.vue 中定义的 notifications、unreadNotificationCount、handleNewNotification 等相关代码。

- [ ] **步骤 5：Commit**

```bash
git add src/composables/useNotifications.ts src/views/Main.vue
git commit -m "refactor: 提取 useNotifications composable"
```

---

### 任务 2：创建 useAppState composable

**文件：**
- 创建：`src/composables/useAppState.ts`
- 修改：`src/views/Main.vue`

- [ ] **步骤 1：创建 useAppState.ts**

```typescript
import { ref } from 'vue'

export function useAppState() {
  const activeOption = ref('recent')
  const selectedAppId = ref<string | null>(null)
  const searchQuery = ref('')
  const searchResults = ref<any[]>([])
  const isLoading = ref(true)
  const showNetworkError = ref(false)
  const networkErrorMsg = ref('网络连接失败，请检查网络后重试')
  const sidebarCollapsed = ref(false)

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  function setLoading(loading: boolean) {
    isLoading.value = loading
  }

  function setNetworkError(show: boolean, msg?: string) {
    showNetworkError.value = show
    if (msg) networkErrorMsg.value = msg
  }

  return {
    activeOption,
    selectedAppId,
    searchQuery,
    searchResults,
    isLoading,
    showNetworkError,
    networkErrorMsg,
    sidebarCollapsed,
    toggleSidebar,
    setLoading,
    setNetworkError
  }
}
```

- [ ] **步骤 2：在 Main.vue 中导入和使用**

```typescript
import { useAppState } from '@/composables/useAppState'

const {
  activeOption,
  selectedAppId,
  searchQuery,
  searchResults,
  isLoading,
  showNetworkError,
  networkErrorMsg,
  sidebarCollapsed,
  toggleSidebar,
  setLoading,
  setNetworkError
} = useAppState()
```

- [ ] **步骤 3：删除 Main.vue 中的重复代码**

删除 Main.vue 中定义的 activeOption、isLoading 等相关变量。

- [ ] **步骤 4：Commit**

```bash
git add src/composables/useAppState.ts src/views/Main.vue
git commit -m "refactor: 提取 useAppState composable"
```

---

### 任务 3：创建 useUI composable

**文件：**
- 创建：`src/composables/useUI.ts`
- 修改：`src/views/Main.vue`

- [ ] **步骤 1：创建 useUI.ts**

```typescript
import { ref } from 'vue'

export function useUI() {
  const showMenu = ref(false)
  const menuPosition = ref({ x: 0, y: 0 })
  const selectedConversation = ref<any>(null)
  const showActionMenuFlag = ref(false)
  const actionMenuPosition = ref({ x: 0, y: 0 })
  const showUserContextMenuFlag = ref(false)
  const userContextMenuPosition = ref({ x: 0, y: 0 })
  const selectedEmployee = ref<any>(null)
  const showMemberContextMenuFlag = ref(false)
  const memberContextMenuPosition = ref({ x: 0, y: 0 })
  const showShareModal = ref(false)
  const shareType = ref('')
  const shareUsers = ref<any[]>([])
  const shareGroups = ref<any[]>([])
  const showUserProfile = ref(false)
  const selectedUser = ref<any>(null)
  const showCreateConversationModal = ref(false)
  const createConversationType = ref('group')
  const createConversationTitle = ref('')
  const showSystemMessageModal = ref(false)
  const systemMessage = ref({ title: '', content: '', target: 'all', groupId: '', userId: '' })
  const showGroupMembersModal = ref(false)
  const groupMembers = ref<any[]>([])
  const showInviteMembersModal = ref(false)

  function showContextMenu(event: MouseEvent, conversation: any) {
    showMenu.value = true
    menuPosition.value = { x: event.clientX, y: event.clientY }
    selectedConversation.value = conversation
  }

  function hideContextMenu() {
    showMenu.value = false
    selectedConversation.value = null
  }

  function showActionMenu(event: MouseEvent) {
    showActionMenuFlag.value = true
    actionMenuPosition.value = { x: event.clientX, y: event.clientY }
  }

  function hideActionMenu() {
    showActionMenuFlag.value = false
  }

  function showUserContextMenu(event: MouseEvent, user: any) {
    showUserContextMenuFlag.value = true
    userContextMenuPosition.value = { x: event.clientX, y: event.clientY }
    selectedEmployee.value = user
  }

  function hideUserContextMenu() {
    showUserContextMenuFlag.value = false
    selectedEmployee.value = null
  }

  function showShareModalFn(type: string, users: any[], groups: any[]) {
    showShareModal.value = true
    shareType.value = type
    shareUsers.value = users
    shareGroups.value = groups
  }

  function hideShareModal() {
    showShareModal.value = false
  }

  function closeUserProfile() {
    showUserProfile.value = false
    selectedUser.value = null
  }

  function openCreateGroupModal() {
    createConversationType.value = 'group'
    createConversationTitle.value = '创建群聊'
    showCreateConversationModal.value = true
    hideActionMenu()
  }

  function closeCreateConversationModal() {
    showCreateConversationModal.value = false
  }

  function openSystemMessageModal() {
    showSystemMessageModal.value = true
    hideActionMenu()
  }

  function closeSystemMessageModal() {
    showSystemMessageModal.value = false
    systemMessage.value = { title: '', content: '', target: 'all', groupId: '', userId: '' }
  }

  function showGroupMembersModalFn() {
    showGroupMembersModal.value = true
  }

  function closeGroupMembersModal() {
    showGroupMembersModal.value = false
  }

  function showInviteMembersModalFn() {
    showInviteMembersModal.value = true
  }

  function closeInviteMembersModal() {
    showInviteMembersModal.value = false
  }

  return {
    showMenu,
    menuPosition,
    selectedConversation,
    showActionMenuFlag,
    actionMenuPosition,
    showUserContextMenuFlag,
    userContextMenuPosition,
    selectedEmployee,
    showMemberContextMenuFlag,
    memberContextMenuPosition,
    showShareModal,
    shareType,
    shareUsers,
    shareGroups,
    showUserProfile,
    selectedUser,
    showCreateConversationModal,
    createConversationType,
    createConversationTitle,
    showSystemMessageModal,
    systemMessage,
    showGroupMembersModal,
    groupMembers,
    showInviteMembersModal,
    showContextMenu,
    hideContextMenu,
    showActionMenu,
    hideActionMenu,
    showUserContextMenu,
    hideUserContextMenu,
    showShareModalFn,
    hideShareModal,
    closeUserProfile,
    openCreateGroupModal,
    closeCreateConversationModal,
    openSystemMessageModal,
    closeSystemMessageModal,
    showGroupMembersModalFn,
    closeGroupMembersModal,
    showInviteMembersModalFn,
    closeInviteMembersModal
  }
}
```

- [ ] **步骤 2：在 Main.vue 中导入和使用**

```typescript
import { useUI } from '@/composables/useUI'

const ui = useUI()
// 解构需要的属性
const { showMenu, menuPosition, selectedConversation, ... } = ui
```

- [ ] **步骤 3：删除 Main.vue 中的重复代码**

删除 Main.vue 中定义的相关 UI 状态和函数。

- [ ] **步骤 4：Commit**

```bash
git add src/composables/useUI.ts src/views/Main.vue
git commit -m "refactor: 提取 useUI composable"
```

---

### 任务 4：创建 useConversation composable

**文件：**
- 创建：`src/composables/useConversation.ts`
- 修改：`src/views/Main.vue`

- [ ] **步骤 1：创建 useConversation.ts**

```typescript
import { ref, computed } from 'vue'

export function useConversation() {
  const conversations = ref<any[]>([])
  const currentConversationId = ref<string | null>(null)

  const currentConversation = computed(() => {
    return conversations.value.find(c => c.id === currentConversationId.value) || null
  })

  function handleConversationSelect(conversation: any) {
    currentConversationId.value = conversation.id
    // 加载消息等逻辑
  }

  function handleSwitchConversation(conversationId: string) {
    currentConversationId.value = conversationId
  }

  function handlePin(conversation: any) {
    conversation.pinned = !conversation.pinned
  }

  function handleMute(conversation: any) {
    conversation.muted = !conversation.muted
  }

  function handleRemove(conversation: any) {
    const index = conversations.value.findIndex(c => c.id === conversation.id)
    if (index !== -1) {
      conversations.value.splice(index, 1)
    }
  }

  return {
    conversations,
    currentConversationId,
    currentConversation,
    handleConversationSelect,
    handleSwitchConversation,
    handlePin,
    handleMute,
    handleRemove
  }
}
```

- [ ] **步骤 2：在 Main.vue 中导入和使用**

- [ ] **步骤 3：删除 Main.vue 中的重复代码**

- [ ] **步骤 4：Commit**

```bash
git add src/composables/useConversation.ts src/views/Main.vue
git commit -m "refactor: 提取 useConversation composable"
```

---

### 任务 5：创建 useChat composable

**文件：**
- 创建：`src/composables/useChat.ts`
- 修改：`src/views/Main.vue`

- [ ] **步骤 1：创建 useChat.ts**

```typescript
import { ref } from 'vue'

export function useChat() {
  const messages = ref<any[]>([])
  const hasMoreMessages = ref(true)

  async function loadMessages(conversationId: string) {
    // 加载消息逻辑
  }

  async function handleSendMessage(content: string) {
    // 发送消息逻辑
  }

  async function handleLoadMore() {
    // 加载更多消息逻辑
  }

  async function handleRetrySendMessage(message: any) {
    // 重发消息逻辑
  }

  function handleRecallMessage(message: any) {
    // 撤回消息逻辑
  }

  function getMessageReadUsers(message: any) {
    // 获取已读用户逻辑
  }

  return {
    messages,
    hasMoreMessages,
    loadMessages,
    handleSendMessage,
    handleLoadMore,
    handleRetrySendMessage,
    handleRecallMessage,
    getMessageReadUsers
  }
}
```

- [ ] **步骤 2：在 Main.vue 中导入和使用**

- [ ] **步骤 3：删除 Main.vue 中的重复代码**

- [ ] **步骤 4：Commit**

```bash
git add src/composables/useChat.ts src/views/Main.vue
git commit -m "refactor: 提取 useChat composable"
```

---

### 任务 6：创建 useWebSocket composable

**文件：**
- 创建：`src/composables/useWebSocket.ts`
- 修改：`src/views/Main.vue`

- [ ] **步骤 1：创建 useWebSocket.ts**

```typescript
import { ref, onUnmounted } from 'vue'

export function useWebSocket() {
  const ws = ref<WebSocket | null>(null)
  const isConnected = ref(false)
  const reconnectAttempts = ref(0)
  const maxReconnectAttempts = 5

  function connect(url: string) {
    // WebSocket 连接逻辑
  }

  function disconnect() {
    if (ws.value) {
      ws.value.close()
      ws.value = null
      isConnected.value = false
    }
  }

  function reconnect(url: string) {
    // 重连逻辑
  }

  function sendMessage(data: any) {
    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
      ws.value.send(JSON.stringify(data))
    }
  }

  onUnmounted(() => {
    disconnect()
  })

  return {
    ws,
    isConnected,
    connect,
    disconnect,
    reconnect,
    sendMessage
  }
}
```

- [ ] **步骤 2：在 Main.vue 中导入和使用**

- [ ] **步骤 3：删除 Main.vue 中的重复代码**

- [ ] **步骤 4：Commit**

```bash
git add src/composables/useWebSocket.ts src/views/Main.vue
git commit -m "refactor: 提取 useWebSocket composable"
```

---

### 任务 7：创建 useUserManagement composable

**文件：**
- 创建：`src/composables/useUserManagement.ts`
- 修改：`src/views/Main.vue`

- [ ] **步骤 1：创建 useUserManagement.ts**

```typescript
import { ref } from 'vue'

export function useUserManagement() {
  const currentUser = ref<any>(null)
  const orgStructure = ref<any>({})
  const selectedGroup = ref<any>(null)
  const selectedChannel = ref<any>(null)
  const allEmployees = ref<any[]>([])

  async function loadOrgStructure() {
    // 加载组织架构逻辑
  }

  async function loadAllEmployees() {
    // 加载所有员工逻辑
  }

  function handleUserClick(user: any) {
    // 用户点击逻辑
  }

  async function startPrivateChat(user: any) {
    // 发起私聊逻辑
  }

  function handleInviteMembers() {
    // 邀请成员逻辑
  }

  function handleChannelSelect(channel: any) {
    selectedChannel.value = channel
  }

  async function unsubscribeChannel(channel: any) {
    // 取消订阅频道逻辑
  }

  return {
    currentUser,
    orgStructure,
    selectedGroup,
    selectedChannel,
    allEmployees,
    loadOrgStructure,
    loadAllEmployees,
    handleUserClick,
    startPrivateChat,
    handleInviteMembers,
    handleChannelSelect,
    unsubscribeChannel
  }
}
```

- [ ] **步骤 2：在 Main.vue 中导入和使用**

- [ ] **步骤 3：删除 Main.vue 中的重复代码**

- [ ] **步骤 4：Commit**

```bash
git add src/composables/useUserManagement.ts src/views/Main.vue
git commit -m "refactor: 提取 useUserManagement composable"
```

---

### 任务 8：创建 useRTC composable

**文件：**
- 创建：`src/composables/useRTC.ts`
- 修改：`src/views/Main.vue`

- [ ] **步骤 1：创建 useRTC.ts**

```typescript
import { ref } from 'vue'

export function useRTC() {
  const remoteScreenSharing = ref(false)
  const remoteScreenUserId = ref<string | null>(null)
  const remoteScreenData = ref<any>(null)

  function handleScreenShareStart(data: any) {
    remoteScreenSharing.value = true
    remoteScreenUserId.value = data.userId
  }

  function handleScreenShareStop() {
    remoteScreenSharing.value = false
    remoteScreenUserId.value = null
    remoteScreenData.value = null
  }

  function handleScreenShareData(data: any) {
    remoteScreenData.value = data
  }

  return {
    remoteScreenSharing,
    remoteScreenUserId,
    remoteScreenData,
    handleScreenShareStart,
    handleScreenShareStop,
    handleScreenShareData
  }
}
```

- [ ] **步骤 2：在 Main.vue 中导入和使用**

- [ ] **步骤 3：删除 Main.vue 中的重复代码**

- [ ] **步骤 4：Commit**

```bash
git add src/composables/useRTC.ts src/views/Main.vue
git commit -m "refactor: 提取 useRTC composable"
```

---

### 任务 9：清理 Main.vue 和验证

**文件：**
- 修改：`src/views/Main.vue`

- [ ] **步骤 1：删除空注释**

删除 Main.vue L6634-6671 的空"主题注释"。

- [ ] **步骤 2：检查未使用的导入**

确保所有导入都被使用，删除未使用的。

- [ ] **步骤 3：验证功能完整性**

检查所有模板中使用的变量和函数是否都已正确定义。

- [ ] **步骤 4：运行 lint 和 typecheck**

```bash
cd /Users/gracegaoya/work/project/qim/qim-client && npm run lint && npm run typecheck
```

- [ ] **步骤 5：Commit**

```bash
git add src/views/Main.vue
git commit -m "refactor: 清理 Main.vue 并验证功能完整性"
```
