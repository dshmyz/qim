# Main.vue Composables 提取设计文档

**日期**: 2026-04-25  
**作者**: AI Assistant  
**状态**: 待实施

## 概述

将 Main.vue 中约 6000 行的 script 部分拆分为 8 个独立的 composables，每个 composable 有明确的职责边界，最终将 Main.vue 精简到约 800 行，只保留组合逻辑和模板。

## 当前问题

- **Main.vue script 部分**: ~6000 行，包含过多职责
- **职责混杂**: 聊天消息、WebSocket、RTC、用户管理、应用管理、通知等全部耦合在一起
- **维护困难**: 难以定位代码、难以测试、难以复用逻辑
- **存在空注释**: L6634-6671 有 10 多行空的"主题注释"，但内容已被删除

## 目标结构

```
src/
  views/
    Main.vue (清理后 ~800行，只保留组合逻辑和模板)
  composables/
    useChat.ts              ~800行 - 聊天消息管理
    useWebSocket.ts         ~600行 - WebSocket 连接管理
    useConversation.ts      ~400行 - 会话管理
    useNotifications.ts     ~300行 - 通知中心
    useUserManagement.ts    ~500行 - 用户信息、组织架构、群组管理
    useAppState.ts          ~300行 - 应用状态管理
    useRTC.ts               ~400行 - WebRTC 屏幕共享、音视频通话
    useUI.ts                ~300行 - UI 相关（右键菜单、弹窗、设置面板）
```

## Composable 职责定义

### useChat.ts
- 发送消息（handleSendMessage）
- 消息列表管理（messages, hasMoreMessages）
- 加载历史消息（handleLoadMore, loadMessages）
- 重发消息（handleRetrySendMessage）
- 撤回消息（handleRecallMessage）
- 消息读取用户（getMessageReadUsers）

### useWebSocket.ts
- WebSocket 连接管理（connect, disconnect, reconnect）
- 事件监听和分发（onMessage, onConnect, onDisconnect）
- 重连机制（maxRetries, retryInterval）
- 连接状态（isConnected, connectionState）

### useConversation.ts
- 会话列表加载（loadConversations）
- 会话切换（handleConversationSelect, handleSwitchConversation）
- 会话操作（置顶、删除、标记已读）
- 当前会话状态（currentConversation, currentConversationId）

### useNotifications.ts
- 通知中心管理（notifications, unreadNotificationCount）
- 新通知处理（handleNewNotification, handleNotificationCenter）
- 通知过滤和分类
- 通知点击处理（handleNotificationClick）

### useUserManagement.ts
- 当前用户信息（currentUser）
- 组织架构（orgStructure, selectedGroup）
- 群组管理（selectedChannel）
- 用户搜索和操作（handleUserClick, startPrivateChat, handleInviteMembers）
- 频道操作（handleChannelSelect, unsubscribeChannel）

### useAppState.ts
- 侧边栏选项（activeOption: 'recent' | 'org' | 'groups' | 'apps'）
- 应用管理（selectedAppId, recentApps, allApps, appCategories）
- 搜索状态（searchQuery, searchResults）
- 加载和网络状态（isLoading, showNetworkError, networkErrorMsg）

### useRTC.ts
- 屏幕共享（remoteScreenSharing, remoteScreenUserId, remoteScreenData）
- 屏幕共享事件处理（handleScreenShareStart, handleScreenShareStop, handleScreenShareData）
- 音视频通话（call, hangup, mute, unmute）

### useUI.ts
- 右键菜单（showContextMenu, showUserContextMenu, showGroupContextMenu）
- 弹窗管理（showShareModal, showThemeMenu, showSettingsMenu）
- 侧边栏切换（toggleSidebar）
- 分享操作（shareType, shareUsers, shareGroups, handleShareConfirm）

## 数据流

```
Main.vue (组合层)
  ├── useChat(messages, sendMessage, loadMore, retrySend, recall)
  ├── useWebSocket(connect, disconnect, reconnect, onMessage)
  ├── useConversation(conversations, currentConversation, selectConversation)
  ├── useNotifications(notifications, unreadCount, handleNotification)
  ├── useUserManagement(currentUser, orgStructure, groups, channels)
  ├── useAppState(activeOption, selectedAppId, searchQuery, isLoading)
  ├── useRTC(screenShare, call, mute)
  └── useUI(contextMenu, modals, settings, sidebar)
```

## 执行策略

**阶段一**: 创建 composable 骨架，定义接口和返回值  
**阶段二**: 逐个提取逻辑（从最独立的开始）  
  1. useNotifications（最独立，只涉及通知状态）
  2. useAppState（状态管理，不依赖其他模块）
  3. useUI（UI 交互，不依赖其他模块）

**阶段三**: 提取核心逻辑  
  4. useChat（消息管理，依赖 useConversation）
  5. useConversation（会话管理，依赖 useAppState）
  6. useWebSocket（连接管理，被多个模块依赖）

**阶段四**: 提取复杂逻辑  
  7. useUserManagement（用户管理，依赖 useAppState）
  8. useRTC（RTC 功能，依赖 useWebSocket）

**阶段五**: 清理 Main.vue，删除空注释和未使用代码，验证功能完整性

## 风险控制

- 每次只提取一个 composable，完成后验证
- 保留原有函数签名和返回值，确保向后兼容
- 使用 Git 分支开发，每个 composable 一个 commit
- 提取过程中保持代码可运行
- 每个 composable 提取后进行基本功能测试

## 验收标准

1. Main.vue script 部分 ≤ 1000 行
2. 所有 composables 职责单一，无循环依赖
3. 所有原有功能正常工作
4. TypeScript 类型定义完整
5. 无未使用的导入或变量
6. 代码可通过现有的 lint 和 typecheck 检查
