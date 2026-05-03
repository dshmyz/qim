<template>
  <div class="channel-header">
    <div class="header-info">
      <ChannelAvatar
        :avatar="channel.avatar"
        :name="channel.name"
        :publish-permission="channel.publish_permission"
        size="sm"
        shape="circle"
      />
      <div class="header-text">
        <div class="header-title-row">
          <h2 class="header-title">{{ channel.name }}</h2>
          <span class="channel-type-tag" :class="channel.publish_permission === 'creator_only' ? 'broadcast' : 'collaborative'">
            {{ channel.publish_permission === 'creator_only' ? '广播' : '协作' }}
          </span>
          <span v-if="messageCount > 0" class="meta-item highlight">
            <i class="fas fa-comment"></i>
            {{ messageCount }}
          </span>
          <span v-if="channel.subscriber_count" class="meta-item">
            <i class="fas fa-users"></i>
            {{ channel.subscriber_count }}
          </span>
        </div>
        <div class="header-subtitle">
          <span class="subtitle-creator">{{ getDisplayName(channel.creator) }}</span>
          <span v-if="channel.description" class="subtitle-sep">·</span>
          <span class="subtitle-desc">{{ channel.description }}</span>
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
import { computed } from 'vue'
import { getDisplayName } from '../../utils/avatar'
import type { Channel } from '../../types'
import ChannelAvatar from './ChannelAvatar.vue'

interface Props {
  channel: Channel
}

const props = defineProps<Props>()

defineEmits<{
  subscribe: [channel: Channel]
  unsubscribe: [channel: Channel]
}>()

const messageCount = computed(() => {
  return props.channel.messages?.length || 0
})
</script>

<style scoped>
.channel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  height: 72px;
  box-sizing: border-box;
  background: var(--sidebar-bg);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  flex-shrink: 0;
}

.header-info {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  min-width: 0;
}

.header-text {
  flex: 1;
  min-width: 0;
}

.header-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.channel-type-tag {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 4px;
  font-weight: 500;
  flex-shrink: 0;
  white-space: nowrap;
}

.channel-type-tag.broadcast {
  color: var(--primary-color);
  background: var(--primary-light, rgba(51, 133, 255, 0.1));
}

.channel-type-tag.collaborative {
  color: var(--success-color);
  background: rgba(103, 194, 58, 0.1);
}

.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 12px;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.meta-item.highlight {
  color: var(--primary-color);
  font-weight: 500;
}

.meta-item i {
  font-size: 11px;
}

.header-subtitle {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 2px;
  overflow: hidden;
}

.subtitle-creator {
  flex-shrink: 0;
}

.subtitle-sep {
  flex-shrink: 0;
}

.subtitle-desc {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.header-actions {
  flex-shrink: 0;
  margin-left: 12px;
}

.subscribe-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border: none;
  border-radius: 6px;
  background: var(--primary-color);
  color: white;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
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
