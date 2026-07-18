import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import GroupFilesPanel from '@/components/groups/GroupFilesPanel.vue'

const { list, attach, download } = vi.hoisted(() => ({
  list: vi.fn(() => Promise.resolve({ data: { data: { files: [], folders: [], total: 0, page: 1, page_size: 20 } } })),
  attach: vi.fn(),
  download: vi.fn(),
}))
const { uploadFile } = vi.hoisted(() => ({ uploadFile: vi.fn() }))

vi.mock('@/api/groupFiles', () => ({
  groupFiles: { list, attach, download },
}))

vi.mock('@/api/file', () => ({
  fileApi: { uploadFile },
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

  it('uploads a member file then attaches it to the current group directory', async () => {
    uploadFile.mockResolvedValueOnce({ data: { data: { id: 24 } } })
    attach.mockResolvedValueOnce({})
    const wrapper = mount(GroupFilesPanel, { props: { groupId: 7, canManage: false } })
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      value: [new File(['report'], 'report.txt', { type: 'text/plain' })],
    })

    await input.trigger('change')
    await flushPromises()

    expect(uploadFile).toHaveBeenCalledWith(expect.any(File))
    expect(attach).toHaveBeenCalledWith(7, 24, null)
  })

  it('downloads files through the group-scoped endpoint', async () => {
    list.mockResolvedValueOnce({ data: { data: {
      files: [{ id: 9, user_id: 1, name: 'report.txt', size: 3, created_at: '2026-07-18T00:00:00Z' }],
      folders: [], total: 1, page: 1, page_size: 20,
    } } })
    download.mockResolvedValueOnce({ data: new Blob(['report']) })
    const wrapper = mount(GroupFilesPanel, { props: { groupId: 7, canManage: false } })
    await flushPromises()

    await wrapper.find('.file-list__actions button').trigger('click')
    await flushPromises()

    expect(download).toHaveBeenCalledWith(7, 9)
  })
})
