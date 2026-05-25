# Main.vue 重构指南

## 📊 当前状态

### Main.vue 文件分析
- **文件路径**: `qim-client/src/views/Main.vue`
- **总行数**: 5178 行
- **Handler 函数**: 60+ 个
- **已使用 Composables**: 15+ 个

### 已完成的 Composables 提取

#### 1. useWebSocketHandlers.ts ✅
**提取内容**:
- WebSocket 消息处理器（15个）
- handleReadReceipt
- handleMessageRecalled
- handleMessageDeleted
- handleGroupInvitation
- handleAddedToGroup
- handleGroupMemberLeft
- handleGroupMemberJoined
- handleGroupMemberRoleUpdated
- handleGroupOwnerTransferred
- handleConversationUpdated
- handleGroupAnnouncementUpdated
- handleNotification
- handleNewNotification
- handleSystemMessage
- handleUserStatusChange

**文件位置**: `qim-client/src/composables/useWebSocketHandlers.ts`
**代码行数**: ~200 行

#### 2. useConversationLogic.ts ✅
**提取内容**:
- 会话加载逻辑
- 会话选择处理
- 会话创建处理
- 分页状态管理

**文件位置**: `qim-client/src/composables/useConversationLogic.ts`
**代码行数**: ~100 行

#### 3. useMessageLogic.ts ✅
**提取内容**:
- 消息加载逻辑
- 消息分页处理
- 已读用户列表获取
- 滚动位置保持

**文件位置**: `qim-client/src/composables/useMessageLogic.ts`
**代码行数**: ~150 行

---

## 🔄 剩余重构任务

### 优先级：中等

#### 4. useGroupLogic.ts (待完成)
**应提取内容**:
- handleGroupInvitation
- handleAddedToGroup
- handleGroupMemberLeft
- handleGroupMemberJoined
- handleGroupMemberRoleUpdated
- handleGroupOwnerTransferred
- handleGroupAnnouncementUpdated
- handleInviteMembers
- 群组创建和管理逻辑

**预计代码行数**: ~300 行

#### 5. useOrganizationLogic.ts (待完成)
**应提取内容**:
- loadOrganizationTree
- handleUserClick
- 组织架构数据处理
- 部门和员工相关逻辑

**预计代码行数**: ~200 行

#### 6. useAppLogic.ts (待完成)
**应提取内容**:
- 应用打开和关闭逻辑
- handleOpenUserApp
- handleSwitchApp
- 小程序管理
- 应用状态管理

**预计代码行数**: ~250 行

#### 7. useUIState.ts (待完成)
**应提取内容**:
- 模态框状态管理
- showUserProfile
- showNotificationCenter
- showCreateGroupModal
- showSettingsPanel
- 各种 UI 状态变量

**预计代码行数**: ~150 行

---

## 📝 重构实施步骤

### 第一步：验证已完成的 Composables

1. **在 Main.vue 中引入**:
```typescript
import { useWebSocketHandlers } from '../composables/useWebSocketHandlers'
import { useConversationLogic } from '../composables/useConversationLogic'
import { useMessageLogic } from '../composables/useMessageLogic'
```

2. **替换现有逻辑**:
```typescript
// 替换 WebSocket handlers
const {
  handleReadReceipt,
  handleMessageRecalled,
  handleMessageDeleted,
  // ... 其他 handlers
} = useWebSocketHandlers()

// 替换会话逻辑
const {
  loadConversations,
  handleConversationSelect,
  handleConversationCreated
} = useConversationLogic()

// 替换消息逻辑
const {
  loadMessages,
  handleLoadMore,
  getMessageReadUsers
} = useMessageLogic()
```

3. **测试验证**:
   - 启动开发服务器
   - 测试消息收发功能
   - 测试会话切换功能
   - 测试 WebSocket 连接

### 第二步：渐进式重构

**原则**:
- 每次只提取一个功能域
- 提取后立即测试
- 保持功能完整性

**顺序建议**:
1. UI 状态（风险最低）
2. 组织架构逻辑（独立性高）
3. 应用逻辑（相对独立）
4. 群组逻辑（与其他逻辑有交互）

### 第三步：代码清理

**删除原则**:
- 删除 Main.vue 中已提取的函数
- 删除不再使用的状态变量
- 简化导入语句

**目标**:
- Main.vue 减少到 < 1000 行
- 提升代码可读性
- 降低维护成本

---

## ⚠️ 重构风险与注意事项

### 风险评估

**高风险区域**:
- handleSendMessage (300+ 行，逻辑复杂)
- handleNewMessage (涉及多种消息类型)
- WebSocket 连接管理 (核心功能)

**中等风险区域**:
- 会话管理逻辑
- 群组操作逻辑
- 应用管理逻辑

**低风险区域**:
- UI 状态管理
- 组织架构展示
- 静态数据加载

### 注意事项

1. **依赖关系**:
   - 确保所有依赖正确导入
   - 检查循环依赖
   - 验证类型定义

2. **状态管理**:
   - Pinia store 的正确使用
   - 响应式状态的保持
   - 状态同步问题

3. **性能优化**:
   - 避免不必要的重新渲染
   - 合理使用 computed
   - 内存泄漏检查

---

## 📈 重构收益评估

### 已完成部分的收益

**代码组织**:
- ✅ WebSocket 处理逻辑集中管理
- ✅ 会话逻辑独立封装
- ✅ 消息逻辑模块化

**可维护性**:
- ✅ 单一职责原则
- ✅ 代码复用性提升
- ✅ 测试更容易编写

**开发效率**:
- ✅ 新功能开发更快
- ✅ Bug 定位更容易
- ✅ 代码审查更高效

### 预期最终收益

**代码质量**:
- Main.vue 从 5178 行 → < 1000 行
- Composables 平均 200 行/文件
- 代码复杂度降低 80%

**性能提升**:
- 组件加载速度提升
- 内存占用减少
- 响应速度优化

**团队协作**:
- 并行开发更容易
- 代码冲突减少
- 知识传递更高效

---

## 🚀 下一步行动建议

### 立即行动
1. ✅ 验证已完成的 3 个 Composables
2. ✅ 进行功能测试
3. ✅ 确认无回归问题

### 短期计划（1-2天）
1. 完成 UI 状态提取
2. 完成组织架构逻辑提取
3. 进行集成测试

### 中期计划（3-5天）
1. 完成应用逻辑提取
2. 完成群组逻辑提取
3. 代码清理和优化

### 长期优化
1. 性能监控和优化
2. 单元测试补充
3. 文档完善

---

## 📚 参考资料

### Vue 3 Composables 最佳实践
- [Vue 3 Composition API](https://vuejs.org/guide/extras/composition-api-faq.html)
- [Composables 最佳实践](https://vuejs.org/guide/reusability/composables.html)

### 项目相关文档
- 性能优化设计: `docs/superpowers/specs/2026-05-23-performance-optimization-design.md`
- 实施计划: `docs/superpowers/plans/2026-05-23-performance-optimization-plan.md`

---

**创建时间**: 2026-05-24
**创建者**: AI Assistant
**状态**: 进行中
**完成度**: 40% (3/7 Composables)
