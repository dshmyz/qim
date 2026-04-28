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
          @click="showCreateDialog = true"
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

    <!-- 搜索框 -->
    <div class="folder-tree__search" v-if="treeData.length > 5">
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
    <div v-else-if="error" class="folder-tree__error">
      <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="10" />
        <line x1="12" y1="8" x2="12" y2="12" />
        <line x1="12" y1="16" x2="12.01" y2="16" />
      </svg>
      <p>{{ error }}</p>
      <button class="folder-tree__retry-btn" @click="refresh">重试</button>
    </div>

    <!-- 空状态 -->
    <div v-else-if="filteredFolders.length === 0" class="folder-tree__empty">
      <svg viewBox="0 0 24 24" width="40" height="40" fill="none" stroke="currentColor" stroke-width="1.5">
        <path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z" />
      </svg>
      <p>{{ searchQuery ? '未找到匹配的文件夹' : '暂无文件夹' }}</p>
      <button class="folder-tree__create-btn" @click="showCreateDialog = true" v-if="!searchQuery">
        新建文件夹
      </button>
    </div>

    <!-- 文件夹列表 -->
    <div v-else class="folder-tree__list">
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
        @delete="handleDelete"
      />
    </div>

    <!-- 底部统计信息 -->
    <div class="folder-tree__footer" v-if="!isLoading && !error">
      <span>共 {{ totalFolders }} 个文件夹</span>
    </div>

    <!-- 新建文件夹对话框 -->
    <Teleport to="body">
      <div v-if="showCreateDialog" class="folder-tree-modal-overlay" @click="closeCreateDialog">
        <div class="folder-tree-modal" @click.stop>
          <div class="folder-tree-modal__header">
            <h3>新建文件夹</h3>
            <button class="folder-tree-modal__close" @click="closeCreateDialog">
              <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          </div>
          <div class="folder-tree-modal__body">
            <div class="folder-tree-form-group">
              <label for="folder-name">文件夹名称</label>
              <input
                id="folder-name"
                ref="folderNameInput"
                v-model="newFolderName"
                class="folder-tree-form-input"
                placeholder="请输入文件夹名称"
                type="text"
                @keyup.enter="handleCreate"
              />
            </div>
            <div v-if="createError" class="folder-tree-form-error">
              {{ createError }}
            </div>
          </div>
          <div class="folder-tree-modal__footer">
            <button class="folder-tree-btn folder-tree-btn--secondary" @click="closeCreateDialog">
              取消
            </button>
            <button
              class="folder-tree-btn folder-tree-btn--primary"
              :disabled="!newFolderName.trim() || isCreating"
              @click="handleCreate"
            >
              {{ isCreating ? '创建中...' : '创建' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick, watch, onBeforeUnmount } from 'vue'
import QMessage from '../../../utils/qmessage'
import FolderTreeItem from './FolderTreeItem.vue'
import { useFolderTree, type FolderNode } from '../../../composables/useFolderTree'

interface Props {
  selectedSource?: string | null
}

const props = withDefaults(defineProps<Props>(), {
  selectedSource: null
})

const emit = defineEmits<{
  (e: 'select', folder: FolderNode): void
  (e: 'sourceChange', source: string | null): void
}>()

const {
  treeData,
  expandedIds,
  selectedFolder,
  isLoading,
  error,
  totalFolders,
  loadRootFolders,
  toggleExpand,
  selectFolder,
  createFolder,
  deleteFolder,
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

// 新建文件夹对话框
const showCreateDialog = ref(false)
const newFolderName = ref('')
const isCreating = ref(false)
const createError = ref('')
const folderNameInput = ref<HTMLInputElement | null>(null)

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

// 选择文件夹
const handleSelect = (folder: FolderNode) => {
  selectFolder(folder)
  emit('select', folder)
}

// 删除文件夹
const handleDelete = async (folder: FolderNode) => {
  const success = await deleteFolder(folder.id)
  if (success) {
    QMessage.success('文件夹已删除')
  } else {
    QMessage.error('删除文件夹失败')
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

// 新建文件夹对话框
const openCreateDialog = async () => {
  showCreateDialog.value = true
  newFolderName.value = ''
  createError.value = ''
  await nextTick()
  folderNameInput.value?.focus()
}

const closeCreateDialog = () => {
  showCreateDialog.value = false
  newFolderName.value = ''
  createError.value = ''
}

const handleCreate = async () => {
  if (!newFolderName.value.trim()) return

  isCreating.value = true
  createError.value = ''

  try {
    const success = await createFolder(newFolderName.value.trim())
    if (success) {
      QMessage.success('文件夹创建成功')
      closeCreateDialog()
    } else {
      createError.value = '创建失败，请稍后重试'
    }
  } catch (e) {
    createError.value = e instanceof Error ? e.message : '创建失败'
  } finally {
    isCreating.value = false
  }
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
  // Escape 收起对话框
  if (e.key === 'Escape' && showCreateDialog.value) {
    closeCreateDialog()
  }
}

// 监听对话框显示，设置焦点
watch(showCreateDialog, async (visible) => {
  if (visible) {
    await nextTick()
    folderNameInput.value?.focus()
  }
})

// 组件挂载时加载数据
onMounted(async () => {
  await loadRootFolders()
  window.addEventListener('keydown', handleKeyDown)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeyDown)
})

// 暴露方法给父组件
defineExpose({
  refresh,
  openCreateDialog,
  selectedFolder
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
  font-size: var(--font-size-base);
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
  font-size: 13px;
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

/* 底部统计 */
.folder-tree__footer {
  padding: var(--spacing-2) var(--spacing-4);
  border-top: 1px solid var(--border-color);
  background: var(--card-bg);
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
  text-align: center;
}

/* 模态框覆盖层 */
.folder-tree-modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: var(--z-modal);
  animation: fadeIn var(--transition-base);
}

/* 模态框 */
.folder-tree-modal {
  background: var(--card-bg);
  border-radius: var(--radius-lg);
  width: 90%;
  max-width: 420px;
  box-shadow: var(--shadow-xl);
  animation: slideUp var(--transition-slow);
}

.folder-tree-modal__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-4) var(--spacing-5);
  border-bottom: 1px solid var(--border-color);
}

.folder-tree-modal__header h3 {
  margin: 0;
  font-size: var(--font-size-base);
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
}

.folder-tree-modal__close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
  color: var(--text-secondary);
  transition: all var(--transition-base);
}

.folder-tree-modal__close:hover {
  background: var(--hover-color);
  color: var(--text-color);
}

.folder-tree-modal__body {
  padding: var(--spacing-5);
}

.folder-tree-form-group {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2);
}

.folder-tree-form-group label {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--text-color);
}

.folder-tree-form-input {
  width: 100%;
  padding: 10px 14px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  background: var(--input-bg);
  color: var(--text-color);
  transition: all var(--transition-base);
  box-sizing: border-box;
}

.folder-tree-form-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(51, 133, 255, 0.1);
}

.folder-tree-form-error {
  margin-top: var(--spacing-2);
  padding: 8px 12px;
  background: var(--color-error-50);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  color: var(--color-error-500);
}

.folder-tree-modal__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--spacing-3);
  padding: var(--spacing-4) var(--spacing-5);
  border-top: 1px solid var(--border-color);
}

/* 按钮 */
.folder-tree-btn {
  padding: 8px 20px;
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  cursor: pointer;
  transition: all var(--transition-base);
}

.folder-tree-btn--secondary {
  border: 1px solid var(--border-color);
  background: var(--card-bg);
  color: var(--text-color);
}

.folder-tree-btn--secondary:hover {
  background: var(--hover-color);
}

.folder-tree-btn--primary {
  border: 1px solid var(--primary-color);
  background: var(--primary-color);
  color: white;
}

.folder-tree-btn--primary:hover:not(:disabled) {
  background: var(--primary-dark);
  border-color: var(--primary-dark);
}

.folder-tree-btn--primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* 动画 */
@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(16px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
