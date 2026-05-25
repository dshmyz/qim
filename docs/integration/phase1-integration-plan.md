# 阶段 1 集成计划

**日期**: 2026-05-24
**状态**: 准备集成
**目标**: 减少 Main.vue ~500 行代码

---

## ✅ 已创建的 Composables

### 1. useMainMessageHandlers.ts (147 行)
- `processMessage` - 处理消息对象
- `handleMessageLike` - 点赞消息
- `handleMessageUnlike` - 取消点赞
- `handleMessageComment` - 评论消息

### 2. useMainMessageLoading.ts (111 行)
- `loadMessages` - 加载消息
- `getMessageReadUsers` - 获取已读用户列表
- `messagePage` - 消息页码
- `messagePageSize` - 每页消息数
- `hasMoreMessages` - 是否有更多消息

### 3. useMainConversationHandlers.ts (85 行)
- `handleConversationSelect` - 选择会话
- `handleConversationCreated` - 创建会话
- `handleConversationUpdated` - 更新会话
- `handleLoadMore` - 加载更多

### 4. useMainMessageSending.ts (46 行)
- `handleRecallMessage` - 撤回消息
- `handleRetrySendMessage` - 重试发送

**总计**: 4 个 Composables，389 行代码

---

## 📝 集成步骤

### 步骤 1: 导入 Composables

在 Main.vue 的 `<script setup>` 部分添加导入：

```typescript
// Composables 导入
import { useMainMessageHandlers } from '../composables/useMainMessageHandlers'
import { useMainMessageLoading } from '../composables/useMainMessageLoading'
import { useMainConversationHandlers } from '../composables/useMainConversationHandlers'
import { useMainMessageSending } from '../composables/useMainMessageSending'
```

---

### 步骤 2: 初始化 Composables

在 Main.vue 的 `<script setup>` 部分添加初始化：

```typescript
// Composables 初始化
const mainMessageHandlers = useMainMessageHandlers()
const { processMessage } = mainMessageHandlers

const mainMessageLoading = useMainMessageLoading(conversations, processMessage)
const { loadMessages, getMessageReadUsers, messagePage, messagePageSize, hasMoreMessages } = mainMessageLoading

const mainConversationHandlers = useMainConversationHandlers(
  currentConversationId,
  messages,
  activeOption,
  loadMessages,
  loadConversations,
  messagePage,
  hasMoreMessages
)

const mainMessageSending = useMainMessageSending(
  currentConversationId,
  messages,
  handleSendMessage
)
const { handleRecallMessage, handleRetrySendMessage } = mainMessageSending
```

---

### 步骤 3: 删除旧函数

删除 Main.vue 中的以下函数：

1. `processMessage` (行 1820-1915, ~95 行)
2. `loadMessages` (行 1916-2016, ~100 行)
3. `getMessageReadUsers` (行 2017-2030, ~13 行)
4. `handleConversationSelect` (行 1053-1076, ~23 行)
5. `handleConversationCreated` (行 3372-3390, ~18 行)
6. `handleConversationUpdated` (行 1503-1525, ~22 行)
7. `handleLoadMore` (行 2443-2448, ~5 行)
8. `handleRecallMessage` (行 2322-2334, ~12 行)
9. `handleRetrySendMessage` (行 2449-2466, ~17 行)
10. `handleMessageLike` (行 3501-3514, ~13 行)
11. `handleMessageUnlike` (行 3515-3528, ~13 行)
12. `handleMessageComment` (行 3529-3532, ~3 行)

**预估删除**: ~334 行

---

### 步骤 4: 更新依赖

确保所有依赖都正确传递：

1. `useMainMessageLoading` 需要 `conversations` 和 `processMessage`
2. `useMainConversationHandlers` 需要 `currentConversationId`, `messages`, `activeOption`, `loadMessages`, `loadConversations`, `messagePage`, `hasMoreMessages`
3. `useMainMessageSending` 需要 `currentConversationId`, `messages`, `handleSendMessage`

---

## ⚠️ 注意事项

### 循环依赖问题

`useMainMessageSending` 需要 `handleSendMessage`，但 `handleSendMessage` 还在 Main.vue 中。这会导致循环依赖。

**解决方案**:
1. 暂时不提取 `handleSendMessage`
2. 或者将 `handleSendMessage` 也提取到 Composable 中

### 依赖注入

所有 Composables 都使用依赖注入模式，通过参数接收依赖，避免隐式依赖。

---

## 📊 预期成果

| 项目 | 当前 | 集成后 | 减少 |
|------|------|--------|------|
| Main.vue 行数 | 5084 | ~4750 | ~334 |
| Composables | 3 | 7 | +4 |
| 函数数 | 56 | ~45 | ~11 |

---

## 🚀 执行顺序

1. ✅ 创建 4 个新 Composables
2. ⏸️ 导入 Composables 到 Main.vue
3. ⏸️ 初始化 Composables
4. ⏸️ 删除旧函数
5. ⏸️ 验证功能

---

## 📝 后续工作

### 阶段 1 剩余工作

- 提取 `handleSendMessage` (~300 行)
- 提取 `handleStreamMessage` (~100 行)
- 提取 `handleNewMessage` (~191 行)

### 阶段 2 工作

- 提取搜索函数 (~74 行)
- 提取通知函数 (~102 行)
- 提取应用函数 (~254 行)
- 提取分享函数 (~130 行)

### 阶段 3 工作

- 提取通话函数 (~55 行)
- 提取频道函数 (~57 行)
- 提取设置函数 (~283 行)
- 提取其他函数 (~106 行)

---

**报告生成时间**: 2026-05-24
**状态**: 准备集成
**建议**: 立即开始集成步骤 2
