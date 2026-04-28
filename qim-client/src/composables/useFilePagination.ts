import { ref, computed, nextTick } from 'vue'

/**
 * 分页配置选项
 */
export interface UseFilePaginationOptions<T = any> {
  /** 每页条数 */
  pageSize?: number
  /** 加载函数，接收页码和每页条数，返回分页数据 */
  fetchFn: (page: number, pageSize: number) => Promise<{
    list: T[]
    total: number
  }>
  /** 是否在初始化时自动加载第一页 */
  immediate?: boolean
}

/**
 * 分页状态返回值
 */
export interface UseFilePaginationReturn<T> {
  /** 当前页码 */
  page: Readonly<ReturnType<typeof ref<number>>>
  /** 每页条数 */
  pageSize: Readonly<ReturnType<typeof ref<number>>>
  /** 所有已加载的条目列表 */
  items: Readonly<ReturnType<typeof ref<T[]>>>
  /** 总条数 */
  total: Readonly<ReturnType<typeof ref<number>>>
  /** 是否正在加载 */
  loading: Readonly<ReturnType<typeof ref<boolean>>>
  /** 总页数 */
  totalPages: Readonly<ReturnType<typeof computed<number>>>
  /** 是否还有更多数据可加载 */
  hasMore: Readonly<ReturnType<typeof computed<boolean>>>
  /** 加载下一页数据并追加到列表 */
  loadNextPage: () => Promise<void>
  /** 加载更多（与 loadNextPage 相同，用于无限滚动场景） */
  loadMore: () => Promise<void>
  /** 重置分页状态并重新加载第一页 */
  reset: () => Promise<void>
  /** 手动更新列表数据（不改变分页状态） */
  setItems: (newItems: T[]) => void
}

/**
 * 文件列表分页加载 Composable
 *
 * 支持：
 * - 分页状态管理（page, pageSize, total, loading）
 * - 自动追加下一页数据
 * - 无限滚动（hasMore 判断是否还有更多）
 * - 重置与重新加载
 *
 * @example
 * const { items, loading, hasMore, loadMore, reset } = useFilePagination({
 *   pageSize: 20,
 *   fetchFn: async (page, pageSize) => {
 *     const res = await request('/api/files', { params: { page, pageSize } })
 *     return { list: res.data.files, total: res.data.total }
 *   },
 *   immediate: true
 * })
 */
export function useFilePagination<T = any>(
  options: UseFilePaginationOptions<T>
): UseFilePaginationReturn<T> {
  const { pageSize: initialPageSize = 20, fetchFn, immediate = false } = options

  const page = ref<number>(1)
  const pageSize = ref<number>(initialPageSize)
  const items = ref<T[]>([]) as ReturnType<typeof ref<T[]>>
  const total = ref<number>(0)
  const loading = ref<boolean>(false)

  /** 总页数 */
  const totalPages = computed<number>(() => {
    if (total.value === 0) return 0
    return Math.ceil(total.value / pageSize.value)
  })

  /** 是否还有更多数据 */
  const hasMore = computed<boolean>(() => {
    return items.value.length < total.value
  })

  /**
   * 加载下一页数据，并追加到现有列表
   */
  const loadNextPage = async (): Promise<void> => {
    // 防止重复加载
    if (loading.value) {
      return
    }

    // 没有更多数据时不再请求
    if (!hasMore.value && items.value.length > 0) {
      return
    }

    loading.value = true

    try {
      const result = await fetchFn(page.value, pageSize.value)

      // 更新总条数
      total.value = result.total

      // 第一页替换数据，后续页追加数据
      if (page.value === 1) {
        items.value = result.list
      } else {
        items.value = [...items.value, ...result.list]
      }

      // 请求成功后，页码递增，为下次加载做准备
      await nextTick()
      page.value++
    } finally {
      loading.value = false
    }
  }

  /**
   * 加载更多（与 loadNextPage 相同，语义上用于无限滚动）
   */
  const loadMore = loadNextPage

  /**
   * 重置分页状态并重新加载第一页
   */
  const reset = async (): Promise<void> => {
    page.value = 1
    items.value = []
    total.value = 0

    await loadNextPage()
  }

  /**
   * 手动设置列表数据
   */
  const setItems = (newItems: T[]): void => {
    items.value = newItems
  }

  // 如果设置 immediate，则自动加载第一页
  if (immediate) {
    loadNextPage()
  }

  return {
    page,
    pageSize,
    items,
    total,
    loading,
    totalPages,
    hasMore,
    loadNextPage,
    loadMore,
    reset,
    setItems
  }
}
