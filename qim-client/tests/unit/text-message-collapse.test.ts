import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import TextMessage from '@/components/message/TextMessage.vue'

// 渲染规则 store 只在 test 里 mock：避免装 Pinia，也避免挂载即触发 fetchRules 网络请求
vi.mock('@/stores/renderRules', () => ({
  useRenderRulesStore: () => ({
    loaded: true,
    fetchRules: vi.fn(),
    rulesForConversation: () => [] as unknown[]
  })
}))

// jsdom 不做真实布局，scrollHeight/clientHeight 恒为 0；按用例桩控二者模拟溢出
const stubSize = (scrollHeight: number, clientHeight: number) => {
  Object.defineProperty(HTMLElement.prototype, 'scrollHeight', {
    configurable: true,
    get: () => scrollHeight
  })
  Object.defineProperty(HTMLElement.prototype, 'clientHeight', {
    configurable: true,
    get: () => clientHeight
  })
}

const mountText = async (content: string) => {
  const wrapper = mount(TextMessage, { props: { content, isSelf: false } })
  // onMounted 里测完溢出才渲染按钮，DOM 补丁在下一个 tick
  await nextTick()
  return wrapper
}

describe('TextMessage 长文本折叠', () => {
  it('内容高度超过折叠阈值时渲染「展开全文」按钮，点击可展开/收起', async () => {
    // 溢出：自然高度 500px > 钳制后可视高度 100px
    stubSize(500, 100)
    const wrapper = await mountText('这是一段很长很长的文本内容')

    const btn = wrapper.find('.text-expand-btn')
    expect(btn.exists()).toBe(true)
    expect(btn.text()).toBe('▾ 展开全文')
    expect(wrapper.find('.text-content').classes()).toContain('is-collapsed')

    // 展开：折叠 class 移除、文案切换
    await btn.trigger('click')
    expect(wrapper.find('.text-content').classes()).not.toContain('is-collapsed')
    expect(wrapper.find('.text-expand-btn').text()).toBe('▴ 收起')

    // 收起：恢复折叠
    await wrapper.find('.text-expand-btn').trigger('click')
    expect(wrapper.find('.text-content').classes()).toContain('is-collapsed')
    expect(wrapper.find('.text-expand-btn').text()).toBe('▾ 展开全文')
  })

  it('内容未溢出（含大量换行但总高不超过阈值）时不渲染按钮', async () => {
    stubSize(100, 100)
    const wrapper = await mountText('不长的文本')
    expect(wrapper.find('.text-expand-btn').exists()).toBe(false)
    // 未溢出时折叠 class 存在但 max-height 是 no-op，不影响展示
    expect(wrapper.find('.text-content').classes()).toContain('is-collapsed')
  })
})
