<template>
  <div class="realtime-session-card">
    <div class="session-header">
      <div class="session-type-icon">
        <i :class="typeIcon"></i>
      </div>
      <div class="session-info">
        <div class="session-title">{{ sessionTitle }}</div>
        <div class="session-initiator">{{ initiatorName }}</div>
      </div>
      <div class="session-status" :class="statusClass">
        <span class="status-dot"></span>
        <span class="status-text">{{ statusText }}</span>
      </div>
    </div>

    <div v-if="showViewers && session.participants && session.participants.length > 0" class="viewers-section">
      <div class="viewers-header">
        <i class="fas fa-users"></i>
        <span>观看者 ({{ viewerCount }})</span>
      </div>
      <div class="viewers-list">
        <div
          v-for="participant in viewers"
          :key="participant.id"
          class="viewer-item"
        >
          <img
            v-if="participant.user?.avatar"
            :src="participant.user.avatar"
            :alt="participant.user.nickname"
            class="viewer-avatar"
          />
          <div v-else class="viewer-avatar-placeholder">
            <i class="fas fa-user"></i>
          </div>
          <span class="viewer-name">{{ participant.user?.nickname || `用户${participant.user_id}` }}</span>
        </div>
      </div>
    </div>

    <div class="session-actions">
      <button
        v-if="canJoin"
        class="action-btn join-btn"
        @click="handleJoin"
      >
        <i class="fas fa-eye"></i>
        <span>加入观看</span>
      </button>
      <button
        v-if="canLeave"
        class="action-btn leave-btn"
        @click="handleLeave"
      >
        <i class="fas fa-sign-out-alt"></i>
        <span>离开</span>
      </button>
      <button
        v-if="canEnd"
        class="action-btn end-btn"
        @click="handleEnd"
      >
        <i class="fas fa-stop-circle"></i>
        <span>结束共享</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { RealtimeSession, RealtimeParticipant } from '../../types/realtime'

interface Props {
  session: RealtimeSession
  currentUserId: number
  showViewers?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  showViewers: false
})

const emit = defineEmits<{
  join: []
  leave: []
  end: []
}>()

const typeIcon = computed(() => {
  switch (props.session.type) {
    case 'screen_share':
      return 'fas fa-desktop'
    case 'video_call':
      return 'fas fa-video'
    case 'voice_call':
      return 'fas fa-phone'
    default:
      return 'fas fa-broadcast-tower'
  }
})

const sessionTitle = computed(() => {
  switch (props.session.type) {
    case 'screen_share':
      return '屏幕共享'
    case 'video_call':
      return '视频通话'
    case 'voice_call':
      return '语音通话'
    default:
      return '实时会话'
  }
})

const initiatorName = computed(() => {
  return props.session.initiator?.nickname || `用户${props.session.initiator_id}`
})

const statusClass = computed(() => {
  return {
    'status-active': props.session.status === 'active',
    'status-pending': props.session.status === 'pending',
    'status-paused': props.session.status === 'paused',
    'status-ended': props.session.status === 'ended'
  }
})

const statusText = computed(() => {
  switch (props.session.status) {
    case 'active':
      return '进行中'
    case 'pending':
      return '等待中'
    case 'paused':
      return '已暂停'
    case 'ended':
      return '已结束'
    default:
      return '未知'
  }
})

const viewers = computed(() => {
  if (!props.session.participants) return []
  return props.session.participants.filter(
    (p: RealtimeParticipant) => p.role === 'viewer' && p.status === 'joined'
  )
})

const viewerCount = computed(() => viewers.value.length)

const isInitiator = computed(() => {
  return props.session.initiator_id === props.currentUserId
})

const isParticipant = computed(() => {
  if (!props.session.participants) return false
  return props.session.participants.some(
    (p: RealtimeParticipant) => p.user_id === props.currentUserId && p.status === 'joined'
  )
})

const isActive = computed(() => {
  return props.session.status === 'active' || props.session.status === 'paused'
})

const canJoin = computed(() => {
  return !isInitiator.value && !isParticipant.value && isActive.value
})

const canLeave = computed(() => {
  return isParticipant.value && isActive.value
})

const canEnd = computed(() => {
  return isInitiator.value && isActive.value
})

const handleJoin = () => {
  emit('join')
}

const handleLeave = () => {
  emit('leave')
}

const handleEnd = () => {
  emit('end')
}
</script>

<style scoped>
.realtime-session-card {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 16px;
  padding: 16px;
  color: #fff;
  box-shadow: 0 4px 20px rgba(102, 126, 234, 0.3);
}

.session-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.session-type-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
}

.session-info {
  flex: 1;
  min-width: 0;
}

.session-title {
  font-weight: 600;
  font-size: 15px;
  margin-bottom: 2px;
}

.session-initiator {
  font-size: 12px;
  opacity: 0.8;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-status {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.15);
  font-size: 12px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #fff;
}

.status-active .status-dot {
  background: #22c55e;
  animation: pulse-dot 2s infinite;
}

.status-pending .status-dot {
  background: #f59e0b;
  animation: pulse-dot 1.5s infinite;
}

.status-paused .status-dot {
  background: #6b7280;
}

.status-ended .status-dot {
  background: #9ca3af;
}

@keyframes pulse-dot {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.6;
    transform: scale(1.2);
  }
}

.status-text {
  font-weight: 500;
}

.viewers-section {
  margin-bottom: 12px;
  padding: 10px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 10px;
}

.viewers-header {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  opacity: 0.9;
  margin-bottom: 8px;
}

.viewers-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.viewer-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  font-size: 12px;
}

.viewer-avatar {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  object-fit: cover;
}

.viewer-avatar-placeholder {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
}

.viewer-name {
  max-width: 80px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-actions {
  display: flex;
  gap: 8px;
}

.action-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 16px;
  border: none;
  border-radius: 10px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.join-btn {
  background: rgba(255, 255, 255, 0.2);
  color: #fff;
}

.join-btn:hover {
  background: rgba(255, 255, 255, 0.3);
  transform: translateY(-1px);
}

.leave-btn {
  background: rgba(239, 68, 68, 0.3);
  color: #fff;
}

.leave-btn:hover {
  background: rgba(239, 68, 68, 0.5);
  transform: translateY(-1px);
}

.end-btn {
  background: rgba(239, 68, 68, 0.4);
  color: #fff;
}

.end-btn:hover {
  background: rgba(239, 68, 68, 0.6);
  transform: translateY(-1px);
}
</style>
