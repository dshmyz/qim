<template>
  <AttachmentCard
    class="file-message"
    :is-self="isSelf"
    :style="{ '--ac-icon-bg': 'linear-gradient(135deg, #7c3aed 0%, #a855f7 100%)' }"
    @click="handleCardClick"
  >
    <i :class="fileIcon" />
    <template #content>
      <Tooltip :text="fileName || fileUrl.split('/').pop() || fileUrl">
        <div class="file-title">{{ fileName || fileUrl.split('/').pop() || fileUrl }}</div>
      </Tooltip>
      <div class="file-bottom">
        <div class="file-meta">
          <span v-if="fileTypeLabel">{{ fileTypeLabel }}</span>
          <span v-if="fileTypeLabel && fileSize"> · </span>
          <span v-if="fileSize">{{ formatFileSize(fileSize) }}</span>
        </div>
        <div class="file-actions">
          <template v-if="isDownloaded">
            <button class="attachment-card__btn" @click.stop="openFile" title="打开文件">
              <i class="fas fa-external-link-alt"></i>
            </button>
            <button class="attachment-card__btn" @click.stop="showInFolder" title="在文件夹中显示">
              <i class="fas fa-folder-open"></i>
            </button>
          </template>
          <template v-else>
            <button class="attachment-card__btn" @click.stop="downloadFile" title="下载文件">
              <i class="fas fa-download"></i>
            </button>
            <button class="attachment-card__btn" @click.stop="saveFileAs" title="另存为">
              <i class="fas fa-save"></i>
            </button>
          </template>
        </div>
      </div>
    </template>
    <template v-if="isDownloading" #below>
      <div class="file-progress">
        <div class="file-progress__bar" :style="{ width: progressPercent + '%' }"></div>
      </div>
    </template>
  </AttachmentCard>
</template>

<script setup lang="ts">
import { computed, inject, type Ref } from 'vue'
import { getFileIcon, getFileTypeLabel } from '../../utils/fileType'
import { getFileExtension } from '../../utils/fileType'
import Tooltip from '../shared/Tooltip.vue'
import AttachmentCard from './AttachmentCard.vue'

const props = defineProps<{
  content: string
  isSelf?: boolean
  serverUrl: string
  messageId?: string
}>()

const emit = defineEmits<{
  download: [content: string, messageId?: string]
  saveAs: [content: string, messageId?: string]
}>()

// 下载进度（由 ChatWindow provide，key = 消息 id）
const downloadProgress = inject<Ref<Record<string, number>>>('downloadProgress')
const isDownloading = computed(() => {
  if (!props.messageId || !downloadProgress?.value) return false
  const percent = downloadProgress.value[props.messageId]
  return percent !== undefined && percent >= 0 && percent < 100
})
const progressPercent = computed(() =>
  props.messageId ? (downloadProgress?.value?.[props.messageId] ?? 0) : 0
)

// 已下载文件路径（由 ChatWindow provide）
const downloadedFiles = inject<Ref<Record<string, string>>>('downloadedFiles')
const isDownloaded = computed(() => {
  const result = !!props.messageId && downloadedFiles?.value?.[props.messageId] !== undefined
  return result
})
const filePath = computed(() =>
  props.messageId ? downloadedFiles?.value?.[props.messageId] : undefined
)

// 点击卡片：已下载就打开，否则下载
const handleCardClick = () => {
  if (isDownloaded.value) {
    openFile()
  } else {
    downloadFile()
  }
}

const openFile = async () => {
  if (!filePath.value || !window.electron?.ipcRenderer) return
  await window.electron.ipcRenderer.invoke('open-file', filePath.value)
}

const showInFolder = async () => {
  if (!filePath.value || !window.electron?.ipcRenderer) return
  await window.electron.ipcRenderer.invoke('show-file-in-folder', filePath.value)
}

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
  emit('download', props.content, props.messageId)
}

const saveFileAs = () => {
  emit('saveAs', props.content, props.messageId)
}
</script>

<style scoped>
.file-message :deep(.attachment-card__icon) {
  color: #ffffff;
  background: var(--ac-icon-bg);
  font-size: 18px;
}

.file-title {
  font-size: 14px;
  font-weight: 600;
  line-height: 1.35;
  color: var(--text-color);
  word-break: break-all;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  letter-spacing: -0.01em;
}

.file-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.file-meta {
  min-height: 16px;
  font-size: 12px;
  line-height: 1.35;
  color: var(--text-secondary);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.file-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.file-progress {
  height: 4px;
  border-radius: 2px;
  overflow: hidden;
  background: color-mix(in srgb, var(--text-secondary), transparent 82%);
}

.file-progress__bar {
  height: 100%;
  border-radius: 2px;
  background: var(--primary-color);
  transition: width 0.15s ease;
}
</style>
