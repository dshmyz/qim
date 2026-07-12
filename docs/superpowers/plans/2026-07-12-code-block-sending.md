# 代码块发送功能 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 让用户在聊天中通过工具栏"代码块"按钮打开 CodeMirror 代码编辑器，编辑完成后以 `type: 'markdown'` 消息发送，并在接收端以语法高亮渲染。

**架构：** 新增 `CodeBlockEditor.vue` 子组件（复用已有 `QDialog` 弹窗 + CodeMirror 编辑器，参考 `NoteEditor.vue`），通过 `ChatToolbar → MessageInput → ChatInputArea → ChatWindow` 事件链路触发；发送时复用已有的 `type: 'markdown'` 渲染分支（`MarkdownMessage.vue`）；在渲染端注册已安装的 `highlight.js` 启用语法高亮；后端扩展敏感词检查覆盖 markdown 类型。

**技术栈：** Vue 3 + TypeScript、CodeMirror 6（已安装）、highlight.js（已安装）、marked + DOMPurify（已安装）、vitest（前端测试）、Go testify（后端测试）

---

## 文件结构

### 新建文件
| 文件 | 职责 |
|------|------|
| `qim-client/src/utils/codeBlock.ts` | 纯函数 `formatCodeBlock(code, language)`：将代码与语言标记组装为 markdown 围栏代码块字符串 |
| `qim-client/src/composables/useCodeHighlight.ts` | 组合式函数：监听容器内容变化，对 `pre code` 元素调用 `hljs.highlightElement`，DRY 供所有 markdown 渲染组件复用 |
| `qim-client/src/components/chat/CodeBlockEditor.vue` | 代码块编辑器弹窗子组件：QDialog + CodeMirror + 语言选择器，确认时 emit 格式化后的 markdown |
| `qim-client/tests/unit/utils/codeBlock.test.ts` | `formatCodeBlock` 单元测试 |
| `qim-client/tests/unit/components/CodeBlockEditor.test.ts` | `CodeBlockEditor` 组件测试（CodeMirror mock） |
| `qim-client/tests/unit/composables/useCodeHighlight.test.ts` | 高亮组合式函数测试 |
| `qim-server/service/message_service_sensitive_test.go` | 后端 markdown 敏感词检查测试 |

### 修改文件
| 文件 | 改动 |
|------|------|
| `qim-client/src/components/chat/ChatToolbar.vue` | 新增"代码块"按钮，emit `open-code-block` |
| `qim-client/src/components/chat/MessageInput.vue` | 透传 `open-code-block` 事件 |
| `qim-client/src/components/chat/ChatInputArea.vue` | 透传 `open-code-block` 事件 |
| `qim-client/src/components/chat/ChatWindow.vue` | 处理 `open-code-block` 显示弹窗；确认时以 `type:'markdown'` 发送 |
| `qim-client/src/components/message/MarkdownMessage.vue` | 接入 `useCodeHighlight` 启用语法高亮 |
| `qim-client/src/components/message/StreamingMessage.vue` | 接入 `useCodeHighlight`（bot 流式回复可能含代码） |
| `qim-client/src/components/shared/MarkdownRenderer.vue` | 接入 `useCodeHighlight`（笔记/AI 摘要可能含代码） |
| `qim-server/service/message_service.go` | 第 112 行敏感词检查条件扩展为覆盖 `markdown` 类型 |

---

## 任务 1：`formatCodeBlock` 工具函数（TDD）

**文件：**
- 创建：`qim-client/src/utils/codeBlock.ts`
- 测试：`qim-client/tests/unit/utils/codeBlock.test.ts`

- [ ] **步骤 1：编写失败的测试**

创建 `qim-client/tests/unit/utils/codeBlock.test.ts`：

```typescript
import { describe, expect, it } from 'vitest'
import { formatCodeBlock } from '@/utils/codeBlock'

describe('formatCodeBlock', () => {
  it('wraps code with language fence', () => {
    expect(formatCodeBlock('console.log(1)', 'javascript'))
      .toBe('```javascript\nconsole.log(1)\n```')
  })

  it('wraps code with python fence', () => {
    expect(formatCodeBlock('print(1)', 'python'))
      .toBe('```python\nprint(1)\n```')
  })

  it('produces fence without language when language is empty', () => {
    expect(formatCodeBlock('hello', ''))
      .toBe('```\nhello\n```')
  })

  it('trims trailing newlines from code to keep fence clean', () => {
    expect(formatCodeBlock('hello\n\n', 'js'))
      .toBe('```js\nhello\n```')
  })

  it('trims surrounding whitespace from language identifier', () => {
    expect(formatCodeBlock('x = 1', '  python  '))
      .toBe('```python\nx = 1\n```')
  })

  it('handles multi-line code', () => {
    const code = 'function add(a, b) {\n  return a + b\n}'
    expect(formatCodeBlock(code, 'typescript'))
      .toBe('```typescript\n' + code + '\n```')
  })
})
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd qim-client && npx vitest run tests/unit/utils/codeBlock.test.ts`
预期：FAIL，报错 `Failed to resolve import "@/utils/codeBlock"` 或 `formatCodeBlock is not a function`

- [ ] **步骤 3：编写最少实现代码**

创建 `qim-client/src/utils/codeBlock.ts`：

```typescript
/**
 * 将代码与语言标记组装为 markdown 围栏代码块字符串。
 * @param code 代码内容
 * @param language 语言标识（如 'javascript'、'python'），空字符串表示无语言标记
 * @returns 形如 ```lang\ncode\n``` 的字符串
 */
export function formatCodeBlock(code: string, language: string = ''): string {
  const lang = language.trim()
  const trimmedCode = code.replace(/\n+$/, '')
  return '```' + lang + '\n' + trimmedCode + '\n```'
}
```

- [ ] **步骤 4：运行测试验证通过**

运行：`cd qim-client && npx vitest run tests/unit/utils/codeBlock.test.ts`
预期：PASS，全部 6 个用例通过

- [ ] **步骤 5：Commit**

```bash
cd qim-client && git add src/utils/codeBlock.ts tests/unit/utils/codeBlock.test.ts
git commit -m "feat: 添加 formatCodeBlock 工具函数用于组装 markdown 代码块"
```

---

## 任务 2：`useCodeHighlight` 组合式函数（TDD）

**文件：**
- 创建：`qim-client/src/composables/useCodeHighlight.ts`
- 测试：`qim-client/tests/unit/composables/useCodeHighlight.test.ts`

- [ ] **步骤 1：编写失败的测试**

创建 `qim-client/tests/unit/composables/useCodeHighlight.test.ts`：

```typescript
import { describe, expect, it, vi } from 'vitest'
import { ref, nextTick } from 'vue'
import { useCodeHighlight } from '@/composables/useCodeHighlight'

vi.mock('highlight.js', () => ({
  default: {
    highlightElement: vi.fn((el: HTMLElement) => {
      el.classList.add('hljs')
      el.setAttribute('data-highlighted', 'true')
    }),
  },
}))

import hljs from 'highlight.js'

describe('useCodeHighlight', () => {
  it('highlights pre>code elements when trigger changes', async () => {
    const container = document.createElement('div')
    const codeEl = document.createElement('code')
    codeEl.textContent = 'const x = 1'
    const pre = document.createElement('pre')
    pre.appendChild(codeEl)
    container.appendChild(pre)

    const containerRef = ref<HTMLElement | null>(container)
    const trigger = ref('initial')

    useCodeHighlight(containerRef, trigger)

    trigger.value = 'updated'
    await nextTick()
    await nextTick()

    expect(hljs.highlightElement).toHaveBeenCalled()
    expect(codeEl.getAttribute('data-highlighted')).toBe('true')
  })

  it('does not re-highlight already highlighted elements', async () => {
    const container = document.createElement('div')
    const codeEl = document.createElement('code')
    codeEl.textContent = 'const x = 1'
    codeEl.setAttribute('data-highlighted', 'true')
    const pre = document.createElement('pre')
    pre.appendChild(codeEl)
    container.appendChild(pre)

    const containerRef = ref<HTMLElement | null>(container)
    const trigger = ref('initial')

    useCodeHighlight(containerRef, trigger)

    trigger.value = 'updated'
    await nextTick()
    await nextTick()

    expect(hljs.highlightElement).not.toHaveBeenCalled()
  })

  it('does nothing when container is null', async () => {
    const containerRef = ref<HTMLElement | null>(null)
    const trigger = ref('initial')

    useCodeHighlight(containerRef, trigger)

    trigger.value = 'updated'
    await nextTick()
    await nextTick()

    expect(hljs.highlightElement).not.toHaveBeenCalled()
  })
})
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd qim-client && npx vitest run tests/unit/composables/useCodeHighlight.test.ts`
预期：FAIL，报错无法解析 `@/composables/useCodeHighlight`

- [ ] **步骤 3：编写最少实现代码**

创建 `qim-client/src/composables/useCodeHighlight.ts`：

```typescript
import { watch, nextTick, type Ref } from 'vue'
import hljs from 'highlight.js'
// 引入浅色主题（用户偏好浅色，不使用暗色主题）
import 'highlight.js/styles/github.css'

/**
 * 监听容器内 markdown 渲染结果，对 pre>code 元素应用 highlight.js 语法高亮。
 * 已高亮的元素不会重复处理。
 * @param containerRef 容器元素引用
 * @param trigger 触发重新高亮的响应式值（通常是渲染后的 HTML 字符串）
 */
export function useCodeHighlight(
  containerRef: Ref<HTMLElement | null>,
  trigger: Ref<string>
): void {
  watch(
    trigger,
    () => {
      nextTick(() => {
        if (!containerRef.value) return
        const blocks = containerRef.value.querySelectorAll<HTMLElement>('pre code')
        blocks.forEach((el) => {
          if (el.dataset.highlighted) return
          try {
            hljs.highlightElement(el)
            el.dataset.highlighted = 'true'
          } catch {
            // 忽略无法识别的语言，保持原始文本
          }
        })
      })
    },
    { immediate: true }
  )
}
```

- [ ] **步骤 4：运行测试验证通过**

运行：`cd qim-client && npx vitest run tests/unit/composables/useCodeHighlight.test.ts`
预期：PASS，全部 3 个用例通过

- [ ] **步骤 5：Commit**

```bash
cd qim-client && git add src/composables/useCodeHighlight.ts tests/unit/composables/useCodeHighlight.test.ts
git commit -m "feat: 添加 useCodeHighlight 组合式函数用于 markdown 代码语法高亮"
```

---

## 任务 3：`CodeBlockEditor.vue` 子组件

**文件：**
- 创建：`qim-client/src/components/chat/CodeBlockEditor.vue`
- 测试：`qim-client/tests/unit/components/CodeBlockEditor.test.ts`

- [ ] **步骤 1：编写失败的测试**

创建 `qim-client/tests/unit/components/CodeBlockEditor.test.ts`：

```typescript
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import CodeBlockEditor from '@/components/chat/CodeBlockEditor.vue'

// mock CodeMirror，避免 jsdom 下初始化失败
vi.mock('@codemirror/view', () => ({
  EditorView: class {
    state = { doc: { toString: () => '' } }
    constructor() {}
    dispatch() {}
    destroy() {}
  },
  keymap: { of: () => ({}) },
  lineNumbers: () => ({}),
  highlightActiveLine: () => ({}),
  highlightActiveLineGutter: () => ({}),
  drawSelection: () => ({}),
  rectangularSelection: () => ({}),
  crosshairCursor: () => ({}),
  highlightSpecialChars: () => ({}),
  EditorView: { lineWrapping: {}, theme: () => ({}), editable: { of: () => ({}) } },
}))
vi.mock('@codemirror/state', () => ({
  EditorState: { create: () => ({}) },
  Compartment: class { of() { return {} } reconfigure() { return {} } },
}))
vi.mock('@codemirror/commands', () => ({
  defaultKeymap: [],
  historyKeymap: [],
  history: () => ({}),
  indentWithTab: {},
}))
vi.mock('@codemirror/language', () => ({
  LanguageDescription: { matchLanguageName: () => null },
  defaultHighlightStyle: {},
  syntaxHighlighting: () => ({}),
  bracketMatching: () => ({}),
  indentOnInput: () => ({}),
  foldGutter: () => ({}),
  foldKeymap: [],
  closeBrackets: () => ({}),
  HighlightStyle: { define: () => ({}) },
}))
vi.mock('@codemirror/autocomplete', () => ({
  closeBracketsKeymap: [],
}))
vi.mock('@codemirror/search', () => ({
  searchKeymap: [],
  highlightSelectionMatches: () => ({}),
}))
vi.mock('@lezer/highlight', () => ({ tags: {} }))
vi.mock('@codemirror/lang-markdown', () => ({ markdown: () => ({}), markdownLanguage: {} }))
vi.mock('@codemirror/language-data', () => ({ languages: [] }))
vi.mock('@codemirror/theme-one-dark', () => ({ oneDark: {} }))

const mountEditor = (props: Record<string, unknown> = {}) =>
  mount(CodeBlockEditor, {
    props: { visible: true, ...props },
    global: {
      stubs: { QDialog: { template: '<div><slot/><slot name="footer"/></div>' } },
    },
  })

describe('CodeBlockEditor', () => {
  it('renders language selector and editor when visible', () => {
    const wrapper = mountEditor()
    expect(wrapper.find('.code-block-editor__lang-select').exists()).toBe(true)
    expect(wrapper.find('.code-block-editor__codemirror').exists()).toBe(true)
  })

  it('emits update:visible(false) when cancel is clicked', async () => {
    const wrapper = mountEditor()
    await wrapper.find('.code-block-editor__btn--cancel').trigger('click')
    expect(wrapper.emitted('update:visible')).toBeTruthy()
    expect(wrapper.emitted('update:visible')![0]).toEqual([false])
  })

  it('confirm button is disabled when code is empty', () => {
    const wrapper = mountEditor()
    const confirmBtn = wrapper.find('.code-block-editor__btn--confirm')
    expect(confirmBtn.attributes('disabled')).toBeDefined()
  })
})
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd qim-client && npx vitest run tests/unit/components/CodeBlockEditor.test.ts`
预期：FAIL，报错无法解析 `@/components/chat/CodeBlockEditor`

- [ ] **步骤 3：编写实现代码**

创建 `qim-client/src/components/chat/CodeBlockEditor.vue`：

```vue
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
```

- [ ] **步骤 4：运行测试验证通过**

运行：`cd qim-client && npx vitest run tests/unit/components/CodeBlockEditor.test.ts`
预期：PASS，全部 3 个用例通过

- [ ] **步骤 5：Commit**

```bash
cd qim-client && git add src/components/chat/CodeBlockEditor.vue tests/unit/components/CodeBlockEditor.test.ts
git commit -m "feat: 添加 CodeBlockEditor 代码块编辑器弹窗组件"
```

---

## 任务 4：ChatToolbar 添加代码块按钮

**文件：**
- 修改：`qim-client/src/components/chat/ChatToolbar.vue`
- 测试：`qim-client/tests/unit/components/ChatToolbar.test.ts`（新建）

- [ ] **步骤 1：编写失败的测试**

创建 `qim-client/tests/unit/components/ChatToolbar.test.ts`：

```typescript
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ChatToolbar from '@/components/chat/ChatToolbar.vue'

const mountToolbar = (props: Record<string, unknown> = {}) =>
  mount(ChatToolbar, {
    props: { isElectron: false, showAiActions: false, ...props },
    global: {
      stubs: { ChatToolbarButton: { template: '<button @click="$emit(\'click\')"><slot/></button>' } },
    },
  })

describe('ChatToolbar code block button', () => {
  it('renders a code block button', () => {
    const wrapper = mountToolbar()
    const btn = wrapper.find('[title="代码块"]')
    expect(btn.exists()).toBe(true)
  })

  it('emits open-code-block when code block button is clicked', async () => {
    const wrapper = mountToolbar()
    const btn = wrapper.find('[title="代码块"]')
    await btn.trigger('click')
    expect(wrapper.emitted('open-code-block')).toBeTruthy()
    expect(wrapper.emitted('open-code-block')).toHaveLength(1)
  })
})
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd qim-client && npx vitest run tests/unit/components/ChatToolbar.test.ts`
预期：FAIL，找不到 `[title="代码块"]` 元素

- [ ] **步骤 3：修改 ChatToolbar.vue**

在 `ChatToolbar.vue` 模板中，"发送图片"按钮（第 39-43 行）之后插入代码块按钮：

```html
    <ChatToolbarButton
      icon="fas fa-image"
      title="发送图片"
      @click="$emit('select-image')"
    />
    <ChatToolbarButton
      icon="fas fa-code"
      title="代码块"
      @click="$emit('open-code-block')"
    />
```

在 `defineEmits` 类型（第 131-143 行）中新增事件：

```typescript
const emit = defineEmits<{
  'start-voice-call': []
  'start-video-call': []
  'start-screen-share': []
  'toggle-emoji-panel': []
  'select-file': []
  'select-image': []
  'open-code-block': []
  'take-screenshot': []
  'take-screenshot-hidden': []
  'open-message-manager': []
  'open-mini-app-list': []
  'toggle-ai-actions': []
}>()
```

- [ ] **步骤 4：运行测试验证通过**

运行：`cd qim-client && npx vitest run tests/unit/components/ChatToolbar.test.ts`
预期：PASS，全部 2 个用例通过

- [ ] **步骤 5：运行已有 MessageInput 测试确保无回归**

运行：`cd qim-client && npx vitest run tests/unit/components/MessageInput.test.ts`
预期：PASS（MessageInput 测试 stub 了 ChatToolbar，不受影响）

- [ ] **步骤 6：Commit**

```bash
cd qim-client && git add src/components/chat/ChatToolbar.vue tests/unit/components/ChatToolbar.test.ts
git commit -m "feat: ChatToolbar 添加代码块按钮"
```

---

## 任务 5：事件链路接线（MessageInput → ChatInputArea → ChatWindow）

**文件：**
- 修改：`qim-client/src/components/chat/MessageInput.vue`
- 修改：`qim-client/src/components/chat/ChatInputArea.vue`
- 修改：`qim-client/src/components/chat/ChatWindow.vue`
- 测试：`qim-client/tests/unit/components/MessageInput.test.ts`（追加用例）

- [ ] **步骤 1：编写失败的测试**

在 `qim-client/tests/unit/components/MessageInput.test.ts` 末尾追加：

```typescript
describe('MessageInput code block forwarding', () => {
  it('forwards open-code-block event from ChatToolbar', async () => {
    const wrapper = mountMessageInput({ showAtMembersPanel: false })
    // ChatToolbar 被 stub，通过 vm 直接触发其 emit
    const toolbar = wrapper.findComponent({ name: 'ChatToolbar' })
    toolbar.vm.$emit('open-code-block')
    await wrapper.vm.$nextTick()
    expect(wrapper.emitted('open-code-block')).toBeTruthy()
  })
})
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd qim-client && npx vitest run tests/unit/components/MessageInput.test.ts`
预期：FAIL，`wrapper.emitted('open-code-block')` 为 undefined（事件未透传）

- [ ] **步骤 3：修改 MessageInput.vue 透传事件**

在 `MessageInput.vue` 模板中，给 `<ChatToolbar>`（第 18-32 行）新增事件绑定：

```html
    <ChatToolbar
      :is-electron="isElectron"
      :show-ai-actions="localShowAIActions"
      @start-voice-call="$emit('start-voice-call')"
      @start-video-call="$emit('start-video-call')"
      @start-screen-share="$emit('start-screen-share')"
      @toggle-emoji-panel="$emit('toggle-emoji-panel')"
      @select-file="handleSelectFile"
      @select-image="$emit('select-image')"
      @open-code-block="$emit('open-code-block')"
      @take-screenshot="$emit('take-screenshot')"
      @take-screenshot-hidden="$emit('take-screenshot-hidden')"
      @open-message-manager="$emit('open-message-manager')"
      @open-mini-app-list="$emit('open-mini-app-list')"
      @toggle-ai-actions="toggleAI"
    />
```

在 `MessageInput.vue` 的 `defineEmits`（第 157-187 行）中新增：

```typescript
  (e: 'open-code-block'): void
```

（插入在 `(e: 'select-image'): void` 之后）

- [ ] **步骤 4：运行测试验证通过**

运行：`cd qim-client && npx vitest run tests/unit/components/MessageInput.test.ts`
预期：PASS，包括新增的转发用例

- [ ] **步骤 5：修改 ChatInputArea.vue 透传事件**

在 `ChatInputArea.vue` 模板的 `<MessageInput>`（第 3-46 行）中新增绑定：

```html
    @select-image="emit('select-image')"
    @open-code-block="emit('open-code-block')"
    @take-screenshot="emit('take-screenshot')"
```

在 `defineEmits`（第 80-112 行）中新增：

```typescript
  'open-code-block': []
```

（插入在 `'select-image': []` 之后）

- [ ] **步骤 6：修改 ChatWindow.vue 接入弹窗并发送 markdown**

在 `ChatWindow.vue` 模板中，`<ChatInputArea>`（第 79 行起）新增事件绑定：

```html
      @open-code-block="showCodeBlockEditor = true"
```

（插入在 `@select-image="selectImage"` 之后的位置，参考第 99 行附近的事件绑定区）

在 `ChatWindow.vue` 模板中，`</ChatInputArea>` 之后插入弹窗组件：

```html
    <CodeBlockEditor
      v-model:visible="showCodeBlockEditor"
      @confirm="handleSendCodeBlock"
    />
```

在 `<script setup>` 的 import 区（第 214-220 行附近）新增：

```typescript
import CodeBlockEditor from './CodeBlockEditor.vue'
```

在 `<script setup>` 的状态定义区（第 503 行 `inputMessage` 附近）新增：

```typescript
const showCodeBlockEditor = ref(false)
```

在 `handleSend` 函数（第 953 行）之后新增处理函数：

```typescript
const handleSendCodeBlock = (markdown: string) => {
  if (!markdown.trim()) return
  const messageData = {
    content: markdown,
    type: 'markdown' as const,
    quotedMessage: quotedMessage.value
  }
  emit('send', messageData)
  quotedMessage.value = null
}
```

- [ ] **步骤 7：运行全部前端测试确保无回归**

运行：`cd qim-client && npx vitest run`
预期：全部 PASS

- [ ] **步骤 8：Commit**

```bash
cd qim-client && git add src/components/chat/MessageInput.vue src/components/chat/ChatInputArea.vue src/components/chat/ChatWindow.vue tests/unit/components/MessageInput.test.ts
git commit -m "feat: 接线代码块按钮事件链路，以 markdown 类型发送"
```

---

## 任务 6：渲染端启用 highlight.js 语法高亮

**文件：**
- 修改：`qim-client/src/components/message/MarkdownMessage.vue`
- 修改：`qim-client/src/components/message/StreamingMessage.vue`
- 修改：`qim-client/src/components/shared/MarkdownRenderer.vue`

- [ ] **步骤 1：修改 MarkdownMessage.vue 接入 useCodeHighlight**

在模板第 2 行添加 ref：

```html
  <div ref="containerRef" class="markdown-message" :class="{ self: isSelf }" v-html="renderedContent" @click="handleLinkClick"></div>
```

在 `<script setup>` 中新增导入与调用（在现有 import 之后）：

```typescript
import { ref } from 'vue'
import { useCodeHighlight } from '../../composables/useCodeHighlight'

const containerRef = ref<HTMLElement | null>(null)
useCodeHighlight(containerRef, renderedContent)
```

注意：`renderedContent` 已是 computed，`useCodeHighlight` 的第二个参数接收该 computed 的值（解包后为 string）。需确保 `renderedContent` 在 `useCodeHighlight` 调用前已定义——当前 `renderedContent` 定义在第 36 行，`useCodeHighlight` 调用应放在其后。

在 `<style>` 中新增覆盖，避免 hljs 主题背景与现有样式冲突：

```css
.markdown-message .hljs {
  background: transparent !important;
  padding: 0 !important;
}
```

- [ ] **步骤 2：修改 StreamingMessage.vue 接入 useCodeHighlight**

读取 `StreamingMessage.vue`，找到其 `renderedContent` computed 与 `v-html` 容器，按相同模式添加 `ref="containerRef"` 与 `useCodeHighlight(containerRef, renderedContent)` 及 `.hljs` 背景覆盖样式。

- [ ] **步骤 3：修改 MarkdownRenderer.vue 接入 useCodeHighlight**

读取 `MarkdownRenderer.vue`，按相同模式接入。

- [ ] **步骤 4：运行前端测试确保无回归**

运行：`cd qim-client && npx vitest run`
预期：全部 PASS（如有 MessageItem.test.ts 等渲染测试因 hljs mock 失败，在测试中 stub `useCodeHighlight` 或在测试 setup 中 mock `highlight.js`）

- [ ] **步骤 5：Commit**

```bash
cd qim-client && git add src/components/message/MarkdownMessage.vue src/components/message/StreamingMessage.vue src/components/shared/MarkdownRenderer.vue
git commit -m "feat: markdown 渲染组件接入 highlight.js 语法高亮"
```

---

## 任务 7：后端敏感词检查扩展覆盖 markdown 类型

**文件：**
- 修改：`qim-server/service/message_service.go`（第 112 行）
- 测试：`qim-server/service/message_service_sensitive_test.go`（新建）

- [ ] **步骤 1：编写失败的测试**

创建 `qim-server/service/message_service_sensitive_test.go`：

```go
package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 测试 SendMessage 对 markdown 类型消息也执行敏感词检查
func TestSendMessage_SensitiveCheck_MarkdownType(t *testing.T) {
	db := setupServiceTestDB(t)

	// 准备：创建用户、会话、成员
	user := createTestUser(db, "u-markdown", "markdown-user")
	conv := createTestSingleConversation(db, user.ID)

	// 注入敏感词
	svc := &MessageService{db: db, sensitiveWords: []string{"禁用词"}}

	// markdown 类型消息含敏感词，应被拦截
	_, err := svc.SendMessage(conv.ID, user.ID, "markdown", "```go\n// 禁用词\nfmt.Println(1)\n```", nil)
	assert.ErrorIs(t, err, ErrSensitiveWordBlocked)

	// markdown 类型消息不含敏感词，正常发送
	msg, err := svc.SendMessage(conv.ID, user.ID, "markdown", "```go\nfmt.Println(1)\n```", nil)
	require.NoError(t, err)
	assert.Equal(t, "markdown", msg.Type)
}
```

注：`createTestUser` / `createTestSingleConversation` 若在 `service_test.go` 中已存在则复用；若不存在，在本文件内补充辅助函数（参考 `service_test.go` 中已有测试的建表方式）。

运行前先确认 `MessageService` 结构体是否有 `sensitiveWords` 字段：`grep -n "sensitiveWords" qim-server/service/message_service.go`。若字段名不同，按实际字段名调整；若 `CheckSensitiveContent` 依赖全局缓存而非结构体字段，则在测试中通过设置全局缓存注入敏感词（参考已有敏感词测试模式）。

- [ ] **步骤 2：运行测试验证失败**

运行：`cd qim-server && go test ./service/ -run TestSendMessage_SensitiveCheck_MarkdownType -v`
预期：FAIL——markdown 类型未触发敏感词检查，含敏感词的消息未被拦截（`assert.ErrorIs` 失败）

- [ ] **步骤 3：修改 message_service.go 第 112 行**

将：

```go
	if msgType == "text" && content != "" {
```

改为：

```go
	if (msgType == "text" || msgType == "markdown") && content != "" {
```

- [ ] **步骤 4：运行测试验证通过**

运行：`cd qim-server && go test ./service/ -run TestSendMessage_SensitiveCheck_MarkdownType -v`
预期：PASS

- [ ] **步骤 5：运行全部后端 service 测试确保无回归**

运行：`cd qim-server && go test ./service/ -v`
预期：全部 PASS

- [ ] **步骤 6：Commit**

```bash
cd qim-server && git add service/message_service.go service/message_service_sensitive_test.go
git commit -m "feat: 敏感词检查覆盖 markdown 类型消息"
```

---

## 任务 8：集成验证

- [ ] **步骤 1：前端构建检查**

运行：`cd qim-client && npm run build`
预期：构建成功，无 TypeScript 错误

- [ ] **步骤 2：启动开发环境手动验证**

运行前端 dev server 与后端 server。

手动测试清单：
1. 打开任意会话，确认输入区工具栏出现"代码块"按钮（`fa-code` 图标）
2. 点击"代码块"按钮，确认弹出代码编辑器弹窗
3. 选择语言（如 javascript），输入代码，确认 CodeMirror 有行号、括号匹配
4. 切换语言，确认编辑器语言高亮跟随切换
5. 代码为空时"发送"按钮禁用；输入代码后启用
6. 点击"发送"，确认消息以代码块形式出现在聊天列表（type=markdown，走 MarkdownMessage 渲染）
7. 确认渲染出的代码块有语法高亮颜色（highlight.js 生效）
8. 点击"取消"或遮罩层，确认弹窗关闭且不发送
9. 对方接收该消息，确认同样以高亮代码块渲染
10. 发送含敏感词的代码块，确认被后端拦截并提示

- [ ] **步骤 3：全量测试回归**

运行：
```bash
cd qim-client && npx vitest run
cd qim-server && go test ./...
```
预期：全部 PASS

---

## 自检

**1. 规格覆盖度：**
- 工具栏代码块入口 → 任务 4 ✓
- CodeMirror 编辑器子组件 → 任务 3 ✓
- 以 type:markdown 发送 → 任务 5 步骤 6 ✓
- 渲染端代码块显示 → 已有 MarkdownMessage 分支（无需新增）✓
- 语法高亮 → 任务 6 ✓
- 后端存储 → 已支持透传（无需改动）✓
- 后端敏感词检查 → 任务 7 ✓
- XSS 防护 → sanitize.ts 白名单已含 pre/code（无需改动）✓

**2. 占位符扫描：** 无 TODO/待定，所有代码步骤均含完整代码。

**3. 类型一致性：**
- `formatCodeBlock(code: string, language: string): string` — 任务 1 定义，任务 3 调用 ✓
- `useCodeHighlight(containerRef, trigger)` — 任务 2 定义，任务 6 三处调用签名一致 ✓
- `CodeBlockEditor` props `visible`、emits `update:visible`/`confirm` — 任务 3 定义，任务 5 使用 `v-model:visible` + `@confirm` ✓
- `open-code-block` 事件名在 ChatToolbar/MessageInput/ChatInputArea/ChatWindow 中一致 ✓
- `handleSendCodeBlock(markdown: string)` 在 ChatWindow 中定义，emit `('send', messageData)` 与已有 `handleSend` 的 emit 签名一致 ✓
