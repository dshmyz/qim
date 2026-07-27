<template>
  <div class="message-bubble card-message" :class="{ self: isSelf }">
    <div class="card-container">
      <div v-if="card.title" class="card-title">{{ card.title }}</div>
      <div v-if="card.text" class="card-text">{{ card.text }}</div>
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
          <span class="card-btn-text">{{ btn.text }}</span>
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
// 初始化时读 localStorage 恢复已选态（切走再进仍标记已处理）
const submitted = ref(false)
const submitting = ref(false)
const selectedId = ref<string>(readPersistedAction())
if (selectedId.value) submitted.value = true

const card = computed<CardPayload>(() => {
  try {
    const p = JSON.parse(props.content)
    return p && typeof p === 'object' && !Array.isArray(p) ? (p as CardPayload) : {}
  } catch {
    return {}
  }
})

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
// （卡片不走流式，content 变化只来自 agent 显式更新，安全）
watch(() => props.content, () => {
  submitted.value = false
  submitting.value = false
  selectedId.value = ''
  writePersistedAction('')
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
  font-size: 14px;
  font-weight: 600;
  line-height: 1.4;
  color: var(--text-color);
  letter-spacing: -0.01em;
}

.card-text {
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-secondary);
  white-space: pre-wrap;
  word-break: break-word;
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
  font-size: 13px;
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

.card-btn.default.selected {
  background: color-mix(in srgb, var(--primary-color), transparent 88%);
  color: var(--primary-color);
  border-color: color-mix(in srgb, var(--primary-color), transparent 40%);
}

.card-empty {
  font-size: 12px;
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
</style>
