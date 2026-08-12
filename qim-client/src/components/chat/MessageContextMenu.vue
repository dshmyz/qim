<template>
  <UniversalContextMenu menuId="message" :items="menuItems" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Message } from '../../types'
import { useSystemConfigStore } from '../../stores/systemConfig'
import UniversalContextMenu from '../shared/UniversalContextMenu.vue'
import type { ContextMenuItem } from '../shared/context-menu-types'

const systemConfigStore = useSystemConfigStore()

interface Props {
  message: Message | null
  conversationType?: string
  canManageGroupFiles?: boolean
}

interface Emits {
  (e: 'preview-image', imageUrl: string): void
  (e: 'save-file-as', content: string): void
  (e: 'download-file', content: string): void
  (e: 'copy-message'): void
  (e: 'copy-code'): void
  (e: 'forward-message'): void
  (e: 'select-messages'): void
  (e: 'quote-message'): void
  (e: 'add-to-notes-app'): void
  (e: 'create-task'): void
  (e: 'recall-message'): void
  (e: 'send-message-reminder'): void
  (e: 'ai-summary'): void
  (e: 'translate'): void
  (e: 'smart-reply'): void
  (e: 'save-to-group-files'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const canRecall = computed(() => {
  if (!props.message) return false
  return systemConfigStore.canRecall(props.message.timestamp)
})

const aiEnabled = computed(() => systemConfigStore.enableAI)

const isAIMessage = computed(() => {
  if (!props.message) return false
  const msgOrigin = props.message.origin
  const senderType = props.message.sender?.type
  return msgOrigin === 'assistant' || msgOrigin === 'avatar' || senderType === 'bot' || senderType === 'system' || props.message.isAIMessage || props.message.is_ai_message
})

const isTextLikeMessage = computed(() => {
  if (!props.message) return false
  const type = props.message.type
  return type === 'text' || type === 'markdown' || isAIMessage.value
})

const hasCodeBlock = computed(() => {
  if (!props.message || !props.message.content) return false
  return /```[a-zA-Z0-9]*\n[\s\S]*?```/.test(props.message.content)
})

const canSendReminder = computed((): boolean => systemConfigStore.canRemind(props.message, props.conversationType))

const canSmartReply = computed(() => {
  if (!props.message || !aiEnabled.value) return false
  if (props.message.isSelf) return false
  return props.message.type === 'text'
})

const canSaveFileToGroup = computed(() =>
  props.message?.type === 'file' && props.conversationType === 'group' && props.canManageGroupFiles === true
)

const menuItems = computed<ContextMenuItem[]>(() => {
  const msg = props.message
  if (!msg) return []
  const items: ContextMenuItem[] = []

  // 图片消息
  if (msg.type === 'image') {
    items.push(
      { label: '复制', icon: 'fas fa-copy', action: () => emit('copy-message') },
      { label: '另存为', icon: 'fas fa-save', action: () => { if (msg.content) emit('save-file-as', msg.content) } }
    )
  }

  // 文件消息
  if (msg.type === 'file') {
    items.push(
      { label: '下载', icon: 'fas fa-download', action: () => { if (msg.content) emit('download-file', msg.content) } },
      { label: '另存为', icon: 'fas fa-save', action: () => { if (msg.content) emit('save-file-as', msg.content) } }
    )
    if (canSaveFileToGroup.value) {
      items.push({ label: '保存到群文件', icon: 'fas fa-folder-plus', action: () => emit('save-to-group-files') })
    }
  }

  // 分隔线
  if (msg.type === 'image' || msg.type === 'file') {
    items.push({ divider: true })
  }

  // AI 操作
  if (isAIMessage.value) {
    items.push({ label: 'AI 总结此消息', icon: 'fas fa-robot', action: () => emit('ai-summary') })
  }
  if (isTextLikeMessage.value && aiEnabled.value) {
    items.push({ label: '翻译为中文', icon: 'fas fa-language', action: () => emit('translate') })
  }
  if (canSmartReply.value) {
    items.push({ label: '智能回复', icon: 'fas fa-reply', action: () => emit('smart-reply') })
  }
  if (isAIMessage.value || (isTextLikeMessage.value && aiEnabled.value)) {
    items.push({ divider: true })
  }

  // 通用选项
  if (isTextLikeMessage.value) {
    items.push({ label: '复制', icon: 'fas fa-copy', action: () => emit('copy-message') })
  }
  if (hasCodeBlock.value) {
    items.push({ label: '复制代码', icon: 'fas fa-code', action: () => emit('copy-code') })
  }
  items.push(
    { label: '转发', icon: 'fas fa-share-alt', action: () => emit('forward-message') },
    { label: '多选', icon: 'fas fa-check-square', visible: msg.type !== 'system', action: () => emit('select-messages') },
    { label: '引用', icon: 'fas fa-quote-right', action: () => emit('quote-message') }
  )
  if (isTextLikeMessage.value) {
    items.push(
      { label: '保存到笔记', icon: 'fas fa-book', action: () => emit('add-to-notes-app') },
      { label: '创建为任务', icon: 'fas fa-tasks', action: () => emit('create-task') }
    )
  }
  if (msg.type === 'share' && msg.shareData?.type === 'note') {
    items.push({ label: '保存到笔记', icon: 'fas fa-book', action: () => emit('add-to-notes-app') })
  }
  if (msg.isSelf && canRecall.value) {
    items.push({ label: '撤回', icon: 'fas fa-undo', action: () => emit('recall-message') })
  }
  if (msg.isSelf && canSendReminder.value) {
    items.push({ label: '发送提醒', icon: 'fas fa-bell', action: () => emit('send-message-reminder') })
  }

  return items
})
</script>
