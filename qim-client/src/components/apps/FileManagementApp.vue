<template>
  <div class="file-management-app">
    <!-- 顶部导航栏 -->
    <div class="app-nav">
      <button class="nav-back" @click="$emit('back')">
        <i class="fas fa-chevron-left"></i>
      </button>
      <div class="nav-title">
        <i class="fas fa-folder-open"></i>
        <span>文件箱</span>
      </div>
      <div class="nav-actions">
        <button class="nav-btn" @click="showCreateFolderModal = true" title="新建文件夹">
          <i class="fas fa-folder-plus"></i>
        </button>
        <button class="nav-btn primary" @click="triggerFileUpload" title="上传文件">
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
      </div>
    </div>

    <!-- 筛选工具栏 -->
    <div class="filter-bar">
      <div class="filter-left">
        <!-- 来源筛选 -->
        <div class="filter-chips">
          <button
            :class="['chip', { active: sourceFilter === null }]"
            @click="handleSourceChange(null)"
          >
            <i class="fas fa-layer-group"></i>
            <span>全部</span>
          </button>
          <button
            :class="['chip', { active: sourceFilter === 'upload' }]"
            @click="handleSourceChange('upload')"
          >
            <i class="fas fa-cloud-upload-alt"></i>
            <span>我的上传</span>
          </button>
          <button
            :class="['chip', { active: sourceFilter === 'chat' }]"
            @click="handleSourceChange('chat')"
          >
            <i class="fas fa-comment-dots"></i>
            <span>聊天文件</span>
          </button>
        </div>

        <div class="filter-divider"></div>

        <!-- 文件夹选择 -->
        <div class="folder-picker">
          <i class="fas fa-folder picker-icon"></i>
          <select v-model="selectedFolderId" @change="handleFolderChange" class="picker-select">
            <option :value="null">全部文件</option>
            <option :value="-1">⭐ 星标文件</option>
            <optgroup label="文件夹">
              <option v-for="folder in folders" :key="folder.id" :value="folder.id">
                📁 {{ folder.name }}
              </option>
            </optgroup>
          </select>
          <i class="fas fa-chevron-down picker-arrow"></i>
        </div>
      </div>

      <div class="filter-right">
        <span class="file-count">{{ total }} 个文件</span>
      </div>
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
import { useFilePagination } from '../../composables/useFilePagination'
import { useFolderTree, type FolderNode } from '../../composables/useFolderTree'
import { fileApi, type FileItem, type FolderItem } from '../../api/file'
import QMessage from '../../utils/qmessage'

const emit = defineEmits(['back'])

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
  loadFiles,
  refresh,
  changePage,
  changeFolder,
  changeFilterType,
  toggleStarred,
  changeSource,
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
const selectedFolderId = ref<number | null | -1>(null)

const currentFolderPath = computed(() => {
  if (!selectedFolder.value) return ''
  return selectedFolder.value.path || selectedFolder.value.name
})

const handleFolderSelect = (folder: FolderNode) => {
  changeFolder(folder.id)
}

const handleFolderChange = () => {
  if (selectedFolderId.value === -1) {
    showStarred.value = true
    changeFolder(null)
  } else {
    showStarred.value = false
    changeFolder(selectedFolderId.value)
  }
}

const handleSearch = (query: string) => {
  searchQuery.value = query
}

const handleFilterChange = (type: string) => {
  changeFilterType(type)
}

const handleSourceChange = (source: string | null) => {
  changeSource(source)
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
  background: var(--content-bg, #f5f7fa);
  border-radius: 12px;
  overflow: hidden;
}

/* ===== 顶部导航栏 ===== */
.app-nav {
  display: flex;
  align-items: center;
  padding: 0 20px;
  height: 56px;
  background: var(--card-bg, #fff);
  border-bottom: 1px solid var(--border-color, #e8ecf0);
  gap: 16px;
  flex-shrink: 0;
}

.nav-back {
  width: 36px;
  height: 36px;
  border: none;
  background: transparent;
  border-radius: 10px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary, #8c95a6);
  font-size: 14px;
  transition: all 0.2s ease;
}

.nav-back:hover {
  background: var(--hover-color, #f0f2f5);
  color: var(--primary-color, #4f6ef7);
}

.nav-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 17px;
  font-weight: 600;
  color: var(--text-primary, #1a2233);
}

.nav-title i {
  color: var(--primary-color, #4f6ef7);
  font-size: 18px;
}

.nav-actions {
  margin-left: auto;
  display: flex;
  gap: 8px;
  align-items: center;
}

.nav-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border: 1px solid var(--border-color, #e8ecf0);
  background: var(--card-bg, #fff);
  border-radius: 10px;
  cursor: pointer;
  color: var(--text-color, #4a5568);
  font-size: 13px;
  font-weight: 500;
  transition: all 0.2s ease;
}

.nav-btn:hover {
  background: var(--hover-color, #f0f2f5);
  border-color: var(--primary-color, #4f6ef7);
  color: var(--primary-color, #4f6ef7);
}

.nav-btn.primary {
  background: var(--primary-color, #4f6ef7);
  border-color: var(--primary-color, #4f6ef7);
  color: #fff;
}

.nav-btn.primary:hover {
  background: var(--primary-hover, #3b5de7);
  border-color: var(--primary-hover, #3b5de7);
  box-shadow: 0 2px 8px rgba(79, 110, 247, 0.35);
}

.nav-btn.primary span {
  display: inline;
}

/* ===== 筛选工具栏 ===== */
.filter-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px;
  background: var(--card-bg, #fff);
  border-bottom: 1px solid var(--border-color, #e8ecf0);
  flex-shrink: 0;
}

.filter-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.filter-right {
  display: flex;
  align-items: center;
}

.file-count {
  font-size: 13px;
  color: var(--text-secondary, #8c95a6);
  font-weight: 500;
}

/* 筛选标签 */
.filter-chips {
  display: flex;
  gap: 6px;
}

.chip {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border: 1px solid var(--border-color, #e8ecf0);
  background: var(--card-bg, #fff);
  border-radius: 20px;
  cursor: pointer;
  color: var(--text-secondary, #8c95a6);
  font-size: 13px;
  font-weight: 500;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.chip i {
  font-size: 12px;
}

.chip:hover {
  border-color: var(--primary-color, #4f6ef7);
  color: var(--primary-color, #4f6ef7);
  background: rgba(79, 110, 247, 0.04);
}

.chip.active {
  background: var(--primary-color, #4f6ef7);
  border-color: var(--primary-color, #4f6ef7);
  color: #fff;
  box-shadow: 0 2px 6px rgba(79, 110, 247, 0.3);
}

.chip.active i {
  color: #fff;
}

/* 分隔线 */
.filter-divider {
  width: 1px;
  height: 24px;
  background: var(--border-color, #e8ecf0);
}

/* 文件夹选择器 */
.folder-picker {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  border: 1px solid var(--border-color, #e8ecf0);
  border-radius: 10px;
  background: var(--card-bg, #fff);
  cursor: pointer;
  transition: all 0.2s ease;
  position: relative;
}

.folder-picker:hover {
  border-color: var(--primary-color, #4f6ef7);
  box-shadow: 0 0 0 3px rgba(79, 110, 247, 0.08);
}

.picker-icon {
  color: var(--primary-color, #4f6ef7);
  font-size: 14px;
}

.picker-select {
  border: none;
  background: transparent;
  color: var(--text-color, #4a5568);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  outline: none;
  appearance: none;
  padding-right: 16px;
  min-width: 120px;
}

.picker-arrow {
  color: var(--text-secondary, #8c95a6);
  font-size: 10px;
  position: absolute;
  right: 12px;
  pointer-events: none;
}

/* ===== 主内容区域 ===== */
.app-content {
  flex: 1;
  overflow: hidden;
  background: var(--content-bg, #f5f7fa);
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
@media (max-width: 768px) {
  .app-nav {
    padding: 0 12px;
    height: 48px;
  }

  .nav-title {
    font-size: 15px;
  }

  .nav-btn span {
    display: none;
  }

  .filter-bar {
    padding: 10px 12px;
    flex-direction: column;
    gap: 10px;
    align-items: flex-start;
  }

  .filter-left {
    flex-wrap: wrap;
    gap: 8px;
  }

  .filter-divider {
    display: none;
  }

  .chip {
    padding: 5px 10px;
    font-size: 12px;
  }

  .folder-picker {
    width: 100%;
  }

  .picker-select {
    flex: 1;
    min-width: 0;
  }
}
</style>
