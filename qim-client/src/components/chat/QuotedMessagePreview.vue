<template>
  <div
    class="quoted-message-preview"
    :class="{ 'is-compact': compact }"
    @click="handleClick"
  >
    <div class="quoted-message-preview__indicator" />
    <div class="quoted-message-preview__content">
      <div class="quoted-message-preview__header">
        <span class="quoted-message-preview__sender">
          {{ senderName }}
        </span>
        <span v-if="typeLabel" class="quoted-message-preview__type">
          {{ typeLabel }}
        </span>
      </div>
      <div class="quoted-message-preview__text">
        {{ displayContent }}
      </div>
    </div>
    <button
      v-if="showClose"
      class="quoted-message-preview__close"
      aria-label="关闭引用"
      @click.stop="handleClose"
    >
      &times;
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

export interface QuotedMessage {
  id: string
  content: string
  sender: {
    id?: string
    name: string
    avatar?: string
  }
  type: 'text' | 'image' | 'file' | 'share' | 'miniApp' | 'news' | 'system' | string
}

interface Props {
  quotedMessage: QuotedMessage
  showClose?: boolean
  compact?: boolean
  maxLength?: number
}

const props = withDefaults(defineProps<Props>(), {
  showClose: true,
  compact: false,
  maxLength: 80,
})

const emit = defineEmits<{
  close: []
  scrollTo: [messageId: string]
}>()

const senderName = computed(() => {
  return props.quotedMessage.sender?.name || '未知用户'
})

const typeLabel = computed(() => {
  const type = props.quotedMessage.type
  const typeMap: Record<string, string> = {
    image: '[图片]',
    file: '[文件]',
    share: '[分享]',
    miniApp: '[小程序]',
    news: '[资讯]',
    system: '[系统]',
  }
  return typeMap[type] || ''
})

const displayContent = computed(() => {
  const content = props.quotedMessage.content || '无内容'
  if (content.length <= props.maxLength) {
    return content
  }
  return `${content.slice(0, props.maxLength)}...`
})

const handleClose = () => {
  emit('close')
}

const handleClick = () => {
  emit('scrollTo', props.quotedMessage.id)
}
</script>

<style scoped>
.quoted-message-preview {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 12px;
  background: var(--hover-color, rgba(0, 0, 0, 0.04));
  border-left: 3px solid var(--primary-color, #3b82f6);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
  position: relative;
  user-select: none;
}

.quoted-message-preview:hover {
  background: var(--border-color, rgba(0, 0, 0, 0.08));
  transform: translateX(2px);
}

.quoted-message-preview__indicator {
  width: 3px;
  min-height: 100%;
  background: var(--primary-color, #3b82f6);
  border-radius: 2px;
  opacity: 0.6;
  flex-shrink: 0;
  align-self: stretch;
}

.quoted-message-preview__content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.quoted-message-preview__header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.quoted-message-preview__sender {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-color, #333);
  opacity: 0.85;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.quoted-message-preview__type {
  font-size: 11px;
  font-weight: 500;
  color: var(--text-secondary, #666);
  opacity: 0.7;
  white-space: nowrap;
  flex-shrink: 0;
}

.quoted-message-preview__text {
  font-size: 13px;
  color: var(--text-color, #333);
  line-height: 1.5;
  word-break: break-word;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
}

.quoted-message-preview__close {
  width: 20px;
  height: 20px;
  border: none;
  background: transparent;
  color: var(--text-secondary, #666);
  font-size: 16px;
  line-height: 1;
  cursor: pointer;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  flex-shrink: 0;
  padding: 0;
}

.quoted-message-preview__close:hover {
  background: rgba(0, 0, 0, 0.1);
  color: var(--text-color, #333);
}

/* 紧凑模式 */
.quoted-message-preview.is-compact {
  padding: 6px 10px;
  gap: 8px;
}

.quoted-message-preview.is-compact .quoted-message-preview__sender {
  font-size: 11px;
}

.quoted-message-preview.is-compact .quoted-message-preview__text {
  font-size: 12px;
  -webkit-line-clamp: 1;
}

.quoted-message-preview.is-compact .quoted-message-preview__type {
  font-size: 10px;
}

/* 优雅紫色主题 */
[data-theme='elegant-purple'] .quoted-message-preview {
  border-left-color: #8b5cf6;
}
</style>
