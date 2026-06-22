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
