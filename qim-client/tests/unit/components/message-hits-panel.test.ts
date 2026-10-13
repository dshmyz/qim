import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import MessageHitsPanel from '../../../src/components/chat/MessageHitsPanel.vue'
import type { MessageHit } from '../../../src/components/chat/MessageHitsPanel.vue'

const makeHit = (overrides: Partial<MessageHit> = {}): MessageHit => ({
  id: 1,
  conversationId: 100,
  senderName: '张三',
  time: '10:00',
  previewText: '今天讨论项目排期',
  type: 'text',
  ...overrides,
})

const resolveConv = (id: number | string) =>
  String(id) === '100' ? { name: '项目群', avatar: '/a.png' } : null

const mountPanel = (hits: MessageHit[], extra: Record<string, unknown> = {}) =>
  mount(MessageHitsPanel, {
    props: {
      hits,
      total: hits.length,
      keyword: '',
      serverUrl: '',
      resolveConversation: resolveConv,
      ...extra,
    },
  })

describe('MessageHitsPanel', () => {
  it('按会话分组渲染，组头显示会话名与已加载数', () => {
    const hits = [
      makeHit({ id: 1, conversationId: 100 }),
      makeHit({ id: 2, conversationId: 100 }),
      makeHit({ id: 3, conversationId: 200, senderName: '李四' }),
    ]
    const wrapper = mountPanel(hits)
    expect(wrapper.findAll('.hits-group')).toHaveLength(2)
    const names = wrapper.findAll('.hits-group-name').map(n => n.text())
    expect(names).toContain('项目群')
    expect(names).toContain('会话') // 未解析到的会话显示占位
    const counts = wrapper.findAll('.hits-group-count').map(n => n.text())
    expect(counts).toEqual(['2 条', '1 条'])
  })

  it('文本命中按关键词高亮（大小写不敏感）', () => {
    const wrapper = mountPanel([makeHit({ previewText: 'Project Plan discussion' })], { keyword: 'plan' })
    const mark = wrapper.find('.hits-item-content mark')
    expect(mark.exists()).toBe(true)
    expect(mark.text()).toBe('Plan')
  })

  it('长预览从首个命中位置截断，命中点保留在窗口内', () => {
    const long = '前置背景'.repeat(20) + '目标关键词在这里' + '后续内容'.repeat(20)
    const wrapper = mountPanel([makeHit({ previewText: long })], { keyword: '目标关键词' })
    const content = wrapper.find('.hits-item-content').text()
    expect(content).toContain('目标关键词')
    expect(content.startsWith('…')).toBe(true)
  })

  it('HTML 特殊字符转义，消息内容不注入元素', () => {
    const wrapper = mountPanel([makeHit({ previewText: '<img src=x onerror=alert(1)>' })], { keyword: '' })
    const content = wrapper.find('.hits-item-content')
    expect(content.find('img').exists()).toBe(false)
    expect(content.text()).toContain('<img src=x onerror=alert(1)>')
  })

  it('非文本命中显示类型徽标（图标+文案），不渲染正文', () => {
    const wrapper = mountPanel([makeHit({ previewText: '', type: 'image' })])
    const badge = wrapper.find('.hits-item-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toContain('图片')
    expect(wrapper.find('.hits-item-content').text()).toBe('')
  })

  it('card 是 bot 可交互卡片而非名片——徽标文案与既有消息语义对齐', () => {
    const wrapper = mountPanel([makeHit({ previewText: '', type: 'card' })])
    const badge = wrapper.find('.hits-item-badge')
    expect(badge.text()).toContain('卡片')
    expect(badge.text()).not.toContain('名片')
    expect(badge.find('i').classes()).toContain('fa-rectangle-list')
  })

  it('点击命中 emit select 并携带原始 hit', async () => {
    const hit = makeHit()
    const wrapper = mountPanel([hit])
    await wrapper.find('.hits-item').trigger('click')
    expect(wrapper.emitted('select')![0][0]).toEqual(hit)
  })

  it('点击会话组头 emit openConversation', async () => {
    const wrapper = mountPanel([makeHit()])
    await wrapper.find('.hits-group-header').trigger('click')
    expect(wrapper.emitted('openConversation')![0][0]).toBe(100)
  })

  it('组内已加载条数达到上限时显示"查看更多"，exhausted 后隐藏', () => {
    const hits = Array.from({ length: 5 }, (_, i) => makeHit({ id: i + 1 }))
    const withMore = mountPanel(hits, { total: 12 })
    expect(withMore.find('.hits-more-btn').exists()).toBe(true)

    const exhausted = mountPanel(hits, { total: 12, exhaustedIds: ['100'] })
    expect(exhausted.find('.hits-more-btn').exists()).toBe(false)
  })

  it('组内条数未达上限时不显示"查看更多"', () => {
    const wrapper = mountPanel([makeHit({ id: 1 }), makeHit({ id: 2 })], { total: 12 })
    expect(wrapper.find('.hits-more-btn').exists()).toBe(false)
  })

  it('点击"查看更多" emit loadMore 并携带分组', async () => {
    const hits = Array.from({ length: 5 }, (_, i) => makeHit({ id: i + 1 }))
    const wrapper = mountPanel(hits, { total: 12 })
    await wrapper.find('.hits-more-btn').trigger('click')
    const group = wrapper.emitted('loadMore')![0][0] as { conversationId: number | string, hits: MessageHit[] }
    expect(String(group.conversationId)).toBe('100')
    expect(group.hits).toHaveLength(5)
  })
})
