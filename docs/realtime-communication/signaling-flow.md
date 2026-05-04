# 实时通信信令流程

## 概述

本文档描述了实时通信（屏幕共享、视频通话、语音通话）的信令流程。

## 信令流程图

### 屏幕共享流程

```mermaid
sequenceDiagram
    participant A as 发送方
    participant S as 服务器
    participant B as 接收方
    
    Note over A,B: 1. 请求阶段
    A->>S: screen-share.request
    S->>B: screen-share.request
    B->>S: screen-share.response (accept/reject)
    S->>A: screen-share.response
    
    Note over A,B: 2. WebRTC 连接建立
    A->>S: webrtc.offer (media_type: screen)
    S->>B: webrtc.offer (media_type: screen)
    B->>S: webrtc.answer (media_type: screen)
    S->>A: webrtc.answer (media_type: screen)
    
    Note over A,B: 3. ICE 候选者交换
    A->>S: webrtc.ice-candidate
    S->>B: webrtc.ice-candidate
    B->>S: webrtc.ice-candidate
    S->>A: webrtc.ice-candidate
    
    Note over A,B: 4. 媒体流传输
    A-->>B: 视频流传输
    
    Note over A,B: 5. 结束阶段
    A->>S: screen-share.stop
    S->>B: screen-share.stop
```

### 视频通话流程

```mermaid
sequenceDiagram
    participant A as 发起方
    participant S as 服务器
    participant B as 接收方
    
    Note over A,B: 1. 呼叫阶段
    A->>S: call.start (media_type: video)
    S->>B: call.start
    B->>S: call.answer (accept/reject)
    S->>A: call.answer
    
    Note over A,B: 2. WebRTC 连接建立
    A->>S: webrtc.offer (media_type: video)
    S->>B: webrtc.offer (media_type: video)
    B->>S: webrtc.answer (media_type: video)
    S->>A: webrtc.answer (media_type: video)
    
    Note over A,B: 3. ICE 候选者交换
    A->>S: webrtc.ice-candidate
    S->>B: webrtc.ice-candidate
    B->>S: webrtc.ice-candidate
    S->>A: webrtc.ice-candidate
    
    Note over A,B: 4. 媒体流传输
    A-->>B: 视频流传输
    B-->>A: 视频流传输
    
    Note over A,B: 5. 结束阶段
    A->>S: call.end
    S->>B: call.end
```

## 消息格式

### 业务层消息

#### screen-share.request

```typescript
{
  type: 'screen-share.request',
  data: {
    target_user_id: number,
    from_user_id: number,
    conversation_id: number
  }
}
```

#### screen-share.response

```typescript
{
  type: 'screen-share.response',
  data: {
    target_user_id: number,
    from_user_id: number,
    accepted: boolean
  }
}
```

#### screen-share.stop

```typescript
{
  type: 'screen-share.stop',
  data: {
    from_user_id: number
  }
}
```

#### call.start

```typescript
{
  type: 'call.start',
  data: {
    target_user_id: number,
    from_user_id: number,
    media_type: 'video' | 'audio'
  }
}
```

#### call.answer

```typescript
{
  type: 'call.answer',
  data: {
    target_user_id: number,
    from_user_id: number,
    accepted: boolean
  }
}
```

#### call.end

```typescript
{
  type: 'call.end',
  data: {
    from_user_id: number
  }
}
```

### 信令层消息

#### webrtc.offer

```typescript
{
  type: 'webrtc.offer',
  data: {
    target_user_id: number,
    from_user_id: number,
    media_type: 'screen' | 'video' | 'audio',
    signal: RTCSessionDescriptionInit
  }
}
```

#### webrtc.answer

```typescript
{
  type: 'webrtc.answer',
  data: {
    target_user_id: number,
    from_user_id: number,
    media_type: 'screen' | 'video' | 'audio',
    signal: RTCSessionDescriptionInit
  }
}
```

#### webrtc.ice-candidate

```typescript
{
  type: 'webrtc.ice-candidate',
  data: {
    target_user_id: number,
    from_user_id: number,
    media_type: 'screen' | 'video' | 'audio',
    candidate: RTCIceCandidateInit
  }
}
```

## 关键要点

1. **统一的媒体类型标识**
   - 使用 `media_type` 字段统一标识
   - 取值：`'screen'` | `'video'` | `'audio'`
   - 替代原来的 `share_type` 和 `call_type`

2. **消息路由**
   - 业务层消息：根据 `type` 前缀路由（`screen-share.*` 或 `call.*`）
   - 信令层消息：根据 `media_type` 字段路由

3. **服务器转发**
   - 服务器必须转发所有字段，包括 `media_type`
   - 不修改消息内容，只添加 `from_user_id`

4. **状态管理**
   - 业务层管理会话状态（idle, connecting, active, ended）
   - 信令层管理连接状态（disconnected, connecting, connected）
