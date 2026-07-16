<template>
  <div class="message-bubble merged-forward-message" :class="{ self: isSelf }">
    <template v-if="payload">
      <div class="merged-forward-header">
        <span class="merged-forward-icon" aria-hidden="true"><i class="fas fa-comments"></i></span>
        <span class="merged-forward-title">聊天记录（{{ payload.messages.length }}条）</span>
        <button :aria-expanded="expanded" data-testid="merged-forward-toggle" type="button" @click="expanded = !expanded">
          {{ expanded ? '收起' : '展开' }}
          <i :class="expanded ? 'fas fa-chevron-up' : 'fas fa-chevron-down'" aria-hidden="true"></i>
        </button>
      </div>
      <div v-if="expanded" class="merged-forward-list">
        <div v-for="message in payload.messages" :key="message.id" class="merged-forward-item">
          <strong>{{ message.senderName }}</strong>
          <span class="merged-forward-preview">
            <i v-if="message.type === 'image'" class="fas fa-image" aria-hidden="true"></i>
            <i v-else-if="message.type === 'file'" class="fas fa-file" aria-hidden="true"></i>
            <span>{{ messagePreview(message) }}</span>
          </span>
        </div>
      </div>
    </template>
    <span v-else>聊天记录无法加载</span>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { parseMergedForwardPayload, type MergedForwardItem } from '@/utils/mergedForward'

const props = defineProps<{
  content: string
  isSelf?: boolean
}>()

const expanded = ref(false)
const payload = computed(() => parseMergedForwardPayload(props.content))

const fileName = (content: string): string => {
  try {
    const value = JSON.parse(content)
    if (value?.name || value?.fileName) return value.name || value.fileName
  } catch {
    // File content may be a URL rather than JSON metadata.
  }
  return content.split('/').pop() || content
}

const messagePreview = (message: MergedForwardItem): string => {
  if (message.type === 'image') return '[图片]'
  if (message.type === 'file') return fileName(message.content)
  return message.content
}
</script>

<style scoped>
.merged-forward-message {
  width: min(360px, 100%);
  padding: 14px;
  border: 1px solid var(--border-color);
  border-radius: 12px;
  background: var(--card-bg);
  color: var(--text-color);
  box-shadow: 0 4px 14px rgb(0 0 0 / 8%);
}

.merged-forward-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.merged-forward-icon {
  color: var(--primary-color);
}

.merged-forward-title {
  min-width: 0;
  flex: 1;
  font-weight: 600;
}

.merged-forward-header button {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border: 0;
  border-radius: 6px;
  padding: 4px 8px;
  color: var(--primary-color);
  background: transparent;
  cursor: pointer;
}

.merged-forward-header button:focus-visible {
  outline: 2px solid var(--primary-color);
  outline-offset: 2px;
}

.merged-forward-list {
  display: grid;
  gap: 8px;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--border-color);
}

.merged-forward-item {
  display: grid;
  gap: 2px;
  min-width: 0;
  font-size: 13px;
  line-height: 1.4;
}

.merged-forward-preview {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  color: var(--text-secondary);
}

.merged-forward-preview > span {
  overflow: hidden;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

@media (max-width: 640px) {
  .merged-forward-message {
    padding: 10px;
    border-radius: 8px;
  }
}
</style>
