<template>
  <div class="pending-files-preview" :class="{ 'has-content': hasContent }">
    <!-- 图片缩略图水平滚动画廊 -->
    <div v-if="imageItems.length > 0" class="preview-gallery">
      <div
        v-for="item in imageItems"
        :key="item.originalIndex"
        class="preview-thumb"
      >
        <img :src="item.thumbnailUrl" :alt="item.name" class="preview-thumb-img" />
        <button
          class="preview-thumb-remove"
          @click="handleRemove(item.originalIndex)"
          title="移除"
        >×</button>
      </div>
    </div>

    <!-- 非图片文件紧凑列表 -->
    <div v-if="fileItems.length > 0" class="preview-files">
      <div
        v-for="item in fileItems"
        :key="item.originalIndex"
        class="preview-file-item"
      >
        <span class="preview-file-icon">
          <i :class="getFileIcon(item.name)"></i>
        </span>
        <span class="preview-file-name">{{ item.name }}</span>
        <button
          class="preview-file-remove"
          @click="handleRemove(item.originalIndex)"
          title="移除"
        >×</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount } from 'vue'

interface PendingFile {
  file: File
  name: string
}

interface PreviewItem {
  file: File
  name: string
  originalIndex: number
  thumbnailUrl: string
}

interface Props {
  pendingFiles: PendingFile[]
  getFileIcon: (fileName: string) => string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'remove', index: number): void
}>()

const objectUrls = new Map<number, string>()

const createThumbnailUrl = (file: File, index: number): string => {
  if (!objectUrls.has(index)) {
    objectUrls.set(index, URL.createObjectURL(file))
  }
  return objectUrls.get(index)!
}

const isImageFile = (file: File): boolean => {
  return file.type.startsWith('image/')
}

const imageItems = computed<PreviewItem[]>(() => {
  return props.pendingFiles
    .map((f, i) => ({
      file: f.file,
      name: f.name,
      originalIndex: i,
      thumbnailUrl: createThumbnailUrl(f.file, i)
    }))
    .filter(item => isImageFile(item.file))
})

const fileItems = computed(() => {
  return props.pendingFiles
    .map((f, i) => ({ file: f.file, name: f.name, originalIndex: i }))
    .filter(item => !isImageFile(item.file))
})

const hasContent = computed(() => {
  return imageItems.value.length > 0 || fileItems.value.length > 0
})

const handleRemove = (index: number) => {
  const url = objectUrls.get(index)
  if (url) {
    URL.revokeObjectURL(url)
    objectUrls.delete(index)
  }
  emit('remove', index)
}

onBeforeUnmount(() => {
  objectUrls.forEach(url => URL.revokeObjectURL(url))
  objectUrls.clear()
})
</script>

<style scoped>
.pending-files-preview {
  position: relative;
}

/* 图片缩略画廊 */
.preview-gallery {
  display: flex;
  gap: 8px;
  padding: 10px 12px 8px;
  overflow-x: auto;
  scrollbar-width: thin;
}

.preview-thumb {
  position: relative;
  flex-shrink: 0;
  width: 80px;
  height: 80px;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--border-color);
  background: var(--card-bg);
  cursor: default;
}

.preview-thumb-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.preview-thumb-remove {
  position: absolute;
  top: 4px;
  right: 4px;
  width: 20px;
  height: 20px;
  border: none;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  border-radius: 50%;
  font-size: 14px;
  line-height: 1;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.15s ease;
  padding: 0;
}

.preview-thumb:hover .preview-thumb-remove {
  opacity: 1;
}

.preview-thumb-remove:hover {
  background: rgba(244, 67, 54, 0.8);
}

/* 非图片文件 */
.preview-files {
  display: flex;
  gap: 6px;
  padding: 6px 12px 8px;
  flex-wrap: wrap;
}

.preview-file-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 12px;
  transition: border-color 0.15s ease;
}

.preview-file-item:hover {
  border-color: var(--primary-color);
}

.preview-file-icon {
  display: flex;
  align-items: center;
  color: var(--primary-color);
  font-size: 14px;
}

.preview-file-name {
  color: var(--text-color);
  max-width: 120px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.preview-file-remove {
  width: 16px;
  height: 16px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  font-size: 14px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  transition: all 0.15s ease;
  padding: 0;
  flex-shrink: 0;
}

.preview-file-remove:hover {
  background: rgba(244, 67, 54, 0.1);
  color: #f44336;
}

/* 滚动条美化 */
.preview-gallery::-webkit-scrollbar {
  height: 4px;
}

.preview-gallery::-webkit-scrollbar-track {
  background: transparent;
}

.preview-gallery::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 2px;
}
</style>