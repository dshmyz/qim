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
import { computed, onBeforeUnmount, watch } from 'vue'

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

// blob URL 缓存：以文件唯一标识为键，而非数组下标。
// 历史问题：原先以 index 为键，handleSend 用 pendingFiles.value = [] 清空数组时
// 绕过了 handleRemove，缓存残留；下次截图复用 index 0，命中旧 blob URL，预览显示上一次的图。
// 改用 fileKey 后，新截图的 File 对象不同，key 不同，不会命中旧缓存。
const objectUrls = new Map<string, string>()

const fileKey = (file: File): string => {
  return `${file.name}|${file.size}|${file.lastModified}`
}

const createThumbnailUrl = (file: File): string => {
  const key = fileKey(file)
  let url = objectUrls.get(key)
  if (!url) {
    url = URL.createObjectURL(file)
    objectUrls.set(key, url)
  }
  return url
}

// pendingFiles 变化时清理不再被引用的 blob URL，避免内存泄漏。
// 覆盖发送清空、splice 移除等所有路径，无需依赖父组件调用 handleRemove。
watch(() => props.pendingFiles, (files) => {
  const aliveKeys = new Set<string>()
  for (const f of files) {
    aliveKeys.add(fileKey(f.file))
  }
  for (const [key, url] of objectUrls) {
    if (!aliveKeys.has(key)) {
      URL.revokeObjectURL(url)
      objectUrls.delete(key)
    }
  }
}, { deep: true })

const isImageFile = (file: File): boolean => {
  return file.type.startsWith('image/')
}

const imageItems = computed<PreviewItem[]>(() => {
  return props.pendingFiles
    .map((f, i) => ({
      file: f.file,
      name: f.name,
      originalIndex: i,
      thumbnailUrl: createThumbnailUrl(f.file)
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
  // index 仍用于通知父组件按位置 splice（父组件 removePendingFile 用 splice(index, 1)）
  // blob URL 的清理统一由 watch 负责，此处无需手动 revoke
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
  font-size: var(--font-size-sm);
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
  font-size: var(--font-size-xxs);
  transition: border-color 0.15s ease;
}

.preview-file-item:hover {
  border-color: var(--primary-color);
}

.preview-file-icon {
  display: flex;
  align-items: center;
  color: var(--primary-color);
  font-size: var(--font-size-sm);
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
  font-size: var(--font-size-sm);
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