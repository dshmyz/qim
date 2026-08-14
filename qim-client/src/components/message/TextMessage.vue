<template>
  <div class="message-bubble text-message" :class="{ self: isSelf }" @click="handleClick">
    <template v-for="(seg, i) in segments" :key="i">
      <span
        v-if="seg.type === 'mention'"
        class="at-mention-chip"
        :class="{ 'at-mention-chip--all': seg.userId === 'all' }"
        :data-user-id="seg.userId"
      >{{ seg.text }}</span>
      <TaskRefCard
        v-else-if="seg.type === 'task_ref'"
        :task-id="seg.taskId"
        :conversation-id="seg.conversationId"
        @click="handleTaskClick"
      />
      <span v-else v-html="seg.html"></span>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { escapeHTML, sanitizeHTML } from '../../utils/sanitize'
import { parseContent } from '../../utils/mentions'
import { emojiToHtml, classicToHtml } from '../../utils/emoji'
import { applyRenderRules } from '../../utils/renderRules'
import { useRenderRulesStore } from '../../stores/renderRules'
import TaskRefCard from './TaskRefCard.vue'

const props = defineProps<{
  content: string
  isSelf?: boolean
  conversationType?: string
  conversationId?: string | number
}>()

// 渲染规则 store（首次使用时触发拉取）
const renderRulesStore = useRenderRulesStore()
if (!renderRulesStore.loaded) renderRulesStore.fetchRules()

type Segment =
  | { type: 'text'; html: string }
  | { type: 'mention'; text: string; userId: number | 'all' }
  | { type: 'task_ref'; taskId: number; conversationId: number }

// 任务引用正则：#T-123
const taskRefRegex = /#T-(\d+)\b/g

// 数值化 conversationId（无效时返回 0，0 表示不渲染任务卡片）
const numericConvId = computed((): number => {
  const v = props.conversationId
  if (v == null || v === '') return 0
  const n = typeof v === 'number' ? v : Number(v)
  return Number.isFinite(n) && n > 0 ? n : 0
})

const markdownLinkRegex = /\[([^\]]+)\]\((https?:\/\/[^\s<>()\[\]]+)\)/g
const urlRegex = /(https?:\/\/[^\s<>()\[\]]+)/g

const renderLink = (url: string, label = url): string => {
  const classes = props.isSelf ? 'message-link message-link--self' : 'message-link'
  return `<a href="${url}" target="_blank" rel="noopener noreferrer" class="${classes}">${label}</a>`
}

const linkifyUrls = (text: string): string => text.replace(urlRegex, (matchedUrl) => {
  const url = matchedUrl.replace(/[.,:;!?]+$/, '')
  const trailingText = matchedUrl.slice(url.length)
  return renderLink(url) + trailingText
})

// 将纯文本片段转为带链接的 HTML（先转义，再处理 Markdown 链接，最后 linkify 剩余 URL）
const textToHtml = (text: string): string => {
  const escaped = escapeHTML(text)
  let linked = ''
  let lastIndex = 0
  let match: RegExpExecArray | null

  markdownLinkRegex.lastIndex = 0
  while ((match = markdownLinkRegex.exec(escaped)) !== null) {
    linked += linkifyUrls(escaped.slice(lastIndex, match.index))
    linked += renderLink(match[2], match[1])
    lastIndex = match.index + match[0].length
  }
  linked += linkifyUrls(escaped.slice(lastIndex))

  // 应用配置化渲染规则（Jira 工单号等），在 linkify 之后、sanitize 之前
  const rules = renderRulesStore.rulesForConversation(props.conversationType || 'group')
  if (rules.length) {
    linked = applyRenderRules(linked, rules)
  }

  // 先消毒（sanitizeHTML 不允许 <img>，用户输入无法伪造），再把表情字符/经典标记替换为 <img>
  return classicToHtml(emojiToHtml(sanitizeHTML(linked)))
}

const segments = computed<Segment[]>(() => {
  const { text, mentions } = parseContent(props.content)
  // 先按 mention 拆分，得到 text 段和 mention 段
  // 再对每个 text 段按 #T-数字 拆分出 task_ref 段
  const rawSegments: Array<{ type: 'text'; text: string } | { type: 'mention'; text: string; userId: number | 'all' }> = []

  if (mentions.length === 0) {
    rawSegments.push({ type: 'text', text })
  } else {
    let lastEnd = 0
    for (const m of mentions) {
      if (m.start > lastEnd) {
        rawSegments.push({ type: 'text', text: text.slice(lastEnd, m.start) })
      }
      rawSegments.push({ type: 'mention', text: m.text, userId: m.userId })
      lastEnd = m.end
    }
    if (lastEnd < text.length) {
      rawSegments.push({ type: 'text', text: text.slice(lastEnd) })
    }
  }

  // 对 text 段再拆 task_ref
  const convId = numericConvId.value
  const result: Segment[] = []
  for (const seg of rawSegments) {
    if (seg.type !== 'text') {
      result.push(seg)
      continue
    }
    // conversationId 无效时不拆 task_ref（降级为纯文本，后端无法校验权限）
    if (convId === 0) {
      result.push({ type: 'text', html: textToHtml(seg.text) })
      continue
    }
    taskRefRegex.lastIndex = 0
    let lastIdx = 0
    let m: RegExpExecArray | null
    while ((m = taskRefRegex.exec(seg.text)) !== null) {
      if (m.index > lastIdx) {
        result.push({ type: 'text', html: textToHtml(seg.text.slice(lastIdx, m.index)) })
      }
      result.push({ type: 'task_ref', taskId: Number(m[1]), conversationId: convId })
      lastIdx = m.index + m[0].length
    }
    if (lastIdx < seg.text.length) {
      result.push({ type: 'text', html: textToHtml(seg.text.slice(lastIdx)) })
    }
  }
  return result
})

// 任务卡片点击：预留扩展点，后续可通过事件链打开任务应用
// TODO: 通过 emit 链 MessageItem → MessageListView → 父组件调用 openApp('task')
const handleTaskClick = (_taskId: number) => {
  // 第一版仅展示卡片，点击跳转待接入应用打开机制
}

const handleClick = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  // 链接点击：Electron 外链打开
  const link = target.closest('a')
  if (link && window.electron?.shell?.openExternal) {
    event.preventDefault()
    const href = link.getAttribute('href')
    if (href) {
      window.electron.shell.openExternal(href)
    }
    return
  }
  // mention chip 点击：预留扩展点（未来可打开用户资料卡）
  const chip = target.closest('.at-mention-chip')
  if (chip) {
    // 暂不处理，保持默认行为
  }
}
</script>

<style scoped>
.text-message {
  padding: 10px 14px;
  border-radius: 12px;
  background: var(--sidebar-bg);
  color: var(--text-color);
  font-size: var(--font-size-sm);
  line-height: 1.6;
  word-break: break-word;
  white-space: pre-wrap;
}

:deep(.at-mention-chip) {
  color: var(--color-primary-600, #2563eb);
  font-weight: 600;
  cursor: default;
}

:deep(.at-mention-chip--all) {
  color: var(--color-primary-600, #2563eb);
}

:deep(.message-link) {
  color: var(--primary-color, #2563eb);
  font-weight: 500;
  overflow-wrap: anywhere;
  text-decoration: none;
}

:deep(.message-link--self) {
  color: var(--self-message-link-color, #1d4ed8);
}
</style>
