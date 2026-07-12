import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import CodeBlockEditor from '@/components/chat/CodeBlockEditor.vue'

// mock CodeMirror，避免 jsdom 下初始化失败
vi.mock('@codemirror/view', () => ({
  EditorView: class {
    static lineWrapping = {}
    static theme = () => ({})
    static editable = { of: () => ({}) }
    static updateListener = { of: () => ({}) }
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
  closeBrackets: () => ({}),
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
