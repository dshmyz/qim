import { mount, flushPromises } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AISidebarPanel from '@/components/ai/AISidebarPanel.vue'

// 捕获 stream options：不发真实请求，由测试手动驱动各回调
let streamOptions: any = null
const mockStream = vi.fn((options: any) => {
  streamOptions = options
  return Promise.resolve()
})

vi.mock('@/composables/useAIStream', () => ({
  useAIStream: () => ({ stream: mockStream, abort: vi.fn() }),
}))

vi.mock('@/composables/useServerUrl', () => ({
  getStoredServerUrl: () => 'http://localhost:8080',
}))

vi.mock('@/api/ai', () => ({
  aiPromptAPI: { getSuggested: vi.fn(() => Promise.resolve([])) },
  aiPendingAPI: { confirm: vi.fn(), cancel: vi.fn() },
}))

async function mountAndSend(text: string) {
  const wrapper = mount(AISidebarPanel, { props: { visible: true }, attachTo: document.body })
  await flushPromises()
  await wrapper.find('textarea').setValue(text)
  await wrapper.find('.send-btn').trigger('click')
  await flushPromises()
  expect(mockStream).toHaveBeenCalled()
  return wrapper
}

// tests/setup.ts 把 localStorage 替换为 vi.fn 桩（不落数据），持久化断言看 setItem 调用
function lastPersistedChat(): any[] {
  const calls = vi.mocked(localStorage.setItem).mock.calls
  const last = calls.filter(([key]) => key === 'qim:ai_sidebar_chat_v1').pop()
  return last ? JSON.parse(last[1]) : []
}

describe('AISidebarPanel 流式思考/回撤', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    streamOptions = null
  })

  it('回撤草稿按后缀切除：正文前部有相同子串时切尾部真草稿，不拦腰删前文', async () => {
    const wrapper = await mountAndSend('问题')

    // 正文先流出一句与后文草稿相同的话，再流出工具回合草稿（尾部）
    streamOptions.onChunk('先查一下待办。')
    streamOptions.onChunk('让我想想。')
    streamOptions.onReasoning('让我想想。')
    await flushPromises()

    streamOptions.onComplete()
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('先查一下待办。')
    // 草稿被精确切走：答案区不再出现，只保留思考过程区一份
    const draftOccurrences = text.split('让我想想').length - 1
    expect(draftOccurrences).toBe(1)

    wrapper.unmount()
  })

  it('找不到原文不强切：正文原样保留', async () => {
    const wrapper = await mountAndSend('问题')
    streamOptions.onChunk('答案正文。')
    await flushPromises()
    expect(wrapper.text()).toContain('答案正文。')

    streamOptions.onReasoning('从未流出的草稿')
    streamOptions.onComplete()
    await flushPromises()
    expect(wrapper.text()).toContain('答案正文。')

    wrapper.unmount()
  })

  it('回撤帧封口当前增量轮：草稿自成一轮，后续 reasoning delta 开新轮不混写', async () => {
    const wrapper = await mountAndSend('问题')

    streamOptions.onReasoningDelta('增量思考A')
    // 回撤的草稿此前已按正文流出（真实时序），回撤时切走并自成一轮
    streamOptions.onChunk('回撤草稿B')
    streamOptions.onReasoning('回撤草稿B')
    streamOptions.onReasoningDelta('增量思考C')
    streamOptions.onComplete()
    await flushPromises()

    const roundTexts = wrapper.findAll('.reasoning-round').map(r => r.text())
    expect(roundTexts).toContain('增量思考A')
    expect(roundTexts).toContain('回撤草稿B')
    // C 独立成轮，不拼进 B
    expect(roundTexts).toContain('增量思考C')

    wrapper.unmount()
  })

  it('纯思考回合（无正文/工具/pending）也落库：思考过程不随流结束消失', async () => {
    const wrapper = await mountAndSend('问题')

    streamOptions.onReasoningDelta('只有思考没有答案')
    streamOptions.onComplete()
    await flushPromises()

    expect(wrapper.text()).toContain('只有思考没有答案')

    // 持久化里也应有这条带 reasoning 的 assistant 消息
    const stored = lastPersistedChat()
    const last = stored[stored.length - 1]
    expect(last.role).toBe('assistant')
    expect(last.reasoning).toContain('只有思考没有答案')

    wrapper.unmount()
  })
})
