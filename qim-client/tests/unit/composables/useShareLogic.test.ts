import { describe, it, expect, beforeEach, vi } from 'vitest'
import { ref } from 'vue'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('@/composables/useRequest', () => ({
  request: vi.fn(),
}))

vi.mock('@/composables/useCurrentUser', () => ({
  useCurrentUser: () => ({ currentUser: { value: { id: '1', name: '当前用户' } } }),
}))

vi.mock('@/composables/useServerUrl', () => ({
  useServerUrl: () => ({ serverUrl: { value: 'http://localhost:8080' } }),
}))

vi.mock('@/utils/logger', () => ({
  logger: { error: vi.fn(), log: vi.fn() },
}))

vi.mock('@/utils/qmessage', () => ({
  default: { error: vi.fn(), success: vi.fn() },
}))

vi.mock('@/utils/avatar', () => ({
  generateAvatar: (name: string) => `avatar:${name}`,
  isAbsoluteUrl: (url: string) => url.startsWith('http'),
  getAvatarUrl: (avatar: string | undefined, name: string) => avatar || `avatar:${name}`,
}))

import { useShareLogic } from '@/composables/useShareLogic'
import { request } from '@/composables/useRequest'

const createShareLogic = () => {
  const shareUsers = ref<any[]>([])
  const shareGroups = ref<any[]>([])

  const logic = useShareLogic(
    ref(null),
    ref('message'),
    shareUsers,
    shareGroups,
    ref([]),
    ref(null),
    vi.fn(),
    vi.fn(),
    vi.fn()
  )

  return { ...logic, shareUsers, shareGroups }
}

describe('useShareLogic - loadShareUsersAndGroups', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('loads group conversations from paginated conversation response data', async () => {
    ;(request as any)
      .mockResolvedValueOnce({
        code: 0,
        data: {
          departments: [
            {
              name: '研发部',
              employees: [{ id: 2, username: 'alice' }],
            },
          ],
        },
      })
      .mockResolvedValueOnce({
        code: 0,
        data: {
          list: [
            { id: 10, type: 'single', name: '单聊' },
            { id: 20, type: 'group', name: '产品群', avatar: '', members: [{ id: 2 }] },
          ],
          total: 2,
          page: 1,
          pageSize: 20,
        },
      })

    const { loadShareUsersAndGroups, shareGroups } = createShareLogic()
    await loadShareUsersAndGroups()

    expect(shareGroups.value).toEqual([
      {
        id: '20',
        name: '产品群',
        avatar: 'avatar:产品群',
        members: [{ id: 2 }],
      },
    ])
  })
})
