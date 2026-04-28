import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useFilePagination } from '@/composables/useFilePagination'

describe('useFilePagination', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // 辅助函数：创建 mock fetchFn
  const createMockFetchFn = () => {
    const allItems = Array.from({ length: 55 }, (_, i) => ({
      id: i + 1,
      name: `file-${i + 1}`
    }))

    return vi.fn(async (page: number, pageSize: number) => {
      const start = (page - 1) * pageSize
      const end = start + pageSize
      const list = allItems.slice(start, end)
      return { list, total: allItems.length }
    })
  }

  describe('初始状态', () => {
    it('应该初始化正确的默认状态', () => {
      const fetchFn = vi.fn()
      const { page, pageSize, items, total, loading } = useFilePagination({
        fetchFn,
        immediate: false
      })

      expect(page.value).toBe(1)
      expect(pageSize.value).toBe(20)
      expect(items.value).toEqual([])
      expect(total.value).toBe(0)
      expect(loading.value).toBe(false)
    })

    it('应该支持自定义 pageSize', () => {
      const fetchFn = vi.fn()
      const { pageSize } = useFilePagination({
        fetchFn,
        pageSize: 50,
        immediate: false
      })

      expect(pageSize.value).toBe(50)
    })
  })

  describe('计算属性', () => {
    it('totalPages 应该正确计算总页数', async () => {
      const fetchFn = vi.fn().mockResolvedValue({
        list: [{ id: 1 }],
        total: 55
      })

      const { totalPages, reset } = useFilePagination({
        fetchFn,
        pageSize: 20,
        immediate: false
      })

      expect(totalPages.value).toBe(0)

      await reset()
      expect(totalPages.value).toBe(3) // ceil(55/20) = 3
    })

    it('totalPages 在 total 为 0 时应该返回 0', () => {
      const fetchFn = vi.fn().mockResolvedValue({
        list: [],
        total: 0
      })

      const { totalPages } = useFilePagination({
        fetchFn,
        immediate: false
      })

      expect(totalPages.value).toBe(0)
    })

    it('hasMore 应该在还有更多数据时返回 true', async () => {
      const fetchFn = createMockFetchFn()

      const { hasMore, reset } = useFilePagination({
        fetchFn,
        pageSize: 20,
        immediate: false
      })

      expect(hasMore.value).toBe(false)

      await reset()
      // 加载第一页后：items.length = 20, total = 55, hasMore = true
      expect(hasMore.value).toBe(true)
    })

    it('hasMore 应该在加载完所有数据后返回 false', async () => {
      const fetchFn = vi.fn().mockResolvedValue({
        list: [{ id: 1 }, { id: 2 }, { id: 3 }],
        total: 3
      })

      const { hasMore, reset } = useFilePagination({
        fetchFn,
        pageSize: 20,
        immediate: false
      })

      await reset()

      // total = 3, pageSize = 20, totalPages = 1, page 已被递增为 2
      // hasMore = page < totalPages => 2 < 1 => false
      expect(hasMore.value).toBe(false)
    })
  })

  describe('loadNextPage', () => {
    it('应该加载第一页数据并追加到列表', async () => {
      const fetchFn = createMockFetchFn()

      const { items, total, loadNextPage, page } = useFilePagination({
        fetchFn,
        pageSize: 20,
        immediate: false
      })

      await loadNextPage()

      expect(items.value).toHaveLength(20)
      expect(items.value[0].id).toBe(1)
      expect(items.value[19].id).toBe(20)
      expect(total.value).toBe(55)
      // page 应该在请求成功后递增
      expect(page.value).toBe(2)
    })

    it('应该加载第二页数据并追加到列表', async () => {
      const fetchFn = createMockFetchFn()

      const { items, loadNextPage, hasMore } = useFilePagination({
        fetchFn,
        pageSize: 20,
        immediate: false
      })

      // 加载第一页
      await loadNextPage()
      expect(items.value).toHaveLength(20)

      // 加载第二页
      await loadNextPage()
      expect(items.value).toHaveLength(40)
      expect(items.value[20].id).toBe(21)
      expect(items.value[39].id).toBe(40)
    })

    it('应该加载第三页剩余数据', async () => {
      const fetchFn = createMockFetchFn()

      const { items, loadNextPage, hasMore } = useFilePagination({
        fetchFn,
        pageSize: 20,
        immediate: false
      })

      await loadNextPage() // 页1: 20条
      expect(items.value).toHaveLength(20)
      expect(hasMore.value).toBe(true)

      await loadNextPage() // 页2: 40条
      expect(items.value).toHaveLength(40)
      expect(hasMore.value).toBe(true)

      await loadNextPage() // 页3: 55条
      expect(items.value).toHaveLength(55)
      expect(items.value[54].id).toBe(55)
      expect(hasMore.value).toBe(false)
    })

    it('在没有更多数据时不应该继续加载', async () => {
      const fetchFn = createMockFetchFn()

      const { items, loadNextPage, hasMore } = useFilePagination({
        fetchFn,
        pageSize: 20,
        immediate: false
      })

      // 加载所有三页
      await loadNextPage()
      await loadNextPage()
      await loadNextPage()

      expect(hasMore.value).toBe(false)
      const previousLength = items.value.length

      // 尝试再次加载，应该被 hasMore 阻止
      await loadNextPage()
      expect(items.value).toHaveLength(previousLength)
      expect(fetchFn).toHaveBeenCalledTimes(3)
    })

    it('在正在加载时不应该重复加载', async () => {
      let resolvePromise: (value: { list: { id: number }[]; total: number }) => void
      const fetchFn = vi.fn().mockImplementation(() => {
        return new Promise(resolve => {
          resolvePromise = resolve
        })
      })

      const { loadNextPage, loading } = useFilePagination({
        fetchFn,
        immediate: false
      })

      loadNextPage()

      // 等待一个 tick 以确保 loading 状态已更新
      await Promise.resolve()
      await Promise.resolve()
      expect(loading.value).toBe(true)

      // 第二次加载应该被阻止
      await loadNextPage()
      expect(fetchFn).toHaveBeenCalledTimes(1)

      // 解决第一个请求
      if (resolvePromise) {
        resolvePromise({
          list: [{ id: 1 }],
          total: 10
        })
      }
      await Promise.resolve()
      await Promise.resolve()
    })

    it('应该在加载完成后将 loading 设置为 false', async () => {
      const fetchFn = createMockFetchFn()

      const { loadNextPage, loading } = useFilePagination({
        fetchFn,
        immediate: false
      })

      expect(loading.value).toBe(false)

      await loadNextPage()

      expect(loading.value).toBe(false)
    })

    it('应该在请求失败时将 loading 设置为 false', async () => {
      const fetchFn = vi.fn().mockRejectedValue(new Error('Network error'))

      const { loadNextPage, loading } = useFilePagination({
        fetchFn,
        immediate: false
      })

      await expect(loadNextPage()).rejects.toThrow('Network error')
      expect(loading.value).toBe(false)
    })
  })

  describe('loadMore', () => {
    it('loadMore 应该与 loadNextPage 行为相同', async () => {
      const fetchFn = createMockFetchFn()

      const { items, loadMore } = useFilePagination({
        fetchFn,
        pageSize: 20,
        immediate: false
      })

      await loadMore()

      expect(items.value).toHaveLength(20)
    })
  })

  describe('reset', () => {
    it('应该重置分页状态并重新加载', async () => {
      const fetchFn = createMockFetchFn()

      const { items, total, page, reset } = useFilePagination({
        fetchFn,
        pageSize: 20,
        immediate: false
      })

      // 先加载一些数据
      await loadNextPageHelper(reset)

      expect(items.value.length).toBeGreaterThan(0)

      // 重置
      await reset()

      expect(page.value).toBe(2) // reset 会调用 loadNextPage，page 会递增
      expect(total.value).toBe(55)
      expect(items.value).toHaveLength(20) // 重置后只加载第一页
      expect(fetchFn).toHaveBeenCalledTimes(2) // 第一次 + reset
    })

    it('应该清空列表再重新加载', async () => {
      const fetchFn = createMockFetchFn()

      const { items, reset, loadNextPage } = useFilePagination({
        fetchFn,
        pageSize: 20,
        immediate: false
      })

      await loadNextPage()
      await loadNextPage()
      expect(items.value).toHaveLength(40)

      await reset()
      expect(items.value).toHaveLength(20)
    })
  })

  describe('setItems', () => {
    it('应该手动更新列表数据', () => {
      const fetchFn = vi.fn()

      const { items, setItems } = useFilePagination({
        fetchFn,
        immediate: false
      })

      const newItems = [{ id: 100, name: 'custom' }]
      setItems(newItems)

      expect(items.value).toEqual(newItems)
    })
  })

  describe('immediate', () => {
    it('immediate 为 true 时应该自动加载第一页', async () => {
      const fetchFn = createMockFetchFn()

      const { items, total, loading } = useFilePagination({
        fetchFn,
        pageSize: 20,
        immediate: true
      })

      // 等待异步请求完成
      await new Promise(resolve => setTimeout(resolve, 0))

      expect(fetchFn).toHaveBeenCalledTimes(1)
      expect(items.value).toHaveLength(20)
      expect(total.value).toBe(55)
    })
  })

  describe('fetchFn 调用参数', () => {
    it('应该在加载时传入正确的页码和 pageSize', async () => {
      const fetchFn = createMockFetchFn()

      const { loadNextPage } = useFilePagination({
        fetchFn,
        pageSize: 15,
        immediate: false
      })

      await loadNextPage()

      expect(fetchFn).toHaveBeenCalledWith(1, 15)
    })

    it('应该在加载第二页时传入正确的页码', async () => {
      const fetchFn = createMockFetchFn()

      const { loadNextPage } = useFilePagination({
        fetchFn,
        pageSize: 15,
        immediate: false
      })

      await loadNextPage() // page 1
      await loadNextPage() // page 2

      expect(fetchFn).toHaveBeenLastCalledWith(2, 15)
    })
  })
})

// 辅助函数：等待异步操作完成
async function loadNextPageHelper(reset: () => Promise<void>) {
  await reset()
}
