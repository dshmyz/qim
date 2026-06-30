import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('@/composables/useRequest', () => ({
  request: vi.fn(),
  useRequest: () => ({ serverUrl: { value: 'http://localhost:8080' } }),
}))

vi.mock('@/utils/qmessage', () => ({
  default: { success: vi.fn(), error: vi.fn() },
}))

vi.mock('@/utils/qmessagebox', () => ({
  default: { confirm: vi.fn().mockResolvedValue(true) },
}))

vi.mock('@/utils/avatar', () => ({
  isAbsoluteUrl: (url: string) => url.startsWith('http'),
}))

import { useGroup } from '@/composables/useGroup'
import { request } from '@/composables/useRequest'

const setWindowSize = (width: number, height: number) => {
  Object.defineProperty(window, 'innerWidth', { configurable: true, value: width })
  Object.defineProperty(window, 'innerHeight', { configurable: true, value: height })
}

describe('useGroup - group context menu', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('keeps the group context menu inside the bottom of the viewport', () => {
    setWindowSize(1200, 800)
    const { showGroupContextMenu, groupContextMenuPosition } = useGroup()

    showGroupContextMenu({
      preventDefault: vi.fn(),
      clientX: 200,
      clientY: 750,
    } as unknown as MouseEvent, { id: '5', name: '测试群', type: 'group' })

    expect(groupContextMenuPosition.value.y).toBeLessThanOrEqual(590)
  })
})

describe('useGroup - loadGroupMembers 改用 conversations 接口', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('应调用 GET /api/v1/conversations/:id（而非 /groups/:id/members）并取 members 字段', async () => {
    const mockMembers = [
      { id: '1', name: '群主', role: 'owner' },
      { id: '2', name: '成员', role: 'member' },
    ]
    ;(request as any).mockResolvedValueOnce({
      code: 0,
      data: { id: '5', type: 'group', name: '测试群', members: mockMembers },
    })

    const { loadGroupMembers, groupMembers } = useGroup()
    await loadGroupMembers('5')

    expect(request).toHaveBeenCalledWith('/api/v1/conversations/5')
    expect(groupMembers.value).toEqual(mockMembers)
  })

  it('过滤已删除和禁用的成员', async () => {
    ;(request as any).mockResolvedValueOnce({
      code: 0,
      data: {
        id: '5',
        type: 'group',
        name: '测试群',
        members: [
          { id: '1', name: '正常成员', role: 'member' },
          { id: '2', name: '禁用成员', role: 'member', disabled: true },
          { id: '3', name: '停用成员', role: 'member', is_disabled: true },
          { id: '4', name: '状态禁用成员', role: 'member', status: 'disabled' },
          { id: '5', name: '删除成员', role: 'member', deletedAt: 1710000000 },
          { id: '6', name: '软删成员', role: 'member', deleted_at: '2026-06-30T00:00:00Z' },
          { id: '7', name: '已删成员', role: 'member', is_deleted: true },
        ],
      },
    })

    const { loadGroupMembers, groupMembers } = useGroup()
    await loadGroupMembers('5')

    expect(groupMembers.value).toEqual([
      { id: '1', name: '正常成员', role: 'member' },
    ])
  })

  it('当 conversation 无 members 字段时应设置空数组', async () => {
    ;(request as any).mockResolvedValueOnce({
      code: 0,
      data: { id: '5', type: 'group', name: '测试群' },
    })

    const { loadGroupMembers, groupMembers } = useGroup()
    await loadGroupMembers('5')

    expect(request).toHaveBeenCalledWith('/api/v1/conversations/5')
    expect(groupMembers.value).toEqual([])
  })
})
