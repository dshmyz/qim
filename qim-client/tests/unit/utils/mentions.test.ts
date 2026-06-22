import { describe, expect, it } from 'vitest'
import { displayMentionTokens, reconcileMentionSpans, serializeMentionTokens } from '@/utils/mentions'

describe('resolveMentionUserIds', () => {
  it('does not treat @v1.0.20 inside Go module URLs as a user mention', () => {
    const content = `go get gitee.com/xxx/xxxx/xxx/@v1.0.20

go: gitee.com/xxx/xxxx/xxx/@v1.0.20: reading https://bigmodel.cn/glm-coding/@v/v1.0.20.info: 502 Bad Gateway`

    expect(serializeMentionTokens(content, [])).toBe(content)
  })

  it('keeps a member selected from the mention panel while ordinary text changes', () => {
    const spans = reconcileMentionSpans(
      [{ start: 0, end: 3, text: '@张三', userIds: [42] }],
      '@张三 ',
      '@张三 请看一下'
    )

    expect(serializeMentionTokens('@张三 请看一下', spans)).toBe('@{mention:42|%E5%BC%A0%E4%B8%89} 请看一下')
  })

  it('renders persisted mention tokens as readable text', () => {
    expect(displayMentionTokens('@{mention:42|%E5%BC%A0%E4%B8%89} 请看')).toBe('@张三 请看')
  })
})
