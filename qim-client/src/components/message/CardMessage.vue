<template>
  <div class="message-bubble card-message" :class="{ self: isSelf }">
    <div class="card-container">
      <div v-if="card.title" class="card-title" v-html="previewTextToHtml(card.title)"></div>
      <div
        v-if="card.text"
        class="card-text"
        :class="{ clamped: textClamped }"
        v-html="previewTextToHtml(card.text)"
      ></div>
      <button v-if="textLong" class="text-toggle" @click="textClamped = !textClamped">
        {{ textClamped ? '展开全文' : '收起' }}
      </button>
      <div
        v-if="isAIConfirm && card.target_conversation_id"
        class="card-jump"
        title="跳转到目标会话"
        @click="jumpToTarget"
      >
        在目标会话中查看 <i class="fas fa-arrow-right"></i>
      </div>
      <div v-if="card.buttons && card.buttons.length" class="card-actions">
        <button
          v-for="btn in card.buttons"
          :key="btn.id"
          class="card-btn"
          :class="[btn.style || 'default', { selected: selectedId === btn.id, done: submitted }]"
          :disabled="submitted || submitting"
          @click="handleClick(btn)"
        >
          <i v-if="submitting && selectedId === btn.id" class="fas fa-spinner fa-spin"></i>
          <span class="card-btn-text" v-html="previewTextToHtml(btn.text)"></span>
          <i v-if="submitted && selectedId === btn.id" class="fas fa-check"></i>
        </button>
      </div>
      <div v-else class="card-empty">卡片无可用按钮</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useBotCardAction } from '../../composables/useBotCardAction'
import { previewTextToHtml } from '../../utils/emoji'

interface CardPayloadWithConfirm extends CardPayload {
  kind?: string
  pending_id?: number
  target_conversation_id?: number
}

interface CardButton {
  id: string
  text: string
  style?: string
  value?: string
}

interface CardPayload {
  title?: string
  text?: string
  buttons?: CardButton[]
}

const props = defineProps<{
  content: string
  messageId: string
  isSelf?: boolean
  // 服务端从 CardActionRecord 派生的已点击 action_id（跨设备一致）。
  // 空串表示服务端无记录：回退 localStorage（同设备刷新仍能恢复，跨设备则视为未点）。
  actionTaken?: string
  serverUrl: string
}>()

const { submitCardAction } = useBotCardAction(props.serverUrl)

// 卡片点击持久化（localStorage）：按 messageId 记录已选 actionId，
// 防止组件重建/切走再进后重复点击。仅本设备生效；agent 回写新 content 时清除。
const CARD_ACTION_STORAGE_KEY = 'qim:card_actions'
const storageKey = () => `${CARD_ACTION_STORAGE_KEY}:${props.messageId}`

const readPersistedAction = (): string => {
  try {
    const v = localStorage.getItem(storageKey())
    return v || ''
  } catch {
    return ''
  }
}
const writePersistedAction = (actionId: string) => {
  try {
    if (actionId) localStorage.setItem(storageKey(), actionId)
    else localStorage.removeItem(storageKey())
  } catch {
    /* 忽略隐私模式/配额，不影响交互主流程 */
  }
}

// submitted：已成功提交，禁用全部按钮并高亮已选；submitting：当前请求中
// 已点击态优先取服务端派生（跨设备一致），其次回退 localStorage（同设备恢复）。
const serverAction = props.actionTaken || ''
const persistedAction = readPersistedAction()
const selectedId = ref<string>(serverAction || persistedAction)
const submitted = ref(false)
const submitting = ref(false)
if (selectedId.value) submitted.value = true
// 服务端有记录但 localStorage 没有时，补写本地缓存，保持两边一致
if (serverAction && !persistedAction) writePersistedAction(serverAction)

const card = computed<CardPayloadWithConfirm>(() => {
  try {
    const p = JSON.parse(props.content)
    return p && typeof p === 'object' && !Array.isArray(p) ? (p as CardPayloadWithConfirm) : {}
  } catch {
    return {}
  }
})

// 长文本折叠：超过 ~160 字默认收起，展开/收起切换（确认卡全文可见是确认制的前提）
const textClamped = ref(true)
const textLong = computed(() => (card.value.text || '').length > 160)

// 内部 AI 确认卡：提供「在目标会话中查看」回跳（确认后定位发送结果）
const isAIConfirm = computed(() => card.value.kind === 'ai_confirm')
const jumpToTarget = () => {
  const cid = card.value.target_conversation_id
  if (!cid) return
  window.dispatchEvent(new CustomEvent('ai-open-conversation', { detail: { conversationId: Number(cid) } }))
}

const handleClick = async (btn: CardButton) => {
  if (submitted.value || submitting.value) return
  submitting.value = true
  selectedId.value = btn.id
  const result = await submitCardAction(props.messageId, btn.id, btn.value)
  submitting.value = false
  if (result.ok) {
    // 成功（含幂等命中）：标记已处理，禁用按钮。后端幂等保证不会重复触发 webhook。
    submitted.value = true
    writePersistedAction(btn.id)
  } else {
    // 失败重置，允许用户重试
    selectedId.value = ''
  }
}

// agent 回写更新卡片 content 时，重置交互态并清除旧持久标记，让新按钮恢复可点
// （卡片不走流式，content 变化只来自 agent 显式更新，安全）。
// 已提交（submitted）的卡片不重置：确认卡终态回写（"✅ 已发送到 X"）也是 content
// 变化，若盲目重置会让按钮复活，再次点击产生重复气泡与重复结果行。按钮集合本身
// 变化（新按钮/按钮 id 变了）时才解除禁用。
const buttonsSignature = () => JSON.stringify((card.value.buttons || []).map(b => b.id))
const resetInteraction = () => {
  submitted.value = false
  submitting.value = false
  selectedId.value = ''
  writePersistedAction('')
}
watch(buttonsSignature, (sig, prevSig) => {
  if (sig !== prevSig) resetInteraction()
})
// 非确认卡：content 回写即新一轮（服务端 bot_messaging_service 对任意卡片改写删除幂等记录），
// 随内容重置交互恢复可点；确认卡终态回写不重置，避免按钮复活重复触发。
watch(() => props.content, () => {
  if (!isAIConfirm.value) resetInteraction()
})
</script>

<style scoped>
.card-message {
  padding: 0;
  background: transparent;
}

.card-container {
  min-width: 240px;
  max-width: min(100%, 360px);
  padding: 14px 16px;
  border-radius: 14px;
  background: color-mix(in srgb, var(--sidebar-bg), transparent 4%);
  border: 1px solid color-mix(in srgb, var(--border-color), transparent 20%);
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.06);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.card-title {
  font-size: var(--font-size-sm);
  font-weight: 600;
  line-height: 1.4;
  color: var(--text-color);
  letter-spacing: -0.01em;
}

.card-text {
  font-size: var(--font-size-xs);
  line-height: 1.6;
  color: var(--text-secondary);
  white-space: pre-wrap;
  word-break: break-word;
}

.card-text :deep(.emoji-img) {
  width: 16px;
  height: 16px;
  vertical-align: middle;
  margin: 0 1px;
}

.card-title :deep(.emoji-img) {
  width: 16px;
  height: 16px;
  vertical-align: middle;
  margin: 0 1px;
}

.card-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 2px;
}

.card-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 14px;
  border-radius: 10px;
  font-size: var(--font-size-xs);
  font-weight: 500;
  line-height: 1.2;
  cursor: pointer;
  border: 1px solid transparent;
  transition: background 0.16s ease, color 0.16s ease, border-color 0.16s ease, transform 0.16s ease, opacity 0.16s ease;
}

.card-btn:disabled {
  cursor: not-allowed;
}

/* default：描边次要按钮 */
.card-btn.default {
  background: transparent;
  color: var(--text-color);
  border-color: color-mix(in srgb, var(--border-color), transparent 10%);
}

.card-btn.default:not(:disabled):hover {
  background: color-mix(in srgb, var(--primary-color), transparent 92%);
  border-color: color-mix(in srgb, var(--primary-color), transparent 60%);
  color: var(--primary-color);
}

/* primary：主色实心按钮 */
.card-btn.primary {
  background: var(--primary-color);
  color: #fff;
}

.card-btn.primary:not(:disabled):hover {
  filter: brightness(1.05);
  transform: translateY(-1px);
}

/* danger：红色实心按钮 */
.card-btn.danger {
  background: #ef4444;
  color: #fff;
}

.card-btn.danger:not(:disabled):hover {
  filter: brightness(1.05);
  transform: translateY(-1px);
}

/* 已提交后：未选按钮淡出，已选按钮保持高亮 */
.card-btn.done:not(.selected) {
  opacity: 0.45;
}

.card-btn.selected {
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--primary-color), transparent 30%);
}

.card-btn-text :deep(.emoji-img) {
  width: 14px;
  height: 14px;
  vertical-align: middle;
  margin: 0 1px;
}

.card-btn.default.selected {
  background: color-mix(in srgb, var(--primary-color), transparent 88%);
  color: var(--primary-color);
  border-color: color-mix(in srgb, var(--primary-color), transparent 40%);
}

.card-empty {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
}

/* 自发送（理论上是 bot 发卡片，人类一侧不出现，保留样式一致性） */
.card-message.self .card-container {
  background: color-mix(in srgb, var(--sidebar-bg), transparent 4%);
  border-color: transparent;
}

[data-theme="elegant-dark"] .card-container {
  background: color-mix(in srgb, var(--panel-bg), white 5%);
  border-color: rgba(255, 255, 255, 0.12);
  box-shadow: none;
}

/* 长文本折叠 */
.card-text.clamped {
  display: -webkit-box;
  -webkit-line-clamp: 4;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.text-toggle {
  border: none;
  background: transparent;
  padding: 2px 0;
  font-size: var(--font-size-xxs);
  color: var(--el-color-primary, #6366f1);
  cursor: pointer;
}

.card-jump {
  margin-top: 6px;
  font-size: var(--font-size-xxs);
  color: var(--el-color-primary, #6366f1);
  cursor: pointer;
  opacity: 0.85;
}

.card-jump:hover {
  opacity: 1;
  text-decoration: underline;
}
</style>
