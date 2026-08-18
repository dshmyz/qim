<template>
  <div class="folder-tree">
    <!-- 头部区域 -->
    <div class="folder-tree__header">
      <h3 class="folder-tree__title">
        <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
          <path d="M10 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z" />
        </svg>
        文件夹
      </h3>
      <div class="folder-tree__actions">
        <button
          class="folder-tree__action-btn"
          title="新建文件夹"
          @click="openCreateDialog()"
        >
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
          </svg>
        </button>
        <button
          class="folder-tree__action-btn"
          title="刷新"
          @click="refresh"
        >
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="23 4 23 10 17 10" />
            <path d="M20.49 15a9 9 0 11-2.12-9.36L23 10" />
          </svg>
        </button>
        <button
          class="folder-tree__action-btn"
          :title="allExpanded ? '收起全部' : '展开全部'"
          @click="toggleExpandAll"
        >
          <svg v-if="allExpanded" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="4 14 10 14 10 20" />
            <polyline points="20 10 14 10 14 4" />
            <line x1="14" y1="10" x2="21" y2="3" />
            <line x1="3" y1="21" x2="10" y2="14" />
          </svg>
          <svg v-else viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="15 3 21 3 21 9" />
            <polyline points="9 21 3 21 3 15" />
            <line x1="21" y1="3" x2="14" y2="10" />
            <line x1="3" y1="21" x2="10" y2="14" />
          </svg>
        </button>
      </div>
    </div>

    <!-- 来源筛选标签 -->
    <div class="source-filter">
      <button
        :class="['source-tab', { active: props.selectedSource === null }]"
        @click="handleSourceChange(null)"
      >
        全部
      </button>
      <button
        :class="['source-tab', { active: props.selectedSource === 'upload' }]"
        @click="handleSourceChange('upload')"
      >
        上传
      </button>
      <button
        :class="['source-tab', { active: props.selectedSource === 'chat' }]"
        @click="handleSourceChange('chat')"
      >
        聊天
      </button>
    </div>

    <!-- 搜索框（常驻：小树也能搜，避免跨阈值时布局跳动） -->
    <div class="folder-tree__search">
      <input
        v-model="searchQuery"
        class="folder-tree__search-input"
        placeholder="搜索文件夹..."
        type="text"
      />
      <svg v-if="searchQuery" class="folder-tree__search-clear" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" @click="searchQuery = ''">
        <line x1="18" y1="6" x2="6" y2="18" />
        <line x1="6" y1="6" x2="18" y2="18" />
      </svg>
    </div>

    <!-- 加载状态 -->
    <div v-if="isLoading" class="folder-tree__loading">
      <div class="folder-tree__spinner" />
      <span>加载中...</span>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="loadFailed" class="folder-tree__error">
      <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="10" />
        <line x1="12" y1="8" x2="12" y2="12" />
        <line x1="12" y1="16" x2="12.01" y2="16" />
      </svg>
      <p>{{ error }}</p>
      <button class="folder-tree__retry-btn" @click="refresh">重试</button>
    </div>

    <!-- 列表区域（根伪节点「全部文件」+ 文件夹树） -->
    <div v-else class="folder-tree__list">
      <!-- 根伪节点：未选中任何文件夹 = 根（初始态即根） -->
      <div
        class="folder-root-row"
        :class="{ 'folder-root-row--active': !selectedFolder }"
        @click="handleSelectRoot"
      >
        <span class="folder-root-indent" />
        <span class="folder-root-icon">
          <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
            <path d="M3 13h8V3H3v10zm0 8h8v-6H3v6zm10 0h8V11h-8v10zm0-18v6h8V3h-8z" />
          </svg>
        </span>
        <span class="folder-root-name">全部文件</span>
      </div>

      <FolderTreeItem
        v-for="folder in filteredFolders"
        :key="folder.id"
        :folder="folder"
        :expanded-ids="expandedIds"
        :selected-id="selectedFolder?.id ?? null"
        :is-expandable-fn="isExpandable"
        :loading-ids="loadingChildrenIds"
        @toggle="handleToggle"
        @select="handleSelect"
        @rename="openRenameDialog"
        @delete="handleDelete"
        @contextmenu="handleRowContextMenu"
      />

      <!-- 空状态 -->
      <div v-if="filteredFolders.length === 0" class="folder-tree__empty">
        <svg viewBox="0 0 24 24" width="40" height="40" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z" />
        </svg>
        <p>{{ searchQuery ? '未找到匹配的文件夹' : '暂无文件夹' }}</p>
        <button class="folder-tree__create-btn" @click="openCreateDialog()" v-if="!searchQuery">
          新建文件夹
        </button>
      </div>
    </div>

    <!-- 底部统计信息 -->
    <div class="folder-tree__footer" v-if="!isLoading && !error">
      <span>共 {{ totalFolders }} 个文件夹</span>
    </div>

    <!-- 树行右键菜单 -->
    <div
      v-if="contextMenuVisible && contextMenu"
      class="folder-context-menu"
      :style="contextMenuStyle"
      @click.stop
    >
      <button class="folder-context-menu__item" @click="contextAction('create')">
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z" />
        </svg>
        在此新建子文件夹
      </button>
      <button class="folder-context-menu__item" @click="contextAction('toggle')">
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
          <polyline v-if="contextMenuIsExpanded" points="18 15 12 9 6 15" />
          <polyline v-else points="6 9 12 15 18 9" />
        </svg>
        {{ contextMenuIsExpanded ? '收起' : '展开' }}
      </button>
      <div class="folder-context-menu__divider" />
      <button class="folder-context-menu__item" @click="contextAction('rename')">
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7" />
          <path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z" />
        </svg>
        重命名
      </button>
      <button class="folder-context-menu__item folder-context-menu__item--danger" @click="contextAction('delete')">
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="3 6 5 6 21 6" />
          <path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2" />
        </svg>
        删除
      </button>
    </div>

    <!-- 新建/重命名文件夹对话框 -->
    <CreateFolderModal
      :visible="createDialogMode !== null"
      :is-editing="createDialogMode === 'rename'"
      :folder="renameTarget"
      :fixed-parent-id="createDialogMode === 'create' ? (createTargetFolder?.id ?? selectedFolder?.id ?? null) : undefined"
      @close="createDialogMode = null"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import QMessage from '../../../utils/qmessage'
import QMessageBox from '../../../utils/qmessagebox'
import FolderTreeItem from './FolderTreeItem.vue'
import CreateFolderModal from './CreateFolderModal.vue'
import { useFolderTree, type FolderNode } from '../../../composables/useFolderTree'

interface Props {
  selectedSource?: string | null
}

const props = withDefaults(defineProps<Props>(), {
  selectedSource: null
})

const emit = defineEmits<{
  (e: 'select', folder: FolderNode | null): void
  (e: 'sourceChange', source: string | null): void
  (e: 'deleted', folder: FolderNode): void
}>()

const {
  treeData,
  expandedIds,
  selectedFolder,
  isLoading,
  loadFailed,
  error,
  totalFolders,
  loadRootFolders,
  toggleExpand,
  selectFolder,
  selectRoot,
  deleteFolder,
  loadChildren,
  updateFolderInTree,
  findFolderInTree,
  expandAll,
  collapseAll,
  isExpandable
} = useFolderTree()

// 搜索
const searchQuery = ref('')

// 处理来源变化
function handleSourceChange(source: string | null) {
  emit('sourceChange', source)
}

// 新建/重命名对话框状态（复用 CreateFolderModal）
const createDialogMode = ref<'create' | 'rename' | null>(null)
const renameTarget = ref<FolderNode | null>(null)
/** 新建对话框的父级：右键「在此新建子文件夹」优先于当前选中节点 */
const createTargetFolder = ref<FolderNode | null>(null)

// 树行右键菜单状态
const contextMenu = ref<{ folder: FolderNode; x: number; y: number } | null>(null)
const contextMenuVisible = ref(false)

const contextMenuIsExpanded = computed(() => {
  const folder = contextMenu.value?.folder
  return !!folder && expandedIds.value.has(folder.id)
})

const contextMenuStyle = computed(() => {
  const m = contextMenu.value
  if (!m) return {}
  // 防溢出：菜单宽约 180px、高约 160px
  return {
    left: `${Math.min(m.x, Math.max(0, window.innerWidth - 190))}px`,
    top: `${Math.min(m.y, Math.max(0, window.innerHeight - 170))}px`
  }
})

const handleRowContextMenu = (folder: FolderNode, event: MouseEvent) => {
  contextMenu.value = { folder, x: event.clientX, y: event.clientY }
  contextMenuVisible.value = true
}

const closeContextMenu = () => {
  contextMenuVisible.value = false
  contextMenu.value = null
}

const contextAction = (action: 'create' | 'toggle' | 'rename' | 'delete') => {
  const folder = contextMenu.value?.folder
  if (!folder) return
  closeContextMenu()
  switch (action) {
    case 'create':
      openCreateDialog(folder)
      break
    case 'toggle':
      handleToggle(folder)
      break
    case 'rename':
      openRenameDialog(folder)
      break
    case 'delete':
      handleDelete(folder)
      break
  }
}

// 子节点加载状态追踪
const loadingChildrenIds = ref<Set<number>>(new Set())

// 判断是否全部展开
const allExpanded = computed(() => {
  if (treeData.value.length === 0) return false
  const checkAll = (nodes: FolderNode[]): boolean => {
    for (const node of nodes) {
      if (isExpandable(node) && !expandedIds.value.has(node.id)) return false
      if (node.children?.length && !checkAll(node.children)) return false
    }
    return true
  }
  return checkAll(treeData.value)
})

// 搜索过滤
const filteredFolders = computed(() => {
  if (!searchQuery.value) return treeData.value

  const query = searchQuery.value.toLowerCase()
  const filterNodes = (nodes: FolderNode[]): FolderNode[] => {
    return nodes.reduce<FolderNode[]>((acc, node) => {
      const matches = node.name.toLowerCase().includes(query)
      const filteredChildren = node.children ? filterNodes(node.children) : []

      if (matches || filteredChildren.length > 0) {
        const filteredNode = { ...node }
        if (filteredChildren.length > 0) {
          filteredNode.children = filteredChildren
        }
        acc.push(filteredNode)
      }
      return acc
    }, [])
  }

  return filterNodes(treeData.value)
})

// 刷新
const refresh = async () => {
  await loadRootFolders()
}

// 展开/切换
const handleToggle = async (folder: FolderNode) => {
  loadingChildrenIds.value.add(folder.id)
  await toggleExpand(folder)
  loadingChildrenIds.value.delete(folder.id)
}

// 选择根目录（全部文件）
const handleSelectRoot = () => {
  selectRoot()
  emit('select', null)
}

// 选择文件夹
const handleSelect = (folder: FolderNode) => {
  selectFolder(folder)
  emit('select', folder)
}

// 删除文件夹：先一次确认；后端提示非空（含子文件夹/文件）时二次确认递归删除
const handleDelete = async (folder: FolderNode) => {
  const ok = await QMessageBox.confirm(`确定删除文件夹「${folder.name}」？`, '删除文件夹', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消'
  })
  if (ok.action !== 'confirm') return

  const success = await deleteFolder(folder.id, false)
  if (success) {
    QMessage.success('文件夹已删除')
    emit('deleted', folder)
    return
  }

  // 后端 400 文案：「文件夹包含子文件夹…」/「文件夹包含文件…」
  if (error.value && /子文件夹|包含文件/.test(error.value)) {
    const recursiveOk = await QMessageBox.confirm(
      `「${folder.name}」包含子文件夹或文件，递归删除将一并删除其中全部内容，且不可恢复。`,
      '递归删除',
      { type: 'warning', confirmButtonText: '删除全部', cancelButtonText: '取消' }
    )
    if (recursiveOk.action !== 'confirm') return

    const recursiveSuccess = await deleteFolder(folder.id, true)
    if (recursiveSuccess) {
      QMessage.success('文件夹及内容已删除')
      emit('deleted', folder)
    } else {
      QMessage.error(error.value || '删除失败')
    }
  } else {
    QMessage.error(error.value || '删除文件夹失败')
  }
}

// 展开全部/收起全部
const toggleExpandAll = async () => {
  if (allExpanded.value) {
    collapseAll()
  } else {
    await expandAll()
  }
}

// 新建文件夹（父级 = 右键目标 ?? 当前选中节点，未选中时建在根）
const openCreateDialog = (parentOverride: FolderNode | null = null) => {
  createTargetFolder.value = parentOverride
  renameTarget.value = null
  createDialogMode.value = 'create'
}

// 重命名文件夹
const openRenameDialog = (folder: FolderNode) => {
  renameTarget.value = folder
  createDialogMode.value = 'rename'
}

/**
 * 跳转到指定文件夹（面包屑点击）：沿路径懒加载并展开各级祖先，然后选中目标
 */
const navigateTo = async (folderId: number | null) => {
  if (folderId === null) {
    selectRoot()
    emit('select', null)
    return
  }
  const target = findFolderInTree(folderId)
  if (!target) return

  // 从目标回溯到根的路径（visited 防环）
  const path: FolderNode[] = []
  const visited = new Set<number>()
  let node: FolderNode | null = target
  while (node && !visited.has(node.id)) {
    visited.add(node.id)
    path.unshift(node)
    if (node.parent_id === null) break
    node = findFolderInTree(node.parent_id)
  }

  // 祖先段缺 children 的先懒加载（目标自身无需展开子级）
  for (const seg of path.slice(0, -1)) {
    if (seg.children && seg.children.length > 0) continue
    const children = await loadChildren(seg.id)
    if (children === undefined) continue // 加载失败：跳过，不覆盖 hasChildren
    updateFolderInTree(seg.id, { children, hasChildren: children.length > 0 })
  }
  // 展开祖先，让目标可见
  for (const seg of path.slice(0, -1)) {
    expandedIds.value.add(seg.id)
  }

  selectFolder(target)
  emit('select', target)
}

// 键盘快捷键
const handleKeyDown = (e: KeyboardEvent) => {
  // Ctrl+N 新建文件夹
  if (e.ctrlKey && e.key === 'n') {
    e.preventDefault()
    openCreateDialog()
  }
  // Ctrl+Shift+E 展开全部
  if (e.ctrlKey && e.shiftKey && e.key === 'E') {
    e.preventDefault()
    toggleExpandAll()
  }
  // Escape 关闭对话框
  if (e.key === 'Escape' && createDialogMode.value !== null) {
    createDialogMode.value = null
  }
}

// 组件挂载时加载数据
onMounted(async () => {
  await loadRootFolders()
  window.addEventListener('keydown', handleKeyDown)
  // 外部点击/滚动/缩放时关闭右键菜单（浮层自身 @click.stop 不冒泡到 window）
  window.addEventListener('click', closeContextMenu)
  window.addEventListener('scroll', closeContextMenu, true)
  window.addEventListener('resize', closeContextMenu)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeyDown)
  window.removeEventListener('click', closeContextMenu)
  window.removeEventListener('scroll', closeContextMenu, true)
  window.removeEventListener('resize', closeContextMenu)
})

// 暴露方法给父组件
defineExpose({
  refresh,
  openCreateDialog,
  selectedFolder,
  navigateTo
})
</script>

<style scoped>
.folder-tree {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--panel-bg);
  border-radius: var(--radius-md);
  overflow: hidden;
}

/* 头部 */
.folder-tree__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-3) var(--spacing-4);
  border-bottom: 1px solid var(--border-color);
  background: var(--card-bg);
}

.folder-tree__title {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  margin: 0;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
}

.folder-tree__title svg {
  color: var(--color-warning-500);
}

.folder-tree__actions {
  display: flex;
  align-items: center;
  gap: var(--spacing-1);
}

.folder-tree__action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
  color: var(--text-secondary);
  transition: all var(--transition-base);
}

.folder-tree__action-btn:hover {
  background: var(--hover-color);
  color: var(--primary-color);
}

/* 来源筛选 */
.source-filter {
  display: flex;
  gap: 4px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border-color);
}

.source-tab {
  flex: 1;
  padding: 6px 12px;
  border: none;
  background: var(--hover-color);
  color: var(--text-color);
  border-radius: 4px;
  font-size: var(--font-size-xs);
  cursor: pointer;
  transition: all 0.2s;
}

.source-tab:hover {
  background: var(--primary-light);
  color: var(--primary-color);
}

.source-tab.active {
  background: var(--primary-color);
  color: white;
}

/* 搜索框 */
.folder-tree__search {
  position: relative;
  padding: var(--spacing-2) var(--spacing-4);
}

.folder-tree__search-input {
  width: 100%;
  padding: 8px 32px 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  background: var(--input-bg);
  color: var(--text-color);
  transition: all var(--transition-base);
  box-sizing: border-box;
}

.folder-tree__search-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(51, 133, 255, 0.1);
}

.folder-tree__search-input::placeholder {
  color: var(--text-secondary);
  opacity: 0.7;
}

.folder-tree__search-clear {
  position: absolute;
  right: 16px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-secondary);
  cursor: pointer;
  transition: color var(--transition-base);
}

.folder-tree__search-clear:hover {
  color: var(--text-color);
}

/* 加载状态 */
.folder-tree__loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-10);
  gap: var(--spacing-3);
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
}

.folder-tree__spinner {
  width: 28px;
  height: 28px;
  border: 3px solid var(--border-color);
  border-top-color: var(--primary-color);
  border-radius: var(--radius-full);
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* 错误状态 */
.folder-tree__error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-10);
  gap: var(--spacing-3);
  color: var(--error-color);
  text-align: center;
}

.folder-tree__error svg {
  opacity: 0.6;
}

.folder-tree__error p {
  margin: 0;
  font-size: var(--font-size-sm);
}

.folder-tree__retry-btn {
  padding: 8px 20px;
  border: 1px solid var(--primary-color);
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--primary-color);
  font-size: var(--font-size-sm);
  cursor: pointer;
  transition: all var(--transition-base);
}

.folder-tree__retry-btn:hover {
  background: var(--primary-color);
  color: white;
}

/* 空状态 */
.folder-tree__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-10);
  gap: var(--spacing-3);
  color: var(--text-secondary);
  text-align: center;
}

.folder-tree__empty svg {
  opacity: 0.4;
}

.folder-tree__empty p {
  margin: 0;
  font-size: var(--font-size-sm);
}

.folder-tree__create-btn {
  padding: 8px 20px;
  border: none;
  border-radius: var(--radius-md);
  background: var(--primary-color);
  color: white;
  font-size: var(--font-size-sm);
  cursor: pointer;
  transition: all var(--transition-base);
}

.folder-tree__create-btn:hover {
  background: var(--primary-dark);
}

/* 列表区域 */
.folder-tree__list {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-2) var(--spacing-2) var(--spacing-2) var(--spacing-3);
}

.folder-tree__list::-webkit-scrollbar {
  width: 4px;
}

.folder-tree__list::-webkit-scrollbar-track {
  background: transparent;
}

.folder-tree__list::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 3px;
  transition: background var(--transition-base);
}

.folder-tree__list::-webkit-scrollbar-thumb:hover {
  background: var(--text-secondary);
}

/* 根伪节点「全部文件」（行样式镜像 FolderTreeItem 的 folder-row） */
.folder-root-row {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  cursor: pointer;
  border-radius: 6px;
  transition: background-color var(--transition-base);
}

.folder-root-row:hover {
  background-color: var(--hover-color);
}

.folder-root-row--active {
  background-color: var(--primary-light);
  box-shadow: inset 2px 0 0 var(--primary-color);
}

.folder-root-indent {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}

.folder-root-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.folder-root-row--active .folder-root-icon {
  color: var(--primary-color);
}

.folder-root-name {
  font-size: var(--font-size-sm);
  color: var(--text-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
  min-width: 0;
}

.folder-root-row--active .folder-root-name {
  font-weight: var(--font-weight-medium);
  color: var(--primary-dark);
}

/* 底部统计 */
.folder-tree__footer {
  padding: var(--spacing-2) var(--spacing-4);
  border-top: 1px solid var(--border-color);
  background: var(--card-bg);
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  text-align: center;
}

/* 树行右键菜单 */
.folder-context-menu {
  position: fixed;
  z-index: 3000;
  min-width: 172px;
  padding: 6px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  box-shadow: 0 6px 24px rgba(0, 0, 0, 0.12);
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.folder-context-menu__item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: none;
  background: transparent;
  border-radius: var(--radius-sm);
  font-size: var(--font-size-sm);
  color: var(--text-color);
  cursor: pointer;
  text-align: left;
  white-space: nowrap;
}

.folder-context-menu__item:hover {
  background: var(--hover-color);
  color: var(--primary-color);
}

.folder-context-menu__item--danger {
  color: var(--error-color);
}

.folder-context-menu__item--danger:hover {
  background: var(--color-error-100);
  color: var(--color-error-500);
}

.folder-context-menu__divider {
  height: 1px;
  background: var(--border-color);
  margin: 4px 6px;
}
</style>
