<template>
  <Teleport to="body">
    <div
      v-if="showOverlay"
      ref="callOverlayRef"
      class="call-overlay"
      :class="{ minimized: isMinimized, dragging: isDragging }"
    >
      <div
        class="call-header"
        @mousedown="startDrag"
        @dblclick="toggleMinimize"
      >
        <div class="header-left">
          <span class="status-dot" :class="statusDotClass"></span>
          <span class="title">{{ headerTitle }}</span>
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
            @click="handleEndCall"
            title="关闭"
          >
            <i class="fas fa-times"></i>
          </button>
        </div>
      </div>

      <!-- 展开状态 -->
      <div v-if="!isMinimized" class="call-body">
        <!-- 来电邀请 -->
        <div v-if="showIncomingCall" class="incoming-call">
          <div class="incoming-icon" :class="callTypeClass">
            <i :class="callTypeIcon"></i>
          </div>
          <div class="incoming-text">
            <span class="incoming-title">{{ callerName }}</span>
            <span class="incoming-subtitle">{{ incomingCallLabel }}</span>
          </div>
          <div class="incoming-actions">
            <button class="btn-reject" @click="handleReject">
              <i class="fas fa-phone-slash"></i>
              <span>拒绝</span>
            </button>
            <button class="btn-accept" :class="callTypeClass" @click="handleAccept">
              <i :class="callTypeIcon"></i>
              <span>接听</span>
            </button>
          </div>
        </div>

        <!-- 呼叫等待 -->
        <div v-else-if="showOutgoingCall" class="outgoing-call" :class="callTypeClass">
          <div class="outgoing-icon" :class="callTypeClass">
            <i :class="callTypeIcon"></i>
          </div>
          <div class="calling-animation">
            <span class="calling-ring ring-1"></span>
            <span class="calling-ring ring-2"></span>
            <span class="calling-ring ring-3"></span>
          </div>
          <div class="outgoing-text">
            <span class="outgoing-name">{{ callerName }}</span>
            <span class="outgoing-status">正在呼叫...</span>
          </div>
          <button class="btn-cancel-call" @click="handleEndCall">
            <i class="fas fa-phone-slash"></i>
            <span>取消</span>
          </button>
        </div>

        <!-- 连接中 -->
        <div v-else-if="callStatus === 'connecting'" class="connecting-state">
          <div class="connecting-icon">
            <i class="fas fa-spinner fa-spin"></i>
          </div>
          <span class="connecting-text">连接中...</span>
        </div>

        <!-- 通话中 -->
        <div v-else-if="callStatus === 'connected'" class="active-call">
          <!-- 视频通话 -->
          <div v-if="isVideoCall" class="video-call-view">
            <video
              ref="remoteVideoRef"
              v-show="remoteStream"
              autoplay
              playsinline
              class="remote-video"
            ></video>
            <div v-if="!remoteStream" class="video-placeholder">
              <i class="fas fa-user-circle"></i>
              <span>等待视频...</span>
            </div>
            <div
              v-if="localStream && isCameraEnabled"
              class="local-video-pip"
              :class="{ 'pip-hidden': !showLocalPip }"
              @click="toggleLocalPip"
            >
              <video
                ref="localVideoRef"
                autoplay
                playsinline
                muted
                class="local-video"
              ></video>
            </div>
            <button
              v-if="localStream && isCameraEnabled && showLocalPip"
              class="pip-toggle-btn"
              @click="toggleLocalPip"
              title="隐藏本地视频"
            >
              <i class="fas fa-eye-slash"></i>
            </button>
          </div>

          <!-- 语音通话 -->
          <div v-else class="voice-call-view">
            <div class="voice-avatar">
              <Avatar :name="callerName" size="xl" />
            </div>
            <span class="voice-name">{{ callerName }}</span>
            <span class="voice-duration">{{ formattedDuration }}</span>
          </div>
        </div>
      </div>

      <!-- 最小化状态 -->
      <div v-if="isMinimized" class="minimized-content" @click="toggleMinimize">
        <div class="minimized-icon" :class="callTypeClass">
          <i :class="callTypeIcon"></i>
        </div>
        <div class="minimized-info">
          <div class="info-top">
            <div class="minimized-status">
              <span class="pulse-dot" :class="{ active: callStatus === 'connected' }"></span>
              <span>{{ minimizedStatusLabel }}</span>
            </div>
            <div class="minimized-duration">{{ formattedDuration }}</div>
            <div class="minimized-name">{{ callerName }}</div>
          </div>
          <div class="minimized-actions" @click.stop>
            <button class="action-btn expand-btn" @click="toggleMinimize">
              <i class="fas fa-expand"></i>
              <span>展开</span>
            </button>
            <button class="action-btn close-btn" @click="handleEndCall">
              <i class="fas fa-phone-slash"></i>
              <span>挂断</span>
            </button>
          </div>
        </div>
      </div>

      <!-- 控制栏（展开且通话中/连接中时显示） -->
      <div
        v-if="!isMinimized && (callStatus === 'connected' || callStatus === 'connecting')"
        class="call-controls"
      >
        <div class="duration">{{ formattedDuration }}</div>
        <div class="controls-actions">
          <button
            class="control-btn"
            :class="{ active: !isMicrophoneEnabled }"
            @click="handleToggleMute"
            :title="isMicrophoneEnabled ? '静音' : '取消静音'"
          >
            <i :class="isMicrophoneEnabled ? 'fas fa-microphone' : 'fas fa-microphone-slash'"></i>
          </button>
          <button
            v-if="isVideoCall"
            class="control-btn"
            :class="{ active: !isCameraEnabled }"
            @click="handleToggleCamera"
            :title="isCameraEnabled ? '关闭摄像头' : '开启摄像头'"
          >
            <i :class="isCameraEnabled ? 'fas fa-video' : 'fas fa-video-slash'"></i>
          </button>
          <button
            class="control-btn end-btn"
            @click="handleEndCall"
            title="结束通话"
          >
            <i class="fas fa-phone-slash"></i>
            <span>挂断</span>
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted, nextTick } from 'vue'
import { useVideoCallNew } from '@/composables/useVideoCallNew'
import Avatar from '@/components/shared/Avatar.vue'
import QMessage from '@/utils/qmessage'

interface Props {
  receiverId?: number
  conversationId?: number
  senderName?: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'call-start': [data: { conversationId: number }]
  'call-stop': []
}>()

const videoCall = useVideoCallNew()

// Refs
const callOverlayRef = ref<HTMLElement | null>(null)
const localVideoRef = ref<HTMLVideoElement | null>(null)
const remoteVideoRef = ref<HTMLVideoElement | null>(null)

// UI 状态
const isMinimized = ref(false)
const isDragging = ref(false)
const showLocalPip = ref(true)
const dragState = ref({
  startX: 0,
  startY: 0,
  elementX: 0,
  elementY: 0
})
let rafId: number | null = null

// 计时
const duration = ref(0)
let durationTimer: number | null = null

// Composable 属性
const callType = computed(() => videoCall.callType.value)
const callStatus = computed(() => videoCall.callStatus.value)
const localStream = computed(() => videoCall.localStream.value)
const remoteStream = computed(() => videoCall.remoteStream.value)

watch(videoCall.remoteStream, (stream) => {
  console.log('[CallOverlay] videoCall.remoteStream watch triggered:', stream)
}, { immediate: true })
const isCameraEnabled = computed(() => videoCall.isCameraEnabled.value)
const isMicrophoneEnabled = computed(() => videoCall.isMicrophoneEnabled.value)
const pendingOffer = computed(() => videoCall.pendingOffer.value)

// 计算属性
const isVideoCall = computed(() => callType.value === 'video')

const showIncomingCall = computed(() => {
  return callStatus.value === 'calling' && pendingOffer.value !== null
})

const showOutgoingCall = computed(() => {
  return callStatus.value === 'calling' && pendingOffer.value === null
})

const showOverlay = computed(() => {
  return showIncomingCall.value || showOutgoingCall.value || callStatus.value === 'connecting' || callStatus.value === 'connected'
})

const callerName = computed(() => {
  return props.senderName || '对方'
})

const callTypeClass = computed(() => {
  return isVideoCall.value ? 'video-type' : 'voice-type'
})

const callTypeIcon = computed(() => {
  return isVideoCall.value ? 'fas fa-video' : 'fas fa-phone'
})

const incomingCallLabel = computed(() => {
  return isVideoCall.value ? '邀请你进行视频通话' : '邀请你进行语音通话'
})

const statusDotClass = computed(() => {
  if (callStatus.value === 'connected') return 'active'
  if (callStatus.value === 'connecting') return 'connecting'
  return ''
})

const headerTitle = computed(() => {
  if (showIncomingCall.value) return '来电'
  if (showOutgoingCall.value) return '呼叫中'
  if (callStatus.value === 'connecting') return '连接中'
  if (callStatus.value === 'connected') {
    return isVideoCall.value ? '视频通话' : '语音通话'
  }
  return '通话'
})

const minimizedStatusLabel = computed(() => {
  if (callStatus.value === 'connected') {
    return isVideoCall.value ? '视频通话中' : '语音通话中'
  }
  if (callStatus.value === 'connecting') return '连接中'
  return '呼叫中'
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

// 拖拽（与 ScreenShareSimple 一致的实现）
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

// 最小化/展开
const toggleMinimize = () => {
  isMinimized.value = !isMinimized.value
}

// 本地视频 PiP 切换
const toggleLocalPip = () => {
  showLocalPip.value = !showLocalPip.value
}

// 计时器
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

// 通话操作
const handleAccept = async () => {
  try {
    await videoCall.acceptCall()
  } catch (error) {
    console.error('[CallOverlay] 接听失败:', error)
  }
}

const handleReject = () => {
  videoCall.rejectCall()
  emit('call-stop')
}

const handleEndCall = () => {
  videoCall.endCall()
  emit('call-stop')
}

const handleRemoteEndCall = () => {
  console.log('[CallOverlay] handleRemoteEndCall 被调用')
  console.log('[CallOverlay] 当前 callStatus:', videoCall.callStatus.value)
  console.log('[CallOverlay] 当前 showOverlay:', showOverlay.value)
  
  videoCall.handleRemoteEndCall()
  emit('call-stop')
  
  console.log('[CallOverlay] handleRemoteEndCall 执行完成')
}

const handleToggleMute = () => {
  videoCall.toggleMute()
}

const handleToggleCamera = () => {
  videoCall.toggleCamera()
}

// 暴露方法
const initiateCall = async (callTypeParam: 'voice' | 'video') => {
  if (!props.receiverId) {
    QMessage.warning('无法发起通话，请先选择联系人')
    return
  }
  
  try {
    await videoCall.startCall(props.receiverId, callTypeParam)
  } catch (error: any) {
    console.error('[CallOverlay] 发起通话失败:', error)
    
    if (error.message && !error.message.includes('当前正在通话中')) {
      QMessage.warning(error.message)
    } else if (error.message === '当前正在通话中') {
      QMessage.info('您正在通话中，请先结束当前通话')
    } else {
      QMessage.error('发起通话失败，请稍后重试')
    }
  }
}

const handleIncomingOffer = (signal: RTCSessionDescriptionInit, fromUserId: number, type: 'voice' | 'video') => {
  videoCall.handleIncomingCall(signal, fromUserId, type)
}

const showIncomingCallUI = (_fromUserId: number, type: 'voice' | 'video') => {
  videoCall.setIncomingCallType(type)
}

// 视频流绑定
watch(localStream, (stream) => {
  if (stream) {
    nextTick(() => {
      if (localVideoRef.value && localVideoRef.value.srcObject !== stream) {
        localVideoRef.value.srcObject = stream
      }
    })
  }
}, { immediate: true })

watch(remoteStream, (stream) => {
  if (stream && remoteVideoRef.value) {
    console.log('[CallOverlay] Setting remote stream to video element')
    remoteVideoRef.value.srcObject = stream
    remoteVideoRef.value.play().catch((e) => {
      console.error('[CallOverlay] Failed to play remote video:', e)
    })
  }
}, { immediate: true })

watch(remoteVideoRef, (videoEl) => {
  if (videoEl && remoteStream.value) {
    console.log('[CallOverlay] Video element available, setting stream')
    videoEl.srcObject = remoteStream.value
    videoEl.play().catch((e) => {
      console.error('[CallOverlay] Failed to play remote video:', e)
    })
  }
})

// callStatus watch — connected 时启动计时器并 emit call-start，idle 时停止计时器
watch(callStatus, (status) => {
  if (status === 'connected') {
    startDurationTimer()
    emit('call-start', { conversationId: props.conversationId || 0 })
  } else if (status === 'idle') {
    stopDurationTimer()
  }
})

// 清理
onUnmounted(() => {
  stopDurationTimer()
  if (rafId) {
    cancelAnimationFrame(rafId)
  }
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
  
  videoCall.endCall()
})

defineExpose({
  initiateCall,
  handleIncomingOffer,
  showIncomingCallUI,
  handleEndCall,
  handleRemoteEndCall
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
  width: 300px;
  height: auto;
  min-height: 100px;
}

.call-overlay.dragging {
  cursor: grabbing;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.5);
  transition: box-shadow 0.2s ease-out;
}

/* Header */
.call-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.1) 0%, rgba(255, 255, 255, 0.05) 100%);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  cursor: grab;
  user-select: none;
}

.call-header:active {
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

.status-dot.connecting {
  background: #f59e0b;
  animation: pulse 1s infinite;
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

/* Body */
.call-body {
  display: flex;
  flex-direction: column;
}

/* 来电邀请 */
.incoming-call {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 28px 24px;
  gap: 14px;
}

.incoming-icon {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  color: #fff;
}

.incoming-icon.video-type {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
}

.incoming-icon.voice-type {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
}

.incoming-text {
  text-align: center;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.incoming-title {
  color: #fff;
  font-size: 16px;
  font-weight: 600;
}

.incoming-subtitle {
  color: rgba(255, 255, 255, 0.6);
  font-size: 14px;
}

.incoming-actions {
  display: flex;
  gap: 16px;
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
  display: flex;
  align-items: center;
  gap: 6px;
}

.btn-reject:hover {
  background: rgba(239, 68, 68, 0.3);
}

.btn-accept {
  padding: 10px 24px;
  border: none;
  border-radius: 8px;
  color: #fff;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 6px;
}

.btn-accept.video-type {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
}

.btn-accept.video-type:hover {
  background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
}

.btn-accept.voice-type {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
}

.btn-accept.voice-type:hover {
  background: linear-gradient(135deg, #059669 0%, #047857 100%);
}

/* 呼叫等待 */
.outgoing-call {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 32px 24px;
  gap: 14px;
  position: relative;
}

.outgoing-icon {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  color: #fff;
  position: relative;
  z-index: 1;
}

.outgoing-icon.video-type {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
}

.outgoing-icon.voice-type {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
}

.calling-animation {
  position: absolute;
  top: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.calling-ring {
  position: absolute;
  border-radius: 50%;
  border: 2px solid rgba(255, 255, 255, 0.2);
  animation: ring-expand 2s infinite;
}

.calling-ring.ring-1 {
  width: 60px;
  height: 60px;
  animation-delay: 0s;
}

.calling-ring.ring-2 {
  width: 60px;
  height: 60px;
  animation-delay: 0.5s;
}

.calling-ring.ring-3 {
  width: 60px;
  height: 60px;
  animation-delay: 1s;
}

.video-type .calling-ring {
  border-color: rgba(59, 130, 246, 0.3);
}

.voice-type .calling-ring {
  border-color: rgba(16, 185, 129, 0.3);
}

@keyframes ring-expand {
  0% {
    transform: scale(1);
    opacity: 0.6;
  }
  100% {
    transform: scale(2.2);
    opacity: 0;
  }
}

.outgoing-text {
  text-align: center;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.outgoing-name {
  color: #fff;
  font-size: 16px;
  font-weight: 600;
}

.outgoing-status {
  color: rgba(255, 255, 255, 0.6);
  font-size: 14px;
}

.btn-cancel-call {
  padding: 10px 24px;
  background: rgba(239, 68, 68, 0.2);
  border: 1px solid rgba(239, 68, 68, 0.5);
  border-radius: 8px;
  color: #ef4444;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 4px;
}

.btn-cancel-call:hover {
  background: rgba(239, 68, 68, 0.3);
}

/* 连接中 */
.connecting-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
  gap: 12px;
}

.connecting-icon {
  font-size: 32px;
  color: #f59e0b;
}

.connecting-text {
  color: rgba(255, 255, 255, 0.7);
  font-size: 15px;
}

/* 通话中 — 视频 */
.active-call {
  position: relative;
}

.video-call-view {
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
  gap: 10px;
  color: rgba(255, 255, 255, 0.4);
}

.video-placeholder i {
  font-size: 48px;
}

.video-placeholder span {
  font-size: 14px;
}

.local-video-pip {
  position: absolute;
  bottom: 12px;
  right: 12px;
  width: 120px;
  height: 90px;
  border-radius: 8px;
  overflow: hidden;
  border: 2px solid rgba(255, 255, 255, 0.3);
  cursor: pointer;
  transition: all 0.3s ease;
  z-index: 2;
}

.local-video-pip:hover {
  border-color: rgba(255, 255, 255, 0.6);
  transform: scale(1.05);
}

.local-video-pip.pip-hidden {
  width: 0;
  height: 0;
  border: none;
  overflow: hidden;
  opacity: 0;
}

.local-video {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transform: scaleX(-1);
}

.pip-toggle-btn {
  position: absolute;
  bottom: 12px;
  right: 140px;
  padding: 6px 8px;
  background: rgba(0, 0, 0, 0.5);
  border: none;
  border-radius: 4px;
  color: rgba(255, 255, 255, 0.7);
  cursor: pointer;
  font-size: 12px;
  z-index: 2;
  transition: all 0.2s;
}

.pip-toggle-btn:hover {
  background: rgba(0, 0, 0, 0.7);
  color: #fff;
}

/* 通话中 — 语音 */
.voice-call-view {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 36px 24px 24px;
  gap: 12px;
}

.voice-avatar {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  overflow: hidden;
}

.voice-name {
  color: #fff;
  font-size: 18px;
  font-weight: 600;
}

.voice-duration {
  color: rgba(255, 255, 255, 0.6);
  font-size: 14px;
  font-family: 'SF Mono', Monaco, monospace;
}

/* 控制栏 */
.call-controls {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
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
  padding: 8px 14px;
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

.control-btn.active {
  background: rgba(239, 68, 68, 0.3);
  color: #ef4444;
}

.end-btn {
  background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
}

.end-btn:hover {
  background: linear-gradient(135deg, #dc2626 0%, #b91c1c 100%);
}

/* 最小化状态 */
.minimized-content {
  display: flex;
  padding: 10px;
  gap: 10px;
  cursor: pointer;
  transition: background 0.2s;
}

.minimized-content:hover {
  background: rgba(255, 255, 255, 0.05);
}

.minimized-icon {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  color: #fff;
  flex-shrink: 0;
}

.minimized-icon.video-type {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
}

.minimized-icon.voice-type {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
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
  gap: 2px;
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

.minimized-name {
  color: rgba(255, 255, 255, 0.6);
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.minimized-actions {
  display: flex;
  gap: 6px;
}

.minimized-actions .action-btn {
  flex: 1;
  padding: 5px 8px;
  font-size: 11px;
  justify-content: center;
}

.expand-btn {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
}

.expand-btn:hover {
  background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
}
</style>
