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
    <MessageList
      :messages="channel.messages || []"
      :mode="displayMode"
      :is-creator="isCreator"
      :loading="loading"
      @update:mode="handleModeChange"
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
import { ref, computed } from 'vue'
import ChannelHeader from './ChannelHeader.vue'
import MessageList from './MessageList.vue'
import type { Channel, ChannelMessage } from '../../types'

type DisplayMode = 'card' | 'timeline'

interface Props {
  channel: Channel
  isCreator?: boolean
  loading?: boolean
  initialMessage?: string
}

const props = withDefaults(defineProps<Props>(), {
  isCreator: false,
  loading: false,
  initialMessage: ''
})

const emit = defineEmits<{
  subscribe: [channel: Channel]
  unsubscribe: [channel: Channel]
  sendMessage: [channel: Channel, message: string]
  like: [message: ChannelMessage]
  unlike: [message: ChannelMessage]
  comment: [message: ChannelMessage]
  copyLink: [message: ChannelMessage]
}>()

// 显示模式
const displayMode = ref<DisplayMode>('card')

// 消息输入
const localMessage = ref(props.initialMessage)

// 切换显示模式
const handleModeChange = (mode: DisplayMode) => {
  displayMode.value = mode
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
  padding: var(--spacing-4);
  border-top: 1px solid var(--border-color);
  background: var(--card-bg);
}

.message-textarea {
  width: 100%;
  padding: var(--spacing-3);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  resize: none;
  font-family: inherit;
  font-size: var(--font-size-sm);
  line-height: 1.5;
  background: var(--input-bg, var(--bg-color));
  color: var(--text-color);
  transition: border-color var(--transition-fast);
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
  margin-top: var(--spacing-3);
}

.input-hint {
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
}

.send-btn {
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
  transition: all var(--transition-fast);
}

.send-btn:hover:not(:disabled) {
  background: var(--primary-dark);
  transform: translateY(-1px);
}

.send-btn:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: 2px;
}

.send-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none;
}

.message-readonly-hint {
  padding: var(--spacing-3) var(--spacing-4);
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
</style>
