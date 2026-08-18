import { shallowMount } from '@vue/test-utils'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import NotesApp from '@/components/apps/NotesApp.vue'
import AiFormatModal from '@/components/apps/notes/AiFormatModal.vue'
import QMessage from '@/utils/qmessage'

// vi.mock 工厂被提升到模块顶部执行，mock 函数实例必须用 vi.hoisted 提前声明
const mocks = vi.hoisted(() => ({
  fetchNotes: vi.fn(),
  updateNote: vi.fn(),
  formatNote: vi.fn(),
}))

vi.mock('@/composables/useNotes', () => ({
  useNotes: () => ({
    fetchNotes: mocks.fetchNotes,
    createNote: vi.fn(),
    updateNote: mocks.updateNote,
    deleteNote: vi.fn(),
    analyzeNote: vi.fn(),
    formatNote: mocks.formatNote,
    updateNoteTags: vi.fn(),
    updateNoteSummary: vi.fn(),
    exportNote: vi.fn(),
    setNoteAiAccessible: vi.fn(),
    error: { value: null },
  }),
}))

vi.mock('@/utils/qmessage', () => ({
  default: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), info: vi.fn() },
}))

vi.mock('@/utils/qmessagebox', () => ({
  default: { confirm: vi.fn().mockResolvedValue(true) },
}))

vi.mock('@/composables/useAutoSave', () => ({
  useAutoSave: () => ({
    status: { value: 'idle' },
    save: vi.fn(),
    schedule: vi.fn(),
    flush: vi.fn().mockResolvedValue('idle'),
  }),
}))

vi.mock('@/composables/useNoteDraft', () => ({
  useNoteDraft: () => ({
    save: vi.fn(),
    load: vi.fn().mockReturnValue(null),
    clear: vi.fn(),
  }),
}))

vi.mock('@/composables/useUI', () => ({
  openMenu: vi.fn(),
}))

const note = {
  id: 1,
  title: '测试笔记',
  content: '原始内容',
  tags: [],
  ai_accessible: true,
}

async function mountWithSelectedNote() {
  mocks.fetchNotes.mockResolvedValue([note])
  const wrapper = shallowMount(NotesApp)
  await new Promise((r) => setTimeout(r, 0)) // 等待 onMounted 的 fetchNotes 完成
  await wrapper.vm.$nextTick()
  // 选中笔记：NoteCard stub 触发 select
  wrapper.findComponent({ name: 'NoteCard' }).vm.$emit('select', 1)
  await wrapper.vm.$nextTick()
  return wrapper
}

async function runFormatConfirm(wrapper: ReturnType<typeof shallowMount>, formatResult: { content: string; truncated: boolean }) {
  mocks.formatNote.mockResolvedValue(formatResult)
  // NoteToolbar stub 触发 ai-format → handleAiFormat 拉取格式化结果并打开确认弹窗
  wrapper.findComponent({ name: 'NoteToolbar' }).vm.$emit('ai-format')
  await wrapper.vm.$nextTick()
  // AiFormatModal stub 触发 confirm → handleFormatConfirm 落库
  wrapper.findComponent(AiFormatModal).vm.$emit('confirm')
  await wrapper.vm.$nextTick()
  await new Promise((r) => setTimeout(r, 0))
}

describe('NotesApp AI 格式化保存守卫', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.updateNote.mockResolvedValue(true)
  })

  it('truncated=true 时禁止整篇覆盖保存，避免超长笔记尾部丢失', async () => {
    const wrapper = await mountWithSelectedNote()

    await runFormatConfirm(wrapper, { content: '只格式化到了前一万字', truncated: true })

    expect(mocks.updateNote).not.toHaveBeenCalled()
    expect(QMessage.warning).toHaveBeenCalledWith(expect.stringContaining('10000'))
  })

  it('truncated=false 时正常替换保存', async () => {
    const wrapper = await mountWithSelectedNote()

    await runFormatConfirm(wrapper, { content: '格式化完成的内容', truncated: false })

    expect(mocks.updateNote).toHaveBeenCalledTimes(1)
    expect(mocks.updateNote).toHaveBeenCalledWith(1, expect.objectContaining({ content: '格式化完成的内容' }))
    expect(QMessage.warning).not.toHaveBeenCalled()
    expect(QMessage.success).toHaveBeenCalled()
  })
})
