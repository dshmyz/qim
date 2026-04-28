<template>
  <div
    class="file-list-item"
    :class="{ 'file-list-item--selected': isSelected }"
    @click="handleClick"
    @dblclick="handleDoubleClick"
    @mouseenter="isHovered = true"
    @mouseleave="isHovered = false"
  >
    <!-- 文件图标 -->
    <div class="file-icon">
      <i :class="getFileIcon(file.mime_type)" :style="{ color: getFileIconColor(file.mime_type) }"></i>
    </div>

    <!-- 文件名 -->
    <div class="file-name" :title="file.name">
      {{ file.name }}
    </div>

    <!-- 文件类型 -->
    <div class="file-type">
      {{ getFileTypeLabel(file.mime_type) }}
    </div>

    <!-- 文件大小 -->
    <div class="file-size">
      {{ formatFileSize(file.size) }}
    </div>

    <!-- 修改时间 -->
    <div class="file-date">
      {{ formatFileDate(file.updated_at) }}
    </div>

    <!-- 星标状态 -->
    <div class="file-star">
      <i v-if="file.is_starred" class="fas fa-star" style="color: var(--color-warning-500)"></i>
      <i v-else class="far fa-star" style="color: var(--text-secondary)"></i>
    </div>

    <!-- 操作按钮 -->
    <div class="file-actions">
      <transition name="fade">
        <div v-if="isHovered" class="actions-wrapper">
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
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { FileItem } from '../../../api/file'
import {
  getFileIcon,
  getFileIconColor,
  formatFileSize,
  formatFileDate,
  getFileTypeLabel
} from '../../../utils/fileType'

defineOptions({
  name: 'FileListItem'
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
  (e: 'preview', file: FileItem): void
  (e: 'download', file: FileItem): void
  (e: 'star', file: FileItem): void
  (e: 'share', file: FileItem): void
  (e: 'delete', file: FileItem): void
}>()

const isHovered = ref(false)

function handleClick() {
  emit('click', props.file)
}

function handleDoubleClick() {
  emit('dblclick', props.file)
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
.file-list-item {
  display: grid;
  grid-template-columns: 40px 1fr 100px 100px 140px 40px 140px;
  gap: 12px;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  cursor: pointer;
  transition: all var(--transition-base);
}

.file-list-item:hover {
  background: var(--hover-color);
}

.file-list-item--selected {
  background: var(--primary-light);
  border-left: 3px solid var(--primary-color);
}

.file-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
}

.file-name {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--text-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-type,
.file-size,
.file-date {
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-star {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
}

.file-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.actions-wrapper {
  display: flex;
  gap: 4px;
}

.action-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  transition: all var(--transition-base);
}

.action-btn:hover {
  background: var(--hover-color);
  color: var(--primary-color);
}

.action-btn--danger:hover {
  background: var(--color-error-100);
  color: var(--color-error-500);
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
@media (max-width: 1024px) {
  .file-list-item {
    grid-template-columns: 40px 1fr 80px 80px 120px 40px 120px;
    gap: 8px;
    padding: 10px 12px;
  }
}

@media (max-width: 768px) {
  .file-list-item {
    grid-template-columns: 40px 1fr 60px 40px 100px;
    gap: 8px;
    padding: 8px;
  }

  .file-type,
  .file-star {
    display: none;
  }

  .file-actions {
    justify-content: center;
  }
}
</style>
