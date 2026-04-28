<!-- src/components/data/SearchField.vue -->
<template>
  <el-form-item :label="label">
    <el-input
      v-if="type === 'input'"
      :model-value="modelValue"
      :placeholder="placeholder"
      clearable
      @update:model-value="$emit('update:modelValue', $event)"
      @keyup.enter="$emit('search')"
    />
    <el-select
      v-else-if="type === 'select'"
      :model-value="modelValue"
      :placeholder="placeholder"
      clearable
      @update:model-value="$emit('update:modelValue', $event)"
    >
      <el-option
        v-for="opt in options"
        :key="opt.value"
        :label="opt.label"
        :value="opt.value"
      />
    </el-select>
  </el-form-item>
</template>

<script setup lang="ts">
interface Option {
  label: string
  value: string | number
}

interface Props {
  modelValue?: string | number
  label: string
  type?: 'input' | 'select'
  placeholder?: string
  options?: Option[]
}

withDefaults(defineProps<Props>(), {
  type: 'input',
  placeholder: '请输入',
  options: () => []
})

defineEmits<{
  'update:modelValue': [value: string | number]
  'search': []
}>()
</script>
