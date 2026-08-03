<template>
  <div class="capture-groups-editor">
    <div class="capture-groups-list">
      <div
        v-for="(item, index) in items"
        :key="index"
        class="capture-groups-row"
      >
        <input
          v-model="item.key"
          type="text"
          class="rr-input capture-groups-key"
          placeholder="变量名（如 number）"
          @input="emitChange"
        />
        <span class="capture-groups-arrow">→ 第</span>
        <input
          v-model.number="item.value"
          type="number"
          min="1"
          class="rr-input capture-groups-value"
          @input="emitChange"
        />
        <span class="capture-groups-suffix">个捕获组</span>
        <button
          type="button"
          class="capture-groups-remove"
          title="删除"
          @click="removeItem(index)"
        >
          <i class="fas fa-times"></i>
        </button>
      </div>
    </div>
    <button type="button" class="rr-btn rr-btn-text" @click="addItem">
      <i class="fas fa-plus"></i> 添加捕获组
    </button>
    <p class="capture-groups-hint">
      捕获组序号对应正则中的括号顺序。例如 <code>([A-Z]+)-(\\d+)</code> 中第1个是 project，第2个是 number。
    </p>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

interface CaptureItem {
  key: string
  value: number
}

const props = defineProps<{
  modelValue: Record<string, number>
}>()

const emit = defineEmits<{
  'update:modelValue': [value: Record<string, number>]
}>()

const items = ref<CaptureItem[]>([])

// 把 Record<string, number> 转成数组形式方便编辑
function syncFromModel() {
  const obj = props.modelValue || {}
  items.value = Object.entries(obj).map(([key, value]) => ({ key, value }))
}

watch(() => props.modelValue, syncFromModel, { immediate: true })

function emitChange() {
  const result: Record<string, number> = {}
  for (const item of items.value) {
    const key = item.key.trim()
    if (key && Number.isFinite(item.value) && item.value >= 1) {
      result[key] = item.value
    }
  }
  emit('update:modelValue', result)
}

function addItem() {
  items.value.push({ key: '', value: 1 })
}

function removeItem(index: number) {
  items.value.splice(index, 1)
  emitChange()
}
</script>

<style scoped>
.capture-groups-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.capture-groups-row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.capture-groups-key {
  flex: 1;
}
.capture-groups-arrow {
  color: #909399;
  font-size: 13px;
  white-space: nowrap;
}
.capture-groups-value {
  width: 60px;
  text-align: center;
}
.capture-groups-suffix {
  color: #909399;
  font-size: 13px;
  white-space: nowrap;
}
.capture-groups-remove {
  background: none;
  border: none;
  color: #f56c6c;
  cursor: pointer;
  padding: 4px 6px;
  font-size: 13px;
}
.capture-groups-remove:hover {
  color: #e64242;
}
.capture-groups-hint {
  margin-top: 6px;
  font-size: 12px;
  color: #909399;
  line-height: 1.5;
}
.capture-groups-hint code {
  background: #f5f7fa;
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 11px;
  color: #e6a23c;
}
</style>
