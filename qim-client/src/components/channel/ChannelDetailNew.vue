<!--
  ChannelDetailNew.vue - 新的频道详情组件

  功能：
  - 显示频道详情（头部 + 消息列表）
  - 支持创建者发送消息
  - 组合使用 ChannelHeader 和 MessageList 组件

  使用示例：
  <ChannelDetailNew
    :channel="channel"
    @subscribe="handleSubscribe"
    @unsubscribe="handleUnsubscribe"
    @send-message="handleSendMessage"
  />
-->
<template>
  <div class="channel-detail-new">
    <!-- 频道头部 -->
    <ChannelHeader
      :channel="channel"
      @subscribe="$emit('subscribe', $event)"
      @unsubscribe="$emit('unsubscribe', $event)"
    />

    <!-- 消息列表 -->
    <div v-if="!channel.is_subscribed" class="subscribe-prompt">
      <div class="prompt-content">
        <i class="fas fa-lock prompt-icon"></i>
        <h3 class="prompt-title">订阅后查看消息</h3>
        <p class="prompt-description">订阅此频道后即可查看所有历史消息</p>
      </div>
    </div>
    <MessageList
      v-else
      :messages="channel.messages || []"
      :mode="displayMode"
      :is-creator="isCreator"
      :loading="loading"
      :sort-order="sortOrder"
      :creator-id="channel.creator_id"
      @update:mode="handleModeChange"
      @update:sort-order="handleSortOrderChange"
      @like="handleLike"
      @unlike="handleUnlike"
      @comment="handleComment"
      @copy-link="handleCopyLink"
    />

    <!-- 创建者消息输入区域 -->
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

    <!-- 订阅者只读提示 -->
    <div v-else-if="channel.is_subscribed" class="message-readonly-hint">
      <i class="fas fa-bullhorn"></i>
      <span>频道为广播模式，仅创建者可发布消息</span>
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

// 消息输入
const localMessage = ref(props.initialMessage)

// 监听 initialMessage prop 变化，更新本地消息
watch(
  () => props.initialMessage,
  (newValue) => {
    localMessage.value = newValue
  }
)

// 切换显示模式
const handleModeChange = (mode: DisplayMode) => {
  emit('update:displayMode', mode)
}

// 切换排序顺序
const handleSortOrderChange = (sortOrder: 'asc' | 'desc') => {
  emit('update:sortOrder', sortOrder)
}

// 发送消息
const handleSendMessage = () => {
  if (!localMessage.value.trim()) return

  emit('sendMessage', props.channel, localMessage.value.trim())
  localMessage.value = ''
}

// 点赞处理
const handleLike = (message: ChannelMessage) => {
  emit('like', message)
}

// 取消点赞处理
const handleUnlike = (message: ChannelMessage) => {
  emit('unlike', message)
}

// 评论处理
const handleComment = (message: ChannelMessage) => {
  emit('comment', message)
}

// 复制链接处理
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

.subscribe-prompt {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-8);
  background: var(--bg-color);
}

.prompt-content {
  text-align: center;
  max-width: 400px;
}

.prompt-icon {
  font-size: 64px;
  color: var(--text-secondary);
  margin-bottom: var(--spacing-4);
  opacity: 0.5;
}

.prompt-title {
  margin: 0 0 var(--spacing-3) 0;
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
}

.prompt-description {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  line-height: 1.5;
}
</style>
