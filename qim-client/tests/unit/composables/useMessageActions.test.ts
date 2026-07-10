import { ref } from 'vue'
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { useMessageActions } from '@/composables/useMessageActions'
import type { Message } from '@/types'

// Mock request
vi.mock('@/composables/useRequest', () => ({
  request: vi.fn(),
}))

// Mock pinia store
vi.mock('pinia', () => ({
  storeToRefs: vi.fn(() => ({
    showReadUsersModal: { value: false },
  })),
  defineStore: vi.fn(),
}))

vi.mock('@/stores/ui', () => ({
  useUIStore: vi.fn(() => ({})),
}))

// Mock qmessage
vi.mock('@/utils/qmessage', () => ({
  default: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
    info: vi.fn(),
  },
}))

// Mock mentions — 保留 decodeToPlainText 真实逻辑
vi.mock('@/utils/mentions', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/utils/mentions')>()
  return {
    ...actual,
  }
})

// clipboard mock
const mockWriteText = vi.fn().mockResolvedValue(undefined)
const mockClipboardWrite = vi.fn().mockResolvedValue(undefined)

// 可变 selection 文本，用于确认复制函数不再依赖点击菜单后的实时选区
let mockSelectionText = ''

beforeEach(() => {
  mockSelectionText = ''
  vi.stubGlobal('navigator', {
    clipboard: {
      writeText: mockWriteText,
      write: mockClipboardWrite,
    },
  })
  vi.stubGlobal('window', {
    ...window,
    getSelection: () => ({
      toString: () => mockSelectionText,
    }),
  })
  mockWriteText.mockClear()
  mockClipboardWrite.mockClear()
})

afterEach(() => {
  vi.unstubAllGlobals()
})

const makeMessage = (overrides: Partial<Message> = {}): Message => ({
  id: '1',
  content: '这是一条测试消息',
  type: 'text',
  conversationId: '1',
  senderId: 2,
  senderName: 'test',
  timestamp: Date.now(),
  isSelf: false,
  ...overrides,
})

describe('copyMessage — 选中内容优先', () => {
  it('菜单打开后选区变化时，仍复制菜单打开时的完整选中内容', async () => {
    const serverUrl = ref('http://localhost:3000')
    const currentUser = ref({ id: 1 })
    const { copyMessage } = useMessageActions(serverUrl, currentUser)

    mockSelectionText = ''
    await copyMessage(makeMessage({ content: '当前消息内容', type: 'text' }), '跨消息选中的全部内容')

    expect(mockWriteText).toHaveBeenCalledWith('跨消息选中的全部内容')
  })

  it('保留菜单打开时选中内容的首尾空白', async () => {
    const { copyMessage } = useMessageActions(ref('http://localhost:3000'), ref({ id: 1 }))

    await copyMessage(makeMessage(), '  跨消息选中内容\n')

    expect(mockWriteText).toHaveBeenCalledWith('  跨消息选中内容\n')
  })

  it('即使右键消息内容为空，也复制已保存的选中内容', async () => {
    const { copyMessage } = useMessageActions(ref('http://localhost:3000'), ref({ id: 1 }))

    await copyMessage(makeMessage({ content: '' }), '来自其他消息的选中内容')

    expect(mockWriteText).toHaveBeenCalledWith('来自其他消息的选中内容')
  })

  it('无选中时复制整条消息内容', async () => {
    mockSelectionText = ''

    const serverUrl = ref('http://localhost:3000')
    const currentUser = ref({ id: 1 })
    const { copyMessage } = useMessageActions(serverUrl, currentUser)

    const message = makeMessage({ content: '完整消息内容', type: 'text' })
    await copyMessage(message)

    expect(mockWriteText).toHaveBeenCalledWith('完整消息内容')
  })

})
