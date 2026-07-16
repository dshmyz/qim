<template>
  <div class="message-bubble merged-forward-message" :class="{ self: isSelf }">
    <template v-if="payload">
      <div class="merged-forward-header">
        <span class="merged-forward-icon" aria-hidden="true"><i class="fas fa-comments"></i></span>
        <span class="merged-forward-title">聊天记录（{{ payload.messages.length }}条）</span>
        <button data-testid="merged-forward-open" type="button" @click="isRecordVisible = true">
          查看聊天记录
          <i class="fas fa-chevron-right" aria-hidden="true"></i>
        </button>
      </div>
      <div class="merged-forward-list">
        <div v-for="message in payload.messages.slice(0, 3)" :key="message.id" class="merged-forward-item">
          <strong>{{ message.senderName }}</strong>
          <span class="merged-forward-preview">
            <span>{{ getMessagePreview(message).label }}</span>
          </span>
        </div>
        <span v-if="payload.messages.length > 3" class="merged-forward-more">还有 {{ payload.messages.length - 3 }} 条消息</span>
      </div>
      <MergedForwardRecordDialog
        :payload="payload"
        :visible="isRecordVisible"
        @close="isRecordVisible = false"
        @download="(content, messageId) => emit('download', content, messageId)"
        @save-as="(content, messageId) => emit('saveAs', content, messageId)"
      />
    </template>
    <span v-else>聊天记录无法加载</span>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import MergedForwardRecordDialog from './MergedForwardRecordDialog.vue'
import { parseMergedForwardPayload } from '@/utils/mergedForward'
import { getMessagePreview } from '@/utils/messagePreview'

const props = defineProps<{
  content: string
  isSelf?: boolean
}>()

const emit = defineEmits<{
  download: [content: string, messageId?: string]
  saveAs: [content: string, messageId?: string]
}>()

const isRecordVisible = ref(false)
const payload = computed(() => parseMergedForwardPayload(props.content))

</script>

<style scoped>
.merged-forward-message {
  width: fit-content;
  max-width: 100%;
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
  color: var(--text-secondary);
}

.merged-forward-preview > span {
  overflow: hidden;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.merged-forward-more {
  color: var(--text-secondary);
  font-size: 13px;
}

@media (max-width: 640px) {
  .merged-forward-message {
    padding: 10px;
    border-radius: 8px;
  }
}
</style>
