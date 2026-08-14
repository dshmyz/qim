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
        <MarkdownRenderer :content="localContent" @click="handlePreviewClick" />
      </div>
      <!-- 大纲目录 -->
      <div v-if="showToc && tocItems.length > 0" class="toc-panel">
        <div class="toc-header">
          <span>大纲</span>
          <button class="toc-close" @click="showToc = false"><i class="fas fa-times"></i></button>
        </div>
        <div class="toc-list">
          <a
            v-for="item in tocItems"
            :key="item.line"
            class="toc-item"
            :class="'toc-level-' + item.level"
            @click="scrollToLine(item.line)"
          >{{ item.text }}</a>
        </div>
      </div>
    </div>
    <!-- 状态栏 -->
    <div class="editor-statusbar">
      <span class="status-item">{{ wordCount }} 字</span>
      <span class="status-sep">|</span>
      <span class="status-item">{{ charCount }} 字符</span>
      <span class="status-sep">|</span>
      <span class="status-item">行 {{ cursorLine }}, 列 {{ cursorCol }}</span>
      <div class="status-spacer"></div>
      <button
        class="status-btn"
        :class="{ active: showToc }"
        @click="showToc = !showToc"
        title="大纲目录"
      ><i class="fas fa-list-ul"></i> 大纲</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed, onMounted, onUnmounted, shallowRef, nextTick } from 'vue'
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
import { fileApi } from '../../../api/file'
import { getStoredServerUrl } from '../../../composables/useServerUrl'

const props = defineProps<{
  title: string
  content: string
  mode: 'edit' | 'split' | 'preview'
  /** 笔记列表，用于 [[内链]] 自动补全和跳转 */
  noteList?: Array<{ id: number; title: string }>
}>()

const emit = defineEmits<{
  'update:title': [title: string]
  'update:content': [content: string]
  save: []
  /** 点击 [[内链]] 时通知父组件跳转到对应笔记 */
  'navigate-note': [noteId: number]
}>()

const localTitle = ref(props.title)
const localContent = ref(props.content)
const codemirrorRef = ref<HTMLElement | null>(null)
const editorContainerRef = ref<HTMLElement | null>(null)
const editorView = shallowRef<EditorView | null>(null)
const themeCompartment = new Compartment()
const keymapCompartment = new Compartment()

// 状态栏数据
const cursorLine = ref(1)
const cursorCol = ref(1)
const wordCount = computed(() => {
  const text = localContent.value.trim()
  if (!text) return 0
  // 中文按字符数，英文按空格分词
  const chinese = text.match(/[一-鿿]/g)
  const english = text.replace(/[一-鿿]/g, '').trim().split(/\s+/).filter(Boolean)
  return (chinese?.length || 0) + english.length
})
const charCount = computed(() => localContent.value.length)

// ---- 大纲目录 ----
interface TocItem { level: number; text: string; line: number }
const showToc = ref(false)
const tocItems = computed<TocItem[]>(() => {
  const items: TocItem[] = []
  const lines = localContent.value.split('\n')
  for (let i = 0; i < lines.length; i++) {
    const m = lines[i].match(/^(#{1,3})\s+(.+)$/)
    if (m) items.push({ level: m[1].length, text: m[2].replace(/\*\*|~~|`/g, ''), line: i + 1 })
  }
  return items
})

function scrollToLine(lineNumber: number) {
  if (!editorView.value) return
  const view = editorView.value
  const line = view.state.doc.line(Math.min(lineNumber, view.state.doc.lines))
  view.dispatch({ selection: { anchor: line.from }, effects: EditorView.scrollIntoView(line.from, { y: 'center' }) })
  view.focus()
}

// ---- 笔记内链 [[title]] ----
function handlePreviewClick(e: MouseEvent) {
  const target = (e.target as HTMLElement).closest('.note-link') as HTMLElement | null
  if (!target) return
  const title = target.dataset.noteTitle
  if (!title) return
  e.preventDefault()
  const note = props.noteList?.find(n => n.title === title)
  if (note) emit('navigate-note', note.id)
}

// ---- 图片粘贴 / 拖拽上传 ----
let imageUploading = ref(false)

async function uploadImage(file: File): Promise<string | null> {
  try {
    const res = await fileApi.uploadFile(file)
    if (res.data?.code === 0 && res.data.data) {
      const serverUrl = getStoredServerUrl() || ''
      return `${serverUrl}/api/v1/files/${res.data.data.id}/download`
    }
  } catch (err) {
    console.error('图片上传失败:', err)
  }
  return null
}

function insertAtCursor(text: string) {
  if (!editorView.value) return
  const view = editorView.value
  const { from } = view.state.selection.main
  view.dispatch({ changes: { from, to: from, insert: text } })
}

async function handleImageFile(file: File) {
  imageUploading.value = true
  insertAtCursor(`\n![上传中...]()\n`)
  const url = await uploadImage(file)
  imageUploading.value = false
  if (!url) return
  // 替换 "上传中..." 为实际 URL
  const view = editorView.value
  if (!view) return
  const doc = view.state.doc.toString()
  const idx = doc.indexOf('![上传中...]()')
  if (idx !== -1) {
    view.dispatch({
      changes: { from: idx, to: idx + '![上传中...]()'.length, insert: `![${file.name}](${url})` }
    })
  }
}

// ---- 富文本粘贴 → Markdown 转换 ----
function htmlToMarkdown(html: string): string {
  const parser = new DOMParser()
  const doc = parser.parseFromString(html, 'text/html')
  let md = ''

  function walk(node: Node): string {
    if (node.nodeType === Node.TEXT_NODE) return node.textContent || ''
    if (node.nodeType !== Node.ELEMENT_NODE) return ''
    const el = node as HTMLElement
    const tag = el.tagName.toLowerCase()
    const children = Array.from(el.childNodes).map(walk).join('')

    switch (tag) {
      case 'h1': return `\n# ${children.trim()}\n`
      case 'h2': return `\n## ${children.trim()}\n`
      case 'h3': return `\n### ${children.trim()}\n`
      case 'h4': return `\n#### ${children.trim()}\n`
      case 'h5': return `\n##### ${children.trim()}\n`
      case 'h6': return `\n###### ${children.trim()}\n`
      case 'strong': case 'b': return `**${children}**`
      case 'em': case 'i': return `*${children}*`
      case 's': case 'del': case 'strike': return `~~${children}~~`
      case 'code': return `\`${children}\``
      case 'pre': {
        const code = el.querySelector('code')
        const lang = code?.className?.match(/language-(\w+)/)?.[1] || ''
        return `\n\`\`\`${lang}\n${(code || el).textContent?.trim() || ''}\n\`\`\`\n`
      }
      case 'a': return `[${children}](${el.getAttribute('href') || ''})`
      case 'img': return `![${el.getAttribute('alt') || ''}](${el.getAttribute('src') || ''})`
      case 'br': return '\n'
      case 'p': case 'div': return `\n${children.trim()}\n`
      case 'blockquote': return children.split('\n').map(l => `> ${l}`).join('\n')
      case 'ul': case 'ol': {
        const items = Array.from(el.children).map((li, i) => {
          const prefix = tag === 'ol' ? `${i + 1}. ` : '- '
          return prefix + Array.from(li.childNodes).map(walk).join('').trim()
        })
        return '\n' + items.join('\n') + '\n'
      }
      case 'li': return children
      case 'hr': return '\n---\n'
      case 'table': {
        const rows = Array.from(el.querySelectorAll('tr'))
        if (rows.length === 0) return children
        const result: string[] = []
        rows.forEach((tr, ri) => {
          const cells = Array.from(tr.querySelectorAll('th, td')).map(c =>
            Array.from(c.childNodes).map(walk).join('').trim()
          )
          result.push('| ' + cells.join(' | ') + ' |')
          if (ri === 0) result.push('| ' + cells.map(() => '---').join(' | ') + ' |')
        })
        return '\n' + result.join('\n') + '\n'
      }
      default: return children
    }
  }

  md = Array.from(doc.body.childNodes).map(walk).join('')
  // 清理多余空行
  return md.replace(/\n{3,}/g, '\n\n').trim()
}

/**
 * 粘贴处理：返回 true 表示事件已处理（阻止 CM6 默认剪贴板插入，避免双份），
 * 未处理的（纯文本粘贴）返回 false 交给 CM6 默认行为。
 */
function handlePaste(e: ClipboardEvent): boolean {
  if (!editorView.value) return false
  const items = e.clipboardData?.items
  if (!items) return false

  // 优先处理图片粘贴
  for (const item of items) {
    if (item.type.startsWith('image/')) {
      e.preventDefault()
      const file = item.getAsFile()
      if (file) void handleImageFile(file)
      return true
    }
  }

  // HTML 粘贴 → 转 Markdown
  const html = e.clipboardData?.getData('text/html')
  if (html && html.includes('<')) {
    e.preventDefault()
    const md = htmlToMarkdown(html)
    const view = editorView.value
    const { from, to } = view.state.selection.main
    view.dispatch({ changes: { from, to, insert: md } })
    return true
  }
  return false
}

// ---- 拖拽上传 ----
function handleDrop(e: DragEvent): boolean {
  const files = e.dataTransfer?.files
  if (!files) return false
  let handled = false
  for (const file of files) {
    if (file.type.startsWith('image/')) {
      e.preventDefault()
      void handleImageFile(file)
      handled = true
    }
  }
  return handled
}

// ---- 插入表格 ----
function insertTable() {
  insertAtCursor('\n| 列1 | 列2 | 列3 |\n| --- | --- | --- |\n| 内容 | 内容 | 内容 |\n')
  editorView.value?.focus()
}

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
    fontSize: '13px',
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

interface EditorShortcutConf {
  accelerator: string
  enabled: boolean
}

function buildEditorKeymap(shortcuts?: Record<string, EditorShortcutConf>) {
  const editorShortcuts = shortcuts || {
    bold:        { accelerator: 'Mod-b', enabled: true },
    italic:      { accelerator: 'Mod-i', enabled: true },
    link:        { accelerator: 'Mod-k', enabled: true },
    save:        { accelerator: 'Mod-s', enabled: true },
    strikethrough: { accelerator: 'Mod-Shift-x', enabled: true },
    heading1:    { accelerator: 'Mod-Shift-1', enabled: true },
    heading2:    { accelerator: 'Mod-Shift-2', enabled: true },
    heading3:    { accelerator: 'Mod-Shift-3', enabled: true },
    code:        { accelerator: 'Mod-Shift-c', enabled: true },
    codeBlock:   { accelerator: 'Mod-Shift-b', enabled: true },
    blockquote:  { accelerator: 'Mod-Shift-q', enabled: true },
    unorderedList: { accelerator: 'Mod-Shift-u', enabled: true },
    orderedList:   { accelerator: 'Mod-Shift-o', enabled: true },
    taskList:      { accelerator: 'Mod-Shift-t', enabled: true },
  }
  const customKeymap: { key: string, run: () => boolean }[] = []
  if (editorShortcuts.bold?.enabled) {
    customKeymap.push({ key: editorShortcuts.bold.accelerator, run: () => { insertFormat('**', '**'); return true } })
  }
  if (editorShortcuts.italic?.enabled) {
    customKeymap.push({ key: editorShortcuts.italic.accelerator, run: () => { insertFormat('*', '*'); return true } })
  }
  if (editorShortcuts.link?.enabled) {
    customKeymap.push({ key: editorShortcuts.link.accelerator, run: () => { insertLink(); return true } })
  }
  if (editorShortcuts.save?.enabled) {
    customKeymap.push({ key: editorShortcuts.save.accelerator, run: () => { emit('save'); return true } })
  }
  if (editorShortcuts.strikethrough?.enabled) {
    customKeymap.push({ key: editorShortcuts.strikethrough.accelerator, run: () => { insertFormat('~~', '~~'); return true } })
  }
  if (editorShortcuts.heading1?.enabled) {
    customKeymap.push({ key: editorShortcuts.heading1.accelerator, run: () => { insertLinePrefix('# '); return true } })
  }
  if (editorShortcuts.heading2?.enabled) {
    customKeymap.push({ key: editorShortcuts.heading2.accelerator, run: () => { insertLinePrefix('## '); return true } })
  }
  if (editorShortcuts.heading3?.enabled) {
    customKeymap.push({ key: editorShortcuts.heading3.accelerator, run: () => { insertLinePrefix('### '); return true } })
  }
  if (editorShortcuts.code?.enabled) {
    customKeymap.push({ key: editorShortcuts.code.accelerator, run: () => { insertFormat('`', '`'); return true } })
  }
  if (editorShortcuts.codeBlock?.enabled) {
    customKeymap.push({ key: editorShortcuts.codeBlock.accelerator, run: () => { insertFormat('```\n', '\n```'); return true } })
  }
  if (editorShortcuts.blockquote?.enabled) {
    customKeymap.push({ key: editorShortcuts.blockquote.accelerator, run: () => { insertLinePrefix('> '); return true } })
  }
  if (editorShortcuts.unorderedList?.enabled) {
    customKeymap.push({ key: editorShortcuts.unorderedList.accelerator, run: () => { insertLinePrefix('- '); return true } })
  }
  if (editorShortcuts.orderedList?.enabled) {
    customKeymap.push({ key: editorShortcuts.orderedList.accelerator, run: () => { insertLinePrefix('1. '); return true } })
  }
  if (editorShortcuts.taskList?.enabled) {
    customKeymap.push({ key: editorShortcuts.taskList.accelerator, run: () => { insertLinePrefix('- [ ] '); return true } })
  }
  return customKeymap
}

function buildFullKeymap(shortcuts?: Record<string, EditorShortcutConf>) {
  return keymap.of([
    ...closeBracketsKeymap,
    ...defaultKeymap,
    ...searchKeymap,
    ...historyKeymap,
    ...foldKeymap,
    indentWithTab,
    ...buildEditorKeymap(shortcuts),
  ])
}

function createEditor() {
  if (!codemirrorRef.value) return

  const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches

  const updateListener = EditorView.updateListener.of((update) => {
    if (update.docChanged) {
      const newContent = update.state.doc.toString()
      localContent.value = newContent
      emit('update:content', newContent)
    }
    // 光标位置更新
    if (update.selectionSet || update.docChanged) {
      const pos = update.state.selection.main.head
      const line = update.state.doc.lineAt(pos)
      cursorLine.value = line.number
      cursorCol.value = pos - line.from + 1
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
      keymapCompartment.of(buildFullKeymap()),
      updateListener,
      EditorView.editable.of(true),
      // 粘贴/拖拽走 domEventHandlers：handler 返回 true 表示已处理，
      // CM6 不再执行默认剪贴板插入（避免与手动 dispatch 双份插入）；
      // 纯文本粘贴返回 false，走 CM6 默认行为
      EditorView.domEventHandlers({
        paste: handlePaste,
        drop: handleDrop,
        dragover: () => true,
      }),
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

/** 在当前行首插入前缀（标题、列表、引用等） */
function insertLinePrefix(prefix: string) {
  if (!editorView.value) return
  const view = editorView.value
  const { from } = view.state.selection.main
  const line = view.state.doc.lineAt(from)
  const lineText = line.text

  // 如果行首已有相同前缀，去掉（切换行为）
  if (lineText.startsWith(prefix)) {
    view.dispatch({
      changes: { from: line.from, to: line.from + prefix.length, insert: '' },
      selection: { anchor: from - prefix.length },
    })
  } else {
    // 去掉已有的标题/列表前缀再加新的
    const stripped = lineText.replace(/^#{1,3}\s|^>\s|^[-*]\s|^\d+\.\s|^[-*]\s\[[ x]\]\s/, '')
    const diff = lineText.length - stripped.length
    view.dispatch({
      changes: { from: line.from, to: line.from + diff, insert: prefix },
      selection: { anchor: from + prefix.length - diff },
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

defineExpose({ insertFormat, insertLink, insertLinePrefix, insertTable })

let themeQuery: MediaQueryList | null = null

// 快捷键更新监听器（提升为组件级，便于 onUnmounted 清理）
const handleShortcutsUpdated = (_event: unknown, shortcuts: { editor?: Record<string, EditorShortcutConf> }) => {
  if (editorView.value && shortcuts?.editor) {
    editorView.value.dispatch({
      effects: keymapCompartment.reconfigure(buildFullKeymap(shortcuts.editor))
    })
  }
}

onMounted(() => {
  nextTick(() => {
    createEditor()
    // 拉取快捷键配置并应用
    if (window.electron?.ipcRenderer?.invoke) {
      window.electron.ipcRenderer.invoke('get-shortcuts').then((shortcuts) => {
        if (editorView.value && shortcuts?.editor) {
          editorView.value.dispatch({
            effects: keymapCompartment.reconfigure(buildFullKeymap(shortcuts.editor))
          })
        }
      }).catch(() => {})
    }
  })

  themeQuery = window.matchMedia('(prefers-color-scheme: dark)')
  themeQuery.addEventListener('change', handleThemeChange)

  // 监听快捷键更新
  window.electron?.ipcRenderer?.on('shortcuts-updated', handleShortcutsUpdated)
})

onUnmounted(() => {
  if (editorView.value) {
    editorView.value.destroy()
    editorView.value = null
  }
  if (themeQuery) {
    themeQuery.removeEventListener('change', handleThemeChange)
  }
  if (window.electron?.ipcRenderer?.removeListener) {
    window.electron.ipcRenderer.removeListener('shortcuts-updated', handleShortcutsUpdated)
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

.preview-pane :deep(.markdown-content) {
  width: 100%;
  font-size: var(--font-size-xs);
  line-height: 1.7;
}

/* 状态栏 */
.editor-statusbar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 2px 12px;
  font-size: var(--font-size-xxxs);
  color: var(--text-secondary);
  background: var(--card-bg);
  border-top: 1px solid var(--border-color);
  border-radius: 0 0 var(--radius-lg) var(--radius-lg);
  flex-shrink: 0;
  user-select: none;
}

.status-sep {
  opacity: 0.3;
}

.status-item {
  white-space: nowrap;
}

.status-spacer {
  flex: 1;
}

.status-btn {
  border: none;
  background: none;
  color: var(--text-secondary);
  font-size: var(--font-size-xxxs);
  cursor: pointer;
  padding: 1px 6px;
  border-radius: 3px;
  display: flex;
  align-items: center;
  gap: 3px;
}

.status-btn:hover,
.status-btn.active {
  color: var(--primary-color);
  background: var(--primary-light);
}

/* 大纲目录 */
.toc-panel {
  width: 180px;
  min-width: 150px;
  border-left: 1px solid var(--border-color);
  background: var(--card-bg);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.toc-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 10px;
  font-size: var(--font-size-xxs);
  font-weight: 600;
  color: var(--text-color);
  border-bottom: 1px solid var(--border-color);
}

.toc-close {
  border: none;
  background: none;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: var(--font-size-xxs);
  padding: 0;
}

.toc-list {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}

.toc-item {
  display: block;
  padding: 3px 10px;
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  text-decoration: none;
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  transition: color 0.15s, background 0.15s;
}

.toc-item:hover {
  color: var(--primary-color);
  background: var(--primary-light);
}

.toc-level-1 { padding-left: 10px; font-weight: 600; color: var(--text-color); }
.toc-level-2 { padding-left: 22px; }
.toc-level-3 { padding-left: 34px; }
</style>
