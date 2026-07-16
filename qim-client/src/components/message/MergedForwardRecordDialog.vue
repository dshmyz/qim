<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="merged-forward-record-backdrop"
      data-testid="merged-forward-record-dialog"
      role="presentation"
      @click.self="close"
    >
      <section
        v-if="payload"
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
            <div
              v-if="showTimestampDivider(message.timestamp, index)"
              class="merged-forward-record-time"
              data-testid="merged-forward-time-divider"
            >
              {{ formatTimestamp(message.timestamp) }}
            </div>
            <article class="merged-forward-record-item">
              <strong class="merged-forward-record-sender">{{ message.senderName }}</strong>
              <div class="merged-forward-record-content">
                <TextMessage v-if="message.type === 'text'" :content="message.content" />
                <MarkdownMessage v-else-if="message.type === 'markdown'" :content="message.content" />
                <ImageMessage v-else-if="message.type === 'image'" :src="message.content" :server-url="serverUrl" />
                <FileMessage
                  v-else-if="message.type === 'file'"
                  :content="message.content"
                  :message-id="message.id"
                  :server-url="serverUrl"
                  @download="(content, messageId) => emit('download', content, messageId)"
                  @save-as="(content, messageId) => emit('saveAs', content, messageId)"
                />
                <ShareMessage
                  v-else-if="message.type === 'share'"
                  :content="message.content"
                  :share-data="recordMessageData(message)"
                />
                <MiniAppMessage
                  v-else-if="message.type === 'miniApp' || message.type === 'mini-app'"
                  :mini-app-data="recordMessageData(message)"
                />
                <NewsMessage v-else-if="message.type === 'news'" :news-data="recordMessageData(message)" />
                <p v-else class="merged-forward-record-summary">{{ resolveMessageDisplay(message).label }}</p>
              </div>
            </article>
          </template>
        </div>
      </section>
      <section v-else class="merged-forward-record-dialog merged-forward-record-fallback" role="alert">
        聊天记录无法加载
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import { type MergedForwardPayload } from '@/utils/mergedForward'
import { resolveMessageDisplay } from '@/utils/messageDisplay'
import { useServerUrl } from '@/composables/useServerUrl'
import TextMessage from './TextMessage.vue'
import MarkdownMessage from './MarkdownMessage.vue'
import ImageMessage from './ImageMessage.vue'
import FileMessage from './FileMessage.vue'
import ShareMessage from './ShareMessage.vue'
import MiniAppMessage from './MiniAppMessage.vue'
import NewsMessage from './NewsMessage.vue'

const props = defineProps<{
  payload: MergedForwardPayload | null
  visible: boolean
}>()

const emit = defineEmits<{
  close: []
  download: [content: string, messageId?: string]
  saveAs: [content: string, messageId?: string]
}>()

const { serverUrl } = useServerUrl()

const close = () => emit('close')

const handleKeydown = (event: KeyboardEvent) => {
  if (props.visible && event.key === 'Escape') close()
}

const showTimestampDivider = (timestamp: number, index: number): boolean => index > 0
  && timestamp - props.payload!.messages[index - 1].timestamp > 300_000

const formatTimestamp = (timestamp: number): string => new Date(timestamp).toLocaleString()

const recordMessageData = (message: { type: string; content: string }): Record<string, any> | undefined => {
  const data = resolveMessageDisplay(message).data
  return data && !('messages' in data) ? data : undefined
}

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
  background: var(--modal-overlay, rgb(0 0 0 / 50%));
}

.merged-forward-record-dialog {
  display: flex;
  width: min(720px, calc(100vw - 32px));
  max-height: min(720px, calc(100vh - 40px));
  min-height: 0;
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
  min-height: 0;
  flex: 1;
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

.merged-forward-record-content {
  min-width: 0;
}

.merged-forward-record-summary {
  margin: 0;
  white-space: pre-wrap;
  color: var(--text-color);
}

.merged-forward-record-content :deep(.text-message){
  padding: 5px 0px;
}

.merged-forward-record-content :deep(.message-bubble),
.merged-forward-record-content :deep(.markdown-message),
.merged-forward-record-content :deep(.message-content-image) {
  max-width: 100%;
}

.merged-forward-record-time {
  margin: 5px 0 5px;
  text-align: center;
  font-size: 12px;
  color: var(--text-secondary);
}

.merged-forward-record-fallback {
  padding: 24px;
  color: var(--text-secondary);
  text-align: center;
}
</style>
