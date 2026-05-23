<template>
  <el-dialog
    :model-value="visible"
    title="用户 AI 配置"
    width="600px"
    @update:model-value="$emit('update:visible', $event)"
  >
    <p class="ai-config-hint">为用户 <strong>{{ username }}</strong> 管理 AI 配置</p>

    <div v-loading="loading">
      <el-empty v-if="!loading && configs.length === 0" description="该用户暂无 AI 配置" />

      <div v-else class="config-list">
        <div v-for="config in configs" :key="config.id" class="config-card">
          <div class="config-header">
            <span class="config-name">{{ config.config_name || '默认配置' }}</span>
            <el-tag :type="config.is_verified ? 'success' : 'warning'" size="small">
              {{ config.is_verified ? '已验证' : '未验证' }}
            </el-tag>
          </div>
          <div class="config-body">
            <div class="config-row">
              <span class="config-label">提供商</span>
              <span class="config-value">{{ config.provider }}</span>
            </div>
            <div class="config-row">
              <span class="config-label">模型</span>
              <span class="config-value">{{ config.model_name }}</span>
            </div>
            <div class="config-row">
              <span class="config-label">温度</span>
              <span class="config-value">{{ config.temperature }}</span>
            </div>
            <div class="config-row">
              <span class="config-label">最大 Token</span>
              <span class="config-value">{{ config.max_tokens }}</span>
            </div>
            <div v-if="config.base_url" class="config-row">
              <span class="config-label">API 端点</span>
              <span class="config-value config-url">{{ config.base_url }}</span>
            </div>
          </div>
          <div class="config-footer">
            <el-button size="small" type="primary" @click="handleEditConfig(config)">编辑</el-button>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <el-button @click="$emit('update:visible', false)">关闭</el-button>
    </template>
  </el-dialog>

  <el-dialog
    v-model="editDialogVisible"
    title="编辑 AI 配置"
    width="500px"
    :close-on-click-modal="false"
  >
    <el-form ref="formRef" :model="editForm" label-width="100px">
      <el-form-item label="配置名称">
        <el-input v-model="editForm.config_name" placeholder="配置名称" />
      </el-form-item>
      <el-form-item label="提供商">
        <el-select v-model="editForm.provider" placeholder="选择提供商">
          <el-option label="OpenAI" value="openai" />
          <el-option label="百度" value="baidu" />
          <el-option label="阿里" value="alibaba" />
          <el-option label="腾讯" value="tencent" />
          <el-option label="字节跳动" value="bytedance" />
          <el-option label="Anthropic" value="anthropic" />
        </el-select>
      </el-form-item>
      <el-form-item label="API Key">
        <el-input v-model="editForm.api_key" type="password" show-password placeholder="留空表示不修改" />
      </el-form-item>
      <el-form-item label="模型名称">
        <el-input v-model="editForm.model_name" placeholder="例如 gpt-4" />
      </el-form-item>
      <el-form-item label="API 端点">
        <el-input v-model="editForm.base_url" placeholder="例如 https://api.openai.com" />
      </el-form-item>
      <el-form-item label="最大 Token">
        <el-input-number v-model="editForm.max_tokens" :min="1" :max="32768" />
      </el-form-item>
      <el-form-item label="温度参数">
        <el-input-number v-model="editForm.temperature" :min="0" :max="2" :precision="1" :step="0.1" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="editDialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="editSaving" @click="handleSaveConfig">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { FormInstance } from 'element-plus'
import { getUserAIConfigs, updateUserAIConfig } from '@/api/users'
import type { UserAIConfig } from '@/types'

interface Props {
  visible: boolean
  userId: number
  username: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
}>()

const loading = ref(false)
const configs = ref<UserAIConfig[]>([])

const fetchConfigs = async () => {
  if (!props.userId) return
  loading.value = true
  try {
    const { data } = await getUserAIConfigs(props.userId)
    configs.value = data.data.list
  } catch (error) {
    console.error('[AIConfigDialog] fetch configs failed:', error)
    ElMessage.error('获取AI配置失败')
  } finally {
    loading.value = false
  }
}

watch(() => props.visible, (val) => {
  if (val && props.userId) {
    fetchConfigs()
  }
})

const editDialogVisible = ref(false)
const editSaving = ref(false)
const editingConfig = ref<UserAIConfig | null>(null)
const formRef = ref<FormInstance>()

const editForm = ref({
  config_name: '',
  provider: '',
  api_key: '',
  model_name: '',
  base_url: '',
  max_tokens: 1000,
  temperature: 0.7,
})

const handleEditConfig = (config: UserAIConfig) => {
  editingConfig.value = config
  editForm.value = {
    config_name: config.config_name || '',
    provider: config.provider || '',
    api_key: '',
    model_name: config.model_name || '',
    base_url: config.base_url || '',
    max_tokens: config.max_tokens || 1000,
    temperature: config.temperature || 0.7,
  }
  editDialogVisible.value = true
}

const handleSaveConfig = async () => {
  if (!editingConfig.value || !props.userId) return
  editSaving.value = true
  try {
    const payload: Record<string, unknown> = {
      config_name: editForm.value.config_name || undefined,
      provider: editForm.value.provider || undefined,
      model_name: editForm.value.model_name || undefined,
      base_url: editForm.value.base_url || undefined,
      max_tokens: editForm.value.max_tokens,
      temperature: editForm.value.temperature,
    }
    if (editForm.value.api_key) {
      payload.api_key = editForm.value.api_key
    }
    await updateUserAIConfig(props.userId, editingConfig.value.id, payload as any)
    ElMessage.success('AI配置更新成功')
    editDialogVisible.value = false
    fetchConfigs()
  } catch (error) {
    console.error('[AIConfigDialog] save config failed:', error)
    ElMessage.error('AI配置保存失败')
  } finally {
    editSaving.value = false
  }
}
</script>

<style scoped>
.ai-config-hint {
  margin-bottom: 16px;
  color: var(--color-text-secondary);
  font-weight: 500;
}

.config-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.config-card {
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  padding: 16px;
  transition: border-color 0.2s;
}

.config-card:hover {
  border-color: var(--el-color-primary);
}

.config-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.config-name {
  font-weight: 600;
  font-size: 15px;
  color: var(--color-text-primary);
}

.config-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}

.config-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.config-label {
  min-width: 80px;
  font-size: 13px;
  color: var(--color-text-secondary);
}

.config-value {
  font-size: 13px;
  color: var(--color-text-primary);
}

.config-url {
  font-size: 12px;
  color: var(--color-text-muted);
  word-break: break-all;
}

.config-footer {
  display: flex;
  justify-content: flex-end;
}
</style>