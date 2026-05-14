<template>
  <div class="ai-assistant-app">
    <AppHeader :title="showChatView ? currentBotName : 'AI 工作台'" @back="handleBack">
      <template #extra-buttons>
        <button class="toggle-sidebar-btn" @click="$emit('toggleSidebar')">
          <i class="fas fa-compress"></i>
        </button>
      </template>
      <template #actions>
        <button v-if="showChatView" class="header-action-btn" @click="backToDashboard">
          <i class="fas fa-th"></i>
          工作台
        </button>
      </template>
    </AppHeader>

    <div class="ai-content">
      <AIWorkbenchDashboard
        v-if="!showChatView"
        @use-bot="handleUseBot"
      />
      <BotChatView
        v-else
        :bot="selectedBot"
        :messages="botMessages"
        :is-loading="isLoading"
        :is-sending="isSending"
        :is-streaming="isStreaming"
        :error="chatError"
        @back="backToDashboard"
        @send="handleSendMessage"
        @clear-messages="handleClearMessages"
        @new-conversation="handleNewConversation"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import AppHeader from './AppHeader.vue'
import AIWorkbenchDashboard from './ai/AIWorkbenchDashboard.vue'
import BotChatView from './ai/BotChatView.vue'
import { useBotChat } from '../../composables/useBotChat'

defineEmits(['back', 'toggleSidebar'])

interface Bot {
  id: number
  name: string
  description?: string
  avatar?: string
}

const showChatView = ref(false)
const selectedBot = ref<Bot | null>(null)

const selectedBotId = computed(() => selectedBot.value?.id ?? null)

const {
  messages: botMessages,
  isLoading,
  isSending,
  isStreaming,
  error: chatError,
  loadMessages,
  sendMessage,
  clearMessages,
  reset
} = useBotChat(selectedBotId)

const currentBotName = computed(() => selectedBot.value?.name || 'AI 对话')

function handleBack() {
  if (showChatView.value) {
    backToDashboard()
  } else {
  }
}

function backToDashboard() {
  showChatView.value = false
  selectedBot.value = null
  // 不调用 reset()，保留 conversationId 以便下次重新进入时能加载历史记录
}

async function handleUseBot(bot: Bot | null) {
  selectedBot.value = bot
  showChatView.value = true
  await loadMessages()
}

async function handleSendMessage(content: string) {
  await sendMessage(content)
}

function handleClearMessages() {
  clearMessages()
}

async function handleNewConversation() {
  reset()
  await loadMessages()
}
</script>

<style scoped>
.ai-assistant-app {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--content-bg);
  overflow: hidden;
}

.ai-content {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
</style>