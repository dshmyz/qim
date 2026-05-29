<template>
  <div class="channel-detail-new">
    <ChannelHeader
      :channel="channel"
      @subscribe="$emit('subscribe', $event)"
      @unsubscribe="$emit('unsubscribe', $event)"
    />

    <div v-if="!channel.is_subscribed && !isCreator" class="subscribe-banner">
      <i class="fas fa-bell banner-icon"></i>
      <div class="banner-text">
        <span class="banner-title">订阅此频道以参与互动</span>
        <span class="banner-desc">你可以浏览消息，但订阅后才能点赞、评论和发消息</span>
      </div>
    </div>

    <MessageList
      :messages="channel.messages || []"
      :channel="channel"
      :mode="displayMode"
      :is-creator="isCreator"
      :loading="loading"
      :sort-order="sortOrder"
      :creator-id="channel.creator_id"
      :interactive="channel.is_subscribed || isCreator"
      @update:mode="handleModeChange"
      @update:sort-order="handleSortOrderChange"
      @like="handleLike"
      @unlike="handleUnlike"
      @comment="handleComment"
      @copy-link="handleCopyLink"
    />

    <div v-if="isCreator" class="message-input-area">
      <textarea
        v-model="localMessage"
        placeholder="输入消息内容..."
        rows="3"
        class="message-textarea"
        @keydown.enter.ctrl="handleSendMessage"
        :aria-label="'消息输入框'"
      ></textarea>
      <div class="input-actions">
        <span class="input-hint">Ctrl + Enter 发送</span>
        <button
          class="send-btn"
          @click="handleSendMessage"
          :disabled="!localMessage.trim()"
          :aria-label="'发送消息'"
        >
          <i class="fas fa-paper-plane"></i>
          <span>发送</span>
        </button>
      </div>
    </div>

    <div v-else-if="channel.publish_permission === 'all_subscribers' && channel.is_subscribed" class="message-input-area">
      <textarea
        v-model="localMessage"
        placeholder="输入消息内容..."
        rows="3"
        class="message-textarea"
        @keydown.enter.ctrl="handleSendMessage"
        :aria-label="'消息输入框'"
      ></textarea>
      <div class="input-actions">
        <span class="input-hint">Ctrl + Enter 发送</span>
        <button
          class="send-btn"
          @click="handleSendMessage"
          :disabled="!localMessage.trim()"
          :aria-label="'发送消息'"
        >
          <i class="fas fa-paper-plane"></i>
          <span>发送</span>
        </button>
      </div>
    </div>

    <div v-else-if="channel.is_subscribed && channel.publish_permission === 'creator_only'" class="message-readonly-hint">
      <i class="fas fa-bullhorn"></i>
      <span>广播频道，仅创建者可发布消息</span>
    </div>

    <div v-else-if="!channel.is_subscribed && !isCreator" class="message-subscribe-bottom">
      <button class="bottom-subscribe-btn" @click="$emit('subscribe', channel)">
        <i class="fas fa-plus"></i>
        订阅频道参与互动
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import ChannelHeader from './ChannelHeader.vue'
import MessageList from './MessageList.vue'
import type { Channel, ChannelMessage } from '../../types'

type DisplayMode = 'card' | 'timeline'

interface Props {
  channel: Channel
  isCreator?: boolean
  loading?: boolean
  initialMessage?: string
  displayMode?: DisplayMode
  sortOrder?: 'asc' | 'desc'
}

const props = withDefaults(defineProps<Props>(), {
  isCreator: false,
  loading: false,
  initialMessage: '',
  displayMode: 'card',
  sortOrder: 'desc'
})

const emit = defineEmits<{
  subscribe: [channel: Channel]
  unsubscribe: [channel: Channel]
  sendMessage: [channel: Channel, message: string]
  'update:displayMode': [mode: DisplayMode]
  'update:sortOrder': [sortOrder: 'asc' | 'desc']
  like: [message: ChannelMessage]
  unlike: [message: ChannelMessage]
  comment: [message: ChannelMessage]
  copyLink: [message: ChannelMessage]
}>()

const localMessage = ref(props.initialMessage)

watch(
  () => props.initialMessage,
  (newValue) => {
    localMessage.value = newValue
  }
)

const handleModeChange = (mode: DisplayMode) => {
  emit('update:displayMode', mode)
}

const handleSortOrderChange = (sortOrder: 'asc' | 'desc') => {
  emit('update:sortOrder', sortOrder)
}

const handleSendMessage = () => {
  if (!localMessage.value.trim()) return

  emit('sendMessage', props.channel, localMessage.value.trim())
  localMessage.value = ''
}

const handleLike = (message: ChannelMessage) => {
  emit('like', message)
}

const handleUnlike = (message: ChannelMessage) => {
  emit('unlike', message)
}

const handleComment = (message: ChannelMessage) => {
  emit('comment', message)
}

const handleCopyLink = (message: ChannelMessage) => {
  emit('copyLink', message)
}
</script>

<style scoped>
.channel-detail-new {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--bg-color);
}

.subscribe-banner {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
  padding: var(--spacing-3) var(--spacing-4);
  background: linear-gradient(135deg, var(--primary-light, rgba(51, 133, 255, 0.08)), rgba(103, 194, 58, 0.06));
  border-bottom: 1px solid var(--border-color);
}

.banner-icon {
  font-size: 20px;
  color: var(--primary-color);
  flex-shrink: 0;
}

.banner-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.banner-title {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
}

.banner-desc {
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
}

.message-input-area {
  padding: var(--spacing-3);
  border-top: 1px solid var(--border-color);
  background: var(--card-bg);
}

.message-textarea {
  width: 100%;
  padding: var(--spacing-2);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  resize: none;
  font-family: inherit;
  font-size: var(--font-size-sm);
  line-height: 1.5;
  background: var(--input-bg, var(--bg-color));
  color: var(--text-color);
}

.message-textarea:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(51, 133, 255, 0.1);
}

.message-textarea::placeholder {
  color: var(--text-secondary);
}

.input-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: var(--spacing-2);
}

.input-hint {
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
}

.send-btn {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  padding: var(--spacing-2) var(--spacing-3);
  border: none;
  border-radius: var(--radius-md);
  background: var(--primary-color);
  color: white;
  cursor: pointer;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
}

.send-btn:hover:not(:disabled) {
  background: var(--primary-dark);
}

.send-btn:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: 2px;
}

.send-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.message-readonly-hint {
  padding: var(--spacing-2) var(--spacing-3);
  border-top: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
  background: var(--hover-color);
}

.message-readonly-hint i {
  font-size: 14px;
  color: var(--primary-color);
}

.message-subscribe-bottom {
  padding: var(--spacing-3) var(--spacing-4);
  border-top: 1px solid var(--border-color);
  display: flex;
  justify-content: center;
  background: var(--hover-color);
}

.bottom-subscribe-btn {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  padding: var(--spacing-2) var(--spacing-4);
  border: none;
  border-radius: var(--radius-md);
  background: var(--primary-color);
  color: white;
  cursor: pointer;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
}

.bottom-subscribe-btn:hover {
  background: var(--primary-dark);
}

.bottom-subscribe-btn:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: 2px;
}
</style>
