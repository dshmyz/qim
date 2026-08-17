<template>
  <div class="chat-center">
    <BotList
      v-if="!selectedBotId"
      :bots="bots"
      :loading="loadingBots"
      @select="selectBot"
      @createBot="$emit('switchTab', 'create')"
    />
    <BotChatView
      v-else
      :bot="currentBot"
      :messages="botMessages"
      :is-loading="isLoading"
      :is-sending="isSending"
      :is-streaming="isStreaming"
      :has-more-messages="hasMoreMessages"
      :error="chatError"
      :history-threads="conversationThreads"
      :current-conversation-id="conversationId"
      @back="handleBack"
      @send="handleSendMessage"
      @clear-messages="handleClearMessages"
      @new-conversation="handleNewConversation"
      @load-more="handleLoadMore"
      @stop-stream="handleStopStream"
      @switch-history="handleSwitchHistory"
      @retry-message="handleRetryMessage"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useBots } from '../../../composables/useBots'
import { useBotChat } from '../../../composables/useBotChat'
import BotList from './BotList.vue'
import BotChatView from './BotChatView.vue'
import type { BotMessage } from '../../../types/bot'

interface Bot {
  id: number
  name: string
  description?: string
  avatar?: string
  approval_status?: string
}

defineEmits<{
  switchTab: [tab: string]
}>()

const { fetchMyBots } = useBots()
const bots = ref<Bot[]>([])
const loadingBots = ref(false)
const selectedBotId = ref<number | null>(null)

// 使用 useBotChat 管理 Bot 对话
const {
  conversationId,
  conversationThreads,
  messages: botMessages,
  isLoading,
  isSending,
  isStreaming,
  error: chatError,
  hasMoreMessages,
  loadMoreMessages,
  sendMessage,
  retryMessage,
  cancelStream,
  clearMessages,
  reset,
  openBot,
  startNewConversation,
  setActiveThread
} = useBotChat(selectedBotId)

onMounted(async () => {
  loadingBots.value = true
  try {
    const allBots = await fetchMyBots()
    bots.value = allBots.filter((bot: Bot) => bot.name !== '系统助手')
  } finally {
    loadingBots.value = false
  }
})

const currentBot = computed<Bot | null>(() =>
  bots.value.find(b => b.id === selectedBotId.value) || null
)

/**
 * 选择 Bot 并初始化会话
 */
async function selectBot(botId: number) {
  selectedBotId.value = botId
  // openBot：对象级开关，切换 bot 时内部 reset，复用会话/多轮历史语义与工作台一致
  await openBot(botId)
}

/**
 * 返回 Bot 列表
 */
function handleBack() {
  selectedBotId.value = null
  reset()
}

/**
 * 发送消息
 */
async function handleSendMessage(content: string) {
  await sendMessage(content)
}

/**
 * 重发发送失败的用户消息
 */
function handleRetryMessage(msg: BotMessage) {
  retryMessage(msg)
}

/**
 * 清空对话
 */
function handleClearMessages() {
  clearMessages()
}

/**
 * 新建对话（多会话「新话题」：bot 真正开新段，上下文互不污染）
 */
async function handleNewConversation() {
  await startNewConversation()
}

/**
 * 历史「加载更多」/ 停止生成 接线
 */
function handleLoadMore() {
  loadMoreMessages()
}

function handleStopStream() {
  cancelStream()
}

/**
 * 历史会话下拉：切回某段旧线程
 */
async function handleSwitchHistory(id: number | string) {
  await setActiveThread(Number(id))
}

// 监听 selectedBotId 变化，重置状态
watch(selectedBotId, (newId, oldId) => {
  if (oldId !== null && newId !== oldId) {
    reset()
  }
})
</script>

<style scoped>
.chat-center {
  height: 100%;
  display: flex;
  flex-direction: column;
}
</style>
