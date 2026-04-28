# 消息已读回执功能 - 架构设计文档

> 创建日期: 2026-04-27
> 状态: 已批准，进入实现阶段

## 1. 需求概述

### 1.1 用户故事
1. **发送者**看到消息已读状态（已发送/已送达/已读）
2. **群聊发送者**查看已读人员列表
3. **接收者**阅读时自动标记已读，不打断体验

### 1.2 技术指标
- 已读状态同步延迟 < 2 秒（P95）
- 已读列表加载 < 500ms（P95）
- 合并推送 + 节流更新
- 分页加载 + 虚拟滚动

### 1.3 群聊规模
中型企业群：50-500 人

---

## 2. 系统上下文图

```
┌─────────────────────────────────────────────────────────┐
│                     Electron Client                      │
│  ┌──────────────────────────────────────────────────┐   │
│  │                  Vue 3 UI Layer                   │   │
│  │  ┌──────────┐ ┌──────────────┐ ┌──────────────┐  │   │
│  │  │MessageItem││MessageStatus ││ReadUsersModal │  │   │
│  │  └─────┬────┘ └──────┬───────┘ └──────┬───────┘  │   │
│  │        │              │               │           │   │
│  │  ┌─────▼──────────────▼───────────────▼──────┐   │   │
│  │  │         ReadReceiptManager                 │   │   │
│  │  │  - Throttle & Merge Updates                │   │   │
│  │  │  - IntersectionObserver (auto read)        │   │   │
│  │  │  - Pagination & Virtual Scroll             │   │   │
│  │  └──────────────────┬────────────────────────┘   │   │
│  │                     │                             │   │
│  │  ┌──────────────────▼────────────────────────┐   │   │
│  │  │            WebSocket Layer                 │   │   │
│  │  │  - Real-time sync channel                  │   │   │
│  │  │  - Message type: read_receipt              │   │   │
│  │  └──────────────────┬────────────────────────┘   │   │
│  └─────────────────────┼────────────────────────────┘   │
│                        │ HTTP / WSS                      │
└────────────────────────┼────────────────────────────────┘
                         │
              ┌──────────▼──────────┐
              │   Go Backend (Gin)   │
              │  ┌────────────────┐  │
              │  │  REST API      │  │
              │  │  /read         │  │
              │  │  /read-users   │  │
              │  └───────┬────────┘  │
              │          │            │
              │  ┌───────▼────────┐  │
              │  │  WebSocket Hub │  │
              │  │  broadcast     │  │
              │  │  read_receipt  │  │
              │  └───────┬────────┘  │
              │          │            │
              │  ┌───────▼────────┐  │
              │  │  Service Layer │  │
              │  │  ReadReceipt   │  │
              │  │  Service       │  │
              │  └───────┬────────┘  │
              └──────────┼───────────┘
                         │
              ┌──────────▼──────────┐
              │   SQLite Database   │
              │  ┌────────────────┐ │
              │  │ message_reads  │ │
              │  │ (receipt table)│ │
              │  └────────────────┘ │
              └─────────────────────┘
```

---

## 3. 前端组件架构

```
qim-client/src/
├── components/
│   └── chat/
│       ├── MessageStatus.vue          [已有] 消息状态指示器
│       ├── ReadUsersModal.vue         [已有] 已读用户弹窗（需重构）
│       └── ReadReceiptIndicator.vue   [新建] 已读回执指示器组件
│
├── composables/
│   ├── useReadReceipt.ts              [新建] 已读回执核心逻辑
│   │   ├── throttle 管理 (300ms)
│   │   ├── pending 队列（合并多会话）
│   │   ├── markAsRead() 方法
│   │   └── syncReadStatus() 实时同步
│   │
│   └── useReadUsersModal.ts           [新建] 已读列表管理
│       ├── pagination 状态
│       ├── fetchReadUsers() API 调用
│       └── 虚拟滚动配置
│
└── components/
    └── read-users/                    [新建] 已读用户子组件目录
        ├── ReadUsersModal.vue         [重构] 弹窗容器
        ├── ReadUsersList.vue          [新建] 列表容器
        └── ReadUserItem.vue           [新建] 单用户行
```

### 3.1 数据流图

```
[接收者阅读消息]
       │
       ▼
┌─────────────────┐
│ IntersectionObserver │ ◄── 消息进入可视区域
│ 自动触发已读标记   │
└────────┬────────┘
         │ throttle 300ms 合并
         ▼
┌─────────────────┐     POST /api/v1/conversations/{id}/read
│ useReadReceipt  │───────────────────────────────────────────┐
│ pending queue   │                                            │
└─────────────────┘                                            ▼
                                                        ┌──────────────┐
                                                        │ Go Backend   │
                                                        │ 1. 查询成员  │
                                                        │ 2. 批量插入  │
                                                        │ 3. 广播 WS   │
                                                        └──────┬───────┘
                                                               │ WS: type=read_receipt
                                ┌──────────────────────┐       ▼
                                │  发送者收到已读更新   │◄──────┘
                                │  更新 MessageStatus   │
                                └──────────────────────┘

[发送者点击查看已读列表]
       │
       ▼
┌──────────────────────┐
│ ReadUsersModal.vue   │
│  ├─ ReadUsersList    │ ◄── 虚拟滚动 (20条/页)
│  │   └─ ReadUserItem │
│  └─ 分页加载更多     │
└──────────────────────┘
       │ GET /api/v1/messages/{msgId}/read-users?page=1&page_size=20
       ▼
┌──────────────────────┐
│ Go Backend           │
│ JOIN message_reads   │
│ + conversation_members│
│ 分页返回             │
└──────────────────────┘
```

---

## 4. 技术选型

### 4.1 实时通信方案
- **已选**: WebSocket（现有基础设施已就绪）
- **备选**: SSE（不满足双向需求）、HTTP Long Polling（延迟不达标）

### 4.2 已读标记触发
- **已选**: IntersectionObserver（精确、自动、不打断体验）
- **备选**: 停留时间（延迟高）、手动点击（打断体验）

### 4.3 节流策略
- **已选**: 300ms 节流 + 按会话批量标记
- **理由**: 平衡延迟和请求数，确保 < 2 秒同步要求

### 4.4 已读列表渲染
- **已选**: 分页加载 (20条/页) + 虚拟滚动
- **理由**: 500 人群场景下内存恒定，仅渲染可视区域

---

## 5. 数据库设计

### 5.1 message_reads 表

```sql
CREATE TABLE IF NOT EXISTS message_reads (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id  INTEGER NOT NULL,
    user_id     INTEGER NOT NULL,
    read_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(message_id, user_id),
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_message_reads_message_id ON message_reads(message_id);
CREATE INDEX idx_message_reads_message_read_at ON message_reads(message_id, read_at DESC);
CREATE INDEX idx_message_reads_user_id ON message_reads(user_id);
CREATE INDEX idx_message_reads_count ON message_reads(message_id, user_id);
```

### 5.2 现有表变更

```sql
ALTER TABLE messages ADD COLUMN read_count INTEGER DEFAULT 0;
ALTER TABLE conversations ADD COLUMN last_read_message_id INTEGER DEFAULT 0;
CREATE INDEX idx_messages_conversation_read ON messages(conversation_id, read_count);
```

---

## 6. API 接口设计

### 6.1 REST API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/conversations/{conversation_id}/read` | 批量标记消息已读 |
| GET | `/api/v1/messages/{message_id}/read-users?page=1&page_size=20` | 获取已读用户列表 |
| POST | `/api/v1/messages/read-status` | 批量获取消息已读状态摘要 |

### 6.2 WebSocket 消息类型

| 类型 | 方向 | 说明 |
|------|------|------|
| `mark_read` | C → S | 客户端请求标记已读 |
| `mark_read_ack` | S → C | 服务端确认已读标记 |
| `read_receipt` | S → C | 服务端推送已读状态给发送者 |
| `read_receipt_batch` | S → C | 批量已读状态推送（节流合并） |

### 6.3 错误码

| 错误码 | 说明 | 处理建议 |
|--------|------|---------|
| 0 | 成功 | - |
| 400 | 参数错误 | 检查请求参数 |
| 401 | 未授权 | 重新登录 |
| 403 | 无权限 | 检查会话成员身份 |
| 404 | 资源不存在 | 检查 ID 是否正确 |
| 500 | 服务器内部错误 | 联系管理员 |

---

## 7. 性能目标

| 指标 | 目标 | 策略 |
|------|------|------|
| 已读状态同步延迟 | < 2 秒 (P95) | WebSocket 推送 + 300ms 节流 |
| 已读列表加载 | < 500ms (P95) | SQLite 索引 + 分页 + 虚拟滚动 |
| 合并推送 | 请求数降低 80%+ | 300ms 节流 + 按会话批量标记 |
| 虚拟滚动 | 内存恒定 | 仅渲染可视区域 ~15 个 DOM 节点 |

---

## 8. 扩展性路径

| 阶段 | 架构 | 并发支持 |
|------|------|---------|
| 当前 | SQLite + 单实例 | 1000 并发 |
| V2 | PostgreSQL + Redis Pub/Sub | 10000+ 并发 |
| V3 | 微服务 + Kafka | 100000+ 并发 |
