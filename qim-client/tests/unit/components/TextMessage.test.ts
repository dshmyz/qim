import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
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

    it('自身消息的链接使用高对比样式钩子', () => {
      const wrapper = mount(TextMessage, {
        props: { content: 'https://bigmodel.cn/glm-coding', isSelf: true },
      })

      expect(wrapper.find('a').classes()).toContain('message-link--self')
    })

    it('正确解析自身消息中的 Markdown 链接，不把括号吞进 URL', () => {
      const url = 'https://bigmodel.cn/glm-coding'
      const wrapper = mount(TextMessage, {
        props: { content: `@v1.0.0 [${url}](${url})`, isSelf: true },
      })

      const links = wrapper.findAll('a')
      expect(links).toHaveLength(1)
      expect(links[0].attributes('href')).toBe(url)
      expect(links[0].text()).toBe(url)
      expect(wrapper.text()).toBe(`@v1.0.0 ${url}`)
    })

    it('keeps message links readable without underlining them', () => {
      const source = readFileSync(resolve(__dirname, '../../../src/components/message/TextMessage.vue'), 'utf8')

      expect(source).toContain('text-decoration: none;')
      expect(source).not.toContain('text-decoration: underline;')
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

    it('不会把 Go 模块 URL 中的 @版本号作为用户提及', () => {
      const url = 'https://bigmodel.cn/glm-coding/@v/v1.0.20.info'
      const wrapper = mount(TextMessage, {
        props: { content: `go: reading ${url}: 502 Bad Gateway` },
      })

      expect(wrapper.findAll('.at-user')).toHaveLength(0)
      expect(wrapper.find('a').attributes('href')).toBe(url)
      expect(wrapper.find('a').text()).toBe(url)
    })

    it('不会把没有协议的 Go 模块路径中的 @版本号作为用户提及', () => {
      const content = 'go get gitee.com/xxx/xxxx/xxx/@v1.0.20'
      const wrapper = mount(TextMessage, { props: { content } })

      expect(wrapper.findAll('.at-user')).toHaveLength(0)
      expect(wrapper.text()).toBe(content)
    })

    it('纯文本中的 URL 能正确 linkify（非 Markdown 格式）', () => {
      const wrapper = mount(TextMessage, {
        props: { content: '访问 https://example.com 看看' },
      })
      const links = wrapper.findAll('a')
      expect(links).toHaveLength(1)
      expect(links[0].attributes('href')).toBe('https://example.com')
      expect(links[0].text()).toBe('https://example.com')
    })
  })

  describe('@用户 高亮', () => {
    it('应该高亮 @用户名', () => {
      const wrapper = mount(TextMessage, {
        props: { content: '@{mention:42|admin} 你好' },
      })
      expect(wrapper.find('.at-mention-chip').exists()).toBe(true)
      expect(wrapper.find('.at-mention-chip').text()).toBe('@admin')
    })

    it('应该处理多个 @用户', () => {
      const wrapper = mount(TextMessage, {
        props: { content: '@{mention:42|admin} @{mention:43|testuser} 请查看' },
      })
      const atUsers = wrapper.findAll('.at-mention-chip')
      expect(atUsers.length).toBe(2)
    })

    it('应该支持中文用户名', () => {
      const wrapper = mount(TextMessage, {
        props: { content: '@{mention:42|%E7%AE%A1%E7%90%86%E5%91%98} 你好' },
      })
      expect(wrapper.find('.at-mention-chip').exists()).toBe(true)
      expect(wrapper.find('.at-mention-chip').text()).toBe('@管理员')
    })
  })

  describe('混合内容', () => {
    it('应该同时处理 URL 和 @用户', () => {
      const wrapper = mount(TextMessage, {
        props: { content: '@{mention:42|admin} 请查看 https://example.com' },
      })
      expect(wrapper.find('.at-mention-chip').exists()).toBe(true)
      expect(wrapper.find('a').exists()).toBe(true)
    })
  })
})
