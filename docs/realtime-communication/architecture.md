# 实时通信架构设计

## 架构概览

```
┌─────────────────────────────────────────────────────────────┐
│                         业务层                               │
│  useScreenShare    useVideoCall    useAudioCall             │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                         会话层                               │
│                    useSession                               │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                         能力层                               │
│              useRealtimeCommunication                       │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                       基础设施层                             │
│  useConnection    useMediaStream    useSignaling            │
└─────────────────────────────────────────────────────────────┘
```

## 层次职责

### 1. 基础设施层

**职责：** 提供最基础的能力

#### useConnection

- 管理 RTCPeerConnection
- 提供 WebRTC 连接操作
- 管理连接状态

```typescript
export function useConnection() {
  const state = ref<ConnectionState>('disconnected')
  
  const createConnection = async (config: RTCConfig) => { }
  const addTrack = (track: MediaStreamTrack, stream: MediaStream) => { }
  const createOffer = async () => { }
  const createAnswer = async () => { }
  const setLocalDescription = async (description: RTCSessionDescriptionInit) => { }
  const setRemoteDescription = async (description: RTCSessionDescriptionInit) => { }
  const addIceCandidate = async (candidate: RTCIceCandidateInit) => { }
  const close = () => { }
  
  return {
    state,
    createConnection,
    addTrack,
    createOffer,
    createAnswer,
    setLocalDescription,
    setRemoteDescription,
    addIceCandidate,
    close
  }
}
```

#### useMediaStream

- 管理媒体流
- 提供媒体流操作
- 管理流状态

```typescript
export function useMediaStream(source: MediaStreamSourceType) {
  const stream = ref<MediaStream | null>(null)
  const state = ref<StreamState>('stopped')
  
  const start = async () => { }
  const stop = () => { }
  const toggleMute = (kind: 'audio' | 'video') => { }
  
  return {
    stream,
    state,
    start,
    stop,
    toggleMute
  }
}
```

#### useSignaling

- 管理信令消息
- 提供消息发送功能
- 统一消息格式

```typescript
export function useSignaling() {
  const sendMessage = (type: string, data: any) => { }
  const createSignalingMessage = (type: string, targetUserId: number, mediaType: MediaType, payload: any) => { }
  const sendOffer = (targetUserId: number, mediaType: MediaType, offer: RTCSessionDescriptionInit) => { }
  const sendAnswer = (targetUserId: number, mediaType: MediaType, answer: RTCSessionDescriptionInit) => { }
  const sendIceCandidate = (targetUserId: number, mediaType: MediaType, candidate: RTCIceCandidate) => { }
  
  return {
    sendMessage,
    createSignalingMessage,
    sendOffer,
    sendAnswer,
    sendIceCandidate
  }
}
```

### 2. 能力层

**职责：** 组合基础设施，提供实时通信能力

#### useRealtimeCommunication

- 组合连接、媒体流、信令
- 提供 WebRTC 完整流程
- 管理远程流

```typescript
export function useRealtimeCommunication(mediaType: MediaType) {
  const connection = useConnection()
  const mediaStream = useMediaStream(getSourceFromMediaType(mediaType))
  const signaling = useSignaling()
  
  const remoteStream = ref<MediaStream | null>(null)
  
  const initiate = async (targetUserId: number) => { }
  const receive = async (signal: RTCSessionDescriptionInit, fromUserId: number) => { }
  const handleAnswer = async (signal: RTCSessionDescriptionInit) => { }
  const handleIceCandidate = async (candidate: RTCIceCandidateInit) => { }
  const close = () => { }
  
  return {
    state: connection.state,
    localStream: mediaStream.stream,
    remoteStream,
    initiate,
    receive,
    handleAnswer,
    handleIceCandidate,
    close
  }
}
```

### 3. 会话层

**职责：** 管理会话生命周期

#### useSession

- 管理会话状态
- 管理参与者
- 提供会话操作

```typescript
export function useSession(type: SessionType) {
  const mediaType = getMediaTypeFromSessionType(type)
  const rtc = useRealtimeCommunication(mediaType)
  
  const sessionState = ref<SessionState>('idle')
  const participants = ref<Participant[]>([])
  
  const start = async (targetUserId: number) => { }
  const join = async (signal: RTCSessionDescriptionInit, fromUserId: number) => { }
  const end = () => { }
  
  return {
    sessionState,
    participants,
    localStream: rtc.localStream,
    remoteStream: rtc.remoteStream,
    start,
    join,
    end,
    handleAnswer: rtc.handleAnswer,
    handleIceCandidate: rtc.handleIceCandidate
  }
}
```

### 4. 业务层

**职责：** 实现具体业务逻辑

#### useScreenShare

- 屏幕共享特定逻辑
- 源选择
- 窗口管理

```typescript
export function useScreenShare() {
  const session = useSession('screen-share')
  
  const selectedSource = ref<DisplayMediaStreamOptions | null>(null)
  
  const selectSource = async () => { }
  
  return {
    ...session,
    selectedSource,
    selectSource
  }
}
```

#### useVideoCall

- 视频通话特定逻辑
- 摄像头切换
- 通话管理

```typescript
export function useVideoCall() {
  const session = useSession('video-call')
  
  const toggleCamera = () => { }
  
  return {
    ...session,
    toggleCamera
  }
}
```

## 数据流

### 发送方流程

```
1. 用户点击"开始共享"
   ↓
2. 业务层调用 session.start(targetUserId)
   ↓
3. 会话层调用 rtc.initiate(targetUserId)
   ↓
4. 能力层执行：
   - mediaStream.start() 获取媒体流
   - connection.createConnection() 创建连接
   - connection.addTrack() 添加轨道
   - connection.createOffer() 创建 offer
   - signaling.sendOffer() 发送 offer
   ↓
5. 等待 answer
   ↓
6. 能力层处理 answer：
   - connection.setRemoteDescription(answer)
   ↓
7. ICE 候选者交换
   ↓
8. 连接建立，媒体流传输
```

### 接收方流程

```
1. 收到 webrtc.offer 消息
   ↓
2. 业务层调用 session.join(signal, fromUserId)
   ↓
3. 会话层调用 rtc.receive(signal, fromUserId)
   ↓
4. 能力层执行：
   - connection.createConnection() 创建连接
   - connection.setRemoteDescription(signal) 设置远程描述
   - connection.createAnswer() 创建 answer
   - signaling.sendAnswer() 发送 answer
   ↓
5. ICE 候选者交换
   ↓
6. 连接建立，接收媒体流
```

## 状态管理

### 状态层次

```
业务状态（SessionState）
  - idle: 空闲
  - connecting: 连接中
  - active: 活跃
  - ended: 已结束

连接状态（ConnectionState）
  - disconnected: 未连接
  - connecting: 连接中
  - connected: 已连接

流状态（StreamState）
  - stopped: 已停止
  - starting: 启动中
  - active: 活跃
```

### 状态转换规则

```
SessionState 转换：
  idle -> connecting: 调用 start() 或 join()
  connecting -> active: WebRTC 连接建立成功
  connecting -> idle: 连接失败
  active -> ended: 调用 end()
  ended -> idle: 清理完成

ConnectionState 转换：
  disconnected -> connecting: 创建 RTCPeerConnection
  connecting -> connected: ICE 连接成功
  connecting -> disconnected: 连接失败
  connected -> disconnected: 连接关闭

StreamState 转换：
  stopped -> starting: 调用 start()
  starting -> active: 获取流成功
  starting -> stopped: 获取流失败
  active -> stopped: 调用 stop()
```

## 错误处理

### 错误类型

1. **媒体错误**
   - 用户拒绝授权
   - 设备不可用
   - 格式不支持

2. **连接错误**
   - ICE 连接失败
   - 信令错误
   - 网络错误

3. **信令错误**
   - 消息格式错误
   - 状态不匹配
   - 超时

### 错误处理策略

```typescript
try {
  await session.start(targetUserId)
} catch (error) {
  if (error.name === 'NotAllowedError') {
    // 用户拒绝授权
  } else if (error.name === 'NotFoundError') {
    // 设备不可用
  } else if (error.name === 'InvalidStateError') {
    // 状态错误
  } else {
    // 其他错误
  }
}
```

## 扩展性

### 添加新的会话类型

1. 在 `MediaType` 中添加新类型
2. 在 `SessionType` 中添加新类型
3. 创建新的业务层 composable
4. 无需修改基础设施层和会话层

### 添加新的连接协议

1. 实现 `ConnectionProtocol` 接口
2. 在 `useConnection` 中添加新协议支持
3. 无需修改业务层

### 添加新的媒体源

1. 在 `MediaStreamSourceType` 中添加新类型
2. 在 `useMediaStream` 中添加新源支持
3. 无需修改业务层
