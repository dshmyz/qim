<template>
  <div class="shortcut-input-wrapper">
    <div class="shortcut-label">{{ label }}</div>
    <div class="shortcut-controls">
      <input
        ref="inputRef"
        class="shortcut-input"
        :value="displayText"
        :placeholder="capturing ? '按下组合键...' : '点击设置'"
        readonly
        :class="{ capturing }"
        @focus="startCapture"
        @blur="stopCapture"
        @keydown.prevent="handleKeyDown"
      />
      <button
        v-if="modelValue.accelerator"
        class="shortcut-clear-btn"
        title="清除"
        @mousedown.prevent="clearAccelerator"
      >×</button>
      <Switch
        :modelValue="modelValue.enabled"
        @update:modelValue="toggleEnabled"
        size="small"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import Switch from '../common/Switch.vue'
import { formatAccelerator, type ShortcutItem } from '../../composables/useShortcuts'

const props = defineProps<{
  modelValue: ShortcutItem
  label: string
  /** 'global' 用 Electron accelerator 格式，'editor' 用 CodeMirror Mod 格式 */
  scope?: 'global' | 'editor'
}>()

const emit = defineEmits<{
  'update:modelValue': [value: ShortcutItem]
}>()

const inputRef = ref<HTMLInputElement | null>(null)
const capturing = ref(false)

const displayText = computed(() => formatAccelerator(props.modelValue.accelerator))

function startCapture() {
  capturing.value = true
}

function stopCapture() {
  capturing.value = false
}

function toggleEnabled(value: boolean) {
  emit('update:modelValue', { ...props.modelValue, enabled: value })
}

function clearAccelerator() {
  emit('update:modelValue', { ...props.modelValue, accelerator: '' })
  inputRef.value?.focus()
}

function handleKeyDown(e: KeyboardEvent) {
  // Esc 取消
  if (e.key === 'Escape') {
    inputRef.value?.blur()
    return
  }
  // Backspace/Delete 清空
  if (e.key === 'Backspace' || e.key === 'Delete') {
    clearAccelerator()
    return
  }
  // 忽略纯修饰键按下
  if (['Shift', 'Control', 'Alt', 'Meta'].includes(e.key)) {
    return
  }

  // 组装 accelerator
  const parts: string[] = []
  const isEditor = props.scope === 'editor'
  // editor 用 Mod，global 用 CommandOrControl
  if (e.metaKey || e.ctrlKey) parts.push(isEditor ? 'Mod' : 'CommandOrControl')
  if (e.shiftKey) parts.push('Shift')
  if (e.altKey) parts.push('Alt')

  // 主键
  let key = e.key
  // 单字母大写
  if (key.length === 1) key = key.toUpperCase()
  // 特殊键名转换
  const keyMap: Record<string, string> = {
    'Enter': 'Return',
    ' ': 'Space',
    'ArrowUp': 'Up',
    'ArrowDown': 'Down',
    'ArrowLeft': 'Left',
    'ArrowRight': 'Right',
  }
  if (keyMap[key]) key = keyMap[key]

  // 必须包含修饰键，否则不合法
  if (parts.length === 0) return

  parts.push(key)
  const accelerator = parts.join('+')
  emit('update:modelValue', { ...props.modelValue, accelerator })
  inputRef.value?.blur()
}
</script>

<style scoped>
.shortcut-input-wrapper {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
}

.shortcut-label {
  color: var(--text-color);
  font-size: 14px;
}

.shortcut-controls {
  display: flex;
  align-items: center;
  gap: 8px;
}

.shortcut-input {
  width: 180px;
  padding: 6px 10px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md, 6px);
  background: var(--card-bg);
  color: var(--text-color);
  font-size: 13px;
  cursor: pointer;
  text-align: center;
  outline: none;
  transition: border-color 0.2s;
}

.shortcut-input:focus,
.shortcut-input.capturing {
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(51, 133, 255, 0.15);
}

.shortcut-input::placeholder {
  color: var(--text-secondary, #999);
}

.shortcut-clear-btn {
  border: none;
  background: transparent;
  color: var(--text-secondary, #999);
  cursor: pointer;
  font-size: 18px;
  padding: 2px 6px;
  line-height: 1;
  border-radius: 4px;
}

.shortcut-clear-btn:hover {
  color: var(--text-color);
  background: var(--hover-bg, rgba(0, 0, 0, 0.05));
}
</style>
