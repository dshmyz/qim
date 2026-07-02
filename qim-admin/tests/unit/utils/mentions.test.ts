import { describe, expect, it } from 'vitest'
import { decodeMentionTokens } from '@/utils/mentions'

describe('decodeMentionTokens', () => {
  it('renders user mention tokens as readable at text', () => {
    expect(decodeMentionTokens('@{mention:3|%E6%B5%8B%E8%AF%95%E7%94%A8%E6%88%B7} 下午两点开会')).toBe('@测试用户 下午两点开会')
  })

  it('renders all mention tokens', () => {
    expect(decodeMentionTokens('@{mention:all|%E6%89%80%E6%9C%89%E4%BA%BA} 收到请回复')).toBe('@所有人 收到请回复')
  })

  it('falls back for missing or malformed names', () => {
    expect(decodeMentionTokens('@{mention:5} @{mention:6|%E0%A4%A}')).toBe('@用户5 @%E0%A4%A')
  })
})
