<template>
  <div class="markdown-message" v-html="renderedContent"></div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { marked } from 'marked'

const props = defineProps<{
  content: string
  isSelf: boolean
}>()

// 使用marked库渲染markdown
const renderedContent = computed(() => {
  return marked(props.content)
})
</script>

<style scoped>
.markdown-message {
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
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
  border-radius: 4px;
  margin: 10px 0;
  overflow-x: auto;
  font-family: 'Courier New', Courier, monospace;
  font-size: 14px;
  line-height: 1.4;
  color: var(--text-color);
}

.markdown-message code {
  background-color: var(--hover-color);
  padding: 2px 4px;
  border-radius: 3px;
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
</style>