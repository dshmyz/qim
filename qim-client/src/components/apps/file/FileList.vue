<template>
  <div class="file-list">
    <div
      ref="scrollContainerRef"
      class="file-list-container"
      @scroll="handleScroll"
    >
      <!-- 网格视图 -->
      <div v-if="viewMode === 'grid'" class="file-grid">
        <FileGridItem
          v-for="file in files"
          :key="file.id"
          :file="file"
          :is-selected="selectedFileIds.has(file.id)"
          @click="handleFileClick"
          @dblclick="handleFileDoubleClick"
          @preview="handleFilePreview"
          @download="handleFileDownload"
          @star="handleFileStar"
          @share="handleFileShare"
          @delete="handleFileDelete"
        />
      </div>

      <!-- 列表视图 -->
      <div v-else class="file-table" :style="gridStyle">
        <!-- 表头 -->
        <div class="file-table-header" :style="gridStyle">
          <div class="header-cell header-icon"></div>
          <div class="header-cell header-name">文件名</div>
          <div class="header-cell header-type">类型</div>
          <div class="header-cell header-size">大小</div>
          <div class="header-cell header-date">修改时间</div>
          <div class="header-cell header-star">星标</div>
          <div class="header-cell header-actions">操作</div>
        </div>

        <!-- 表体 -->
        <div class="file-table-body">
          <FileListItem
            v-for="file in files"
            :key="file.id"
            :file="file"
            :is-selected="selectedFileIds.has(file.id)"
            :grid-style="gridStyle"
            @click="handleFileClick"
            @dblclick="handleFileDoubleClick"
            @preview="handleFilePreview"
            @download="handleFileDownload"
            @star="handleFileStar"
            @share="handleFileShare"
            @delete="handleFileDelete"
          />
        </div>
      </div>

      <!-- 加载状态 -->
      <div v-if="loading" class="loading-state">
        <div class="loading-spinner"></div>
        <span>加载中...</span>
      </div>

      <!-- 加载更多 -->
      <div v-if="hasMore && !loading" class="load-more">
        <button class="load-more-btn" @click="loadMore">
          加载更多
        </button>
      </div>

      <!-- 空状态 -->
      <div v-if="!loading && files.length === 0" class="empty-state">
        <i class="fas fa-folder-open"></i>
        <p>暂无文件</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { FileItem } from '../../../api/file'
import FileGridItem from './FileGridItem.vue'
import FileListItem from './FileListItem.vue'

defineOptions({
  name: 'FileList'
})

interface Props {
  files: FileItem[]
  total: number
  loading?: boolean
  hasMore?: boolean
  viewMode?: 'grid' | 'list'
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  hasMore: false,
  viewMode: 'list'
})

const emit = defineEmits<{
  (e: 'load-more'): void
  (e: 'preview', file: FileItem): void
  (e: 'download', file: FileItem): void
  (e: 'star', file: FileItem): void
  (e: 'share', file: FileItem): void
  (e: 'delete', file: FileItem): void
  (e: 'selection-change', fileIds: Set<number>): void
}>()

const selectedFileIds = ref<Set<number>>(new Set())
const scrollContainerRef = ref<HTMLElement | null>(null)

const columnMinWidths = [40, 120, 60, 60, 100, 40, 160]

const gridStyle = computed(() => {
  const m = columnMinWidths
  return {
    gridTemplateColumns: `minmax(${m[0]}px, ${m[0]}px) minmax(${m[1]}px, 1fr) minmax(${m[2]}px, 80px) minmax(${m[3]}px, 80px) minmax(${m[4]}px, 140px) minmax(${m[5]}px, ${m[5]}px) minmax(${m[6]}px, ${m[6]}px)`
  } as Record<string, string>
})

function handleFileClick(file: FileItem) {
  const newSelection = new Set(selectedFileIds.value)
  if (newSelection.has(file.id)) {
    newSelection.delete(file.id)
  } else {
    newSelection.clear()
    newSelection.add(file.id)
  }
  selectedFileIds.value = newSelection
  emit('selection-change', newSelection)
}

function handleFileDoubleClick(file: FileItem) {
  emit('preview', file)
}

function handleFilePreview(file: FileItem) {
  emit('preview', file)
}

function handleFileDownload(file: FileItem) {
  emit('download', file)
}

function handleFileStar(file: FileItem) {
  emit('star', file)
}

function handleFileShare(file: FileItem) {
  emit('share', file)
}

function handleFileDelete(file: FileItem) {
  emit('delete', file)
}

function handleScroll() {
  if (!scrollContainerRef.value || props.loading || !props.hasMore) return
  const { scrollTop, scrollHeight, clientHeight } = scrollContainerRef.value
  const scrollBottom = scrollHeight - scrollTop - clientHeight
  if (scrollBottom < 100) {
    loadMore()
  }
}

function loadMore() {
  if (!props.loading && props.hasMore) {
    emit('load-more')
  }
}

function clearSelection() {
  selectedFileIds.value = new Set()
  emit('selection-change', selectedFileIds.value)
}

defineExpose({
  clearSelection
})
</script>

<style scoped>
.file-list {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.file-list-container {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
}

.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 16px;
  padding: 16px;
}

.file-table {
  display: flex;
  flex-direction: column;
}

.file-table-header {
  display: grid;
  gap: 0;
  padding: 0 16px;
  background: var(--hover-color);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border-color);
  position: sticky;
  top: 0;
  z-index: 10;
}

.header-cell {
  display: flex;
  align-items: center;
  padding: 10px 8px;
  white-space: nowrap;
}

.header-icon {
  justify-content: center;
}

.header-actions {
  justify-content: center;
}

.file-table-body {
  flex: 1;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px;
  gap: 12px;
  color: var(--text-secondary);
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--border-color);
  border-top-color: var(--primary-color);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.load-more {
  display: flex;
  justify-content: center;
  padding: 20px;
}

.load-more-btn {
  padding: 8px 24px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 8px;
  font-size: var(--font-size-sm);
  cursor: pointer;
  transition: all var(--transition-base);
}

.load-more-btn:hover {
  background: var(--primary-hover);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: var(--text-secondary);
}

.empty-state i {
  font-size: 64px;
  margin-bottom: 16px;
  opacity: 0.3;
}

.empty-state p {
  font-size: var(--font-size-base);
  margin: 0;
}

/* ===== 响应式 ===== */
@media (max-width: 1024px) {
  .file-table-header,
  .file-table-body :deep(.file-list-item) {
    grid-template-columns: 40px minmax(120px, 1fr) minmax(60px, 80px) minmax(60px, 80px) minmax(100px, 140px) 0px 160px;
  }

  .header-star,
  .file-table-body :deep(.file-star) {
    display: none;
  }
}

@media (max-width: 768px) {
  .file-table-header,
  .file-table-body :deep(.file-list-item) {
    grid-template-columns: 40px minmax(100px, 1fr) 0px minmax(60px, 80px) minmax(80px, 120px) 0px 0px;
  }

  .header-type,
  .header-star,
  .header-actions,
  .file-table-body :deep(.file-type),
  .file-table-body :deep(.file-star),
  .file-table-body :deep(.file-actions) {
    display: none;
  }

  .file-table-header {
    padding: 0 8px;
  }
}

@media (max-width: 480px) {
  .file-table-header,
  .file-table-body :deep(.file-list-item) {
    grid-template-columns: 0px minmax(80px, 1fr) 0px 0px minmax(80px, 120px) 0px 0px;
  }

  .header-icon,
  .header-type,
  .header-size,
  .header-star,
  .header-actions,
  .file-table-body :deep(.file-icon),
  .file-table-body :deep(.file-type),
  .file-table-body :deep(.file-size),
  .file-table-body :deep(.file-star),
  .file-table-body :deep(.file-actions) {
    display: none;
  }
}
</style>
