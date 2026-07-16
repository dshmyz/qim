<template>
  <div
    v-if="visible && payload"
    class="merged-forward-record-backdrop"
    data-testid="merged-forward-record-dialog"
    role="presentation"
    @click.self="close"
  >
    <section
      aria-modal="true"
      aria-labelledby="merged-forward-record-title"
      class="merged-forward-record-dialog"
      role="dialog"
    >
      <header class="merged-forward-record-header">
        <h2 id="merged-forward-record-title">聊天记录（共 {{ payload.messages.length }} 条）</h2>
        <button aria-label="关闭聊天记录" class="merged-forward-record-close" type="button" @click="close">
          <i aria-hidden="true" class="fas fa-xmark"></i>
        </button>
      </header>
      <div class="merged-forward-record-list">
        <template v-for="(message, index) in payload.messages" :key="message.id">
          <div v-if="showTimestampDivider(message.timestamp, index)" class="merged-forward-record-time">
            {{ formatTimestamp(message.timestamp) }}
          </div>
          <article class="merged-forward-record-item">
            <strong>{{ message.senderName }}</strong>
            <p>{{ getMessagePreview(message).label }}</p>
          </article>
        </template>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import { type MergedForwardPayload } from '@/utils/mergedForward'
import { getMessagePreview } from '@/utils/messagePreview'

const props = defineProps<{
  payload: MergedForwardPayload | null
  visible: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const close = () => emit('close')

const handleKeydown = (event: KeyboardEvent) => {
  if (props.visible && event.key === 'Escape') close()
}

const showTimestampDivider = (timestamp: number, index: number): boolean => index > 0
  && timestamp - props.payload!.messages[index - 1].timestamp > 300_000

const formatTimestamp = (timestamp: number): string => new Date(timestamp).toLocaleString()

onMounted(() => window.addEventListener('keydown', handleKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', handleKeydown))
</script>

<style scoped>
.merged-forward-record-backdrop {
  position: fixed;
  inset: 0;
  z-index: 10000;
  display: grid;
  place-items: center;
  padding: 20px;
  background: rgb(0 0 0 / 50%);
}

.merged-forward-record-dialog {
  display: flex;
  width: min(640px, 100%);
  max-height: min(720px, calc(100vh - 40px));
  flex-direction: column;
  overflow: hidden;
  border-radius: 12px;
  background: var(--panel-bg);
  box-shadow: 0 16px 40px rgb(0 0 0 / 25%);
}

.merged-forward-record-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
}

.merged-forward-record-header h2 {
  margin: 0;
  font-size: 16px;
}

.merged-forward-record-close {
  border: 0;
  padding: 4px 8px;
  color: var(--text-secondary);
  background: transparent;
  cursor: pointer;
}

.merged-forward-record-list {
  overflow-y: auto;
  padding: 16px 20px;
}

.merged-forward-record-item {
  display: grid;
  gap: 4px;
  padding: 10px 0;
}

.merged-forward-record-item strong {
  font-size: 13px;
}

.merged-forward-record-item p {
  margin: 0;
  white-space: pre-wrap;
  color: var(--text-color);
}

.merged-forward-record-time {
  margin: 12px 0 4px;
  text-align: center;
  font-size: 12px;
  color: var(--text-secondary);
}
</style>
