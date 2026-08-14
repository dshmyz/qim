<template>
  <div class="emoji-panel">
    <div class="emoji-content">
      <div v-if="activeTab === 'unicode'" class="emoji-section">
        <div
          v-for="emoji in allEmojis"
          :key="emoji"
          class="emoji-item"
          @click="selectEmoji(emoji)"
        >
          <img
            class="emoji-item-img"
            :src="emojiUrl(emoji)"
            :alt="emoji"
            draggable="false"
          />
        </div>
      </div>
      <div v-else class="emoji-section">
        <div
          v-for="c in classicEmojis"
          :key="c.id"
          class="emoji-item"
          :title="c.name"
          @click="selectClassic(c.name)"
        >
          <img
            class="emoji-item-img classic-item-img"
            :src="classicUrl(c.id)"
            :alt="c.name"
            draggable="false"
          />
        </div>
      </div>
    </div>
    <div class="emoji-tabs">
      <button
        class="emoji-tab"
        :class="{ active: activeTab === 'unicode' }"
        @click="activeTab = 'unicode'"
      >默认表情</button>
      <button
        class="emoji-tab"
        :class="{ active: activeTab === 'classic' }"
        @click="activeTab = 'classic'"
      >经典表情</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { emojiUrl } from '../../utils/emoji'
import { CLASSIC_EMOJIS, classicUrl, classicMarker } from '../../utils/classic-emoji'

interface Emits {
  (e: 'select', text: string): void
}

const emit = defineEmits<Emits>()

const activeTab = ref<'unicode' | 'classic'>('unicode')
const classicEmojis = CLASSIC_EMOJIS

const allEmojis = [
  '😊', '😂', '❤️', '👍', '🎉', '🔥', '🤔', '😢', '😡', '👏',
  '😀', '😃', '😄', '😁', '😆', '😅', '😂', '🤣', '😊', '😇',
  '🙂', '🙃', '😉', '😌', '😍', '🥰', '😘', '😗', '😙', '😚',
  '😋', '😛', '😝', '😜', '🤪', '🤨', '🧐', '🤓', '😎', '🤩',
  '🥳', '😏', '😒', '😞', '😔', '😟', '😕', '🙁', '☹️', '😣',
  '😖', '😫', '😩', '🥺', '😢', '😭', '😤', '😠', '😡', '🤬',
  '🤯', '😳', '🥵', '🥶', '😱', '😨', '😰', '😥', '😓', '🤗',
  '🤔', '🤭', '🤫', '🤥', '😶', '😐', '😑', '😬', '🙄', '😯',
  '😦', '😧', '😮', '😲', '🥱', '😴', '🤤', '😪', '😵', '🤐',
  '🥴', '🤢', '🤮', '🤧', '🥵', '🤒', '🤕', '🤠',
  '🐶', '🐱', '🐭', '🐹', '🐰', '🦊', '🐻', '🐼', '🐨', '🐯',
  '🦁', '🐮', '🐷', '🐸', '🐵', '🐔', '🐧', '🐦', '🐤', '🐣',
  '🐥', '🦆', '🦅', '🦉', '🦇', '🐺', '🐗', '🐴', '🦄', '🐝',
  '🐛', '🦋', '🐌', '🐞', '🐜', '🕷️', '🦂', '🐢', '🐍', '🦎',
  '🦖', '🦕', '🐙', '🦑', '🦐', '🦞', '🦀', '🐡', '🐠', '🐟',
  '🐬', '🐳', '🐋', '🐊', '🐅', '🐆', '🐈', '🐩'
]

const selectEmoji = (emoji: string) => {
  emit('select', emoji)
}

const selectClassic = (name: string) => {
  emit('select', classicMarker(name))
}
</script>

<style scoped>
.emoji-panel {
  position: absolute;
  bottom: 100%;
  left: 0;
  width: 440px;
  height: 360px;
  background: var(--sidebar-bg);
  border-radius: 14px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
  display: flex;
  flex-direction: column;
  z-index: 100;
  overflow: hidden;
  border: 1px solid var(--border-color);
}

.emoji-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border-color);
}

.emoji-header-title {
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--text-color);
}

.emoji-count-text {
  font-size: var(--font-size-xxxs);
  color: var(--text-color);
  opacity: 0.4;
}

.emoji-content {
  padding: 12px;
  overflow-y: auto;
  flex: 1;
  min-height: 0;
}

.emoji-section {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(34px, 1fr));
  gap: 4px;
}

.emoji-item {
  display: flex;
  align-items: center;
  justify-content: center;
  aspect-ratio: 1;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;
  background: transparent;
  user-select: none;
}

.emoji-item-img {
  width: 20px;
  height: 20px;
  pointer-events: none;
}

.classic-item-img {
  width: 26px;
  height: 26px;
}

.emoji-tabs {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 10px;
  border-top: 1px solid var(--border-color);
  flex-shrink: 0;
}

.emoji-tab {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 30px;
  padding: 0 14px;
  border: none;
  border-radius: 8px;
  background: transparent;
  cursor: pointer;
  font-size: var(--font-size-xs);
  color: var(--text-color);
  line-height: 1;
  transition: background 0.15s ease;
}

.emoji-tab:hover {
  background: var(--hover-color);
}

.emoji-tab.active {
  background: color-mix(in srgb, var(--primary-color), transparent 88%);
  color: var(--primary-color);
  font-weight: 600;
}

.emoji-item:hover {
  background: var(--hover-color);
  transform: scale(1.12);
}

.emoji-item:active {
  transform: scale(0.95);
}

.emoji-content::-webkit-scrollbar {
  width: 6px;
}

.emoji-content::-webkit-scrollbar-track {
  background: transparent;
}

.emoji-content::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 3px;
}

.emoji-content::-webkit-scrollbar-thumb:hover {
  background: var(--text-color);
  opacity: 0.5;
}
</style>
