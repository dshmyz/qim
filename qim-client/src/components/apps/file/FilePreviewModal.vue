<template>
  <QDialog
    :visible="visible"
    :title="file?.name || '文件预览'"
    width="95vw"
    class-name="file-preview-modal preview-modal"
    @update:visible="handleVisibleUpdate"
    @close="handleClose"
  >
    <div class="preview-container">
      <!-- 加载错误（优先，与预览内容互斥） -->
      <div v-if="previewError" class="error-state">
        <i class="fas fa-exclamation-circle"></i>
        <p>预览加载失败</p>
        <button class="download-btn" @click="handleDownload">
          <i class="fas fa-download"></i> 下载查看
        </button>
      </div>

      <!-- 图片预览 -->
      <div v-else-if="isImage(file?.mime_type) && previewUrl" class="preview-wrapper image-preview">
        <img
          :src="previewUrl"
          :alt="file?.name"
          @error="handlePreviewError"
        />
      </div>

      <!-- 视频预览 -->
      <div v-else-if="isVideo(file?.mime_type) && previewUrl" class="preview-wrapper video-preview">
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
      <div v-else-if="isAudio(file?.mime_type) && previewUrl" class="preview-wrapper audio-preview">
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
      <div v-else-if="isPDF(file?.mime_type) && previewUrl" class="preview-wrapper pdf-preview-wrapper">
        <PDFPreview
          :url="previewUrl"
          :filename="file?.name"
          @error="handlePreviewError"
        />
      </div>

      <!-- 文本预览 -->
      <div v-else-if="isText(file?.mime_type, file?.name) && previewUrl" class="preview-wrapper text-preview-wrapper">
        <TextPreview
          :url="previewUrl"
          :filename="file?.name"
          @error="handlePreviewError"
        />
      </div>

      <!-- 加载中（可预览类型但 blob 尚未就绪） -->
      <div v-else-if="isPreviewable(file) && !previewUrl" class="preview-wrapper loading-preview">
        <LoadingSpinner text="加载预览中..." />
      </div>

      <!-- 不支持预览 -->
      <div v-else class="preview-wrapper unsupported-preview">
        <i :class="getFileIcon(file?.mime_type)" class="unsupported-icon"></i>
        <p>此文件类型暂不支持在线预览</p>
        <button class="download-btn" @click="handleDownload">
          <i class="fas fa-download"></i> 下载查看
        </button>
      </div>
    </div>

    <template #footer>
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
    </template>
  </QDialog>
</template>

<script setup lang="ts">
import { watch, ref, onUnmounted } from 'vue'
import { fileApi, type FileItem } from '../../../api/file'
import PDFPreview from './PDFPreview.vue'
import TextPreview from './TextPreview.vue'
import LoadingSpinner from '../../shared/LoadingSpinner.vue'
import QDialog from '../../shared/QDialog.vue'

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

// preview 接口需 JWT 认证，无法用裸 URL，改为带 token 请求拉取 blob 再生成本地 URL
const previewUrl = ref('')

function revokePreviewUrl() {
  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value)
    previewUrl.value = ''
  }
}

async function loadPreview() {
  revokePreviewUrl()
  previewError.value = false
  if (!props.file?.id) return
  if (!isPreviewable(props.file)) return
  try {
    const res = await fileApi.previewFile(props.file.id)
    previewUrl.value = URL.createObjectURL(res.data as Blob)
  } catch (err) {
    console.error('预览加载失败:', err)
    previewError.value = true
  }
}

watch(() => [props.visible, props.file?.id] as const, ([newVal]) => {
  if (newVal && props.file?.id) {
    loadPreview()
  } else {
    revokePreviewUrl()
  }
}, { immediate: true })

onUnmounted(() => {
  revokePreviewUrl()
})

function handleClose() {
  emit('close')
}

function handleVisibleUpdate(value: boolean) {
  if (!value) {
    handleClose()
  }
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
  emit('close')
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

function isPreviewable(file?: FileItem | null): boolean {
  if (!file) return false
  return isImage(file.mime_type) || isVideo(file.mime_type) ||
    isAudio(file.mime_type) || isPDF(file.mime_type) || isText(file.mime_type, file.name)
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
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  text-align: center;
}

.unsupported-preview {
  gap: 16px;
}

.loading-preview {
  min-height: 300px;
}

.unsupported-icon {
  font-size: 64px;
  color: var(--text-secondary);
  opacity: 0.4;
  margin-bottom: 16px;
}

.unsupported-preview p {
  font-size: var(--font-size-base);
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
  font-size: var(--font-size-base);
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
  font-size: var(--font-size-sm);
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.download-btn:hover {
  background: var(--primary-hover);
}

.file-meta-info {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: var(--font-size-xs);
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
  font-size: var(--font-size-xs);
  cursor: pointer;
  transition: all 0.2s ease;
}

.action-btn:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

@media (max-width: 480px) {
  .footer-actions {
    width: 100%;
  }

  .action-btn {
    flex: 1;
    justify-content: center;
  }
}
</style>

<style>
.file-preview-modal {
  max-width: 800px;
  width: 95vw;
}

.file-preview-modal .q-dialog__body {
  padding: 20px;
  min-height: 0;
}

.file-preview-modal .q-dialog__footer {
  justify-content: space-between;
  align-items: center;
}

@media (max-width: 480px) {
  .file-preview-modal {
    width: 100vw;
    max-width: 100vw;
    max-height: 100vh;
    border-radius: 0;
  }

  .file-preview-modal .q-dialog__footer {
    flex-direction: column;
    gap: 12px;
  }
}
</style>
