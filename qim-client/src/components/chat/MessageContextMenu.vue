<template>
  <div v-if="visible" class="context-menu" :style="{ left: position.x + 'px', top: position.y + 'px' }" @click.stop>
    <!-- 图片消息选项 -->
    <div v-if="message && message.type === 'image'" class="context-menu-item" @click="handlePreviewImage">
      <span class="context-menu-icon"><i class="fas fa-eye"></i></span>
      <span>预览</span>
    </div>
    <div v-if="message && message.type === 'image'" class="context-menu-item" @click="handleSaveImage">
      <span class="context-menu-icon"><i class="fas fa-save"></i></span>
      <span>保存图片</span>
    </div>
    <!-- 文件消息选项 -->
    <div v-if="message && message.type === 'file'" class="context-menu-item" @click="handleDownloadFile">
      <span class="context-menu-icon"><i class="fas fa-download"></i></span>
      <span>下载</span>
    </div>
    <div v-if="message && message.type === 'file'" class="context-menu-item" @click="handleSaveFileAs">
      <span class="context-menu-icon"><i class="fas fa-save"></i></span>
      <span>另存为</span>
    </div>
    <!-- 分隔线 -->
    <div v-if="message && (message.type === 'image' || message.type === 'file')" class="context-menu-divider"></div>

    <!-- AI 操作 -->
    <div v-if="isAIMessage" class="context-menu-item" @click="handleAIAction('ai_summary')">
      <span class="context-menu-icon"><i class="fas fa-robot"></i></span>
      <span>AI 总结此消息</span>
    </div>
    <div v-if="isTextLikeMessage" class="context-menu-item" @click="handleAIAction('translate')">
      <span class="context-menu-icon"><i class="fas fa-language"></i></span>
      <span>翻译为中文</span>
    </div>
    <div v-if="isAIMessage || isTextLikeMessage" class="context-menu-divider"></div>

    <!-- 通用选项 -->
    <div v-if="isTextLikeMessage" class="context-menu-item" @click="handleCopyMessage">
      <span class="context-menu-icon"><i class="fas fa-copy"></i></span>
      <span>复制</span>
    </div>
    <div class="context-menu-item" @click="handleForwardMessage">
      <span class="context-menu-icon"><i class="fas fa-share-alt"></i></span>
      <span>转发</span>
    </div>
    <div class="context-menu-item" @click="handleQuoteMessage">
      <span class="context-menu-icon"><i class="fas fa-quote-right"></i></span>
      <span>引用</span>
    </div>
    <div v-if="isTextLikeMessage" class="context-menu-item" @click="handleAddToNotesApp">
      <span class="context-menu-icon"><i class="fas fa-book"></i></span>
      <span>保存到笔记</span>
    </div>
    <div v-if="isTextLikeMessage" class="context-menu-item" @click="handleCreateTask">
      <span class="context-menu-icon"><i class="fas fa-check-square"></i></span>
      <span>创建为任务</span>
    </div>
    <div v-if="message && message.isSelf" class="context-menu-item" @click="handleRecallMessage">
      <span class="context-menu-icon"><i class="fas fa-undo"></i></span>
      <span>撤回</span>
    </div>
    <div v-if="message && message.isSelf && canSendReminder" class="context-menu-item" @click="handleSendMessageReminder">
      <span class="context-menu-icon"><i class="fas fa-bell"></i></span>
      <span>发送提醒</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Message } from '../../types'
import { useAIActions } from '../../composables/useAIActions'

interface Props {
  visible: boolean
  position: { x: number; y: number }
  message: Message | null
}

interface Emits {
  (e: 'preview-image', imageUrl: string): void
  (e: 'save-file-as', content: string): void
  (e: 'download-file', content: string): void
  (e: 'copy-message'): void
  (e: 'forward-message'): void
  (e: 'quote-message'): void
  (e: 'add-to-notes-app'): void
  (e: 'create-task'): void
  (e: 'recall-message'): void
  (e: 'send-message-reminder'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { translateText } = useAIActions()

const isAIMessage = computed(() => {
  if (!props.message) return false
  const senderType = props.message.sender?.type
  return senderType === 'bot' || senderType === 'system' || props.message.isAIMessage || props.message.is_ai_message
})

const isTextLikeMessage = computed(() => {
  if (!props.message) return false
  const type = props.message.type
  return type === 'text' || type === 'image' || isAIMessage.value
})

const canSendReminder = computed((): boolean => {
  if (!props.message || !props.message.timestamp || props.message.isRead) return false
  if (props.message.sender?.isBot) return false

  const now = Date.now()
  const messageTime = new Date(props.message.timestamp).getTime()
  const oneHour = 60 * 60 * 1000

  return now - messageTime > oneHour
})

const handlePreviewImage = () => {
  if (props.message && props.message.content) {
    emit('preview-image', props.message.content)
  }
}

const handleSaveImage = () => {
  if (props.message && props.message.content) {
    emit('save-file-as', props.message.content)
  }
}

const handleDownloadFile = () => {
  if (props.message && props.message.content) {
    emit('download-file', props.message.content)
  }
}

const handleSaveFileAs = () => {
  if (props.message && props.message.content) {
    emit('save-file-as', props.message.content)
  }
}

const handleCopyMessage = () => {
  emit('copy-message')
}

const handleForwardMessage = () => {
  emit('forward-message')
}

const handleQuoteMessage = () => {
  emit('quote-message')
}

const handleAddToNotesApp = () => {
  emit('add-to-notes-app')
}

const handleCreateTask = () => {
  emit('create-task')
}

const handleRecallMessage = () => {
  emit('recall-message')
}

const handleSendMessageReminder = () => {
  emit('send-message-reminder')
}

const handleAIAction = async (actionId: string) => {
  if (!props.message || !props.message.content) return

  try {
    switch (actionId) {
      case 'ai_summary':
        emit('copy-message')
        break
      case 'translate':
        await translateText(props.message.content, 'zh')
        break
    }
  } catch {
  }
}
</script>

<style scoped>
.context-menu {
  position: fixed;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  padding: 6px 0;
  z-index: 3000;
  min-width: 180px;
}

.context-menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  cursor: pointer;
  transition: background 0.15s;
  font-size: 14px;
  color: var(--text-color);
}

.context-menu-item:hover {
  background: var(--hover-color);
}

.context-menu-icon {
  width: 16px;
  text-align: center;
  font-size: 14px;
  color: var(--primary-color);
}

.context-menu-divider {
  height: 1px;
  background: var(--border-color);
  margin: 4px 0;
}
</style>