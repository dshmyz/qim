<template>
  <div class="input-toolbar">
    <div class="call-dropdown">
      <ChatToolbarButton
        class="call-btn"
        icon="fas fa-phone-alt"
        title="通话"
        @click="$emit('start-voice-call')"
      />
      <button class="call-dropdown-trigger" @click="toggleCallMenu($event)" title="更多通话选项">
        <i class="fas fa-caret-down"></i>
      </button>
      <UniversalContextMenu menuId="call" :items="callMenuItems" />
    </div>
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
      icon="fas fa-code"
      title="代码块"
      @click="$emit('open-code-block')"
    />
    <div
      class="screenshot-dropdown"
      v-if="isElectron"
      @mouseleave="showScreenshotShortcutTooltip = false"
    >
      <ChatToolbarButton
        class="screenshot-btn"
        icon="fas fa-scissors"
        :title="screenshotButtonTitle"
        @mouseenter="showScreenshotTooltip"
        @click="$emit('take-screenshot')"
      />
      <div
        v-if="showScreenshotShortcutTooltip"
        class="screenshot-shortcut-tooltip"
        data-testid="screenshot-shortcut-tooltip"
      >
        {{ screenshotButtonTitle }}
      </div>
      <button class="screenshot-dropdown-trigger" @click="toggleScreenshotMenu($event)" title="更多截图选项">
        <i class="fas fa-caret-down"></i>
      </button>
      <UniversalContextMenu menuId="screenshot" :items="screenshotMenuItems" />
    </div>
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
    <ChatToolbarButton
      class="message-manager-btn"
      icon="fas fa-history"
      title="消息管理"
      @click="$emit('open-message-manager')"
    />
  </div>
</template>

<script setup lang="ts">
import ChatToolbarButton from './ChatToolbarButton.vue'
import { useSystemConfigStore } from '../../stores/systemConfig'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { DEFAULT_SHORTCUTS, type ShortcutsConfig } from '../../composables/useShortcuts'
import UniversalContextMenu from '../shared/UniversalContextMenu.vue'
import { activeMenu, openMenu, closeMenu } from '../../composables/useUI'

const systemConfigStore = useSystemConfigStore()

const showScreenshotMenu = computed(() => activeMenu.value === 'screenshot')
const showCallMenu = computed(() => activeMenu.value === 'call')

const screenshotMenuItems = computed(() => [
  { label: screenshotButtonTitle.value, icon: 'fas fa-crop-alt', action: () => { emit('take-screenshot'); closeMenu() } },
  { label: '隐藏窗口截图', icon: 'fas fa-window-minimize', action: () => { emit('take-screenshot-hidden'); closeMenu() } }
])

const callMenuItems = computed(() => [
  { label: '语音通话', icon: 'fas fa-phone-alt', action: () => { emit('start-voice-call'); closeMenu() } },
  { label: '视频通话', icon: 'fas fa-video', action: () => { emit('start-video-call'); closeMenu() } }
])
const showScreenshotShortcutTooltip = ref(false)
const shortcuts = ref<ShortcutsConfig>(DEFAULT_SHORTCUTS)

const formatShortcutForTooltip = (accelerator: string): string => {
  return accelerator
    .split('+')
    .map(part => {
      const key = part.trim()
      if (key === 'CommandOrControl' || key === 'Control' || key === 'Mod') return 'Ctrl'
      if (key === 'Alt' || key === 'Option') return 'Alt'
      return key.length === 1 ? key.toUpperCase() : key
    })
    .join('+')
}

const screenshotButtonTitle = computed(() => {
  const shortcut = shortcuts.value.global?.screenshot
  if (!shortcut?.enabled || !shortcut.accelerator) return '截图'
  return `截图（${formatShortcutForTooltip(shortcut.accelerator)}）`
})

const loadScreenshotShortcut = async () => {
  if (!window.electron?.ipcRenderer?.invoke) return
  const loaded = await window.electron.ipcRenderer.invoke('get-shortcuts')
  shortcuts.value = {
    ...DEFAULT_SHORTCUTS,
    ...loaded,
    global: {
      ...DEFAULT_SHORTCUTS.global,
      ...loaded?.global,
    },
    editor: {
      ...DEFAULT_SHORTCUTS.editor,
      ...loaded?.editor,
    },
  }
}

const handleShortcutsUpdated = (_event: unknown, updatedShortcuts: ShortcutsConfig) => {
  shortcuts.value = {
    ...DEFAULT_SHORTCUTS,
    ...updatedShortcuts,
    global: {
      ...DEFAULT_SHORTCUTS.global,
      ...updatedShortcuts?.global,
    },
    editor: {
      ...DEFAULT_SHORTCUTS.editor,
      ...updatedShortcuts?.editor,
    },
  }
}

// 用点击目标自身定位锚点，避免 document.querySelector 误命中隐藏/错位元素导致菜单定位失效
const anchorRectFromEvent = (event: MouseEvent): { left: number; top: number } => {
  const btn = (event.currentTarget as HTMLElement) || event.target as HTMLElement
  const r = btn?.getBoundingClientRect?.()
  if (!r) return { left: 0, top: 0 }
  return { left: r.left, top: r.bottom + 4 }
}

const toggleScreenshotMenu = (event: MouseEvent) => {
  if (showScreenshotMenu.value) closeMenu()
  else {
    const { left, top } = anchorRectFromEvent(event)
    openMenu('screenshot', left, top)
  }
}

const toggleCallMenu = (event: MouseEvent) => {
  if (showCallMenu.value) closeMenu()
  else {
    const { left, top } = anchorRectFromEvent(event)
    openMenu('call', left, top)
  }
}

const showScreenshotTooltip = () => {
  showScreenshotShortcutTooltip.value = true
  void loadScreenshotShortcut()
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
  'open-code-block': []
  'take-screenshot': []
  'take-screenshot-hidden': []
  'open-message-manager': []
  'open-mini-app-list': []
  'toggle-ai-actions': []
}>()

// 点击外部关闭由 UniversalContextMenu 内部处理

onMounted(() => {
  void loadScreenshotShortcut()
  window.electron?.ipcRenderer?.on?.('shortcuts-updated', handleShortcutsUpdated)
})

onUnmounted(() => {
  window.electron?.ipcRenderer?.removeListener?.('shortcuts-updated', handleShortcutsUpdated)
})
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

.screenshot-shortcut-tooltip {
  position: absolute;
  left: 50%;
  bottom: calc(100% + 8px);
  z-index: 1200;
  transform: translateX(-50%);
  padding: 6px 9px;
  border-radius: 6px;
  background: var(--tooltip-bg, rgba(32, 32, 32, 0.92));
  color: #fff;
  font-size: 12px;
  line-height: 1;
  white-space: nowrap;
  pointer-events: none;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.18);
}

.screenshot-shortcut-tooltip::after {
  content: '';
  position: absolute;
  left: 50%;
  top: 100%;
  transform: translateX(-50%);
  border: 5px solid transparent;
  border-top-color: var(--tooltip-bg, rgba(32, 32, 32, 0.92));
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

.call-dropdown {
  position: relative;
  display: inline-flex;
  align-items: center;
}

.call-btn {
  border-radius: 4px 0 0 4px !important;
}

.call-dropdown-trigger {
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

.call-dropdown-trigger:hover {
  background: var(--hover-bg, rgba(0, 0, 0, 0.05));
  color: var(--text-primary, #333);
}

.call-menu {
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

.call-menu-item {
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

.call-menu-item:hover {
  background: var(--hover-bg, rgba(0, 0, 0, 0.05));
}

.call-menu-item i {
  width: 16px;
  font-size: 13px;
  color: var(--text-secondary, #666);
}

.message-manager-btn {
  margin-left: auto;
}
</style>
