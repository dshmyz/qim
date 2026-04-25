<template>
  <div v-if="visible" class="share-preview-modal" @click="$emit('close')">
    <div class="share-preview-content" @click.stop :class="{ 'sticky-note-preview': previewData.type === 'sticky' }">
      <div class="share-preview-header">
        <h3>{{ previewData.type === 'file' ? '文件详情' : (previewData.type === 'note' ? '笔记详情' : '便签详情') }}</h3>
        <button class="close-btn" @click.stop="$emit('close')">
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
          <div class="share-preview-content-text" v-if="previewData.content" v-html="renderMarkdown(previewData.content)"></div>
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
        <button class="share-file-action-btn" @click="$emit('download-file', previewData.url || previewData.path, previewData.name)">下载</button>
        <button class="share-file-action-btn" @click="$emit('save-file-as', previewData.url || previewData.path, previewData.name)">另存为</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
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
defineEmits<{
  (e: 'close'): void
  (e: 'download-file', url: string, name: string): void
  (e: 'save-file-as', url: string, name: string): void
}>()
</script>
