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
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.95) 0%, rgba(255, 255, 255, 0.85) 100%);
  border-radius: 14px;
  padding: 14px;
  width: fit-content;
  min-width: 260px;
  max-width: 100%;
  box-shadow: 
    0 2px 8px rgba(0, 0, 0, 0.04),
    0 8px 24px rgba(0, 0, 0, 0.06),
    inset 0 1px 0 rgba(255, 255, 255, 0.8);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  box-sizing: border-box;
  border: 1px solid rgba(0, 0, 0, 0.04);
  backdrop-filter: blur(10px);
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
  transition: opacity 0.3s ease;
}

.share-message:hover {
  box-shadow: 
    0 4px 12px rgba(0, 0, 0, 0.06),
    0 12px 32px rgba(0, 0, 0, 0.08),
    inset 0 1px 0 rgba(255, 255, 255, 0.9);
  transform: translateY(-2px);
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
  font-size: 22px;
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  border-radius: 12px;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: 
    0 4px 12px rgba(240, 147, 251, 0.25),
    inset 0 1px 0 rgba(255, 255, 255, 0.2);
  position: relative;
  overflow: hidden;
  color: #ffffff;
}

.share-icon::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.2) 0%, transparent 100%);
  border-radius: 12px;
}

.share-icon:hover {
  transform: scale(1.08) rotate(-2deg);
  box-shadow: 
    0 6px 20px rgba(240, 147, 251, 0.35),
    inset 0 1px 0 rgba(255, 255, 255, 0.3);
}

.share-type-label {
  font-size: 9px;
  font-weight: 500;
  color: #6b7280;
  background: rgba(107, 114, 128, 0.08);
  padding: 2px 6px;
  border-radius: 6px;
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
  color: #1a1a2e;
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
  padding: 6px 14px;
  font-size: 12px;
  border-radius: 8px;
  border: none;
  background: linear-gradient(135deg, #f8f9fa 0%, #e9ecef 100%);
  color: #495057;
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 5px;
  white-space: nowrap;
  box-shadow: 
    0 2px 4px rgba(0, 0, 0, 0.04),
    inset 0 1px 0 rgba(255, 255, 255, 0.8);
}

.share-action-btn i {
  font-size: 11px;
  opacity: 0.8;
}

.share-action-btn:hover {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  color: #ffffff;
  box-shadow: 
    0 4px 12px rgba(240, 147, 251, 0.3),
    inset 0 1px 0 rgba(255, 255, 255, 0.2);
  transform: translateY(-2px);
}

.share-action-btn:hover i {
  opacity: 1;
}

.share-action-btn:active {
  transform: translateY(0);
  box-shadow: 0 2px 6px rgba(240, 147, 251, 0.3);
}

.share-expanded-content {
  margin-top: 10px;
  padding: 10px;
  background: rgba(0, 0, 0, 0.02);
  border-radius: 8px;
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.5;
  border: 1px solid rgba(0, 0, 0, 0.04);
}

/* 自己的分享消息样式 */
.share-message.self {
  background: linear-gradient(135deg, #d53f8c 0%, #ed64a6 50%, #f687b3 100%);
  border: none;
  box-shadow: 
    0 4px 12px rgba(213, 63, 140, 0.25),
    0 12px 32px rgba(237, 100, 166, 0.2),
    inset 0 1px 0 rgba(255, 255, 255, 0.15);
}

.share-message.self::before {
  background: linear-gradient(90deg, rgba(255, 255, 255, 0.3), rgba(255, 255, 255, 0.1), rgba(255, 255, 255, 0.3));
}

.share-message.self .share-name {
  color: #ffffff;
  font-weight: 600;
}

.share-message.self .share-icon {
  background: rgba(255, 255, 255, 0.95);
  color: #f5576c;
  box-shadow: 
    0 4px 12px rgba(0, 0, 0, 0.15),
    inset 0 1px 0 rgba(255, 255, 255, 0.5);
}

.share-message.self .share-type-label {
  background: rgba(255, 255, 255, 0.2);
  color: rgba(255, 255, 255, 0.85);
}

.share-message.self .share-action-btn {
  background: rgba(255, 255, 255, 0.95);
  color: #f5576c;
  box-shadow: 
    0 2px 8px rgba(0, 0, 0, 0.1),
    inset 0 1px 0 rgba(255, 255, 255, 0.8);
}

.share-message.self .share-action-btn:hover {
  background: #ffffff;
  box-shadow: 
    0 4px 12px rgba(0, 0, 0, 0.15),
    inset 0 1px 0 rgba(255, 255, 255, 0.9);
  transform: translateY(-2px);
}

.share-message.self .share-expanded-content {
  background: rgba(255, 255, 255, 0.15);
  border-color: rgba(255, 255, 255, 0.2);
  color: rgba(255, 255, 255, 0.9);
}
</style>
