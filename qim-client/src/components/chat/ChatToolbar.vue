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
    <div class="screenshot-dropdown" v-if="isElectron">
      <ChatToolbarButton
        class="screenshot-btn"
        icon="fas fa-scissors"
        title="截图"
        @click="$emit('take-screenshot')"
      />
      <button class="screenshot-dropdown-trigger" @click="toggleScreenshotMenu" title="更多截图选项">
        <i class="fas fa-caret-down"></i>
      </button>
      <div v-show="showScreenshotMenu" class="screenshot-menu" @click.stop>
        <div class="screenshot-menu-item" @click="selectScreenshot('region')">
          <i class="fas fa-crop-alt"></i>
          <span>区域截图</span>
        </div>
        <div class="screenshot-menu-item" @click="selectScreenshot('hidden')">
          <i class="fas fa-window-minimize"></i>
          <span>隐藏窗口截图</span>
        </div>
      </div>
    </div>
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
    <div v-if="systemConfigStore.enableAI" class="toolbar-divider"></div>
    <ChatToolbarButton
      v-if="systemConfigStore.enableAI"
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
import { useSystemConfigStore } from '../../stores/systemConfig'
import { ref } from 'vue'

const systemConfigStore = useSystemConfigStore()

const showScreenshotMenu = ref(false)

const toggleScreenshotMenu = () => {
  showScreenshotMenu.value = !showScreenshotMenu.value
}

const selectScreenshot = (type: 'region' | 'hidden') => {
  showScreenshotMenu.value = false
  if (type === 'region') {
    emit('take-screenshot')
  } else {
    emit('take-screenshot-hidden')
  }
}

interface Props {
  isElectron: boolean
  showAiActions: boolean
}

defineProps<Props>()

const emit = defineEmits<{
  'start-voice-call': []
  'start-video-call': []
  'start-screen-share': []
  'toggle-emoji-panel': []
  'select-file': []
  'select-image': []
  'take-screenshot': []
  'take-screenshot-hidden': []
  'open-message-manager': []
  'open-mini-app-list': []
  'toggle-ai-actions': []
}>()

// 点击外部关闭截图下拉菜单
const onDocumentClick = (e: MouseEvent) => {
  const target = e.target as HTMLElement
  if (!target.closest('.screenshot-dropdown')) {
    showScreenshotMenu.value = false
  }
}

import { onMounted, onUnmounted } from 'vue'
onMounted(() => document.addEventListener('mousedown', onDocumentClick))
onUnmounted(() => document.removeEventListener('mousedown', onDocumentClick))
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

.screenshot-dropdown {
  position: relative;
  display: inline-flex;
  align-items: center;
}

.screenshot-btn {
  border-radius: 4px 0 0 4px !important;
}

.screenshot-dropdown-trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 32px;
  background: transparent;
  border: none;
  border-radius: 0 4px 4px 0;
  cursor: pointer;
  color: var(--text-secondary, #666);
  font-size: 10px;
  padding: 0;
  margin-left: -6px;
  transition: all 0.2s ease;
}

.screenshot-dropdown-trigger:hover {
  background: var(--hover-bg, rgba(0, 0, 0, 0.05));
  color: var(--text-primary, #333);
}

.screenshot-menu {
  position: absolute;
  top: 100%;
  left: 0;
  z-index: 1000;
  min-width: 140px;
  background: var(--bg-primary, #fff);
  border: 1px solid var(--border-color, #E5E5E5);
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
  padding: 4px 0;
  margin-top: 4px;
}

.screenshot-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  cursor: pointer;
  font-size: 13px;
  color: var(--text-primary, #333);
  white-space: nowrap;
  transition: background 0.15s ease;
}

.screenshot-menu-item:hover {
  background: var(--hover-bg, rgba(0, 0, 0, 0.05));
}

.screenshot-menu-item i {
  width: 16px;
  font-size: 13px;
  color: var(--text-secondary, #666);
}
</style>
