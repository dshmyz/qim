import { mount } from '@vue/test-utils'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import ContentSkeleton from '@/components/skeleton/ContentSkeleton.vue'

describe('ContentSkeleton', () => {
  it('renders the chat window frame for the chat variant', () => {
    const wrapper = mount(ContentSkeleton, { props: { type: 'chat' as any, count: 6 } })

    expect(wrapper.find('.skeleton-chat').exists()).toBe(true)
    expect(wrapper.find('.skeleton-chat-header').exists()).toBe(true)
    expect(wrapper.find('.skeleton-chat-body').exists()).toBe(true)
    expect(wrapper.find('.skeleton-chat-composer').exists()).toBe(true)
    expect(wrapper.findAll('.skeleton-chat-message')).toHaveLength(6)
  })

  it('uses the chat skeleton for the chat-area suspense fallback', () => {
    const mainView = readFileSync(resolve(process.cwd(), 'src/views/Main.vue'), 'utf8')

    expect(mainView).toContain('<ContentSkeleton type="chat" />')
    expect(mainView).not.toContain('<ContentSkeleton type="recent" :count="6" />')
  })
})
