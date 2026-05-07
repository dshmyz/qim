import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Message, Conversation, User } from '../types'

export interface MessageReadInfo {
  read_users: User[]
  total_members: number
}

const STORAGE_KEY = 'qim_conversations'

function saveToStorage(convs: Conversation[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(convs))
  } catch (error) {
    console.warn('保存会话失败:', error)
  }
}

function loadFromStorage(): Conversation[] {
  try {
    const data = localStorage.getItem(STORAGE_KEY)
    return data ? JSON.parse(data) : []
  } catch (error) {
    console.warn('加载会话失败:', error)
    return []
  }
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

  const sortedConversations = computed(() => {
    return [...conversations.value].sort((a, b) => {
      if (a.pinned && !b.pinned) return -1
      if (!a.pinned && b.pinned) return 1
      return (b.timestamp || 0) - (a.timestamp || 0)
    })
  })

  // 基础方法
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
      conversations.value = [...conversations.value]
    }
  }

  function patchConversation(id: string, updates: Partial<Conversation>) {
    const index = conversations.value.findIndex(c => c.id === id)
    if (index !== -1) {
      conversations.value[index] = {
        ...conversations.value[index],
        ...updates
      }
      conversations.value = [...conversations.value]
      saveToStorage(conversations.value)
    }
  }

  function addConversation(conversation: Conversation) {
    const exists = conversations.value.some(c => c.id === conversation.id)
    if (!exists) {
      conversations.value = [conversation, ...conversations.value]
      saveToStorage(conversations.value)
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

  // 业务逻辑方法
  function pinConversation(id: string, pinned: boolean) {
    const index = conversations.value.findIndex(c => c.id === id)
    if (index !== -1) {
      conversations.value[index] = {
        ...conversations.value[index],
        pinned,
        pinnedAt: pinned ? Date.now() : undefined
      }
      conversations.value = [...conversations.value]
      saveToStorage(conversations.value)
    }
  }

  function muteConversation(id: string, muted: boolean) {
    const index = conversations.value.findIndex(c => c.id === id)
    if (index !== -1) {
      conversations.value[index] = {
        ...conversations.value[index],
        muted
      }
      conversations.value = [...conversations.value]
      saveToStorage(conversations.value)
    }
  }

  function removeConversation(id: string) {
    const index = conversations.value.findIndex(c => c.id === id)
    if (index !== -1) {
      conversations.value.splice(index, 1)
      conversations.value = [...conversations.value]
      messages.value.delete(id)
      drafts.value.delete(id)
      saveToStorage(conversations.value)
    }
  }

  function recallMessage(conversationId: string, messageId: string) {
    updateMessage(conversationId, messageId, {
      content: '[消息已撤回]',
      isRecalled: true
    })

    const convIndex = conversations.value.findIndex(c => c.id === conversationId)
    if (convIndex !== -1) {
      const conv = conversations.value[convIndex]
      if (conv.lastMessage && conv.lastMessage.id === messageId) {
        conversations.value[convIndex] = {
          ...conv,
          lastMessage: {
            ...conv.lastMessage,
            content: '[消息已撤回]',
            isRecalled: true
          }
        }
        conversations.value = [...conversations.value]
        saveToStorage(conversations.value)
      }
    }
  }

  function receiveMessage(conversationId: string, message: Message, isCurrentConversation: boolean) {
    appendMessage(conversationId, message)

    const convIndex = conversations.value.findIndex(c => c.id === conversationId)
    if (convIndex !== -1) {
      const conv = conversations.value[convIndex]
      const updatedConv = {
        ...conv,
        lastMessage: message,
        timestamp: message.timestamp || Date.now()
      }

      if (!isCurrentConversation && !message.isStreaming) {
        updatedConv.unreadCount = (updatedConv.unreadCount || 0) + 1
      }

      conversations.value[convIndex] = updatedConv
      conversations.value = [...conversations.value]
      saveToStorage(conversations.value)
    }
  }

  function markConversationRead(id: string) {
    const index = conversations.value.findIndex(c => c.id === id)
    if (index !== -1) {
      conversations.value[index] = {
        ...conversations.value[index],
        unreadCount: 0
      }
      conversations.value = [...conversations.value]
      saveToStorage(conversations.value)
    }
  }

  function addGroupMember(conversationId: string, member: any) {
    const index = conversations.value.findIndex(c => c.id === conversationId)
    if (index !== -1) {
      const conv = conversations.value[index]
      const members = conv.members || []
      const exists = members.some(m => m.id === member.id)
      if (!exists) {
        conversations.value[index] = {
          ...conv,
          members: [...members, member]
        }
        conversations.value = [...conversations.value]
        saveToStorage(conversations.value)
      }
    }
  }

  function removeGroupMember(conversationId: string, userId: string) {
    const index = conversations.value.findIndex(c => c.id === conversationId)
    if (index !== -1) {
      const conv = conversations.value[index]
      if (conv.members) {
        conversations.value[index] = {
          ...conv,
          members: conv.members.filter(m => m.id !== userId)
        }
        conversations.value = [...conversations.value]
        saveToStorage(conversations.value)
      }
    }
  }

  function updateMemberRole(conversationId: string, userId: string, role: string) {
    const index = conversations.value.findIndex(c => c.id === conversationId)
    if (index !== -1) {
      const conv = conversations.value[index]
      if (conv.members) {
        conversations.value[index] = {
          ...conv,
          members: conv.members.map(m => m.id === userId ? { ...m, role } : m)
        }
        conversations.value = [...conversations.value]
        saveToStorage(conversations.value)
      }
    }
  }

  function loadConversationsFromStorage() {
    const stored = loadFromStorage()
    if (stored.length > 0) {
      conversations.value = stored
    }
  }

  function clearAllMessages(conversationId: string) {
    messages.value.delete(conversationId)
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
    sortedConversations,
    // 基础方法
    setCurrentConversation,
    setMessages,
    appendMessage,
    prependMessages,
    updateMessage,
    setConversations,
    updateConversation,
    patchConversation,
    addConversation,
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
    // 业务逻辑方法
    pinConversation,
    muteConversation,
    removeConversation,
    recallMessage,
    receiveMessage,
    markConversationRead,
    addGroupMember,
    removeGroupMember,
    updateMemberRole,
    loadConversationsFromStorage,
    clearAllMessages,
  }
})
