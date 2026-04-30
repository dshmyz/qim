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
  gap: var(--spacing-4);
  overflow: hidden;
}

.note-title-input {
  padding: var(--spacing-3) var(--spacing-4);
  border: 2px solid var(--border-color);
  border-radius: var(--radius-lg);
  font-size: var(--font-size-xl);
  font-weight: var(--font-weight-bold);
  color: var(--text-color);
  background: var(--card-bg);
  outline: none;
  transition: all var(--transition-base);
  box-shadow: var(--shadow-xs);
}

.note-title-input:focus {
  border-color: var(--primary-color);
  box-shadow: 0 0 0 4px rgba(51, 133, 255, 0.15);
}

.note-title-input::placeholder {
  color: var(--text-secondary);
  font-weight: var(--font-weight-normal);
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
  gap: var(--spacing-1);
  padding: var(--spacing-2);
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg) var(--radius-lg) 0 0;
  flex-wrap: wrap;
  border-bottom: none;
}

.format-btn {
  padding: var(--spacing-2) var(--spacing-3);
  border: 1px solid var(--border-color);
  background: var(--btn-bg);
  color: var(--text-color);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  transition: all var(--transition-fast);
  min-width: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.format-btn:hover {
  background: var(--primary-light);
  border-color: var(--primary-color);
  color: var(--primary-color);
  transform: translateY(-1px);
}

.format-btn:active {
  transform: translateY(0);
}

.note-content-input {
  flex: 1;
  padding: var(--spacing-4);
  border: 1px solid var(--border-color);
  border-top: none;
  border-radius: 0 0 var(--radius-lg) var(--radius-lg);
  font-size: var(--font-size-sm);
  font-family: var(--font-family-mono);
  line-height: var(--line-height-relaxed);
  color: var(--text-color);
  background: var(--card-bg);
  resize: none;
  outline: none;
  transition: border-color var(--transition-base);
}

.note-content-input:focus {
  border-color: var(--primary-color);
}

.note-content-input::placeholder {
  color: var(--text-secondary);
}

.preview-area {
  padding: var(--spacing-4);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  background: var(--card-bg);
  overflow-y: auto;
  box-shadow: var(--shadow-xs);
}

.preview-content {
  font-size: var(--font-size-sm);
  line-height: var(--line-height-relaxed);
  color: var(--text-color);
}

.preview-content :deep(h1) {
  font-size: var(--font-size-2xl);
  font-weight: var(--font-weight-bold);
  margin: var(--spacing-4) 0 var(--spacing-2);
  padding-bottom: var(--spacing-2);
  border-bottom: 2px solid var(--border-color);
  color: var(--text-color);
}

.preview-content :deep(h2) {
  font-size: var(--font-size-xl);
  font-weight: var(--font-weight-semibold);
  margin: var(--spacing-4) 0 var(--spacing-2);
  color: var(--text-color);
}

.preview-content :deep(h3) {
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-semibold);
  margin: var(--spacing-3) 0 var(--spacing-2);
  color: var(--text-color);
}

.preview-content :deep(code) {
  background: var(--list-bg);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  font-family: var(--font-family-mono);
  font-size: var(--font-size-xs);
  color: var(--color-error-600);
}

.preview-content :deep(pre) {
  background: var(--list-bg);
  padding: var(--spacing-3);
  border-radius: var(--radius-md);
  overflow-x: auto;
  border: 1px solid var(--border-color);
  margin: var(--spacing-3) 0;
}

.preview-content :deep(pre code) {
  background: transparent;
  padding: 0;
  color: var(--text-color);
}

.preview-content :deep(a) {
  color: var(--primary-color);
  text-decoration: none;
  font-weight: var(--font-weight-medium);
}

.preview-content :deep(a:hover) {
  text-decoration: underline;
}

.preview-content :deep(li) {
  margin: var(--spacing-1) 0;
  padding-left: var(--spacing-2);
}

.preview-content :deep(strong) {
  font-weight: var(--font-weight-semibold);
}
</style>
