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
  // 加载状态
  const isLoading = ref(false)
  // 错误信息
  const error = ref<string | null>(null)

  /**
   * 加载文件列表
   */
  const loadFiles = async () => {
    isLoading.value = true
    error.value = null

    try {
      const params: FileListParams = {
        page: currentPage.value,
        page_size: pageSize.value,
        folder_id: currentFolderId.value,
        search: searchQuery.value || undefined,
        type: filterType.value || undefined,
        starred: showStarred.value || undefined
      }

      const response = await fileApi.getFiles(params)

      if (response.data.code === 0) {
        const data: FileListResponse = response.data.data
        files.value = data.files
        total.value = data.total
      } else {
        files.value = []
        total.value = 0
      }
    } catch (e) {
      error.value = e instanceof Error ? e.message : '加载文件失败'
      files.value = []
      total.value = 0
      QMessage.error('加载文件失败，请稍后重试')
    } finally {
      isLoading.value = false
    }
  }

  /**
   * 刷新当前页
   */
  const refresh = async () => {
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
      const response = await fileApi.batchOperation(fileIds, 'move', { folder_id: targetFolderId })
      if (response.data.code === 0) {
        QMessage.success(`已移动 ${fileIds.length} 个文件`)
        await refresh()
        return true
      }
      return false
    } catch (e) {
      QMessage.error('批量移动失败')
      return false
    }
  }

  // 计算总页数
  const totalPages = computed(() => Math.ceil(total.value / pageSize.value))

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
    isLoading,
    error,

    // 计算属性
    totalPages,
    hasFiles,

    // 方法
    loadFiles,
    refresh,
    changePage,
    changePageSize,
    search,
    changeFolder,
    changeFilterType,
    toggleStarred,
    uploadFile,
    deleteFile,
    toggleFileStar,
    batchDelete,
    batchMove
  }
}
