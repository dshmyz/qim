<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h3>{{ isEdit ? '编辑配置' : '添加配置' }}</h3>
        <button class="close-btn" @click="$emit('close')">
          <i class="fas fa-times"></i>
        </button>
      </div>
      <div class="modal-body">
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
      <div class="modal-footer">
        <button class="btn-secondary" @click="$emit('close')">取消</button>
        <button class="btn-primary" @click="handleSubmit" :disabled="loading">
          {{ loading ? '保存中...' : '保存' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { AI_PROVIDERS, type UserAIConfig, type CreateConfigRequest } from '../../../types/ai'

const props = defineProps<{
  config?: UserAIConfig | null
}>()

const emit = defineEmits(['close', 'save'])

const providers = AI_PROVIDERS
const showKey = ref(false)
const loading = ref(false)
const error = ref('')

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
  }
}, { immediate: true })

function onProviderChange() {
  const provider = providers.find(p => p.id === form.value.provider)
  if (provider) {
    form.value.model_name = provider.defaultModel
    form.value.base_url = provider.defaultBaseURL
  }
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
  try {
    await emit('save', { ...form.value })
  } catch (e: any) {
    error.value = e.message || '保存失败'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: var(--card-bg);
  border-radius: 12px;
  width: 90%;
  max-width: 500px;
  max-height: 80vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.modal-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.modal-header h3 {
  margin: 0;
  font-size: 16px;
}

.close-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  cursor: pointer;
  border-radius: 6px;
  color: var(--text-color);
}

.modal-body {
  padding: 20px;
  overflow-y: auto;
  flex: 1;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
}

.form-group input,
.form-group select {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-color);
  color: var(--text-color);
  box-sizing: border-box;
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
}

.error-message {
  color: #d32f2f;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
}

.modal-footer {
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.btn-primary,
.btn-secondary {
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}

.btn-primary {
  background: var(--primary-color);
  color: white;
  border: none;
}

.btn-secondary {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
}
</style>
