import { mount } from '@vue/test-utils'
import { ref, computed } from 'vue'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import ScreenShareSimple from '@/components/shared/ScreenShareSimple.vue'

// Mock useScreenShareNew — 返回可控的 refs
const mockSessionState = ref('idle')
const mockLocalStream = ref<MediaStream | null>(null)
const mockRemoteStream = ref<MediaStream | null>(null)
const mockParticipants = ref<any[]>([])
const mockIsPaused = ref(false)

vi.mock('@/composables/useScreenShareNew', () => ({
  useScreenShareNew: () => ({
    sessionState: mockSessionState,
    localStream: mockLocalStream,
    remoteStream: mockRemoteStream,
    participants: mockParticipants,
    isPaused: mockIsPaused,
    sendRequest: vi.fn(),
    startConnection: vi.fn(),
    startConnectionWithStream: vi.fn(),
    acceptShare: vi.fn(),
    acceptRequest: vi.fn(),
    rejectRequest: vi.fn(),
    stop: vi.fn(),
    togglePause: vi.fn(),
    getScreenSources: vi.fn().mockResolvedValue([]),
    switchSource: vi.fn(),
  }),
}))

vi.mock('@/utils/qmessage', () => ({
  default: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), info: vi.fn() },
}))

const mountComponent = (props: Partial<any> = {}) => {
  return mount(ScreenShareSimple, {
    props: {
      receiverId: 2,
      conversationId: 1,
      ...props,
    },
    global: {
      stubs: {
        Teleport: { template: '<div><slot /></div>' },
      },
    },
  })
}

beforeEach(() => {
  mockSessionState.value = 'idle'
  mockLocalStream.value = null
  mockRemoteStream.value = null
  mockParticipants.value = []
  mockIsPaused.value = false
})

describe('ScreenShareSimple — 发起方最小化预览', () => {
  it('发起方共享开始时应自动进入最小化', async () => {
    // 先以 idle 状态挂载
    const wrapper = mountComponent()
    await wrapper.vm.$nextTick()

    // 模拟发起方：participants 里有 receiver
    mockParticipants.value = [{ role: 'receiver' }]
    // 模拟有 localStream
    mockLocalStream.value = {} as MediaStream
    // sessionState 变为 active（触发 watch）
    mockSessionState.value = 'active'
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    // 应自动最小化
    expect(wrapper.find('.minimized-content').exists()).toBe(true)
    // 不应显示全尺寸 video-container
    expect(wrapper.find('.video-container').exists()).toBe(false)
  })

  it('发起方最小化时不应显示全屏按钮', async () => {
    mockParticipants.value = [{ role: 'receiver' }]
    mockLocalStream.value = {} as MediaStream
    mockSessionState.value = 'active'

    const wrapper = mountComponent()
    await wrapper.vm.$nextTick()

    // 最小化状态下没有全屏按钮
    const fullscreenBtn = wrapper.find('[title="全屏"]')
    expect(fullscreenBtn.exists()).toBe(false)
  })

  it('发起方最小化时不应显示展开按钮', async () => {
    mockParticipants.value = [{ role: 'receiver' }]
    mockLocalStream.value = {} as MediaStream
    mockSessionState.value = 'active'

    const wrapper = mountComponent()
    await wrapper.vm.$nextTick()

    // 不应有"展开"按钮
    const expandBtn = wrapper.find('.expand-btn')
    expect(expandBtn.exists()).toBe(false)
  })

  it('接收方不应自动最小化', async () => {
    // 接收方：participants 里没有 receiver（自己是 receiver）
    mockParticipants.value = []
    mockRemoteStream.value = {} as MediaStream
    mockSessionState.value = 'active'

    const wrapper = mountComponent()
    await wrapper.vm.$nextTick()

    // 接收方不应自动最小化
    expect(wrapper.find('.video-container').exists()).toBe(true)
    expect(wrapper.find('.minimized-content').exists()).toBe(false)
  })

  it('接收方应有全屏按钮', async () => {
    mockParticipants.value = []
    mockRemoteStream.value = {} as MediaStream
    mockSessionState.value = 'active'

    const wrapper = mountComponent()
    await wrapper.vm.$nextTick()

    const fullscreenBtn = wrapper.find('[title="全屏"]')
    expect(fullscreenBtn.exists()).toBe(true)
  })

  it('发起方最小化时仍显示停止按钮', async () => {
    const wrapper = mountComponent()
    await wrapper.vm.$nextTick()

    mockParticipants.value = [{ role: 'receiver' }]
    mockLocalStream.value = {} as MediaStream
    mockSessionState.value = 'active'
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    // 最小化状态下应有停止按钮
    const stopBtn = wrapper.find('.minimized-actions .close-btn')
    expect(stopBtn.exists()).toBe(true)
  })
})
