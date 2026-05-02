<template>
  <div class="quick-generate-section">
    <div class="generate-header">
      <h3>快速生成</h3>
      <div class="generate-actions">
        <button class="action-btn" @click="$emit('batch')">批量生成</button>
        <button class="action-btn" @click="$emit('advanced')">高级选项</button>
      </div>
    </div>

    <div class="generate-input-group">
      <input
        ref="inputRef"
        v-model="urlInput"
        type="text"
        class="url-input"
        placeholder="粘贴URL,按Enter快速生成..."
        @keydown.enter="handleGenerate"
      />
      <div class="input-hint">Cmd + V</div>
      <button class="generate-btn" @click="handleGenerate" :disabled="!urlInput.trim() || isGenerating">
        {{ isGenerating ? '生成中...' : '生成短链接' }}
      </button>
    </div>

    <div v-if="generatedUrl" class="generate-result">
      <div class="result-info">
        <div class="result-label">生成的短链接</div>
        <div class="result-url">{{ generatedUrl }}</div>
      </div>
      <button class="copy-btn" @click="handleCopy" :disabled="isCopying">
        {{ isCopying ? '已复制' : '复制' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  generatedUrl?: string
  isGenerating?: boolean
  isCopying?: boolean
}>()

const emit = defineEmits<{
  generate: [url: string]
  copy: []
  batch: []
  advanced: []
}>()

const urlInput = ref('')
const inputRef = ref<HTMLInputElement | null>(null)

const handleGenerate = () => {
  const url = urlInput.value.trim()
  if (url) {
    emit('generate', url)
  }
}

const handleCopy = () => {
  emit('copy')
}

defineExpose({
  focus: () => inputRef.value?.focus(),
  clear: () => {
    urlInput.value = ''
  },
  getCurrentUrl: () => urlInput.value.trim()
})
</script>

<style scoped>
.quick-generate-section {
  background: linear-gradient(135deg, var(--accent-gradient-start, #667eea), var(--accent-gradient-end, #764ba2));
  border-radius: 16px;
  padding: 24px;
  margin-bottom: 24px;
  box-shadow: 0 10px 30px var(--accent-shadow, rgba(102, 126, 234, 0.2));
}

.generate-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.generate-header h3 {
  margin: 0;
  color: white;
  font-size: 18px;
  font-weight: 600;
}

.generate-actions {
  display: flex;
  gap: 8px;
}

.action-btn {
  padding: 6px 12px;
  background: rgba(255, 255, 255, 0.2);
  color: white;
  border: 1px solid rgba(255, 255, 255, 0.3);
  border-radius: 8px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.action-btn:hover {
  background: rgba(255, 255, 255, 0.3);
}

.generate-input-group {
  display: flex;
  gap: 12px;
  align-items: stretch;
  position: relative;
}

.url-input {
  flex: 1;
  padding: 14px 16px;
  padding-right: 60px;
  border: none;
  border-radius: 12px;
  font-size: 14px;
  background: white;
  color: var(--text-primary, #1f2937);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.url-input:focus {
  outline: none;
  box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.3);
}

.input-hint {
  position: absolute;
  right: 140px;
  top: 50%;
  transform: translateY(-50%);
  color: #9ca3af;
  font-size: 12px;
  pointer-events: none;
}

.generate-btn {
  padding: 14px 32px;
  background: white;
  color: var(--accent-color, #667eea);
  border: none;
  border-radius: 12px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  transition: all 0.2s;
}

.generate-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.2);
}

.generate-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.generate-result {
  margin-top: 16px;
  background: rgba(255, 255, 255, 0.15);
  border-radius: 12px;
  padding: 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.result-info {
  flex: 1;
}

.result-label {
  color: rgba(255, 255, 255, 0.7);
  font-size: 12px;
  margin-bottom: 4px;
}

.result-url {
  color: white;
  font-size: 16px;
  font-weight: 600;
}

.copy-btn {
  padding: 8px 20px;
  background: white;
  color: var(--accent-color, #667eea);
  border: none;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.copy-btn:hover:not(:disabled) {
  transform: translateY(-1px);
}

.copy-btn:disabled {
  opacity: 0.7;
}
</style>
