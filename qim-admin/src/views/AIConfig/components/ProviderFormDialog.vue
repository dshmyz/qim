<template>
  <el-dialog
    :model-value="visible"
    :title="isEdit ? '编辑提供商' : '添加提供商'"
    width="650px"
    @closed="handleReset"
    @update:model-value="(val: boolean) => emit('update:visible', val)"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="100px"
    >
      <el-form-item label="提供商类型" prop="type">
        <el-select
          v-model="form.type"
          :disabled="isEdit"
          placeholder="选择提供商类型"
          style="width: 100%"
          @change="handleTypeChange"
        >
          <el-option
            v-for="item in providerTypeOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="名称" prop="name">
        <el-input v-model="form.name" placeholder="例如：OpenAI 官方" />
      </el-form-item>

      <el-form-item label="API 密钥" prop="apiKey">
        <el-input
          v-model="form.apiKey"
          type="password"
          show-password
          placeholder="请输入 API Key"
        />
      </el-form-item>

      <el-form-item label="API 端点" prop="apiEndpoint">
        <el-input v-model="form.apiEndpoint" placeholder="请输入 API 端点地址" />
        <div class="form-item-tip">{{ endpointTip }}</div>
      </el-form-item>

      <el-form-item label="支持模型" prop="models">
        <el-select
          v-model="form.models"
          multiple
          filterable
          allow-create
          default-first-option
          placeholder="输入或选择模型名称"
          style="width: 100%"
        >
          <el-option
            v-for="model in availableModels"
            :key="model"
            :label="model"
            :value="model"
          />
        </el-select>
        <div class="form-item-tip">可手动输入自定义模型名称</div>
      </el-form-item>

      <el-form-item label="优先级">
        <el-input-number
          v-model="form.priority"
          :min="0"
          :max="100"
          :step="1"
          controls-position="right"
          style="width: 100%"
        />
        <div class="form-item-tip">数值越大优先级越高，用于多提供商时选择默认提供商</div>
      </el-form-item>

      <el-form-item label="启用">
        <el-switch v-model="form.enabled" />
      </el-form-item>

      <el-form-item label="备注">
        <el-input
          v-model="form.remark"
          type="textarea"
          :rows="3"
          placeholder="可选，填写提供商备注信息"
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" :loading="submitLoading" @click="handleConfirm">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import {
  PROVIDER_TYPE_LABELS,
  DEFAULT_ENDPOINTS,
  DEFAULT_MODELS,
  type ProviderType,
  type AIProvider,
} from '@/types/ai'

interface Props {
  visible: boolean
  isEdit: boolean
  providerData?: Partial<AIProvider> | null
  submitLoading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  visible: false,
  isEdit: false,
  providerData: null,
  submitLoading: false,
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'confirm': [data: Record<string, unknown>]
}>()

const formRef = ref<FormInstance>()

const providerTypeOptions = Object.entries(PROVIDER_TYPE_LABELS).map(([value, label]) => ({
  value,
  label,
}))

const form = reactive({
  id: 0,
  name: '',
  type: 'openai' as ProviderType,
  apiKey: '',
  apiEndpoint: '',
  models: [] as string[],
  priority: 0,
  enabled: true,
  remark: '',
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入提供商名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择提供商类型', trigger: 'change' }],
  apiKey: [{ required: true, message: '请输入 API Key', trigger: 'blur' }],
  apiEndpoint: [
    { required: true, message: '请输入 API 端点', trigger: 'blur' },
    { type: 'url', message: '请输入有效的 URL 地址', trigger: 'blur' },
  ],
  models: [{ required: true, message: '请至少添加一个模型', trigger: 'change', type: 'array' as const }],
}

const availableModels = computed(() => DEFAULT_MODELS[form.type] || [])

const endpointTip = computed(() => {
  const defaultEndpoint = DEFAULT_ENDPOINTS[form.type]
  return defaultEndpoint ? `默认端点：${defaultEndpoint}` : '请输入自定义 API 端点'
})

watch(() => props.providerData, (newData) => {
  if (newData) {
    form.id = newData.id || 0
    form.name = newData.name || ''
    form.type = (newData.type as ProviderType) || 'openai'
    form.apiKey = newData.apiKey || ''
    form.apiEndpoint = newData.apiEndpoint || ''
    form.models = Array.isArray(newData.models) ? [...newData.models] : []
    form.priority = newData.priority ?? 0
    form.enabled = newData.enabled ?? true
    form.remark = newData.remark || ''
  }
}, { deep: true, immediate: true })

function handleTypeChange(type: ProviderType) {
  form.apiEndpoint = DEFAULT_ENDPOINTS[type] || ''
  if (!props.isEdit && (!form.models.length || form.models.length === 0)) {
    form.models = [...(DEFAULT_MODELS[type] || [])]
  }
}

function handleReset() {
  formRef.value?.resetFields()
  form.id = 0
  form.name = ''
  form.type = 'openai'
  form.apiKey = ''
  form.apiEndpoint = DEFAULT_ENDPOINTS.openai
  form.models = [...DEFAULT_MODELS.openai]
  form.priority = 0
  form.enabled = true
  form.remark = ''
}

function handleClose() {
  emit('update:visible', false)
}

async function handleConfirm() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    emit('confirm', { ...form })
  })
}
</script>

<style scoped>
.form-item-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}
</style>
