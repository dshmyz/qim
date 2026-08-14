import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it } from 'vitest'
import MainContextMenus from '@/components/menus/MainContextMenus.vue'
import { openMenu, closeMenu } from '@/composables/useUI'

// 仅使用 MainContextMenus.vue 声明的 props
const baseProps = {
  selectedConversation: null,
  selectedEmployee: null,
  selectedGroupForContextMenu: null,
  isGroupOwner: false,
  currentUser: {},
}

// 菜单可见性由 useUI 模块级 activeMenu 门控（UniversalContextMenu 的
// isVisible = activeMenu === menuId），通过真实 API openMenu 打开对应菜单
const mountWithMenu = (menuId: string, props: Record<string, unknown> = {}) => {
  openMenu(menuId, 0, 0)
  return mount(MainContextMenus, {
    props: { ...baseProps, ...props },
    global: { stubs: { Teleport: true } },
  })
}

beforeEach(() => {
  // 重置全局菜单状态，避免用例间 activeMenu 残留互相干扰
  closeMenu()
})

describe('MainContextMenus settings menu', () => {
  it('opens feedback from the settings menu', async () => {
    const wrapper = mountWithMenu('settings')

    const menuItems = wrapper.findAll('.ucm-item').map(item => item.text())
    expect(menuItems.slice(0, 4)).toEqual(['关于', '问题反馈', '检查更新', '设置'])

    const feedbackItem = wrapper
      .findAll('.ucm-item')
      .find(item => item.text().includes('问题反馈'))

    expect(feedbackItem).toBeTruthy()
    await feedbackItem!.trigger('click')

    expect(wrapper.emitted('openFeedback')).toHaveLength(1)
    expect(wrapper.emitted('closeAllMenus')).toHaveLength(1)
  })
})

describe('MainContextMenus group menu', () => {
  it('emits the selected group when exiting from the group context menu', async () => {
    const selectedGroup = { id: 'group-1', name: '测试群', type: 'group' }
    const wrapper = mountWithMenu('group', { selectedGroupForContextMenu: selectedGroup })

    const exitItem = wrapper
      .findAll('.ucm-item')
      .find(item => item.text().includes('退出群聊'))

    expect(exitItem).toBeTruthy()
    await exitItem!.trigger('click')

    expect(wrapper.emitted('exitGroup')?.[0]).toEqual([selectedGroup])
  })

  it('hides 退出群聊 and shows 解散群聊 for the group owner', () => {
    const selectedGroup = { id: 'group-1', name: '测试群', type: 'group' }
    const wrapper = mountWithMenu('group', {
      selectedGroupForContextMenu: selectedGroup,
      isGroupOwner: true,
    })

    const labels = wrapper.findAll('.ucm-item').map(item => item.text())
    expect(labels.some(text => text.includes('解散群聊'))).toBe(true)
    expect(labels.some(text => text.includes('退出群聊'))).toBe(false)
  })

  it('shows 退出群聊 and hides 解散群聊 for a non-owner', () => {
    const selectedGroup = { id: 'group-1', name: '测试群', type: 'group' }
    const wrapper = mountWithMenu('group', {
      selectedGroupForContextMenu: selectedGroup,
      isGroupOwner: false,
    })

    const labels = wrapper.findAll('.ucm-item').map(item => item.text())
    expect(labels.some(text => text.includes('退出群聊'))).toBe(true)
    expect(labels.some(text => text.includes('解散群聊'))).toBe(false)
  })
})
