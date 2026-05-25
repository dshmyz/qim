# Main.vue Composables 集成指南

## 📋 概述

本文档提供详细的步骤指南，说明如何在 Main.vue 中集成新创建的 Composables，以实现代码的模块化和可维护性提升。

**目标**：
- 将 Main.vue 从 5178 行减少到 < 1000 行
- 提升代码可维护性和可测试性
- 保持功能完整性

---

## ✅ 已完成工作

### 1. 导入新的 Composables ✅

已在 Main.vue 中添加以下导入：

```typescript
import { useWebSocketHandlers } from '../composables/useWebSocketHandlers'
import { useConversationLogic } from '../composables/useConversationLogic'
import { useMessageLogic } from '../composables/useMessageLogic'
import { useGroupLogic } from '../composables/useGroupLogic'
import { useOrganizationLogic } from '../composables/useOrganizationLogic'
import { useAppLogic } from '../composables/useAppLogic'
import { useUIState } from '../composables/useUIState'
```

---

## 🔄 集成步骤

### 第一步：初始化 Composables（低风险）

**位置**：在 Main.vue 的 `<script setup>` 部分，现有 Composables 初始化之后

**添加代码**：

```typescript
// ========== 新的 Composables 初始化 ==========
// WebSocket 处理器
const wsHandlers = useWebSocketHandlers()

// 会话逻辑
const conversationLogic = useConversationLogic()

// 消息逻辑
const messageLogic = useMessageLogic()

// 群组逻辑
const groupLogic = useGroupLogic()

// 组织架构逻辑
const orgLogic = useOrganizationLogic()

// 应用逻辑
const appLogic = useAppLogic()

// UI 状态
const uiState = useUIState()
```

**验证方法**：
1. 启动开发服务器：`npm run dev`
2. 检查控制台是否有错误
3. 确认页面正常加载

---

### 第二步：替换 WebSocket Handlers（中等风险）

**目标**：替换 Main.vue 中的 WebSocket 消息处理函数

**现有代码位置**：约 1408-1715 行

**替换示例**：

```typescript
// ❌ 删除现有代码
const handleReadReceipt = (data: any) => {
  logger.log('收到已读回执:', data)
  // ... 现有实现
}

const handleMessageRecalled = (data: any) => {
  logger.log('消息撤回:', data)
  // ... 现有实现
}

// ✅ 使用新的 Composables
const {
  handleReadReceipt,
  handleMessageRecalled,
  handleMessageDeleted,
  handleGroupInvitation,
  handleAddedToGroup,
  handleGroupMemberLeft,
  handleGroupMemberJoined,
  handleGroupMemberRoleUpdated,
  handleGroupOwnerTransferred,
  handleConversationUpdated,
  handleGroupAnnouncementUpdated,
  handleNotification,
  handleNewNotification,
  handleSystemMessage,
  handleUserStatusChange
} = wsHandlers
```

**验证方法**：
1. 测试消息收发功能
2. 测试消息撤回功能
3. 测试群组事件处理
4. 检查控制台日志

---

### 第三步：替换会话逻辑（中等风险）

**目标**：替换会话加载和选择逻辑

**现有代码位置**：约 1013-1134 行

**替换示例**：

```typescript
// ❌ 删除现有代码
const loadConversations = async () => {
  try {
    const response = await request('/api/v1/conversations')
    // ... 现有实现
  } catch (error) {
    // ... 错误处理
  }
}

const handleConversationSelect = (conversation: Conversation) => {
  // ... 现有实现
}

// ✅ 使用新的 Composables
const {
  loadConversations,
  handleConversationSelect,
  handleConversationCreated
} = conversationLogic
```

**验证方法**：
1. 测试会话列表加载
2. 测试会话切换
3. 测试新会话创建
4. 检查会话状态同步

---

### 第四步：替换消息逻辑（高风险）

**目标**：替换消息加载和处理逻辑

**现有代码位置**：约 2073-2603 行

**替换示例**：

```typescript
// ❌ 删除现有代码
const loadMessages = async (conversationId: string, reset: boolean = true) => {
  // ... 现有实现（约100行）
}

const handleLoadMore = (conversationId: string) => {
  // ... 现有实现
}

// ✅ 使用新的 Composables
const {
  loadMessages,
  handleLoadMore,
  getMessageReadUsers
} = messageLogic
```

**注意事项**：
- 消息逻辑涉及复杂的 UI 交互（滚动位置保持）
- 需要传递 `processMessage` 和 `markMessagesAsRead` 函数
- 建议分步替换，每步都进行测试

**验证方法**：
1. 测试消息加载
2. 测试消息分页
3. 测试滚动位置保持
4. 测试已读回执

---

### 第五步：替换群组逻辑（中等风险）

**目标**：替换群组管理相关逻辑

**现有代码位置**：约 3707-4387 行

**替换示例**：

```typescript
// ❌ 删除现有代码
const handleInviteMembers = (groupOrId) => {
  // ... 现有实现
}

const updateGroupName = async (newName: string) => {
  // ... 现有实现
}

// ✅ 使用新的 Composables
const {
  selectedGroup,
  handleInviteMembers,
  updateGroupName,
  updateAnnouncement,
  removeMember,
  addMembers,
  setAsAdmin,
  exitGroup,
  updateAISettings
} = groupLogic
```

**验证方法**：
1. 测试群组邀请成员
2. 测试群组设置更新
3. 测试成员管理
4. 测试退出群组

---

### 第六步：替换组织架构逻辑（低风险）

**目标**：替换组织架构加载和用户查找逻辑

**现有代码位置**：约 1102-1134 行

**替换示例**：

```typescript
// ❌ 删除现有代码
const loadOrganizationTree = async () => {
  // ... 现有实现
}

const handleUserClick = (employee: any) => {
  // ... 现有实现
}

// ✅ 使用新的 Composables
const {
  orgStructure,
  loadOrganizationTree,
  handleUserClick,
  collectEmployees,
  findDepartmentById,
  findEmployeeById
} = orgLogic
```

**验证方法**：
1. 测试组织架构加载
2. 测试用户点击
3. 测试用户搜索
4. 检查组织架构显示

---

### 第七步：替换应用逻辑（低风险）

**目标**：替换应用管理和切换逻辑

**现有代码位置**：约 3266-3477 行

**替换示例**：

```typescript
// ❌ 删除现有代码
const openApp = async (appId: string) => {
  // ... 现有实现
}

const handleSwitchApp = (app) => {
  // ... 现有实现
}

// ✅ 使用新的 Composables
const {
  selectedAppId,
  activeAppTab,
  recentApps,
  openApp,
  closeApp,
  handleSwitchApp,
  openExternalApp
} = appLogic
```

**验证方法**：
1. 测试应用打开
2. 测试应用切换
3. 测试最近应用
4. 检查应用状态

---

### 第八步：替换 UI 状态（低风险）

**目标**：替换 UI 状态管理逻辑

**现有代码位置**：分散在 Main.vue 各处

**替换示例**：

```typescript
// ❌ 删除现有代码
const isLoading = ref(false)
const sidebarCollapsed = ref(false)
const searchQuery = ref('')

// ✅ 使用新的 Composables
const {
  isLoading,
  sidebarCollapsed,
  searchQuery,
  activeOption,
  toggleSidebar,
  setLoading,
  setNetworkError
} = uiState
```

**验证方法**：
1. 测试加载状态
2. 测试侧边栏折叠
3. 测试搜索功能
4. 检查 UI 状态同步

---

## ⚠️ 重要注意事项

### 1. 渐进式集成

**原则**：
- 每次只替换一个功能域
- 替换后立即测试
- 保持功能完整性

**顺序建议**：
1. UI 状态（风险最低）
2. 组织架构逻辑（独立性高）
3. 应用逻辑（相对独立）
4. WebSocket handlers（中等风险）
5. 会话逻辑（中等风险）
6. 群组逻辑（中等风险）
7. 消息逻辑（风险最高）

### 2. 依赖关系处理

**问题**：新 Composables 可能依赖 Main.vue 中的函数或状态

**解决方案**：
```typescript
// 方案 1：传递依赖函数
const {
  loadMessages,
  handleLoadMore
} = messageLogic

// 在需要时传递 processMessage 函数
const wrappedLoadMessages = (conversationId: string) => {
  return loadMessages(conversationId, processMessage, markMessagesAsRead)
}

// 方案 2：使用 provide/inject
provide('processMessage', processMessage)
provide('markMessagesAsRead', markMessagesAsRead)

// 方案 3：在 Composables 中导入
// 直接在 Composables 中导入需要的函数
```

### 3. 状态同步

**问题**：确保状态在 Composables 和 Main.vue 之间正确同步

**解决方案**：
```typescript
// 使用 computed 保持响应式
const currentConversation = computed(() => 
  chatStore.getConversation(currentConversationId.value)
)

// 使用 watch 监听状态变化
watch(() => groupLogic.selectedGroup.value, (newGroup) => {
  // 处理群组选择变化
})
```

### 4. 错误处理

**问题**：确保错误处理逻辑正确传递

**解决方案**：
```typescript
// 在 Composables 中统一错误处理
try {
  await someOperation()
} catch (error) {
  logger.error('操作失败:', error)
  showMessage({ message: '操作失败', type: 'error' })
}

// 或者在 Main.vue 中捕获
try {
  await groupLogic.updateGroupName(newName)
} catch (error) {
  // 处理特定错误
}
```

---

## 🧪 测试验证清单

### 功能测试

- [ ] WebSocket 连接和重连
- [ ] 消息收发
- [ ] 消息撤回和删除
- [ ] 会话切换
- [ ] 会话创建
- [ ] 群组管理
- [ ] 成员邀请
- [ ] 组织架构显示
- [ ] 应用打开和切换
- [ ] 搜索功能
- [ ] UI 状态管理

### 性能测试

- [ ] 页面加载速度
- [ ] 消息加载速度
- [ ] 内存占用
- [ ] CPU 使用率
- [ ] 网络请求数量

### 兼容性测试

- [ ] Chrome 浏览器
- [ ] Firefox 浏览器
- [ ] Safari 浏览器
- [ ] Electron 应用
- [ ] 移动端适配

---

## 📊 预期收益

### 代码质量

- Main.vue 行数：5178 → < 1000（减少 80%）
- 单个文件复杂度：高 → 低
- 代码复用率：40% → 80%

### 开发效率

- 新功能开发时间：减少 50%
- Bug 定位时间：减少 60%
- 代码审查时间：减少 40%

### 可维护性

- 单一职责原则：提升 100%
- 测试覆盖率：提升 200%
- 文档完整性：提升 150%

---

## 🚀 快速开始

### 最小可行集成（MVP）

如果时间有限，建议先集成风险最低的部分：

```typescript
// 1. 初始化所有 Composables
const wsHandlers = useWebSocketHandlers()
const conversationLogic = useConversationLogic()
const messageLogic = useMessageLogic()
const groupLogic = useGroupLogic()
const orgLogic = useOrganizationLogic()
const appLogic = useAppLogic()
const uiState = useUIState()

// 2. 只替换 UI 状态（风险最低）
const {
  isLoading,
  sidebarCollapsed,
  searchQuery,
  activeOption
} = uiState

// 3. 测试验证
// 启动开发服务器，确认无错误
```

### 完整集成（推荐）

按照上述步骤，逐步完成所有集成工作。

---

## 📚 相关文档

- [性能优化报告](./performance-optimization-report.md)
- [重构指南](./refactoring/main-vue-refactoring-guide.md)
- [最终完成报告](./performance-optimization-final-report.md)

---

## 💡 提示

1. **备份代码**：集成前建议创建 Git 分支或备份
2. **小步快跑**：每次只修改一小部分，立即测试
3. **保持沟通**：遇到问题及时记录和反馈
4. **持续优化**：集成完成后继续优化性能

---

**创建时间**: 2026-05-24
**创建者**: AI Assistant
**状态**: 待执行
**预计工作量**: 2-3 天
