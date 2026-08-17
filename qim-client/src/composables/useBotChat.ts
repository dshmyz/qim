// src/composables/useBotChat.ts

import { ref, computed, watch, type Ref } from 'vue'
import { useServerUrl } from './useServerUrl'
import { request } from './useRequest'
import { useCurrentUser } from './useCurrentUser'
import { useProcessConversation } from './useProcessConversation'
import { useChatStore } from '../stores/chat'
import type { BotMessage } from '../types/bot'

function normalizeSenderType(msg: any): BotMessage['senderType'] {
  // sender_type 可能是 bot/system/api 或普通用户角色（user/admin/...）。
  // 除 bot/system/api 外一律归一为 user，否则管理员消息会带着 "admin"
  // 渲染成 bot 侧（左边白气泡），用户自己的消息显示错位。
  if (msg.sender_type) {
    if (msg.sender_type === 'bot' || msg.sender_type === 'system' || msg.sender_type === 'api') {
      return msg.sender_type
    }
    return 'user'
  }
  const senderType = msg.sender?.type
	if (
		msg.origin === 'assistant' ||
		msg.is_ai_message === true ||
		msg.isAIMessage === true ||
		senderType === 'bot' ||
		senderType === 'system'
	) {
    return senderType === 'system' ? 'system' : 'bot'
  }
  return 'user'
}

export function processBotMessage(msg: any): BotMessage {
  return {
    id: msg.id,
    conversationId: msg.conversation_id,
    senderId: msg.sender_id,
    senderType: normalizeSenderType(msg),
    sender: msg.sender ? {
      id: msg.sender.id,
      nickname: msg.sender.nickname || msg.sender.username || '未知用户',
      avatar: msg.sender.avatar,
      type: msg.sender.type
    } : undefined,
    type: msg.type || 'text',
    content: msg.content,
    timestamp: new Date(msg.created_at || msg.timestamp || Date.now()),
    isStreaming: false,
    // Bot 回复命中笔记时的知识来源（后端从 message.Extra 解析后放入响应体顶层）
    knowledge_sources: Array.isArray(msg.knowledge_sources) ? msg.knowledge_sources : undefined,
  }
}

export function normalizeBotHistoryMessages(messages: any[]): BotMessage[] {
  return messages.map((msg: any) => processBotMessage(msg))
}

export interface BotConversationThread {
  id: number
  name: string
  last_message_at: string | null
  updated_at: string
}

/**
 * Bot 对话管理 composable
 * 负责 Bot 会话的初始化、消息加载、发送和流式处理
 */
export function useBotChat(botId: Ref<number | null>) {
  // 会话状态
  const conversationId = ref<number | null>(null)
  const messages = ref<BotMessage[]>([])
  const isLoading = ref(false)
  const isSending = ref(false)
  const error = ref<string | null>(null)

  // 流式消息状态
  const streamingMessageId = ref<string | null>(null)
  const abortController = ref<AbortController | null>(null)

  // 当前用户信息
  const { currentUser } = useCurrentUser()

  // 最近会话列表实时更新：bot 会话建成后插入 store，与 openChat 私聊路径保持一致
  const { serverUrl } = useServerUrl()
  const { processConversation } = useProcessConversation(serverUrl, currentUser)
  const chatStore = useChatStore()

  // 分页状态
  const currentPage = ref(1)
  const pageSize = ref(20)
  const hasMoreMessages = ref(false)

  // 当前 bot 的多会话线程列表（「历史会话」下拉数据源）
  const conversationThreads = ref<BotConversationThread[]>([])

  // 已加载会话对应的 botId：openBot 用它做对象级重置。
  // 同一 bot 重新进入（AIAssistantApp.backToDashboard 保留 conversationId）时
  // 复用会话、仅重新拉取最新消息；切换到另一个 bot 时重置，避免相互串话。
  let loadedBotId: number | null | undefined = undefined

  /**
   * 打开一个 bot 对话对象。
   * 同一 bot 重新进入：保留会话状态，仅重新拉取最新消息；
   * 切换到另一个 bot：重置会话状态，避免相互串话。
   */
  const openBot = async (targetBotId: number | null): Promise<void> => {
    if (loadedBotId === targetBotId) {
      await loadMessages()
      return
    }
    reset()
    loadedBotId = targetBotId
    await loadMessages()
  }

  /**
   * 初始化 Bot 会话
   * 创建或获取与指定 Bot 的会话
   * @param fresh true=强制开一段全新会话（多会话「新话题」），false=复用最近一段
   *
   * Single-flight：并发调用（「新话题」POST 往返期间用户立即发送消息时，
   * sendMessage 会兜底 initConversation(false)）共用同一次请求，避免两个 POST
   * 竞速导致 conversationId 被后返回者覆盖、消息落入非预期线程。
   * 规则：复用请求（fresh=false）可共享任何在途初始化；强制新段（fresh=true）
   * 不能共享 fresh=false 的在途请求（新话题必须开新段），等它结束后再发。
   */
  let inflightInit: { fresh: boolean; promise: Promise<boolean> } | null = null

  const initConversation = async (fresh: boolean = false): Promise<boolean> => {
    while (inflightInit) {
      if (!fresh || inflightInit.fresh) {
        return inflightInit.promise
      }
      await inflightInit.promise.catch(() => false)
    }
    const promise = doInitConversation(fresh).finally(() => {
      if (inflightInit?.promise === promise) inflightInit = null
    })
    inflightInit = { fresh, promise }
    return promise
  }

  const doInitConversation = async (fresh: boolean = false): Promise<boolean> => {
    isLoading.value = true
    error.value = null

    try {
      const response: any = await request('/api/v1/conversations', {
        method: 'POST',
        body: JSON.stringify({ type: 'bot', bot_id: botId.value, fresh })
      })

      if (response.code === 0 && response.data) {
        conversationId.value = response.data.id || response.data.conversationId
        // 服务端可能返回的是复用已有会话（findBotSingleConversation），同样规范化后入列
        const processed = processConversation(response.data) as any
        const cid = String(response.data.id || response.data.conversationId)
        if (processed && !chatStore.conversations.some(c => String(c.id) === cid)) {
          chatStore.addConversation(processed)
        }
        return true
      } else {
        error.value = response.message || '初始化会话失败'
        return false
      }
    } catch (e: any) {
      error.value = e.message || '初始化会话失败'
      console.error('初始化 Bot 会话失败:', e)
      return false
    } finally {
      isLoading.value = false
    }
  }

  /**
   * 加载历史消息
   * @param reset 是否重置分页
   */
  const loadMessages = async (reset: boolean = true): Promise<void> => {
    if (!conversationId.value) {
      const initSuccess = await initConversation()
      if (!initSuccess || !conversationId.value) {
        return
      }
    }

    if (isLoading.value) return

    if (reset) {
      currentPage.value = 1
      hasMoreMessages.value = true
    }

    if (!hasMoreMessages.value) return

    isLoading.value = true
    error.value = null

    try {
      const response: any = await request(
        `/api/v1/conversations/${conversationId.value}/messages?page=${currentPage.value}&page_size=${pageSize.value}`
      )

      if (response.code === 0) {
        // 兼容两种返回格式：{ messages: [] } 或直接是 []
        const messagesArray = response.data?.messages || (Array.isArray(response.data) ? response.data : [])
        const serverMessages = Array.isArray(messagesArray)
          ? normalizeBotHistoryMessages(messagesArray)
          : []

        if (reset) {
          messages.value = serverMessages
        } else {
          messages.value = [...serverMessages, ...messages.value]
        }

        // 处理分页信息
        if (response.pagination) {
          const { current_page, total_pages } = response.pagination
          hasMoreMessages.value = current_page < total_pages
          currentPage.value = current_page + 1
        } else {
          hasMoreMessages.value = serverMessages.length === pageSize.value
          currentPage.value++
        }
      } else {
        if (reset) {
          messages.value = []
        }
        hasMoreMessages.value = false
      }
    } catch (e: any) {
      error.value = e.message || '加载消息失败'
      console.error('加载消息失败:', e)
      if (reset) {
        messages.value = []
      }
      hasMoreMessages.value = false
    } finally {
      isLoading.value = false
    }
  }

  /**
   * 加载更多消息
   */
  const loadMoreMessages = async (): Promise<void> => {
    await loadMessages(false)
  }

  /**
   * 新建会话（多会话「新话题」）。
   * 强制服务端开一段全新会话（fresh=true），各段上下文互相隔离；
   * 新线程会自动入列「历史会话」下拉。
   */
  const startNewConversation = async (): Promise<void> => {
    if (abortController.value) abortController.value.abort() // 终止进行中的流
    reset()
    await initConversation(true) // 新会话为空，无需再拉历史
    await loadConversationThreads() // 新线程入列「历史会话」下拉
  }

  /**
   * 拉取当前 bot 的全部会话线程（「历史会话」下拉数据源）。
   * 服务端按 bot_id 过滤（避免全局列表先按 100 条截断再客户端过滤导致旧线程丢失），
   * 循环翻页拉全量（上限 10 页为防御性兜底）；最新在前。
   */
  const loadConversationThreads = async (): Promise<void> => {
    if (!botId.value) return
    try {
      const rows: any[] = []
      for (let page = 1; page <= 10; page++) {
        const response: any = await request(
          `/api/v1/conversations?type=bot&bot_id=${botId.value}&page=${page}&page_size=100`
        )
        rows.push(...(response?.data?.list || []))
        if (!response?.data?.has_more) break
      }
      conversationThreads.value = rows
        .filter((c: any) => c.type === 'bot' && Number(c.bot_id) === botId.value)
        .map((c: any) => ({
          id: Number(c.id),
          name: c.name || '',
          last_message_at: c.last_message_at || null,
          updated_at: c.updated_at || c.created_at || ''
        }))
        .sort((a, b) => (a.updated_at > b.updated_at ? -1 : 1)) // 最新优先
    } catch (e) {
      console.error('拉取会话线程列表失败:', e)
    }
  }

  /**
   * 切换到历史会话线程（「历史会话」下拉点击）：在该 bot 内换段，不重置对象状态
   */
  const setActiveThread = async (threadId: number): Promise<void> => {
    if (abortController.value) abortController.value.abort() // 终止进行中的流
    // 切换线程前清除旧线程的「思考中」占位标记（与 reset 同理，满足切换即消失）
    const prevCid = conversationId.value
    if (prevCid != null && Number(prevCid) !== threadId) {
      chatStore.setAiThinking(String(prevCid), false)
    }
    conversationId.value = threadId
    await loadMessages(true)
  }

  /**
   * 发送消息（REST + WS streaming）
   * REST 立即返回用户消息，bot 回复由 WS streaming 推送，
   * 通过 watcher 同步 chatStore → 本 composable 的 messages。
   */
  const sendMessage = async (content: string): Promise<void> => {
    if (!content.trim()) {
      error.value = '消息内容不能为空'
      return
    }

    isSending.value = true
    error.value = null

    // 先确保会话就绪：openBot 已初始化过则复用；万一尚未就绪，这里补齐
    if (!conversationId.value) {
      const ok = await initConversation()
      if (!ok || !conversationId.value) return
    }

    // 添加用户消息
    const userMessage: BotMessage = {
      id: Date.now(),
      conversationId: conversationId.value,
      senderId: Number(currentUser.value?.id) || 0,
      senderType: 'user',
      sender: currentUser.value ? {
        id: Number(currentUser.value.id),
        nickname: currentUser.value.nickname || currentUser.value.username || '用户',
        avatar: currentUser.value.avatar,
        type: 'user'
      } : undefined,
      type: 'text',
      content: content.trim(),
      timestamp: new Date(),
      isStreaming: false
    }
    messages.value.push(userMessage)

    // 统一走 REST SendMessage（bot 回复由 WS streaming 推送，与群AI/分身同一通道）
    try {
      const response: any = await request(
        `/api/v1/conversations/${conversationId.value}/messages`,
        { method: 'POST', body: JSON.stringify({ type: 'text', content: content.trim() }) }
      )

      if (response.code !== 0) {
        throw new Error(response.message || '发送失败')
      }
      // REST 成功：用户消息已展示，bot 回复将由 WS streaming 推送
      // （1~3秒后 watcher 从 chatStore 同步到 BotChatView messages，无需本地占位）
    } catch (e: any) {
      error.value = e.message || '发送消息失败'
      console.error('发送消息失败:', e)
      // 标记用户消息为发送失败（气泡提供重发）
      const userMsgIndex = messages.value.findIndex(m => m.id === userMessage.id)
      if (userMsgIndex !== -1) {
        messages.value[userMsgIndex].isFailed = true
      }
    } finally {
      isSending.value = false
      streamingMessageId.value = null
    }
  }

  /**
   * 重发失败的用户消息：先从列表移除该失败消息（避免列表残留重复），再原样重发。
   * @param msg 发送失败的用户消息（isFailed === true）
   */
  const retryMessage = (msg: BotMessage): void => {
    const idx = messages.value.findIndex(m => m.id === msg.id)
    if (idx !== -1) {
      messages.value.splice(idx, 1)
    }
    sendMessage(msg.content)
  }

  /**
   * 取消当前流式消息
   */
  const cancelStream = (): void => {
    if (abortController.value) {
      abortController.value.abort()
    }
  }

  /**
   * 清空消息
   */
  const clearMessages = (): void => {
    messages.value = []
    currentPage.value = 1
    hasMoreMessages.value = false
  }

  /**
   * 重置会话
   */
  const reset = (): void => {
    // 离开当前 bot 会话（切换 bot / 新话题 / 清空）：先清除旧线程的「思考中」
    // 占位标记，避免标记滞留——否则切回旧线程时若回复仍未到会误显示占位，
    // 与「切换/离开会话占位即消失」的需求相悖。
    const prevCid = conversationId.value
    if (prevCid != null) {
      chatStore.setAiThinking(String(prevCid), false)
    }
    conversationId.value = null
    clearMessages()
    error.value = null
    isLoading.value = false
    isSending.value = false
    streamingMessageId.value = null
    if (abortController.value) {
      abortController.value.abort()
      abortController.value = null
    }
  }

  /**
   * 是否正在流式传输
   */
  const isStreaming = computed(() => streamingMessageId.value !== null)

  // bot 切换时刷新「历史会话」线程列表（含 null → 具体 bot / 具体 bot → 另一个 bot）
  watch(botId, () => {
    loadConversationThreads()
  })

  // WS bot 回复桥接：REST 发送后 bot 回复经 WS → Main.vue → chatStore，
  // 此处监听 chatStore 中当前会话的新 bot 消息，同步到 BotChatView 自有的 messages。
  watch(
    () => {
      const cid = conversationId.value
      return cid ? chatStore.messages.get(String(cid)) : undefined
    },
    (storeMsgs) => {
      if (!storeMsgs?.length || !conversationId.value) return
      for (const msg of storeMsgs) {
        if (msg.isSelf) continue
        const existing = messages.value.find(m => String(m.id) === String(msg.id))
        if (existing) {
          // 流式 chunk 更新：仅合并内容和流式标记，保留本地字段（isSelf/sender）
          existing.content = msg.content
          if (msg.type) existing.type = msg.type as BotMessage['type']
          if (msg.isStreaming !== undefined) existing.isStreaming = msg.isStreaming
          if (msg.tool_calls) (existing as any).tool_calls = msg.tool_calls
        } else {
          // 新消息：必须走 processBotMessage 规范化（chatStore 消息 timestamp 是 string，
          // BotMessage.timestamp 契约是 Date——否则 BotChatView.shouldShowDivider 调
          // getTime() 崩溃）；再叠加流式字段（首次 chunk 即带 isStreaming/tool_calls）
          const normalized = processBotMessage(msg as any)
          if (msg.isStreaming !== undefined) normalized.isStreaming = msg.isStreaming
          if (msg.tool_calls) (normalized as any).tool_calls = msg.tool_calls
          messages.value.push(normalized)
        }
      }
    },
    { deep: true }
  )

  return {
    // 状态
    conversationId,
    messages,
    isLoading,
    isSending,
    isStreaming,
    error,
    hasMoreMessages,
    conversationThreads,

    // 方法
    initConversation,
    loadMessages,
    loadMoreMessages,
    sendMessage,
    retryMessage,
    cancelStream,
    clearMessages,
    reset,
    openBot,
    startNewConversation,
    loadConversationThreads,
    setActiveThread
  }
}
