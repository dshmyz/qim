<!--
  MessageTimeline.vue - 时间线模式消息组件

  功能：
  - 显示时间线消息列表
  - 显示时间线圆点和连接线
  - 支持创建者标识

  使用示例：
  <MessageTimeline
    :messages="messages"
    :creator-id="channel.creator_id"
  />
-->
<template>
  <div class="message-timeline" role="feed" aria-label="消息时间线">
    <div
      v-for="(message, index) in messages"
      :key="message.id"
      class="timeline-item"
      role="article"
      :aria-label="`来自 ${getSenderName(message)} 的消息`"
    >
      <div class="timeline-marker">
        <div class="timeline-dot" :class="{ 'creator-dot': isCreator(message) }"></div>
        <div v-if="index < messages.length - 1" class="timeline-line"></div>
      </div>
      <div class="timeline-content">
        <div class="timeline-header">
          <img
            :src="getAvatarUrl(message.sender?.avatar, getSenderName(message), serverUrl)"
            :alt="`${getSenderName(message)}的头像`"
            class="timeline-avatar"
          />
          <div class="timeline-info">
            <span class="timeline-sender">
              {{ getSenderName(message) }}
              <span v-if="isCreator(message)" class="creator-badge">创建者</span>
            </span>
            <span class="timeline-time">{{ formatTime(message.created_at) }}</span>
          </div>
        </div>
        <div class="timeline-body">
          <p class="timeline-text">{{ message.content }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { getAvatarUrl } from '../../utils/avatar'
import { API_BASE_URL } from '../../config'
import { useChatUtils } from '../../composables/useChatUtils'
import type { ChannelMessage } from '../../types'

const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

interface Props {
  messages: ChannelMessage[]
  creatorId?: string | number
}

const props = withDefaults(defineProps<Props>(), {
  creatorId: ''
})

const { formatTime } = useChatUtils()

const getSenderName = (message: ChannelMessage): string => {
  return message.sender?.name || '未知用户'
}

const isCreator = (message: ChannelMessage): boolean => {
  if (!props.creatorId) return false
  return String(message.sender_id) === String(props.creatorId)
}
</script>

<style scoped>
.message-timeline {
  position: relative;
  padding: var(--spacing-2) 0;
}

.timeline-item {
  display: flex;
  gap: var(--spacing-4);
  position: relative;
  padding-bottom: var(--spacing-4);
}

.timeline-item:last-child {
  padding-bottom: 0;
}

.timeline-marker {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex-shrink: 0;
  width: 12px;
}

.timeline-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--primary-color);
  flex-shrink: 0;
  z-index: 1;
}

.timeline-dot.creator-dot {
  background: var(--success-color);
  box-shadow: 0 0 0 3px rgba(103, 194, 58, 0.2);
}

.timeline-line {
  flex: 1;
  width: 2px;
  background: var(--border-color);
  margin-top: var(--spacing-1);
  min-height: 20px;
}

.timeline-content {
  flex: 1;
  min-width: 0;
  background: var(--card-bg);
  border-radius: var(--radius-lg);
  padding: var(--spacing-3);
  border: 1px solid var(--border-color);
  transition: all var(--transition-fast);
}

.timeline-content:hover {
  box-shadow: var(--shadow-sm);
  border-color: var(--primary-color);
}

.timeline-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
  margin-bottom: var(--spacing-2);
}

.timeline-avatar {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-md);
  object-fit: cover;
  flex-shrink: 0;
}

.timeline-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
}

.timeline-sender {
  font-weight: var(--font-weight-medium);
  font-size: var(--font-size-sm);
  color: var(--text-color);
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
}

.creator-badge {
  font-size: var(--font-size-xs);
  padding: 1px var(--spacing-2);
  background: var(--primary-color);
  color: white;
  border-radius: var(--radius-sm);
  font-weight: var(--font-weight-medium);
}

.timeline-time {
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
}

.timeline-body {
  margin: 0;
}

.timeline-text {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--text-color);
  line-height: 1.6;
  word-break: break-word;
  white-space: pre-wrap;
}
</style>
