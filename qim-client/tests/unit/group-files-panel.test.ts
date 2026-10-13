import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import GroupFilesPanel from '@/components/groups/GroupFilesPanel.vue'

const { list, attach, createFolder, downloadUrl, shareReference, remove, startFileDownload } = vi.hoisted(() => ({
  list: vi.fn(() => Promise.resolve({ data: { data: { files: [], folders: [], total: 0, page: 1, page_size: 20 } } })),
  attach: vi.fn(),
  createFolder: vi.fn(),
  downloadUrl: vi.fn((groupId: number, fileId: number) => `/api/v1/groups/${groupId}/files/${fileId}/download`),
  shareReference: vi.fn(),
  remove: vi.fn(),
  startFileDownload: vi.fn(),
}))
const { uploadFile } = vi.hoisted(() => ({ uploadFile: vi.fn() }))

vi.mock('@/api/groupFiles', () => ({
  groupFiles: { list, attach, createFolder, downloadUrl, shareReference, remove },
}))

vi.mock('@/composables/useFileDownload', () => ({
  useFileDownload: () => ({ downloadFile: startFileDownload, saveFileAs: vi.fn() }),
}))

vi.mock('@/composables/useFileUpload', () => ({
  uploadFile,
  uploadFilesWithLimit: async (files: File[] | FileList, folderId?: number, options?: {
    onFileUploaded?: (file: File, fileId: number) => Promise<void>
  }): Promise<Array<{ file: File; success: boolean; fileId?: number }>> => {
    const list = Array.from(files)
    const results: Array<{ file: File; success: boolean; fileId?: number }> = []
    for (const file of list) {
      const result = await uploadFile(file, folderId)
      if (result?.fileId && options?.onFileUploaded) {
        await options.onFileUploaded(file, result.fileId)
      }
      results.push({ file, success: !!result?.fileId, fileId: result?.fileId })
    }
    return results
  },
}))

describe('GroupFilesPanel', () => {
  it('shows upload and download to every member but directory management only to managers', async () => {
    const member = mount(GroupFilesPanel, { props: { groupId: 1, canManage: false } })
    expect(member.text()).toContain('上传文件')
    expect(member.text()).toContain('下载')
    expect(member.text()).not.toContain('新建文件夹')

    const manager = mount(GroupFilesPanel, { props: { groupId: 1, canManage: true } })
    expect(manager.text()).toContain('新建文件夹')
  })

  it('uploads a member file via the generic uploader then attaches it to the group directory', async () => {
    uploadFile.mockResolvedValueOnce({ fileId: 24 })
    attach.mockResolvedValueOnce({})
    const wrapper = mount(GroupFilesPanel, { props: { groupId: 7, canManage: false } })
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      value: [new File(['report'], 'report.txt', { type: 'text/plain' })],
    })

    await input.trigger('change')
    await flushPromises()

    expect(uploadFile).toHaveBeenCalledTimes(1)
    expect(uploadFile.mock.calls[0][0]).toMatchObject({ name: 'report.txt', type: 'text/plain' })
    expect(attach).toHaveBeenCalledWith(7, 24, null)
  })

  it('opens an in-app folder name dialog instead of using the native prompt', async () => {
    const wrapper = mount(GroupFilesPanel, { props: { groupId: 7, canManage: true } })

    await wrapper.get('button.gfp-btn--secondary').trigger('click')

    expect(wrapper.get('[role="dialog"] input[aria-label="文件夹名称"]').exists()).toBe(true)
    expect(wrapper.get('[role="dialog"] h3').text()).toBe('新建文件夹')
  })

  it('downloads files through the unified downloader with the group-scoped URL', async () => {
    list.mockResolvedValueOnce({ data: { data: {
      files: [{ id: 9, user_id: 1, name: 'report.txt', size: 3, created_at: '2026-07-18T00:00:00Z' }],
      folders: [], total: 1, page: 1, page_size: 20,
    } } })
    startFileDownload.mockResolvedValueOnce('download_1')
    const wrapper = mount(GroupFilesPanel, { props: { groupId: 7, canManage: false } })
    await flushPromises()

    await wrapper.find('.gfp-table__actions button').trigger('click')
    await flushPromises()

    expect(startFileDownload).toHaveBeenCalledWith({
      name: 'report.txt',
      size: 3,
      url: '/api/v1/groups/7/files/9/download',
    })
  })

  it('shares a chat attachment using its message and file IDs', async () => {
    shareReference.mockResolvedValueOnce({})
    const wrapper = mount(GroupFilesPanel, {
      props: { groupId: 7, canManage: true, referenceMessageId: 18, referenceFileId: 24 },
    })

    await wrapper.get('.gfp-callout button').trigger('click')
    await flushPromises()

    expect(shareReference).toHaveBeenCalledWith(7, 18, 24, null)
  })

  it('deletes every selected file in a single batch action', async () => {
    list.mockResolvedValueOnce({ data: { data: {
      files: [
        { id: 9, user_id: 1, name: 'a.txt', size: 3, created_at: '2026-07-18T00:00:00Z' },
        { id: 10, user_id: 1, name: 'b.txt', size: 3, created_at: '2026-07-18T00:00:00Z' },
      ],
      folders: [], total: 2, page: 1, page_size: 20,
    } } })
    remove.mockResolvedValue({})
    const wrapper = mount(GroupFilesPanel, { props: { groupId: 7, canManage: true } })
    await flushPromises()

    await wrapper.get('input[aria-label="全选当前页"]').setValue(true)
    expect(wrapper.text()).toContain('已选 2 项')

    await wrapper.get('.gfp-batch-bar .gfp-btn--danger').trigger('click')
    await wrapper.get('form.gfp-dialog').trigger('submit')
    await flushPromises()

    expect(remove).toHaveBeenCalledWith(7, 9)
    expect(remove).toHaveBeenCalledWith(7, 10)
  })

  it('requests the list with sort params when a column header is clicked', async () => {
    list.mockResolvedValue({ data: { data: {
      files: [{ id: 1, user_id: 1, name: 'a.txt', size: 1, created_at: '' }],
      folders: [], total: 1, page: 1, page_size: 20,
    } } })
    const wrapper = mount(GroupFilesPanel, { props: { groupId: 7, canManage: false } })
    await flushPromises()

    list.mockClear()
    await wrapper.get('.gfp-sort').trigger('click')
    await flushPromises()

    expect(list).toHaveBeenCalledWith(7, expect.objectContaining({ sort_by: 'name', sort_order: 'asc' }))
  })

  it('keeps the newest group response when group requests resolve out of order', async () => {    let resolveGroupOne: (value: any) => void
    let resolveGroupTwo: (value: any) => void
    list
      .mockImplementationOnce(() => new Promise(resolve => { resolveGroupOne = resolve }))
      .mockImplementationOnce(() => new Promise(resolve => { resolveGroupTwo = resolve }))

    const wrapper = mount(GroupFilesPanel, { props: { groupId: 1, canManage: false } })
    await wrapper.setProps({ groupId: 2 })

    resolveGroupTwo!({ data: { data: {
      files: [{ id: 2, user_id: 2, name: 'group-two.txt', size: 2, created_at: '' }],
      folders: [{ id: 22, name: '群组二' }], total: 1, page: 1, page_size: 20,
    } } })
    await flushPromises()

    resolveGroupOne!({ data: { data: {
      files: [{ id: 1, user_id: 1, name: 'group-one.txt', size: 1, created_at: '' }],
      folders: [{ id: 11, name: '群组一' }], total: 1, page: 1, page_size: 20,
    } } })
    await flushPromises()

    expect(wrapper.text()).toContain('group-two.txt')
    expect(wrapper.text()).toContain('群组二')
    expect(wrapper.text()).not.toContain('group-one.txt')
    expect(wrapper.text()).not.toContain('群组一')
  })
})
