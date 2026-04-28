<!-- src/components/forms/FieldRenderer.vue -->
<template>
  <el-form-item :label="field.label" :prop="field.name" :rules="rules">
    <el-input
      v-if="field.type === 'input'"
      v-model="model[field.name]"
      v-bind="field.props"
    />
    <el-input
      v-else-if="field.type === 'textarea'"
      v-model="model[field.name]"
      type="textarea"
      v-bind="field.props"
    />
    <el-input
      v-else-if="field.type === 'password'"
      v-model="model[field.name]"
      type="password"
      show-password
      v-bind="field.props"
    />
    <el-select
      v-else-if="field.type === 'select'"
      v-model="model[field.name]"
      v-bind="field.props"
    >
      <el-option
        v-for="opt in field.options"
        :key="opt.value"
        :label="opt.label"
        :value="opt.value"
      />
    </el-select>
    <el-switch
      v-else-if="field.type === 'switch'"
      v-model="model[field.name]"
      v-bind="field.props"
    />
    <el-input-number
      v-else-if="field.type === 'number'"
      v-model="model[field.name]"
      v-bind="field.props"
    />
  </el-form-item>
</template>

<script setup lang="ts">
import type { FormItemRule } from 'element-plus'

interface FieldOption {
  label: string
  value: string | number | boolean
}

export interface FormField {
  name: string
  label: string
  type: 'input' | 'textarea' | 'password' | 'select' | 'switch' | 'number'
  props?: Record<string, unknown>
  options?: FieldOption[]
  required?: boolean
}

interface Props {
  field: FormField
  model: Record<string, unknown>
  rules?: FormItemRule[]
}

defineProps<Props>()
</script>
