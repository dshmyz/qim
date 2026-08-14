<template>
  <!-- 外部 AI 工具调用追踪：独立卡片，与 markdown 正文视觉分层（参考 capability-console
       的工具卡片 + 现有 AISources 折叠标签 idiom）。放在 AI 气泡下方，
       实时由 ai_tool_call WS 事件累积，回放从消息 Extra 解析。 -->
  <details v-if="calls && calls.length > 0" ref="box" class="tool-call-trace">
    <summary @click.prevent="toggle">
      <i class="fas fa-wrench"></i>
      <span>工具调用</span>
      <span class="count">{{ calls.length }}</span>
      <span class="chevron" :class="{ open: opened }"><i class="fas fa-chevron-down"></i></span>
    </summary>
    <ul>
      <li v-for="(call, i) in calls" :key="i" class="trace-row" :class="{ error: call.status === 'error' }">
        <span class="trace-icon" aria-hidden="true">🔧</span>
        <span class="trace-label" :title="displayLabel(call)">{{ displayLabel(call) }}</span>
        <span v-if="call.status === 'running'" class="trace-status running" title="进行中"><span class="spin"></span>进行中</span>
        <span v-if="call.status === 'ok'" class="trace-status ok" title="已完成">✓</span>
        <span v-if="call.status === 'error'" class="trace-status error">失败</span>
        <span v-if="formatArgs(call.args)" class="trace-args" :title="formatArgs(call.args)">{{ formatArgs(call.args) }}</span>
      </li>
    </ul>
  </details>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { ToolCallRecord } from '../../types'

const props = defineProps<{
  calls?: ToolCallRecord[]
  open?: boolean
}>()

// 展开交互（option B）：streaming 期间 open=true 自动展开，结束时 open=false 自动收起；
// 之后留待用户手动开合。用原生 <details> 未绑定 :open，靠 open prop 的边沿驱动 + 手动
// toggle，避免终态用户展开后又被任一重渲染 snap 回折叠。
const box = ref<HTMLDetailsElement | null>(null)
const opened = ref(false)

const setOpen = (v: boolean) => {
  opened.value = v
  if (box.value) box.value.open = v
}

// 仅在 open prop 发生边沿变化时驱动（进入 streaming 展开 / 结束 streaming 收起），
// 其余时间不动原生 <details>，把控制权交还用户点击。
// immediate: 组件首次挂载时若 open 已为 true（流式期间延迟渲染），立即展开。
watch(
  () => props.open,
  (v, prev) => {
    if (v === prev) return
    setOpen(v)
  },
  { immediate: true }
)

function toggle() {
  setOpen(!Boolean(box.value?.open))
}

// 泛化标签：tool_label 为这些值时，用 tool_name 做 fallback 显示
const GENERIC_LABELS = new Set(['外部服务', '外部工具'])

// 显示标签：优先 tool_label（人类可读中文），泛化时 fallback 到 tool_name（原始工具名）
function displayLabel(call: ToolCallRecord): string {
  const label = call.tool_label?.trim()
  if (label && !GENERIC_LABELS.has(label)) return label
  // fallback：从 tool_name 提取 mcp_<conn>_<tool> 的 <tool> 部分
  const name = call.tool_name?.trim()
  if (name) {
    if (name.startsWith('mcp_')) {
      const rest = name.slice(4)
      const idx = rest.indexOf('_')
      if (idx >= 0) {
        const toolPart = rest.slice(idx + 1)
        if (toolPart) return toolPart.replace(/_/g, ' ').replace(/-/g, ' ')
      }
    }
    return name.replace(/_/g, ' ').replace(/-/g, ' ')
  }
  return label || '工具调用'
}

// 输入参数摘要：取前若干个 key=value 拼成一行，过长省略（title 全量）。
function formatArgs(args?: Record<string, unknown>): string {
  if (!args) return ''
  const entries = Object.entries(args).filter(([, v]) => v != null && v !== '')
  if (entries.length === 0) return ''
  const parts = entries
    .slice(0, 3)
    .map(([k, v]) => `${k}=${typeof v === 'object' ? JSON.stringify(v) : String(v)}`)
  return parts.join(' ')
}
</script>

<style scoped>
.tool-call-trace {
  margin-top: 6px;
  padding: 5px 10px;
  border: 1px solid var(--border-color, #e8ecf3);
  border-left: 3px solid #4f7cff;
  border-radius: 6px;
  background: var(--hover-color, #f9fafc);
  font-size: var(--font-size-xxs);
  color: var(--text-secondary, #666);
  max-width: 100%;
  box-sizing: border-box;
}
.tool-call-trace summary {
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  list-style: none;
  user-select: none;
  color: var(--text-secondary, #888);
}
.tool-call-trace summary::-webkit-details-marker { display: none; }
.tool-call-trace summary i { color: #4f7cff; font-size: var(--font-size-xxxs); }
.tool-call-trace .chevron {
  margin-left: auto;
  font-size: 9px;
  color: #b0b8c4;
  transition: transform 0.2s ease;
}
.tool-call-trace .chevron.open { transform: rotate(180deg); }
.tool-call-trace .count {
  display: inline-block;
  min-width: 16px;
  padding: 0 5px;
  height: 16px;
  line-height: 16px;
  text-align: center;
  background: var(--border-color, #e8ecf3);
  color: var(--text-secondary, #666);
  border-radius: 8px;
  font-size: var(--font-size-tiny);
}
.tool-call-trace ul {
  margin: 6px 0 2px;
  padding: 0;
  list-style: none;
}
.tool-call-trace .trace-row {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 0;
  border-top: 1px dashed var(--border-color, #eef1f6);
  min-width: 0;
}
.tool-call-trace .trace-row:first-child { border-top: none; }
.tool-call-trace .trace-icon { flex-shrink: 0; font-size: var(--font-size-xxxs); }
.tool-call-trace .trace-label {
  flex-shrink: 0;
  color: var(--text-color, #333);
  font-weight: 500;
  white-space: nowrap;
}
.tool-call-trace .trace-args {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-secondary, #999);
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: var(--font-size-xxxs);
}
.tool-call-trace .trace-status.error {
  flex-shrink: 0;
  color: #d9534f;
  font-size: var(--font-size-xxxs);
  font-weight: 500;
}
.tool-call-trace .trace-status.ok {
  flex-shrink: 0;
  color: #52c41a;
  font-size: var(--font-size-xxxs);
  font-weight: 600;
}
.tool-call-trace .trace-status.running {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: #4f7cff;
  font-size: var(--font-size-xxxs);
  font-weight: 500;
}
.tool-call-trace .trace-status.running .spin {
  width: 9px;
  height: 9px;
  border: 2px solid #4f7cff;
  border-top-color: transparent;
  border-radius: 50%;
  animation: toolcall-spin 0.8s linear infinite;
}
@keyframes toolcall-spin {
  to { transform: rotate(360deg); }
}
.tool-call-trace .trace-row.error .trace-label { color: #d9534f; }

/* 深色主题适配 */
[data-theme="elegant-dark"] .tool-call-trace {
  border-color: #3a3f4b;
  border-left-color: #6f9bff;
  background: #262b36;
  color: #aab;
}
[data-theme="elegant-dark"] .tool-call-trace summary { color: #8899aa; }
[data-theme="elegant-dark"] .tool-call-trace .chevron { color: #5a6a80; }
[data-theme="elegant-dark"] .tool-call-trace .trace-label { color: #d5dde7; }
[data-theme="elegant-dark"] .tool-call-trace .trace-args { color: #7a8699; }
[data-theme="elegant-dark"] .tool-call-trace .trace-row { border-top-color: #333a47; }
[data-theme="elegant-dark"] .tool-call-trace .count {
  background: #333a47;
  color: #8899aa;
}
</style>
