<template>
  <div class="input-toolbar">
    <ChatToolbarButton
      icon="fas fa-phone-alt"
      title="语音通话"
      @click="$emit('start-voice-call')"
    />
    <ChatToolbarButton
      icon="fas fa-video"
      title="视频通话"
      @click="$emit('start-video-call')"
    />
    <ChatToolbarButton
      icon="fas fa-desktop"
      title="屏幕共享"
      @click="$emit('start-screen-share')"
    />
    <ChatToolbarButton
      icon="fas fa-smile"
      title="表情"
      @click="$emit('toggle-emoji-panel')"
    />
    <ChatToolbarButton
      icon="fas fa-paperclip"
      title="发送文件"
      @click="$emit('select-file')"
    />
    <ChatToolbarButton
      icon="fas fa-image"
      title="发送图片"
      @click="$emit('select-image')"
    />
    <ChatToolbarButton
      v-if="isElectron"
      icon="fas fa-scissors"
      title="截图"
      @click="$emit('take-screenshot')"
    />
    <ChatToolbarButton
      icon="fas fa-history"
      title="消息管理"
      @click="$emit('open-message-manager')"
    />
    <ChatToolbarButton
      icon="fas fa-th-large"
      title="小程序"
      @click="$emit('open-mini-app-list')"
    />
    <div class="toolbar-divider"></div>
    <ChatToolbarButton
      icon="fas fa-robot"
      title="AI 功能"
      variant="ai"
      :class="{ 'ai-active': showAiActions }"
      @click="$emit('toggle-ai-actions')"
    />
  </div>
</template>

<script setup lang="ts">
import ChatToolbarButton from './ChatToolbarButton.vue'

interface Props {
  isElectron: boolean
  showAiActions: boolean
}

defineProps<Props>()

defineEmits<{
  'start-voice-call': []
  'start-video-call': []
  'start-screen-share': []
  'toggle-emoji-panel': []
  'select-file': []
  'select-image': []
  'take-screenshot': []
  'open-message-manager': []
  'open-mini-app-list': []
  'toggle-ai-actions': []
}>()
</script>

<style scoped>
.input-toolbar {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 12px;
  /* border-bottom: 1px solid var(--border-color, #E5E5E5); */
  /* background: var(--bg-secondary, #F9F9F9); */
}

.toolbar-divider {
  width: 1px;
  height: 20px;
  background: var(--border-color, #E5E5E5);
  margin: 0 4px;
}
</style>
