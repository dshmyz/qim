import { ref, computed, readonly } from 'vue'
import type { Conversation, Message } from '../types'
import { request } from './useRequest'

/**
 * 会话管理 composable
 * 管理会话列表、群组、频道选择、会话操作等
 */
export function useConversation() {
  // 会话列表
  const conversations = ref<Conversation[]>([])

  // 当前会话 ID
  const currentConversationId = ref<string | null>(null)

  // 消息列表
  const messages = ref<Message[]>([])

  // 是否有更多消息
  const hasMoreMessages = ref(false)

  // 选中的会话
  const selectedConversation = ref<Conversation | null>(null)

  // 选中的群组
  const selectedGroup = ref<any>(null)

  // 选中的频道
  const selectedChannel = ref<any>(null)

  // 群聊列表
  const groups = ref<any[]>([])

  // 搜索相关
  const searchQuery = ref('')
  const searchResults = ref<Conversation[]>([])
  const isSearching = ref(false)

  // 加载状态
  const isLoading = ref(false)

  /**
   * 当前会话计算属性
   */
  const currentConversation = computed(() => {
    return conversations.value.find(c => c.id === currentConversationId.value) || null
  })

  /**
   * 置顶会话列表
   */
  const pinnedConversations = computed(() => {
    return conversations.value.filter(c => c.pinned)
  })

  /**
   * 未置顶会话列表
   */
  const unpinnedConversations = computed(() => {
    return conversations.value.filter(c => !c.pinned)
  })

  /**
   * 处理会话选择
   */
  const handleConversationSelect = (conversation: Conversation) => {
    currentConversationId.value = conversation.id
    selectedConversation.value = conversation
  }

  /**
   * 处理群聊选择
   */
  const handleGroupChatSelect = (group: any) => {
    selectedGroup.value = group
    currentConversationId.value = group.id
  }

  /**
   * 处理频道选择
   */
  const handleChannelSelect = (channel: any) => {
    selectedChannel.value = channel
  }

  /**
   * 置顶/取消置顶会话
   */
  const handlePin = async (conversation: Conversation) => {
    const index = conversations.value.findIndex(c => c.id === conversation.id)
    if (index === -1) return

    try {
      await request(`/api/v1/conversations/${conversation.id}/pin`, {
        method: 'PUT',
        body: JSON.stringify({ pinned: !conversation.pinned })
      })
      conversations.value[index] = { ...conversations.value[index], pinned: !conversations.value[index].pinned }
    } catch (error) {
      console.error('置顶会话失败:', error)
    }
  }

  /**
   * 静音/取消静音会话
   */
  const handleMute = async (conversation: Conversation) => {
    const index = conversations.value.findIndex(c => c.id === conversation.id)
    if (index === -1) return

    try {
      await request(`/api/v1/conversations/${conversation.id}/mute`, {
        method: 'PUT',
        body: JSON.stringify({ muted: !conversation.muted })
      })
      conversations.value[index] = { ...conversations.value[index], muted: !conversations.value[index].muted }
    } catch (error) {
      console.error('静音会话失败:', error)
    }
  }

  /**
   * 移除会话
   */
  const handleRemove = async (conversation: Conversation) => {
    try {
      await request(`/api/v1/conversations/${conversation.id}`, {
        method: 'DELETE'
      })
      const index = conversations.value.findIndex(c => c.id === conversation.id)
      if (index !== -1) {
        conversations.value.splice(index, 1)
      }
    } catch (error) {
      console.error('移除会话失败:', error)
    }
  }

  /**
   * 标记已读
   */
  const handleMarkRead = async (conversation: Conversation) => {
    try {
      await request(`/api/v1/conversations/${conversation.id}/read`, {
        method: 'PUT'
      })
      conversation.unreadCount = 0
    } catch (error) {
      console.error('标记已读失败:', error)
    }
  }

  /**
   * 更新会话
   */
  const updateConversation = (updated: Conversation) => {
    const index = conversations.value.findIndex(c => c.id === updated.id)
    if (index !== -1) {
      conversations.value[index] = { ...conversations.value[index], ...updated }
    } else {
      conversations.value.unshift(updated)
    }
  }

  /**
   * 批量更新会话列表
   */
  const updateConversations = (newConversations: Conversation[]) => {
    conversations.value = newConversations
  }

  /**
   * 添加消息
   */
  const addMessage = (message: Message) => {
    messages.value.push(message)
  }

  /**
   * 清空消息
   */
  const clearMessages = () => {
    messages.value = []
    hasMoreMessages.value = false
  }

  /**
   * 添加会话
   */
  const addConversation = (conversation: Conversation) => {
    conversations.value.unshift(conversation)
  }

  /**
   * 退出群组
   */
  const handleExitGroup = async (groupId: string) => {
    try {
      await request(`/api/v1/conversations/${groupId}/leave`, {
        method: 'POST'
      })
      const index = conversations.value.findIndex(c => c.id === groupId)
      if (index !== -1) {
        conversations.value.splice(index, 1)
      }
    } catch (error) {
      console.error('退出群组失败:', error)
    }
  }

  /**
   * 加载群组列表
   */
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

  /**
   * 加载会话列表
   */
  const loadConversations = async () => {
    isLoading.value = true
    try {
      const response: any = await request('/api/v1/conversations')
      if (response.code === 0 && response.data) {
        conversations.value = response.data
      } else {
        conversations.value = []
      }
    } catch (error) {
      console.error('加载会话失败:', error)
      conversations.value = []
    } finally {
      isLoading.value = false
    }
  }

  /**
   * 搜索会话
   */
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

  /**
   * 设置当前会话 ID
   */
  const setCurrentConversationId = (id: string | null) => {
    currentConversationId.value = id
  }

  /**
   * 重置所有状态
   */
  const resetState = () => {
    currentConversationId.value = null
    selectedConversation.value = null
    selectedGroup.value = null
    selectedChannel.value = null
    messages.value = []
    searchQuery.value = ''
    searchResults.value = []
  }

  return {
    // 状态
    conversations,  // 允许外部直接修改（WebSocket 消息处理等场景）
    currentConversationId: readonly(currentConversationId),
    messages,  // 不包 readonly，允许外部直接修改（用于消息加载）
    hasMoreMessages,  // 不包 readonly，允许外部直接修改
    selectedConversation: readonly(selectedConversation),
    selectedGroup,  // 允许外部直接设置（群聊选择等场景）
    selectedChannel,  // 允许外部直接设置（频道选择等场景）
    groups: readonly(groups),
    currentConversation,
    pinnedConversations,
    unpinnedConversations,
    searchQuery: readonly(searchQuery),
    searchResults: readonly(searchResults),
    isSearching: readonly(isSearching),
    isLoading: readonly(isLoading),

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
