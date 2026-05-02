<!--
  ChannelListItem.vue - 频道列表项组件

  功能：
  - 显示频道的列表视图
  - 支持选中状态
  - 支持订阅/取消订阅操作

  使用示例：
  <ChannelListItem
    :channel="channel"
    :is-selected="selectedChannelId === channel.id"
    @select="handleSelect"
    @subscribe="handleSubscribe"
    @unsubscribe="handleUnsubscribe"
  />
-->
<template>
  <div
    class="channel-list-item"
    :class="{ active: isSelected }"
    @click="$emit('select', channel)"
    role="button"
    :aria-label="`选择频道 ${channel.name}`"
    :aria-pressed="isSelected"
    tabindex="0"
    @keydown.enter="$emit('select', channel)"
    @keydown.space.prevent="$emit('select', channel)"
  >
    <img
      :src="channelAvatar"
      :alt="`${channel.name}的头像`"
      class="channel-avatar"
    />
    <div class="channel-info">
      <div class="channel-name">{{ channel.name }}</div>
      <div class="channel-desc">{{ channel.description }}</div>
    </div>
    <div class="channel-actions">
      <button
        v-if="channel.is_subscribed"
        class="subscribe-btn subscribed"
        @click.stop="$emit('unsubscribe', channel)"
        :aria-label="`取消订阅 ${channel.name}`"
        title="取消订阅"
      >
        <i class="fas fa-check"></i>
      </button>
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
import { ref, computed } from 'vue'
import { getAvatarUrl } from '../../utils/avatar'
import { API_BASE_URL } from '../../config'
import type { Channel } from '../../types'

const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

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
.channel-list-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
  padding: var(--spacing-3);
  border-radius: var(--radius-md);
  cursor: pointer;
  border: 2px solid transparent;
}

.channel-list-item:hover {
  background: var(--color-gray-100);
}

.channel-list-item.active {
  background: var(--color-gray-100);
  border-color: var(--primary-color);
}

.channel-list-item:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: -2px;
}

.channel-avatar {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  object-fit: cover;
  flex-shrink: 0;
}

.channel-info {
  flex: 1;
  min-width: 0;
}

.channel-name {
  font-size: 14px;
  font-weight: var(--font-weight-medium);
  color: var(--text-color);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.channel-desc {
  font-size: 12px;
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-top: 2px;
}

.channel-actions {
  flex-shrink: 0;
}

.subscribe-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
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
</style>
