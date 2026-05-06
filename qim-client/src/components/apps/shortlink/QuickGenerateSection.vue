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
      <div class="input-wrapper">
        <input
          ref="inputRef"
          v-model="urlInput"
          type="text"
          class="url-input"
          placeholder="粘贴URL,按Enter快速生成..."
          @keydown.enter="handleGenerate"
        />
        <div class="input-hint">Cmd + V</div>
      </div>
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
  background: var(--card-bg, #fff);
  border-radius: 16px;
  padding: 24px;
  margin-bottom: 24px;
  box-shadow: 0 2px 8px var(--shadow-color, rgba(0, 0, 0, 0.1));
  border: 1px solid var(--border-color, #eee);
  box-sizing: border-box;
  width: 100%;
  max-width: 100%;
}

.generate-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
}

.generate-header h3 {
  margin: 0;
  color: var(--text-primary, #1f2937);
  font-size: 18px;
  font-weight: 600;
  flex-shrink: 0;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.generate-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.action-btn {
  padding: 6px 12px;
  background: var(--bg-color, #f5f5f5);
  color: var(--text-secondary, #666);
  border: 1px solid var(--border-color, #ddd);
  border-radius: 8px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}

.action-btn:hover {
  background: var(--hover-color, #e8e8e8);
  border-color: var(--primary-color, #409eff);
  color: var(--primary-color, #409eff);
}

.generate-input-group {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: stretch;
}

.input-wrapper {
  flex: 1;
  min-width: 200px;
  position: relative;
}

.url-input {
  width: 100%;
  padding: 14px 16px;
  padding-right: 80px;
  border: none;
  border-radius: 12px;
  font-size: 14px;
  background: white;
  color: var(--text-primary, #1f2937);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  box-sizing: border-box;
}

.url-input:focus {
  outline: none;
  box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.3);
}

.input-hint {
  position: absolute;
  right: 16px;
  top: 50%;
  transform: translateY(-50%);
  color: #9ca3af;
  font-size: 12px;
  pointer-events: none;
}

.generate-btn {
  padding: 14px 32px;
  background: var(--primary-color, #409eff);
  color: white;
  border: none;
  border-radius: 12px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.2);
  transition: all 0.2s;
}

.generate-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(64, 158, 255, 0.3);
}

.generate-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.generate-result {
  margin-top: 16px;
  background: var(--bg-color, #f9fafb);
  border-radius: 12px;
  padding: 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
  border: 1px solid var(--border-color, #eee);
}

.result-info {
  flex: 1;
  min-width: 0;
  overflow: hidden;
}

.result-label {
  color: var(--text-secondary, #6b7280);
  font-size: 12px;
  margin-bottom: 4px;
}

.result-url {
  color: var(--text-primary, #1f2937);
  font-size: 16px;
  font-weight: 600;
  word-break: break-all;
  overflow-wrap: break-word;
}

.copy-btn {
  padding: 8px 20px;
  background: var(--primary-color, #409eff);
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.copy-btn:hover:not(:disabled) {
  background: var(--active-color, #66b1ff);
  transform: translateY(-1px);
}

.copy-btn:disabled {
  opacity: 0.7;
}

@media (max-width: 768px) {
  .quick-generate-section {
    padding: 20px;
  }

  .generate-header {
    flex-wrap: wrap;
    gap: 12px;
  }

  .generate-actions {
    width: 100%;
    justify-content: flex-end;
  }

  .input-hint {
    display: none;
  }

  .url-input {
    padding-right: 16px;
  }

  .generate-result {
    flex-wrap: wrap;
    gap: 12px;
  }

  .result-info {
    width: 100%;
  }

  .copy-btn {
    width: 100%;
  }
}

@media (max-width: 480px) {
  .quick-generate-section {
    padding: 16px;
  }

  .generate-input-group {
    flex-direction: column;
  }

  .input-wrapper {
    min-width: 0;
  }

  .generate-btn {
    width: 100%;
  }
}
</style>
