<template>
  <Teleport to="body">
    <div
      v-if="showOverlay"
      ref="screenShareOverlayRef"
      class="screen-share-overlay"
      :class="{ minimized: isMinimized, dragging: isDragging, fullscreen: isFullscreen }"
    >
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

      <div v-if="!isMinimized" class="screen-share-body">
        <div v-if="showSourcePicker && isInitiator" class="source-picker">
          <div class="source-picker-header">
            <h3>选择共享来源</h3>
            <button class="close-btn" @click="cancelShare"><i class="fas fa-times"></i></button>
          </div>

          <div class="source-picker-body">
            <template v-if="screenSourceList.length > 0">
              <div class="source-group-title"><i class="fas fa-desktop"></i> 整个屏幕</div>
              <div class="source-grid">
                <div
                  v-for="source in screenSourceList"
                  :key="source.id"
                  class="source-item"
                  :class="{ selected: selectedSource?.id === source.id }"
                  @click="selectedSource = source"
                  @dblclick="selectedSource = source; confirmShare()"
                >
                  <div class="source-thumbnail">
                    <img v-if="source.thumbnail" :src="source.thumbnail" :alt="source.name" />
                    <div v-else class="source-placeholder"><i class="fas fa-desktop"></i></div>
                  </div>
                  <div class="source-name">{{ source.name }}</div>
                </div>
              </div>
            </template>

            <template v-if="windowSourceList.length > 0">
              <div class="source-group-title"><i class="far fa-window-restore"></i> 应用窗口</div>
              <div class="source-grid">
                <div
                  v-for="source in windowSourceList"
                  :key="source.id"
                  class="source-item"
                  :class="{ selected: selectedSource?.id === source.id }"
                  @click="selectedSource = source"
                  @dblclick="selectedSource = source; confirmShare()"
                >
                  <div class="source-thumbnail">
                    <img v-if="source.thumbnail" :src="source.thumbnail" :alt="source.name" />
                    <div v-else class="source-placeholder"><i class="far fa-window-restore"></i></div>
                  </div>
                  <div class="source-name">{{ source.name }}</div>
                </div>
              </div>
            </template>

            <div v-if="screenSourceList.length === 0 && windowSourceList.length === 0" class="source-empty">
              <i class="fas fa-desktop"></i>
              <span>未检测到可共享的来源</span>
            </div>
          </div>

          <div class="source-picker-footer">
            <span class="source-hint" v-if="selectedSource">
              <i class="fas fa-check-circle"></i> 已选择：{{ selectedSource.name }}
            </span>
            <span class="source-hint" v-else>选择要共享的屏幕或窗口</span>
            <div class="source-picker-actions">
              <button class="btn-secondary" @click="cancelShare">取消</button>
              <button class="btn-primary" @click="confirmShare" :disabled="!selectedSource">
                <i class="fas fa-share"></i> 开始共享
              </button>
            </div>
          </div>
        </div>

        <div v-else-if="showIncomingRequest" class="incoming-request">
          <div class="incoming-icon">
            <i class="fas fa-desktop"></i>
          </div>
          <div class="incoming-text">
            <span class="incoming-title">{{ incomingRequestInfo?.fromUserName || props.senderName || '对方' }} 邀请你观看屏幕共享</span>
          </div>
          <div class="incoming-actions">
            <button class="btn-reject" @click="handleReject">拒绝</button>
            <button class="btn-accept" @click="handleAccept">接受</button>
          </div>
        </div>

        <div v-else-if="showWaitingAccept" class="waiting-accept">
          <div class="waiting-icon">
            <i class="fas fa-clock"></i>
          </div>
          <div class="waiting-text">
            <div class="waiting-title">正在等待 <strong>{{ props.senderName || '对方' }}</strong> 接受</div>
            <div class="waiting-sub">对方将收到屏幕共享邀请</div>
          </div>
          <div class="waiting-countdown" :class="{ urgent: waitingCountdown <= 10 }">
            <i class="fas fa-hourglass-half"></i>
            <span>{{ waitingCountdown }}s 后自动取消</span>
          </div>
          <button class="btn-secondary" @click="handleStop">取消共享</button>
        </div>

        <div v-else class="video-container" :class="{ 'fullscreen-mode': isFullscreen }">
          <video
            ref="localVideoRef"
            v-show="localStream && isInitiator"
            autoplay
            playsinline
            muted
            class="local-video"
          ></video>
          <video
            ref="remoteVideoRef"
            v-show="remoteStream && !isInitiator"
            autoplay
            playsinline
            class="remote-video"
          ></video>
          <div v-if="!localStream && !remoteStream && !showSourcePicker && !showWaitingAccept" class="video-placeholder">
            <i class="fas fa-spinner fa-spin"></i>
            <span>连接中...</span>
          </div>

          <!-- 暂停提示（发起方暂停时，接收方看到冻结画面） -->
          <div v-if="isInitiator && isPaused && localStream" class="pause-overlay">
            <div class="pause-overlay__inner">
              <i class="fas fa-pause-circle"></i>
              <span class="pause-overlay__title">共享已暂停</span>
              <span class="pause-overlay__desc">对方当前看到的是冻结画面</span>
            </div>
          </div>
          
          <div v-if="isFullscreen" class="fullscreen-header">
            <span class="fullscreen-title">{{ title }}</span>
            <button class="fullscreen-close-btn" @click="exitFullscreen" title="退出全屏">
              <i class="fas fa-times"></i>
            </button>
          </div>
          
          <div v-if="isFullscreen" class="fullscreen-controls">
            <div class="fullscreen-duration">{{ formattedDuration }}</div>
            <div class="fullscreen-actions">
              <button
                v-if="isInitiator && sessionState === 'active'"
                class="fullscreen-btn"
                @click="togglePause"
                :title="isPaused ? '恢复' : '暂停'"
              >
                <i :class="isPaused ? 'fas fa-play' : 'fas fa-pause'"></i>
              </button>
              <button
                class="fullscreen-btn"
                @click="toggleFullscreen"
                title="退出全屏"
              >
                <i class="fas fa-compress"></i>
              </button>
              <button
                class="fullscreen-btn stop-btn"
                @click="handleStop"
                title="停止共享"
              >
                <i class="fas fa-stop"></i>
              </button>
            </div>
          </div>
        </div>

        <div v-if="!showSourcePicker && !showIncomingRequest && !showWaitingAccept" class="screen-share-controls">
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
              v-if="isInitiator && sessionState === 'active'"
              class="control-btn"
              @click="switchSource"
              title="切换窗口"
            >
              <i class="fas fa-exchange-alt"></i>
            </button>
            <button
              v-if="!isInitiator"
              class="control-btn"
              @click="toggleFullscreen"
              :title="isFullscreen ? '退出全屏' : '全屏'"
            >
              <i :class="isFullscreen ? 'fas fa-compress' : 'fas fa-expand'"></i>
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

      <div v-if="isMinimized" class="minimized-content" @click="toggleMinimize">
        <div class="minimized-preview">
          <video
            ref="minimizedVideoRef"
            autoplay
            playsinline
            muted
          ></video>
          <div v-if="isInitiator && isPaused" class="minimized-paused-badge">
            <i class="fas fa-pause"></i> 已暂停
          </div>
        </div>
        <div class="minimized-info">
          <div class="info-top">
            <div class="minimized-status" :class="{ paused: isInitiator && isPaused }">
              <span class="pulse-dot" :class="{ active: sessionState === 'active' && !(isInitiator && isPaused) }"></span>
              <span>{{ isInitiator ? (isPaused ? '已暂停' : '共享中') : '观看中' }}</span>
            </div>
            <div class="minimized-duration">{{ formattedDuration }}</div>
            <div class="minimized-name">{{ screenShareName || '屏幕共享' }}</div>
          </div>
          <div class="minimized-actions" @click.stop>
            <button v-if="!isInitiator" class="action-btn expand-btn" @click="toggleMinimize">
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
import { useScreenShareNew } from '@/composables/useScreenShareNew'
import QMessage from '@/utils/qmessage'

interface ScreenSource {
  id: string
  name: string
  thumbnail?: string
}

const props = defineProps<{
  receiverId?: number
  conversationId?: number
  senderName?: string
}>()

const emit = defineEmits<{
  'screen-share-start': [data: { conversationId: string | number }]
  'screen-share-stop': []
}>()

const injectedScreenShare = inject<any>('screenShare')
const screenShare = injectedScreenShare || useScreenShareNew()

const screenShareOverlayRef = ref<HTMLElement | null>(null)
const localVideoRef = ref<HTMLVideoElement | null>(null)
const remoteVideoRef = ref<HTMLVideoElement | null>(null)
const minimizedVideoRef = ref<HTMLVideoElement | null>(null)

const isMinimized = ref(false)
const isDragging = ref(false)
const isFullscreen = ref(false)
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
const showSourcePicker = ref(false)
const screenSources = ref<ScreenSource[]>([])
const selectedSource = ref<ScreenSource | null>(null)
const showIncomingRequest = ref(false)
const showWaitingAccept = ref(false)
const pendingOffer = ref<{ signal: RTCSessionDescriptionInit; fromUserId: number } | null>(null)
const incomingRequestInfo = ref<{ fromUserId: number; conversationId: number; fromUserName: string } | null>(null)
const requestAccepted = ref(false)
const savedStream = ref<MediaStream | null>(null)

/* ---- 等待接受倒计时 ---- */
const WAITING_TIMEOUT = 60 // 秒
const waitingCountdown = ref(WAITING_TIMEOUT)
let waitingTimer: number | null = null

/* ---- 选源分组 ---- */
const screenSourceList = computed(() =>
  screenSources.value.filter(s => String(s.id).startsWith('screen:'))
)
const windowSourceList = computed(() =>
  screenSources.value.filter(s => String(s.id).startsWith('window:'))
)

const sessionState = computed(() => screenShare.sessionState.value)
const localStream = computed(() => screenShare.localStream.value)
const remoteStream = computed(() => screenShare.remoteStream.value)
const isPaused = computed(() => screenShare.isPaused.value)

const isInitiator = computed(() => {
  if (showSourcePicker.value || showWaitingAccept.value) {
    return true
  }
  return screenShare.participants.value.some((p: any) => p.role === 'receiver')
})

const showOverlay = computed(() => {
  return sessionState.value !== 'idle' || showSourcePicker.value || showIncomingRequest.value || showWaitingAccept.value
})

const togglePause = () => {
  screenShare.togglePause()
}

const title = computed(() => {
  if (showIncomingRequest.value) {
    return '屏幕共享邀请'
  }
  if (showSourcePicker.value) {
    return '选择共享来源'
  }
  if (showWaitingAccept.value) {
    return '等待对方接受...'
  }
  if (isInitiator.value) {
    return isPaused.value ? '共享已暂停' : '正在共享屏幕'
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

const getScreenSources = async (): Promise<ScreenSource[]> => {
  const electron = window.electron as any
  if (electron?.ipcRenderer) {
    try {
      console.log('[ScreenShareSimple] 请求屏幕源...')
      electron.ipcRenderer.send('start-screen-share')

      const sources = await new Promise<any[]>((resolve, reject) => {
        electron.ipcRenderer.once('screen-sources', (_event: any, sources: any[]) => {
          console.log('[ScreenShareSimple] 收到屏幕源:', sources?.length || 0)
          resolve(sources)
        })

        setTimeout(() => reject(new Error('获取屏幕源超时')), 10000)
      })

      if (Array.isArray(sources) && sources.length > 0) {
        return sources.map((source: any) => ({
          id: source.id,
          name: source.name,
          thumbnail: source.thumbnail
        }))
      }
    } catch (error) {
      console.error('[ScreenShareSimple] 获取屏幕源失败:', error)
    }
  } else {
    console.log('[ScreenShareSimple] 非 Electron 环境，无法获取屏幕源列表')
  }
  return []
}

const initiateShare = async () => {
  try {
    const sources = await getScreenSources()
    if (sources.length > 0) {
      screenSources.value = sources
      showSourcePicker.value = true
    } else {
      const electron = window.electron as any
      if (electron?.ipcRenderer) {
        console.warn('[ScreenShareSimple] Electron 环境但未获取到屏幕源，重试一次')
        const retrySources = await getScreenSources()
        if (retrySources.length > 0) {
          screenSources.value = retrySources
          showSourcePicker.value = true
          return
        }
        console.error('[ScreenShareSimple] 重试后仍未获取到屏幕源')
        showSourcePicker.value = false
        return
      }

      console.log('[ScreenShareSimple] 非 Electron 环境，使用 getDisplayMedia')
      try {
        const stream = await navigator.mediaDevices.getDisplayMedia({
          video: true,
          audio: false
        })
        if (stream) {
          savedStream.value = stream
          if (props.receiverId && props.conversationId) {
            screenShare.sendRequest(props.receiverId, props.conversationId, stream)
          }
          showWaitingAccept.value = true
          screenShareName.value = '屏幕共享'
        }
      } catch (getDisplayError) {
        console.error('[ScreenShareSimple] getDisplayMedia 失败:', getDisplayError)
      }
    }
  } catch (error) {
    console.error('[ScreenShareSimple] 初始化共享失败:', error)
    showSourcePicker.value = false
  }
}

const confirmShare = async () => {
  if (!selectedSource.value) return

  try {
    showSourcePicker.value = false
    screenShareName.value = selectedSource.value.name

    const stream = await navigator.mediaDevices.getUserMedia({
      audio: false,
      video: {
        mandatory: {
          chromeMediaSource: 'desktop',
          chromeMediaSourceId: selectedSource.value.id
        }
      } as any
    })

    savedStream.value = stream

    if (props.receiverId && props.conversationId) {
      screenShare.sendRequest(props.receiverId, props.conversationId, stream)
    }
    showWaitingAccept.value = true
  } catch (error) {
    console.error('[ScreenShareSimple] 开始共享失败:', error)
    showSourcePicker.value = true
  }
}

const cancelShare = () => {
  showSourcePicker.value = false
  selectedSource.value = null
}

const switchSource = async () => {
  const sources = await getScreenSources()
  if (sources.length > 0) {
    screenSources.value = sources
    showSourcePicker.value = true
  }
}

const handleAccept = async () => {
  showIncomingRequest.value = false

  if (pendingOffer.value) {
    if (screenShare.sessionState.value !== 'idle') {
      console.log('[ScreenShareSimple] Session not idle, cannot accept offer')
      pendingOffer.value = null
      incomingRequestInfo.value = null
      return
    }
    try {
      const convId = incomingRequestInfo.value?.conversationId || props.conversationId
      await screenShare.acceptShare(pendingOffer.value.signal, pendingOffer.value.fromUserId, convId)
    } catch (error) {
      console.error('[ScreenShareSimple] 接受屏幕共享失败:', error)
    }
    pendingOffer.value = null
    incomingRequestInfo.value = null
  } else if (incomingRequestInfo.value) {
    const convId = incomingRequestInfo.value.conversationId
    const fromUserId = incomingRequestInfo.value.fromUserId
    screenShare.acceptRequest(fromUserId, convId)
    requestAccepted.value = true
  }
}

const handleReject = () => {
  showIncomingRequest.value = false

  if (incomingRequestInfo.value) {
    const convId = incomingRequestInfo.value.conversationId
    const fromUserId = incomingRequestInfo.value.fromUserId
    screenShare.rejectRequest(fromUserId, convId)
  }

  pendingOffer.value = null
  incomingRequestInfo.value = null
  requestAccepted.value = false
}

const handleIncomingOffer = (signal: RTCSessionDescriptionInit, fromUserId: number) => {
  if (screenShare.sessionState.value !== 'idle') {
    console.log('[ScreenShareSimple] Session not idle, ignoring offer. State:', screenShare.sessionState.value)
    return
  }

  if (requestAccepted.value) {
    const convId = incomingRequestInfo.value?.conversationId || props.conversationId
    screenShare.acceptShare(signal, fromUserId, convId)
    requestAccepted.value = false
    incomingRequestInfo.value = null
    return
  }

  pendingOffer.value = { signal, fromUserId }
  showIncomingRequest.value = true
}

const handleIncomingRequest = (data: any) => {
  const fromUserId = data.from_user_id || data.user_id
  const conversationId = data.conversation_id
  const fromUserName = data.from_user_name || props.senderName || '对方'

  incomingRequestInfo.value = {
    fromUserId,
    conversationId,
    fromUserName
  }

  showIncomingRequest.value = true
}

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
      const stream = isInitiator.value ? localStream.value : remoteStream.value
      if (minimizedVideoRef.value && stream) {
        minimizedVideoRef.value.srcObject = stream
        minimizedVideoRef.value.play().catch(err => {
          if (err.name !== 'AbortError') {
            console.error('最小化视频播放失败:', err)
          }
        })
      }
    })
  } else {
    nextTick(() => {
      if (isInitiator.value && localStream.value && localVideoRef.value) {
        localVideoRef.value.srcObject = localStream.value
        localVideoRef.value.play().catch(err => {
          if (err.name !== 'AbortError') {
            console.error('展开后本地视频播放失败:', err)
          }
        })
      } else if (!isInitiator.value && remoteStream.value && remoteVideoRef.value) {
        remoteVideoRef.value.srcObject = remoteStream.value
        remoteVideoRef.value.play().catch(err => {
          if (err.name !== 'AbortError') {
            console.error('展开后远程视频播放失败:', err)
          }
        })
      }
    })
  }
}

// 全屏：对 <video> 元素（remoteVideoRef）调原生 requestFullscreen（旧版 ScreenShare.vue 的写法）。
// 之前不工作的根因是 main.js 的 setPermissionRequestHandler 拒了 fullscreen 权限，导致
// requestFullscreen 挂起；现已把 'fullscreen' 加入白名单，元素全屏可用。按 document.fullscreenElement
// 定方向，isFullscreen 成功后置位，fullscreenchange 监听同步 Esc 退出。
const toggleFullscreen = async () => {
  try {
    if (!document.fullscreenElement) {
      if (remoteVideoRef.value?.requestFullscreen) {
        await remoteVideoRef.value.requestFullscreen()
      }
      isFullscreen.value = true
    } else {
      if (document.exitFullscreen) {
        await document.exitFullscreen()
      }
      isFullscreen.value = false
    }
  } catch (error) {
    console.warn('[ScreenShareSimple] 全屏切换失败:', error)
  }
}

const exitFullscreen = () => {
  if (!isFullscreen.value) return
  isFullscreen.value = false
  if (document.exitFullscreen) {
    document.exitFullscreen().catch(() => {})
  }
}

// Esc 退出全屏：原生全屏时浏览器会先拦截 Esc 退出原生全屏（由 fullscreenchange 同步状态）；
// 原生未生效（CSS 兜底）时由本监听退出 CSS 全屏。
const handleFullscreenEsc = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && isFullscreen.value) {
    exitFullscreen()
  }
}

// 原生全屏状态变化（含 Esc/F11 退出）：同步 isFullscreen，不再做内联样式操纵--
// 视觉统一由 .fullscreen 类 + :fullscreen 伪类 CSS 接管。
const handleFullscreenChange = () => {
  isFullscreen.value = !!document.fullscreenElement
}

const handleStop = () => {
  screenShare.stopSharing()
  showSourcePicker.value = false
  showIncomingRequest.value = false
  showWaitingAccept.value = false
  stopWaitingTimer()
  pendingOffer.value = null
  incomingRequestInfo.value = null
  requestAccepted.value = false
  selectedSource.value = null
  screenShareName.value = ''
  if (savedStream.value) {
    savedStream.value.getTracks().forEach(t => t.stop())
    savedStream.value = null
  }
  emit('screen-share-stop')
}

const startDurationTimer = () => {
  // 幂等：已运行则不重启，避免重复调用导致计时丢失
  if (durationTimer) return
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

/* ---- 等待接受倒计时管理 ---- */
const startWaitingTimer = () => {
  stopWaitingTimer()
  waitingCountdown.value = WAITING_TIMEOUT
  waitingTimer = window.setInterval(() => {
    waitingCountdown.value--
    if (waitingCountdown.value <= 0) {
      stopWaitingTimer()
      QMessage.warning('对方未响应，屏幕共享已取消')
      handleStop()
    }
  }, 1000)
}

const stopWaitingTimer = () => {
  if (waitingTimer) {
    clearInterval(waitingTimer)
    waitingTimer = null
  }
  waitingCountdown.value = WAITING_TIMEOUT
}

watch([localStream, isInitiator], ([stream, initiator]) => {
  if (stream && initiator) {
    nextTick(() => {
      if (localVideoRef.value && localVideoRef.value.srcObject !== stream) {
        localVideoRef.value.srcObject = stream
      }
      // 发起方最小化预览也需要绑定 stream
      if (isMinimized.value && minimizedVideoRef.value && minimizedVideoRef.value.srcObject !== stream) {
        minimizedVideoRef.value.srcObject = stream
      }
    })
  }
}, { immediate: true })

watch([remoteStream, isInitiator], ([stream, initiator]) => {
  if (stream && !initiator) {
    nextTick(() => {
      if (remoteVideoRef.value && remoteVideoRef.value.srcObject !== stream) {
        remoteVideoRef.value.srcObject = stream
        remoteVideoRef.value.play().catch(() => {})
      }
    })
  }
}, { immediate: true })

watch(showWaitingAccept, (waiting) => {
  if (waiting) {
    startWaitingTimer()
  } else {
    stopWaitingTimer()
    if (localStream.value && isInitiator.value) {
      nextTick(() => {
        if (localVideoRef.value) {
          localVideoRef.value.srcObject = localStream.value
        }
      })
    }
  }
})

watch(sessionState, (state) => {
  if (state === 'active') {
    startDurationTimer()
    showWaitingAccept.value = false
    emit('screen-share-start', { conversationId: props.conversationId || props.receiverId || 0 })
    // 发起方共享开始时自动最小化，避免全屏预览导致 Droste 嵌套回路
    if (isInitiator.value) {
      isMinimized.value = true
    }
  } else if (state === 'idle' || state === 'ended') {
    stopDurationTimer()
    isMinimized.value = false
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
  stopWaitingTimer()
  if (rafId) {
    cancelAnimationFrame(rafId)
  }
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
  document.removeEventListener('keydown', handleFullscreenEsc)
  document.removeEventListener('fullscreenchange', handleFullscreenChange)
})

const initFullscreenListener = () => {
  document.addEventListener('keydown', handleFullscreenEsc)
  document.addEventListener('fullscreenchange', handleFullscreenChange)
}

initFullscreenListener()

const stopWaitingAccept = async () => {
  console.log('[ScreenShareSimple] stopWaitingAccept called')
  console.log('[ScreenShareSimple] savedStream.value:', !!savedStream.value)
  console.log('[ScreenShareSimple] props.receiverId:', props.receiverId)
  console.log('[ScreenShareSimple] screenShare.pendingStream.value:', !!screenShare.pendingStream.value)
  console.log('[ScreenShareSimple] sessionState:', screenShare.sessionState.value)

  showWaitingAccept.value = false

  if (savedStream.value && props.receiverId) {
    if (screenShare.sessionState.value !== 'idle') {
      console.warn('[ScreenShareSimple] Session not idle, cannot start connection. State:', screenShare.sessionState.value)
      QMessage.warning('当前已有屏幕共享连接，请先结束')
      return
    }

    try {
      if (screenShare.pendingStream.value) {
        console.log('[ScreenShareSimple] Calling startConnection')
        await screenShare.startConnection()
      } else {
        console.log('[ScreenShareSimple] Calling startConnectionWithStream')
        await screenShare.startConnectionWithStream(props.receiverId, savedStream.value)
      }
      console.log('[ScreenShareSimple] Connection started successfully')
      // 发起方：连接已开始即启动计时（sessionState→active 的回调时序不可靠，作为兜底触发）
      startDurationTimer()
    } catch (error: any) {
      console.error('[ScreenShareSimple] 建立连接失败:', error)
      if (error.message) {
        QMessage.warning(error.message)
      } else {
        QMessage.error('建立屏幕共享连接失败，请稍后重试')
      }
      showWaitingAccept.value = false
    }
  } else {
    console.warn('[ScreenShareSimple] Missing savedStream or receiverId, cannot start connection')
    QMessage.warning('无法开始屏幕共享，请检查连接信息')
  }
}

defineExpose({
  selectSource: screenShare.selectSource,
  startSharing: screenShare.startSharing,
  startSharingWithStream: screenShare.startSharingWithStream,
  initiateShare,
  handleIncomingOffer,
  handleIncomingRequest,
  stopWaitingAccept,
  acceptRequest: screenShare.acceptRequest,
  rejectRequest: screenShare.rejectRequest,
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

.source-picker {
  padding: 16px;
}

.source-picker-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.source-picker-header h3 {
  color: #fff;
  font-size: 16px;
  margin: 0;
}

.source-picker-header .close-btn {
  background: none;
  border: none;
  color: rgba(255, 255, 255, 0.6);
  cursor: pointer;
  font-size: 16px;
  padding: 4px;
}

.source-picker-header .close-btn:hover {
  color: #fff;
}

.source-picker-body {
  max-height: 360px;
  overflow-y: auto;
  padding-right: 4px;
}

.source-group-title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: rgba(255, 255, 255, 0.85);
  font-size: 13px;
  font-weight: 600;
  margin: 4px 0 10px;
}

.source-group-title:first-child {
  margin-top: 0;
}

.source-group-title + .source-grid + .source-group-title {
  margin-top: 18px;
}

.source-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.source-item {
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  border: 2px solid transparent;
  background: rgba(255, 255, 255, 0.04);
  transition: all 0.2s;
}

.source-item:hover {
  border-color: rgba(59, 130, 246, 0.5);
  transform: translateY(-1px);
}

.source-item.selected {
  border-color: #3b82f6;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.25);
}

.source-thumbnail {
  width: 100%;
  height: 110px;
  background: #000;
  display: flex;
  align-items: center;
  justify-content: center;
}

.source-thumbnail img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.source-placeholder {
  color: rgba(255, 255, 255, 0.3);
  font-size: 32px;
}

.source-name {
  padding: 8px;
  color: #fff;
  font-size: 12px;
  background: rgba(0, 0, 0, 0.3);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.source-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 40px 0;
  color: rgba(255, 255, 255, 0.4);
}

.source-empty i {
  font-size: 28px;
}

.source-picker-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
}

.source-hint {
  color: rgba(255, 255, 255, 0.55);
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.source-hint i {
  color: #3b82f6;
}

.source-picker-actions {
  display: flex;
  gap: 12px;
  flex-shrink: 0;
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

.btn-primary {
  padding: 8px 20px;
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  border: none;
  border-radius: 6px;
  color: #fff;
  cursor: pointer;
  font-size: 14px;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary:hover:not(:disabled) {
  background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
}

.incoming-request {
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
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  color: #fff;
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

.waiting-accept {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 32px 24px 28px;
  gap: 14px;
}

.waiting-icon {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  color: #fff;
  animation: pulse 2s infinite;
}

.waiting-text {
  text-align: center;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.waiting-title {
  color: #fff;
  font-size: 15px;
  font-weight: 500;
}

.waiting-title strong {
  color: #60a5fa;
  font-weight: 600;
}

.waiting-sub {
  color: rgba(255, 255, 255, 0.55);
  font-size: 12px;
}

.waiting-countdown {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.08);
  color: rgba(255, 255, 255, 0.7);
  font-size: 12px;
  transition: all 0.2s;
}

.waiting-countdown i {
  font-size: 11px;
}

.waiting-countdown.urgent {
  background: rgba(239, 68, 68, 0.18);
  color: #fca5a5;
  animation: pulse 1s infinite;
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

.pause-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 5;
}

.pause-overlay__inner {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  color: #fff;
}

.pause-overlay__inner i {
  font-size: 40px;
  color: #f59e0b;
  margin-bottom: 4px;
}

.pause-overlay__title {
  font-size: 16px;
  font-weight: 600;
}

.pause-overlay__desc {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.7);
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
  position: relative;
}

.minimized-preview video {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.minimized-paused-badge {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  background: rgba(0, 0, 0, 0.6);
  color: #f59e0b;
  font-size: 11px;
  font-weight: 500;
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

.minimized-status.paused {
  color: #f59e0b;
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

.screen-share-overlay.fullscreen {
  width: 100vw !important;
  height: 100vh !important;
  max-width: none !important;
  border-radius: 0;
  top: 0 !important;
  left: 0 !important;
  transform: none !important;
  display: flex;
  flex-direction: column;
  background: #000;
}

.screen-share-overlay.fullscreen .screen-share-header {
  display: none;
}

.screen-share-overlay.fullscreen .screen-share-body {
  height: 100%;
  flex: 1;
  display: flex;
  flex-direction: column;
}

.screen-share-overlay.fullscreen .video-container {
  height: 100% !important;
  min-height: 100vh;
  flex: 1;
}

.screen-share-overlay.fullscreen .screen-share-controls {
  display: none;
}

.screen-share-overlay:fullscreen,
.screen-share-overlay:-webkit-full-screen,
.screen-share-overlay:-moz-full-screen,
.screen-share-overlay:-ms-fullscreen {
  width: 100vw !important;
  height: 100vh !important;
  max-width: none !important;
  border-radius: 0;
  top: 0;
  left: 0;
  transform: none;
  display: flex;
  flex-direction: column;
  background: #000;
}

.screen-share-overlay:fullscreen .screen-share-header,
.screen-share-overlay:-webkit-full-screen .screen-share-header,
.screen-share-overlay:-moz-full-screen .screen-share-header,
.screen-share-overlay:-ms-fullscreen .screen-share-header {
  display: none;
}

.screen-share-overlay:fullscreen .screen-share-body,
.screen-share-overlay:-webkit-full-screen .screen-share-body,
.screen-share-overlay:-moz-full-screen .screen-share-body,
.screen-share-overlay:-ms-fullscreen .screen-share-body {
  height: 100%;
  flex: 1;
  display: flex;
  flex-direction: column;
}

.screen-share-overlay:fullscreen .video-container,
.screen-share-overlay:-webkit-full-screen .video-container,
.screen-share-overlay:-moz-full-screen .video-container,
.screen-share-overlay:-ms-fullscreen .video-container {
  height: 100% !important;
  min-height: 100vh;
  flex: 1;
}

.screen-share-overlay:fullscreen .screen-share-controls,
.screen-share-overlay:-webkit-full-screen .screen-share-controls,
.screen-share-overlay:-moz-full-screen .screen-share-controls,
.screen-share-overlay:-ms-fullscreen .screen-share-controls {
  display: none;
}

.video-container.fullscreen-mode {
  position: relative;
}

.fullscreen-header {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  padding: 16px 20px;
  background: linear-gradient(to bottom, rgba(0, 0, 0, 0.7) 0%, transparent 100%);
  display: flex;
  align-items: center;
  justify-content: space-between;
  z-index: 10;
  opacity: 0;
  transition: opacity 0.3s;
}

.video-container.fullscreen-mode:hover .fullscreen-header {
  opacity: 1;
}

.fullscreen-title {
  color: #fff;
  font-size: 16px;
  font-weight: 500;
}

.fullscreen-close-btn {
  padding: 8px 12px;
  background: rgba(255, 255, 255, 0.1);
  border: none;
  border-radius: 6px;
  color: #fff;
  cursor: pointer;
  font-size: 16px;
  transition: all 0.2s;
}

.fullscreen-close-btn:hover {
  background: rgba(239, 68, 68, 0.8);
}

.fullscreen-controls {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 16px 20px;
  background: linear-gradient(to top, rgba(0, 0, 0, 0.7) 0%, transparent 100%);
  display: flex;
  align-items: center;
  justify-content: space-between;
  z-index: 10;
  opacity: 0;
  transition: opacity 0.3s;
}

.video-container.fullscreen-mode:hover .fullscreen-controls {
  opacity: 1;
}

.fullscreen-duration {
  color: rgba(255, 255, 255, 0.9);
  font-size: 16px;
  font-family: 'SF Mono', Monaco, monospace;
}

.fullscreen-actions {
  display: flex;
  gap: 12px;
}

.fullscreen-btn {
  padding: 10px 16px;
  background: rgba(255, 255, 255, 0.15);
  border: none;
  border-radius: 8px;
  color: #fff;
  cursor: pointer;
  font-size: 16px;
  transition: all 0.2s;
}

.fullscreen-btn:hover {
  background: rgba(255, 255, 255, 0.25);
}

.fullscreen-btn.stop-btn {
  background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
}

.fullscreen-btn.stop-btn:hover {
  background: linear-gradient(135deg, #dc2626 0%, #b91c1c 100%);
}
</style>
