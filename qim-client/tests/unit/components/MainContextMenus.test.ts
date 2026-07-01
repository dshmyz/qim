import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import MainContextMenus from '@/components/menus/MainContextMenus.vue'

const baseProps = {
  showMenu: false,
  selectedConversation: null,
  menuPosition: { x: 0, y: 0 },
  showActionMenuFlag: false,
  actionMenuPosition: { x: 0, y: 0 },
  showUserContextMenuFlag: false,
  userContextMenuPosition: { x: 0, y: 0 },
  selectedEmployee: null,
  showMemberContextMenuFlag: false,
  memberContextMenuPosition: { x: 0, y: 0 },
  showGroupContextMenuFlag: false,
  groupContextMenuPosition: { x: 0, y: 0 },
  selectedGroupForContextMenu: null,
  isGroupOwner: false,
  showSettingsMenuFlag: false,
  settingsMenuPosition: { x: 0, y: 0 },
  showThemeMenuFlag: false,
  themeMenuPosition: { x: 0, y: 0 },
  showMoreMenuFlag: false,
  moreMenuPosition: { x: 0, y: 0 },
  currentUser: {},
}

describe('MainContextMenus settings menu', () => {
  it('opens feedback from the settings menu', async () => {
    const wrapper = mount(MainContextMenus, {
      props: {
        ...baseProps,
        showSettingsMenuFlag: true,
      },
    })

    const menuItems = wrapper.findAll('.context-menu-item').map(item => item.text())
    expect(menuItems.slice(0, 4)).toEqual(['关于', '问题反馈', '检查更新', '设置'])

    const feedbackItem = wrapper
      .findAll('.context-menu-item')
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
    const wrapper = mount(MainContextMenus, {
      props: {
        ...baseProps,
        showGroupContextMenuFlag: true,
        selectedGroupForContextMenu: selectedGroup,
      },
    })

    const exitItem = wrapper
      .findAll('.context-menu-item')
      .find(item => item.text().includes('退出群聊'))

    expect(exitItem).toBeTruthy()
    await exitItem!.trigger('click')

    expect(wrapper.emitted('exitGroup')?.[0]).toEqual([selectedGroup])
  })

  it('hides 退出群聊 and shows 解散群聊 for the group owner', () => {
    const selectedGroup = { id: 'group-1', name: '测试群', type: 'group' }
    const wrapper = mount(MainContextMenus, {
      props: {
        ...baseProps,
        showGroupContextMenuFlag: true,
        selectedGroupForContextMenu: selectedGroup,
        isGroupOwner: true,
      },
    })

    const labels = wrapper.findAll('.context-menu-item').map(item => item.text())
    expect(labels.some(text => text.includes('解散群聊'))).toBe(true)
    expect(labels.some(text => text.includes('退出群聊'))).toBe(false)
  })

  it('shows 退出群聊 and hides 解散群聊 for a non-owner', () => {
    const selectedGroup = { id: 'group-1', name: '测试群', type: 'group' }
    const wrapper = mount(MainContextMenus, {
      props: {
        ...baseProps,
        showGroupContextMenuFlag: true,
        selectedGroupForContextMenu: selectedGroup,
        isGroupOwner: false,
      },
    })

    const labels = wrapper.findAll('.context-menu-item').map(item => item.text())
    expect(labels.some(text => text.includes('退出群聊'))).toBe(true)
    expect(labels.some(text => text.includes('解散群聊'))).toBe(false)
  })
})
