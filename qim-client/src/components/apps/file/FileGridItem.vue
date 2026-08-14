<template>
  <div
    class="file-grid-item"
    :class="{ 'file-grid-item--selected': isSelected }"
    @click="handleClick"
    @dblclick="handleDoubleClick"
    @contextmenu.prevent="handleContextMenu"
    @mouseenter="isHovered = true"
    @mouseleave="isHovered = false"
  >
    <!-- 文件预览区域 -->
    <div class="file-preview">
      <!-- 图片缩略图 -->
      <div v-if="isImage(file.mime_type)" class="preview-thumbnail">
        <img :src="thumbnailUrl" :alt="file.name" @error="handleImageError" />
      </div>

      <!-- 其他文件类型图标 -->
      <div v-else class="preview-icon">
        <i :class="getFileIcon(file.mime_type)" :style="{ color: getFileIconColor(file.mime_type) }"></i>
      </div>

      <!-- 星标标记 -->
      <div v-if="file.is_starred" class="star-badge">
        <i class="fas fa-star"></i>
      </div>

      <!-- 操作按钮 -->
      <transition name="fade">
        <div v-if="isHovered" class="file-actions">
          <button class="action-btn" title="预览" @click.stop="handlePreview">
            <i class="fas fa-eye"></i>
          </button>
          <button class="action-btn" title="下载" @click.stop="handleDownload">
            <i class="fas fa-download"></i>
          </button>
          <button
            class="action-btn"
            :title="file.is_starred ? '取消星标' : '添加星标'"
            @click.stop="handleStar"
          >
            <i :class="file.is_starred ? 'fas fa-star' : 'far fa-star'"></i>
          </button>
          <button class="action-btn" title="分享" @click.stop="handleShare">
            <i class="fas fa-share-alt"></i>
          </button>
          <button class="action-btn action-btn--danger" title="删除" @click.stop="handleDelete">
            <i class="fas fa-trash"></i>
          </button>
        </div>
      </transition>
    </div>

    <!-- 文件信息 -->
    <div class="file-info">
      <div
        class="file-name-wrap"
        @mouseenter="showTooltip"
        @mouseleave="hideTooltip"
      >
        <div class="file-name">{{ file.name }}</div>
      </div>
      <Teleport to="body">
        <div
          v-if="tooltipVisible"
          class="name-tooltip"
          :style="tooltipStyle"
        >
          {{ file.name }}
        </div>
      </Teleport>
      <div class="file-meta">
        <span class="file-size">{{ formatFileSize(file.size) }}</span>
        <span class="file-type">{{ getFileTypeLabel(file.mime_type) }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onUnmounted } from 'vue'
import { fileApi, type FileItem } from '../../../api/file'
import {
  isImage,
  getFileIcon,
  getFileIconColor,
  formatFileSize,
  getFileTypeLabel
} from '../../../utils/fileType'

defineOptions({
  name: 'FileGridItem'
})

interface Props {
  file: FileItem
  isSelected?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  isSelected: false
})

const emit = defineEmits<{
  (e: 'click', file: FileItem): void
  (e: 'dblclick', file: FileItem): void
  (e: 'context-menu', file: FileItem, event: MouseEvent): void
  (e: 'preview', file: FileItem): void
  (e: 'download', file: FileItem): void
  (e: 'star', file: FileItem): void
  (e: 'share', file: FileItem): void
  (e: 'delete', file: FileItem): void
}>()

const isHovered = ref(false)
const imageError = ref(false)
const tooltipVisible = ref(false)
const tooltipStyle = ref({})

const showTooltip = (event: MouseEvent) => {
  const target = event.currentTarget as HTMLElement
  const nameEl = target.querySelector('.file-name') as HTMLElement
  if (!nameEl) return
  const isTruncated = nameEl.scrollWidth > nameEl.clientWidth
  if (!isTruncated) return
  const rect = nameEl.getBoundingClientRect()
  tooltipStyle.value = {
    position: 'fixed',
    top: `${rect.top - 30}px`,
    left: `${rect.left}px`,
  }
  tooltipVisible.value = true
}

const hideTooltip = () => {
  tooltipVisible.value = false
}

// 缩略图接口需 JWT 认证，无法用裸 URL，改为带 token 请求拉取 blob 再生成本地 URL
const thumbnailUrl = ref('')

function revokeThumbnail() {
  if (thumbnailUrl.value) {
    URL.revokeObjectURL(thumbnailUrl.value)
    thumbnailUrl.value = ''
  }
}

async function loadThumbnail() {
  revokeThumbnail()
  imageError.value = false
  if (!props.file?.id || !isImage(props.file.mime_type)) return
  try {
    const res = await fileApi.previewFile(props.file.id, true)
    thumbnailUrl.value = URL.createObjectURL(res.data as Blob)
  } catch {
    imageError.value = true
  }
}

watch(() => props.file?.id, () => {
  loadThumbnail()
}, { immediate: true })

onUnmounted(() => {
  revokeThumbnail()
})

function handleImageError() {
  imageError.value = true
}

function handleClick() {
  emit('click', props.file)
}

function handleDoubleClick() {
  emit('dblclick', props.file)
}

function handleContextMenu(event: MouseEvent) {
  emit('context-menu', props.file, event)
}

function handlePreview() {
  emit('preview', props.file)
}

function handleDownload() {
  emit('download', props.file)
}

function handleStar() {
  emit('star', props.file)
}

function handleShare() {
  emit('share', props.file)
}

function handleDelete() {
  emit('delete', props.file)
}
</script>

<style scoped>
.file-grid-item {
  display: flex;
  flex-direction: column;
  border-radius: 12px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  cursor: pointer;
  transition: all var(--transition-base);
  overflow: hidden;
}

.file-grid-item:hover {
  border-color: var(--primary-color);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  transform: translateY(-2px);
}

.file-grid-item--selected {
  border-color: var(--primary-color);
  background: var(--primary-light);
  box-shadow: 0 0 0 2px rgba(0, 122, 255, 0.2);
}

.file-preview {
  position: relative;
  width: 100%;
  padding-top: 100%; /* 1:1 宽高比 */
  background: var(--hover-color);
  overflow: hidden;
}

.preview-thumbnail {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.preview-thumbnail img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.preview-icon {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.star-badge {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--color-warning-500);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--font-size-xxs);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
}

.file-actions {
  position: absolute;
  top: 8px;
  left: 8px;
  display: flex;
  gap: 4px;
  background: rgba(0, 0, 0, 0.7);
  padding: 4px;
  border-radius: 8px;
  backdrop-filter: blur(4px);
}

.action-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  color: white;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--font-size-xxs);
  transition: all var(--transition-base);
}

.action-btn:hover {
  background: rgba(255, 255, 255, 0.2);
}

.action-btn--danger:hover {
  background: var(--color-error-500);
}

.file-info {
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.file-name-wrap {
  position: relative;
}

.file-name {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--text-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.name-tooltip {
  padding: 4px 10px;
  background: var(--color-gray-900, #1a202c);
  color: #fff;
  font-size: var(--font-size-xxs);
  border-radius: 4px;
  white-space: nowrap;
  max-width: 320px;
  overflow: hidden;
  text-overflow: ellipsis;
  z-index: 1070;
  pointer-events: none;
}

.file-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
}

.file-size {
  flex-shrink: 0;
}

.file-type {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 过渡动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity var(--transition-base);
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* 响应式 */
@media (max-width: 768px) {
  .file-preview {
    padding-top: 75%; /* 4:3 宽高比 */
  }

  .preview-icon {
    font-size: 36px;
  }

  .file-info {
    padding: 8px;
  }
}
</style>
