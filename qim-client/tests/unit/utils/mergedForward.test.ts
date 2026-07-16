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
})
