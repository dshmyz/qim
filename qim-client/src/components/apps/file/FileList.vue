<template>
  <div class="file-list">
    <!-- 工具栏 -->
    <div class="file-list-toolbar">
      <div class="toolbar-left">
        <span class="file-count">共 {{ total }} 个文件</span>
      </div>
      <div class="toolbar-right">
        <!-- 视图切换 -->
        <div class="view-toggle">
          <button
            :class="['toggle-btn', { active: viewMode === 'grid' }]"
            title="网格视图"
            @click="viewMode = 'grid'"
          >
            <i class="fas fa-th"></i>
          </button>
          <button
            :class="['toggle-btn', { active: viewMode === 'list' }]"
            title="列表视图"
            @click="viewMode = 'list'"
          >
            <i class="fas fa-list"></i>
          </button>
        </div>
      </div>
    </div>

    <!-- 文件列表 -->
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
      <div v-else class="file-table">
        <!-- 表头 -->
        <div class="file-table-header">
          <div class="header-name">文件名</div>
          <div class="header-type">类型</div>
          <div class="header-size">大小</div>
          <div class="header-date">修改时间</div>
          <div class="header-star">星标</div>
          <div class="header-actions">操作</div>
        </div>

        <!-- 表体 -->
        <div class="file-table-body">
          <FileListItem
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
import { ref } from 'vue'
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
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  hasMore: false
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

const viewMode = ref<'grid' | 'list'>('grid')
const selectedFileIds = ref<Set<number>>(new Set())
const scrollContainerRef = ref<HTMLElement | null>(null)

// 处理文件点击
function handleFileClick(file: FileItem) {
  const newSelection = new Set(selectedFileIds.value)
  if (newSelection.has(file.id)) {
    newSelection.delete(file.id)
  } else {
    newSelection.clear() // 单选模式
    newSelection.add(file.id)
  }
  selectedFileIds.value = newSelection
  emit('selection-change', newSelection)
}

// 处理文件双击
function handleFileDoubleClick(file: FileItem) {
  emit('preview', file)
}

// 处理文件预览
function handleFilePreview(file: FileItem) {
  emit('preview', file)
}

// 处理文件下载
function handleFileDownload(file: FileItem) {
  emit('download', file)
}

// 处理文件星标
function handleFileStar(file: FileItem) {
  emit('star', file)
}

// 处理文件分享
function handleFileShare(file: FileItem) {
  emit('share', file)
}

// 处理文件删除
function handleFileDelete(file: FileItem) {
  emit('delete', file)
}

// 处理滚动（无限滚动）
function handleScroll() {
  if (!scrollContainerRef.value || props.loading || !props.hasMore) return

  const { scrollTop, scrollHeight, clientHeight } = scrollContainerRef.value
  const scrollBottom = scrollHeight - scrollTop - clientHeight

  // 距离底部 100px 时触发加载
  if (scrollBottom < 100) {
    loadMore()
  }
}

// 加载更多
function loadMore() {
  if (!props.loading && props.hasMore) {
    emit('load-more')
  }
}

// 清除选择
function clearSelection() {
  selectedFileIds.value = new Set()
  emit('selection-change', selectedFileIds.value)
}

// 暴露方法给父组件
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

/* 工具栏 */
.file-list-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  background: var(--card-bg);
  flex-shrink: 0;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.file-count {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.view-toggle {
  display: flex;
  gap: 4px;
  background: var(--hover-color);
  padding: 4px;
  border-radius: 8px;
}

.toggle-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  transition: all var(--transition-base);
}

.toggle-btn:hover {
  color: var(--text-color);
}

.toggle-btn.active {
  background: var(--card-bg);
  color: var(--primary-color);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

/* 文件列表容器 */
.file-list-container {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
}

/* 网格视图 */
.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 16px;
  padding: 16px;
}

/* 列表视图 */
.file-table {
  display: flex;
  flex-direction: column;
}

.file-table-header {
  display: grid;
  grid-template-columns: 40px 1fr 100px 100px 140px 40px 140px;
  gap: 12px;
  padding: 12px 16px;
  background: var(--hover-color);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border-color);
  position: sticky;
  top: 0;
  z-index: 10;
}

.header-name {
  grid-column: 2;
}

.file-table-body {
  flex: 1;
}

/* 加载状态 */
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

/* 加载更多 */
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

/* 空状态 */
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

/* 响应式 */
@media (max-width: 1024px) {
  .file-table-header {
    grid-template-columns: 40px 1fr 80px 80px 120px 40px 120px;
    gap: 8px;
    padding: 10px 12px;
  }
}

@media (max-width: 768px) {
  .file-list-toolbar {
    padding: 8px 12px;
  }

  .file-grid {
    grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
    gap: 12px;
    padding: 12px;
  }

  .file-table-header {
    grid-template-columns: 40px 1fr 60px 40px 100px;
    gap: 8px;
    padding: 8px;
  }

  .header-type,
  .header-star {
    display: none;
  }
}
</style>
