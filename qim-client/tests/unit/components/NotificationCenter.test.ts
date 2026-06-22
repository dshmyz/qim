import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { flushPromises } from '@vue/test-utils'
import NotificationCenter from '@/components/notification/NotificationCenter.vue'

// mock axios
vi.mock('axios', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    delete: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
  },
}))

// mock useServerUrl
vi.mock('@/composables/useServerUrl', () => ({
  useServerUrl: () => ({ serverUrl: { value: 'http://localhost:8080' } }),
  getStoredServerUrl: () => 'http://localhost:8080',
}))

import axios from 'axios'

const rawNotification = {
  id: 1,
  title: '加群申请',
  content: '用户申请加入群聊',
  type: 'group_invitation',
  action_type: 'approve_reject',
  action_payload: JSON.stringify({ conversation_id: 5, user_id: 10 }),
  handled: false,
  created_at: Date.now(),
}

describe('NotificationCenter - 加群审批 API 路由修正', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.getItem = vi.fn(() => 'test-token')
    ;(axios.get as any).mockResolvedValue({ data: { code: 0, data: [rawNotification] } })
    ;(axios.post as any).mockResolvedValue({ data: { code: 0 } })
    ;(axios.delete as any).mockResolvedValue({ data: { code: 0 } })
  })

  it('批准加群时应调用 POST /api/v1/groups/:id/members（而非 /conversations/:id/members）', async () => {
    const wrapper = mount(NotificationCenter, {
      props: { show: true, position: { x: 0, y: 0 } },
    })

    // 手动触发通知加载（onMounted 未自动调用 loadNotifications）
    await (wrapper.vm as any).loadNotifications()
    await flushPromises()

    // 点击"同意"按钮（primary 样式）
    const approveBtn = wrapper.find('.action-btn.primary')
    expect(approveBtn.exists()).toBe(true)
    await approveBtn.trigger('click')
    await flushPromises()

    expect(axios.post).toHaveBeenCalledTimes(1)
    const postUrl = (axios.post as any).mock.calls[0][0]
    expect(postUrl).toBe('http://localhost:8080/api/v1/groups/5/members')
  })

  it('拒绝加群时应调用 DELETE /api/v1/groups/:id/join-requests/:user_id（而非 /conversations/:id/...）', async () => {
    const wrapper = mount(NotificationCenter, {
      props: { show: true, position: { x: 0, y: 0 } },
    })

    await (wrapper.vm as any).loadNotifications()
    await flushPromises()

    // 点击"拒绝"按钮（secondary 样式）
    const rejectBtn = wrapper.find('.action-btn.secondary')
    expect(rejectBtn.exists()).toBe(true)
    await rejectBtn.trigger('click')
    await flushPromises()

    expect(axios.delete).toHaveBeenCalledTimes(1)
    const deleteUrl = (axios.delete as any).mock.calls[0][0]
    expect(deleteUrl).toBe('http://localhost:8080/api/v1/groups/5/join-requests/10')
  })
})
