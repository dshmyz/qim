<template>
  <div class="message-bubble file-message attachment-card" :class="{ self: isSelf }">
    <div class="attachment-card__icon file-attachment-icon" :style="{ color: iconColor }">
      <i :class="fileIcon"></i>
    </div>
    <div class="attachment-card__content">
      <div class="attachment-card__title">{{ fileName || fileUrl.split('/').pop() || fileUrl }}</div>
      <div class="attachment-card__meta">
        <span v-if="fileTypeLabel">{{ fileTypeLabel }}</span>
        <span v-if="fileTypeLabel && fileSize"> · </span>
        <span v-if="fileSize">{{ formatFileSize(fileSize) }}</span>
      </div>
    </div>
    <div class="attachment-card__actions">
      <button class="file-action-btn attachment-card__action" @click="downloadFile" title="下载文件">
        <i class="fas fa-download"></i>
      </button>
      <button class="file-action-btn attachment-card__action" @click="saveFileAs" title="另存为">
        <i class="fas fa-save"></i>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { getFileIcon, getFileIconColor, getFileTypeLabel } from '../../utils/fileType'
import { getFileExtension } from '../../utils/fileType'

const props = defineProps<{
  content: string
  isSelf?: boolean
  serverUrl: string
}>()

const emit = defineEmits<{
  download: [url: string, fileName?: string]
  saveAs: [url: string, fileName?: string]
}>()

// 解析文件数据
const fileData = computed(() => {
  try {
    return JSON.parse(props.content)
  } catch {
    return { url: props.content, name: '', size: 0, mimeType: '' }
  }
})

// 获取文件URL
const fileUrl = computed(() => {
  const url = fileData.value.url || ''
  if (url.startsWith('http')) {
    return url
  } else {
    return props.serverUrl + url
  }
})

// 获取文件名
const fileName = computed(() => fileData.value.name || '')

// 获取文件大小
const fileSize = computed(() => Number(fileData.value.size) || 0)

// 获取文件MIME类型
const mimeType = computed(() => fileData.value.mimeType || '')

// 根据MIME类型或扩展名获取文件图标
const fileIcon = computed(() => {
  if (mimeType.value) {
    return getFileIcon(mimeType.value)
  }
  // 回退到基于扩展名的匹配
  const ext = getFileExtension(fileName.value)
  return getIconByExtension(ext)
})

// 根据MIME类型或扩展名获取文件图标颜色
const iconColor = computed(() => {
  if (mimeType.value) {
    return getFileIconColor(mimeType.value)
  }
  const ext = getFileExtension(fileName.value)
  return getIconColorByExtension(ext)
})

// 文件类型标签
const fileTypeLabel = computed(() => {
  if (mimeType.value) {
    return getFileTypeLabel(mimeType.value)
  }
  const ext = getFileExtension(fileName.value)
  return getLabelByExtension(ext)
})

// 根据扩展名获取图标（回退方案）
const getIconByExtension = (ext: string): string => {
  switch (ext) {
    case 'doc':
    case 'docx':
      return 'fas fa-file-word'
    case 'xls':
    case 'xlsx':
      return 'fas fa-file-excel'
    case 'ppt':
    case 'pptx':
      return 'fas fa-file-powerpoint'
    case 'pdf':
      return 'fas fa-file-pdf'
    case 'jpg':
    case 'jpeg':
    case 'png':
    case 'gif':
    case 'webp':
    case 'svg':
    case 'bmp':
      return 'fas fa-file-image'
    case 'mp3':
    case 'wav':
    case 'ogg':
    case 'flac':
    case 'aac':
      return 'fas fa-file-audio'
    case 'mp4':
    case 'avi':
    case 'mov':
    case 'wmv':
    case 'mkv':
    case 'flv':
      return 'fas fa-file-video'
    case 'zip':
    case 'rar':
    case '7z':
    case 'tar':
    case 'gz':
      return 'fas fa-file-archive'
    case 'txt':
    case 'md':
    case 'rtf':
      return 'fas fa-file-alt'
    case 'js':
    case 'ts':
    case 'jsx':
    case 'tsx':
    case 'html':
    case 'htm':
    case 'css':
    case 'scss':
    case 'less':
    case 'php':
    case 'py':
    case 'java':
    case 'cpp':
    case 'c':
    case 'h':
    case 'go':
    case 'rs':
    case 'swift':
    case 'kt':
    case 'rb':
    case 'vue':
    case 'json':
    case 'xml':
    case 'yaml':
    case 'yml':
    case 'toml':
      return 'fas fa-file-code'
    default:
      return 'fas fa-file'
  }
}

// 根据扩展名获取图标颜色（回退方案）
const getIconColorByExtension = (ext: string): string => {
  switch (ext) {
    case 'doc':
    case 'docx':
      return '#2b579a'
    case 'xls':
    case 'xlsx':
      return '#217346'
    case 'ppt':
    case 'pptx':
      return '#d24726'
    case 'pdf':
      return '#e74c3c'
    case 'jpg':
    case 'jpeg':
    case 'png':
    case 'gif':
    case 'webp':
    case 'svg':
    case 'bmp':
      return 'var(--color-success-500)'
    case 'mp3':
    case 'wav':
    case 'ogg':
    case 'flac':
    case 'aac':
      return 'var(--color-warning-500)'
    case 'mp4':
    case 'avi':
    case 'mov':
    case 'wmv':
    case 'mkv':
    case 'flv':
      return 'var(--color-error-500)'
    case 'zip':
    case 'rar':
    case '7z':
    case 'tar':
    case 'gz':
      return '#f39c12'
    case 'txt':
    case 'md':
    case 'rtf':
      return 'var(--primary-color)'
    case 'js':
    case 'ts':
    case 'jsx':
    case 'tsx':
    case 'html':
    case 'htm':
    case 'css':
    case 'scss':
    case 'less':
    case 'php':
    case 'py':
    case 'java':
    case 'cpp':
    case 'c':
    case 'h':
    case 'go':
    case 'rs':
    case 'swift':
    case 'kt':
    case 'rb':
    case 'vue':
    case 'json':
    case 'xml':
    case 'yaml':
    case 'yml':
    case 'toml':
      return '#6e7681'
    default:
      return 'var(--text-secondary)'
  }
}

// 根据扩展名获取类型标签（回退方案）
const getLabelByExtension = (ext: string): string => {
  switch (ext) {
    case 'doc':
    case 'docx':
      return 'Word'
    case 'xls':
    case 'xlsx':
      return 'Excel'
    case 'ppt':
    case 'pptx':
      return 'PPT'
    case 'pdf':
      return 'PDF'
    case 'jpg':
    case 'jpeg':
    case 'png':
    case 'gif':
    case 'webp':
    case 'svg':
    case 'bmp':
      return '图片'
    case 'mp3':
    case 'wav':
    case 'ogg':
    case 'flac':
    case 'aac':
      return '音频'
    case 'mp4':
    case 'avi':
    case 'mov':
    case 'wmv':
    case 'mkv':
    case 'flv':
      return '视频'
    case 'zip':
    case 'rar':
    case '7z':
    case 'tar':
    case 'gz':
      return '压缩包'
    case 'txt':
    case 'md':
    case 'rtf':
      return '文本'
    case 'js':
    case 'ts':
    case 'jsx':
    case 'tsx':
    case 'html':
    case 'htm':
    case 'css':
    case 'scss':
    case 'less':
    case 'php':
    case 'py':
    case 'java':
    case 'cpp':
    case 'c':
    case 'h':
    case 'go':
    case 'rs':
    case 'swift':
    case 'kt':
    case 'rb':
    case 'vue':
    case 'json':
    case 'xml':
    case 'yaml':
    case 'yml':
    case 'toml':
      return '代码'
    default:
      return ''
  }
}

const formatFileSize = (size: number): string => {
  if (size < 1024) {
    return `${size} B`
  } else if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} KB`
  } else if (size < 1024 * 1024 * 1024) {
    return `${(size / (1024 * 1024)).toFixed(1)} MB`
  } else {
    return `${(size / (1024 * 1024 * 1024)).toFixed(1)} GB`
  }
}

const downloadFile = () => {
  emit('download', fileUrl.value, fileName.value)
}

const saveFileAs = () => {
  emit('saveAs', fileUrl.value, fileName.value)
}
</script>

<style scoped>
.attachment-card {
  display: grid;
  grid-template-columns: 42px minmax(0, 1fr) 64px;
  align-items: center;
  gap: 12px;
  width: 280px;
  max-width: min(100%, 320px);
  padding: 12px;
  border-radius: 14px;
  background: color-mix(in srgb, var(--sidebar-bg), transparent 4%);
  border: 1px solid color-mix(in srgb, var(--border-color), transparent 20%);
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.06);
  box-sizing: border-box;
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
  background: color-mix(in srgb, currentColor, transparent 90%);
  font-size: 18px;
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
  min-height: 16px;
  font-size: 12px;
  line-height: 1.35;
  color: var(--text-secondary);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.attachment-card__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
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
  transition: background 0.16s ease, color 0.16s ease, transform 0.16s ease;
}

.attachment-card__action i {
  font-size: 12px;
}

.attachment-card:hover .attachment-card__action {
  color: var(--primary-color);
  background: color-mix(in srgb, var(--primary-color), transparent 90%);
}

.attachment-card__action:active {
  transform: scale(0.96);
}

.file-message.self {
  background: color-mix(in srgb, var(--sidebar-bg), transparent 4%);
  border-color: color-mix(in srgb, var(--border-color), transparent 20%);
  color: var(--text-color);
}

:global(.message-item.self) .file-message.self {
  background: color-mix(in srgb, var(--sidebar-bg), transparent 4%);
  border-color: transparent;
  color: var(--text-color);
}

[data-theme="elegant-dark"] .attachment-card {
  background: color-mix(in srgb, var(--panel-bg), white 5%);
  border-color: rgba(255, 255, 255, 0.12);
  box-shadow: none;
}

[data-theme="elegant-dark"] .file-message.self {
  background: color-mix(in srgb, var(--panel-bg), white 5%);
  border-color: rgba(255, 255, 255, 0.12);
  color: var(--text-color);
}

:global([data-theme="elegant-dark"] .message-item.self) .file-message.self {
  background: color-mix(in srgb, var(--panel-bg), white 5%);
  border-color: transparent;
  color: var(--text-color);
}
</style>
