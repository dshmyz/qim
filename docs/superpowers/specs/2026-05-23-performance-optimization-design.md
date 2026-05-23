# 性能优化设计方案

> **目标**：解决项目中的性能隐患，提升用户体验和系统稳定性
> **策略**：快速见效优先，渐进式优化，风险可控
> **日期**：2026-05-23

---

## 问题分析

### 已识别的性能隐患

| 优先级 | 问题 | 影响 | 状态 |
|--------|------|------|------|
| 🔴 高 | WebSocket 连接管理缺陷 | 网络波动时用户体验差 | 待优化 |
| 🔴 高 | Main.vue 组件过大（5178行） | 页面加载慢，维护困难 | 待优化 |
| 🟡 中 | 请求缺少重试机制 | 网络不稳定时失败率高 | 待优化 |
| ❌ 删除 | 图片缓存使用 localStorage | 未被使用，应删除 | 待删除 |

### 优化策略

**快速见效优先**：按风险从低到高排序
1. 删除未使用代码（零风险，立即见效）
2. 请求重试机制（低风险，快速见效）
3. WebSocket 连接优化（中风险，高收益）
4. Main.vue 组件拆分（高风险，高收益）

---

## 阶段 1：清理与快速优化

### 1.1 删除未使用的图片缓存代码

**删除文件**：
- `qim-client/src/composables/useImageCache.ts`
- `qim-client/tests/unit/composables/useImageCache.test.ts`（如果存在）

**理由**：
- 该文件未被任何组件引用
- localStorage 有 5-10MB 限制，不适合存储图片
- 浏览器原生缓存机制已足够

**影响**：
- ✅ 无破坏性变更
- ✅ 减少代码体积
- ✅ 消除潜在隐患

---

### 1.2 请求重试机制

**目标文件**：
- `qim-admin/src/utils/request.ts`
- `qim-client/src/api/core.ts`

**配置参数**：
```typescript
interface RetryConfig {
  maxRetries: number           // 最大重试次数：3
  baseDelay: number            // 基础延迟：1000ms
  maxDelay: number             // 最大延迟：10000ms
  exponentialBackoff: boolean  // 指数退避：true
}
```

**重试策略**：

**可重试的错误**：
- 网络错误：ECONNABORTED, ENETDOWN, ENETUNREACH
- 服务器错误：502, 503, 504
- 限流错误：429（等待后重试）

**不可重试的错误**：
- 客户端错误：400, 401, 403, 404
- 业务逻辑错误

**指数退避算法**：
```
delay = min(baseDelay * 2^retryCount, maxDelay) + random(0, 1000)
```

**重试延迟示例**：
```
第1次重试: 1s + jitter
第2次重试: 2s + jitter
第3次重试: 4s + jitter
```

**实现位置**：
- 在 axios response interceptor 中添加重试逻辑
- 使用 `config.__retryCount` 记录重试次数

---

### 1.3 请求缓存和去重

**缓存策略**：

**GET 请求缓存**：
- 缓存时间：30 秒（可配置）
- 缓存键：`{method}:{url}:{JSON.stringify(params)}`
- 自动失效：页面刷新、手动清除

**请求去重**：
- 相同请求在 pending 状态时，复用 Promise
- 避免并发重复请求

**实现**：
```typescript
// 请求缓存
const requestCache = new Map<string, {
  data: any
  timestamp: number
}>()

// 请求去重
const pendingRequests = new Map<string, Promise<any>>()
```

**配置选项**：
```typescript
interface RequestConfig extends AxiosRequestConfig {
  cache?: boolean        // 是否启用缓存（默认 false）
  cacheTime?: number     // 缓存时间（毫秒）
  dedupe?: boolean       // 是否去重（默认 true）
}
```

---

### 1.4 错误处理增强

**统一错误提示**：

| 错误类型 | 提示信息 | 操作 |
|---------|---------|------|
| 网络错误 | "网络异常，正在重试..." | 自动重试 |
| 超时错误 | "请求超时，请检查网络" | 自动重试 |
| 服务器错误 | "服务器异常，请稍后重试" | 自动重试 |
| 重试失败 | "请求失败，请稍后重试" | 提供"重试"按钮 |
| 认证失败 | "登录已过期，请重新登录" | 跳转登录页 |

---

## 阶段 2：WebSocket 连接管理优化

### 2.1 增强重连策略

**当前问题**：
- 固定重连间隔 5000ms
- 最大重连次数仅 3 次
- 无抖动机制

**优化方案**：

**指数退避 + 抖动**：
```typescript
const RECONNECT_CONFIG = {
  baseDelay: 1000,        // 基础延迟 1s
  maxDelay: 30000,        // 最大延迟 30s
  maxAttempts: 10,        // 最大重连次数 10
  jitterRange: [0, 2000]  // 抖动范围 0-2s
}

function calculateReconnectDelay(attempt: number): number {
  const exponentialDelay = RECONNECT_CONFIG.baseDelay * Math.pow(2, attempt)
  const cappedDelay = Math.min(exponentialDelay, RECONNECT_CONFIG.maxDelay)
  const jitter = Math.random() * (RECONNECT_CONFIG.jitterRange[1] - RECONNECT_CONFIG.jitterRange[0])
  return cappedDelay + jitter
}
```

**重连流程**：
```
第1次失败: 1s + jitter(0-2s)
第2次失败: 2s + jitter(0-2s)
第3次失败: 4s + jitter(0-2s)
第4次失败: 8s + jitter(0-2s)
第5次失败: 16s + jitter(0-2s)
第6次及以后: 30s + jitter(0-2s)
```

**智能重连**：
- 网络恢复检测：监听 `online` 事件，立即重连
- 用户主动重连：提供手动重连按钮
- 重连成功后重置计数器

---

### 2.2 连接质量监控

**监控指标**：
```typescript
interface ConnectionQuality {
  latency: number          // 延迟（ms）
  packetLoss: number       // 丢包率（0-1）
  lastHeartbeat: number    // 最后心跳时间
  reconnectCount: number   // 重连次数
  connectionDuration: number // 连接持续时间
}
```

**自适应心跳**：

**配置**：
```typescript
const HEARTBEAT_CONFIG = {
  minInterval: 10000,   // 最小间隔 10s（网络好时）
  maxInterval: 30000,   // 最大间隔 30s（网络差时）
  timeout: 5000,        // 心跳超时 5s
  samples: 5            // 采样数量
}
```

**自适应算法**：
```typescript
function adjustHeartbeatInterval(quality: ConnectionQuality): number {
  // 延迟低、无丢包：使用最小间隔
  if (quality.latency < 100 && quality.packetLoss === 0) {
    return HEARTBEAT_CONFIG.minInterval
  }
  
  // 延迟高或有丢包：使用最大间隔
  if (quality.latency > 500 || quality.packetLoss > 0.1) {
    return HEARTBEAT_CONFIG.maxInterval
  }
  
  // 线性插值
  const ratio = (quality.latency - 100) / 400
  return HEARTBEAT_CONFIG.minInterval + 
         ratio * (HEARTBEAT_CONFIG.maxInterval - HEARTBEAT_CONFIG.minInterval)
}
```

**心跳检测流程**：
1. 发送 ping，记录发送时间
2. 等待 pong 响应
3. 计算延迟 = 收到时间 - 发送时间
4. 如果超时未收到：标记连接异常，触发重连
5. 根据延迟调整下次心跳间隔

---

### 2.3 连接状态可视化

**状态定义**：
```typescript
enum ConnectionState {
  CONNECTING = 'connecting',    // 连接中
  CONNECTED = 'connected',      // 已连接
  RECONNECTING = 'reconnecting', // 重连中
  DISCONNECTED = 'disconnected', // 已断开
  ERROR = 'error'               // 错误
}
```

**UI 提示**：

| 状态 | 提示 | 图标 | 操作 |
|------|------|------|------|
| CONNECTING | "正在连接..." | 无 | 无 |
| CONNECTED | 无（正常） | 无 | 无 |
| RECONNECTING | "网络不稳定，正在重连..." | 黄色警告 | 无 |
| DISCONNECTED | "网络已断开" | 红色错误 | "重新连接"按钮 |
| ERROR | 具体错误信息 | 红色错误 | "重新登录"按钮 |

---

### 2.4 消息队列和离线处理

**离线消息队列**：
- 连接断开时，消息存入本地队列
- 连接恢复后，自动重发队列中的消息
- 队列持久化到 localStorage，避免刷新丢失

**实现**：
```typescript
interface QueuedMessage {
  id: string
  data: any
  timestamp: number
  retryCount: number
}

const messageQueue: QueuedMessage[] = []

// 发送消息
function sendMessage(data: any) {
  if (isConnected()) {
    ws.send(JSON.stringify(data))
  } else {
    messageQueue.push({
      id: generateId(),
      data,
      timestamp: Date.now(),
      retryCount: 0
    })
    saveQueueToStorage()
  }
}

// 连接恢复后重发
function flushMessageQueue() {
  while (messageQueue.length > 0) {
    const msg = messageQueue.shift()
    ws.send(JSON.stringify(msg.data))
  }
  clearQueueStorage()
}
```

---

### 2.5 错误处理和日志

**错误分类**：
```typescript
enum WSErrorType {
  CONNECTION_FAILED = 'connection_failed',
  AUTH_FAILED = 'auth_failed',
  NETWORK_ERROR = 'network_error',
  SERVER_ERROR = 'server_error',
  TIMEOUT = 'timeout'
}
```

**日志记录**：
- 连接事件（连接、断开、重连）
- 心跳事件（发送、接收、超时）
- 错误事件（类型、时间、详情）
- 性能指标（延迟、丢包率）

**调试工具**：
```typescript
if (import.meta.env.DEV) {
  window.__wsDebug = {
    getState: () => connectionState,
    getQuality: () => connectionQuality,
    getQueue: () => messageQueue,
    forceReconnect: () => reconnect(),
    clearQueue: () => clearQueue()
  }
}
```

---

## 阶段 3：Main.vue 组件拆分

### 3.1 当前状态

**已优化**：
- ✅ onMounted 已分阶段加载数据
- ✅ 已使用 defineAsyncComponent 懒加载部分组件
- ✅ 已使用 Suspense 包裹异步组件

**仍存在问题**：
- ❌ 文件仍然 5178 行
- ❌ 大量业务逻辑函数混杂
- ❌ 状态管理分散

---

### 3.2 拆分策略

**原则**：
1. 不破坏现有功能
2. 保持性能优化
3. 提高可维护性

**拆分方案**：

#### A. 提取 Composables

**按功能域划分**：

```typescript
// 1. 会话相关逻辑
// composables/useConversationLogic.ts
export function useConversationLogic() {
  // 会话选择、切换、搜索
  // 会话列表排序、过滤
  // 会话更新处理
}

// 2. 消息相关逻辑
// composables/useMessageLogic.ts
export function useMessageLogic() {
  // 消息发送、接收、撤回
  // 消息加载、分页
  // 消息已读状态
}

// 3. 群组相关逻辑
// composables/useGroupLogic.ts
export function useGroupLogic() {
  // 群组创建、解散
  // 群成员管理
  // 群权限处理
}

// 4. 组织架构相关逻辑
// composables/useOrganizationLogic.ts
export function useOrganizationLogic() {
  // 组织树加载
  // 部门、员工查询
  // 权限验证
}

// 5. 应用相关逻辑
// composables/useAppLogic.ts
export function useAppLogic() {
  // 应用打开、关闭
  // 应用状态管理
  // 最近使用记录
}

// 6. WebSocket 事件处理
// composables/useWebSocketHandlers.ts
export function useWebSocketHandlers() {
  // 所有 WebSocket 消息处理器
  // 连接状态管理
  // 重连逻辑
}

// 7. UI 状态管理
// composables/useUIState.ts
export function useUIState() {
  // 弹窗状态
  // 菜单状态
  // 加载状态
}
```

---

#### B. 组件懒加载优化

**首屏必需组件**（同步加载）：
- Sidebar
- ChatWindow
- SideOptions
- WindowControls

**用户交互后加载**（懒加载）：
```typescript
const SelfProfileModal = defineAsyncComponent(() => 
  import('../components/modals/SelfProfileModal.vue')
)

const CreateGroupModal = defineAsyncComponent(() => 
  import('../components/modals/CreateGroupModal.vue')
)

const UserDetailPanel = defineAsyncComponent(() => 
  import('../components/organization/UserDetailPanel.vue')
)

const ChannelDetailNew = defineAsyncComponent(() => 
  import('../components/channel/ChannelDetailNew.vue')
)
```

**按需加载**（已实现）：
- FileManagementApp
- NotesApp
- 其他应用组件

---

#### C. 状态管理优化

**提取到 Pinia Store**：

```typescript
// stores/ui.ts - UI 状态
export const useUIStore = defineStore('ui', {
  state: () => ({
    sidebarCollapsed: false,
    activeOption: 'recent',
    showNetworkError: false,
    networkErrorMsg: '',
  })
})

// stores/app.ts - 应用状态
export const useAppStore = defineStore('app', {
  state: () => ({
    selectedAppId: null,
    recentApps: [],
    customApps: [],
  })
})

// stores/organization.ts - 组织架构状态
export const useOrganizationStore = defineStore('organization', {
  state: () => ({
    orgStructure: [],
    selectedDepartment: null,
    selectedUser: null,
  })
})
```

---

### 3.3 重构步骤

**步骤 1：提取 WebSocket 处理器**（低风险）
- 创建 `useWebSocketHandlers.ts`
- 移动所有 WebSocket 消息处理函数
- 在 Main.vue 中引入并使用
- **验证**：WebSocket 功能正常

**步骤 2：提取业务逻辑 Composables**（中风险）
- 按功能域逐个提取
- 每提取一个，立即测试
- **验证**：业务功能正常

**步骤 3：提取状态到 Store**（中风险）
- 创建新的 Store
- 逐步迁移状态
- **验证**：状态管理正常

**步骤 4：组件懒加载**（低风险）
- 识别可懒加载组件
- 添加 defineAsyncComponent
- **验证**：首屏加载时间减少

---

### 3.4 性能指标

**优化目标**：
- 文件大小：5178 行 → < 1000 行
- 首屏加载时间：减少 30%
- 代码可维护性：单一职责，易于测试

**测量方法**：
```typescript
const startTime = performance.now()

onMounted(async () => {
  // ... 加载逻辑
  
  const endTime = performance.now()
  console.log(`首屏加载时间: ${endTime - startTime}ms`)
})
```

---

### 3.5 风险控制

**回滚策略**：
- 每个步骤独立提交
- 保留原文件备份
- 使用 feature branch 开发

**测试策略**：
- 手动测试核心功能
- E2E 测试关键路径
- 性能测试对比

---

## 实施计划

### 时间估算

| 阶段 | 任务 | 时间 | 风险 |
|------|------|------|------|
| 1.1 | 删除图片缓存代码 | 10 分钟 | 零 |
| 1.2-1.4 | 请求重试机制 | 1-2 小时 | 低 |
| 2.1-2.5 | WebSocket 优化 | 半天 | 中 |
| 3.1-3.5 | Main.vue 拆分 | 1-2 天 | 高 |

### 里程碑

**里程碑 1**：阶段 1 完成
- ✅ 删除无用代码
- ✅ 请求重试机制上线
- ✅ 验证网络稳定性提升

**里程碑 2**：阶段 2 完成
- ✅ WebSocket 重连优化上线
- ✅ 连接质量监控上线
- ✅ 验证用户体验提升

**里程碑 3**：阶段 3 完成
- ✅ Main.vue 重构完成
- ✅ 性能指标达标
- ✅ 代码质量提升

---

## 验收标准

### 功能验收

**阶段 1**：
- [ ] 图片缓存代码已删除
- [ ] 请求失败时自动重试
- [ ] 重试次数符合配置
- [ ] 缓存和去重功能正常

**阶段 2**：
- [ ] WebSocket 断开后自动重连
- [ ] 重连延迟符合指数退避算法
- [ ] 连接状态正确显示
- [ ] 离线消息队列正常工作

**阶段 3**：
- [ ] Main.vue 行数 < 1000
- [ ] 所有功能正常工作
- [ ] 首屏加载时间减少 30%
- [ ] 代码结构清晰，易于维护

### 性能验收

**网络性能**：
- [ ] 请求成功率提升 > 20%
- [ ] WebSocket 重连成功率 > 95%
- [ ] 平均重连时间 < 10s

**页面性能**：
- [ ] 首屏加载时间 < 2s
- [ ] Main.vue 文件大小 < 50KB
- [ ] 内存占用稳定，无泄漏

---

## 风险评估

### 技术风险

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|---------|
| 重试导致请求延迟增加 | 中 | 低 | 可配置重试次数和延迟 |
| WebSocket 重连风暴 | 低 | 中 | 抖动机制避免雷崩效应 |
| Main.vue 重构引入 Bug | 中 | 高 | 渐进式重构，充分测试 |

### 业务风险

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|---------|
| 用户体验短暂下降 | 低 | 中 | 灰度发布，快速回滚 |
| 功能缺失 | 低 | 高 | 完整的功能测试 |

---

## 后续优化建议

### 短期（1-2 周）
- 添加性能监控和告警
- 优化数据库查询
- 实现虚拟滚动

### 中期（1-2 月）
- 服务端性能优化
- CDN 和缓存策略
- 代码分割优化

### 长期（3-6 月）
- 微服务架构演进
- 性能测试自动化
- 容量规划

---

## 附录

### A. 相关文档
- [Main.vue 性能优化方案](../plans/2025-05-20-main-performance-optimization.md)
- [WebSocket 设计文档](../../realtime-communication/README.md)
- [请求重试最佳实践](https://axios-http.com/docs/interceptors)

### B. 性能测试工具
- Chrome DevTools Performance
- Lighthouse
- WebPageTest
- k6（后端性能测试）

### C. 监控指标
- 前端：FCP, LCP, TTI, FID
- 网络：请求成功率、平均延迟、错误率
- WebSocket：连接成功率、重连次数、消息延迟
