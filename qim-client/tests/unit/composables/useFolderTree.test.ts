import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useFolderTree } from '@/composables/useFolderTree'
import type { FolderTreeNode } from '@/composables/useFolderTree'

describe('useFolderTree', () => {
  // Mock data
  const mockRootFolders: FolderTreeNode[] = [
    { id: 1, name: 'Documents', parent_id: null },
    { id: 2, name: 'Photos', parent_id: null },
    { id: 3, name: 'Projects', parent_id: null }
  ]

  const mockSubFolders: FolderTreeNode[] = [
    { id: 10, name: 'Work', parent_id: 1 },
    { id: 11, name: 'Personal', parent_id: 1 }
  ]

  const mockSuccessResponse = (data: FolderTreeNode[]) => ({
    code: 0,
    data,
    message: 'success'
  })

  const mockErrorResponse = (message: string) => ({
    code: 1,
    data: [],
    message
  })

  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.getItem = vi.fn(() => null)
  })

  describe('初始状态', () => {
    it('folders 应该初始化为空数组', () => {
      const { folders } = useFolderTree()
      expect(folders.value).toEqual([])
    })

    it('expandedFolders 应该初始化为空集合', () => {
      const { expandedFolders } = useFolderTree()
      expect(expandedFolders.value.size).toBe(0)
    })

    it('subFoldersCache 应该初始化为空 Map', () => {
      const { subFoldersCache } = useFolderTree()
      expect(subFoldersCache.value.size).toBe(0)
    })

    it('error 应该初始化为 null', () => {
      const { error } = useFolderTree()
      expect(error.value).toBeNull()
    })

    it('loadingFolders 应该初始化为空集合', () => {
      const { loadingFolders } = useFolderTree()
      expect(loadingFolders.value.size).toBe(0)
    })
  })

  describe('isExpanded', () => {
    it('未展开的文件夹应该返回 false', () => {
      const { isExpanded } = useFolderTree()
      expect(isExpanded(1)).toBe(false)
    })
  })

  describe('getSubFolders', () => {
    it('未缓存的父节点应该返回空数组', () => {
      const { getSubFolders } = useFolderTree()
      const result = getSubFolders(1)
      expect(result).toEqual([])
    })
  })

  describe('reset', () => {
    it('应该重置所有状态到初始值', async () => {
      // Mock successful load first
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockSuccessResponse(mockRootFolders))
      })

      const tree = useFolderTree()
      await tree.loadRootFolders()
      await tree.loadSubFolders(1)

      tree.reset()

      expect(tree.folders.value).toEqual([])
      expect(tree.expandedFolders.value.size).toBe(0)
      expect(tree.subFoldersCache.value.size).toBe(0)
      expect(tree.error.value).toBeNull()
      expect(tree.loadingFolders.value.size).toBe(0)
    })
  })

  describe('loadRootFolders', () => {
    it('成功时应该设置根文件夹列表', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockSuccessResponse(mockRootFolders))
      })
      global.fetch = mockFetch

      const { folders, loadRootFolders, error } = useFolderTree()
      await loadRootFolders()

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/folders/tree'),
        expect.objectContaining({
          signal: expect.any(AbortSignal)
        })
      )
      expect(folders.value).toHaveLength(3)
      expect(folders.value[0].name).toBe('Documents')
      expect(error.value).toBeNull()
    })

    it('API 返回错误码时应该设置 error', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockErrorResponse('权限不足'))
      })
      global.fetch = mockFetch

      const { error, loadRootFolders, folders } = useFolderTree()
      await loadRootFolders()

      expect(error.value).toBe('权限不足')
      expect(folders.value).toEqual([])
    })

    it('网络请求失败时应该设置 error', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        json: () => Promise.resolve({ message: 'Network Error' })
      })
      global.fetch = mockFetch

      const { error, loadRootFolders } = useFolderTree()
      await loadRootFolders()

      // useRequest().get() catches the error and returns null
      // so error is set to the fallback message
      expect(error.value).toBe('加载根文件夹失败')
    })
  })

  describe('loadSubFolders', () => {
    it('成功时应该返回 true 并缓存结果', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockSuccessResponse(mockSubFolders))
      })
      global.fetch = mockFetch

      const { loadSubFolders, subFoldersCache, getSubFolders } = useFolderTree()
      const result = await loadSubFolders(1)

      expect(result).toBe(true)
      expect(subFoldersCache.value.has(1)).toBe(true)
      expect(subFoldersCache.value.get(1)).toEqual(mockSubFolders)
      expect(getSubFolders(1)).toEqual(mockSubFolders)
    })

    it('缓存命中时应该直接返回 true 而不发起请求', async () => {
      // First load to populate cache
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockSuccessResponse(mockSubFolders))
      })
      global.fetch = mockFetch

      const { loadSubFolders } = useFolderTree()
      await loadSubFolders(1)
      expect(mockFetch).toHaveBeenCalledTimes(1)

      // Second call should hit cache
      const result = await loadSubFolders(1)
      expect(result).toBe(true)
      expect(mockFetch).toHaveBeenCalledTimes(1) // Still only 1 call
    })

    it('API 返回错误码时应该返回 false', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockErrorResponse('加载失败'))
      })
      global.fetch = mockFetch

      const { loadSubFolders, error } = useFolderTree()
      const result = await loadSubFolders(1)

      expect(result).toBe(false)
      expect(error.value).toBe('加载失败')
    })

    it('网络请求失败时应该返回 false', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        json: () => Promise.resolve({ message: 'Failed' })
      })
      global.fetch = mockFetch

      const { loadSubFolders } = useFolderTree()
      const result = await loadSubFolders(1)

      expect(result).toBe(false)
    })

    it('重复请求同一节点时应该避免重复调用', async () => {
      let resolveFetch: (value: any) => void
      const fetchPromise = new Promise(resolve => {
        resolveFetch = resolve
      })
      const mockFetch = vi.fn().mockImplementation(() => fetchPromise)
      global.fetch = mockFetch

      const { loadSubFolders } = useFolderTree()

      // Start two concurrent requests
      const promise1 = loadSubFolders(1)
      const promise2 = loadSubFolders(1)

      // Resolve the fetch
      resolveFetch!({
        ok: true,
        json: () => Promise.resolve(mockSuccessResponse(mockSubFolders))
      })

      const [result1, result2] = await Promise.all([promise1, promise2])

      // Only one fetch should have been made
      expect(mockFetch).toHaveBeenCalledTimes(1)
      expect(result1).toBe(true)
      expect(result2).toBe(false) // Second call hit loading guard
    })
  })

  describe('toggleFolder', () => {
    it('折叠状态的文件夹应该展开并加载子文件夹', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockSuccessResponse(mockSubFolders))
      })
      global.fetch = mockFetch

      const { toggleFolder, isExpanded, getSubFolders } = useFolderTree()
      await toggleFolder(1)

      expect(isExpanded(1)).toBe(true)
      expect(getSubFolders(1)).toEqual(mockSubFolders)
    })

    it('展开状态的文件夹应该收起但不清除缓存', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockSuccessResponse(mockSubFolders))
      })
      global.fetch = mockFetch

      const { toggleFolder, isExpanded, getSubFolders } = useFolderTree()

      // 先展开
      await toggleFolder(1)
      expect(isExpanded(1)).toBe(true)

      // 再收起
      await toggleFolder(1)
      expect(isExpanded(1)).toBe(false)

      // 缓存应该保留
      expect(getSubFolders(1)).toEqual(mockSubFolders)
    })

    it('展开失败时不应该添加展开状态', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        json: () => Promise.resolve({ message: 'Load failed' })
      })
      global.fetch = mockFetch

      const { toggleFolder, isExpanded } = useFolderTree()
      await toggleFolder(1)

      expect(isExpanded(1)).toBe(false)
    })

    it('API 返回错误码时不应该添加展开状态', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockErrorResponse('Folder not found'))
      })
      global.fetch = mockFetch

      const { toggleFolder, isExpanded, error } = useFolderTree()
      await toggleFolder(1)

      expect(isExpanded(1)).toBe(false)
      expect(error.value).toBe('Folder not found')
    })
  })

  describe('readonly 状态保护', () => {
    it('folders 应该是 readonly，设置操作会触发 Vue 警告', () => {
      const { folders } = useFolderTree()
      // Vue readonly wraps prevent direct assignment (warns but doesn't throw in test)
      // We verify the behavior by checking the value wasn't changed
      const originalValue = folders.value
      ;(folders as any).value = [{ id: 999, name: 'hacked' }]
      // The value should remain unchanged due to readonly
      expect(folders.value).toBe(originalValue)
    })

    it('subFoldersCache 应该是 readonly，不能直接替换', () => {
      const { subFoldersCache } = useFolderTree()
      const originalValue = subFoldersCache.value
      ;(subFoldersCache as any).value = new Map()
      expect(subFoldersCache.value).toBe(originalValue)
    })
  })
})
