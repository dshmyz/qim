<template>
  <div class="message-bubble text-message" :class="{ self: isSelf }" @click="handleClick">
    <template v-for="(seg, i) in segments" :key="i">
      <span
        v-if="seg.type === 'mention'"
        class="at-mention-chip"
        :class="{ 'at-mention-chip--all': seg.userId === 'all' }"
        :data-user-id="seg.userId"
      >{{ seg.text }}</span>
      <span v-else v-html="seg.html"></span>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { escapeHTML, sanitizeHTML } from '../../utils/sanitize'
import { parseContent } from '../../utils/mentions'

const props = defineProps<{
  content: string
  isSelf?: boolean
}>()

type Segment =
  | { type: 'text'; html: string }
  | { type: 'mention'; text: string; userId: number | 'all' }

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

// 将纯文本片段转为带链接的 HTML（先转义，再插链接）
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

  return sanitizeHTML(linked)
}

const segments = computed<Segment[]>(() => {
  const { text, mentions } = parseContent(props.content)
  if (mentions.length === 0) {
    return [{ type: 'text', html: textToHtml(text) }]
  }

  const result: Segment[] = []
  let lastEnd = 0
  for (const m of mentions) {
    if (m.start > lastEnd) {
      result.push({ type: 'text', html: textToHtml(text.slice(lastEnd, m.start)) })
    }
    result.push({ type: 'mention', text: m.text, userId: m.userId })
    lastEnd = m.end
  }
  if (lastEnd < text.length) {
    result.push({ type: 'text', html: textToHtml(text.slice(lastEnd)) })
  }
  return result
})

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
  font-size: 14px;
  line-height: 1.5;
  word-break: break-word;
  white-space: pre-wrap;
}

:deep(.at-mention-chip) {
  color: var(--color-primary-600, #2563eb);
  font-weight: 600;
  cursor: default;
  padding: 1px 4px;
  border-radius: 4px;
  /* background: rgba(59, 130, 246, 0.16); */
}

:deep(.at-mention-chip--all) {
  color: var(--color-warning-600, #d97706);
  background: rgba(245, 158, 11, 0.18);
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
