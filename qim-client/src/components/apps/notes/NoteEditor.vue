<template>
  <div class="note-editor" ref="editorContainerRef">
    <input
      v-model="localTitle"
      class="note-title-input"
      placeholder="笔记标题"
      @input="$emit('update:title', localTitle)"
    />
    <div class="editor-body">
      <div
        v-show="layoutMode === 'edit' || layoutMode === 'split'"
        class="editor-pane"
        :class="{ 'split-mode': layoutMode === 'split' }"
      >
        <div ref="codemirrorRef" class="codemirror-container"></div>
      </div>
      <div
        v-show="layoutMode === 'preview' || layoutMode === 'split'"
        class="preview-pane"
        :class="{ 'split-mode': layoutMode === 'split' }"
      >
        <MarkdownRenderer :content="localContent" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, shallowRef, nextTick } from 'vue'
import { EditorView, keymap, lineNumbers, highlightActiveLine, highlightActiveLineGutter, drawSelection, rectangularSelection, crosshairCursor, highlightSpecialChars } from '@codemirror/view'
import { EditorState, Compartment } from '@codemirror/state'
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
import { markdown, markdownLanguage } from '@codemirror/lang-markdown'
import { languages } from '@codemirror/language-data'
import { syntaxHighlighting, indentOnInput, bracketMatching, foldGutter, foldKeymap, defaultHighlightStyle, HighlightStyle } from '@codemirror/language'
import { oneDark } from '@codemirror/theme-one-dark'
import { closeBrackets, closeBracketsKeymap } from '@codemirror/autocomplete'
import { searchKeymap, highlightSelectionMatches } from '@codemirror/search'
import { tags } from '@lezer/highlight'
import MarkdownRenderer from '../../shared/MarkdownRenderer.vue'

const props = defineProps<{
  title: string
  content: string
  mode: 'edit' | 'split' | 'preview'
}>()

const emit = defineEmits<{
  'update:title': [title: string]
  'update:content': [content: string]
  save: []
}>()

const localTitle = ref(props.title)
const localContent = ref(props.content)
const codemirrorRef = ref<HTMLElement | null>(null)
const editorContainerRef = ref<HTMLElement | null>(null)
const editorView = shallowRef<EditorView | null>(null)
const themeCompartment = new Compartment()

type LayoutMode = 'edit' | 'split' | 'preview'
const layoutMode = ref<LayoutMode>('edit')

watch(() => props.title, (val) => { localTitle.value = val })
watch(() => props.content, (val) => {
  if (val !== localContent.value) {
    localContent.value = val
    if (editorView.value && editorView.value.state.doc.toString() !== val) {
      editorView.value.dispatch({
        changes: { from: 0, to: editorView.value.state.doc.length, insert: val }
      })
    }
  }
})

watch(() => props.mode, (val) => {
  layoutMode.value = val
})

const noteEditorTheme = EditorView.theme({
  '&': {
    height: '100%',
    fontSize: '14px',
  },
  '.cm-content': {
    fontFamily: 'var(--font-family-mono, "SF Mono", "Fira Code", "Consolas", monospace)',
    lineHeight: '1.6',
    padding: '12px 0',
  },
  '.cm-cursor': {
    borderLeftColor: 'var(--primary-color, #3385ff)',
    borderLeftWidth: '2px',
  },
  '.cm-activeLine': {
    backgroundColor: 'rgba(51, 133, 255, 0.06)',
  },
  '.cm-activeLineGutter': {
    backgroundColor: 'rgba(51, 133, 255, 0.06)',
  },
  '.cm-gutters': {
    backgroundColor: 'var(--card-bg, #fff)',
    color: 'var(--text-secondary, #999)',
    border: 'none',
    borderRight: '1px solid var(--border-color, #e5e5e5)',
    minWidth: '36px',
  },
  '.cm-lineNumbers .cm-gutterElement': {
    fontSize: '12px',
    padding: '0 8px',
  },
  '.cm-scroller': {
    overflow: 'auto',
  },
  '&.cm-focused': {
    outline: 'none',
  },
  '.cm-selectionBackground, &.cm-focused .cm-selectionBackground': {
    backgroundColor: 'rgba(51, 133, 255, 0.2) !important',
  },
  '.cm-foldGutter .cm-gutterElement': {
    cursor: 'pointer',
    color: 'var(--text-secondary, #999)',
  },
})

const markdownHighlightStyle = HighlightStyle.define([
  { tag: tags.heading1, fontSize: '1.4em', fontWeight: 'bold', color: 'var(--text-color, #333)' },
  { tag: tags.heading2, fontSize: '1.2em', fontWeight: 'bold', color: 'var(--text-color, #333)' },
  { tag: tags.heading3, fontSize: '1.1em', fontWeight: 'bold', color: 'var(--text-color, #333)' },
  { tag: tags.emphasis, fontStyle: 'italic' },
  { tag: tags.strong, fontWeight: 'bold' },
  { tag: tags.strikethrough, textDecoration: 'line-through' },
  { tag: tags.link, color: 'var(--primary-color, #3385ff)', textDecoration: 'underline' },
  { tag: tags.url, color: 'var(--primary-color, #3385ff)' },
  { tag: tags.monospace, fontFamily: 'var(--font-family-mono, monospace)', color: 'var(--color-error-600, #e53935)' },
  { tag: tags.quote, color: 'var(--text-secondary, #666)', fontStyle: 'italic' },
  { tag: tags.comment, color: 'var(--text-secondary, #999)', fontStyle: 'italic' },
  { tag: tags.meta, color: 'var(--text-secondary, #999)' },
  { tag: tags.processingInstruction, color: 'var(--text-secondary, #999)' },
])

function createEditor() {
  if (!codemirrorRef.value) return

  const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches

  const updateListener = EditorView.updateListener.of((update) => {
    if (update.docChanged) {
      const newContent = update.state.doc.toString()
      localContent.value = newContent
      emit('update:content', newContent)
    }
  })

  const state = EditorState.create({
    doc: localContent.value,
    extensions: [
      lineNumbers(),
      highlightActiveLineGutter(),
      highlightSpecialChars(),
      history(),
      foldGutter(),
      drawSelection(),
      indentOnInput(),
      bracketMatching(),
      closeBrackets(),
      highlightActiveLine(),
      highlightSelectionMatches(),
      rectangularSelection(),
      crosshairCursor(),
      EditorView.lineWrapping,
      markdown({ base: markdownLanguage, codeLanguages: languages }),
      syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
      syntaxHighlighting(markdownHighlightStyle),
      noteEditorTheme,
      themeCompartment.of(isDark ? oneDark : []),
      keymap.of([
        ...closeBracketsKeymap,
        ...defaultKeymap,
        ...searchKeymap,
        ...historyKeymap,
        ...foldKeymap,
        indentWithTab,
        { key: 'Mod-b', run: () => { insertFormat('**', '**'); return true } },
        { key: 'Mod-i', run: () => { insertFormat('*', '*'); return true } },
        { key: 'Mod-k', run: () => { insertLink(); return true } },
        { key: 'Mod-s', run: () => { emit('save'); return true } },
      ]),
      updateListener,
      EditorView.editable.of(true),
    ],
  })

  editorView.value = new EditorView({
    state,
    parent: codemirrorRef.value,
  })
}

function insertFormat(prefix: string, suffix: string) {
  if (!editorView.value) return
  const view = editorView.value
  const { from, to } = view.state.selection.main
  const selectedText = view.state.sliceDoc(from, to)

  view.dispatch({
    changes: { from, to, insert: prefix + selectedText + suffix },
    selection: { anchor: from + prefix.length, head: from + prefix.length + selectedText.length },
  })
  view.focus()
}

function insertLink() {
  if (!editorView.value) return
  const view = editorView.value
  const { from, to } = view.state.selection.main
  const selectedText = view.state.sliceDoc(from, to)

  if (selectedText) {
    view.dispatch({
      changes: { from, to, insert: `[${selectedText}](url)` },
      selection: { anchor: from + selectedText.length + 3, head: from + selectedText.length + 6 },
    })
  } else {
    view.dispatch({
      changes: { from, to, insert: '[链接文本](url)' },
      selection: { anchor: from + 1, head: from + 5 },
    })
  }
  view.focus()
}

function handleThemeChange(e: MediaQueryListEvent) {
  if (!editorView.value) return
  const isDark = e.matches
  editorView.value.dispatch({
    effects: themeCompartment.reconfigure(isDark ? oneDark : []),
  })
}

defineExpose({ insertFormat, insertLink })

let themeQuery: MediaQueryList | null = null

onMounted(() => {
  nextTick(() => {
    createEditor()
  })

  themeQuery = window.matchMedia('(prefers-color-scheme: dark)')
  themeQuery.addEventListener('change', handleThemeChange)
})

onUnmounted(() => {
  if (editorView.value) {
    editorView.value.destroy()
    editorView.value = null
  }
  if (themeQuery) {
    themeQuery.removeEventListener('change', handleThemeChange)
  }
})
</script>

<style scoped>
.note-editor {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: var(--spacing-3);
  overflow: hidden;
}

.note-title-input {
  padding: var(--spacing-1) var(--spacing-4);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  font-size: var(--font-size-lg);
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

.editor-body {
  flex: 1;
  display: flex;
  overflow: hidden;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  background: var(--card-bg);
  box-shadow: var(--shadow-xs);
}

.editor-pane {
  flex: 1;
  display: flex;
  overflow: hidden;
  min-width: 0;
}

.editor-pane.split-mode {
  border-right: 1px solid var(--border-color);
}

.codemirror-container {
  flex: 1;
  overflow: hidden;
}

.codemirror-container :deep(.cm-editor) {
  height: 100%;
}

.preview-pane {
  flex: 1;
  display: flex;
  overflow: hidden;
  min-width: 0;
  padding: var(--spacing-4);
  overflow-y: auto;
}

.preview-pane :deep(.markdown-renderer) {
  width: 100%;
}
</style>
