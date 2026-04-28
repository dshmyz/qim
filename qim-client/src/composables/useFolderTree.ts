import { ref, computed } from 'vue'
import { useRequest, type ApiResponse } from './useRequest'

/**
 * 文件夹节点接口
 */
export interface FolderNode {
  id: number
  user_id: number
  name: string
  parent_id: number | null
  sort_order?: number
  icon?: string
  color?: string
  created_at?: string
  updated_at?: string
  children?: FolderNode[]
  hasChildren?: boolean
  path?: string
}

/**
 * 文件夹树 composable
 * 支持懒加载、展开/收起、选择文件夹
 */
export function useFolderTree() {
  const requestMethods = useRequest()
  const { get, post } = requestMethods
  // useRequest 返回的 delete 方法被命名为 delete（TS 关键字），需要这样访问
  const deleteRequest = requestMethods.delete as <T = any>(url: string, options?: import('./useRequest').RequestOptions) => Promise<T | null>

  // 树根节点
  const treeData = ref<FolderNode[]>([])
  // 已展开的节点 ID 集合
  const expandedIds = ref<Set<number>>(new Set())
  // 当前选中的文件夹
  const selectedFolder = ref<FolderNode | null>(null)
  // 加载状态
  const isLoading = ref(false)
  // 错误信息
  const error = ref<string | null>(null)

  /**
   * 加载根文件夹列表
   */
  const loadRootFolders = async () => {
    isLoading.value = true
    error.value = null

    try {
      const response = await get<ApiResponse<FolderNode[]>>('/api/v1/folders/tree')
      if (response?.code === 0 && Array.isArray(response.data)) {
        treeData.value = response.data
      } else {
        treeData.value = []
      }
    } catch (e) {
      error.value = e instanceof Error ? e.message : '加载文件夹失败'
      treeData.value = []
    } finally {
      isLoading.value = false
    }
  }

  /**
   * 懒加载子文件夹
   */
  const loadChildren = async (folderId: number): Promise<FolderNode[]> => {
    try {
      const response = await get<ApiResponse<FolderNode[]>>(
        `/api/v1/folders/tree`,
        { params: { parent_id: folderId } }
      )

      if (response?.code === 0 && Array.isArray(response.data)) {
        return response.data
      }
      return []
    } catch (e) {
      console.error('加载子文件夹失败:', e)
      return []
    }
  }

  /**
   * 切换展开/收起状态
   */
  const toggleExpand = async (folder: FolderNode) => {
    const id = folder.id
    const isExpanded = expandedIds.value.has(id)

    if (isExpanded) {
      // 收起：移除 ID，同时移除所有子节点的展开状态
      expandedIds.value.delete(id)
      removeChildrenExpanded(id)
    } else {
      // 展开：加载子节点（如果还没有的话）
      if (!folder.children || folder.children.length === 0) {
        const children = await loadChildren(id)
        // 通过遍历树更新对应节点的 children
        updateFolderInTree(folder.id, { children, hasChildren: children.length > 0 })
      }
      expandedIds.value.add(id)
    }
  }

  /**
   * 移除子节点的展开状态（收起时清理）
   */
  const removeChildrenExpanded = (parentId: number) => {
    const parent = findFolderInTree(parentId)
    if (parent && parent.children) {
      for (const child of parent.children) {
        expandedIds.value.delete(child.id)
        removeChildrenExpanded(child.id)
      }
    }
  }

  /**
   * 在树中查找并更新文件夹
   */
  const updateFolderInTree = (folderId: number, updates: Partial<FolderNode>) => {
    const updateNode = (nodes: FolderNode[]): boolean => {
      for (const node of nodes) {
        if (node.id === folderId) {
          Object.assign(node, updates)
          return true
        }
        if (node.children && node.children.length > 0) {
          if (updateNode(node.children)) return true
        }
      }
      return false
    }
    updateNode(treeData.value)
  }

  /**
   * 在树中查找文件夹节点
   */
  const findFolderInTree = (folderId: number): FolderNode | null => {
    const find = (nodes: FolderNode[]): FolderNode | null => {
      for (const node of nodes) {
        if (node.id === folderId) return node
        if (node.children && node.children.length > 0) {
          const found = find(node.children)
          if (found) return found
        }
      }
      return null
    }
    return find(treeData.value)
  }

  /**
   * 选择文件夹
   */
  const selectFolder = (folder: FolderNode) => {
    selectedFolder.value = folder
  }

  /**
   * 创建文件夹
   */
  const createFolder = async (name: string, parentId: number | null = null): Promise<boolean> => {
    try {
      const response = await post<ApiResponse<FolderNode>>('/api/v1/folders', {
        name,
        parent_id: parentId
      })

      if (response?.code === 0) {
        // 刷新根目录或父目录
        if (!parentId) {
          await loadRootFolders()
        } else {
          // 如果父节点已展开，重新加载子节点
          const parent = findFolderInTree(parentId)
          if (parent && expandedIds.value.has(parentId)) {
            const children = await loadChildren(parentId)
            updateFolderInTree(parentId, { children, hasChildren: children.length > 0 })
          }
        }
        return true
      }
      return false
    } catch (e) {
      error.value = e instanceof Error ? e.message : '创建文件夹失败'
      return false
    }
  }

  /**
   * 删除文件夹
   */
  const deleteFolder = async (folderId: number): Promise<boolean> => {
    try {
      const response = await deleteRequest<ApiResponse<void>>(`/api/v1/folders/${folderId}`)

      if (response?.code === 0) {
        // 从树中移除该节点
        removeFolderFromTree(folderId)
        // 如果选中的是被删除的文件夹，清空选择
        if (selectedFolder.value?.id === folderId) {
          selectedFolder.value = null
        }
        return true
      }
      return false
    } catch (e) {
      error.value = e instanceof Error ? e.message : '删除文件夹失败'
      return false
    }
  }

  /**
   * 从树中移除文件夹节点
   */
  const removeFolderFromTree = (folderId: number) => {
    const removeFromNodes = (nodes: FolderNode[]): boolean => {
      const index = nodes.findIndex(n => n.id === folderId)
      if (index !== -1) {
        nodes.splice(index, 1)
        return true
      }
      for (const node of nodes) {
        if (node.children && removeFromNodes(node.children)) return true
      }
      return false
    }
    removeFromNodes(treeData.value)
    expandedIds.value.delete(folderId)
  }

  /**
   * 展开所有节点（慎用，节点多时会很卡）
   */
  const expandAll = async () => {
    const expandNode = async (nodes: FolderNode[]) => {
      for (const node of nodes) {
        expandedIds.value.add(node.id)
        if (!node.children || node.children.length === 0) {
          const children = await loadChildren(node.id)
          updateFolderInTree(node.id, { children, hasChildren: children.length > 0 })
          // 获取更新后的 children
          const updatedNode = findFolderInTree(node.id)
          if (updatedNode?.children && updatedNode.children.length > 0) {
            await expandNode(updatedNode.children)
          }
        } else if (node.children.length > 0) {
          await expandNode(node.children)
        }
      }
    }
    await expandNode(treeData.value)
  }

  /**
   * 收起所有节点
   */
  const collapseAll = () => {
    expandedIds.value.clear()
  }

  /**
   * 判断节点是否展开
   */
  const isExpanded = (folderId: number): boolean => {
    return expandedIds.value.has(folderId)
  }

  /**
   * 判断节点是否选中
   */
  const isSelected = (folderId: number): boolean => {
    return selectedFolder.value?.id === folderId
  }

  /**
   * 判断节点是否可展开（有子节点或有加载子节点的标志）
   */
  const isExpandable = (folder: FolderNode): boolean => {
    return folder.hasChildren === true || !!(folder.children && folder.children.length > 0)
  }

  // 统计信息
  const totalFolders = computed(() => countFolders(treeData.value))

  return {
    // 状态
    treeData,
    expandedIds,
    selectedFolder,
    isLoading,
    error,
    totalFolders,

    // 方法
    loadRootFolders,
    loadChildren,
    toggleExpand,
    selectFolder,
    createFolder,
    deleteFolder,
    expandAll,
    collapseAll,
    isExpanded,
    isSelected,
    isExpandable,
    findFolderInTree
  }
}

/**
 * 递归统计文件夹数量
 */
function countFolders(nodes: FolderNode[]): number {
  let count = nodes.length
  for (const node of nodes) {
    if (node.children && node.children.length > 0) {
      count += countFolders(node.children)
    }
  }
  return count
}
