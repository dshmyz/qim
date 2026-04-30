<template>
  <div class="chat-center">
    <BotList
      v-if="!selectedBotId"
      :bots="bots"
      @select="selectBot"
      @createBot="$emit('switchTab', 'create')"
    />
    <BotChatView
      v-else
      :bot="currentBot"
      :messages="messages"
      :thinking="thinking"
      @back="selectedBotId = ''"
      @send="handleSendMessage"
      @setThinking="thinking = $event"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useBots } from '../../../composables/useBots'
import BotList from './BotList.vue'
import BotChatView from './BotChatView.vue'

interface Bot {
  id: number
  name: string
  description?: string
  avatar?: string
}

interface Message {
  id: number
  content: string
  sender: 'user' | 'bot' | 'system'
  timestamp: Date
}

defineEmits<{
  switchTab: [tab: string]
}>()

const { fetchMyBots } = useBots()
const bots = ref<Bot[]>([])
const selectedBotId = ref('')
const messages = ref<Message[]>([])
const thinking = ref(false)

onMounted(async () => {
  const allBots = await fetchMyBots()
  bots.value = allBots.filter((bot: Bot) => bot.name !== '系统助手')
})

const currentBot = computed<Bot | null>(() =>
  bots.value.find(b => b.id === parseInt(selectedBotId.value)) || null
)

function selectBot(botId: number) {
  selectedBotId.value = botId.toString()
  messages.value = []
  thinking.value = false
}

function handleSendMessage(content: string) {
  messages.value.push({
    id: Date.now(),
    content,
    sender: 'user',
    timestamp: new Date()
  })
}
</script>

<style scoped>
.chat-center {
  height: 100%;
  display: flex;
  flex-direction: column;
}
</style>
