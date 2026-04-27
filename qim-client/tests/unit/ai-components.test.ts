import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { ref, nextTick } from 'vue'
import AIQuickActionItem from '../../src/components/ai/AIQuickActionItem.vue'
import AIQuickActions from '../../src/components/ai/AIQuickActions.vue'
import AIMessageBadge from '../../src/components/ai/AIMessageBadge.vue'

describe('AI组件单元测试', () => {
  describe('AIQuickActionItem', () => {
    it('应正确渲染图标和标签', () => {
      const wrapper = mount(AIQuickActionItem, {
        props: {
          icon: '<svg></svg>',
          label: '翻译',
          tooltip: '翻译文本',
        },
      })

      expect(wrapper.text()).toContain('翻译')
      expect(wrapper.find('.action-label').text()).toBe('翻译')
    })

    it('应触发点击事件', async () => {
      const wrapper = mount(AIQuickActionItem, {
        props: {
          icon: '<svg></svg>',
          label: '翻译',
        },
      })

      await wrapper.trigger('click')
      expect(wrapper.emitted('click')).toHaveLength(1)
    })

    it('应显示tooltip属性', () => {
      const wrapper = mount(AIQuickActionItem, {
        props: {
          icon: '<svg></svg>',
          label: '翻译',
          tooltip: '翻译文本',
        },
      })

      const button = wrapper.find('button')
      expect(button.attributes('title')).toBe('翻译文本')
    })
  })

  describe('AIQuickActions', () => {
    it('应渲染默认的快捷指令按钮', () => {
      const wrapper = mount(AIQuickActions, {
        props: {
          isProcessing: false,
        },
      })

      const actions = wrapper.findAllComponents(AIQuickActionItem)
      expect(actions.length).toBeGreaterThanOrEqual(5) // 默认5个按钮
    })

    it('应支持自定义actions列表', () => {
      const customActions = [
        { id: 'custom', icon: '<svg></svg>', label: '自定义' },
      ]

      const wrapper = mount(AIQuickActions, {
        props: {
          actions: customActions,
          isProcessing: false,
        },
      })

      const actions = wrapper.findAllComponents(AIQuickActionItem)
      expect(actions.length).toBe(1)
      expect(wrapper.text()).toContain('自定义')
    })

    it('处理中时应显示加载状态', () => {
      const wrapper = mount(AIQuickActions, {
        props: {
          isProcessing: true,
        },
      })

      expect(wrapper.find('.ai-processing').exists()).toBe(true)
      expect(wrapper.text()).toContain('处理中')
    })

    it('点击按钮应触发action事件', async () => {
      const wrapper = mount(AIQuickActions, {
        props: {
          isProcessing: false,
        },
      })

      const firstAction = wrapper.findComponent(AIQuickActionItem)
      await firstAction.trigger('click')

      expect(wrapper.emitted('action')).toHaveLength(1)
      expect(wrapper.emitted('action')![0][0]).toBe('summary')
    })
  })

  describe('AIMessageBadge', () => {
    it('应渲染AI标识', () => {
      const wrapper = mount(AIMessageBadge)

      expect(wrapper.find('.ai-message-badge').exists()).toBe(true)
      expect(wrapper.text()).toContain('AI')
    })

    it('应支持自定义助手名称', () => {
      const wrapper = mount(AIMessageBadge, {
        props: {
          assistantName: '小助手',
        },
      })

      expect(wrapper.text()).toContain('小助手')
    })

    it('compact模式应使用紧凑样式', () => {
      const wrapper = mount(AIMessageBadge, {
        props: {
          compact: true,
        },
      })

      expect(wrapper.find('.ai-message-badge').classes()).toContain('compact')
    })

    it('非compact模式应显示"由AI生成"标签', () => {
      const wrapper = mount(AIMessageBadge, {
        props: {
          compact: false,
        },
      })

      expect(wrapper.text()).toContain('由 AI 生成')
    })
  })
})