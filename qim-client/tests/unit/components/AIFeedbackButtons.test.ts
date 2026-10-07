import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { ref, nextTick } from 'vue'
import AIFeedbackButtons from '../../../src/components/message/AIFeedbackButtons.vue'
import { aiFeedbackAPI } from '../../../src/api/ai'
import { aiFeedbackRatingsKey, type AIFeedbackRatings } from '../../../src/composables/useAIFeedbackBatch'

vi.mock('../../../src/api/ai', () => ({
  aiFeedbackAPI: {
    set: vi.fn(),
    get: vi.fn(),
    getBatch: vi.fn(),
  },
}))

const setMock = vi.mocked(aiFeedbackAPI.set)
const getMock = vi.mocked(aiFeedbackAPI.get)

describe('AIFeedbackButtons', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setMock.mockResolvedValue()
    getMock.mockResolvedValue(0)
  })

  it('无 messageId 时不渲染反馈按钮', () => {
    const wrapper = mount(AIFeedbackButtons, { props: { messageId: undefined } })

    expect(wrapper.find('.fb-btn').exists()).toBe(false)
  })

  it('流式占位 id（stream_xxx）归一化为 NaN 不渲染', () => {
    const wrapper = mount(AIFeedbackButtons, { props: { messageId: 'stream_abc' } })

    expect(wrapper.find('.fb-btn').exists()).toBe(false)
  })

  it('点赞调用 set(1) 并置 active', async () => {
    const wrapper = mount(AIFeedbackButtons, { props: { messageId: 42 } })
    const up = wrapper.findAll('.fb-btn')[0]

    await up.trigger('click')

    expect(setMock).toHaveBeenCalledWith(42, 1)
    await nextTick()
    expect(up.classes()).toContain('active')
  })

  it('再点同键撤销并调用 set(0)', async () => {
    const wrapper = mount(AIFeedbackButtons, { props: { messageId: 42 } })
    const up = wrapper.findAll('.fb-btn')[0]

    await up.trigger('click')
    await up.trigger('click')

    expect(setMock).toHaveBeenLastCalledWith(42, 0)
    await nextTick()
    expect(up.classes()).not.toContain('active')
  })

  it('接口失败回滚本地选中态', async () => {
    let rejectFn!: (e: Error) => void
    setMock.mockImplementation(() => new Promise((_, reject) => { rejectFn = reject }))
    const wrapper = mount(AIFeedbackButtons, { props: { messageId: 42 } })
    const up = wrapper.findAll('.fb-btn')[0]

    await up.trigger('click')
    expect(up.classes()).toContain('active') // 乐观更新先置位

    rejectFn(new Error('network'))
    await nextTick()
    await nextTick()
    expect(up.classes()).not.toContain('active') // 失败后回滚
  })

  it('注入批量 map 时挂载即恢复选中态，且不单条拉取', async () => {
    const ratings = ref<AIFeedbackRatings>(new Map([[42, -1]]))
    const wrapper = mount(AIFeedbackButtons, {
      props: { messageId: 42 },
      global: { provide: { [aiFeedbackRatingsKey]: ratings } },
    })

    await nextTick()
    expect(wrapper.findAll('.fb-btn')[1].classes()).toContain('active')
    expect(getMock).not.toHaveBeenCalled()
  })

  it('无注入时不单条拉取（选中态恢复仅走批量注入）', async () => {
    const wrapper = mount(AIFeedbackButtons, { props: { messageId: 42 } })

    await nextTick()
    expect(getMock).not.toHaveBeenCalled()
    expect(wrapper.findAll('.fb-btn')[0].classes()).not.toContain('active')
  })

  it('撤销后批量 map 刷新不回退本地态（setFb 回写 map）', async () => {
    const ratings = ref<AIFeedbackRatings>(new Map([[42, 1]]))
    const wrapper = mount(AIFeedbackButtons, {
      props: { messageId: 42 },
      global: { provide: { [aiFeedbackRatingsKey]: ratings } },
    })

    await nextTick()
    const up = wrapper.findAll('.fb-btn')[0]
    expect(up.classes()).toContain('active') // 挂载恢复：已点赞

    await up.trigger('click') // 撤销
    await nextTick()
    expect(up.classes()).not.toContain('active')

    // 新 AI 消息到达触发批量合并：产生新的 map 引用（合并自旧 map）
    ratings.value = new Map([...ratings.value, ...new Map([[43, -1]])])
    await nextTick()
    expect(up.classes()).not.toContain('active') // 撤销不能被回退
  })
})
