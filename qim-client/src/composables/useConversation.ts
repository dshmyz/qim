import { ref, computed, type Ref } from 'vue'
import type { Conversation, Message } from '../types'
import { request } from './useRequest'
import { useChatStore } from '../stores/chat'

/**
 * 会话管理 composable
 * 使用单向数据流：直接从 Store 读取状态，所有修改都通过 Store methods
 */
export function useConversation() {
  const chatStore = useChatStore()

  // 特有状态(非 Store 管理的 UI 状态)
  const hasMoreMessages = ref(false)
  const selectedConversation = ref<Conversation | null>(null)
  const selectedGroup = ref<any>(null)
  const selectedChannel = ref<any>(null)
  const groups = ref<any[]>([])
  const searchQuery = ref('')
  const searchResults = ref<Conversation[]>([])
  const isSearching = ref(false)
  const isLoading = ref(false)

  // 单向数据流：直接从 Store 读取状态（只读 computed）
  const conversations = computed(() => chatStore.conversations)
  const currentConversationId = computed(() => chatStore.currentConversationId)
  const messages = computed(() => chatStore.currentMessages)

  const currentConversation = computed(() => {
    return conversations.value.find(c => c.id === currentConversationId.value) || null
  })

  const pinnedConversations = computed(() => {
    return conversations.value.filter(c => c.is_pinned)
  })

  const unpinnedConversations = computed(() => {
    return conversations.value.filter(c => !c.is_pinned)
  })

  // 会话操作（通过 Store methods 修改状态）
  const handleConversationSelect = (conversation: Conversation) => {
    chatStore.setCurrentConversation(String(conversation.id))
    selectedConversation.value = conversation
  }

  const handleGroupChatSelect = (group: any) => {
    selectedGroup.value = group
    chatStore.setCurrentConversation(group.id ? String(group.id) : null)
  }

  const handleChannelSelect = (channel: any) => {
    selectedChannel.value = channel
  }

  const handlePin = async (conversation: Conversation) => {
    try {
      await request(`/api/v1/conversations/${conversation.id}/pin`, {
        method: 'PUT',
        body: JSON.stringify({ is_pinned: !conversation.is_pinned })
      })
      chatStore.pinConversation(conversation.id, !conversation.is_pinned)
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

  // 消息操作（通过 Store methods）
  const updateConversation = (updated: Conversation) => {
    chatStore.patchConversation(updated.id, updated)
  }

  const updateConversations = (newConversations: Conversation[]) => {
    chatStore.setConversations(newConversations)
  }

  const addMessage = (message: Message) => {
    if (currentConversationId.value) {
      chatStore.appendMessage(currentConversationId.value, message)
    }
  }

  const clearMessages = () => {
    if (currentConversationId.value) {
      chatStore.clearAllMessages(currentConversationId.value)
    }
    hasMoreMessages.value = false
  }

  const addConversation = (conversation: Conversation) => {
    chatStore.addConversation(conversation)
  }

  const handleExitGroup = async (groupOrId: any) => {
    try {
      const groupId = typeof groupOrId === 'string' ? groupOrId : groupOrId?.id
      if (!groupId) return
      
      await request(`/api/v1/groups/${groupId}/exit`, {
        method: 'POST'
      })
      chatStore.patchConversation(String(groupId), { isExited: true } as any)
    } catch (error) {
      console.error('退出群组失败:', error)
    }
  }

  // 数据加载
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
        chatStore.setConversations(response.data.list || [])
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
    chatStore.setCurrentConversation(id !== null ? String(id) : null)
  }

  const resetState = () => {
    chatStore.setCurrentConversation(null)
    selectedConversation.value = null
    selectedGroup.value = null
    selectedChannel.value = null
    searchQuery.value = ''
    searchResults.value = []
  }

  return {
    // 状态（只读 computed）
    conversations,
    currentConversationId,
    messages,
    hasMoreMessages,
    selectedConversation,
    selectedGroup,
    selectedChannel,
    groups,
    currentConversation,
    pinnedConversations,
    unpinnedConversations,
    searchQuery,
    searchResults,
    isSearching,
    isLoading,

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
    resetState,

    // 直接暴露 Store 实例(用于需要直接操作 Store 的场景)
    store: chatStore
  }
}
