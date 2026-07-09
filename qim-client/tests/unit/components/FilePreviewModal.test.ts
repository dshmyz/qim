import { mount, flushPromises } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import FilePreviewModal from '../../../src/components/apps/file/FilePreviewModal.vue'

const { previewFile } = vi.hoisted(() => ({
  previewFile: vi.fn(),
}))

vi.mock('../../../src/api/file', () => ({
  fileApi: {
    previewFile,
  },
}))

describe('FilePreviewModal', () => {
  function deferred<T>() {
    let resolve!: (value: T) => void
    const promise = new Promise<T>((res) => {
      resolve = res
    })
    return { promise, resolve }
  }

  beforeEach(() => {
    previewFile.mockReset()
  })

  afterEach(() => {
    document.body.innerHTML = ''
    vi.restoreAllMocks()
  })

  it('shows loading instead of mounting the PDF canvas before the preview blob URL is ready', async () => {
    const pendingPreview = deferred<{ data: Blob }>()
    previewFile.mockReturnValueOnce(pendingPreview.promise)
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:pdf')
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})

    mount(FilePreviewModal, {
      attachTo: document.body,
      props: {
        visible: true,
        file: {
          id: 1,
          user_id: 1,
          name: 'report.pdf',
          original_name: 'report.pdf',
          size: 1024,
          mime_type: 'application/pdf',
          storage_path: '/uploads/report.pdf',
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
      },
    })

    await flushPromises()

    expect(document.body.querySelector('.loading-preview')).not.toBeNull()
    expect(document.body.querySelector('.pdf-preview')).toBeNull()

    pendingPreview.resolve({
      data: new Blob(['pdf'], { type: 'application/pdf' }),
    })
    await flushPromises()
    await flushPromises()

    expect(document.body.querySelector('.pdf-preview')).not.toBeNull()
  })

  it('shows unsupported preview without requesting preview blob for non-previewable files', async () => {
    mount(FilePreviewModal, {
      attachTo: document.body,
      props: {
        visible: true,
        file: {
          id: 2,
          user_id: 1,
          name: 'archive.zip',
          original_name: 'archive.zip',
          size: 2048,
          mime_type: 'application/zip',
          storage_path: '/uploads/archive.zip',
          checksum: 'zip',
          folder_id: null,
          source: 'upload',
          source_id: null,
          is_starred: false,
          starred_at: null,
          tags: null,
          created_at: '2026-07-09T00:00:00Z',
          updated_at: '2026-07-09T00:00:00Z',
        },
      },
    })

    await flushPromises()

    expect(previewFile).not.toHaveBeenCalled()
    expect(document.body.textContent).toContain('此文件类型暂不支持在线预览')
    expect(document.body.textContent).not.toContain('预览加载失败')
  })
})
