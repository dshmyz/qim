<template>
  <div class="channel-detail-new">
    <ChannelHeader
      :channel="channel"
      @subscribe="$emit('subscribe', $event)"
      @unsubscribe="$emit('unsubscribe', $event)"
      @refresh="$emit('refresh')"
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

    <div v-if="channelUsable && isCreator" class="message-input-area">
      <div class="composer-toolbar">
        <div class="mode-toggle">
          <button
            class="mode-btn"
            :class="{ active: publishMode === 'edit' }"
            @click="publishMode = 'edit'"
          >编辑</button>
          <button
            class="mode-btn"
            :class="{ active: publishMode === 'preview' }"
            @click="publishMode = 'preview'"
          >预览</button>
        </div>
      </div>
      <textarea
        v-if="publishMode === 'edit'"
        v-model="localMessage"
        placeholder="输入消息内容，支持 Markdown（# 标题、**粗体**、[链接](url) 等）"
        rows="5"
        class="message-textarea"
        @keydown.enter.ctrl="handleSendMessage"
        :aria-label="'消息输入框'"
      ></textarea>
      <div v-else class="md-preview" v-html="previewContent"></div>
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

    <div v-else-if="channelUsable && channel.publish_permission === 'all_subscribers' && channel.is_subscribed" class="message-input-area">
      <div class="composer-toolbar">
        <div class="mode-toggle">
          <button
            class="mode-btn"
            :class="{ active: publishMode === 'edit' }"
            @click="publishMode = 'edit'"
          >编辑</button>
          <button
            class="mode-btn"
            :class="{ active: publishMode === 'preview' }"
            @click="publishMode = 'preview'"
          >预览</button>
        </div>
      </div>
      <textarea
        v-if="publishMode === 'edit'"
        v-model="localMessage"
        placeholder="输入消息内容，支持 Markdown（# 标题、**粗体**、[链接](url) 等）"
        rows="5"
        class="message-textarea"
        @keydown.enter.ctrl="handleSendMessage"
        :aria-label="'消息输入框'"
      ></textarea>
      <div v-else class="md-preview" v-html="previewContent"></div>
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

    <div v-else-if="channelUsable && channel.is_subscribed && channel.publish_permission === 'creator_only'" class="message-readonly-hint">
      <i class="fas fa-bullhorn"></i>
      <span>广播频道，仅创建者可发布消息</span>
    </div>

    <div v-else-if="channelUsable && !channel.is_subscribed && !isCreator" class="message-subscribe-bottom">
      <button class="bottom-subscribe-btn" @click="$emit('subscribe', channel)">
        <i class="fas fa-plus"></i>
        订阅频道参与互动
      </button>
    </div>

    <div v-else class="message-readonly-hint">
      <i :class="channelStatusIcon"></i>
      <span>{{ channelStatusText }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import ChannelHeader from './ChannelHeader.vue'
import MessageList from './MessageList.vue'
import { renderChannelMarkdown } from '../../utils/channelMarkdown'
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
  refresh: []
}>()

const localMessage = ref(props.initialMessage)

// 频道是否可发布：仅 active 频道允许发布，审批中/已拒绝/停用则展示禁用提示（服务层同样拦截）
const channelUsable = computed(() => props.channel.status === 'active')
const channelStatusText = computed(() => {
  switch (props.channel.status) {
    case 'pending': return '频道正在审批中，暂不可发布消息'
    case 'rejected': return '频道已被拒绝，暂不可发布消息'
    case 'inactive': return '频道已停用，暂不可发布消息'
    default: return '频道暂不可用，暂不可发布消息'
  }
})
const channelStatusIcon = computed(() => {
  switch (props.channel.status) {
    case 'pending': return 'fas fa-hourglass-half'
    case 'rejected': return 'fas fa-times-circle'
    case 'inactive': return 'fas fa-ban'
    default: return 'fas fa-ban'
  }
})

// 发布侧编辑/预览切换：预览与正文渲染同链路（含 emoji），保持所见即所得
const publishMode = ref<'edit' | 'preview'>('edit')
const previewContent = computed(() => renderChannelMarkdown(localMessage.value))

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
  publishMode.value = 'edit'
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

.composer-toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: var(--spacing-2);
}

.mode-toggle {
  display: inline-flex;
  background: var(--hover-color);
  border-radius: var(--radius-md);
  padding: 2px;
  gap: 2px;
}

.mode-btn {
  border: none;
  background: transparent;
  padding: 3px 12px;
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.15s;
}

.mode-btn.active {
  background: var(--card-bg);
  color: var(--primary-color);
  font-weight: var(--font-weight-medium);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

/* 发布预览排版（与频道正文 Markdown 一致） */
.md-preview {
  width: 100%;
  max-height: 240px;
  overflow-y: auto;
  padding: var(--spacing-2);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--bg-color);
  font-size: var(--font-size-sm);
  line-height: 1.7;
  color: var(--text-color);
  word-break: break-word;
  margin-bottom: var(--spacing-2);
}

.md-preview :deep(h1),
.md-preview :deep(h2),
.md-preview :deep(h3),
.md-preview :deep(h4) {
  font-weight: 600;
  margin: 0.8em 0 0.4em;
  line-height: 1.3;
}
.md-preview :deep(h1) { font-size: 1.45em; }
.md-preview :deep(h2) { font-size: 1.25em; }
.md-preview :deep(h3) { font-size: 1.15em; }
.md-preview :deep(h4) { font-size: 1.05em; }
.md-preview :deep(p) { margin: 0.5em 0; }
.md-preview :deep(strong) { font-weight: 600; }
.md-preview :deep(em) { font-style: italic; }
.md-preview :deep(a) { color: var(--primary-color); text-decoration: none; }
.md-preview :deep(a:hover) { text-decoration: underline; }
.md-preview :deep(ul),
.md-preview :deep(ol) { padding-left: 1.6em; margin: 0.5em 0; }
.md-preview :deep(li) { margin: 0.25em 0; }
.md-preview :deep(blockquote) {
  margin: 0.5em 0;
  padding: 0.25em 1em;
  border-left: 3px solid var(--primary-color);
  color: var(--text-secondary);
  background: var(--hover-color);
  border-radius: 0 4px 4px 0;
}
.md-preview :deep(pre) {
  background: var(--hover-color);
  padding: 12px;
  border-radius: 6px;
  overflow-x: auto;
  font-size: 13px;
  line-height: 1.5;
  margin: 0.5em 0;
}
.md-preview :deep(code) {
  background: var(--hover-color);
  padding: 2px 5px;
  border-radius: 4px;
  font-family: 'Courier New', Courier, monospace;
  font-size: 0.92em;
}
.md-preview :deep(pre code) { background: transparent; padding: 0; border-radius: 0; }
.md-preview :deep(hr) { border: none; border-top: 1px solid var(--border-color); margin: 1em 0; }
.md-preview :deep(table) { border-collapse: collapse; margin: 0.5em 0; }
.md-preview :deep(th),
.md-preview :deep(td) { border: 1px solid var(--border-color); padding: 6px 12px; text-align: left; }
.md-preview :deep(th) { background: var(--hover-color); font-weight: 600; }
.md-preview :deep(img) { max-width: 100%; border-radius: 6px; }

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
