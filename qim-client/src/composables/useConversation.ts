import { ref } from 'vue'
import type { Conversation, Message, User } from '../types'

export function useConversation() {
  const conversations = ref<Conversation[]>([])
  const currentConversationId = ref<string | null>(null)
  const messages = ref<Message[]>([])
  const hasMoreMessages = ref(false)
  const selectedConversation = ref<Conversation | null>(null)

  const selectConversation = (conversation: Conversation) => {
    currentConversationId.value = conversation.id
    selectedConversation.value = conversation
  }

  const updateConversation = (updated: Conversation) => {
    const index = conversations.value.findIndex(c => c.id === updated.id)
    if (index !== -1) {
      conversations.value[index] = updated
    } else {
      conversations.value.unshift(updated)
    }
  }

  const addMessage = (message: Message) => {
    messages.value.push(message)
  }

  const clearMessages = () => {
    messages.value = []
  }

  return {
    conversations,
    currentConversationId,
    messages,
    hasMoreMessages,
    selectedConversation,
    selectConversation,
    updateConversation,
    addMessage,
    clearMessages
  }
}
