import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { useAutoSave } from '@/composables/useAutoSave'

describe('useAutoSave', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('schedule 后未到 delay 不调用 saveFn', () => {
    const saveFn = vi.fn().mockResolvedValue(true)
    const { schedule, status } = useAutoSave(saveFn, { delay: 2000 })

    schedule(1, { title: 't', content: 'c', tags: [] })

    expect(status.value).toBe('pending')
    expect(saveFn).not.toHaveBeenCalled()
  })

  it('多次 schedule 在 delay 内合并为一次保存（最后一次数据）', async () => {
    const saveFn = vi.fn().mockResolvedValue(true)
    const { schedule, status } = useAutoSave(saveFn, { delay: 2000 })

    schedule(1, { title: 't1', content: 'c1', tags: [] })
    vi.advanceTimersByTime(1000)
    schedule(1, { title: 't2', content: 'c2', tags: ['a'] })
    vi.advanceTimersByTime(1000)
    expect(saveFn).not.toHaveBeenCalled()

    vi.advanceTimersByTime(1000)
    await vi.runAllTicks()

    expect(saveFn).toHaveBeenCalledTimes(1)
    expect(saveFn).toHaveBeenCalledWith(1, { title: 't2', content: 'c2', tags: ['a'] })
    expect(status.value).toBe('saved')
  })

  it('flush 立即触发保存并取消定时器', async () => {
    const saveFn = vi.fn().mockResolvedValue(true)
    const { schedule, flush, status } = useAutoSave(saveFn, { delay: 2000 })

    schedule(1, { title: 't', content: 'c', tags: [] })
    expect(status.value).toBe('pending')

    const result = await flush()

    expect(result).toBe('saved')
    expect(saveFn).toHaveBeenCalledTimes(1)
    expect(saveFn).toHaveBeenCalledWith(1, { title: 't', content: 'c', tags: [] })
    expect(status.value).toBe('saved')

    vi.advanceTimersByTime(5000)
    expect(saveFn).toHaveBeenCalledTimes(1)
  })

  it('flush 无待保存数据时返回 noop', async () => {
    const saveFn = vi.fn().mockResolvedValue(true)
    const { flush, status } = useAutoSave(saveFn, { delay: 2000 })

    const result = await flush()
    expect(result).toBe('noop')
    expect(saveFn).not.toHaveBeenCalled()
    expect(status.value).toBe('idle')
  })

  it('cancel 取消待保存并清空状态', () => {
    const saveFn = vi.fn().mockResolvedValue(true)
    const { schedule, cancel, status } = useAutoSave(saveFn, { delay: 2000 })

    schedule(1, { title: 't', content: 'c', tags: [] })
    expect(status.value).toBe('pending')

    cancel()

    expect(status.value).toBe('idle')
    vi.advanceTimersByTime(5000)
    expect(saveFn).not.toHaveBeenCalled()
  })

  it('saveFn 失败时 status 为 error', async () => {
    const saveFn = vi.fn().mockResolvedValue(false)
    const { schedule, flush, status } = useAutoSave(saveFn, { delay: 2000, retryDelays: [] })

    schedule(1, { title: 't', content: 'c', tags: [] })
    const result = await flush()

    expect(result).toBe('error')
    expect(status.value).toBe('error')
  })

  it('saveFn 抛异常时 status 为 error 且不抛出', async () => {
    const saveFn = vi.fn().mockRejectedValue(new Error('network'))
    const { schedule, flush, status } = useAutoSave(saveFn, { delay: 2000, retryDelays: [] })

    schedule(1, { title: 't', content: 'c', tags: [] })
    await expect(flush()).resolves.not.toThrow()

    expect(status.value).toBe('error')
  })

  it('定时器到期后自动 flush 并切换状态', async () => {
    const saveFn = vi.fn().mockResolvedValue(true)
    const { schedule, status } = useAutoSave(saveFn, { delay: 2000 })

    schedule(1, { title: 't', content: 'c', tags: [] })
    vi.advanceTimersByTime(2000)
    await vi.runAllTicks()

    expect(saveFn).toHaveBeenCalledTimes(1)
    expect(status.value).toBe('saved')
  })

  it('flush 完成后再次 schedule 可正常工作', async () => {
    const saveFn = vi.fn().mockResolvedValue(true)
    const { schedule, flush, status } = useAutoSave(saveFn, { delay: 2000 })

    schedule(1, { title: 't1', content: 'c1', tags: [] })
    await flush()
    expect(status.value).toBe('saved')

    schedule(2, { title: 't2', content: 'c2', tags: [] })
    expect(status.value).toBe('pending')
    await flush()

    expect(saveFn).toHaveBeenCalledTimes(2)
    expect(saveFn).toHaveBeenNthCalledWith(2, 2, { title: 't2', content: 'c2', tags: [] })
  })

  it('保存过程中再次 schedule 不丢失新数据（保存完成后重新进入 pending）', async () => {
    const saveFn = vi.fn().mockResolvedValue(true)
    const { schedule, flush, status } = useAutoSave(saveFn, { delay: 2000 })

    schedule(1, { title: 't1', content: 'c1', tags: [] })
    const flushPromise = flush()
    schedule(1, { title: 't2', content: 'c2', tags: [] })
    await flushPromise

    expect(saveFn).toHaveBeenCalledTimes(1)
    expect(status.value).toBe('pending')
    await flush()
    expect(saveFn).toHaveBeenCalledTimes(2)
    expect(saveFn).toHaveBeenLastCalledWith(1, { title: 't2', content: 'c2', tags: [] })
  })

  it('不同 id 切换时外部需先 flush，否则 schedule 会覆盖待保存数据', async () => {
    const saveFn = vi.fn().mockResolvedValue(true)
    const { schedule, flush } = useAutoSave(saveFn, { delay: 2000 })

    schedule(1, { title: 't1', content: 'c1', tags: [] })
    await flush()
    schedule(2, { title: 't2', content: 'c2', tags: [] })
    await flush()

    expect(saveFn).toHaveBeenCalledTimes(2)
    expect(saveFn).toHaveBeenNthCalledWith(1, 1, { title: 't1', content: 'c1', tags: [] })
    expect(saveFn).toHaveBeenNthCalledWith(2, 2, { title: 't2', content: 'c2', tags: [] })
  })
})

describe('useAutoSave - saved 状态淡出', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('saved 后淡出延迟到期恢复 idle', async () => {
    const saveFn = vi.fn().mockResolvedValue(true)
    const { schedule, flush, status } = useAutoSave(saveFn, {
      delay: 2000,
      savedFadeDelay: 3000,
    })

    schedule(1, { title: 't', content: 'c', tags: [] })
    await flush()
    expect(status.value).toBe('saved')

    vi.advanceTimersByTime(3000)
    expect(status.value).toBe('idle')
  })

  it('saved 淡出期间 schedule 取消淡出定时器', async () => {
    const saveFn = vi.fn().mockResolvedValue(true)
    const { schedule, flush, status } = useAutoSave(saveFn, {
      delay: 5000,
      savedFadeDelay: 3000,
    })

    schedule(1, { title: 't1', content: 'c1', tags: [] })
    await flush()
    expect(status.value).toBe('saved')

    vi.advanceTimersByTime(1000)
    schedule(1, { title: 't2', content: 'c2', tags: [] })
    expect(status.value).toBe('pending')

    // 原淡出定时器（3s）已被取消，不会把 pending 改成 idle
    // delay=5000，advance 4000 不会触发 debounce
    vi.advanceTimersByTime(4000)
    expect(status.value).toBe('pending')
  })
})

describe('useAutoSave - maxWait 强制保存', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('连续 schedule 超过 maxWait 后强制 flush', async () => {
    const saveFn = vi.fn().mockResolvedValue(true)
    const { schedule } = useAutoSave(saveFn, { delay: 3000, maxWait: 5000 })

    schedule(1, { title: 't1', content: 'c1', tags: [] })
    vi.advanceTimersByTime(2000)
    schedule(1, { title: 't2', content: 'c2', tags: [] })
    vi.advanceTimersByTime(2000)
    schedule(1, { title: 't3', content: 'c3', tags: [] })

    // debounce=7000, maxWait=5000, 当前 t=4000
    expect(saveFn).not.toHaveBeenCalled()

    vi.advanceTimersByTime(1000) // t=5000, maxWait 到期
    await vi.runAllTicks()

    expect(saveFn).toHaveBeenCalledTimes(1)
    expect(saveFn).toHaveBeenCalledWith(1, { title: 't3', content: 'c3', tags: [] })
  })

  it('flush 后 maxWait 重新计时', async () => {
    const saveFn = vi.fn().mockResolvedValue(true)
    const { schedule, flush } = useAutoSave(saveFn, { delay: 10000, maxWait: 5000 })

    schedule(1, { title: 't1', content: 'c1', tags: [] })
    await flush()
    expect(saveFn).toHaveBeenCalledTimes(1)

    schedule(1, { title: 't2', content: 'c2', tags: [] })
    vi.advanceTimersByTime(4000) // 不到 maxWait
    expect(saveFn).toHaveBeenCalledTimes(1)

    vi.advanceTimersByTime(1000) // maxWait 到期
    await vi.runAllTicks()
    expect(saveFn).toHaveBeenCalledTimes(2)
    expect(saveFn).toHaveBeenLastCalledWith(1, { title: 't2', content: 'c2', tags: [] })
  })
})

describe('useAutoSave - 失败退避重试', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('失败后自动重试一次并成功', async () => {
    const saveFn = vi.fn()
      .mockResolvedValueOnce(false)
      .mockResolvedValueOnce(true)
    const { schedule, flush, status } = useAutoSave(saveFn, {
      delay: 2000,
      retryDelays: [5000, 10000, 20000],
    })

    schedule(1, { title: 't', content: 'c', tags: [] })
    const result = await flush()
    expect(result).toBe('error')
    expect(status.value).toBe('error')
    expect(saveFn).toHaveBeenCalledTimes(1)

    vi.advanceTimersByTime(5000)
    await vi.runAllTicks()

    expect(saveFn).toHaveBeenCalledTimes(2)
    expect(status.value).toBe('saved')
  })

  it('重试全部失败后保持 error 不再重试', async () => {
    const saveFn = vi.fn().mockResolvedValue(false)
    const { schedule, flush, status } = useAutoSave(saveFn, {
      delay: 2000,
      retryDelays: [5000, 10000],
    })

    schedule(1, { title: 't', content: 'c', tags: [] })
    await flush()
    expect(saveFn).toHaveBeenCalledTimes(1)

    vi.advanceTimersByTime(5000)
    await vi.runAllTicks()
    expect(saveFn).toHaveBeenCalledTimes(2)
    expect(status.value).toBe('error')

    vi.advanceTimersByTime(10000)
    await vi.runAllTicks()
    expect(saveFn).toHaveBeenCalledTimes(3)
    expect(status.value).toBe('error')

    // 不再重试
    vi.advanceTimersByTime(60000)
    expect(saveFn).toHaveBeenCalledTimes(3)
  })

  it('重试期间用户编辑新数据取消重试', async () => {
    const saveFn = vi.fn()
      .mockResolvedValueOnce(false)
      .mockResolvedValue(true)
    const { schedule, flush, status } = useAutoSave(saveFn, {
      delay: 5000,
      retryDelays: [5000, 10000],
    })

    schedule(1, { title: 't1', content: 'c1', tags: [] })
    await flush()
    expect(status.value).toBe('error')

    // 重试定时器已启动（5s），用户在 2s 时编辑了新数据
    vi.advanceTimersByTime(2000)
    schedule(1, { title: 't2', content: 'c2', tags: [] })
    expect(status.value).toBe('pending')

    // 推进到原重试时间（t=5000），retry 已被 schedule 取消，不触发旧数据保存
    // debounce(5000) 在 t=7000 才到期，此时也不触发
    vi.advanceTimersByTime(3000)
    expect(saveFn).toHaveBeenCalledTimes(1)

    // 新数据防抖到期，正常保存
    vi.advanceTimersByTime(2000)
    await vi.runAllTicks()
    expect(saveFn).toHaveBeenCalledTimes(2)
    expect(saveFn).toHaveBeenLastCalledWith(1, { title: 't2', content: 'c2', tags: [] })
  })
})

describe('useAutoSave - markManuallySaved', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('无 pending 时置 saved 并启动淡出', async () => {
    const saveFn = vi.fn().mockResolvedValue(true)
    const { markManuallySaved, status } = useAutoSave(saveFn, {
      delay: 2000,
      savedFadeDelay: 3000,
    })

    markManuallySaved()
    expect(status.value).toBe('saved')

    vi.advanceTimersByTime(3000)
    expect(status.value).toBe('idle')
  })

  it('有 pending 时保持 pending 并重启防抖定时器', async () => {
    const saveFn = vi.fn().mockResolvedValue(true)
    const { schedule, markManuallySaved, status } = useAutoSave(saveFn, {
      delay: 2000,
      savedFadeDelay: 3000,
    })

    // 模拟手动保存期间用户编辑了新数据
    schedule(1, { title: 't2', content: 'c2', tags: [] })
    markManuallySaved()

    // 有 pending 数据，不应是 saved，应是 pending
    expect(status.value).toBe('pending')

    // 防抖到期后自动保存
    vi.advanceTimersByTime(2000)
    await vi.runAllTicks()
    expect(saveFn).toHaveBeenCalledTimes(1)
    expect(saveFn).toHaveBeenLastCalledWith(1, { title: 't2', content: 'c2', tags: [] })
  })

  it('清空重试状态', async () => {
    const saveFn = vi.fn().mockResolvedValue(false)
    const { schedule, flush, markManuallySaved, status } = useAutoSave(saveFn, {
      delay: 2000,
      retryDelays: [5000, 10000],
    })

    schedule(1, { title: 't', content: 'c', tags: [] })
    await flush()
    expect(status.value).toBe('error')

    // 手动保存成功后清空重试
    markManuallySaved()
    expect(status.value).toBe('saved')

    // 原重试定时器不应触发
    vi.advanceTimersByTime(60000)
    expect(saveFn).toHaveBeenCalledTimes(1)
  })
})
