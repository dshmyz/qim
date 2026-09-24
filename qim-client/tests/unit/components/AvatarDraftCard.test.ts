import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import AvatarDraftCard from '../../../src/components/avatar/AvatarDraftCard.vue'
import type { AvatarReplyDraft } from '../../../src/types/avatar'
import { avatarAPI } from '../../../src/api/avatar'
import { useAvatarDraftApproval } from '../../../src/composables/useAvatarDraftApproval'

vi.mock('../../../src/api/avatar', () => ({
  avatarAPI: {
    listReplyDrafts: vi.fn(),
    approveReplyDraft: vi.fn(),
    discardReplyDraft: vi.fn(),
  },
}))

const approveMock = vi.mocked(avatarAPI.approveReplyDraft)
const discardMock = vi.mocked(avatarAPI.discardReplyDraft)

const draft: AvatarReplyDraft = {
  id: 7,
  conversation_id: 10,
  trigger_message_id: 100,
  content: '好的，明天见',
  reason: '包含未证实事实',
  sources: [],
  expires_at: '2026-09-24T00:00:00Z',
  created_at: '2026-09-23T00:00:00Z',
}

describe('AvatarDraftCard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    approveMock.mockResolvedValue()
    discardMock.mockResolvedValue()
  })

  it('渲染草稿正文与核验原因', () => {
    const wrapper = mount(AvatarDraftCard, { props: { draft } })

    expect(wrapper.find('.avatar-draft-card__preview').text()).toContain('好的，明天见')
    expect(wrapper.find('.avatar-draft-card__meta').text()).toContain('包含未证实事实')
  })

  it('点击「发送」调用批准接口', async () => {
    const wrapper = mount(AvatarDraftCard, { props: { draft } })

    await wrapper.findAll('.draft-action')[0].trigger('click')
    await nextTick()

    expect(approveMock).toHaveBeenCalledWith(7)
  })

  it('点击「丢弃」调用丢弃接口', async () => {
    const wrapper = mount(AvatarDraftCard, { props: { draft } })

    await wrapper.findAll('.draft-action')[2].trigger('click')
    await nextTick()

    expect(discardMock).toHaveBeenCalledWith(7)
    expect(approveMock).not.toHaveBeenCalled()
  })

  it('点击「编辑后发送」发起编辑请求，不调接口', async () => {
    const { editRequest } = useAvatarDraftApproval()
    expect(editRequest.value).toBeNull()

    const wrapper = mount(AvatarDraftCard, { props: { draft } })
    await wrapper.findAll('.draft-action')[1].trigger('click')

    expect(editRequest.value).not.toBeNull()
    expect(editRequest.value?.id).toBe(7)
    expect(approveMock).not.toHaveBeenCalled()
    expect(discardMock).not.toHaveBeenCalled()

    editRequest.value = null
  })

  it('批准接口失败时恢复按钮可点', async () => {
    approveMock.mockRejectedValue(new Error('network'))
    const wrapper = mount(AvatarDraftCard, { props: { draft } })
    const btn = wrapper.findAll('.draft-action')[0]

    await btn.trigger('click')
    await nextTick()
    await nextTick()

    expect(btn.attributes('disabled')).toBeUndefined()
  })
})
