<template>
  <div class="note-editor">
    <input
      v-model="localTitle"
      class="note-title-input"
      placeholder="笔记标题"
      @input="$emit('update:title', localTitle)"
    />
    <div v-if="mode === 'edit'" class="editor-area">
      <div class="editor-toolbar">
        <button class="format-btn" @click="insertFormat('**', '**')" title="粗体">
          <strong>B</strong>
        </button>
        <button class="format-btn" @click="insertFormat('*', '*')" title="斜体">
          <em>I</em>
        </button>
        <button class="format-btn" @click="insertFormat('# ', '')" title="标题">
          H
        </button>
        <button class="format-btn" @click="insertFormat('- ', '')" title="列表">
          <i class="fas fa-list"></i>
        </button>
        <button class="format-btn" @click="insertFormat('`', '`')" title="代码">
          <i class="fas fa-code"></i>
        </button>
        <button class="format-btn" @click="insertFormat('```\n', '\n```')" title="代码块">
          <i class="fas fa-file-code"></i>
        </button>
      </div>
      <textarea
        ref="textareaRef"
        v-model="localContent"
        class="note-content-input"
        placeholder="使用 Markdown 编写笔记..."
        @input="$emit('update:content', localContent)"
      ></textarea>
    </div>
    <div v-else class="preview-area">
      <div class="preview-content" v-html="renderedContent"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'

const props = defineProps<{
  title: string
  content: string
  mode: 'edit' | 'preview'
}>()

const emit = defineEmits<{
  'update:title': [title: string]
  'update:content': [content: string]
}>()

const localTitle = ref(props.title)
const localContent = ref(props.content)
const textareaRef = ref<HTMLTextAreaElement | null>(null)

watch(() => props.title, (val) => { localTitle.value = val })
watch(() => props.content, (val) => { localContent.value = val })

const renderedContent = computed(() => {
  return renderMarkdown(localContent.value)
})

function renderMarkdown(content: string): string {
  let html = content
  
  html = html.replace(/^# (.*$)/gm, '<h1>$1</h1>')
  html = html.replace(/^## (.*$)/gm, '<h2>$1</h2>')
  html = html.replace(/^### (.*$)/gm, '<h3>$1</h3>')
  html = html.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
  html = html.replace(/\*(.*?)\*/g, '<em>$1</em>')
  html = html.replace(/```([\s\S]*?)```/g, '<pre><code>$1</code></pre>')
  html = html.replace(/`(.*?)`/g, '<code>$1</code>')
  html = html.replace(/^- (.*$)/gm, '<li>$1</li>')
  html = html.replace(/\[(.*?)\]\((.*?)\)/g, '<a href="$2" target="_blank">$1</a>')
  html = html.replace(/\n/g, '<br>')
  
  return html
}

function insertFormat(prefix: string, suffix: string) {
  if (!textareaRef.value) return
  
  const textarea = textareaRef.value
  const start = textarea.selectionStart
  const end = textarea.selectionEnd
  const selectedText = localContent.value.substring(start, end)
  
  const newContent = 
    localContent.value.substring(0, start) +
    prefix + selectedText + suffix +
    localContent.value.substring(end)
  
  localContent.value = newContent
  emit('update:content', newContent)
  
  setTimeout(() => {
    textarea.focus()
    textarea.setSelectionRange(start + prefix.length, end + prefix.length)
  }, 0)
}
</script>

<style scoped>
.note-editor {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
  overflow: hidden;
}

.note-title-input {
  padding: 12px 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
  background: var(--bg-color);
  outline: none;
  transition: all 0.2s;
}

.note-title-input:focus {
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.editor-area,
.preview-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.editor-toolbar {
  display: flex;
  gap: 4px;
  padding: 8px;
  background: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: 8px 8px 0 0;
  flex-wrap: wrap;
}

.format-btn {
  padding: 6px 12px;
  border: 1px solid var(--border-color);
  background: var(--card-bg);
  color: var(--text-primary);
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s;
}

.format-btn:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.note-content-input {
  flex: 1;
  padding: 16px;
  border: 1px solid var(--border-color);
  border-top: none;
  border-radius: 0 0 8px 8px;
  font-size: 14px;
  font-family: 'Monaco', 'Menlo', monospace;
  line-height: 1.6;
  color: var(--text-primary);
  background: var(--bg-color);
  resize: none;
  outline: none;
}

.note-content-input:focus {
  border-color: var(--primary-color);
}

.preview-area {
  padding: 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--bg-color);
  overflow-y: auto;
}

.preview-content {
  font-size: 14px;
  line-height: 1.6;
  color: var(--text-primary);
}

.preview-content :deep(h1) {
  font-size: 24px;
  margin: 16px 0 8px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-color);
}

.preview-content :deep(h2) {
  font-size: 20px;
  margin: 14px 0 6px;
}

.preview-content :deep(h3) {
  font-size: 18px;
  margin: 12px 0 4px;
}

.preview-content :deep(code) {
  background: var(--code-bg);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'Monaco', monospace;
}

.preview-content :deep(pre) {
  background: var(--code-bg);
  padding: 12px;
  border-radius: 6px;
  overflow-x: auto;
}

.preview-content :deep(a) {
  color: var(--primary-color);
}
</style>
