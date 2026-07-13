import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import CodeBlockEditor from '@/components/chat/CodeBlockEditor.vue'

// 保留 EditorView 的真实行为，只 mock 静态方法和 DOM 相关部分
const dispatchMock = vi.fn()
const destroyMock = vi.fn()
const EditorViewMock = vi.hoisted(() => {
  return class EditorView {
    static lineWrapping = {}
    static theme = () => ({})
    static editable = { of: () => ({}) }
    static updateListener = { of: () => ({}) }
    state = { doc: { toString: () => '' } }
    constructor() {}
    dispatch = dispatchMock
    destroy = destroyMock
  }
})

vi.mock('@codemirror/view', () => ({
  EditorView: EditorViewMock,
  keymap: { of: () => ({}) },
  lineNumbers: () => ({}),
  highlightActiveLine: () => ({}),
  highlightActiveLineGutter: () => ({}),
  drawSelection: () => ({}),
  rectangularSelection: () => ({}),
  crosshairCursor: () => ({}),
  highlightSpecialChars: () => ({}),
}))
vi.mock('@codemirror/state', () => ({
  EditorState: { create: () => ({}) },
  Compartment: class {
    of() { return {} }
    reconfigure() { return {} }
  },
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
  closeBrackets: () => ({}),
  closeBracketsKeymap: [],
}))
vi.mock('@codemirror/search', () => ({
  searchKeymap: [],
  highlightSelectionMatches: () => ({}),
}))
vi.mock('@lezer/highlight', () => ({ tags: {} }))
vi.mock('@codemirror/lang-markdown', () => ({ markdown: () => ({}), markdownLanguage: {} }))
vi.mock('@codemirror/language-data', () => ({ languages: [{ name: 'javascript' }] }))
vi.mock('@codemirror/theme-one-dark', () => ({ oneDark: {} }))

describe('CodeBlockEditor repro', () => {
  beforeEach(() => {
    dispatchMock.mockClear()
    destroyMock.mockClear()
  })

  it('visible 快速切换 true → false → true 时不抛错', async () => {
    const wrapper = mount(CodeBlockEditor, {
      props: { visible: false },
      global: {
        stubs: { QDialog: { props: ['visible'], template: '<div v-if="visible"><slot/><slot name="footer"/></div>' } },
      },
    })

    // true → 创建 editor
    await wrapper.setProps({ visible: true })
    await flushPromises()
    expect(destroyMock).not.toHaveBeenCalled()

    // false → 销毁 editor
    await wrapper.setProps({ visible: false })
    await flushPromises()
    expect(destroyMock).toHaveBeenCalledTimes(1)

    // true → 再次创建 editor
    await wrapper.setProps({ visible: true })
    await flushPromises()

    // 不应抛出未捕获异常
    expect(destroyMock).toHaveBeenCalledTimes(1) // 只销毁过一次（上次关闭时）
  })

  it('visible 从 true 启动时 onMounted 创建 editor', async () => {
    const wrapper = mount(CodeBlockEditor, {
      props: { visible: true },
      global: {
        stubs: { QDialog: { props: ['visible'], template: '<div v-if="visible"><slot/><slot name="footer"/></div>' } },
      },
    })
    await flushPromises()
    // onMounted 里 nextTick(createEditor) 应该执行
    expect(destroyMock).not.toHaveBeenCalled()
  })
})
