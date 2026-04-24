<template>
  <div v-if="visible" class="context-menu" :style="{ left: position.x + 'px', top: position.y + 'px' }">
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
    <!-- 通用选项 -->
    <div v-if="message && (message.type === 'text' || message.type === 'image')" class="context-menu-item" @click="handleCopyMessage">
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
    <div v-if="message && message.type === 'text'" class="context-menu-item" @click="handleAddToNote">
      <span class="context-menu-icon"><i class="fas fa-sticky-note"></i></span>
      <span>添加到便签</span>
    </div>
    <!-- <div class="context-menu-item" @click="handleDeleteMessage">
      <span class="context-menu-icon"><i class="fas fa-trash"></i></span>
      <span>删除</span>
    </div> -->
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
  (e: 'add-to-note'): void
  (e: 'recall-message'): void
  (e: 'send-message-reminder'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

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

const handleAddToNote = () => {
  emit('add-to-note')
}

const handleRecallMessage = () => {
  emit('recall-message')
}

const handleSendMessageReminder = () => {
  emit('send-message-reminder')
}
</script>
