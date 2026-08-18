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
  /** 展开依据（懒加载），进入树前由 has_children 归一化而来 */
  hasChildren?: boolean
  /** 后端原始字段（Go json tag 蛇形命名），仅 wire 层存在，loadRootFolders/loadChildren 会归一化 */
  has_children?: boolean
  /** 直接子文件数（树行计数展示，normalizeFolders 归一化） */
  fileCount?: number
  file_count?: number
  path?: string
}

// ===== 模块级单例状态 =====
// 提升到模块作用域：FileManagementApp 与 FolderTree 两个调用方共享同一棵树，
// 树内 CRUD 后所有派生状态（folderOptions/folderPath）自动一致，且避免双 mount 双请求
const treeData = ref<FolderNode[]>([])
// 已展开的节点 ID 集合
const expandedIds = ref<Set<number>>(new Set())
// 当前选中的文件夹
const selectedFolder = ref<FolderNode | null>(null)
// 加载状态
const isLoading = ref(false)
// 树加载失败标志（仅初始/刷新加载失败时驱动侧栏错误屏；CRUD 失败只写 error 文案不置位）
const loadFailed = ref(false)
// 错误信息：加载失败 or 最近一次 CRUD 失败的后端文案（删除两段式确认的递归提示依据）
const error = ref<string | null>(null)

/**
 * 文件夹选择项（树选择器与面包屑用）
 */
export interface FolderOption {
  folder: FolderNode
  depth: number
}

/**
 * 文件夹树 composable
 * 支持懒加载、展开/收起、选择文件夹
 */
export function useFolderTree() {
  const requestMethods = useRequest()
  const { get, post, put } = requestMethods
  // 删除必须走原始 request（非 safeRequest）：safeRequest 吞异常返回 null，
  // 后端 400 的真实文案（「文件夹包含子文件夹…」）会丢失，导致二次递归确认无法触发
  const request = requestMethods.request

  /**
   * 加载根文件夹列表
   */
  const loadRootFolders = async () => {
    if (isLoading.value) return
    isLoading.value = true
    error.value = null

    try {
      const response = await get<ApiResponse<FolderNode[]>>('/api/v1/folders/tree')
      if (response?.code === 0 && Array.isArray(response.data)) {
        treeData.value = normalizeFolders(response.data)
        loadFailed.value = false
        // 树就绪后自动全量拉取深层节点：FileManagementApp onMounted 里的调用会因
        // FolderTree（子组件）先挂载 + 本函数 isLoading 幂等守卫被跳过，届时 treeData
        // 为空导致 fetchAllFolders 空转（fetchingAll 守卫防重入，此处不 await）
        void fetchAllFolders()
      } else {
        treeData.value = []
      }
    } catch (e) {
      error.value = e instanceof Error ? e.message : '加载文件夹失败'
      loadFailed.value = true
      treeData.value = []
    } finally {
      isLoading.value = false
    }
  }

  /**
   * 懒加载子文件夹
   * 失败返回 undefined（而非 []）：调用方据此跳过 updateFolderInTree，
   * 避免把 hasChildren 覆盖为 false 而丢失展开箭头
   */
  const loadChildren = async (folderId: number): Promise<FolderNode[] | undefined> => {
    try {
      const response = await get<ApiResponse<FolderNode[]>>(
        `/api/v1/folders/tree`,
        { params: { parent_id: folderId } }
      )

      if (response?.code === 0 && Array.isArray(response.data)) {
        return normalizeFolders(response.data)
      }
      return undefined
    } catch (e) {
      console.error('加载子文件夹失败:', e)
      return undefined
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
        if (children !== undefined) {
          updateFolderInTree(folder.id, { children, hasChildren: children.length > 0 })
        }
        // 加载失败：不覆盖 hasChildren（箭头保留可重试）
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
            if (children !== undefined) {
              updateFolderInTree(parentId, { children, hasChildren: children.length > 0 })
            }
          } else if (parent) {
            // 父节点未展开：标记可展开，让箭头出现（展开时再懒加载真实子节点）
            updateFolderInTree(parentId, { hasChildren: true })
          }
        }
        return true
      }
      error.value = '创建文件夹失败，请稍后重试'
      return false
    } catch (e) {
      error.value = e instanceof Error ? e.message : '创建文件夹失败'
      return false
    }
  }

  /**
   * 重命名文件夹
   */
  const renameFolder = async (folderId: number, name: string): Promise<boolean> => {
    try {
      const response = await put<ApiResponse<FolderNode>>(`/api/v1/folders/${folderId}`, { name })
      if (response?.code === 0) {
        updateFolderInTree(folderId, { name })
        return true
      }
      error.value = '重命名文件夹失败，请稍后重试'
      return false
    } catch (e) {
      error.value = e instanceof Error ? e.message : '重命名文件夹失败'
      return false
    }
  }

  /**
   * 删除文件夹
   * @param recursive 是否递归删除子文件夹及其中文件（不可恢复）
   */
  const deleteFolder = async (folderId: number, recursive = false): Promise<boolean> => {
    try {
      const response = await request<ApiResponse<void>>(`/api/v1/folders/${folderId}`, {
        method: 'DELETE',
        ...(recursive ? { params: { recursive: true } } : {})
      })

      if (response?.code === 0) {
        // 从树中移除该节点
        removeFolderFromTree(folderId)
        // 如果选中的是被删除的文件夹，清空选择
        if (selectedFolder.value?.id === folderId) {
          selectedFolder.value = null
        }
        return true
      }
      error.value = '删除文件夹失败，请稍后重试'
      return false
    } catch (e) {
      // 400 时后端 message（如「文件夹包含子文件夹…」）从这里透出，供二次确认分支匹配
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
          if (children === undefined) continue // 加载失败：跳过，箭头保留可重试
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

  /**
   * 全量拉取文件夹树（BFS 逐层懒加载），供树选择器/面包屑等需要完整选项的场景
   * 串行化：若已有进行中的实例，等待其完成后再重跑一次（幂等，已加载节点跳过）。
   * 原因：loadRootFolders 成功分支会 fire-and-forget 触发本函数；调用方（如 FileManagementApp
   * onMounted）紧随其后的调用若被「进行中直接返回」守卫挡掉，会因看不到待拉取的
   * has_children 节点而空转（A 挂载竞态：子组件先 loadRootFolders，父组件的调用被
   * isLoading 守卫跳过时 treeData 已就绪，但 fetchAllFolders 被挡 → 深层永久缺失）
   */
  let fetchingAll: Promise<void> | null = null
  const fetchAllFolders = (): Promise<void> => {
    const prev = fetchingAll
    const run = async () => {
      await prev
      const queue: FolderNode[] = [...treeData.value]
      let cursor = 0
      // 小并发批量拉取：同批内并行、批间串行，避免大树一次性打爆连接数
      while (cursor < queue.length) {
        const batch = queue.slice(cursor, cursor + 6)
        cursor += batch.length
        const results = await Promise.all(batch.map(async (node) => {
          if (!(node.hasChildren && (!node.children || node.children.length === 0))) return null
          const children = await loadChildren(node.id)
          if (children === undefined) return null // 加载失败：该子树下次再拉
          updateFolderInTree(node.id, { children, hasChildren: children.length > 0 })
          return children
        }))
        for (const children of results) {
          if (children) queue.push(...children)
        }
      }
    }
    const p = run()
    fetchingAll = p
    return p
  }

  /**
   * 树选择器选项：递归平铺带深度（根 → 全部，按树序）
   */
  const folderOptions = computed<FolderOption[]>(() => {
    const out: FolderOption[] = []
    const walk = (nodes: FolderNode[], depth: number) => {
      for (const node of nodes) {
        out.push({ folder: node, depth })
        if (node.children && node.children.length > 0) walk(node.children, depth + 1)
      }
    }
    walk(treeData.value, 0)
    return out
  })

  /**
   * 当前选中文件夹的父链（根 → 当前，不含根段；面包屑数据源）
   * 沿 parent_id 回溯，visited 防环
   */
  const folderPath = computed<FolderNode[]>(() => {
    if (!selectedFolder.value) return []
    const path: FolderNode[] = []
    const visited = new Set<number>()
    let node: FolderNode | null = selectedFolder.value
    while (node && !visited.has(node.id)) {
      visited.add(node.id)
      path.unshift(node)
      if (node.parent_id === null) break
      node = findFolderInTree(node.parent_id)
    }
    return path
  })

  /**
   * 选中根目录（全部文件）
   */
  const selectRoot = () => {
    selectedFolder.value = null
  }

  // 统计信息
  const totalFolders = computed(() => countFolders(treeData.value))

  return {
    // 状态
    treeData,
    expandedIds,
    selectedFolder,
    isLoading,
    loadFailed,
    error,
    totalFolders,

    // 计算属性
    folderOptions,
    folderPath,

    // 方法
    loadRootFolders,
    loadChildren,
    toggleExpand,
    selectFolder,
    selectRoot,
    createFolder,
    renameFolder,
    deleteFolder,
    fetchAllFolders,
    expandAll,
    collapseAll,
    isExpanded,
    isSelected,
    isExpandable,
    findFolderInTree,
    updateFolderInTree
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

/**
 * 归一化后端节点：has_children / file_count（Go json tag 蛇形命名）→ hasChildren / fileCount
 * 懒加载展开依据必须落到 camelCase，否则 isExpandable 恒 false、箭头永不渲染
 */
function normalizeFolders(nodes: FolderNode[]): FolderNode[] {
  return nodes.map(f => ({
    ...f,
    hasChildren: f.has_children === true,
    fileCount: f.file_count ?? 0
  }))
}
