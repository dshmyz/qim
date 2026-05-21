<template>
  <Teleport to="body">
    <div v-if="visible" class="ai-translate-overlay" @click.self="close">
      <div class="ai-translate-panel">
        <div class="panel-header">
          <h3>翻译结果</h3>
          <button class="close-btn" @click="close">&times;</button>
        </div>

        <div v-if="isTranslating" class="translating-state">
          <div class="translating-spinner"></div>
          <p>正在翻译...</p>
          <p class="translating-hint">这可能需要几秒钟</p>
        </div>

        <div v-else-if="translatedText" class="translate-content">
          <div v-if="messageType === 'image'" class="image-preview">
            <img :src="originalText" alt="待翻译图片" />
          </div>
          <div v-else class="original-section">
            <div class="section-label">原文</div>
            <div class="section-text">{{ originalText }}</div>
          </div>
          <div class="divider"></div>
          <div class="translated-section">
            <div class="section-label">译文</div>
            <div class="section-text">{{ translatedText }}</div>
          </div>
          <div class="translate-actions">
            <button @click="copyTranslation">复制译文</button>
          </div>
        </div>

        <div v-else class="error-state">
          <p>翻译失败</p>
          <button @click="translate">重试</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useAIActions } from '../../composables/useAIActions'
import QMessage from '../../utils/qmessage'

const props = defineProps<{
  visible: boolean
  originalText: string
  messageType?: string
  targetLang?: string
}>()

const emit = defineEmits<{
  close: []
}>()

const { translateText, translateImage, isProcessing: isTranslating } = useAIActions()
const translatedText = ref<string | null>(null)

watch(() => props.visible, async (newVal) => {
  if (newVal && props.originalText) {
    await translate()
  }
})

const translate = async () => {
  translatedText.value = null
  try {
    if (props.messageType === 'image') {
      translatedText.value = await translateImage(
        props.originalText,
        props.targetLang || 'zh'
      )
    } else {
      translatedText.value = await translateText(
        props.originalText,
        props.targetLang || 'zh'
      )
    }
  } catch {
  }
}

const close = () => {
  emit('close')
}

const copyTranslation = async () => {
  if (translatedText.value) {
    await navigator.clipboard.writeText(translatedText.value)
    QMessage.success('已复制译文')
  }
}
</script>

<style scoped>
.ai-translate-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.ai-translate-panel {
  background: var(--card-bg);
  border-radius: 12px;
  width: 90%;
  max-width: 600px;
  max-height: 85vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
}

.panel-header h3 {
  margin: 0;
  font-size: 18px;
}

.close-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  font-size: 24px;
  cursor: pointer;
  color: var(--text-secondary);
  border-radius: 6px;
  transition: background 0.2s;
}

.close-btn:hover {
  background: var(--hover-color);
}

.translating-state {
  padding: 60px 20px;
  text-align: center;
  color: var(--text-secondary);
}

.translating-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid var(--border-color);
  border-top-color: var(--primary-color);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto 16px;
}

.translating-hint {
  font-size: 12px;
  margin-top: 8px;
  opacity: 0.7;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.translate-content {
  padding: 20px 24px;
  overflow-y: auto;
}

.image-preview {
  margin-bottom: 16px;
  text-align: center;
}

.image-preview img {
  max-width: 100%;
  max-height: 300px;
  object-fit: contain;
  border-radius: 8px;
}

.section-label {
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 8px;
  font-weight: 500;
}

.section-text {
  line-height: 1.7;
  color: var(--text-color);
  word-break: break-word;
}

.original-section {
  margin-bottom: 0;
}

.divider {
  height: 1px;
  background: var(--border-color);
  margin: 16px 0;
}

.translated-section {
  margin-bottom: 16px;
}

.translate-actions {
  display: flex;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
}

.translate-actions button {
  padding: 8px 16px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: transparent;
  color: var(--text-primary);
  cursor: pointer;
  transition: all 0.2s;
}

.translate-actions button:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.error-state {
  padding: 40px 20px;
  text-align: center;
  color: var(--text-secondary);
}

.error-state p {
  margin-bottom: 16px;
}

.error-state button {
  padding: 8px 20px;
  border: none;
  border-radius: 6px;
  background: var(--primary-color);
  color: white;
  cursor: pointer;
}

.error-state button:hover {
  opacity: 0.9;
}
</style>