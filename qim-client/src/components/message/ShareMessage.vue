<template>
  <AttachmentCard
    class="share-message"
    :class="{ 'share-message--expanded': isExpanded && (shareType === 'note' || shareType === 'sticky') }"
    :is-self="isSelf"
    @click="toggleContent"
  >
      <i
        class="share-icon"
        :class="shareIconClass"
        :style="{ background: shareIconBg }"
      />
      <template #content>
        <div class="share-title">{{ shareData?.name || content }}</div>
        <div class="share-bottom">
          <div class="share-meta">
            <span class="share-type-label">{{ typeLabel }}</span>
            <span v-if="typeLabel && sizeText"> · </span>
            <span v-if="sizeText">{{ sizeText }}</span>
          </div>
          <div class="share-actions">
            <button v-if="shareType === 'note' || shareType === 'sticky'" class="attachment-card__btn" @click.stop="showPreview = true" title="预览">
              <i class="fas fa-expand"></i>
            </button>
            <button class="attachment-card__btn" @click.stop="toggleContent" :title="isExpanded ? '收起' : '查看'">
              <i :class="isExpanded ? 'fas fa-chevron-up' : 'fas fa-chevron-down'"></i>
            </button>
          </div>
        </div>
      </template>
      <template v-if="isExpanded && (shareType === 'note' || shareType === 'sticky')" #below>
        <div class="share-expanded-content" @click="handleLinkClick">
          <div v-if="shareType === 'note'" class="note-content" v-html="sanitizeMarkdown(renderedNoteContent)"></div>
          <div v-else-if="shareType === 'sticky'" class="sticky-content">{{ noteContent }}</div>
        </div>
      </template>
    </AttachmentCard>

    <Teleport to="body">
      <div v-if="showPreview" class="note-preview-backdrop" @click.self="showPreview = false">
        <div class="note-preview-dialog" role="dialog" aria-modal="true">
          <div class="note-preview-header">
            <h3 class="note-preview-title">{{ shareData?.name || '笔记预览' }}</h3>
            <button class="note-preview-close" @click="showPreview = false">
              <i class="fas fa-times"></i>
            </button>
          </div>
          <div class="note-preview-body" @click="handleLinkClick">
            <div v-if="shareType === 'note'" class="note-preview-content" v-html="sanitizeMarkdown(renderedNoteContent)"></div>
            <div v-else-if="shareType === 'sticky'" class="note-preview-content note-preview-plain">{{ noteContent }}</div>
          </div>
        </div>
      </div>
    </Teleport>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { marked } from 'marked'
import { sanitizeMarkdown } from '../../utils/sanitize'
import { formatFileSize } from '../../utils/fileType'
import AttachmentCard from './AttachmentCard.vue'

const props = defineProps<{
  content: string
  shareData?: {
    type: string
    name: string
  }
  isSelf?: boolean
}>()

const isExpanded = ref(false)
const showPreview = ref(false)

const shareType = computed(() => {
  return props.shareData?.type || ''
})

const shareIconClass = computed(() => {
  switch (props.shareData?.type) {
    case 'file': return 'fas fa-file'
    case 'note': return 'fas fa-file-alt'
    case 'sticky': return 'fas fa-sticky-note'
    default: return 'fas fa-share-alt'
  }
})

const shareIconBg = computed(() => {
  switch (props.shareData?.type) {
    case 'file': return 'linear-gradient(135deg, #2563eb 0%, #0ea5e9 100%)'
    case 'note': return 'linear-gradient(135deg, #7c3aed 0%, #a855f7 100%)'
    case 'sticky': return 'linear-gradient(135deg, #f59e0b 0%, #f97316 100%)'
    default: return 'linear-gradient(135deg, #7c3aed 0%, #2563eb 100%)'
  }
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

const typeLabel = computed(() => {
  switch (props.shareData?.type) {
    case 'file': return '文件'
    case 'note': return '笔记'
    case 'sticky': return '便签'
    case 'message': return '消息'
    default: return '分享'
  }
})

const sizeText = computed(() => {
  try {
    const data = JSON.parse(props.content)
    if (props.shareData?.type === 'file' && typeof data.size === 'number' && data.size > 0) {
      return formatFileSize(data.size)
    }
    if ((props.shareData?.type === 'note' || props.shareData?.type === 'sticky') && data.originalContent) {
      const len = data.originalContent.length
      if (len >= 1024) return `${(len / 1024).toFixed(1)} KB`
      return `${len} 字`
    }
  } catch { /* ignore */ }
  return ''
})
</script>

<style scoped>
.share-message {
  width: 300px;
  max-width: min(100%, 340px);
  min-width: 0;
  box-sizing: border-box;
}

.share-message--expanded {
  width: 100%;
  max-width: 100%;
  align-items: start;
}

.share-icon {
  color: #ffffff;
  font-size: 17px;
}

.share-title {
  font-size: 14px;
  font-weight: 600;
  line-height: 1.35;
  color: var(--text-color);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  letter-spacing: -0.01em;
}

.share-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.share-type-label {
  color: var(--text-secondary);
}

.share-meta {
  min-height: 16px;
  font-size: 12px;
  line-height: 1.35;
  color: var(--text-secondary);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.share-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

/* Note preview dialog */
.note-preview-backdrop {
  position: fixed;
  inset: 0;
  z-index: 3000;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
}

.note-preview-dialog {
  width: min(680px, 90vw);
  max-height: 80vh;
  background: var(--card-bg);
  border-radius: 14px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.note-preview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}

.note-preview-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color);
  margin: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.note-preview-close {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  background: transparent;
  cursor: pointer;
  font-size: 14px;
  flex-shrink: 0;
}

.note-preview-close:hover {
  background: var(--hover-color);
  color: var(--text-color);
}

.note-preview-body {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.note-preview-content {
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-color);
}

.note-preview-plain {
  white-space: pre-wrap;
  word-break: break-word;
}

.note-preview-content :deep(h1),
.note-preview-content :deep(h2),
.note-preview-content :deep(h3),
.note-preview-content :deep(h4) {
  margin: 16px 0 8px 0;
  font-weight: 700;
  color: var(--text-color);
  line-height: 1.4;
}

.note-preview-content :deep(h1) { font-size: 1.4em; }
.note-preview-content :deep(h2) { font-size: 1.2em; }
.note-preview-content :deep(h3) { font-size: 1.1em; }
.note-preview-content :deep(h4) { font-size: 1em; }

.note-preview-content :deep(p) {
  margin: 8px 0;
}

.note-preview-content :deep(strong) {
  font-weight: 700;
}

.note-preview-content :deep(em) {
  font-style: italic;
}

.note-preview-content :deep(pre) {
  background: var(--hover-color);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 14px;
  margin: 12px 0;
  overflow-x: auto;
  font-family: 'Fira Code', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.5;
}

.note-preview-content :deep(code) {
  background: var(--hover-color);
  border-radius: 4px;
  padding: 2px 6px;
  font-family: 'Fira Code', 'Courier New', monospace;
  font-size: 13px;
}

.note-preview-content :deep(pre code) {
  background: none;
  padding: 0;
  border-radius: 0;
}

.note-preview-content :deep(ul),
.note-preview-content :deep(ol) {
  margin: 8px 0;
  padding-left: 24px;
}

.note-preview-content :deep(li) {
  margin: 4px 0;
}

.note-preview-content :deep(a) {
  color: var(--primary-color);
  text-decoration: underline;
}

.note-preview-content :deep(blockquote) {
  margin: 12px 0;
  padding: 10px 16px;
  border-left: 4px solid var(--primary-color);
  background: var(--hover-color);
  border-radius: 0 8px 8px 0;
  color: var(--text-secondary);
}

.note-preview-content :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 12px 0;
  font-size: 14px;
}

.note-preview-content :deep(th),
.note-preview-content :deep(td) {
  border: 1px solid var(--border-color);
  padding: 8px 12px;
  text-align: left;
}

.note-preview-content :deep(th) {
  background: var(--hover-color);
  font-weight: 600;
}

.note-preview-content :deep(hr) {
  border: none;
  border-top: 1px solid var(--border-color);
  margin: 16px 0;
}

.note-preview-content :deep(img) {
  max-width: 100%;
  border-radius: 8px;
  margin: 8px 0;
}

.note-preview-content :deep(input[type="checkbox"]) {
  margin-right: 6px;
}

/* Expanded note/sticky content */
.share-expanded-content {
  margin-top: 8px;
  padding: 12px;
  background: var(--card-bg);
  border-radius: 8px;
  font-size: 13px;
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
</style>
