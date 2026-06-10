<template>
  <div class="message-bubble file-message" :class="{ self: isSelf }">
    <div class="file-info">
      <div class="file-icon-container">
        <div class="file-icon" :style="{ color: iconColor }">
          <i :class="fileIcon"></i>
        </div>
        <div class="file-type-label" v-if="fileTypeLabel">{{ fileTypeLabel }}</div>
      </div>
      <div class="file-details">
        <div class="file-name">{{ fileName || fileUrl.split('/').pop() || fileUrl }}</div>
        <div class="file-size" v-if="fileSize">{{ formatFileSize(fileSize) }}</div>
        <div class="file-actions">
          <button class="file-action-btn" @click="downloadFile">
            <i class="fas fa-download"></i>
            下载
          </button>
          <button class="file-action-btn" @click="saveFileAs">
            <i class="fas fa-save"></i>
            另存为
          </button>
        </div>
      </div>
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
.file-message {
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

.file-message::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: linear-gradient(90deg, var(--primary-color), #667eea, #764ba2);
  opacity: 0;
  transition: opacity 0.2s ease;
}

.file-message:hover {
  transform: translateY(-1px);
}

.file-message:hover::before {
  opacity: 1;
}

.file-info {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.file-icon-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex-shrink: 0;
}

.file-icon {
  font-size: 20px;
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(128, 128, 128, 0.06);
  border-radius: 12px;
}

.file-type-label {
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

.file-details {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.file-name {
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

.file-size {
  font-size: 12px;
  color: var(--text-secondary);
  white-space: nowrap;
  text-align: left;
  font-weight: 500;
}

.file-actions {
  display: flex;
  gap: 6px;
  margin-top: 4px;
}

.file-action-btn {
  padding: 6px 12px;
  font-size: 12px;
  border-radius: 6px;
  border: none;
  background: rgba(128, 128, 128, 0.08);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 5px;
  white-space: nowrap;
}

.file-action-btn i {
  font-size: 11px;
  opacity: 0.7;
}

.file-action-btn:hover {
  background: rgba(128, 128, 128, 0.12);
  color: var(--text-color);
  transform: translateY(-1px);
}

.file-action-btn:hover i {
  opacity: 1;
}

.file-action-btn:active {
  transform: translateY(0);
}

/* 自己的文件消息样式 */
.file-message.self {
  background: var(--primary-color);
  border: none;
}

.file-message.self::before {
  background: linear-gradient(90deg, rgba(255, 255, 255, 0.4), rgba(255, 255, 255, 0.15), rgba(255, 255, 255, 0.4));
  opacity: 1;
}

.file-message.self .file-name {
  color: #ffffff;
  font-weight: 600;
}

.file-message.self .file-size {
  color: rgba(255, 255, 255, 0.85);
}

.file-message.self .file-icon {
  background: rgba(255, 255, 255, 0.95);
  color: var(--primary-color);
}

.file-message.self .file-type-label {
  background: rgba(255, 255, 255, 0.2);
  color: rgba(255, 255, 255, 0.85);
}

.file-message.self .file-action-btn {
  background: rgba(255, 255, 255, 0.95);
  color: var(--primary-color);
}

.file-message.self .file-action-btn:hover {
  background: #ffffff;
  transform: translateY(-1px);
}

/* 深色主题下对方消息的文件图标 */
[data-theme="elegant-dark"] .file-message:not(.self) .file-icon {
  background: rgba(255, 255, 255, 0.1);
  border-color: rgba(255, 255, 255, 0.15);
}
</style>