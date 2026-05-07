import { ref, computed, watch, type Ref } from 'vue'
import type { Conversation, Message } from '../types'
import { request } from './useRequest'
import { useChatStore } from '../stores/chat'

/**
 * 会话管理 composable
 * 作为 Pinia Store 的包装层，提供便捷的访问接口
 * 返回可写的 ref，内部自动同步到 Store
 */
export function useConversation() {
  const chatStore = useChatStore()

  // 可写 ref，内部同步到 Store
  const conversations: Ref<Conversation[]> = ref([])
  const currentConversationId: Ref<string | null> = ref(null)
  const messages: Ref<Message[]> = ref([])

  // composable 特有状态
  const hasMoreMessages = ref(false)
  const selectedConversation = ref<Conversation | null>(null)
  const selectedGroup = ref<any>(null)
  const selectedChannel = ref<any>(null)
  const groups = ref<any[]>([])
  const searchQuery = ref('')
  const searchResults = ref<Conversation[]>([])
  const isSearching = ref(false)
  const isLoading = ref(false)

  // Store → ref 同步
  watch(() => chatStore.conversations, (newConvs) => {
    if (JSON.stringify(newConvs) !== JSON.stringify(conversations.value)) {
      conversations.value = [...newConvs]
    }
  }, { deep: true })

  watch(() => chatStore.currentConversationId, (newId) => {
    if (newId !== currentConversationId.value) {
      currentConversationId.value = newId
    }
  })

  watch(() => chatStore.currentMessages, (newMsgs) => {
    if (JSON.stringify(newMsgs) !== JSON.stringify(messages.value)) {
      messages.value = [...newMsgs]
    }
  }, { deep: true })

  // ref → Store 同步（当外部直接修改 ref 时）
  watch(conversations, (newConvs) => {
    if (JSON.stringify(newConvs) !== JSON.stringify(chatStore.conversations)) {
      chatStore.setConversations([...newConvs])
    }
  }, { deep: true })

  watch(currentConversationId, (newId) => {
    if (newId !== chatStore.currentConversationId) {
      chatStore.setCurrentConversation(newId)
    }
  })

  watch(messages, (newMsgs) => {
    if (chatStore.currentConversationId && JSON.stringify(newMsgs) !== JSON.stringify(chatStore.currentMessages)) {
      chatStore.setMessages(chatStore.currentConversationId, [...newMsgs])
    }
  }, { deep: true })

  const currentConversation = computed(() => {
    return conversations.value.find(c => c.id === currentConversationId.value) || null
  })

  const pinnedConversations = computed(() => {
    return conversations.value.filter(c => c.pinned)
  })

  const unpinnedConversations = computed(() => {
    return conversations.value.filter(c => !c.pinned)
  })

  const handleConversationSelect = (conversation: Conversation) => {
    currentConversationId.value = String(conversation.id)
    selectedConversation.value = conversation
  }

  const handleGroupChatSelect = (group: any) => {
    selectedGroup.value = group
    currentConversationId.value = group.id ? String(group.id) : null
  }

  const handleChannelSelect = (channel: any) => {
    selectedChannel.value = channel
  }

  const handlePin = async (conversation: Conversation) => {
    try {
      await request(`/api/v1/conversations/${conversation.id}/pin`, {
        method: 'PUT',
        body: JSON.stringify({ pinned: !conversation.pinned })
      })
      chatStore.pinConversation(conversation.id, !conversation.pinned)
    } catch (error) {
      console.error('置顶会话失败:', error)
    }
  }

  const handleMute = async (conversation: Conversation) => {
    try {
      await request(`/api/v1/conversations/${conversation.id}/mute`, {
        method: 'PUT',
        body: JSON.stringify({ muted: !conversation.muted })
      })
      chatStore.muteConversation(conversation.id, !conversation.muted)
    } catch (error) {
      console.error('静音会话失败:', error)
    }
  }

  const handleRemove = async (conversation: Conversation) => {
    try {
      await request(`/api/v1/conversations/${conversation.id}`, {
        method: 'DELETE'
      })
      chatStore.removeConversation(conversation.id)
    } catch (error) {
      console.error('移除会话失败:', error)
    }
  }

  const handleMarkRead = async (conversation: Conversation) => {
    try {
      await request(`/api/v1/conversations/${conversation.id}/read`, {
        method: 'PUT'
      })
      chatStore.markConversationRead(conversation.id)
    } catch (error) {
      console.error('标记已读失败:', error)
    }
  }

  const updateConversation = (updated: Conversation) => {
    chatStore.patchConversation(updated.id, updated)
  }

  const updateConversations = (newConversations: Conversation[]) => {
    chatStore.setConversations(newConversations)
  }

  const addMessage = (message: Message) => {
    if (currentConversationId.value) {
      chatStore.addMessage(currentConversationId.value, message)
    }
  }

  const clearMessages = () => {
    if (currentConversationId.value) {
      chatStore.clearMessages(currentConversationId.value)
    }
    hasMoreMessages.value = false
  }

  const addConversation = (conversation: Conversation) => {
    chatStore.addConversation(conversation)
  }

  const handleExitGroup = async (groupId: string) => {
    try {
      await request(`/api/v1/conversations/${groupId}/leave`, {
        method: 'POST'
      })
      chatStore.removeConversation(groupId)
    } catch (error) {
      console.error('退出群组失败:', error)
    }
  }

  const loadGroups = async () => {
    try {
      const response: any = await request('/api/v1/groups')
      if (response.code === 0) {
        groups.value = response.data || []
      }
    } catch (error) {
      console.error('加载群组失败:', error)
    }
  }

  const loadConversations = async () => {
    isLoading.value = true
    try {
      const response: any = await request('/api/v1/conversations')
      if (response.code === 0 && response.data) {
        chatStore.setConversations(response.data)
      } else {
        chatStore.setConversations([])
      }
    } catch (error) {
      console.error('加载会话失败:', error)
      chatStore.setConversations([])
    } finally {
      isLoading.value = false
    }
  }

  const searchConversations = async (query: string) => {
    searchQuery.value = query

    if (!query.trim()) {
      searchResults.value = []
      isSearching.value = false
      return
    }

    isSearching.value = true
    try {
      const response: any = await request(`/api/v1/conversations/search?query=${encodeURIComponent(query)}`)
      if (response.code === 0) {
        searchResults.value = response.data || []
      }
    } catch (error) {
      console.error('搜索会话失败:', error)
      searchResults.value = []
    } finally {
      isSearching.value = false
    }
  }

  const setCurrentConversationId = (id: string | number | null) => {
    currentConversationId.value = id !== null ? String(id) : null
  }

  const resetState = () => {
    currentConversationId.value = null
    selectedConversation.value = null
    selectedGroup.value = null
    selectedChannel.value = null
    searchQuery.value = ''
    searchResults.value = []
  }

  return {
    // 状态
    conversations,
    currentConversationId: currentConversationId,
    messages,
    hasMoreMessages,
    selectedConversation: selectedConversation,
    selectedGroup,
    selectedChannel,
    groups: groups,
    currentConversation,
    pinnedConversations,
    unpinnedConversations,
    searchQuery: searchQuery,
    searchResults: searchResults,
    isSearching: isSearching,
    isLoading: isLoading,

    // 操作方法
    handleConversationSelect,
    handleGroupChatSelect,
    handleChannelSelect,
    handlePin,
    handleMute,
    handleRemove,
    handleMarkRead,
    updateConversation,
    updateConversations,
    addMessage,
    clearMessages,
    addConversation,
    handleExitGroup,
    loadGroups,
    loadConversations,
    searchConversations,
    setCurrentConversationId,
    resetState
  }
}
