<template>
  <div class="user-profile-card">
    <div class="popover-header">
      <div class="popover-avatar-wrap">
        <Avatar
          :src="member.avatar || fallbackAvatar"
          :name="displayName"
          :alt="displayName"
          :server-url="serverUrl"
          size="md"
          class="popover-avatar"
        />
        <span v-if="statusDotClass" class="popover-status-dot" :class="statusDotClass"></span>
      </div>
      <div class="popover-identity">
        <div class="popover-name-row">
          <span class="popover-name">{{ displayName }}</span>
        </div>
        <span class="popover-status-text">{{ statusText }}</span>
      </div>
    </div>

    <div v-if="infoRows.length" class="popover-info">
      <div v-for="row in infoRows" :key="row.icon" class="popover-info-row">
        <i :class="row.icon"></i>
        <span>{{ row.text }}</span>
      </div>
    </div>

    <div v-if="$slots.actions" class="popover-actions">
      <slot name="actions" :member="member"></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Avatar from './Avatar.vue'
import type { ProfileMember } from '../../composables/useProfilePopover'

const props = defineProps<{
  member: ProfileMember
  serverUrl: string
  fallbackAvatar?: string
}>()

const displayName = computed(() => props.member.name || props.member.username || '未知用户')

// bot/系统身份：不显示在线状态点，统一显示「AI 助手」身份标识
const isBotLike = computed(() => props.member.type === 'bot' || props.member.type === 'system')

const statusDotClass = computed(() => {
  if (isBotLike.value) return ''
  switch (props.member.status) {
    case 'online': return 'online'
    case 'busy': return 'busy'
    case 'offline': return 'offline'
    default: return ''
  }
})

const statusText = computed(() => {
  if (isBotLike.value) return 'AI 助手'
  switch (props.member.status) {
    case 'online': return '在线'
    case 'busy': return '忙碌'
    case 'offline': return '离线'
    default: return ''
  }
})

const infoRows = computed(() => {
  const m = props.member
  if (!m) return []
  const rows: { icon: string; text: string }[] = []
  if (m.username) rows.push({ icon: 'fas fa-at', text: m.username })
  const org = [m.department, m.position].filter(Boolean).join(' · ')
  if (org) rows.push({ icon: 'fas fa-briefcase', text: org })
  if (m.email) rows.push({ icon: 'fas fa-envelope', text: m.email })
  if (m.mobile) rows.push({ icon: 'fas fa-mobile-alt', text: m.mobile })
  if (m.signature) rows.push({ icon: 'fas fa-quote-left', text: m.signature })
  if (m.ip) rows.push({ icon: 'fas fa-globe', text: `IP: ${m.ip}` })
  return rows
})
</script>

<style scoped>
/* 随行资料小卡：Teleport 到 body 的 fixed 卡片，样式与群成员名片一致 */
.user-profile-card {
  position: fixed;
  z-index: 1200;
  width: 248px;
  background: var(--card-bg, #fff);
  border-radius: 12px;
  box-shadow: 0 8px 28px rgba(0, 0, 0, 0.16), 0 0 0 1px rgba(0, 0, 0, 0.05);
  padding: 14px 14px 12px;
  box-sizing: border-box;
  max-height: calc(100vh - 16px);
  overflow-y: auto;
  animation: user-profile-card-in 0.14s ease-out;
}

@keyframes user-profile-card-in {
  from { opacity: 0; transform: translateY(-4px); }
  to { opacity: 1; transform: translateY(0); }
}

.user-profile-card .popover-header {
  display: flex;
  align-items: center;
  gap: 10px;
}

.user-profile-card .popover-avatar-wrap {
  position: relative;
  flex-shrink: 0;
}

.user-profile-card .popover-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
}

.user-profile-card .popover-status-dot {
  position: absolute;
  bottom: 0;
  right: 0;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  border: 2px solid var(--card-bg, #fff);
  background-color: #d9d9d9;
}

.user-profile-card .popover-status-dot.online { background-color: #52c41a; }
.user-profile-card .popover-status-dot.busy { background-color: #ff4d4f; }
.user-profile-card .popover-status-dot.offline { background-color: #d9d9d9; }

.user-profile-card .popover-identity {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
  flex: 1;
}

.user-profile-card .popover-name-row {
  display: flex;
  align-items: center;
  gap: 5px;
  min-width: 0;
}

.user-profile-card .popover-name {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--text-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

.user-profile-card .popover-status-text {
  font-size: var(--font-size-xxxs);
  color: var(--text-secondary);
}

.user-profile-card .popover-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--border-color, rgba(0, 0, 0, 0.06));
}

.user-profile-card .popover-info-row {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  line-height: 1.5;
  min-width: 0;
}

.user-profile-card .popover-info-row i {
  flex-shrink: 0;
  width: 14px;
  text-align: center;
  font-size: var(--font-size-xxxs);
  opacity: 0.6;
}

.user-profile-card .popover-info-row span {
  min-width: 0;
  overflow-wrap: break-word;
}

.user-profile-card .popover-actions {
  display: flex;
  gap: 8px;
  margin-top: 12px;
  padding-top: 10px;
  border-top: 1px solid var(--border-color, rgba(0, 0, 0, 0.06));
}
</style>