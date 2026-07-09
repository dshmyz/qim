import { mount, flushPromises } from '@vue/test-utils'
import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import FileManagementApp from '../../../src/components/apps/FileManagementApp.vue'

const { mockFile, mockDownloadFile } = vi.hoisted(() => ({
  mockFile: {
    id: 1,
    user_id: 1,
    name: 'hello.txt',
    original_name: 'hello.txt',
    size: 12,
    mime_type: 'text/plain',
    storage_path: '/uploads/hello.txt',
    checksum: 'abc',
    folder_id: null,
    source: 'upload',
    source_id: null,
    is_starred: false,
    starred_at: null,
    tags: null,
    created_at: '2026-07-09T00:00:00Z',
    updated_at: '2026-07-09T00:00:00Z',
  },
  mockDownloadFile: vi.fn(),
}))

vi.mock('../../../src/api/file', () => ({
  fileApi: {
    getFiles: vi.fn().mockResolvedValue({
      data: {
        code: 0,
        data: {
          files: [mockFile],
          total: 1,
          page: 1,
          page_size: 20,
        },
      },
    }),
    previewFile: vi.fn().mockResolvedValue({
      data: new Blob(['hello world'], { type: 'text/plain' }),
    }),
    downloadFile: mockDownloadFile,
    toggleStar: vi.fn(),
    deleteFile: vi.fn(),
  },
  folderApi: {
    getFolderTree: vi.fn().mockResolvedValue({
      data: {
        code: 0,
        data: [],
      },
    }),
  },
}))

vi.mock('../../../src/composables/useFileUpload', () => ({
  useFileUpload: () => ({ tasks: { value: [] } }),
  uploadFile: vi.fn(),
}))

vi.mock('../../../src/stores/upload', () => ({
  useUploadStore: () => ({}),
}))

vi.mock('../../../src/utils/qmessage', () => ({
  default: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

describe('FileManagementApp preview', () => {
  const originalCreateObjectURL = URL.createObjectURL
  const originalRevokeObjectURL = URL.revokeObjectURL

  beforeEach(() => {
    URL.createObjectURL = vi.fn(() => 'blob:preview')
    URL.revokeObjectURL = vi.fn()
    mockDownloadFile.mockReset()
    mockDownloadFile.mockResolvedValue({
      data: new Blob(['downloaded'], { type: 'text/plain' }),
    })
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      text: () => Promise.resolve('hello world'),
    }))
  })

  afterEach(() => {
    document.body.innerHTML = ''
    URL.createObjectURL = originalCreateObjectURL
    URL.revokeObjectURL = originalRevokeObjectURL
    vi.unstubAllGlobals()
  })

  function mountApp() {
    return mount(FileManagementApp, {
      attachTo: document.body,
      global: {
        stubs: {
          UploadProgressBar: true,
          CreateFolderModal: true,
          FileActionsModal: true,
        },
      },
    })
  }

  it('opens a stable preview modal when a file row is double-clicked', async () => {
    const wrapper = mountApp()
    await flushPromises()

    await wrapper.find('.file-list-item').trigger('dblclick')
    await flushPromises()

    expect(document.body.querySelector('.file-preview-modal')).not.toBeNull()
  })

  it('does not use the globally hidden modal overlay class for the preview modal', async () => {
    const wrapper = mountApp()
    await flushPromises()

    await wrapper.find('.file-list-item').trigger('dblclick')
    await flushPromises()

    expect(document.body.querySelector('.file-preview-modal.modal-overlay')).toBeNull()
  })

  it('renders the preview modal through the shared dialog component', async () => {
    const wrapper = mountApp()
    await flushPromises()

    await wrapper.find('.file-list-item').trigger('dblclick')
    await flushPromises()

    expect(document.body.querySelector('.q-dialog.file-preview-modal')).not.toBeNull()
    expect(document.body.querySelector('.file-preview-overlay')).toBeNull()
  })

  it('opens the preview modal when the row preview button is clicked', async () => {
    const wrapper = mountApp()
    await flushPromises()

    const row = wrapper.find('.file-list-item')
    await row.trigger('mouseenter')
    await flushPromises()

    const previewButton = wrapper.find('button[title="预览"]')
    expect(previewButton.exists()).toBe(true)

    await previewButton.trigger('click')
    await flushPromises()

    expect(document.body.querySelector('.file-preview-modal')).not.toBeNull()
  })

  it('opens the preview modal from the file row context menu', async () => {
    const wrapper = mountApp()
    await flushPromises()

    await wrapper.find('.file-list-item').trigger('contextmenu', {
      clientX: 24,
      clientY: 32,
    })
    await flushPromises()

    const previewAction = Array.from(document.body.querySelectorAll('.context-menu-item'))
      .find((item) => item.textContent?.includes('预览')) as HTMLElement | undefined

    expect(previewAction).toBeTruthy()
    previewAction?.click()
    await flushPromises()

    expect(document.body.querySelector('.file-preview-modal')).not.toBeNull()
  })

  it('downloads the file when the row download button is clicked', async () => {
    const click = vi.fn()
    const appendChild = vi.spyOn(document.body, 'appendChild')
    const removeChild = vi.spyOn(document.body, 'removeChild')
    vi.spyOn(document, 'createElement').mockImplementation(((tagName: string) => {
      const element = document.createElementNS('http://www.w3.org/1999/xhtml', tagName) as HTMLElement
      if (tagName.toLowerCase() === 'a') {
        Object.assign(element, { click })
      }
      return element
    }) as typeof document.createElement)

    const wrapper = mountApp()
    await flushPromises()

    await wrapper.find('.file-list-item').trigger('mouseenter')
    await flushPromises()
    await wrapper.find('button[title="下载"]').trigger('click')
    await flushPromises()

    expect(mockDownloadFile).toHaveBeenCalledWith(mockFile.id)
    expect(click).toHaveBeenCalledTimes(1)
    expect(appendChild).toHaveBeenCalled()
    expect(removeChild).toHaveBeenCalled()
  })

  it('dispatches the selected file when the row share button is clicked', async () => {
    const dispatchEvent = vi.spyOn(window, 'dispatchEvent')
    const wrapper = mountApp()
    await flushPromises()

    await wrapper.find('.file-list-item').trigger('mouseenter')
    await flushPromises()
    await wrapper.find('button[title="分享"]').trigger('click')
    await flushPromises()

    expect(dispatchEvent).toHaveBeenCalledWith(expect.objectContaining({
      type: 'openShareModal',
      detail: { type: 'file', data: mockFile },
    }))
  })

  it('closes the preview modal before opening share from the preview footer', async () => {
    const dispatchEvent = vi.spyOn(window, 'dispatchEvent')
    const wrapper = mountApp()
    await flushPromises()

    await wrapper.find('.file-list-item').trigger('dblclick')
    await flushPromises()

    const shareButton = Array.from(document.body.querySelectorAll('.file-preview-modal .q-dialog__footer button'))
      .find((button) => button.textContent?.includes('分享')) as HTMLButtonElement | undefined

    expect(shareButton).toBeTruthy()
    shareButton?.click()
    await flushPromises()

    expect(dispatchEvent).toHaveBeenCalledWith(expect.objectContaining({
      type: 'openShareModal',
      detail: { type: 'file', data: mockFile },
    }))
    expect(document.body.querySelector('.file-preview-modal')).toBeNull()
  })
})
