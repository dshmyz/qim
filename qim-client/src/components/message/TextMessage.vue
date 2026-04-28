<template>
  <div class="message-bubble text-message" :class="{ self: isSelf }" v-html="convertedContent"></div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { escapeHTML, sanitizeHTML } from '../../utils/sanitize'

const props = defineProps<{
  content: string
  isSelf?: boolean
}>()

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
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

.message-link {
  color: #3b82f6;
  text-decoration: none;
  font-weight: 500;
  transition: all 0.3s ease;
}

.message-link:hover {
  color: #2563eb;
  text-decoration: underline;
  transform: translateY(-1px);
}

.at-user {
  color: #3b82f6;
  font-weight: 600;
  background-color: rgba(59, 130, 246, 0.1);
  padding: 2px 6px;
  border-radius: 4px;
  transition: all 0.3s ease;
}

.at-user:hover {
  background-color: rgba(59, 130, 246, 0.2);
  transform: translateY(-1px);
}

/* 自己的文本消息样式 */
.text-message.self {
  background: var(--primary-color);
  color: white;
  border: none;
}

.text-message.self .message-link {
  color: #e3f2fd;
}

.text-message.self .message-link:hover {
  color: white;
  text-decoration: underline;
}

.text-message.self .at-user {
  color: #e3f2fd;
  background-color: rgba(255, 255, 255, 0.1);
}

.text-message.self .at-user:hover {
  background-color: rgba(255, 255, 255, 0.2);
}

.text-message.self ::selection {
  background: rgba(0, 0, 0, 0.25);
  color: white;
}
</style>