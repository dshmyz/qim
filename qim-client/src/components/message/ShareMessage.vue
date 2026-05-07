<template>
  <div class="message-bubble share-message" :class="{ self: isSelf }">
    <div class="share-info">
      <div class="share-icon-container">
        <div class="share-icon" :class="shareData?.type">
          <i v-if="shareData?.type === 'file'" class="fas fa-file"></i>
          <i v-else-if="shareData?.type === 'note'" class="fas fa-file-alt"></i>
          <i v-else-if="shareData?.type === 'sticky'" class="fas fa-sticky-note"></i>
          <i v-else class="fas fa-share-alt"></i>
        </div>
        <div class="share-type">{{ getShareTypeText(shareData?.type) }}</div>
      </div>
      <div class="share-details">
        <div class="share-name">{{ shareData?.name || content }}</div>
        <div class="share-actions">
          <button class="share-action-btn" @click="toggleContent">
            <i :class="isExpanded ? 'fas fa-chevron-up' : 'fas fa-chevron-down'"></i>
            {{ isExpanded ? '收起' : '查看' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="isExpanded && (shareType === 'note' || shareType === 'sticky')" class="share-expanded-content">
      <div v-if="shareType === 'note'" class="note-content" v-html="sanitizeMarkdown(renderMarkdown(noteContent))"></div>
      <div v-else-if="shareType === 'sticky'" class="sticky-content">{{ noteContent }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { sanitizeMarkdown } from '../../utils/sanitize'
import { useChatUtils } from '../../composables/useChatUtils'

const { renderMarkdown } = useChatUtils()

const props = defineProps<{
  content: string
  shareData?: {
    type: string
    name: string
  }
  isSelf?: boolean
}>()

const isExpanded = ref(false)

const shareType = computed(() => {
  return props.shareData?.type || ''
})

const noteContent = computed(() => {
  try {
    const shareData = JSON.parse(props.content)
    if (shareData.type === 'note' || shareData.type === 'sticky') {
      return shareData.originalContent || shareData.content || ''
    }
    return ''
  } catch {
    return ''
  }
})

const toggleContent = () => {
  isExpanded.value = !isExpanded.value
}

const getShareTypeText = (type?: string): string => {
  switch (type) {
    case 'file':
      return '文件'
    case 'note':
      return '笔记'
    case 'message':
      return '消息'
    case 'sticky':
      return '便签'
    default:
      return '分享'
  }
}
</script>

<style scoped>
.share-message {
  background: var(--sidebar-bg);
  border-radius: 12px;
  padding: 14px;
  width: fit-content;
  min-width: 250px;
  max-width: 100%;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  transition: all 0.2s ease;
  box-sizing: border-box;
}

.share-message:hover {
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.12);
}

.share-info {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.share-icon-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.share-icon {
  font-size: 24px;
  margin-top: 2px;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: var(--list-bg);
  border-radius: 6px;
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.share-type {
  font-size: 11px;
  color: var(--text-secondary);
  line-height: 1.2;
  white-space: nowrap;
  text-align: center;
  margin-bottom: 4px;
}

.share-details {
  flex: 1;
  min-width: 0;
}

.share-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
  margin-bottom: 8px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.4;
  word-break: break-all;
}

.share-actions {
  display: flex;
  gap: 8px;
  margin-top: 4px;
}

.share-action-btn {
  padding: 6px 16px;
  font-size: 12px;
  border-radius: 8px;
  border: none;
  background-color: var(--primary-light);
  color: var(--primary-color);
  cursor: pointer;
  transition: all 0.3s ease;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}

.share-action-btn:hover {
  background-color: var(--primary-color);
  color: #fff;
  border-color: var(--primary-color);
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.3);
  transform: translateY(-1px);
}

.share-action-btn:active {
  transform: translateY(0);
  box-shadow: 0 2px 8px rgba(24, 144, 255, 0.3);
}

.share-expanded-content {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border-color);
  animation: expandContent 0.3s ease-out;
}

@keyframes expandContent {
  from {
    opacity: 0;
    max-height: 0;
  }
  to {
    opacity: 1;
    max-height: 500px;
  }
}

.note-content {
  font-size: 14px;
  line-height: 1.6;
  color: var(--text-color);
  max-height: 400px;
  overflow-y: auto;
  padding: 12px;
  background: var(--secondary-color);
  border-radius: 8px;
}

.note-content :deep(h1),
.note-content :deep(h2),
.note-content :deep(h3) {
  margin-top: 16px;
  margin-bottom: 8px;
  font-weight: 600;
}

.note-content :deep(h1):first-child,
.note-content :deep(h2):first-child,
.note-content :deep(h3):first-child {
  margin-top: 0;
}

.note-content :deep(p) {
  margin: 8px 0;
}

.note-content :deep(code) {
  background: var(--hover-color);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
}

.note-content :deep(pre) {
  background: var(--hover-color);
  padding: 12px;
  border-radius: 6px;
  overflow-x: auto;
  margin: 8px 0;
}

.note-content :deep(pre code) {
  background: none;
  padding: 0;
}

.note-content :deep(ul),
.note-content :deep(ol) {
  padding-left: 20px;
  margin: 8px 0;
}

.note-content :deep(li) {
  margin: 4px 0;
}

.note-content :deep(blockquote) {
  border-left: 3px solid var(--primary-color);
  padding-left: 12px;
  margin: 8px 0;
  color: var(--text-secondary);
}

.note-content :deep(a) {
  color: var(--primary-color);
  text-decoration: none;
}

.note-content :deep(a:hover) {
  text-decoration: underline;
}

.sticky-content {
  font-size: 14px;
  line-height: 1.6;
  color: var(--text-color);
  max-height: 400px;
  overflow-y: auto;
  padding: 12px;
  background: #fff9c4;
  border-radius: 8px;
  white-space: pre-wrap;
  word-break: break-word;
}

/* 自己的分享消息样式 */
.share-message.self {
  background: var(--primary-color);
}

.share-message.self .share-name {
  color: #fff;
}

.share-message.self .share-type {
  color: rgba(255, 255, 255, 0.8);
}

.share-message.self .share-icon {
  background-color: rgba(255, 255, 255, 0.2);
  border-color: rgba(255, 255, 255, 0.3);
  color: #fff;
}

.share-message.self .share-action-btn {
  background-color: rgba(255, 255, 255, 0.2);
  border: none;
  border-color: rgba(255, 255, 255, 0.3);
  color: #fff;
}

.share-message.self .share-action-btn:hover {
  background-color: rgba(255, 255, 255, 0.3);
  border-color: rgba(255, 255, 255, 0.4);
  box-shadow: 0 4px 12px rgba(255, 255, 255, 0.3);
}

.share-message.self .share-expanded-content {
  border-top-color: rgba(255, 255, 255, 0.2);
}

.share-message.self .note-content {
  /* background: rgba(255, 255, 255, 0.1); */
  /* color: #fff; */
}

.share-message.self .note-content :deep(code) {
  background: rgba(255, 255, 255, 0.15);
}

.share-message.self .note-content :deep(pre) {
  background: rgba(255, 255, 255, 0.15);
}

.share-message.self .note-content :deep(blockquote) {
  border-left-color: rgba(255, 255, 255, 0.6);
  color: rgba(255, 255, 255, 0.8);
}

.share-message.self .note-content :deep(a) {
  color: #fff;
  text-decoration: underline;
}

.share-message.self .sticky-content {
  background: rgba(255, 255, 255, 0.9);
  color: #333;
}
</style>
