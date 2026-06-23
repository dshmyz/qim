import { describe, expect, it } from 'vitest'
import {
  decodeToPlainText,
  findActiveMentionToken,
  reconcileMentionSpans,
  replaceMentionToken,
  serializeToContent,
} from '@/utils/mentions'

describe('resolveMentionUserIds', () => {
  it('extracts a valid @ query and the full replacement range', () => {
    expect(findActiveMentionToken('请 @ali 看看', 6)).toEqual({
      start: 2,
      end: 6,
      query: 'ali',
    })
  })

  it('includes the entire token when the cursor is in the middle', () => {
    expect(findActiveMentionToken('@alice ', 3)).toEqual({
      start: 0,
      end: 6,
      query: 'al',
    })
  })

  it('rejects @ in email and URL text', () => {
    expect(findActiveMentionToken('mail a@b.com', 8)).toBeNull()
    expect(findActiveMentionToken('go get host/@v1', 15)).toBeNull()
  })

  it('replaces the complete query token instead of leaving its query behind', () => {
    const token = findActiveMentionToken('请 @ali 看看', 6)

    expect(token).not.toBeNull()
    expect(replaceMentionToken('请 @ali 看看', token!, '@Alice ')).toBe('请 @Alice 看看')
  })

  it('does not treat @v1.0.20 inside Go module URLs as a user mention', () => {
    const content = `go get gitee.com/xxx/xxxx/xxx/@v1.0.20

go: gitee.com/xxx/xxxx/xxx/@v1.0.20: reading https://bigmodel.cn/glm-coding/@v/v1.0.20.info: 502 Bad Gateway`

    expect(serializeToContent(content, [])).toBe(content)
  })

  it('keeps a member selected from the mention panel while ordinary text changes', () => {
    const spans = reconcileMentionSpans(
      [{ start: 0, end: 3, text: '@张三', userId: 42 }],
      '@张三 ',
      '@张三 请看一下'
    )

    expect(serializeToContent('@张三 请看一下', spans)).toBe('@{mention:42|%E5%BC%A0%E4%B8%89} 请看一下')
  })

  it('renders persisted mention tokens as readable text', () => {
    expect(decodeToPlainText('@{mention:42|%E5%BC%A0%E4%B8%89} 请看')).toBe('@张三 请看')
  })
})
