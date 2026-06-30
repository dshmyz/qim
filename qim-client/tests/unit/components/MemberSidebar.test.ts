import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import MemberSidebar from '../../../src/components/chat/MemberSidebar.vue'

const members = [
  { id: 'owner', name: '群主', avatar: '', role: 'owner' as const },
  { id: 'admin', name: '管理员', avatar: '', role: 'admin' as const },
  { id: 'member', name: '成员', avatar: '', role: 'member' as const },
]

const mountSidebar = (props: Record<string, unknown>) => mount(MemberSidebar, {
  props: {
    members,
    isExpanded: false,
    showSearch: false,
    searchQuery: '',
    ...props,
  },
  global: {
    stubs: {
      Avatar: {
        props: ['name'],
        template: '<div class="avatar-stub">{{ name }}</div>',
      },
      ToggleSidebarBtn: true,
    },
  },
})

const mountCollapsedSidebar = () => mountSidebar({})

describe('MemberSidebar member filtering', () => {
  it('hides disabled and deleted members', () => {
    const wrapper = mountSidebar({
      isExpanded: true,
      members: [
        { id: 'active', name: '正常成员', avatar: '', role: 'member' as const },
        { id: 'disabled', name: '禁用成员', avatar: '', role: 'member' as const, disabled: true },
        { id: 'is-disabled', name: '停用成员', avatar: '', role: 'member' as const, is_disabled: true },
        { id: 'status-disabled', name: '状态禁用成员', avatar: '', role: 'member' as const, status: 'disabled' },
        { id: 'deleted-at', name: '删除成员', avatar: '', role: 'member' as const, deletedAt: 1710000000 },
        { id: 'deleted-snake', name: '软删成员', avatar: '', role: 'member' as const, deleted_at: '2026-06-30T00:00:00Z' },
        { id: 'is-deleted', name: '已删成员', avatar: '', role: 'member' as const, is_deleted: true },
      ],
    })

    expect(wrapper.findAll('.member-item')).toHaveLength(1)
    expect(wrapper.text()).toContain('正常成员')
    expect(wrapper.text()).not.toContain('禁用成员')
    expect(wrapper.text()).not.toContain('停用成员')
    expect(wrapper.text()).not.toContain('状态禁用成员')
    expect(wrapper.text()).not.toContain('删除成员')
    expect(wrapper.text()).not.toContain('软删成员')
    expect(wrapper.text()).not.toContain('已删成员')
  })
})

describe('MemberSidebar collapsed state', () => {
  it('renders collapsed member avatars and keeps member interactions', async () => {
    const wrapper = mountCollapsedSidebar()

    const collapsedMembers = wrapper.findAll('.collapsed-avatar-item')
    expect(collapsedMembers).toHaveLength(3)
    expect(wrapper.find('.collapsed-role.owner').exists()).toBe(true)
    expect(wrapper.find('.collapsed-role.admin').exists()).toBe(true)

    await collapsedMembers[2].trigger('contextmenu')
    await collapsedMembers[2].trigger('dblclick')

    expect(wrapper.emitted('show-member-context-menu')?.[0]?.[1]).toEqual(members[2])
    expect(wrapper.emitted('start-private-chat')?.[0]).toEqual([members[2]])
  })

  it('anchors collapsed member context menu near the avatar', async () => {
    const wrapper = mountCollapsedSidebar()
    const collapsedMember = wrapper.findAll('.collapsed-avatar-item')[2]

    vi.spyOn(collapsedMember.element, 'getBoundingClientRect').mockReturnValue({
      x: 420,
      y: 160,
      left: 420,
      top: 160,
      right: 456,
      bottom: 196,
      width: 36,
      height: 36,
      toJSON: () => ({}),
    } as DOMRect)

    await collapsedMember.trigger('contextmenu', {
      clientX: 900,
      clientY: 500,
    })

    const event = wrapper.emitted('show-member-context-menu')?.[0]?.[0] as MouseEvent
    expect(event.clientX).toBe(456)
    expect(event.clientY).toBe(160)
  })
})
