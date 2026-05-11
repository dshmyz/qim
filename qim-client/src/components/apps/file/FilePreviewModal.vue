<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click="handleClose">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <div class="header-info">
            <i :class="getFileIcon(file?.mime_type)" class="header-icon"></i>
            <h3 class="header-title" :title="file?.name">{{ file?.name }}</h3>
          </div>
          <button class="modal-close" @click="handleClose">&times;</button>
        </div>
        <div class="modal-body">
          <div class="preview-container">
            <!-- 图片预览 -->
            <div v-if="isImage(file?.mime_type)" class="preview-wrapper image-preview">
              <img
                :src="previewUrl"
                :alt="file?.name"
                @error="handlePreviewError"
              />
            </div>

            <!-- 视频预览 -->
            <div v-else-if="isVideo(file?.mime_type)" class="preview-wrapper video-preview">
              <video
                :src="previewUrl"
                controls
                autoplay
                @error="handlePreviewError"
              >
                您的浏览器不支持视频播放
              </video>
            </div>

            <!-- 音频预览 -->
            <div v-else-if="isAudio(file?.mime_type)" class="preview-wrapper audio-preview">
              <div class="audio-icon-wrapper">
                <i :class="getFileIcon(file?.mime_type)" class="audio-large-icon"></i>
              </div>
              <audio
                :src="previewUrl"
                controls
                autoplay
                @error="handlePreviewError"
              >
                您的浏览器不支持音频播放
              </audio>
              <p class="audio-filename">{{ file?.name }}</p>
            </div>

            <!-- PDF 预览 -->
            <div v-else-if="isPDF(file?.mime_type)" class="preview-wrapper pdf-preview-wrapper">
              <PDFPreview
                :url="previewUrl"
                :filename="file?.name"
                @error="handlePreviewError"
              />
            </div>

            <!-- 文本预览 -->
            <div v-else-if="isText(file?.mime_type, file?.name)" class="preview-wrapper text-preview-wrapper">
              <TextPreview
                :url="previewUrl"
                :filename="file?.name"
                @error="handlePreviewError"
              />
            </div>

            <!-- 不支持预览 -->
            <div v-else class="preview-wrapper unsupported-preview">
              <i :class="getFileIcon(file?.mime_type)" class="unsupported-icon"></i>
              <p>此文件类型暂不支持在线预览</p>
              <button class="download-btn" @click="handleDownload">
                <i class="fas fa-download"></i> 下载查看
              </button>
            </div>

            <!-- 加载错误 -->
            <div v-if="previewError" class="error-state">
              <i class="fas fa-exclamation-circle"></i>
              <p>预览加载失败</p>
              <button class="download-btn" @click="handleDownload">
                <i class="fas fa-download"></i> 下载查看
              </button>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <div class="file-meta-info">
            <span class="meta-size">{{ formatFileSize(file?.size) }}</span>
            <span class="meta-divider">&middot;</span>
            <span class="meta-date">{{ formatFileDate(file?.created_at) }}</span>
          </div>
          <div class="footer-actions">
            <button class="action-btn" @click="handleDownload">
              <i class="fas fa-download"></i> 下载
            </button>
            <button class="action-btn" @click="handleShare">
              <i class="fas fa-share-alt"></i> 分享
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, watch, ref } from 'vue'
import { type FileItem } from '../../../api/file'
import { API_BASE_URL } from '../../../config'
import PDFPreview from './PDFPreview.vue'
import TextPreview from './TextPreview.vue'

interface Props {
  visible: boolean
  file?: FileItem | null
}

const props = withDefaults(defineProps<Props>(), {
  file: null
})

const emit = defineEmits<{
  close: []
  download: [file: FileItem]
  share: [file: FileItem]
}>()

const previewError = ref(false)

// 使用 computed 保证 URL 的响应性
const serverUrl = localStorage.getItem('serverUrl') || API_BASE_URL

const previewUrl = computed(() => {
  if (!props.file?.id) return ''
  return `${serverUrl}/api/v1/files/${props.file.id}/preview`
})

watch(() => props.visible, (newVal) => {
  if (newVal) {
    previewError.value = false
  }
})

function handleClose() {
  emit('close')
}

function handlePreviewError() {
  previewError.value = true
}

async function handleDownload() {
  if (!props.file) return
  emit('download', props.file)
}

function handleShare() {
  if (!props.file) return
  emit('share', props.file)
}

// 文件类型判断
function isImage(mimeType?: string): boolean {
  return !!mimeType && mimeType.startsWith('image/')
}

function isVideo(mimeType?: string): boolean {
  return !!mimeType && mimeType.startsWith('video/')
}

function isAudio(mimeType?: string): boolean {
  return !!mimeType && mimeType.startsWith('audio/')
}

function isPDF(mimeType?: string): boolean {
  return !!mimeType && mimeType === 'application/pdf'
}

function isText(mimeType?: string, filename?: string): boolean {
  // 检查 MIME 类型
  if (mimeType?.startsWith('text/')) return true

  // 检查文件扩展名
  if (filename) {
    const ext = filename.split('.').pop()?.toLowerCase()
    const textExtensions = ['txt', 'log', 'md', 'json', 'xml', 'csv', 'yml', 'yaml']
    return textExtensions.includes(ext || '')
  }

  return false
}

// 获取对应的 FontAwesome 图标
function getFileIcon(mimeType?: string): string {
  if (!mimeType) return 'fas fa-file'
  if (mimeType.startsWith('image/')) return 'fas fa-image'
  if (mimeType.startsWith('video/')) return 'fas fa-video'
  if (mimeType.startsWith('audio/')) return 'fas fa-music'
  if (mimeType.includes('pdf')) return 'fas fa-file-pdf'
  if (mimeType.includes('word') || mimeType.includes('document')) return 'fas fa-file-word'
  if (mimeType.includes('excel') || mimeType.includes('sheet')) return 'fas fa-file-excel'
  if (mimeType.includes('powerpoint') || mimeType.includes('presentation')) return 'fas fa-file-powerpoint'
  if (mimeType.startsWith('text/')) return 'fas fa-file-alt'
  return 'fas fa-file'
}

// 格式化文件大小
function formatFileSize(size?: number): string {
  if (size === undefined || size === null) return '未知大小'
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  if (size < 1024 * 1024 * 1024) return `${(size / (1024 * 1024)).toFixed(1)} MB`
  return `${(size / (1024 * 1024 * 1024)).toFixed(1)} GB`
}

// 格式化日期
function formatFileDate(dateString?: string): string {
  if (!dateString) return ''
  const date = new Date(dateString)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
  animation: fadeIn 0.3s ease;
  padding: 16px;
  box-sizing: border-box;
}

.modal-content {
  background: var(--card-bg);
  border-radius: 12px;
  width: 95%;
  max-width: 800px;
  max-height: 90vh;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.4);
  animation: slideIn 0.3s ease;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}

.header-info {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.header-icon {
  font-size: 20px;
  color: var(--primary-color);
  flex-shrink: 0;
}

.header-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color);
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.modal-close {
  background: none;
  border: none;
  font-size: 24px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  flex-shrink: 0;
}

.modal-close:hover {
  color: var(--text-color);
  background: var(--hover-color);
}

.modal-body {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
  min-height: 0;
}

.preview-container {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 300px;
}

.preview-wrapper {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 100%;
}

.preview-wrapper img,
.preview-wrapper video {
  max-width: 100%;
  max-height: 60vh;
  object-fit: contain;
  border-radius: 8px;
}

.pdf-preview-wrapper,
.text-preview-wrapper {
  width: 100%;
  height: 60vh;
  min-height: 400px;
}

.audio-preview {
  gap: 20px;
}

.audio-icon-wrapper {
  width: 80px;
  height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--hover-color);
  border-radius: 50%;
  margin-bottom: 16px;
}

.audio-large-icon {
  font-size: 36px;
  color: var(--primary-color);
}

.audio-preview audio {
  width: 100%;
  max-width: 400px;
}

.audio-filename {
  margin: 12px 0 0;
  font-size: 14px;
  color: var(--text-secondary);
  text-align: center;
}

.unsupported-preview {
  gap: 16px;
}

.unsupported-icon {
  font-size: 64px;
  color: var(--text-secondary);
  opacity: 0.4;
  margin-bottom: 16px;
}

.unsupported-preview p {
  font-size: 16px;
  color: var(--text-secondary);
  margin: 0 0 20px;
}

.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 40px;
}

.error-state i {
  font-size: 48px;
  color: var(--error-color);
}

.error-state p {
  font-size: 16px;
  color: var(--text-secondary);
  margin: 0;
}

.download-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 24px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.download-btn:hover {
  background: var(--primary-hover);
}

.modal-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
  flex-shrink: 0;
}

.file-meta-info {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-secondary);
}

.meta-divider {
  opacity: 0.5;
}

.footer-actions {
  display: flex;
  gap: 8px;
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--card-bg);
  color: var(--text-color);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.action-btn:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes slideIn {
  from {
    transform: translateY(-20px) scale(0.95);
    opacity: 0;
  }
  to {
    transform: translateY(0) scale(1);
    opacity: 1;
  }
}

@media (max-width: 480px) {
  .modal-overlay {
    padding: 0;
  }

  .modal-content {
    width: 100%;
    max-width: 100%;
    max-height: 100vh;
    border-radius: 0;
  }

  .modal-footer {
    flex-direction: column;
    gap: 12px;
  }

  .footer-actions {
    width: 100%;
  }

  .action-btn {
    flex: 1;
    justify-content: center;
  }
}
</style>
