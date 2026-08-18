<template>
  <select
    class="form-input folder-tree-select"
    :value="modelValue === null ? '' : modelValue"
    @change="handleChange"
  >
    <option :value="''">根目录</option>
    <option v-for="opt in visibleOptions" :key="opt.folder.id" :value="opt.folder.id">
      {{ '　'.repeat(opt.depth) }}{{ opt.folder.name }}
    </option>
  </select>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { FolderOption } from '../../../composables/useFolderTree'

defineOptions({
  name: 'FolderTreeSelect'
})

interface Props {
  options: FolderOption[]
  modelValue: number | null
  excludeIds?: number[]
}

const props = withDefaults(defineProps<Props>(), {
  options: () => [],
  modelValue: null,
  excludeIds: () => []
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: number | null): void
}>()

// 排除指定文件夹（如当前文件夹自身）；子树不排除——移入子目录是合法操作
const visibleOptions = computed(() =>
  props.options.filter(o => !props.excludeIds.includes(o.folder.id))
)

function handleChange(event: Event) {
  const value = (event.target as HTMLSelectElement).value
  emit('update:modelValue', value === '' ? null : Number(value))
}
</script>

<style scoped>
/* 复用各 modal 的 form-input 体系（根元素继承父级 scoped 类）；缩进用全角空格无需额外样式 */
.folder-tree-select {
  width: 100%;
}
</style>
