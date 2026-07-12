<template>
  <QDialog
    :visible="visible"
    @update:visible="$emit('update:visible', $event)"
    title="发送代码块"
    width="720px"
    :close-on-click-mask="false"
  >
    <div class="code-block-editor">
      <div class="code-block-editor__toolbar">
        <label class="code-block-editor__lang-label">语言：</label>
        <select v-model="selectedLanguage" class="code-block-editor__lang-select">
          <option value="">纯文本</option>
          <option v-for="lang in languageOptions" :key="lang" :value="lang">{{ lang }}</option>
        </select>
      </div>
      <div ref="codemirrorRef" class="code-block-editor__codemirror"></div>
    </div>
    <template #footer>
      <button class="code-block-editor__btn code-block-editor__btn--cancel" @click="handleCancel">取消</button>
      <button
        class="code-block-editor__btn code-block-editor__btn--confirm"
        :disabled="!code.trim()"
        @click="handleConfirm"
      >发送</button>
    </template>
  </QDialog>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, nextTick, shallowRef } from 'vue'
import { EditorView, keymap, lineNumbers, highlightActiveLine, highlightActiveLineGutter, drawSelection, rectangularSelection, crosshairCursor, highlightSpecialChars } from '@codemirror/view'
import { EditorState, Compartment } from '@codemirror/state'
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
import { defaultHighlightStyle, syntaxHighlighting, bracketMatching, indentOnInput, foldGutter, foldKeymap, LanguageDescription } from '@codemirror/language'
import { closeBrackets, closeBracketsKeymap } from '@codemirror/autocomplete'
import { searchKeymap, highlightSelectionMatches } from '@codemirror/search'
import { oneDark } from '@codemirror/theme-one-dark'
import QDialog from '../shared/QDialog.vue'
import { formatCodeBlock } from '../../utils/codeBlock'

interface Props {
  visible: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'confirm': [markdown: string]
}>()

const languageOptions = [
  'javascript', 'typescript', 'python', 'go', 'java', 'c', 'cpp', 'csharp',
  'rust', 'ruby', 'php', 'swift', 'kotlin', 'sql', 'html', 'css', 'json',
  'yaml', 'bash', 'shell', 'markdown',
]

const selectedLanguage = ref('javascript')
const code = ref('')
const codemirrorRef = ref<HTMLElement | null>(null)
const editorView = shallowRef<EditorView | null>(null)
const langCompartment = new Compartment()
const themeCompartment = new Compartment()

const editorTheme = EditorView.theme({
  '&': { height: '360px', fontSize: '13px' },
  '.cm-content': {
    fontFamily: 'var(--font-family-mono, "SF Mono", "Fira Code", "Consolas", monospace)',
    lineHeight: '1.5',
    padding: '8px 0',
  },
  '.cm-scroller': { overflow: 'auto' },
  '&.cm-focused': { outline: 'none' },
  '.cm-gutters': {
    backgroundColor: 'var(--card-bg, #fff)',
    color: 'var(--text-secondary, #999)',
    border: 'none',
    borderRight: '1px solid var(--border-color, #e5e5e5)',
  },
})

function applyLanguage(langName: string) {
  if (!editorView.value) return
  if (!langName) {
    editorView.value.dispatch({ effects: langCompartment.reconfigure([]) })
    return
  }
  const desc = LanguageDescription.matchLanguageName(langName)
  if (desc) {
    desc.load().then((support) => {
      editorView.value?.dispatch({ effects: langCompartment.reconfigure(support) })
    }).catch(() => {})
  }
}

function createEditor() {
  if (!codemirrorRef.value) return
  const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches

  const updateListener = EditorView.updateListener.of((update) => {
    if (update.docChanged) {
      code.value = update.state.doc.toString()
    }
  })

  const state = EditorState.create({
    doc: '',
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
      syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
      editorTheme,
      langCompartment.of([]),
      themeCompartment.of(isDark ? oneDark : []),
      keymap.of([
        ...closeBracketsKeymap,
        ...defaultKeymap,
        ...searchKeymap,
        ...historyKeymap,
        ...foldKeymap,
        indentWithTab,
      ]),
      updateListener,
      EditorView.editable.of(true),
    ],
  })

  editorView.value = new EditorView({ state, parent: codemirrorRef.value })
  applyLanguage(selectedLanguage.value)
}

function destroyEditor() {
  if (editorView.value) {
    editorView.value.destroy()
    editorView.value = null
  }
  code.value = ''
}

function handleCancel() {
  emit('update:visible', false)
}

function handleConfirm() {
  const markdown = formatCodeBlock(code.value, selectedLanguage.value)
  emit('confirm', markdown)
  emit('update:visible', false)
  code.value = ''
}

watch(
  () => props.visible,
  (val) => {
    if (val) {
      nextTick(() => {
        destroyEditor()
        createEditor()
      })
    } else {
      destroyEditor()
    }
  }
)

watch(selectedLanguage, (lang) => applyLanguage(lang))

onMounted(() => {
  if (props.visible) {
    nextTick(() => createEditor())
  }
})

onUnmounted(() => {
  destroyEditor()
})
</script>

<style scoped>
.code-block-editor__toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.code-block-editor__lang-label {
  font-size: 13px;
  color: var(--text-color);
}

.code-block-editor__lang-select {
  padding: 4px 8px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background: var(--card-bg);
  color: var(--text-color);
  font-size: 13px;
  outline: none;
}

.code-block-editor__codemirror {
  border: 1px solid var(--border-color);
  border-radius: 6px;
  overflow: hidden;
}

.code-block-editor__codemirror :deep(.cm-editor) {
  height: 360px;
}

.code-block-editor__btn {
  padding: 8px 24px;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.code-block-editor__btn--cancel {
  background: var(--card-bg);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.code-block-editor__btn--cancel:hover {
  background: var(--hover-color);
}

.code-block-editor__btn--confirm {
  background: var(--primary-color);
  color: #fff;
}

.code-block-editor__btn--confirm:hover:not(:disabled) {
  opacity: 0.9;
}

.code-block-editor__btn--confirm:disabled {
  background: var(--border-color);
  cursor: not-allowed;
}
</style>
