<template>
  <!-- 关于对话框 -->
  <div v-if="showAboutDialog" class="about-dialog-overlay" @click="$emit('closeAbout')">
    <div class="about-dialog" @click.stop>
      <div class="about-dialog-header">
        <h3>关于</h3>
        <button class="about-dialog-close" @click="$emit('closeAbout')">×</button>
      </div>
      <div class="about-dialog-content">
        <div class="about-dialog-logo">
          <i class="fas fa-comments fa-4x"></i>
        </div>
        <h2>QIM</h2>
        <p class="version">版本: 1.0.0</p>
        <p class="date">发布日期: 2026-04-11</p>
        <p class="author">作者: huangqun@buaa.edu.cn</p>
        <p class="copyright">© 2026 QIM</p>
        <p class="description">一个现代化的即时通讯界面，提供简洁、高效的聊天体验。</p>
      </div>
      <div class="about-dialog-footer">
        <button class="about-dialog-button" @click="$emit('closeAbout')">确定</button>
      </div>
    </div>
  </div>

  <!-- 退出登录确认对话框 -->
  <div v-if="showLogoutDialog" class="logout-dialog-overlay" @click="$emit('cancelLogout')">
    <div class="logout-dialog" @click.stop>
      <div class="logout-dialog-header">
        <h3>退出登录</h3>
        <button class="logout-dialog-close" @click="$emit('cancelLogout')">×</button>
      </div>
      <div class="logout-dialog-content">
        <p class="logout-dialog-message">确定要退出登录吗？</p>
      </div>
      <div class="logout-dialog-footer">
        <button class="logout-dialog-button cancel-button" @click="$emit('cancelLogout')">取消</button>
        <button class="logout-dialog-button confirm-button" @click="$emit('confirmLogout')">确定</button>
      </div>
    </div>
  </div>

  <!-- 检查更新对话框 -->
  <div v-if="showUpdateDialog" class="update-dialog-overlay" @click="$emit('closeUpdate')">
    <div class="update-dialog" @click.stop>
      <div class="update-dialog-header">
        <h3>检查更新</h3>
        <button class="update-dialog-close" @click="$emit('closeUpdate')">×</button>
      </div>
      <div class="update-dialog-content">
        <div v-if="isCheckingUpdate" class="update-loading">
          <div class="loading-spinner"></div>
          <p class="loading-text">正在检查更新...</p>
        </div>
        <div v-else-if="isDownloading" class="update-downloading">
          <div class="download-icon"><i class="fas fa-download"></i></div>
          <p class="download-text">正在下载更新...</p>
          <div class="download-progress">
            <div class="progress-bar">
              <div class="progress-fill" :style="{ width: downloadProgress + '%' }"></div>
            </div>
            <p class="progress-text">{{ downloadProgress }}%</p>
          </div>
        </div>
        <div v-else class="update-result">
          <div class="result-icon" :class="{ 'new-version': hasNewVersion }">
            <i v-if="hasNewVersion" class="fas fa-arrow-circle-up"></i>
            <i v-else class="fas fa-check-circle"></i>
          </div>
          <p class="result-text">{{ updateResult }}</p>
          <p class="version-info">当前版本: 1.0.0</p>
        </div>
      </div>
      <div class="update-dialog-footer">
        <button v-if="hasNewVersion && !isDownloading" class="update-dialog-button update-button" @click="$emit('downloadUpdate')">升级</button>
        <button class="update-dialog-button" @click="$emit('closeUpdate')">关闭</button>
      </div>
    </div>
  </div>

  <!-- 系统消息发布模态框 -->
  <div v-if="showSystemMessageModal" class="user-profile-modal" @click="$emit('closeSystemMessage')">
    <div class="user-profile-content" @click.stop>
      <div class="user-profile-header">
        <h3>发布系统消息</h3>
        <button class="close-btn" @click="$emit('closeSystemMessage')">×</button>
      </div>
      <div class="user-profile-body">
        <div class="profile-info">
          <div class="info-item">
            <label>消息标题</label>
            <input type="text" v-model="localMessage.title" class="profile-input" placeholder="请输入消息标题" />
          </div>
          <div class="info-item">
            <label>消息内容</label>
            <textarea v-model="localMessage.content" class="profile-textarea" placeholder="请输入消息内容" rows="4"></textarea>
          </div>
          <div class="info-item">
            <label>发送范围</label>
            <select v-model="localMessage.target" class="profile-input">
              <option value="all">所有用户</option>
              <option value="group">指定群聊</option>
              <option value="user">指定用户</option>
            </select>
          </div>
          <div v-if="localMessage.target === 'group'" class="info-item">
            <label>选择群聊</label>
            <select v-model="localMessage.groupId" class="profile-input">
              <option v-for="group in groupConversations" :key="group.id" :value="group.id">{{ group.name }}</option>
            </select>
          </div>
          <div v-if="localMessage.target === 'user'" class="info-item">
            <label>选择用户</label>
            <select v-model="localMessage.userId" class="profile-input">
              <option v-for="employee in allEmployees" :key="employee.id" :value="employee.id">{{ employee.name }}</option>
            </select>
          </div>
        </div>
      </div>
      <div class="user-profile-footer">
        <button class="cancel-btn" @click="$emit('closeSystemMessage')">取消</button>
        <button class="save-btn" @click="$emit('sendSystemMessage', { ...localMessage })" :disabled="!localMessage.title || !localMessage.content">发布</button>
      </div>
    </div>
  </div>

  <!-- 语音通话模态框 -->
  <div v-if="showVoiceCallModal" class="voice-call-modal" @click="$emit('endCall')">
    <div class="voice-call-content" @click.stop>
      <div class="voice-call-header">
        <h3>语音通话</h3>
      </div>
      <div class="voice-call-body">
        <div class="call-status">
          <div v-if="callStatus === 'calling'" class="call-status-text">
            <i class="fas fa-phone-alt"></i>
            <span>正在呼叫...</span>
          </div>
          <div v-else-if="callStatus === 'ringing'" class="call-status-text">
            <i class="fas fa-phone-ring"></i>
            <span>对方正在接听...</span>
          </div>
          <div v-else-if="callStatus === 'active'" class="call-status-text">
            <i class="fas fa-phone"></i>
            <span>通话中</span>
            <div class="call-duration">{{ formattedDuration }}</div>
          </div>
          <div v-else-if="callStatus === 'ended'" class="call-status-text">
            <i class="fas fa-phone-slash"></i>
            <span>通话已结束</span>
          </div>
        </div>
      </div>
      <div class="voice-call-footer">
        <button class="end-call-btn" @click="$emit('endCall')">
          <i class="fas fa-phone-slash"></i>
          结束通话
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

interface Conversation {
  id: string | number
  name: string
  type: string
}

interface SystemMessage {
  title: string
  content: string
  target: string
  groupId?: string | number
  userId?: string | number
}

interface Props {
  showAboutDialog: boolean
  showLogoutDialog: boolean
  showUpdateDialog: boolean
  showSystemMessageModal: boolean
  showVoiceCallModal: boolean
  isCheckingUpdate: boolean
  isDownloading: boolean
  downloadProgress: number
  hasNewVersion: boolean
  updateResult: string
  callStatus: string
  formattedDuration: string
  groupConversations: Conversation[]
  allEmployees: any[]
  systemMessage: SystemMessage
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'closeAbout': []
  'cancelLogout': []
  'confirmLogout': []
  'closeUpdate': []
  'downloadUpdate': []
  'closeSystemMessage': []
  'sendSystemMessage': [message: SystemMessage]
  'endCall': []
}>()

const localMessage = ref<SystemMessage>({ ...props.systemMessage })

watch(() => props.systemMessage, (val) => {
  localMessage.value = { ...val }
}, { deep: true })
</script>

<style scoped>
.about-dialog-overlay,
.logout-dialog-overlay,
.update-dialog-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.about-dialog,
.logout-dialog,
.update-dialog {
  background: var(--modal-bg, #fff);
  border-radius: 12px;
  width: 400px;
  overflow: hidden;
}

.about-dialog-header,
.logout-dialog-header,
.update-dialog-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color, #eee);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.about-dialog-header h3,
.logout-dialog-header h3,
.update-dialog-header h3 {
  margin: 0;
  font-size: 16px;
  color: var(--text-color, #333);
}

.about-dialog-close,
.logout-dialog-close,
.update-dialog-close {
  background: none;
  border: none;
  font-size: 20px;
  cursor: pointer;
  color: var(--text-secondary, #999);
}

.about-dialog-content {
  padding: 24px 20px;
  text-align: center;
}

.about-dialog-logo {
  margin-bottom: 16px;
  color: var(--primary-color, #409eff);
}

.about-dialog-content h2 {
  margin: 0 0 8px 0;
  color: var(--text-color, #333);
}

.version,
.date,
.author,
.copyright,
.description {
  margin: 4px 0;
  font-size: 14px;
  color: var(--text-secondary, #999);
}

.about-dialog-footer,
.logout-dialog-footer,
.update-dialog-footer {
  padding: 16px 20px;
  border-top: 1px solid var(--border-color, #eee);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.about-dialog-button,
.logout-dialog-button,
.update-dialog-button {
  padding: 8px 24px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.about-dialog-button {
  background: var(--primary-color, #409eff);
  color: white;
}

.logout-dialog-button.cancel-button,
.update-dialog-button {
  background: var(--btn-bg, #f5f5f5);
  color: var(--text-color, #333);
}

.logout-dialog-button.confirm-button {
  background: #f56c6c;
  color: white;
}

.update-dialog-button.update-button {
  background: var(--primary-color, #409eff);
  color: white;
}

.logout-dialog-content {
  padding: 24px 20px;
}

.logout-dialog-message {
  margin: 0;
  text-align: center;
  font-size: 16px;
  color: var(--text-color, #333);
}

.update-dialog-content {
  padding: 24px 20px;
  min-height: 150px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.update-loading,
.update-downloading,
.update-result {
  text-align: center;
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--border-color, #eee);
  border-top-color: var(--primary-color, #409eff);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto 12px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.loading-text,
.download-text,
.result-text,
.version-info {
  margin: 8px 0;
  color: var(--text-color, #333);
}

.download-icon {
  font-size: 32px;
  color: var(--primary-color, #409eff);
  margin-bottom: 12px;
}

.download-progress {
  margin-top: 16px;
}

.progress-bar {
  height: 8px;
  background: var(--border-color, #eee);
  border-radius: 4px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--primary-color, #409eff);
  transition: width 0.3s ease;
}

.progress-text {
  font-size: 12px;
  color: var(--text-secondary, #999);
}

.result-icon {
  font-size: 48px;
  margin-bottom: 12px;
}

.result-icon.new-version {
  color: var(--primary-color, #409eff);
}

.result-icon:not(.new-version) {
  color: #67c23a;
}

.voice-call-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.voice-call-content {
  background: var(--modal-bg, #fff);
  border-radius: 12px;
  width: 350px;
  overflow: hidden;
}

.voice-call-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color, #eee);
}

.voice-call-header h3 {
  margin: 0;
  font-size: 16px;
  color: var(--text-color, #333);
  text-align: center;
}

.voice-call-body {
  padding: 40px 20px;
  display: flex;
  justify-content: center;
}

.call-status-text {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: var(--text-color, #333);
  font-size: 16px;
}

.call-status-text i {
  font-size: 48px;
  color: var(--primary-color, #409eff);
}

.call-duration {
  font-size: 24px;
  font-weight: 500;
  color: var(--text-color, #333);
}

.voice-call-footer {
  padding: 20px;
  display: flex;
  justify-content: center;
}

.end-call-btn {
  padding: 12px 32px;
  border: none;
  border-radius: 24px;
  background: #f56c6c;
  color: white;
  cursor: pointer;
  font-size: 16px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-profile-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.user-profile-content {
  background: var(--modal-bg, #fff);
  border-radius: 12px;
  width: 500px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
}

.user-profile-header {
  padding: 20px;
  border-bottom: 1px solid var(--border-color, #eee);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.user-profile-header h3 {
  margin: 0;
  font-size: 18px;
  color: var(--text-color, #333);
}

.close-btn {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: var(--text-secondary, #999);
}

.user-profile-body {
  padding: 20px;
  overflow-y: auto;
  flex: 1;
}

.profile-info {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-item label {
  font-size: 12px;
  color: var(--text-secondary, #999);
}

.profile-input,
.profile-textarea {
  padding: 8px 12px;
  border: 1px solid var(--border-color, #ddd);
  border-radius: 4px;
}

.profile-textarea {
  min-height: 80px;
  resize: vertical;
}

.user-profile-footer {
  padding: 16px 20px;
  border-top: 1px solid var(--border-color, #eee);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.cancel-btn {
  padding: 8px 24px;
  border: none;
  border-radius: 4px;
  background: var(--btn-bg, #f5f5f5);
  color: var(--text-color, #333);
  cursor: pointer;
}

.save-btn {
  padding: 8px 24px;
  border: none;
  border-radius: 4px;
  background: var(--primary-color, #409eff);
  color: white;
  cursor: pointer;
}

.save-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
