import { ref, readonly } from 'vue'
import { useRequest } from './useRequest'

/**
 * 文件夹树节点接口
 */
export interface FolderTreeNode {
  id: number | string
  name: string
  parent_id?: number | string | null
  children?: FolderTreeNode[]
  icon?: string
  color?: string
  sort_order?: number
  [key: string]: unknown
}

/**
 * 文件夹树状态与操作接口
 */
export interface UseFolderTreeReturn {
  /** 根文件夹列表 */
  folders: Readonly<ReturnType<typeof ref<FolderTreeNode[]>>>
  /** 已展开的文件夹 ID 集合 */
  expandedFolders: Readonly<ReturnType<typeof ref<Set<number | string>>>>
  /** 子文件夹缓存（key: 父节点 ID, value: 子节点列表） */
  subFoldersCache: Readonly<
    ReturnType<typeof ref<Map<number | string, FolderTreeNode[]>>>
  >
  /** 加载中的节点 ID 集合 */
  loadingFolders: Readonly<ReturnType<typeof ref<Set<number | string>>>>
  /** 错误信息 */
  error: Readonly<ReturnType<typeof ref<string | null>>>

  /** 加载根文件夹 */
  loadRootFolders: () => Promise<void>
  /** 加载指定父节点的子文件夹（带缓存），返回是否加载成功 */
  loadSubFolders: (parentId: number | string) => Promise<boolean>
  /** 切换展开/收起状态 */
  toggleFolder: (folderId: number | string) => Promise<void>
  /** 获取指定节点的子文件夹 */
  getSubFolders: (parentId: number | string) => FolderTreeNode[]
  /** 判断节点是否展开 */
  isExpanded: (folderId: number | string) => boolean
  /** 清除缓存 */
  clearCache: () => void
  /** 重置所有状态 */
  reset: () => void
}

/**
 * API 响应包装类型
 */
interface FolderTreeApiResponse {
  code: number
  data: FolderTreeNode[]
  message?: string
}

/**
 * 文件夹树 Composable - 处理文件夹树懒加载、展开/收起及缓存
 *
 * 职责：
 * - 管理根文件夹列表
 * - 懒加载子文件夹并缓存结果
 * - 维护展开/收起状态
 * - 提供查询方法
 */
export function useFolderTree(): UseFolderTreeReturn {
  const { get } = useRequest()

  // --- State ---
  const folders = ref<FolderTreeNode[]>([])
  const expandedFolders = ref<Set<number | string>>(new Set())
  const subFoldersCache = ref<Map<number | string, FolderTreeNode[]>>(new Map())
  const loadingFolders = ref<Set<number | string>>(new Set())
  const error = ref<string | null>(null)

  // --- Public Methods ---

  /**
   * 加载根文件夹列表（parent_id 为 null 或不存在）
   */
  const loadRootFolders = async (): Promise<void> => {
    error.value = null
    const response = await get<FolderTreeApiResponse>('/api/v1/folders/tree', {
      params: { lazy: true }
    })

    if (response?.code === 0 && Array.isArray(response.data)) {
      folders.value = response.data
    } else {
      error.value = response?.message ?? '加载根文件夹失败'
    }
  }

  /**
   * 加载指定父节点的子文件夹
   * 如果已缓存则直接返回缓存数据，否则发起请求
   * 返回 true 表示加载成功（或缓存命中），false 表示加载失败
   */
  const loadSubFolders = async (
    parentId: number | string
  ): Promise<boolean> => {
    error.value = null

    // 缓存命中，直接返回
    const cached = subFoldersCache.value.get(parentId)
    if (cached !== undefined) {
      return true
    }

    // 避免重复请求
    if (loadingFolders.value.has(parentId)) {
      return false
    }

    loadingFolders.value.add(parentId)

    try {
      const response = await get<FolderTreeApiResponse>('/api/v1/folders/tree', {
        params: { lazy: true, parent_id: parentId }
      })

      if (response?.code === 0 && Array.isArray(response.data)) {
        subFoldersCache.value.set(parentId, response.data)
        return true
      }

      error.value = response?.message ?? '加载子文件夹失败'
      return false
    } catch (e) {
      error.value = e instanceof Error ? e.message : '加载子文件夹失败'
      return false
    } finally {
      loadingFolders.value.delete(parentId)
    }
  }

  /**
   * 切换文件夹展开/收起状态
   * 展开时自动加载子文件夹（若未缓存），仅加载成功时才标记为展开
   */
  const toggleFolder = async (folderId: number | string): Promise<void> => {
    if (expandedFolders.value.has(folderId)) {
      // 收起：仅更新状态，不清除缓存
      expandedFolders.value.delete(folderId)
    } else {
      // 展开：先加载子文件夹，仅成功时才更新展开状态
      const success = await loadSubFolders(folderId)
      if (success) {
        expandedFolders.value.add(folderId)
      }
    }
  }

  /**
   * 获取指定节点的子文件夹（从缓存中读取）
   */
  const getSubFolders = (parentId: number | string): FolderTreeNode[] => {
    return subFoldersCache.value.get(parentId) ?? []
  }

  /**
   * 判断节点是否处于展开状态
   */
  const isExpanded = (folderId: number | string): boolean => {
    return expandedFolders.value.has(folderId)
  }

  /**
   * 清除所有子文件夹缓存
   */
  const clearCache = (): void => {
    subFoldersCache.value.clear()
  }

  /**
   * 重置所有状态
   */
  const reset = (): void => {
    folders.value = []
    expandedFolders.value = new Set()
    subFoldersCache.value = new Map()
    loadingFolders.value = new Set()
    error.value = null
  }

  return {
    folders: readonly(folders),
    expandedFolders: readonly(expandedFolders),
    subFoldersCache: readonly(subFoldersCache),
    loadingFolders: readonly(loadingFolders),
    error: readonly(error),
    loadRootFolders,
    loadSubFolders,
    toggleFolder,
    getSubFolders,
    isExpanded,
    clearCache,
    reset
  }
}
