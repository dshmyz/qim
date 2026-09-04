<template>
  <el-select
    :model-value="modelValue"
    :placeholder="placeholder"
    clearable
    filterable
    remote
    :remote-method="remoteSearch"
    :loading="loading"
    style="width: 220px"
    @update:model-value="emit('update:modelValue', $event)"
    @clear="loadInitial"
    @change="trackName"
  >
    <el-option
      v-for="u in options"
      :key="u.id"
      :label="u.name"
      :value="u.id"
    />
  </el-select>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getUsers } from '@/api/users'

const props = withDefaults(defineProps<{
  modelValue?: number
  placeholder?: string
}>(), {
  placeholder: '输入姓名/用户名搜索',
})

const emit = defineEmits<{ (e: 'update:modelValue', v: number | undefined): void }>()

const options = ref<Array<{ id: number; name: string }>>([])
const loading = ref(false)
const selectedName = ref('')
let timer: ReturnType<typeof setTimeout> | null = null

function formatOptions(list: Array<{ id: number; nickname?: string; username: string }>) {
  return list.map((u) => ({ id: u.id, name: u.nickname || u.username }))
}

async function loadInitial() {
  try {
    const { data } = await getUsers({ page: 1, pageSize: 20 })
    options.value = formatOptions(data.data.list ?? [])
  } catch {
    options.value = []
  }
}

function trackName(id: number | undefined) {
  if (id == null) {
    selectedName.value = ''
    return
  }
  const hit = options.value.find((u) => u.id === id)
  if (hit) selectedName.value = hit.name
}

function ensureSelectedOption() {
  if (props.modelValue == null || !selectedName.value) return
  if (!options.value.some((u) => u.id === props.modelValue)) {
    options.value.unshift({ id: props.modelValue, name: selectedName.value })
  }
}

function remoteSearch(keyword: string) {
  if (timer) clearTimeout(timer)
  timer = setTimeout(async () => {
    loading.value = true
    try {
      const { data } = await getUsers({ page: 1, pageSize: 20, keyword })
      options.value = formatOptions(data.data.list ?? [])
      ensureSelectedOption()
    } finally {
      loading.value = false
    }
  }, 300)
}

onMounted(loadInitial)
</script>
