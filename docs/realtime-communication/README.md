# 实时通信重构 - 使用指南

## 概述

本次重构将实时通信功能（屏幕共享、视频通话、语音通话）进行了分层抽象，提高了代码的可维护性和可扩展性。

## 架构层次

```
业务层：useScreenShareNew, useVideoCallNew
  ↓
会话层：useSession
  ↓
能力层：useRealtimeCommunication
  ↓
基础设施层：useConnection, useMediaStream, useSignaling
```

## 快速开始

### 屏幕共享

```typescript
import { useScreenShareNew } from '@/composables/useScreenShareNew'

const screenShare = useScreenShareNew()

// 发送方：开始共享
await screenShare.startSharing(targetUserId, conversationId)

// 接收方：接受共享
await screenShare.acceptShare(signal, fromUserId)

// 接收方：拒绝共享
screenShare.rejectShare(fromUserId)

// 停止共享
screenShare.stopSharing()

// 暂停/恢复
screenShare.togglePause()

// 访问状态
console.log(screenShare.sessionState.value)  // 'idle' | 'connecting' | 'active' | 'ended'
console.log(screenShare.localStream.value)   // 本地媒体流
console.log(screenShare.remoteStream.value)  // 远程媒体流
console.log(screenShare.isPaused.value)      // 是否暂停
```

### 视频通话

```typescript
import { useVideoCallNew } from '@/composables/useVideoCallNew'

const videoCall = useVideoCallNew()

// 发起方：开始通话
await videoCall.startCall(targetUserId)

// 接收方：接受通话
await videoCall.acceptCall(signal, fromUserId)

// 接收方：拒绝通话
videoCall.rejectCall(fromUserId)

// 结束通话
videoCall.endCall()

// 切换摄像头/麦克风
videoCall.toggleCamera()
videoCall.toggleMicrophone()

// 访问状态
console.log(videoCall.sessionState.value)     // 'idle' | 'connecting' | 'active' | 'ended'
console.log(videoCall.localStream.value)      // 本地媒体流
console.log(videoCall.remoteStream.value)     // 远程媒体流
console.log(videoCall.isCameraEnabled.value)  // 摄像头是否启用
console.log(videoCall.isMicrophoneEnabled.value) // 麦克风是否启用
```

## 迁移指南

### 从旧代码迁移

#### 1. 屏幕共享

**旧代码：**
```typescript
import { useScreenShare } from '@/composables/useScreenShare'

const {
  isSharing,
  isInitiator,
  isViewer,
  startShare,
  stopShare,
  handleOffer,
  handleAnswer,
  handleIceCandidate
} = useScreenShare()
```

**新代码：**
```typescript
import { useScreenShareNew } from '@/composables/useScreenShareNew'

const screenShare = useScreenShareNew()

// 状态
const isSharing = computed(() => screenShare.sessionState.value === 'active')
const isInitiator = computed(() => screenShare.participants.value.some(p => p.role === 'receiver'))
const isViewer = computed(() => screenShare.participants.value.some(p => p.role === 'initiator'))

// 操作
await screenShare.startSharing(targetUserId, conversationId)
screenShare.stopSharing()
await screenShare.acceptShare(signal, fromUserId)
screenShare.rejectShare(fromUserId)
await screenShare.handleAnswer(signal)
await screenShare.handleIceCandidate(candidate)
```

#### 2. 视频通话

**旧代码：**
```typescript
import { useVideoCall } from '@/composables/useVideoCall'

const {
  isCalling,
  isInCall,
  startCall,
  endCall,
  handleOffer,
  handleAnswer,
  handleIceCandidate
} = useVideoCall()
```

**新代码：**
```typescript
import { useVideoCallNew } from '@/composables/useVideoCallNew'

const videoCall = useVideoCallNew()

// 状态
const isCalling = computed(() => videoCall.sessionState.value === 'connecting')
const isInCall = computed(() => videoCall.sessionState.value === 'active')

// 操作
await videoCall.startCall(targetUserId)
videoCall.endCall()
await videoCall.acceptCall(signal, fromUserId)
videoCall.rejectCall(fromUserId)
await videoCall.handleAnswer(signal)
await videoCall.handleIceCandidate(candidate)
```

### 消息格式变化

#### 旧格式：
```typescript
{
  type: 'webrtc_offer',
  data: {
    target_user_id: 123,
    signal: {...},
    share_type: 'screen'  // 或 call_type: 'video'
  }
}
```

#### 新格式：
```typescript
{
  type: 'webrtc.offer',
  data: {
    target_user_id: 123,
    media_type: 'screen',  // 统一使用 media_type
    signal: {...}
  }
}
```

## API 参考

### useScreenShareNew

**状态：**
- `sessionState`: 会话状态
- `participants`: 参与者列表
- `localStream`: 本地媒体流
- `remoteStream`: 远程媒体流
- `selectedSource`: 选中的共享源
- `isPaused`: 是否暂停

**方法：**
- `selectSource()`: 选择共享源
- `startSharing(targetUserId, conversationId)`: 开始共享
- `acceptShare(signal, fromUserId)`: 接受共享
- `rejectShare(fromUserId)`: 拒绝共享
- `stopSharing()`: 停止共享
- `pause()`: 暂停共享
- `resume()`: 恢复共享
- `togglePause()`: 切换暂停状态
- `handleAnswer(signal)`: 处理 answer
- `handleIceCandidate(candidate)`: 处理 ICE candidate

### useVideoCallNew

**状态：**
- `sessionState`: 会话状态
- `participants`: 参与者列表
- `localStream`: 本地媒体流
- `remoteStream`: 远程媒体流
- `isCameraEnabled`: 摄像头是否启用
- `isMicrophoneEnabled`: 麦克风是否启用
- `isVideoEnabled`: 视频是否启用（计算属性）
- `isAudioEnabled`: 音频是否启用（计算属性）

**方法：**
- `startCall(targetUserId)`: 开始通话
- `acceptCall(signal, fromUserId)`: 接受通话
- `rejectCall(fromUserId)`: 拒绝通话
- `endCall()`: 结束通话
- `toggleCamera()`: 切换摄像头
- `toggleMicrophone()`: 切换麦克风
- `enableCamera()`: 启用摄像头
- `disableCamera()`: 禁用摄像头
- `enableMicrophone()`: 启用麦克风
- `disableMicrophone()`: 禁用麦克风
- `handleAnswer(signal)`: 处理 answer
- `handleIceCandidate(candidate)`: 处理 ICE candidate

## 测试清单

### 屏幕共享

- [ ] 发送方：开始共享
- [ ] 接收方：收到共享请求
- [ ] 接收方：接受共享
- [ ] 接收方：拒绝共享
- [ ] 双方：视频流正常显示
- [ ] 发送方：暂停/恢复共享
- [ ] 发送方：停止共享
- [ ] 接收方：关闭窗口
- [ ] 网络断开重连

### 视频通话

- [ ] 发起方：开始通话
- [ ] 接收方：收到通话请求
- [ ] 接收方：接受通话
- [ ] 接收方：拒绝通话
- [ ] 双方：视频流正常显示
- [ ] 双方：音频正常
- [ ] 切换摄像头
- [ ] 切换麦克风
- [ ] 结束通话
- [ ] 网络断开重连

## 注意事项

1. **消息格式兼容性**：新代码使用 `media_type` 字段，需要确保服务器端正确转发该字段。

2. **状态管理**：新代码使用状态机管理状态，状态转换有明确的规则。

3. **错误处理**：所有操作都有完整的错误处理和日志记录。

4. **资源清理**：组件卸载时会自动清理资源，无需手动处理。

5. **向后兼容**：迁移期间，新旧代码可以共存，但不要在同一个组件中混用。

## 下一步

1. 更新服务器端消息处理逻辑，支持新的消息格式
2. 逐步迁移现有组件到新架构
3. 添加单元测试和集成测试
4. 完善文档和示例
