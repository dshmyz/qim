<!--
  ChannelHeader.vue - 频道头部组件

  功能：
  - 显示频道头像、名称、描述
  - 显示订阅按钮（订阅/取消订阅）
  - 显示频道元信息（创建者、创建时间）

  使用示例：
  <ChannelHeader
    :channel="channel"
    @subscribe="handleSubscribe"
    @unsubscribe="handleUnsubscribe"
  />
-->
<template>
  <div class="channel-header">
    <div class="header-info">
      <img
        :src="channel.avatar || generateAvatar(channel.name)"
        :alt="`${channel.name}的头像`"
        class="header-avatar"
      />
      <div class="header-text">
        <h2 class="header-title">{{ channel.name }}</h2>
        <p class="header-description">{{ channel.description }}</p>
        <div class="header-meta">
          <span class="meta-item">
            <i class="fas fa-user"></i>
            {{ channel.creator?.name || '未知' }}
          </span>
          <span v-if="channel.created_at" class="meta-item">
            <i class="fas fa-clock"></i>
            {{ formatTime(channel.created_at) }}
          </span>
        </div>
      </div>
    </div>
    <div class="header-actions">
      <button
        v-if="channel.is_subscribed"
        class="subscribe-btn subscribed"
        @click="$emit('unsubscribe', channel)"
        :aria-label="`取消订阅 ${channel.name}`"
      >
        <i class="fas fa-check"></i>
        <span>已订阅</span>
      </button>
      <button
        v-else
        class="subscribe-btn"
        @click="$emit('subscribe', channel)"
        :aria-label="`订阅 ${channel.name}`"
      >
        <i class="fas fa-plus"></i>
        <span>订阅</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { generateAvatar } from '../../utils/avatar'
import { useChatUtils } from '../../composables/useChatUtils'
import type { Channel } from '../../types'

interface Props {
  channel: Channel
}

defineProps<Props>()

defineEmits<{
  subscribe: [channel: Channel]
  unsubscribe: [channel: Channel]
}>()

const { formatTime } = useChatUtils()
</script>

<style scoped>
.channel-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: var(--spacing-5);
  border-bottom: 1px solid var(--border-color);
  background: var(--card-bg);
}

.header-info {
  display: flex;
  gap: var(--spacing-4);
  flex: 1;
  min-width: 0;
}

.header-avatar {
  width: 64px;
  height: 64px;
  border-radius: var(--radius-lg);
  object-fit: cover;
  flex-shrink: 0;
}

.header-text {
  flex: 1;
  min-width: 0;
}

.header-title {
  margin: 0 0 var(--spacing-2) 0;
  font-size: var(--font-size-xl);
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.header-description {
  margin: 0 0 var(--spacing-3) 0;
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  line-height: 1.5;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.header-meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-4);
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
}

.meta-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-1);
}

.meta-item i {
  font-size: 12px;
}

.header-actions {
  flex-shrink: 0;
  margin-left: var(--spacing-4);
}

.subscribe-btn {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  padding: var(--spacing-2) var(--spacing-4);
  border: 1px solid var(--primary-color);
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--primary-color);
  cursor: pointer;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  transition: all var(--transition-fast);
}

.subscribe-btn:hover {
  background: var(--primary-color);
  color: white;
}

.subscribe-btn:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: 2px;
}

.subscribe-btn.subscribed {
  background: var(--success-color);
  border-color: var(--success-color);
  color: white;
}

.subscribe-btn.subscribed:hover {
  background: var(--success-dark, #5daf34);
  border-color: var(--success-dark, #5daf34);
}
</style>
