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

    <div v-if="isExpanded && (shareType === 'note' || shareType === 'sticky')" class="share-expanded-content" @click="handleLinkClick">
      <div v-if="shareType === 'note'" class="note-content" v-html="sanitizeMarkdown(renderedNoteContent)"></div>
      <div v-else-if="shareType === 'sticky'" class="sticky-content">{{ noteContent }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { marked } from 'marked'
import { sanitizeMarkdown } from '../../utils/sanitize'

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

const renderedNoteContent = computed(() => {
  if (!noteContent.value) return ''
  const html = marked(noteContent.value)
  return typeof html === 'string' ? html : String(html)
})

const handleLinkClick = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  const link = target.closest('a')
  if (link && window.electron?.shell?.openExternal) {
    event.preventDefault()
    const href = link.getAttribute('href')
    if (href) {
      window.electron.shell.openExternal(href)
    }
  }
}

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
  padding: 10px 12px;
  width: fit-content;
  min-width: 260px;
  max-width: 100%;
  transition: all 0.2s ease;
  box-sizing: border-box;
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
  gap: 10px;
}

.share-icon-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex-shrink: 0;
  gap: 6px;
}

.share-icon {
  font-size: 18px;
  width: 42px;
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  border-radius: 10px;
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
  margin-top: 8px;
  padding: 12px;
  background: var(--card-bg);
  border-radius: 8px;
  font-size: 14px;
  color: var(--text-color);
  line-height: 1.7;
  border: 1px solid var(--border-color);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  max-height: 400px;
  overflow-y: auto;
}

.share-expanded-content :deep(h1),
.share-expanded-content :deep(h2),
.share-expanded-content :deep(h3),
.share-expanded-content :deep(h4) {
  margin: 14px 0 8px 0;
  font-weight: 700;
  color: var(--text-color);
  line-height: 1.4;
}

.share-expanded-content :deep(h1) { font-size: 1.3em; }
.share-expanded-content :deep(h2) { font-size: 1.15em; }
.share-expanded-content :deep(h3) { font-size: 1.05em; }
.share-expanded-content :deep(h4) { font-size: 1em; }

.share-expanded-content :deep(p) {
  margin: 6px 0;
}

.share-expanded-content :deep(strong) {
  font-weight: 700;
}

.share-expanded-content :deep(em) {
  font-style: italic;
}

.share-expanded-content :deep(pre) {
  background: var(--hover-color);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  padding: 12px;
  margin: 10px 0;
  overflow-x: auto;
  font-family: 'Fira Code', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.5;
}

.share-expanded-content :deep(code) {
  background: var(--hover-color);
  border-radius: 3px;
  padding: 2px 6px;
  font-family: 'Fira Code', 'Courier New', monospace;
  font-size: 13px;
  color: var(--text-color);
}

.share-expanded-content :deep(pre code) {
  background: none;
  padding: 0;
  border-radius: 0;
}

.share-expanded-content :deep(ul),
.share-expanded-content :deep(ol) {
  margin: 6px 0;
  padding-left: 24px;
}

.share-expanded-content :deep(li) {
  margin: 4px 0;
}

.share-expanded-content :deep(a) {
  color: var(--primary-color);
  text-decoration: underline;
}

.share-expanded-content :deep(blockquote) {
  margin: 10px 0;
  padding: 8px 14px;
  border-left: 4px solid var(--primary-color);
  background: var(--hover-color);
  border-radius: 0 6px 6px 0;
  color: var(--text-secondary);
}

.share-expanded-content :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 10px 0;
  font-size: 13px;
}

.share-expanded-content :deep(th),
.share-expanded-content :deep(td) {
  border: 1px solid var(--border-color);
  padding: 8px 12px;
  text-align: left;
}

.share-expanded-content :deep(th) {
  background: var(--hover-color);
  font-weight: 600;
}

.share-expanded-content :deep(hr) {
  border: none;
  border-top: 1px solid var(--border-color);
  margin: 14px 0;
}

.share-expanded-content :deep(img) {
  max-width: 100%;
  border-radius: 6px;
  margin: 8px 0;
}

.share-expanded-content :deep(input[type="checkbox"]) {
  margin-right: 6px;
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
  background: var(--card-bg);
  border-color: rgba(255, 255, 255, 0.3);
  color: var(--text-color);
}

.share-message.self .share-expanded-content :deep(h1),
.share-message.self .share-expanded-content :deep(h2),
.share-message.self .share-expanded-content :deep(h3),
.share-message.self .share-expanded-content :deep(h4) {
  color: var(--text-color);
}

.share-message.self .share-expanded-content :deep(a) {
  color: var(--primary-color);
}

.share-message.self .share-expanded-content :deep(blockquote) {
  background: var(--hover-color);
  color: var(--text-secondary);
}

.share-message.self .share-expanded-content :deep(pre),
.share-message.self .share-expanded-content :deep(code) {
  background: var(--hover-color);
  color: var(--text-color);
}

.share-message.self .share-expanded-content :deep(th) {
  background: var(--hover-color);
}

.share-message.self .share-expanded-content :deep(th),
.share-message.self .share-expanded-content :deep(td) {
  border-color: var(--border-color);
}
</style>
