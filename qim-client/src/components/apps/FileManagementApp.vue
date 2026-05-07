<template>
  <div class="file-management-app">
    <!-- 顶部导航栏 -->
    <AppHeader title="文件箱" icon="fas fa-folder-open" @back="$emit('back')">
      <template #extra-buttons>
        <button class="icon-btn" @click="$emit('toggleSidebar')" title="切换侧边栏">
          <i class="fas fa-compress"></i>
        </button>
      </template>
      <template #actions>
        <button class="action-btn" @click="showCreateFolderModal = true" title="新建文件夹">
          <i class="fas fa-folder-plus"></i>
          <span>新建</span>
        </button>
        <button class="action-btn primary" @click="triggerFileUpload" title="上传文件">
          <i class="fas fa-cloud-upload-alt"></i>
          <span>上传</span>
        </button>
        <input
          ref="fileInputRef"
          type="file"
          multiple
          style="display: none"
          @change="handleFileUpload"
        />
      </template>
    </AppHeader>

    <!-- 筛选工具栏 -->
    <div class="filter-bar">
      <!-- 搜索框 -->
      <div class="search-wrap">
        <i class="fas fa-search search-icon-inline"></i>
        <input
          type="text"
          class="search-input-inline"
          :value="searchQuery"
          placeholder="搜索文件名..."
          @input="handleSearchInput"
          @keydown.escape="handleSearchClear"
        />
        <button
          v-if="searchQuery"
          class="search-clear-inline"
          @click="handleSearchClear"
        >
          <i class="fas fa-times"></i>
        </button>
      </div>

      <div class="bar-divider"></div>

      <!-- 来源+文件夹 统一下拉 -->
      <div class="filter-select-wrap">
        <i class="fas fa-filter filter-icon"></i>
        <select v-model="filterValue" @change="handleFilterValueChange" class="filter-select">
          <optgroup label="来源">
            <option value="all">全部文件</option>
            <option value="upload">我的上传</option>
            <option value="chat">聊天文件</option>
          </optgroup>
          <optgroup label="快捷">
            <option value="starred">⭐ 星标文件</option>
          </optgroup>
          <optgroup v-if="folders.length" label="文件夹">
            <option v-for="folder in folders" :key="'f-'+folder.id" :value="'folder-'+folder.id">
              📁 {{ folder.name }}
            </option>
          </optgroup>
        </select>
        <i class="fas fa-chevron-down select-arrow"></i>
      </div>

      <!-- 日期筛选 -->
      <FileDateFilter
        :date-from="dateFrom"
        :date-to="dateTo"
        @change="handleDateChange"
        @clear="handleDateClear"
      />

      <!-- 排序 -->
      <div class="filter-select-wrap">
        <i class="fas fa-sort-amount-down filter-icon"></i>
        <select v-model="sortValue" @change="handleSortChange" class="filter-select">
          <option value="created_at_desc">最新优先</option>
          <option value="created_at_asc">最早优先</option>
          <option value="name_asc">名称 A→Z</option>
          <option value="name_desc">名称 Z→A</option>
        </select>
        <i class="fas fa-chevron-down select-arrow"></i>
      </div>

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

      <span class="file-count">{{ total }} 个文件</span>
    </div>

    <!-- 主内容区域 -->
    <div class="app-content">
      <FileList
        ref="fileListRef"
        :files="files"
        :total="total"
        :current-page="currentPage"
        :total-pages="totalPages"
        :is-loading="isLoading"
        :error="error"
        :search-query="searchQuery"
        :filter-type="filterType"
        :show-starred="showStarred"
        :has-files="hasFiles"
        :view-mode="viewMode"
        @refresh="refresh"
        @search="handleSearch"
        @filter-change="handleFilterChange"
        @toggle-starred="toggleStarred"
        @page-change="changePage"
        @preview="handleFilePreview"
        @download="handleFileDownload"
        @star="handleFileStar"
        @share="handleFileShare"
        @delete="handleFileDelete"
        @upload="handleFileUpload"
        @context-menu="handleContextMenu"
        @select="handleFileSelect"
      />
    </div>

    <!-- 创建文件夹模态框 -->
    <CreateFolderModal
      :visible="showCreateFolderModal"
      :folders="folders"
      @close="showCreateFolderModal = false"
      @success="handleFolderCreated"
    />

    <!-- 文件预览模态框 -->
    <FilePreviewModal
      :visible="showPreviewModal"
      :file="previewFile"
      @close="closePreviewModal"
      @download="handleFileDownload"
      @share="handleFileShare"
    />

    <!-- 文件操作模态框 -->
    <FileActionsModal
      :visible="showActionsModal"
      :file="actionFile"
      :folders="folders"
      @close="showActionsModal = false"
      @success="handleActionSuccess"
    />

    <!-- 右键菜单 -->
    <Teleport to="body">
      <div
        v-if="contextMenu.visible"
        class="context-menu"
        :style="{ top: contextMenu.y + 'px', left: contextMenu.x + 'px' }"
      >
        <div class="context-menu-item" @click="handleContextMenuAction('preview')">
          <i class="fas fa-eye"></i>
          <span>预览</span>
        </div>
        <div class="context-menu-item" @click="handleContextMenuAction('download')">
          <i class="fas fa-download"></i>
          <span>下载</span>
        </div>
        <div class="context-menu-item" @click="handleContextMenuAction('rename')">
          <i class="fas fa-edit"></i>
          <span>重命名</span>
        </div>
        <div class="context-menu-item" @click="handleContextMenuAction('move')">
          <i class="fas fa-arrows-alt"></i>
          <span>移动</span>
        </div>
        <div class="context-menu-divider"></div>
        <div class="context-menu-item" @click="handleContextMenuAction('star')">
          <i :class="contextMenu.file?.is_starred ? 'fas fa-star' : 'far fa-star'"></i>
          <span>{{ contextMenu.file?.is_starred ? '取消星标' : '添加星标' }}</span>
        </div>
        <div class="context-menu-item" @click="handleContextMenuAction('share')">
          <i class="fas fa-share-alt"></i>
          <span>分享</span>
        </div>
        <div class="context-menu-divider"></div>
        <div class="context-menu-item danger" @click="handleContextMenuAction('delete')">
          <i class="fas fa-trash"></i>
          <span>删除</span>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import FolderTree from './file/FolderTree.vue'
import FileList from './file/FileList.vue'
import CreateFolderModal from './file/CreateFolderModal.vue'
import FilePreviewModal from './file/FilePreviewModal.vue'
import FileActionsModal from './file/FileActionsModal.vue'
import FileDateFilter from './file/FileDateFilter.vue'
import AppHeader from './AppHeader.vue'
import { useFilePagination } from '../../composables/useFilePagination'
import { useFolderTree, type FolderNode } from '../../composables/useFolderTree'
import { fileApi, type FileItem, type FolderItem } from '../../api/file'
import QMessage from '../../utils/qmessage'

const emit = defineEmits(['back', 'toggleSidebar'])

const {
  files,
  total,
  currentPage,
  totalPages,
  isLoading,
  error,
  searchQuery,
  filterType,
  showStarred,
  sourceFilter,
  hasFiles,
  sortBy,
  sortOrder,
  dateFrom,
  dateTo,
  loadFiles,
  refresh,
  changePage,
  changeFolder,
  changeFilterType,
  toggleStarred,
  changeSource,
  changeSort,
  changeDateRange,
  clearDateRange,
  uploadFile,
  deleteFile,
  toggleFileStar
} = useFilePagination()

const {
  selectedFolder,
  loadRootFolders
} = useFolderTree()

const folderTreeRef = ref<InstanceType<typeof FolderTree> | null>(null)
const fileListRef = ref<InstanceType<typeof FileList> | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)

const showCreateFolderModal = ref(false)
const showPreviewModal = ref(false)
const showActionsModal = ref(false)
const previewFile = ref<FileItem | null>(null)
const actionFile = ref<FileItem | null>(null)

const contextMenu = ref({
  visible: false,
  x: 0,
  y: 0,
  file: null as FileItem | null
})

const folders = ref<FolderItem[]>([])
const filterValue = ref('all')
const viewMode = ref<'grid' | 'list'>('list')

const currentFolderPath = computed(() => {
  if (!selectedFolder.value) return ''
  return selectedFolder.value.path || selectedFolder.value.name
})

const handleFolderSelect = (folder: FolderNode) => {
  changeFolder(folder.id)
}

const handleFilterValueChange = () => {
  const val = filterValue.value
  if (val === 'all') {
    showStarred.value = false
    changeSource(null)
    changeFolder(null)
  } else if (val === 'upload' || val === 'chat') {
    showStarred.value = false
    changeFolder(null)
    changeSource(val)
  } else if (val === 'starred') {
    showStarred.value = true
    changeFolder(null)
    changeSource(null)
  } else if (val.startsWith('folder-')) {
    const folderId = parseInt(val.replace('folder-', ''))
    showStarred.value = false
    changeSource(null)
    changeFolder(folderId)
  }
}

const handleSearch = (query: string) => {
  searchQuery.value = query
}

let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null

const handleSearchInput = (event: Event) => {
  const value = (event.target as HTMLInputElement).value
  searchQuery.value = value
  if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
  searchDebounceTimer = setTimeout(() => {
    searchQuery.value = value
  }, 300)
}

const handleSearchClear = () => {
  searchQuery.value = ''
}

const handleFilterChange = (type: string) => {
  changeFilterType(type)
}

const handleSourceChange = (source: string | null) => {
  changeSource(source)
}

const sortValue = ref('created_at_desc')

const handleSortChange = () => {
  const val = sortValue.value
  let field: string
  let order: string
  if (val.startsWith('created_at_')) {
    field = 'created_at'
    order = val.replace('created_at_', '')
  } else if (val.startsWith('name_')) {
    field = 'name'
    order = val.replace('name_', '')
  } else {
    field = 'created_at'
    order = 'desc'
  }
  changeSort(field, order)
}

const handleDateChange = (from: string, to: string) => {
  changeDateRange(from, to)
}

const handleDateClear = () => {
  clearDateRange()
}

const handleFilePreview = (file: FileItem) => {
  previewFile.value = file
  showPreviewModal.value = true
}

const closePreviewModal = () => {
  showPreviewModal.value = false
  previewFile.value = null
}

const handleFileDownload = async (file: FileItem) => {
  try {
    const response = await fileApi.downloadFile(file.id)
    const url = window.URL.createObjectURL(new Blob([response.data]))
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', file.name)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
    QMessage.success('文件下载成功')
  } catch (e) {
    QMessage.error('文件下载失败')
  }
}

const handleFileStar = async (file: FileItem) => {
  await toggleFileStar(file.id, !file.is_starred)
}

const handleFileDelete = async (file: FileItem) => {
  if (confirm(`确定要删除文件 "${file.name}" 吗？`)) {
    await deleteFile(file.id)
  }
}

const handleFileUpload = async (event: Event | FileList) => {
  const files = event instanceof Event ? (event.target as HTMLInputElement).files : event
  if (!files || files.length === 0) return

  let successCount = 0
  let failCount = 0

  for (let i = 0; i < files.length; i++) {
    const file = files[i]
    const success = await uploadFile(file)
    if (success) {
      successCount++
    } else {
      failCount++
    }
  }

  if (successCount > 0) {
    QMessage.success(`成功上传 ${successCount} 个文件`)
  }
  if (failCount > 0) {
    QMessage.error(`${failCount} 个文件上传失败`)
  }

  if (fileInputRef.value) {
    fileInputRef.value.value = ''
  }
}

const triggerFileUpload = () => {
  fileInputRef.value?.click()
}

const handleContextMenu = (file: FileItem, event: MouseEvent) => {
  contextMenu.value = {
    visible: true,
    x: event.clientX,
    y: event.clientY,
    file
  }
}

const handleContextMenuAction = (action: string) => {
  const file = contextMenu.value.file
  if (!file) return

  contextMenu.value.visible = false

  switch (action) {
    case 'preview':
      handleFilePreview(file)
      break
    case 'download':
      handleFileDownload(file)
      break
    case 'rename':
    case 'move':
      actionFile.value = file
      showActionsModal.value = true
      break
    case 'star':
      handleFileStar(file)
      break
    case 'share':
      handleFileShare(file)
      break
    case 'delete':
      handleFileDelete(file)
      break
  }
}

const handleFileShare = (file: FileItem) => {
  window.dispatchEvent(new CustomEvent('openShareModal', {
    detail: { type: 'file', data: file }
  }))
}

const handleFileSelect = (fileIds: number[]) => {
  console.log('Selected files:', fileIds)
}

const handleFolderCreated = () => {
  loadRootFolders()
  QMessage.success('文件夹创建成功')
}

const handleActionSuccess = () => {
  refresh()
}

const navigateToRoot = () => {
  changeFolder(null)
}

const navigateToPath = (index: number) => {
  console.log('Navigate to path index:', index)
}

const handleClickOutside = () => {
  if (contextMenu.value.visible) {
    contextMenu.value.visible = false
  }
}

onMounted(async () => {
  await loadFiles()
  await loadRootFolders()
  document.addEventListener('click', handleClickOutside)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.file-management-app {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--card-bg, #fff);
  border-radius: 12px;
  overflow: hidden;
}

/* ===== 筛选工具栏 ===== */
.filter-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 20px;
  background: var(--card-bg, #fff);
  border-bottom: 1px solid var(--border-color, #e8ecf0);
  flex-shrink: 0;
  flex-wrap: wrap;
}

/* 内联搜索 */
.search-icon-inline {
  color: var(--text-secondary, #8c95a6);
  font-size: 12px;
  flex-shrink: 0;
}

.search-input-inline {
  border: none;
  background: transparent;
  color: var(--text-color, #4a5568);
  font-size: 12px;
  outline: none;
  width: 120px;
  min-width: 60px;
  flex-shrink: 1;
}

.search-input-inline::placeholder {
  color: var(--text-secondary, #8c95a6);
}

.search-wrap {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 0 8px;
  height: 28px;
  border: 1px solid var(--border-color, #e8ecf0);
  border-radius: 16px;
  transition: border-color 0.2s ease;
}

.search-wrap:focus-within {
  border-color: var(--primary-color, #4f6ef7);
}

.search-clear-inline {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border: none;
  background: var(--hover-color, #f0f2f5);
  border-radius: 50%;
  cursor: pointer;
  color: var(--text-secondary, #8c95a6);
  font-size: 9px;
  flex-shrink: 0;
  transition: all 0.15s ease;
}

.search-clear-inline:hover {
  background: var(--border-color, #e8ecf0);
  color: var(--text-color, #4a5568);
}

.bar-divider {
  width: 1px;
  height: 20px;
  background: var(--border-color, #e8ecf0);
  flex-shrink: 0;
}

.file-count {
  font-size: 12px;
  color: var(--text-secondary, #8c95a6);
  font-weight: 500;
  margin-left: auto;
  white-space: nowrap;
}

/* 统一下拉选择器 */
.filter-select-wrap {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 0 8px;
  height: 28px;
  border: 1px solid var(--border-color, #e8ecf0);
  border-radius: 16px;
  background: var(--card-bg, #fff);
  position: relative;
  transition: all 0.2s ease;
}

.filter-select-wrap:hover {
  border-color: var(--primary-color, #4f6ef7);
}

.filter-icon {
  color: var(--text-secondary, #8c95a6);
  font-size: 11px;
  flex-shrink: 0;
}

.filter-select {
  border: none;
  background: transparent;
  color: var(--text-color, #4a5568);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  outline: none;
  appearance: none;
  padding-right: 10px;
}

.select-arrow {
  color: var(--text-secondary, #8c95a6);
  font-size: 9px;
  position: absolute;
  right: 10px;
  pointer-events: none;
}

/* 视图切换 */
.view-toggle {
  display: flex;
  gap: 2px;
  background: var(--hover-color, #f0f2f5);
  padding: 2px;
  border-radius: 8px;
}

.toggle-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  color: var(--text-secondary, #8c95a6);
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  transition: all 0.15s ease;
}

.toggle-btn:hover {
  color: var(--text-color, #4a5568);
}

.toggle-btn.active {
  background: var(--card-bg, #fff);
  color: var(--primary-color, #4f6ef7);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

/* ===== 主内容区域 ===== */
.app-content {
  flex: 1;
  overflow: hidden;
  background: var(--card-bg, #fff);
}

/* ===== 右键菜单 ===== */
.context-menu {
  position: fixed;
  background: var(--card-bg, #fff);
  border: 1px solid var(--border-color, #e8ecf0);
  border-radius: 12px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12), 0 2px 8px rgba(0, 0, 0, 0.06);
  padding: 6px;
  min-width: 180px;
  z-index: 10000;
  animation: menuIn 0.15s ease;
}

.context-menu-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  cursor: pointer;
  color: var(--text-color, #4a5568);
  font-size: 14px;
  border-radius: 8px;
  transition: all 0.15s ease;
}

.context-menu-item:hover {
  background: var(--hover-color, #f0f2f5);
  color: var(--primary-color, #4f6ef7);
}

.context-menu-item.danger {
  color: var(--error-color, #e53e3e);
}

.context-menu-item.danger:hover {
  background: rgba(229, 62, 62, 0.06);
  color: var(--error-color, #e53e3e);
}

.context-menu-item i {
  width: 16px;
  text-align: center;
  font-size: 14px;
}

.context-menu-divider {
  height: 1px;
  background: var(--border-color, #e8ecf0);
  margin: 4px 8px;
}

@keyframes menuIn {
  from {
    opacity: 0;
    transform: scale(0.95) translateY(-4px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

/* ===== 响应式 ===== */
@media (max-width: 1024px) {
  .filter-bar {
    padding: 8px 16px;
    gap: 8px;
  }

  .search-input-inline {
    width: 100px;
  }

  .file-count {
    display: none;
  }
}

@media (max-width: 768px) {
  .filter-bar {
    padding: 8px 12px;
    gap: 6px;
  }

  .search-input-inline {
    width: 80px;
  }

  .filter-select-wrap {
    padding: 3px 8px;
  }

  .filter-select {
    font-size: 11px;
    padding-right: 10px;
  }

  .filter-icon {
    font-size: 10px;
  }

  .select-arrow {
    display: none;
  }

  .view-toggle {
    padding: 1px;
  }

  .toggle-btn {
    width: 24px;
    height: 24px;
    font-size: 10px;
  }
}

@media (max-width: 480px) {
  .filter-bar {
    flex-wrap: wrap;
    gap: 6px;
  }

  .bar-divider {
    display: none;
  }

  .search-input-inline {
    width: 80px;
  }

  .filter-select-wrap {
    flex: 0 0 auto;
  }
}
</style>
