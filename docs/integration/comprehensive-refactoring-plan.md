# Main.vue 彻底重构分析报告

**日期**: 2026-05-24
**当前状态**: 5084 行代码
**目标**: 减少到 2000 行以下

---

## 📊 当前状态分析

### 文件大小
- **总行数**: 5084 行
- **已提取**: ~197 行 (3.9%)
- **剩余**: ~4887 行 (96.1%)
- **目标**: 减少 3000+ 行

### 函数统计
- **总函数数**: 56 个 handle/load/fetch/process 函数
- **已提取**: 10 个
- **剩余**: 46 个

---

## 🎯 可提取的函数分类

### 1. 消息处理函数 (高优先级)

| 函数名 | 预估行数 | 复杂度 | 依赖 | 可提取性 |
|--------|----------|--------|------|----------|
| handleNewMessage | ~191 行 | 高 | processMessage, loadConversations | 中 |
| handleSendMessage | ~300 行 | 高 | WebSocket, Store | 中 |
| handleStreamMessage | ~100 行 | 高 | WebSocket | 中 |
| processMessage | ~100 行 | 中 | - | 高 |
| handleRecallMessage | ~13 行 | 低 | WebSocket | 高 |
| handleMessageLike | ~14 行 | 低 | API | 高 |
| handleMessageUnlike | ~14 行 | 低 | API | 高 |
| handleMessageComment | ~30 行 | 中 | API | 高 |

**小计**: ~762 行

---

### 2. 会话处理函数 (高优先级)

| 函数名 | 预估行数 | 复杂度 | 依赖 | 可提取性 |
|--------|----------|--------|------|----------|
| handleConversationSelect | ~24 行 | 中 | loadMessages | 中 |
| handleConversationCreated | ~72 行 | 中 | Store | 高 |
| handleConversationUpdated | ~22 行 | 低 | Store | 高 |
| loadMessages | ~100 行 | 高 | API, Store | 中 |
| handleLoadMore | ~7 行 | 低 | loadMessages | 高 |

**小计**: ~225 行

---

### 3. 搜索函数 (中优先级)

| 函数名 | 预估行数 | 复杂度 | 依赖 | 可提取性 |
|--------|----------|--------|------|----------|
| handleSearch | ~20 行 | 中 | API | 高 |
| handleSearchItemClick | ~15 行 | 低 | - | 高 |
| handleApplyJoinGroup | ~39 行 | 中 | API | 高 |

**小计**: ~74 行

---

### 4. 通知处理函数 (中优先级)

| 函数名 | 预估行数 | 复杂度 | 依赖 | 可提取性 |
|--------|----------|--------|------|----------|
| handleNotification | ~24 行 | 低 | Store | 高 |
| handleNewNotification | ~18 行 | 低 | - | 高 |
| handleNotificationClick | ~13 行 | 低 | - | 高 |
| handleSystemMessage | ~27 行 | 低 | Store | 高 |
| handleCallNotification | ~20 行 | 低 | - | 高 |

**小计**: ~102 行

---

### 5. 应用相关函数 (中优先级)

| 函数名 | 预估行数 | 复杂度 | 依赖 | 可提取性 |
|--------|----------|--------|------|----------|
| loadRecentApps | ~73 行 | 中 | API | 高 |
| loadBuiltInApps | ~64 行 | 中 | API | 高 |
| loadUserApps | ~31 行 | 中 | API | 高 |
| loadApps | ~10 行 | 低 | - | 高 |
| handleOpenUserApp | ~50 行 | 中 | - | 高 |
| handleRefreshUserApps | ~7 行 | 低 | loadUserApps | 高 |
| handleSwitchApp | ~19 行 | 低 | - | 高 |

**小计**: ~254 行

---

### 6. 分享相关函数 (中优先级)

| 函数名 | 预估行数 | 复杂度 | 依赖 | 可提取性 |
|--------|----------|--------|------|----------|
| handleShareStickyNote | ~7 行 | 低 | - | 高 |
| handleForwardMessage | ~9 行 | 低 | - | 高 |
| handleOpenShareModal | ~7 行 | 低 | - | 高 |
| loadShareUsersAndGroups | ~57 行 | 中 | API | 高 |
| handleShareConfirm | ~50 行 | 中 | API | 高 |

**小计**: ~130 行

---

### 7. 通话相关函数 (低优先级)

| 函数名 | 预估行数 | 复杂度 | 依赖 | 可提取性 |
|--------|----------|--------|------|----------|
| handleStartVoiceCall | ~10 行 | 低 | - | 高 |
| handleStartVideoCall | ~10 行 | 低 | - | 高 |
| handleStartScreenShare | ~10 行 | 低 | - | 高 |
| handleScreenShareStart | ~6 行 | 低 | - | 高 |
| handleScreenShareStop | ~9 行 | 低 | - | 高 |
| handleScreenShareData | ~10 行 | 低 | - | 高 |

**小计**: ~55 行

---

### 8. 频道相关函数 (低优先级)

| 函数名 | 预估行数 | 复杂度 | 依赖 | 可提取性 |
|--------|----------|--------|------|----------|
| handleCreateChannel | ~5 行 | 低 | - | 高 |
| handleCreateChannelSubmit | ~36 行 | 中 | API | 高 |
| handleChannelSubscribe | ~4 行 | 低 | API | 高 |
| handleChannelUnsubscribe | ~4 行 | 低 | API | 高 |
| handleChannelSendMessage | ~4 行 | 低 | API | 高 |
| handleDisplayModeChange | ~4 行 | 低 | - | 高 |

**小计**: ~57 行

---

### 9. 设置相关函数 (低优先级)

| 函数名 | 预估行数 | 复杂度 | 依赖 | 可提取性 |
|--------|----------|--------|------|----------|
| handleSaveSettings | ~200 行 | 高 | API, Store | 中 |
| handleAvatarChange | ~50 行 | 中 | API | 高 |
| handleConfirmLogout | ~33 行 | 中 | API | 高 |

**小计**: ~283 行

---

### 10. 其他函数 (低优先级)

| 函数名 | 预估行数 | 复杂度 | 依赖 | 可提取性 |
|--------|----------|--------|------|----------|
| handleManualReconnect | ~10 行 | 低 | - | 高 |
| handleUserStatusChange | ~20 行 | 低 | Store | 高 |
| handleSidebarOptionClick | ~12 行 | 低 | - | 高 |
| handleInviteMembers | ~21 行 | 低 | - | 高 |
| handleSwitchConversation | ~16 行 | 低 | - | 高 |
| handleRetrySendMessage | ~18 行 | 中 | - | 高 |
| handleMessageDeleted | ~9 行 | 低 | - | 高 |

**小计**: ~106 行

---

## 📈 总计

| 分类 | 函数数 | 预估行数 | 优先级 |
|------|--------|----------|--------|
| 消息处理 | 8 | ~762 行 | 高 |
| 会话处理 | 5 | ~225 行 | 高 |
| 搜索函数 | 3 | ~74 行 | 中 |
| 通知处理 | 5 | ~102 行 | 中 |
| 应用相关 | 7 | ~254 行 | 中 |
| 分享相关 | 5 | ~130 行 | 中 |
| 通话相关 | 6 | ~55 行 | 低 |
| 频道相关 | 6 | ~57 行 | 低 |
| 设置相关 | 3 | ~283 行 | 低 |
| 其他函数 | 7 | ~106 行 | 低 |
| **总计** | **55** | **~2048 行** | - |

---

## 🎯 重构计划

### 阶段 1: 高优先级函数 (目标: 减少 ~987 行)

**目标**: 提取消息处理和会话处理函数

1. **创建 useMainMessageHandlers.ts**
   - handleNewMessage
   - processMessage
   - handleMessageLike
   - handleMessageUnlike
   - handleMessageComment
   - handleRecallMessage
   - 预估: ~362 行

2. **创建 useMainMessageSending.ts**
   - handleSendMessage
   - handleStreamMessage
   - handleRetrySendMessage
   - 预估: ~413 行

3. **创建 useMainConversationHandlers.ts**
   - handleConversationSelect
   - handleConversationCreated
   - handleConversationUpdated
   - handleLoadMore
   - 预估: ~153 行

4. **创建 useMainMessageLoading.ts**
   - loadMessages
   - 预估: ~100 行

**阶段 1 总计**: ~1028 行

---

### 阶段 2: 中优先级函数 (目标: 减少 ~560 行)

**目标**: 提取搜索、通知、应用、分享相关函数

1. **创建 useMainSearch.ts**
   - handleSearch
   - handleSearchItemClick
   - handleApplyJoinGroup
   - 预估: ~74 行

2. **创建 useMainNotificationHandlers.ts**
   - handleNotification
   - handleNewNotification
   - handleNotificationClick
   - handleSystemMessage
   - handleCallNotification
   - 预估: ~102 行

3. **创建 useMainAppHandlers.ts**
   - loadRecentApps
   - loadBuiltInApps
   - loadUserApps
   - loadApps
   - handleOpenUserApp
   - handleRefreshUserApps
   - handleSwitchApp
   - 预估: ~254 行

4. **创建 useMainShareHandlers.ts**
   - handleShareStickyNote
   - handleForwardMessage
   - handleOpenShareModal
   - loadShareUsersAndGroups
   - handleShareConfirm
   - 预估: ~130 行

**阶段 2 总计**: ~560 行

---

### 阶段 3: 低优先级函数 (目标: 减少 ~501 行)

**目标**: 提取通话、频道、设置、其他函数

1. **创建 useMainCallHandlers.ts**
   - handleStartVoiceCall
   - handleStartVideoCall
   - handleStartScreenShare
   - handleScreenShareStart
   - handleScreenShareStop
   - handleScreenShareData
   - 预估: ~55 行

2. **创建 useMainChannelHandlers.ts**
   - handleCreateChannel
   - handleCreateChannelSubmit
   - handleChannelSubscribe
   - handleChannelUnsubscribe
   - handleChannelSendMessage
   - handleDisplayModeChange
   - 预估: ~57 行

3. **创建 useMainSettingsHandlers.ts**
   - handleSaveSettings
   - handleAvatarChange
   - handleConfirmLogout
   - 预估: ~283 行

4. **创建 useMainOtherHandlers.ts**
   - handleManualReconnect
   - handleUserStatusChange
   - handleSidebarOptionClick
   - handleInviteMembers
   - handleSwitchConversation
   - handleMessageDeleted
   - 预估: ~106 行

**阶段 3 总计**: ~501 行

---

## 📊 预期成果

### 重构后 Main.vue

| 项目 | 当前 | 目标 | 减少 |
|------|------|------|------|
| 总行数 | 5084 | ~2000 | ~3084 |
| 函数数 | 56 | ~10 | ~46 |
| Composables | 3 | 15 | +12 |

### 新增 Composables

| Composable | 函数数 | 行数 | 阶段 |
|------------|--------|------|------|
| useMainMessageHandlers | 6 | ~362 | 1 |
| useMainMessageSending | 3 | ~413 | 1 |
| useMainConversationHandlers | 4 | ~153 | 1 |
| useMainMessageLoading | 1 | ~100 | 1 |
| useMainSearch | 3 | ~74 | 2 |
| useMainNotificationHandlers | 5 | ~102 | 2 |
| useMainAppHandlers | 7 | ~254 | 2 |
| useMainShareHandlers | 5 | ~130 | 2 |
| useMainCallHandlers | 6 | ~55 | 3 |
| useMainChannelHandlers | 6 | ~57 | 3 |
| useMainSettingsHandlers | 3 | ~283 | 3 |
| useMainOtherHandlers | 6 | ~106 | 3 |
| **总计** | **55** | **~2089** | - |

---

## 🚀 执行策略

### 策略 1: 渐进式重构
- 按阶段逐步执行
- 每个阶段完成后验证
- 随时可以停止

### 策略 2: 依赖注入
- 通过参数传递依赖
- 避免循环依赖
- 保持逻辑一致

### 策略 3: 测试驱动
- 每个阶段完成后测试
- 确保功能正常
- 及时修复问题

---

## ⚠️ 风险评估

### 高风险函数
- handleSendMessage (复杂度高，依赖多)
- handleStreamMessage (复杂度高)
- loadMessages (依赖多)
- handleSaveSettings (复杂度高)

### 中风险函数
- handleNewMessage (依赖 processMessage)
- handleConversationSelect (依赖 loadMessages)

### 低风险函数
- 其他所有函数

---

## 📝 建议

### 立即执行
1. **阶段 1**: 提取高优先级函数 (~1028 行)
2. **验证**: 测试消息收发、会话加载功能

### 后续执行
3. **阶段 2**: 提取中优先级函数 (~560 行)
4. **验证**: 测试搜索、通知、应用、分享功能

### 最后执行
5. **阶段 3**: 提取低优先级函数 (~501 行)
6. **验证**: 测试通话、频道、设置功能

---

## 🎯 成功标准

- Main.vue 减少到 2000 行以下
- 创建 12+ 个新 Composables
- 所有功能正常工作
- 没有引入新的 bug
- 代码质量显著提高

---

**报告生成时间**: 2026-05-24
**状态**: 分析完成，准备执行
**建议**: 立即开始阶段 1
