import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useChannelStore } from '@/stores/channel'
import type { Channel } from '@/types'

// 存储 space（本 store 只用到 localStorage 的 viewMode），并模拟 request 返回频道列表
const localStorageStore: Record<string, string> = {}

vi.stubGlobal('localStorage', {
  getItem: (key: string) => localStorageStore[key] || null,
  setItem: (key: string, value: string) => { localStorageStore[key] = value },
  removeItem: (key: string) => { delete localStorageStore[key] },
  clear: () => { Object.keys(localStorageStore).forEach(k => delete localStorageStore[k]) },
  get length() { return Object.keys(localStorageStore).length },
  key: (index: number) => Object.keys(localStorageStore)[index] || null,
})

const channels: Channel[] = [
  { id: 2, name: '系统频道', is_subscribed: true, creator_id: 2 } as Channel,
  { id: 3, name: '菜单', is_subscribed: true, creator_id: 4 } as Channel,
]

// 把 useRequest 的 request 替换为可控 mock：GET /channels 返回列表，其余返回空
const requestMock = vi.fn()
vi.mock('@/composables/useRequest', () => ({
  getToken: () => 'test-token',
  onUnauthorized: () => {},
  request: (...args: any[]) => requestMock(...args),
  updateServerUrl: () => {},
  useRequest: () => ({ request: requestMock }),
}))
vi.mock('@/utils/qmessage', () => ({ default: { success: () => {}, error: () => {} } }))
vi.mock('@/utils/user', () => ({ getCurrentUser: () => null }))

describe('useChannelStore - 频道 id 类型一致性（通知中心深链定位的根因）', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    // GET /api/v1/channels 返回频道列表（后端 id 为 number）
    requestMock.mockImplementation((url: string) => {
      if (url === '/api/v1/channels') {
        return Promise.resolve({ code: 0, data: channels })
      }
      if (url === '/api/v1/channels/2/messages') {
        return Promise.resolve({ code: 0, data: [] })
      }
      return Promise.resolve({ code: 0, data: null })
    })
  })

  it('selectChannel 传字符串 id（如通知导航 "2"）能正确选中数字 id 的频道', async () => {
    const store = useChannelStore()
    // 通知导航把 channel_id 转成字符串传进来（与 resolveNotificationNav 的 String() 一致）
    await store.selectChannel('2' as any, '5' as any)

    expect(store.selectedChannelId).toBe('2')
    // 关键回归点：此前 `2 === "2"` 恒 false，selectedChannel 一直是 null，频道详情空白
    expect(store.selectedChannel).not.toBeNull()
    expect(store.selectedChannel!.id).toBe(2)
    expect(store.selectedChannel!.name).toBe('系统频道')
  })

  it('selectChannel 在频道列表尚未加载时先拉取再选中（不依赖 ChannelSidebar 先挂载）', async () => {
    const store = useChannelStore()
    // 未加载任何频道（channels 为空）——模拟首次启动即点通知
    expect(store.channels.length).toBe(0)
    await store.selectChannel('3' as any)

    expect(store.channels.length).toBeGreaterThan(0)
    expect(store.selectedChannelId).toBe('3')
    expect(store.selectedChannel!.id).toBe(3)
  })
})
