<template>
  <div ref="containerRef" class="markdown-renderer" v-html="html" @click="handleLinkClick"></div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useMarkdownRender, handleLinkClick } from '../../composables/useMarkdownRender'

const props = withDefaults(
  defineProps<{
    content: string
  }>(),
  {
    content: ''
  }
)

// 统一走 useMarkdownRender（单一渲染管道）。BotChatView/NoteEditor 用纯 markdown，不解码 mention、不替换 emoji。
const { html, containerRef } = useMarkdownRender(
  computed(() => props.content),
  computed(() => ({}))
)
</script>

<style scoped>
.markdown-renderer {
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.markdown-renderer h1 {
  font-size: 1.5em;
  font-weight: 600;
  margin: 1em 0 0.5em 0;
  color: var(--text-color);
}

.markdown-renderer h2 {
  font-size: 1.3em;
  font-weight: 600;
  margin: 0.8em 0 0.4em 0;
  color: var(--text-color);
}

.markdown-renderer h3 {
  font-size: 1.1em;
  font-weight: 600;
  margin: 0.6em 0 0.3em 0;
  color: var(--text-color);
}

.markdown-renderer strong {
  font-weight: 600;
  color: var(--text-color);
}

.markdown-renderer em {
  font-style: italic;
  color: var(--text-color);
}

.markdown-renderer pre {
  background-color: var(--hover-color);
  padding: 12px;
  border-radius: 4px;
  margin: 10px 0;
  overflow-x: auto;
  font-family: 'Courier New', Courier, monospace;
  font-size: 14px;
  line-height: 1.4;
  color: var(--text-color);
}

.markdown-renderer code {
  background-color: var(--hover-color);
  padding: 2px 4px;
  border-radius: 3px;
  font-family: 'Courier New', Courier, monospace;
  font-size: 13px;
  color: var(--text-color);
}

.markdown-renderer pre code {
  background-color: transparent;
  padding: 0;
  border-radius: 0;
}

.markdown-renderer a {
  color: var(--primary-color);
  background: transparent;
  text-decoration: none;
  transition: color 0.2s ease;
}

.markdown-renderer a:hover {
  color: var(--primary-hover);
  text-decoration: underline;
}

.markdown-renderer ul,
.markdown-renderer ol {
  margin: 10px 0;
  padding-left: 20px;
}

.markdown-renderer li {
  margin: 5px 0;
  color: var(--text-color);
}

.markdown-renderer p {
  margin: 10px 0;
  color: var(--text-color);
}

.markdown-renderer blockquote {
  border-left: 4px solid var(--primary-color);
  padding: 8px 16px;
  margin: 10px 0;
  background: var(--hover-color);
  border-radius: 0 6px 6px 0;
  color: var(--text-secondary);
}

.markdown-renderer table {
  width: 100%;
  border-collapse: collapse;
  margin: 10px 0;
}

.markdown-renderer th,
.markdown-renderer td {
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  text-align: left;
}

.markdown-renderer th {
  background: var(--hover-color);
  font-weight: 600;
}

.markdown-renderer .hljs {
  background: transparent !important;
  padding: 0 !important;
}
</style>
