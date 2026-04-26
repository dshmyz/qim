<template>
  <el-dialog
    :model-value="modelValue"
    :title="title"
    :width="width"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <el-form
      ref="formRef"
      :model="model"
      :rules="rules"
      :label-width="labelWidth"
    >
      <slot></slot>
    </el-form>
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="loading" @click="$emit('confirm')">
        {{ confirmText }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import { ref } from 'vue'

interface Props {
  modelValue: boolean
  title: string
  model: Record<string, any>
  rules?: FormRules
  width?: string
  labelWidth?: string
  confirmText?: string
  loading?: boolean
}

withDefaults(defineProps<Props>(), {
  width: '500px',
  labelWidth: '80px',
  confirmText: '确定',
  loading: false
})

defineEmits<{
  'update:modelValue': [value: boolean]
  'confirm': []
}>()

const formRef = ref<FormInstance>()

defineExpose({ formRef })
</script>
