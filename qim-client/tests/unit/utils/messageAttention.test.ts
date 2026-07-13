import { describe, expect, it } from 'vitest'
import { shouldRequestMessageAttention } from '../../../src/utils/messageAttention'

describe('shouldRequestMessageAttention', () => {
  it.each([
    { name: 'focused selected conversation', current: true, streaming: false, active: true, expected: false },
    { name: 'unfocused selected conversation', current: true, streaming: false, active: false, expected: true },
    { name: 'focused other conversation', current: false, streaming: false, active: true, expected: true },
    { name: 'unfocused other conversation', current: false, streaming: false, active: false, expected: true },
    { name: 'streaming message', current: false, streaming: true, active: false, expected: false },
  ])('$name => $expected', ({ current, streaming, active, expected }) => {
    expect(shouldRequestMessageAttention({
      isCurrentConversation: current,
      isStreaming: streaming,
      isWindowActive: active,
    })).toBe(expected)
  })
})
