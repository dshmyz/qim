<template>
  <div class="file-management-app">
    <!-- 顶部导航栏 -->
    <AppHeader title="文件箱" icon="fas fa-folder-open" @back="$emit('back')">
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
      <!-- 侧栏折叠按钮（放最左侧，折叠后仍可达） -->
      <button
        class="sidebar-toggle-btn"
        :title="sidebarCollapsed ? '展开文件夹侧栏' : '收起文件夹侧栏'"
        @click="sidebarCollapsed = !sidebarCollapsed"
      >
        <i :class="sidebarCollapsed ? 'fas fa-chevron-right' : 'fas fa-chevron-left'"></i>
      </button>

      <!-- 面包屑（当前文件夹路径，中间段可跳转） -->
      <nav v-if="folderPath.length" class="breadcrumb">
        <button class="breadcrumb-item" @click="handleBreadcrumbClick(null)">全部文件</button>
        <template v-for="(seg, idx) in folderPath" :key="seg.id">
          <i class="fas fa-chevron-right breadcrumb-sep"></i>
          <button
            class="breadcrumb-item"
            :class="{ 'breadcrumb-item--current': idx === folderPath.length - 1 }"
            @click="handleBreadcrumbClick(seg.id)"
          >
            {{ seg.name }}
          </button>
        </template>
      </nav>

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

      <!-- 星标过滤（跨文件夹全局视图） -->
      <button
        class="starred-toggle-btn"
        :class="{ 'starred-toggle-btn--active': showStarred }"
        :title="showStarred ? '取消只看星标' : '只看星标'"
        @click="handleStarredToggle"
      >
        <i class="fas fa-star"></i>
      </button>

      <div class="bar-divider"></div>

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
      <!-- 文件夹树侧栏（折叠按钮在筛选栏最左侧，折叠后仍可达） -->
      <div class="folder-tree-panel" :class="{ 'folder-tree-panel--collapsed': sidebarCollapsed }">
        <FolderTree
          ref="folderTreeRef"
          :selected-source="sourceFilter"
          @select="handleTreeSelect"
          @source-change="handleTreeSourceChange"
          @deleted="handleTreeDeleted"
        />
        <!-- 左侧底部：存储容量条 -->
        <StorageUsageBar ref="storageUsageRef" />
      </div>

      <div class="file-list-panel">
        <FileList
          ref="fileListRef"
          :files="files"
          :total="total"
          :loading="isLoading"
          :has-more="hasMore"
          :view-mode="viewMode"
          @load-more="loadMore"
          @preview="handleFilePreview"
          @download="handleFileDownload"
          @star="handleFileStar"
          @share="handleFileShare"
          @delete="handleFileDelete"
          @context-menu="handleContextMenu"
          @selection-change="handleFileSelect"
        />
      </div>
    </div>

    <!-- 批量操作条 -->
    <transition name="batch-fade">
      <div v-if="selectedFileIds.size > 0" class="batch-bar">
        <span class="batch-count">已选 {{ selectedFileIds.size }} 项</span>
        <div class="batch-divider"></div>
        <button class="batch-btn" title="顺序下载选中的文件" @click="handleBatchDownload">
          <i class="fas fa-download"></i> 下载
        </button>
        <button class="batch-btn" @click="openBatchMoveModal">
          <i class="fas fa-arrows-alt"></i> 移动到
        </button>
        <button class="batch-btn" @click="handleBatchStar">
          <i class="fas fa-star"></i> {{ selectedAllStarred ? '取消星标' : '添加星标' }}
        </button>
        <button class="batch-btn batch-btn--danger" @click="handleBatchDelete">
          <i class="fas fa-trash"></i> 删除
        </button>
        <div class="batch-divider"></div>
        <button class="batch-btn" @click="handleClearSelection">
          <i class="fas fa-times"></i> 取消选择
        </button>
      </div>
    </transition>

    <!-- 批量移动目标选择 -->
    <ModalContainer
      :visible="showBatchMoveModal"
      title="移动文件"
      width="480px"
      @close="closeBatchMoveModal"
      @cancel="closeBatchMoveModal"
    >
      <div class="batch-move-form-group">
        <label for="batch-move-folder-select">目标文件夹</label>
        <FolderTreeSelect
          id="batch-move-folder-select"
          class="batch-move-select"
          v-model="batchMoveFolderId"
          :options="folderOptions"
          :exclude-ids="currentFolderId ? [currentFolderId] : []"
        />
        <p class="batch-move-count">将移动选中 {{ selectedFileIds.size }} 个文件</p>
      </div>
      <template #footer>
        <button class="batch-modal-btn batch-modal-btn--cancel" @click="closeBatchMoveModal">取消</button>
        <button class="batch-modal-btn batch-modal-btn--primary" @click="handleBatchMoveConfirm">移动</button>
      </template>
    </ModalContainer>

    <!-- 创建文件夹模态框（新建默认落在当前文件夹，可在树选择器中更改） -->
    <CreateFolderModal
      :visible="showCreateFolderModal"
      :options="folderOptions"
      :initial-parent-id="currentFolderId"
      @close="showCreateFolderModal = false"
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
      :options="folderOptions"
      :initial-tab="actionTab"
      @close="showActionsModal = false"
      @success="handleActionSuccess"
    />

    <!-- 右键菜单 -->
    <UniversalContextMenu menuId="file-management" :items="contextMenuItems" />

    <!-- 上传进度条 -->
    <UploadProgressBar :visible="true" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onBeforeUnmount, defineAsyncComponent } from 'vue'
import FileList from './file/FileList.vue'
import CreateFolderModal from './file/CreateFolderModal.vue'
import FolderTree from './file/FolderTree.vue'
import FolderTreeSelect from './file/FolderTreeSelect.vue'
import StorageUsageBar from './file/StorageUsageBar.vue'
import FileDateFilter from './file/FileDateFilter.vue'
import AppHeader from './AppHeader.vue'
import UploadProgressBar from '../common/UploadProgressBar.vue'
import FilePreviewModal from './file/FilePreviewModal.vue'
// 大组件懒加载
const FileActionsModal = defineAsyncComponent(() => import('./file/FileActionsModal.vue'))
import { useFilePagination } from '../../composables/useFilePagination'
import { useFolderTree, type FolderNode } from '../../composables/useFolderTree'
import UniversalContextMenu from '../shared/UniversalContextMenu.vue'
import ModalContainer from '../shared/ModalContainer.vue'
import type { ContextMenuItem } from '../shared/context-menu-types'
import { useFileUpload, uploadFilesWithLimit } from '../../composables/useFileUpload'
import { useFileDownload } from '../../composables/useFileDownload'
import { useUploadStore } from '../../stores/upload'
import { type FileItem } from '../../api/file'
import QMessage from '../../utils/qmessage'
import QMessageBox from '../../utils/qmessagebox'
import { openMenu, closeMenu } from '../../composables/useUI'

const emit = defineEmits(['back'])

const {
  files,
  total,
  isLoading,
  searchQuery,
  dateFrom,
  dateTo,
  currentFolderId,
  sourceFilter,
  showStarred,
  loadFiles,
  refresh,
  hasMore,
  loadMore,
  changeFolder,
  changeSource,
  changeSort,
  changeDateRange,
  clearDateRange,
  deleteFile,
  toggleFileStar,
  toggleStarred,
  batchDelete,
  batchMove,
  batchToggleStar
} = useFilePagination()

const {
  folderOptions,
  folderPath,
  loadRootFolders,
  fetchAllFolders,
  selectRoot,
  findFolderInTree
} = useFolderTree()

const uploadStore = useUploadStore()
const { tasks } = useFileUpload()
const { downloadFile } = useFileDownload()

const fileListRef = ref<InstanceType<typeof FileList> | null>(null)
const folderTreeRef = ref<InstanceType<typeof FolderTree> | null>(null)
const storageUsageRef = ref<InstanceType<typeof StorageUsageBar> | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)

const sidebarCollapsed = ref(false)

const showCreateFolderModal = ref(false)
const showPreviewModal = ref(false)
const showActionsModal = ref(false)
const previewFile = ref<FileItem | null>(null)
const actionFile = ref<FileItem | null>(null)
const actionTab = ref<'rename' | 'move'>('rename')

// 批量选择与批量操作
const selectedFileIds = ref<Set<number>>(new Set())
const showBatchMoveModal = ref(false)
const batchMoveFolderId = ref<number | null>(null)

const contextMenu = ref({
  file: null as FileItem | null
})

// folders 由侧栏 FolderTree + useFolderTree 单例树接管（本文件不再维护下拉选项）
const viewMode = ref<'grid' | 'list'>('list')

// 树联动：点击文件夹/全部文件 → 切换文件列表
const handleTreeSelect = (folder: FolderNode | null) => {
  changeFolder(folder?.id ?? null)
  fileListRef.value?.clearSelection()
}

// 树来源 tabs：与文件夹选择正交
const handleTreeSourceChange = (source: string | null) => {
  changeSource(source)
}

// 树内删除：当前所在文件夹被删或所在祖先被删（节点已不在树中）→ 回根
const handleTreeDeleted = (folder: FolderNode) => {
  if (currentFolderId.value !== null && !findFolderInTree(currentFolderId.value)) {
    selectRoot()
    changeFolder(null)
  }
  // 递归删除可能连带删除文件，刷新容量条
  refreshStorageUsage()
}

// 星标视图：跨文件夹全局过滤，切换时回到「全部文件」，避免与文件夹选择矛盾
const handleStarredToggle = async () => {
  if (currentFolderId.value !== null) {
    selectRoot()
    await changeFolder(null)
  }
  await toggleStarred()
}

// 刷新侧栏容量条（上传/删除等文件变更后）
const refreshStorageUsage = () => {
  storageUsageRef.value?.reload()
}

// 面包屑跳转（全部文件或中间段）
const handleBreadcrumbClick = (folderId: number | null) => {
  folderTreeRef.value?.navigateTo(folderId)
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
    await downloadFile(file)
  } catch (e) {
    QMessage.error('文件下载失败')
  }
}

const handleFileStar = async (file: FileItem) => {
  await toggleFileStar(file.id, !file.is_starred)
}

const handleFileDelete = async (file: FileItem) => {
  const result = await QMessageBox.confirm(
    `确定要删除文件 "${file.name}" 吗？`,
    '删除文件',
    { confirmButtonText: '删除', type: 'warning' }
  )
  if (result.action !== 'confirm') return
  const ok = await deleteFile(file.id)
  if (ok) refreshStorageUsage()
}

const handleFileUpload = async (event: Event | FileList) => {
  const files = event instanceof Event ? (event.target as HTMLInputElement).files : event
  if (!files || files.length === 0) return

  // 使用并发限制上传（最多同时 3 个文件），避免浏览器并发连接数限制和内存压力
  await uploadFilesWithLimit(files, currentFolderId.value ?? undefined)

  // 刷新文件列表与容量条
  await refresh()
  refreshStorageUsage()

  // 清空文件输入
  if (fileInputRef.value) {
    fileInputRef.value.value = ''
  }
}

const triggerFileUpload = () => {
  fileInputRef.value?.click()
}

const handleContextMenu = (file: FileItem, event: MouseEvent) => {
  event.preventDefault()
  contextMenu.value = { file }
  openMenu('file-management', event.clientX, event.clientY)
}

const handleContextMenuAction = (action: string) => {
  const file = contextMenu.value.file
  if (!file) return

  closeMenu()

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
      actionTab.value = action === 'move' ? 'move' : 'rename'
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

const handleFileSelect = (fileIds: Set<number>) => {
  selectedFileIds.value = fileIds
}

// 当前选中的文件（用于批量操作与星标归类）
const selectedFiles = computed(() => files.value.filter(f => selectedFileIds.value.has(f.id)))
// 选中文件是否已全部星标（决定批量按钮文案）
const selectedAllStarred = computed(
  () => selectedFiles.value.length > 0 && selectedFiles.value.every(f => f.is_starred)
)

const clearSelectionAndRefresh = () => {
  fileListRef.value?.clearSelection()
  refresh()
}

const handleBatchDownload = async () => {
  const sel = selectedFiles.value
  if (sel.length === 0) return
  for (const file of sel) {
    try {
      await downloadFile(file)
    } catch {
      // 单个文件下载失败不阻塞其余
    }
  }
}

const openBatchMoveModal = () => {
  batchMoveFolderId.value = null
  showBatchMoveModal.value = true
}

const closeBatchMoveModal = () => {
  showBatchMoveModal.value = false
}

const handleBatchMoveConfirm = async () => {
  const ids = [...selectedFileIds.value]
  if (ids.length === 0) return
  const ok = await batchMove(ids, batchMoveFolderId.value)
  if (ok) {
    closeBatchMoveModal()
    clearSelectionAndRefresh()
  }
}

const handleBatchStar = async () => {
  const ids = [...selectedFileIds.value]
  if (ids.length === 0) return
  const ok = await batchToggleStar(ids, !selectedAllStarred.value)
  if (ok) clearSelectionAndRefresh()
}

const handleBatchDelete = async () => {
  const ids = [...selectedFileIds.value]
  if (ids.length === 0) return
  const result = await QMessageBox.confirm(
    `确定要删除选中的 ${ids.length} 个文件吗？`,
    '批量删除',
    { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
  )
  if (result.action !== 'confirm') return
  const ok = await batchDelete(ids)
  if (ok) {
    clearSelectionAndRefresh()
    refreshStorageUsage()
  }
}

const handleClearSelection = () => {
  fileListRef.value?.clearSelection()
}

const handleActionSuccess = () => {
  refresh()
}

// 滚动或窗口缩放时关闭菜单，避免菜单飘在原地与内容错位
const contextMenuItems = computed<ContextMenuItem[]>(() => [
  { label: '预览', icon: 'fas fa-eye', action: () => handleContextMenuAction('preview') },
  { label: '下载', icon: 'fas fa-download', action: () => handleContextMenuAction('download') },
  { label: '重命名', icon: 'fas fa-edit', action: () => handleContextMenuAction('rename') },
  { label: '移动', icon: 'fas fa-arrows-alt', action: () => handleContextMenuAction('move') },
  { divider: true },
  { label: contextMenu.value.file?.is_starred ? '取消星标' : '添加星标', icon: contextMenu.value.file?.is_starred ? 'fas fa-star' : 'far fa-star', action: () => handleContextMenuAction('star') },
  { label: '分享', icon: 'fas fa-share-alt', action: () => handleContextMenuAction('share') },
  { divider: true },
  { label: '删除', icon: 'fas fa-trash', danger: true, action: () => handleContextMenuAction('delete') }
])

const handleContextMenuDismiss = () => {
  closeMenu()
}

onMounted(async () => {
  await loadFiles()
  await loadRootFolders()
  // 全量拉取文件夹树，保证树选择器/面包屑拥有完整深层选项
  await fetchAllFolders()
  // capture: true 以便捕获子元素（文件列表、文件夹树）内部的滚动
  window.addEventListener('scroll', handleContextMenuDismiss, true)
  window.addEventListener('resize', handleContextMenuDismiss)
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', handleContextMenuDismiss, true)
  window.removeEventListener('resize', handleContextMenuDismiss)
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
  position: relative;
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

/* 面包屑 */
.breadcrumb {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
}

.breadcrumb-item {
  border: none;
  background: transparent;
  padding: 2px 4px;
  border-radius: 4px;
  font-size: var(--font-size-xxs);
  color: var(--text-secondary, #8c95a6);
  cursor: pointer;
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  flex-shrink: 1;
  transition: all 0.15s ease;
}

.breadcrumb-item:hover {
  color: var(--primary-color, #4f6ef7);
  background: var(--hover-color, #f0f2f5);
}

.breadcrumb-item--current {
  color: var(--text-color, #4a5568);
  font-weight: 600;
  cursor: default;
}

.breadcrumb-item--current:hover {
  color: var(--text-color, #4a5568);
  background: transparent;
}

.breadcrumb-sep {
  color: var(--text-secondary, #8c95a6);
  font-size: 9px;
  flex-shrink: 0;
}

/* 内联搜索 */
.search-icon-inline {
  color: var(--text-secondary, #8c95a6);
  font-size: var(--font-size-xxs);
  flex-shrink: 0;
}

.search-input-inline {
  border: none;
  background: transparent;
  color: var(--text-color, #4a5568);
  font-size: var(--font-size-xxs);
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

/* 星标过滤按钮 */
.starred-toggle-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 1px solid var(--border-color, #e8ecf0);
  border-radius: var(--radius-sm);
  background: var(--card-bg, #fff);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-base);
  flex-shrink: 0;
}

.starred-toggle-btn:hover {
  color: var(--color-warning-500);
  border-color: var(--color-warning-500);
}

.starred-toggle-btn--active {
  color: var(--color-warning-500);
  border-color: var(--color-warning-500);
  background: rgba(255, 193, 7, 0.12);
}

.file-count {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary, #8c95a6);
  font-weight: 500;
  margin-left: auto;
  white-space: nowrap;
}

/* ===== 批量操作条 ===== */
.batch-bar {
  position: absolute;
  left: 50%;
  bottom: 16px;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: var(--text-color, #4a5568);
  border-radius: 12px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.18);
  z-index: 30;
  color: #fff;
}

.batch-count {
  font-size: var(--font-size-xs);
  font-weight: 500;
  color: #fff;
  white-space: nowrap;
}

.batch-divider {
  width: 1px;
  height: 16px;
  background: rgba(255, 255, 255, 0.22);
  flex-shrink: 0;
}

.batch-btn {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 6px 10px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: #fff;
  font-size: var(--font-size-xs);
  cursor: pointer;
  white-space: nowrap;
  transition: background 0.15s ease, color 0.15s ease;
}

.batch-btn:hover {
  background: rgba(255, 255, 255, 0.14);
}

.batch-btn--danger:hover {
  background: rgba(229, 62, 62, 0.75);
  color: #fff;
}

.batch-fade-enter-active,
.batch-fade-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}

.batch-fade-enter-from,
.batch-fade-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(6px);
}

/* ===== 批量移动弹窗 ===== */
.batch-move-form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.batch-move-form-group label {
  font-size: var(--font-size-sm);
  font-weight: 500;
  color: var(--text-color, #4a5568);
}

.batch-move-select {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-color, #e8ecf0);
  border-radius: 8px;
  font-size: var(--font-size-sm);
  background: var(--input-bg, #fff);
  color: var(--text-color, #4a5568);
  outline: none;
  box-sizing: border-box;
  cursor: pointer;
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%23999' d='M6 8L1 3h10z'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;
  padding-right: 36px;
}

.batch-move-select:focus {
  border-color: var(--primary-color, #4f6ef7);
}

.batch-move-count {
  margin: 0;
  font-size: var(--font-size-xs);
  color: var(--text-secondary, #8c95a6);
}

.batch-modal-btn {
  padding: 8px 24px;
  border-radius: 8px;
  font-size: var(--font-size-sm);
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  border: 1px solid transparent;
}

.batch-modal-btn--cancel {
  background: var(--card-bg, #fff);
  color: var(--text-color, #4a5568);
  border-color: var(--border-color, #e8ecf0);
}

.batch-modal-btn--cancel:hover {
  border-color: var(--primary-color, #4f6ef7);
  color: var(--primary-color, #4f6ef7);
}

.batch-modal-btn--primary {
  background: var(--primary-color, #4f6ef7);
  color: #fff;
  border-color: var(--primary-color, #4f6ef7);
}

.batch-modal-btn--primary:hover {
  background: var(--primary-hover, #3d5ce0);
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
  font-size: var(--font-size-xxxs);
  flex-shrink: 0;
}

.filter-select {
  border: none;
  background: transparent;
  color: var(--text-color, #4a5568);
  font-size: var(--font-size-xxs);
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
  font-size: var(--font-size-xxs);
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
  display: flex;
  align-items: stretch;
  position: relative;
  overflow: hidden;
  background: var(--card-bg, #fff);
}

/* 侧栏折叠按钮（筛选栏最左侧，折叠后仍可达） */
.sidebar-toggle-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: none;
  background: transparent;
  border-radius: 6px;
  cursor: pointer;
  color: var(--text-secondary, #8c95a6);
  font-size: var(--font-size-xxs);
  flex-shrink: 0;
  transition: all 0.15s ease;
}

.sidebar-toggle-btn:hover {
  background: var(--hover-color, #f0f2f5);
  color: var(--primary-color, #4f6ef7);
}

/* 文件夹树侧栏（可折叠，width 过渡） */
.folder-tree-panel {
  width: 240px;
  flex-shrink: 0;
  overflow: hidden;
  border-right: 1px solid var(--border-color, #e8ecf0);
  background: var(--card-bg, #fff);
  transition: width 0.2s ease;
  display: flex;
  flex-direction: column;
}

.folder-tree-panel--collapsed {
  width: 0;
  border-right: none;
}

.folder-tree-panel :deep(.folder-tree) {
  flex: 1;
  min-height: 0;
  height: auto;
}

/* 文件列表面板 */
.file-list-panel {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
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
    font-size: var(--font-size-xxxs);
    padding-right: 10px;
  }

  .filter-icon {
    font-size: var(--font-size-tiny);
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
    font-size: var(--font-size-tiny);
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
