<template>
  <div class="file-management-app">
    <!-- 头部 -->
    <div class="app-header">
      <div class="header-left">
        <button class="back-btn" @click="$emit('back')">
          <i class="fas fa-arrow-left"></i>
        </button>
        <div class="header-info">
          <h2>文件管理</h2>
          <div class="breadcrumb" v-if="currentFolderPath">
            <span class="breadcrumb-item" @click="navigateToRoot">
              <i class="fas fa-home"></i>
            </span>
            <template v-for="(segment, index) in currentFolderPath.split('/').filter(s => s)" :key="index">
              <span class="breadcrumb-separator">/</span>
              <span class="breadcrumb-item" @click="navigateToPath(index)">
                {{ segment }}
              </span>
            </template>
          </div>
        </div>
      </div>
      <div class="header-actions">
        <button class="action-btn" @click="showCreateFolderModal = true" title="新建文件夹">
          <i class="fas fa-folder-plus"></i>
        </button>
        <button class="action-btn primary" @click="triggerFileUpload" title="上传文件">
          <i class="fas fa-upload"></i>
          <span class="btn-text">上传</span>
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

    <!-- 主内容区域 -->
    <div class="app-content">
      <!-- 左侧文件夹树 -->
      <div class="sidebar">
        <FolderTree
          ref="folderTreeRef"
          :selected-source="sourceFilter"
          @select="handleFolderSelect"
          @source-change="handleSourceChange"
        />
      </div>

      <!-- 中间文件列表 -->
      <div class="main-content">
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

// 定义事件
const emit = defineEmits(['back'])

// 使用 composables
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

// 组件引用
const folderTreeRef = ref<InstanceType<typeof FolderTree> | null>(null)
const fileListRef = ref<InstanceType<typeof FileList> | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)

// 模态框状态
const showCreateFolderModal = ref(false)
const showPreviewModal = ref(false)
const showActionsModal = ref(false)
const previewFile = ref<FileItem | null>(null)
const actionFile = ref<FileItem | null>(null)

// 右键菜单状态
const contextMenu = ref({
  visible: false,
  x: 0,
  y: 0,
  file: null as FileItem | null
})

// 文件夹列表（用于模态框）
const folders = ref<FolderItem[]>([])

// 当前文件夹路径
const currentFolderPath = computed(() => {
  if (!selectedFolder.value) return ''
  return selectedFolder.value.path || selectedFolder.value.name
})

// 文件夹选择
const handleFolderSelect = (folder: FolderNode) => {
  changeFolder(folder.id)
}

// 搜索
const handleSearch = (query: string) => {
  searchQuery.value = query
}

// 过滤类型变化
const handleFilterChange = (type: string) => {
  changeFilterType(type)
}

// 来源变化
const handleSourceChange = (source: string | null) => {
  changeSource(source)
}

// 文件预览
const handleFilePreview = (file: FileItem) => {
  previewFile.value = file
  showPreviewModal.value = true
}

// 关闭预览模态框
const closePreviewModal = () => {
  showPreviewModal.value = false
  previewFile.value = null
}

// 文件下载
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

// 文件星标
const handleFileStar = async (file: FileItem) => {
  await toggleFileStar(file.id, !file.is_starred)
}

// 文件删除
const handleFileDelete = async (file: FileItem) => {
  if (confirm(`确定要删除文件 "${file.name}" 吗？`)) {
    await deleteFile(file.id)
  }
}

// 文件上传
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

  // 重置文件输入
  if (fileInputRef.value) {
    fileInputRef.value.value = ''
  }
}

// 触发文件上传
const triggerFileUpload = () => {
  fileInputRef.value?.click()
}

// 右键菜单
const handleContextMenu = (file: FileItem, event: MouseEvent) => {
  contextMenu.value = {
    visible: true,
    x: event.clientX,
    y: event.clientY,
    file
  }
}

// 右键菜单操作
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

// 文件分享
const handleFileShare = (file: FileItem) => {
  window.dispatchEvent(new CustomEvent('openShareModal', {
    detail: { type: 'file', data: file }
  }))
}

// 文件选择
const handleFileSelect = (fileIds: number[]) => {
  console.log('Selected files:', fileIds)
}

// 文件夹创建成功
const handleFolderCreated = () => {
  loadRootFolders()
  QMessage.success('文件夹创建成功')
}

// 操作成功
const handleActionSuccess = () => {
  refresh()
}

// 导航到根目录
const navigateToRoot = () => {
  changeFolder(null)
}

// 导航到指定路径
const navigateToPath = (index: number) => {
  // 实现路径导航逻辑
  console.log('Navigate to path index:', index)
}

// 点击外部关闭右键菜单
const handleClickOutside = () => {
  if (contextMenu.value.visible) {
    contextMenu.value.visible = false
  }
}

// 组件挂载
onMounted(async () => {
  await loadFiles()
  await loadRootFolders()
  document.addEventListener('click', handleClickOutside)
})

// 组件卸载
onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.file-management-app {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--content-bg);
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

/* 头部 */
.app-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  background: var(--card-bg);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  height: 72px;
  box-sizing: border-box;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.back-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: var(--hover-color);
  border-radius: 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary-color);
  transition: all 0.2s ease;
}

.back-btn:hover {
  background: var(--primary-light);
}

.header-info h2 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 4px 0;
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-secondary);
}

.breadcrumb-item {
  cursor: pointer;
  transition: color 0.2s ease;
}

.breadcrumb-item:hover {
  color: var(--primary-color);
}

.breadcrumb-separator {
  color: var(--text-tertiary);
}

.header-actions {
  display: flex;
  gap: 8px;
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border: 1px solid var(--border-color);
  background: var(--card-bg);
  border-radius: 8px;
  cursor: pointer;
  color: var(--text-color);
  font-size: 14px;
  transition: all 0.2s ease;
}

.action-btn:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.action-btn.primary {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.action-btn.primary:hover {
  background: var(--primary-hover);
  border-color: var(--primary-hover);
}

.btn-text {
  display: inline;
}

/* 主内容区域 */
.app-content {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.sidebar {
  width: 240px;
  border-right: 1px solid var(--border-color);
  background: var(--card-bg);
  overflow: hidden;
}

.main-content {
  flex: 1;
  overflow: hidden;
}

/* 右键菜单 */
.context-menu {
  position: fixed;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  padding: 8px 0;
  min-width: 160px;
  z-index: 10000;
  animation: fadeIn 0.15s ease;
}

.context-menu-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 16px;
  cursor: pointer;
  color: var(--text-color);
  font-size: 14px;
  transition: all 0.2s ease;
}

.context-menu-item:hover {
  background: var(--hover-color);
  color: var(--primary-color);
}

.context-menu-item.danger {
  color: var(--error-color);
}

.context-menu-item.danger:hover {
  background: var(--color-error-50);
  color: var(--error-color);
}

.context-menu-item i {
  width: 16px;
  text-align: center;
}

.context-menu-divider {
  height: 1px;
  background: var(--border-color);
  margin: 8px 0;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(-4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 响应式 */
@media (max-width: 768px) {
  .app-header {
    padding: 12px 16px;
    height: auto;
    flex-wrap: wrap;
    gap: 12px;
  }

  .header-info h2 {
    font-size: 16px;
  }

  .breadcrumb {
    font-size: 12px;
  }

  .btn-text {
    display: none;
  }

  .app-content {
    flex-direction: column;
  }

  .sidebar {
    width: 100%;
    border-right: none;
    border-bottom: 1px solid var(--border-color);
    max-height: 200px;
  }
}
</style>
