<template>
  <QDialog v-model:visible="visible" :title="isEdit ? '编辑配置' : '添加配置'" width="500px" @close="handleClose">
    <div class="config-form">
      <div class="form-group">
        <label>配置名称</label>
        <input v-model="form.config_name" placeholder="例如：我的GPT-4">
      </div>
      <div class="form-group">
        <label>提供商</label>
        <select v-model="form.provider" @change="onProviderChange">
          <option v-for="p in providers" :key="p.id" :value="p.id">
            {{ p.icon }} {{ p.name }}
          </option>
        </select>
      </div>
      <div class="form-group">
        <label>API Key</label>
        <div class="input-wrapper">
          <input v-model="form.api_key" :type="showKey ? 'text' : 'password'" placeholder="sk-...">
          <button class="toggle-btn" @click="showKey = !showKey" type="button">
            <i :class="showKey ? 'fas fa-eye-slash' : 'fas fa-eye'"></i>
          </button>
        </div>
      </div>
      <div class="form-group">
        <label>模型名称</label>
        <input v-model="form.model_name" placeholder="gpt-3.5-turbo">
      </div>
      <div class="form-group">
        <label>Base URL（可选）</label>
        <input v-model="form.base_url" placeholder="https://api.openai.com/v1">
      </div>
      <div v-if="error" class="error-message">
        <i class="fas fa-exclamation-circle"></i>
        {{ error }}
      </div>
    </div>

    <div class="form-footer">
      <button class="btn-cancel" @click="handleClose">取消</button>
      <button class="btn-primary" @click="handleSubmit" :disabled="loading">
        {{ loading ? '保存中...' : '保存' }}
      </button>
    </div>
  </QDialog>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import QDialog from '../../shared/QDialog.vue'
import { AI_PROVIDERS, type UserAIConfig, type CreateConfigRequest } from '../../../types/ai'

const props = defineProps<{
  modelValue: boolean
  config?: UserAIConfig | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'close'): void
  (e: 'save', config: CreateConfigRequest): void
}>()

const providers = AI_PROVIDERS
const showKey = ref(false)
const loading = ref(false)
const error = ref('')

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const form = ref<CreateConfigRequest>({
  config_name: '',
  provider: 'openai',
  api_key: '',
  model_name: 'gpt-3.5-turbo',
  base_url: 'https://api.openai.com/v1'
})

const isEdit = computed(() => !!props.config)

watch(() => props.config, (newConfig) => {
  if (newConfig) {
    form.value = {
      config_name: newConfig.config_name,
      provider: newConfig.provider,
      api_key: '',
      model_name: newConfig.model_name,
      base_url: newConfig.base_url
    }
  } else {
    form.value = {
      config_name: '',
      provider: 'openai',
      api_key: '',
      model_name: 'gpt-3.5-turbo',
      base_url: 'https://api.openai.com/v1'
    }
  }
}, { immediate: true })

function onProviderChange() {
  const provider = providers.find(p => p.id === form.value.provider)
  if (provider) {
    form.value.model_name = provider.defaultModel
    form.value.base_url = provider.defaultBaseURL
  }
}

function handleClose() {
  error.value = ''
  visible.value = false
}

async function handleSubmit() {
  if (!form.value.config_name.trim()) {
    error.value = '请输入配置名称'
    return
  }
  if (!form.value.api_key.trim() && !isEdit.value) {
    error.value = '请输入API Key'
    return
  }

  error.value = ''
  loading.value = true
  emit('save', { ...form.value })
  loading.value = false
}
</script>

<style scoped>
.config-form {
  padding: 4px 0;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}

.form-group input,
.form-group select {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-color);
  color: var(--text-primary);
  box-sizing: border-box;
  font-size: 14px;
}

.form-group input:focus,
.form-group select:focus {
  outline: none;
  border-color: var(--primary-color);
}

.input-wrapper {
  position: relative;
}

.toggle-btn {
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text-secondary);
  font-size: 14px;
}

.error-message {
  color: #d32f2f;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
}

.form-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
}

.form-footer button {
  padding: 10px 24px;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-cancel {
  background: var(--bg-color, #fff);
  color: var(--text-secondary, #666);
  border: 1px solid var(--border-color, #ddd) !important;
}

.btn-cancel:hover {
  background: var(--hover-color, #f5f5f5);
  color: var(--text-primary, #333);
  border-color: var(--primary-color, #409eff) !important;
}

.btn-primary {
  background: var(--primary-color, #409eff);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: var(--active-color, #66b1ff);
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.3);
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
