<template>
  <div v-if="visible" class="user-profile-modal" @click="close">
    <div class="user-profile-content" @click.stop>
      <!-- 关闭按钮 -->
      <button class="modal-close-btn" @click="close">
        <i class="fas fa-times"></i>
      </button>

      <!-- Hero 头像区 -->
      <div class="profile-hero">
        <div class="profile-avatar-wrapper">
          <Avatar
            :src="user.avatar"
            :name="user.name"
            :server-url="serverUrl"
            :alt="user.name"
            size="xl"
            class="profile-avatar-img"
          />
        </div>
        <div class="profile-name">{{ user.name }}</div>
        <div v-if="user.signature" class="profile-signature">
          {{ user.signature }}
        </div>
        <div class="profile-status-line" :class="{ 'is-online': statusInfo?.status === 'online' }">
          <span class="status-dot" :class="statusInfo?.status || 'offline'"></span>
          {{ statusText }}
        </div>
      </div>

      <!-- 信息区 -->
      <div class="profile-info">
        <div class="info-item">
          <span class="info-icon"><i class="fas fa-at"></i></span>
          <div class="info-content">
            <label>账号</label>
            <span class="info-value">{{ user.username || '未设置' }}</span>
          </div>
        </div>
        <div v-if="user.department || user.position" class="info-item">
          <span class="info-icon"><i class="fas fa-building"></i></span>
          <div class="info-content">
            <label>部门 / 职位</label>
            <span class="info-value">{{ [user.department || '未设置', user.position].filter(Boolean).join(' · ') }}</span>
          </div>
        </div>
        <div class="info-item">
          <span class="info-icon"><i class="fas fa-envelope"></i></span>
          <div class="info-content">
            <label>邮箱</label>
            <span class="info-value">{{ user.email || '未设置' }}</span>
          </div>
        </div>
        <div class="info-item">
          <span class="info-icon"><i class="fas fa-mobile-alt"></i></span>
          <div class="info-content">
            <label>手机</label>
            <span class="info-value">{{ user.mobile || '未设置' }}</span>
          </div>
        </div>
        <div v-if="user.ip" class="info-item">
          <span class="info-icon"><i class="fas fa-globe"></i></span>
          <div class="info-content">
            <label>IP</label>
            <span class="info-value info-value-muted">{{ user.ip }}</span>
          </div>
        </div>
      </div>

      <!-- 操作栏 -->
      <div class="user-profile-footer">
        <button v-if="showAction && !isBot" class="action-btn primary" @click="handleSendPrivateMessage">
          <i class="fas fa-comment"></i>
          <span>发起私聊</span>
        </button>
        <button v-else-if="showAction && isBot" class="action-btn primary" @click="handleSendPrivateMessage">
          <i class="fas fa-robot"></i>
          <span>开始对话</span>
        </button>
        <button class="action-btn secondary" @click="close">
          <span>关闭</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onUnmounted, watch } from 'vue'
import { getStoredServerUrl } from '../../composables/useServerUrl'
import { useUserStatus } from '../../composables/useUserStatus'
import Avatar from '../shared/Avatar.vue'
import { generateAvatar, isAbsoluteUrl } from '../../utils/avatar'

interface User {
  id: string | number
  name: string
  username?: string
  email?: string
  mobile?: string
  department?: string
  position?: string
  signature?: string
  status?: string
  ip?: string
  avatar?: string
  type?: string
}

interface Props {
  visible: boolean
  user: User
  showAction?: boolean
}

const props = withDefaults(defineProps<Props>(), { showAction: true })
const emit = defineEmits<{
  close: []
  sendPrivateMessage: [user: User]
}>()

const close = () => {
  emit('close')
}

const isBot = computed(() => {
  const t = props.user.type
  return t === 'bot'
})

// 用户在线状态
const { subscribeUserStatus, unsubscribeUserStatus, getUserStatus, formatLastOnline } = useUserStatus()

watch(() => props.visible, (visible) => {
  if (visible && props.user?.id) {
    subscribeUserStatus(Number(props.user.id))
  } else if (!visible && props.user?.id) {
    unsubscribeUserStatus(Number(props.user.id))
  }
})

onUnmounted(() => {
  if (props.user?.id) unsubscribeUserStatus(Number(props.user.id))
})

const statusInfo = computed(() => getUserStatus(Number(props.user?.id)))

const avatarBadge = computed(() => {
  if (props.user.type === 'bot' || props.user.type === 'system') {
    return { type: 'icon' as const, icon: 'fas fa-robot', title: 'AI 助手', color: 'var(--primary-color, #1890ff)' }
  }
  const s = statusInfo.value?.status
  if (s) {
    return { type: 'status' as const, status: s, title: s === 'online' ? '在线' : s === 'busy' ? '忙碌' : '离线' }
  }
  return null
})

const statusText = computed(() => {
  if (props.user.type === 'bot' || props.user.type === 'system') return 'AI 助手'
  const s = statusInfo.value?.status
  if (s === 'online') return '在线'
  if (s === 'busy') return '忙碌'
  if (s === 'offline') {
    const lastOnline = statusInfo.value?.lastOnline
    if (lastOnline) return formatLastOnline(lastOnline) || '离线'
    return '离线'
  }
  return '离线'
})

const handleSendPrivateMessage = () => {
  emit('sendPrivateMessage', props.user)
  emit('close')
}

const serverUrl = getStoredServerUrl()

// 头像 URL
const avatarUrl = computed(() => {
  if (!props.user.avatar) {
    return generateAvatar(props.user.name)
  }
  if (isAbsoluteUrl(props.user.avatar)) {
    return props.user.avatar
  }
  return `${serverUrl}${props.user.avatar}`
})
</script>

<style scoped>
.user-profile-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.45);
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.user-profile-content {
  background-color: var(--card-bg, #fff);
  border-radius: 16px;
  width: 400px;
  max-width: 90%;
  max-height: 85vh;
  overflow: hidden;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15), 0 0 0 1px rgba(0, 0, 0, 0.05);
  display: flex;
  flex-direction: column;
  position: relative;
}

.modal-close-btn {
  position: absolute;
  top: 12px;
  right: 12px;
  z-index: 2;
  width: 30px;
  height: 30px;
  border: none;
  background: rgba(255, 255, 255, 0.85);
  color: #666;
  font-size: var(--font-size-xxs);
  cursor: pointer;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.15);
  transition: all 0.2s;
}

.modal-close-btn:hover {
  background: #fff;
  color: #333;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}

/* ---- Hero 区 ---- */
.profile-hero {
  position: relative;
  padding: 32px 24px 20px;
  text-align: center;
  background: linear-gradient(135deg, var(--primary-color) 0%, var(--active-color, #4f6ef7) 100%);
  overflow: hidden;
}

.profile-hero::before {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at 30% 20%, rgba(255,255,255,0.15), transparent 60%);
  pointer-events: none;
}

.profile-avatar-wrapper {
  position: relative;
  display: inline-block;
  margin-bottom: 14px;
}

.profile-avatar-img {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  border: 3px solid rgba(255, 255, 255, 0.5);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
}

.profile-name {
  font-size: var(--font-size-lg);
  font-weight: 700;
  color: #fff;
  text-shadow: 0 1px 3px rgba(0,0,0,0.15);
  margin-bottom: 6px;
}

.profile-signature {
  font-size: var(--font-size-xs);
  color: rgba(255, 255, 255, 0.8);
  line-height: 1.5;
  margin-bottom: 8px;
  padding: 0 8px;
}

.profile-status-line {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: var(--font-size-xxxs);
  color: rgba(255, 255, 255, 0.7);
  background: rgba(0, 0, 0, 0.15);
  padding: 3px 12px;
  border-radius: 20px;
}

.profile-status-line.is-online {
  color: #b7eb8f;
}

.status-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: rgba(255,255,255,0.4);
  flex-shrink: 0;
}

.status-dot.online { background-color: #b7eb8f; }
.status-dot.busy { background-color: #ff7875; }
.status-dot.offline { background-color: rgba(255,255,255,0.4); }

/* ---- 信息区 ---- */
.profile-info {
  padding: 16px 20px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow-y: auto;
  flex: 1;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 8px;
  transition: background-color 0.15s;
}

.info-item:hover {
  background: var(--hover-color, rgba(0,0,0,0.03));
}

.info-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 8px;
  background: var(--hover-color, rgba(0,0,0,0.04));
  color: var(--text-secondary);
  font-size: var(--font-size-xxs);
  flex-shrink: 0;
}

.info-content {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
  flex: 1;
}

.info-content label {
  font-size: var(--font-size-xxxs);
  color: var(--text-secondary, #999);
  font-weight: 500;
}

.info-content .info-value {
  font-size: var(--font-size-xs);
  color: var(--text-color, #333);
  font-weight: 500;
  word-break: break-all;
  line-height: 1.4;
}

.info-content .info-value-muted {
  color: var(--text-secondary, #999);
  font-weight: 400;
  font-size: var(--font-size-xxs);
}

/* ---- 操作栏 ---- */
.user-profile-footer {
  padding: 16px 20px;
  border-top: 1px solid var(--border-color, rgba(0,0,0,0.06));
  display: flex;
  gap: 8px;
}

.action-btn {
  flex: 1;
  padding: 10px 16px;
  border: none;
  border-radius: 10px;
  font-size: var(--font-size-xs);
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.action-btn i {
  font-size: var(--font-size-xs);
}

.action-btn.primary {
  background: var(--primary-color, #3b82f6);
  color: #fff;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.25);
}

.action-btn.primary:hover {
  background: var(--active-color, #2563eb);
  box-shadow: 0 4px 14px rgba(59, 130, 246, 0.35);
  transform: translateY(-1px);
}

.action-btn.secondary {
  background: var(--hover-color, rgba(0,0,0,0.04));
  color: var(--text-color, #333);
}

.action-btn.secondary:hover {
  background: var(--border-color, rgba(0,0,0,0.08));
  transform: translateY(-1px);
}

/* 高雅紫主题 */
[data-theme="elegant-purple"] .action-btn.primary {
  background: var(--primary-color, #75629a);
}

[data-theme="elegant-purple"] .action-btn.primary:hover {
  background: var(--active-color, #665486);
}

/* 中国红主题 */
[data-theme="chinesered"] .action-btn.primary {
  background: var(--primary-color, #c41e3a);
}

[data-theme="chinesered"] .action-btn.primary:hover {
  background: var(--active-color, #a31d32);
}
</style>
