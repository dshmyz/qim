import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ThinkingIndicator from '../../../src/components/shared/ThinkingIndicator.vue'

// 阶段文案来自 ai_reply_started.status（"正在看图片…"/"正在处理…"）；
// 没有文案时保持原来的纯动画点渲染。
describe('ThinkingIndicator', () => {
  it('无 label 时只渲染动画点', () => {
    const wrapper = mount(ThinkingIndicator)
    expect(wrapper.find('.thinking-label').exists()).toBe(false)
    expect(wrapper.findAll('.dot')).toHaveLength(3)
  })

  it('传 label 时显示阶段文案', () => {
    const wrapper = mount(ThinkingIndicator, { props: { label: '正在看图片…' } })
    expect(wrapper.find('.thinking-label').text()).toBe('正在看图片…')
    expect(wrapper.findAll('.dot')).toHaveLength(3)
  })
})
