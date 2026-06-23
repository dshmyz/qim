import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import MessageInput from '@/components/chat/MessageInput.vue'

const members = [
  { id: 'alice', name: 'Alice', username: 'alice-account', avatar: '' },
  { id: 'bob', name: 'Bob', username: 'bobby', avatar: '' },
]

const mountMessageInput = (props: Record<string, unknown> = {}) => mount(MessageInput, {
  props: {
    conversation: { id: 'group-1', type: 'group', members },
    inputMessage: '@ali',
    pendingFiles: [],
    showEmojiPanel: false,
    showAtMembersPanel: true,
    showMiniAppList: false,
    quotedMessage: null,
    isElectron: false,
    getFileIcon: () => '',
    atMembersQuery: 'ali',
    ...props,
  },
  global: {
    stubs: {
      ChatToolbar: true,
      AIQuickActions: true,
      EmojiPanel: true,
      MiniAppManager: true,
      QuotedMessageInput: true,
      PendingFilesPreview: true,
      Avatar: true,
    },
  },
})

describe('MessageInput mention suggestions', () => {
  it('filters candidates from the textarea query', () => {
    const wrapper = mountMessageInput()

    expect(wrapper.findAll('.at-member-item')).toHaveLength(2)
    expect(wrapper.text()).toContain('Alice')
    expect(wrapper.text()).not.toContain('Bob')
  })

  it('filters candidates by username and displays the matched account', () => {
    const wrapper = mountMessageInput({ atMembersQuery: 'bobby' })

    expect(wrapper.findAll('.at-member-item')).toHaveLength(2)
    expect(wrapper.text()).toContain('Bob')
    expect(wrapper.text()).toContain('@bobby')
    expect(wrapper.text()).not.toContain('Alice')
  })

  it('selects a suggestion from textarea Enter instead of sending a message', async () => {
    const wrapper = mountMessageInput()
    const textarea = wrapper.find('textarea')

    await textarea.trigger('keydown', { key: 'ArrowDown' })
    await textarea.trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('select-at-member')).toEqual([[members[0]]])
    expect(wrapper.emitted('handle-keydown')).toBeUndefined()
  })

  it('notifies the parent when the textarea cursor moves', async () => {
    const wrapper = mountMessageInput({ showAtMembersPanel: false })
    const textarea = wrapper.find('textarea')

    await textarea.trigger('keyup', { key: 'ArrowLeft' })
    await textarea.trigger('click')

    expect(wrapper.emitted('cursor-change')).toHaveLength(2)
  })
})
