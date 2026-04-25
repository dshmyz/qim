<template>
  <div class="streaming-message" :class="{ 'self': isSelf }">
    <div class="message-content">
      <div v-html="renderedContent" class="markdown-content"></div>
      <div v-if="isStreaming" class="typing-indicator">
        <span class="typing-dot"></span>
        <span class="typing-dot"></span>
        <span class="typing-dot"></span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { marked } from 'marked'
import { sanitizeMarkdown } from '../../utils/sanitize'

const props = defineProps<{
  content: string
  isSelf: boolean
  isStreaming: boolean
}>()

// 使用marked库渲染markdown，并进行消毒处理防止XSS攻击
const renderedContent = computed(() => {
  if (!props.content) {
    return ''
  }
  try {
    const result = marked(props.content)
    const html = typeof result === 'string' ? result : String(result)
    // 使用 DOMPurify 进行消毒，防止 XSS 攻击
    return sanitizeMarkdown(html)
  } catch (e) {
    console.error('Markdown render error:', e)
    return props.content
  }
})
</script>

<style scoped>
.streaming-message {
  padding: 10px 14px;
  border-radius: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  max-width: 70%;
  position: relative;
}

.streaming-message.self {
  background: var(--primary-color);
  color: white;
  border-bottom-right-radius: 4px;
  align-self: flex-end;
}

.streaming-message:not(.self) {
  background: var(--sidebar-bg);
  color: var(--text-color);
  border-bottom-left-radius: 4px;
  align-self: flex-start;
}

.markdown-content {
  min-height: 20px;
}

.markdown-content h1,
.markdown-content h2,
.markdown-content h3 {
  margin: 10px 0 5px 0;
  font-weight: 600;
}

.markdown-content h1 {
  font-size: 18px;
}

.markdown-content h2 {
  font-size: 16px;
}

.markdown-content h3 {
  font-size: 14px;
}

.markdown-content strong {
  font-weight: 600;
}

.markdown-content em {
  font-style: italic;
}

.markdown-content pre {
  background: rgba(0, 0, 0, 0.1);
  padding: 8px;
  border-radius: 4px;
  overflow-x: auto;
  margin: 8px 0;
}

.markdown-content code {
  background: rgba(0, 0, 0, 0.1);
  padding: 2px 4px;
  border-radius: 3px;
  font-family: 'Courier New', monospace;
  font-size: 0.9em;
}

.markdown-content pre code {
  background: transparent;
  padding: 0;
  border-radius: 0;
}

.markdown-content a {
  color: var(--primary-color);
  text-decoration: none;
}

.markdown-content a:hover {
  text-decoration: underline;
}

.markdown-content ul,
.markdown-content ol {
  margin: 8px 0;
  padding-left: 20px;
}

.markdown-content li {
  margin: 4px 0;
}

.markdown-content p {
  margin: 6px 0;
}

.typing-indicator {
  display: flex;
  align-items: center;
  margin-top: 5px;
  gap: 3px;
}

.typing-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: currentColor;
  opacity: 0.6;
  animation: typing 1.4s infinite ease-in-out both;
}

.typing-dot:nth-child(1) {
  animation-delay: -0.32s;
}

.typing-dot:nth-child(2) {
  animation-delay: -0.16s;
}

@keyframes typing {
  0%, 80%, 100% {
    transform: scale(0);
  }
  40% {
    transform: scale(1);
  }
}

/* 适配深色主题 */
@media (prefers-color-scheme: dark) {
  .streaming-message:not(.self) {
    background: #333;
    color: #e0e0e0;
  }
  
  .markdown-content pre {
    background: rgba(255, 255, 255, 0.1);
  }
  
  .markdown-content code {
    background: rgba(255, 255, 255, 0.1);
  }
}
</style>