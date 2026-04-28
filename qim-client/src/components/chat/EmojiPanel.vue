<template>
  <div class="emoji-panel">
    <div class="emoji-content">
      <div class="emoji-section">
        <div
          v-for="emoji in allEmojis"
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
interface Emits {
  (e: 'select', emoji: string): void
}

const emit = defineEmits<Emits>()

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
  font-size: 13px;
  font-weight: 600;
  color: var(--text-color);
}

.emoji-count-text {
  font-size: 11px;
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
  font-size: 20px;
  transition: all 0.15s ease;
  background: transparent;
  user-select: none;
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
