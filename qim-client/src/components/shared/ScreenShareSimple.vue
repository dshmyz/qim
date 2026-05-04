<template>
  <Teleport to="body">
    <div 
      v-if="sessionState !== 'idle'"
      ref="screenShareOverlayRef"
      class="screen-share-overlay"
      :class="{ minimized: isMinimized, dragging: isDragging }"
    >
      <!-- 标题栏 -->
      <div 
        class="screen-share-header"
        @mousedown="startDrag"
        @dblclick="toggleMinimize"
      >
        <div class="header-left">
          <span class="status-dot" :class="{ active: sessionState === 'active' }"></span>
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

      <!-- 正常视图 -->
      <div v-if="!isMinimized" class="screen-share-body">
        <div class="video-container">
          <video
            ref="localVideoRef"
            v-if="localStream && isInitiator"
            autoplay
            playsinline
            muted
            class="local-video"
          ></video>
          <video
            ref="remoteVideoRef"
            v-if="remoteStream && !isInitiator"
            autoplay
            playsinline
            class="remote-video"
          ></video>
          <div v-if="!localStream && !remoteStream" class="video-placeholder">
            <i class="fas fa-spinner fa-spin"></i>
            <span>连接中...</span>
          </div>
        </div>

        <!-- 控制栏 -->
        <div class="screen-share-controls">
          <div class="duration">{{ formattedDuration }}</div>
          <div class="controls-actions">
            <button 
              v-if="isInitiator && sessionState === 'active'"
              class="control-btn"
              @click="togglePause"
              :title="isPaused ? '恢复' : '暂停'"
            >
              <i :class="isPaused ? 'fas fa-play' : 'fas fa-pause'"></i>
            </button>
            <button 
              class="control-btn stop-btn"
              @click="handleStop"
              title="停止"
            >
              <i class="fas fa-stop"></i>
              <span>停止共享</span>
            </button>
          </div>
        </div>
      </div>

      <!-- 最小化视图 -->
      <div v-else class="minimized-content" @click="toggleMinimize">
        <div class="minimized-preview">
          <video
            ref="minimizedVideoRef"
            autoplay
            playsinline
            muted
          ></video>
        </div>
        <div class="minimized-info">
          <div class="info-top">
            <div class="minimized-status">
              <span class="pulse-dot" :class="{ active: sessionState === 'active' }"></span>
              <span>{{ isInitiator ? '共享中' : '观看中' }}</span>
            </div>
            <div class="minimized-duration">{{ formattedDuration }}</div>
            <div class="minimized-name">{{ screenShareName || '屏幕共享' }}</div>
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
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useScreenShareNew } from '@/composables/useScreenShareNew'

const props = defineProps<{
  receiverId?: number
  conversationId?: number
  senderName?: string
}>()

const emit = defineEmits<{
  'screen-share-start': [data: { conversationId: string | number }]
  'screen-share-stop': []
}>()

const screenShare = useScreenShareNew()

const screenShareOverlayRef = ref<HTMLElement | null>(null)
const localVideoRef = ref<HTMLVideoElement | null>(null)
const remoteVideoRef = ref<HTMLVideoElement | null>(null)
const minimizedVideoRef = ref<HTMLVideoElement | null>(null)

const isMinimized = ref(false)
const isDragging = ref(false)
const dragState = ref({
  startX: 0,
  startY: 0,
  elementX: 0,
  elementY: 0
})
let rafId: number | null = null

const duration = ref(0)
let durationTimer: number | null = null

const screenShareName = ref('')

const sessionState = computed(() => screenShare.sessionState.value)
const localStream = computed(() => screenShare.localStream.value)
const remoteStream = computed(() => screenShare.remoteStream.value)
const isPaused = computed(() => screenShare.isPaused.value)

const isInitiator = computed(() => {
  return screenShare.participants.value.some(p => p.role === 'receiver')
})

const title = computed(() => {
  if (isInitiator.value) {
    return '正在共享屏幕'
  }
  return `${props.senderName || '对方'}的屏幕共享`
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

const startDrag = (e: MouseEvent) => {
  e.preventDefault()
  const element = screenShareOverlayRef.value
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
  
  const element = screenShareOverlayRef.value
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
  
  if (isMinimized.value) {
    nextTick(() => {
      if (minimizedVideoRef.value && remoteStream.value) {
        minimizedVideoRef.value.srcObject = remoteStream.value
        minimizedVideoRef.value.play().catch(err => {
          if (err.name !== 'AbortError') {
            console.error('最小化视频播放失败:', err)
          }
        })
      }
    })
  }
}

const handleStop = () => {
  screenShare.stopSharing()
  emit('screen-share-stop')
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

watch(localStream, (stream) => {
  if (stream && localVideoRef.value) {
    localVideoRef.value.srcObject = stream
  }
})

watch(remoteStream, (stream) => {
  if (stream && remoteVideoRef.value) {
    remoteVideoRef.value.srcObject = stream
  }
})

watch(sessionState, (state) => {
  if (state === 'active') {
    startDurationTimer()
    emit('screen-share-start', { conversationId: props.conversationId || props.receiverId || 0 })
  } else if (state === 'idle' || state === 'ended') {
    stopDurationTimer()
  }
})

watch(isDragging, (dragging) => {
  if (screenShareOverlayRef.value) {
    if (dragging) {
      screenShareOverlayRef.value.classList.add('dragging')
    } else {
      screenShareOverlayRef.value.classList.remove('dragging')
    }
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
  startSharing: screenShare.startSharing,
  acceptShare: screenShare.acceptShare,
  rejectShare: screenShare.rejectShare,
  handleAnswer: screenShare.handleAnswer,
  handleIceCandidate: screenShare.handleIceCandidate,
  sessionState,
  localStream,
  remoteStream
})
</script>

<style scoped>
.screen-share-overlay {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 640px;
  background: rgba(30, 30, 30, 0.95);
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  z-index: 10000;
  overflow: hidden;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.screen-share-overlay.minimized {
  width: 320px;
  height: auto;
  min-height: 140px;
}

.screen-share-overlay.dragging {
  cursor: grabbing;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.5);
  transition: box-shadow 0.2s ease-out;
}

.screen-share-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.1) 0%, rgba(255, 255, 255, 0.05) 100%);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  cursor: grab;
  user-select: none;
}

.screen-share-header:active {
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

.screen-share-body {
  display: flex;
  flex-direction: column;
}

.video-container {
  position: relative;
  width: 100%;
  height: 360px;
  background: #000;
}

.local-video,
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

.screen-share-controls {
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

.minimized-preview {
  width: 120px;
  height: 68px;
  border-radius: 8px;
  overflow: hidden;
  background: rgba(0, 0, 0, 0.4);
  flex-shrink: 0;
}

.minimized-preview video {
  width: 100%;
  height: 100%;
  object-fit: cover;
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

.minimized-name {
  color: rgba(255, 255, 255, 0.6);
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
