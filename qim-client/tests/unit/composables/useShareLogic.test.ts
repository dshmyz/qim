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
    // clearAllMocks 不清未消费的 mockResolvedValueOnce 队列，须用 resetAllMocks
    vi.resetAllMocks()
    createShareLogic().invalidateShareCache()
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

  it('keeps usernames on loaded share users for account search', async () => {
    ;(request as any)
      .mockResolvedValueOnce({
        code: 0,
        data: {
          departments: [
            {
              name: '研发部',
              employees: [{ id: 2, username: 'alice', nickname: 'Alice' }],
            },
          ],
        },
      })
      .mockResolvedValueOnce({
        code: 0,
        data: [],
      })

    const { loadShareUsersAndGroups, shareUsers } = createShareLogic()
    await loadShareUsersAndGroups()

    expect(shareUsers.value[0]).toMatchObject({
      id: '2',
      name: 'Alice',
      username: 'alice',
      department: '研发部',
    })
  })
})

describe('useShareLogic - message forwarding', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    // clearAllMocks 不清未消费的 mockResolvedValueOnce 队列，须用 resetAllMocks
    vi.resetAllMocks()
    createShareLogic().invalidateShareCache()
  })

  it('sends one merged forward message for multiple source messages', async () => {
    ;(request as any)
      .mockResolvedValueOnce({ code: 0, data: { id: 10 } })
      .mockResolvedValueOnce({ code: 0, data: { id: 99 } })
    const logic = useShareLogic(ref([
      { id: '1', type: 'text', content: '第一条', sender: { name: '甲' }, timestamp: 1 },
      { id: '2', type: 'text', content: '第二条', sender: { name: '乙' }, timestamp: 2 },
    ]), ref('message'), ref([]), ref([]), ref([]), ref(null), vi.fn(), vi.fn(), vi.fn())

    await logic.handleShareConfirm({ users: ['2'], groups: [] })

    expect(request).toHaveBeenLastCalledWith('/api/v1/conversations/10/messages', expect.objectContaining({
      body: expect.stringContaining('"type":"merged_forward"'),
    }))
  })

  it('uses the source conversation title when creating a merged forward record', async () => {
    ;(request as any)
      .mockResolvedValueOnce({ code: 0, data: { id: 10 } })
      .mockResolvedValueOnce({ code: 0, data: { id: 99 } })
    const logic = useShareLogic(ref({
      messages: [
        { id: '1', type: 'text', content: '第一条', sender: { name: '甲' }, timestamp: 1 },
        { id: '2', type: 'text', content: '第二条', sender: { name: '乙' }, timestamp: 2 },
      ],
      sourceTitle: '来自「产品讨论群」的聊天记录',
    }), ref('message'), ref([]), ref([]), ref([]), ref(null), vi.fn(), vi.fn(), vi.fn())

    await logic.handleShareConfirm({ users: ['2'], groups: [] })

    const [, requestOptions] = (request as any).mock.calls.at(-1)
    expect(JSON.parse(JSON.parse(requestOptions.body).content).title).toBe('来自「产品讨论群」的聊天记录')
  })

  it('preserves a single merged forward message instead of degrading it to a share card', async () => {
    const content = JSON.stringify({
      version: 1,
      title: '聊天记录',
      messages: [{ id: 'nested-1', type: 'text', content: '原消息', senderName: '甲', timestamp: 1 }],
    })
    ;(request as any)
      .mockResolvedValueOnce({ code: 0, data: { id: 10 } })
      .mockResolvedValueOnce({ code: 0, data: { id: 99 } })
    const logic = useShareLogic(ref({
      id: 'record-1', type: 'merged_forward', content, sender: { name: '甲' }, timestamp: 1,
    }), ref('message'), ref([]), ref([]), ref([]), ref(null), vi.fn(), vi.fn(), vi.fn())

    await logic.handleShareConfirm({ users: ['2'], groups: [] })

    const [, requestOptions] = (request as any).mock.calls.at(-1)
    expect(JSON.parse(requestOptions.body)).toMatchObject({ type: 'merged_forward', content })
  })

  it('keeps merged forward records as nested record items during multi-message forwarding', async () => {
    const nestedContent = JSON.stringify({
      version: 1,
      title: '聊天记录',
      messages: [{ id: 'nested-1', type: 'text', content: '原消息', senderName: '甲', timestamp: 1 }],
    })
    ;(request as any)
      .mockResolvedValueOnce({ code: 0, data: { id: 10 } })
      .mockResolvedValueOnce({ code: 0, data: { id: 99 } })
    const logic = useShareLogic(ref([
      { id: 'text-1', type: 'text', content: '外层消息', sender: { name: '乙' }, timestamp: 2 },
      { id: 'record-1', type: 'merged_forward', content: nestedContent, sender: { name: '甲' }, timestamp: 1 },
    ]), ref('message'), ref([]), ref([]), ref([]), ref(null), vi.fn(), vi.fn(), vi.fn())

    await logic.handleShareConfirm({ users: ['2'], groups: [] })

    const [, requestOptions] = (request as any).mock.calls.at(-1)
    const sentMessage = JSON.parse(requestOptions.body)
    const payload = JSON.parse(sentMessage.content)
    expect(payload.messages).toContainEqual(expect.objectContaining({
      type: 'merged_forward',
      content: nestedContent,
    }))
  })

  it('sends a single markdown message to a group as the legacy share payload', async () => {
    ;(request as any).mockResolvedValueOnce({ code: 0, data: { id: 99 } })
    const logic = useShareLogic(ref({
      id: '1', type: 'markdown', content: '# AI response', sender: { name: '甲' }, timestamp: 1,
    }), ref('message'), ref([]), ref([]), ref([]), ref(null), vi.fn(), vi.fn(), vi.fn())

    await logic.handleShareConfirm({ users: [], groups: ['20'] })

    expect(request).toHaveBeenLastCalledWith('/api/v1/conversations/20/messages', expect.objectContaining({
      body: expect.stringContaining('"type":"share"'),
    }))
  })
})
