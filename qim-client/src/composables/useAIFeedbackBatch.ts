import { onScopeDispose, ref, watch, type Ref } from 'vue'
import { aiFeedbackAPI } from '../api/ai'
import { isAIMessage } from '../utils/aiMessage'

/** provide/inject 键：列表层批量拉取的选中态 map（messageId -> rating） */
export const aiFeedbackRatingsKey = 'aiFeedbackRatings' as const
export type AIFeedbackRatings = Map<number, number>

/** 与服务端 ai_pending_action_handler 单次上限一致，超出必须分片 */
const BATCH_MAX_IDS = 200

/**
 * useAIFeedbackBatch 批量拉取会话内 AI 消息的反馈选中态（👍/👎）。
 *
 * 背景：AIAnswerBubble 每条挂载即 GET /ai/feedback/:id，打开含 50 条 AI 回复的
 * 会话形成 50 次请求的 N+1。列表层改用本 composable 一次 POST /ai/feedback/batch
 * 拉取全量选中态，再经 provide 注入给各 AIAnswerBubble。
 *
 * messages 变化时去抖 300ms 合并拉取（分页追加不重复拉已查过的 id）；
 * 超过服务端单次上限时按片并发提交，单片失败只回滚该片重试。
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
      if (!id || fetched.has(id) || !isAIMessage(m)) continue
      fetched.add(id)
      ids.push(id)
    }
    if (!ids.length) return
    for (let i = 0; i < ids.length; i += BATCH_MAX_IDS) {
      const chunk = ids.slice(i, i + BATCH_MAX_IDS)
      aiFeedbackAPI.getBatch(chunk).then(map => {
        if (map.size) ratings.value = new Map([...ratings.value, ...map])
      }).catch(() => {
        // 仅回滚本片 id，允许下次触发重试（不牵连其它已成功片）
        for (const id of chunk) fetched.delete(id)
      })
    }
  }

  const scheduleFetch = () => {
    if (timer) clearTimeout(timer)
    timer = setTimeout(fetchPending, 300)
  }

  watch(messages, scheduleFetch, { deep: false })
  // 挂载即首次拉取（watch 默认不立即触发）
  scheduleFetch()
  // 组件卸载时清掉未决去抖定时器，避免拉取作用在已卸载列表的 stale ratings 上
  onScopeDispose(() => {
    if (timer) clearTimeout(timer)
    timer = null
  })

  return ratings
}
