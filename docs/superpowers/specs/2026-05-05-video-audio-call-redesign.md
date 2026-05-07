# 视频和语音通话重新设计

## 背景

当前视频/语音通话功能存在架构不一致的问题：

- `CallModal.vue` 使用旧的 `useVideoCall.ts`（依赖 `videoCallManager`），而屏幕共享使用新的 `useScreenShareNew.ts`（基于 `useSession` + `useSignaling`）
- CallModal 是固定模态框，缺少浮动、拖拽、最小化、时长计时等交互功能
- 来电时自动接听，没有给用户接受/拒绝的交互流程
- `RealtimeCommunication.vue` 中视频/语音通话的处理逻辑不完整

屏幕共享功能已经建立了良好的架构模式，视频和语音通话应参照该模式重新设计。

## 设计决策

1. **统一组件** — 视频和语音通话使用同一个 `CallOverlay.vue` 组件，通过 `callType` 区分
2. **浮动覆盖层** — 采用可拖拽、可最小化的浮动覆盖层，和屏幕共享一致
3. **统一新架构** — 废弃旧的 `useVideoCall.ts`，统一使用 `useVideoCallNew.ts`
4. **来电不自动接听** — 收到来电 offer 后路由给 UI，由用户决定接听/拒绝

## 组件设计：CallOverlay.vue

### 状态机

```
idle → incoming-call    （收到来电）
idle → outgoing-call    （发起呼叫）
incoming-call → active  （接听）
incoming-call → idle    （拒绝）
outgoing-call → active  （对方接听）
outgoing-call → idle    （对方拒绝/取消）
active → idle           （结束通话）
```

### UI 布局

**正常状态：**

```
┌─────────────────────────────────────┐
│ [●] 正在通话...        [—] [×]     │  ← 可拖拽 header
├─────────────────────────────────────┤
│                                     │
│  [来电状态]                          │
│    头像 + 姓名                       │
│    [拒绝] [接听]                     │
│                                     │
│  [呼叫中状态]                        │
│    头像 + 姓名 + 呼叫动画            │
│    [取消]                           │
│                                     │
│  [视频通话中]                        │
│    远程视频（大窗口）                 │
│    本地视频（小窗口 PiP）             │
│                                     │
│  [语音通话中]                        │
│    头像 + 姓名 + 通话时长            │
│                                     │
├─────────────────────────────────────┤
│ 00:32  [🎤] [📷] [📞结束]          │  ← 控制栏
└─────────────────────────────────────┘
```

**最小化状态：**

```
┌──────────────────────────────────┐
│ [●] 通话中 01:23  [展开] [停止]   │
└──────────────────────────────────┘
```

### Props

```ts
interface Props {
  receiverId?: number
  conversationId?: number
  senderName?: string
}
```

### Events

```ts
emit('call-start', { conversationId: number })
emit('call-stop')
```

### Expose 方法

```ts
{
  initiateCall: (callType: 'voice' | 'video') => void
  handleIncomingOffer: (signal: RTCSessionDescriptionInit, fromUserId: number, callType: 'voice' | 'video') => void
  handleIncomingRequest: (data: any) => void
}
```

### 功能清单

- 来电邀请（接受/拒绝）
- 呼叫等待（等待对方接听，可取消）
- 视频通话显示（远程大窗口 + 本地 PiP）
- 语音通话显示（头像 + 姓名 + 时长）
- 静音切换
- 摄像头切换（仅视频通话）
- 结束通话
- 可拖拽（header 区域拖拽）
- 可最小化/展开
- 时长计时
- 本地预览窗口显示/隐藏

## Composable 改造：useVideoCallNew.ts

### 新增状态

```ts
const callType = ref<'voice' | 'video'>('video')
const callStatus = ref<'idle' | 'calling' | 'connecting' | 'connected' | 'ended'>('idle')
```

### 新增/修改方法

| 方法 | 说明 |
|------|------|
| `startCall(targetUserId, type)` | 发起呼叫，根据 type 获取不同媒体流 |
| `handleIncomingCall(signal, fromUserId, callType)` | 收到来电，不自动接听，等待 UI 操作 |
| `answerCall()` | 接听来电 |
| `rejectCall()` | 拒绝来电 |
| `toggleMute()` | 切换静音 |
| `toggleCamera()` | 切换摄像头 |

### Session 类型

统一使用 `'video-call'` session type，通过 `callType` 标志区分语音/视频。原因：

- 语音和视频的 WebRTC 连接本质相同（都是 getUserMedia）
- 区别仅在于是否包含视频轨道
- 避免创建两个独立的 session 实例导致状态管理复杂

### 媒体流获取

```ts
// 视频通话
const stream = await navigator.mediaDevices.getUserMedia({
  video: true,
  audio: true
})

// 语音通话
const stream = await navigator.mediaDevices.getUserMedia({
  video: false,
  audio: true
})
```

## 编排层改造：useRealtimeMessaging.ts

### 关键变更

`handleWebRTCOffer` 中对 video/audio 类型不再自动接听：

```ts
// 之前：自动接听
case 'video':
case 'audio':
  await videoCall.acceptCall(data.signal, data.from_user_id)
  break

// 之后：路由给 UI，由用户决定
case 'video':
case 'audio':
  // 不自动接听，通过回调通知 UI 层
  if (onCallOffer.value) {
    onCallOffer.value(data)
  }
  break
```

新增回调：

```ts
const onCallOffer = ref<((data: any) => void) | null>(null)
```

## 编排层改造：RealtimeCommunication.vue

### 替换 CallModal 为 CallOverlay

```vue
<Teleport to="body">
  <ScreenShareSimple ... />
  <CallOverlay
    v-if="showCallOverlay"
    ref="callOverlayRef"
    :receiver-id="callReceiverId"
    :conversation-id="callConversationId"
    :sender-name="remoteCallUserName"
    @call-start="handleCallStart"
    @call-stop="handleCallStop"
  />
</Teleport>
```

### 信令路由

`handleWebRTCOffer` 中 video/audio 类型的处理：

1. 判断 mediaType 为 video/audio
2. 显示 CallOverlay
3. 将 offer 数据传递给 CallOverlay 的 `handleIncomingOffer`
4. 由 CallOverlay 内部决定接听/拒绝

## 文件变更清单

| 操作 | 文件 | 说明 |
|------|------|------|
| 新建 | `src/components/shared/CallOverlay.vue` | 通话浮动覆盖层组件 |
| 修改 | `src/composables/useVideoCallNew.ts` | 增加状态管理、来电处理、媒体控制 |
| 修改 | `src/composables/useRealtimeMessaging.ts` | 来电不自动接听，路由给 UI |
| 修改 | `src/components/realtime/RealtimeCommunication.vue` | 替换 CallModal 为 CallOverlay |
| 废弃 | `src/components/chat/CallModal.vue` | 被 CallOverlay 替代 |
| 废弃 | `src/composables/useVideoCall.ts` | 被 useVideoCallNew 替代 |

## 错误处理

- 媒体设备获取失败：显示错误提示，重置状态为 idle
- WebRTC 连接失败：显示错误提示，自动结束通话
- 对方拒绝/无响应：显示相应提示，重置状态
- 网络断开：检测连接状态变化，自动结束通话

## 测试要点

- 发起视频通话 → 对方接听 → 通话中 → 结束
- 发起语音通话 → 对方接听 → 通话中 → 结束
- 收到来电 → 接听 → 通话中 → 结束
- 收到来电 → 拒绝
- 发起呼叫 → 对方拒绝
- 通话中切换静音
- 通话中切换摄像头（视频通话）
- 最小化/展开
- 拖拽
- 时长计时
- 网络异常断开
