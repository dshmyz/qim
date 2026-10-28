import { describe, it, expect } from 'vitest'
import { buildMessageUpdate } from '../../../src/utils/messageUpdate'

// 核验存疑标注是事后增量推送（只带 id + quality），这条契约一旦破坏，
// 客户端会把它当成"无内容事件"丢掉 → 徽章永远不出现。
describe('buildMessageUpdate', () => {
  it('只带正文的流式片段 → 合入 content', () => {
    expect(buildMessageUpdate({ content: '你好' })).toEqual({ content: '你好' })
  })

  it('流式帧携带 type / is_streaming → 一并合入', () => {
    const got = buildMessageUpdate({ content: '你好', type: 'markdown', is_streaming: false })
    expect(got).toEqual({ content: '你好', type: 'markdown', isStreaming: false })
  })

  it('只带 quality 的增量更新 → 不能丢弃，且不得写入 content', () => {
    const quality = { status: 'doubtful', reason: '包含不存在的事实' }
    const got = buildMessageUpdate({ id: 7, conversation_id: 98, quality })
    expect(got).toEqual({ quality })
    expect(got).not.toHaveProperty('content')
  })

  it('正文与 quality 同时到达 → 字段都合入', () => {
    const quality = { status: 'doubtful' }
    const got = buildMessageUpdate({ content: 'x', quality })
    expect(got).toEqual({ content: 'x', quality })
  })

  it('既无正文也无 quality → 返回 null（无需处理）', () => {
    expect(buildMessageUpdate({ id: 7, type: 'markdown' })).toBeNull()
    expect(buildMessageUpdate({})).toBeNull()
    expect(buildMessageUpdate(null)).toBeNull()
  })

  it('空字符串正文不算正文变更', () => {
    expect(buildMessageUpdate({ content: '', type: 'markdown' })).toBeNull()
  })
})
