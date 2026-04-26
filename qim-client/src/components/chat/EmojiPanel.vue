<template>
  <div class="emoji-panel">
    <div class="emoji-tabs">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        :class="['emoji-tab-btn', { active: activeTab === tab.key }]"
        @click="activeTab = tab.key"
      >
        {{ tab.label }}
      </button>
    </div>

    <div class="emoji-content">
      <div class="emoji-section" v-if="activeTab === 'common'">
        <div
          v-for="emoji in commonEmojis"
          :key="emoji"
          class="emoji-item"
          @click="selectEmoji(emoji)"
        >
          {{ emoji }}
        </div>
      </div>

      <div class="emoji-section" v-if="activeTab === 'face'">
        <div
          v-for="emoji in faceEmojis"
          :key="emoji"
          class="emoji-item"
          @click="selectEmoji(emoji)"
        >
          {{ emoji }}
        </div>
      </div>

      <div class="emoji-section" v-if="activeTab === 'animal'">
        <div
          v-for="emoji in animalEmojis"
          :key="emoji"
          class="emoji-item"
          @click="selectEmoji(emoji)"
        >
          {{ emoji }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

interface Emits {
  (e: 'select', emoji: string): void
}

const emit = defineEmits<Emits>()

const activeTab = ref('common')

const tabs = [
  { key: 'common', label: '常用' },
  { key: 'face', label: '表情' },
  { key: 'animal', label: '动物' }
]

const commonEmojis = ['😊', '😂', '❤️', '👍', '🎉', '🔥', '🤔', '😢', '😡', '👏']

const faceEmojis = [
  '😀', '😃', '😄', '😁', '😆', '😅', '😂', '🤣', '😊', '😇',
  '🙂', '🙃', '😉', '😌', '😍', '🥰', '😘', '😗', '😙', '😚',
  '😋', '😛', '😝', '😜', '🤪', '🤨', '🧐', '🤓', '😎', '🤩',
  '🥳', '😏', '😒', '😞', '😔', '😟', '😕', '🙁', '☹️', '😣',
  '😖', '😫', '😩', '🥺', '😢', '😭', '😤', '😠', '😡', '🤬',
  '🤯', '😳', '🥵', '🥶', '😱', '😨', '😰', '😥', '😓', '🤗',
  '🤔', '🤭', '🤫', '🤥', '😶', '😐', '😑', '😬', '🙄', '😯',
  '😦', '😧', '😮', '😲', '🥱', '😴', '🤤', '😪', '😵', '🤐',
  '🥴', '🤢', '🤮', '🤧', '🥵', '🤒', '🤕', '🤠'
]

const animalEmojis = [
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
</script>

<style scoped>
.emoji-panel {
  position: absolute;
  bottom: 100%;
  left: 0;
  width: 320px;
  max-height: 400px;
  background: var(--sidebar-bg);
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  display: flex;
  flex-direction: column;
  z-index: 100;
  overflow: hidden;
}

.emoji-tabs {
  display: flex;
  border-bottom: 1px solid var(--border-color);
  background: var(--sidebar-bg);
}

.emoji-tab-btn {
  flex: 1;
  padding: 10px;
  border: none;
  background: transparent;
  color: var(--text-color);
  cursor: pointer;
  font-size: 13px;
  transition: all 0.2s ease;
}

.emoji-tab-btn:hover {
  background: var(--hover-color);
}

.emoji-tab-btn.active {
  color: var(--primary-color);
  border-bottom: 2px solid var(--primary-color);
  background: var(--primary-light);
}

.emoji-content {
  padding: 12px;
  overflow-y: auto;
  max-height: 350px;
}

.emoji-section {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 8px;
}

.emoji-item {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  cursor: pointer;
  font-size: 20px;
  transition: all 0.2s ease;
}

.emoji-item:hover {
  background: var(--hover-color);
  transform: scale(1.1);
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
