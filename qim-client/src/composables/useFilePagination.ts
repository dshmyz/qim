import { ref, computed, watch } from 'vue'
import { fileApi, type FileItem, type FileListParams, type FileListResponse } from '../api/file'
import QMessage from '../utils/qmessage'

/**
 * 文件分页 composable
 * 支持分页、搜索、过滤、排序
 */
export function useFilePagination() {
  // 文件列表
  const files = ref<FileItem[]>([])
  // 总数
  const total = ref(0)
  // 当前页
  const currentPage = ref(1)
  // 每页数量
  const pageSize = ref(20)
  // 搜索关键词
  const searchQuery = ref('')
  // 当前文件夹 ID
  const currentFolderId = ref<number | null>(null)
  // 文件类型过滤
  const filterType = ref<string>('')
  // 是否只显示星标
  const showStarred = ref(false)
  // 文件来源筛选
  const sourceFilter = ref<string | null>(null)
  // 排序字段
  const sortBy = ref<string>('created_at')
  // 排序方向
  const sortOrder = ref<string>('desc')
  // 日期范围 - 起始
  const dateFrom = ref<string>('')
  // 日期范围 - 结束
  const dateTo = ref<string>('')
  // 加载状态
  const isLoading = ref(false)
  // 错误信息
  const error = ref<string | null>(null)

  /**
   * 拉取指定页文件
   * @param page 页码
   * @param mode replace=整页替换；append=追加下一页（无限滚动）
   * @returns 是否成功
   */
  const fetchFiles = async (page: number, mode: 'replace' | 'append' = 'replace'): Promise<boolean> => {
    isLoading.value = true
    error.value = null

    try {
      const params: FileListParams = {
        page,
        page_size: pageSize.value,
        folder_id: currentFolderId.value,
        search: searchQuery.value || undefined,
        type: filterType.value || undefined,
        starred: showStarred.value || undefined,
        source: sourceFilter.value || undefined,
        sort_by: sortBy.value,
        sort_order: sortOrder.value,
        date_from: dateFrom.value || undefined,
        date_to: dateTo.value || undefined
      }

      const response = await fileApi.getFiles(params)

      if (response.data.code === 0) {
        const data: FileListResponse = response.data.data
        if (mode === 'append') {
          files.value = [...files.value, ...data.files]
        } else {
          files.value = data.files
        }
        total.value = data.total
        return true
      }
      if (mode === 'replace') {
        files.value = []
        total.value = 0
      }
      return false
    } catch (e) {
      error.value = e instanceof Error ? e.message : '加载文件失败'
      if (mode === 'replace') {
        files.value = []
        total.value = 0
      }
      QMessage.error('加载文件失败，请稍后重试')
      return false
    } finally {
      isLoading.value = false
    }
  }

  /**
   * 加载文件列表（整页替换）
   */
  const loadFiles = async (): Promise<void> => {
    await fetchFiles(currentPage.value, 'replace')
  }

  /**
   * 加载下一页并追加（无限滚动）
   */
  const loadMore = async (): Promise<void> => {
    if (isLoading.value) return
    const nextPage = currentPage.value + 1
    currentPage.value = nextPage
    const ok = await fetchFiles(nextPage, 'append')
    if (!ok) {
      currentPage.value = nextPage - 1
    }
  }

  /**
   * 刷新（回到第 1 页整页加载；删除/移动/星标等变更后回到顶部，行为可预期）
   */
  const refresh = async (): Promise<void> => {
    currentPage.value = 1
    await loadFiles()
  }

  /**
   * 切换页码
   */
  const changePage = async (page: number) => {
    currentPage.value = page
    await loadFiles()
  }

  /**
   * 切换每页数量
   */
  const changePageSize = async (size: number) => {
    pageSize.value = size
    currentPage.value = 1
    await loadFiles()
  }

  /**
   * 搜索文件
   */
  const search = async (query: string) => {
    searchQuery.value = query
    currentPage.value = 1
    await loadFiles()
  }

  /**
   * 切换文件夹
   */
  const changeFolder = async (folderId: number | null) => {
    currentFolderId.value = folderId
    currentPage.value = 1
    await loadFiles()
  }

  /**
   * 切换文件类型过滤
   */
  const changeFilterType = async (type: string) => {
    filterType.value = type
    currentPage.value = 1
    await loadFiles()
  }

  /**
   * 切换星标过滤
   */
  const toggleStarred = async () => {
    showStarred.value = !showStarred.value
    currentPage.value = 1
    await loadFiles()
  }

  /**
   * 切换来源过滤
   */
  const changeSource = async (source: string | null) => {
    sourceFilter.value = source
    currentPage.value = 1
    await loadFiles()
  }

  /**
   * 切换排序
   */
  const changeSort = async (field: string, order: string) => {
    sortBy.value = field
    sortOrder.value = order
    currentPage.value = 1
    await loadFiles()
  }

  /**
   * 设置日期范围
   */
  const changeDateRange = async (from: string, to: string) => {
    dateFrom.value = from
    dateTo.value = to
    currentPage.value = 1
    await loadFiles()
  }

  /**
   * 清除日期范围
   */
  const clearDateRange = async () => {
    dateFrom.value = ''
    dateTo.value = ''
    currentPage.value = 1
    await loadFiles()
  }

  /**
   * 上传文件
   */
  const uploadFile = async (file: File, folderId?: number) => {
    try {
      const response = await fileApi.uploadFile(file, folderId || currentFolderId.value || undefined)
      if (response.data.code === 0) {
        QMessage.success('文件上传成功')
        await refresh()
        return true
      }
      return false
    } catch (e) {
      QMessage.error('文件上传失败')
      return false
    }
  }

  /**
   * 删除文件
   */
  const deleteFile = async (fileId: number) => {
    try {
      const response = await fileApi.deleteFile(fileId)
      if (response.data.code === 0) {
        QMessage.success('文件已删除')
        await refresh()
        return true
      }
      return false
    } catch (e) {
      QMessage.error('删除文件失败')
      return false
    }
  }

  /**
   * 切换星标
   */
  const toggleFileStar = async (fileId: number, starred: boolean) => {
    try {
      const response = await fileApi.toggleStar(fileId, starred)
      if (response.data.code === 0) {
        await refresh()
        return true
      }
      return false
    } catch (e) {
      QMessage.error('操作失败')
      return false
    }
  }

  /**
   * 批量删除
   */
  const batchDelete = async (fileIds: number[]) => {
    try {
      const response = await fileApi.batchOperation(fileIds, 'delete')
      if (response.data.code === 0) {
        QMessage.success(`已删除 ${fileIds.length} 个文件`)
        await refresh()
        return true
      }
      return false
    } catch (e) {
      QMessage.error('批量删除失败')
      return false
    }
  }

  /**
   * 批量移动
   */
  const batchMove = async (fileIds: number[], targetFolderId: number | null) => {
    try {
      if (targetFolderId === null) {
        // 移至根目录：批量接口要求非空目标文件夹，退化为逐个 update
        for (const id of fileIds) {
          await fileApi.updateFile(id, { folder_id: null })
        }
      } else {
        // 后端 BatchOperation 绑定 target_folder_id
        const response = await fileApi.batchOperation(fileIds, 'move', { target_folder_id: targetFolderId })
        if (response.data.code !== 0) {
          QMessage.error('批量移动失败')
          return false
        }
      }
      QMessage.success(`已移动 ${fileIds.length} 个文件`)
      await refresh()
      return true
    } catch (e) {
      QMessage.error('批量移动失败')
      return false
    }
  }

  /**
   * 批量星标 / 取消星标
   */
  const batchToggleStar = async (fileIds: number[], starred: boolean) => {
    try {
      const response = await fileApi.batchOperation(fileIds, starred ? 'star' : 'unstar')
      if (response.data.code === 0) {
        QMessage.success(starred ? `已星标 ${fileIds.length} 个文件` : `已取消 ${fileIds.length} 个文件的星标`)
        await refresh()
        return true
      }
      return false
    } catch (e) {
      QMessage.error('批量星标操作失败')
      return false
    }
  }

  // 计算总页数
  const totalPages = computed(() => Math.ceil(total.value / pageSize.value))

  // 是否还有更多页（供无限滚动判断）
  const hasMore = computed(() => files.value.length < total.value)

  // 计算是否有文件
  const hasFiles = computed(() => files.value.length > 0)

  // 监听搜索关键词变化（防抖）
  let searchTimer: NodeJS.Timeout | null = null
  watch(searchQuery, () => {
    if (searchTimer) clearTimeout(searchTimer)
    searchTimer = setTimeout(() => {
      currentPage.value = 1
      loadFiles()
    }, 300)
  })

  return {
    // 状态
    files,
    total,
    currentPage,
    pageSize,
    searchQuery,
    currentFolderId,
    filterType,
    showStarred,
    sourceFilter,
    sortBy,
    sortOrder,
    dateFrom,
    dateTo,
    isLoading,
    error,

    // 计算属性
    totalPages,
    hasMore,
    hasFiles,

    // 方法
    loadFiles,
    loadMore,
    refresh,
    changePage,
    changePageSize,
    search,
    changeFolder,
    changeFilterType,
    toggleStarred,
    changeSource,
    changeSort,
    changeDateRange,
    clearDateRange,
    uploadFile,
    deleteFile,
    toggleFileStar,
    batchDelete,
    batchMove,
    batchToggleStar
  }
}
