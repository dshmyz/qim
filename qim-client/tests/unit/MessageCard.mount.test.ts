import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import MessageCard from '@/components/channel/MessageCard.vue'

// 复现并回归：MessageCard setup 期不得因 props 未初始化而抛 TDZ 错误
// （此前 renderedContent 的 immediate watcher 在 props 声明前同步求值导致）
describe('MessageCard 挂载回归', () => {
  it('能在正文为 Markdown 时正常挂载，不抛 ReferenceError', () => {
    const wrapper = mount(MessageCard, {
      props: {
        message: {
          id: '2',
          channel_id: '3',
          sender_id: '4',
          content: '# 标题\n\n你好世界',
          type: 'text',
          created_at: Date.now(),
          sender: { id: '4', name: '测试', avatar: '' }
        } as any,
        channel: { id: '3', name: '菜单', status: 'active' } as any,
        isCreator: true,
        interactive: true
      },
      global: {
        stubs: { Avatar: true }
      }
    })
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.find('.content-text').exists()).toBe(true)
    expect(wrapper.html()).toContain('<h1>标题</h1>')
  })
})
