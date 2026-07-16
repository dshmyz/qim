<template>
  <div class="message-bubble merged-forward-message" :class="{ self: isSelf }">
    <template v-if="payload">
      <div class="merged-forward-header">
        <span>聊天记录（{{ payload?.messages.length ?? 0 }}条）</span>
        <button data-testid="merged-forward-toggle" type="button" @click="expanded = !expanded">
          {{ expanded ? '收起' : '展开' }}
        </button>
      </div>
      <div v-if="expanded" class="merged-forward-list">
        <div v-for="message in payload.messages" :key="message.id" class="merged-forward-item">
          <strong>{{ message.senderName }}</strong>
          <span>{{ messagePreview(message) }}</span>
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
  width: 300px;
  max-width: 100%;
  padding: 12px;
  border: 1px solid var(--border-color);
  border-radius: 12px;
  background: var(--card-bg);
  color: var(--text-color);
}

.merged-forward-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-weight: 600;
}

.merged-forward-header button {
  border: 0;
  padding: 4px 8px;
  color: var(--primary-color);
  background: transparent;
  cursor: pointer;
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

.merged-forward-item span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-secondary);
}
</style>
