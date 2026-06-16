<template>
  <div class="message-bubble text-message" :class="{ self: isSelf }" v-html="convertedContent" @click="handleLinkClick"></div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { escapeHTML, sanitizeHTML } from '../../utils/sanitize'

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
  // 先对用户输入进行 HTML 转义，防止 XSS 攻击
  const escapedText = escapeHTML(text)
  
  // 正则表达式匹配URL（在转义后的文本中）
  const urlRegex = /(https?:\/\/[\w\-._~:/?#[\]@!$&'()*+,;=.]+)/g
  // 正则表达式匹配@用户
  const atRegex = /@([\u4e00-\u9fa5\w]+)/g
  
  let result = escapedText
  
  // 处理URL - 注意转义后的 URL 中的特殊字符已被转义
  result = result.replace(urlRegex, (url) => {
    return `<a href="${url}" target="_blank" rel="noopener noreferrer" class="message-link">${url}</a>`
  })
  
  // 处理@用户
  result = result.replace(atRegex, (match, username) => {
    return `<span class="at-user">@${username}</span>`
  })
  
  return result
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