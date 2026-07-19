import { describe, expect, it } from 'vitest'
import { createMergedForwardPayload, parseMergedForwardPayload } from '@/utils/mergedForward'

describe('merged forward payload', () => {
  it('keeps selected messages in list order and rejects invalid payloads', () => {
    const payload = createMergedForwardPayload([
      { id: '2', type: 'image', content: '/a.png', sender: { name: '乙' }, timestamp: 2 },
      { id: '1', type: 'text', content: '你好', sender: { name: '甲' }, timestamp: 1 },
    ] as any)

    expect(payload.messages.map(item => item.id)).toEqual(['2', '1'])
    expect(parseMergedForwardPayload('{invalid')).toBeNull()
  })

  it('rejects payloads with malformed message items', () => {
    expect(parseMergedForwardPayload(JSON.stringify({
      version: 1,
      title: '聊天记录',
      messages: [null],
    }))).toBeNull()
  })

  it('stores a supplied source title in the forwarded record snapshot', () => {
    const payload = createMergedForwardPayload([
      { id: '1', type: 'text', content: '你好', sender: { name: '甲' }, timestamp: 1 },
    ] as any, '来自「产品讨论群」的聊天记录')

    expect(payload.title).toBe('来自「产品讨论群」的聊天记录')
    expect(parseMergedForwardPayload(JSON.stringify(payload))?.title).toBe('来自「产品讨论群」的聊天记录')
  })
})
