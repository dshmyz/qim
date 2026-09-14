import { mount } from '@vue/test-utils'
import { ref, computed } from 'vue'
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import ScreenShareSimple from '@/components/shared/ScreenShareSimple.vue'

// Mock useScreenShareNew — 返回可控的 refs 与稳定 mock 函数（便于断言调用）
const mockSessionState = ref('idle')
const mockLocalStream = ref<MediaStream | null>(null)
const mockRemoteStream = ref<MediaStream | null>(null)
const mockParticipants = ref<any[]>([])
const mockIsPaused = ref(false)
const mockShareMode = ref<'broadcast' | 'view'>('broadcast')
const mockPendingTargetUserId = ref<number | null>(null)

const mockSendRequest = vi.fn()
const mockSendViewRequest = vi.fn()
const mockStartConnection = vi.fn()
const mockStartConnectionWithStream = vi.fn()
const mockAcceptShare = vi.fn()
const mockAcceptRequest = vi.fn()
const mockRejectRequest = vi.fn()
const mockRejectShare = vi.fn()
const mockStop = vi.fn()
const mockStopSharing = vi.fn()
const mockTogglePause = vi.fn()
const mockSetShareMode = vi.fn((mode: 'broadcast' | 'view') => {
  mockShareMode.value = mode
})

vi.mock('@/composables/useScreenShareNew', () => ({
  useScreenShareNew: () => ({
    sessionState: mockSessionState,
    localStream: mockLocalStream,
    remoteStream: mockRemoteStream,
    participants: mockParticipants,
    isPaused: mockIsPaused,
    shareMode: mockShareMode,
    pendingTargetUserId: mockPendingTargetUserId,
    sendRequest: mockSendRequest,
    sendViewRequest: mockSendViewRequest,
    startConnection: mockStartConnection,
    startConnectionWithStream: mockStartConnectionWithStream,
    acceptShare: mockAcceptShare,
    acceptRequest: mockAcceptRequest,
    rejectRequest: mockRejectRequest,
    rejectShare: mockRejectShare,
    stop: mockStop,
    stopSharing: mockStopSharing,
    togglePause: mockTogglePause,
    setShareMode: mockSetShareMode,
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
  mockShareMode.value = 'broadcast'
  mockPendingTargetUserId.value = null
  vi.clearAllMocks()
  mockSetShareMode.mockImplementation((mode: 'broadcast' | 'view') => {
    mockShareMode.value = mode
  })
})

describe('ScreenShareSimple — 发起方最小化预览', () => {
  it('发起方共享开始时应自动进入最小化', async () => {
    // 先以 idle 状态挂载
    const wrapper = mountComponent()
    await wrapper.vm.$nextTick()

    // 模拟发起方：participants 里有 receiver
    mockParticipants.value = [{ role: 'receiver' }]
    // 模拟有 localStream
    mockLocalStream.value = new MediaStream() as MediaStream
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
    mockLocalStream.value = new MediaStream() as MediaStream
    mockSessionState.value = 'active'

    const wrapper = mountComponent()
    await wrapper.vm.$nextTick()

    // 最小化状态下没有全屏按钮
    const fullscreenBtn = wrapper.find('[title="全屏"]')
    expect(fullscreenBtn.exists()).toBe(false)
  })

  it('发起方最小化时不应显示展开按钮', async () => {
    mockParticipants.value = [{ role: 'receiver' }]
    mockLocalStream.value = new MediaStream() as MediaStream
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
    mockRemoteStream.value = new MediaStream() as MediaStream
    mockSessionState.value = 'active'

    const wrapper = mountComponent()
    await wrapper.vm.$nextTick()

    // 接收方不应自动最小化
    expect(wrapper.find('.video-container').exists()).toBe(true)
    expect(wrapper.find('.minimized-content').exists()).toBe(false)
  })

  it('接收方应有全屏按钮', async () => {
    mockParticipants.value = []
    mockRemoteStream.value = new MediaStream() as MediaStream
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
    mockLocalStream.value = new MediaStream() as MediaStream
    mockSessionState.value = 'active'
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    // 最小化状态下应有停止按钮
    const stopBtn = wrapper.find('.minimized-actions .close-btn')
    expect(stopBtn.exists()).toBe(true)
  })
})

/* ---- view 共享者推流中切源/停止的守卫 ----
 * 回归：view 被查看方已选源推流（savedStream 已建立）后，点「切换窗口」会重新弹源选择器；
 * 此时取消选源或点关闭，旧守卫（viewSharerTargetId && incomingRequestInfo）仍会误发
 * screen-share.response rejected，把还看着的查看方踢掉。正确语义：只有尚未推流
 * （savedStream 为空）的放弃才需要回发拒绝。 */
describe('ScreenShareSimple — view 推流中取消/停止不误发 rejected', () => {
  const fakeSources = [{ id: 'screen:0', name: '整个屏幕', thumbnail: null }]
  let sourceCallbacks: Array<(_e: unknown, sources: any[]) => void>
  // happy-dom 会校验 srcObject 类型，优先用其全局 MediaStream 构造真实例；
  // 其 MediaStream 未实现 getTracks，缺了再补
  const fakeStream = (() => {
    if (typeof MediaStream !== 'undefined') {
      const s: any = new MediaStream()
      if (typeof s.getTracks !== 'function') s.getTracks = () => []
      return s
    }
    return { getTracks: () => [] }
  })() as unknown as MediaStream

  const flushAsync = async () => {
    // 等待 getScreenSources 的 setTimeout 回调与后续微任务全部落地
    await new Promise(resolve => setTimeout(resolve, 0))
    await new Promise(resolve => setTimeout(resolve, 0))
  }

  const installStubs = () => {
    sourceCallbacks = []
    ;(window as any).electron = {
      ipcRenderer: {
        send: vi.fn((channel: string) => {
          if (channel === 'start-screen-share') {
            setTimeout(() => {
              const cbs = sourceCallbacks
              sourceCallbacks = []
              cbs.forEach(cb => cb(null, fakeSources))
            }, 0)
          }
        }),
        once: vi.fn((channel: string, cb: (_e: unknown, sources: any[]) => void) => {
          if (channel === 'screen-sources') sourceCallbacks.push(cb)
        }),
        removeAllListeners: vi.fn(),
      },
    }
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: { getUserMedia: vi.fn().mockResolvedValue(fakeStream) },
    })
  }

  const uninstallStubs = () => {
    delete (window as any).electron
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: undefined,
    })
  }

  // 完整走通：收到 view 申请 → 接受 → 选源推流 → 进入 active → 点「切换窗口」重新弹源选择器
  const reachSwitchSourcePicker = async () => {
    installStubs()
    const wrapper = mountComponent()
    await wrapper.vm.$nextTick()

    ;(wrapper.vm as any).handleIncomingRequest({
      from_user_id: 5,
      conversation_id: 1,
      from_user_name: '小李',
      mode: 'view',
    })
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.incoming-request').exists()).toBe(true)

    await wrapper.find('.btn-accept').trigger('click')
    await flushAsync()

    // 接受后弹出源选择器，双击第一个源开始推流
    expect(wrapper.find('.source-picker').exists()).toBe(true)
    await wrapper.find('.source-item').trigger('dblclick')
    await flushAsync()

    // 已建立流并进入 active，切换窗口按钮可见
    mockParticipants.value = [{ role: 'receiver' }]
    mockSessionState.value = 'active'
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()
    // active 后发起方会自动最小化，点击悬浮条展开恢复全尺寸面板
    expect(wrapper.find('.minimized-content').exists()).toBe(true)
    await wrapper.find('.minimized-content').trigger('click')
    await wrapper.vm.$nextTick()
    const switchBtn = wrapper.find('[title="切换窗口"]')
    expect(switchBtn.exists()).toBe(true)

    await switchBtn.trigger('click')
    await flushAsync()
    expect(wrapper.find('.source-picker').exists()).toBe(true)

    return wrapper
  }

  beforeEach(() => {
    installStubs()
  })
  afterEach(() => {
    uninstallStubs()
  })

  it('推流中切源后取消选源：不得回发 rejected，共享继续', async () => {
    const wrapper = await reachSwitchSourcePicker()
    expect(mockRejectRequest).not.toHaveBeenCalled()

    await wrapper.find('.source-picker-footer .btn-secondary').trigger('click')
    await wrapper.vm.$nextTick()

    expect(mockRejectRequest).not.toHaveBeenCalled()
    uninstallStubs()
  })

  it('推流中切源后点关闭停止：不得回发 rejected，走正常 stop', async () => {
    const wrapper = await reachSwitchSourcePicker()

    await wrapper.find('.screen-share-header .close-btn').trigger('click')
    await wrapper.vm.$nextTick()

    expect(mockRejectRequest).not.toHaveBeenCalled()
    expect(mockStopSharing).toHaveBeenCalled()
    uninstallStubs()
  })

  it('回归钉死：尚未推流时取消选源仍须回发 rejected（申请方不能干等）', async () => {
    installStubs()
    const wrapper = mountComponent()
    await wrapper.vm.$nextTick()

    ;(wrapper.vm as any).handleIncomingRequest({
      from_user_id: 5,
      conversation_id: 1,
      from_user_name: '小李',
      mode: 'view',
    })
    await wrapper.vm.$nextTick()
    await wrapper.find('.btn-accept').trigger('click')
    await flushAsync()

    // 弹出源选择器后直接取消（未选源、未推流）
    expect(wrapper.find('.source-picker').exists()).toBe(true)
    await wrapper.find('.source-picker-footer .btn-secondary').trigger('click')
    await wrapper.vm.$nextTick()

    expect(mockRejectRequest).toHaveBeenCalledWith(5, 1, 'view')
    uninstallStubs()
  })
})
