<template>
  <div v-if="visible" class="call-modal" @click="handleEndCall">
    <div class="call-modal-content" @click.stop>
      <div class="call-modal-header">
        <h3>{{ currentCallType === 'voice' ? '语音通话' : '视频通话' }}</h3>
      </div>
      <div class="call-modal-body">
        <!-- 视频通话区域 -->
        <div v-if="currentCallType === 'video'" class="video-container">
          <!-- 远程视频（大窗口） -->
          <div class="remote-video">
            <video
              v-if="remoteStream"
              ref="remoteVideoRef"
              class="video-element"
              autoplay
              playsinline
            ></video>
            <div v-else class="video-placeholder">
              <i class="fas fa-user"></i>
              <span>{{ name || '对方' }}</span>
            </div>
          </div>

          <!-- 本地视频预览（小窗口悬浮） -->
          <div v-if="currentStatus === 'answered' && showLocalPreview" class="local-video">
            <video
              v-if="localStream && isVideoEnabled"
              ref="localVideoRef"
              class="video-element"
              autoplay
              playsinline
              muted
            ></video>
            <div v-else class="video-placeholder small">
              <i class="fas fa-user"></i>
            </div>
            
            <!-- 隐藏本地视频预览按钮 -->
            <button class="local-video-close-btn" @click.stop="handleToggleLocalPreview" title="隐藏己方预览">
              <i class="fas fa-times"></i>
            </button>
          </div>
          
          <!-- 本地预览隐藏后显示的重启按钮 -->
          <div v-if="currentStatus === 'answered' && !showLocalPreview" class="local-video-toggle-btn">
            <button @click.stop="handleToggleLocalPreview" title="显示己方预览">
              <i class="fas fa-user"></i>
            </button>
          </div>

          <!-- 通话信息覆盖层 -->
          <div v-if="currentStatus !== 'answered'" class="call-info">
            <div class="call-avatar">
              <img :src="avatar" :alt="name || '未知'" />
            </div>
            <div class="call-name">{{ name || '未知' }}</div>
            <div class="call-status">
              <span v-if="currentStatus === 'ringing'" class="status-ringing">正在呼叫...</span>
              <span v-else-if="currentStatus === 'ended'" class="status-ended">通话结束</span>
            </div>
          </div>
        </div>

        <!-- 语音通话时显示的头像和信息 -->
        <div v-else class="call-info">
          <div class="call-avatar">
            <img :src="avatar" :alt="name || '未知'" />
          </div>
          <div class="call-name">{{ name || '未知' }}</div>
          <div class="call-status">
            <span v-if="currentStatus === 'ringing'" class="status-ringing">正在呼叫...</span>
            <span v-else-if="currentStatus === 'answered'" class="status-answered">通话中</span>
            <span v-else-if="currentStatus === 'ended'" class="status-ended">通话结束</span>
          </div>
        </div>
      </div>
      <div class="call-modal-footer">
        <!-- 通话控制按钮 -->
        <template v-if="status === 'answered'">
          <!-- 静音按钮 -->
          <button
            class="call-btn control-btn"
            :class="{ 'control-btn-active': isMuted }"
            @click.stop="handleToggleMute"
            :title="isMuted ? '取消静音' : '静音'"
          >
            <i :class="isMuted ? 'fas fa-microphone-slash' : 'fas fa-microphone'"></i>
            <span>{{ isMuted ? '静音' : '麦克风' }}</span>
          </button>

          <!-- 视频开关按钮（仅视频通话显示） -->
          <button
            v-if="callType === 'video'"
            class="call-btn control-btn"
            :class="{ 'control-btn-active': !isVideoEnabled }"
            @click.stop="handleToggleVideo"
            :title="isVideoEnabled ? '关闭摄像头' : '开启摄像头'"
          >
            <i :class="isVideoEnabled ? 'fas fa-video' : 'fas fa-video-slash'"></i>
            <span>{{ isVideoEnabled ? '摄像头' : '已关闭' }}</span>
          </button>

          <!-- 结束通话按钮 -->
          <button class="call-btn end-btn" @click.stop="handleEndCall">
            <i class="fas fa-phone-slash"></i>
            <span>结束通话</span>
          </button>
        </template>

        <!-- 呼入时显示的按钮 -->
        <template v-else-if="status === 'ringing'">
          <button class="call-btn reject-btn" @click.stop="handleRejectCall">
            <i class="fas fa-phone-slash"></i>
            <span>拒绝</span>
          </button>
          <button class="call-btn answer-btn" @click.stop="handleAnswerCall">
            <i class="fas fa-phone"></i>
            <span>接听</span>
          </button>
        </template>

        <!-- 通话结束时显示关闭按钮 -->
        <template v-else>
          <button class="call-btn close-btn" @click.stop="handleClose">
            <i class="fas fa-times"></i>
            <span>关闭</span>
          </button>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useVideoCall } from '../../composables/useVideoCall'

interface Props {
  visible: boolean
  callType?: 'voice' | 'video' | ''
  status?: 'ringing' | 'answered' | 'ended' | ''
  avatar: string
  name: string
}

const props = withDefaults(defineProps<Props>(), {
  callType: '',
  status: ''
})

const emit = defineEmits<{
  (e: 'reject-call'): void
  (e: 'answer-call'): void
  (e: 'end-call'): void
  (e: 'close'): void
}>()

// 集成 useVideoCall
const {
  callStatus,
  callType: videoCallType,
  localStream,
  remoteStream,
  isMuted,
  isVideoEnabled,
  toggleMute,
  toggleVideo,
  endCall: endVideoCall,
  rejectCall: rejectVideoCall,
  answerCall: answerVideoCall
} = useVideoCall()

// 视频元素引用
const localVideoRef = ref<HTMLVideoElement | null>(null)
const remoteVideoRef = ref<HTMLVideoElement | null>(null)

// 本地预览窗口显示状态
const showLocalPreview = ref(true)

// 计算属性：优先使用 props 的值，否则使用 useVideoCall 的值
const currentCallType = computed(() => {
  return props.callType || videoCallType.value
})

const currentStatus = computed(() => {
  return props.status || callStatus.value
})

// 监听 localStream 变化，自动设置 video 元素的 srcObject
watch(localStream, (stream) => {
  if (localVideoRef.value && stream) {
    localVideoRef.value.srcObject = stream
  }
})

// 监听 remoteStream 变化，自动设置 video 元素的 srcObject
watch(remoteStream, (stream) => {
  if (remoteVideoRef.value && stream) {
    remoteVideoRef.value.srcObject = stream
  }
})

// 监听通话状态变化，通话开始时重置预览显示状态
watch(currentStatus, (newStatus) => {
  if (newStatus === 'answered') {
    showLocalPreview.value = true
  }
})

// 处理静音切换
const handleToggleMute = () => {
  toggleMute()
}

// 处理视频开关切换
const handleToggleVideo = () => {
  toggleVideo()
}

// 处理本地预览窗口显示/隐藏
const handleToggleLocalPreview = () => {
  showLocalPreview.value = !showLocalPreview.value
}

// 处理拒绝通话
const handleRejectCall = () => {
  rejectVideoCall()
  emit('reject-call')
}

// 处理接听通话
const handleAnswerCall = () => {
  answerVideoCall()
  emit('answer-call')
}

// 处理结束通话
const handleEndCall = () => {
  endVideoCall()
  emit('end-call')
}

// 处理关闭（通话结束后）
const handleClose = () => {
  emit('close')
}
</script>

<style scoped>
/* 通话模态框样式 */
.call-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
  backdrop-filter: blur(10px);
}

.call-modal-content {
  background: var(--sidebar-bg);
  border-radius: 16px;
  width: 90%;
  max-width: 500px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 15px 30px rgba(0, 0, 0, 0.3);
  animation: modalFadeIn 0.3s ease;
}

.call-modal-header {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 20px;
  background: var(--sidebar-bg);
  border-bottom: 1px solid var(--border-color);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.call-modal-header h3 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--text-color);
}

.call-modal-body {
  flex: 1;
  padding: 32px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: var(--content-bg);
  position: relative;
  min-height: 400px;
}

.call-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 32px;
}

.call-avatar {
  width: 120px;
  height: 120px;
  border-radius: 50%;
  overflow: hidden;
  margin-bottom: 20px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  border: 3px solid var(--primary-color);
}

.call-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.call-name {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-color);
  margin-bottom: 12px;
}

.call-status {
  font-size: 16px;
  color: var(--text-secondary);
}

.status-ringing {
  animation: pulse 1.5s ease-in-out infinite;
  color: #ff9800;
}

.status-answered {
  color: #4caf50;
}

.status-ended {
  color: #f44336;
}

/* 视频通话区域 */
.video-container {
  width: 100%;
  height: 300px;
  position: relative;
  border-radius: 12px;
  overflow: hidden;
  background: var(--sidebar-bg);
}

.remote-video {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #1a1a1a;
}

.remote-video .video-element {
  width: 100%;
  height: 100%;
  object-fit: cover;
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
  border: 2px solid var(--primary-color);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.local-video .video-element {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transform: scaleX(-1);
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
  border: 2px solid var(--primary-color);
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

.video-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  background: var(--sidebar-bg);
  color: var(--text-secondary);
}

.video-placeholder i {
  font-size: 48px;
  margin-bottom: 12px;
  opacity: 0.5;
}

.video-placeholder span {
  font-size: 14px;
  font-weight: 500;
}

.video-placeholder.small {
  padding: 8px;
}

.video-placeholder.small i {
  font-size: 24px;
  margin-bottom: 0;
}

.call-modal-footer {
  display: flex;
  justify-content: center;
  gap: 16px;
  padding: 24px;
  background: var(--sidebar-bg);
  border-top: 1px solid var(--border-color);
  box-shadow: 0 -2px 4px rgba(0, 0, 0, 0.05);
}

.call-btn {
  min-width: 70px;
  height: 70px;
  border-radius: 50%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border: none;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.call-btn i {
  font-size: 22px;
  margin-bottom: 4px;
}

.reject-btn {
  background: #f44336;
  color: #fff;
}

.reject-btn:hover {
  background: #d32f2f;
  transform: scale(1.1);
  box-shadow: 0 6px 16px rgba(244, 67, 54, 0.4);
}

.answer-btn {
  background: #4caf50;
  color: #fff;
}

.answer-btn:hover {
  background: #388e3c;
  transform: scale(1.1);
  box-shadow: 0 6px 16px rgba(76, 175, 80, 0.4);
}

.end-btn {
  background: #f44336;
  color: #fff;
}

.end-btn:hover {
  background: #d32f2f;
  transform: scale(1.1);
  box-shadow: 0 6px 16px rgba(244, 67, 54, 0.4);
}

.close-btn {
  background: #757575;
  color: #fff;
}

.close-btn:hover {
  background: #616161;
  transform: scale(1.1);
  box-shadow: 0 6px 16px rgba(117, 117, 117, 0.4);
}

/* 控制按钮样式 */
.control-btn {
  background: #424242;
  color: #fff;
}

.control-btn:hover {
  background: #616161;
  transform: scale(1.1);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.3);
}

.control-btn-active {
  background: #ff9800;
  color: #fff;
}

.control-btn-active:hover {
  background: #f57c00;
}

/* 动画 */
@keyframes modalFadeIn {
  from {
    opacity: 0;
    transform: scale(0.9);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

/* 暗黑主题适配 */
:deep([data-theme="dark"]) .call-modal-content {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 15px 30px rgba(0, 0, 0, 0.5) !important;
  border: 1px solid var(--border-color) !important;
}

:deep([data-theme="dark"]) .call-modal-header {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3) !important;
  border-bottom: 1px solid var(--border-color) !important;
}

:deep([data-theme="dark"]) .call-modal-header h3 {
  color: var(--text-color) !important;
}

:deep([data-theme="dark"]) .call-modal-body {
  background: var(--secondary-color) !important;
}

:deep([data-theme="dark"]) .call-name {
  color: var(--text-color) !important;
}

:deep([data-theme="dark"]) .call-status {
  color: var(--text-secondary) !important;
}

:deep([data-theme="dark"]) .local-video,
:deep([data-theme="dark"]) .remote-video {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3) !important;
  border: 1px solid var(--border-color) !important;
}

:deep([data-theme="dark"]) .video-placeholder {
  background: var(--sidebar-bg) !important;
  color: var(--text-secondary) !important;
}

:deep([data-theme="dark"]) .call-modal-footer {
  background: var(--sidebar-bg) !important;
  border-top: 1px solid var(--border-color) !important;
  box-shadow: 0 -2px 4px rgba(0, 0, 0, 0.3) !important;
}

:deep([data-theme="dark"]) .control-btn {
  background: #424242;
}

:deep([data-theme="dark"]) .control-btn:hover {
  background: #616161;
}

:deep([data-theme="dark"]) .control-btn-active {
  background: #ff9800;
}
</style>
