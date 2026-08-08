<template>
  <!-- 外部 AI 工具调用追踪：独立卡片，与 markdown 正文视觉分层（参考 capability-console
       的工具卡片 + 现有 KnowledgeSources 折叠标签 idiom）。放在 AI 气泡下方，
       实时由 ai_tool_call WS 事件累积，回放从消息 Extra 解析。 -->
  <details v-if="calls && calls.length > 0" class="tool-call-trace" :open="open">
    <summary>
      <i class="fas fa-wrench"></i>
      <span>工具调用</span>
      <span class="count">{{ calls.length }}</span>
    </summary>
    <ul>
      <li v-for="(call, i) in calls" :key="i" class="trace-row" :class="{ error: call.status === 'error' }">
        <span class="trace-icon" aria-hidden="true">🔧</span>
        <span class="trace-label" :title="call.tool_label">{{ call.tool_label }}</span>
        <span v-if="call.status === 'error'" class="trace-status error">失败</span>
        <span v-if="formatArgs(call.args)" class="trace-args" :title="formatArgs(call.args)">{{ formatArgs(call.args) }}</span>
      </li>
    </ul>
  </details>
</template>

<script setup lang="ts">
import type { ToolCallRecord } from '../../types'

defineProps<{
  calls?: ToolCallRecord[]
  open?: boolean
}>()

// 输入参数摘要：取前若干个 key=value 拼成一行，过长省略（title 全量）。
function formatArgs(args?: Record<string, unknown>): string {
  if (!args) return ''
  const entries = Object.entries(args)
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
  border: 1px solid #e8ecf3;
  border-left: 3px solid #4f7cff;
  border-radius: 6px;
  background: #f9fafc;
  font-size: 12px;
  color: #666;
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
  color: #888;
}
.tool-call-trace summary::-webkit-details-marker { display: none; }
.tool-call-trace summary i { color: #4f7cff; font-size: 11px; }
.tool-call-trace .count {
  display: inline-block;
  min-width: 16px;
  padding: 0 5px;
  height: 16px;
  line-height: 16px;
  text-align: center;
  background: #e8ecf3;
  color: #666;
  border-radius: 8px;
  font-size: 10px;
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
  border-top: 1px dashed #eef1f6;
  min-width: 0;
}
.tool-call-trace .trace-row:first-child { border-top: none; }
.tool-call-trace .trace-icon { flex-shrink: 0; font-size: 11px; }
.tool-call-trace .trace-label {
  flex-shrink: 0;
  color: #333;
  font-weight: 500;
  white-space: nowrap;
}
.tool-call-trace .trace-args {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #999;
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 11px;
}
.tool-call-trace .trace-status.error {
  flex-shrink: 0;
  color: #d9534f;
  font-size: 11px;
  font-weight: 500;
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
[data-theme="elegant-dark"] .tool-call-trace .trace-label { color: #d5dde7; }
[data-theme="elegant-dark"] .tool-call-trace .trace-args { color: #7a8699; }
[data-theme="elegant-dark"] .tool-call-trace .trace-row { border-top-color: #333a47; }
[data-theme="elegant-dark"] .tool-call-trace .count {
  background: #333a47;
  color: #8899aa;
}
</style>
