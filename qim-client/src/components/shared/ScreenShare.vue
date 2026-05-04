<template>
  <Teleport to="body">
    <div v-if="isInitiator || isViewer" ref="screenShareOverlayRef" class="screen-share-overlay" :class="{ 'is-initiator': isInitiator, 'is-viewer': isViewer, 'minimized': isMinimized }">
      <div class="screen-share-header" @mousedown="startDrag" @dblclick="handleHeaderDblClick">
        <div class="share-indicator">
          <span class="pulse-dot"></span>
          <span class="share-label">{{ isInitiator ? '正在共享屏幕' : '正在观看' }}</span>
          <span v-if="isInitiator && screenShareName" class="share-name">{{ screenShareName }}</span>
          <span v-else-if="isViewer && senderName" class="share-name">{{ senderName }} 的屏幕</span>
        </div>
        <div class="header-actions">
          <button v-if="isViewer" class="action-btn join-btn" @click.stop="joinShare" :class="{ 'joined': hasJoined }">
            <i :class="hasJoined ? 'fas fa-minus' : 'fas fa-plus'"></i>
            <span>{{ hasJoined ? '退出' : '加入' }}</span>
          </button>
          <button class="action-btn" @click.stop="toggleMinimize" :title="isMinimized ? '展开' : '最小化'">
            <i :class="isMinimized ? 'fas fa-expand' : 'fas fa-minus'"></i>
          </button>
          <button class="action-btn close-btn" @click.stop="stopShare" title="停止共享">
            <i class="fas fa-times"></i>
          </button>
        </div>
      </div>

      <div v-if="isMinimized" class="minimized-content">
        <div class="minimized-preview">
          <video ref="minimizedVideoRef" autoplay playsinline muted></video>
        </div>
        <div class="minimized-info">
          <div class="info-top">
            <div class="minimized-status">
              <span class="pulse-dot" :class="{ active: isSharing }"></span>
              <span>{{ isInitiator ? '共享中' : '观看中' }}</span>
            </div>
            <div class="minimized-duration">{{ formattedDuration }}</div>
            <div class="minimized-name">{{ screenShareName || senderName || '屏幕共享' }}</div>
          </div>
          <div class="minimized-actions" @click.stop>
            <button class="action-btn expand-btn" @click="expandFromMinimized">
              <i class="fas fa-expand"></i>
              <span>展开</span>
            </button>
            <button class="action-btn close-btn" @click="stopShare">
              <i class="fas fa-stop"></i>
              <span>停止</span>
            </button>
          </div>
        </div>
      </div>

      <div v-if="!isMinimized" class="screen-share-body">
        <div v-if="showSourcePicker && isInitiator" class="source-picker">
          <div class="source-picker-header">
            <h3>选择共享来源</h3>
            <button class="close-btn" @click="cancelShare"><i class="fas fa-times"></i></button>
          </div>
          <div class="source-grid">
            <div
              v-for="source in screenSources"
              :key="source.id"
              class="source-item"
              :class="{ selected: selectedSource?.id === source.id }"
              @click="selectSource(source)"
            >
              <div class="source-thumbnail">
                <img v-if="source.thumbnail" :src="source.thumbnail" :alt="source.name" />
                <div v-else class="source-placeholder">
                  <i class="fas fa-desktop"></i>
                </div>
              </div>
              <div class="source-name">{{ source.name }}</div>
            </div>
          </div>
          <div class="source-picker-footer">
            <button class="btn-secondary" @click="cancelShare">取消</button>
            <button class="btn-primary" @click="startSharing" :disabled="!selectedSource">开始共享</button>
          </div>
        </div>

        <div v-show="!showSourcePicker || !isInitiator" class="video-container">
          <video
            ref="remoteVideoRef"
            class="remote-video"
            autoplay
            playsinline
            muted
          ></video>

          <div v-if="!remoteStreamActive && isViewer && !hasJoined" class="waiting-state">
            <div class="waiting-icon">
              <i class="fas fa-desktop"></i>
            </div>
            <div class="waiting-text">
              <span class="waiting-title">{{ senderName }} 正在共享屏幕</span>
              <span class="waiting-subtitle">点击"加入"开始观看</span>
            </div>
          </div>

          <div v-if="!remoteStreamActive && isViewer && hasJoined" class="connecting-state">
            <div class="spinner"></div>
            <span>正在连接...</span>
          </div>

        </div>
      </div>

      <div v-if="!isMinimized && isViewer && hasJoined" class="screen-share-controls">
        <button class="control-btn" @click="toggleFullscreen" :title="isFullscreen ? '退出全屏' : '全屏'">
          <i :class="isFullscreen ? 'fas fa-compress' : 'fas fa-expand'"></i>
        </button>
        <button class="control-btn" @click="togglePictureInPicture" title="画中画">
          <i class="fas fa-clone"></i>
        </button>
      </div>

      <div v-if="isInitiator && isSharing && !isMinimized" class="initiator-controls">
        <div class="control-bar">
          <div class="sharing-status">
            <span class="pulse-dot active"></span>
            <span>共享中 · {{ formattedDuration }}</span>
          </div>
          <div class="control-buttons">
            <button class="control-btn" @click="pauseShare" :title="isPaused ? '恢复' : '暂停'">
              <i :class="isPaused ? 'fas fa-play' : 'fas fa-pause'"></i>
            </button>
            <button class="control-btn" @click="switchSource" title="切换窗口">
              <i class="fas fa-exchange-alt"></i>
            </button>
            <button class="control-btn danger" @click="stopShare" title="停止共享">
              <i class="fas fa-stop"></i>
            </button>
          </div>
        </div>
      </div>
    </div>

    <div v-if="showFloatingWindow && floatingMode" class="floating-window" :style="floatingWindowStyle">
      <div class="floating-header" @mousedown="startFloatingDrag">
        <span class="floating-title">
          <i class="fas fa-desktop"></i>
          {{ isInitiator ? '共享屏幕' : (senderName ? senderName + ' 的屏幕' : '屏幕共享') }}
        </span>
        <div class="floating-actions">
          <button @click.stop="expandFromFloating" title="展开">
            <i class="fas fa-external-link-alt"></i>
          </button>
          <button @click.stop="closeFloating" class="close">
            <i class="fas fa-times"></i>
          </button>
        </div>
      </div>
      <div class="floating-body">
        <video
          ref="floatingVideoRef"
          class="floating-video"
          autoplay
          playsinline
          :srcObject="floatingStream"
        ></video>
        <div v-if="!floatingStream" class="floating-placeholder">
          <i class="fas fa-desktop"></i>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted, nextTick } from 'vue'
// @ts-ignore - WebRTC module has no type declarations
// @ts-ignore - WebRTC module has no type declarations
import { screenShareSender, screenShareReceiver } from '../../utils/webrtc'
import { logger } from '../../utils/logger';

interface ScreenSource {
  id: string
  name: string
  thumbnail?: string
}

interface Props {
  receiverId?: string | number
  senderId?: number | null
  senderName?: string
  conversationId?: string | number
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'screen-share-start': [data: { conversationId: string | number }]
  'screen-share-stop': []
  'screen-share-join': []
  'screen-share-leave': []
}>()

const isInitiator = ref(false)
const isViewer = ref(false)
const isSharing = ref(false)
const isPaused = ref(false)
const isMinimized = ref(false)
const isFullscreen = ref(false)
const hasJoined = ref(false)
const showSourcePicker = ref(false)
const showFloatingWindow = ref(false)
const floatingMode = ref(false)
const remoteStreamActive = ref(false)
const screenSources = ref<ScreenSource[]>([])
const selectedSource = ref<ScreenSource | null>(null)
const screenShareName = ref('')
const sharingStartTime = ref<number | null>(null)
const durationInterval = ref<number | null>(null)
const formattedDuration = ref('00:00')

// 内部保存的 senderId，用于 handleOffer 传入的 fromUserId
const internalSenderId = ref<number | null>(null)

const remoteVideoRef = ref<HTMLVideoElement | null>(null)
const floatingVideoRef = ref<HTMLVideoElement | null>(null)
const minimizedVideoRef = ref<HTMLVideoElement | null>(null)
const floatingStream = ref<MediaStream | null>(null)

const floatingPosition = ref({ x: 20, y: 20 })
const windowPosition = ref({
  x: window.innerWidth / 2 - 210, // 420px 宽的一半
  y: window.innerHeight / 2 - 200  // 居中位置
})

const dragState = ref({
  isDragging: false,
  startX: 0,
  startY: 0,
  elementX: 0,
  elementY: 0
})

let rafId: number | null = null

const floatingWindowStyle = computed(() => ({
  right: `${floatingPosition.value.x}px`,
  bottom: `${floatingPosition.value.y}px`
}))

watch(() => props.senderId, (newId) => {
  if (newId) {
    isViewer.value = true
    hasJoined.value = true
    // 注意：不在这里调用 startReceivingStream()
    // 接收器初始化应该在 handleOffer 中完成，避免重复初始化
    logger.log('ScreenShare: 检测到senderId变化，设置为观看者模式', newId)
  }
})

watch(isSharing, (sharing) => {
  if (sharing) {
    startDurationTimer()
  } else {
    stopDurationTimer()
  }
})

watch(() => dragState.value.isDragging, (isDragging) => {
  if (screenShareOverlayRef.value) {
    if (isDragging) {
      screenShareOverlayRef.value.classList.add('dragging')
    } else {
      screenShareOverlayRef.value.classList.remove('dragging')
    }
  }
})

const startDurationTimer = () => {
  if (durationInterval.value) clearInterval(durationInterval.value)
  sharingStartTime.value = Date.now()
  durationInterval.value = window.setInterval(() => {
    if (sharingStartTime.value) {
      const elapsed = Math.floor((Date.now() - sharingStartTime.value) / 1000)
      const mins = Math.floor(elapsed / 60).toString().padStart(2, '0')
      const secs = (elapsed % 60).toString().padStart(2, '0')
      formattedDuration.value = `${mins}:${secs}`
    }
  }, 1000)
}

const stopDurationTimer = () => {
  if (durationInterval.value) {
    clearInterval(durationInterval.value)
    durationInterval.value = null
  }
}

const initiateShare = async () => {
  logger.log('ScreenShare: initiateShare函数被调用')
  try {
    // 设置为发起者状态
    isInitiator.value = true
    logger.log('ScreenShare: 开始获取屏幕源')
    const sources = await getScreenSources()
    logger.log('ScreenShare: 获取到屏幕源:', sources)
    if (sources.length > 0) {
      logger.log('ScreenShare: 显示源选择器')
      screenSources.value = sources
      showSourcePicker.value = true
    } else {
      logger.log('ScreenShare: 没有屏幕源，直接开始共享')
      await startSharing()
    }
  } catch (error) {
    console.error('ScreenShare: 初始化共享失败:', error)
    showSourcePicker.value = true
  }
}

// 当isInitiator变为true且未共享时，自动显示源选择器
watch([isInitiator, isSharing], ([initiator, sharing]) => {
  if (initiator && !sharing) {
    initiateShare()
  }
})

const getScreenSources = async (): Promise<ScreenSource[]> => {
  const electron = window.electron as any
  if (electron?.ipcRenderer) {
    try {
      // 发送请求到主进程获取屏幕源
      electron.ipcRenderer.send('start-screen-share');
      
      // 等待屏幕源信息
      const sources = await new Promise((resolve) => {
        electron.ipcRenderer.once('screen-sources', (_event: any, sources: any[]) => {
          resolve(sources);
        });
      });
      
      logger.log('收到屏幕源:', sources);
      
      if (Array.isArray(sources) && sources.length > 0) {
        return sources.map((source: any) => ({
          id: source.id,
          name: source.name,
          thumbnail: source.thumbnail // 直接使用thumbnail数据，不调用toDataURL
        }))
      }
    } catch (error) {
      console.error('获取屏幕源失败:', error)
    }
  }
  return []
}

const selectSource = (source: ScreenSource) => {
  selectedSource.value = source
}

const cancelShare = () => {
  showSourcePicker.value = false
  selectedSource.value = null
  isInitiator.value = false
}

// 保存选择的屏幕源ID，用于后续建立连接
const selectedSourceId = ref('')

const startSharing = async () => {
  logger.log('ScreenShare: startSharing函数被调用')
  if (!selectedSource.value) {
    logger.log('ScreenShare: 未选择屏幕源')
    return
  }

  try {
    logger.log('ScreenShare: 开始共享，源:', selectedSource.value.name)
    showSourcePicker.value = false
    screenShareName.value = selectedSource.value.name
    isSharing.value = true
    isInitiator.value = true
    
    // 保存选择的屏幕源ID
    selectedSourceId.value = selectedSource.value.id

    // 使用 screenShareSender 开始共享
    if (props.receiverId !== undefined) {
      logger.log('ScreenShare: 发送屏幕共享请求，接收者ID:', props.receiverId, '会话ID:', props.conversationId)
      // 发送屏幕共享请求，使用会话ID
      screenShareSender.sendShareRequest(props.receiverId, props.conversationId)
      
      // 等待对方接受
      logger.log('ScreenShare: 等待对方接受屏幕共享请求...')
      
      // 发送屏幕共享开始事件
      logger.log('ScreenShare: 发送屏幕共享开始事件，接收者ID:', props.receiverId, '会话ID:', props.conversationId)
      emit('screen-share-start', { conversationId: props.conversationId || props.receiverId })
      
      // 开始共享（注意：这里会直接发送 webrtc_offer，实际应该在对方接受后再发送）
      logger.log('ScreenShare: 使用 screenShareSender 开始共享，接收者ID:', props.receiverId)
      await screenShareSender.startScreenShare(props.receiverId)
      
      // 等待 DOM 更新
      await nextTick()
      
      // 建立连接并显示本地预览
      logger.log('ScreenShare: 调用 establishConnection 显示本地预览')
      await establishConnection()
    }
  } catch (error) {
    console.error('ScreenShare: 开始屏幕共享失败:', error)
    isSharing.value = false
    isInitiator.value = false
  }
}

// 当收到对方接受后，开始建立 WebRTC 连接
let isComponentMounted = true

const establishConnection = async () => {
  logger.log('ScreenShare: establishConnection函数被调用')
  
  try {
    // screenShareSender 已经在 startScreenShare 中建立了连接
    // 这里不需要重复建立连接
    logger.log('ScreenShare: WebRTC 连接已经在 startScreenShare 中建立')
    
    // 使用本地预览视频元素显示自己的屏幕（预览用）
    if (remoteVideoRef.value && isComponentMounted) {
      // 获取屏幕共享流并设置到预览视频元素
      const screenStream = screenShareSender.getScreenStream()
      if (screenStream) {
        logger.log('ScreenShare: 设置本地预览视频流')
        logger.log('ScreenShare: screenStream tracks:', screenStream.getTracks().map((t: MediaStreamTrack) => ({ kind: t.kind, id: t.id, enabled: t.enabled })))
        
        remoteVideoRef.value.srcObject = screenStream
        
        // 尝试播放视频（不等待元数据加载）
        try {
          await remoteVideoRef.value.play()
          logger.log('ScreenShare: 预览视频播放成功')
        } catch (err: any) {
          console.error('尝试播放预览视频失败:', err)
          logger.log('ScreenShare: 播放失败，错误名称:', err.name)
          
          // 如果自动播放被阻止，尝试静音后再播放
          if (err.name === 'NotAllowedError') {
            logger.log('ScreenShare: 自动播放被阻止，尝试静音后播放')
            remoteVideoRef.value.muted = true
            await remoteVideoRef.value.play().catch((e: any) => console.error('静音后播放仍然失败:', e))
          }
        }
      } else {
        logger.log('ScreenShare: 未获取到屏幕共享流，预览功能暂时不可用')
      }
    } else {
      logger.log('ScreenShare: remoteVideoRef 未准备好或组件已卸载')
    }
  } catch (error) {
    if (isComponentMounted) {
      console.error('ScreenShare: 建立连接失败:', error)
      isSharing.value = false
      isInitiator.value = false
    } else {
      console.warn('ScreenShare: 组件已卸载，跳过状态更新')
    }
  }
}

const pauseShare = () => {
  isPaused.value = !isPaused.value
  const video = remoteVideoRef.value
  if (video) {
    if (isPaused.value) {
      video.pause()
    } else {
      video.play()
    }
  }
}

const switchSource = async () => {
  const sources = await getScreenSources()
  if (sources.length > 0) {
    screenSources.value = sources
    showSourcePicker.value = true
  }
}

const stopShare = () => {
  logger.log('ScreenShare: stopShare 被调用')
  logger.log('ScreenShare: isInitiator:', isInitiator.value, 'isViewer:', isViewer.value)
  
  // 使用 screenShareSender 停止共享
  if (isInitiator.value) {
    logger.log('ScreenShare: 停止发起方共享')
    screenShareSender.stopScreenShare()
  } else if (isViewer.value) {
    logger.log('ScreenShare: 停止接收方共享')
    screenShareReceiver.stop()
  }

  // 清理视频流
  if (remoteVideoRef.value?.srcObject) {
    logger.log('ScreenShare: 清理视频流')
    const stream = remoteVideoRef.value.srcObject as MediaStream
    stream.getTracks().forEach(track => track.stop())
    remoteVideoRef.value.srcObject = null
  }
  
  // 清理最小化视频流
  if (minimizedVideoRef.value?.srcObject) {
    logger.log('ScreenShare: 清理最小化视频流')
    const stream = minimizedVideoRef.value.srcObject as MediaStream
    stream.getTracks().forEach(track => track.stop())
    minimizedVideoRef.value.srcObject = null
  }

  // 重置所有状态
  isSharing.value = false
  isInitiator.value = false
  isViewer.value = false
  isPaused.value = false
  isMinimized.value = false
  hasJoined.value = false
  remoteStreamActive.value = false
  showSourcePicker.value = false
  selectedSource.value = null
  screenShareName.value = ''
  floatingStream.value = null
  internalSenderId.value = null

  if (floatingMode.value) {
    closeFloating()
  }

  logger.log('ScreenShare: 所有状态已重置')
  emit('screen-share-stop')
}

const joinShare = () => {
  hasJoined.value = !hasJoined.value
  if (hasJoined.value) {
    emit('screen-share-join')
    // 只有在 senderId 存在时才初始化接收器
    const actualSenderId = internalSenderId.value || props.senderId
    if (actualSenderId) {
      startReceivingStream()
    } else {
      logger.log('ScreenShare: 加入共享，但 senderId 还未设置，等待 handleOffer 后再初始化')
    }
  } else {
    emit('screen-share-leave')
    stopReceivingStream()
  }
}

const startReceivingStream = () => {
  // 使用内部 senderId 或 prop senderId
  const senderId = internalSenderId.value || props.senderId
  if (senderId) {
    // 优先使用浮窗视频元素（如果浮窗模式开启）
    const videoElement = floatingMode.value ? floatingVideoRef.value : remoteVideoRef.value
    
    // 确保视频元素已经初始化
    if (videoElement) {
      logger.log('ScreenShare: 初始化screenShareReceiver，视频元素:', videoElement)
      // 初始化screenShareReceiver，传递远程流接收回调
      screenShareReceiver.init(videoElement, (stream: MediaStream) => {
        logger.log('ScreenShare: 收到远程流，更新状态')
        remoteStreamActive.value = true
        if (floatingMode.value) {
          // 浮窗模式下通过模板绑定设置srcObject，这里只更新floatingStream
          floatingStream.value = stream
        }
      })
    } else {
      logger.log('ScreenShare: 视频元素还未初始化，等待下一帧')
      // 等待下一帧，确保视频元素已经渲染
      setTimeout(() => {
        // 再次检查浮窗模式
        const delayedVideoElement = floatingMode.value ? floatingVideoRef.value : remoteVideoRef.value
        
        if (delayedVideoElement) {
          logger.log('ScreenShare: 延迟初始化screenShareReceiver，视频元素:', delayedVideoElement)
          // 初始化screenShareReceiver，传递远程流接收回调
          screenShareReceiver.init(delayedVideoElement, (stream: MediaStream) => {
            logger.log('ScreenShare: 收到远程流，更新状态')
            remoteStreamActive.value = true
            // 只在浮窗模式下设置floatingStream，避免冲突
            if (floatingMode.value) {
              floatingStream.value = stream
            }
          })
        } else {
          console.error('ScreenShare: 无法初始化screenShareReceiver，视频元素为null')
          console.error('ScreenShare: floatingMode:', floatingMode.value)
          console.error('ScreenShare: floatingVideoRef.value:', floatingVideoRef.value)
          console.error('ScreenShare: remoteVideoRef.value:', remoteVideoRef.value)
        }
      }, 100)
    }
  }else{
      console.error('ScreenShare: 无法初始化screenShareReceiver，senderId为null')
  }
}

const stopReceivingStream = () => {
  screenShareReceiver.stop()
  if (floatingMode.value && floatingVideoRef.value) {
    floatingVideoRef.value.srcObject = null
    floatingStream.value = null
  } else if (remoteVideoRef.value) {
    remoteVideoRef.value.srcObject = null
  }
  remoteStreamActive.value = false
}

const toggleMinimize = () => {
  isMinimized.value = !isMinimized.value
  
  if (isMinimized.value) {
    nextTick(() => {
      // 根据当前模式获取正确的视频源
      const sourceVideo = floatingMode.value ? floatingVideoRef.value : remoteVideoRef.value
      if (minimizedVideoRef.value && sourceVideo?.srcObject) {
        // 先暂停当前播放，避免 AbortError
        try {
          minimizedVideoRef.value.pause()
        } catch (e) {
          // 忽略暂停错误
        }
        
        minimizedVideoRef.value.srcObject = sourceVideo.srcObject
        minimizedVideoRef.value.play().catch(err => {
          if (err.name !== 'AbortError') {
            console.error('最小化视频播放失败:', err)
          }
        })
      }
    })
  }
}

const expandFromMinimized = () => {
  isMinimized.value = false
}

const handleHeaderDblClick = () => {
  if (isMinimized.value) {
    expandFromMinimized()
  }
}

const expandFromFloating = () => {
  isMinimized.value = false
  floatingMode.value = false
  if (floatingVideoRef.value?.srcObject) {
    if (remoteVideoRef.value) {
      remoteVideoRef.value.srcObject = floatingVideoRef.value.srcObject
    }
    floatingStream.value = null
  }
}

const closeFloating = () => {
  showFloatingWindow.value = false
  floatingMode.value = false
  if (!isSharing.value && !hasJoined.value) {
    isViewer.value = false
    isInitiator.value = false
  }
}

const toggleFullscreen = async () => {
  if (!document.fullscreenElement) {
    await remoteVideoRef.value?.requestFullscreen()
    isFullscreen.value = true
  } else {
    await document.exitFullscreen()
    isFullscreen.value = false
  }
}

const togglePictureInPicture = async () => {
  if (document.pictureInPictureElement) {
    await document.exitPictureInPicture()
  } else if (remoteVideoRef.value) {
    const video = remoteVideoRef.value
    if (video.readyState >= 1 && video.srcObject) {
      await video.requestPictureInPicture()
    }
  }
}

const screenShareOverlayRef = ref<HTMLElement | null>(null)

const startDrag = (e: MouseEvent) => {
  e.preventDefault()
  const element = screenShareOverlayRef.value
  if (!element) return
  
  const rect = element.getBoundingClientRect()
  
  dragState.value = {
    isDragging: true,
    startX: e.clientX,
    startY: e.clientY,
    elementX: rect.left,
    elementY: rect.top
  }
  
  document.addEventListener('mousemove', onDrag)
  document.addEventListener('mouseup', stopDrag)
}

const onDrag = (e: MouseEvent) => {
  if (!dragState.value.isDragging) return
  
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
  dragState.value.isDragging = false
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
}

const startFloatingDrag = (e: MouseEvent) => {
  const target = e.target as HTMLElement
  if (target.closest('.floating-actions')) return
  
  e.preventDefault()
  
  const element = document.querySelector('.floating-window') as HTMLElement
  if (!element) return
  
  const rect = element.getBoundingClientRect()
  
  dragState.value = {
    isDragging: true,
    startX: e.clientX,
    startY: e.clientY,
    elementX: window.innerWidth - rect.right,
    elementY: window.innerHeight - rect.bottom
  }
  
  document.addEventListener('mousemove', onFloatingDrag)
  document.addEventListener('mouseup', stopFloatingDrag)
}

const onFloatingDrag = (e: MouseEvent) => {
  if (!dragState.value.isDragging) return
  
  const deltaX = e.clientX - dragState.value.startX
  const deltaY = e.clientY - dragState.value.startY
  
  let newX = dragState.value.elementX - deltaX
  let newY = dragState.value.elementY - deltaY
  
  const element = document.querySelector('.floating-window') as HTMLElement
  if (element) {
    const rect = element.getBoundingClientRect()
    newX = Math.max(0, Math.min(newX, window.innerWidth - rect.width))
    newY = Math.max(0, Math.min(newY, window.innerHeight - rect.height))
  }
  
  floatingPosition.value = {
    x: newX,
    y: newY
  }
}

const stopFloatingDrag = () => {
  dragState.value.isDragging = false
  document.removeEventListener('mousemove', onFloatingDrag)
  document.removeEventListener('mouseup', stopFloatingDrag)
}

defineExpose({
  startScreenShare: initiateShare,
  stopReceiving: stopShare,
  establishConnection: establishConnection,
  handleOffer: async (signal: any, fromUserId: number) => {
    logger.log('ScreenShare: 处理WebRTC offer，来自用户:', fromUserId)
    logger.log('ScreenShare: signal.type:', signal?.type)
    logger.log('ScreenShare: signal.sdp 长度:', signal?.sdp?.length)

    // 设置当前用户为观看者
    isViewer.value = true
    hasJoined.value = true
    isMinimized.value = false // 确保不是最小化状态
    // 保存发送者 ID
    internalSenderId.value = fromUserId

    logger.log('ScreenShare: isViewer:', isViewer.value, 'hasJoined:', hasJoined.value, 'isMinimized:', isMinimized.value)

    // 等待 DOM 更新（确保视频元素渲染完成）
    await nextTick()
    await new Promise(resolve => setTimeout(resolve, 100))
    
    logger.log('ScreenShare: DOM 更新完成，remoteVideoRef:', remoteVideoRef.value)

    const tryHandleOffer = (retries = 10) => {
      // 始终使用 remoteVideoRef 建立连接（它始终在 DOM 中）
      const videoElement = remoteVideoRef.value
      
      logger.log('ScreenShare: tryHandleOffer - 视频元素:', videoElement, '剩余重试:', retries)
      
      if (videoElement) {
        logger.log('ScreenShare: 视频元素已就绪，开始初始化接收器')
        logger.log('ScreenShare: 视频元素 readyState:', videoElement.readyState)
        logger.log('ScreenShare: 视频元素 networkState:', videoElement.networkState)
        
        // 初始化 screenShareReceiver
        screenShareReceiver.init(videoElement, async (stream: MediaStream) => {
          logger.log('ScreenShare: 收到远程流回调')
          logger.log('Stream ID:', stream?.id)
          logger.log('Stream tracks:', stream?.getTracks()?.map((t: MediaStreamTrack) => ({ kind: t.kind, id: t.id, enabled: t.enabled, muted: t.muted })))
          remoteStreamActive.value = true
          
          // 设置到 remoteVideoRef
          if (remoteVideoRef.value) {
            logger.log('ScreenShare: 设置 remoteVideoRef.srcObject')
            logger.log('ScreenShare: remoteVideoRef.value:', remoteVideoRef.value)
            logger.log('ScreenShare: remoteVideoRef.value.paused:', remoteVideoRef.value.paused)
            
            // 检查视频元素的尺寸
            const rect = remoteVideoRef.value.getBoundingClientRect()
            logger.log('ScreenShare: 视频元素尺寸:', rect.width, 'x', rect.height)
            logger.log('ScreenShare: 视频元素位置:', rect.left, rect.top)
            logger.log('ScreenShare: 视频元素 display:', window.getComputedStyle(remoteVideoRef.value).display)
            logger.log('ScreenShare: 视频元素 visibility:', window.getComputedStyle(remoteVideoRef.value).visibility)
            
            remoteVideoRef.value.srcObject = stream
            
            logger.log('ScreenShare: srcObject 已设置，尝试播放')
            logger.log('ScreenShare: remoteVideoRef.value.srcObject:', remoteVideoRef.value.srcObject)
            
            // 尝试播放视频（不等待元数据加载）
            try {
              logger.log('ScreenShare: 调用 play()')
              await remoteVideoRef.value.play()
              logger.log('ScreenShare: 远程视频播放成功')
              logger.log('ScreenShare: 播放后 paused:', remoteVideoRef.value.paused)
              logger.log('ScreenShare: 播放后 videoWidth:', remoteVideoRef.value.videoWidth)
              logger.log('ScreenShare: 播放后 videoHeight:', remoteVideoRef.value.videoHeight)
            } catch (err: any) {
              console.error('远程视频播放失败:', err)
              logger.log('ScreenShare: 播放失败，错误名称:', err.name)
              logger.log('ScreenShare: 播放失败，错误消息:', err.message)
              
              // 如果自动播放被阻止，尝试静音后再播放
              if (err.name === 'NotAllowedError') {
                logger.log('ScreenShare: 自动播放被阻止，尝试静音后播放')
                remoteVideoRef.value.muted = true
                await remoteVideoRef.value.play().catch((e: any) => console.error('静音后播放仍然失败:', e))
              }
            }
          } else {
            console.error('ScreenShare: remoteVideoRef 为 null，无法设置视频流')
            logger.log('ScreenShare: remoteVideoRef 为 null')
          }
        })
        
        logger.log('ScreenShare: 调用 screenShareReceiver.handleOffer')
        screenShareReceiver.handleOffer(signal, fromUserId)
      } else if (retries > 0) {
        logger.log('ScreenShare: 视频元素还未初始化，100ms后重试，剩余重试次数:', retries)
        setTimeout(() => tryHandleOffer(retries - 1), 100)
      } else {
        console.error('ScreenShare: 无法处理offer，视频元素为null，已重试10次')
      }
    }
    
    tryHandleOffer()
  },
  handleIceCandidate: (candidate: any) => {
    screenShareReceiver.addIceCandidate(candidate)
  },
  receiveScreenShareStream: async () => {
    isViewer.value = true
    hasJoined.value = true
    remoteStreamActive.value = true
    showFloatingWindow.value = true
    // 先不开启浮窗模式，等待 DOM 更新
    floatingMode.value = false
    logger.log('ScreenShare: receiveScreenShareStream被调用，开始初始化接收流')
    
    // 等待 DOM 更新
    await nextTick()
    await new Promise(resolve => setTimeout(resolve, 50))
    floatingMode.value = true
    
    // 使用 internalSenderId 或 props.senderId
    const actualSenderId = internalSenderId.value || props.senderId
    if (actualSenderId) {
      startReceivingStream()
    } else {
      logger.log('ScreenShare: senderId 还未设置，等待 handleOffer 设置后再初始化')
    }
  },
  showViewer: async () => {
    isViewer.value = true
    hasJoined.value = true
    remoteStreamActive.value = false
    showFloatingWindow.value = true
    // 先不开启浮窗模式，等待 DOM 更新
    floatingMode.value = false
    logger.log('ScreenShare: showViewer被调用，显示浮窗')
    
    // 等待 DOM 更新
    await nextTick()
    await new Promise(resolve => setTimeout(resolve, 50))
    floatingMode.value = true
    
    // 使用 internalSenderId 或 props.senderId
    const actualSenderId = internalSenderId.value || props.senderId
    if (actualSenderId) {
      startReceivingStream()
    } else {
      logger.log('ScreenShare: senderId 还未设置，等待 handleOffer 设置后再初始化')
    }
  }
})

onUnmounted(() => {
  isComponentMounted = false
  stopDurationTimer()
  stopReceivingStream()
  if (isSharing.value && isInitiator.value) {
    screenShareSender.stopScreenShare()
  }
  
  // 清理拖拽相关资源
  if (rafId) {
    cancelAnimationFrame(rafId)
    rafId = null
  }
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
  document.removeEventListener('mousemove', onFloatingDrag)
  document.removeEventListener('mouseup', stopFloatingDrag)
})
</script>

<style scoped>
.screen-share-overlay {
  position: fixed;
  z-index: 9999;
  background: rgba(30, 30, 40, 0.95);
  backdrop-filter: blur(20px);
  border-radius: 16px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  overflow: hidden;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
}

.screen-share-overlay.is-initiator {
  width: 420px;
}

.screen-share-overlay.is-viewer {
  width: 480px;
}

.screen-share-overlay.minimized {
  width: 320px;
  height: auto;
  min-height: 140px;
}

.screen-share-overlay.minimized .screen-share-body,
.screen-share-overlay.minimized .screen-share-controls,
.screen-share-overlay.minimized .initiator-controls {
  display: none;
}

.screen-share-overlay.dragging {
  cursor: grabbing;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.5);
  /* 故意覆盖基础 transition：拖拽时只对 box-shadow 应用快速过渡，
     避免 transform 等属性在拖拽过程中产生延迟感 */
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

.share-indicator {
  display: flex;
  align-items: center;
  gap: 10px;
}

.pulse-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #ef4444;
  animation: pulse 2s infinite;
  position: relative;
}

.pulse-dot::after {
  content: '';
  position: absolute;
  inset: -4px;
  border-radius: 50%;
  background: rgba(239, 68, 68, 0.3);
  animation: pulse-ring 2s infinite;
}

.pulse-dot.active {
  background: #22c55e;
}

.pulse-dot.active::after {
  background: rgba(34, 197, 94, 0.3);
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

@keyframes pulse-ring {
  0% { transform: scale(1); opacity: 1; }
  100% { transform: scale(2); opacity: 0; }
}

.share-label {
  color: #fff;
  font-weight: 600;
  font-size: 14px;
}

.share-name {
  color: rgba(255, 255, 255, 0.7);
  font-size: 12px;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border: none;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.action-btn:hover {
  background: rgba(255, 255, 255, 0.2);
}

.action-btn.join-btn {
  background: linear-gradient(135deg, #22c55e 0%, #16a34a 100%);
}

.action-btn.join-btn:hover {
  background: linear-gradient(135deg, #16a34a 0%, #15803d 100%);
}

.action-btn.join-btn.joined {
  background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
}

.action-btn.join-btn.joined:hover {
  background: linear-gradient(135deg, #dc2626 0%, #b91c1c 100%);
}

.action-btn.close-btn:hover {
  background: rgba(239, 68, 68, 0.3);
}

.screen-share-body {
  padding: 16px;
}

.source-picker {
  background: rgba(0, 0, 0, 0.3);
  border-radius: 12px;
  overflow: hidden;
}

.source-picker-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.source-picker-header h3 {
  margin: 0;
  color: #fff;
  font-size: 14px;
  font-weight: 600;
}

.source-picker-header .close-btn {
  background: none;
  border: none;
  color: rgba(255, 255, 255, 0.7);
  cursor: pointer;
  padding: 4px;
}

.source-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
  padding: 16px;
  max-height: 280px;
  overflow-y: auto;
}

.source-item {
  background: rgba(255, 255, 255, 0.05);
  border: 2px solid transparent;
  border-radius: 10px;
  padding: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.source-item:hover {
  background: rgba(255, 255, 255, 0.1);
  border-color: rgba(255, 255, 255, 0.2);
}

.source-item.selected {
  border-color: #22c55e;
  background: rgba(34, 197, 94, 0.1);
}

.source-thumbnail {
  width: 100%;
  aspect-ratio: 16/9;
  border-radius: 6px;
  overflow: hidden;
  background: rgba(0, 0, 0, 0.3);
  margin-bottom: 6px;
}

.source-thumbnail img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.source-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.4);
  font-size: 24px;
}

.source-name {
  color: rgba(255, 255, 255, 0.9);
  font-size: 11px;
  text-align: center;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.source-picker-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 12px 16px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.btn-secondary {
  padding: 8px 16px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 8px;
  background: transparent;
  color: rgba(255, 255, 255, 0.8);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-secondary:hover {
  background: rgba(255, 255, 255, 0.1);
}

.btn-primary {
  padding: 8px 16px;
  border: none;
  border-radius: 8px;
  background: linear-gradient(135deg, #22c55e 0%, #16a34a 100%);
  color: #fff;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-primary:hover:not(:disabled) {
  background: linear-gradient(135deg, #16a34a 0%, #15803d 100%);
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.video-container {
  position: relative;
  width: 100%;
  aspect-ratio: 16/9;
  background: rgba(0, 0, 0, 0.4);
  border-radius: 10px;
  overflow: hidden;
}

.remote-video {
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: #000;
  display: block;
  position: relative;
  z-index: 1;
}

.waiting-state {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.6);
  gap: 16px;
}

.waiting-icon {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 28px;
}

.waiting-text {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}

.waiting-title {
  color: #fff;
  font-size: 15px;
  font-weight: 500;
}

.waiting-subtitle {
  color: rgba(255, 255, 255, 0.6);
  font-size: 13px;
}

.connecting-state {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.7);
  gap: 12px;
  color: #fff;
  font-size: 14px;
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid rgba(255, 255, 255, 0.2);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.initiator-prompt {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.start-share-btn {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 24px;
  border: none;
  border-radius: 12px;
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  box-shadow: 0 4px 16px rgba(59, 130, 246, 0.4);
}

.start-share-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(59, 130, 246, 0.5);
}

.screen-share-controls {
  display: flex;
  justify-content: center;
  gap: 12px;
  padding: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.control-btn {
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.control-btn:hover {
  background: rgba(255, 255, 255, 0.2);
}

.control-btn.danger {
  background: rgba(239, 68, 68, 0.2);
  color: #fca5a5;
}

.control-btn.danger:hover {
  background: rgba(239, 68, 68, 0.4);
}

.initiator-controls {
  padding: 0 16px 16px;
}

.control-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 10px;
}

.sharing-status {
  display: flex;
  align-items: center;
  gap: 8px;
  color: rgba(255, 255, 255, 0.8);
  font-size: 12px;
}

.control-buttons {
  display: flex;
  gap: 8px;
}

.floating-window {
  position: fixed;
  z-index: 10000;
  width: 280px;
  background: rgba(30, 30, 40, 0.95);
  backdrop-filter: blur(20px);
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  overflow: hidden;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.floating-window:hover {
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.5);
}

.floating-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.1) 0%, rgba(255, 255, 255, 0.05) 100%);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  cursor: move;
  user-select: none;
}

.floating-title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #fff;
  font-size: 12px;
  font-weight: 500;
}

.floating-title i {
  color: #22c55e;
}

.floating-actions {
  display: flex;
  gap: 6px;
}

.floating-actions button {
  width: 24px;
  height: 24px;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: rgba(255, 255, 255, 0.7);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  transition: all 0.2s;
}

.floating-actions button:hover {
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
}

.floating-actions button.close:hover {
  background: rgba(239, 68, 68, 0.3);
  color: #fca5a5;
}

.floating-body {
  height: 158px;
  background: rgba(0, 0, 0, 0.4);
}

.floating-video {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.floating-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.3);
  font-size: 32px;
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
