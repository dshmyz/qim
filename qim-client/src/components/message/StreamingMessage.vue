<template>
  <div class="streaming-message" :class="{ 'self': isSelf }" @click="handleLinkClick">
    <div class="message-content">
      <div ref="containerRef" v-html="renderedContent" class="markdown-content"></div>
      <div v-if="isStreaming && content" class="typing-indicator">
        <span class="typing-dot"></span>
        <span class="typing-dot"></span>
        <span class="typing-dot"></span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { marked } from 'marked'
import { sanitizeMarkdown } from '../../utils/sanitize'
import { emojiToHtml, classicToHtml } from '../../utils/emoji'
import { useCodeHighlight } from '../../composables/useCodeHighlight'

const props = defineProps<{
  content: string
  isSelf: boolean
  isStreaming: boolean
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

// 流式过程中 markdown 不完整（代码块/行内代码未闭合），marked 解析结果
// 与最终全文不一致。预处理：自动闭合未关闭的代码块和行内反引号。
// 段落分隔不在此处理——GFM 模式下 marked 自身会正确处理换行。
function normalizePartialMarkdown(md: string): string {
  // 闭合未关闭的 ``` 代码块
  const fenceCount = (md.match(/^```/gm) || []).length
  if (fenceCount % 2 !== 0) {
    md += '\n```'
  }

  // 闭合未关闭的行内 ` 反引号
  const backtickSegments = md.split('`')
  if (backtickSegments.length % 2 === 0) {
    md += '`'
  }

  return md
}

// 使用marked库渲染markdown，并进行消毒处理防止XSS攻击
const renderedContent = computed(() => {
  // AI 还没吐第一个字（content 空 + 仍在流）：显示「思考中」+ dots 同行占位，首段到达后自然替换
  if (!props.content) {
    return props.isStreaming
      ? '<span class="thinking-placeholder">思考中</span><span class="typing-indicator-inline"><span class="typing-dot"></span><span class="typing-dot"></span><span class="typing-dot"></span></span>'
      : ''
  }
  try {
    const md = props.isStreaming ? normalizePartialMarkdown(props.content) : props.content
    const result = marked(md)
    const html = typeof result === 'string' ? result : String(result)
    // 使用 DOMPurify 进行消毒，防止 XSS 攻击，再把表情字符/经典标记替换为 <img>
    return classicToHtml(emojiToHtml(sanitizeMarkdown(html)))
  } catch (e) {
    console.error('Markdown render error:', e)
    return props.content
  }
})

const containerRef = ref<HTMLElement | null>(null)
useCodeHighlight(containerRef, renderedContent)
</script>

<style>
/* 非 scoped：v-html 注入的 markdown 元素（h1/pre/code…）不带 scoped 的 data-v 属性，
   scoped 下这些选择器匹配不上 -> 标题等回退到浏览器默认（h1=2em=28px），流式中文字
   异常偏大，收尾切到 MarkdownMessage 才正常。改非 scoped 与 MarkdownMessage 对齐，
   让 .markdown-content 下的 markdown 元素样式真正生效。
   注：MarkdownMessage 同为非 scoped 且工作正常，两组件渲染同一份 markdown。 */
.streaming-message {
  padding: 10px 14px;
  border-radius: 12px;
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  width: 100%;
  max-width: 80%;
  position: relative;
}

.streaming-message.self {
  background: var(--hover-color);
  background: color-mix(in srgb, var(--primary-color), white 88%);
  color: var(--text-color);
  border-bottom-right-radius: 4px;
  align-self: flex-end;
}

[data-theme="elegant-dark"] .streaming-message.self {
  background: var(--primary-color);
  color: white;
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
  margin: 8px 0 4px 0;
  font-weight: 700;
  color: var(--text-color);
}

.markdown-content h1:first-child,
.markdown-content h2:first-child,
.markdown-content h3:first-child {
  margin-top: 0;
}

.markdown-content h1 { font-size: 1.4em; }
.markdown-content h2 { font-size: 1.2em; }
.markdown-content h3 { font-size: 1.05em; }

.markdown-content strong {
  font-weight: 700;
  color: var(--text-color);
}

.markdown-content em {
  font-style: italic;
}

.markdown-content pre {
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

.markdown-content code {
  background: var(--hover-color);
  padding: 2px 5px;
  border-radius: 3px;
  font-family: 'SF Mono', 'Fira Code', 'Courier New', monospace;
  font-size: 0.88em;
  color: var(--primary-color);
}

.markdown-content pre code {
  background: transparent;
  padding: 0;
  border-radius: 0;
  color: var(--text-color);
  font-size: 13px;
}

.markdown-content a {
  color: var(--primary-color);
  background: transparent;
  text-decoration: none;
  transition: color 0.2s ease;
}

.markdown-content a:hover {
  text-decoration: underline;
}

.markdown-content ul,
.markdown-content ol {
  margin: 6px 0;
  padding-left: 20px;
}

.markdown-content li {
  margin: 2px 0;
  line-height: 1.6;
  color: var(--text-color);
}

.markdown-content p {
  margin: 4px 0;
  line-height: 1.6;
  color: var(--text-color);
}

.markdown-content p:first-child {
  margin-top: 0;
}

.markdown-content p:last-child {
  margin-bottom: 0;
}

.typing-indicator {
  display: flex;
  align-items: center;
  margin-top: 5px;
  gap: 3px;
}

/* 思考中 + dots 同行：全部通过 v-html 内联渲染，不依赖 flex 布局，
   彻底避免父级 scoped min-width:0 压缩容器导致 CJK 竖排。 */
.typing-indicator-inline {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  margin-left: 6px;
  vertical-align: baseline;
}
.typing-indicator-inline .typing-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: currentColor;
  opacity: 0.5;
  animation: typing 1.4s infinite ease-in-out both;
}
.typing-indicator-inline .typing-dot:nth-child(1) { animation-delay: -0.32s; }
.typing-indicator-inline .typing-dot:nth-child(2) { animation-delay: -0.16s; }

/* AI 尚未吐第一个字时的「思考中」占位，弱化呈现，首段到达后由正文替换。
   不用斜体（中文斜体观感差）、不用省略号（右侧已有 typing dots 表示进行中），
   改用次级色 + 略小字号，与气泡正文协调。 */
.streaming-message .thinking-placeholder {
  color: color-mix(in srgb, var(--text-color), transparent 45%);
  font-size: 0.92em;
  letter-spacing: 0.5px;
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

.markdown-content .hljs {
  background: transparent !important;
  padding: 0 !important;
}
</style>