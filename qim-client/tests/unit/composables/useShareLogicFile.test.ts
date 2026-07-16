import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useShareLogic } from '../../../src/composables/useShareLogic'

vi.mock('../../../src/stores/chat', () => ({
  useChatStore: () => ({
    receiveMessage: vi.fn(),
  }),
}))

vi.mock('../../../src/composables/useCurrentUser', () => ({
  useCurrentUser: () => ({
    currentUser: ref({ id: 1, name: 'Tester' }),
  }),
}))

vi.mock('../../../src/composables/useServerUrl', () => ({
  useServerUrl: () => ({
    serverUrl: ref('http://localhost:8080'),
  }),
}))

vi.mock('../../../src/composables/useRequest', () => ({
  request: vi.fn(),
}))

vi.mock('../../../src/utils/qmessage', () => ({
  default: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('../../../src/utils/logger', () => ({
  logger: {
    log: vi.fn(),
    error: vi.fn(),
  },
}))

describe('useShareLogic file content', () => {
  it('preserves the source file URL when forwarding a file with an id', () => {
    const { buildFileContent } = useShareLogic(
      ref(null),
      ref('file'),
      ref([]),
      ref([]),
      ref([]),
      ref(null),
      vi.fn(),
      vi.fn(),
      vi.fn(),
    )

    const content = JSON.parse(buildFileContent({
      id: 7,
      name: 'report.pdf',
      size: 1024,
      mime_type: 'application/pdf',
      storage_path: 'uploads/2026/07/report.pdf',
    }))

    expect(content).toEqual({
      url: '/uploads/2026/07/report.pdf',
      id: 7,
      name: 'report.pdf',
      size: 1024,
      mimeType: 'application/pdf',
    })
  })
})
