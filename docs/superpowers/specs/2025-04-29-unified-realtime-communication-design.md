# 统一实时通信架构设计

## 1. 概述

### 1.1 背景

当前 QIM 的实时通信功能（屏幕共享、视频通话、语音通话）存在以下问题：

1. **架构分离**：每种类型独立实现，代码重复
2. **状态不持久**：刷新页面后状态丢失
3. **跨会话不可见**：A 和 B 共享时，C 看不到
4. **不支持多人**：屏幕共享只能一对一

### 1.2 目标

1. 统一实时通信架构，支持屏幕共享、视频通话、语音通话
2. 状态持久化到数据库，支持断线重连
3. 跨会话可见，C 可以看到 A 正在共享
4. 屏幕共享支持多人加入（Mesh 模式）

### 1.3 范围

本次实现范围：
- ✅ 统一架构基础
- ✅ 屏幕共享重构（多人、审批、持久化）
- ⏸️ 视频通话重构（后续迭代）
- ⏸️ 语音通话重构（后续迭代）

---

## 2. 架构设计

### 2.1 领域模型

```
┌─────────────────────────────────────────────────────────────┐
│                   RealtimeSession                           │
│                   (统一实时会话)                              │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  id: string (UUID)                                          │
│  type: 'screen_share' | 'voice_call' | 'video_call'        │
│  status: 'pending' | 'active' | 'paused' | 'ended'         │
│  initiator_id: number                                       │
│  conversation_id: number                                    │
│  started_at: Date                                           │
│  ended_at: Date | null                                      │
│  metadata: JSON (扩展字段)                                   │
│                                                             │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                      Participant                            │
│                      (参与者)                                │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  id: string (UUID)                                          │
│  session_id: string                                         │
│  user_id: number                                            │
│  role: 'initiator' | 'viewer'                               │
│  status: 'pending' | 'approved' | 'rejected' | 'joined'     │
│  requested_at: Date                                         │
│  approved_at: Date | null                                   │
│  joined_at: Date | null                                     │
│  left_at: Date | null                                       │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 状态机

#### 会话状态机

```
┌─────────┐
│ pending │ ← 创建会话
└────┬────┘
     │ 第一个参与者加入
     ▼
┌─────────┐
│ active  │←──────────────────┐
└────┬────┘                    │
     │                         │
     │ 暂停        恢复        │
     ▼             │           │
┌─────────┐        │      ┌────────┐
│ paused  │────────┘      │ ended  │
└─────────┘               └────────┘
```

#### 参与者状态机

```
┌─────────┐    申请加入    ┌──────────┐    同意    ┌─────────┐
│  idle   │──────────────→│ pending  │───────────→│ joined  │
└─────────┘               └──────────┘            └─────────┘
                                │                      │
                                │ 拒绝                 │ 离开
                                ▼                      ▼
                          ┌──────────┐          ┌─────────┐
                          │ rejected │          │  left   │
                          └──────────┘          └─────────┘
```

### 2.3 事件驱动架构

```typescript
type RealtimeSessionEvent =
  | { type: 'session_created'; payload: { session: RealtimeSession } }
  | { type: 'join_requested'; payload: { sessionId, userId } }
  | { type: 'join_approved'; payload: { sessionId, userId } }
  | { type: 'join_rejected'; payload: { sessionId, userId } }
  | { type: 'participant_joined'; payload: { sessionId, userId } }
  | { type: 'participant_left'; payload: { sessionId, userId } }
  | { type: 'session_ended'; payload: { sessionId } };
```

---

## 3. 数据库设计

### 3.1 表结构

```sql
-- 实时会话表
CREATE TABLE realtime_sessions (
  id VARCHAR(36) PRIMARY KEY,
  type VARCHAR(20) NOT NULL,
  initiator_id INT NOT NULL,
  conversation_id INT NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  started_at TIMESTAMP,
  ended_at TIMESTAMP,
  metadata TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  
  FOREIGN KEY (initiator_id) REFERENCES users(id),
  FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

CREATE INDEX idx_realtime_sessions_initiator ON realtime_sessions(initiator_id);
CREATE INDEX idx_realtime_sessions_conversation ON realtime_sessions(conversation_id);
CREATE INDEX idx_realtime_sessions_status ON realtime_sessions(status);

-- 参与者表
CREATE TABLE realtime_participants (
  id VARCHAR(36) PRIMARY KEY,
  session_id VARCHAR(36) NOT NULL,
  user_id INT NOT NULL,
  role VARCHAR(20) NOT NULL DEFAULT 'viewer',
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  requested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  approved_at TIMESTAMP,
  joined_at TIMESTAMP,
  left_at TIMESTAMP,
  
  FOREIGN KEY (session_id) REFERENCES realtime_sessions(id),
  FOREIGN KEY (user_id) REFERENCES users(id),
  UNIQUE (session_id, user_id)
);

CREATE INDEX idx_realtime_participants_session ON realtime_participants(session_id);
CREATE INDEX idx_realtime_participants_user ON realtime_participants(user_id);
```

---

## 4. API 设计

### 4.1 REST API

```
POST   /api/realtime/sessions                  创建会话
GET    /api/realtime/sessions/:id              获取会话详情
GET    /api/realtime/sessions/active           获取活跃会话列表
PATCH  /api/realtime/sessions/:id              更新会话状态

POST   /api/realtime/sessions/:id/participants 申请加入
PATCH  /api/realtime/sessions/:id/participants/:userId 审批/离开
```

### 4.2 WebSocket 事件

```
// 会话管理
'realtime:session:create'     创建会话
'realtime:session:end'        结束会话

// 参与者管理
'realtime:join:request'       申请加入
'realtime:join:approve'       批准加入
'realtime:join:reject'        拒绝加入
'realtime:leave'              离开会话

// WebRTC 信令
'realtime:webrtc:offer'       发送 offer
'realtime:webrtc:answer'      发送 answer
'realtime:webrtc:ice'         发送 ICE candidate
```

---

## 5. 前端架构

### 5.1 状态管理 (Pinia Store)

```typescript
// stores/realtime.ts
export const useRealtimeStore = defineStore('realtime', {
  state: () => ({
    activeSessions: [] as RealtimeSession[],
    mySession: null as RealtimeSession | null,
    pendingRequests: [] as JoinRequest[],
  }),
  
  getters: {
    isSharing(): boolean {
      return this.mySession?.type === 'screen_share' && 
             this.mySession?.status === 'active';
    },
    
    getActiveSessionByUser(): (userId: number) => RealtimeSession | undefined {
      return (userId) => this.activeSessions.find(s => s.initiatorId === userId);
    }
  },
  
  actions: {
    async createSession(type: SessionType, conversationId: number) { ... },
    async requestJoin(sessionId: string) { ... },
    async approveJoin(sessionId: string, userId: number) { ... },
    async leaveSession(sessionId: string) { ... },
    async endSession(sessionId: string) { ... },
  }
});
```

### 5.2 组件结构

```
components/
├── realtime/
│   ├── RealtimeSessionCard.vue    # 统一会话卡片
│   ├── ScreenShareCard.vue        # 屏幕共享卡片
│   ├── JoinRequestModal.vue       # 加入请求弹窗
│   └── ViewerList.vue             # 观看者列表
├── chat/
│   └── ChatWindow.vue             # 集成实时通信
```

---

## 6. 屏幕共享流程

### 6.1 发起共享

```
A 点击 🖥️ 按钮
    │
    ▼
前端调用 realtimeStore.createSession('screen_share', conversationId)
    │
    ▼
服务端创建 RealtimeSession 记录
    │
    ▼
服务端在会话中插入系统消息卡片
    │
    ▼
服务端广播 session_created 事件给会话成员
    │
    ▼
A 开始获取屏幕流
    │
    ▼
A 进入"共享中"状态
```

### 6.2 申请加入

```
B 看到系统消息卡片，点击"加入观看"
    │
    ▼
前端调用 realtimeStore.requestJoin(sessionId)
    │
    ▼
服务端创建 Participant 记录（status: 'pending'）
    │
    ▼
服务端发送 join_requested 事件给 A
    │
    ▼
A 收到通知弹窗："B 想加入观看你的屏幕共享"
    │
    ▼
A 点击"同意"
    │
    ▼
前端调用 realtimeStore.approveJoin(sessionId, B.id)
    │
    ▼
服务端更新 Participant 状态为 'approved'
    │
    ▼
服务端发送 join_approved 事件给 B
    │
    ▼
A 为 B 创建 WebRTC 连接并发送 offer
    │
    ▼
B 收到 offer，发送 answer
    │
    ▼
WebRTC 连接建立，B 开始观看
```

### 6.3 跨会话可见

```
C 打开和 A 的一对一会话（另一个会话）
    │
    ▼
前端查询 A 是否有活跃的共享会话
    │
    ▼
API: GET /api/realtime/sessions/active?initiatorId=A
    │
    ▼
返回 A 的活跃共享会话
    │
    ▼
C 在聊天窗口顶部看到横幅：
    "A 正在共享屏幕    [加入观看]"
    │
    ▼
C 点击"加入观看" → 跳转到发起共享的会话
```

---

## 7. WebRTC 架构

### 7.1 Mesh 模式（一对多）

```
        A (发起者)
       /   |   \
      /    |    \
     B     C     D
   (独立) (独立) (独立)
```

- A 为每个观看者创建独立的 PeerConnection
- 不需要额外的服务器
- 适合 2-10 人

### 7.2 连接管理

```typescript
class ConnectionManager {
  private connections: Map<string, RTCPeerConnection> = new Map();
  
  async createConnection(sessionId: string, viewerId: number): Promise<void> {
    const pc = new RTCPeerConnection(config);
    
    // 添加本地流
    const stream = this.getLocalStream(sessionId);
    stream.getTracks().forEach(track => pc.addTrack(track, stream));
    
    // 设置事件处理
    pc.onicecandidate = (event) => {
      this.sendIceCandidate(sessionId, viewerId, event.candidate);
    };
    
    // 创建 offer
    const offer = await pc.createOffer();
    await pc.setLocalDescription(offer);
    
    // 发送 offer
    this.sendOffer(sessionId, viewerId, offer);
    
    // 保存连接
    this.connections.set(`${sessionId}:${viewerId}`, pc);
  }
  
  closeConnection(sessionId: string, viewerId: number): void {
    const key = `${sessionId}:${viewerId}`;
    const pc = this.connections.get(key);
    if (pc) {
      pc.close();
      this.connections.delete(key);
    }
  }
}
```

---

## 8. 向后兼容

### 8.1 保留现有消息类型

本次重构允许破坏性修改，但为了平滑过渡，建议：

1. 保留现有消息类型（`screen-share-request`、`call_invite` 等）
2. 新增统一消息类型（`realtime:*`）
3. 前端同时监听两种消息类型
4. 后续版本移除旧消息类型

### 8.2 数据迁移

无需迁移，新表为空。

---

## 9. 测试计划

### 9.1 单元测试

- [ ] RealtimeSession 状态机测试
- [ ] Participant 状态机测试
- [ ] ConnectionManager 连接管理测试

### 9.2 集成测试

- [ ] 发起共享 → 申请加入 → 审批 → 观看
- [ ] 多人同时加入
- [ ] 跨会话可见
- [ ] 断线重连
- [ ] 发起者离线 → 会话自动结束

### 9.3 E2E 测试

- [ ] 完整用户流程测试

---

## 10. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| WebRTC 连接失败 | 用户无法观看 | 重试机制 + ICE 服务器 fallback |
| 数据库性能 | 大量会话查询慢 | 索引优化 + 缓存活跃会话 |
| 并发冲突 | 多人同时申请加入 | 数据库事务 + 乐观锁 |
| 内存泄漏 | 长时间运行后崩溃 | 定期清理已结束会话 |

---

## 11. 时间估算

| 阶段 | 任务 | 预估时间 |
|------|------|---------|
| 1 | 数据库模型 + 迁移 | 0.5 天 |
| 2 | 服务端 API + WebSocket | 1 天 |
| 3 | 前端 Store + 组件 | 1.5 天 |
| 4 | WebRTC 连接管理 | 1 天 |
| 5 | 集成测试 | 0.5 天 |
| 6 | 文档 + Code Review | 0.5 天 |
| **总计** | | **5 天** |

---

## 12. 附录

### 12.1 名词解释

- **RealtimeSession**: 统一的实时会话抽象
- **Participant**: 参与者，包括发起者和观看者
- **Mesh**: WebRTC 架构模式，每个参与者独立连接

### 12.2 参考资料

- [WebRTC API - MDN](https://developer.mozilla.org/en-US/docs/Web/API/WebRTC_API)
- [Mesh vs SFU vs MCU](https://webrtcglossary.com/mesh/)
