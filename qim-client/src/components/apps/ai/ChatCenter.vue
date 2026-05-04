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
      :error="chatError"
      @back="handleBack"
      @send="handleSendMessage"
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
  messages: botMessages,
  isLoading,
  isSending,
  isStreaming,
  error: chatError,
  loadMessages,
  sendMessage,
  reset
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
  // 加载历史消息
  await loadMessages()
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
