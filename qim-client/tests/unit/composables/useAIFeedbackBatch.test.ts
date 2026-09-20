import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ref, nextTick, effectScope, type Ref } from 'vue'
import { useAIFeedbackBatch } from '../../../src/composables/useAIFeedbackBatch'
import { aiFeedbackAPI } from '../../../src/api/ai'

vi.mock('../../../src/api/ai', () => ({
  aiFeedbackAPI: {
    set: vi.fn(),
    get: vi.fn(),
    getBatch: vi.fn(),
  },
}))

const getBatchMock = vi.mocked(aiFeedbackAPI.getBatch)

const aiMsg = (id: number | string) => ({ id, sender: { type: 'bot' }, content: 'hi' })

// onScopeDispose 需要活跃的 effect scope，测试统一在 scope 内创建
const setupBatch = (messages: Ref<{ id: number | string }[]>) => {
  const scope = effectScope()
  let ratings: ReturnType<typeof useAIFeedbackBatch> | null = null
  scope.run(() => {
    ratings = useAIFeedbackBatch(messages)
  })
  return { ratings, stop: () => scope.stop() }
}

describe('useAIFeedbackBatch', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getBatchMock.mockResolvedValue(new Map())
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('挂载去抖 300ms 后合并拉取一次，仅数字 id 的 AI 消息', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'] })
    const messages = ref<any[]>([])
    const { stop } = setupBatch(messages)

    messages.value = [aiMsg(1), aiMsg(2), { id: 3, content: '普通消息' }, aiMsg('stream_x')]
    await nextTick() // 让 watch 先注册新的去抖定时器

    await vi.advanceTimersByTimeAsync(299)
    expect(getBatchMock).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(1)
    expect(getBatchMock).toHaveBeenCalledTimes(1)
    expect(getBatchMock).toHaveBeenCalledWith([1, 2])
    stop()
  })

  it('已查过的 id 不重复拉取（分页/流式追加合并）', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'] })
    const messages = ref<any[]>([])
    const { stop } = setupBatch(messages)

    messages.value = [aiMsg(1)]
    await nextTick()
    await vi.advanceTimersByTimeAsync(300)
    expect(getBatchMock).toHaveBeenCalledTimes(1)

    messages.value = [aiMsg(1), aiMsg(2)]
    await nextTick()
    await vi.advanceTimersByTimeAsync(300)
    expect(getBatchMock).toHaveBeenCalledTimes(2)
    expect(getBatchMock).toHaveBeenLastCalledWith([2])
    stop()
  })

  it('超过 200 条按片并发提交', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'] })
    const messages = ref<any[]>([])
    const { stop } = setupBatch(messages)

    messages.value = Array.from({ length: 201 }, (_, i) => aiMsg(i + 1))
    await nextTick()
    await vi.advanceTimersByTimeAsync(300)

    expect(getBatchMock).toHaveBeenCalledTimes(2)
    expect(getBatchMock.mock.calls[0][0]).toHaveLength(200)
    expect(getBatchMock.mock.calls[1][0]).toHaveLength(1)
    stop()
  })

  it('单片失败只回滚该片 id，允许下次重试', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'] })
    getBatchMock.mockRejectedValueOnce(new Error('boom'))
    const messages = ref<any[]>([])
    const { stop } = setupBatch(messages)

    messages.value = [aiMsg(1), aiMsg(2)]
    await nextTick()
    await vi.advanceTimersByTimeAsync(300)
    expect(getBatchMock).toHaveBeenCalledTimes(1)

    messages.value = [aiMsg(1), aiMsg(2), aiMsg(3)]
    await nextTick()
    await vi.advanceTimersByTimeAsync(300)
    expect(getBatchMock).toHaveBeenCalledTimes(2)
    expect(getBatchMock.mock.calls[1][0]).toContain(1)
    stop()
  })

  it('批量结果合并进 ratings map', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'] })
    getBatchMock.mockResolvedValue(new Map([[1, 1], [2, -1]]))
    const messages = ref<any[]>([])
    const { ratings, stop } = setupBatch(messages)

    messages.value = [aiMsg(1), aiMsg(2)]
    await nextTick()
    await vi.advanceTimersByTimeAsync(300)

    expect(ratings!.value.get(1)).toBe(1)
    expect(ratings!.value.get(2)).toBe(-1)
    stop()
  })
})
