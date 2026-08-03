<template>
  <!-- 文件候选项：图标 + 名称 + 大小/时间 -->
  <span class="file-cmd-item">
    <span class="file-cmd-item__icon" :style="{ color: iconColor }">
      <i :class="fileIcon"></i>
    </span>
    <span class="file-cmd-item__main">
      <span class="file-cmd-item__name">{{ fileName }}</span>
      <span class="file-cmd-item__meta">{{ sizeLabel }}</span>
    </span>
    <span v-if="file.updated_at" class="file-cmd-item__time">
      <i class="far fa-clock"></i>
      {{ timeLabel }}
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { FileItem } from '../../api/file'

const props = defineProps<{
  item: FileItem
  active: boolean
}>()

const file = computed(() => props.item)

const fileName = computed(() => file.value.name || file.value.original_name || '未命名文件')

// 文件大小格式化
const sizeLabel = computed(() => {
  const size = file.value.size
  if (!size) return ''
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  if (size < 1024 * 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)} MB`
  return `${(size / 1024 / 1024 / 1024).toFixed(1)} GB`
})

// 根据 mime_type 或扩展名选图标
const fileIcon = computed(() => {
  const mime = file.value.mime_type || ''
  const name = fileName.value
  if (mime.startsWith('image/')) return 'fas fa-image'
  if (mime.startsWith('video/')) return 'fas fa-video'
  if (mime.startsWith('audio/')) return 'fas fa-music'
  if (mime.includes('pdf') || name.endsWith('.pdf')) return 'fas fa-file-pdf'
  if (mime.includes('word') || /\.(doc|docx)$/i.test(name)) return 'fas fa-file-word'
  if (mime.includes('excel') || mime.includes('sheet') || /\.(xls|xlsx|csv)$/i.test(name)) return 'fas fa-file-excel'
  if (mime.includes('powerpoint') || mime.includes('presentation') || /\.(ppt|pptx)$/i.test(name)) return 'fas fa-file-powerpoint'
  if (mime.includes('zip') || mime.includes('compressed') || /\.(zip|rar|7z|tar|gz)$/i.test(name)) return 'fas fa-file-archive'
  return 'fas fa-file'
})

const iconColor = computed(() => {
  const icon = fileIcon.value
  if (icon.includes('image')) return '#67c23a'
  if (icon.includes('video')) return '#e6a23c'
  if (icon.includes('audio')) return '#e6a23c'
  if (icon.includes('pdf')) return '#f56c6c'
  if (icon.includes('word')) return '#409eff'
  if (icon.includes('excel')) return '#67c23a'
  if (icon.includes('powerpoint')) return '#f56c6c'
  if (icon.includes('archive')) return '#909399'
  return '#909399'
})

const timeLabel = computed(() => {
  const d = new Date(file.value.updated_at)
  if (isNaN(d.getTime())) return ''
  const now = new Date()
  const sameYear = d.getFullYear() === now.getFullYear()
  const opts: Intl.DateTimeFormatOptions = sameYear
    ? { month: 'numeric', day: 'numeric' }
    : { year: 'numeric', month: 'numeric', day: 'numeric' }
  return d.toLocaleDateString('zh-CN', opts)
})
</script>

<style scoped>
.file-cmd-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}
.file-cmd-item__icon {
  font-size: 14px;
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  width: 16px;
}
.file-cmd-item__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.file-cmd-item__name {
  font-size: 13px;
  color: var(--text-color, #303133);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.file-cmd-item__meta {
  font-size: 11px;
  color: var(--text-secondary, #909399);
}
.file-cmd-item__time {
  font-size: 11px;
  color: var(--text-secondary, #909399);
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 3px;
}
</style>
