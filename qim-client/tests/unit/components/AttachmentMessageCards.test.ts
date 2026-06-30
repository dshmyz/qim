import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import FileMessage from '@/components/message/FileMessage.vue'
import MiniAppMessage from '@/components/message/MiniAppMessage.vue'
import NewsMessage from '@/components/message/NewsMessage.vue'
import ShareMessage from '@/components/message/ShareMessage.vue'
import { generateAvatar } from '@/utils/avatar'

describe('attachment message card style', () => {
  it('renders file messages as a lightweight attachment card', () => {
    const wrapper = mount(FileMessage, {
      props: {
        content: JSON.stringify({
          url: '/uploads/product.pdf',
          name: '产品需求说明.pdf',
          size: 2516582,
          mimeType: 'application/pdf',
        }),
        serverUrl: 'http://localhost:8080',
      },
    })

    expect(wrapper.find('.attachment-card').exists()).toBe(true)
    expect(wrapper.find('.attachment-card__icon').exists()).toBe(true)
    expect(wrapper.find('.attachment-card__title').text()).toBe('产品需求说明.pdf')
    expect(wrapper.find('.attachment-card__meta').text()).toContain('PDF')
    expect(wrapper.find('.attachment-card__meta').text()).toContain('2.4 MB')
    expect(wrapper.find('.file-type-label').exists()).toBe(false)
    expect(wrapper.findAll('.file-action-btn')).toHaveLength(2)
  })

  it('renders mini app messages with the same attachment card language', () => {
    const wrapper = mount(MiniAppMessage, {
      props: {
        miniAppData: {
          icon: '',
          name: '项目日报',
          description: '同步项目进展',
        },
      },
    })

    expect(wrapper.find('.attachment-card').exists()).toBe(true)
    expect(wrapper.find('.attachment-card__icon').exists()).toBe(true)
    expect(wrapper.find('.attachment-card__title').text()).toBe('项目日报')
    expect(wrapper.find('.attachment-card__meta').text()).toBe('小程序 · 点击打开')
    expect(wrapper.find('.mini-app-type-label').exists()).toBe(false)
  })

  it('uses the Chinese display name for mini app fallback initials', () => {
    const wrapper = mount(MiniAppMessage, {
      props: {
        miniAppData: {
          icon: '',
          name: 'Default',
          display_name: '审批助手',
          description: '处理审批',
        } as any,
      },
    })

    expect(wrapper.find('.attachment-card__title').text()).toBe('审批助手')
    expect(wrapper.find('.mini-app-icon-fallback').text()).toBe('审')
  })

  it('ignores persisted default mini app avatar icons and falls back to Chinese initials', () => {
    const wrapper = mount(MiniAppMessage, {
      props: {
        miniAppData: {
          icon: generateAvatar('default'),
          name: '审批助手',
          description: '处理审批',
        },
      },
    })

    expect(wrapper.find('img.mini-app-icon').exists()).toBe(false)
    expect(wrapper.find('.mini-app-icon-fallback').text()).toBe('审')
  })

  it('renders news messages with the same attachment card language', () => {
    const wrapper = mount(NewsMessage, {
      props: {
        newsData: {
          title: '产品更新公告',
          summary: '查看本周产品更新重点',
          url: 'https://example.com/news',
        },
      },
    })

    expect(wrapper.find('.attachment-card').exists()).toBe(true)
    expect(wrapper.find('.attachment-card__icon').exists()).toBe(true)
    expect(wrapper.find('.attachment-card__title').text()).toBe('产品更新公告')
    expect(wrapper.find('.attachment-card__meta').text()).toBe('资讯 · 查看详情')
    expect(wrapper.find('.news-info').exists()).toBe(false)
  })

  it('renders share messages with the same attachment card language', () => {
    const wrapper = mount(ShareMessage, {
      props: {
        content: JSON.stringify({ type: 'note', originalContent: '会议纪要内容' }),
        shareData: {
          type: 'note',
          name: '会议纪要',
        },
      },
    })

    expect(wrapper.find('.attachment-card').exists()).toBe(true)
    expect(wrapper.find('.attachment-card__icon').exists()).toBe(true)
    expect(wrapper.find('.attachment-card__title').text()).toBe('会议纪要')
    expect(wrapper.find('.attachment-card__meta').text()).toBe('笔记 · 点击查看')
    expect(wrapper.find('.share-type-label').exists()).toBe(false)
  })
})
