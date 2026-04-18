<template>
  <div class="message-bubble text-message" :class="{ self: isSelf }" v-html="convertedContent"></div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  content: string
  isSelf?: boolean
}>()

// 转换URL为链接的函数
const convertUrlsToLinks = (text: string): string => {
  // 正则表达式匹配URL
  const urlRegex = /(https?:\/\/[\w\-._~:/?#[\]@!$&'()*+,;=.]+)/g
  // 正则表达式匹配@用户
  const atRegex = /@([\u4e00-\u9fa5\w]+)/g
  
  let result = text
  
  // 先处理URL
  result = result.replace(urlRegex, (url) => {
    return `<a href="${url}" target="_blank" rel="noopener noreferrer" class="message-link">${url}</a>`
  })
  
  // 再处理@用户
  result = result.replace(atRegex, (match, username) => {
    return `<span class="at-user">@${username}</span>`
  })
  
  return result
}

const convertedContent = computed(() => {
  return convertUrlsToLinks(props.content)
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
</style>