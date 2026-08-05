<template>
  <div class="message-timeline" role="feed" aria-label="消息时间线">
    <div
      v-for="(message, index) in messages"
      :key="message.id"
      :id="'channel-msg-' + message.id"
      class="timeline-item"
      role="article"
      :aria-label="`来自 ${getSenderName(message)} 的消息`"
    >
      <div class="timeline-marker">
        <div class="timeline-dot" :class="{ 'creator-dot': isCreator(message) }"></div>
        <div v-if="index < messages.length - 1" class="timeline-line"></div>
      </div>
      <div class="timeline-content">
        <div class="timeline-header">
          <Avatar
            :src="message.sender?.avatar"
            :name="getSenderName(message)"
            :server-url="serverUrl"
            :alt="`${getSenderName(message)}的头像`"
            size="sm"
            shape="rounded"
            class="timeline-avatar"
          />
          <div class="timeline-info">
            <span class="timeline-sender">
              {{ getSenderName(message) }}
              <span v-if="isCreator(message)" class="creator-badge">创建者</span>
            </span>
            <span class="timeline-time">{{ formatTime(message.created_at) }}</span>
          </div>
        </div>
        <div class="timeline-body">
          <div ref="contentRef" class="timeline-text" v-html="renderedContent(message)"></div>
        </div>
        <div v-if="!interactive" class="timeline-interact-hint">
          订阅后可互动
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { getDisplayName } from '../../utils/avatar'
import { useServerUrl } from '../../composables/useServerUrl'
import { useChatUtils } from '../../composables/useChatUtils'
import { renderChannelMarkdown } from '../../utils/channelMarkdown'
import Avatar from '../shared/Avatar.vue'
import type { ChannelMessage } from '../../types'

const { serverUrl } = useServerUrl()

interface Props {
  messages: ChannelMessage[]
  creatorId?: string | number
  interactive?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  creatorId: '',
  interactive: true
})

const { formatTime } = useChatUtils()

// 频道正文：声明式 Markdown（方案 A）；时序为 v-for，逐条渲染
const renderedContent = (message: ChannelMessage) => renderChannelMarkdown(message.content)

const getSenderName = (message: ChannelMessage): string => {
  return getDisplayName(message.sender)
}

const isCreator = (message: ChannelMessage): boolean => {
  if (!props.creatorId) return false
  return String(message.sender_id) === String(props.creatorId)
}
</script>

<style scoped>
.message-timeline {
  position: relative;
  padding: var(--spacing-2) 0;
}

.timeline-item {
  display: flex;
  gap: var(--spacing-4);
  position: relative;
  padding-bottom: var(--spacing-4);
  border-radius: 8px;
}

/* 通知中心/深链定位：目标消息瞬时闪烁高亮 */
.timeline-item.msg-highlight {
  animation: timeline-highlight-flash 2.4s ease-out;
}

@keyframes timeline-highlight-flash {
  0% { background: var(--primary-light, rgba(51, 133, 255, 0.18)); box-shadow: 0 0 0 3px var(--primary-light, rgba(51, 133, 255, 0.35)); }
  50% { background: var(--primary-light, rgba(51, 133, 255, 0.18)); box-shadow: 0 0 0 3px var(--primary-light, rgba(51, 133, 255, 0.35)); }
  100% { background: transparent; box-shadow: none; }
}

.timeline-item:last-child {
  padding-bottom: 0;
}

.timeline-marker {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex-shrink: 0;
  width: 12px;
}

.timeline-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--primary-color);
  flex-shrink: 0;
  z-index: 1;
}

.timeline-dot.creator-dot {
  background: var(--success-color);
  box-shadow: 0 0 0 3px rgba(103, 194, 58, 0.2);
}

.timeline-line {
  flex: 1;
  width: 2px;
  background: var(--border-color);
  margin-top: var(--spacing-1);
  min-height: 20px;
}

.timeline-content {
  flex: 1;
  min-width: 0;
  background: var(--card-bg);
  border-radius: var(--radius-lg);
  padding: var(--spacing-3);
  border: 1px solid var(--border-color);
  transition: all var(--transition-fast);
}

.timeline-content:hover {
  box-shadow: var(--shadow-sm);
  border-color: var(--primary-color);
}

.timeline-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
  margin-bottom: var(--spacing-2);
}

.timeline-avatar {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
}

.timeline-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
}

.timeline-sender {
  font-weight: var(--font-weight-medium);
  font-size: var(--font-size-sm);
  color: var(--text-color);
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
}

.creator-badge {
  font-size: var(--font-size-xs);
  padding: 1px var(--spacing-2);
  background: var(--primary-color);
  color: white;
  border-radius: var(--radius-sm);
  font-weight: var(--font-weight-medium);
}

.timeline-time {
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
}

.timeline-body {
  margin: 0;
}

.timeline-text {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--text-color);
  line-height: 1.6;
  word-break: break-word;
  white-space: pre-wrap;
}

.timeline-text :deep(.emoji-img) {
  width: 18px;
  height: 18px;
  vertical-align: middle;
  margin: 0 1px;
}

/* Markdown 声明的排版 */
.timeline-text :deep(h1),
.timeline-text :deep(h2),
.timeline-text :deep(h3) {
  font-weight: 600;
  color: var(--text-color);
  margin: 0.8em 0 0.4em 0;
  line-height: 1.3;
}
.timeline-text :deep(h1) { font-size: 1.4em; }
.timeline-text :deep(h2) { font-size: 1.2em; }
.timeline-text :deep(h3) { font-size: 1.1em; }
.timeline-text :deep(strong) { font-weight: 600; }
.timeline-text :deep(em) { font-style: italic; }
.timeline-text :deep(a) { color: var(--primary-color); text-decoration: none; }
.timeline-text :deep(a:hover) { text-decoration: underline; }
.timeline-text :deep(p) { margin: 0.4em 0; }
.timeline-text :deep(ul),
.timeline-text :deep(ol) { margin: 0.4em 0; padding-left: 1.6em; }
.timeline-text :deep(li) { margin: 0.2em 0; }
.timeline-text :deep(blockquote) {
  margin: 0.4em 0;
  padding: 0.25em 1em;
  border-left: 3px solid var(--primary-color);
  color: var(--text-secondary);
  background: var(--hover-color);
  border-radius: 0 4px 4px 0;
}
.timeline-text :deep(pre) {
  background: var(--hover-color);
  padding: 12px;
  border-radius: 6px;
  overflow-x: auto;
  font-size: 13px;
  line-height: 1.5;
  margin: 0.4em 0;
}
.timeline-text :deep(code) {
  background: var(--hover-color);
  padding: 2px 5px;
  border-radius: 4px;
  font-family: 'Courier New', Courier, monospace;
  font-size: 0.92em;
}
.timeline-text :deep(pre code) { background: transparent; padding: 0; border-radius: 0; }
.timeline-text :deep(img) { max-width: 100%; border-radius: 6px; }

.timeline-interact-hint {
  margin-top: var(--spacing-2);
  padding-top: var(--spacing-2);
  border-top: 1px solid var(--border-color);
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
  opacity: 0.6;
  text-align: center;
}
</style>
