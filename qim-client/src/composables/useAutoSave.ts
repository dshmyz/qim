import { ref, type Ref } from 'vue'

export type AutoSaveStatus = 'idle' | 'pending' | 'saving' | 'saved' | 'draft' | 'error'

export type FlushResult = 'saved' | 'error' | 'noop'

export interface AutoSavePayload {
  title: string
  content: string
  tags: string[]
}

export interface UseAutoSaveOptions {
  /** 防抖时长（毫秒），默认 2000 */
  delay?: number
  /** 最长等待时长（毫秒），超过后强制 flush，默认 10000 */
  maxWait?: number
  /** saved 状态淡出回 idle 的延迟（毫秒），默认 3000 */
  savedFadeDelay?: number
  /** 失败后退避重试间隔数组（毫秒），耗尽后停止重试，默认 [5000, 10000, 20000] */
  retryDelays?: number[]
}

export interface UseAutoSaveReturn<T> {
  status: Ref<AutoSaveStatus>
  /** 排定一次防抖保存；多次调用会合并为最后一次的数据 */
  schedule: (id: T, data: AutoSavePayload) => void
  /** 立即触发待保存数据写入并清空定时器；返回 saved/error/noop */
  flush: () => Promise<FlushResult>
  /** 取消待保存并重置状态为 idle */
  cancel: () => void
  /** 手动保存成功后同步状态：有新 pending 则保持 pending 并重启防抖，否则置 saved */
  markManuallySaved: () => void
}

/**
 * 笔记/文档类编辑器的防抖自动保存。
 *
 * 特性：
 * - 防抖合并：连续编辑只保存最后一次数据
 * - maxWait 强制保存：避免连续输入导致永远不保存
 * - saved 淡出：保存成功后短暂显示"已保存"，随后恢复中性状态
 * - 失败退避重试：网络抖动时自动重试，达上限后停止并保持 error
 * - 保存期间用户编辑新数据：旧数据结果不覆盖新 pending 状态
 */
export function useAutoSave<T>(
  saveFn: (id: T, data: AutoSavePayload) => Promise<boolean>,
  options: UseAutoSaveOptions = {}
): UseAutoSaveReturn<T> {
  const delay = options.delay ?? 2000
  const maxWait = options.maxWait ?? 10000
  const savedFadeDelay = options.savedFadeDelay ?? 3000
  const retryDelays = options.retryDelays ?? [5000, 10000, 20000]

  const status = ref<AutoSaveStatus>('idle')

  let debounceTimer: ReturnType<typeof setTimeout> | null = null
  let maxWaitTimer: ReturnType<typeof setTimeout> | null = null
  let savedFadeTimer: ReturnType<typeof setTimeout> | null = null
  let retryTimer: ReturnType<typeof setTimeout> | null = null

  let pendingId: T | null = null
  let pendingData: AutoSavePayload | null = null

  // 重试相关：保存失败后保留数据用于退避重试
  let retryId: T | null = null
  let retryData: AutoSavePayload | null = null
  let retryCount = 0

  function clearDebounce() {
    if (debounceTimer !== null) {
      clearTimeout(debounceTimer)
      debounceTimer = null
    }
  }

  function clearMaxWait() {
    if (maxWaitTimer !== null) {
      clearTimeout(maxWaitTimer)
      maxWaitTimer = null
    }
  }

  function clearSavedFade() {
    if (savedFadeTimer !== null) {
      clearTimeout(savedFadeTimer)
      savedFadeTimer = null
    }
  }

  function clearRetry() {
    if (retryTimer !== null) {
      clearTimeout(retryTimer)
      retryTimer = null
    }
  }

  function clearAllTimers() {
    clearDebounce()
    clearMaxWait()
    clearSavedFade()
    clearRetry()
  }

  function markSaved() {
    clearSavedFade()
    status.value = 'saved'
    savedFadeTimer = setTimeout(() => {
      savedFadeTimer = null
      if (status.value === 'saved') {
        status.value = 'idle'
      }
    }, savedFadeDelay)
  }

  function clearRetryState() {
    clearRetry()
    retryId = null
    retryData = null
    retryCount = 0
  }

  /**
   * 执行保存。isRetry=true 表示这是退避重试触发的。
   * 返回 saved/error，调用方据此决定反馈。
   */
  async function doSave(id: T, data: AutoSavePayload, isRetry: boolean): Promise<FlushResult> {
    clearDebounce()
    clearMaxWait()
    status.value = 'saving'

    let ok = false
    let threw = false
    try {
      ok = await saveFn(id, data)
    } catch {
      threw = true
    }

    // 保存期间用户又编辑了新数据：旧数据结果不影响新 pending 状态
    if (pendingId !== null) {
      if (isRetry) {
        clearRetryState()
      }
      status.value = 'pending'
      return threw || !ok ? 'error' : 'saved'
    }

    if (!threw && ok) {
      clearRetryState()
      markSaved()
      return 'saved'
    }

    // 失败：启动退避重试
    if (isRetry) {
      retryCount++
    } else {
      retryCount = 0
    }

    if (retryCount >= retryDelays.length) {
      // 达到重试上限，停止重试
      clearRetryState()
      status.value = 'error'
      return 'error'
    }

    retryId = id
    retryData = data
    clearRetry()
    const retryDelay = retryDelays[retryCount]
    retryTimer = setTimeout(() => {
      retryTimer = null
      if (retryId !== null && retryData !== null) {
        void doSave(retryId, retryData, true)
      }
    }, retryDelay)
    status.value = 'error'
    return 'error'
  }

  function schedule(id: T, data: AutoSavePayload) {
    // 用户编辑了新数据，取消正在进行的重试（新数据走防抖流程）
    clearRetryState()

    pendingId = id
    pendingData = data
    clearSavedFade()
    status.value = 'pending'

    clearDebounce()
    debounceTimer = setTimeout(() => {
      debounceTimer = null
      void flush()
    }, delay)

    // maxWait 只在不存在时启动，保证从首次编辑起最长等待 maxWait
    if (maxWaitTimer === null) {
      maxWaitTimer = setTimeout(() => {
        maxWaitTimer = null
        void flush()
      }, maxWait)
    }
  }

  async function flush(): Promise<FlushResult> {
    clearDebounce()
    clearMaxWait()

    const id = pendingId
    const data = pendingData
    if (id === null || data === null) {
      return 'noop'
    }
    pendingId = null
    pendingData = null
    return doSave(id, data, false)
  }

  function cancel() {
    clearAllTimers()
    pendingId = null
    pendingData = null
    clearRetryState()
    status.value = 'idle'
  }

  function markManuallySaved() {
    clearRetryState()
    clearSavedFade()
    clearDebounce()
    clearMaxWait()

    if (pendingId !== null) {
      // 手动保存期间用户编辑了新数据：保持 pending 并重启防抖
      status.value = 'pending'
      debounceTimer = setTimeout(() => {
        debounceTimer = null
        void flush()
      }, delay)
      maxWaitTimer = setTimeout(() => {
        maxWaitTimer = null
        void flush()
      }, maxWait)
    } else {
      markSaved()
    }
  }

  return {
    status,
    schedule,
    flush,
    cancel,
    markManuallySaved,
  }
}
