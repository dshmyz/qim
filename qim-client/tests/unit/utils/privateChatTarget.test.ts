import { describe, expect, it } from 'vitest'
import { getPrivateChatUserId, isPrivateChatSearchResult, normalizePrivateChatUserId } from '@/utils/privateChatTarget'

describe('private chat search target helpers', () => {
  it('treats account search results as private-chat targets unless they are groups', () => {
    expect(isPrivateChatSearchResult({ type: 'user' })).toBe(true)
    expect(isPrivateChatSearchResult({ type: 'bot' })).toBe(true)
    expect(isPrivateChatSearchResult({ type: 'admin' })).toBe(true)
    expect(isPrivateChatSearchResult({ type: 'api' })).toBe(true)
    expect(isPrivateChatSearchResult({ type: 'system' })).toBe(true)

    expect(isPrivateChatSearchResult({ type: 'group' })).toBe(false)
    expect(isPrivateChatSearchResult({ type: 'discussion' })).toBe(false)
  })

  it('normalizes supported user id shapes without producing NaN', () => {
    expect(normalizePrivateChatUserId(42)).toBe(42)
    expect(normalizePrivateChatUserId('42')).toBe(42)
    expect(normalizePrivateChatUserId('emp42')).toBe(42)

    expect(normalizePrivateChatUserId('abc')).toBeNull()
    expect(normalizePrivateChatUserId('emp')).toBeNull()
    expect(normalizePrivateChatUserId(null)).toBeNull()
  })

  it('extracts private-chat user id from common user object shapes', () => {
    expect(getPrivateChatUserId({ id: '12' })).toBe(12)
    expect(getPrivateChatUserId({ user_id: 13 })).toBe(13)
    expect(getPrivateChatUserId({ userId: '14' })).toBe(14)
    expect(getPrivateChatUserId({ UserID: '15' })).toBe(15)
    expect(getPrivateChatUserId({ id: 'bad', user_id: '16' })).toBe(16)
    expect(getPrivateChatUserId({ id: 'bad' })).toBeNull()
  })
})
