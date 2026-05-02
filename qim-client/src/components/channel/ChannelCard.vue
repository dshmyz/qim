<!--
  ChannelCard.vue - 频道卡片组件

  功能：
  - 显示频道的卡片视图
  - 支持选中状态
  - 支持订阅/取消订阅操作

  使用示例：
  <ChannelCard
    :channel="channel"
    :is-selected="selectedChannelId === channel.id"
    @select="handleSelect"
    @subscribe="handleSubscribe"
    @unsubscribe="handleUnsubscribe"
  />
-->
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
      <img
        :src="channelAvatar"
        :alt="`${channel.name}的头像`"
        class="card-avatar"
      />
      <button
        v-if="channel.is_subscribed"
        class="card-subscribe-btn subscribed"
        @click.stop="$emit('unsubscribe', channel)"
        :aria-label="`取消订阅 ${channel.name}`"
        title="取消订阅"
      >
        <i class="fas fa-check"></i> 已订阅
      </button>
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
    <div class="card-body">
      <h4 class="card-title">{{ channel.name }}</h4>
      <p class="card-description">{{ channel.description }}</p>
    </div>
    <div class="card-footer">
      <span class="card-creator">
        <i class="fas fa-user"></i>
        {{ channel.creator?.name || '未知' }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { getAvatarUrl } from '../../utils/avatar'
import { API_BASE_URL } from '../../config'
import type { Channel } from '../../types'

const serverUrl = computed(() => localStorage.getItem('serverUrl') || API_BASE_URL)

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

const channelAvatar = computed(() => {
  return getAvatarUrl(props.channel.avatar, props.channel.name, serverUrl.value)
})
</script>

<style scoped>
.channel-card {
  background: var(--card-bg);
  border-radius: var(--radius-lg);
  padding: var(--spacing-3);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  cursor: pointer;
  border: 1px solid var(--border-color);
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
  align-items: center;
  margin-bottom: var(--spacing-3);
}

.card-avatar {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  object-fit: cover;
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

.card-footer {
  font-size: 11px;
  color: var(--text-secondary);
  border-top: 1px solid var(--border-color);
  padding-top: var(--spacing-2);
  display: flex;
  align-items: center;
  gap: var(--spacing-1);
}
</style>
