<template>
  <div
    class="channel-list-item"
    :class="{ active: isSelected, 'has-unread': channel.unread_count && channel.unread_count > 0 }"
    @click="$emit('select', channel)"
    role="button"
    :aria-label="`选择频道 ${channel.name}`"
    :aria-pressed="isSelected"
    tabindex="0"
    @keydown.enter="$emit('select', channel)"
    @keydown.space.prevent="$emit('select', channel)"
  >
    <ChannelAvatar
      :avatar="channel.avatar"
      :name="channel.name"
      :publish-permission="channel.publish_permission"
      size="md"
      shape="circle"
    />
    <div class="channel-info">
      <div class="channel-name-row">
        <span class="channel-name">{{ channel.name }}</span>
        <span v-if="messageCount > 0" class="message-count-tag">{{ messageCount }}</span>
      </div>
      <div class="channel-desc">{{ channel.description || '暂无描述' }}</div>
      <div v-if="latestMessage" class="channel-latest">
        <i class="fas fa-comment-dots"></i>
        <span class="latest-text">{{ latestMessage }}</span>
      </div>
      <div class="channel-meta-row">
        <span v-if="channel.subscriber_count" class="channel-meta-item">
          <i class="fas fa-users"></i> {{ formatSubscriberCount(channel.subscriber_count) }}
        </span>
        <span v-if="channel.last_active_at" class="channel-meta-item">
          <i class="fas fa-clock"></i> {{ formatRelativeTime(channel.last_active_at) }}
        </span>
        <span v-else-if="channel.messages && channel.messages.length > 0" class="channel-meta-item">
          <i class="fas fa-clock"></i> {{ formatRelativeTime(getLatestMessageTime()) }}
        </span>
      </div>
    </div>
    <div class="channel-actions">
      <span v-if="channel.unread_count && channel.unread_count > 0" class="unread-dot">{{ channel.unread_count }}</span>
      <button
        v-if="channel.is_subscribed && !channel.is_default"
        class="subscribe-btn subscribed"
        @click.stop="$emit('unsubscribe', channel)"
        :aria-label="`取消订阅 ${channel.name}`"
        title="取消订阅"
      >
        <i class="fas fa-check"></i>
      </button>
      <span
        v-else-if="channel.is_subscribed && channel.is_default"
        class="subscribe-btn default-subscribed"
        :aria-label="`默认频道 ${channel.name} 不可取消订阅`"
        title="默认频道，不可取消"
      >
        <i class="fas fa-lock"></i>
      </span>
      <button
        v-else
        class="subscribe-btn"
        @click.stop="$emit('subscribe', channel)"
        :aria-label="`订阅 ${channel.name}`"
        title="订阅频道"
      >
        <i class="fas fa-plus"></i>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { getDisplayName } from '../../utils/avatar'
import type { Channel } from '../../types'
import ChannelAvatar from './ChannelAvatar.vue'

interface Props {
  channel: Channel
  isSelected: boolean
}

const props = defineProps<Props>()

defineEmits<{
  select: [channel: Channel]
  subscribe: [channel: Channel]
  unsubscribe: [channel: Channel]
}>()

const messageCount = computed(() => {
  return props.channel.messages?.length || 0
})

const latestMessage = computed(() => {
  if (!props.channel.messages || props.channel.messages.length === 0) return ''
  const sorted = [...props.channel.messages].sort((a, b) => b.created_at - a.created_at)
  const msg = sorted[0]
  const senderName = getDisplayName(msg.sender)
  const content = msg.content.length > 30 ? msg.content.slice(0, 30) + '...' : msg.content
  return `${senderName}: ${content}`
})

const getLatestMessageTime = (): number => {
  if (!props.channel.messages || props.channel.messages.length === 0) return 0
  const sorted = [...props.channel.messages].sort((a, b) => b.created_at - a.created_at)
  return sorted[0].created_at
}

const formatSubscriberCount = (count: number): string => {
  if (count >= 10000) return `${(count / 10000).toFixed(1)}万`
  if (count >= 1000) return `${(count / 1000).toFixed(1)}k`
  return String(count)
}

const formatRelativeTime = (timestamp: number): string => {
  const now = Date.now()
  const diff = now - timestamp
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}小时前`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days}天前`
  return new Date(timestamp).toLocaleDateString()
}
</script>

<style scoped>
.channel-list-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
  padding: var(--spacing-3);
  border-radius: var(--radius-md);
  cursor: pointer;
  border: 2px solid transparent;
  position: relative;
  transition: background 0.15s;
}

.channel-list-item:hover {
  background: var(--color-gray-100);
}

.channel-list-item.active {
  background: var(--color-gray-100);
  border-color: var(--primary-color);
}

.channel-list-item.has-unread {
  background: rgba(51, 133, 255, 0.04);
}

.channel-list-item:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: -2px;
}

.channel-info {
  flex: 1;
  min-width: 0;
}

.channel-name-row {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
}

.channel-name {
  font-size: 14px;
  font-weight: var(--font-weight-medium);
  color: var(--text-color);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.has-unread .channel-name {
  font-weight: var(--font-weight-semibold);
}

.message-count-tag {
  font-size: 10px;
  padding: 1px 6px;
  background: var(--hover-color);
  color: var(--text-secondary);
  border-radius: var(--radius-full);
  flex-shrink: 0;
}

.channel-desc {
  font-size: 12px;
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-top: 2px;
}

.channel-latest {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 4px;
  font-size: 11px;
  color: var(--text-secondary);
  opacity: 0.8;
}

.channel-latest i {
  font-size: 10px;
  color: var(--primary-color);
  flex-shrink: 0;
}

.latest-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.channel-meta-row {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  margin-top: 4px;
}

.channel-meta-item {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 11px;
  color: var(--text-secondary);
  opacity: 0.7;
}

.channel-meta-item i {
  font-size: 10px;
}

.channel-actions {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
}

.unread-dot {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  font-size: 10px;
  font-weight: var(--font-weight-semibold);
  color: white;
  background: var(--danger-color);
  border-radius: var(--radius-full);
  line-height: 1;
}

.subscribe-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 25px;
  height: 25px;
  border: none;
  background: var(--primary-color);
  color: white;
  border-radius: var(--radius-md);
  cursor: pointer;
}

.subscribe-btn:hover {
  background: var(--primary-dark);
}

.subscribe-btn:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: 2px;
}

.subscribe-btn.subscribed {
  background: var(--success-color);
}

.subscribe-btn.subscribed:hover {
  background: var(--success-dark, #5daf34);
}

.subscribe-btn.default-subscribed {
  background: rgba(156, 163, 175, 0.15);
  color: var(--text-secondary);
  cursor: default;
}
</style>
