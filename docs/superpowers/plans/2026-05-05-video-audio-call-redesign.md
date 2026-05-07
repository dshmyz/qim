# 视频和语音通话重新设计 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 参照屏幕共享的架构模式，重新设计视频和语音通话功能，使用浮动覆盖层 UI 和统一的新架构 composable。

**架构：** 创建 CallOverlay.vue（参照 ScreenShareSimple.vue 的模式），改造 useVideoCallNew.ts 增加来电处理和媒体控制，修改 useRealtimeMessaging.ts 使来电不自动接听，修改 RealtimeCommunication.vue 替换 CallModal 为 CallOverlay，从 OverlayManager/ChatWindow 中移除旧的 CallModal 集成。

**技术栈：** Vue 3 + TypeScript + WebRTC + Composable 模式（useSession + useSignaling）

---

## 文件结构

| 操作 | 文件 | 职责 |
|------|------|------|
| 新建 | `src/components/shared/CallOverlay.vue` | 通话浮动覆盖层组件，自包含 UI 和交互逻辑 |
| 修改 | `src/composables/useVideoCallNew.ts` | 增加通话状态管理、来电处理流程、媒体控制 |
| 修改 | `src/composables/useRealtimeMessaging.ts` | 来电不自动接听，增加 onCallOffer 回调路由给 UI |
| 修改 | `src/components/realtime/RealtimeCommunication.vue` | 替换 CallModal 为 CallOverlay，调整信令路由 |
| 修改 | `src/components/chat/OverlayManager.vue` | 移除 CallModal 及相关 props/events |
| 修改 | `src/components/chat/ChatWindow.vue` | 移除旧的 useVideoCall 集成和 call 相关 props/events |
| 废弃 | `src/components/chat/CallModal.vue` | 被 CallOverlay 替代，不再被引用 |
| 废弃 | `src/composables/useVideoCall.ts` | 被 useVideoCallNew 替代，不再被引用 |

---

### 任务 1：改造 useVideoCallNew.ts — 增加通话状态和来电处理

**文件：**
- 修改：`src/composables/useVideoCallNew.ts`

- [ ] **步骤 1：增加通话类型和状态管理**

在 `createVideoCall` 函数中，在 `isCameraEnabled` 和 `isMicrophoneEnabled` 之后添加：

```ts
const callType = ref<'voice' | 'video'>('video')
const callStatus = ref<'idle' | 'calling' | 'connecting' | 'connected' | 'ended'>('idle')
```

- [ ] **步骤 2：修改 startCall 方法，支持通话类型参数并更新状态**

将现有 `startCall` 方法替换为：

```ts
const startCall = async (targetUserId: number, type: 'voice' | 'video' = 'video') => {
  console.log('[VideoCall] Starting call with user:', targetUserId, 'type:', type)

  if (callStatus.value !== 'idle') {
    console.warn('[VideoCall] Cannot start call in state:', callStatus.value)
    throw new Error('当前正在通话中')
  }

  callType.value = type
  callStatus.value = 'calling'

  try {
    signaling.sendCallStart(targetUserId, type)
    await session.start(targetUserId)

    console.log('[VideoCall] Call started successfully')
  } catch (error) {
    console.error('[VideoCall] Failed to start call:', error)
    callStatus.value = 'idle'
    throw error
  }
}
```

- [ ] **步骤 3：增加 handleIncomingCall 方法**

在 `startCall` 方法之后添加：

```ts
const handleIncomingCall = (signal: RTCSessionDescriptionInit, fromUserId: number, type: 'voice' | 'video' = 'video') => {
  console.log('[VideoCall] Incoming call from user:', fromUserId, 'type:', type)

  if (callStatus.value !== 'idle') {
    console.warn('[VideoCall] Cannot accept incoming call in state:', callStatus.value)
    return
  }

  callType.value = type
  callStatus.value = 'calling'
  pendingOffer.value = { signal, fromUserId }
}
```

在 `isMicrophoneEnabled` 之后添加：

```ts
const pendingOffer = ref<{ signal: RTCSessionDescriptionInit; fromUserId: number } | null>(null)
```

- [ ] **步骤 4：修改 acceptCall 方法，支持从 pendingOffer 接听**

将现有 `acceptCall` 方法替换为：

```ts
const acceptCall = async (signal?: RTCSessionDescriptionInit, fromUserId?: number) => {
  console.log('[VideoCall] Accepting call')

  const offer = signal || pendingOffer.value?.signal
  const userId = fromUserId || pendingOffer.value?.fromUserId

  if (!offer || !userId) {
    console.error('[VideoCall] No offer data to accept call')
    throw new Error('没有来电数据')
  }

  callStatus.value = 'connecting'

  try {
    signaling.sendCallAnswer(userId, true)
    await session.join(offer, userId)
    pendingOffer.value = null

    console.log('[VideoCall] Call accepted successfully')
  } catch (error) {
    console.error('[VideoCall] Failed to accept call:', error)
    signaling.sendCallAnswer(userId, false)
    callStatus.value = 'idle'
    pendingOffer.value = null
    throw error
  }
}
```

- [ ] **步骤 5：修改 rejectCall 方法**

将现有 `rejectCall` 方法替换为：

```ts
const rejectCall = () => {
  console.log('[VideoCall] Rejecting call')

  const fromUserId = pendingOffer.value?.fromUserId
  if (fromUserId) {
    signaling.sendCallAnswer(fromUserId, false)
  }

  callStatus.value = 'idle'
  pendingOffer.value = null
}
```

- [ ] **步骤 6：修改 endCall 方法，更新状态**

将现有 `endCall` 方法替换为：

```ts
const endCall = () => {
  console.log('[VideoCall] Ending call')

  signaling.sendCallEnd()
  session.end()
  callStatus.value = 'idle'
  pendingOffer.value = null

  console.log('[VideoCall] Call ended successfully')
}
```

- [ ] **步骤 7：增加 toggleMute 和 toggleCamera 方法**

现有 `toggleMicrophone` 和 `toggleCamera` 已存在，但需要增加别名并确保状态同步。在 `disableMicrophone` 之后添加：

```ts
const toggleMute = () => {
  toggleMicrophone()
}

const toggleCamera = () => {
  toggleCamera()
}
```

注意：现有 `toggleCamera` 方法名与新增别名冲突。将现有的 `toggleCamera` 重命名为 `toggleCameraInternal`，然后：

```ts
const toggleCameraInternal = () => {
  console.log('[VideoCall] Toggling camera')

  if (session.localStream.value) {
    const videoTracks = session.localStream.value.getVideoTracks()
    videoTracks.forEach(track => {
      track.enabled = !track.enabled
    })
    isCameraEnabled.value = videoTracks.some(t => t.enabled)

    console.log('[VideoCall] Camera enabled:', isCameraEnabled.value)
  }
}
```

然后添加对外暴露的 `toggleCamera`：

```ts
const toggleCamera = () => {
  toggleCameraInternal()
}
```

- [ ] **步骤 8：监听 session 状态同步 callStatus**

在 `createVideoCall` 函数末尾、return 之前添加：

```ts
watch(session.sessionState, (state) => {
  if (state === 'active' && callStatus.value === 'connecting') {
    callStatus.value = 'connected'
  } else if (state === 'idle' || state === 'ended') {
    if (callStatus.value !== 'idle') {
      callStatus.value = 'idle'
      pendingOffer.value = null
    }
  }
})
```

在文件顶部 import 中添加 `watch`：

```ts
import { ref, computed, watch } from 'vue'
```

- [ ] **步骤 9：更新 return 对象，暴露新增属性和方法**

将 return 对象替换为：

```ts
return {
  ...session,
  callType,
  callStatus,
  isCameraEnabled,
  isMicrophoneEnabled,
  isVideoEnabled,
  isAudioEnabled,
  pendingOffer,
  startCall,
  acceptCall,
  rejectCall,
  endCall,
  toggleMute,
  toggleCamera,
  toggleMicrophone,
  enableCamera,
  disableCamera,
  enableMicrophone,
  disableMicrophone
}
```

- [ ] **步骤 10：验证编译通过**

运行：`cd /Users/gracegaoya/work/project/qim/qim-client && npx vue-tsc --noEmit 2>&1 | head -30`
预期：无与 useVideoCallNew.ts 相关的类型错误

---

### 任务 2：改造 useRealtimeMessaging.ts — 来电不自动接听

**文件：**
- 修改：`src/composables/useRealtimeMessaging.ts`

- [ ] **步骤 1：增加 onCallOffer 回调**

在 `onWebRTCOffer` 之后添加：

```ts
const onCallOffer = ref<((data: any) => void) | null>(null)
```

- [ ] **步骤 2：修改 handleWebRTCOffer，video/audio 不自动接听**

将 `handleWebRTCOffer` 中 `case 'video'` 和 `case 'audio'` 分支替换为：

```ts
case 'video':
case 'audio':
  console.log('[RealtimeMessaging] Call offer - routing to UI for acceptance')
  if (onCallOffer.value) {
    onCallOffer.value(data)
  }
  break
```

- [ ] **步骤 3：更新 return 对象，暴露 onCallOffer**

在 return 对象中添加 `onCallOffer`：

```ts
return {
  screenShare,
  videoCall,
  onScreenShareRequest,
  onScreenShareAccepted,
  onScreenShareRejected,
  onWebRTCOffer,
  onCallOffer,
  // ... 其余不变
}
```

- [ ] **步骤 4：验证编译通过**

运行：`cd /Users/gracegaoya/work/project/qim/qim-client && npx vue-tsc --noEmit 2>&1 | head -30`
预期：无与 useRealtimeMessaging.ts 相关的类型错误

---

### 任务 3：创建 CallOverlay.vue — 通话浮动覆盖层组件

**文件：**
- 创建：`src/components/shared/CallOverlay.vue`

- [ ] **步骤 1：创建 CallOverlay.vue 完整文件**

```vue
<template>
  <Teleport to="body">
    <div
      v-if="showOverlay"
      ref="callOverlayRef"
      class="call-overlay"
      :class="{ minimized: isMinimized, dragging: isDragging }"
    >
      <div
        class="call-overlay-header"
        @mousedown="startDrag"
        @dblclick="toggleMinimize"
      >
        <div class="header-left">
          <span class="status-dot" :class="{ active: callStatus === 'connected' }"></span>
          <span class="title">{{ title }}</span>
        </div>
        <div class="header-actions">
          <button
            v-if="!isMinimized"
            class="action-btn"
            @click="toggleMinimize"
            title="最小化"
          >
            <i class="fas fa-minus"></i>
          </button>
          <button
            class="action-btn close-btn"
            @click="handleStop"
            title="关闭"
          >
            <i class="fas fa-times"></i>
          </button>
        </div>
      </div>

      <div v-if="!isMinimized" class="call-overlay-body">
        <div v-if="showIncomingCall" class="incoming-call">
          <div class="incoming-icon" :class="incomingCallType === 'voice' ? 'voice' : 'video'">
            <i :class="incomingCallType === 'voice' ? 'fas fa-phone' : 'fas fa-video'"></i>
          </div>
          <div class="incoming-text">
            <span class="incoming-title">{{ incomingCallInfo?.fromUserName || props.senderName || '对方' }} 邀请你{{ incomingCallType === 'voice' ? '语音' : '视频' }}通话</span>
          </div>
          <div class="incoming-actions">
            <button class="btn-reject" @click="handleReject">拒绝</button>
            <button class="btn-accept" @click="handleAccept">接听</button>
          </div>
        </div>

        <div v-else-if="showOutgoingCall" class="outgoing-call">
          <div class="outgoing-icon" :class="outgoingCallType === 'voice' ? 'voice' : 'video'">
            <i :class="outgoingCallType === 'voice' ? 'fas fa-phone' : 'fas fa-video'"></i>
          </div>
          <div class="outgoing-text">
            <span>正在呼叫{{ outgoingCallType === 'voice' ? '语音' : '视频' }}通话...</span>
          </div>
          <button class="btn-secondary" @click="handleStop">取消</button>
        </div>

        <div v-else class="call-content">
          <div v-if="currentCallType === 'video'" class="video-container">
            <video
              ref="remoteVideoRef"
              v-show="remoteStream"
              autoplay
              playsinline
              class="remote-video"
            ></video>
            <div v-if="!remoteStream" class="video-placeholder">
              <i class="fas fa-spinner fa-spin"></i>
              <span>连接中...</span>
            </div>
            <div v-if="showLocalPreview" class="local-video">
              <video
                v-if="localStream && isCameraEnabled"
                ref="localVideoRef"
                autoplay
                playsinline
                muted
                class="video-element"
              ></video>
              <div v-else class="local-video-placeholder">
                <i class="fas fa-user"></i>
              </div>
              <button class="local-video-close-btn" @click.stop="showLocalPreview = false" title="隐藏预览">
                <i class="fas fa-times"></i>
              </button>
            </div>
            <div v-if="!showLocalPreview" class="local-video-toggle-btn">
              <button @click.stop="showLocalPreview = true" title="显示预览">
                <i class="fas fa-user"></i>
              </button>
            </div>
          </div>

          <div v-else class="voice-call-info">
            <div class="voice-avatar">
              <img v-if="callAvatar" :src="callAvatar" alt="avatar" />
              <i v-else class="fas fa-user"></i>
            </div>
            <div class="voice-name">{{ callName || '对方' }}</div>
            <div class="voice-duration">{{ formattedDuration }}</div>
          </div>
        </div>

        <div v-if="!showIncomingCall && !showOutgoingCall" class="call-controls">
          <div class="duration" v-if="currentCallType === 'video'">{{ formattedDuration }}</div>
          <div class="controls-actions">
            <button
              class="control-btn"
              :class="{ 'control-btn-active': !isAudioEnabled }"
              @click="handleToggleMute"
              :title="isAudioEnabled ? '静音' : '取消静音'"
            >
              <i :class="isAudioEnabled ? 'fas fa-microphone' : 'fas fa-microphone-slash'"></i>
              <span>{{ isAudioEnabled ? '麦克风' : '静音' }}</span>
            </button>
            <button
              v-if="currentCallType === 'video'"
              class="control-btn"
              :class="{ 'control-btn-active': !isCameraEnabled }"
              @click="handleToggleCamera"
              :title="isCameraEnabled ? '关闭摄像头' : '开启摄像头'"
            >
              <i :class="isCameraEnabled ? 'fas fa-video' : 'fas fa-video-slash'"></i>
              <span>{{ isCameraEnabled ? '摄像头' : '已关闭' }}</span>
            </button>
            <button
              class="control-btn stop-btn"
              @click="handleStop"
              title="结束通话"
            >
              <i class="fas fa-phone-slash"></i>
              <span>结束</span>
            </button>
          </div>
        </div>
      </div>

      <div v-if="isMinimized" class="minimized-content" @click="toggleMinimize">
        <div class="minimized-info">
          <div class="info-top">
            <div class="minimized-status">
              <span class="pulse-dot" :class="{ active: callStatus === 'connected' }"></span>
              <span>{{ currentCallType === 'voice' ? '语音通话' : '视频通话' }}</span>
            </div>
            <div class="minimized-duration">{{ formattedDuration }}</div>
          </div>
          <div class="minimized-actions" @click.stop>
            <button class="action-btn expand-btn" @click="toggleMinimize">
              <i class="fas fa-expand"></i>
              <span>展开</span>
            </button>
            <button class="action-btn close-btn" @click="handleStop">
              <i class="fas fa-stop"></i>
              <span>停止</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted, nextTick, inject } from 'vue'
import { useVideoCallNew } from '@/composables/useVideoCallNew'

const props = defineProps<{
  receiverId?: number
  conversationId?: number
  senderName?: string
}>()

const emit = defineEmits<{
  'call-start': [data: { conversationId: string | number }]
  'call-stop': []
}>()

const videoCall = useVideoCallNew()

const callOverlayRef = ref<HTMLElement | null>(null)
const localVideoRef = ref<HTMLVideoElement | null>(null)
const remoteVideoRef = ref<HTMLVideoElement | null>(null)

const isMinimized = ref(false)
const isDragging = ref(false)
const showLocalPreview = ref(true)
const dragState = ref({
  startX: 0,
  startY: 0,
  elementX: 0,
  elementY: 0
})
let rafId: number | null = null

const duration = ref(0)
let durationTimer: number | null = null

const callAvatar = ref('')
const callName = ref('')

const showIncomingCall = ref(false)
const showOutgoingCall = ref(false)
const incomingCallType = ref<'voice' | 'video'>('video')
const outgoingCallType = ref<'voice' | 'video'>('video')
const incomingCallInfo = ref<{ fromUserId: number; fromUserName: string } | null>(null)

const callStatus = computed(() => videoCall.callStatus.value)
const callType = computed(() => videoCall.callType.value)
const localStream = computed(() => videoCall.localStream.value)
const remoteStream = computed(() => videoCall.remoteStream.value)
const isCameraEnabled = computed(() => videoCall.isCameraEnabled.value)
const isAudioEnabled = computed(() => videoCall.isAudioEnabled.value)

const currentCallType = computed(() => callType.value || (showIncomingCall.value ? incomingCallType.value : outgoingCallType.value))

const showOverlay = computed(() => {
  return showIncomingCall.value || showOutgoingCall.value || callStatus.value === 'connecting' || callStatus.value === 'connected'
})

const title = computed(() => {
  if (showIncomingCall.value) {
    return '来电邀请'
  }
  if (showOutgoingCall.value) {
    return '正在呼叫...'
  }
  if (currentCallType.value === 'voice') {
    return '语音通话'
  }
  return '视频通话'
})

const formattedDuration = computed(() => {
  const hours = Math.floor(duration.value / 3600)
  const minutes = Math.floor((duration.value % 3600) / 60)
  const seconds = duration.value % 60

  if (hours > 0) {
    return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
  }
  return `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
})

const initiateCall = async (type: 'voice' | 'video') => {
  if (!props.receiverId) {
    console.error('[CallOverlay] No receiverId')
    return
  }

  outgoingCallType.value = type
  showOutgoingCall.value = true

  try {
    await videoCall.startCall(props.receiverId, type)
  } catch (error) {
    console.error('[CallOverlay] Failed to start call:', error)
    showOutgoingCall.value = false
  }
}

const handleIncomingOffer = (signal: RTCSessionDescriptionInit, fromUserId: number, type: 'voice' | 'video') => {
  if (callStatus.value !== 'idle' && !showIncomingCall.value) {
    console.warn('[CallOverlay] Cannot accept incoming call in state:', callStatus.value)
    return
  }

  incomingCallType.value = type
  incomingCallInfo.value = {
    fromUserId,
    fromUserName: props.senderName || '对方'
  }
  videoCall.handleIncomingCall(signal, fromUserId, type)
  showIncomingCall.value = true
}

const handleAccept = async () => {
  showIncomingCall.value = false

  try {
    await videoCall.acceptCall()
  } catch (error) {
    console.error('[CallOverlay] Failed to accept call:', error)
    showIncomingCall.value = false
  }
}

const handleReject = () => {
  showIncomingCall.value = false
  videoCall.rejectCall()
  incomingCallInfo.value = null
}

const handleStop = () => {
  videoCall.endCall()
  showIncomingCall.value = false
  showOutgoingCall.value = false
  incomingCallInfo.value = null
  callAvatar.value = ''
  callName.value = ''
  emit('call-stop')
}

const handleToggleMute = () => {
  videoCall.toggleMute()
}

const handleToggleCamera = () => {
  videoCall.toggleCamera()
}

const startDrag = (e: MouseEvent) => {
  e.preventDefault()
  const element = callOverlayRef.value
  if (!element) return

  const rect = element.getBoundingClientRect()

  dragState.value = {
    startX: e.clientX,
    startY: e.clientY,
    elementX: rect.left,
    elementY: rect.top
  }

  isDragging.value = true

  document.addEventListener('mousemove', onDrag)
  document.addEventListener('mouseup', stopDrag)
}

const onDrag = (e: MouseEvent) => {
  if (!isDragging.value) return

  const deltaX = e.clientX - dragState.value.startX
  const deltaY = e.clientY - dragState.value.startY

  let newX = dragState.value.elementX + deltaX
  let newY = dragState.value.elementY + deltaY

  const element = callOverlayRef.value
  if (element) {
    const rect = element.getBoundingClientRect()
    newX = Math.max(0, Math.min(newX, window.innerWidth - rect.width))
    newY = Math.max(0, Math.min(newY, window.innerHeight - rect.height))
  }

  if (rafId) {
    cancelAnimationFrame(rafId)
  }

  rafId = requestAnimationFrame(() => {
    if (element) {
      element.style.left = `${newX}px`
      element.style.top = `${newY}px`
      element.style.transform = 'none'
    }
  })
}

const stopDrag = () => {
  if (rafId) {
    cancelAnimationFrame(rafId)
    rafId = null
  }
  isDragging.value = false
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
}

const toggleMinimize = () => {
  isMinimized.value = !isMinimized.value
}

const startDurationTimer = () => {
  if (durationTimer) {
    clearInterval(durationTimer)
  }
  durationTimer = window.setInterval(() => {
    duration.value++
  }, 1000)
}

const stopDurationTimer = () => {
  if (durationTimer) {
    clearInterval(durationTimer)
    durationTimer = null
  }
  duration.value = 0
}

watch([localStream, isCameraEnabled], ([stream, cameraEnabled]) => {
  if (stream && cameraEnabled) {
    nextTick(() => {
      if (localVideoRef.value && localVideoRef.value.srcObject !== stream) {
        localVideoRef.value.srcObject = stream
      }
    })
  }
}, { immediate: true })

watch(remoteStream, (stream) => {
  if (stream) {
    nextTick(() => {
      if (remoteVideoRef.value && remoteVideoRef.value.srcObject !== stream) {
        remoteVideoRef.value.srcObject = stream
        remoteVideoRef.value.play().catch(() => {})
      }
    })
  }
}, { immediate: true })

watch(callStatus, (status) => {
  if (status === 'connected') {
    startDurationTimer()
    showOutgoingCall.value = false
    emit('call-start', { conversationId: props.conversationId || props.receiverId || 0 })
  } else if (status === 'idle' || status === 'ended') {
    stopDurationTimer()
    showIncomingCall.value = false
    showOutgoingCall.value = false
  }
})

onUnmounted(() => {
  stopDurationTimer()
  if (rafId) {
    cancelAnimationFrame(rafId)
  }
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
})

defineExpose({
  initiateCall,
  handleIncomingOffer
})
</script>

<style scoped>
.call-overlay {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 480px;
  background: rgba(30, 30, 30, 0.95);
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  z-index: 10000;
  overflow: hidden;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.call-overlay.minimized {
  width: 280px;
  height: auto;
  min-height: 80px;
}

.call-overlay.dragging {
  cursor: grabbing;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.5);
  transition: box-shadow 0.2s ease-out;
}

.call-overlay-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.1) 0%, rgba(255, 255, 255, 0.05) 100%);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  cursor: grab;
  user-select: none;
}

.call-overlay-header:active {
  cursor: grabbing;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.3);
}

.status-dot.active {
  background: #10b981;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.title {
  color: #fff;
  font-size: 14px;
  font-weight: 500;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.action-btn {
  padding: 6px 12px;
  background: rgba(255, 255, 255, 0.1);
  border: none;
  border-radius: 6px;
  color: #fff;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 4px;
}

.action-btn:hover {
  background: rgba(255, 255, 255, 0.2);
}

.close-btn:hover {
  background: rgba(239, 68, 68, 0.8);
}

.call-overlay-body {
  display: flex;
  flex-direction: column;
}

.incoming-call {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 32px 24px;
  gap: 16px;
}

.incoming-icon {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  color: #fff;
}

.incoming-icon.video {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
}

.incoming-icon.voice {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
}

.incoming-text {
  text-align: center;
}

.incoming-title {
  color: #fff;
  font-size: 16px;
  font-weight: 500;
}

.incoming-actions {
  display: flex;
  gap: 12px;
  margin-top: 8px;
}

.btn-reject {
  padding: 10px 24px;
  background: rgba(239, 68, 68, 0.2);
  border: 1px solid rgba(239, 68, 68, 0.5);
  border-radius: 8px;
  color: #ef4444;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
}

.btn-reject:hover {
  background: rgba(239, 68, 68, 0.3);
}

.btn-accept {
  padding: 10px 24px;
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
  border: none;
  border-radius: 8px;
  color: #fff;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
}

.btn-accept:hover {
  background: linear-gradient(135deg, #059669 0%, #047857 100%);
}

.outgoing-call {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 40px 24px;
  gap: 16px;
}

.outgoing-icon {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  color: #fff;
  animation: ringPulse 2s infinite;
}

.outgoing-icon.video {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
}

.outgoing-icon.voice {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
}

@keyframes ringPulse {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.1); }
}

.outgoing-text {
  color: rgba(255, 255, 255, 0.8);
  font-size: 15px;
}

.btn-secondary {
  padding: 8px 20px;
  background: rgba(255, 255, 255, 0.1);
  border: none;
  border-radius: 6px;
  color: #fff;
  cursor: pointer;
  font-size: 14px;
}

.btn-secondary:hover {
  background: rgba(255, 255, 255, 0.2);
}

.call-content {
  display: flex;
  flex-direction: column;
}

.video-container {
  position: relative;
  width: 100%;
  height: 300px;
  background: #000;
}

.remote-video {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.video-placeholder {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: rgba(255, 255, 255, 0.5);
}

.video-placeholder i {
  font-size: 32px;
}

.local-video {
  position: absolute;
  bottom: 16px;
  right: 16px;
  width: 120px;
  height: 90px;
  border-radius: 8px;
  overflow: hidden;
  background: #2a2a2a;
  border: 2px solid rgba(59, 130, 246, 0.5);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.local-video .video-element {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transform: scaleX(-1);
}

.local-video-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.3);
  font-size: 24px;
}

.local-video-close-btn {
  position: absolute;
  top: 4px;
  right: 4px;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.6);
  color: #fff;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  transition: all 0.2s ease;
  z-index: 10;
}

.local-video-close-btn:hover {
  background: rgba(0, 0, 0, 0.8);
  transform: scale(1.1);
}

.local-video-toggle-btn {
  position: absolute;
  bottom: 16px;
  right: 16px;
  z-index: 10;
}

.local-video-toggle-btn button {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.6);
  color: #fff;
  border: 2px solid rgba(59, 130, 246, 0.5);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  transition: all 0.2s ease;
}

.local-video-toggle-btn button:hover {
  background: rgba(0, 0, 0, 0.8);
  transform: scale(1.1);
}

.voice-call-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 40px 24px;
  gap: 12px;
}

.voice-avatar {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.3);
  font-size: 32px;
  border: 2px solid rgba(59, 130, 246, 0.5);
}

.voice-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.voice-name {
  color: #fff;
  font-size: 18px;
  font-weight: 500;
}

.voice-duration {
  color: rgba(255, 255, 255, 0.7);
  font-size: 14px;
  font-family: 'SF Mono', Monaco, monospace;
}

.call-controls {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: rgba(0, 0, 0, 0.3);
}

.duration {
  color: rgba(255, 255, 255, 0.7);
  font-size: 14px;
  font-family: 'SF Mono', Monaco, monospace;
}

.controls-actions {
  display: flex;
  gap: 8px;
}

.control-btn {
  padding: 8px 16px;
  background: rgba(255, 255, 255, 0.1);
  border: none;
  border-radius: 6px;
  color: #fff;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 6px;
}

.control-btn:hover {
  background: rgba(255, 255, 255, 0.2);
}

.control-btn-active {
  background: rgba(255, 152, 0, 0.8);
}

.control-btn-active:hover {
  background: rgba(255, 152, 0, 0.9);
}

.stop-btn {
  background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
}

.stop-btn:hover {
  background: linear-gradient(135deg, #dc2626 0%, #b91c1c 100%);
}

.minimized-content {
  display: flex;
  padding: 12px;
  gap: 12px;
  cursor: pointer;
  transition: background 0.2s;
}

.minimized-content:hover {
  background: rgba(255, 255, 255, 0.05);
}

.minimized-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  min-width: 0;
}

.info-top {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.minimized-status {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #fff;
  font-size: 13px;
  font-weight: 500;
}

.pulse-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.3);
}

.pulse-dot.active {
  background: #10b981;
  animation: pulse 2s infinite;
}

.minimized-duration {
  color: rgba(255, 255, 255, 0.7);
  font-size: 12px;
  font-family: 'SF Mono', Monaco, monospace;
}

.minimized-actions {
  display: flex;
  gap: 8px;
}

.minimized-actions .action-btn {
  flex: 1;
  padding: 6px 10px;
  font-size: 11px;
}

.expand-btn {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
}

.expand-btn:hover {
  background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
}
</style>
```

- [ ] **步骤 2：验证编译通过**

运行：`cd /Users/gracegaoya/work/project/qim/qim-client && npx vue-tsc --noEmit 2>&1 | head -30`
预期：无与 CallOverlay.vue 相关的类型错误

---

### 任务 4：改造 RealtimeCommunication.vue — 替换 CallModal 为 CallOverlay

**文件：**
- 修改：`src/components/realtime/RealtimeCommunication.vue`

- [ ] **步骤 1：替换 import 和组件注册**

将 `import CallModal from '../chat/CallModal.vue'` 替换为：

```ts
import CallOverlay from '../shared/CallOverlay.vue'
```

- [ ] **步骤 2：替换 template 中的 CallModal**

将 template 中的 CallModal 部分：

```vue
<CallModal
  :visible="showCallModal"
  :call-type="callType"
  :call-status="callStatus"
  :avatar="callAvatar || ''"
  :name="callName || '未知用户'"
  @reject-call="handleRejectCall"
  @answer-call="handleAnswerCall"
  @end-call="handleEndCall"
  @close-call-modal="handleCallModalClose"
/>
```

替换为：

```vue
<CallOverlay
  v-if="showCallOverlay"
  ref="callOverlayRef"
  :receiver-id="callReceiverId"
  :conversation-id="callConversationId"
  :sender-name="remoteCallUserName"
  @call-start="handleCallStart"
  @call-stop="handleCallStop"
/>
```

- [ ] **步骤 3：替换 script 中的状态变量**

移除以下变量：

```ts
const showCallModal = ref(false)
const callType = ref<'voice' | 'video'>('video')
const callStatus = ref<'idle' | 'calling' | 'connecting' | 'connected' | 'ended'>('idle')
const callAvatar = ref('')
const callName = ref('')
```

替换为：

```ts
const showCallOverlay = ref(false)
const callReceiverId = ref<number | undefined>(undefined)
const callConversationId = ref<number | undefined>(undefined)
const remoteCallUserName = ref('')
const callOverlayRef = ref<InstanceType<typeof CallOverlay>>()
```

- [ ] **步骤 4：替换事件处理方法**

移除以下方法：

```ts
const handleRejectCall = async () => { ... }
const handleAnswerCall = async () => { ... }
const handleEndCall = async () => { ... }
const handleCallModalClose = async () => { ... }
```

替换为：

```ts
const startCall = async (type: 'voice' | 'video') => {
  if (!props.currentConversation) {
    QMessage.warning('请先选择一个会话')
    return
  }

  const user = getCurrentUser()
  if (!user || !user.id) {
    QMessage.warning('用户信息未加载，无法使用通话功能')
    return
  }

  const conv = props.currentConversation
  if (conv?.type === 'single' && conv.members && conv.members.length === 2) {
    const otherMember = conv.members.find(m => String(m.id) !== String(user.id))
    if (otherMember) {
      callReceiverId.value = Number(otherMember.id)
      callConversationId.value = Number(conv.id)
    }
  }

  showCallOverlay.value = true

  await nextTick()

  if (callOverlayRef.value?.initiateCall) {
    try {
      await callOverlayRef.value.initiateCall(type)
    } catch (error) {
      console.error('[RealtimeCommunication] 发起通话失败:', error)
      showCallOverlay.value = false
    }
  }
}

const handleCallStart = (data: any) => {
  console.log('[RealtimeCommunication] 通话开始', data)
  emit('call-state-change', 'connected')
}

const handleCallStop = () => {
  console.log('[RealtimeCommunication] 通话结束')
  showCallOverlay.value = false
  emit('call-state-change', 'idle')
}
```

- [ ] **步骤 5：修改 handleWebRTCOffer 中 video/audio 分支**

将 `handleWebRTCOffer` 中 video/audio 分支：

```ts
} else if (mediaType === 'video' || mediaType === 'audio') {
  console.log('[RealtimeCommunication] 处理视频/语音通话 offer')

  showCallModal.value = true
  callStatus.value = 'calling'
  callType.value = mediaType === 'audio' ? 'voice' : 'video'

  const fromUserId = data.from_user_id
  const memberInfo = getMemberInfo(fromUserId)
  callAvatar.value = memberInfo.avatar
  callName.value = memberInfo.name

  await realtimeMessaging.handleWebRTCOffer(data)
}
```

替换为：

```ts
} else if (mediaType === 'video' || mediaType === 'audio') {
  console.log('[RealtimeCommunication] 处理视频/语音通话 offer')

  const fromUserId = data.from_user_id
  const memberInfo = getMemberInfo(fromUserId)
  remoteCallUserName.value = memberInfo.name

  showCallOverlay.value = true

  await nextTick()

  if (callOverlayRef.value?.handleIncomingOffer) {
    callOverlayRef.value.handleIncomingOffer(data.signal, fromUserId, mediaType === 'audio' ? 'voice' : 'video')
  }
}
```

- [ ] **步骤 6：修改 handleVideoCallSignaling**

将 `handleVideoCallSignaling` 方法替换为：

```ts
const handleVideoCallSignaling = (message: { type: string; data: any }) => {
  console.log('[RealtimeCommunication] 收到视频通话信令', message)

  switch (message.type) {
    case 'call_invite':
      showCallOverlay.value = true
      const fromUserId = message.data.from_user_id
      const memberInfo = getMemberInfo(fromUserId)
      remoteCallUserName.value = memberInfo.name

      nextTick(() => {
        if (callOverlayRef.value?.handleIncomingOffer) {
          callOverlayRef.value.handleIncomingOffer(
            message.data.signal,
            fromUserId,
            message.data.call_type === 'voice' ? 'voice' : 'video'
          )
        }
      })
      break
    case 'call_accept':
      break
    case 'call_end':
    case 'call_reject':
      showCallOverlay.value = false
      break
  }
}
```

- [ ] **步骤 7：更新 defineExpose**

将 defineExpose 替换为：

```ts
defineExpose({
  startScreenShare,
  startCall,
  handleWebRTCOffer,
  handleWebRTCAnswer,
  handleWebRTCIceCandidate,
  handleScreenShareStart,
  handleScreenShareMessage,
  handleScreenShareRequest,
  handleScreenShareAccepted,
  handleScreenShareRejected,
  handleScreenShareStop: handleScreenShareStopGlobal,
  handleRealtimeSessionCreated,
  handleVideoCallSignaling,
  screenShareRef,
  callOverlayRef
})
```

- [ ] **步骤 8：更新 emit 定义**

移除 `'call-state-change'` 之外不需要的 emit（已存在，无需改动）。

- [ ] **步骤 9：验证编译通过**

运行：`cd /Users/gracegaoya/work/project/qim/qim-client && npx vue-tsc --noEmit 2>&1 | head -30`
预期：无与 RealtimeCommunication.vue 相关的类型错误

---

### 任务 5：从 OverlayManager.vue 移除 CallModal

**文件：**
- 修改：`src/components/chat/OverlayManager.vue`

- [ ] **步骤 1：移除 CallModal 组件使用**

删除 template 中的 CallModal 部分：

```vue
<!-- 通话模态框 -->
<CallModal
  :visible="showCallModal"
  :call-type="callType"
  :status="callStatus"
  :avatar="callAvatar"
  :name="callName"
  @reject-call="emit('reject-call')"
  @answer-call="emit('answer-call')"
  @end="emit('end-call')"
  @close="emit('close-call-modal')"
/>
```

- [ ] **步骤 2：移除 CallModal import**

删除：

```ts
import CallModal from './CallModal.vue'
```

- [ ] **步骤 3：移除 Props 中的 call 相关属性**

从 Props interface 中删除：

```ts
showCallModal: boolean
callType: 'voice' | 'video' | ''
callStatus: 'ringing' | 'answered' | 'ended' | ''
callAvatar: string
callName: string
```

- [ ] **步骤 4：移除 emit 中的 call 相关事件**

从 emit 定义中删除：

```ts
'reject-call': []
'answer-call': []
'end-call': []
'close-call-modal': []
```

- [ ] **步骤 5：验证编译通过**

运行：`cd /Users/gracegaoya/work/project/qim/qim-client && npx vue-tsc --noEmit 2>&1 | head -30`
预期：无与 OverlayManager.vue 相关的类型错误

---

### 任务 6：从 ChatWindow.vue 移除旧的 useVideoCall 集成

**文件：**
- 修改：`src/components/chat/ChatWindow.vue`

- [ ] **步骤 1：移除 useVideoCall import**

删除：

```ts
import { useVideoCall } from '../../composables/useVideoCall'
```

- [ ] **步骤 2：移除 useVideoCall 实例和解构**

删除：

```ts
const videoCall = useVideoCall()
const {
  callStatus: videoCallStatus,
  callType: videoCallType,
  localStream: videoCallLocalStream,
  remoteStream: videoCallRemoteStream,
  isMuted: videoCallIsMuted,
  isVideoEnabled: videoCallIsVideoEnabled,
  remoteUser: videoCallRemoteUser,
  incomingCall: videoCallIncomingCall,
  startCall: videoCallStart,
  answerCall: videoCallAnswer,
  endCall: videoCallEnd,
  rejectCall: videoCallReject,
  toggleMute: videoCallToggleMute,
  toggleVideo: videoCallToggleVideo,
  handleSignalingMessage: videoCallHandleSignaling
} = videoCall
```

- [ ] **步骤 3：移除 OverlayManager 中的 call 相关 props**

在 ChatWindow 的 template 中，找到 OverlayManager 的使用位置，移除以下 props：

```vue
:show-call-modal="showCallModal || videoCallStatus !== 'idle'"
:call-type="videoCallType"
:call-status="(videoCallStatus === 'idle' || videoCallStatus === 'calling') ? '' : videoCallStatus"
:call-avatar="getAvatarUrl(videoCallRemoteUser?.avatar || props.conversation?.avatar, 'user', serverUrl)"
:call-name="videoCallRemoteUser?.name || props.conversation?.name || '未知'"
```

- [ ] **步骤 4：移除 OverlayManager 中的 call 相关 events**

在 ChatWindow 的 template 中，移除 OverlayManager 的以下 events：

```vue
@reject-call="rejectCall"
@answer-call="answerCall"
@end-call="endCall"
@close-call-modal="handleCallModalClose"
```

- [ ] **步骤 5：移除 WebSocket 消息处理中的 videoCall 相关代码**

在 ChatWindow 的 WebSocket 消息处理中，找到 `videoCallMessageTypes` 和 `videoCallHandleSignaling` 相关的代码块，移除对 `videoCallHandleSignaling` 的调用和 `showCallModal.value = true` 的设置。

具体来说，将消息处理中的 video call 分支改为不再处理（因为通话信令现在由 RealtimeCommunication 层处理）。

- [ ] **步骤 6：移除 showCallModal ref 和相关方法**

删除 `showCallModal` ref 以及 `rejectCall`、`answerCall`、`endCall`、`handleCallModalClose` 等方法（如果它们仅服务于旧的 CallModal）。

- [ ] **步骤 7：验证编译通过**

运行：`cd /Users/gracegaoya/work/project/qim/qim-client && npx vue-tsc --noEmit 2>&1 | head -30`
预期：无与 ChatWindow.vue 相关的类型错误

---

### 任务 7：最终验证和清理

**文件：**
- 检查所有修改的文件

- [ ] **步骤 1：全局搜索确认 CallModal 不再被引用**

运行：`cd /Users/gracegaoya/work/project/qim/qim-client && grep -r "CallModal" src/ --include="*.vue" --include="*.ts"`

预期：仅出现在 CallModal.vue 自身和 RealtimeCommunicationOld.vue（旧文件）中

- [ ] **步骤 2：全局搜索确认旧 useVideoCall 不再被引用**

运行：`cd /Users/gracegaoya/work/project/qim/qim-client && grep -r "useVideoCall[^N]" src/ --include="*.vue" --include="*.ts"`

预期：仅出现在 useVideoCall.ts 自身和 RealtimeCommunicationOld.vue（旧文件）中

- [ ] **步骤 3：运行完整类型检查**

运行：`cd /Users/gracegaoya/work/project/qim/qim-client && npx vue-tsc --noEmit 2>&1 | head -50`

预期：无类型错误

- [ ] **步骤 4：运行 lint 检查**

运行：`cd /Users/gracegaoya/work/project/qim/qim-client && npm run lint 2>&1 | head -50`

预期：无 lint 错误（或仅有与本次改动无关的已有问题）

- [ ] **步骤 5：验证构建**

运行：`cd /Users/gracegaoya/work/project/qim/qim-client && npm run build 2>&1 | tail -20`

预期：构建成功
