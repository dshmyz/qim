<template>
  <div v-if="visible" class="call-modal" @click="handleEndCall">
    <div class="call-modal-content" @click.stop>
      <div class="call-modal-header">
        <h3>{{ callType === 'voice' ? '语音通话' : '视频通话' }}</h3>
      </div>
      <div class="call-modal-body">
        <div class="call-info">
          <div class="call-avatar">
            <img :src="avatar" :alt="name || '未知'" />
          </div>
          <div class="call-name">{{ name || '未知' }}</div>
          <div class="call-status">
            <span v-if="status === 'ringing'" class="status-ringing">正在呼叫...</span>
            <span v-else-if="status === 'answered'" class="status-answered">通话中</span>
            <span v-else-if="status === 'ended'" class="status-ended">通话结束</span>
          </div>
        </div>

        <!-- 视频通话区域 -->
        <div v-if="callType === 'video' && status === 'answered'" class="video-container">
          <div class="local-video">
            <div class="video-placeholder">
              <i class="fas fa-user"></i>
              <span>您</span>
            </div>
          </div>
          <div class="remote-video">
            <div class="video-placeholder">
              <i class="fas fa-user"></i>
              <span>{{ name || '对方' }}</span>
            </div>
          </div>
        </div>
      </div>
      <div class="call-modal-footer">
        <button v-if="status === 'ringing'" class="call-btn reject-btn" @click.stop="handleRejectCall">
          <i class="fas fa-phone-slash"></i>
          <span>拒绝</span>
        </button>
        <button v-if="status === 'ringing'" class="call-btn answer-btn" @click.stop="handleAnswerCall">
          <i class="fas fa-phone"></i>
          <span>接听</span>
        </button>
        <button v-else class="call-btn end-btn" @click.stop="handleEndCall">
          <i class="fas fa-phone-slash"></i>
          <span>结束通话</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  visible: boolean
  callType: 'voice' | 'video' | ''
  status: 'ringing' | 'answered' | 'ended' | ''
  avatar: string
  name: string
}

defineProps<Props>()
const emit = defineEmits<{
  (e: 'reject-call'): void
  (e: 'answer-call'): void
  (e: 'end-call'): void
}>()

const handleRejectCall = () => {
  emit('reject-call')
}

const handleAnswerCall = () => {
  emit('answer-call')
}

const handleEndCall = () => {
  emit('end-call')
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
  max-width: 400px;
  margin-top: 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.local-video,
.remote-video {
  width: 100%;
  height: 200px;
  border-radius: 8px;
  overflow: hidden;
  background: var(--sidebar-bg);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-color);
}

.local-video {
  width: 120px;
  height: 90px;
  position: absolute;
  top: 20px;
  right: 20px;
  border: 2px solid var(--primary-color);
  z-index: 10;
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

.call-modal-footer {
  display: flex;
  justify-content: center;
  gap: 24px;
  padding: 24px;
  background: var(--sidebar-bg);
  border-top: 1px solid var(--border-color);
  box-shadow: 0 -2px 4px rgba(0, 0, 0, 0.05);
}

.call-btn {
  width: 60px;
  height: 60px;
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
  font-size: 20px;
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

.screen-share-btn {
  background: #ff9800;
  color: #fff;
  margin-left: 10px;
}

.screen-share-btn:hover {
  background: #f57c00;
  transform: scale(1.1);
  box-shadow: 0 6px 16px rgba(255, 152, 0, 0.4);
}

/* 炫酷黑主题 - 通话模态框 */
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
</style>
