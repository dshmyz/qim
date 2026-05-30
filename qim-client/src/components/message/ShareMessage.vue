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
        <div class="share-type-label">{{ getShareTypeText(shareData?.type) }}</div>
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
  padding: 16px;
  width: fit-content;
  min-width: 260px;
  max-width: 100%;
  transition: all 0.2s ease;
  box-sizing: border-box;
  border: 1px solid var(--border-color);
  position: relative;
  overflow: hidden;
}

.share-message::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: linear-gradient(90deg, #f093fb, #f5576c, #4facfe);
  opacity: 0;
  transition: opacity 0.2s ease;
}

.share-message:hover {
  transform: translateY(-1px);
}

.share-message:hover::before {
  opacity: 1;
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
  flex-shrink: 0;
  gap: 6px;
}

.share-icon {
  font-size: 20px;
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  border-radius: 12px;
  color: #ffffff;
}

.share-type-label {
  font-size: 10px;
  font-weight: 500;
  color: var(--text-secondary);
  background: var(--hover-color);
  padding: 2px 8px;
  border-radius: 4px;
  display: block;
  text-align: center;
  white-space: nowrap;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.share-details {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.share-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.4;
  word-break: break-all;
  text-align: left;
  letter-spacing: -0.01em;
}

.share-actions {
  display: flex;
  gap: 6px;
  margin-top: 4px;
}

.share-action-btn {
  padding: 6px 12px;
  font-size: 12px;
  border-radius: 6px;
  border: none;
  background: var(--hover-color);
  color: var(--text-color);
  cursor: pointer;
  transition: all 0.15s ease;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 5px;
  white-space: nowrap;
}

.share-action-btn i {
  font-size: 11px;
  opacity: 0.8;
}

.share-action-btn:hover {
  background: var(--primary-color);
  color: #ffffff;
  transform: translateY(-1px);
}

.share-action-btn:hover i {
  opacity: 1;
}

.share-action-btn:active {
  transform: translateY(0);
}

.share-expanded-content {
  margin-top: 10px;
  padding: 10px;
  background: var(--hover-color);
  border-radius: 6px;
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.5;
  border: 1px solid var(--border-color);
}

/* 自己的分享消息样式 */
.share-message.self {
  background: var(--primary-color);
  border: none;
}

.share-message.self::before {
  background: linear-gradient(90deg, rgba(255, 255, 255, 0.4), rgba(255, 255, 255, 0.15), rgba(255, 255, 255, 0.4));
  opacity: 1;
}

.share-message.self .share-name {
  color: #ffffff;
  font-weight: 600;
}

.share-message.self .share-icon {
  background: rgba(255, 255, 255, 0.95);
  color: var(--primary-color);
}

.share-message.self .share-type-label {
  background: rgba(255, 255, 255, 0.2);
  color: rgba(255, 255, 255, 0.85);
}

.share-message.self .share-action-btn {
  background: rgba(255, 255, 255, 0.95);
  color: var(--primary-color);
}

.share-message.self .share-action-btn:hover {
  background: #ffffff;
  transform: translateY(-1px);
}

.share-message.self .share-expanded-content {
  background: rgba(255, 255, 255, 0.1);
  border-color: rgba(255, 255, 255, 0.15);
  color: rgba(255, 255, 255, 0.9);
}
</style>
