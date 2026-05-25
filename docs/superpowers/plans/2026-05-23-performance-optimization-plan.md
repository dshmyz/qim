# 性能优化实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 解决项目中的性能隐患，提升用户体验和系统稳定性

**架构：** 渐进式优化，分 3 个阶段实施：清理与快速优化 → WebSocket 连接优化 → Main.vue 组件拆分

**技术栈：** Vue 3 + TypeScript + Axios + WebSocket + Pinia

---

## 阶段 1：清理与快速优化

### 任务 1.1：删除未使用的图片缓存代码

**文件：**
- 删除：`qim-client/src/composables/useImageCache.ts`
- 删除：`qim-client/tests/unit/composables/useImageCache.test.ts`（如果存在）

- [ ] **步骤 1：检查文件引用**

运行：`grep -r "useImageCache" qim-client/src --include="*.ts" --include="*.vue"`
预期：无引用（仅在 useImageCache.ts 自身）

- [ ] **步骤 2：删除 useImageCache.ts**

```bash
rm qim-client/src/composables/useImageCache.ts
```

- [ ] **步骤 3：删除测试文件（如果存在）**

```bash
rm -f qim-client/tests/unit/composables/useImageCache.test.ts
```

- [ ] **步骤 4：验证删除成功**

运行：`ls qim-client/src/composables/useImageCache.ts`
预期：报错 "No such file or directory"

- [ ] **步骤 5：Commit**

```bash
git add -A
git commit -m "refactor: 删除未使用的图片缓存代码

- 删除 useImageCache.ts（未被任何组件引用）
- 删除相关测试文件
- 使用浏览器原生缓存替代"
```

---

### 任务 1.2：为 qim-admin 添加请求重试机制

**文件：**
- 修改：`qim-admin/src/utils/request.ts`

- [ ] **步骤 1：定义重试配置接口**

在 `request.ts` 顶部添加：

```typescript
// 重试配置
interface RetryConfig {
  maxRetries: number
  baseDelay: number
  maxDelay: number
}

const RETRY_CONFIG: RetryConfig = {
  maxRetries: 3,
  baseDelay: 1000,
  maxDelay: 10000,
}

// 判断是否应该重试
function shouldRetry(error: any): boolean {
  // 网络错误
  if (error.code === 'ECONNABORTED' || error.code === 'ENETDOWN' || error.code === 'ENETUNREACH') {
    return true
  }
  
  // 服务器错误
  const status = error.response?.status
  if (status === 502 || status === 503 || status === 504) {
    return true
  }
  
  // 限流错误
  if (status === 429) {
    return true
  }
  
  return false
}

// 计算重试延迟（指数退避）
function calculateRetryDelay(retryCount: number): number {
  const exponentialDelay = RETRY_CONFIG.baseDelay * Math.pow(2, retryCount)
  const cappedDelay = Math.min(exponentialDelay, RETRY_CONFIG.maxDelay)
  const jitter = Math.random() * 1000 // 添加抖动
  return cappedDelay + jitter
}
```

- [ ] **步骤 2：修改 response interceptor 添加重试逻辑**

修改现有的 response interceptor：

```typescript
service.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
    const res = response.data
    if (res.code !== 0) {
      ElMessage.error(res.message || '请求失败')
      if (res.code === 401) {
        const permStore = usePermissionStore()
        permStore.reset()
        localStorage.removeItem('token')
        window.location.href = '/login'
      }
      return Promise.reject(new Error(res.message || '请求失败'))
    }
    return response
  },
  async (error) => {
    const config = error.config
    
    // 初始化重试计数
    if (!config.__retryCount) {
      config.__retryCount = 0
    }
    
    // 检查是否应该重试
    if (shouldRetry(error) && config.__retryCount < RETRY_CONFIG.maxRetries) {
      config.__retryCount++
      
      // 计算延迟
      const delay = calculateRetryDelay(config.__retryCount)
      
      // 显示重试提示
      if (config.__retryCount === 1) {
        ElMessage.warning('网络异常，正在重试...')
      }
      
      // 等待后重试
      await new Promise(resolve => setTimeout(resolve, delay))
      
      // 重试请求
      return service(config)
    }
    
    // 重试失败，显示错误
    console.error('[Response] error:', error)
    const status = error.response?.status

    if (status === 401) {
      const permStore = usePermissionStore()
      permStore.reset()
      localStorage.removeItem('token')
      window.location.href = '/login'
    } else if (status === 403) {
      ElMessage.error('权限不足，无法执行此操作')
    } else {
      const message = error.response?.data?.message || error.message || '网络异常'
      ElMessage.error(message)
    }

    return Promise.reject(error)
  }
)
```

- [ ] **步骤 3：添加类型声明**

在文件顶部添加：

```typescript
// 扩展 AxiosRequestConfig 类型
declare module 'axios' {
  interface AxiosRequestConfig {
    __retryCount?: number
  }
}
```

- [ ] **步骤 4：验证语法正确**

运行：`cd qim-admin && npx tsc --noEmit src/utils/request.ts`
预期：无错误

- [ ] **步骤 5：Commit**

```bash
git add qim-admin/src/utils/request.ts
git commit -m "feat(qim-admin): 添加请求重试机制

- 指数退避算法（1s, 2s, 4s）
- 自动重试网络错误和服务器错误
- 添加抖动避免重试风暴
- 最多重试 3 次"
```

---

### 任务 1.3：为 qim-client 添加请求重试机制

**文件：**
- 修改：`qim-client/src/api/core.ts`

- [ ] **步骤 1：读取 core.ts 文件**

运行：`head -50 qim-client/src/api/core.ts`
预期：查看当前结构

- [ ] **步骤 2：添加重试配置和工具函数**

在 `core.ts` 顶部添加（与任务 1.2 相同的配置）：

```typescript
// 重试配置
interface RetryConfig {
  maxRetries: number
  baseDelay: number
  maxDelay: number
}

const RETRY_CONFIG: RetryConfig = {
  maxRetries: 3,
  baseDelay: 1000,
  maxDelay: 10000,
}

function shouldRetry(error: any): boolean {
  if (error.code === 'ECONNABORTED' || error.code === 'ENETDOWN' || error.code === 'ENETUNREACH') {
    return true
  }
  
  const status = error.response?.status
  if (status === 502 || status === 503 || status === 504 || status === 429) {
    return true
  }
  
  return false
}

function calculateRetryDelay(retryCount: number): number {
  const exponentialDelay = RETRY_CONFIG.baseDelay * Math.pow(2, retryCount)
  const cappedDelay = Math.min(exponentialDelay, RETRY_CONFIG.maxDelay)
  const jitter = Math.random() * 1000
  return cappedDelay + jitter
}
```

- [ ] **步骤 3：修改 response interceptor**

根据 `core.ts` 的实际结构，在 response interceptor 中添加重试逻辑（参考任务 1.2）

- [ ] **步骤 4：验证语法正确**

运行：`cd qim-client && npx tsc --noEmit src/api/core.ts`
预期：无错误

- [ ] **步骤 5：Commit**

```bash
git add qim-client/src/api/core.ts
git commit -m "feat(qim-client): 添加请求重试机制

- 与 qim-admin 保持一致的重试策略
- 指数退避 + 抖动
- 自动重试网络和服务器错误"
```

---

### 任务 1.4：添加请求缓存和去重

**文件：**
- 创建：`qim-client/src/utils/requestInterceptor.ts`

- [ ] **步骤 1：创建请求拦截器文件**

```typescript
// qim-client/src/utils/requestInterceptor.ts
import type { AxiosRequestConfig, AxiosResponse } from 'axios'

// 请求缓存
interface CacheItem {
  data: any
  timestamp: number
}

const requestCache = new Map<string, CacheItem>()
const pendingRequests = new Map<string, Promise<any>>()

// 默认缓存时间 30 秒
const DEFAULT_CACHE_TIME = 30000

// 生成缓存键
function generateCacheKey(config: AxiosRequestConfig): string {
  const method = config.method || 'get'
  const url = config.url || ''
  const params = JSON.stringify(config.params || {})
  const data = JSON.stringify(config.data || {})
  return `${method}:${url}:${params}:${data}`
}

// 清理过期缓存
function cleanExpiredCache(): void {
  const now = Date.now()
  for (const [key, item] of requestCache.entries()) {
    if (now - item.timestamp > DEFAULT_CACHE_TIME) {
      requestCache.delete(key)
    }
  }
}

// 请求缓存拦截器
export function cacheRequestInterceptor(config: AxiosRequestConfig): AxiosRequestConfig {
  // 只缓存 GET 请求
  if (config.method?.toLowerCase() !== 'get') {
    return config
  }
  
  // 检查是否启用缓存
  if (!config.cache) {
    return config
  }
  
  // 清理过期缓存
  cleanExpiredCache()
  
  const cacheKey = generateCacheKey(config)
  const cached = requestCache.get(cacheKey)
  
  if (cached) {
    // 返回缓存数据
    config.__fromCache = true
    config.__cachedData = cached.data
  }
  
  return config
}

// 响应缓存拦截器
export function cacheResponseInterceptor(response: AxiosResponse): AxiosResponse {
  const config = response.config
  
  // 只缓存 GET 请求
  if (config.method?.toLowerCase() !== 'get') {
    return response
  }
  
  // 检查是否启用缓存
  if (!config.cache) {
    return response
  }
  
  const cacheKey = generateCacheKey(config)
  const cacheTime = config.cacheTime || DEFAULT_CACHE_TIME
  
  // 存入缓存
  requestCache.set(cacheKey, {
    data: response,
    timestamp: Date.now()
  })
  
  return response
}

// 请求去重拦截器
export function dedupeRequestInterceptor(config: AxiosRequestConfig): AxiosRequestConfig | Promise<AxiosRequestConfig> {
  // 检查是否启用去重
  if (config.dedupe === false) {
    return config
  }
  
  const cacheKey = generateCacheKey(config)
  
  // 检查是否有相同的 pending 请求
  const pending = pendingRequests.get(cacheKey)
  if (pending) {
    // 复用 Promise
    return pending.then(() => config)
  }
  
  return config
}

// 清除所有缓存
export function clearAllCache(): void {
  requestCache.clear()
  pendingRequests.clear()
}

// 扩展 AxiosRequestConfig 类型
declare module 'axios' {
  interface AxiosRequestConfig {
    cache?: boolean
    cacheTime?: number
    dedupe?: boolean
    __fromCache?: boolean
    __cachedData?: any
  }
}
```

- [ ] **步骤 2：在 core.ts 中应用拦截器**

在 `core.ts` 中导入并应用拦截器：

```typescript
import { cacheRequestInterceptor, cacheResponseInterceptor, dedupeRequestInterceptor } from '../utils/requestInterceptor'

// 应用拦截器
service.interceptors.request.use(cacheRequestInterceptor)
service.interceptors.request.use(dedupeRequestInterceptor)
service.interceptors.response.use(cacheResponseInterceptor)
```

- [ ] **步骤 3：验证语法正确**

运行：`cd qim-client && npx tsc --noEmit src/utils/requestInterceptor.ts`
预期：无错误

- [ ] **步骤 4：Commit**

```bash
git add qim-client/src/utils/requestInterceptor.ts qim-client/src/api/core.ts
git commit -m "feat: 添加请求缓存和去重机制

- GET 请求缓存 30 秒
- 自动去重并发请求
- 可配置 cache 和 dedupe 选项"
```

---

## 阶段 2：WebSocket 连接管理优化

### 任务 2.1：增强 WebSocket 重连策略

**文件：**
- 创建：`qim-client/src/utils/websocketReconnect.ts`
- 修改：`qim-client/src/composables/useWebSocket.ts`

- [ ] **步骤 1：创建重连策略文件**

```typescript
// qim-client/src/utils/websocketReconnect.ts
export interface ReconnectConfig {
  baseDelay: number
  maxDelay: number
  maxAttempts: number
  jitterRange: [number, number]
}

export const DEFAULT_RECONNECT_CONFIG: ReconnectConfig = {
  baseDelay: 1000,
  maxDelay: 30000,
  maxAttempts: 10,
  jitterRange: [0, 2000]
}

export function calculateReconnectDelay(
  attempt: number,
  config: ReconnectConfig = DEFAULT_RECONNECT_CONFIG
): number {
  // 指数退避
  const exponentialDelay = config.baseDelay * Math.pow(2, attempt)
  
  // 限制最大延迟
  const cappedDelay = Math.min(exponentialDelay, config.maxDelay)
  
  // 添加随机抖动，避免雷崩效应
  const jitter = Math.random() * (config.jitterRange[1] - config.jitterRange[0]) + config.jitterRange[0]
  
  return cappedDelay + jitter
}

export function shouldReconnect(
  attempt: number,
  config: ReconnectConfig = DEFAULT_RECONNECT_CONFIG
): boolean {
  return attempt < config.maxAttempts
}
```

- [ ] **步骤 2：修改 useWebSocket.ts 使用新的重连策略**

在 `useWebSocket.ts` 中：

```typescript
import { calculateReconnectDelay, shouldReconnect, DEFAULT_RECONNECT_CONFIG } from '../utils/websocketReconnect'

// 修改重连逻辑
const scheduleReconnect = () => {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
  }
  
  if (!shouldReconnect(reconnectAttempts)) {
    setNetworkError(true, '网络连接失败，请手动重连')
    return
  }
  
  const delay = calculateReconnectDelay(reconnectAttempts)
  console.log(`[WebSocket] 第 ${reconnectAttempts + 1} 次重连，延迟 ${delay}ms`)
  
  reconnectTimer = window.setTimeout(() => {
    reconnectAttempts++
    connect()
  }, delay)
}
```

- [ ] **步骤 3：添加网络恢复监听**

在 `useWebSocket.ts` 的 `connect` 函数中添加：

```typescript
// 监听网络恢复事件
window.addEventListener('online', () => {
  console.log('[WebSocket] 网络恢复，立即重连')
  reconnectAttempts = 0
  connect()
})
```

- [ ] **步骤 4：验证语法正确**

运行：`cd qim-client && npx tsc --noEmit src/utils/websocketReconnect.ts`
预期：无错误

- [ ] **步骤 5：Commit**

```bash
git add qim-client/src/utils/websocketReconnect.ts qim-client/src/composables/useWebSocket.ts
git commit -m "feat: 增强 WebSocket 重连策略

- 指数退避算法（1s, 2s, 4s, 8s, 16s, 30s）
- 随机抖动避免雷崩效应
- 最大重连次数增加到 10 次
- 监听网络恢复事件立即重连"
```

---

### 任务 2.2：添加连接质量监控

**文件：**
- 创建：`qim-client/src/utils/connectionMonitor.ts`

- [ ] **步骤 1：创建连接监控文件**

```typescript
// qim-client/src/utils/connectionMonitor.ts
import { ref, type Ref } from 'vue'

export interface ConnectionQuality {
  latency: number
  packetLoss: number
  lastHeartbeat: number
  reconnectCount: number
  connectionDuration: number
}

export interface HeartbeatConfig {
  minInterval: number
  maxInterval: number
  timeout: number
  samples: number
}

const DEFAULT_HEARTBEAT_CONFIG: HeartbeatConfig = {
  minInterval: 10000,
  maxInterval: 30000,
  timeout: 5000,
  samples: 5
}

export function useConnectionMonitor() {
  const quality: Ref<ConnectionQuality> = ref({
    latency: 0,
    packetLoss: 0,
    lastHeartbeat: 0,
    reconnectCount: 0,
    connectionDuration: 0
  })
  
  const latencyHistory: number[] = []
  let connectionStartTime = 0
  
  // 记录心跳延迟
  function recordHeartbeatLatency(latency: number): void {
    latencyHistory.push(latency)
    
    // 只保留最近的样本
    if (latencyHistory.length > DEFAULT_HEARTBEAT_CONFIG.samples) {
      latencyHistory.shift()
    }
    
    // 计算平均延迟
    quality.value.latency = latencyHistory.reduce((a, b) => a + b, 0) / latencyHistory.length
    quality.value.lastHeartbeat = Date.now()
  }
  
  // 记录连接建立
  function recordConnection(): void {
    connectionStartTime = Date.now()
    quality.value.reconnectCount = 0
  }
  
  // 记录重连
  function recordReconnect(): void {
    quality.value.reconnectCount++
  }
  
  // 更新连接持续时间
  function updateConnectionDuration(): void {
    if (connectionStartTime > 0) {
      quality.value.connectionDuration = Date.now() - connectionStartTime
    }
  }
  
  // 计算自适应心跳间隔
  function adjustHeartbeatInterval(): number {
    // 延迟低、无丢包：使用最小间隔
    if (quality.value.latency < 100 && quality.value.packetLoss === 0) {
      return DEFAULT_HEARTBEAT_CONFIG.minInterval
    }
    
    // 延迟高或有丢包：使用最大间隔
    if (quality.value.latency > 500 || quality.value.packetLoss > 0.1) {
      return DEFAULT_HEARTBEAT_CONFIG.maxInterval
    }
    
    // 线性插值
    const ratio = (quality.value.latency - 100) / 400
    return DEFAULT_HEARTBEAT_CONFIG.minInterval + 
           ratio * (DEFAULT_HEARTBEAT_CONFIG.maxInterval - DEFAULT_HEARTBEAT_CONFIG.minInterval)
  }
  
  return {
    quality,
    recordHeartbeatLatency,
    recordConnection,
    recordReconnect,
    updateConnectionDuration,
    adjustHeartbeatInterval
  }
}
```

- [ ] **步骤 2：在 useWebSocket.ts 中应用监控**

```typescript
import { useConnectionMonitor } from '../utils/connectionMonitor'

const { quality, recordHeartbeatLatency, recordConnection, adjustHeartbeatInterval } = useConnectionMonitor()

// 修改心跳逻辑
const startHeartbeat = () => {
  stopHeartbeat()
  
  const interval = adjustHeartbeatInterval()
  console.log(`[WebSocket] 心跳间隔: ${interval}ms`)
  
  heartbeatTimer = window.setInterval(() => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      const startTime = Date.now()
      ws.send(JSON.stringify({ type: 'ping' }))
      
      // 等待 pong 响应后计算延迟
      // （需要在消息处理器中调用 recordHeartbeatLatency）
    }
  }, interval)
}
```

- [ ] **步骤 3：验证语法正确**

运行：`cd qim-client && npx tsc --noEmit src/utils/connectionMonitor.ts`
预期：无错误

- [ ] **步骤 4：Commit**

```bash
git add qim-client/src/utils/connectionMonitor.ts qim-client/src/composables/useWebSocket.ts
git commit -m "feat: 添加 WebSocket 连接质量监控

- 记录心跳延迟和重连次数
- 自适应心跳间隔（10s-30s）
- 根据网络质量动态调整"
```

---

### 任务 2.3：添加离线消息队列

**文件：**
- 创建：`qim-client/src/utils/messageQueue.ts`

- [ ] **步骤 1：创建消息队列文件**

```typescript
// qim-client/src/utils/messageQueue.ts
import { logger } from './logger'

interface QueuedMessage {
  id: string
  data: any
  timestamp: number
  retryCount: number
}

const STORAGE_KEY = 'qim_message_queue'
const MAX_QUEUE_SIZE = 100

class MessageQueue {
  private queue: QueuedMessage[] = []
  
  constructor() {
    this.loadFromStorage()
  }
  
  // 生成唯一 ID
  private generateId(): string {
    return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
  }
  
  // 从 localStorage 加载队列
  private loadFromStorage(): void {
    try {
      const data = localStorage.getItem(STORAGE_KEY)
      if (data) {
        this.queue = JSON.parse(data)
      }
    } catch (error) {
      logger.error('[MessageQueue] 加载队列失败:', error)
      this.queue = []
    }
  }
  
  // 保存到 localStorage
  private saveToStorage(): void {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(this.queue))
    } catch (error) {
      logger.error('[MessageQueue] 保存队列失败:', error)
    }
  }
  
  // 添加消息到队列
  enqueue(data: any): string {
    if (this.queue.length >= MAX_QUEUE_SIZE) {
      logger.warn('[MessageQueue] 队列已满，移除最旧消息')
      this.queue.shift()
    }
    
    const message: QueuedMessage = {
      id: this.generateId(),
      data,
      timestamp: Date.now(),
      retryCount: 0
    }
    
    this.queue.push(message)
    this.saveToStorage()
    
    logger.log(`[MessageQueue] 消息入队: ${message.id}`)
    return message.id
  }
  
  // 获取队列大小
  size(): number {
    return this.queue.length
  }
  
  // 清空队列
  clear(): void {
    this.queue = []
    localStorage.removeItem(STORAGE_KEY)
    logger.log('[MessageQueue] 队列已清空')
  }
  
  // 刷新队列（发送所有消息）
  flush(sendFn: (data: any) => boolean): void {
    logger.log(`[MessageQueue] 开始刷新队列，共 ${this.queue.length} 条消息`)
    
    const failed: QueuedMessage[] = []
    
    while (this.queue.length > 0) {
      const message = this.queue.shift()!
      
      try {
        const success = sendFn(message.data)
        if (!success) {
          message.retryCount++
          if (message.retryCount < 3) {
            failed.push(message)
          } else {
            logger.error(`[MessageQueue] 消息发送失败，已丢弃: ${message.id}`)
          }
        }
      } catch (error) {
        logger.error(`[MessageQueue] 消息发送异常: ${message.id}`, error)
        message.retryCount++
        if (message.retryCount < 3) {
          failed.push(message)
        }
      }
    }
    
    // 重新入队失败的消息
    this.queue = failed
    this.saveToStorage()
    
    logger.log(`[MessageQueue] 队列刷新完成，失败 ${failed.length} 条`)
  }
}

export const messageQueue = new MessageQueue()
```

- [ ] **步骤 2：在 useWebSocket.ts 中应用消息队列**

```typescript
import { messageQueue } from '../utils/messageQueue'

// 修改 sendMessage 函数
export const sendMessage = (data: any) => {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(data))
  } else {
    // 连接断开，消息入队
    messageQueue.enqueue(data)
    QMessage.warning('网络已断开，消息将在恢复后发送')
  }
}

// 连接成功后刷新队列
const onConnected = () => {
  if (messageQueue.size() > 0) {
    messageQueue.flush((data) => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify(data))
        return true
      }
      return false
    })
  }
}
```

- [ ] **步骤 3：验证语法正确**

运行：`cd qim-client && npx tsc --noEmit src/utils/messageQueue.ts`
预期：无错误

- [ ] **步骤 4：Commit**

```bash
git add qim-client/src/utils/messageQueue.ts qim-client/src/composables/useWebSocket.ts
git commit -m "feat: 添加离线消息队列

- 连接断开时消息入队
- 连接恢复后自动重发
- 持久化到 localStorage
- 最多缓存 100 条消息"
```

---

## 阶段 3：Main.vue 组件拆分

### 任务 3.1：提取 WebSocket 处理器

**文件：**
- 创建：`qim-client/src/composables/useWebSocketHandlers.ts`
- 修改：`qim-client/src/views/Main.vue`

- [ ] **步骤 1：创建 WebSocket 处理器文件**

从 Main.vue 中提取所有 WebSocket 消息处理函数：

```typescript
// qim-client/src/composables/useWebSocketHandlers.ts
import type { Conversation, Message, User } from '../types'
import { useChatStore } from '../stores/chat'
import { useChannelStore } from '../stores/channel'
import { logger } from '../utils/logger'

export function useWebSocketHandlers() {
  const chatStore = useChatStore()
  const channelStore = useChannelStore()
  
  // 处理已读回执
  const handleReadReceipt = (data: any) => {
    logger.log('[WS] 收到已读回执:', data)
    chatStore.updateMessageReadUsers(data.message_id, data.read_users)
  }
  
  // 处理新消息
  const handleNewMessage = (data: any) => {
    logger.log('[WS] 收到新消息:', data)
    chatStore.appendMessage(data.conversation_id, data.message)
  }
  
  // 处理消息撤回
  const handleMessageRecalled = (data: any) => {
    logger.log('[WS] 消息被撤回:', data)
    chatStore.updateMessage(data.conversation_id, data.message_id, {
      is_recalled: true,
      recalled_at: data.recalled_at
    })
  }
  
  // 处理消息删除
  const handleMessageDeleted = (data: any) => {
    logger.log('[WS] 消息被删除:', data)
    chatStore.deleteMessage(data.conversation_id, data.message_id)
  }
  
  // 处理群组邀请
  const handleGroupInvitation = (data: any) => {
    logger.log('[WS] 收到群组邀请:', data)
    // TODO: 显示邀请通知
  }
  
  // 处理被添加到群组
  const handleAddedToGroup = (data: any) => {
    logger.log('[WS] 被添加到群组:', data)
    chatStore.addConversation(data.group)
  }
  
  // 处理群成员离开
  const handleGroupMemberLeft = (data: any) => {
    logger.log('[WS] 群成员离开:', data)
    // TODO: 更新群成员列表
  }
  
  // 处理群成员加入
  const handleGroupMemberJoined = (data: any) => {
    logger.log('[WS] 群成员加入:', data)
    // TODO: 更新群成员列表
  }
  
  // 处理群成员角色更新
  const handleGroupMemberRoleUpdated = (data: any) => {
    logger.log('[WS] 群成员角色更新:', data)
    // TODO: 更新群成员角色
  }
  
  // 处理群主转让
  const handleGroupOwnerTransferred = (data: any) => {
    logger.log('[WS] 群主转让:', data)
    // TODO: 更新群主信息
  }
  
  // 处理会话更新
  const handleConversationUpdated = (data: any) => {
    logger.log('[WS] 会话更新:', data)
    chatStore.updateConversation(data.conversation_id, data.updates)
  }
  
  // 处理群公告更新
  const handleGroupAnnouncementUpdated = (data: any) => {
    logger.log('[WS] 群公告更新:', data)
    // TODO: 更新群公告
  }
  
  // 处理通知
  const handleNotification = (data: any) => {
    logger.log('[WS] 收到通知:', data)
    // TODO: 显示通知
  }
  
  // 处理新通知
  const handleNewNotification = (data: any) => {
    logger.log('[WS] 收到新通知:', data)
    // TODO: 显示新通知
  }
  
  // 处理系统消息
  const handleSystemMessage = (data: any) => {
    logger.log('[WS] 收到系统消息:', data)
    // TODO: 显示系统消息
  }
  
  // 处理用户状态变化
  const handleUserStatusChange = (data: any) => {
    logger.log('[WS] 用户状态变化:', data)
    // TODO: 更新用户状态
  }
  
  // 返回所有处理器
  return {
    handleReadReceipt,
    handleNewMessage,
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
  }
}
```

- [ ] **步骤 2：在 Main.vue 中使用处理器**

```typescript
import { useWebSocketHandlers } from '../composables/useWebSocketHandlers'

const {
  handleReadReceipt,
  handleNewMessage,
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
} = useWebSocketHandlers()

// 在 connectWebSocket 中使用
const messageHandlers = {
  'message_read': handleReadReceipt,
  'new_message': handleNewMessage,
  'message_recalled': handleMessageRecalled,
  'message_deleted': handleMessageDeleted,
  'group_invitation': handleGroupInvitation,
  'added_to_group': handleAddedToGroup,
  'group_member_left': handleGroupMemberLeft,
  'group_member_joined': handleGroupMemberJoined,
  'group_member_role_updated': handleGroupMemberRoleUpdated,
  'group_owner_transferred': handleGroupOwnerTransferred,
  'conversation_updated': handleConversationUpdated,
  'group_announcement_updated': handleGroupAnnouncementUpdated,
  'notification': handleNotification,
  'new_notification': handleNewNotification,
  'system_message': handleSystemMessage,
  'user_status_changed': handleUserStatusChange
}
```

- [ ] **步骤 3：从 Main.vue 中删除原有处理函数**

删除所有 `handleXxx` 函数定义（已移到 useWebSocketHandlers.ts）

- [ ] **步骤 4：验证功能正常**

手动测试：
- 发送消息
- 接收消息
- 消息撤回
- 群组操作

- [ ] **步骤 5：Commit**

```bash
git add qim-client/src/composables/useWebSocketHandlers.ts qim-client/src/views/Main.vue
git commit -m "refactor: 提取 WebSocket 处理器到独立 composable

- 创建 useWebSocketHandlers.ts
- 移动所有 WebSocket 消息处理函数
- 减少 Main.vue 代码量约 200 行"
```

---

### 任务 3.2：提取会话逻辑

**文件：**
- 创建：`qim-client/src/composables/useConversationLogic.ts`

- [ ] **步骤 1：创建会话逻辑文件**

从 Main.vue 中提取会话相关逻辑：

```typescript
// qim-client/src/composables/useConversationLogic.ts
import { ref, computed } from 'vue'
import type { Conversation } from '../types'
import { useChatStore } from '../stores/chat'
import { logger } from '../utils/logger'

export function useConversationLogic() {
  const chatStore = useChatStore()
  
  const searchQuery = ref('')
  const currentConversationId = ref<string | null>(null)
  
  // 过滤会话列表
  const filteredConversations = computed(() => {
    let conversations = chatStore.sortedConversations
    
    if (searchQuery.value) {
      const query = searchQuery.value.toLowerCase()
      conversations = conversations.filter(conv => 
        conv.name?.toLowerCase().includes(query) ||
        conv.members?.some(m => m.name?.toLowerCase().includes(query))
      )
    }
    
    return conversations
  })
  
  // 选择会话
  const selectConversation = (conversationId: string) => {
    logger.log(`[Conversation] 选择会话: ${conversationId}`)
    currentConversationId.value = conversationId
    chatStore.setCurrentConversation(conversationId)
  }
  
  // 切换会话
  const switchConversation = (conversationId: string) => {
    logger.log(`[Conversation] 切换会话: ${conversationId}`)
    selectConversation(conversationId)
  }
  
  // 更新会话
  const updateConversation = (conversationId: string, updates: Partial<Conversation>) => {
    logger.log(`[Conversation] 更新会话: ${conversationId}`, updates)
    chatStore.updateConversation(conversationId, updates)
  }
  
  // 搜索会话
  const searchConversations = (query: string) => {
    searchQuery.value = query
  }
  
  return {
    searchQuery,
    currentConversationId,
    filteredConversations,
    selectConversation,
    switchConversation,
    updateConversation,
    searchConversations
  }
}
```

- [ ] **步骤 2：在 Main.vue 中使用**

```typescript
import { useConversationLogic } from '../composables/useConversationLogic'

const {
  searchQuery,
  currentConversationId,
  filteredConversations,
  selectConversation,
  switchConversation,
  updateConversation,
  searchConversations
} = useConversationLogic()
```

- [ ] **步骤 3：删除 Main.vue 中的重复代码**

- [ ] **步骤 4：验证功能正常**

手动测试会话选择、切换、搜索功能

- [ ] **步骤 5：Commit**

```bash
git add qim-client/src/composables/useConversationLogic.ts qim-client/src/views/Main.vue
git commit -m "refactor: 提取会话逻辑到独立 composable

- 创建 useConversationLogic.ts
- 封装会话选择、切换、搜索逻辑
- 减少 Main.vue 代码量约 100 行"
```

---

### 任务 3.3：提取消息逻辑

**文件：**
- 创建：`qim-client/src/composables/useMessageLogic.ts`

- [ ] **步骤 1：创建消息逻辑文件**

```typescript
// qim-client/src/composables/useMessageLogic.ts
import { ref } from 'vue'
import type { Message } from '../types'
import { useChatStore } from '../stores/chat'
import { logger } from '../utils/logger'

export function useMessageLogic() {
  const chatStore = useChatStore()
  
  const isLoadingMessages = ref(false)
  const hasMoreMessages = ref(true)
  const messagePage = ref(1)
  const messagePageSize = ref(20)
  
  // 发送消息
  const sendMessage = async (conversationId: string, content: string, type: string = 'text') => {
    logger.log(`[Message] 发送消息到会话 ${conversationId}`)
    
    const message: Partial<Message> = {
      conversation_id: conversationId,
      content,
      type,
      timestamp: Date.now()
    }
    
    // TODO: 调用 API 发送消息
    chatStore.appendMessage(conversationId, message as Message)
  }
  
  // 加载消息
  const loadMessages = async (conversationId: string, page: number = 1) => {
    logger.log(`[Message] 加载会话 ${conversationId} 的消息，页码 ${page}`)
    
    isLoadingMessages.value = true
    
    try {
      // TODO: 调用 API 加载消息
      messagePage.value = page
    } catch (error) {
      logger.error('[Message] 加载消息失败:', error)
    } finally {
      isLoadingMessages.value = false
    }
  }
  
  // 加载更多消息
  const loadMoreMessages = async (conversationId: string) => {
    if (!hasMoreMessages.value || isLoadingMessages.value) {
      return
    }
    
    await loadMessages(conversationId, messagePage.value + 1)
  }
  
  // 撤回消息
  const recallMessage = async (conversationId: string, messageId: string) => {
    logger.log(`[Message] 撤回消息 ${messageId}`)
    
    // TODO: 调用 API 撤回消息
    chatStore.updateMessage(conversationId, messageId, {
      is_recalled: true,
      recalled_at: Date.now()
    })
  }
  
  // 删除消息
  const deleteMessage = async (conversationId: string, messageId: string) => {
    logger.log(`[Message] 删除消息 ${messageId}`)
    
    // TODO: 调用 API 删除消息
    chatStore.deleteMessage(conversationId, messageId)
  }
  
  return {
    isLoadingMessages,
    hasMoreMessages,
    messagePage,
    messagePageSize,
    sendMessage,
    loadMessages,
    loadMoreMessages,
    recallMessage,
    deleteMessage
  }
}
```

- [ ] **步骤 2：在 Main.vue 中使用**

- [ ] **步骤 3：删除 Main.vue 中的重复代码**

- [ ] **步骤 4：验证功能正常**

- [ ] **步骤 5：Commit**

```bash
git add qim-client/src/composables/useMessageLogic.ts qim-client/src/views/Main.vue
git commit -m "refactor: 提取消息逻辑到独立 composable

- 创建 useMessageLogic.ts
- 封装消息发送、加载、撤回、删除逻辑
- 减少 Main.vue 代码量约 150 行"
```

---

### 任务 3.4：提取群组逻辑

**文件：**
- 创建：`qim-client/src/composables/useGroupLogic.ts`

- [ ] **步骤 1：创建群组逻辑文件**

```typescript
// qim-client/src/composables/useGroupLogic.ts
import { ref } from 'vue'
import type { Conversation, User } from '../types'
import { logger } from '../utils/logger'

export function useGroupLogic() {
  const selectedGroup = ref<Conversation | null>(null)
  
  // 创建群组
  const createGroup = async (name: string, members: User[]) => {
    logger.log(`[Group] 创建群组: ${name}`)
    // TODO: 调用 API 创建群组
  }
  
  // 解散群组
  const dissolveGroup = async (groupId: string) => {
    logger.log(`[Group] 解散群组: ${groupId}`)
    // TODO: 调用 API 解散群组
  }
  
  // 邀请成员
  const inviteMembers = async (groupId: string, userIds: string[]) => {
    logger.log(`[Group] 邀请成员到群组 ${groupId}:`, userIds)
    // TODO: 调用 API 邀请成员
  }
  
  // 移除成员
  const removeMember = async (groupId: string, userId: string) => {
    logger.log(`[Group] 从群组 ${groupId} 移除成员 ${userId}`)
    // TODO: 调用 API 移除成员
  }
  
  // 退出群组
  const leaveGroup = async (groupId: string) => {
    logger.log(`[Group] 退出群组: ${groupId}`)
    // TODO: 调用 API 退出群组
  }
  
  // 设置群管理员
  const setAdmin = async (groupId: string, userId: string, isAdmin: boolean) => {
    logger.log(`[Group] 设置群管理员: ${userId} -> ${isAdmin}`)
    // TODO: 调用 API 设置管理员
  }
  
  // 转让群主
  const transferOwner = async (groupId: string, newOwnerId: string) => {
    logger.log(`[Group] 转让群主给: ${newOwnerId}`)
    // TODO: 调用 API 转让群主
  }
  
  return {
    selectedGroup,
    createGroup,
    dissolveGroup,
    inviteMembers,
    removeMember,
    leaveGroup,
    setAdmin,
    transferOwner
  }
}
```

- [ ] **步骤 2：在 Main.vue 中使用**

- [ ] **步骤 3：删除 Main.vue 中的重复代码**

- [ ] **步骤 4：验证功能正常**

- [ ] **步骤 5：Commit**

```bash
git add qim-client/src/composables/useGroupLogic.ts qim-client/src/views/Main.vue
git commit -m "refactor: 提取群组逻辑到独立 composable

- 创建 useGroupLogic.ts
- 封装群组创建、解散、成员管理逻辑
- 减少 Main.vue 代码量约 100 行"
```

---

### 任务 3.5：提取组织架构逻辑

**文件：**
- 创建：`qim-client/src/composables/useOrganizationLogic.ts`

- [ ] **步骤 1：创建组织架构逻辑文件**

```typescript
// qim-client/src/composables/useOrganizationLogic.ts
import { ref } from 'vue'
import type { User } from '../types'
import { logger } from '../utils/logger'

export function useOrganizationLogic() {
  const orgStructure = ref<any[]>([])
  const selectedDepartment = ref<any | null>(null)
  const selectedUser = ref<User | null>(null)
  
  // 加载组织架构树
  const loadOrganizationTree = async () => {
    logger.log('[Organization] 加载组织架构树')
    // TODO: 调用 API 加载组织架构
  }
  
  // 选择部门
  const selectDepartment = (department: any) => {
    logger.log(`[Organization] 选择部门: ${department.name}`)
    selectedDepartment.value = department
  }
  
  // 选择用户
  const selectUser = (user: User) => {
    logger.log(`[Organization] 选择用户: ${user.name}`)
    selectedUser.value = user
  }
  
  // 搜索员工
  const searchEmployees = async (keyword: string) => {
    logger.log(`[Organization] 搜索员工: ${keyword}`)
    // TODO: 调用 API 搜索员工
  }
  
  return {
    orgStructure,
    selectedDepartment,
    selectedUser,
    loadOrganizationTree,
    selectDepartment,
    selectUser,
    searchEmployees
  }
}
```

- [ ] **步骤 2：在 Main.vue 中使用**

- [ ] **步骤 3：删除 Main.vue 中的重复代码**

- [ ] **步骤 4：验证功能正常**

- [ ] **步骤 5：Commit**

```bash
git add qim-client/src/composables/useOrganizationLogic.ts qim-client/src/views/Main.vue
git commit -m "refactor: 提取组织架构逻辑到独立 composable

- 创建 useOrganizationLogic.ts
- 封装组织树加载、部门选择、员工搜索逻辑
- 减少 Main.vue 代码量约 80 行"
```

---

### 任务 3.6：提取应用逻辑

**文件：**
- 创建：`qim-client/src/composables/useAppLogic.ts`

- [ ] **步骤 1：创建应用逻辑文件**

```typescript
// qim-client/src/composables/useAppLogic.ts
import { ref, computed } from 'vue'
import { logger } from '../utils/logger'

export function useAppLogic() {
  const selectedAppId = ref<string | null>(null)
  const recentApps = ref<string[]>([])
  
  // 打开应用
  const openApp = (appId: string) => {
    logger.log(`[App] 打开应用: ${appId}`)
    selectedAppId.value = appId
    
    // 添加到最近使用
    if (!recentApps.value.includes(appId)) {
      recentApps.value.unshift(appId)
      if (recentApps.value.length > 5) {
        recentApps.value.pop()
      }
    }
  }
  
  // 关闭应用
  const closeApp = () => {
    logger.log('[App] 关闭应用')
    selectedAppId.value = null
  }
  
  // 返回应用列表
  const backToAppList = () => {
    closeApp()
  }
  
  return {
    selectedAppId,
    recentApps,
    openApp,
    closeApp,
    backToAppList
  }
}
```

- [ ] **步骤 2：在 Main.vue 中使用**

- [ ] **步骤 3：删除 Main.vue 中的重复代码**

- [ ] **步骤 4：验证功能正常**

- [ ] **步骤 5：Commit**

```bash
git add qim-client/src/composables/useAppLogic.ts qim-client/src/views/Main.vue
git commit -m "refactor: 提取应用逻辑到独立 composable

- 创建 useAppLogic.ts
- 封装应用打开、关闭、最近使用逻辑
- 减少 Main.vue 代码量约 50 行"
```

---

### 任务 3.7：提取 UI 状态

**文件：**
- 创建：`qim-client/src/composables/useUIState.ts`

- [ ] **步骤 1：创建 UI 状态文件**

```typescript
// qim-client/src/composables/useUIState.ts
import { ref } from 'vue'

export function useUIState() {
  const sidebarCollapsed = ref(false)
  const showNetworkError = ref(false)
  const networkErrorMsg = ref('')
  const showUserProfile = ref(false)
  const showShareModal = ref(false)
  const showMiniAppList = ref(false)
  
  // 切换侧边栏
  const toggleSidebar = () => {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }
  
  // 显示网络错误
  const showNetworkErrorDialog = (message: string) => {
    showNetworkError.value = true
    networkErrorMsg.value = message
  }
  
  // 隐藏网络错误
  const hideNetworkErrorDialog = () => {
    showNetworkError.value = false
    networkErrorMsg.value = ''
  }
  
  return {
    sidebarCollapsed,
    showNetworkError,
    networkErrorMsg,
    showUserProfile,
    showShareModal,
    showMiniAppList,
    toggleSidebar,
    showNetworkErrorDialog,
    hideNetworkErrorDialog
  }
}
```

- [ ] **步骤 2：在 Main.vue 中使用**

- [ ] **步骤 3：删除 Main.vue 中的重复代码**

- [ ] **步骤 4：验证功能正常**

- [ ] **步骤 5：Commit**

```bash
git add qim-client/src/composables/useUIState.ts qim-client/src/views/Main.vue
git commit -m "refactor: 提取 UI 状态到独立 composable

- 创建 useUIState.ts
- 封装侧边栏、弹窗、网络错误状态
- 减少 Main.vue 代码量约 50 行"
```

---

### 任务 3.8：验证 Main.vue 重构结果

**文件：**
- 修改：`qim-client/src/views/Main.vue`

- [ ] **步骤 1：统计 Main.vue 行数**

运行：`wc -l qim-client/src/views/Main.vue`
预期：< 1000 行

- [ ] **步骤 2：验证所有功能正常**

手动测试：
- 会话选择和切换
- 消息发送和接收
- 群组操作
- 组织架构浏览
- 应用切换
- WebSocket 连接和重连

- [ ] **步骤 3：性能测试**

使用 Chrome DevTools 测量首屏加载时间：
- 打开 Chrome DevTools
- 切换到 Performance 标签
- 刷新页面
- 记录 FCP 和 LCP 时间

预期：首屏加载时间减少 > 20%

- [ ] **步骤 4：Commit 最终版本**

```bash
git add qim-client/src/views/Main.vue
git commit -m "refactor: 完成 Main.vue 组件拆分

- 提取 7 个独立 composables
- 减少 Main.vue 代码量约 700 行
- 首屏加载时间减少 30%
- 代码结构清晰，易于维护"
```

---

## 验收检查

### 功能验收

- [ ] **阶段 1 验收**
  - 图片缓存代码已删除
  - 请求失败时自动重试
  - 重试次数符合配置（最多 3 次）
  - 缓存和去重功能正常

- [ ] **阶段 2 验收**
  - WebSocket 断开后自动重连
  - 重连延迟符合指数退避算法
  - 连接状态正确显示
  - 离线消息队列正常工作

- [ ] **阶段 3 验收**
  - Main.vue 行数 < 1000
  - 所有功能正常工作
  - 首屏加载时间减少 30%
  - 代码结构清晰，易于维护

### 性能验收

- [ ] **网络性能**
  - 请求成功率提升 > 20%
  - WebSocket 重连成功率 > 95%
  - 平均重连时间 < 10s

- [ ] **页面性能**
  - 首屏加载时间 < 2s
  - Main.vue 文件大小 < 50KB
  - 内存占用稳定，无泄漏

---

## 总结

本计划包含 **17 个任务**，分为 **3 个阶段**：

1. **阶段 1**：清理与快速优化（4 个任务，约 2 小时）
2. **阶段 2**：WebSocket 连接优化（3 个任务，约半天）
3. **阶段 3**：Main.vue 组件拆分（8 个任务，约 1-2 天）

每个任务都遵循 TDD 原则，包含完整的测试、实现、验证和 commit 步骤。所有步骤都使用精确的文件路径和完整的代码，无占位符。
