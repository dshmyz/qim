<template>
  <!-- 回复可信度标注：「可能不准」徽章 + 展开原因 + 一键反馈。
       正文已经发出（后端核验是事后异步完成的），这里只做事后提示与反馈入口，
       不撤回、不改写已发内容。 -->
  <div v-if="visible" class="ai-quality">
    <button class="quality-chip" :class="{ open }" :title="quality?.reason || '这条回复可能不准确'" @click.stop="toggle">
      <i class="fas fa-circle-exclamation"></i>
      <span>可能不准</span>
    </button>
    <div v-if="open" class="quality-panel" @click.stop>
      <p class="quality-reason">{{ quality?.reason || '这条回复没有通过依据核验，内容可能不准确。' }}</p>
      <div class="quality-actions">
        <button class="quality-feedback" :disabled="submitted" @click="submitFeedback">
          {{ submitted ? '已反馈，谢谢' : '这条确实不准' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { aiFeedbackAPI } from '../../api/ai'
import type { AIQualityAnnotation } from '../../types'

const props = defineProps<{
  quality?: AIQualityAnnotation
  /** 持久化消息 id：主窗口为 string，BotChat 为 number；流式占位串归一化为 null 不渲染 */
  messageId?: number | string
}>()

const visible = computed(() => props.quality?.status === 'doubtful')
const messageIdNum = computed<number | null>(() => {
  const v = props.messageId
  if (v === undefined || v === null || v === '') return null
  const n = typeof v === 'number' ? v : Number(v)
  return Number.isFinite(n) && n > 0 ? n : null
})

const open = ref(false)
const submitted = ref(false)
let outsideCloseTimer: ReturnType<typeof setTimeout> | undefined

function toggle() {
  if (open.value) {
    close()
    return
  }
  open.value = true
  // 下一轮事件循环再挂 outside-close：在当前点击的传播途中挂监听，同一个事件冒到
  // document 会立刻把刚打开的面板关掉（点击看起来"没反应"）。timer 需在卸载时清掉，
  // 否则组件卸载后仍会向 document 注册监听。
  outsideCloseTimer = setTimeout(() => {
    outsideCloseTimer = undefined
    if (open.value) document.addEventListener('click', close, { once: true })
  }, 0)
}

function close() {
  open.value = false
  document.removeEventListener('click', close)
}

async function submitFeedback() {
  if (!messageIdNum.value || submitted.value) return
  try {
    await aiFeedbackAPI.set(messageIdNum.value, -1, {
      reasonCategory: 'other',
      comment: `质量核验存疑：${props.quality?.reason || '未通过依据核验'}`,
    })
    submitted.value = true
  } catch {
    // 保留面板与按钮，允许重试
  }
}

onBeforeUnmount(() => {
  if (outsideCloseTimer) {
    clearTimeout(outsideCloseTimer)
    outsideCloseTimer = undefined
  }
  document.removeEventListener('click', close)
})
</script>

<style>
.ai-quality {
  position: relative;
  display: inline-flex;
  margin-top: 6px;
}

.ai-quality .quality-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 1px 8px;
  border: 1px solid color-mix(in srgb, var(--warning-color, #f59e0b) 45%, transparent);
  border-radius: 10px;
  background: color-mix(in srgb, var(--warning-color, #f59e0b) 10%, transparent);
  color: var(--warning-color, #b45309);
  font-size: var(--font-size-xxs);
  cursor: pointer;
  transition: background 0.15s;
}

.ai-quality .quality-chip:hover,
.ai-quality .quality-chip.open {
  background: color-mix(in srgb, var(--warning-color, #f59e0b) 18%, transparent);
}

.ai-quality .quality-panel {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  z-index: 1000;
  width: 260px;
  padding: 10px 12px;
  background: var(--card-bg, #fff);
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 10px;
  box-shadow: 0 8px 28px rgba(0, 0, 0, 0.14);
}

.ai-quality .quality-reason {
  margin: 0 0 8px;
  color: var(--text-primary);
  font-size: var(--font-size-xs);
  line-height: 1.5;
}

.ai-quality .quality-actions {
  display: flex;
  justify-content: flex-end;
}

.ai-quality .quality-feedback {
  padding: 4px 12px;
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 7px;
  background: transparent;
  color: var(--text-secondary);
  font-size: var(--font-size-xs);
  cursor: pointer;
}

.ai-quality .quality-feedback:hover:not(:disabled) {
  border-color: var(--warning-color, #f59e0b);
  color: var(--warning-color, #b45309);
}

.ai-quality .quality-feedback:disabled {
  opacity: 0.6;
  cursor: default;
}
</style>
