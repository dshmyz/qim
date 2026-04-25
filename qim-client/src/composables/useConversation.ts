import { ref, computed } from 'vue'
import type { Conversation, Message, User } from '../types'

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

  /**
   * 当前会话计算属性
   */
  const currentConversation = computed(() => {
    return conversations.value.find(c => c.id === currentConversationId.value) || null
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
  const handlePin = (conversation: Conversation) => {
    conversation.pinned = !conversation.pinned
  }

  /**
   * 静音/取消静音会话
   */
  const handleMute = (conversation: Conversation) => {
    conversation.muted = !conversation.muted
  }

  /**
   * 移除会话
   */
  const handleRemove = (conversation: Conversation) => {
    const index = conversations.value.findIndex(c => c.id === conversation.id)
    if (index !== -1) {
      conversations.value.splice(index, 1)
    }
  }

  /**
   * 标记已读
   */
  const handleMarkRead = (conversation: Conversation) => {
    conversation.unreadCount = 0
  }

  /**
   * 更新会话
   */
  const updateConversation = (updated: Conversation) => {
    const index = conversations.value.findIndex(c => c.id === updated.id)
    if (index !== -1) {
      conversations.value[index] = updated
    } else {
      conversations.value.unshift(updated)
    }
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
  const handleExitGroup = (groupId: string) => {
    const index = conversations.value.findIndex(c => c.id === groupId)
    if (index !== -1) {
      conversations.value.splice(index, 1)
    }
  }

  /**
   * 加载群组列表
   */
  const loadGroups = async () => {
    try {
      // 加载群组逻辑
      groups.value = []
    } catch (error) {
      console.error('加载群组失败:', error)
    }
  }

  /**
   * 重置状态
   */
  const resetState = () => {
    currentConversationId.value = null
    selectedConversation.value = null
    selectedGroup.value = null
    selectedChannel.value = null
    messages.value = []
  }

  return {
    // 状态
    conversations,
    currentConversationId,
    messages,
    hasMoreMessages,
    selectedConversation,
    selectedGroup,
    selectedChannel,
    groups,
    currentConversation,
    
    // 操作方法
    handleConversationSelect,
    handleGroupChatSelect,
    handleChannelSelect,
    handlePin,
    handleMute,
    handleRemove,
    handleMarkRead,
    updateConversation,
    addMessage,
    clearMessages,
    addConversation,
    handleExitGroup,
    loadGroups,
    resetState
  }
}
