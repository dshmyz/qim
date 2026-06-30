<template>
  <div class="message-bubble share-message" :class="{ self: isSelf }">
    <div class="attachment-card" @click="toggleContent">
      <div class="attachment-card__icon share-attachment-icon" :class="shareData?.type">
        <i v-if="shareData?.type === 'file'" class="fas fa-file"></i>
        <i v-else-if="shareData?.type === 'note'" class="fas fa-file-alt"></i>
        <i v-else-if="shareData?.type === 'sticky'" class="fas fa-sticky-note"></i>
        <i v-else class="fas fa-share-alt"></i>
      </div>
      <div class="attachment-card__content">
        <div class="attachment-card__title">{{ shareData?.name || content }}</div>
        <div class="attachment-card__meta">{{ getShareTypeText(shareData?.type) }} · 点击查看</div>
      </div>
      <button class="share-action-btn attachment-card__action" @click.stop="toggleContent" :title="isExpanded ? '收起' : '查看'">
        <i :class="isExpanded ? 'fas fa-chevron-up' : 'fas fa-chevron-down'"></i>
      </button>
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
  width: 280px;
  max-width: min(100%, 320px);
  box-sizing: border-box;
}

.attachment-card {
  display: grid;
  grid-template-columns: 42px minmax(0, 1fr) 28px;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-radius: 14px;
  background: color-mix(in srgb, var(--sidebar-bg), transparent 4%);
  border: 1px solid color-mix(in srgb, var(--border-color), transparent 20%);
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.06);
  cursor: pointer;
  transition: border-color 0.16s ease, box-shadow 0.16s ease, transform 0.16s ease;
}

.attachment-card:hover {
  border-color: color-mix(in srgb, var(--primary-color), transparent 58%);
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.08);
  transform: translateY(-1px);
}

.attachment-card__icon {
  width: 42px;
  height: 42px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #ffffff;
  background: linear-gradient(135deg, #7c3aed 0%, #2563eb 100%);
  font-size: 17px;
}

.attachment-card__icon.file {
  background: linear-gradient(135deg, #2563eb 0%, #0ea5e9 100%);
}

.attachment-card__icon.note {
  background: linear-gradient(135deg, #7c3aed 0%, #a855f7 100%);
}

.attachment-card__icon.sticky {
  background: linear-gradient(135deg, #f59e0b 0%, #f97316 100%);
}

.attachment-card__content {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.attachment-card__title {
  font-size: 14px;
  font-weight: 600;
  line-height: 1.35;
  color: var(--text-color);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  letter-spacing: -0.01em;
}

.attachment-card__meta {
  font-size: 12px;
  line-height: 1.35;
  color: var(--text-secondary);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.attachment-card__action {
  width: 28px;
  height: 28px;
  border-radius: 9px;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  background: transparent;
  cursor: pointer;
  font-size: 12px;
  transition: background 0.16s ease, color 0.16s ease, transform 0.16s ease;
}

.attachment-card:hover .attachment-card__action {
  color: var(--primary-color);
  background: color-mix(in srgb, var(--primary-color), transparent 90%);
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

.share-message.self .attachment-card {
  background: color-mix(in srgb, var(--sidebar-bg), transparent 4%);
  border-color: color-mix(in srgb, var(--border-color), transparent 20%);
  color: var(--text-color);
}

:global(.message-item.self) .share-message.self {
  background: transparent;
  border: none;
  color: var(--text-color);
}

:global(.message-item.self) .share-message.self .attachment-card {
  background: color-mix(in srgb, var(--sidebar-bg), transparent 4%);
  border-color: transparent;
  color: var(--text-color);
}

.share-message.self .share-expanded-content {
  background: var(--card-bg);
  border-color: var(--border-color);
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

[data-theme="elegant-dark"] .attachment-card {
  background: color-mix(in srgb, var(--panel-bg), white 5%);
  border-color: rgba(255, 255, 255, 0.12);
  box-shadow: none;
}

[data-theme="elegant-dark"] .share-message.self .attachment-card {
  background: color-mix(in srgb, var(--panel-bg), white 5%);
  border-color: rgba(255, 255, 255, 0.12);
  color: var(--text-color);
}

:global([data-theme="elegant-dark"] .message-item.self) .share-message.self .attachment-card {
  background: color-mix(in srgb, var(--panel-bg), white 5%);
  border-color: transparent;
  color: var(--text-color);
}
</style>
