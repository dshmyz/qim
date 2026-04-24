<template>
  <div class="ai-config-app">
    <div class="ai-config-header">
      <div class="header-left">
        <button class="back-btn" @click="$emit('back')">
          <i class="fas fa-arrow-left"></i>
        </button>
        <h2>AI 大模型配置</h2>
      </div>
    </div>
    <div class="ai-config-content">
      <div class="config-form">
        <div class="form-section">
          <h3>基础设置</h3>
          <div class="form-group">
            <label>AI 提供商</label>
            <select v-model="config.provider">
              <option value="openai">OpenAI</option>
              <option value="baidu">百度文心一言</option>
              <option value="alibaba">阿里通义千问</option>
              <option value="tencent">腾讯混元大模型</option>
              <option value="bytedance">字节跳动豆包</option>
              <option value="anthropic">Anthropic Claude</option>
            </select>
          </div>
          <div class="form-group">
            <label>最大 Tokens</label>
            <input v-model.number="config.max_tokens" type="number" min="100" max="10000" placeholder="1000">
          </div>
          <div class="form-group">
            <label>温度 (Temperature)</label>
            <input v-model.number="config.temperature" type="number" min="0" max="2" step="0.1" placeholder="0.7">
          </div>
        </div>

        <div class="form-section" v-if="config.provider === 'openai'">
          <h3>OpenAI 设置</h3>
          <div class="form-group">
            <label>API Key</label>
            <input v-model="config.openai_api_key" type="password" placeholder="sk-...">
          </div>
          <div class="form-group">
            <label>模型</label>
            <input v-model="config.openai_model" type="text" placeholder="gpt-3.5-turbo">
          </div>
          <div class="form-group">
            <label>Base URL</label>
            <input v-model="config.openai_base_url" type="text" placeholder="https://api.openai.com/v1">
          </div>
        </div>

        <div class="form-section" v-if="config.provider === 'baidu'">
          <h3>百度文心一言设置</h3>
          <div class="form-group">
            <label>API Key</label>
            <input v-model="config.baidu_api_key" type="text" placeholder="API Key">
          </div>
          <div class="form-group">
            <label>Secret Key</label>
            <input v-model="config.baidu_secret_key" type="password" placeholder="Secret Key">
          </div>
          <div class="form-group">
            <label>模型</label>
            <input v-model="config.baidu_model" type="text" placeholder="ERNIE-Bot-4.0">
          </div>
          <div class="form-group">
            <label>Base URL</label>
            <input v-model="config.baidu_base_url" type="text" placeholder="https://aip.baidubce.com">
          </div>
        </div>

        <div class="form-section" v-if="config.provider === 'alibaba'">
          <h3>阿里通义千问设置</h3>
          <div class="form-group">
            <label>API Key</label>
            <input v-model="config.alibaba_api_key" type="password" placeholder="API Key">
          </div>
          <div class="form-group">
            <label>模型</label>
            <input v-model="config.alibaba_model" type="text" placeholder="qwen-plus">
          </div>
          <div class="form-group">
            <label>Base URL</label>
            <input v-model="config.alibaba_base_url" type="text" placeholder="https://dashscope.aliyuncs.com/api/v1">
          </div>
        </div>

        <div class="form-section" v-if="config.provider === 'tencent'">
          <h3>腾讯混元大模型设置</h3>
          <div class="form-group">
            <label>Secret ID</label>
            <input v-model="config.tencent_secret_id" type="text" placeholder="Secret ID">
          </div>
          <div class="form-group">
            <label>Secret Key</label>
            <input v-model="config.tencent_secret_key" type="password" placeholder="Secret Key">
          </div>
          <div class="form-group">
            <label>模型</label>
            <input v-model="config.tencent_model" type="text" placeholder="hunyuan-pro">
          </div>
          <div class="form-group">
            <label>Base URL</label>
            <input v-model="config.tencent_base_url" type="text" placeholder="https://hunyuan.tencentcloudapi.com">
          </div>
        </div>

        <div class="form-section" v-if="config.provider === 'bytedance'">
          <h3>字节跳动豆包设置</h3>
          <div class="form-group">
            <label>API Key</label>
            <input v-model="config.bytedance_api_key" type="password" placeholder="API Key">
          </div>
          <div class="form-group">
            <label>模型</label>
            <input v-model="config.bytedance_model" type="text" placeholder="doubao-pro-1.0">
          </div>
          <div class="form-group">
            <label>Base URL</label>
            <input v-model="config.bytedance_base_url" type="text" placeholder="https://ark.cn-beijing.volces.com/api/v3">
          </div>
        </div>

        <div class="form-section" v-if="config.provider === 'anthropic'">
          <h3>Anthropic Claude 设置</h3>
          <div class="form-group">
            <label>API Key</label>
            <input v-model="config.anthropic_api_key" type="password" placeholder="sk-ant-...">
          </div>
          <div class="form-group">
            <label>模型</label>
            <input v-model="config.anthropic_model" type="text" placeholder="claude-3-5-sonnet-20241022">
          </div>
          <div class="form-group">
            <label>Base URL</label>
            <input v-model="config.anthropic_base_url" type="text" placeholder="https://api.anthropic.com/v1">
          </div>
        </div>

        <div class="form-actions">
          <button class="btn-secondary" @click="loadConfig">重置</button>
          <button class="btn-primary" @click="saveConfig" :disabled="saving">
            {{ saving ? '保存中...' : '保存配置' }}
          </button>
        </div>
      </div>

      <div v-if="message" :class="['message', message.type]">
        {{ message.text }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { API_BASE_URL } from '../../config'

const emit = defineEmits(['back'])

const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

const config = ref({
  provider: 'openai',
  max_tokens: 1000,
  temperature: 0.7,
  openai_api_key: '',
  openai_model: 'gpt-3.5-turbo',
  openai_base_url: 'https://api.openai.com/v1',
  baidu_api_key: '',
  baidu_secret_key: '',
  baidu_model: 'ERNIE-Bot-4.0',
  baidu_base_url: 'https://aip.baidubce.com',
  alibaba_api_key: '',
  alibaba_model: 'qwen-plus',
  alibaba_base_url: 'https://dashscope.aliyuncs.com/api/v1',
  tencent_secret_id: '',
  tencent_secret_key: '',
  tencent_model: 'hunyuan-pro',
  tencent_base_url: 'https://hunyuan.tencentcloudapi.com',
  bytedance_api_key: '',
  bytedance_model: 'doubao-pro-1.0',
  bytedance_base_url: 'https://ark.cn-beijing.volces.com/api/v3',
  anthropic_api_key: '',
  anthropic_model: 'claude-3-5-sonnet-20241022',
  anthropic_base_url: 'https://api.anthropic.com/v1'
})

const saving = ref(false)
const message = ref<{ type: string; text: string } | null>(null)

const getToken = () => {
  return localStorage.getItem('token')
}

const loadConfig = async () => {
  try {
    const token = getToken()
    const response = await axios.get(`${serverUrl.value}/api/v1/ai/config`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })

    if (response.data.code === 0 && response.data.data) {
      const data = response.data.data
      config.value = {
        provider: data.provider || 'openai',
        max_tokens: data.max_tokens || 1000,
        temperature: data.temperature || 0.7,
        openai_api_key: data.openai_api_key || '',
        openai_model: data.openai_model || 'gpt-3.5-turbo',
        openai_base_url: data.openai_base_url || 'https://api.openai.com/v1',
        baidu_api_key: data.baidu_api_key || '',
        baidu_secret_key: data.baidu_secret_key || '',
        baidu_model: data.baidu_model || 'ERNIE-Bot-4.0',
        baidu_base_url: data.baidu_base_url || 'https://aip.baidubce.com',
        alibaba_api_key: data.alibaba_api_key || '',
        alibaba_model: data.alibaba_model || 'qwen-plus',
        alibaba_base_url: data.alibaba_base_url || 'https://dashscope.aliyuncs.com/api/v1',
        tencent_secret_id: data.tencent_secret_id || '',
        tencent_secret_key: data.tencent_secret_key || '',
        tencent_model: data.tencent_model || 'hunyuan-pro',
        tencent_base_url: data.tencent_base_url || 'https://hunyuan.tencentcloudapi.com',
        bytedance_api_key: data.bytedance_api_key || '',
        bytedance_model: data.bytedance_model || 'doubao-pro-1.0',
        bytedance_base_url: data.bytedance_base_url || 'https://ark.cn-beijing.volces.com/api/v3',
        anthropic_api_key: data.anthropic_api_key || '',
        anthropic_model: data.anthropic_model || 'claude-3-5-sonnet-20241022',
        anthropic_base_url: data.anthropic_base_url || 'https://api.anthropic.com/v1'
      }
    }
  } catch (error) {
    console.error('加载配置失败:', error)
    showMessage('error', '加载配置失败')
  }
}

const saveConfig = async () => {
  saving.value = true
  try {
    const token = getToken()
    const response = await axios.put(`${serverUrl.value}/api/v1/ai/config`, config.value, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })

    if (response.data.code === 0) {
      showMessage('success', '配置保存成功')
    } else {
      showMessage('error', '配置保存失败')
    }
  } catch (error) {
    console.error('保存配置失败:', error)
    showMessage('error', '保存配置失败')
  } finally {
    saving.value = false
  }
}

const showMessage = (type: string, text: string) => {
  message.value = { type, text }
  setTimeout(() => {
    message.value = null
  }, 3000)
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.ai-config-app {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--content-bg);
  overflow: hidden;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.ai-config-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  background-color: var(--card-bg);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  height: 72px;
  box-sizing: border-box;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.back-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: var(--hover-color);
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  transition: background 0.2s;
  color: var(--primary-color);
}

.back-btn:hover {
  background: var(--primary-light);
}

.ai-config-header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
}

.ai-config-content {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
}

.config-form {
  max-width: 800px;
  margin: 0 auto;
}

.form-section {
  background: var(--card-bg);
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 20px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.form-section h3 {
  margin: 0 0 20px 0;
  font-size: 16px;
  color: var(--text-primary);
  font-weight: 600;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--border-color);
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
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s;
  background: var(--bg-color);
  color: var(--text-primary);
  box-sizing: border-box;
}

.form-group input:focus,
.form-group select:focus {
  border-color: var(--primary-color);
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 20px;
}

.btn-primary,
.btn-secondary {
  padding: 10px 20px;
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-primary {
  background: var(--primary-color);
  color: white;
  border: none;
}

.btn-primary:hover:not(:disabled) {
  background: var(--primary-hover);
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--card-bg);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-secondary:hover {
  background: var(--hover-color);
}

.message {
  padding: 12px 16px;
  border-radius: 6px;
  margin-top: 16px;
  text-align: center;
  font-size: 14px;
}

.message.success {
  background: #E8F5E8;
  color: #388E3C;
}

.message.error {
  background: #FFEBEE;
  color: #D32F2F;
}
</style>
