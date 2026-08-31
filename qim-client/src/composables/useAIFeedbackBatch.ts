import { ref, watch, type Ref } from 'vue'
import { aiFeedbackAPI } from '../api/ai'

/** 与 MessageItem.isAIMessage 同判据：origin/sender/is_ai_message 任一命中即 AI 消息 */
export const isAIMessageLike = (m: any): boolean => {
  return m?.origin === 'assistant' || m?.origin === 'avatar' ||
    m?.sender?.type === 'bot' || m?.sender?.type === 'system' ||
    !!(m?.is_ai_message || m?.isAIMessage)
}

/** provide/inject 键：列表层批量拉取的选中态 map（messageId -> rating） */
export const aiFeedbackRatingsKey = 'aiFeedbackRatings' as const
export type AIFeedbackRatings = Map<number, number>

/**
 * useAIFeedbackBatch 批量拉取会话内 AI 消息的反馈选中态（👍/👎）。
 *
 * 背景：AIAnswerBubble 每条挂载即 GET /ai/feedback/:id，打开含 50 条 AI 回复的
 * 会话形成 50 次请求的 N+1。列表层改用本 composable 一次 POST /ai/feedback/batch
 * 拉取全量选中态，再经 provide 注入给各 AIAnswerBubble。
 *
 * messages 变化时去抖 300ms 合并拉取（分页追加不重复拉已查过的 id）。
 */
export function useAIFeedbackBatch(messages: Ref<{ id: number | string }[]>) {
  /** messageId -> rating（-1/0/1） */
  const ratings = ref<AIFeedbackRatings>(new Map())
  const fetched = new Set<number>()
  let timer: ReturnType<typeof setTimeout> | null = null

  const fetchPending = () => {
    timer = null
    const ids: number[] = []
    for (const m of messages.value) {
      const id = typeof m.id === 'number' ? m.id : Number(m.id)
      // 仅数字 id（持久化消息）且未查过；Number('stream_xxx') = NaN 自动排除
      if (!id || fetched.has(id) || !isAIMessageLike(m)) continue
      fetched.add(id)
      ids.push(id)
    }
    if (!ids.length) return
    aiFeedbackAPI.getBatch(ids).then(map => {
      if (map.size) ratings.value = new Map([...ratings.value, ...map])
    }).catch(() => {
      // 拉取失败：允许下次触发重试
      for (const id of ids) fetched.delete(id)
    })
  }

  const scheduleFetch = () => {
    if (timer) clearTimeout(timer)
    timer = setTimeout(fetchPending, 300)
  }

  watch(messages, scheduleFetch, { deep: false })
  // 挂载即首次拉取（watch 默认不立即触发）
  scheduleFetch()

  return ratings
}
