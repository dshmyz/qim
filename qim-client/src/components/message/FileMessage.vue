<template>
  <div class="message-bubble file-message" :class="{ self: isSelf }">
    <div class="file-info">
      <div class="file-icon-container">
        <div class="file-icon">
          <i :class="getFileIcon(content)"></i>
        </div>
        <div class="file-size" v-if="fileSize">
          {{ formatFileSize(fileSize) }}
        </div>
      </div>
      <div class="file-details">
        <div class="file-name">{{ fileName || content.split('/').pop() || content }}</div>
        <div class="file-actions">
          <button class="file-action-btn" @click="downloadFile">下载</button>
          <button class="file-action-btn" @click="saveFileAs">另存为</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  content: string
  fileName?: string
  fileSize?: number
  isSelf?: boolean
}>()

const emit = defineEmits<{
  download: [url: string, fileName?: string]
  saveAs: [url: string, fileName?: string]
}>()

const getFileIcon = (filePath: string): string => {
  const ext = filePath.split('.').pop()?.toLowerCase()
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
      return 'fas fa-file-image'
    case 'mp3':
    case 'wav':
    case 'ogg':
      return 'fas fa-file-audio'
    case 'mp4':
    case 'avi':
    case 'mov':
    case 'wmv':
      return 'fas fa-file-video'
    case 'zip':
    case 'rar':
    case '7z':
      return 'fas fa-file-archive'
    case 'txt':
      return 'fas fa-file-alt'
    case 'js':
    case 'ts':
    case 'html':
    case 'css':
    case 'php':
    case 'py':
    case 'java':
    case 'cpp':
      return 'fas fa-file-code'
    default:
      return 'fas fa-file'
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
  emit('download', props.content, props.fileName)
}

const saveFileAs = () => {
  emit('saveAs', props.content, props.fileName)
}
</script>

<style scoped>
.file-message {
  background: var(--sidebar-bg);
  border-radius: 12px;
  padding: 14px;
  width: fit-content;
  max-width: 100%;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  transition: all 0.2s ease;
  box-sizing: border-box;
}

.file-message:hover {
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.12);
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
  gap: 4px;
  flex-shrink: 0;
}

.file-icon {
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

.file-size {
  font-size: 11px;
  color: var(--text-secondary);
  line-height: 1.2;
  white-space: nowrap;
  text-align: center;
  margin-bottom: 4px;
}

.file-details {
  flex: 1;
  min-width: 0;
}

.file-name {
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

.file-actions {
  display: flex;
  gap: 8px;
  margin-top: 4px;
}

.file-action-btn {
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

.file-action-btn:hover {
  background-color: var(--primary-color);
  color: #fff;
  border-color: var(--primary-color);
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.3);
  transform: translateY(-1px);
}

.file-action-btn:active {
  transform: translateY(0);
  box-shadow: 0 2px 8px rgba(24, 144, 255, 0.3);
}

/* 自己的文件消息样式 */
.file-message.self {
  background: var(--primary-color);
}

.file-message.self .file-name {
  color: #fff;
}

.file-message.self .file-icon {
  background-color: rgba(255, 255, 255, 0.2);
  border-color: rgba(255, 255, 255, 0.3);
  color: #fff;
}

.file-message.self .file-size {
  color: rgba(255, 255, 255, 0.8);
}

.file-message.self .file-action-btn {
  background-color: rgba(255, 255, 255, 0.2);
  border: none;
  border-color: rgba(255, 255, 255, 0.3);
  color: #fff;
}

.file-message.self .file-action-btn:hover {
  background-color: rgba(255, 255, 255, 0.3);
  border-color: rgba(255, 255, 255, 0.4);
  box-shadow: 0 4px 12px rgba(255, 255, 255, 0.3);
}
</style>