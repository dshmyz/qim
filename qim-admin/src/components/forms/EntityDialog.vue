<!-- src/components/forms/EntityDialog.vue -->
<template>
  <el-dialog
    :model-value="modelValue"
    :title="mode === 'create' ? createTitle : editTitle"
    width="500px"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <el-form
      ref="formRef"
      :model="formData"
      :rules="rules"
      label-width="80px"
    >
      <FieldRenderer
        v-for="field in fields"
        :key="field.name"
        :field="field"
        :model="formData"
      />
    </el-form>
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="loading" @click="handleSave">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import FieldRenderer from './FieldRenderer.vue'
import type { FormField } from './FieldRenderer.vue'

interface Props {
  modelValue: boolean
  mode: 'create' | 'edit'
  fields: FormField[]
  rules?: FormRules
  initialData?: Record<string, unknown>
  createTitle?: string
  editTitle?: string
}

const props = withDefaults(defineProps<Props>(), {
  rules: () => ({}),
  initialData: () => ({}),
  createTitle: '创建',
  editTitle: '编辑',
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'save': [data: Record<string, unknown>, callbacks: { onSuccess: () => void; onError: (error: any) => void }]
}>()

const formRef = ref<FormInstance>()
const formData = ref<Record<string, unknown>>({})
const loading = ref(false)

watch(
  () => props.modelValue,
  (val) => {
    if (val) {
      formData.value = { ...props.initialData }
      // 清除验证状态
      nextTick(() => {
        formRef.value?.resetFields()
      })
    }
  }
)

async function handleSave() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  loading.value = true
  
  emit('save', { ...formData.value }, {
    onSuccess: () => {
      loading.value = false
      emit('update:modelValue', false)
    },
    onError: (error: any) => {
      console.error('[EntityDialog] save failed:', error)
      loading.value = false
    }
  })
}
</script>
