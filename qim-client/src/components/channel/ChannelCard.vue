<template>
  <div
    class="channel-card"
    :class="{ active: isSelected }"
    @click="$emit('select', channel)"
    role="button"
    :aria-label="`选择频道 ${channel.name}`"
    :aria-pressed="isSelected"
    tabindex="0"
    @keydown.enter="$emit('select', channel)"
    @keydown.space.prevent="$emit('select', channel)"
  >
    <div class="card-header">
      <ChannelAvatar
        :avatar="channel.avatar"
        :name="channel.name"
        :publish-permission="channel.publish_permission"
        size="lg"
        shape="circle"
      />
      <div class="card-header-right">
        <span class="channel-type-tag" :class="channel.publish_permission === 'creator_only' ? 'broadcast' : 'collaborative'">
          <i :class="channel.publish_permission === 'creator_only' ? 'fas fa-bullhorn' : 'fas fa-comments'"></i>
          {{ channel.publish_permission === 'creator_only' ? '广播' : '协作' }}
        </span>
        <button
          v-if="channel.is_subscribed && !channel.is_default"
          class="card-subscribe-btn subscribed"
          @click.stop="$emit('unsubscribe', channel)"
          :aria-label="`取消订阅 ${channel.name}`"
          title="取消订阅"
        >
          <i class="fas fa-check"></i> 已订阅
        </button>
        <span
          v-else-if="channel.is_subscribed && channel.is_default"
          class="card-subscribe-btn default-subscribed"
          :aria-label="`默认频道 ${channel.name} 不可取消订阅`"
          title="默认频道，不可取消"
        >
          <i class="fas fa-lock"></i> 默认
        </span>
        <button
          v-else
          class="card-subscribe-btn"
          @click.stop="$emit('subscribe', channel)"
          :aria-label="`订阅 ${channel.name}`"
          title="订阅频道"
        >
          <i class="fas fa-plus"></i> 订阅
        </button>
      </div>
    </div>
    <div class="card-body">
      <h4 class="card-title">{{ channel.name }}</h4>
      <p class="card-description">{{ channel.description || '暂无描述' }}</p>
      <div v-if="latestMessage" class="card-latest">
        <i class="fas fa-comment-dots"></i>
        <span>{{ latestMessage }}</span>
      </div>
    </div>
    <div class="card-footer">
      <span class="card-creator">
        <i class="fas fa-user"></i>
        {{ getDisplayName(channel.creator) }}
      </span>
      <span v-if="messageCount > 0" class="card-messages">
        <i class="fas fa-comment"></i>
        {{ messageCount }} 条消息
      </span>
      <span v-if="channel.subscriber_count" class="card-subscribers">
        <i class="fas fa-users"></i>
        {{ formatSubscriberCount(channel.subscriber_count) }}
      </span>
      <span v-if="channel.last_active_at" class="card-activity">
        <i class="fas fa-clock"></i>
        {{ formatRelativeTime(channel.last_active_at) }}
      </span>
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
  const content = msg.content.length > 40 ? msg.content.slice(0, 40) + '...' : msg.content
  return content
})

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
.channel-card {
  background: var(--card-bg);
  border-radius: var(--radius-lg);
  padding: var(--spacing-3);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  cursor: pointer;
  border: 1px solid var(--border-color);
  transition: border-color 0.15s, box-shadow 0.15s;
}

.channel-card:hover {
  border-color: var(--primary-color);
}

.channel-card.active {
  border: 2px solid var(--primary-color);
  box-shadow: 0 2px 12px rgba(51, 133, 255, 0.15);
}

.channel-card:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: 2px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: var(--spacing-3);
}

.card-header-right {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: var(--spacing-2);
}

.channel-type-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  padding: 3px var(--spacing-2);
  border-radius: var(--radius-sm);
  font-weight: var(--font-weight-medium);
}

.channel-type-tag.broadcast {
  color: var(--primary-color);
  background: var(--primary-light, rgba(51, 133, 255, 0.1));
}

.channel-type-tag.collaborative {
  color: var(--success-color);
  background: rgba(103, 194, 58, 0.1);
}

.channel-type-tag i {
  font-size: 10px;
}

.card-subscribe-btn {
  padding: var(--spacing-1) var(--spacing-3);
  border: 1px solid var(--primary-color);
  background: var(--card-bg);
  color: var(--primary-color);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: 12px;
  font-weight: var(--font-weight-medium);
  display: flex;
  align-items: center;
  gap: var(--spacing-1);
}

.card-subscribe-btn:hover {
  background: var(--primary-color);
  color: white;
}

.card-subscribe-btn.subscribed {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

.card-subscribe-btn.default-subscribed {
  background: rgba(156, 163, 175, 0.15);
  color: var(--text-secondary);
  border-color: transparent;
  cursor: default;
}

.card-body {
  margin-bottom: var(--spacing-2);
}

.card-title {
  margin: 0 0 var(--spacing-1) 0;
  font-size: 14px;
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-description {
  margin: 0;
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.card-latest {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: var(--spacing-2);
  padding: var(--spacing-1) var(--spacing-2);
  background: var(--hover-color);
  border-radius: var(--radius-sm);
  font-size: 11px;
  color: var(--text-secondary);
}

.card-latest i {
  font-size: 10px;
  color: var(--primary-color);
  flex-shrink: 0;
}

.card-latest span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-footer {
  font-size: 11px;
  color: var(--text-secondary);
  border-top: 1px solid var(--border-color);
  padding-top: var(--spacing-2);
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
  flex-wrap: wrap;
}

.card-creator,
.card-messages,
.card-subscribers,
.card-activity {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}

.card-creator i,
.card-messages i,
.card-subscribers i,
.card-activity i {
  font-size: 10px;
}

.card-messages {
  color: var(--primary-color);
  font-weight: var(--font-weight-medium);
}
</style>
