<template>
  <div class="ai-message-content">
    <div v-if="!isExpanded && isLongContent" class="preview-content">
      <div v-html="renderMarkdown(previewText)" class="markdown-content"></div>
    </div>
    <div v-else class="full-content">
      <div v-html="renderMarkdown(content)" class="markdown-content"></div>
    </div>

    <div v-if="isLongContent" class="ai-content-footer">
      <button v-if="!isExpanded" class="expand-btn" @click="isExpanded = true">
        <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
          <path d="M7 10l5 5 5-5z"/>
        </svg>
        &#x5C55;&#x5F00;&#x5168;&#x90E8; (&#x5171; {{ contentLength }} &#x5B57;&#x7B26;)
      </button>
      <div v-else class="expanded-actions">
        <button class="collapse-btn" @click="isExpanded = false">
          <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
            <path d="M7 14l5-5 5 5z"/>
          </svg>
          &#x6536;&#x8D77;
        </button>
        <div class="export-actions">
          <button @click="copyContent">&#x590D;&#x5236;</button>
          <button @click="exportMarkdown">&#x5BFC;&#x51FA; Markdown</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { marked } from 'marked'
import { sanitizeMarkdown } from '../../utils/sanitize'

const props = withDefaults(defineProps<{
  content: string
  maxLength?: number
}>(), {
  maxLength: 500
})

const isExpanded = ref(false)
const previewLines = 5

const previewText = computed(() => {
  const lines = props.content.split('\n').slice(0, previewLines)
  return lines.join('\n')
})

const isLongContent = computed(() => {
  return props.content.length > props.maxLength
})

const contentLength = computed(() => props.content.length)

const renderMarkdown = (text: string): string => {
  try {
    const result = marked(text)
    const html = typeof result === 'string' ? result : String(result)
    return sanitizeMarkdown(html)
  } catch {
    return sanitizeMarkdown(text.replace(/\r\n|\n|\r/g, '<br>'))
  }
}

const copyContent = async () => {
  try {
    await navigator.clipboard.writeText(props.content)
    // &#x53EF;&#x4F7F;&#x7528;&#x5168;&#x5C40; message &#x7EC4;&#x4EF6;&#x63D0;&#x793A;&#x6210;&#x529F;
  } catch {
    // &#x964D;&#x7EA7;&#x5904;&#x7406;
  }
}

const exportMarkdown = () => {
  const blob = new Blob([props.content], { type: 'text/markdown' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'ai-content.md'
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<style scoped>
.ai-message-content {
  width: 100%;
}

.preview-content {
  max-height: 150px;
  overflow: hidden;
  position: relative;
}

.preview-content::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 40px;
  background: linear-gradient(transparent, var(--sidebar-bg));
}

.full-content {
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(-5px); }
  to { opacity: 1; transform: translateY(0); }
}

.markdown-content {
  line-height: 1.6;
  word-break: break-word;
  color: var(--text-color);
}

.markdown-content h1,
.markdown-content h2,
.markdown-content h3 {
  margin: 10px 0 5px 0;
  font-weight: 600;
  color: var(--text-color);
}

.markdown-content h1 { font-size: 18px; }
.markdown-content h2 { font-size: 16px; }
.markdown-content h3 { font-size: 14px; }

.markdown-content pre {
  background: var(--hover-color);
  padding: 8px;
  border-radius: 4px;
  overflow-x: auto;
  margin: 8px 0;
}

.markdown-content code {
  background: var(--hover-color);
  padding: 2px 4px;
  border-radius: 3px;
  font-family: 'Courier New', monospace;
  font-size: 0.9em;
  color: var(--text-color);
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

.ai-content-footer {
  margin-top: 12px;
  padding-top: 8px;
  border-top: 1px solid var(--border-color);
}

.expand-btn, .collapse-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  border: 1px solid var(--border-color);
  background: var(--hover-color);
  color: var(--text-color);
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  transition: all var(--duration-fast) var(--ease-out);
}

.expand-btn:hover, .collapse-btn:hover {
  background: rgba(59, 130, 246, 0.1);
  border-color: var(--primary-color);
}

.expanded-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.export-actions {
  display: flex;
  gap: 8px;
}

.export-actions button {
  padding: 4px 10px;
  border: 1px solid var(--border-color);
  background: transparent;
  color: var(--text-secondary);
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  transition: all var(--duration-fast) var(--ease-out);
}

.export-actions button:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

[data-theme="elegant-dark"] .preview-content::after {
  background: linear-gradient(transparent, #1e1e2e);
}

[data-theme="elegant-dark"] .expand-btn:hover,
[data-theme="elegant-dark"] .collapse-btn:hover {
  background: rgba(59, 130, 246, 0.2);
}
</style>
