import { describe, it, expect, beforeEach, vi } from 'vitest'

// 单例工厂：resetModules 后工厂重跑，模块内 useRequest() 与测试内 useRequest() 拿到同一组 mock
// 注意：deleteFolder 走原始 request（非 safeRequest），后端 400 以 reject 形式抛出
vi.mock('@/composables/useRequest', () => {
  const requestMethods = {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    request: vi.fn()
  }
  return {
    useRequest: () => requestMethods
  }
})

// 动态 import：resetModules 重置模块级单例状态（treeData/selectedFolder 等）
async function createTree() {
  const { useFolderTree } = await import('@/composables/useFolderTree')
  const { useRequest } = await import('@/composables/useRequest')
  const { get, post, put, request } = useRequest()
  return { tree: useFolderTree(), get, post, put, request }
}

describe('useFolderTree - 树状态中枢', () => {
  beforeEach(() => {
    vi.resetModules()
    // resetAllMocks（非 clearAllMocks）：mockResolvedValueOnce 队列必须一并清空，否则上个用例未消费的 Once 会泄漏给后续用例
    vi.resetAllMocks()
  })

  it('loadRootFolders 填充树并保留 has_children（isExpandable 展开依据）', async () => {
    const { tree, get } = await createTree()
    get.mockResolvedValueOnce({
      code: 0,
      data: [
        { id: 1, name: 'A', parent_id: null, has_children: true },
        { id: 2, name: 'B', parent_id: null, has_children: false }
      ]
    })

    await tree.loadRootFolders()

    expect(tree.treeData.value).toHaveLength(2)
    expect(tree.isExpandable(tree.treeData.value[0])).toBe(true)
    expect(tree.isExpandable(tree.treeData.value[1])).toBe(false)
    expect(tree.loadFailed.value).toBe(false)
  })

  it('loadRootFolders 失败置 loadFailed（错误屏依据），CRUD 失败只写 error 文案', async () => {
    const { tree, get, request } = await createTree()
    get.mockRejectedValueOnce(new Error('网络错误'))

    await tree.loadRootFolders()
    expect(tree.loadFailed.value).toBe(true)
    expect(tree.error.value).toBe('网络错误')

    // 重新加载成功后 loadFailed 复位
    get.mockResolvedValueOnce({ code: 0, data: [] })
    await tree.loadRootFolders()
    expect(tree.loadFailed.value).toBe(false)

    // CRUD 失败（如删除非空文件夹 400）：原始 request 抛出后端文案，error 记录、loadFailed 不受影响
    request.mockRejectedValueOnce(new Error('文件夹包含子文件夹，请使用递归删除或先移走子文件夹'))
    const ok = await tree.deleteFolder(1, false)
    expect(ok).toBe(false)
    expect(tree.error.value).toContain('子文件夹')
    expect(tree.loadFailed.value).toBe(false)
  })

  it('loadRootFolders 幂等：进行中直接返回，不重复请求', async () => {
    const { tree, get } = await createTree()
    get.mockImplementation(() =>
      new Promise(resolve => setTimeout(() => resolve({ code: 0, data: [] }), 10))
    )

    const p1 = tree.loadRootFolders()
    const p2 = tree.loadRootFolders()
    await Promise.all([p1, p2])

    expect(get).toHaveBeenCalledTimes(1)
  })

  it('createFolder：父节点已展开 → 重载 children；未展开 → 标记 hasChildren 让箭头出现', async () => {
    const { tree, get, post } = await createTree()
    get.mockResolvedValueOnce({
      code: 0,
      data: [
        { id: 1, name: 'A', parent_id: null, has_children: true },
        { id: 2, name: 'B', parent_id: null, has_children: false }
      ]
    })
    await tree.loadRootFolders()

    // 展开 A：首次懒加载 children（空）
    get.mockResolvedValueOnce({ code: 0, data: [] })
    await tree.toggleExpand(tree.treeData.value[0])
    expect(get).toHaveBeenLastCalledWith('/api/v1/folders/tree', { params: { parent_id: 1 } })

    // 在已展开的 A 下创建 → 重载 children
    post.mockResolvedValueOnce({ code: 0, data: { id: 11, name: 'A1', parent_id: 1 } })
    get.mockResolvedValueOnce({ code: 0, data: [{ id: 11, name: 'A1', parent_id: 1, has_children: false }] })
    const ok = await tree.createFolder('A1', 1)
    expect(ok).toBe(true)
    expect(tree.treeData.value[0].children).toHaveLength(1)
    expect(tree.treeData.value[0].children![0].name).toBe('A1')

    // 在未展开的 B 下创建 → 只标记 hasChildren，不额外请求
    post.mockResolvedValueOnce({ code: 0, data: { id: 21, name: 'B1', parent_id: 2 } })
    const calledBefore = get.mock.calls.length
    const ok2 = await tree.createFolder('B1', 2)
    expect(ok2).toBe(true)
    expect(tree.treeData.value[1].hasChildren).toBe(true)
    expect(get.mock.calls.length).toBe(calledBefore)
  })

  it('createFolder：根目录创建触发 loadRootFolders 刷新根列表', async () => {
    const { tree, get, post } = await createTree()
    get.mockResolvedValueOnce({ code: 0, data: [] })
    await tree.loadRootFolders()

    post.mockResolvedValueOnce({ code: 0, data: { id: 5, name: 'R', parent_id: null } })
    const ok = await tree.createFolder('R', null)
    expect(ok).toBe(true)
    // 根目录创建 → 重新加载根
    expect(tree.treeData.value).toHaveLength(0)
    expect(get).toHaveBeenCalledTimes(2)
  })

  it('renameFolder 更新树内节点名称', async () => {
    const { tree, get, put } = await createTree()
    get.mockResolvedValueOnce({ code: 0, data: [{ id: 1, name: '旧名', parent_id: null }] })
    await tree.loadRootFolders()

    put.mockResolvedValueOnce({ code: 0, data: { id: 1, name: '新名' } })
    const ok = await tree.renameFolder(1, '新名')

    expect(ok).toBe(true)
    expect(put).toHaveBeenCalledWith('/api/v1/folders/1', { name: '新名' })
    expect(tree.treeData.value[0].name).toBe('新名')
  })

  it('deleteFolder 透传 recursive 参数，删除选中节点后清空选中', async () => {
    const { tree, get, request } = await createTree()
    get.mockResolvedValueOnce({ code: 0, data: [{ id: 1, name: 'A', parent_id: null }] })
    await tree.loadRootFolders()
    tree.selectFolder(tree.treeData.value[0])

    // 普通删除
    request.mockResolvedValueOnce({ code: 0, data: null })
    const ok1 = await tree.deleteFolder(1, false)
    expect(ok1).toBe(true)
    expect(request).toHaveBeenCalledWith('/api/v1/folders/1', { method: 'DELETE' })
    expect(tree.treeData.value).toHaveLength(0)
    expect(tree.selectedFolder.value).toBeNull()

    // 递归删除
    get.mockResolvedValueOnce({ code: 0, data: [{ id: 2, name: 'B', parent_id: null }] })
    await tree.loadRootFolders()
    request.mockResolvedValueOnce({ code: 0, data: null })
    await tree.deleteFolder(2, true)
    expect(request).toHaveBeenLastCalledWith('/api/v1/folders/2', { method: 'DELETE', params: { recursive: true } })
  })

  it('folderPath 多级回溯 + 防环终止', async () => {
    const { tree, get } = await createTree()
    get.mockResolvedValueOnce({ code: 0, data: [{ id: 1, name: 'A', parent_id: null, has_children: true }] })
    await tree.loadRootFolders()

    get.mockResolvedValueOnce({ code: 0, data: [{ id: 2, name: 'A1', parent_id: 1, has_children: true }] })
    await tree.toggleExpand(tree.treeData.value[0])

    get.mockResolvedValueOnce({ code: 0, data: [{ id: 3, name: 'A1a', parent_id: 2, has_children: false }] })
    await tree.toggleExpand(tree.findFolderInTree(2)!)

    tree.selectFolder(tree.findFolderInTree(3)!)
    expect(tree.folderPath.value.map(n => n.name)).toEqual(['A', 'A1', 'A1a'])

    // 人为制造环：A 的 parent_id 指向其深层子孙，folderPath 应防环终止而非死循环
    tree.treeData.value[0].parent_id = 3
    const path = tree.folderPath.value
    expect(path.length).toBeGreaterThan(0)
    expect(path.map(n => n.id)).toContain(3)
  })

  it('fetchAllFolders 全量懒加载 + folderOptions 按树序平铺带深度', async () => {
    const { tree, get } = await createTree()
    get.mockResolvedValueOnce({
      code: 0,
      data: [
        { id: 1, name: 'A', parent_id: null, has_children: true },
        { id: 2, name: 'B', parent_id: null, has_children: false }
      ]
    })
    await tree.loadRootFolders()

    // A 有 has_children 且无 children → 拉取一层
    get.mockResolvedValueOnce({
      code: 0,
      data: [{ id: 3, name: 'A1', parent_id: 1, has_children: false }]
    })
    await tree.fetchAllFolders()

    expect(tree.folderOptions.value.map(o => o.folder.name)).toEqual(['A', 'A1', 'B'])
    expect(tree.folderOptions.value.map(o => o.depth)).toEqual([0, 1, 0])
  })

  it('selectRoot 回到根（全部文件）', async () => {
    const { tree, get } = await createTree()
    get.mockResolvedValueOnce({ code: 0, data: [{ id: 1, name: 'A', parent_id: null }] })
    await tree.loadRootFolders()

    tree.selectFolder(tree.treeData.value[0])
    expect(tree.selectedFolder.value?.id).toBe(1)
    tree.selectRoot()
    expect(tree.selectedFolder.value).toBeNull()
  })
})
