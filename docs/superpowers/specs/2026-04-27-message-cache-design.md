# 消息缓存设计方案

## 目标

实现消息本地缓存，支持：
1. **离线支持** - 网络断开时仍能查看历史消息
2. **性能优化** - 首次加载立即显示，无需等待网络请求

## 设计原则

- **本地优先** - 优先显示本地缓存，后台静默同步服务器
- **增量同步** - 只同步增量消息，减少数据传输
- **服务器为准** - 合并时以服务器数据为准

## 数据结构

### IndexedDB Schema

```
数据库: QIMChatDB (或 QIMChatDB_user_{userId})

表: messages
- id: number (主键)
- conversation_id: string (索引)
- content: string
- sender_id: number
- sender_name: string
- sender_avatar: string
- type: string
- timestamp: number (索引，用于排序)
- created_at: string

表: sync_states
- conversation_id: string (主键)
- latest_id: number (本地最新消息 ID)
- latest_time: number (本地最新消息时间戳)
- last_sync_time: number (最后同步时间)
```

## 数据流

### 首次加载（冷启动）

```
loadMessages(id):
  1. 从 IndexedDB 加载本地缓存 → 立即显示
  2. 获取 SyncState(id)
     - 如果有 latest_id，用 after_id=latest_id 调用 /messages
     - 如果没有 latest_id，调用 /messages (获取全部)
  3. 合并去重 → 更新 UI
  4. 保存消息到 IndexedDB → 更新 SyncState
```

### 后续加载（切换会话）

```
loadMessages(id):
  1. 从 IndexedDB 加载本地缓存 → 立即显示
  2. 调用 /messages?after_id={latest_id}
  3. 合并去重 → 更新 UI
  4. 保存消息到 IndexedDB → 更新 SyncState
```

### WebSocket 新消息

```
收到 new_message:
  1. 写入 IndexedDB
  2. 更新 SyncState
  3. 如果是当前会话，插入消息列表
```

### 消息发送

```
发送消息成功:
  1. 写入 IndexedDB
  2. 更新 SyncState
```

## 合并策略

```typescript
function mergeMessages(local: Message[], server: Message[]): Message[] {
  const serverIds = new Set(server.map(m => m.id))
  const localOnly = local.filter(m => !serverIds.has(m.id))
  const merged = [...server, ...localOnly]
  return merged.sort((a, b) => a.timestamp - b.timestamp)
}
```

## 后端 API（无需改动）

现有 `GET /api/v1/conversations/:id/messages?after_id=X` 已支持增量查询：
- `after_id > 0` → 返回 id > after_id 的消息
- `after_id = 0` 或不传 → 返回全量消息

## 实现任务

### Phase 1: 基础设施
1. 扩展 storage.ts - 新增 sync_states 表
2. 新增 getSyncState / saveSyncState 函数

### Phase 2: loadMessages 重构
1. 本地优先加载
2. 后台增量同步
3. 合并去重
4. 更新缓存

### Phase 3: 实时同步
1. WebSocket 收到消息时更新本地缓存
2. 发送消息时写入本地缓存

## 风险与注意事项

1. **循环引用** - processMessage 需避免序列化 sender.user
2. **时间戳精度** - 使用服务器返回的 created_at 转换
3. **离线发送** - 离线时消息暂存，联网后重试
4. **缓存清理** - 定期清理过期缓存，避免占用过大

## 预期效果

- 消息加载: 立即显示（< 100ms）
- 离线可用: 能查看历史消息
- 数据一致: 与服务器保持同步
