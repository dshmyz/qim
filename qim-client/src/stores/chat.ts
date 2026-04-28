import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Message, Conversation, User } from '../types'

export interface MessageReadInfo {
  read_users: User[]
  total_members: number
}

export const useChatStore = defineStore('chat', () => {
  // 状态
  const messages = ref<Map<string, Message[]>>(new Map())
  const conversations = ref<Conversation[]>([])
  const currentConversationId = ref<string | null>(null)
  const drafts = ref<Map<string, string>>(new Map())
  const isLoadingMessages = ref(false)
  const hasMoreMessages = ref<Map<string, boolean>>(new Map())
  const messagePage = ref<Map<string, number>>(new Map())
  const readUsersMap = ref<Map<string, MessageReadInfo>>(new Map())

  // 计算属性
  const currentConversation = computed(() => {
    if (!currentConversationId.value) return null
    return conversations.value.find(c => c.id === currentConversationId.value) || null
  })

  const currentMessages = computed(() => {
    if (!currentConversationId.value) return []
    return messages.value.get(currentConversationId.value) || []
  })

  // 方法
  function setCurrentConversation(id: string | null) {
    currentConversationId.value = id
  }

  function setMessages(conversationId: string, msgs: Message[]) {
    messages.value.set(conversationId, msgs)
  }

  function appendMessage(conversationId: string, message: Message) {
    const msgs = messages.value.get(conversationId) || []
    msgs.push(message)
    messages.value.set(conversationId, [...msgs])
  }

  function prependMessages(conversationId: string, newMessages: Message[]) {
    const existing = messages.value.get(conversationId) || []
    messages.value.set(conversationId, [...newMessages, ...existing])
  }

  function updateMessage(conversationId: string, messageId: string, updates: Partial<Message>) {
    const msgs = messages.value.get(conversationId) || []
    const index = msgs.findIndex(m => m.id === messageId)
    if (index !== -1) {
      msgs[index] = { ...msgs[index], ...updates }
      messages.value.set(conversationId, [...msgs])
    }
  }

  function setConversations(convs: Conversation[]) {
    conversations.value = convs
  }

  function updateConversation(conversation: Conversation) {
    const index = conversations.value.findIndex(c => c.id === conversation.id)
    if (index !== -1) {
      conversations.value[index] = conversation
    }
  }

  function setDraft(conversationId: string, text: string) {
    drafts.value.set(conversationId, text)
  }

  function getDraft(conversationId: string) {
    return drafts.value.get(conversationId) || ''
  }

  function clearDraft(conversationId: string) {
    drafts.value.delete(conversationId)
  }

  function setLoading(loading: boolean) {
    isLoadingMessages.value = loading
  }

  function setHasMore(conversationId: string, hasMore: boolean) {
    hasMoreMessages.value.set(conversationId, hasMore)
  }

  function getHasMore(conversationId: string) {
    return hasMoreMessages.value.get(conversationId) ?? true
  }

  function setPage(conversationId: string, page: number) {
    messagePage.value.set(conversationId, page)
  }

  function getPage(conversationId: string) {
    return messagePage.value.get(conversationId) ?? 1
  }

  function setReadUsersMap(conversationId: string, readInfo: MessageReadInfo) {
    readUsersMap.value.set(conversationId, readInfo)
  }

  function getReadUsersMap() {
    return readUsersMap.value
  }

  return {
    // 状态
    messages,
    conversations,
    currentConversationId,
    drafts,
    isLoadingMessages,
    hasMoreMessages,
    messagePage,
    readUsersMap,
    // 计算属性
    currentConversation,
    currentMessages,
    // 方法
    setCurrentConversation,
    setMessages,
    appendMessage,
    prependMessages,
    updateMessage,
    setConversations,
    updateConversation,
    setDraft,
    getDraft,
    clearDraft,
    setLoading,
    setHasMore,
    getHasMore,
    setPage,
    getPage,
    setReadUsersMap,
    getReadUsersMap,
  }
})
