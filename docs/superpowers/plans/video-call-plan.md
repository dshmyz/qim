# 视频通话功能实现计划

## 项目背景

QIM 是一个即时通讯项目，目前已有：
- `CallModal.vue` - 通话 UI 框架（视频区域只有占位符）
- `webrtc.js` - 屏幕共享的 WebRTC 实现
- 需要将视频通话功能从占位符状态变为真正可用

## 技术架构

```
┌─────────────┐     WebSocket      ┌─────────────┐
│  Caller     │◄─────────────────►│  Callee     │
│  Browser    │   (信令通道)       │  Browser    │
└──────┬──────┘                   └──────┬──────┘
       │                                   │
       │    ┌─────────────────┐           │
       │    │   STUN/TURN     │           │
       │    │   Server        │           │
       │    └─────────────────┘           │
       │              │                    │
       └──────────────┴────────────────────┘
              (WebRTC 媒体通道)
```

## 实现任务

### 任务 1：创建 WebRTC 视频通话核心模块

**文件**: `qim-client/src/utils/videoCall.ts`

**功能**:
- `VideoCallManager` 类 - 管理视频通话的完整生命周期
- 摄像头/麦克风采集
- RTCPeerConnection 管理
- offer/answer 生成和处理
- ICE 候选者处理
- 远程视频流管理

**关键实现**:
```typescript
class VideoCallManager {
  private peerConnection: RTCPeerConnection | null = null
  private localStream: MediaStream | null = null
  private remoteStream: MediaStream | null = null
  private callStatus: 'idle' | 'calling' | 'ringing' | 'answered' | 'ended' = 'idle'

  async startCall(targetUserId: string, callType: 'voice' | 'video'): Promise<void>
  async answerCall(): Promise<void>
  async endCall(): Promise<void>
  async toggleMute(muted: boolean): void
  async toggleVideo(enabled: boolean): void
  getLocalStream(): MediaStream | null
  getRemoteStream(): MediaStream | null
}
```

### 任务 2：创建视频通话 Composable

**文件**: `qim-client/src/composables/useVideoCall.ts`

**功能**:
- 封装 VideoCallManager 为 Vue Composable
- 提供响应式的通话状态
- 管理通话 UI 状态
- 与 CallModal 组件对接

**接口**:
```typescript
export function useVideoCall() {
  const callStatus = ref<'idle' | 'calling' | 'ringing' | 'answered' | 'ended'>('idle')
  const callType = ref<'voice' | 'video'>('video')
  const localStream = ref<MediaStream | null>(null)
  const remoteStream = ref<MediaStream | null>(null)
  const isMuted = ref(false)
  const isVideoEnabled = ref(true)
  const remoteUser = ref<{ id: number; name: string; avatar: string } | null>(null)

  const startCall = (user: User, type: 'voice' | 'video') => Promise<void>
  const answerCall = () => Promise<void>
  const endCall = () => Promise<void>
  const toggleMute = () => void
  const toggleVideo = () => void

  return {
    callStatus, callType, localStream, remoteStream,
    isMuted, isVideoEnabled, remoteUser,
    startCall, answerCall, endCall, toggleMute, toggleVideo
  }
}
```

### 任务 3：更新 CallModal 组件

**文件**: `qim-client/src/components/chat/CallModal.vue`

**修改**:
- 添加 `<video>` 元素显示本地和远程视频
- 添加通话控制按钮（静音、摄像头、结束通话）
- 集成 `useVideoCall` Composable
- 更新 UI 样式适配视频显示

**新增按钮**:
```vue
<button v-if="callType === 'video' && callStatus === 'answered'" @click="toggleVideo">
  {{ isVideoEnabled ? '关闭摄像头' : '开启摄像头' }}
</button>
<button v-if="callStatus === 'answered'" @click="toggleMute">
  {{ isMuted ? '取消静音' : '静音' }}
</button>
```

### 任务 4：实现信令流程

**修改文件**: `qim-client/src/composables/useWebSocket.ts` 或创建新文件

**WebSocket 消息类型**:
```typescript
// 发起通话
{ type: 'call_invite', data: { target_user_id, call_type, caller_info } }

// 取消通话
{ type: 'call_cancel', data: { target_user_id } }

// 接听通话
{ type: 'call_accept', data: { caller_id } }

// 拒绝通话
{ type: 'call_reject', data: { caller_id, reason? } }

// 结束通话
{ type: 'call_end', data: { target_user_id } }

// WebRTC 信令
{ type: 'webrtc_offer', data: { target_user_id, signal: RTCSessionDescription } }
{ type: 'webrtc_answer', data: { target_user_id, signal: RTCSessionDescription } }
{ type: 'webrtc_ice_candidate', data: { target_user_id, signal: RTCIceCandidate } }
```

### 任务 5：添加通话控制功能

**功能**:
- 静音切换（麦克风）
- 摄像头开关
- 通话计时器显示

## 约束条件

1. **复用现有架构**: 复用 `webrtc.js` 中的信令处理逻辑
2. **兼容性**: 支持现代浏览器（Chrome、Firefox、Safari、Edge）
3. **性能**: 视频通话使用适当的分辨率和帧率
4. **错误处理**: 网络断开、对方挂断、浏览器不支持等情况

## 成功标准

1. ✅ 用户可以发起视频通话
2. ✅ 被呼叫用户收到来电通知
3. ✅ 双方可以建立 WebRTC 连接
4. ✅ 双方可以看到对方的视频画面
5. ✅ 支持静音/取消静音
6. ✅ 支持关闭/开启摄像头
7. ✅ 通话结束正确释放资源

## 后续扩展

- 群组视频通话
- 屏幕共享集成到视频通话中
- 通话录制
