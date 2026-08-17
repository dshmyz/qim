import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Message, Conversation, User } from '../types'
import { sameConversationId } from '../utils/conversationId'

export interface MessageReadInfo {
  read_users: User[]
  total_members: number
}

// ai_reply_started 事件携带的回复者信息（群 AI 助手/默认 AI/bot 虚拟用户），
// 供「思考中」占位渲染成带头像的正常消息行。
export interface ThinkingSender {
  id: string | number
  nickname?: string
  name?: string
  avatar?: string
}

const STORAGE_KEY = 'qim_conversations'
let saveTimer: ReturnType<typeof setTimeout> | null = null
let lastSaveTime = 0
const SAVE_THROTTLE = 500 // ms

// AI 回复开始（ai_reply_started）到回复消息落库之间的「思考中」占位安全超时：
// 前端靠非本人 new_message（流式首帧/完整回复/系统提示）清除占位，此兜底防止
// 事件丢失导致某会话永久卡在思考态。取 180s 对齐后端 aiReplyTimeout 的最大生成
// 预算（带图 180s / 文本 60s）——若取 90s，慢回复（图片+多步 ReAct 或外部 agent
// webhook）在首帧到达前占位会先熄灭，造成「占位没了回复还没来」的空窗。
const AI_THINKING_TIMEOUT = 180_000 // ms
const aiThinkingTimers = new Map<string, ReturnType<typeof setTimeout>>()

function mergeByConversationId(existing: Conversation[], incoming: Conversation[]) {
  const merged = new Map<string, Conversation>()
  for (const conversation of [...existing, ...incoming]) {
    merged.set(String(conversation.id), conversation)
  }
  return [...merged.values()]
}

function saveToStorage(convs: Conversation[]) {
  const now = Date.now()
  // 如果距上次写入不足 throttle 时间，延迟写入
  if (now - lastSaveTime < SAVE_THROTTLE) {
    if (saveTimer) clearTimeout(saveTimer)
    saveTimer = setTimeout(() => {
      lastSaveTime = Date.now()
      try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(convs))
      } catch (error) {
        console.warn('保存会话失败:', error)
      }
    }, SAVE_THROTTLE)
    return
  }
  lastSaveTime = now
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
  const aiThinking = ref<Map<string, boolean>>(new Map())
  const aiThinkingSenders = ref<Map<string, ThinkingSender>>(new Map())

  // 计算属性
  const currentConversation = computed(() => {
    if (!currentConversationId.value) return null
    return conversations.value.find(c => sameConversationId(c.id, currentConversationId.value)) || null
  })

  const currentMessages = computed(() => {
    if (!currentConversationId.value) return []
    return messages.value.get(currentConversationId.value) || []
  })

  const sortedConversations = computed(() => {
    return [...conversations.value].sort((a, b) => {
      if (a.is_pinned && !b.is_pinned) return -1
      if (!a.is_pinned && b.is_pinned) return 1
      return (b.timestamp || 0) - (a.timestamp || 0)
    })
  })

  // 会话未读消息总数（排除已静音的会话）
  const totalUnreadCount = computed(() => {
    return conversations.value.reduce((sum, c) => {
      if (c.muted) return sum
      return sum + (c.unread_count || 0)
    }, 0)
  })

  // 基础方法
  function setCurrentConversation(id: string | null) {
    // 会话切换（含离开到 null）：清除旧会话的「思考中」占位标记，满足需求
    // 「切换/离开会话时占位即消失」。同会话重复设置（仅切视图）不清除，
    // 避免回复仍在途中时占位被误灭。bot 会话不走本方法，由 useBotChat 自管。
    const prev = currentConversationId.value
    if (prev !== null && prev !== id) {
      setAiThinking(prev, false)
    }
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

  // 合并消息到现有列表（去重，按时间排序）
  function mergeMessages(conversationId: string, newMessages: Message[]) {
    const existing = messages.value.get(conversationId) || []
    const messageMap = new Map<string, Message>()

    // 先加入现有消息
    for (const msg of existing) {
      messageMap.set(String(msg.id), msg)
    }

    // 合并新消息（去重）
    for (const msg of newMessages) {
      const msgId = String(msg.id)
      if (!messageMap.has(msgId)) {
        messageMap.set(msgId, msg)
      }
    }

    // 按时间戳排序（从旧到新）
    const merged = Array.from(messageMap.values()).sort((a, b) => {
      return (a.timestamp || 0) - (b.timestamp || 0)
    })

    messages.value.set(conversationId, merged)
  }

  function updateMessage(conversationId: string, messageId: string, updates: Partial<Message>) {
    const msgs = messages.value.get(conversationId) || []
    const index = msgs.findIndex(m => m.id === messageId)
    if (index !== -1) {
      const newMsgs = [...msgs]
      newMsgs[index] = { ...msgs[index], ...updates }
      messages.value.set(conversationId, newMsgs)
    }
  }

  function setConversations(convs: Conversation[]) {
    conversations.value = mergeByConversationId([], convs)
  }

  function mergeConversations(incoming: Conversation[]) {
    conversations.value = mergeByConversationId(conversations.value, incoming)
  }

  function updateConversation(conversation: Conversation) {
    const index = conversations.value.findIndex(c => sameConversationId(c.id, conversation.id))
    if (index !== -1) {
      conversations.value[index] = conversation
      conversations.value = [...conversations.value]
    }
  }

  function patchConversation(id: string, updates: Partial<Conversation>) {
    const index = conversations.value.findIndex(c => sameConversationId(c.id, id))
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
    const exists = conversations.value.some(c => sameConversationId(c.id, conversation.id))
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

  // AI 回复「思考中」占位状态：收到 ai_reply_started 置 true（并挂 90s 兜底计时），
  // 回复消息/系统提示到达（isAiThinking=false）或兜底超时置 false。
  // sender 为事件携带的回复者信息，用于把占位渲染成带头像的正常消息行。
  function setAiThinking(conversationId: string, thinking: boolean, sender?: ThinkingSender | null) {
    if (thinking) {
      aiThinking.value = new Map(aiThinking.value).set(conversationId, true)
      if (sender) {
        aiThinkingSenders.value = new Map(aiThinkingSenders.value).set(conversationId, sender)
      }
      const prev = aiThinkingTimers.get(conversationId)
      if (prev) clearTimeout(prev)
      aiThinkingTimers.set(conversationId, setTimeout(() => {
        aiThinkingTimers.delete(conversationId)
        setAiThinking(conversationId, false)
      }, AI_THINKING_TIMEOUT))
    } else {
      const prev = aiThinkingTimers.get(conversationId)
      if (prev) clearTimeout(prev)
      aiThinkingTimers.delete(conversationId)
      if (aiThinking.value.has(conversationId)) {
        const next = new Map(aiThinking.value)
        next.delete(conversationId)
        aiThinking.value = next
      }
      if (aiThinkingSenders.value.has(conversationId)) {
        const next = new Map(aiThinkingSenders.value)
        next.delete(conversationId)
        aiThinkingSenders.value = next
      }
    }
  }

  function isAiThinking(conversationId: string): boolean {
    return aiThinking.value.get(conversationId) === true
  }

  function getAiThinkingSender(conversationId: string): ThinkingSender | null {
    return aiThinkingSenders.value.get(conversationId) || null
  }

  // 业务逻辑方法
  function pinConversation(id: string, is_pinned: boolean) {
    const index = conversations.value.findIndex(c => sameConversationId(c.id, id))
    if (index !== -1) {
      conversations.value[index] = {
        ...conversations.value[index],
        is_pinned,
        pinnedAt: is_pinned ? Date.now() : undefined
      }
      conversations.value = [...conversations.value]
      saveToStorage(conversations.value)
    }
  }

  function muteConversation(id: string, muted: boolean) {
    const index = conversations.value.findIndex(c => sameConversationId(c.id, id))
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
    const index = conversations.value.findIndex(c => sameConversationId(c.id, id))
    if (index !== -1) {
      conversations.value.splice(index, 1)
      conversations.value = [...conversations.value]
      messages.value.delete(id)
      drafts.value.delete(id)
      setAiThinking(id, false)
      saveToStorage(conversations.value)
    }
  }

  function recallMessage(conversationId: string, messageId: string, extra?: string) {
    updateMessage(conversationId, messageId, {
      content: '[消息已撤回]',
      isRecalled: true,
      extra: extra || ''
    })

    const convIndex = conversations.value.findIndex(c => sameConversationId(c.id, conversationId))
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
    const msgs = messages.value.get(conversationId) || []
    const existingIndex = msgs.findIndex(m => m.id === message.id)

    if (existingIndex !== -1) {
      // 消息已存在，更新（流式消息的 chunk 更新）
      msgs[existingIndex] = { ...msgs[existingIndex], ...message }
      messages.value.set(conversationId, [...msgs])
    } else {
      // 新消息，追加
      msgs.push(message)
      messages.value.set(conversationId, [...msgs])
    }

    const convIndex = conversations.value.findIndex(c => sameConversationId(c.id, conversationId))
    if (convIndex !== -1) {
      const conv = conversations.value[convIndex]
      const updatedConv = {
        ...conv,
        lastMessage: message,
        timestamp: message.timestamp || Date.now()
      }

      if (!isCurrentConversation && !message.isStreaming) {
        const isMuted = conv.muted === true
        if (!isMuted) {
          updatedConv.unread_count = (updatedConv.unread_count || 0) + 1
        }
      }

      conversations.value[convIndex] = updatedConv
      conversations.value = [...conversations.value]
      saveToStorage(conversations.value)
    }
  }

  function deleteMessage(conversationId: string, messageId: string) {
    const msgs = messages.value.get(conversationId) || []
    const index = msgs.findIndex(m => m.id === messageId)
    if (index !== -1) {
      msgs.splice(index, 1)
      messages.value.set(conversationId, [...msgs])
    }
  }

  function markConversationRead(id: string) {
    const index = conversations.value.findIndex(c => sameConversationId(c.id, id))
    if (index !== -1) {
      conversations.value[index] = {
        ...conversations.value[index],
        unread_count: 0
      }
      conversations.value = [...conversations.value]
      saveToStorage(conversations.value)
    }
  }

  function addGroupMember(conversationId: string, member: any) {
    const index = conversations.value.findIndex(c => sameConversationId(c.id, conversationId))
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
    const index = conversations.value.findIndex(c => sameConversationId(c.id, conversationId))
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
    const index = conversations.value.findIndex(c => sameConversationId(c.id, conversationId))
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

  function getLastMessageId(conversationId: string): string | null {
    const msgs = messages.value.get(conversationId)
    if (!msgs || msgs.length === 0) return null
    return msgs[msgs.length - 1].id
  }

  function appendMessagesSilent(conversationId: string, newMessages: Message[]) {
    if (newMessages.length === 0) return
    const msgs = messages.value.get(conversationId) || []
    const existingIds = new Set(msgs.map(m => m.id))
    const filtered = newMessages.filter(m => !existingIds.has(m.id))
    if (filtered.length === 0) return
    messages.value.set(conversationId, [...msgs, ...filtered])
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
    aiThinking,
    // 计算属性
    currentConversation,
    currentMessages,
    sortedConversations,
    totalUnreadCount,
    // 基础方法
    setCurrentConversation,
    setMessages,
    appendMessage,
    prependMessages,
    mergeMessages,
    updateMessage,
    setConversations,
    mergeConversations,
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
    setAiThinking,
    isAiThinking,
    getAiThinkingSender,
    // 业务逻辑方法
    pinConversation,
    muteConversation,
    removeConversation,
    recallMessage,
    receiveMessage,
    deleteMessage,
    markConversationRead,
    addGroupMember,
    removeGroupMember,
    updateMemberRole,
    loadConversationsFromStorage,
    clearAllMessages,
    clearMessages: clearAllMessages,
    getLastMessageId,
    appendMessagesSilent,
  }
})
