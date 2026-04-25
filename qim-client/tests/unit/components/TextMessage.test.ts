import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import TextMessage from '@/components/message/TextMessage.vue'

describe('TextMessage', () => {
  describe('基本渲染', () => {
    it('应该渲染文本消息', () => {
      const wrapper = mount(TextMessage, {
        props: { content: 'Hello World' },
      })
      expect(wrapper.find('.text-message').exists()).toBe(true)
      expect(wrapper.text()).toBe('Hello World')
    })

    it('应该为自身消息添加 self class', () => {
      const wrapper = mount(TextMessage, {
        props: { content: 'My message', isSelf: true },
      })
      expect(wrapper.find('.text-message').classes()).toContain('self')
    })

    it('默认不应该有 self class', () => {
      const wrapper = mount(TextMessage, {
        props: { content: 'Other message' },
      })
      expect(wrapper.find('.text-message').classes()).not.toContain('self')
    })
  })

  describe('URL 转换', () => {
    it('应该将 URL 转换为链接', () => {
      const wrapper = mount(TextMessage, {
        props: { content: '访问 https://example.com 查看' },
      })
      const links = wrapper.findAll('a')
      expect(links.length).toBe(1)
      expect(links[0].attributes('href')).toBe('https://example.com')
      expect(links[0].attributes('target')).toBe('_blank')
    })

    it('应该正确处理多个 URL', () => {
      const wrapper = mount(TextMessage, {
        props: { content: 'https://a.com 和 https://b.com' },
      })
      const links = wrapper.findAll('a')
      expect(links.length).toBe(2)
    })

    it('应该处理 http 链接', () => {
      const wrapper = mount(TextMessage, {
        props: { content: '访问 http://example.com' },
      })
      const links = wrapper.findAll('a')
      expect(links.length).toBe(1)
      expect(links[0].attributes('href')).toBe('http://example.com')
    })
  })

  describe('@用户 高亮', () => {
    it('应该高亮 @用户名', () => {
      const wrapper = mount(TextMessage, {
        props: { content: '@admin 你好' },
      })
      expect(wrapper.find('.at-user').exists()).toBe(true)
      expect(wrapper.find('.at-user').text()).toBe('@admin')
    })

    it('应该处理多个 @用户', () => {
      const wrapper = mount(TextMessage, {
        props: { content: '@admin @testuser 请查看' },
      })
      const atUsers = wrapper.findAll('.at-user')
      expect(atUsers.length).toBe(2)
    })

    it('应该支持中文用户名', () => {
      const wrapper = mount(TextMessage, {
        props: { content: '@管理员 你好' },
      })
      expect(wrapper.find('.at-user').exists()).toBe(true)
      expect(wrapper.find('.at-user').text()).toBe('@管理员')
    })
  })

  describe('混合内容', () => {
    it('应该同时处理 URL 和 @用户', () => {
      const wrapper = mount(TextMessage, {
        props: { content: '@admin 请查看 https://example.com' },
      })
      expect(wrapper.find('.at-user').exists()).toBe(true)
      expect(wrapper.find('a').exists()).toBe(true)
    })
  })
})
