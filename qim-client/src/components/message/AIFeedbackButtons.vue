<template>
  <!-- AI 消息 👍/👎 反馈按钮（内联版）：供 MessageItem 的 .message-meta 行与
       BotChatView 气泡下方复用。逻辑自 AIAnswerBubble 平移：本地选中态 + 服务端
       upsert（再点同键 = 撤销），选中态经列表层批量拉取 provide 注入恢复。 -->
  <div v-if="messageIdNum" class="ai-fb" :class="{ 'fb-visible': fb !== 0 }">
    <button
      class="fb-btn"
      :class="{ active: fb === 1 }"
      :title="fb === 1 ? '取消点赞' : '赞，有帮助'"
      @click.stop="setFb(1)"
    ><i class="fas fa-thumbs-up"></i></button>
    <button
      class="fb-btn"
      :class="{ active: fb === -1 }"
      :title="fb === -1 ? '取消点踩' : '踩，没帮助'"
      @click.stop="setFb(-1)"
    ><i class="fas fa-thumbs-down"></i></button>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, inject, watch, onMounted, type Ref } from 'vue'
import { aiFeedbackAPI } from '../../api/ai'
import { aiFeedbackRatingsKey, type AIFeedbackRatings } from '../../composables/useAIFeedbackBatch'

const props = defineProps<{
  /** 持久化消息 id：主窗口消息 id 为 string（store 归一化），BotChatView 为 number，两者都接受。
      流式占位串（stream_xxx）归一化为 NaN → 不渲染 */
  messageId?: number | string
}>()

// 选中态恢复：列表层（MessageListView/BotChatView）批量拉取后经 provide 注入；
// 组件响应注入 map 变化；无注入（孤立使用）时回退挂载单条拉取
const batchRatings = inject<Ref<AIFeedbackRatings> | null>(aiFeedbackRatingsKey, null)
const fb = ref<1 | -1 | 0>(0)
// 用户已操作后忽略挂载期单条拉取结果，避免服务端（点击前）旧值覆盖本地新选择
let touched = false
// 统一归一化为数字 id（主窗口 string 型 → Number；流式占位串 → null 不渲染）
const messageIdNum = computed<number | null>(() => {
  const v = props.messageId
  if (v === undefined || v === null || v === '') return null
  const n = typeof v === 'number' ? v : Number(v)
  return Number.isFinite(n) && n > 0 ? n : null
})
if (batchRatings && messageIdNum.value) {
  // immediate：气泡可能在列表层已批量拉取之后才挂载（会话切换返回、分页重挂载），
  // 此时 map 引用不会再变，非 immediate 的 watch 永不触发导致选中态静默丢失
  watch(batchRatings, (map) => {
    const r = map.get(messageIdNum.value!)
    if (r === 1 || r === -1) fb.value = r
  }, { immediate: true })
}
const setFb = async (v: 1 | -1) => {
  if (!messageIdNum.value) return
  const prev = fb.value
  const next = prev === v ? 0 : v
  touched = true
  fb.value = next
  try {
    await aiFeedbackAPI.set(messageIdNum.value, next)
    // 回写批量注入的 map：新消息合并产生新 map 引用时，撤销/点赞不被旧值回退
    batchRatings?.value.set(messageIdNum.value, next)
  } catch {
    fb.value = prev // 失败回滚本地态，允许重试
  }
}
onMounted(() => {
  if (!messageIdNum.value) return
  // 列表层批量注入存在时不再单条拉取（N+1 消除）
  if (batchRatings) return
  aiFeedbackAPI.get(messageIdNum.value).then(r => {
    if (touched) return
    fb.value = (r === 1 || r === -1) ? r : 0
  }).catch(() => {})
})
</script>

<style>
/* 内联反馈按钮：默认透明（与 .message-time 同款 hover 显现），已选时常显。
   IM 侧由 .message-item:hover 联动显现（本文件底部规则），其他宿主自行加 hover 规则。 */
.ai-fb {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex: 0 0 auto;
}

.ai-fb .fb-btn {
  width: 18px;
  height: 18px;
  border: none;
  background: transparent;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  font-size: var(--font-size-xxxs);
  color: var(--text-secondary, #9ca3af);
  opacity: 0;
  transition: opacity 0.15s, color 0.15s, background 0.15s;
}

.ai-fb:hover .fb-btn,
.ai-fb.fb-visible .fb-btn,
.message-item:hover .ai-fb .fb-btn {
  opacity: 1;
}

.ai-fb .fb-btn:hover {
  background: color-mix(in srgb, var(--primary-color, #6366f1) 10%, transparent);
  color: var(--primary-color, #6366f1);
}

.ai-fb .fb-btn.active {
  color: var(--primary-color, #6366f1);
  background: color-mix(in srgb, var(--primary-color, #6366f1) 14%, transparent);
}
</style>
