<template>
  <div class="message-bubble text-message" :class="{ self: isSelf }" v-html="convertedContent" @click="handleLinkClick"></div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { escapeHTML, sanitizeHTML } from '../../utils/sanitize'
import { displayMentionTokens } from '../../utils/mentions'

const props = defineProps<{
  content: string
  isSelf?: boolean
}>()

const handleLinkClick = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  const link = target.closest('a')
  
  if (link && window.electron?.shell?.openExternal) {
    event.preventDefault()
    const href = link.getAttribute('href')
    if (href) {
      window.electron.shell.openExternal(href)
    }
  }
}

// 转换URL为链接的函数
// 注意：必须先对内容进行 HTML 转义，然后再插入链接，防止 XSS 攻击
const convertUrlsToLinks = (text: string): string => {
  const linkify = (plainText: string): string => {
    const escapedText = escapeHTML(plainText)
    const urlRegex = /(https?:\/\/[\w\-._~:/?#[\]@!$&'()*+,;=.]+)/g
    let result = ''
    let lastIndex = 0
    for (const match of escapedText.matchAll(urlRegex)) {
      const rawURL = match[0]
      const trailingPunctuation = rawURL.match(/[.,:;!?]+$/)?.[0] || ''
      const url = rawURL.slice(0, rawURL.length - trailingPunctuation.length)
      const index = match.index ?? 0
      result += escapedText.slice(lastIndex, index)
      result += `<a href="${url}" target="_blank" rel="noopener noreferrer" class="message-link">${url}</a>`
      result += trailingPunctuation
      lastIndex = index + rawURL.length
    }
    return result + escapedText.slice(lastIndex)
  }

  const mentionTokenPattern = /@\{mention:(?:all|[1-9]\d*)(?:\|[^}]+)?\}/g
  let result = ''
  let lastIndex = 0
  for (const match of text.matchAll(mentionTokenPattern)) {
    const token = match[0]
    const displayText = displayMentionTokens(token)
    if (displayText === token) continue
    const index = match.index ?? 0
    result += linkify(text.slice(lastIndex, index))
    result += `<span class="at-user">${escapeHTML(displayText)}</span>`
    lastIndex = index + token.length
  }
  return result + linkify(text.slice(lastIndex))
}

const convertedContent = computed(() => {
  // 使用 sanitizeHTML 确保输出安全
  return sanitizeHTML(convertUrlsToLinks(props.content))
})
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
</style>
