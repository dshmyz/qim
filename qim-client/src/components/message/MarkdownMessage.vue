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

<style>
.markdown-message {
  padding: 10px 14px;
  border-radius: 12px;
  background: var(--sidebar-bg);
  color: var(--text-color);
  font-size: 14px;
  line-height: 1.6;
  word-break: break-word;
}

.markdown-message h1,
.markdown-message h2,
.markdown-message h3 {
  margin: 8px 0 4px 0;
  font-weight: 700;
  color: var(--text-color);
}

.markdown-message h1:first-child,
.markdown-message h2:first-child,
.markdown-message h3:first-child {
  margin-top: 0;
}

.markdown-message h1 { font-size: 1.4em; }
.markdown-message h2 { font-size: 1.2em; }
.markdown-message h3 { font-size: 1.05em; }

.markdown-message pre {
  background: var(--hover-color);
  padding: 8px 10px;
  border-radius: 6px;
  overflow-x: auto;
  font-family: 'SF Mono', 'Fira Code', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.5;
  color: var(--text-color);
  margin: 8px 0;

}

.markdown-message code {
  background: var(--hover-color);
  padding: 2px 5px;
  border-radius: 3px;
  font-family: 'SF Mono', 'Fira Code', 'Courier New', monospace;
  font-size: 0.88em;
  color: var(--primary-color);
}

.markdown-message pre code {
  background: transparent;
  padding: 0;
  border-radius: 0;
  color: var(--text-color);
  font-size: 13px;
}

.markdown-message blockquote {
  border-left: 3px solid var(--primary-color);
  padding-left: 12px;
  margin: 8px 0;
  color: var(--text-secondary);
  opacity: 0.9;
}

.markdown-message a {
  color: var(--primary-color);
  text-decoration: none;
}

.markdown-message a:hover {
  text-decoration: underline;
}

.markdown-message ul,
.markdown-message ol {
  margin: 6px 0;
  padding-left: 20px;
}

.markdown-message li {
  margin: 2px 0;
  line-height: 1.6;
  color: var(--text-color);
}

.markdown-message p {
  margin: 4px 0;
  line-height: 1.6;
  color: var(--text-color);
}

.markdown-message p:first-child {
  margin-top: 0;
}

.markdown-message p:last-child {
  margin-bottom: 0;
}

.markdown-message strong {
  font-weight: 700;
  color: var(--text-color);
}

.markdown-message em {
  font-style: italic;
}

.markdown-message hr {
  border: none;
  border-top: 1px solid var(--border-color);
  margin: 10px 0;
}

.markdown-message table {
  border-collapse: collapse;
  margin: 8px 0;
  width: 100%;
}

.markdown-message th,
.markdown-message td {
  border: 1px solid var(--border-color);
  padding: 4px 8px;
  font-size: 13px;
}

.markdown-message th {
  background: var(--hover-color);
  font-weight: 600;
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
.markdown-message.self li,
.markdown-message.self p {
  color: white;
}

.markdown-message.self code {
  background-color: rgba(255, 255, 255, 0.2);
  color: white;
}

.markdown-message.self pre {
  background-color: rgba(255, 255, 255, 0.12);
  color: white;
  border-color: rgba(255, 255, 255, 0.2);
}

.markdown-message.self blockquote {
  border-left-color: rgba(255, 255, 255, 0.5);
  color: rgba(255, 255, 255, 0.8);
}

.markdown-message.self a {
  color: #e3f2fd;
}

.markdown-message.self a:hover {
  color: white;
}

.markdown-message.self ::selection {
  background: rgba(0, 0, 0, 0.25);
  color: white;
}

.markdown-message.self th {
  background: rgba(255, 255, 255, 0.15);
}

.markdown-message.self th,
.markdown-message.self td {
  border-color: rgba(255, 255, 255, 0.2);
}
</style>