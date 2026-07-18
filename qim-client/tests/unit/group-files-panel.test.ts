import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import GroupFilesPanel from '@/components/groups/GroupFilesPanel.vue'

vi.mock('@/api/groupFiles', () => ({
  groupFiles: {
    list: () => Promise.resolve({ data: { data: { files: [], folders: [], total: 0, page: 1, page_size: 20 } } }),
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
})
