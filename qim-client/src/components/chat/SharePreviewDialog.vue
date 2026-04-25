<template>
  <div v-if="visible" class="share-preview-modal" @click="handleClose">
    <div class="share-preview-content" @click.stop :class="{ 'sticky-note-preview': previewData.type === 'sticky' }">
      <div class="share-preview-header">
        <h3>{{ previewData.type === 'file' ? '文件详情' : (previewData.type === 'note' ? '笔记详情' : '便签详情') }}</h3>
        <button class="close-btn" @click.stop="handleClose">
          <i class="fas fa-times"></i>
        </button>
      </div>
      <div class="share-preview-body">
        <!-- 文件类型 -->
        <div v-if="previewData.type === 'file'" class="share-file-content">
          <div class="share-file-icon">
            <i :class="getFileIcon(previewData.url || previewData.path)"></i>
          </div>
          <div class="share-file-info">
            <div class="share-preview-title">{{ previewData.name }}</div>
            <div class="share-file-size" v-if="previewData.size">{{ formatFileSize(previewData.size) }}</div>
          </div>
        </div>
        <!-- 笔记类型 -->
        <div v-else-if="previewData.type === 'note'">
          <div class="share-preview-title">{{ previewData.name }}</div>
          <div class="share-preview-content-text" v-if="previewData.content" v-html="sanitizeMarkdown(renderMarkdown(previewData.content))"></div>
        </div>
        <!-- 便签类型 -->
        <div v-else-if="previewData.type === 'sticky'" class="sticky-note-content">
          <div class="sticky-note-title">{{ previewData.name }}</div>
          <div class="sticky-note-body" v-if="previewData.content">{{ previewData.content }}</div>
        </div>
        <div class="share-preview-meta">
          <span class="share-preview-type">{{ previewData.type === 'file' ? '文件' : (previewData.type === 'note' ? '笔记' : '便签') }}</span>
          <span class="share-preview-time" v-if="previewData.created_at">{{ formatTime(new Date(previewData.created_at).getTime()) }}</span>
        </div>
      </div>
      <!-- 文件操作按钮 -->
      <div v-if="previewData.type === 'file'" class="share-preview-footer">
        <button class="share-file-action-btn" @click="handleDownloadFile(previewData.url || previewData.path, previewData.name)">下载</button>
        <button class="share-file-action-btn" @click="handleSaveFileAs(previewData.url || previewData.path, previewData.name)">另存为</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { sanitizeMarkdown } from '../../utils/sanitize'
interface SharePreviewData {
  type: 'file' | 'note' | 'sticky'
  name: string
  content?: string
  url?: string
  path?: string
  size?: number
  created_at?: string
}

interface Props {
  visible: boolean
  previewData: SharePreviewData
  getFileIcon: (url: string) => string
  formatFileSize: (bytes: number) => string
  renderMarkdown: (content: string) => string
  formatTime: (timestamp: number | string | null | undefined) => string
}

defineProps<Props>()
const emit = defineEmits<{
  (e: 'close'): void
  (e: 'download-file', url: string, name: string): void
  (e: 'save-file-as', url: string, name: string): void
}>()

const handleClose = () => {
  emit('close')
}

const handleDownloadFile = (url: string, name: string) => {
  emit('download-file', url, name)
}

const handleSaveFileAs = (url: string, name: string) => {
  emit('save-file-as', url, name)
}
</script>

<style scoped>
/* 分享内容预览弹窗样式 */
.share-preview-modal {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
  backdrop-filter: blur(8px);
  animation: modalFadeIn 0.2s ease-out;
}

.share-preview-content {
  background: var(--sidebar-bg);
  border-radius: 12px;
  width: 500px;
  max-width: 90%;
  max-height: 80vh;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2), 0 8px 25px rgba(0, 0, 0, 0.15);
  overflow: hidden;
  animation: modalSlideIn 0.3s ease-out;
  display: flex;
  flex-direction: column;
}

/* 便签预览样式 */
.sticky-note-preview {
  background: #fff9c4;
  border-radius: 4px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.sticky-note-preview .share-preview-header {
  background: rgba(0, 0, 0, 0.05);
  border-bottom: 1px solid rgba(0, 0, 0, 0.1);
}

.sticky-note-preview .sticky-note-content {
  font-family: 'Comic Sans MS', cursive, sans-serif;
}

.sticky-note-preview .sticky-note-title {
  font-size: 18px;
  font-weight: bold;
  color: #333;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px dashed rgba(0, 0, 0, 0.2);
}

.sticky-note-preview .sticky-note-body {
  font-size: 16px;
  line-height: 1.5;
  color: #333;
  white-space: pre-wrap;
  word-break: break-word;
}

.sticky-note-preview .share-preview-meta {
  color: #666;
  border-top: 1px dashed rgba(0, 0, 0, 0.2);
  padding-top: 8px;
  margin-top: 12px;
}

.share-preview-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  background: var(--primary-color);
  color: white;
}

.share-preview-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.share-preview-header .close-btn {
  background: rgba(255, 255, 255, 0.2);
  border: none;
  font-size: 18px;
  cursor: pointer;
  color: white;
  padding: 8px;
  border-radius: 50%;
  transition: all 0.3s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.share-preview-header .close-btn:hover {
  background: rgba(255, 255, 255, 0.3);
  transform: rotate(90deg);
}

.share-preview-body {
  padding: 24px;
  overflow-y: auto;
  flex: 1;
}

.share-preview-title {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 16px;
  color: var(--text-color);
  word-break: break-word;
}

.share-preview-content-text {
  font-size: 14px;
  line-height: 1.6;
  color: var(--text-color);
  margin-bottom: 24px;
  white-space: pre-wrap;
  word-break: break-word;
}

.share-preview-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
  color: var(--text-secondary);
  font-size: 12px;
}

.share-preview-type {
  background: var(--primary-light);
  padding: 4px 12px;
  border-radius: 12px;
  color: var(--primary-color);
  font-weight: 500;
}

/* 文件分享内容样式 */
.share-file-content {
  display: flex;
  align-items: center;
  margin-bottom: 24px;
}

.share-file-icon {
  width: 60px;
  height: 60px;
  background: var(--primary-light);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16px;
  font-size: 24px;
  color: var(--primary-color);
  flex-shrink: 0;
}

.share-file-info {
  flex: 1;
}

.share-file-size {
  color: var(--text-secondary);
  font-size: 14px;
  margin-top: 4px;
}

/* 文件操作按钮 */
.share-preview-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 24px;
  border-top: 1px solid var(--border-color);
  background: var(--secondary-color);
}

.share-file-action-btn {
  padding: 8px 20px;
  border: 1px solid var(--primary-color);
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s;
  background: white;
  color: var(--primary-color);
}

.share-file-action-btn:hover {
  background: var(--primary-color);
  color: white;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}
</style>
