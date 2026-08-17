<template>
  <div class="ai-assistant-app">
    <AppHeader :title="showChatView ? currentBotName : 'AI 工作台'" @back="handleBack">
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
        :has-more-messages="hasMoreMessages"
        :error="chatError"
        :history-threads="conversationThreads"
        :current-conversation-id="conversationId"
        @back="backToDashboard"
        @send="handleSendMessage"
        @clear-messages="handleClearMessages"
        @new-conversation="handleNewConversation"
        @load-more="handleLoadMore"
        @stop-stream="handleStopStream"
        @switch-history="handleSwitchHistory"
        @retry-message="handleRetryMessage"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import AppHeader from './AppHeader.vue'
import AIWorkbenchDashboard from './ai/AIWorkbenchDashboard.vue'
import BotChatView from './ai/BotChatView.vue'
import { useBotChat } from '../../composables/useBotChat'
import { useBots } from '../../composables/useBots'
import type { BotMessage } from '../../types/bot'

const emit = defineEmits(['back'])

const props = defineProps<{
  // 悬浮球「AI 工作台」入口的复位信号：每次点击递增，回到工作台仪表盘
  workbenchResetKey?: number
}>()

// 悬浮球「AI 工作台」：即便已在 bot 聊天子视图内，也回到工作台仪表盘
watch(() => props.workbenchResetKey, () => {
  if (props.workbenchResetKey && props.workbenchResetKey > 0) {
    backToDashboard()
  }
})

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
  openBot,
  startNewConversation,
  setActiveThread
} = useBotChat(selectedBotId)

const { fetchBots } = useBots()

const currentBotName = computed(() => selectedBot.value?.name || 'AI 对话')

function handleBack() {
  if (showChatView.value) {
    backToDashboard()
  } else {
    emit('back')
  }
}

function backToDashboard() {
  showChatView.value = false
  selectedBot.value = null
  // 不调用 reset()，保留 conversationId 以便下次重新进入时能加载历史记录
}

async function handleUseBot(bot: Bot | null) {
  // 「找AI聊聊」入口传 null：先解析为系统内置 bot（系统助手），走真实 bot 会话通道
  // —— 服务端 conversations 持久化、回复带 bot 身份、历史跨设备可见。
  if (!bot) {
    bot = await resolveGenericAssistant()
    if (!bot) return // 解析失败，停留工作台
  }
  selectedBot.value = bot
  showChatView.value = true
  // 对象级开关：同一 bot 重新进入复用会话（AIAssistantApp.backToDashboard 保留
  // conversationId），切换 bot 时内部自动 reset，避免串话。
  await openBot(bot.id)
}

/**
 * 「找AI聊聊」通用入口解析：返回系统内置的 AI 助手 bot。
 *
 * 对端身份只认系统内置 assistant（type=assistant、is_template=false、creator_id=0）——
 * 模板是创建向导用的样板（is_template=true），不参与聊天；系统账号（users.type=system）
 * 是管理员「群发私聊/系统通知」用的广播身份，不应作为个人 AI 对话对端。内置 bot 缺失时
 * 依次兜底任意非模板 assistant → 模板 assistant → 系统账号（仅极端环境）。
 */
async function resolveGenericAssistant(): Promise<Bot | null> {
  try {
    const bots: any[] = await fetchBots()
    const target =
      bots.find(b => b.type === 'assistant' && b.is_template === false && b.creator_id === 0) ||
      bots.find(b => b.type === 'assistant' && b.is_template === false) ||
      bots.find(b => b.type === 'assistant' && b.is_template === true) ||
      bots.find(b => b.type === 'system')
    if (!target) {
      console.warn('[AI工作台] 未找到 AI助手/助手 bot，找AI聊聊入口不可用')
      return null
    }
    return {
      id: target.id,
      name: target.name,
      description: target.description,
      avatar: target.avatar
    }
  } catch (e) {
    console.error('[AI工作台] 解析 AI助手 bot 失败:', e)
    return null
  }
}

async function handleSendMessage(content: string) {
  await sendMessage(content)
}

function handleRetryMessage(msg: BotMessage) {
  retryMessage(msg)
}

function handleClearMessages() {
  clearMessages()
}

async function handleNewConversation() {
  // 多会话「新话题」：bot 真正开新段，各段上下文互不污染
  await startNewConversation()
}

// 历史「加载更多」/ 停止生成 接线
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