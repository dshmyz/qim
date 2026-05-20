# 主界面性能优化方案

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划

**目标：** 解决主界面（Main.vue）打开卡顿问题，提升页面加载性能和用户体验

**架构：** 将 5178 行的巨型单文件组件拆分为多个功能模块，对非首屏组件实施懒加载，优化数据加载策略

**技术栈：** Vue 3 + TypeScript + Composition API + defineAsyncComponent

---

## 📊 现状分析

### 问题清单

| 问题 | 严重程度 | 当前状态 | 影响 |
|------|----------|----------|------|
| 文件过大 | 🔴 严重 | **5178行 / 154KB** | 解析编译慢、渲染检查慢 |
| 组件导入过多 | 🔴 严重 | **40+ 个组件同步导入** | 初始 bundle 膨胀 |
| API 并发加载 | 🟡 中等 | onMounted 同时发 5 个请求 | 首屏等待时间长 |
| 业务逻辑混杂 | 🟡 中等 | 所有逻辑在一个文件 | 维护困难、耦合严重 |

### 当前 onMounted 加载流程

```
onMounted
├── refreshUser()              // 同步等待：获取当前用户
├── Promise.all([
│   ├── loadConversations()    // 加载会话列表
│   ├── loadOrganizationTree() // 加载组织架构树
│   ├── loadUserApps()         // 加载用户自定义应用
│   └── loadBuiltInApps()      // 加载内置应用
│ ])
├── connectWebSocket()         // 建立 WebSocket 连接
├── systemConfigStore.fetchPublicConfig()  // 获取系统配置
├── registerUpdateEventListeners()         // 注册更新事件
└── registerCustomEventListeners()         // 注册自定义事件
```

### 当前组件导入列表（40+ 个）

```
应用类 (11个): CalendarApp, StickyNotesApp, NotesApp, TaskManagementApp,
               FileManagementApp, AppManagementApp, AIAssistantApp,
               ShortLinkManager, MiniAppManager, UserAppContainer, AvatarSettingsPanel

布局类 (3个): Sidebar, SideOptions, WindowControls

聊天类 (2个): ChatWindow, RealtimeCommunication

弹窗/面板类 (10个): GroupDetail, ModalContainer, ShareModal, UserProfile,
                    NotificationCenter, CreateGroupModal, ChannelDetailNew,
                    UserDetailPanel, AppsPanel, SelfProfileModal, GroupModals,
                    GroupAIPanel, MainContextMenus, MainDialogs, SettingsPanel
```

---

## 🎯 优化方案

### 方案一：组件拆分（核心优化）

将 Main.vue 按功能域拆分为独立模块：

#### 新建文件结构

```
qim-client/src/views/main/
├── index.vue                    # 主入口（精简后的 Main.vue，约 200-300 行）
├── composables/
│   ├── useChatManager.ts        # 聊天相关逻辑（消息收发、会话管理）
│   ├── useContactManager.ts     # 通讯录相关逻辑（组织架构、用户操作）
│   ├── useAppManager.ts         # 应用中心相关逻辑（应用加载、管理）
│   ├── useWebSocketHandlers.ts  # WebSocket 消息处理器
│   └── useEventListeners.ts     # 事件监听器注册
├── components/
│   ├── RightContentArea.vue     # 右侧内容区域（根据 activeOption 渲染不同内容）
│   ├── AppRenderer.vue          # 应用渲染器（处理所有应用的 v-else-if）
│   └── ChatArea.vue             # 聊天区域（ChatWindow + RealtimeCommunication）
```

#### 拆分映射表

| 原代码位置 | 移动到 | 预估行数 |
|------------|--------|----------|
| 消息相关函数 | useChatManager.ts | ~800 行 |
| 组织架构/用户 | useContactManager.ts | ~400 行 |
| 应用相关逻辑 | useAppManager.ts | ~500 行 |
| WebSocket 处理器 | useWebSocketHandlers.ts | ~300 行 |
| 事件监听器 | useEventListeners.ts | ~200 行 |
| 应用渲染模板 | AppRenderer.vue | ~250 行 |
| 右侧内容区 | RightContentArea.vue | ~200 行 |
| 主入口保留 | index.vue | ~300 行 |

---

### 方案二：组件懒加载（立竿见影）

对非首屏必需的组件使用 `defineAsyncComponent`：

#### 需要懒加载的组件（按优先级）

**P0 - 立即实施（应用类组件）：**

```typescript
// 这些组件只在点击"应用"选项时才显示
const FileManagementApp = defineAsyncComponent(() =>
  import('../components/apps/FileManagementApp.vue')
)
const NotesApp = defineAsyncComponent(() =>
  import('../components/apps/NotesApp.vue')
)
const TaskManagementApp = defineAsyncComponent(() =>
  import('../components/apps/task/TaskManagementApp.vue')
)
const CalendarApp = defineAsyncComponent(() =>
  import('../components/apps/CalendarApp.vue')
)
const StickyNotesApp = defineAsyncComponent(() =>
  import('../components/apps/StickyNotesApp.vue')
)
const AIAssistantApp = defineAsyncComponent(() =>
  import('../components/apps/AIAssistantApp.vue')
)
const ShortLinkManager = defineAsyncComponent(() =>
  import('../components/apps/ShortLinkManager.vue')
)
const AppManagementApp = defineAsyncComponent(() =>
  import('../components/apps/AppManagementApp.vue')
)
const AvatarSettingsPanel = defineAsyncComponent(() =>
  import('../components/avatar/AvatarSettingsPanel.vue')
)
const UserAppContainer = defineAsyncComponent(() =>
  import('../components/apps/UserAppContainer.vue')
)
```

**P1 - 延迟实施（条件显示的组件）：**

```typescript
// 这些组件需要特定条件才显示
const GroupDetail = defineAsyncComponent(() =>
  import('../components/shared/GroupDetail.vue')
)
const ChannelDetailNew = defineAsyncComponent(() =>
  import('../components/channel/ChannelDetailNew.vue')
)
const UserDetailPanel = defineAsyncComponent(() =>
  import('../components/user/UserDetailPanel.vue')
)
const SettingsPanel = defineAsyncComponent(() =>
  import('../components/settings/SettingsPanel.vue')
)
const GroupAIPanel = defineAsyncComponent(() =>
  import('../components/ai/GroupAIPanel.vue')
)

// 弹窗类组件（默认不渲染）
const UserProfile = defineAsyncComponent(() =>
  import('../components/modals/UserProfile.vue')
)
const NotificationCenter = defineAsyncComponent(() =>
  import('../components/notification/NotificationCenter.vue')
)
const CreateGroupModal = defineAsyncComponent(() =>
  import('../components/modals/CreateGroupModal.vue')
)
```

**保持同步加载的首屏组件：**

```typescript
// 这些是首屏必须的，保持同步导入
import Sidebar from '../components/layout/Sidebar.vue'
import SideOptions from '../components/layout/SideOptions.vue'
import WindowControls from '../components/layout/WindowControls.vue'
import ChatWindow from '../components/chat/ChatWindow.vue'
import ShareModal from '../components/modals/ShareModal.vue'
import AppsPanel from '../components/apps/AppsPanel.vue'
import MiniAppManager from '../components/apps/MiniAppManager.vue'
import MainContextMenus from '../components/menus/MainContextMenus.vue'
import MainDialogs from '../components/modals/MainDialogs.vue'
import ModalContainer from '../components/shared/ModalContainer.vue'
```

---

### 方案三：数据加载策略优化

#### 分阶段加载策略

```
阶段 1: 核心数据（阻塞渲染）
├── refreshUser()           // 必须先有用户信息
└── loadConversations()     // 会话列表是核心功能

阶段 2: 重要数据（并行加载，不阻塞首屏）
├── loadOrganizationTree()  // 组织架构（切换到通讯录时才需要）
├── loadUserApps()          // 用户应用（切换到应用时才需要）
├── loadBuiltInApps()       // 内置应用（同上）
└── connectWebSocket()      // WebSocket 连接

阶段 3: 辅助数据（后台静默加载）
├── systemConfigStore.fetchPublicConfig()
├── registerUpdateEventListeners()
└── registerCustomEventListeners()
```

#### 优化后的代码结构

```typescript
onMounted(async () => {
  isLoading.value = true

  try {
    // 阶段 1：核心数据（必须等待）
    await refreshUser()
    await loadConversations()

    // 立即隐藏 loading，展示主界面
    isLoading.value = false

    // 阶段 2：重要数据（后台并行加载）
    Promise.allSettled([
      loadOrganizationTree(),
      loadUserApps(),
      loadBuiltInApps(),
    ]).catch(err => console.warn('次要数据加载失败:', err))

    // 阶段 3：连接与注册
    setupWebSocketConnection()
    loadSystemConfig()
    registerEventListeners()

  } catch (error) {
    console.error('核心数据加载失败:', error)
    isLoading.value = false
    showNetworkError.value = true
  }
})
```

---

### 方案四：骨架屏 / 渐进式加载（体验优化）

为右侧内容区域添加骨架屏：

```vue
<template>
  <!-- 右侧内容区 -->
  <div class="main-content">
    <!-- 首屏核心：侧边栏始终显示 -->
    <Sidebar ... />

    <!-- 右侧区域：根据状态显示 -->
    <Suspense timeout="0">
      <template #default>
        <RightContentArea :activeOption="activeOption" ... />
      </template>
      <template #fallback>
        <ContentSkeleton :type="activeOption" />
      </template>
    </Suspense>
  </div>
</template>
```

---

## 📋 实施任务清单

### 任务 1：实施组件懒加载（预计收益最大）

**文件：**
- 修改：`qim-client/src/views/Main.vue:528-571`

- [ ] 步骤 1：在文件顶部添加 `defineAsyncComponent` 导入

- [ ] 步骤 2：将 11 个应用类组件改为懒加载

- [ ] 步骤 3：将 5 个条件显示组件改为懒加载

- [ ] 步骤 4：将 4 个弹窗类组件改为懒加载

- [ ] 步骤 5：验证功能正常，测试各模块切换

**预期效果：** 初始 bundle 体积减少 30-40%

---

### 任务 2：优化数据加载顺序

**文件：**
- 修改：`qim-client/src/views/Main.vue:1117-1155`

- [ ] 步骤 1：重构 onMounted，分为三个阶段

- [ ] 步骤 2：核心数据加载完成后立即关闭 loading

- [ ] 步骤 3：次要数据使用 Promise.allSettled 后台加载

- [ ] 步骤 4：提取 WebSocket 设置为独立函数

- [ ] 步骤 5：验证首屏加载速度提升

**预期效果：** 首屏可交互时间减少 40-60%

---

### 任务 3：拆分 Composable（代码质量）

**文件：**
- 创建：`qim-client/src/views/main/composables/useChatManager.ts`
- 创建：`qim-client/src/views/main/composables/useContactManager.ts`
- 创建：`qim-client/src/views/main/composables/useAppManager.ts`
- 创建：`qim-client/src/views/main/composables/useWebSocketHandlers.ts`
- 修改：`qim-client/src/views/Main.vue`

- [ ] 步骤 1：创建 composables 目录结构

- [ ] 步骤 2：提取聊天相关逻辑到 useChatManager

- [ ] 步骤 3：提取通讯录逻辑到 useContactManager

- [ ] 步骤 4：提取应用逻辑到 useAppManager

- [ ] 步骤 5：提取 WebSocket 处理器到 useWebSocketHandlers

- [ ] 步骤 6：重构 Main.vue 引用新的 composables

**预期效果：** Main.vue 从 5178 行降至约 800 行

---

### 任务 4：拆分子组件（模板优化）

**文件：**
- 创建：`qim-client/src/views/main/components/AppRenderer.vue`
- 创建：`qim-client/src/views/main/components/RightContentArea.vue`
- 修改：`qim-client/src/views/Main.vue:109-288`

- [ ] 步骤 1：创建 AppRenderer.vue，封装所有应用的条件渲染

- [ ] 步骤 2：创建 RightContentArea.vue，封装右侧内容区逻辑

- [ ] 步骤 3：替换 Main.vue 中的模板代码为新组件

**预期效果：** 模板代码从 520 行降至约 150 行

---

## 🎯 推荐实施顺序

```
建议按以下顺序实施，每步都可独立验证效果：

1️⃣  任务 1：组件懒加载（改动最小，收益最大，风险最低）
        ↓
2️⃣  任务 2：优化数据加载顺序（用户体验提升明显）
        ↓
3️⃣  任务 4：拆分子组件（模板简化）
        ↓
4️⃣  任务 3：拆分 Composable（代码质量提升，长期维护性）
```

---

## ⚠️ 注意事项

1. **向后兼容**：所有重构必须保持现有功能不变
2. **渐进式迁移**：可以逐步实施，不需要一次性完成
3. **测试验证**：每完成一个任务都需要验证：
   - 登录流程正常
   - 会话列表加载正常
   - 消息收发正常
   - 各应用模块切换正常
   - WebSocket 连接正常
4. **性能基线**：建议在优化前后分别记录 Lighthouse 性能分数作为对比

---

## 📈 预期收益

| 指标 | 优化前 | 优化后（预估） | 提升幅度 |
|------|--------|----------------|----------|
| Main.vue 行数 | 5178 行 | ~800 行 | ↓ 85% |
| 首次加载体积 | ~154KB | ~80KB | ↓ 48% |
| 首屏可交互时间 | ~2-3s | ~0.8-1.2s | ↓ 60% |
| 组件导入数 | 40+ 同步 | 15 同步 + 25 懒加载 | bundle 体积 ↓ 35% |
