import { describe, it, expect, beforeEach } from 'vitest'
import { ref } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import { useMainWebSocketHandlers } from '@/composables/useMainWebSocketHandlers'
import type { Message } from '@/types'

const createMessage = (overrides: Partial<Message> = {}): Message => ({
  id: '101',
  content: 'hello',
  sender: {
    id: '1',
    name: 'me',
    avatar: ''
  },
  timestamp: Date.now(),
  type: 'text',
  isSelf: true,
  isRead: true,
  conversationId: '7',
  ...overrides
})

describe('useMainWebSocketHandlers', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  describe('handleReadReceipt', () => {
    it('requests read-users refresh when a read receipt arrives for an already-read self message', () => {
      const messages = ref<Message[]>([
        createMessage({ id: '101', isRead: true })
      ])
      const receivedEvents: Array<{ conversationId: string; messageIds: string[]; readerUserId?: string }> = []
      const listener = (event: Event) => {
        receivedEvents.push((event as CustomEvent).detail)
      }
      window.addEventListener('message-read-receipt-updated', listener)

      try {
        const handlers = useMainWebSocketHandlers(ref('7'), messages)

        handlers.handleReadReceipt({
          conversation_id: 7,
          user_id: 3,
          message_ids: [101]
        })

        expect(receivedEvents).toEqual([
          {
            conversationId: '7',
            messageIds: ['101'],
            readerUserId: '3'
          }
        ])
      } finally {
        window.removeEventListener('message-read-receipt-updated', listener)
      }
    })
  })
})
