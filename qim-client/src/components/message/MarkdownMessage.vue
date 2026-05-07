<template>
  <div class="markdown-message" :class="{ self: isSelf }" v-html="renderedContent" @click="handleLinkClick"></div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { marked } from 'marked'
import { sanitizeMarkdown } from '../../utils/sanitize'

const props = defineProps<{
  content: string
  isSelf: boolean
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

// 使用marked库渲染markdown，并进行消毒处理防止XSS攻击
const renderedContent = computed(() => {
  const html = marked(props.content)
  const htmlString = typeof html === 'string' ? html : String(html)
  // 使用 DOMPurify 进行消毒，防止 XSS 攻击
  return sanitizeMarkdown(htmlString)
})
</script>

<style scoped>
.markdown-message {
  padding: 10px 14px;
  border-radius: 16px;
  background: #ffffff;
  color: var(--text-color);
  font-size: 15px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}

.markdown-message h1 {
  font-size: 1.5em;
  font-weight: 600;
  margin: 1em 0 0.5em 0;
  color: var(--text-color);
}

.markdown-message h2 {
  font-size: 1.3em;
  font-weight: 600;
  margin: 0.8em 0 0.4em 0;
  color: var(--text-color);
}

.markdown-message h3 {
  font-size: 1.1em;
  font-weight: 600;
  margin: 0.6em 0 0.3em 0;
  color: var(--text-color);
}

.markdown-message strong {
  font-weight: 600;
  color: var(--text-color);
}

.markdown-message em {
  font-style: italic;
  color: var(--text-color);
}

.markdown-message pre {
  background-color: var(--hover-color);
  padding: 12px;
  border-radius: 8px;
  margin: 10px 0;
  overflow-x: auto;
  font-family: 'Courier New', Courier, monospace;
  font-size: 14px;
  line-height: 1.4;
  color: var(--text-color);
}

.markdown-message code {
  background-color: var(--hover-color);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'Courier New', Courier, monospace;
  font-size: 13px;
  color: var(--text-color);
}

.markdown-message pre code {
  background-color: transparent;
  padding: 0;
  border-radius: 0;
}

.markdown-message a {
  color: var(--primary-color);
  text-decoration: none;
  transition: color 0.2s;
}

.markdown-message a:hover {
  color: var(--primary-hover);
  text-decoration: underline;
}

.markdown-message ul,
.markdown-message ol {
  margin: 10px 0;
  padding-left: 20px;
}

.markdown-message li {
  margin: 5px 0;
  color: var(--text-color);
}

.markdown-message p {
  margin: 10px 0;
  color: var(--text-color);
}

/* 炫酷黑主题适配 */
[data-theme="elegant-dark"] .markdown-message {
  background: #1e1e1e;
  color: #e5e7eb;
}

[data-theme="elegant-dark"] .markdown-message h1,
[data-theme="elegant-dark"] .markdown-message h2,
[data-theme="elegant-dark"] .markdown-message h3,
[data-theme="elegant-dark"] .markdown-message strong,
[data-theme="elegant-dark"] .markdown-message em,
[data-theme="elegant-dark"] .markdown-message li,
[data-theme="elegant-dark"] .markdown-message p {
  color: #e5e7eb;
}

[data-theme="elegant-dark"] .markdown-message code {
  background-color: rgba(255, 255, 255, 0.08);
  color: #e5e7eb;
}

[data-theme="elegant-dark"] .markdown-message pre {
  background-color: rgba(255, 255, 255, 0.05);
  color: #e5e7eb;
}

/* 自己发送的 Markdown 消息 */
.markdown-message.self {
  background: var(--primary-color);
  color: white;
}

.markdown-message.self h1,
.markdown-message.self h2,
.markdown-message.self h3,
.markdown-message.self strong,
.markdown-message.self em,
.markdown-message.self li,
.markdown-message.self p {
  color: white;
}

.markdown-message.self code {
  background-color: rgba(255, 255, 255, 0.15);
  color: white;
}

.markdown-message.self pre {
  background-color: rgba(255, 255, 255, 0.1);
  color: white;
}

.markdown-message.self a {
  color: #e3f2fd;
}

.markdown-message.self a:hover {
  color: white;
}
</style>