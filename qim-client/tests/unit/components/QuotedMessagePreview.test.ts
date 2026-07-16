import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import QuotedMessagePreview from '@/components/chat/QuotedMessagePreview.vue'

describe('QuotedMessagePreview', () => {
  it('formats a merged-forward quote without exposing its JSON payload', () => {
    const wrapper = mount(QuotedMessagePreview, {
      props: {
        quotedMessage: {
          id: 'quoted-1',
          type: 'merged_forward',
          content: JSON.stringify({
            version: 1,
            title: '聊天记录',
            messages: [{ id: 'message-1', type: 'text', content: '第一条', senderName: 'Alice', timestamp: 1 }],
          }),
          sender: { name: 'Alice' },
        },
      },
    })

    expect(wrapper.text()).toContain('[聊天记录]')
    expect(wrapper.text()).toContain('1 条消息')
    expect(wrapper.text()).not.toContain('{"version"')
  })
})
