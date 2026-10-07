<template>
  <!-- AI 消息 👍/👎 反馈按钮（内联版）：供 MessageItem 的 .message-meta 行与
       BotChatView 气泡下方复用。逻辑自 AIAnswerBubble 平移：本地选中态 + 服务端
       upsert（再点同键 = 撤销），选中态经列表层批量拉取 provide 注入恢复。
       👎 后展开原因分类面板，反馈信号供失败聚类与自我进化分析。 -->
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

    <div v-if="panelOpen" class="fb-reason-panel" @click.stop>
      <p class="reason-title">哪里不好？（点选后可补充说明）</p>
      <div class="reason-chips">
        <button
          v-for="reason in reasonOptions"
          :key="reason.key"
          class="reason-chip"
          :class="{ selected: selectedReason === reason.key }"
          @click="selectedReason = reason.key"
        >{{ reason.label }}</button>
      </div>
      <textarea
        v-model="comment"
        class="reason-comment"
        rows="2"
        placeholder="补充说明（可选）"
      ></textarea>
      <div class="reason-actions">
        <button class="reason-submit" :disabled="!selectedReason" @click="submitReason">
          提交反馈
        </button>
        <button class="reason-skip" @click="closePanel">跳过</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, inject, watch, onBeforeUnmount, type Ref } from 'vue'
import { aiFeedbackAPI } from '../../api/ai'
import { aiFeedbackRatingsKey, type AIFeedbackRatings } from '../../composables/useAIFeedbackBatch'

const props = defineProps<{
  /** 持久化消息 id：主窗口消息 id 为 string（store 归一化），BotChatView 为 number，两者都接受。
      流式占位串（stream_xxx）归一化为 NaN → 不渲染 */
  messageId?: number | string
}>()

// 选中态恢复：列表层（MessageListView/BotChatView）批量拉取后经 provide 注入；
// 组件响应注入 map 变化；无注入（孤立使用）时不拉取、不恢复选中态（单条拉取已随 N+1 消除移除）
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

// —— 原因分类面板 ——
const reasonOptions = [
  { key: 'off_topic', label: '答非所问' },
  { key: 'hallucinated', label: '编造信息' },
  { key: 'wrong_trigger', label: '不该回复' },
  { key: 'incomplete', label: '没说完' },
  { key: 'factual_error', label: '事实错误' },
  { key: 'bad_tone', label: '语气不当' },
  { key: 'other', label: '其他' }
]
const panelOpen = ref(false)
const selectedReason = ref('')
const comment = ref('')

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
    return
  }
  if (v === -1 && next === -1) openPanel()
  else closePanel()
}

async function submitReason() {
  if (!messageIdNum.value || !selectedReason.value) return
  const reason = selectedReason.value
  const text = comment.value.trim()
  try {
    await aiFeedbackAPI.set(messageIdNum.value, -1, { reasonCategory: reason, comment: text })
    closePanel()
  } catch {
    // 保留面板与已填内容，允许重试
  }
}

function openPanel() {
  panelOpen.value = true
  // 点面板外关闭：one-shot document 监听，关闭即移除，避免累积残留
  document.addEventListener('click', closePanel, { once: true })
}

function closePanel() {
  panelOpen.value = false
  selectedReason.value = ''
  comment.value = ''
  document.removeEventListener('click', closePanel)
}

onBeforeUnmount(() => document.removeEventListener('click', closePanel))
</script>

<style>
/* 内联反馈按钮：默认透明（与 .message-time 同款 hover 显现），已选时常显。
   IM 侧由 .message-item:hover 联动显现（本文件底部规则），其他宿主自行加 hover 规则。 */
.ai-fb {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex: 0 0 auto;
  position: relative;
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

/* ---- 原因分类面板 ---- */
.ai-fb .fb-reason-panel {
  position: absolute;
  top: calc(100% + 8px);
  left: 0;
  z-index: 1000;
  width: 280px;
  padding: 12px;
  background: var(--card-bg, #fff);
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 10px;
  box-shadow: 0 8px 28px rgba(0, 0, 0, 0.14);
}

.ai-fb .reason-title {
  margin: 0 0 10px;
  font-size: var(--font-size-xs);
  color: var(--text-primary);
  font-weight: 500;
}

.ai-fb .reason-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.ai-fb .reason-chip {
  padding: 4px 10px;
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 12px;
  background: var(--card-bg, #fff);
  color: var(--text-secondary);
  font-size: var(--font-size-xxs);
  cursor: pointer;
  transition: all 0.15s;
}

.ai-fb .reason-chip:hover {
  border-color: var(--primary-color, #6366f1);
  color: var(--primary-color, #6366f1);
}

.ai-fb .reason-chip.selected {
  background: color-mix(in srgb, var(--primary-color, #6366f1) 12%, transparent);
  border-color: var(--primary-color, #6366f1);
  color: var(--primary-color, #6366f1);
}

.ai-fb .reason-comment {
  width: 100%;
  margin-top: 10px;
  padding: 8px;
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 8px;
  background: var(--input-bg, var(--card-bg, #fff));
  color: var(--text-primary);
  font-size: var(--font-size-xs);
  resize: none;
  outline: none;
  box-sizing: border-box;
  font-family: inherit;
}

.ai-fb .reason-comment:focus {
  border-color: var(--primary-color, #6366f1);
}

.ai-fb .reason-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 10px;
}

.ai-fb .reason-submit {
  padding: 5px 14px;
  border: none;
  border-radius: 7px;
  background: var(--primary-color, #6366f1);
  color: #fff;
  font-size: var(--font-size-xs);
  cursor: pointer;
}

.ai-fb .reason-submit:disabled {
  opacity: 0.45;
  cursor: default;
}

.ai-fb .reason-skip {
  padding: 5px 8px;
  border: none;
  border-radius: 7px;
  background: transparent;
  color: var(--text-secondary);
  font-size: var(--font-size-xs);
  cursor: pointer;
}

.ai-fb .reason-skip:hover {
  color: var(--text-primary);
}
</style>
