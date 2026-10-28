import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import AIQualityBadge from '../../../src/components/message/AIQualityBadge.vue'
import { aiFeedbackAPI } from '../../../src/api/ai'

vi.mock('../../../src/api/ai', () => ({
  aiFeedbackAPI: {
    set: vi.fn(),
    get: vi.fn(),
    getBatch: vi.fn(),
  },
}))

const setMock = vi.mocked(aiFeedbackAPI.set)

const doubtful = { status: 'doubtful' as const, reason: '回复包含资料中不存在的具体事实', confidence: 0.8 }

describe('AIQualityBadge', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setMock.mockResolvedValue()
  })

  it('无标注 / 非 doubtful 时不渲染', () => {
    expect(mount(AIQualityBadge, { props: {} }).find('.ai-quality').exists()).toBe(false)
    expect(mount(AIQualityBadge, { props: { quality: { status: 'ok' } } }).find('.ai-quality').exists()).toBe(false)
  })

  it('doubtful 时渲染「可能不准」徽章', () => {
    const wrapper = mount(AIQualityBadge, { props: { quality: doubtful, messageId: 42 } })
    expect(wrapper.find('.quality-chip').text()).toContain('可能不准')
    expect(wrapper.find('.quality-panel').exists()).toBe(false)
  })

  it('点开徽章显示核验原因与反馈按钮', async () => {
    const wrapper = mount(AIQualityBadge, { props: { quality: doubtful, messageId: 42 } })
    await wrapper.find('.quality-chip').trigger('click')
    await nextTick()

    expect(wrapper.find('.quality-panel').exists()).toBe(true)
    expect(wrapper.find('.quality-reason').text()).toContain('回复包含资料中不存在的具体事实')
    expect(wrapper.find('.quality-feedback').text()).toContain('这条确实不准')
  })

  it('点反馈提交 -1 评分并带上核验原因，按钮变为已反馈', async () => {
    const wrapper = mount(AIQualityBadge, { props: { quality: doubtful, messageId: 42 } })
    await wrapper.find('.quality-chip').trigger('click')
    await nextTick()
    await wrapper.find('.quality-feedback').trigger('click')
    await nextTick()

    expect(setMock).toHaveBeenCalledTimes(1)
    const [messageId, rating, extra] = setMock.mock.calls[0]
    expect(messageId).toBe(42)
    expect(rating).toBe(-1)
    expect(extra?.reasonCategory).toBe('other')
    expect(extra?.comment).toContain('质量核验存疑')
    expect(extra?.comment).toContain('回复包含资料中不存在的具体事实')
    expect(wrapper.find('.quality-feedback').text()).toContain('已反馈')
  })

  it('提交失败时保留按钮，允许重试', async () => {
    setMock.mockRejectedValueOnce(new Error('network'))
    const wrapper = mount(AIQualityBadge, { props: { quality: doubtful, messageId: 42 } })
    await wrapper.find('.quality-chip').trigger('click')
    await nextTick()
    await wrapper.find('.quality-feedback').trigger('click')
    await nextTick()

    expect(wrapper.find('.quality-feedback').text()).toContain('这条确实不准')
  })

  it('流式占位 id 无法提交反馈（不发起请求）', async () => {
    const wrapper = mount(AIQualityBadge, { props: { quality: doubtful, messageId: 'stream_abc' } })
    await wrapper.find('.quality-chip').trigger('click')
    await nextTick()
    await wrapper.find('.quality-feedback').trigger('click')
    await nextTick()

    expect(setMock).not.toHaveBeenCalled()
  })

  it('再次点击徽章收起面板', async () => {
    const wrapper = mount(AIQualityBadge, { props: { quality: doubtful, messageId: 42 } })
    await wrapper.find('.quality-chip').trigger('click')
    await nextTick()
    await wrapper.find('.quality-chip').trigger('click')
    await nextTick()

    expect(wrapper.find('.quality-panel').exists()).toBe(false)
  })
})
