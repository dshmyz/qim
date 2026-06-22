import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import GroupDetail from '@/components/shared/GroupDetail.vue'
import type { Conversation } from '@/types'

// mock 外部依赖
vi.mock('@/composables/useServerUrl', () => ({
  getStoredServerUrl: () => 'http://localhost:8080',
}))

vi.mock('@/utils/user', () => ({
  getCurrentUser: () => ({ id: '1', name: '群主' }),
}))

vi.mock('@/utils/qmessage', () => ({
  default: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('@/utils/logger', () => ({
  logger: { log: vi.fn(), error: vi.fn() },
}))

vi.mock('@/utils/avatar', () => ({
  generateAvatar: () => 'data:image/svg+xml;base64,mock',
  getAvatarUrl: () => 'data:image/svg+xml;base64,mock',
  isAbsoluteUrl: (url: string) => url.startsWith('http'),
}))

const mockGroup = {
  id: '3',
  type: 'group',
  name: '测试群',
  invite_permission: 'owner_admin',
  members: [
    { id: '1', name: '群主', role: 'owner' },
    { id: '2', name: '成员', role: 'member' },
  ],
} as unknown as Conversation

describe('GroupDetail - API 路由修正', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    global.fetch = vi.fn()
  })

  it('更新邀请权限时应调用 PUT /api/v1/groups/:id（而非 /conversations/:id）', async () => {
    ;(global.fetch as any).mockResolvedValueOnce({ ok: true })

    const wrapper = mount(GroupDetail, {
      props: { group: mockGroup },
      global: {
        stubs: ['Avatar'],
      },
    })

    const select = wrapper.find('.permission-select')
    await select.setValue('all')
    await nextTick()

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [url, options] = (global.fetch as any).mock.calls[0]
    expect(url).toBe('http://localhost:8080/api/v1/groups/3')
    expect(options.method).toBe('PUT')
    expect(JSON.parse(options.body)).toEqual({ invite_permission: 'all' })
  })

  it('更新群头像时应调用 PUT /api/v1/groups/:id（而非 /conversations/:id）', async () => {
    ;(global.fetch as any).mockResolvedValueOnce({ ok: true })

    const wrapper = mount(GroupDetail, {
      props: { group: mockGroup },
      global: {
        stubs: ['Avatar'],
      },
    })

    // mock FileReader 为构造函数（class 形式，支持 new 调用）
    class MockFileReader {
      onload: ((e: any) => void) | null = null
      readAsDataURL = vi.fn(() => {
        setTimeout(() => {
          if (this.onload) {
            this.onload({ target: { result: 'data:image/png;base64,mock' } })
          }
        }, 0)
      })
    }
    vi.stubGlobal('FileReader', MockFileReader)

    const mockFile = new File(['mock-image'], 'avatar.png', { type: 'image/png' })
    const fileInput = wrapper.find('input[type="file"]')
    Object.defineProperty(fileInput.element, 'files', {
      value: [mockFile],
      writable: false,
    })

    await fileInput.trigger('change')
    await nextTick()
    // 等待 FileReader 异步回调
    await new Promise((resolve) => setTimeout(resolve, 10))

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [url, options] = (global.fetch as any).mock.calls[0]
    expect(url).toBe('http://localhost:8080/api/v1/groups/3')
    expect(options.method).toBe('PUT')
  })
})
