import { describe, expect, it } from 'vitest'
import { normalizeBotHistoryMessages, processBotMessage } from '../../src/composables/useBotChat'

describe('useBotChat message normalization', () => {
  it('recognizes historical assistant messages without sender_type', () => {
    const message = processBotMessage({
      id: 11,
      conversation_id: 7,
      sender_id: 42,
      sender: {
        id: 42,
        nickname: 'Helper',
        type: 'bot_assistant'
      },
      type: 'markdown',
      content: 'answer',
      origin: 'assistant',
      created_at: '2026-06-25T08:00:00Z'
    })

    expect(message.senderType).toBe('bot')
    expect(message.sender?.nickname).toBe('Helper')
    expect(message.content).toBe('answer')
  })

  it('keeps server chronological order for history', () => {
    const messages = normalizeBotHistoryMessages([
      { id: 1, sender_type: 'user', content: 'question', created_at: '2026-06-25T08:00:00Z' },
      { id: 2, origin: 'assistant', content: 'answer', created_at: '2026-06-25T08:00:01Z' }
    ])

    expect(messages.map(message => message.content)).toEqual(['question', 'answer'])
    expect(messages[1].senderType).toBe('bot')
  })
})
