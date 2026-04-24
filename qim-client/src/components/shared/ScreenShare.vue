<template>
  <Teleport to="body">
    <div v-if="isInitiator || isViewer" ref="screenShareOverlayRef" class="screen-share-overlay" :class="{ 'is-initiator': isInitiator, 'is-viewer': isViewer, 'minimized': isMinimized }">
      <div class="screen-share-header" @mousedown="startDrag">
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
import { ref, computed, watch, onUnmounted } from 'vue'
// @ts-ignore - WebRTC module has no type declarations
// @ts-ignore - WebRTC module has no type declarations
import { screenShareSender, screenShareReceiver } from '../../utils/webrtc'

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

const remoteVideoRef = ref<HTMLVideoElement | null>(null)
const floatingVideoRef = ref<HTMLVideoElement | null>(null)
const floatingStream = ref<MediaStream | null>(null)

const floatingPosition = ref({ x: 20, y: 20 })
const windowPosition = ref({ 
  x: window.innerWidth / 2 - 210, // 420px 宽的一半
  y: window.innerHeight / 2 - 200  // 居中位置
})
const isDragging = ref(false)
const dragOffset = ref({ x: 0, y: 0 })

const floatingWindowStyle = computed(() => ({
  right: `${floatingPosition.value.x}px`,
  bottom: `${floatingPosition.value.y}px`
}))

watch(() => props.senderId, (newId) => {
  if (newId) {
    isViewer.value = true
    hasJoined.value = true
    remoteStreamActive.value = true
    showFloatingWindow.value = true
    floatingMode.value = true
    console.log('ScreenShare: 检测到senderId变化，自动加入共享', newId)
    startReceivingStream()
  }
})

watch(isSharing, (sharing) => {
  if (sharing) {
    startDurationTimer()
  } else {
    stopDurationTimer()
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
  console.log('ScreenShare: initiateShare函数被调用')
  try {
    // 设置为发起者状态
    isInitiator.value = true
    console.log('ScreenShare: 开始获取屏幕源')
    const sources = await getScreenSources()
    console.log('ScreenShare: 获取到屏幕源:', sources)
    if (sources.length > 0) {
      console.log('ScreenShare: 显示源选择器')
      screenSources.value = sources
      showSourcePicker.value = true
    } else {
      console.log('ScreenShare: 没有屏幕源，直接开始共享')
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
      
      console.log('收到屏幕源:', sources);
      
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
  console.log('ScreenShare: startSharing函数被调用')
  if (!selectedSource.value) {
    console.log('ScreenShare: 未选择屏幕源')
    return
  }

  try {
    console.log('ScreenShare: 开始共享，源:', selectedSource.value.name)
    showSourcePicker.value = false
    screenShareName.value = selectedSource.value.name
    isSharing.value = true
    isInitiator.value = true
    
    // 保存选择的屏幕源ID
    selectedSourceId.value = selectedSource.value.id

    // 使用 screenShareSender 开始共享
    if (props.receiverId !== undefined) {
      console.log('ScreenShare: 发送屏幕共享请求，接收者ID:', props.receiverId, '会话ID:', props.conversationId)
      // 发送屏幕共享请求，使用会话ID
      screenShareSender.sendShareRequest(props.receiverId, props.conversationId)
      
      // 等待对方接受
      console.log('ScreenShare: 等待对方接受屏幕共享请求...')
      
      // 发送屏幕共享开始事件
      console.log('ScreenShare: 发送屏幕共享开始事件，接收者ID:', props.receiverId, '会话ID:', props.conversationId)
      emit('screen-share-start', { conversationId: props.conversationId || props.receiverId })
      
      // 开始共享（注意：这里会直接发送 webrtc_offer，实际应该在对方接受后再发送）
      console.log('ScreenShare: 使用 screenShareSender 开始共享，接收者ID:', props.receiverId)
      await screenShareSender.startScreenShare(props.receiverId)
    }
  } catch (error) {
    console.error('ScreenShare: 开始屏幕共享失败:', error)
    isSharing.value = false
    isInitiator.value = false
  }
}

// 当收到对方接受后，开始建立 WebRTC 连接
const establishConnection = async () => {
  console.log('ScreenShare: establishConnection函数被调用')
  
  try {
    // screenShareSender 已经在 startScreenShare 中建立了连接
    // 这里不需要重复建立连接
    console.log('ScreenShare: WebRTC 连接已经在 startScreenShare 中建立')
    
    // 使用本地预览视频元素显示自己的屏幕（预览用）
    if (remoteVideoRef.value) {
      // 获取屏幕共享流并设置到预览视频元素
      const screenStream = screenShareSender.getScreenStream()
      if (screenStream) {
        console.log('ScreenShare: 设置本地预览视频流')
        remoteVideoRef.value.srcObject = screenStream
        // 尝试播放视频
        try {
          remoteVideoRef.value.play().catch(err => {
            console.error('尝试播放预览视频失败:', err)
          })
        } catch (error) {
          console.error('播放预览视频出错:', error)
        }
      } else {
        console.log('ScreenShare: 未获取到屏幕共享流，预览功能暂时不可用')
      }
    }
  } catch (error) {
    console.error('ScreenShare: 建立连接失败:', error)
    isSharing.value = false
    isInitiator.value = false
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
  // 使用 screenShareSender 停止共享
  if (isInitiator.value) {
    screenShareSender.stopScreenShare()
  } else if (isViewer.value) {
    screenShareReceiver.stop()
  }

  if (remoteVideoRef.value?.srcObject) {
    const stream = remoteVideoRef.value.srcObject as MediaStream
    stream.getTracks().forEach(track => track.stop())
    remoteVideoRef.value.srcObject = null
  }

  isSharing.value = false
  isInitiator.value = false
  isPaused.value = false
  isMinimized.value = false
  hasJoined.value = false
  remoteStreamActive.value = false
  showSourcePicker.value = false
  selectedSource.value = null
  screenShareName.value = ''
  floatingStream.value = null

  if (floatingMode.value) {
    closeFloating()
  }

  emit('screen-share-stop')
}

const joinShare = () => {
  hasJoined.value = !hasJoined.value
  if (hasJoined.value) {
    emit('screen-share-join')
    startReceivingStream()
  } else {
    emit('screen-share-leave')
    stopReceivingStream()
  }
}

const startReceivingStream = () => {
  if (props.senderId) {
    // 优先使用浮窗视频元素（如果浮窗模式开启）
    const videoElement = floatingMode.value ? floatingVideoRef.value : remoteVideoRef.value
    
    // 确保视频元素已经初始化
    if (videoElement) {
      console.log('ScreenShare: 初始化screenShareReceiver，视频元素:', videoElement)
      // 初始化screenShareReceiver，传递远程流接收回调
      screenShareReceiver.init(videoElement, (stream) => {
        console.log('ScreenShare: 收到远程流，更新状态')
        remoteStreamActive.value = true
        if (floatingMode.value) {
          // 浮窗模式下通过模板绑定设置srcObject，这里只更新floatingStream
          floatingStream.value = stream
        }
      })
    } else {
      console.log('ScreenShare: 视频元素还未初始化，等待下一帧')
      // 等待下一帧，确保视频元素已经渲染
      setTimeout(() => {
        // 再次检查浮窗模式
        const delayedVideoElement = floatingMode.value ? floatingVideoRef.value : remoteVideoRef.value
        
        if (delayedVideoElement) {
          console.log('ScreenShare: 延迟初始化screenShareReceiver，视频元素:', delayedVideoElement)
          // 初始化screenShareReceiver，传递远程流接收回调
          screenShareReceiver.init(delayedVideoElement, (stream) => {
            console.log('ScreenShare: 收到远程流，更新状态')
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
  if (isMinimized.value && !floatingMode.value) {
    floatingMode.value = true
    floatingStream.value = remoteVideoRef.value?.srcObject as MediaStream
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

const screenShareOverlayRef = ref(null)

const startDrag = (e: MouseEvent) => {
  isDragging.value = true
  dragOffset.value = { x: e.clientX, y: e.clientY }
  document.addEventListener('mousemove', onWindowDrag)
  document.addEventListener('mouseup', stopWindowDrag)
}

const onDrag = (e: MouseEvent) => {
  if (isDragging.value) {
    floatingPosition.value = {
      x: Math.max(0, floatingPosition.value.x + e.clientX - dragOffset.value.x),
      y: Math.max(0, floatingPosition.value.y + e.clientY - dragOffset.value.y)
    }
    dragOffset.value = { x: e.clientX, y: e.clientY }
  }
}

const onWindowDrag = (e: MouseEvent) => {
  if (isDragging.value && screenShareOverlayRef.value) {
    const element = screenShareOverlayRef.value
    const rect = element.getBoundingClientRect()
    
    const newX = rect.left + e.clientX - dragOffset.value.x
    const newY = rect.top + e.clientY - dragOffset.value.y
    
    element.style.left = `${newX}px`
    element.style.top = `${newY}px`
    element.style.transform = 'none'
    
    dragOffset.value = { x: e.clientX, y: e.clientY }
  }
}

const stopDrag = () => {
  isDragging.value = false
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
}

const stopWindowDrag = () => {
  isDragging.value = false
  document.removeEventListener('mousemove', onWindowDrag)
  document.removeEventListener('mouseup', stopWindowDrag)
}

const startFloatingDrag = (e: MouseEvent) => {
  const target = e.target as HTMLElement
  if (target.closest('.floating-actions')) return
  isDragging.value = true
  dragOffset.value = { x: e.clientX, y: e.clientY }
  document.addEventListener('mousemove', onFloatingDrag)
  document.addEventListener('mouseup', stopFloatingDrag)
}

const onFloatingDrag = (e: MouseEvent) => {
  if (isDragging.value) {
    floatingPosition.value = {
      x: Math.max(0, window.innerWidth - 320 - (e.clientX - dragOffset.value.x)),
      y: Math.max(0, window.innerHeight - 200 - (e.clientY - dragOffset.value.y))
    }
    dragOffset.value = { x: e.clientX, y: e.clientY }
  }
}

const stopFloatingDrag = () => {
  isDragging.value = false
  document.removeEventListener('mousemove', onFloatingDrag)
  document.removeEventListener('mouseup', stopFloatingDrag)
}

defineExpose({
  startScreenShare: initiateShare,
  stopReceiving: stopShare,
  establishConnection: establishConnection,
  handleOffer: (signal: any, fromUserId: number) => {
    console.log('ScreenShare: 处理WebRTC offer，来自用户:', fromUserId)
        console.log('ScreenShare: 处理WebRTC signal:', signal)

    const tryHandleOffer = (retries = 10) => {
      // 优先使用浮窗视频元素（如果浮窗模式开启）
      const videoElement = floatingMode.value ? floatingVideoRef.value : remoteVideoRef.value
      
      if (videoElement) {
        console.log('ScreenShare: screenShareReceiver已初始化，使用视频元素:', videoElement)
        // 初始化screenShareReceiver，传递远程流接收回调
        screenShareReceiver.init(videoElement, (stream) => {
          console.log('ScreenShare: 收到远程流，更新状态')
          remoteStreamActive.value = true
          // 只在浮窗模式下设置floatingStream，避免冲突
          if (floatingMode.value) {
            floatingStream.value = stream
          }
        })
        screenShareReceiver.handleOffer(signal, fromUserId)
      } else if (retries > 0) {
        console.log('ScreenShare: 视频元素还未初始化，100ms后重试，剩余重试次数:', retries)
        console.log('ScreenShare: floatingMode:', floatingMode.value)
        console.log('ScreenShare: floatingVideoRef.value:', floatingVideoRef.value)
        console.log('ScreenShare: remoteVideoRef.value:', remoteVideoRef.value)
        setTimeout(() => tryHandleOffer(retries - 1), 100)
      } else {
        console.error('ScreenShare: 无法处理offer，视频元素为null，已重试10次')
        console.error('ScreenShare: floatingMode:', floatingMode.value)
        console.error('ScreenShare: floatingVideoRef.value:', floatingVideoRef.value)
        console.error('ScreenShare: remoteVideoRef.value:', remoteVideoRef.value)
      }
    }
    
    tryHandleOffer()
  },
  handleIceCandidate: (candidate: any) => {
    screenShareReceiver.addIceCandidate(candidate)
  },
  receiveScreenShareStream: () => {
    isViewer.value = true
    hasJoined.value = true
    remoteStreamActive.value = true
    showFloatingWindow.value = true
    floatingMode.value = true
    console.log('ScreenShare: receiveScreenShareStream被调用，开始初始化接收流')
    
    // 初始化screenShareReceiver并开始接收流
    if (props.senderId) {
      startReceivingStream()
    }
  },
  showViewer: () => {
    isViewer.value = true
    hasJoined.value = true
    remoteStreamActive.value = false
    showFloatingWindow.value = true
    floatingMode.value = true
    console.log('ScreenShare: showViewer被调用，显示浮窗')
    
    // 初始化screenShareReceiver并开始接收流
    if (props.senderId) {
      startReceivingStream()
    }
  }
})

onUnmounted(() => {
  stopDurationTimer()
  stopReceivingStream()
  if (isSharing.value && isInitiator.value) {
    screenShareSender.stopScreenShare()
  }
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
  height: 48px;
}

.screen-share-overlay.minimized .screen-share-body,
.screen-share-overlay.minimized .screen-share-controls,
.screen-share-overlay.minimized .initiator-controls {
  display: none;
}

.screen-share-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.1) 0%, rgba(255, 255, 255, 0.05) 100%);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  cursor: move;
  user-select: none;
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
</style>
